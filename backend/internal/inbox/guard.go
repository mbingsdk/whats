package inbox

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"net/http"
	"time"
	"waba.local/control/internal/identity"
)

type GuardInput struct {
	Now                                           time.Time
	WindowExpiry                                  *time.Time
	TrustedWindow                                 bool
	Handoff, MessageType, Category, CurrencyState string
	PolicyFrom, PolicyTo, ReviewAt                time.Time
	DeliveryLatest                                *time.Time
	SendingEnabled, Authorized, Suppressed        bool
	Assignment, ExpectedAssignment                int64
}
type Decision struct {
	Allowed                      bool   `json:"allowed"`
	Code                         string `json:"code"`
	Window                       string `json:"window"`
	Category                     string `json:"category"`
	Pricing                      string `json:"pricing"`
	CurrencyState                string `json:"billing_currency_state"`
	ConfirmationRequired         bool   `json:"confirmation_required"`
	FinancialReservationRequired bool   `json:"financial_reservation_required"`
}

func Evaluate(in GuardInput) Decision {
	d := Decision{Window: "UNKNOWN", Category: in.Category, Pricing: "UNKNOWN", CurrencyState: in.CurrencyState}
	if in.TrustedWindow && in.WindowExpiry != nil {
		switch delta := in.WindowExpiry.Sub(in.Now); {
		case delta <= 0:
			d.Window = "EXPIRED"
		case delta <= 15*time.Minute:
			d.Window = "EXPIRING_SOON"
		default:
			d.Window = "ACTIVE"
		}
	}
	block := func(code string) Decision { d.Code = code; return d }
	if !in.Authorized {
		return block("PERMISSION_DENIED")
	}
	if !in.SendingEnabled {
		return block("SENDING_DISABLED")
	}
	if in.Suppressed {
		return block("RECIPIENT_SUPPRESSED")
	}
	if in.ExpectedAssignment != in.Assignment {
		return block("ASSIGNMENT_CHANGED")
	}
	if in.Handoff == "BOT" {
		return block("HANDOFF_CONFLICT")
	}
	if in.MessageType != "TEXT" || in.Category != "SERVICE" {
		if in.CurrencyState != "VERIFIED" {
			return block("BILLING_CURRENCY_UNVERIFIED")
		}
		return block("PAID_TEMPLATE_AUTHORITY_CLOSED")
	}
	if d.Window == "UNKNOWN" {
		return block("SERVICE_WINDOW_UNKNOWN")
	}
	if d.Window == "EXPIRED" || in.WindowExpiry.Sub(in.Now) <= 5*time.Second {
		return block("TEMPLATE_REQUIRED")
	}
	if in.PolicyFrom.IsZero() || in.PolicyTo.IsZero() || in.Now.Before(in.PolicyFrom) || !in.Now.Before(in.PolicyTo) {
		return block("PRICING_COVERAGE_MISSING")
	}
	if !in.Now.Before(in.ReviewAt) {
		return block("PRICING_POLICY_REVIEW_OVERDUE")
	}
	// Zero-cost proof is policy-based, independent of the unknown currency.
	// A bounded delivery horizon must be evidenced, never supplied by the client.
	if in.DeliveryLatest == nil {
		return block("PRICING_HORIZON_NOT_FULLY_COVERED")
	}
	if _, _, e := intervalMaximum(in.Now, *in.DeliveryLatest, []ServicePriceInterval{{StartsAt: in.PolicyFrom, EndsAt: in.PolicyTo, UnitMaximum: "0", ZeroPolicy: true}}); e != nil {
		return block(string(e.(Error)))
	}
	d.Allowed = true
	d.Code = "ZERO_COST_SERVICE"
	d.Pricing = "NO_CHARGE_REVIEWED_POLICY"
	return d
}

type sendInput struct {
	Conversation  uuid.UUID `json:"conversation_id"`
	Assignment    int64     `json:"assignment_revision"`
	Text          string    `json:"text"`
	Type          string    `json:"message_type"`
	Category      string    `json:"category"`
	ClientKey     string    `json:"client_idempotency_key"`
	Authorization string    `json:"authorization"`
}

