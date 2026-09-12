// Package meta owns the read-only Graph boundary and authenticated ingestion.
package meta

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"waba.local/control/internal/config"
)

type GraphError struct {
	Kind                  string
	Status, Code, Subcode int
	Retryable             bool
	RequestID             string
}

func (e *GraphError) Error() string { return "META_" + e.Kind } // Never retain provider messages or URLs.
type Graph struct {
	base    string
	version string
	token   config.Secret
	client  *http.Client
}

func NewGraph(c config.Meta) *Graph {
	return &Graph{base: "https://graph.facebook.com", version: c.Version, token: c.AccessToken, client: &http.Client{Timeout: 25 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirect rejected") }}}
}

var numericID = regexp.MustCompile(`^[0-9]{1,32}$`)
var diagnosticID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

func safeDiagnostic(v string) string {
	if diagnosticID.MatchString(v) {
		return v
	}
	return ""
}
func (g *Graph) get(ctx context.Context, path string, q url.Values) (json.RawMessage, error) {
	u := g.base + "/" + g.version + "/" + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	r, e := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if e != nil {
		return nil, &GraphError{Kind: "INVALID_REQUEST"}
	}
	r.Header.Set("Authorization", "Bearer "+string(g.token))
	r.Header.Set("Accept", "application/json")
	resp, e := g.client.Do(r)
	if e != nil {
		return nil, &GraphError{Kind: "TRANSPORT", Retryable: true}
	}
	defer resp.Body.Close()
	b, e := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024+1))
	if e != nil {
		return nil, &GraphError{Kind: "TRANSPORT", Retryable: true}
	}
	if len(b) > 2*1024*1024 {
		return nil, &GraphError{Kind: "RESPONSE_TOO_LARGE"}
	}
	var envelope struct {
		Error *struct {
			Code      int  `json:"code"`
			Subcode   int  `json:"error_subcode"`
			Transient bool `json:"is_transient"`
		} `json:"error"`
	}
	if json.Unmarshal(b, &envelope) != nil {
		return nil, &GraphError{Kind: "INVALID_RESPONSE", Status: resp.StatusCode}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || envelope.Error != nil {
		v := &GraphError{Kind: "DOMAIN", Status: resp.StatusCode, RequestID: safeDiagnostic(resp.Header.Get("x-fb-request-id"))}
		if envelope.Error != nil {
			v.Code = envelope.Error.Code
			v.Subcode = envelope.Error.Subcode
		}
		switch {
		case resp.StatusCode == 401 || v.Code == 190 || v.Code == 102:
			v.Kind = "AUTHENTICATION"
		case v.Code == 10 || v.Code == 200 || resp.StatusCode == 403:
			v.Kind = "PERMISSION"
		case resp.StatusCode == 429 || v.Code == 4 || v.Code == 17 || v.Code == 32 || v.Code == 613:
			v.Kind = "RATE_LIMIT"
			v.Retryable = true
		case resp.StatusCode >= 500 || (envelope.Error != nil && envelope.Error.Transient):
			v.Kind = "TRANSIENT"
			v.Retryable = true
		case resp.StatusCode == 400:
			v.Kind = "INVALID_REQUEST"
		}
		return nil, v
	}
	if version := resp.Header.Get("facebook-api-version"); version != "" && version != g.version {
		return nil, &GraphError{Kind: "VERSION_MISMATCH"}
	}
	return b, nil
}
func (g *Graph) pages(ctx context.Context, path string, q url.Values) ([]json.RawMessage, error) {
	if q == nil {
		q = url.Values{}
	}
	var out []json.RawMessage
	seen := map[string]bool{}
	for page := 0; page < 100; page++ {
		b, e := g.get(ctx, path, q)
		if e != nil {
			return nil, e
		}
		var v struct {
			Data   []json.RawMessage `json:"data"`
			Paging struct {
				Next string `json:"next"`
			} `json:"paging"`
		}
		if json.Unmarshal(b, &v) != nil || v.Data == nil {
			return nil, &GraphError{Kind: "INVALID_RESPONSE"}
		}
		out = append(out, v.Data...)
		if len(out) > 10000 {
			return nil, &GraphError{Kind: "PAGINATION_LIMIT"}
		}
		if v.Paging.Next == "" {
			return out, nil
		}
		next, e := url.Parse(v.Paging.Next)
		base, _ := url.Parse(g.base)
		if e != nil || next.Scheme != base.Scheme || next.Host != base.Host || next.User != nil || next.Fragment != "" || next.Path != "/"+g.version+"/"+path {
			return nil, &GraphError{Kind: "UNSAFE_PAGINATION"}
		}
		// Only a cursor is accepted; preserve the original fields and authorization.
		after := next.Query().Get("after")
		if after == "" || len(after) > 4096 || seen[after] {
			return nil, &GraphError{Kind: "PAGINATION_LOOP"}
		}
		seen[after] = true
		q.Set("after", after)
	}
	return nil, &GraphError{Kind: "PAGINATION_LIMIT"}
}

type WABA struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	TimezoneID string `json:"timezone_id"`
}
type Phone struct {
	ID               string `json:"id"`
	Name             string `json:"verified_name"`
	Number           string `json:"display_phone_number"`
	Quality          string `json:"quality_rating"`
	Platform         string `json:"platform_type"`
	CodeVerification string `json:"code_verification_status"`
}
type Snapshot struct {
	WABA       WABA
	Phones     []Phone
	Profiles   map[string]json.RawMessage
	Subscribed bool
}

