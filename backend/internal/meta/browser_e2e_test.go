//go:build integration && e2e

package meta

import (
	"context"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	osexec "os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"waba.local/control/internal/config"
	"waba.local/control/internal/httpapi"
)

func TestBrowserMetaE2E(t *testing.T) {
	h := newHarness(t)
	h.s.Auth.Origin = "http://127.0.0.1:3101"
	var absent, failed atomic.Bool
	h.graph(t, &absent, &failed)
	h.expect(t, "POST", "/meta/sync", 200)
	if e := h.s.syncOne(h.ctx, h.b); e != nil {
		t.Fatal(e)
	}
	if w := h.deliver(inbound, []string{signature([]byte(inbound), "synthetic-app-secret")}); w.Code != 200 {
		t.Fatal("ingress")
	}
	if e := h.s.processOne(h.ctx, h.b); e != nil {
		t.Fatal(e)
	}
	server := httptest.NewServer(httpapi.Handler(config.Config{MaxBodyBytes: 4 * 1024 * 1024}, slog.New(slog.NewJSONHandler(io.Discard, nil)), func(context.Context) error { return nil }, h.handler))
	defer server.Close()
	command := osexec.CommandContext(context.Background(), "node", "../../../scripts/meta-e2e.mjs")
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
		t.Fatalf("Meta browser E2E: %v\n%s", e, output)
	}
	t.Log(string(output))
}
