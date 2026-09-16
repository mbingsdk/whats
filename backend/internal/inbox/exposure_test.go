package inbox

import (
	"strings"
	"testing"
	"time"
)

func exposureFixture() (time.Time, []ServicePriceInterval) {
	now := time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)
	cutoff := time.Date(2026, 10, 1, 7, 0, 0, 0, time.UTC)
	row := ServicePriceInterval{PublicationID: "synthetic-july", Market: "ID", Currency: "IDR", Category: "SERVICE", StartsAt: now.Add(-24 * time.Hour), EndsAt: cutoff, NextReviewAt: now.Add(24 * time.Hour), SourceID: "synthetic-source", SourceSHA256: strings.Repeat("a", 64), ReviewEvidence: "SYNTHETIC TEST ONLY", UnitMaximum: "0", ZeroPolicy: true}
	next := row
	next.PublicationID = "synthetic-october"
	next.StartsAt = cutoff
	next.EndsAt = now.Add(ServiceDeliveryTTL + 24*time.Hour)
	next.UnitMaximum = "100.00000001"
	next.ZeroPolicy = false
	return now, []ServicePriceInterval{row, next}
}
func TestServiceExposureCoverageAndConservativeMaximum(t *testing.T) {
	cases := []struct {
		name, want, code string
		change           func([]ServicePriceInterval) []ServicePriceInterval
	}{
		{"one publication", "100.00000001", "", func(p []ServicePriceInterval) []ServicePriceInterval { p[1].StartsAt = p[0].StartsAt; return p[1:] }},
		{"two publications and increase", "100.00000001", "", func(p []ServicePriceInterval) []ServicePriceInterval { return p }},
		{"decrease never lowers earlier exposure", "150.00000009", "", func(p []ServicePriceInterval) []ServicePriceInterval {
			p[0].UnitMaximum = "150.00000009"
			p[0].ZeroPolicy = false
			return p
		}},
		{"unordered publications", "100.00000001", "", func(p []ServicePriceInterval) []ServicePriceInterval { return []ServicePriceInterval{p[1], p[0]} }},
		{"one nanosecond gap", "", "PRICING_HORIZON_NOT_FULLY_COVERED", func(p []ServicePriceInterval) []ServicePriceInterval {
			p[1].StartsAt = p[1].StartsAt.Add(time.Nanosecond)
			return p
		}},
		{"missing future publication", "", "PRICING_HORIZON_NOT_FULLY_COVERED", func(p []ServicePriceInterval) []ServicePriceInterval { return p[:1] }},
		{"missing current publication", "", "PRICING_HORIZON_NOT_FULLY_COVERED", func(p []ServicePriceInterval) []ServicePriceInterval { return p[1:] }},
		{"overlap", "", "PRICING_INTERVAL_CONFLICT", func(p []ServicePriceInterval) []ServicePriceInterval {
			p[1].StartsAt = p[1].StartsAt.Add(-time.Nanosecond)
			return p
		}},
		{"overlap after first row covers all", "", "PRICING_INTERVAL_CONFLICT", func(p []ServicePriceInterval) []ServicePriceInterval { p[0].EndsAt = p[1].EndsAt; return p }},
		{"wrong currency", "", "PRICING_HORIZON_NOT_FULLY_COVERED", func(p []ServicePriceInterval) []ServicePriceInterval { p[1].Currency = "USD"; return p }},
		{"wrong market", "", "PRICING_HORIZON_NOT_FULLY_COVERED", func(p []ServicePriceInterval) []ServicePriceInterval { p[1].Market = "US"; return p }},
		{"wrong category", "", "PRICING_HORIZON_NOT_FULLY_COVERED", func(p []ServicePriceInterval) []ServicePriceInterval { p[1].Category = "UTILITY"; return p }},
		{"no safe end", "", "PRICING_HORIZON_NOT_FULLY_COVERED", func(p []ServicePriceInterval) []ServicePriceInterval { p[1].EndsAt = time.Time{}; return p }},
		{"expired review", "", "PRICING_HORIZON_NOT_FULLY_COVERED", func(p []ServicePriceInterval) []ServicePriceInterval { p[1].NextReviewAt = p[0].StartsAt; return p }},
		{"missing review", "", "PRICING_HORIZON_NOT_FULLY_COVERED", func(p []ServicePriceInterval) []ServicePriceInterval { p[1].ReviewEvidence = ""; return p }},
		{"invalid checksum", "", "PRICING_HORIZON_NOT_FULLY_COVERED", func(p []ServicePriceInterval) []ServicePriceInterval {
			p[1].SourceSHA256 = strings.Repeat("z", 64)
			return p
		}},
		{"zero requires policy proof", "", "PRICING_HORIZON_NOT_FULLY_COVERED", func(p []ServicePriceInterval) []ServicePriceInterval { p[0].ZeroPolicy = false; return p }},
		{"negative rate", "", "PRICING_HORIZON_NOT_FULLY_COVERED", func(p []ServicePriceInterval) []ServicePriceInterval { p[1].UnitMaximum = "-1"; return p }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			now, rows := exposureFixture()
			got, e := serviceExposure(now, now, now.Add(ServiceDeliveryTTL), "ID", "IDR", "synthetic-currency-proof", tc.change(rows))
			if tc.code != "" {
				if e != Error(tc.code) {
					t.Fatalf("got %v want %s", e, tc.code)
				}
				return
			}
			if e != nil || got.Maximum != tc.want || got.Currency != "IDR" || got.Kind != "INTERNAL_ESTIMATE" || got.AllowanceApplied || len(got.EvidenceDigest) != 64 {
				t.Fatalf("%+v %v", got, e)
			}
		})
	}
}
func TestServiceExposureCurrencyAndBoundary(t *testing.T) {
	now, rows := exposureFixture()
	end := now.Add(ServiceDeliveryTTL)
	for _, c := range []struct{ currency, proof string }{{"", "proof"}, {"IDR", ""}, {"IDR", " "}, {"idr", "proof"}} {
		if _, e := serviceExposure(now, now, end, "ID", c.currency, c.proof, rows); e != Error("BILLING_CURRENCY_UNVERIFIED") {
			t.Fatal(e)
		}
	}
	if _, e := serviceExposure(now, now, end, "", "IDR", "proof", rows); e != Error("RECIPIENT_MARKET_UNVERIFIED") {
		t.Fatal(e)
	}
	if _, e := serviceExposure(now, now, end.Add(-time.Nanosecond), "ID", "IDR", "proof", rows); e != Error("PRICING_HORIZON_NOT_FULLY_COVERED") {
		t.Fatal("short horizon", e)
	}
	rows[1].EndsAt = end
	if _, e := serviceExposure(now, now, end, "ID", "IDR", "proof", rows); e != Error("PRICING_HORIZON_NOT_FULLY_COVERED") {
		t.Fatal("exclusive end", e)
	}
	// A new period at the exact latest instant must also be priced.
	next := rows[1]
	next.StartsAt = end
	next.EndsAt = end.Add(time.Hour)
	next.PublicationID = "synthetic-third"
	next.UnitMaximum = "200"
	rows = append(rows, next)
	got, e := serviceExposure(now, now, end, "ID", "IDR", "proof", rows)
	if e != nil || got.Maximum != "200.00000000" || len(got.Publications) != 3 {
		t.Fatal(got, e)
	}
}
func TestServiceExposureConfirmationAndUnknownAllowance(t *testing.T) {
	now, rows := exposureFixture()
	quote := func(proof string) ServiceExposure {
		t.Helper()
		got, e := serviceExposure(now, now, now.Add(ServiceDeliveryTTL), "ID", "IDR", proof, rows)
		if e != nil {
			t.Fatal(e)
		}
		return got
	}
	frozen := quote("synthetic-proof")
	if frozen.AllowanceApplied || frozen.Maximum != "100.00000001" {
		t.Fatal("unproven allowance lowered maximum")
	}
	check := func(current ServiceExposure, maximum string, want error) {
		t.Helper()
		e := maximumStillApproved(current, maximum, frozen.Currency, frozen.CurrencyEvidenceDigest, frozen.EvidenceDigest)
		if e != want {
			t.Fatalf("got %v want %v", e, want)
		}
	}
	check(frozen, frozen.Maximum, nil)
	check(frozen, "100", Error("TEST_MAXIMUM_EXCEEDED"))
	check(frozen, "", Error("TEST_MAXIMUM_APPROVAL_REQUIRED"))
	check(quote("replacement-proof"), frozen.Maximum, Error("BILLING_CURRENCY_EVIDENCE_CHANGED"))
	changed := frozen
	changed.Currency = "USD"
	check(changed, frozen.Maximum, Error("BILLING_CURRENCY_EVIDENCE_CHANGED"))
	rows[1].UnitMaximum = "101"
	check(quote("synthetic-proof"), frozen.Maximum, Error("TEST_MAXIMUM_EXCEEDED"))
	rows[1].UnitMaximum = "99"
	check(quote("synthetic-proof"), frozen.Maximum, Error("PRICING_CONFIRMATION_SCOPE_CHANGED"))
}