func (g *Graph) Snapshot(ctx context.Context, waba, app string) (Snapshot, error) {
	v := Snapshot{Profiles: map[string]json.RawMessage{}}
	if !numericID.MatchString(waba) || !numericID.MatchString(app) {
		return v, &GraphError{Kind: "INVALID_REQUEST"}
	}
	b, e := g.get(ctx, waba, url.Values{"fields": {"id,name,timezone_id"}})
	if e != nil {
		return v, e
	}
	// timezone_id can be either a JSON string or number across Graph revisions.
	var raw map[string]json.RawMessage
	if json.Unmarshal(b, &raw) != nil || json.Unmarshal(raw["id"], &v.WABA.ID) != nil || v.WABA.ID != waba {
		return v, &GraphError{Kind: "INVALID_RESPONSE"}
	}
	_ = json.Unmarshal(raw["name"], &v.WABA.Name)
	v.WABA.TimezoneID = strings.Trim(string(raw["timezone_id"]), "\"")
	phones, e := g.pages(ctx, waba+"/phone_numbers", nil)
	if e != nil {
		return v, e
	}
	seen := map[string]bool{}
	for _, b := range phones {
		var p Phone
		if json.Unmarshal(b, &p) != nil || !numericID.MatchString(p.ID) || seen[p.ID] {
			return v, &GraphError{Kind: "INVALID_RESPONSE"}
		}
		seen[p.ID] = true
		v.Phones = append(v.Phones, p)
		profiles, e := g.pages(ctx, p.ID+"/whatsapp_business_profile", url.Values{"fields": {"about,address,description,email,profile_picture_url,websites,vertical"}})
		if e != nil {
			return v, e
		}
		if len(profiles) != 1 {
			return v, &GraphError{Kind: "INVALID_RESPONSE"}
		}
		// Store only the requested business profile surface; unknown fields never become UI output.
		var fields map[string]json.RawMessage
		if json.Unmarshal(profiles[0], &fields) != nil {
			return v, &GraphError{Kind: "INVALID_RESPONSE"}
		}
		filtered := map[string]json.RawMessage{}
		for _, key := range strings.Split("about,address,description,email,profile_picture_url,websites,vertical,messaging_product", ",") {
			if value, ok := fields[key]; ok {
				filtered[key] = value
			}
		}
		v.Profiles[p.ID], _ = json.Marshal(filtered)
	}
	subscriptions, e := g.pages(ctx, waba+"/subscribed_apps", nil)
	if e != nil {
		return v, e
	}
	for _, b := range subscriptions {
		var s struct {
			App struct {
				ID string `json:"id"`
			} `json:"whatsapp_business_api_data"`
		}
		if json.Unmarshal(b, &s) != nil || s.App.ID == "" {
			return v, &GraphError{Kind: "INVALID_RESPONSE"}
		}
		if s.App.ID == app {
			v.Subscribed = true
		}
	}
	return v, nil
}
func errorCode(e error) string {
	var g *GraphError
	if errors.As(e, &g) {
		return g.Error()
	}
	return "INTERNAL_PROCESSING"
}
func (g *Graph) String() string {
	return fmt.Sprintf("Meta Graph %s [credentials redacted]", g.version)
}
