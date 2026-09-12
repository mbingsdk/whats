//go:build integration && controlledmeta

package meta

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"waba.local/control/internal/config"
	"waba.local/control/internal/httpapi"
)

// Explicit operator-only evidence harness. Ordinary CI excludes this build tag.
// It uses a disposable database and the production implementations/limited runtime role.
func TestControlledMeta(t *testing.T) {
	if os.Getenv("META_CONTROLLED_TEST") != "1" {
		t.Fatal("explicit controlled operator run required")
	}
	c, e := config.LoadMeta(os.Getenv)
	if e != nil || !c.Enabled {
		t.Fatal("protected Meta configuration required")
	}
	h := newHarness(t)
	rootFile, e := os.ReadFile(os.Getenv("IDENTITY_ROOT_KEY_FILE"))
	if e != nil {
		t.Fatal("protected root unavailable")
	}
	root, e := base64.StdEncoding.DecodeString(strings.TrimSpace(string(rootFile)))
	if e != nil || len(root) != 32 {
		t.Fatal("protected root invalid")
	}
	h.s.Auth.Root = root
	h.s.Config = c
	h.s.Graph = NewGraph(c)
	h.expect(t, "POST", "/meta/connection", 200)
	h.b, e = h.s.binding(h.ctx)
	if e != nil {
		t.Fatal("binding failed")
	}
	h.expect(t, "POST", "/meta/sync", 200)
	if e = h.s.syncOne(h.ctx, h.b); e != nil {
		t.Fatal("runtime sync unavailable")
	}
	var state string
	if e = h.admin.QueryRow(h.ctx, "SELECT state FROM app.asset_sync_runs WHERE app_id=$1", h.b.App).Scan(&state); e != nil || state != "SUCCEEDED" {
		t.Fatal("runtime sync did not succeed")
	}
	public := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/v1/meta/webhooks/") {
			http.NotFound(w, r)
			return
		}
		h.handler.ServeHTTP(w, r)
	})
	handler := httpapi.Handler(config.Config{MaxBodyBytes: 4 * 1024 * 1024}, slog.New(slog.NewJSONHandler(io.Discard, nil)), func(context.Context) error { return nil }, public)
	listener, e := net.Listen("tcp", "127.0.0.1:8092")
	if e != nil {
		t.Fatal("controlled listener unavailable")
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16384}
	go func() { _ = server.Serve(listener) }()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Minute)
	defer cancel()
	defer server.Close()
	go func() { _ = h.s.RunWorker(ctx) }()
	start := time.Now().UTC()
	path := filepath.Join("..", "..", "..", ".local", "sprint2-controlled-meta.json")
	report := map[string]any{"classification": "CONTROLLED_NONPRODUCTION", "started_at": start, "graph_version": c.Version, "runtime_sync": "SUCCEEDED", "callback_path": "/api/v1/meta/webhooks/" + h.b.Callback.String(), "listener": "http://127.0.0.1:8092", "outbound_sent": false}
	for {
		var received, challenge *time.Time
		var total, processed int
		if e = h.admin.QueryRow(h.ctx, "SELECT challenge_verified_at,last_webhook_at FROM app.meta_apps WHERE id=$1", h.b.App).Scan(&challenge, &received); e != nil {
			t.Fatal("evidence state unavailable")
		}
		if e = h.admin.QueryRow(h.ctx, "SELECT count(*),count(*) FILTER (WHERE state IN ('PROCESSED','UNKNOWN')) FROM app.webhook_events WHERE app_id=$1", h.b.App).Scan(&total, &processed); e != nil {
			t.Fatal("evidence counts unavailable")
		}
		report["last_valid_challenge_at"] = challenge
		report["last_authentic_post_at"] = received
		report["durable_events"] = total
		report["processed_events"] = processed
		report["updated_at"] = time.Now().UTC()
		data, _ := json.MarshalIndent(report, "", "  ")
		if e = os.WriteFile(path, data, 0600); e != nil {
			t.Fatal("evidence write failed")
		}
		select {
		case <-ctx.Done():
			fmt.Println("Controlled callback window closed; consult sanitized evidence.")
			return
		case <-time.After(2 * time.Second):
		}
	}
}
