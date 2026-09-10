//go:build integration && controlledsmtp

package identity

import (
	"context"
	"os"
	"testing"
	"time"
	"waba.local/control/internal/config"
)

type controlledSender struct {
	transport SMTPTransport
	recipient string
	t         *testing.T
	sent      int
}

func (s *controlledSender) Send(ctx context.Context, m Mail) error {
	if m.Recipient != s.recipient {
		s.t.Fatal("Controlled test refused a non-designated recipient")
	}
	if e := s.transport.Send(ctx, m); e != nil {
		return e
	}
	s.sent++
	s.t.Logf("SMTP_ACCEPTED purpose=%s delivery_id=%s UTC=%s", m.Subject, m.ID, time.Now().UTC().Format(time.RFC3339))
	return nil
}

// This manually opted-in test never runs in ordinary CI. Its recipient must be
// explicitly approved by the operator. All accounts/databases are disposable.
func TestControlledSMTPMailbox(t *testing.T) {
	recipient, e := CanonicalEmail(os.Getenv("CONTROLLED_SMTP_RECIPIENT"))
	if e != nil {
		t.Fatal("Explicit controlled recipient required")
	}
	c, e := config.Load()
	if e != nil {
		t.Fatal("Controlled SMTP configuration invalid")
	}
	sender := &controlledSender{transport: SMTPTransport{Config: c.SMTP}, recipient: recipient, t: t}
	deliver := func(h *harness) {
		before := sender.sent
		for i := 0; i < 4 && sender.sent == before; i++ {
			worked, e := h.s.DeliverOne(h.ctx, sender)
			if e != nil {
				t.Fatal("Controlled delivery failed:", e)
			}
			if !worked {
				break
			}
		}
		if sender.sent != before+1 {
			t.Fatal("Expected one controlled relay acceptance")
		}
	}
	h := newHarness(t)
	if _, e = h.admin.Exec(h.ctx, "UPDATE app.users SET canonical_email=$2,verified_at=NULL WHERE id=$1", h.user, recipient); e != nil {
		t.Fatal("Fixture setup failed")
	}
	if _, e = h.s.Run(h.ctx, "public.verify.request", "", Input{Email: recipient}, Meta{IP: "controlled"}); e != nil {
		t.Fatal("Verification request failed")
	}
	proof := h.proof(t, "EMAIL_VERIFY", recipient)
	deliver(h)
	if _, e = h.s.Run(h.ctx, "public.verify.complete", "", Input{Token: proof}, Meta{IP: "controlled"}); e != nil {
		t.Fatal("Verification completion failed")
	}
	if _, e = h.s.Run(h.ctx, "public.reset.request", "", Input{Email: recipient}, Meta{IP: "controlled"}); e != nil {
		t.Fatal("Reset request failed")
	}
	proof = h.proof(t, "PASSWORD_RESET", recipient)
	deliver(h)
	if _, e = h.s.Run(h.ctx, "public.reset.complete", "", Input{Token: proof, Password: randomToken()}, Meta{IP: "controlled"}); e != nil {
		t.Fatal("Reset completion failed")
	}
	deliver(h) // Security notice committed with password reset.
	other := newHarness(t)
	_, proof = other.invite(t, recipient, other.agentRole)
	deliver(other)
	if _, e = other.s.Run(other.ctx, "public.invitation.accept", "", Input{Token: proof, Password: randomToken(), Name: "Controlled mailbox test"}, Meta{IP: "controlled"}); e != nil {
		t.Fatal("Invitation completion failed")
	}
	if sender.sent != 4 {
		t.Fatal("Unexpected controlled mail count")
	}
	t.Log("Four identity messages accepted by the configured relay; proofs consumed successfully. Actual mailbox receipt still requires operator confirmation.")
}
