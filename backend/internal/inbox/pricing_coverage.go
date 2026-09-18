package inbox

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"waba.local/control/internal/identity"
)

type billingEvidence struct {
	Currency   string    `json:"currency"`
	WABA       uuid.UUID `json:"waba_id"`
	Source     string    `json:"source"`
	ReviewedAt time.Time `json:"reviewed_at"`
	RecordedAt time.Time `json:"recorded_at"`
	Actor      uuid.UUID `json:"actor_user_id"`
}

// Billing attestation is an authenticated, recent-MFA operation. It activates
// only the reviewed currency fact, never a rate publication or spending budget.
func (s *Service) recordBillingCurrency(ctx context.Context, tx pgx.Tx, v identity.Session, member uuid.UUID, r *http.Request) (any, error) {
	if e := require(ctx, tx, member, "pricing.registry.publish", nil, nil); e != nil {
		return nil, e
	}
	var in struct {
		Currency   string    `json:"currency"`
		WABA       uuid.UUID `json:"waba_id"`
		Source     string    `json:"source"`
		ReviewedAt time.Time `json:"reviewed_at"`
	}
	if e := decode(r, &in); e != nil {
		return nil, e
	}
	if in.Currency != "IDR" || in.Source != "OPERATOR_REVIEWED_META_BILLING" || in.WABA == uuid.Nil || in.ReviewedAt.IsZero() {
		return nil, identity.ErrValidation
	}
	var now time.Time
	if e := tx.QueryRow(ctx, "SELECT clock_timestamp() FROM app.wabas WHERE id=$1 AND lifecycle='PRESENT'", in.WABA).Scan(&now); e != nil {
		if e == pgx.ErrNoRows {
			return nil, identity.ErrNotFound
		}
		return nil, e
	}
	if in.ReviewedAt.After(now) {
		return nil, identity.ErrValidation
	}
	evidence := billingEvidence{Currency: in.Currency, WABA: in.WABA, Source: in.Source, ReviewedAt: in.ReviewedAt, RecordedAt: now, Actor: v.UserID}
	raw := encode(evidence)
	updated, e := tx.Exec(ctx, "UPDATE app.inbox_settings SET billing_currency_state='VERIFIED',billing_currency=$1,currency_evidence=$2,policy_revision=policy_revision+1 WHERE organization_id=$3", in.Currency, string(raw), *v.OrganizationID)
	if e != nil {
		return nil, e
	}
	if updated.RowsAffected() != 1 {
		return nil, identity.ErrNotFound
	}
	if e := audit(ctx, tx, *v.OrganizationID, v.UserID, in.WABA, "inbox.billing_currency.recorded"); e != nil {
		return nil, e
	}
	return map[string]any{"billing_currency_state": "VERIFIED", "currency": in.Currency, "evidence_digest": hash(raw), "paid_authority": "CLOSED"}, nil
}

