//go:build integration

package meta

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	"waba.local/control/internal/config"
	"waba.local/control/internal/database"
	"waba.local/control/internal/identity"
	"waba.local/control/internal/migrate"
)

type harness struct {
	s         *Service
	admin     *pgx.Conn
	ctx       context.Context
	owner     identity.Result
	org, user uuid.UUID
	handler   http.Handler
	b         binding
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
	name := "waba_meta_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quoted := pgx.Identifier{name}.Sanitize()
	if _, e = admin.Exec(ctx, "CREATE DATABASE "+quoted+" OWNER waba_migrator"); e != nil {
		t.Fatal(e)
	}
	dsn := func(raw string) string {
		u, e := url.Parse(raw)
		if e != nil {
			t.Fatal("configuration")
		}
		u.Path = "/" + name
		return u.String()
	}
	setup, e := pgx.Connect(ctx, dsn(raw))
	if e != nil {
		t.Fatal("setup connection")
	}
	if _, e = setup.Exec(ctx, "REVOKE ALL ON DATABASE "+quoted+" FROM PUBLIC; GRANT CONNECT ON DATABASE "+quoted+" TO waba_runtime,waba_identity,waba_migrator"); e != nil {
		t.Fatal(e)
	}
	if _, e = migrate.Up(ctx, dsn(os.Getenv("TEST_MIGRATION_DATABASE_URL")), os.DirFS(filepath.Join("..", "..", "..", "database", "migrations"))); e != nil {
		t.Fatal(e)
	}
	pool, e := pgxpool.New(ctx, dsn(os.Getenv("TEST_RUNTIME_DATABASE_URL")))
	if e != nil {
		t.Fatal("runtime connection")
	}
	authPool, e := pgxpool.New(ctx, dsn(os.Getenv("TEST_IDENTITY_DATABASE_URL")))
	if e != nil {
		t.Fatal("auth connection")
	}
	auth, e := identity.New(authPool, bytes.Repeat([]byte{7}, 32), "http://localhost:3000", false)
	if e != nil {
		t.Fatal(e)
	}
	if e = auth.Bootstrap(ctx, identity.Input{Email: "owner@example.invalid", Password: "synthetic-password-42", Name: "Meta test company", Slug: "meta-company", Timezone: "UTC"}); e != nil {
		t.Fatal(e)
	}
	h := &harness{admin: setup, ctx: ctx}
	if e = setup.QueryRow(ctx, "SELECT organization_id,user_id FROM app.identity_bootstrap").Scan(&h.org, &h.user); e != nil {
		t.Fatal(e)
	}
	if _, e = setup.Exec(ctx, "UPDATE app.users SET verified_at=now() WHERE id=$1", h.user); e != nil {
		t.Fatal(e)
	}
	h.owner, e = auth.Run(ctx, "public.login", "", identity.Input{Email: "owner@example.invalid", Password: "synthetic-password-42"}, identity.Meta{IP: "synthetic"})
	if e != nil {
		t.Fatal(e)
	}
	h.s = New(config.Meta{Enabled: true, Version: "v26.0", AppID: "111", WABAID: "222", AccessToken: "synthetic-access-token", AppSecret: "synthetic-app-secret", VerifyToken: "synthetic-verify-token"}, pool, auth)
	h.handler = h.s.Handler(auth.Handler())
	t.Cleanup(func() {
		pool.Close()
		authPool.Close()
		_ = setup.Close(ctx)
		_, _ = admin.Exec(ctx, "DROP DATABASE "+quoted+" WITH (FORCE)")
		_ = admin.Close(ctx)
	})
	if e = database.Ready(ctx, pool); e != nil {
		t.Fatal(e)
	}
	h.expect(t, "POST", "/meta/connection", 200)
	h.b, e = h.s.binding(ctx)
	if e != nil {
		t.Fatal(e)
	}
	return h
}
func (h *harness) req(method, path string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "/api/v1"+path, strings.NewReader("{}"))
	r.Header.Set("Origin", h.s.Auth.Origin)
	r.Header.Set("X-CSRF-Token", h.owner.CSRF)
	r.AddCookie(&http.Cookie{Name: "waba_session", Value: h.owner.Cookie})
	r.AddCookie(&http.Cookie{Name: "waba_csrf", Value: h.owner.CSRF})
	w := httptest.NewRecorder()
	h.handler.ServeHTTP(w, r)
	return w
}
func (h *harness) expect(t *testing.T, method, path string, status int) *httptest.ResponseRecorder {
	t.Helper()
	w := h.req(method, path)
	if w.Code != status {
		t.Fatalf("%s %s: %d %s", method, path, w.Code, w.Body.String())
	}
	return w
}
func (h *harness) sql(t *testing.T, sql string, args ...any) {
	t.Helper()
	if _, e := h.admin.Exec(h.ctx, sql, args...); e != nil {
		t.Fatal(e)
	}
}
func (h *harness) count(t *testing.T, table string) int {
	t.Helper()
	var n int
	if e := h.admin.QueryRow(h.ctx, "SELECT count(*) FROM app."+table).Scan(&n); e != nil {
		t.Fatal(e)
	}
	return n
}
func (h *harness) deliver(body string, sig []string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", "/api/v1/meta/webhooks/"+h.b.Callback.String(), strings.NewReader(body))
	r.Header["X-Hub-Signature-256"] = sig
	w := httptest.NewRecorder()
	h.handler.ServeHTTP(w, r)
	return w
}
func (h *harness) graph(t *testing.T, absent, failed *atomic.Bool) {
	h.s.Graph = testGraph(t, func(w http.ResponseWriter, r *http.Request) {
		if failed.Load() {
			w.WriteHeader(503)
			fmt.Fprint(w, `{"error":{"code":2,"is_transient":true}}`)
			return
		}
		switch r.URL.Path {
		case "/v26.0/222":
			fmt.Fprint(w, `{"id":"222","name":"Synthetic company","timezone_id":1}`)
		case "/v26.0/222/phone_numbers":
			if absent.Load() {
				fmt.Fprint(w, `{"data":[]}`)
			} else {
				fmt.Fprint(w, `{"data":[{"id":"333","verified_name":"Synthetic phone","display_phone_number":"REDACTED","quality_rating":"GREEN","platform_type":"CLOUD_API","code_verification_status":"NOT_VERIFIED"}]}`)
			}
		case "/v26.0/333/whatsapp_business_profile":
			fmt.Fprint(w, `{"data":[{"vertical":"OTHER","messaging_product":"whatsapp"}]}`)
		case "/v26.0/222/subscribed_apps":
			fmt.Fprint(w, `{"data":[{"whatsapp_business_api_data":{"id":"111"}}]}`)
		default:
			t.Errorf("unexpected Graph route")
			http.NotFound(w, r)
		}
	})
}
func TestMetaPostgres(t *testing.T) {
	h := newHarness(t)
	var absent, failed atomic.Bool
	h.graph(t, &absent, &failed)
	t.Run("sync_all_assets_concurrently_idempotent", func(t *testing.T) {
		var wg sync.WaitGroup
		for range 5 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				w := h.req("POST", "/meta/sync")
				if w.Code != 200 {
					t.Errorf("queue %d", w.Code)
				}
			}()
		}
		wg.Wait()
		if h.count(t, "asset_sync_runs") != 1 {
			t.Fatal("duplicate sync jobs")
		}
		for range 3 {
			wg.Add(1)
			go func() { defer wg.Done(); _ = h.s.syncOne(h.ctx, h.b) }()
		}
		wg.Wait()
		for _, table := range []string{"wabas", "phone_numbers", "business_profiles"} {
			if h.count(t, table) != 1 {
				t.Fatal(table)
			}
		}
		var subscribed bool
		if e := h.admin.QueryRow(h.ctx, "SELECT subscribed FROM app.wabas").Scan(&subscribed); e != nil || !subscribed {
			t.Fatal("subscription sync")
		}
		h.expect(t, "POST", "/meta/sync", 200)
		if e := h.s.syncOne(h.ctx, h.b); e != nil {
			t.Fatal(e)
		}
		if h.count(t, "phone_numbers") != 1 {
			t.Fatal("idempotency")
		}
	})
	t.Run("disappearance_and_failure_preserve_assets", func(t *testing.T) {
		absent.Store(true)
		h.expect(t, "POST", "/meta/sync", 200)
		if e := h.s.syncOne(h.ctx, h.b); e != nil {
			t.Fatal(e)
		}
		var state string
		_ = h.admin.QueryRow(h.ctx, "SELECT lifecycle FROM app.phone_numbers").Scan(&state)
		if state != "NOT_OBSERVED" || h.count(t, "business_profiles") != 1 {
			t.Fatal("destructive disappearance")
		}
		failed.Store(true)
		h.expect(t, "POST", "/meta/sync", 200)
		if e := h.s.syncOne(h.ctx, h.b); e != nil {
			t.Fatal(e)
		}
		if h.count(t, "phone_numbers") != 1 {
			t.Fatal("failed sync deletion")
		}
		failed.Store(false)
		absent.Store(false)
		h.sql(t, "UPDATE app.asset_sync_runs SET next_attempt_at=now()")
		if e := h.s.syncOne(h.ctx, h.b); e != nil {
			t.Fatal(e)
		}
	})
	t.Run("get_post_authenticity_and_durable_duplicates", func(t *testing.T) {
		q := url.Values{"hub.mode": {"subscribe"}, "hub.verify_token": {"synthetic-verify-token"}, "hub.challenge": {"123456"}}
		w := h.req("GET", "/meta/webhooks/"+h.b.Callback.String()+"?"+q.Encode())
		if w.Code != 200 || w.Body.String() != "123456" {
			t.Fatal("GET challenge")
		}
		for _, suffix := range []string{"&hub.mode=subscribe", "&hub.verify_token=wrong"} {
			w = h.req("GET", "/meta/webhooks/"+h.b.Callback.String()+"?"+q.Encode()+suffix)
			if w.Code != 403 {
				t.Fatal("duplicate query")
			}
		}
		for _, sig := range [][]string{nil, {"sha256=wrong"}, {signature([]byte(inbound), "wrong")}} {
			if w = h.deliver(inbound, sig); w.Code != 403 {
				t.Fatal("signature rejection")
			}
		}
		if h.count(t, "webhook_events") != 0 {
			t.Fatal("unauthenticated persistence")
		}
		sig := signature([]byte(inbound), "synthetic-app-secret")
		var wg sync.WaitGroup
		for range 6 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if w := h.deliver(inbound, []string{sig}); w.Code != 200 {
					t.Errorf("ingress %d", w.Code)
				}
			}()
		}
		wg.Wait()
		if h.count(t, "webhook_events") != 1 {
			t.Fatal("transport duplicate")
		}
		if w = h.deliver(inbound+" ", []string{sig}); w.Code != 403 {
			t.Fatal("raw signature mutation")
		}
		var encrypted []byte
		var event uuid.UUID
		_ = h.admin.QueryRow(h.ctx, "SELECT id,raw_ciphertext FROM app.webhook_events").Scan(&event, &encrypted)
		if bytes.Contains(encrypted, []byte("PRIVATE TEXT")) {
			t.Fatal("plaintext evidence")
		}
		decoded, e := h.s.Auth.OpenEvidence(encrypted, evidenceAAD(h.org, event))
		if e != nil || string(decoded) != inbound {
			t.Fatal("raw recovery")
		}
		if _, e = h.s.Auth.OpenEvidence(encrypted, evidenceAAD(uuid.New(), event)); e == nil {
			t.Fatal("cross-tenant decryption")
		}
		if e = h.s.processOne(h.ctx, h.b); e != nil {
			t.Fatal(e)
		}
		if h.count(t, "webhook_facts") != 1 {
			t.Fatal("classification")
		}
		rebatch := strings.Replace(inbound, `"text":{"body":"PRIVATE TEXT"}`, `"text":{"body":"CHANGED COPY"}`, 1)
		w = h.deliver(rebatch, []string{signature([]byte(rebatch), "synthetic-app-secret")})
		if w.Code != 200 {
			t.Fatal(w.Code)
		}
		if e = h.s.processOne(h.ctx, h.b); e != nil {
			t.Fatal(e)
		}
		if h.count(t, "webhook_facts") != 1 {
			t.Fatal("semantic dedupe")
		}
	})
	t.Run("unknown_quarantine_and_permanent_invalid", func(t *testing.T) {
		for _, tc := range []struct{ body, state string }{
			{strings.Replace(inbound, `"field":"messages"`, `"field":"future_event"`, 1), "UNKNOWN"},
			{strings.Replace(inbound, `"id":"222"`, `"id":"999"`, 1), "QUARANTINED"},
			{strings.Replace(inbound, `"phone_number_id":"333"`, `"phone_number_id":"999"`, 1), "QUARANTINED"},
			{strings.Replace(inbound, `"id":"wamid.synthetic"`, `"id":""`, 1), "INVALID"},
		} {
			w := h.deliver(tc.body, []string{signature([]byte(tc.body), "synthetic-app-secret")})
			if w.Code != 200 {
				t.Fatal(w.Code)
			}
			if tc.state != "QUARANTINED" {
				if e := h.s.processOne(h.ctx, h.b); e != nil {
					t.Fatal(e)
				}
			}
			var state string
			_ = h.admin.QueryRow(h.ctx, "SELECT state FROM app.webhook_events WHERE raw_digest=$1", hash([]byte(tc.body))).Scan(&state)
			if state != tc.state {
				t.Fatalf("state %s want %s", state, tc.state)
			}
		}
	})
	t.Run("bounded_retry_dead_letter_and_replay", func(t *testing.T) {
		body := strings.Replace(inbound, "wamid.synthetic", "wamid.retry", 1)
		w := h.deliver(body, []string{signature([]byte(body), "synthetic-app-secret")})
		if w.Code != 200 {
			t.Fatal(w.Code)
		}
		h.sql(t, `CREATE FUNCTION app.synthetic_failure() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'synthetic storage failure'; END $$; CREATE TRIGGER synthetic_failure BEFORE INSERT ON app.webhook_facts FOR EACH ROW EXECUTE FUNCTION app.synthetic_failure()`)
		var id uuid.UUID
		_ = h.admin.QueryRow(h.ctx, "SELECT id FROM app.webhook_events WHERE raw_digest=$1", hash([]byte(body))).Scan(&id)
		for i := 1; i <= 8; i++ {
			h.sql(t, "UPDATE app.webhook_events SET next_attempt_at=now() WHERE id=$1", id)
			if e := h.s.processOne(h.ctx, h.b); e != nil {
				t.Fatal(e)
			}
			var state string
			_ = h.admin.QueryRow(h.ctx, "SELECT state FROM app.webhook_events WHERE id=$1", id).Scan(&state)
			expected := "RETRY_WAIT"
			if i == 8 {
				expected = "DEAD_LETTER"
			}
			if state != expected {
				t.Fatalf("attempt %d: %s", i, state)
			}
		}
		h.sql(t, "DROP TRIGGER synthetic_failure ON app.webhook_facts")
		h.expect(t, "POST", "/meta/webhook-events/"+id.String()+"/replay", 200)
		h.expect(t, "POST", "/meta/webhook-events/"+id.String()+"/replay", 409)
		if e := h.s.processOne(h.ctx, h.b); e != nil {
			t.Fatal(e)
		}
		before := h.count(t, "webhook_facts")
		h.expect(t, "POST", "/meta/webhook-events/"+id.String()+"/replay", 200)
		if e := h.s.processOne(h.ctx, h.b); e != nil {
			t.Fatal(e)
		}
		if h.count(t, "webhook_facts") != before {
			t.Fatal("replay duplicate")
		}
		if h.count(t, "webhook_replays") != 2 {
			t.Fatal("replay evidence")
		}
		w = h.expect(t, "GET", "/meta/webhook-events/"+id.String()+"/payload", 200)
		if strings.Contains(w.Body.String(), "PRIVATE TEXT") {
			t.Fatal("payload PII")
		}
	})
	t.Run("api_surfaces_permissions_csrf_and_rls_every_table", func(t *testing.T) {
		for _, path := range []string{"/meta/connection", "/meta/wabas", "/meta/phone-numbers", "/meta/business-profiles", "/meta/sync-runs", "/meta/health", "/meta/webhook-events"} {
			h.expect(t, "GET", path, 200)
		}
		var id uuid.UUID
		_ = h.admin.QueryRow(h.ctx, "SELECT id FROM app.webhook_events WHERE state='PROCESSED' LIMIT 1").Scan(&id)
		r := httptest.NewRequest("POST", "/api/v1/meta/webhook-events/"+id.String()+"/replay", strings.NewReader("{}"))
		r.AddCookie(&http.Cookie{Name: "waba_session", Value: h.owner.Cookie})
		w := httptest.NewRecorder()
		h.handler.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatal("CSRF")
		}
		h.sql(t, "DELETE FROM app.role_permissions WHERE permission_key='webhooks.replay'")
		h.expect(t, "POST", "/meta/webhook-events/"+id.String()+"/replay", 403)
		other := uuid.New()
		h.sql(t, "INSERT INTO app.organizations(id,name,slug,timezone) VALUES($1,'Other','other','UTC')", other)
		tables := []string{"meta_apps", "wabas", "phone_numbers", "business_profiles", "asset_sync_runs", "webhook_events", "webhook_facts", "webhook_processing_attempts", "webhook_replays"}
		for _, table := range tables {
			var force, rls bool
			if e := h.admin.QueryRow(h.ctx, "SELECT relforcerowsecurity,relrowsecurity FROM pg_class WHERE oid=$1::regclass", "app."+table).Scan(&force, &rls); e != nil || !force || !rls {
				t.Fatal("RLS " + table)
			}
			if h.count(t, table) == 0 {
				t.Fatal("isolation test must have populated table " + table)
			}
			if e := h.s.scoped(h.ctx, binding{Org: other}, func(tx pgx.Tx) error {
				var count int
				if e := tx.QueryRow(h.ctx, "SELECT count(*) FROM app."+table).Scan(&count); e != nil {
					return e
				}
				if count != 0 {
					t.Error("cross tenant read " + table)
				}
				return nil
			}); e != nil {
				t.Fatal(e)
			}
			var source []byte
			if e := h.admin.QueryRow(h.ctx, "SELECT to_jsonb(t) FROM app."+table+" t LIMIT 1").Scan(&source); e != nil {
				t.Fatal(e)
			}
			var row map[string]any
			if e := json.Unmarshal(source, &row); e != nil {
				t.Fatal(e)
			}
			row["id"] = uuid.NewString()
			source, _ = json.Marshal(row)
			denied := h.s.scoped(h.ctx, binding{Org: other}, func(tx pgx.Tx) error {
				_, e := tx.Exec(h.ctx, "INSERT INTO app."+table+" SELECT (jsonb_populate_record(NULL::app."+table+",$1::jsonb)).*", source)
				return e
			})
			var pg *pgconn.PgError
			if !errors.As(denied, &pg) || pg.Code != "42501" {
				t.Fatalf("WITH CHECK boundary %s: %v", table, denied)
			}
			var unscoped int
			if e := h.s.Pool.QueryRow(h.ctx, "SELECT count(*) FROM app."+table).Scan(&unscoped); e != nil || unscoped != 0 {
				t.Fatal("pool context leak " + table)
			}
		}
		e := h.s.scoped(h.ctx, binding{Org: other}, func(tx pgx.Tx) error {
			_, e := tx.Exec(h.ctx, "INSERT INTO app.meta_apps(id,organization_id,external_id,callback_key,graph_version,configured_waba_id) VALUES($1,$2,'999',$3,'v26.0','999')", uuid.New(), h.org, uuid.New())
			return e
		})
		if e == nil {
			t.Fatal("cross tenant insert")
		}
		for _, query := range []string{"SET ROLE waba_migrator", "SET ROLE waba_authorizer", "ALTER TABLE app.webhook_events DISABLE ROW LEVEL SECURITY"} {
			if _, e = h.s.Pool.Exec(h.ctx, query); e == nil {
				t.Fatal("unsafe runtime privilege")
			}
		}
		e = h.s.scoped(h.ctx, h.b, func(tx pgx.Tx) error {
			_, e := tx.Exec(h.ctx, "UPDATE app.webhook_events SET raw_digest='tampered' WHERE id=$1", id)
			return e
		})
		if e == nil {
			t.Fatal("original mutable")
		}
		e = h.s.scoped(h.ctx, h.b, func(tx pgx.Tx) error {
			_, e := tx.Exec(h.ctx, "UPDATE app.webhook_replays SET request_id='tampered'")
			return e
		})
		if e == nil {
			t.Fatal("replay evidence mutable")
		}
	})
	t.Run("cross_organization_http_authorization", func(t *testing.T) {
		var other uuid.UUID
		if e := h.admin.QueryRow(h.ctx, "SELECT id FROM app.organizations WHERE slug='other'").Scan(&other); e != nil {
			t.Fatal(e)
		}
		member, role := uuid.New(), uuid.New()
		h.sql(t, "INSERT INTO app.organization_members(id,organization_id,user_id) VALUES($1,$2,$3)", member, other, h.user)
		h.sql(t, "INSERT INTO app.roles(id,organization_id,name) VALUES($1,$2,'Scoped integration role')", role, other)
		for _, key := range []string{"organization.view", "meta.view", "meta.manage", "webhooks.view", "webhooks.replay"} {
			h.sql(t, "INSERT INTO app.role_permissions(id,organization_id,role_id,permission_key) VALUES($1,$2,$3,$4)", uuid.New(), other, role, key)
		}
		h.sql(t, "INSERT INTO app.member_roles(id,organization_id,member_id,role_id,scope_kind) VALUES($1,$2,$3,$4,'ORG')", uuid.New(), other, member, role)
		old := h.owner
		selected, e := h.s.Auth.Run(h.ctx, "organization.select", old.Cookie, identity.Input{OrganizationID: other}, identity.Meta{CSRF: old.CSRF})
		if e != nil {
			t.Fatal(e)
		}
		h.owner = selected
		w := h.expect(t, "GET", "/meta/wabas", 200)
		var page struct{ Items []any }
		if json.Unmarshal(w.Body.Bytes(), &page) != nil || len(page.Items) != 0 {
			t.Fatal("foreign assets visible")
		}
		var event uuid.UUID
		_ = h.admin.QueryRow(h.ctx, "SELECT id FROM app.webhook_events LIMIT 1").Scan(&event)
		h.expect(t, "GET", "/meta/webhook-events/"+event.String(), 404)
		h.expect(t, "POST", "/meta/webhook-events/"+event.String()+"/replay", 404)
		h.expect(t, "POST", "/meta/connection", 409)
		selected, e = h.s.Auth.Run(h.ctx, "organization.select", selected.Cookie, identity.Input{OrganizationID: h.org}, identity.Meta{CSRF: selected.CSRF})
		if e != nil {
			t.Fatal(e)
		}
		h.owner = selected
	})
	t.Run("persistence_failure_never_acknowledges", func(t *testing.T) {
		h.sql(t, `CREATE TRIGGER synthetic_failure BEFORE INSERT ON app.webhook_events FOR EACH ROW EXECUTE FUNCTION app.synthetic_failure()`)
		body := strings.Replace(inbound, "wamid.synthetic", "wamid.failure", 1)
		if w := h.deliver(body, []string{signature([]byte(body), "synthetic-app-secret")}); w.Code != 503 {
			t.Fatal("false durable ACK")
		}
	})
}
func TestStatusFactChanges(t *testing.T) {
	base := `{"object":"whatsapp_business_account","entry":[{"id":"222","changes":[{"field":"messages","value":{"metadata":{"phone_number_id":"333"},"statuses":[{"id":"wamid.status","status":"delivered","timestamp":"1","pricing":{"category":"utility"}}]}}]}]}`
	a, _ := classify([]byte(base))
	b, _ := classify([]byte(strings.Replace(base, "utility", "marketing", 1)))
	if len(a) != 1 || len(b) != 1 || a[0].Key == b[0].Key {
		t.Fatal("changed status fact lost")
	}
	var v any
	if json.Unmarshal([]byte(base), &v) != nil {
		t.Fatal("fixture")
	}
}
