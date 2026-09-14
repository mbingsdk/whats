package meta

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func validSignature(raw []byte, headers []string, secret string) bool {
	if len(headers) != 1 || len(headers[0]) != 71 || !strings.HasPrefix(headers[0], "sha256=") {
		return false
	}
	signature, e := hex.DecodeString(headers[0][7:])
	if e != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(raw)
	return hmac.Equal(signature, mac.Sum(nil))
}
func verifyChallenge(query, token string) (string, bool) {
	q, e := url.ParseQuery(query)
	// Meta can include extra verification metadata; only the three required keys carry authority.
	if e != nil {
		return "", false
	}
	for _, key := range []string{"hub.mode", "hub.verify_token", "hub.challenge"} {
		if len(q[key]) != 1 || q.Get(key) == "" {
			return "", false
		}
	}
	challenge := q.Get("hub.challenge")
	actual := sha256.Sum256([]byte(q.Get("hub.verify_token")))
	expected := sha256.Sum256([]byte(token))
	if q.Get("hub.mode") != "subscribe" || len(challenge) > 1024 || strings.ContainsAny(challenge, "\r\n\x00") || subtle.ConstantTimeCompare(actual[:], expected[:]) != 1 {
		return "", false
	}
	return challenge, true
}

type envelope struct {
	Object string  `json:"object"`
	Entry  []entry `json:"entry"`
}
type entry struct {
	ID      string          `json:"id"`
	Time    json.RawMessage `json:"time"`
	Changes []change        `json:"changes"`
}
type change struct {
	Field string          `json:"field"`
	Value json.RawMessage `json:"value"`
}

func parseEnvelope(raw []byte) (envelope, error) {
	var v envelope
	e := json.Unmarshal(raw, &v)
	return v, e
}
func evidenceAAD(org, id uuid.UUID) string {
	return org.String() + ":" + id.String() + ":meta-webhook:v1"
}
func hash(raw []byte) string { v := sha256.Sum256(raw); return hex.EncodeToString(v[:]) }
func (s *Service) webhook(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if !s.Config.Enabled {
		http.Error(w, "Unavailable", 503)
		return
	}
	b, e := s.binding(r.Context())
	if e != nil {
		http.Error(w, "Unavailable", 503)
		return
	}
	if r.PathValue("callback") != b.Callback.String() {
		http.NotFound(w, r)
		return
	}
	if r.Method == http.MethodGet {
		challenge, ok := verifyChallenge(r.URL.RawQuery, string(s.Config.VerifyToken))
		if !ok {
			http.Error(w, "Verification rejected", 403)
			return
		}
		e = s.scoped(r.Context(), b, func(tx pgx.Tx) error {
			_, e := tx.Exec(r.Context(), "UPDATE app.meta_apps SET challenge_verified_at=now() WHERE id=$1", b.App)
			if e != nil {
				return e
			}
			return audit(r.Context(), tx, b.Org, nil, "meta.webhook.challenge", b.App, "meta-callback")
		})
		if e != nil {
			http.Error(w, "Unavailable", 503)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		_, _ = io.WriteString(w, challenge)
		return
	}
	if r.Header.Get("Content-Encoding") != "" && r.Header.Get("Content-Encoding") != "identity" {
		http.Error(w, "Encoding rejected", 415)
		return
	}
	raw, e := io.ReadAll(http.MaxBytesReader(w, r.Body, 4*1024*1024))
	if e != nil {
		http.Error(w, "Body rejected", 413)
		return
	}
	if !validSignature(raw, r.Header.Values("X-Hub-Signature-256"), string(s.Config.AppSecret)) {
		http.Error(w, "Signature rejected", 403)
		return
	}
	v, e := parseEnvelope(raw)
	if e != nil || v.Object != "whatsapp_business_account" || len(v.Entry) == 0 || len(v.Entry) > 1000 {
		http.Error(w, "Envelope rejected", 400)
		return
	}
	id := newID()
	ciphertext, e := s.Auth.SealEvidence(raw, evidenceAAD(b.Org, id))
	if e != nil {
		http.Error(w, "Unavailable", 503)
		return
	}
	request := w.Header().Get("X-Request-ID")
	if request == "" {
		request = uuid.NewString()
	}
	e = s.scoped(r.Context(), b, func(tx pgx.Tx) error {
		// Batch binding reads keep acknowledgement work bounded independently of child count.
		wabas := map[string]bool{}
		wr, e := tx.Query(r.Context(), "SELECT external_id FROM app.wabas WHERE app_id=$1", b.App)
		if e != nil {
			return e
		}
		for wr.Next() {
			var id string
			if e = wr.Scan(&id); e != nil {
				wr.Close()
				return e
			}
			wabas[id] = true
		}
		wr.Close()
		if e = wr.Err(); e != nil {
			return e
		}
		phones := map[string]bool{}
		pr, e := tx.Query(r.Context(), "SELECT w.external_id,p.external_id FROM app.phone_numbers p JOIN app.wabas w ON w.organization_id=p.organization_id AND w.id=p.waba_id WHERE w.app_id=$1", b.App)
		if e != nil {
			return e
		}
		for pr.Next() {
			var waba, phone string
			if e = pr.Scan(&waba, &phone); e != nil {
				pr.Close()
				return e
			}
			phones[waba+":"+phone] = true
		}
		pr.Close()
		if e = pr.Err(); e != nil {
			return e
		}
		quarantine := false
		for _, entry := range v.Entry {
			if !wabas[entry.ID] {
				quarantine = true
			}
			for _, c := range entry.Changes {
				var value struct {
					Metadata struct {
						Phone string `json:"phone_number_id"`
					} `json:"metadata"`
				}
				if json.Unmarshal(c.Value, &value) != nil {
					continue
				}
				if value.Metadata.Phone != "" && !phones[entry.ID+":"+value.Metadata.Phone] {
					quarantine = true
				}
			}
		}
		state := "QUEUED"
		if quarantine {
			state = "QUARANTINED"
		}
		_, e = tx.Exec(r.Context(), "INSERT INTO app.webhook_events(id,organization_id,app_id,raw_digest,raw_ciphertext,request_id,state) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(organization_id,app_id,raw_digest) DO NOTHING", id, b.Org, b.App, hash(raw), ciphertext, request, state)
		if e != nil {
			return e
		}
		_, e = tx.Exec(r.Context(), "UPDATE app.meta_apps SET last_webhook_at=now() WHERE id=$1", b.App)
		return e
	})
	if e != nil {
		http.Error(w, "Persistence unavailable", 503)
		return
	}
	w.WriteHeader(200)
	_, _ = io.WriteString(w, "EVENT_RECEIVED")
}

// Inspection exposes structural fields only; arbitrary keys/values can themselves contain PII.
func redactPayload(raw []byte) any {
	v, e := parseEnvelope(raw)
	if e != nil {
		return map[string]string{"state": "INVALID"}
	}
	entries := []any{}
	for i, entry := range v.Entry {
		if i >= 100 {
			break
		}
		changes := []any{}
		for j, c := range entry.Changes {
			if j >= 100 {
				break
			}
			class := classifyField(c.Field)
			changes = append(changes, map[string]any{"class": class, "value": "[REDACTED]"})
		}
		entries = append(entries, map[string]any{"id": "[REDACTED]", "changes": changes})
	}
	return map[string]any{"object": "whatsapp_business_account", "entry": entries, "redaction": "structure only; provider strings, IDs, contacts and content withheld"}
}
