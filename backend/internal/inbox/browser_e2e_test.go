//go:build integration && e2e

package inbox

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	osexec "os/exec"
	"strings"
	"testing"
	"time"
	"waba.local/control/internal/config"
	"waba.local/control/internal/httpapi"
)

func TestBrowserInboxE2E(t *testing.T) {
	h := newHarness(t)
	h.s.Auth.Origin = "http://127.0.0.1:3102"
	conversation := h.ingest(t, "wamid.browser-inbound", "text", time.Now().Add(-time.Minute))
	team := newID()
	h.sql(t, "INSERT INTO app.teams(id,organization_id,name) VALUES($1,$2,'Inbox support')", team, h.org)
	h.sql(t, "INSERT INTO app.team_members(id,organization_id,team_id,member_id) VALUES($1,$2,$3,$4)", newID(), h.org, team, h.member)
	_, colleague, _ := h.secondActor(t, nil, "organization.view", "inbox.view")
	h.sql(t, "INSERT INTO app.inbox_presence(id,organization_id,conversation_id,member_id,expires_at) VALUES($1,$2,$3,$4,now()+interval '3 minutes')", newID(), h.org, conversation, colleague)
	var eid uuid.UUID
	if e := h.db.QueryRow(h.ctx, "SELECT id FROM app.webhook_events LIMIT 1").Scan(&eid); e != nil {
		t.Fatal(e)
	}
	if e := h.s.workerScope(h.ctx, func(tx pgx.Tx, org uuid.UUID) error {
		return h.s.materializeMessage(h.ctx, tx, org, h.phone, eid, encode(map[string]any{"id": "wamid.browser-expired", "from": "6280000000099", "type": "text", "timestamp": fmt.Sprint(time.Now().Add(-25 * time.Hour).Unix()), "text": map[string]string{"body": "Old inbound, expired window"}}), time.Now(), time.Now())
	}); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); _ = h.s.RunWorker(ctx) }()
	defer func() { cancel(); <-done }()
	server := httptest.NewServer(httpapi.Handler(config.Config{MaxBodyBytes: 4 * 1024 * 1024}, slog.New(slog.NewJSONHandler(io.Discard, nil)), func(context.Context) error { return nil }, h.handler))
	defer server.Close()
	command := osexec.CommandContext(ctx, "node", "../../../scripts/inbox-e2e.mjs")
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		switch strings.ToUpper(key) {
		case "PATH", "SYSTEMROOT", "TEMP", "TMP", "HOME", "USERPROFILE", "LOCALAPPDATA":
			command.Env = append(command.Env, entry)
		}
	}
	command.Env = append(command.Env, "E2E_API="+server.URL, "NEXT_TELEMETRY_DISABLED=1")
	output, e := command.CombinedOutput()
	if e != nil {
		t.Fatalf("Inbox browser E2E: %v\n%s", e, output)
	}
	if h.sender.calls.Load() != 1 {
		t.Fatal("expected exactly one synthetic provider attempt", h.sender.calls.Load())
	}
	t.Log(string(output))
}
