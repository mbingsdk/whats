package inbox

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"math/big"
	"net/http"
	"regexp"
	"sort"
	"time"
	"waba.local/control/internal/identity"
)

//go:embed evidence/gate-c.json
var gateC []byte
var decimalPattern = regexp.MustCompile("^[0-9]{1,16}(\\.[0-9]{1,8})?$")

func amount(v string) (*big.Rat, error) {
	if !decimalPattern.MatchString(v) {
		return nil, identity.ErrValidation
	}
	r, ok := new(big.Rat).SetString(v)
	if !ok {
		return nil, identity.ErrValidation
	}
	return r, nil
}

type rateRow struct {
	Market, Currency, Category string
	Rate                       *string `json:"list_rate"`
	Date                       string  `json:"effective_local_date"`
}
type rateBundle struct {
	Rows    []rateRow        `json:"rows"`
	Sources []map[string]any `json:"sources"`
}

func validateRates(raw []byte) error {
	var bundle rateBundle
	if json.Unmarshal(raw, &bundle) != nil || len(bundle.Rows) != 20 || len(bundle.Sources) != 6 {
		return Error("RATE_EVIDENCE_INVALID")
	}
	seen := map[string]bool{}
	categories := map[string]bool{"MARKETING": true, "UTILITY": true, "AUTHENTICATION": true, "AUTHENTICATION_INTERNATIONAL": true, "SERVICE": true}
	for _, row := range bundle.Rows {
		key := row.Market + ":" + row.Currency + ":" + row.Category + ":" + row.Date
		if seen[key] || row.Market != "ID" || (row.Currency != "USD" && row.Currency != "IDR") || !categories[row.Category] {
			return Error("RATE_EVIDENCE_INVALID")
		}
		seen[key] = true
		if row.Date != "2026-07-01" && row.Date != "2026-10-01" {
			return Error("RATE_INTERVAL_INVALID")
		}
		if row.Rate == nil {
			if row.Category != "SERVICE" || row.Date != "2026-07-01" {
				return Error("RATE_COVERAGE_MISSING")
			}
		} else if _, e := amount(*row.Rate); e != nil {
			return e
		}
	}
	// Every source hash, tier boundary and value must match the reviewed,
	// embedded original-derived bundle. This importer does not accept uploads
	// or silently extend July tier evidence to October.
	var actual, reviewed any
	if json.Unmarshal(raw, &actual) != nil || json.Unmarshal(gateC, &reviewed) != nil || hash(encode(actual)) != hash(encode(reviewed)) {
		return Error("RATE_SOURCE_NOT_REVIEWED")
	}
	return nil
}
func reportHash(importHash, sourceHash string, revision int64, base *uuid.UUID, validation, diff []byte, deadline time.Time) string {
	var a, b any
	_ = json.Unmarshal(validation, &a)
	_ = json.Unmarshal(diff, &b)
	return hash(encode([]any{importHash, sourceHash, revision, base, a, b, deadline.UTC().Format(time.RFC3339Nano)}))
}

