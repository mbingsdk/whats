package meta

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"waba.local/control/internal/config"
)

func testGraph(t *testing.T, h http.HandlerFunc) *Graph {
	t.Helper()
	server := httptest.NewServer(h)
	t.Cleanup(server.Close)
	g := NewGraph(config.Meta{Version: "v26.0", AccessToken: "synthetic-secret-token"})
	g.base = server.URL
	return g
}
func TestGraphTranslationAndRedaction(t *testing.T) {
	for _, tc := range []struct {
		name         string
		status, code int
		kind         string
		retry        bool
	}{
		{"authentication", 401, 190, "AUTHENTICATION", false}, {"permission", 403, 200, "PERMISSION", false}, {"rate", 429, 4, "RATE_LIMIT", true}, {"server", 503, 2, "TRANSIENT", true}, {"invalid", 400, 100, "INVALID_REQUEST", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := testGraph(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" || r.Header.Get("Authorization") != "Bearer synthetic-secret-token" {
					t.Error("authorization/method")
				}
				w.Header().Set("x-fb-request-id", "safe-trace")
				w.WriteHeader(tc.status)
				fmt.Fprintf(w, `{"error":{"code":%d,"message":"synthetic-secret-token"}}`, tc.code)
			})
			_, e := g.get(context.Background(), "123", nil)
			var ge *GraphError
			if !errors.As(e, &ge) || ge.Kind != tc.kind || ge.Retryable != tc.retry || ge.RequestID != "safe-trace" {
				t.Fatalf("classification: %v", e)
			}
			if strings.Contains(fmt.Sprintf("%v %v", g, e), "synthetic-secret-token") {
				t.Fatal("credential leaked")
			}
		})
	}
}
func TestGraphPaginationAndBoundary(t *testing.T) {
	for _, mode := range []string{"valid", "foreign", "loop", "oversize", "version", "malformed", "redirect"} {
		t.Run(mode, func(t *testing.T) {
			count := 0
			g := testGraph(t, func(w http.ResponseWriter, r *http.Request) {
				count++
				switch mode {
				case "oversize":
					fmt.Fprint(w, strings.Repeat("x", 2*1024*1024+1))
					return
				case "version":
					w.Header().Set("facebook-api-version", "v99.0")
					fmt.Fprint(w, `{"data":[]}`)
					return
				case "malformed":
					fmt.Fprint(w, "oops")
					return
				case "redirect":
					http.Redirect(w, r, "https://example.invalid", 302)
					return
				}
				if mode == "valid" && count == 2 {
					if r.URL.Query().Get("fields") != "id" || r.URL.Query().Get("access_token") != "" {
						t.Error("pagination parameters")
					}
					fmt.Fprint(w, `{"data":[{"id":"2"}]}`)
					return
				}
				host := "http://" + r.Host
				if mode == "foreign" {
					host = "https://example.invalid"
				}
				fmt.Fprintf(w, `{"data":[{"id":"1"}],"paging":{"next":%q}}`, host+"/v26.0/123/phone_numbers?after=cursor&access_token=untrusted")
			})
			data, e := g.pages(context.Background(), "123/phone_numbers", url.Values{"fields": {"id"}})
			if mode == "valid" {
				if e != nil || len(data) != 2 {
					t.Fatalf("pagination: %v", e)
				}
			} else if e == nil {
				t.Fatal("unsafe response accepted")
			}
		})
	}
}
func TestGraphTransport(t *testing.T) {
	g := testGraph(t, func(http.ResponseWriter, *http.Request) {})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, e := g.get(ctx, "123", nil)
	var ge *GraphError
	if !errors.As(e, &ge) || ge.Kind != "TRANSPORT" {
		t.Fatal(e)
	}
}
