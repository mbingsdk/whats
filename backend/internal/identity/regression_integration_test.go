//go:build integration

package identity

import (
	"errors"
	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"testing"
	"time"
)

func TestPaginationAndSelfScope(t *testing.T) {
	h := newHarness(t)
	for i := 0; i < 105; i++ {
		if _, e := h.admin.Exec(h.ctx, "INSERT INTO app.teams(id,organization_id,name) VALUES($1,$2,$3)", id(), h.org, id().String()); e != nil {
			t.Fatal(e)
		}
	}
	first, e := h.call("org.teams.list", h.owner, Input{})
	if e != nil {
		t.Fatal(e)
	}
	p := first.Data.(map[string]any)
	cursor := p["next_cursor"].(string)
	if len(p["items"].([]map[string]any)) != 100 || !p["has_more"].(bool) {
		t.Fatal("first page")
	}
	next, e := h.call("org.teams.list", h.owner, Input{Cursor: cursor})
	if e != nil {
		t.Fatal(e)
	}
	p2 := next.Data.(map[string]any)
	if len(p2["items"].([]map[string]any)) != 5 || p2["has_more"].(bool) {
		t.Fatal("last page")
	}
	seen := map[string]bool{}
	for _, page := range []map[string]any{p, p2} {
		for _, row := range page["items"].([]map[string]any) {
			key := row["id"].(string)
			if seen[key] {
				t.Fatal("duplicate page item")
			}
			seen[key] = true
		}
	}
	if _, e = h.call("org.roles.list", h.owner, Input{Cursor: cursor}); !errors.Is(e, ErrValidation) {
		t.Fatal("cross-resource cursor", e)
	}
	if _, e = h.call("org.teams.list", h.owner, Input{Cursor: cursor + "x"}); !errors.Is(e, ErrValidation) {
		t.Fatal("forged cursor", e)
	}
	member, employee := h.employee(t, "self@example.invalid", h.agentRole)
	r, e := h.call("org.role.create", h.owner, Input{Name: "Self view", Permissions: []string{"members.view"}})
	if e != nil {
		t.Fatal(e)
	}
	role := r.Data.(map[string]any)["id"].(uuid.UUID)
	if _, e = h.call("org.member.roles", h.owner, Input{ID: member, Revision: 1, RoleIDs: []uuid.UUID{role}, Scope: "SELF"}); e != nil {
		t.Fatal(e)
	}
	if _, e = h.call("session", employee, Input{}); !errors.Is(e, ErrUnauthenticated) {
		t.Fatal("role change retained session", e)
	}
	employee = h.login(t, "self@example.invalid")
	if _, e = h.call("org.member.get", employee, Input{ID: member}); e != nil {
		t.Fatal("self view", e)
	}
	if _, e = h.call("org.member.get", employee, Input{ID: h.member}); !errors.Is(e, ErrPermission) {
		t.Fatal("self escaped", e)
	}
	if _, e = h.call("org.members.list", employee, Input{}); !errors.Is(e, ErrPermission) {
		t.Fatal("self listed organization", e)
	}
}