// diffArtifacts compares actual canonical values, including deletions, rather
// than describing expected changes without examining the published base.
func diffArtifacts(before, after []byte) map[string]any {
	var old, newValue map[string]any
	_ = json.Unmarshal(before, &old)
	_ = json.Unmarshal(after, &newValue)
	result := map[string]any{}
	for _, key := range []string{"rows", "tiers", "sources"} {
		a, b := map[string]any{}, map[string]any{}
		index := func(value any, target map[string]any) {
			values, _ := value.([]any)
			for _, v := range values {
				row, _ := v.(map[string]any)
				id := fmt.Sprint(row["market"], ":", row["currency"], ":", row["category"], ":", row["effective_local_date"])
				if key == "sources" {
					id = fmt.Sprint(row["id"])
				}
				if key == "tiers" {
					id += ":" + fmt.Sprint(row["from"])
				}
				target[id] = row
			}
		}
		index(old[key], a)
		index(newValue[key], b)
		added, removed, changed := []string{}, []string{}, []string{}
		for id, value := range b {
			previous, ok := a[id]
			if !ok {
				added = append(added, id)
			} else if hash(encode(previous)) != hash(encode(value)) {
				changed = append(changed, id)
			}
		}
		for id := range a {
			if _, ok := b[id]; !ok {
				removed = append(removed, id)
			}
		}
		sort.Strings(added)
		sort.Strings(removed)
		sort.Strings(changed)
		result[key] = map[string]any{"added": added, "removed": removed, "changed": changed}
	}
	return result
}
func (s *Service) registry(ctx context.Context, tx pgx.Tx, v identity.Session, member uuid.UUID, r *http.Request, rt route) (any, error) {
	org := *v.OrganizationID
	key := "pricing.view"
	if rt.Path == "/rate-cards/imports" {
		key = "pricing.registry.import"
	}
	if rt.Path == "/rate-cards/imports/{id}/{action}" {
		switch r.PathValue("action") {
		case "review":
			key = "pricing.registry.review"
		case "publish":
			key = "pricing.registry.publish"
		default:
			key = "pricing.registry.import"
		}
	}
	if rt.Path == "/budgets" && r.Method == "POST" {
		key = "budgets.manage"
	}
	if e := require(ctx, tx, member, key, nil, nil); e != nil {
		return nil, e
	}
	if rt.Path == "/pricing" {
		imports, e := rows(ctx, tx, "SELECT id,state,revision,source_url,source_sha256,retrieved_at,import_hash,correction_reason,validation_report,diff_report,base_publication_id,reviewer_member_id,reviewed_at,next_review_at,created_at FROM app.rate_imports ORDER BY created_at DESC LIMIT 100")
		if e != nil {
			return nil, e
		}
		publications, e := rows(ctx, tx, "SELECT id,import_id,snapshot_hash,source_sha256,next_review_at,supersedes_publication_id,published_at FROM app.rate_publications ORDER BY published_at DESC LIMIT 100")
		if e != nil {
			return nil, e
		}
		policies, e := rows(ctx, tx, "SELECT id,version,kind,source_url,source_sha256,review_evidence,effective_from,effective_to,next_review_at,published_at FROM app.pricing_policies ORDER BY published_at DESC")
		if e != nil {
			return nil, e
		}
		settings, e := rows(ctx, tx, "SELECT billing_currency_state,billing_currency,active_rate_publication_id FROM app.inbox_settings")
		return map[string]any{"imports": imports, "publications": publications, "policies": policies, "settings": settings, "paid_authority": "CLOSED"}, e
	}
	if rt.Path == "/budgets" {
		if r.Method == "GET" {
			return rows(ctx, tx, "SELECT p.id,p.name,p.currency,p.starts_at,p.ends_at,p.hard_limit::text,p.revision,coalesce((SELECT sum(r.amount) FROM app.budget_reservations r WHERE r.period_id=p.id AND r.state<>'RELEASED'),0)::text AS exposure FROM app.budget_periods p ORDER BY starts_at DESC LIMIT 100")
		}
		var in struct {
			Name, Currency, Maximum string
			Starts, Ends            time.Time
		}
		if e := decode(r, &in); e != nil {
			return nil, e
		}
		if !validText(in.Name, 80) || !in.Ends.After(in.Starts) {
			return nil, identity.ErrValidation
		}
		if _, e := amount(in.Maximum); e != nil {
			return nil, e
		}
		var state string
		var currency *string
		if e := tx.QueryRow(ctx, "SELECT billing_currency_state,billing_currency FROM app.inbox_settings FOR SHARE").Scan(&state, &currency); e != nil {
			return nil, e
		}
		if state != "VERIFIED" || currency == nil {
			return nil, Error("BILLING_CURRENCY_UNVERIFIED")
		}
		if in.Currency != *currency {
			return nil, Error("CURRENCY_MISMATCH")
		}
		id := newID()
		_, e := tx.Exec(ctx, "INSERT INTO app.budget_periods(id,organization_id,name,currency,starts_at,ends_at,hard_limit) VALUES($1,$2,$3,$4,$5,$6,$7::numeric)", id, org, in.Name, in.Currency, in.Starts, in.Ends, in.Maximum)
		if e != nil {
			return nil, e
		}
		return map[string]any{"id": id}, audit(ctx, tx, org, v.UserID, id, "inbox.budget.created")
	}
	if rt.Path == "/rate-cards/imports" {
		var in struct {
			Source           string `json:"source"`
			CorrectionReason string `json:"correction_reason"`
		}
		if e := decode(r, &in); e != nil {
			return nil, e
		}
		if in.Source != "GATE_C_20260914" {
			return nil, Error("RATE_SOURCE_NOT_REVIEWED")
		}
		var value any
		if json.Unmarshal(gateC, &value) != nil {
			return nil, Error("RATE_EVIDENCE_INVALID")
		}
		canonical := encode(value)
		var base *uuid.UUID
		if e := tx.QueryRow(ctx, "SELECT active_rate_publication_id FROM app.inbox_settings FOR SHARE").Scan(&base); e != nil {
			return nil, e
		}
		if (base != nil && !validText(in.CorrectionReason, 1000)) || len(in.CorrectionReason) > 1000 {
			return nil, identity.ErrValidation
		}
		id := newID()
		_, e := tx.Exec(ctx, `INSERT INTO app.rate_imports(id,organization_id,importer_member_id,source_url,source_sha256,retrieved_at,artifact,import_hash,next_review_at,base_publication_id,correction_reason)
 VALUES($1,$2,$3,'https://developers.facebook.com/documentation/business-messaging/whatsapp/pricing',$4,'2026-09-14T06:50:00Z',$5,$6,'2026-09-21T00:00:00Z',$7,$8)`, id, org, member, hash(gateC), canonical, hash(canonical), base, in.CorrectionReason)
		if e != nil {
			return nil, e
		}
		return map[string]any{"id": id, "state": "DRAFT"}, audit(ctx, tx, org, v.UserID, id, "inbox.rate.imported")
	}
	id, e := pathID(r)
	if e != nil {
		return nil, e
	}
	var in struct {
		Revision int64 `json:"revision"`
		Approve  bool  `json:"approve"`
	}
	if e = decode(r, &in); e != nil {
		return nil, e
	}
	action := r.PathValue("action")
	var head *uuid.UUID
	var currencyState string
	var currency *string
	// A shared row for review, exclusive for publication: ordinary sends use SHARE.
	lock := " FOR SHARE"
	if action == "publish" {
		lock = " FOR UPDATE"
	}
	if e = tx.QueryRow(ctx, "SELECT active_rate_publication_id,billing_currency_state,billing_currency FROM app.inbox_settings"+lock).Scan(&head, &currencyState, &currency); e != nil {
		return nil, e
	}
	var state, sourceHash, importHash, correctionReason string
	var artifact, validationReport, diffReport []byte
	var revision int64
	var importer uuid.UUID
	var reviewer, base *uuid.UUID
	var reviewed *string
	var reviewAt time.Time
	e = tx.QueryRow(ctx, `SELECT state,source_sha256,import_hash,artifact,revision,importer_member_id,reviewer_member_id,base_publication_id,reviewed_hash,next_review_at,validation_report,diff_report,correction_reason FROM app.rate_imports WHERE id=$1 FOR UPDATE`, id).Scan(&state, &sourceHash, &importHash, &artifact, &revision, &importer, &reviewer, &base, &reviewed, &reviewAt, &validationReport, &diffReport, &correctionReason)
	if e == pgx.ErrNoRows {
		return nil, identity.ErrNotFound
	}
	if e != nil {
		return nil, e
	}
	if in.Revision != revision {
		return nil, identity.ErrConflict
	}
	var parsed any
	if json.Unmarshal(artifact, &parsed) != nil {
		return nil, Error("RATE_EVIDENCE_INVALID")
	}
	canonicalHash := hash(encode(parsed))
	if canonicalHash != importHash {
		return nil, Error("RATE_REVISION_HASH_MISMATCH")
	}
	fingerprint := reportHash(importHash, sourceHash+":"+correctionReason, revision, base, validationReport, diffReport, reviewAt)
	next := ""
	switch action {
	case "validate":
		if state != "DRAFT" {
			return nil, identity.ErrConflict
		}
		if e = validateRates(artifact); e != nil {
			return nil, e
		}
		_, e = tx.Exec(ctx, "UPDATE app.rate_imports SET state='VALIDATED',validation_report=$2 WHERE id=$1", id, encode(map[string]any{"valid": true, "scope": "INDEPENDENT_MULTI_CURRENCY_ARTIFACTS", "rows": 20, "hash": importHash}))
		next = "VALIDATED"
	case "diff":
		if state != "VALIDATED" {
			return nil, identity.ErrConflict
		}
		var previous []byte
		if head != nil {
			if e = tx.QueryRow(ctx, "SELECT snapshot FROM app.rate_publications WHERE id=$1", head).Scan(&previous); e != nil {
				return nil, e
			}
		}
		_, e = tx.Exec(ctx, "UPDATE app.rate_imports SET state='DIFFED',base_publication_id=$2,diff_report=$3 WHERE id=$1", id, head, encode(map[string]any{"base_publication_id": head, "candidate_hash": importHash, "scope": "FULL_SNAPSHOT", "changes": diffArtifacts(previous, artifact)}))
		next = "DIFFED"
	case "submit":
		if state != "DIFFED" {
			return nil, identity.ErrConflict
		}
		_, e = tx.Exec(ctx, "UPDATE app.rate_imports SET state='IN_REVIEW' WHERE id=$1", id)
		next = "IN_REVIEW"
	case "review":
		if state != "IN_REVIEW" || importer == member {
			return nil, identity.ErrPermission
		}
		next = "CHANGES_REQUIRED"
		if in.Approve {
			next = "APPROVED"
		}
		_, e = tx.Exec(ctx, "UPDATE app.rate_imports SET state=$2,reviewer_member_id=$3,reviewed_hash=$4,reviewed_at=now() WHERE id=$1", id, next, member, fingerprint)
	case "revise":
		if state != "CHANGES_REQUIRED" {
			return nil, identity.ErrConflict
		}
		_, e = tx.Exec(ctx, "UPDATE app.rate_imports SET state='DRAFT',revision=revision+1,validation_report=NULL,diff_report=NULL,reviewer_member_id=NULL,reviewed_hash=NULL,reviewed_at=NULL WHERE id=$1", id)
		next = "DRAFT"
	case "publish":
		if state != "APPROVED" || reviewer == nil || reviewed == nil || *reviewed != fingerprint {
			return nil, identity.ErrConflict
		}
		if e = require(ctx, tx, *reviewer, "pricing.registry.review", nil, nil); e != nil {
			return nil, Error("RATE_REVIEW_AUTHORITY_REVOKED")
		}
		var reviewerActive bool
		if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT FROM app.organization_members m JOIN app.users u ON u.id=m.user_id WHERE m.id=$1 AND m.status='ACTIVE' AND u.global_status='ACTIVE')", reviewer).Scan(&reviewerActive); e != nil {
			return nil, e
		}
		if !reviewerActive {
			return nil, Error("RATE_REVIEW_AUTHORITY_REVOKED")
		}
		if currencyState != "VERIFIED" || currency == nil {
			return nil, Error("BILLING_CURRENCY_UNVERIFIED")
		}
		if *currency != "USD" && *currency != "IDR" {
			return nil, Error("PRICING_COVERAGE_MISSING")
		}
		if (head == nil) != (base == nil) || (head != nil && *head != *base) {
			return nil, Error("STALE_RATE_BASE")
		}
		var now time.Time
		if e = tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); e != nil {
			return nil, e
		}
		if !reviewAt.After(now) {
			return nil, Error("RATE_REVIEW_OVERDUE")
		}
		publication := newID()
		if _, e = tx.Exec(ctx, "INSERT INTO app.rate_publications(id,organization_id,import_id,publisher_member_id,reviewer_member_id,snapshot,snapshot_hash,source_sha256,next_review_at,supersedes_publication_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)", publication, org, id, member, reviewer, artifact, importHash, sourceHash, reviewAt, head); e != nil {
			return nil, e
		}
		if _, e = tx.Exec(ctx, "UPDATE app.inbox_settings SET active_rate_publication_id=$1,policy_revision=policy_revision+1", publication); e != nil {
			return nil, e
		}
		_, e = tx.Exec(ctx, "UPDATE app.rate_imports SET state='PUBLISHED' WHERE id=$1", id)
		next = "PUBLISHED"
	default:
		return nil, identity.ErrNotFound
	}
	if e != nil {
		return nil, e
	}
	return map[string]any{"id": id, "state": next}, audit(ctx, tx, org, v.UserID, id, "inbox.rate."+next)
}

