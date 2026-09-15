package meta

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
	"waba.local/control/internal/config"
)

func TestSendTextSingleAttemptClassification(t *testing.T) {
	for _, tc := range []struct {
		name        string
		status      int
		body, state string
	}{
		{"accepted", 200, `{"messages":[{"id":"wamid.synthetic"}]}`, "ACCEPTED"},
		{"missing id", 200, `{"messages":[]}`, "UNCERTAIN"},
		{"malformed", 200, `broken`, "UNCERTAIN"},
		{"server error", 503, `{"error":{"code":2}}`, "UNCERTAIN"},
		{"unknown rejection", 400, `{"error":{"code":999999}}`, "UNCERTAIN"},
		{"permission", 403, `{"error":{"code":10}}`, "REJECTED"},
		{"window", 400, `{"error":{"code":131047}}`, "REJECTED"},
		{"limit", 429, `{"error":{"code":130429}}`, "REJECTED"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != "POST" || r.URL.Path != "/v26.0/333/messages" || r.Header.Get("Authorization") != "Bearer synthetic" {
					t.Error("request boundary")
				}
				var value map[string]any
				if json.NewDecoder(r.Body).Decode(&value) != nil || value["type"] != "text" || value["to"] != "6280000000000" || value["biz_opaque_callback_data"] != "synthetic-intent" {
					t.Error("text payload")
				}
				if _, ok := value["template"]; ok {
					t.Error("template payload")
				}
				if _, ok := value["ttl_seconds"]; ok {
					t.Error("unsupported TTL")
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			g := NewGraph(config.Meta{Version: "v26.0", AccessToken: "synthetic"})
			g.base = server.URL
			got := g.SendText(context.Background(), "333", "6280000000000", "synthetic body", "synthetic-intent")
			if got.State != tc.state || calls.Load() != 1 {
				t.Fatal(got, calls.Load())
			}
		})
	}
}
func TestSendTextTimeoutAndRedirectNeverRetry(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write([]byte(`{"messages":[{"id":"late"}]}`))
	}))
	defer server.Close()
	g := NewGraph(config.Meta{Version: "v26.0", AccessToken: "synthetic"})
	g.base = server.URL
	g.client.Timeout = 10 * time.Millisecond
	if got := g.SendText(context.Background(), "333", "6280000000000", "text", "intent"); got.State != "UNCERTAIN" || calls.Load() != 1 {
		t.Fatal(got)
	}
	var destination atomic.Int32
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { destination.Add(1) }))
	defer other.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL, http.StatusTemporaryRedirect)
	}))
	defer redirect.Close()
	g.base = redirect.URL
	g.client.Timeout = time.Second
	if got := g.SendText(context.Background(), "333", "6280000000000", "text", "intent"); got.State != "UNCERTAIN" || destination.Load() != 0 {
		t.Fatal("redirect followed")
	}
	for _, phone := range []string{"../333", "https://example.invalid", ""} {
		if got := g.SendText(context.Background(), phone, "6280000000000", "text", "intent"); got.State != "REJECTED" {
			t.Fatal(got)
		}
	}
}