func enableMFA(t *testing.T, h *harness, session Result) (Result, []string) {
	t.Helper()
	r, e := h.call("mfa.enroll", session, Input{})
	if e != nil {
		t.Fatal(e)
	}
	code, e := totp.GenerateCode(r.Data.(map[string]any)["secret"].(string), h.s.Now())
	if e != nil {
		t.Fatal(e)
	}
	r, e = h.call("mfa.confirm", session, Input{Code: code})
	if e != nil {
		t.Fatal(e)
	}
	return r, r.Data.(map[string]any)["recovery_codes"].([]string)
}
func TestMFASecurityChangesInvalidateProofs(t *testing.T) {
	h := newHarness(t)
	session, codes := enableMFA(t, h, h.owner)
	challenge := h.login(t, "owner@example.invalid").Data.(map[string]any)["challenge"].(string)
	r, e := h.call("mfa.recovery", session, Input{})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = h.s.Run(h.ctx, "public.mfa", "", Input{Token: challenge, Code: codes[0]}, Meta{IP: "mfa"}); !errors.Is(e, ErrToken) {
		t.Fatal("old login proof survived recovery change", e)
	}
	if _, e = h.call("session", session, Input{}); !errors.Is(e, ErrUnauthenticated) {
		t.Fatal("old session survived", e)
	}
	newCodes := r.Data.(map[string]any)["result"].(map[string]any)["recovery_codes"].([]string)
	challenge = h.login(t, "owner@example.invalid").Data.(map[string]any)["challenge"].(string)
	if _, e = h.s.Run(h.ctx, "public.mfa", "", Input{Token: challenge, Code: codes[1]}, Meta{IP: "mfa"}); !errors.Is(e, ErrAuthentication) {
		t.Fatal("old recovery accepted", e)
	}
	if _, e = h.s.Run(h.ctx, "public.reset.request", "", Input{Email: "owner@example.invalid"}, Meta{IP: "reset"}); e != nil {
		t.Fatal(e)
	}
	proof := h.proof(t, "PASSWORD_RESET", "owner@example.invalid")
	if _, e = h.s.Run(h.ctx, "public.reset.complete", "", Input{Token: proof, Password: "synthetic-password-42"}, Meta{IP: "reset"}); e != nil {
		t.Fatal(e)
	}
	if _, e = h.s.Run(h.ctx, "public.mfa", "", Input{Token: challenge, Code: newCodes[0]}, Meta{IP: "mfa"}); !errors.Is(e, ErrToken) {
		t.Fatal("pre-reset challenge survived", e)
	}
	challenge = h.login(t, "owner@example.invalid").Data.(map[string]any)["challenge"].(string)
	session, e = h.s.Run(h.ctx, "public.mfa", "", Input{Token: challenge, Code: newCodes[0]}, Meta{IP: "mfa"})
	if e != nil {
		t.Fatal(e)
	}
	disabled, e := h.call("mfa.disable", session, Input{})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = h.call("session", session, Input{}); !errors.Is(e, ErrUnauthenticated) {
		t.Fatal(e)
	}
	if _, e = h.call("session", disabled, Input{}); e != nil {
		t.Fatal(e)
	}
	if r := h.login(t, "owner@example.invalid"); r.Cookie == "" {
		t.Fatal("MFA was not disabled")
	}
	var count int
	if e = h.admin.QueryRow(h.ctx, "SELECT count(*) FROM app.auth_recovery_codes").Scan(&count); e != nil || count != 0 {
		t.Fatal("recovery retained", e)
	}
}

