package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"waba.local/control/internal/config"
)

func TestHealthAndErrors(t *testing.T) {
	c := config.Config{MaxBodyBytes: 32}
	var logs bytes.Buffer
	handler := Handler(c, slog.New(slog.NewJSONHandler(&logs, nil)), func(context.Context) error { return errors.New("postgres://secret-must-not-leak") })
	for _, tt := range []struct {
		path   string
		status int
		code   string
	}{{"/healthz", 200, ""}, {"/readyz", 503, "DEPENDENCY_UNAVAILABLE"}, {"/api/v1/missing?token=do-not-log", 404, "RESOURCE_NOT_FOUND"}} {
		req := httptest.NewRequest("GET", tt.path, nil)
		req.Header.Set("X-Request-ID", "request-123")
		req.Header.Set("Authorization", "Bearer do-not-log")
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != tt.status || res.Header().Get("X-Request-ID") != "request-123" {
			t.Fatalf("response: %d %s", res.Code, res.Body.String())
		}
		if res.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Fatal("broad CORS")
		}
		if tt.code != "" {
			var e ErrorEnvelope
			if err := json.Unmarshal(res.Body.Bytes(), &e); err != nil {
				t.Fatal(err)
			}
			if e.Error.Code != tt.code || e.Error.RequestID != "request-123" {
				t.Fatal(e)
			}
		}
		if strings.Contains(res.Body.String(), "secret-must-not-leak") {
			t.Fatal("dependency error leaked")
		}
	}
	if strings.Contains(logs.String(), "do-not-log") {
		t.Fatal("query/header leaked")
	}
	req := httptest.NewRequest("POST", "/healthz", strings.NewReader(strings.Repeat("x", 33)))
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != 413 {
		t.Fatal(res.Code)
	}
	req = httptest.NewRequest("GET", "/healthz", nil)
	req.Header.Set("X-Request-ID", "bad id")
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Header().Get("X-Request-ID") == "bad id" || res.Header().Get("X-Request-ID") == "" {
		t.Fatal("unvalidated request ID")
	}
}
func TestReadinessAvailable(t *testing.T) {
	handler := Handler(config.Config{MaxBodyBytes: 1}, slog.New(slog.NewJSONHandler(io.Discard, nil)), func(context.Context) error { return nil })
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/readyz", nil))
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
}
func TestGracefulShutdown(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })}
	done := make(chan error, 1)
	go func() { done <- Serve(ctx, server, listener, time.Second) }()
	client := &http.Client{Timeout: time.Second}
	res, err := client.Get("http://" + listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	cancel()
	select {
	case err = <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server failed to stop")
	}
}
func TestReadinessCancellation(t *testing.T) {
	handler := Handler(config.Config{MaxBodyBytes: 1}, slog.New(slog.NewJSONHandler(io.Discard, nil)), func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/readyz", nil).WithContext(ctx))
	if w.Code != 503 {
		t.Fatal(w.Code)
	}
}
