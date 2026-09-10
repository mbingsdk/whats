//go:build integration

package identity

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"waba.local/control/internal/config"
	"waba.local/control/internal/httpapi"
)

func TestHTTPAuthenticationBoundary(t *testing.T) {
	h := newHarness(t)
	cfg := config.Config{MaxBodyBytes: 4096}
	handler := httpapi.Handler(cfg, slog.New(slog.NewJSONHandler(io.Discard, nil)), func(context.Context) error { return nil }, h.s.Handler())
	call := func(method, path, body, origin, csrf, cookie string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.RemoteAddr = "127.0.0.1:1234"
		r.Header.Set("X-Request-ID", "test-http")
		r.Header.Set("Origin", origin)
		r.Header.Set("X-CSRF-Token", csrf)
		if csrf != "" {
			r.AddCookie(&http.Cookie{Name: "waba_csrf", Value: csrf})
		}
		if cookie != "" {
			r.AddCookie(&http.Cookie{Name: "waba_session", Value: cookie})
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	sessionResponse := call("GET", "/api/v1/auth/session", "", "", "", h.owner.Cookie)
	var contract struct{ Organizations []struct{ ID string } }
	if e := json.Unmarshal(sessionResponse.Body.Bytes(), &contract); e != nil || len(contract.Organizations) != 1 || contract.Organizations[0].ID != h.org.String() {
		t.Fatal("organization ID contract", e)
	}
	if w := call("GET", "/api/v1/auth/session", "", "", "", ""); w.Code != 401 {
		t.Fatal(w.Code)
	}
	if w := call("POST", "/api/v1/auth/logout", "{}", "http://evil.invalid", h.owner.CSRF, h.owner.Cookie); w.Code != 403 {
		t.Fatal(w.Code)
	}
	if w := call("POST", "/api/v1/auth/logout", "{}", h.s.Origin, randomToken(), h.owner.Cookie); w.Code != 403 {
		t.Fatal("csrf substitution", w.Code)
	}
	w := call("POST", "/api/v1/auth/login", `{"email":"owner@example.invalid","password":"synthetic-password-42"}`, h.s.Origin, randomToken(), "")
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w.Header().Get("X-Request-ID") != "test-http" || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("missing security envelope")
	}
	var raw map[string]any
	if e := json.Unmarshal(w.Body.Bytes(), &raw); e != nil {
		t.Fatal(e)
	}
	for _, c := range w.Result().Cookies() {
		if c.Name == "waba_session" {
			if !c.HttpOnly || c.Path != "/" || c.SameSite != http.SameSiteLaxMode {
				t.Fatal("weak cookie")
			}
			if strings.Contains(w.Body.String(), c.Value) {
				t.Fatal("session secret in JSON")
			}
		}
	}
	h.s.Production = true
	cookieWriter := httptest.NewRecorder()
	h.s.setCookie(cookieWriter, "waba_session", "synthetic", true)
	if !cookieWriter.Result().Cookies()[0].Secure {
		t.Fatal("production cookie not secure")
	}
	h.s.Production = false
	if w := call("POST", "/api/v1/auth/login", `{"email":"owner@example.invalid","password":"synthetic-password-42","user_id":"forged"}`, h.s.Origin, randomToken(), ""); w.Code != 422 {
		t.Fatal("unknown property accepted")
	}
	if w := call("POST", "/api/v1/auth/login", strings.Repeat("x", 5000), h.s.Origin, randomToken(), ""); w.Code != 413 {
		t.Fatal("body limit")
	}
	if w := call("POST", "/api/v1/organizations/"+h.org.String()+"/members/"+h.member.String()+"/deactivate", "{}", h.s.Origin, h.owner.CSRF, h.owner.Cookie); w.Code != 428 {
		t.Fatal("missing revision")
	}
	if _, e := h.call("sessions.revoke", h.owner, Input{}); e != nil {
		t.Fatal(e)
	}
	if _, e := h.call("session.revoke", h.owner, Input{ID: id()}); !errors.Is(e, ErrNotFound) {
		t.Fatal("cross-user session revoke")
	}
	if _, e := h.admin.Exec(h.ctx, "UPDATE app.sessions SET reauth_at=now()-interval '6 minutes' WHERE id=(SELECT id FROM app.sessions WHERE token_digest=$1)", digest(h.owner.Cookie)); e != nil {
		t.Fatal(e)
	}
	if _, e := h.call("mfa.enroll", h.owner, Input{}); !errors.Is(e, ErrReauth) {
		t.Fatal("stale reauthentication")
	}
}
func TestEveryTenantTableRLS(t *testing.T) {
	h := newHarness(t)
	tables := []string{"roles", "role_permissions", "teams", "team_members", "member_roles", "team_roles", "invitations", "invitation_roles", "audit_log"}
	own := map[string]uuid.UUID{}
	foreign := map[string]uuid.UUID{}
	seed := func(org, user, member uuid.UUID, label string, ids map[string]uuid.UUID) {
		for _, table := range tables {
			ids[table] = id()
		}
		queries := []struct {
			q string
			a []any
		}{
			{"INSERT INTO app.roles(id,organization_id,name) VALUES($1,$2,$3)", []any{ids["roles"], org, "Role " + label}},
			{"INSERT INTO app.role_permissions(id,organization_id,role_id,permission_key) VALUES($1,$2,$3,'organization.view')", []any{ids["role_permissions"], org, ids["roles"]}},
			{"INSERT INTO app.teams(id,organization_id,name) VALUES($1,$2,$3)", []any{ids["teams"], org, "Team " + label}},
			{"INSERT INTO app.team_members(id,organization_id,team_id,member_id) VALUES($1,$2,$3,$4)", []any{ids["team_members"], org, ids["teams"], member}},
			{"INSERT INTO app.member_roles(id,organization_id,member_id,role_id,scope_kind) VALUES($1,$2,$3,$4,'ORG')", []any{ids["member_roles"], org, member, ids["roles"]}},
			{"INSERT INTO app.team_roles(id,organization_id,team_id,role_id) VALUES($1,$2,$3,$4)", []any{ids["team_roles"], org, ids["teams"], ids["roles"]}},
			{"INSERT INTO app.invitations(id,organization_id,canonical_email,token_digest,inviter_member_id,expires_at) VALUES($1,$2,$3,$4,$5,now()+interval '1 hour')", []any{ids["invitations"], org, label + "@example.invalid", digest(randomToken()), member}},
			{"INSERT INTO app.invitation_roles(id,organization_id,invitation_id,role_id) VALUES($1,$2,$3,$4)", []any{ids["invitation_roles"], org, ids["invitations"], ids["roles"]}},
			{"INSERT INTO app.audit_log(id,organization_id,actor_user_id,actor_member_id,action,request_id) VALUES($1,$2,$3,$4,'test','synthetic')", []any{ids["audit_log"], org, user, member}},
		}
		for _, q := range queries {
			if _, e := h.admin.Exec(h.ctx, q.q, q.a...); e != nil {
				t.Fatal(e)
			}
		}
	}
	b, u, m := id(), id(), id()
	for _, q := range []struct {
		q string
		a []any
	}{
		{"INSERT INTO app.users(id,canonical_email,display_name) VALUES($1,'b@example.invalid','B')", []any{u}},
		{"INSERT INTO app.organizations(id,name,slug,timezone) VALUES($1,'B','b','UTC')", []any{b}},
		{"INSERT INTO app.organization_members(id,organization_id,user_id) VALUES($1,$2,$3)", []any{m, b, u}},
	} {
		if _, e := h.admin.Exec(h.ctx, q.q, q.a...); e != nil {
			t.Fatal(e)
		}
	}
	seed(h.org, h.user, h.member, "a", own)
	seed(b, u, m, "b", foreign)
	for _, table := range tables {
		t.Run(table, func(t *testing.T) {
			var forced bool
			if e := h.admin.QueryRow(h.ctx, "SELECT relrowsecurity AND relforcerowsecurity FROM pg_class WHERE oid=$1::regclass", "app."+table).Scan(&forced); e != nil || !forced {
				t.Fatal("RLS missing", e)
			}
			var n int
			if e := h.s.Pool.QueryRow(h.ctx, "SELECT count(*) FROM app."+table).Scan(&n); e != nil || n != 0 {
				t.Fatal("missing context read", e, n)
			}
			for _, pair := range []struct{ org, target uuid.UUID }{{h.org, foreign[table]}, {b, own[table]}} {
				tx, e := h.s.Pool.Begin(h.ctx)
				if e != nil {
					t.Fatal(e)
				}
				if e = scope(h.ctx, tx, pair.org); e != nil {
					t.Fatal(e)
				}
				if e = tx.QueryRow(h.ctx, "SELECT count(*) FROM app."+table+" WHERE id=$1", pair.target).Scan(&n); e != nil || n != 0 {
					t.Fatal("cross tenant read", e)
				}
				if table != "audit_log" {
					result, e := tx.Exec(h.ctx, "UPDATE app."+table+" SET organization_id=$2 WHERE id=$1", pair.target, pair.org)
					if e != nil || result.RowsAffected() != 0 {
						t.Fatal("cross tenant write", e)
					}
				}
				rollback(tx)
			}
			tx, e := h.s.Pool.Begin(h.ctx)
			if e != nil {
				t.Fatal(e)
			}
			defer rollback(tx)
			if e = scope(h.ctx, tx, h.org); e != nil {
				t.Fatal(e)
			}
			if _, e = tx.Exec(h.ctx, "UPDATE app."+table+" SET organization_id=$2 WHERE id=$1", own[table], b); e == nil {
				t.Fatal("cross tenant move allowed")
			}
		})
	}
	tx, e := h.s.Pool.Begin(h.ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer rollback(tx)
	if e = scope(h.ctx, tx, h.org); e != nil {
		t.Fatal(e)
	}
	if _, e = tx.Exec(h.ctx, "INSERT INTO app.team_members(id,organization_id,team_id,member_id) VALUES($1,$2,$3,$4)", id(), h.org, own["teams"], m); e == nil {
		t.Fatal("cross-tenant FK accepted")
	}
	var unsafe bool
	if e = h.s.Pool.QueryRow(h.ctx, "SELECT rolsuper OR rolbypassrls OR rolcreaterole OR rolcreatedb OR pg_has_role(current_user,'waba_migrator','MEMBER') OR pg_has_role(current_user,'waba_authorizer','MEMBER') FROM pg_roles WHERE rolname=current_user").Scan(&unsafe); e != nil || unsafe {
		t.Fatal("unsafe identity role")
	}
}
func TestVerificationExpiryAndConcurrentUse(t *testing.T) {
	h := newHarness(t)
	if _, e := h.admin.Exec(h.ctx, "UPDATE app.users SET verified_at=NULL WHERE id=$1", h.user); e != nil {
		t.Fatal(e)
	}
	if _, e := h.s.Run(h.ctx, "public.verify.request", "", Input{Email: "owner@example.invalid"}, Meta{IP: "verify"}); e != nil {
		t.Fatal(e)
	}
	token := h.proof(t, "EMAIL_VERIFY", "owner@example.invalid")
	oneSuccess(t, race(t, func() error {
		_, e := h.s.Run(h.ctx, "public.verify.complete", "", Input{Token: token}, Meta{IP: "verify"})
		return e
	}))
	if _, e := h.s.Run(h.ctx, "public.reset.request", "", Input{Email: "owner@example.invalid"}, Meta{IP: "reset"}); e != nil {
		t.Fatal(e)
	}
	token = h.proof(t, "PASSWORD_RESET", "owner@example.invalid")
	if _, e := h.admin.Exec(h.ctx, "UPDATE app.auth_challenges SET expires_at=now()-interval '1 second' WHERE token_digest=$1", digest(token)); e != nil {
		t.Fatal(e)
	}
	if _, e := h.s.Run(h.ctx, "public.reset.complete", "", Input{Token: token, Password: "synthetic-password-42"}, Meta{IP: "reset"}); !errors.Is(e, ErrToken) {
		t.Fatal("expired reset")
	}
}
func TestOffboardingOrderedAgainstAuthorization(t *testing.T) {
	h := newHarness(t)
	member, employee := h.employee(t, "race@example.invalid", h.agentRole)
	var user uuid.UUID
	if e := h.admin.QueryRow(h.ctx, "SELECT user_id FROM app.organization_members WHERE id=$1", member).Scan(&user); e != nil {
		t.Fatal(e)
	}
	if _, e := h.call("org.get", employee, Input{}); e != nil {
		t.Fatal("precondition", e)
	}
	tx, e := h.s.Pool.Begin(h.ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer rollback(tx)
	var result *uuid.UUID
	if e = tx.QueryRow(h.ctx, "SELECT app.authorize_membership($1,$2)", user, h.org).Scan(&result); e != nil || result == nil {
		t.Fatal(e)
	}
	done := make(chan error, 1)
	go func() { _, e := h.call("org.member.deactivate", h.owner, Input{ID: member, Revision: 1}); done <- e }()
	select {
	case e := <-done:
		t.Fatal("deactivation bypassed in-flight guard", e)
	case <-time.After(80 * time.Millisecond):
	}
	if e = tx.Commit(h.ctx); e != nil {
		t.Fatal(e)
	}
	select {
	case e := <-done:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("deactivation blocked")
	}
	if _, e := h.call("org.get", employee, Input{}); !errors.Is(e, ErrUnauthenticated) {
		t.Fatal("offboarded access", e)
	}
}
func TestLastOwnerConcurrentRemovalAndAdminCeiling(t *testing.T) {
	h := newHarness(t)
	second, other := h.employee(t, "owner2@example.invalid", h.ownerRole)
	start := make(chan struct{})
	done := make(chan error, 2)
	for _, pair := range []struct {
		member  uuid.UUID
		session Result
	}{{h.member, h.owner}, {second, other}} {
		go func(p struct {
			member  uuid.UUID
			session Result
		}) {
			<-start
			_, e := h.call("org.member.deactivate", p.session, Input{ID: p.member, Revision: 1})
			done <- e
		}(pair)
	}
	close(start)
	oneSuccess(t, []error{<-done, <-done})
	h2 := newHarness(t)
	var adminRole uuid.UUID
	if e := h2.admin.QueryRow(h2.ctx, "SELECT id FROM app.roles WHERE name='Admin'").Scan(&adminRole); e != nil {
		t.Fatal(e)
	}
	adminMember, adminSession := h2.employee(t, "admin@example.invalid", adminRole)
	if _, e := h2.call("org.member.roles", adminSession, Input{ID: adminMember, Revision: 1, RoleIDs: []uuid.UUID{h2.ownerRole}}); !errors.Is(e, ErrPermission) {
		t.Fatal("admin owner escalation", e)
	}
	if _, e := h2.call("org.role.update", adminSession, Input{ID: adminRole, Revision: 1, Name: "Admin", Permissions: []string{"owners.manage"}}); !errors.Is(e, ErrPermission) {
		t.Fatal("indirect role escalation", e)
	}
}
func TestMailCancellationExpiryAndLeaseRecovery(t *testing.T) {
	h := newHarness(t)
	inv, _ := h.invite(t, "cancel@example.invalid", h.agentRole)
	if _, e := h.call("org.invite.revoke", h.owner, Input{ID: inv, Revision: 1}); e != nil {
		t.Fatal(e)
	}
	sender := &captureSender{}
	for i := 0; i < 5; i++ {
		worked, e := h.s.DeliverOne(h.ctx, sender)
		if e != nil {
			t.Fatal(e)
		}
		if !worked {
			break
		}
	}
	if len(sender.messages) != 0 {
		t.Fatal("revoked mail transmitted")
	}
	inv, _ = h.invite(t, "crash@example.invalid", h.agentRole)
	var mid uuid.UUID
	if e := h.admin.QueryRow(h.ctx, "SELECT id FROM app.mail_deliveries WHERE invitation_id=$1", inv).Scan(&mid); e != nil {
		t.Fatal(e)
	}
	if _, e := h.admin.Exec(h.ctx, "UPDATE app.mail_deliveries SET state='SENDING',attempt_count=1,lease_token=$2,lease_until=now()-interval '1 second' WHERE id=$1", mid, id()); e != nil {
		t.Fatal(e)
	}
	if _, e := h.admin.Exec(h.ctx, "INSERT INTO app.mail_attempts(id,delivery_id,attempt_no) VALUES($1,$2,1)", id(), mid); e != nil {
		t.Fatal(e)
	}
	if _, e := h.s.DeliverOne(h.ctx, sender); e != nil {
		t.Fatal(e)
	}
	if len(sender.messages) != 1 {
		t.Fatal("lease recovery did not deliver")
	}
	var result string
	if e := h.admin.QueryRow(h.ctx, "SELECT result FROM app.mail_attempts WHERE delivery_id=$1 AND attempt_no=1", mid).Scan(&result); e != nil || result != "UNKNOWN" {
		t.Fatal("crash observation", e)
	}
}
