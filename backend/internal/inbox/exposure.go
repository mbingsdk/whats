package inbox

import (
	"encoding/hex"
	"math/big"
	"sort"
	"strings"
	"time"
)

// ServiceDeliveryTTL is the ordinary Service TTL established by M73.
// It is not a configurable operator bypass or a terminal-webhook deadline.
const ServiceDeliveryTTL = 30 * 24 * time.Hour

// ServicePriceInterval represents reviewed executable evidence. An interval
// must be resolved from an immutable Registry publication, never client input.
// EndsAt is exclusive; missing ends are not unlimited future coverage.
type ServicePriceInterval struct {
	PublicationID  string    `json:"publication_id"`
	Market         string    `json:"market"`
	Currency       string    `json:"currency"`
	Category       string    `json:"category"`
	StartsAt       time.Time `json:"effective_from"`
	EndsAt         time.Time `json:"effective_to"`
	NextReviewAt   time.Time `json:"next_review_at"`
	SourceID       string    `json:"source_id"`
	SourceSHA256   string    `json:"source_sha256"`
	ReviewEvidence string    `json:"review_evidence"`
	UnitMaximum    string    `json:"unit_maximum"`
	ZeroPolicy     bool      `json:"zero_policy"`
}
type ServiceExposure struct {
	Currency               string    `json:"currency"`
	CurrencyEvidenceDigest string    `json:"currency_evidence_digest"`
	Market                 string    `json:"market"`
	StartsAt               time.Time `json:"delivery_interval_start"`
	EndsAt                 time.Time `json:"delivery_interval_end"`
	Maximum                string    `json:"maximum"`
	Publications           []string  `json:"publication_ids"`
	EvidenceDigest         string    `json:"evidence_digest"`
	Kind                   string    `json:"evidence_kind"`
	AllowanceApplied       bool      `json:"allowance_applied"`
}

// intervalMaximum is also used by the existing zero-policy dispatch gate.
// Callers supply only relevant, reviewed intervals for a single scope. The
// requested latest instant is included conservatively, while price ends are
// exclusive. All overlapping rows are checked, even after coverage is reached.
func intervalMaximum(start, end time.Time, intervals []ServicePriceInterval) (*big.Rat, []ServicePriceInterval, error) {
	if start.IsZero() || end.Before(start) {
		return nil, nil, Error("PRICING_HORIZON_NOT_FULLY_COVERED")
	}
	selected := []ServicePriceInterval{}
	for _, p := range intervals {
		if p.StartsAt.IsZero() || !p.EndsAt.After(p.StartsAt) {
			return nil, nil, Error("PRICING_HORIZON_NOT_FULLY_COVERED")
		}
		if p.EndsAt.After(start) && !p.StartsAt.After(end) {
			selected = append(selected, p)
		}
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].StartsAt.Before(selected[j].StartsAt) })
	cursor := start
	maximum := new(big.Rat)
	for i, p := range selected {
		if p.StartsAt.After(cursor) {
			return nil, nil, Error("PRICING_HORIZON_NOT_FULLY_COVERED")
		}
		if i > 0 && p.StartsAt.Before(selected[i-1].EndsAt) {
			return nil, nil, Error("PRICING_INTERVAL_CONFLICT")
		}
		value, e := amount(p.UnitMaximum)
		if e != nil || (value.Sign() == 0 && !p.ZeroPolicy) {
			return nil, nil, Error("PRICING_HORIZON_NOT_FULLY_COVERED")
		}
		if p.ZeroPolicy && value.Sign() != 0 {
			return nil, nil, Error("PRICING_INTERVAL_CONFLICT")
		}
		if value.Cmp(maximum) > 0 {
			maximum.Set(value)
		}
		cursor = p.EndsAt
	}
	if !cursor.After(end) {
		return nil, nil, Error("PRICING_HORIZON_NOT_FULLY_COVERED")
	}
	return maximum, selected, nil
}

// serviceExposure calculates a conservative INTERNAL_ESTIMATE, not dispatch
// authority. A caller must establish the send-time basis and immutable Registry
// provenance before supplying evidence. No allowance or discount is subtracted.
func serviceExposure(now, start, end time.Time, market, currency, currencyEvidence string, intervals []ServicePriceInterval) (ServiceExposure, error) {
	out := ServiceExposure{Currency: currency, Market: market, StartsAt: start, EndsAt: end, Kind: "INTERNAL_ESTIMATE"}
	if strings.TrimSpace(currencyEvidence) == "" || len(currency) != 3 || currency[0] < 'A' || currency[0] > 'Z' || currency[1] < 'A' || currency[1] > 'Z' || currency[2] < 'A' || currency[2] > 'Z' {
		return out, Error("BILLING_CURRENCY_UNVERIFIED")
	}
	out.CurrencyEvidenceDigest = hash([]byte(currencyEvidence))
	if strings.TrimSpace(market) == "" {
		return out, Error("RECIPIENT_MARKET_UNVERIFIED")
	}
	if start.IsZero() || end.Sub(start) < ServiceDeliveryTTL {
		return out, Error("PRICING_HORIZON_NOT_FULLY_COVERED")
	}
	relevant := []ServicePriceInterval{}
	for _, p := range intervals {
		if p.Market != market || p.Currency != currency || p.Category != "SERVICE" {
			continue
		}
		if p.StartsAt.IsZero() || !p.EndsAt.After(p.StartsAt) {
			return out, Error("PRICING_HORIZON_NOT_FULLY_COVERED")
		}
		if !p.EndsAt.After(start) || p.StartsAt.After(end) {
			continue
		}
		digest, e := hex.DecodeString(p.SourceSHA256)
		if p.PublicationID == "" || p.SourceID == "" || e != nil || len(digest) != 32 || strings.TrimSpace(p.ReviewEvidence) == "" || !p.NextReviewAt.After(now) {
			return out, Error("PRICING_HORIZON_NOT_FULLY_COVERED")
		}
		relevant = append(relevant, p)
	}
	maximum, selected, e := intervalMaximum(start, end, relevant)
	if e != nil {
		return out, e
	}
	out.Maximum = maximum.FloatString(8)
	pubs := map[string]bool{}
	for _, p := range selected {
		pubs[p.PublicationID] = true
	}
	for id := range pubs {
		out.Publications = append(out.Publications, id)
	}
	sort.Strings(out.Publications)
	out.EvidenceDigest = hash(encode([]any{market, currency, out.CurrencyEvidenceDigest, start.UTC(), end.UTC(), selected}))
	return out, nil
}

// maximumStillApproved compares frozen evidence; it does not record human
// approval, create a budget or enable the currently closed paid dispatch path.
func maximumStillApproved(current ServiceExposure, approvedMaximum, approvedCurrency, approvedCurrencyEvidence, approvedEvidence string) error {
	if current.Currency != approvedCurrency || current.CurrencyEvidenceDigest != approvedCurrencyEvidence {
		return Error("BILLING_CURRENCY_EVIDENCE_CHANGED")
	}
	approved, e := amount(approvedMaximum)
	if e != nil {
		return Error("TEST_MAXIMUM_APPROVAL_REQUIRED")
	}
	value, e := amount(current.Maximum)
	if e != nil || current.EvidenceDigest == "" {
		return Error("PRICING_HORIZON_NOT_FULLY_COVERED")
	}
	if value.Cmp(approved) > 0 {
		return Error("TEST_MAXIMUM_EXCEEDED")
	}
	if current.EvidenceDigest != approvedEvidence {
		return Error("PRICING_CONFIRMATION_SCOPE_CHANGED")
	}
	return nil
}
