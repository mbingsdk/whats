package inbox

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"strconv"
	"strings"
	"time"
)

type envelope struct {
	Entry []struct {
		ID      string `json:"id"`
		Changes []struct {
			Field string          `json:"field"`
			Value json.RawMessage `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}
type inboundValue struct {
	Metadata struct {
		Phone string `json:"phone_number_id"`
	} `json:"metadata"`
	Messages []json.RawMessage `json:"messages"`
	Statuses []json.RawMessage `json:"statuses"`
}
type inboundMessage struct {
	ID, From, Type, Timestamp string
	Text                      struct {
		Body string `json:"body"`
	} `json:"text"`
	Context struct {
		ID string `json:"id"`
	} `json:"context"`
}

func providerTime(v string) *time.Time {
	seconds, e := strconv.ParseInt(v, 10, 64)
	if e != nil || seconds <= 0 || seconds > 253402300799 {
		return nil
	}
	t := time.Unix(seconds, 0).UTC()
	return &t
}
func windowBasis(provider *time.Time, ingress, now time.Time) *time.Time {
	if provider == nil || provider.After(ingress.Add(5*time.Second)) || provider.After(now) {
		return nil
	}
	basis := *provider
	if ingress.Before(basis) {
		basis = ingress
	}
	return &basis
}
func typedContent(raw json.RawMessage, m inboundMessage) (string, map[string]any, string) {
	kind := strings.ToUpper(m.Type)
	content := map[string]any{}
	switch kind {
	case "TEXT":
		if !validText(m.Text.Body, 4096) {
			return "UNKNOWN", map[string]any{"provider_type": m.Type}, ""
		}
		return kind, map[string]any{"body": m.Text.Body}, m.Text.Body
	case "IMAGE", "VIDEO", "AUDIO", "DOCUMENT", "STICKER":
		var fields map[string]json.RawMessage
		_ = json.Unmarshal(raw, &fields)
		var media map[string]any
		_ = json.Unmarshal(fields[m.Type], &media)
		for _, key := range []string{"id", "mime_type", "sha256", "filename", "caption", "voice"} {
			if value, ok := media[key]; ok {
				content[key] = value
			}
		}
		content["storage_state"] = "METADATA_ONLY"
		return kind, content, ""
	case "CONTACTS", "LOCATION", "REACTION", "INTERACTIVE", "TEMPLATE":
		var fields map[string]json.RawMessage
		_ = json.Unmarshal(raw, &fields)
		// Typed subtree only; HTML is never trusted. No speculative sends from parsing.
		var value any
		if json.Unmarshal(fields[m.Type], &value) != nil {
			return "UNKNOWN", map[string]any{"provider_type": m.Type}, ""
		}
		return kind, map[string]any{"value": value}, ""
	default:
		return "UNKNOWN", map[string]any{"provider_type": m.Type}, ""
	}
}
func (s *Service) MaterializeOnce(ctx context.Context) error {
	if s.Meta == nil || !s.Meta.Config.Enabled {
		return pgx.ErrNoRows
	}
	var org, app, callback uuid.UUID
	if e := s.Auth.Pool.QueryRow(ctx, "SELECT organization_id,id,callback_key FROM app.meta_binding($1)", s.Meta.Config.AppID).Scan(&org, &app, &callback); e != nil {
		return e
	}
	tx, e := s.Auth.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer rollback(tx)
	if _, e = tx.Exec(ctx, "SELECT set_config('app.organization_id',$1,true)", org.String()); e != nil {
		return e
	}
	var eid uuid.UUID
	var ciphertext []byte
	var ingress, now time.Time
	e = tx.QueryRow(ctx, `SELECT e.id,e.raw_ciphertext,e.received_at,clock_timestamp() FROM app.webhook_events e
 WHERE e.app_id=$1 AND e.state IN ('PROCESSED','UNKNOWN') AND NOT EXISTS(SELECT FROM app.inbox_materializations m WHERE m.event_id=e.id)
 ORDER BY e.received_at,e.id FOR UPDATE OF e SKIP LOCKED LIMIT 1`, app).Scan(&eid, &ciphertext, &ingress, &now)
	if e != nil {
		return e
	}
	raw, decryptErr := s.Auth.OpenEvidence(ciphertext, org.String()+":"+eid.String()+":meta-webhook:v1")
	state, code := "MATERIALIZED", ""
	var payload envelope
	if decryptErr != nil || json.Unmarshal(raw, &payload) != nil {
		state, code = "UNAVAILABLE", "EVIDENCE_UNAVAILABLE"
	} else {
		for _, entry := range payload.Entry {
			for _, change := range entry.Changes {
				if change.Field != "messages" {
					continue
				}
				var value inboundValue
				if json.Unmarshal(change.Value, &value) != nil {
					continue
				}
				var phone uuid.UUID
				e = tx.QueryRow(ctx, "SELECT p.id FROM app.phone_numbers p JOIN app.wabas w ON w.organization_id=p.organization_id AND w.id=p.waba_id WHERE p.external_id=$1 AND w.external_id=$2 AND w.app_id=$3", value.Metadata.Phone, entry.ID, app).Scan(&phone)
				if errors.Is(e, pgx.ErrNoRows) {
					continue
				}
				if e != nil {
					return e
				}
				for _, rawMessage := range value.Messages {
					if e = s.materializeMessage(ctx, tx, org, phone, eid, rawMessage, ingress, now); e != nil {
						return e
					}
				}
				for _, rawStatus := range value.Statuses {
					if e = s.materializeStatus(ctx, tx, org, phone, eid, rawStatus); e != nil {
						return e
					}
				}
			}
		}
	}
	if _, e = tx.Exec(ctx, "INSERT INTO app.inbox_materializations(id,organization_id,event_id,state,error_code) VALUES($1,$2,$3,$4,nullif($5,''))", newID(), org, eid, state, code); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s *Service) materializeMessage(ctx context.Context, tx pgx.Tx, org, phone, eid uuid.UUID, raw json.RawMessage, ingress, now time.Time) error {
	var m inboundMessage
	if json.Unmarshal(raw, &m) != nil || len(m.ID) == 0 || len(m.ID) > 512 || !identityDigits(m.From) {
		return nil
	}
	var recipient, conv uuid.UUID
	e := tx.QueryRow(ctx, "INSERT INTO app.recipient_identities(id,organization_id,phone_id,identity_value,is_test) VALUES($1,$2,$3,$4,$5) ON CONFLICT(organization_id,phone_id,identity_value) DO UPDATE SET is_test=app.recipient_identities.is_test OR excluded.is_test RETURNING id", newID(), org, phone, m.From, s.Live.Enabled && s.Live.Recipient == m.From).Scan(&recipient)
	if e != nil {
		return e
	}
	e = tx.QueryRow(ctx, "INSERT INTO app.conversations(id,organization_id,phone_id,recipient_id,last_activity_at) VALUES($1,$2,$3,$4,$5) ON CONFLICT(organization_id,phone_id,recipient_id) DO UPDATE SET recipient_id=excluded.recipient_id RETURNING id", newID(), org, phone, recipient, ingress).Scan(&conv)
	if e != nil {
		return e
	}
	kind, content, text := typedContent(raw, m)
	provider := providerTime(m.Timestamp)
	basis := windowBasis(provider, ingress, now)
	var mid uuid.UUID
	e = tx.QueryRow(ctx, `INSERT INTO app.messages(id,organization_id,conversation_id,phone_id,provider_id,direction,message_type,content,text_body,context_provider_id,provider_at,received_at,source_event_id)
 VALUES($1,$2,$3,$4,$5,'INBOUND',$6,$7,$8,nullif($9,''),$10,$11,$12) ON CONFLICT(organization_id,phone_id,provider_id) DO NOTHING RETURNING id`, newID(), org, conv, phone, m.ID, kind, encode(content), text, m.Context.ID, provider, ingress, eid).Scan(&mid)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil
	}
	if e != nil {
		return e
	}
	activity := ingress
	if provider != nil && !provider.After(ingress) {
		activity = *provider
	}
	var sequence int64
	e = tx.QueryRow(ctx, `UPDATE app.conversations SET inbound_sequence=inbound_sequence+1,revision=revision+1,
 last_activity_at=greatest(last_activity_at,$2),last_inbound_at=greatest(last_inbound_at,$2),
 status=CASE WHEN status='SNOOZED' THEN 'OPEN' ELSE status END,snoozed_until=CASE WHEN status='SNOOZED' THEN NULL ELSE snoozed_until END
 WHERE id=$1 RETURNING inbound_sequence`, conv, activity).Scan(&sequence)
	if e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "UPDATE app.messages SET inbound_sequence=$2 WHERE id=$1", mid, sequence); e != nil {
		return e
	}
	if kind == "TEXT" && basis != nil {
		_, e = tx.Exec(ctx, `UPDATE app.conversations SET last_eligible_inbound_message_id=$2,window_basis_at=$3,
 window_opened_at=coalesce(window_opened_at,$3),window_expires_at=$3::timestamptz+interval '24 hours',
 window_confidence='AUTHENTICATED_TEXT',window_policy='META_CSW_24H_TEXT_V26'
 WHERE id=$1 AND (window_basis_at IS NULL OR window_basis_at<$3)`, conv, mid, *basis)
		if e != nil {
			return e
		}
	}
	return event(ctx, tx, org, conv, "message.created")
}
func identityDigits(v string) bool {
	if len(v) < 5 || len(v) > 32 {
		return false
	}
	for _, r := range v {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
func (s *Service) materializeStatus(ctx context.Context, tx pgx.Tx, org, phone, eid uuid.UUID, raw json.RawMessage) error {
	var v struct {
		ID, Status, Timestamp string
		Pricing               map[string]json.RawMessage
		Errors                []struct{ Code int }
	}
	if json.Unmarshal(raw, &v) != nil || v.ID == "" || len(v.ID) > 512 {
		return nil
	}
	status := strings.ToUpper(v.Status)
	switch status {
	case "SENT", "DELIVERED", "READ", "FAILED":
	default:
		status = "UNKNOWN"
	}
	pricing := map[string]any{}
	for _, key := range []string{"pricing_model", "type", "category", "billable"} {
		if b, ok := v.Pricing[key]; ok {
			var value any
			if json.Unmarshal(b, &value) == nil {
				switch value.(type) {
				case string, bool:
					pricing[key] = value
				}
			}
		}
	}
	var errorCode *int
	if len(v.Errors) > 0 {
		errorCode = &v.Errors[0].Code
	}
	facts := encode([]any{v.ID, status, v.Timestamp, pricing, errorCode})
	result, e := tx.Exec(ctx, `INSERT INTO app.message_statuses(id,organization_id,phone_id,provider_id,status,provider_at,pricing_metadata,error_code,source_event_id,fact_hash)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT(organization_id,phone_id,fact_hash) DO NOTHING`, newID(), org, phone, v.ID, status, providerTime(v.Timestamp), encode(pricing), errorCode, eid, hash(facts))
	if e != nil || result.RowsAffected() == 0 {
		return e
	}
	return correlateStatus(ctx, tx, org, phone, v.ID)
}
func correlateStatus(ctx context.Context, tx pgx.Tx, org, phone uuid.UUID, provider string) error {
	var mid, conv uuid.UUID
	var current string
	e := tx.QueryRow(ctx, "SELECT id,conversation_id,delivery_state FROM app.messages WHERE phone_id=$1 AND provider_id=$2 AND direction='OUTBOUND' FOR UPDATE", phone, provider).Scan(&mid, &conv, &current)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil
	}
	if e != nil {
		return e
	}
	r, e := tx.Query(ctx, "SELECT status FROM app.message_statuses WHERE phone_id=$1 AND provider_id=$2", phone, provider)
	if e != nil {
		return e
	}
	facts, e := pgx.CollectRows(r, pgx.RowTo[string])
	if e != nil {
		return e
	}
	state := deliveryProjection(current, facts)
	if _, e = tx.Exec(ctx, "UPDATE app.messages SET delivery_state=$2,revision=revision+1 WHERE id=$1", mid, state); e != nil {
		return e
	}
	return event(ctx, tx, org, conv, "message.status_changed")
}
func deliveryProjection(current string, facts []string) string {
	rank := map[string]int{"PENDING_LOCAL": 0, "UNCERTAIN": 0, "ACCEPTED": 1, "SENT": 2, "DELIVERED": 3, "READ": 4}
	best := current
	failed := current == "FAILED"
	unknown := current == "UNKNOWN_CONFLICT"
	for _, f := range facts {
		if f == "FAILED" {
			failed = true
		}
		if f == "UNKNOWN" {
			unknown = true
		}
		if rank[f] > rank[best] {
			best = f
		}
	}
	if unknown || (failed && rank[best] >= 3) {
		return "UNKNOWN_CONFLICT"
	}
	if failed {
		return "FAILED"
	}
	return best
}
