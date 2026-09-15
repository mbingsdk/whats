package inbox

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func baseGuard() GuardInput {
	now := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)
	expiry := now.Add(23 * time.Hour)
	delivery := now.Add(time.Minute)
	return GuardInput{Now: now, WindowExpiry: &expiry, TrustedWindow: true, MessageType: "TEXT", Category: "SERVICE", CurrencyState: "UNKNOWN", PolicyFrom: now.Add(-24 * time.Hour), PolicyTo: now.Add(24 * time.Hour), ReviewAt: now.Add(12 * time.Hour), DeliveryLatest: &delivery, SendingEnabled: true, Authorized: true, Assignment: 7, ExpectedAssignment: 7}
}
func TestGuardAuthorityAndPricingAreIndependent(t *testing.T) {
	cases := []struct {
		name, code string
		change     func(*GuardInput)
	}{
		{"unknown currency zero service", "ZERO_COST_SERVICE", func(*GuardInput) {}},
		{"unknown currency marketing", "BILLING_CURRENCY_UNVERIFIED", func(i *GuardInput) { i.MessageType = "TEMPLATE"; i.Category = "MARKETING" }},
		{"verified currency does not enable template", "PAID_TEMPLATE_AUTHORITY_CLOSED", func(i *GuardInput) { i.MessageType = "TEMPLATE"; i.CurrencyState = "VERIFIED" }},
		{"unknown window", "SERVICE_WINDOW_UNKNOWN", func(i *GuardInput) { i.TrustedWindow = false }},
		{"exact expiry", "TEMPLATE_REQUIRED", func(i *GuardInput) { i.WindowExpiry = &i.Now }},
		{"queued expiry", "TEMPLATE_REQUIRED", func(i *GuardInput) { i.Now = i.Now.Add(25 * time.Hour) }},
		{"safety margin", "TEMPLATE_REQUIRED", func(i *GuardInput) { v := i.Now.Add(5 * time.Second); i.WindowExpiry = &v }},
		{"missing policy", "PRICING_COVERAGE_MISSING", func(i *GuardInput) { i.PolicyFrom = time.Time{} }},
		{"expired policy", "PRICING_COVERAGE_MISSING", func(i *GuardInput) { i.PolicyTo = i.Now }},
		{"overdue policy", "PRICING_POLICY_REVIEW_OVERDUE", func(i *GuardInput) { i.ReviewAt = i.Now }},
		{"unknown delivery horizon", "DELIVERY_PRICING_HORIZON_UNVERIFIED", func(i *GuardInput) { i.DeliveryLatest = nil }},
		{"delivery crosses rate boundary", "PRICING_DELIVERY_COVERAGE_MISSING", func(i *GuardInput) { i.DeliveryLatest = &i.PolicyTo }},
		{"permission revoked", "PERMISSION_DENIED", func(i *GuardInput) { i.Authorized = false }},
		{"suppressed", "RECIPIENT_SUPPRESSED", func(i *GuardInput) { i.Suppressed = true }},
		{"sending disabled", "SENDING_DISABLED", func(i *GuardInput) { i.SendingEnabled = false }},
		{"assignment changed", "ASSIGNMENT_CHANGED", func(i *GuardInput) { i.Assignment++ }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := baseGuard()
			tc.change(&in)
			d := Evaluate(in)
			if d.Code != tc.code {
				t.Fatalf("%+v", d)
			}
			if d.Allowed != (tc.code == "ZERO_COST_SERVICE") {
				t.Fatal("wrong authority")
			}
			if d.Allowed && (d.ConfirmationRequired || d.FinancialReservationRequired) {
				t.Fatal("zero-cost ceremony/ledger")
			}
		})
	}
}
func TestOriginalWindowTimeAndDeliveryProjection(t *testing.T) {
	now := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)
	old := now.Add(-25 * time.Hour)
	future := now.Add(time.Minute)
	if got := windowBasis(&old, old.Add(time.Second), now); got == nil || !got.Equal(old) {
		t.Fatal("replay changed source timestamp")
	}
	if windowBasis(&future, now, now) != nil || windowBasis(nil, now, now) != nil {
		t.Fatal("invalid evidence opened window")
	}
	checks := []struct {
		before string
		facts  []string
		want   string
	}{
		{"ACCEPTED", []string{"READ", "SENT", "DELIVERED", "SENT"}, "READ"},
		{"READ", []string{"SENT"}, "READ"}, {"ACCEPTED", []string{"FAILED"}, "FAILED"},
		{"READ", []string{"FAILED"}, "UNKNOWN_CONFLICT"}, {"UNCERTAIN", nil, "UNCERTAIN"},
		{"SENT", []string{"UNKNOWN"}, "UNKNOWN_CONFLICT"},
	}
	for _, c := range checks {
		if got := deliveryProjection(c.before, c.facts); got != c.want {
			t.Fatalf("%s -> %s", c.before, got)
		}
	}
}
func TestOfficialRateBundleAndDecimalValues(t *testing.T) {
	if e := validateRates(gateC); e != nil {
		t.Fatal(e)
	}
	for _, s := range []string{"NaN", "1e4", "-1", "1.000000001", ""} {
		if _, e := amount(s); e == nil {
			t.Fatalf("accepted %q", s)
		}
	}
	a, _ := amount("0.0250")
	b, _ := amount("0.0500")
	if a.Add(a, a).Cmp(b) != 0 {
		t.Fatal("decimal precision")
	}
}
func TestLiveConfigNeverInfersRecipientOrDeliveryEvidence(t *testing.T) {
	c, e := LoadLive(func(string) string { return "" })
	if e != nil || c.Enabled || c.Recipient != "" || c.DeliveryBound != 0 {
		t.Fatal("unsafe default")
	}
	if _, e = LoadLive(func(k string) string {
		if k == "INBOX_LIVE_ACCEPTANCE_ENABLED" {
			return "true"
		}
		return ""
	}); e == nil {
		t.Fatal("missing explicit recipient")
	}
}