// reserveBudget is a foundation for a future authorized paid path; zero-cost
// replies never call it or create zero-value currency-qualified ledger entries.
func reserveBudget(ctx context.Context, tx pgx.Tx, org, period, intent uuid.UUID, requested, currency string) error {
	value, e := amount(requested)
	if e != nil || value.Sign() <= 0 {
		return identity.ErrValidation
	}
	var state string
	var actual *string
	var publication *uuid.UUID
	if e = tx.QueryRow(ctx, "SELECT billing_currency_state,billing_currency,active_rate_publication_id FROM app.inbox_settings FOR SHARE").Scan(&state, &actual, &publication); e != nil {
		return e
	}
	if state != "VERIFIED" || actual == nil {
		return Error("BILLING_CURRENCY_UNVERIFIED")
	}
	if *actual != currency {
		return Error("CURRENCY_MISMATCH")
	}
	if publication == nil {
		return Error("PRICING_COVERAGE_MISSING")
	}
	var currentPublication bool
	if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT FROM app.rate_publications WHERE id=$1 AND next_review_at>clock_timestamp())", publication).Scan(&currentPublication); e != nil {
		return e
	}
	if !currentPublication {
		return Error("RATE_REVIEW_OVERDUE")
	}
	var limit, periodCurrency string
	if e = tx.QueryRow(ctx, "SELECT hard_limit::text,currency FROM app.budget_periods WHERE id=$1 AND starts_at<=clock_timestamp() AND ends_at>clock_timestamp() FOR UPDATE", period).Scan(&limit, &periodCurrency); e != nil {
		return Error("BUDGET_UNAVAILABLE")
	}
	if periodCurrency != currency {
		return Error("CURRENCY_MISMATCH")
	}
	var oldAmount string
	e = tx.QueryRow(ctx, "SELECT amount::text FROM app.budget_reservations WHERE period_id=$1 AND intent_id=$2", period, intent).Scan(&oldAmount)
	if e == nil {
		old, _ := amount(oldAmount)
		if old.Cmp(value) != 0 {
			return Error("INTENT_SCOPE_CHANGED")
		}
		return nil
	}
	if e != pgx.ErrNoRows {
		return e
	}
	var exposure string
	if e = tx.QueryRow(ctx, "SELECT coalesce(sum(amount),0)::text FROM app.budget_reservations WHERE period_id=$1 AND state<>'RELEASED'", period).Scan(&exposure); e != nil {
		return e
	}
	current, _ := amount(exposure)
	maximum, _ := amount(limit)
	if current == nil || maximum == nil {
		return Error("BUDGET_INVALID")
	}
	if new(big.Rat).Add(current, value).Cmp(maximum) > 0 {
		return Error("BUDGET_EXCEEDED")
	}
	id := newID()
	if _, e = tx.Exec(ctx, "INSERT INTO app.budget_reservations(id,organization_id,period_id,intent_id,amount,currency,state) VALUES($1,$2,$3,$4,$5::numeric,$6,'RESERVED')", id, org, period, intent, requested, currency); e != nil {
		return e
	}
	_, e = tx.Exec(ctx, "INSERT INTO app.budget_ledger(id,organization_id,reservation_id,entry_kind,amount,evidence_kind,source_ref) VALUES($1,$2,$3,'RESERVE',$4::numeric,'RESERVED_EXPOSURE',$5)", newID(), org, id, requested, intent.String())
	return e
}
