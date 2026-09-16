//go:build integration

package inbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"waba.local/control/internal/config"
	"waba.local/control/internal/identity"
	"waba.local/control/internal/meta"
	"waba.local/control/internal/migrate"
)

type fakeSender struct {
	calls   atomic.Int32
	outcome meta.SendOutcome
}

func (f *fakeSender) SendText(context.Context, string, string, string, string) meta.SendOutcome {
	f.calls.Add(1)
	return f.outcome
}

type harness struct {
	s                                       *Service
	db                                      *pgx.Conn
	ctx                                     context.Context
	org, user, member, phone, app, callback uuid.UUID
	owner                                   identity.Result
	handler                                 http.Handler
	sender                                  *fakeSender
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	ctx := context.Background()
	raw := os.Getenv("TEST_ADMIN_DATABASE_URL")
	if raw == "" {
		t.Fatal("real PostgreSQL required")
	}
	admin, e := pgx.Connect(ctx, raw)
	if e != nil {
		t.Fatal("admin connection")
	}
	name := "waba_inbox_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quoted := pgx.Identifier{name}.Sanitize()
	if _, e = admin.Exec(ctx, "CREATE DATABASE "+quoted+" OWNER waba_migrator"); e != nil {
		t.Fatal(e)
	}
	dsn := func(raw string) string {
		u, e := url.Parse(raw)
		if e != nil {
			t.Fatal("invalid DSN")
		}
		u.Path = "/" + name
		return u.String()
	}
	db, e := pgx.Connect(ctx, dsn(raw))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		_ = db.Close(ctx)
		_, _ = admin.Exec(ctx, "DROP DATABASE "+quoted+" WITH (FORCE)")
		_ = admin.Close(ctx)
	})
	if _, e = db.Exec(ctx, "REVOKE ALL ON DATABASE "+quoted+" FROM PUBLIC;GRANT CONNECT ON DATABASE "+quoted+" TO waba_runtime,waba_identity,waba_migrator"); e != nil {
		t.Fatal(e)
	}
	if _, e = migrate.Up(ctx, dsn(os.Getenv("TEST_MIGRATION_DATABASE_URL")), os.DirFS(filepath.Join("..", "..", "..", "database", "migrations"))); e != nil {
		t.Fatal(e)
	}
	pool, e := pgxpool.New(ctx, dsn(os.Getenv("TEST_IDENTITY_DATABASE_URL")))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(pool.Close)
	runtime, e := pgxpool.New(ctx, dsn(os.Getenv("TEST_RUNTIME_DATABASE_URL")))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(runtime.Close)
	auth, e := identity.New(pool, bytes.Repeat([]byte{9}, 32), "http://localhost:3000", false)
	if e != nil {
		t.Fatal(e)
	}
	if e = auth.Bootstrap(ctx, identity.Input{Email: "owner@example.invalid", Password: "synthetic-password-42", Name: "Inbox test", Slug: "inbox-test", Timezone: "UTC"}); e != nil {
		t.Fatal(e)
	}
	h := &harness{ctx: ctx, db: db}
	if e = db.QueryRow(ctx, "SELECT organization_id,user_id FROM app.identity_bootstrap").Scan(&h.org, &h.user); e != nil {
		t.Fatal(e)
	}
	h.sql(t, "UPDATE app.users SET verified_at=now() WHERE id=$1", h.user)
	h.owner, e = auth.Run(ctx, "public.login", "", identity.Input{Email: "owner@example.invalid", Password: "synthetic-password-42"}, identity.Meta{IP: "synthetic"})
	if e != nil {
		t.Fatal(e)
	}
	if e = db.QueryRow(ctx, "SELECT id FROM app.organization_members WHERE user_id=$1", h.user).Scan(&h.member); e != nil {
		t.Fatal(e)
	}
	h.app, h.phone, h.callback = newID(), newID(), newID()
	waba := newID()
	h.sql(t, "INSERT INTO app.meta_apps(id,organization_id,external_id,callback_key,graph_version,configured_waba_id) VALUES($1,$2,'111',$3,'v26.0','222')", h.app, h.org, h.callback)
	h.sql(t, "INSERT INTO app.wabas(id,organization_id,app_id,external_id,name,timezone_id,subscribed,lifecycle,graph_version) VALUES($1,$2,$3,'222','Synthetic WABA','1',true,'PRESENT','v26.0')", waba, h.org, h.app)
	h.sql(t, "INSERT INTO app.phone_numbers(id,organization_id,waba_id,external_id,name,display_number,quality,platform,code_verification_status,lifecycle,graph_version) VALUES($1,$2,$3,'333','Synthetic sender','+6280000000001','GREEN','CLOUD_API','VERIFIED','PRESENT','v26.0')", h.phone, h.org, waba)
	h.sql(t, "UPDATE app.inbox_settings SET sending_enabled=true")
	h.sql(t, "UPDATE app.meta_apps SET challenge_verified_at=now(),last_webhook_at=now()")
	h.sql(t, `INSERT INTO app.pricing_policies(id,organization_id,version,kind,source_url,source_sha256,review_evidence,published_at,effective_from,effective_to,next_review_at)
 VALUES($1,$2,'SYNTHETIC_TEST_POLICY','SERVICE_ZERO','https://example.invalid/synthetic-policy',repeat('0',64),'SYNTHETIC TEST ONLY',now()+interval '1 second',now()-interval '1 day',now()+interval '2 days',now()+interval '1 day')`, newID(), h.org)
	m := meta.New(config.Meta{Enabled: true, Version: "v26.0", AppID: "111", WABAID: "222", PhoneID: "333", AccessToken: "synthetic-access-token", AppSecret: "synthetic-app-secret", VerifyToken: "synthetic-verify-token"}, runtime, auth)
	h.s = New(auth, m, LiveConfig{Enabled: true, Recipient: "6280000000000", DeliveryBound: time.Minute, AcceptanceNotBefore: time.Now().Add(-time.Hour)})
	h.sender = &fakeSender{outcome: meta.SendOutcome{State: "ACCEPTED", ProviderID: "wamid.synthetic-outbound"}}
	h.s.Sender = h.sender
	h.handler = h.s.Handler(m.Handler(auth.Handler()))
	return h
}
func (h *harness) sql(t *testing.T, q string, args ...any) {
	t.Helper()
	if _, e := h.db.Exec(h.ctx, q, args...); e != nil {
		t.Fatal(e)
	}
}
func (h *harness) request(method, path string, body any) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "/api/v1"+path, bytes.NewReader(encode(body)))
	r.Header.Set("Origin", h.s.Auth.Origin)
	r.Header.Set("X-CSRF-Token", h.owner.CSRF)
	r.AddCookie(&http.Cookie{Name: "waba_session", Value: h.owner.Cookie})
	r.AddCookie(&http.Cookie{Name: "waba_csrf", Value: h.owner.CSRF})
	w := httptest.NewRecorder()
	h.handler.ServeHTTP(w, r)
	return w
}
func (h *harness) expect(t *testing.T, method, path string, body any, status int) map[string]any {
	t.Helper()
	w := h.request(method, path, body)
	if w.Code != status {
		t.Fatalf("%s %s = %d %s", method, path, w.Code, w.Body.String())
	}
	var result map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &result)
	return result
}
func (h *harness) ingest(t *testing.T, id, typ string, at time.Time) uuid.UUID {
	t.Helper()
	eid := newID()
	message := map[string]any{"id": id, "from": "6280000000000", "type": typ, "timestamp": fmt.Sprint(at.Unix()), "text": map[string]any{"body": "synthetic inbound"}}
	value := map[string]any{"metadata": map[string]any{"phone_number_id": "333"}, "messages": []any{message}}
	change := map[string]any{"field": "messages", "value": value}
	entry := map[string]any{"id": "222", "changes": []any{change}}
	raw := encode(map[string]any{"object": "whatsapp_business_account", "nonce": uuid.NewString(), "entry": []any{entry}})
	sealed, e := h.s.Auth.SealEvidence(raw, h.org.String()+":"+eid.String()+":meta-webhook:v1")
	if e != nil {
		t.Fatal(e)
	}
	h.sql(t, "INSERT INTO app.webhook_events(id,organization_id,app_id,raw_digest,raw_ciphertext,request_id,state) VALUES($1,$2,$3,$4,$5,'synthetic-inbox','PROCESSED')", eid, h.org, h.app, hash(raw), sealed)
	if e = h.s.MaterializeOnce(h.ctx); e != nil {
		t.Fatal(e)
	}
	var c uuid.UUID
	if e = h.db.QueryRow(h.ctx, "SELECT id FROM app.conversations LIMIT 1").Scan(&c); e != nil {
		t.Fatal(e)
	}
	return c
}
func (h *harness) submit(t *testing.T, c uuid.UUID, key, text string) map[string]any {
	t.Helper()
	input := map[string]any{"conversation_id": c, "assignment_revision": 1, "text": text, "message_type": "TEXT", "category": "SERVICE"}
	pre := h.expect(t, "POST", "/pricing/preflight", input, 200)
	auth, ok := pre["authorization"].(string)
	if !ok {
		t.Fatalf("preflight blocked: %v", pre)
	}
	input["authorization"] = auth
	input["client_idempotency_key"] = key
	return h.expect(t, "POST", "/outbound-intents", input, 200)
}
func TestMaterializationReplayWindowAndUnread(t *testing.T) {
	h := newHarness(t)
	now := time.Now().Add(-time.Minute)
	c := h.ingest(t, "wamid.synthetic-in", "text", now)
	h.ingest(t, "wamid.synthetic-in", "text", now)
	h.ingest(t, "wamid.synthetic-old", "text", now.Add(-25*time.Hour))
	var count, sequence int
	var basis time.Time
	if e := h.db.QueryRow(h.ctx, "SELECT (SELECT count(*) FROM app.messages),inbound_sequence,window_basis_at FROM app.conversations WHERE id=$1", c).Scan(&count, &sequence, &basis); e != nil {
		t.Fatal(e)
	}
	if count != 2 || sequence != 2 || basis.Unix() != now.Unix() {
		t.Fatal("duplicate/window regression", count, sequence, basis)
	}
	h.expect(t, "POST", "/conversations/"+c.String()+"/read", map[string]any{"watermark": 2}, 200)
	h.ingest(t, "wamid.synthetic-in", "text", now)
	var watermark int
	if e := h.db.QueryRow(h.ctx, "SELECT inbound_watermark FROM app.conversation_reads").Scan(&watermark); e != nil || watermark != 2 {
		t.Fatal("read watermark")
	}
	unknown := h.ingest(t, "wamid.synthetic-unknown", "future_type", time.Now())
	h.expect(t, "GET", "/conversations/"+unknown.String(), nil, 200)
}
func TestGuardedIntentIdempotencyAndFinalDenials(t *testing.T) {
	h := newHarness(t)
	c := h.ingest(t, "wamid.synthetic-in", "text", time.Now().Add(-time.Minute))
	h.submit(t, c, "synthetic-key-0001", "first")
	h.submit(t, c, "synthetic-key-0001", "first")
	h.submit(t, c, "synthetic-key-0002", "second")
	var count int
	if e := h.db.QueryRow(h.ctx, "SELECT count(*) FROM app.outbound_intents").Scan(&count); e != nil || count != 2 {
		t.Fatal("sequential/idempotent intents", count, e)
	}
	h.sql(t, "UPDATE app.conversations SET assignment_revision=2 WHERE id=$1", c)
	if e := h.s.DispatchOnce(h.ctx); e != nil {
		t.Fatal(e)
	}
	var code string
	if e := h.db.QueryRow(h.ctx, "SELECT error_code FROM app.outbound_intents ORDER BY created_at LIMIT 1").Scan(&code); e != nil || code != "ASSIGNMENT_CHANGED" {
		t.Fatal(code, e)
	}
	if h.sender.calls.Load() != 0 {
		t.Fatal("stale dispatch reached provider")
	}
}
func TestConcurrentDuplicateAndZeroCostNoLedger(t *testing.T) {
	h := newHarness(t)
	c := h.ingest(t, "wamid.synthetic-in", "text", time.Now().Add(-time.Minute))
	in := map[string]any{"conversation_id": c, "assignment_revision": 1, "text": "reply"}
	pre := h.expect(t, "POST", "/pricing/preflight", in, 200)
	in["authorization"] = pre["authorization"]
	in["client_idempotency_key"] = "synthetic-concurrent-0001"
	var wg sync.WaitGroup
	codes := make(chan int, 2)
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); codes <- h.request("POST", "/outbound-intents", in).Code }()
	}
	wg.Wait()
	close(codes)
	for code := range codes {
		if code != 200 {
			t.Fatalf("concurrent submit %d", code)
		}
	}
	if e := h.s.DispatchOnce(h.ctx); e != nil {
		t.Fatal(e)
	}
	if e := h.s.DispatchOnce(h.ctx); e != pgx.ErrNoRows {
		t.Fatal(e)
	}
	var intents, attempts, ledger int
	if e := h.db.QueryRow(h.ctx, "SELECT (SELECT count(*) FROM app.outbound_intents),(SELECT count(*) FROM app.send_attempts),(SELECT count(*) FROM app.budget_ledger)").Scan(&intents, &attempts, &ledger); e != nil {
		t.Fatal(e)
	}
	if intents != 1 || attempts != 1 || ledger != 0 || h.sender.calls.Load() != 1 {
		t.Fatal(intents, attempts, ledger, h.sender.calls.Load())
	}
}
func TestUncertainNeverResends(t *testing.T) {
	h := newHarness(t)
	h.sender.outcome = meta.SendOutcome{State: "UNCERTAIN", Code: "DISPATCH_UNCERTAIN"}
	c := h.ingest(t, "wamid.synthetic-in", "text", time.Now().Add(-time.Minute))
	h.submit(t, c, "synthetic-uncertain-key", "reply")
	if e := h.s.DispatchOnce(h.ctx); e != nil {
		t.Fatal(e)
	}
	for range 3 {
		if e := h.s.DispatchOnce(h.ctx); e != pgx.ErrNoRows {
			t.Fatal(e)
		}
	}
	var state string
	if e := h.db.QueryRow(h.ctx, "SELECT state FROM app.outbound_intents").Scan(&state); e != nil || state != "UNCERTAIN" || h.sender.calls.Load() != 1 {
		t.Fatal(state, e)
	}
}