func TestTypedMessagesAndActualDiff(t *testing.T) {
	for _, kind := range []string{"text", "image", "video", "audio", "document", "sticker", "contacts", "location", "reaction", "interactive", "template", "unknown"} {
		value := map[string]any{"id": "synthetic", "from": "6280000000000", "type": kind, kind: map[string]any{"body": "text", "id": "media", "latitude": 1, "longitude": 2, "emoji": "ok"}}
		raw := encode(value)
		var parsed inboundMessage
		if e := json.Unmarshal(raw, &parsed); e != nil {
			t.Fatal(e)
		}
		got, _, _ := typedContent(raw, parsed)
		if got != strings.ToUpper(kind) {
			t.Fatal(kind, got)
		}
	}
	diff := diffArtifacts(gateC, gateC)
	for _, area := range []string{"rows", "tiers", "sources"} {
		for _, kind := range []string{"added", "removed", "changed"} {
			if len(diff[area].(map[string]any)[kind].([]string)) != 0 {
				t.Fatal("unchanged diff", diff)
			}
		}
	}
	changed := bytes.Replace(gateC, []byte("0.0411"), []byte("0.0412"), 1)
	if validateRates(changed) == nil {
		t.Fatal("unreviewed rate accepted")
	}
	if len(diffArtifacts(gateC, changed)["rows"].(map[string]any)["changed"].([]string)) != 1 {
		t.Fatal("actual rate diff missing")
	}
	in := baseGuard()
	in.Handoff = "BOT"
	if Evaluate(in).Code != "HANDOFF_CONFLICT" {
		t.Fatal("bot handoff allowed")
	}
}

func TestM73ThirtyDayTTLAndOctoberCoverage(t *testing.T) {
	// M73: the general Service messages guide documents a 30-day delivery TTL.
	// This hypothetical deadline tests coverage only; it is not a runtime
	// acceptance-to-expiry mapping, provider observation or permission to send.
	cutoff := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	review := time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)
	ttl := 30 * 24 * time.Hour
	cases := []struct {
		name     string
		now      time.Time
		policyTo time.Time
		currency string
		want     string
	}{
		{"whole horizon before conservative cutoff", cutoff.Add(-ttl - time.Nanosecond), cutoff, "UNKNOWN", "ZERO_COST_SERVICE"},
		{"delivery exactly at cutoff is not covered", cutoff.Add(-ttl), cutoff, "UNKNOWN", "PRICING_DELIVERY_COVERAGE_MISSING"},
		{"delivery beyond cutoff is not covered", cutoff.Add(-ttl + time.Second), cutoff, "UNKNOWN", "PRICING_DELIVERY_COVERAGE_MISSING"},
		{"September acceptance can cross October", time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC), cutoff, "UNKNOWN", "PRICING_DELIVERY_COVERAGE_MISSING"},
		{"known currency cannot fill future policy gap", time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC), cutoff, "VERIFIED", "PRICING_DELIVERY_COVERAGE_MISSING"},
		{"policy expires before complete provider period", time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC), time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), "UNKNOWN", "PRICING_DELIVERY_COVERAGE_MISSING"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := baseGuard()
			in.Now = tc.now
			expiry := tc.now.Add(23 * time.Hour)
			latest := tc.now.Add(ttl)
			in.WindowExpiry = &expiry
			in.PolicyFrom, in.PolicyTo, in.ReviewAt = start, tc.policyTo, review
			in.DeliveryLatest = &latest
			in.CurrencyState = tc.currency
			got := Evaluate(in)
			if got.Code != tc.want || got.Allowed != (tc.want == "ZERO_COST_SERVICE") {
				t.Fatalf("unexpected coverage decision: %+v", got)
			}
			if got.Allowed && (got.FinancialReservationRequired || got.ConfirmationRequired) {
				t.Fatal("zero policy created monetary authority")
			}
		})
	}
}

func TestM73ResearchDoesNotEnableLiveDeliveryAssumption(t *testing.T) {
	c, err := LoadLive(func(string) string { return "" })
	if err != nil || c.Enabled || c.DeliveryBound != 0 {
		t.Fatal("research enabled runtime dispatch")
	}
	in := baseGuard()
	in.Now = time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)
	expiry := in.Now.Add(time.Hour)
	in.WindowExpiry = &expiry
	in.PolicyFrom = in.Now.Add(-time.Hour)
	in.PolicyTo = time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	in.ReviewAt = time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)
	in.DeliveryLatest = nil
	if got := Evaluate(in); got.Allowed || got.Code != "DELIVERY_PRICING_HORIZON_UNVERIFIED" {
		t.Fatalf("missing runtime evidence stopped failing closed: %+v", got)
	}
}