func scopeHash(v identity.Session, c Conversation, in sendInput) string {
	typ, category := in.Type, in.Category
	if typ == "" {
		typ = "TEXT"
	}
	if category == "" {
		category = "SERVICE"
	}
	return hash(encode([]any{c.Org, v.UserID, v.ID, c.Phone, c.PhoneExternal, c.WABAExternal, c.Recipient, c.Identity, c.ID, in.Text, typ, category, in.Assignment}))
}
func (s *Service) evaluate(ctx context.Context, tx pgx.Tx, v identity.Session, member uuid.UUID, c Conversation, in sendInput) (Decision, uuid.UUID, int64, time.Time, error) {
	var now, from, to, review time.Time
	var policy uuid.UUID
	var enabled bool
	var currency string
	var revision int64
	e := tx.QueryRow(ctx, "SELECT sending_enabled,billing_currency_state,policy_revision,clock_timestamp() FROM app.inbox_settings WHERE organization_id=$1 FOR SHARE", c.Org).Scan(&enabled, &currency, &revision, &now)
	if e != nil {
		return Decision{}, policy, revision, now, e
	}
	e = tx.QueryRow(ctx, "SELECT id,effective_from,effective_to,next_review_at FROM app.pricing_policies WHERE kind='SERVICE_ZERO' ORDER BY published_at DESC,id DESC LIMIT 1").Scan(&policy, &from, &to, &review)
	if e != nil && e != pgx.ErrNoRows {
		return Decision{}, policy, revision, now, e
	}
	authorized := require(ctx, tx, member, "messages.send", c.Team, c.Member) == nil && v.Verified
	typ := in.Type
	if typ == "" {
		typ = "TEXT"
	}
	category := in.Category
	if category == "" {
		category = "SERVICE"
	}
	var latest *time.Time
	if s.Live.DeliveryBound > 0 {
		bound := now.Add(s.Live.DeliveryBound)
		latest = &bound
	}
	decision := Evaluate(GuardInput{Handoff: c.Handoff, Now: now, WindowExpiry: c.Window, TrustedWindow: c.WindowConfidence == "AUTHENTICATED_TEXT", MessageType: typ, Category: category, CurrencyState: currency, PolicyFrom: from, PolicyTo: to, ReviewAt: review, DeliveryLatest: latest, SendingEnabled: enabled, Authorized: authorized, Suppressed: c.Suppressed, Assignment: c.Assignment, ExpectedAssignment: in.Assignment})
	if decision.Allowed {
		var binding bool
		if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT FROM app.phone_numbers p JOIN app.wabas w ON w.organization_id=p.organization_id AND w.id=p.waba_id JOIN app.meta_apps a ON a.organization_id=w.organization_id AND a.id=w.app_id WHERE p.id=$1 AND p.lifecycle='PRESENT' AND w.lifecycle='PRESENT' AND w.subscribed AND p.platform='CLOUD_API' AND a.external_id=$2 AND w.external_id=$3 AND p.external_id=$4 AND a.challenge_verified_at>=$5 AND a.last_webhook_at>=a.challenge_verified_at AND EXISTS(SELECT FROM app.messages m JOIN app.webhook_events e ON e.organization_id=m.organization_id AND e.id=m.source_event_id JOIN app.recipient_identities r ON r.organization_id=m.organization_id AND r.id=$7 WHERE m.id=$6 AND e.app_id=a.id AND e.received_at>=a.challenge_verified_at AND e.received_at>=$5 AND r.is_test AND r.campaign_excluded))", c.Phone, s.Meta.Config.AppID, s.Meta.Config.WABAID, s.Meta.Config.PhoneID, s.Live.AcceptanceNotBefore, c.LastEligible, c.Recipient).Scan(&binding); e != nil {
			return decision, policy, revision, now, e
		}
		if !binding {
			decision.Allowed = false
			decision.Code = "SENDER_BINDING_UNAVAILABLE"
		}
	}
	return decision, policy, revision, now, nil
}
func (s *Service) preflight(ctx context.Context, tx pgx.Tx, v identity.Session, member uuid.UUID, r *http.Request) (any, error) {
	var in sendInput
	if e := decode(r, &in); e != nil {
		return nil, e
	}
	if !validText(in.Text, 4096) {
		return nil, identity.ErrValidation
	}
	c, e := conversation(ctx, tx, in.Conversation, member, "messages.send", false)
	if e != nil {
		return nil, e
	}
	decision, policy, revision, now, e := s.evaluate(ctx, tx, v, member, c, in)
	if e != nil {
		return nil, e
	}
	result := map[string]any{"decision": decision, "server_now": now, "policy_id": policy, "window_expires_at": c.Window, "policy_basis": "Gate C reviewed effective-dated Service policy", "paid_authority": "CLOSED", "service_delivery_ttl_seconds": int64(ServiceDeliveryTTL / time.Second)}
	if !decision.Allowed {
		return result, nil
	}
	if !s.Live.Enabled || c.Identity != s.Live.Recipient {
		return map[string]any{"decision": Decision{Code: "CONTROLLED_TEST_RECIPIENT_REQUIRED", Window: decision.Window, Pricing: decision.Pricing, CurrencyState: decision.CurrencyState}}, nil
	}
	authorization := token()
	expires := now.Add(60 * time.Second)
	if c.Window.Add(-5 * time.Second).Before(expires) {
		expires = c.Window.Add(-5 * time.Second)
	}
	var windowEvidence []byte
	if e = tx.QueryRow(ctx, "SELECT jsonb_build_object('message_id',last_eligible_inbound_message_id,'basis_at',window_basis_at,'opened_at',window_opened_at,'expires_at',window_expires_at,'policy',window_policy,'confidence',window_confidence) FROM app.conversations WHERE id=$1", c.ID).Scan(&windowEvidence); e != nil {
		return nil, e
	}
	snapshot := encode(map[string]any{"organization_id": c.Org, "actor_user_id": v.UserID, "session_id": v.ID, "conversation_id": c.ID, "phone_id": c.Phone, "phone_external_id": c.PhoneExternal, "waba_external_id": c.WABAExternal, "recipient_id": c.Recipient, "recipient_namespace": "WA_ID", "recipient_value": c.Identity, "message_type": "TEXT", "category": "SERVICE", "payload_sha256": hash([]byte(in.Text))})
	_, e = tx.Exec(ctx, "INSERT INTO app.pricing_authorizations(id,organization_id,conversation_id,actor_member_id,session_id,token_digest,scope_hash,policy_id,assignment_revision,policy_revision,expires_at,scope_snapshot,window_evidence) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)", newID(), c.Org, c.ID, member, v.ID, hash([]byte(authorization)), scopeHash(v, c, in), policy, c.Assignment, revision, expires, snapshot, windowEvidence)
	if e != nil {
		return nil, e
	}
	result["authorization"] = authorization
	result["expires_at"] = expires
	return result, nil
}
func (s *Service) submit(ctx context.Context, tx pgx.Tx, v identity.Session, member uuid.UUID, r *http.Request) (any, error) {
	var in sendInput
	if e := decode(r, &in); e != nil {
		return nil, e
	}
	if !validText(in.Text, 4096) || len(in.ClientKey) < 16 || len(in.ClientKey) > 128 || len(in.Authorization) != 43 {
		return nil, identity.ErrValidation
	}
	c, e := conversation(ctx, tx, in.Conversation, member, "messages.send", true)
	if e != nil {
		return nil, e
	}
	digest := scopeHash(v, c, in)
	// The unique actor/key also protects concurrent submissions on other threads.
	var existing uuid.UUID
	var oldHash, state string
	e = tx.QueryRow(ctx, "SELECT id,scope_hash,state FROM app.outbound_intents WHERE actor_member_id=$1 AND client_key=$2", member, in.ClientKey).Scan(&existing, &oldHash, &state)
	if e == nil {
		if oldHash != digest {
			return nil, Error("INTENT_SCOPE_CHANGED")
		}
		return map[string]any{"id": existing, "state": state, "reused": true}, nil
	}
	if e != pgx.ErrNoRows {
		return nil, e
	}
	decision, policy, revision, _, e := s.evaluate(ctx, tx, v, member, c, in)
	if e != nil {
		return nil, e
	}
	if !decision.Allowed {
		return nil, Error(decision.Code)
	}
	if !s.Live.Enabled || c.Identity != s.Live.Recipient {
		return nil, Error("CONTROLLED_TEST_RECIPIENT_REQUIRED")
	}
	var aid uuid.UUID
	var checkedHash string
	var consumed *uuid.UUID
	e = tx.QueryRow(ctx, "SELECT id,scope_hash,consumed_intent_id FROM app.pricing_authorizations WHERE token_digest=$1 AND session_id=$2 AND actor_member_id=$3 AND conversation_id=$4 AND policy_id=$5 AND policy_revision=$6 AND expires_at>clock_timestamp() FOR UPDATE", hash([]byte(in.Authorization)), v.ID, member, c.ID, policy, revision).Scan(&aid, &checkedHash, &consumed)
	if e == pgx.ErrNoRows {
		return nil, Error("PRICING_AUTHORIZATION_EXPIRED")
	}
	if e != nil {
		return nil, e
	}
	if checkedHash != digest || consumed != nil {
		return nil, Error("INTENT_SCOPE_CHANGED")
	}
	iid, mid := newID(), newID()
	if _, e = tx.Exec(ctx, "INSERT INTO app.messages(id,organization_id,conversation_id,phone_id,direction,message_type,content,text_body,delivery_state) VALUES($1,$2,$3,$4,'OUTBOUND','TEXT',$5,$6,'PENDING_LOCAL')", mid, c.Org, c.ID, c.Phone, encode(map[string]string{"body": in.Text}), in.Text); e != nil {
		return nil, e
	}
	if _, e = tx.Exec(ctx, "INSERT INTO app.outbound_intents(id,organization_id,conversation_id,actor_member_id,session_id,phone_id,authorization_id,message_id,client_key,scope_hash,text_body,state) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,'QUEUED')", iid, c.Org, c.ID, member, v.ID, c.Phone, aid, mid, in.ClientKey, digest, in.Text); e != nil {
		return nil, e
	}
	if _, e = tx.Exec(ctx, "UPDATE app.pricing_authorizations SET consumed_intent_id=$2 WHERE id=$1 AND consumed_intent_id IS NULL", aid, iid); e != nil {
		return nil, e
	}
	if e = audit(ctx, tx, c.Org, v.UserID, iid, "inbox.intent.created"); e != nil {
		return nil, e
	}
	if e = event(ctx, tx, c.Org, c.ID, "message.created"); e != nil {
		return nil, e
	}
	return map[string]any{"id": iid, "state": "QUEUED", "reused": false}, nil
}