func TestOrganizationSwitchCannotRefreshSensitiveProof(t *testing.T) {
	h := newHarness(t)
	var expiry time.Time
	if e := h.admin.QueryRow(h.ctx, "SELECT expires_at FROM app.sessions WHERE token_digest=$1", digest(h.owner.Cookie)).Scan(&expiry); e != nil {
		t.Fatal(e)
	}
	if _, e := h.admin.Exec(h.ctx, "UPDATE app.sessions SET reauth_at=now()-interval '6 minutes' WHERE token_digest=$1", digest(h.owner.Cookie)); e != nil {
		t.Fatal(e)
	}
	rotated, e := h.call("organization.select", h.owner, Input{})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = h.call("mfa.enroll", rotated, Input{}); !errors.Is(e, ErrReauth) {
		t.Fatal("org switch renewed reauth", e)
	}
	var after time.Time
	if e = h.admin.QueryRow(h.ctx, "SELECT expires_at FROM app.sessions WHERE token_digest=$1", digest(rotated.Cookie)).Scan(&after); e != nil || !after.Equal(expiry) {
		t.Fatal("org switch extended absolute deadline", e)
	}
	if _, e = h.call("session", h.owner, Input{}); !errors.Is(e, ErrUnauthenticated) {
		t.Fatal("old token remained usable")
	}
}
func TestInvitationSupersessionExpiryAndExistingIdentity(t *testing.T) {
	h := newHarness(t)
	inv, old := h.invite(t, "old@example.invalid", h.agentRole)
	if _, e := h.call("org.invite.resend", h.owner, Input{ID: inv, Revision: 1}); e != nil {
		t.Fatal(e)
	}
	if _, e := h.s.Run(h.ctx, "public.invitation.accept", "", Input{Token: old, Password: "synthetic-password-42", Name: "Employee"}, Meta{IP: "invite"}); !errors.Is(e, ErrToken) {
		t.Fatal("superseded invitation", e)
	}
	proof := h.proof(t, "INVITATION", "old@example.invalid")
	if _, e := h.admin.Exec(h.ctx, "UPDATE app.invitations SET expires_at=now()-interval '1 second' WHERE id=$1", inv); e != nil {
		t.Fatal(e)
	}
	if _, e := h.s.Run(h.ctx, "public.invitation.accept", "", Input{Token: proof, Password: "synthetic-password-42", Name: "Employee"}, Meta{IP: "invite"}); !errors.Is(e, ErrToken) {
		t.Fatal("expired invitation", e)
	}
	// A pre-existing global identity without this organization membership.
	u := id()
	hash, e := HashPassword("synthetic-password-42")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = h.admin.Exec(h.ctx, "INSERT INTO app.users(id,canonical_email,display_name,verified_at) VALUES($1,'existing@example.invalid','Existing',now())", u); e != nil {
		t.Fatal(e)
	}
	if _, e = h.admin.Exec(h.ctx, "INSERT INTO app.password_identities(user_id,password_hash) VALUES($1,$2)", u, hash); e != nil {
		t.Fatal(e)
	}
	_, proof = h.invite(t, "existing@example.invalid", h.agentRole)
	if _, e = h.s.Run(h.ctx, "public.invitation.accept", "", Input{Token: proof, Password: "attacker-password-42", Name: "Existing"}, Meta{IP: "existing"}); !errors.Is(e, ErrUnauthenticated) {
		t.Fatal("invitation replaced identity", e)
	}
	session := h.login(t, "existing@example.invalid")
	if _, e = h.s.Run(h.ctx, "public.invitation.accept", session.Cookie, Input{Token: proof}, Meta{IP: "existing", CSRF: randomToken()}); !errors.Is(e, ErrUnauthenticated) {
		t.Fatal("existing identity csrf", e)
	}
	if _, e = h.s.Run(h.ctx, "public.invitation.accept", session.Cookie, Input{Token: proof}, Meta{IP: "existing", CSRF: session.CSRF}); e != nil {
		t.Fatal(e)
	}
	if r := h.login(t, "existing@example.invalid"); r.Cookie == "" {
		t.Fatal("existing password changed")
	}
}
func TestMailPermanentFailurePauseAndDeadline(t *testing.T) {
	for _, kind := range []string{"permanent", "pause", "expired"} {
		t.Run(kind, func(t *testing.T) {
			h := newHarness(t)
			inv, _ := h.invite(t, kind+"@example.invalid", h.agentRole)
			// Bootstrap verification was already consumed; drain its cancelled delivery.
			if _, e := h.s.DeliverOne(h.ctx, &captureSender{}); e != nil {
				t.Fatal(e)
			}
			sender := &captureSender{}
			if kind == "permanent" {
				sender.err = &MailError{Code: "SMTP_REJECTED", Permanent: true}
			}
			if kind == "pause" {
				sender.err = &MailError{Code: "SMTP_AUTH", Pause: true}
			}
			if kind == "expired" {
				if _, e := h.admin.Exec(h.ctx, "UPDATE app.mail_deliveries SET deadline_at=now()-interval '1 second' WHERE invitation_id=$1", inv); e != nil {
					t.Fatal(e)
				}
			}
			_, e := h.s.DeliverOne(h.ctx, sender)
			if (kind == "pause") != (e != nil) {
				t.Fatal("pause classification", e)
			}
			var state string
			var payload []byte
			if e = h.admin.QueryRow(h.ctx, "SELECT state,payload_ciphertext FROM app.mail_deliveries WHERE invitation_id=$1", inv).Scan(&state, &payload); e != nil {
				t.Fatal(e)
			}
			expected := map[string]string{"permanent": "FAILED", "pause": "RETRY_WAIT", "expired": "EXPIRED"}[kind]
			if state != expected {
				t.Fatal(state)
			}
			if kind != "pause" && payload != nil {
				t.Fatal("terminal secret retained")
			}
			if kind == "expired" && len(sender.messages) != 0 {
				t.Fatal("expired mail sent")
			}
		})
	}
}
func TestLoginLimitCannotBeBypassedWithUnrelatedToken(t *testing.T) {
	h := newHarness(t)
	for i := 0; i < 20; i++ {
		_, e := h.s.Run(h.ctx, "public.login", "", Input{Email: "limited@example.invalid", Password: "wrong", Token: randomToken()}, Meta{IP: id().String()})
		if !errors.Is(e, ErrAuthentication) {
			t.Fatal(e)
		}
	}
	if _, e := h.s.Run(h.ctx, "public.login", "", Input{Email: "limited@example.invalid", Password: "wrong", Token: randomToken()}, Meta{IP: "new"}); !errors.Is(e, ErrRate) {
		t.Fatal("account throttle bypassed", e)
	}
}
