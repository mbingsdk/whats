//go:build integration

package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pquerna/otp/totp"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"waba.local/control/internal/migrate"
)

type harness struct {
	s                                       *Service
	admin                                   *pgx.Conn
	ctx                                     context.Context
	owner                                   Result
	org, user, member, ownerRole, agentRole uuid.UUID
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	ctx := context.Background()
	adminURL := os.Getenv("TEST_ADMIN_DATABASE_URL")
	if adminURL == "" {
		t.Fatal("real PostgreSQL configuration required; never skip")
	}
	admin, e := pgx.Connect(ctx, adminURL)
	if e != nil {
		t.Fatal("admin connection failed")
	}
	name := "waba_identity_test_" + strings.ReplaceAll(id().String(), "-", "")
	quoted := pgx.Identifier{name}.Sanitize()
	if _, e = admin.Exec(ctx, "CREATE DATABASE "+quoted+" OWNER waba_migrator"); e != nil {
		t.Fatal(e)
	}
	replaceDB := func(raw string) string {
		u, e := url.Parse(raw)
		if e != nil {
			t.Fatal("bad test config")
		}
		u.Path = "/" + name
		return u.String()
	}
	setup, e := pgx.Connect(ctx, replaceDB(adminURL))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = setup.Exec(ctx, "REVOKE ALL ON DATABASE "+quoted+" FROM PUBLIC; GRANT CONNECT ON DATABASE "+quoted+" TO waba_migrator,waba_runtime,waba_identity"); e != nil {
		t.Fatal(e)
	}
	if _, e = migrate.Up(ctx, replaceDB(os.Getenv("TEST_MIGRATION_DATABASE_URL")), os.DirFS(filepath.Join("..", "..", "..", "database", "migrations"))); e != nil {
		t.Fatal(e)
	}
	pool, e := pgxpool.New(ctx, replaceDB(os.Getenv("TEST_IDENTITY_DATABASE_URL")))
	if e != nil {
		t.Fatal(e)
	}
	s, e := New(pool, bytes.Repeat([]byte{7}, 32), "http://localhost:3000", false)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		pool.Close()
		_ = setup.Close(ctx)
		_, _ = admin.Exec(ctx, "DROP DATABASE "+quoted+" WITH (FORCE)")
		_ = admin.Close(ctx)
	})
	h := &harness{s: s, admin: setup, ctx: ctx}
	if e = s.Bootstrap(ctx, Input{Email: "owner@example.invalid", Password: "synthetic-password-42", Name: "Company", Slug: "company", Timezone: "UTC"}); e != nil {
		t.Fatal(e)
	}
	if e = setup.QueryRow(ctx, "SELECT organization_id,user_id FROM app.identity_bootstrap").Scan(&h.org, &h.user); e != nil {
		t.Fatal(e)
	}
	if e = setup.QueryRow(ctx, "SELECT id FROM app.organization_members WHERE user_id=$1", h.user).Scan(&h.member); e != nil {
		t.Fatal(e)
	}
	if e = setup.QueryRow(ctx, "SELECT id FROM app.roles WHERE name='Owner'").Scan(&h.ownerRole); e != nil {
		t.Fatal(e)
	}
	if e = setup.QueryRow(ctx, "SELECT id FROM app.roles WHERE name='Agent'").Scan(&h.agentRole); e != nil {
		t.Fatal(e)
	}
	proof := h.proof(t, "EMAIL_VERIFY", "owner@example.invalid")
	if _, e = s.Run(ctx, "public.verify.complete", "", Input{Token: proof}, Meta{IP: "verify-owner"}); e != nil {
		t.Fatal(e)
	}
	h.owner = h.login(t, "owner@example.invalid")
	return h
}
func (h *harness) login(t *testing.T, email string) Result {
	t.Helper()
	r, e := h.s.Run(h.ctx, "public.login", "", Input{Email: email, Password: "synthetic-password-42"}, Meta{IP: "test"})
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func (h *harness) call(action string, session Result, in Input) (Result, error) {
	if in.OrganizationID == uuid.Nil {
		in.OrganizationID = h.org
	}
	return h.s.Run(h.ctx, action, session.Cookie, in, Meta{IP: "test", CSRF: session.CSRF, RequestID: "synthetic-request"})
}
func (h *harness) proof(t *testing.T, purpose, email string) string {
	t.Helper()
	rows, e := h.admin.Query(h.ctx, "SELECT id,payload_ciphertext FROM app.mail_deliveries WHERE purpose=$1 AND payload_ciphertext IS NOT NULL ORDER BY created_at DESC", purpose)
	if e != nil {
		t.Fatal(e)
	}
	defer rows.Close()
	for rows.Next() {
		var mid uuid.UUID
		var data []byte
		if e = rows.Scan(&mid, &data); e != nil {
			t.Fatal(e)
		}
		clear, e := h.s.open(data, "mail:"+mid.String())
		if e != nil {
			t.Fatal(e)
		}
		var p MailPayload
		if e = json.Unmarshal(clear, &p); e != nil {
			t.Fatal(e)
		}
		if p.Recipient == email {
			return p.Token
		}
	}
	t.Fatal("proof not queued")
	return ""
}
func (h *harness) invite(t *testing.T, email string, roles ...uuid.UUID) (uuid.UUID, string) {
	t.Helper()
	r, e := h.call("org.invite", h.owner, Input{Email: email, RoleIDs: roles})
	if e != nil {
		t.Fatal(e)
	}
	return r.Data.(map[string]any)["id"].(uuid.UUID), h.proof(t, "INVITATION", email)
}
func (h *harness) employee(t *testing.T, email string, role uuid.UUID) (uuid.UUID, Result) {
	t.Helper()
	_, proof := h.invite(t, email, role)
	if _, e := h.s.Run(h.ctx, "public.invitation.accept", "", Input{Token: proof, Password: "synthetic-password-42", Name: "Employee"}, Meta{IP: email}); e != nil {
		t.Fatal(e)
	}
	var member uuid.UUID
	if e := h.admin.QueryRow(h.ctx, "SELECT m.id FROM app.organization_members m JOIN app.users u ON u.id=m.user_id WHERE u.canonical_email=$1", email).Scan(&member); e != nil {
		t.Fatal(e)
	}
	return member, h.login(t, email)
}
func race(t *testing.T, fn func() error) []error {
	t.Helper()
	start := make(chan struct{})
	out := make([]error, 2)
	var wg sync.WaitGroup
	for i := range out {
		wg.Add(1)
		go func(i int) { defer wg.Done(); <-start; out[i] = fn() }(i)
	}
	close(start)
	wg.Wait()
	return out
}
func oneSuccess(t *testing.T, results []error) {
	t.Helper()
	n := 0
	for _, e := range results {
		if e == nil {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("expected one success, got %d; errors %v", n, results)
	}
}
func TestIdentityLifecycle(t *testing.T) {
	h := newHarness(t)
	t.Run("bootstrap_rerun", func(t *testing.T) {
		e := h.s.Bootstrap(h.ctx, Input{Email: "another@example.invalid", Password: "synthetic-password-42", Name: "Company", Slug: "company"})
		if !errors.Is(e, ErrInitialized) {
			t.Fatal(e)
		}
	})
	t.Run("generic_login", func(t *testing.T) {
		for _, email := range []string{"unknown@example.invalid", "owner@example.invalid"} {
			_, e := h.s.Run(h.ctx, "public.login", "", Input{Email: email, Password: "wrong"}, Meta{IP: email})
			if !errors.Is(e, ErrAuthentication) {
				t.Fatal(e)
			}
		}
	})
	var member uuid.UUID
	var employee Result
	t.Run("invitation_concurrency", func(t *testing.T) {
		_, proof := h.invite(t, "employee@example.invalid", h.agentRole)
		oneSuccess(t, race(t, func() error {
			_, e := h.s.Run(h.ctx, "public.invitation.accept", "", Input{Token: proof, Password: "synthetic-password-42", Name: "Employee"}, Meta{IP: "invite"})
			return e
		}))
		if e := h.admin.QueryRow(h.ctx, "SELECT m.id FROM app.organization_members m JOIN app.users u ON u.id=m.user_id WHERE canonical_email='employee@example.invalid'").Scan(&member); e != nil {
			t.Fatal(e)
		}
		employee = h.login(t, "employee@example.invalid")
	})
	t.Run("agent_permission_and_cross_org", func(t *testing.T) {
		if _, e := h.call("org.members.list", employee, Input{}); !errors.Is(e, ErrPermission) {
			t.Fatal(e)
		}
		if _, e := h.call("org.get", h.owner, Input{OrganizationID: id()}); !errors.Is(e, ErrPermission) {
			t.Fatal(e)
		}
		if _, e := h.call("org.member.roles", employee, Input{ID: member, RoleIDs: []uuid.UUID{h.ownerRole}, Revision: 1}); !errors.Is(e, ErrPermission) {
			t.Fatal(e)
		}
	})
	t.Run("owner_protection", func(t *testing.T) {
		if _, e := h.call("org.member.deactivate", h.owner, Input{ID: h.member, Revision: 1}); !errors.Is(e, ErrOwner) {
			t.Fatal(e)
		}
		if _, e := h.call("org.member.roles", h.owner, Input{ID: h.member, Revision: 1}); !errors.Is(e, ErrOwner) {
			t.Fatal(e)
		}
	})
	t.Run("teams_and_scoped_roles", func(t *testing.T) {
		team, e := h.call("org.team.create", h.owner, Input{Name: "Support"})
		if e != nil {
			t.Fatal(e)
		}
		tid := team.Data.(map[string]any)["id"].(uuid.UUID)
		if _, e = h.call("org.team.member.add", h.owner, Input{ID: tid, MemberID: member, Revision: 1}); e != nil {
			t.Fatal(e)
		}
		if _, e = h.call("org.team.members.list", h.owner, Input{ID: tid}); e != nil {
			t.Fatal(e)
		}
		role, e := h.call("org.role.create", h.owner, Input{Name: "Team coordinator", Permissions: []string{"teams.view", "teams.manage"}})
		if e != nil {
			t.Fatal(e)
		}
		rid := role.Data.(map[string]any)["id"].(uuid.UUID)
		if _, e = h.call("org.member.roles", h.owner, Input{ID: member, Revision: 1, RoleIDs: []uuid.UUID{rid}, Scope: "TEAM", TeamID: &tid}); e != nil {
			t.Fatal(e)
		}
		employee = h.login(t, "employee@example.invalid")
		if _, e = h.call("org.team.members.list", employee, Input{ID: tid}); e != nil {
			t.Fatal(e)
		}
		if _, e = h.call("org.team.members.list", employee, Input{ID: id()}); !errors.Is(e, ErrPermission) {
			t.Fatal(e)
		}
		if _, e = h.call("org.role.create", employee, Input{Name: "Escalate", Permissions: []string{"owners.manage"}}); !errors.Is(e, ErrPermission) {
			t.Fatal(e)
		}
	})
	t.Run("reset_supersession_and_concurrency", func(t *testing.T) {
		request := func() {
			if _, e := h.s.Run(h.ctx, "public.reset.request", "", Input{Email: "employee@example.invalid"}, Meta{IP: "reset"}); e != nil {
				t.Fatal(e)
			}
		}
		request()
		old := h.proof(t, "PASSWORD_RESET", "employee@example.invalid")
		request()
		proof := h.proof(t, "PASSWORD_RESET", "employee@example.invalid")
		if _, e := h.s.Run(h.ctx, "public.reset.complete", "", Input{Token: old, Password: "synthetic-password-42"}, Meta{IP: "reset"}); !errors.Is(e, ErrToken) {
			t.Fatal(e)
		}
		oneSuccess(t, race(t, func() error {
			_, e := h.s.Run(h.ctx, "public.reset.complete", "", Input{Token: proof, Password: "synthetic-password-42"}, Meta{IP: "reset"})
			return e
		}))
		if _, e := h.call("session", employee, Input{}); !errors.Is(e, ErrUnauthenticated) {
			t.Fatal(e)
		}
		employee = h.login(t, "employee@example.invalid")
	})
	t.Run("mfa_and_recovery_concurrency", func(t *testing.T) {
		r, e := h.call("mfa.enroll", employee, Input{})
		if e != nil {
			t.Fatal(e)
		}
		secret := r.Data.(map[string]any)["secret"].(string)
		code, e := totp.GenerateCode(secret, h.s.Now())
		if e != nil {
			t.Fatal(e)
		}
		r, e = h.call("mfa.confirm", employee, Input{Code: code})
		if e != nil {
			t.Fatal(e)
		}
		codes := r.Data.(map[string]any)["recovery_codes"].([]string)
		employee = r
		challenge := h.login(t, "employee@example.invalid").Data.(map[string]any)["challenge"].(string)
		oneSuccess(t, race(t, func() error {
			_, e := h.s.Run(h.ctx, "public.mfa", "", Input{Token: challenge, Code: codes[0]}, Meta{IP: "mfa"})
			return e
		}))
		if _, e = h.s.Run(h.ctx, "public.mfa", "", Input{Token: challenge, Code: codes[0]}, Meta{IP: "mfa"}); !errors.Is(e, ErrToken) {
			t.Fatal(e)
		}
	})
	t.Run("offboarding_and_revision", func(t *testing.T) {
		if _, e := h.call("org.member.deactivate", h.owner, Input{ID: member, Revision: 2}); e != nil {
			t.Fatal(e)
		}
		if _, e := h.call("org.get", employee, Input{}); !errors.Is(e, ErrUnauthenticated) {
			t.Fatal(e)
		}
		var count int
		if e := h.admin.QueryRow(h.ctx, "SELECT count(*) FROM app.team_members WHERE member_id=$1", member).Scan(&count); e != nil || count != 0 {
			t.Fatal("team references retained", e)
		}
	})
	t.Run("audit_redaction_and_append_only", func(t *testing.T) {
		var raw string
		if e := h.admin.QueryRow(h.ctx, "SELECT coalesce(jsonb_agg(to_jsonb(a))::text,'[]') FROM app.audit_log a").Scan(&raw); e != nil {
			t.Fatal(e)
		}
		if strings.Contains(raw, "synthetic-password") || strings.Contains(raw, "@example.invalid") || strings.Contains(raw, h.owner.Cookie) {
			t.Fatal("audit leaked sensitive data")
		}
		if _, e := h.s.Pool.Exec(h.ctx, "DELETE FROM app.identity_audit"); e == nil {
			t.Fatal("audit delete permitted")
		}
	})
}

type captureSender struct {
	messages []Mail
	err      error
}

func (c *captureSender) Send(ctx context.Context, m Mail) error {
	c.messages = append(c.messages, m)
	return c.err
}
func TestMailOutbox(t *testing.T) {
	h := newHarness(t)
	_, proof := h.invite(t, "mail@example.invalid", h.agentRole)
	sender := &captureSender{err: &MailError{Code: "SMTP_UNKNOWN"}}
	for i := 0; i < 5; i++ {
		worked, e := h.s.DeliverOne(h.ctx, sender)
		if e != nil {
			t.Fatal(e)
		}
		if !worked {
			break
		}
	}
	if len(sender.messages) != 1 {
		t.Fatalf("expected invitation only, got %d", len(sender.messages))
	}
	if !strings.Contains(sender.messages[0].Body, "#token="+proof) {
		t.Fatal("wrong action token")
	}
	if _, e := h.admin.Exec(h.ctx, "UPDATE app.mail_deliveries SET next_attempt_at=now() WHERE state='RETRY_WAIT'"); e != nil {
		t.Fatal(e)
	}
	sender.err = nil
	if _, e := h.s.DeliverOne(h.ctx, sender); e != nil {
		t.Fatal(e)
	}
	if len(sender.messages) != 2 || sender.messages[0].ID != sender.messages[1].ID || sender.messages[0].Body != sender.messages[1].Body {
		t.Fatal("logical mail changed on retry")
	}
	if worked, e := h.s.DeliverOne(h.ctx, sender); e != nil || worked {
		t.Fatal("terminal replay", e)
	}
	var count int
	if e := h.admin.QueryRow(h.ctx, "SELECT count(*) FROM app.mail_deliveries WHERE state='SMTP_ACCEPTED' AND payload_ciphertext IS NOT NULL").Scan(&count); e != nil || count != 0 {
		t.Fatal("terminal plaintext retention", e)
	}
}
func TestSessionExpiryAndLimits(t *testing.T) {
	h := newHarness(t)
	if _, e := h.admin.Exec(h.ctx, "UPDATE app.sessions SET idle_expires_at=now()-interval '1 second' WHERE token_digest=$1", digest(h.owner.Cookie)); e != nil {
		t.Fatal(e)
	}
	if _, e := h.call("session", h.owner, Input{}); !errors.Is(e, ErrUnauthenticated) {
		t.Fatal(e)
	}
	for i := 0; i < 20; i++ {
		if e := h.s.limit(h.ctx, "bounded", 20); e != nil {
			t.Fatal(e)
		}
	}
	if e := h.s.limit(h.ctx, "bounded", 20); !errors.Is(e, ErrRate) {
		t.Fatal(e)
	}
	if _, e := h.admin.Exec(h.ctx, "UPDATE app.login_limits SET expires_at=now()-interval '1 second'"); e != nil {
		t.Fatal(e)
	}
	if e := h.s.limit(h.ctx, "bounded", 20); e != nil {
		t.Fatal(e)
	}
}
