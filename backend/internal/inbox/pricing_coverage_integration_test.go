//go:build integration

package inbox

import (
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestPublishedServiceCoveragePreflightNeverAuthorizesSpend(t *testing.T) {
	h := newHarness(t)
	var waba uuid.UUID
	if e := h.db.QueryRow(h.ctx, "SELECT waba_id FROM app.phone_numbers WHERE id=$1", h.phone).Scan(&waba); e != nil {
		t.Fatal(e)
	}
	attestation := map[string]any{"waba_id": waba, "currency": "IDR", "source": "OPERATOR_REVIEWED_META_BILLING", "reviewed_at": time.Now().Add(-time.Minute)}
	h.expect(t, "PUT", "/pricing/billing-currency", attestation, 403)
	h.mfa(t, h.user)
	h.expect(t, "PUT", "/pricing/billing-currency", attestation, 200)
	input := map[string]any{"phone_id": h.phone, "market": "ID"}
	h.expect(t, "POST", "/pricing/service-exposure", input, 409)
	reviewer, _, reviewUser := h.secondActor(t, nil, "pricing.registry.review", "pricing.view")
	h.mfa(t, reviewUser)
	owner := h.owner
	imported := h.expect(t, "POST", "/rate-cards/imports", map[string]any{"source": "SPRINT3_IDR_SERVICE_20260918"}, 200)
	id := imported["id"].(string)
	// Only the isolated fixture changes the import deadline. The pinned interval
	// evidence keeps its actual deadline and must fail closed after it passes.
	h.sql(t, "UPDATE app.rate_imports SET next_review_at=now()+interval '1 day' WHERE id=$1", id)
	for _, action := range []string{"validate", "diff", "submit"} {
		h.expect(t, "POST", "/rate-cards/imports/"+id+"/"+action, map[string]any{"revision": 1}, 200)
	}
	h.expect(t, "POST", "/rate-cards/imports/"+id+"/review", map[string]any{"revision": 1, "approve": true}, 403)
	h.owner = reviewer
	h.expect(t, "POST", "/rate-cards/imports/"+id+"/review", map[string]any{"revision": 1, "approve": true}, 200)
	h.expect(t, "POST", "/rate-cards/imports/"+id+"/publish", map[string]any{"revision": 1}, 403)
	h.expect(t, "PUT", "/pricing/billing-currency", attestation, 403)
	h.owner = owner
	h.expect(t, "POST", "/rate-cards/imports/"+id+"/publish", map[string]any{"revision": 1}, 200)
	now := time.Now()
	eligible := !now.Before(time.Date(2026, 7, 1, 7, 0, 0, 0, time.UTC)) && now.Before(time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC))
	if eligible {
		result := h.expect(t, "POST", "/pricing/service-exposure", input, 200)
		exposure := result["exposure"].(map[string]any)
		if result["pricing_coverage"] != "COMPLETE" || result["dispatch_allowed"] != false || result["owner_approval_required"] != true || exposure["maximum"] != "356.65000000" || exposure["currency"] != "IDR" || exposure["allowance_applied"] != false || result["authorization_expires_at"] != nil {
			t.Fatal(result)
		}
		if result["budget_proposal"].(map[string]any)["state"] != "PENDING_OWNER_APPROVAL" {
			t.Fatal(result)
		}
	} else {
		result := h.expect(t, "POST", "/pricing/service-exposure", input, 409)
		if result["error"].(map[string]any)["code"] != "PRICING_HORIZON_NOT_FULLY_COVERED" {
			t.Fatal(result)
		}
	}
	h.expect(t, "POST", "/pricing/service-exposure", map[string]any{"phone_id": h.phone, "market": "US"}, 409)
	h.expect(t, "POST", "/pricing/service-exposure", map[string]any{"phone_id": newID(), "market": "ID"}, 404)
	h.expect(t, "PUT", "/pricing/billing-currency", map[string]any{"waba_id": newID(), "currency": "IDR", "source": "OPERATOR_REVIEWED_META_BILLING", "reviewed_at": time.Now().Add(-time.Minute)}, 404)
	h.expect(t, "POST", "/pricing/service-exposure", map[string]any{"phone_id": h.phone, "market": "ID", "remaining_allowance": 1000}, 422)
	h.sql(t, "UPDATE app.inbox_settings SET billing_currency='USD'")
	h.expect(t, "POST", "/pricing/service-exposure", input, 409)
	var writes int
	if e := h.db.QueryRow(h.ctx, "SELECT (SELECT count(*) FROM app.budget_periods)+(SELECT count(*) FROM app.budget_reservations)+(SELECT count(*) FROM app.budget_ledger)+(SELECT count(*) FROM app.outbound_intents)+(SELECT count(*) FROM app.pricing_authorizations)+(SELECT count(*) FROM app.send_attempts)").Scan(&writes); e != nil || writes != 0 || h.sender.calls.Load() != 0 {
		t.Fatal("pricing created spend authority", writes, e, h.sender.calls.Load())
	}
}