// The ordinary TTL remains known even when current executable pricing stops
// before its end. Both preflight and the last worker barrier must deny.
func TestKnownTTLIncompleteCoverageBlocksPreflightAndWorker(t *testing.T) {
	h := newHarness(t)
	c := h.ingest(t, "wamid.synthetic-horizon", "text", time.Now().Add(-time.Minute))
	pending := h.submit(t, c, "synthetic-horizon-queued", "synthetic reply")
	h.s.Live.DeliveryBound = ServiceDeliveryTTL
	h.sql(t, "UPDATE app.inbox_settings SET billing_currency_state='VERIFIED',billing_currency='IDR',currency_evidence='SYNTHETIC OPERATOR EVIDENCE'")
	pre := h.expect(t, "POST", "/pricing/preflight", map[string]any{"conversation_id": c, "assignment_revision": 1, "text": "another synthetic reply"}, 200)
	decision := pre["decision"].(map[string]any)
	if decision["allowed"] != false || decision["code"] != "PRICING_HORIZON_NOT_FULLY_COVERED" || pre["service_delivery_ttl_seconds"] != float64(2592000) || pre["authorization"] != nil {
		t.Fatal(pre)
	}
	if e := h.s.DispatchOnce(h.ctx); e != nil {
		t.Fatal(e)
	}
	var state, code string
	if e := h.db.QueryRow(h.ctx, "SELECT state,error_code FROM app.outbound_intents WHERE id=$1", pending["id"]).Scan(&state, &code); e != nil {
		t.Fatal(e)
	}
	if state != "BLOCKED" || code != "PRICING_HORIZON_NOT_FULLY_COVERED" || h.sender.calls.Load() != 0 {
		t.Fatal(state, code, h.sender.calls.Load())
	}
	var attempts, reservations int
	if e := h.db.QueryRow(h.ctx, "SELECT (SELECT count(*) FROM app.send_attempts),(SELECT count(*) FROM app.budget_reservations)").Scan(&attempts, &reservations); e != nil || attempts != 0 || reservations != 0 {
		t.Fatal(attempts, reservations, e)
	}
}
