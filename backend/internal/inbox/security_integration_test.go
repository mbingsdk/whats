//go:build integration

package inbox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
	"waba.local/control/internal/identity"
)

func (h *harness) secondActor(t *testing.T, team *uuid.UUID, permissions ...string) (identity.Result, uuid.UUID, uuid.UUID) {
	t.Helper()
	uid, mid, role := newID(), newID(), newID()
	email := uid.String() + "@example.invalid"
	h.sql(t, "INSERT INTO app.users(id,canonical_email,display_name,verified_at) VALUES($1,$2,'Synthetic colleague',now())", uid, email)
	h.sql(t, "INSERT INTO app.password_identities(user_id,password_hash) SELECT $1,password_hash FROM app.password_identities WHERE user_id=$2", uid, h.user)
	h.sql(t, "INSERT INTO app.organization_members(id,organization_id,user_id) VALUES($1,$2,$3)", mid, h.org, uid)
	h.sql(t, "INSERT INTO app.roles(id,organization_id,name) VALUES($1,$2,$3)", role, h.org, "Synthetic "+role.String())
	for _, key := range permissions {
		h.sql(t, "INSERT INTO app.role_permissions(id,organization_id,role_id,permission_key) VALUES($1,$2,$3,$4)", newID(), h.org, role, key)
	}
	scope := "ORG"
	if team != nil {
		scope = "TEAM"
		h.sql(t, "INSERT INTO app.team_members(id,organization_id,team_id,member_id) VALUES($1,$2,$3,$4)", newID(), h.org, team, mid)
	}
	h.sql(t, "INSERT INTO app.member_roles(id,organization_id,member_id,role_id,scope_kind,team_id) VALUES($1,$2,$3,$4,$5,$6)", newID(), h.org, mid, role, scope, team)
	result, e := h.s.Auth.Run(h.ctx, "public.login", "", identity.Input{Email: email, Password: "synthetic-password-42"}, identity.Meta{IP: uid.String()})
	if e != nil {
		t.Fatal(e)
	}
	return result, mid, uid
}
func (h *harness) mfa(t *testing.T, user uuid.UUID) {
	h.sql(t, "INSERT INTO app.auth_factors(user_id,secret_ciphertext,confirmed_at) VALUES($1,$2,now()) ON CONFLICT(user_id) DO NOTHING", user, []byte("synthetic-test-only"))
	h.sql(t, "UPDATE app.sessions SET mfa_at=now(),reauth_at=now() WHERE user_id=$1", user)
}
func TestInboxTenantRLSAllTablesAndCompositeFK(t *testing.T) {
	h := newHarness(t)
	c := h.ingest(t, "wamid.rls", "text", time.Now().Add(-time.Minute))
	h.expect(t, "POST", "/conversations/"+c.String()+"/read", map[string]any{"watermark": 1}, 200)
	h.expect(t, "POST", "/conversations/"+c.String()+"/notes", map[string]any{"body": "Synthetic RLS note", "mentions": []uuid.UUID{h.member}}, 200)
	h.expect(t, "POST", "/conversations/"+c.String()+"/presence", map[string]any{}, 200)
	h.submit(t, c, "synthetic-rls-intent", "Synthetic RLS reply")
	if e := h.s.DispatchOnce(h.ctx); e != nil {
		t.Fatal(e)
	}
	h.mfa(t, h.user)
	imported := h.expect(t, "POST", "/rate-cards/imports", map[string]any{"source": "GATE_C_20260914"}, 200)["id"].(string)
	publication, period, reservation := newID(), newID(), newID()
	h.sql(t, "INSERT INTO app.rate_publications(id,organization_id,import_id,publisher_member_id,reviewer_member_id,snapshot,snapshot_hash,source_sha256,next_review_at) VALUES($1,$2,$3,$4,$4,'{}',repeat('0',64),repeat('0',64),now()+interval '1 day')", publication, h.org, imported, h.member)
	h.sql(t, "INSERT INTO app.budget_periods(id,organization_id,name,currency,starts_at,ends_at,hard_limit) VALUES($1,$2,'Synthetic RLS period','USD',now(),now()+interval '1 day',1)", period, h.org)
	h.sql(t, "INSERT INTO app.budget_reservations(id,organization_id,period_id,intent_id,amount,currency,state) SELECT $1,organization_id,$2,id,1,'USD','RESERVED' FROM app.outbound_intents", reservation, period)
	h.sql(t, "INSERT INTO app.budget_ledger(id,organization_id,reservation_id,entry_kind,amount,evidence_kind,source_ref) VALUES($1,$2,$3,'RESERVE',1,'RESERVED_EXPOSURE','SYNTHETIC_RLS_ONLY')", newID(), h.org, reservation)
	var source uuid.UUID
	if e := h.db.QueryRow(h.ctx, "SELECT id FROM app.webhook_events LIMIT 1").Scan(&source); e != nil {
		t.Fatal(e)
	}
	if e := h.s.workerScope(h.ctx, func(tx pgx.Tx, org uuid.UUID) error {
		return h.s.materializeStatus(h.ctx, tx, org, h.phone, source, encode(map[string]any{"id": "wamid.synthetic-outbound", "status": "sent", "timestamp": "1700000000"}))
	}); e != nil {
		t.Fatal(e)
	}
	other := newID()
	h.sql(t, "INSERT INTO app.organizations(id,name,slug,timezone) VALUES($1,'Other tenant',$2,'UTC')", other, other.String())
	names := []string{"inbox_settings", "recipient_identities", "conversations", "conversation_reads", "messages", "message_statuses", "inbox_materializations", "internal_notes", "note_mentions", "inbox_presence", "inbox_events", "pricing_policies", "rate_imports", "rate_publications", "pricing_authorizations", "outbound_intents", "send_attempts", "test_send_slots", "budget_periods", "budget_reservations", "budget_ledger"}
	for _, pool := range []*pgxpool.Pool{h.s.Auth.Pool, h.s.Meta.Pool} {
		for _, name := range names {
			var seeded int
			var payload []byte
			if e := h.db.QueryRow(h.ctx, "SELECT count(*) FROM app."+name+" WHERE organization_id=$1", h.org).Scan(&seeded); e != nil || seeded == 0 {
				t.Fatal("empty RLS fixture", name, e)
			}
			if e := h.db.QueryRow(h.ctx, "SELECT row_to_json(t) FROM app."+name+" t WHERE organization_id=$1 LIMIT 1", h.org).Scan(&payload); e != nil {
				t.Fatal(e)
			}
			var row map[string]any
			if e := json.Unmarshal(payload, &row); e != nil {
				t.Fatal(e)
			}
			row["organization_id"] = other.String()
			var force, rls bool
			if e := h.db.QueryRow(h.ctx, "SELECT relforcerowsecurity,relrowsecurity FROM pg_class WHERE oid=$1::regclass", "app."+name).Scan(&force, &rls); e != nil || !force || !rls {
				t.Fatal(name, e)
			}
			tx, e := pool.Begin(h.ctx)
			if e != nil {
				t.Fatal(e)
			}
			var count int
			e = tx.QueryRow(h.ctx, "SELECT count(*) FROM app."+name).Scan(&count)
			if e != nil || count != 0 {
				rollback(tx)
				t.Fatal("no-context exposure", name, count, e)
			}
			_, e = tx.Exec(h.ctx, "SELECT set_config('app.organization_id',$1,true)", other.String())
			if e != nil {
				t.Fatal(e)
			}
			e = tx.QueryRow(h.ctx, "SELECT count(*) FROM app."+name+" WHERE organization_id=$1", h.org).Scan(&count)
			if e != nil || count != 0 {
				rollback(tx)
				t.Fatal("cross-tenant read", name, count, e)
			}
			if _, e = tx.Exec(h.ctx, "SELECT set_config('app.organization_id',$1,true)", h.org.String()); e != nil {
				t.Fatal(e)
			}
			_, e = tx.Exec(h.ctx, "INSERT INTO app."+name+" SELECT (jsonb_populate_record(NULL::app."+name+",$1::jsonb)).*", encode(row))
			var pgerr *pgconn.PgError
			if !errors.As(e, &pgerr) || pgerr.Code != "42501" {
				rollback(tx)
				t.Fatal("cross-tenant INSERT not stopped by RLS", name, e)
			}
			rollback(tx)
		}
	}
	tx, e := h.s.Auth.Pool.Begin(h.ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer rollback(tx)
	_, _ = tx.Exec(h.ctx, "SELECT set_config('app.organization_id',$1,true)", other.String())
	_, e = tx.Exec(h.ctx, "INSERT INTO app.internal_notes(id,organization_id,conversation_id,author_member_id,body) VALUES($1,$2,$3,$4,'synthetic')", newID(), other, c, h.member)
	if e == nil {
		t.Fatal("cross-tenant FK accepted")
	}
	// The runtime role is also unable to read another tenant.

}
func TestInboxTeamScopeCSRFRevocationAndNoHeaderShortcut(t *testing.T) {
	h := newHarness(t)
	c := h.ingest(t, "wamid.scope", "text", time.Now().Add(-time.Minute))
	team := newID()
	h.sql(t, "INSERT INTO app.teams(id,organization_id,name) VALUES($1,$2,'Permitted team')", team, h.org)
	actor, _, _ := h.secondActor(t, &team, "inbox.view", "messages.send", "notes.write", "organization.view")
	owner := h.owner
	h.owner = actor
	h.expect(t, "GET", "/conversations/"+c.String(), nil, 404)
	page := h.expect(t, "GET", "/conversations?q=synthetic", nil, 200)
	if len(page["items"].([]any)) != 0 {
		t.Fatal("search leak")
	}
	h.sql(t, "UPDATE app.conversations SET assigned_team_id=$2 WHERE id=$1", c, team)
	h.expect(t, "GET", "/conversations/"+c.String(), nil, 200)
	h.expect(t, "POST", "/conversations/"+c.String()+"/notes", map[string]any{"body": "internal only"}, 200)
	if h.sender.calls.Load() != 0 {
		t.Fatal("note dispatched")
	}
	req := httptest.NewRequest("POST", "/api/v1/conversations/"+c.String()+"/notes", bytes.NewBufferString(`{"body":"forged"}`))
	req.Header.Set("Origin", h.s.Auth.Origin)
	req.AddCookie(&http.Cookie{Name: "waba_session", Value: actor.Cookie})
	w := httptest.NewRecorder()
	h.handler.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Fatal("missing CSRF", w.Code)
	}
	req = httptest.NewRequest("GET", "/api/v1/conversations/"+c.String(), nil)
	req.Header.Set("X-User-ID", h.user.String())
	req.Header.Set("X-Organization-ID", h.org.String())
	w = httptest.NewRecorder()
	h.handler.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatal("header shortcut", w.Code)
	}
	h.sql(t, "UPDATE app.sessions SET revoked_at=now() WHERE user_id<>(SELECT user_id FROM app.identity_bootstrap)")
	h.expect(t, "GET", "/conversations/"+c.String(), nil, 401)
	h.owner = owner
}
func TestDispatchFinalChecksAndRecovery(t *testing.T) {
	for _, tc := range []struct{ name, query, code string }{
		{"policy", "UPDATE app.inbox_settings SET policy_revision=policy_revision+1", "PRICING_POLICY_CHANGED"},
		{"switch", "UPDATE app.inbox_settings SET sending_enabled=false", "SENDING_DISABLED"},
		{"suppression", "UPDATE app.recipient_identities SET suppressed=true", "RECIPIENT_SUPPRESSED"},
		{"window", "UPDATE app.conversations SET window_expires_at=now()-interval '1 second'", "TEMPLATE_REQUIRED"},
		{"authorization", "UPDATE app.pricing_authorizations SET expires_at=now()-interval '1 second'", "PRICING_AUTHORIZATION_EXPIRED"},
		{"session", "UPDATE app.sessions SET revoked_at=now()", "ACTOR_AUTHORIZATION_REVOKED"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			c := h.ingest(t, "wamid.final", "text", time.Now().Add(-time.Minute))
			h.submit(t, c, "synthetic-final-check", "reply")
			h.sql(t, tc.query)
			if e := h.s.DispatchOnce(h.ctx); e != nil {
				t.Fatal(e)
			}
			var code string
			if e := h.db.QueryRow(h.ctx, "SELECT error_code FROM app.outbound_intents").Scan(&code); e != nil || code != tc.code || h.sender.calls.Load() != 0 {
				t.Fatal(code, e)
			}
		})
	}
	t.Run("lost process", func(t *testing.T) {
		h := newHarness(t)
		c := h.ingest(t, "wamid.lost", "text", time.Now().Add(-time.Minute))
		h.submit(t, c, "synthetic-lost-process", "reply")
		h.sql(t, "UPDATE app.outbound_intents SET state='DISPATCHING',dispatch_started_at=now()-interval '2 minutes'")
		h.sql(t, "INSERT INTO app.send_attempts(id,organization_id,intent_id,state) SELECT $1,organization_id,id,'DISPATCHING' FROM app.outbound_intents", newID())
		if e := h.s.recoverUncertain(h.ctx); e != nil {
			t.Fatal(e)
		}
		if e := h.s.DispatchOnce(h.ctx); e != pgx.ErrNoRows || h.sender.calls.Load() != 0 {
			t.Fatal("recovery resent", e)
		}
	})
}
func TestRegistryMFAReviewPublicationAndBudgetRace(t *testing.T) {
	h := newHarness(t)
	h.expect(t, "POST", "/rate-cards/imports", map[string]any{"source": "GATE_C_20260914"}, 403)
	h.mfa(t, h.user)
	reviewer, _, reviewUser := h.secondActor(t, nil, "pricing.registry.review", "pricing.view")
	h.mfa(t, reviewUser)
	owner := h.owner
	imported := h.expect(t, "POST", "/rate-cards/imports", map[string]any{"source": "GATE_C_20260914"}, 200)
	id := imported["id"].(string)
	// Only the synthetic fixture advances the review deadline; production evidence stays Sep 21.
	h.sql(t, "UPDATE app.rate_imports SET next_review_at=now()+interval '1 day' WHERE id=$1", id)
	for _, action := range []string{"validate", "diff", "submit"} {
		h.expect(t, "POST", "/rate-cards/imports/"+id+"/"+action, map[string]any{"revision": 1}, 200)
	}
	h.expect(t, "POST", "/rate-cards/imports/"+id+"/review", map[string]any{"revision": 1, "approve": true}, 403)
	h.owner = reviewer
	h.expect(t, "POST", "/rate-cards/imports/"+id+"/review", map[string]any{"revision": 1, "approve": true}, 200)
	h.owner = owner
	denied := h.expect(t, "POST", "/rate-cards/imports/"+id+"/publish", map[string]any{"revision": 1}, 409)
	if denied["error"].(map[string]any)["code"] != "BILLING_CURRENCY_UNVERIFIED" {
		t.Fatal(denied)
	}
	h.sql(t, "UPDATE app.inbox_settings SET billing_currency_state='VERIFIED',billing_currency='USD',currency_evidence='SYNTHETIC TEST ONLY'")
	h.expect(t, "POST", "/rate-cards/imports/"+id+"/publish", map[string]any{"revision": 1}, 200)
	if _, e := h.db.Exec(h.ctx, "UPDATE app.rate_publications SET snapshot_hash='tampered'"); e == nil {
		t.Fatal("published mutation accepted")
	}
	c := h.ingest(t, "wamid.budget", "text", time.Now().Add(-time.Minute))
	first := h.submit(t, c, "synthetic-budget-one", "first")["id"].(string)
	second := h.submit(t, c, "synthetic-budget-two", "second")["id"].(string)
	period := newID()
	h.sql(t, "INSERT INTO app.budget_periods(id,organization_id,name,currency,starts_at,ends_at,hard_limit) VALUES($1,$2,'Synthetic final unit','USD',now()-interval '1 hour',now()+interval '1 hour',1)", period, h.org)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, value := range []string{first, second} {
		wg.Add(1)
		go func(value string) {
			defer wg.Done()
			tx, e := h.s.Auth.Pool.Begin(h.ctx)
			if e != nil {
				results <- e
				return
			}
			defer rollback(tx)
			_, e = tx.Exec(h.ctx, "SELECT set_config('app.organization_id',$1,true)", h.org.String())
			if e == nil {
				e = reserveBudget(h.ctx, tx, h.org, period, uuid.MustParse(value), "1", "USD")
			}
			if e == nil {
				e = tx.Commit(h.ctx)
			}
			results <- e
		}(value)
	}
	wg.Wait()
	close(results)
	ok, blocked := 0, 0
	for e := range results {
		if e == nil {
			ok++
		} else if e == Error("BUDGET_EXCEEDED") {
			blocked++
		} else {
			t.Fatal(e)
		}
	}
	if ok != 1 || blocked != 1 {
		t.Fatal("budget overspend", ok, blocked)
	}

	// Recover the winning intent repeatedly, including after an uncertain outcome.
	// The same exact monetary reservation and append-only ledger entry must remain.
	var winner uuid.UUID
	if e := h.db.QueryRow(h.ctx, "SELECT intent_id FROM app.budget_reservations").Scan(&winner); e != nil {
		t.Fatal(e)
	}
	repeat := func(requested string) error {
		tx, e := h.s.Auth.Pool.Begin(h.ctx)
		if e != nil {
			return e
		}
		defer rollback(tx)
		if _, e = tx.Exec(h.ctx, "SELECT set_config('app.organization_id',$1,true)", h.org.String()); e != nil {
			return e
		}
		if e = reserveBudget(h.ctx, tx, h.org, period, winner, requested, "USD"); e != nil {
			return e
		}
		return tx.Commit(h.ctx)
	}
	for range 3 {
		if e := repeat("1.00000000"); e != nil {
			t.Fatal("idempotent reservation", e)
		}
	}
	if e := repeat("0.5"); e != Error("INTENT_SCOPE_CHANGED") {
		t.Fatal("changed reservation accepted", e)
	}
	h.sql(t, "UPDATE app.budget_reservations SET state='UNCERTAIN'")
	if e := repeat("1"); e != nil {
		t.Fatal("uncertain recovery", e)
	}
	var reservations, entries int
	if e := h.db.QueryRow(h.ctx, "SELECT (SELECT count(*) FROM app.budget_reservations),(SELECT count(*) FROM app.budget_ledger WHERE entry_kind='RESERVE')").Scan(&reservations, &entries); e != nil || reservations != 1 || entries != 1 {
		t.Fatal("duplicate financial reservation", reservations, entries, e)
	}

	var exposure string
	if e := h.db.QueryRow(h.ctx, "SELECT sum(amount)::text FROM app.budget_reservations WHERE state<>'RELEASED'").Scan(&exposure); e != nil || exposure != "1.00000000" {
		t.Fatal(exposure, e)
	}
	if h.sender.calls.Load() != 0 {
		t.Fatal("budget test reached provider")
	}
}
func TestOrphanDuplicateOutOfOrderStatuses(t *testing.T) {
	h := newHarness(t)
	c := h.ingest(t, "wamid.status-in", "text", time.Now().Add(-time.Minute))
	var event uuid.UUID
	if e := h.db.QueryRow(h.ctx, "SELECT id FROM app.webhook_events LIMIT 1").Scan(&event); e != nil {
		t.Fatal(e)
	}
	statuses := []string{"read", "delivered", "sent", "read"}
	for _, status := range statuses {
		e := h.s.workerScope(h.ctx, func(tx pgx.Tx, org uuid.UUID) error {
			return h.s.materializeStatus(h.ctx, tx, org, h.phone, event, json.RawMessage(encode(map[string]any{"id": "wamid.synthetic-outbound", "status": status, "timestamp": "1700000000", "pricing": map[string]any{"billable": false, "invoice_amount": "never-trust"}})))
		})
		if e != nil {
			t.Fatal(e)
		}
	}
	h.submit(t, c, "synthetic-status-intent", "reply")
	if e := h.s.DispatchOnce(h.ctx); e != nil {
		t.Fatal(e)
	}
	var state, pricing string
	var facts int
	if e := h.db.QueryRow(h.ctx, "SELECT delivery_state FROM app.messages WHERE direction='OUTBOUND'").Scan(&state); e != nil || state != "READ" {
		t.Fatal(state, e)
	}
	if e := h.db.QueryRow(h.ctx, "SELECT count(*),string_agg(pricing_metadata::text,',') FROM app.message_statuses").Scan(&facts, &pricing); e != nil || facts != 3 || strings.Contains(pricing, "invoice_amount") {
		t.Fatal(facts, pricing, e)
	}
}
func TestSSEDoesNotDiscloseOutOfScopeConversation(t *testing.T) {
	h := newHarness(t)
	c := h.ingest(t, "wamid.sse", "text", time.Now().Add(-time.Minute))
	team := newID()
	h.sql(t, "INSERT INTO app.teams(id,organization_id,name) VALUES($1,$2,'Other team')", team, h.org)
	actor, _, _ := h.secondActor(t, &team, "inbox.view")
	h.owner = actor
	ctx, cancel := context.WithTimeout(h.ctx, 100*time.Millisecond)
	defer cancel()
	r := httptest.NewRequest("GET", "/api/v1/inbox/events", nil).WithContext(ctx)
	r.AddCookie(&http.Cookie{Name: "waba_session", Value: actor.Cookie})
	w := httptest.NewRecorder()
	h.handler.ServeHTTP(w, r)
	if w.Code != 200 || strings.Contains(w.Body.String(), c.String()) || strings.Contains(w.Body.String(), "synthetic inbound") {
		t.Fatal("SSE leak", w.Code)
	}
}