// A pricing-only preflight deliberately requires no conversation, live inbound
// or text body. It cannot issue a send token, intent, approval or reservation.
func (s *Service) servicePricingPreflight(ctx context.Context, tx pgx.Tx, v identity.Session, member uuid.UUID, r *http.Request) (any, error) {
	if e := require(ctx, tx, member, "pricing.view", nil, nil); e != nil {
		return nil, e
	}
	var in struct {
		Phone  uuid.UUID `json:"phone_id"`
		Market string    `json:"market"`
	}
	if e := decode(r, &in); e != nil {
		return nil, e
	}
	if in.Phone == uuid.Nil {
		return nil, identity.ErrValidation
	}
	if in.Market != "ID" {
		return nil, Error("PRICING_HORIZON_NOT_FULLY_COVERED")
	}
	if !identityDigits(s.Live.Recipient) || !strings.HasPrefix(s.Live.Recipient, "62") {
		return nil, Error("CONTROLLED_TEST_RECIPIENT_REQUIRED")
	}
	var waba uuid.UUID
	var zone string
	if e := tx.QueryRow(ctx, "SELECT w.id,w.timezone_id FROM app.phone_numbers p JOIN app.wabas w ON w.organization_id=p.organization_id AND w.id=p.waba_id WHERE p.id=$1 AND p.lifecycle='PRESENT' AND w.lifecycle='PRESENT'", in.Phone).Scan(&waba, &zone); e != nil {
		if e == pgx.ErrNoRows {
			return nil, identity.ErrNotFound
		}
		return nil, e
	}
	// Only the reviewed mapping for this controlled account is executable.
	if zone != "1" {
		return nil, Error("PRICING_HORIZON_NOT_FULLY_COVERED")
	}
	var state string
	var currency, proof *string
	var head *uuid.UUID
	var now time.Time
	if e := tx.QueryRow(ctx, "SELECT billing_currency_state,billing_currency,currency_evidence,active_rate_publication_id,clock_timestamp() FROM app.inbox_settings WHERE organization_id=$1 FOR SHARE", *v.OrganizationID).Scan(&state, &currency, &proof, &head, &now); e != nil {
		return nil, e
	}
	if state != "VERIFIED" || currency == nil || proof == nil {
		return nil, Error("BILLING_CURRENCY_UNVERIFIED")
	}
	var attestation billingEvidence
	if json.Unmarshal([]byte(*proof), &attestation) != nil || attestation.WABA != waba || attestation.Currency != *currency || attestation.Source != "OPERATOR_REVIEWED_META_BILLING" || attestation.ReviewedAt.IsZero() || attestation.ReviewedAt.After(now) {
		return nil, Error("BILLING_CURRENCY_UNVERIFIED")
	}
	if *currency != "IDR" || head == nil {
		return nil, Error("PRICING_HORIZON_NOT_FULLY_COVERED")
	}
	var raw []byte
	var digest string
	var review time.Time
	if e := tx.QueryRow(ctx, "SELECT snapshot,snapshot_hash,next_review_at FROM app.rate_publications WHERE id=$1", *head).Scan(&raw, &digest, &review); e != nil {
		return nil, e
	}
	var canonical any
	if json.Unmarshal(raw, &canonical) != nil || hash(encode(canonical)) != digest || validateRates(raw) != nil || !review.After(now) {
		return nil, Error("PRICING_HORIZON_NOT_FULLY_COVERED")
	}
	var bundle struct {
		Intervals []ServicePriceInterval `json:"service_intervals"`
	}
	if json.Unmarshal(raw, &bundle) != nil || len(bundle.Intervals) == 0 {
		return nil, Error("PRICING_HORIZON_NOT_FULLY_COVERED")
	}
	for i := range bundle.Intervals {
		bundle.Intervals[i].PublicationID = head.String()
		if review.Before(bundle.Intervals[i].NextReviewAt) {
			bundle.Intervals[i].NextReviewAt = review
		}
	}
	exposure, e := serviceExposure(now, now, now.Add(ServiceDeliveryTTL), in.Market, *currency, *proof, bundle.Intervals)
	if e != nil {
		return nil, Error("PRICING_HORIZON_NOT_FULLY_COVERED")
	}
	expires := now.Add(time.Minute)
	if review.Before(expires) {
		expires = review
	}
	for _, row := range bundle.Intervals {
		if row.NextReviewAt.Before(expires) {
			expires = row.NextReviewAt
		}
	}
	// Keep a newly computed horizon strictly inside finite end coverage.
	coverageEnd := bundle.Intervals[len(bundle.Intervals)-1].EndsAt.Add(-ServiceDeliveryTTL - time.Nanosecond)
	if coverageEnd.Before(expires) {
		expires = coverageEnd
	}
	return map[string]any{
		"pricing_coverage": "COMPLETE", "exposure": exposure, "server_now": now, "quote_expires_at": expires,
		"authorization_expires_at": nil, "dispatch_allowed": false, "paid_authority": "CLOSED",
		"owner_approval_required": true, "budget_proposal": map[string]any{"maximum": exposure.Maximum, "currency": exposure.Currency, "state": "PENDING_OWNER_APPROVAL", "message_count": 1, "message_type": "TEXT", "category": "SERVICE", "phone_id": in.Phone, "recipient_scope": "CONFIGURED_TEST_ONLY"},
	}, nil
}
