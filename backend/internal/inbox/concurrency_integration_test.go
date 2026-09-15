//go:build integration

package inbox

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCompetingAuthorizationAndChangedScope(t *testing.T) {
	h := newHarness(t)
	c := h.ingest(t, "wamid.compete", "text", time.Now().Add(-time.Minute))
	base := map[string]any{"conversation_id": c, "assignment_revision": 1, "text": "original"}
	pre := h.expect(t, "POST", "/pricing/preflight", base, 200)
	result := make(chan int, 2)
	for _, key := range []string{"synthetic-competing-a", "synthetic-competing-b"} {
		in := map[string]any{"conversation_id": c, "assignment_revision": 1, "text": "original", "authorization": pre["authorization"], "client_idempotency_key": key}
		go func() { result <- h.request("POST", "/outbound-intents", in).Code }()
	}
	first, second := <-result, <-result
	if !((first == 200 && second == 409) || (first == 409 && second == 200)) {
		t.Fatal(first, second)
	}
	var key string
	if e := h.db.QueryRow(h.ctx, "SELECT client_key FROM app.outbound_intents").Scan(&key); e != nil {
		t.Fatal(e)
	}
	changed := map[string]any{"conversation_id": c, "assignment_revision": 1, "text": "different", "authorization": pre["authorization"], "client_idempotency_key": key}
	out := h.expect(t, "POST", "/outbound-intents", changed, 409)
	if out["error"].(map[string]any)["code"] != "INTENT_SCOPE_CHANGED" {
		t.Fatal(out)
	}
	changed["text"] = "original"
	changed["category"] = "MARKETING"
	changed["message_type"] = "TEMPLATE"
	h.expect(t, "POST", "/outbound-intents", changed, 409)
	var count int
	if e := h.db.QueryRow(h.ctx, "SELECT count(*) FROM app.outbound_intents").Scan(&count); e != nil || count != 1 {
		t.Fatal(count, e)
	}
}
func TestIndependentHumanIntentsIgnoreHandoffEpochQuota(t *testing.T) {
	h := newHarness(t)
	c := h.ingest(t, "wamid.two-agent", "text", time.Now().Add(-time.Minute))
	actor, _, _ := h.secondActor(t, nil, "organization.view", "inbox.view", "messages.send")
	other := *h
	other.owner = actor
	h.sql(t, "UPDATE app.conversations SET handoff_state='HUMAN',handoff_epoch=21")
	prepare := func(v *harness, key string) map[string]any {
		in := map[string]any{"conversation_id": c, "assignment_revision": 1, "text": key}
		pre := v.expect(t, "POST", "/pricing/preflight", in, 200)
		in["authorization"] = pre["authorization"]
		in["client_idempotency_key"] = key
		return in
	}
	a, b := prepare(h, "synthetic-agent-a"), prepare(&other, "synthetic-agent-b")
	h.sql(t, "UPDATE app.conversations SET handoff_epoch=22")
	result := make(chan int, 2)
	go func() { result <- h.request("POST", "/outbound-intents", a).Code }()
	go func() { result <- other.request("POST", "/outbound-intents", b).Code }()
	if x, y := <-result, <-result; x != 200 || y != 200 {
		t.Fatal(x, y)
	}
	h.submit(t, c, "synthetic-agent-followup", "a second human message")
	var count int
	if e := h.db.QueryRow(h.ctx, "SELECT count(*) FROM app.outbound_intents").Scan(&count); e != nil || count != 3 {
		t.Fatal(count, e)
	}
	// The explicit acceptance policy permits one provider attempt per phone.
	// It is independent of human intent eligibility and handoff epochs.
	if e := h.s.DispatchOnce(h.ctx); e != nil {
		t.Fatal(e)
	}
	if e := h.s.DispatchOnce(h.ctx); e != nil {
		t.Fatal(e)
	}
	if h.sender.calls.Load() != 1 {
		t.Fatal("controlled acceptance cap")
	}
}
func TestDispatchWaitsForCommittedDeny(t *testing.T) {
	for _, tc := range []struct{ name, query, code string }{
		{"assignment", "UPDATE app.conversations SET assignment_revision=assignment_revision+1", "ASSIGNMENT_CHANGED"},
		{"policy", "UPDATE app.inbox_settings SET policy_revision=policy_revision+1", "PRICING_POLICY_CHANGED"},
		{"recipient", "UPDATE app.recipient_identities SET suppressed=true", "RECIPIENT_SUPPRESSED"},
		{"expiry", "UPDATE app.conversations SET window_expires_at=now()-interval '1 second'", "TEMPLATE_REQUIRED"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			c := h.ingest(t, "wamid.lock", "text", time.Now().Add(-time.Minute))
			h.submit(t, c, "synthetic-contended-intent", "reply")
			tx, e := h.db.Begin(h.ctx)
			if e != nil {
				t.Fatal(e)
			}
			defer rollback(tx)
			if _, e = tx.Exec(h.ctx, tc.query); e != nil {
				t.Fatal(e)
			}
			done := make(chan error, 1)
			go func() { done <- h.s.DispatchOnce(h.ctx) }()
			select {
			case e := <-done:
				t.Fatalf("dispatch did not wait for deny transaction: %v", e)
			case <-time.After(80 * time.Millisecond):
			}
			if e = tx.Commit(h.ctx); e != nil {
				t.Fatal(e)
			}
			select {
			case e = <-done:
				if e != nil {
					t.Fatal(e)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("dispatch failed to release locks")
			}
			var code string
			if e = h.db.QueryRow(h.ctx, "SELECT error_code FROM app.outbound_intents").Scan(&code); e != nil || code != tc.code || h.sender.calls.Load() != 0 {
				t.Fatal(code, e)
			}
		})
	}
}
func TestNotesMentionsRedactionAndSnooze(t *testing.T) {
	h := newHarness(t)
	c := h.ingest(t, "wamid.notes", "text", time.Now().Add(-time.Minute))
	data := h.expect(t, "POST", "/conversations/"+c.String()+"/notes", map[string]any{"body": "PRIVATE_SYNTHETIC_NOTE", "mentions": []uuid.UUID{h.member}}, 200)
	id := data["id"].(string)
	h.expect(t, "PATCH", "/notes/"+id, map[string]any{"revision": 1, "body": "PRIVATE_EDITED_NOTE"}, 200)
	h.expect(t, "PATCH", "/notes/"+id, map[string]any{"revision": 1, "body": "stale"}, 409)
	h.mfa(t, h.user)
	h.expect(t, "POST", "/notes/"+id+"/redact", map[string]any{"revision": 2}, 200)
	var body, log string
	if e := h.db.QueryRow(h.ctx, "SELECT body FROM app.internal_notes WHERE id=$1", id).Scan(&body); e != nil || body != "" {
		t.Fatal("redaction", e)
	}
	if e := h.db.QueryRow(h.ctx, "SELECT string_agg(row_to_json(a)::text,'') FROM app.audit_log a").Scan(&log); e != nil || strings.Contains(log, "PRIVATE_") {
		t.Fatal("audit body leak", e)
	}
	view := h.expect(t, "GET", "/conversations/"+c.String(), nil, 200)
	revision := view["conversation"].([]any)[0].(map[string]any)["revision"]
	h.expect(t, "PATCH", "/conversations/"+c.String(), map[string]any{"revision": revision, "status": "SNOOZED", "snoozed_until": time.Now().Add(time.Hour)}, 200)
	view = h.expect(t, "GET", "/conversations/"+c.String(), nil, 200)
	revision = view["conversation"].([]any)[0].(map[string]any)["revision"]
	h.expect(t, "PATCH", "/conversations/"+c.String(), map[string]any{"revision": revision, "priority": "HIGH"}, 200)
	var until *time.Time
	if e := h.db.QueryRow(h.ctx, "SELECT snoozed_until FROM app.conversations").Scan(&until); e != nil || until == nil {
		t.Fatal("priority erased snooze", e)
	}
	h.ingest(t, "wamid.wakes", "text", time.Now().Add(-time.Second))
	var state string
	if e := h.db.QueryRow(h.ctx, "SELECT status FROM app.conversations").Scan(&state); e != nil || state != "OPEN" {
		t.Fatal("snooze failed to wake", e)
	}
}
func TestAuthenticIngressThroughBothWorkers(t *testing.T) {
	h := newHarness(t)
	body := fmt.Sprintf(`{"object":"whatsapp_business_account","entry":[{"id":"222","changes":[{"field":"messages","value":{"messaging_product":"whatsapp","metadata":{"phone_number_id":"333"},"messages":[{"id":"wamid.signed-pipeline","from":"6280000000000","type":"text","timestamp":"%d","text":{"body":"synthetic signed pipeline"}}]}}]}]}`, time.Now().Add(-time.Second).Unix())
	mac := hmac.New(sha256.New, []byte("synthetic-app-secret"))
	_, _ = mac.Write([]byte(body))
	deliver := func(signature string) int {
		r := httptest.NewRequest("POST", "/api/v1/meta/webhooks/"+h.callback.String(), strings.NewReader(body))
		r.Header.Set("X-Hub-Signature-256", signature)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.handler.ServeHTTP(w, r)
		return w.Code
	}
	if code := deliver("sha256=" + strings.Repeat("0", 64)); code != 401 && code != 403 {
		t.Fatal("forged webhook accepted", code)
	}
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if code := deliver(signature); code != 200 {
		t.Fatal("signed ingress", code)
	}
	if code := deliver(signature); code != 200 {
		t.Fatal("duplicate ingress", code)
	}
	ctx, cancel := context.WithCancel(h.ctx)
	done := make(chan struct{})
	go func() { defer close(done); _ = h.s.Meta.RunWorker(ctx) }()
	defer func() { cancel(); <-done }()
	deadline := time.Now().Add(5 * time.Second)
	for {
		e := h.s.MaterializeOnce(h.ctx)
		if e == nil {
			break
		}
		if e != pgx.ErrNoRows {
			t.Fatal(e)
		}
		if time.Now().After(deadline) {
			t.Fatal("pipeline did not materialize")
		}
		time.Sleep(20 * time.Millisecond)
	}
	var messages, events, sequence int
	if e := h.db.QueryRow(h.ctx, "SELECT (SELECT count(*) FROM app.messages),(SELECT count(*) FROM app.webhook_events),inbound_sequence FROM app.conversations").Scan(&messages, &events, &sequence); e != nil || messages != 1 || events != 1 || sequence != 1 {
		t.Fatal(messages, events, sequence, e)
	}
}

