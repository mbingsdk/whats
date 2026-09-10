//go:build integration && e2e

package identity

import (
	"context"
	"encoding/json"
	"github.com/pquerna/otp/totp"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	osexec "os/exec"
	"strings"
	"testing"
	"time"
	"waba.local/control/internal/config"
	"waba.local/control/internal/httpapi"
)

func TestBrowserIdentityE2E(t *testing.T) {
	h := newHarness(t)
	h.s.Origin = "http://127.0.0.1:3100"
	sender, mailbox := localSMTP(t, "tls")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for ctx.Err() == nil {
			_, e := h.s.DeliverOne(ctx, sender)
			if e != nil && ctx.Err() == nil {
				t.Log("local mail worker returned an error")
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(40 * time.Millisecond):
			}
		}
	}()
	t.Cleanup(func() { cancel(); <-done })
	mux := http.NewServeMux()
	mux.HandleFunc("POST /__test/totp", func(w http.ResponseWriter, r *http.Request) {
		var v struct{ Secret string }
		_ = json.NewDecoder(r.Body).Decode(&v)
		code, e := totp.GenerateCode(v.Secret, time.Now())
		if e != nil {
			http.Error(w, "invalid", 400)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"code": code})
	})
	mux.HandleFunc("GET /__test/mail", func(w http.ResponseWriter, r *http.Request) { _ = json.NewEncoder(w).Encode(mailbox.all()) })
	cfg := config.Config{MaxBodyBytes: 4096}
	mux.Handle("/", httpapi.Handler(cfg, slog.New(slog.NewJSONHandler(io.Discard, nil)), func(context.Context) error { return nil }, h.s.Handler()))
	server := httptest.NewServer(mux)
	defer server.Close()
	command := osexec.CommandContext(context.Background(), "node", "../../../scripts/e2e.mjs")
	// Frontend/test browser processes receive no database, root-key or SMTP credentials.
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
		t.Fatalf("Browser E2E failed: %v\n%s", e, output)
	}
	t.Log(string(output))
}
