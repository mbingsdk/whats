package meta

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"strings"
)

func classifyField(field string) string {
	switch field {
	case "messages":
		return "MESSAGING"
	case "phone_number_name_update", "phone_number_quality_update", "account_update", "account_review_update", "message_template_status_update", "message_template_quality_update":
		return "ASSET"
	default:
		return "UNKNOWN"
	}
}

type fact struct {
	WABA, Phone, Class, Key string
	Uncertain               bool
}

func canonical(raw json.RawMessage) string {
	var v any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(&v) != nil {
		return ""
	}
	b, _ := json.Marshal(v)
	return string(b)
}
func classify(raw []byte) ([]fact, string) {
	v, e := parseEnvelope(raw)
	if e != nil || len(v.Entry) == 0 {
		return nil, "INVALID"
	}
	out := []fact{}
	unknown := false
	for _, entry := range v.Entry {
		if !numericID.MatchString(entry.ID) || len(entry.Changes) == 0 {
			return nil, "INVALID"
		}
		for _, c := range entry.Changes {
			if len(out) > 10000 {
				return nil, "INVALID"
			}
			if len(c.Value) == 0 || c.Field == "" {
				return nil, "INVALID"
			}
			cls := classifyField(c.Field)
			if cls == "UNKNOWN" {
				unknown = true
				out = append(out, fact{WABA: entry.ID, Class: "UNKNOWN", Key: hash([]byte(entry.ID + ":" + c.Field + ":" + canonical(c.Value))), Uncertain: true})
				continue
			}
			if cls == "ASSET" {
				out = append(out, fact{WABA: entry.ID, Class: "ASSET", Key: hash([]byte(entry.ID + ":" + c.Field + ":" + string(entry.Time) + ":" + canonical(c.Value))), Uncertain: len(entry.Time) == 0})
				continue
			}
			var value struct {
				Metadata struct {
					Phone string `json:"phone_number_id"`
				} `json:"metadata"`
				Messages []json.RawMessage `json:"messages"`
				Statuses []json.RawMessage `json:"statuses"`
			}
			if json.Unmarshal(c.Value, &value) != nil || !numericID.MatchString(value.Metadata.Phone) {
				return nil, "INVALID"
			}
			if len(value.Messages)+len(value.Statuses) == 0 {
				unknown = true
				out = append(out, fact{WABA: entry.ID, Phone: value.Metadata.Phone, Class: "UNKNOWN", Key: hash([]byte(entry.ID + ":" + canonical(c.Value))), Uncertain: true})
			}
			for _, message := range value.Messages {
				var m struct {
					ID string `json:"id"`
				}
				if json.Unmarshal(message, &m) != nil || m.ID == "" || len(m.ID) > 2048 {
					return nil, "INVALID"
				}
				out = append(out, fact{WABA: entry.ID, Phone: value.Metadata.Phone, Class: "INBOUND_MESSAGE", Key: hash([]byte("inbound:" + value.Metadata.Phone + ":" + m.ID))})
			}
			for _, status := range value.Statuses {
				var m struct {
					ID, Status, Timestamp         string
					Errors, Pricing, Conversation json.RawMessage
				}
				if json.Unmarshal(status, &m) != nil || m.ID == "" || m.Status == "" || m.Timestamp == "" {
					return nil, "INVALID"
				}
				// Changed error/pricing/conversation observations are distinct facts. No invoice interpretation.
				key := strings.Join([]string{"status", value.Metadata.Phone, m.ID, m.Status, m.Timestamp, canonical(m.Errors), canonical(m.Pricing), canonical(m.Conversation)}, "\x00")
				out = append(out, fact{WABA: entry.ID, Phone: value.Metadata.Phone, Class: "MESSAGE_STATUS", Key: hash([]byte(key))})
			}
		}
	}
	if unknown {
		return out, "UNKNOWN"
	}
	return out, "PROCESSED"
}
func (s *Service) processOne(ctx context.Context, b binding) error {
	var id, lease uuid.UUID
	var ciphertext []byte
	var count, generation int
	e := s.scoped(ctx, b, func(tx pgx.Tx) error {
		e := tx.QueryRow(ctx, `SELECT id,raw_ciphertext,attempt_count,generation FROM app.webhook_events WHERE app_id=$1 AND ((state IN ('QUEUED','RETRY_WAIT') AND next_attempt_at<=now()) OR (state='PROCESSING' AND lease_until<now())) ORDER BY received_at FOR UPDATE SKIP LOCKED LIMIT 1`, b.App).Scan(&id, &ciphertext, &count, &generation)
		if e != nil {
			return e
		}
		count++
		lease = newID()
		_, e = tx.Exec(ctx, "UPDATE app.webhook_events SET state='PROCESSING',attempt_count=$2,lease_token=$3,lease_until=now()+interval '1 minute' WHERE id=$1", id, count, lease)
		return e
	})
	if e != nil {
		return e
	}
	raw, decryptErr := s.Auth.OpenEvidence(ciphertext, evidenceAAD(b.Org, id))
	facts, state := classify(raw)
	code := ""
	if decryptErr != nil {
		state = "DEAD_LETTER"
		code = "EVIDENCE_UNAVAILABLE"
	}
	if count > 8 {
		state = "DEAD_LETTER"
		code = "ATTEMPTS_EXHAUSTED"
	}
	return s.scoped(ctx, b, func(tx pgx.Tx) error {
		var matches bool
		if e := tx.QueryRow(ctx, "SELECT lease_token=$2 AND lease_until>now() FROM app.webhook_events WHERE id=$1 FOR UPDATE", id, lease).Scan(&matches); e != nil {
			return e
		}
		if !matches {
			return errors.New("webhook lease lost")
		}
		if state == "PROCESSED" || state == "UNKNOWN" {
			work, e := tx.Begin(ctx)
			if e != nil {
				return e
			}
			for _, f := range facts {
				var waba uuid.UUID
				var phone *uuid.UUID
				if e = work.QueryRow(ctx, "SELECT id FROM app.wabas WHERE external_id=$1 AND app_id=$2", f.WABA, b.App).Scan(&waba); e != nil {
					break
				}
				if f.Phone != "" {
					e = work.QueryRow(ctx, "SELECT id FROM app.phone_numbers WHERE external_id=$1 AND waba_id=$2", f.Phone, waba).Scan(&phone)
					if e != nil {
						break
					}
				}
				_, e = work.Exec(ctx, "INSERT INTO app.webhook_facts(id,organization_id,event_id,waba_id,phone_id,event_class,effect_key,uncertain_identity) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(organization_id,effect_key) DO NOTHING", newID(), b.Org, id, waba, phone, f.Class, f.Key, f.Uncertain)
				if e != nil {
					break
				}
			}
			if e != nil {
				_ = work.Rollback(ctx)
				if errors.Is(e, pgx.ErrNoRows) {
					state = "QUARANTINED"
					code = "ASSET_BINDING_MISMATCH"
				} else {
					state = "RETRY_WAIT"
					code = "INTERNAL_PROCESSING"
					if count >= 8 {
						state = "DEAD_LETTER"
					}
				}
			} else if e = work.Commit(ctx); e != nil {
				return e
			}
		}
		if state == "INVALID" {
			code = "INVALID_EVENT"
		}
		eventClass := state
		if state == "PROCESSED" {
			eventClass = "MIXED"
			if len(facts) > 0 {
				eventClass = facts[0].Class
				for _, f := range facts {
					if f.Class != eventClass {
						eventClass = "MIXED"
						break
					}
				}
			}
		}
		if _, e := tx.Exec(ctx, "INSERT INTO app.webhook_processing_attempts(id,organization_id,event_id,generation,attempt_no,result,error_code,parser_version) VALUES($1,$2,$3,$4,$5,$6,nullif($7,''),1)", newID(), b.Org, id, generation, count, state, code); e != nil {
			return e
		}
		if _, e := tx.Exec(ctx, "UPDATE app.webhook_events SET state=$2,event_class=$3,last_error=nullif($4,''),next_attempt_at=now()+$5::interval,lease_token=NULL,lease_until=NULL,processed_at=CASE WHEN $2='RETRY_WAIT' THEN NULL ELSE now() END WHERE id=$1", id, state, eventClass, code, backoff(count)); e != nil {
			return e
		}
		return audit(ctx, tx, b.Org, nil, "meta.webhook."+state, id, "meta-worker")
	})
}