func TestRegistryStaleBaseAndReviewHash(t *testing.T) {
	h := newHarness(t)
	h.mfa(t, h.user)
	reviewer, _, uid := h.secondActor(t, nil, "pricing.registry.review")
	h.mfa(t, uid)
	owner := h.owner
	review := func() string {
		id := h.expect(t, "POST", "/rate-cards/imports", map[string]any{"source": "GATE_C_20260914"}, 200)["id"].(string)
		h.sql(t, "UPDATE app.rate_imports SET next_review_at=now()+interval '1 day' WHERE id=$1", id)
		for _, action := range []string{"validate", "diff", "submit"} {
			h.expect(t, "POST", "/rate-cards/imports/"+id+"/"+action, map[string]any{"revision": 1}, 200)
		}
		h.owner = reviewer
		h.expect(t, "POST", "/rate-cards/imports/"+id+"/review", map[string]any{"revision": 1, "approve": true}, 200)
		h.owner = owner
		return id
	}
	a, b := review(), review()
	h.sql(t, "UPDATE app.inbox_settings SET billing_currency_state='VERIFIED',billing_currency='USD',currency_evidence='SYNTHETIC VERIFIED CURRENCY'")
	h.expect(t, "POST", "/rate-cards/imports/"+a+"/publish", map[string]any{"revision": 1}, 200)
	got := h.expect(t, "POST", "/rate-cards/imports/"+b+"/publish", map[string]any{"revision": 1}, 409)
	if got["error"].(map[string]any)["code"] != "STALE_RATE_BASE" {
		t.Fatal(got)
	}
	h.sql(t, "UPDATE app.rate_imports SET correction_reason='Modified after review' WHERE id=$1", b)
	h.expect(t, "POST", "/rate-cards/imports/"+b+"/publish", map[string]any{"revision": 1}, 409)
	h.expect(t, "POST", "/rate-cards/imports", map[string]any{"source": "GATE_C_20260914"}, 422)
	h.expect(t, "POST", "/rate-cards/imports", map[string]any{"source": "GATE_C_20260914", "correction_reason": "Re-review unchanged official evidence for a new publication"}, 200)
}
func TestCallbackAndWindowEvidenceStayBound(t *testing.T) {
	h := newHarness(t)
	at := time.Now().Add(-2 * time.Minute)
	c := h.ingest(t, "wamid.window-one", "text", at)
	later := at.Add(time.Minute)
	h.ingest(t, "wamid.window-two", "text", later)
	var before time.Time
	if e := h.db.QueryRow(h.ctx, "SELECT window_expires_at FROM app.conversations").Scan(&before); e != nil || before.Unix() != later.Add(24*time.Hour).Unix() {
		t.Fatal(before, e)
	}
	h.submit(t, c, "synthetic-window-outbound", "reply")
	if e := h.s.DispatchOnce(h.ctx); e != nil {
		t.Fatal(e)
	}
	var after time.Time
	if e := h.db.QueryRow(h.ctx, "SELECT window_expires_at FROM app.conversations").Scan(&after); e != nil || !after.Equal(before) {
		t.Fatal("outbound extended window", e)
	}
	var proof string
	if e := h.db.QueryRow(h.ctx, "SELECT window_evidence->>'message_id' FROM app.pricing_authorizations LIMIT 1").Scan(&proof); e != nil || proof == "" {
		t.Fatal("missing frozen window", e)
	}
	h.sql(t, "UPDATE app.meta_apps SET challenge_verified_at=NULL")
	got := h.expect(t, "POST", "/pricing/preflight", map[string]any{"conversation_id": c, "assignment_revision": 1, "text": "must block"}, 200)
	if got["decision"].(map[string]any)["allowed"] != false {
		t.Fatal("missing callback allowed")
	}
}
