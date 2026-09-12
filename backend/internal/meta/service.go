package meta

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"log/slog"
	"net/http"
	"time"
	"waba.local/control/internal/config"
	"waba.local/control/internal/identity"
)

type Service struct {
	Config config.Meta
	Graph  *Graph
	Pool   *pgxpool.Pool
	Auth   *identity.Service
}
type binding struct{ Org, App, Callback uuid.UUID }

func New(c config.Meta, pool *pgxpool.Pool, auth *identity.Service) *Service {
	return &Service{c, NewGraph(c), pool, auth}
}
func rollback(tx pgx.Tx) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = tx.Rollback(ctx)
}
func (s *Service) binding(ctx context.Context) (binding, error) {
	var b binding
	e := s.Pool.QueryRow(ctx, "SELECT organization_id,id,callback_key FROM app.meta_binding($1)", s.Config.AppID).Scan(&b.Org, &b.App, &b.Callback)
	return b, e
}

// Only trusted configured-App routing enters this worker/ingress scope, never a payload organization ID.
func (s *Service) scoped(ctx context.Context, b binding, fn func(pgx.Tx) error) error {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer rollback(tx)
	if _, e = tx.Exec(ctx, "SELECT set_config('app.organization_id',$1,true)", b.Org.String()); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "SET LOCAL lock_timeout='3s'"); e != nil {
		return e
	}
	if e = fn(tx); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func audit(ctx context.Context, tx pgx.Tx, org uuid.UUID, user *uuid.UUID, action string, id uuid.UUID, request string) error {
	kind := "SYSTEM"
	if user != nil {
		kind = "USER"
	}
	_, e := tx.Exec(ctx, "INSERT INTO app.audit_log(id,organization_id,actor_user_id,actor_kind,action,resource_id,request_id) VALUES($1,$2,$3,$4,$5,$6,$7)", newID(), org, user, kind, action, id, request)
	return e
}

type route struct {
	Method, Path, Permission string
	Write                    bool
}

var Routes = []route{
	{"GET", "/meta/connection", "meta.view", false}, {"POST", "/meta/connection", "meta.manage", true},
	{"POST", "/meta/sync", "meta.manage", true}, {"GET", "/meta/sync-runs", "meta.view", false},
	{"GET", "/meta/wabas", "meta.view", false}, {"GET", "/meta/phone-numbers", "meta.view", false},
	{"GET", "/meta/business-profiles", "meta.view", false}, {"GET", "/meta/health", "meta.view", false},
	{"GET", "/meta/webhook-events", "webhooks.view", false}, {"GET", "/meta/webhook-events/{id}", "webhooks.view", false},
	{"GET", "/meta/webhook-events/{id}/payload", "webhooks.payload.view", false}, {"POST", "/meta/webhook-events/{id}/replay", "webhooks.replay", true},
}

func (s *Service) Handler(fallback http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/meta/webhooks/{callback}", s.webhook)
	mux.HandleFunc("POST /api/v1/meta/webhooks/{callback}", s.webhook)
	for _, rt := range Routes {
		mux.HandleFunc(rt.Method+" /api/v1"+rt.Path, func(w http.ResponseWriter, r *http.Request) {
			s.Auth.Authorized(w, r, rt.Permission, rt.Write, func(ctx context.Context, tx pgx.Tx, v identity.Session) (any, error) {
				return s.operation(ctx, tx, v, r, rt)
			})
		})
	}
	mux.Handle("/", fallback)
	return mux
}
func rows(ctx context.Context, tx pgx.Tx, query string, args ...any) ([]map[string]any, error) {
	r, e := tx.Query(ctx, query, args...)
	if e != nil {
		return nil, e
	}
	v, e := pgx.CollectRows(r, pgx.RowToMap)
	if v == nil {
		v = []map[string]any{}
	}
	for _, row := range v {
		for k, item := range row {
			if id, ok := item.([16]byte); ok {
				row[k] = uuid.UUID(id).String()
			}
		}
	}
	return v, e
}
func (s *Service) operation(ctx context.Context, tx pgx.Tx, v identity.Session, r *http.Request, rt route) (any, error) {
	org := *v.OrganizationID
	request := r.Header.Get("X-Request-ID")
	if !diagnosticID.MatchString(request) {
		request = uuid.NewString()
	}
	if rt.Write && r.Body != nil && r.ContentLength != 0 {
		var empty map[string]json.RawMessage
		decoder := json.NewDecoder(io.LimitReader(r.Body, 1025))
		if decoder.Decode(&empty) != nil || empty == nil || len(empty) != 0 {
			return nil, identity.ErrValidation
		}
		var extra any
		if decoder.Decode(&extra) != io.EOF {
			return nil, identity.ErrValidation
		}
	}
	if rt.Method == "POST" && rt.Path == "/meta/connection" {
		if !s.Config.Enabled {
			return nil, identity.ErrValidation
		}
		id := newID()
		_, e := tx.Exec(ctx, `INSERT INTO app.meta_apps(id,organization_id,external_id,callback_key,graph_version,configured_waba_id) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(external_id) DO NOTHING`, id, org, s.Config.AppID, newID(), s.Config.Version, s.Config.WABAID)
		if e != nil {
			return nil, e
		}
		if e = tx.QueryRow(ctx, "SELECT id FROM app.meta_apps WHERE external_id=$1 AND configured_waba_id=$2", s.Config.AppID, s.Config.WABAID).Scan(&id); e != nil {
			return nil, identity.ErrConflict
		}
		if e = audit(ctx, tx, org, &v.UserID, "meta.connection.bind", id, request); e != nil {
			return nil, e
		}
		return map[string]any{"id": id, "bound": true}, nil
	}
	if rt.Method == "POST" && rt.Path == "/meta/sync" {
		var app uuid.UUID
		if e := tx.QueryRow(ctx, "SELECT id FROM app.meta_apps WHERE external_id=$1 AND configured_waba_id=$2", s.Config.AppID, s.Config.WABAID).Scan(&app); e != nil {
			return nil, identity.ErrNotFound
		}
		id := newID()
		_, e := tx.Exec(ctx, `INSERT INTO app.asset_sync_runs(id,organization_id,app_id,request_id,requested_by,graph_version) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT (organization_id,app_id) WHERE state IN ('QUEUED','RUNNING') DO NOTHING`, id, org, app, request, v.UserID, s.Config.Version)
		if e != nil {
			return nil, e
		}
		if e = tx.QueryRow(ctx, "SELECT id FROM app.asset_sync_runs WHERE app_id=$1 AND state IN ('QUEUED','RUNNING')", app).Scan(&id); e != nil {
			return nil, e
		}
		if e = audit(ctx, tx, org, &v.UserID, "meta.sync.request", id, request); e != nil {
			return nil, e
		}
		return map[string]any{"id": id, "queued": true}, nil
	}
	if rt.Path == "/meta/health" {
		a, e := rows(ctx, tx, "SELECT id,graph_version,credential_state,last_sync_at,last_error,challenge_verified_at,last_webhook_at FROM app.meta_apps")
		if e != nil {
			return nil, e
		}
		counts, e := rows(ctx, tx, "SELECT state,count(*) AS count FROM app.webhook_events GROUP BY state ORDER BY state")
		return map[string]any{"connections": a, "processing": counts, "runtime_configured": s.Config.Enabled, "scope": "Meta state and local processing are separate observations"}, e
	}
	if raw := r.PathValue("id"); raw != "" {
		id, e := uuid.Parse(raw)
		if e != nil {
			return nil, identity.ErrNotFound
		}
		var state string
		var generation int
		var retained bool
		if e = tx.QueryRow(ctx, "SELECT state,generation,raw_ciphertext IS NOT NULL AND retain_until>now() FROM app.webhook_events WHERE id=$1 FOR UPDATE", id).Scan(&state, &generation, &retained); e != nil {
			return nil, identity.ErrNotFound
		}
		if rt.Write {
			if !retained || state == "QUARANTINED" || state == "QUEUED" || state == "PROCESSING" || state == "RETRY_WAIT" {
				return nil, identity.ErrConflict
			}
			generation++
			replay := newID()
			if _, e = tx.Exec(ctx, "INSERT INTO app.webhook_replays(id,organization_id,event_id,generation,requested_by,request_id) VALUES($1,$2,$3,$4,$5,$6)", replay, org, id, generation, v.UserID, request); e != nil {
				return nil, e
			}
			if _, e = tx.Exec(ctx, "UPDATE app.webhook_events SET state='QUEUED',generation=$2,attempt_count=0,next_attempt_at=now(),lease_token=NULL,lease_until=NULL,last_error=NULL WHERE id=$1", id, generation); e != nil {
				return nil, e
			}
			if e = audit(ctx, tx, org, &v.UserID, "meta.webhook.replay", id, request); e != nil {
				return nil, e
			}
			return map[string]any{"id": replay, "generation": generation, "queued": true}, nil
		}
		if rt.Permission == "webhooks.payload.view" {
			if state == "QUARANTINED" {
				return nil, identity.ErrPermission
			}
			if !retained {
				return nil, identity.ErrNotFound
			}
			var ciphertext []byte
			if e = tx.QueryRow(ctx, "SELECT raw_ciphertext FROM app.webhook_events WHERE id=$1", id).Scan(&ciphertext); e != nil {
				return nil, e
			}
			raw, e := s.Auth.OpenEvidence(ciphertext, evidenceAAD(org, id))
			if e != nil {
				return nil, e
			}
			if e = audit(ctx, tx, org, &v.UserID, "meta.webhook.payload.inspect", id, request); e != nil {
				return nil, e
			}
			return redactPayload(raw), nil
		}
		attempts, e := rows(ctx, tx, "SELECT generation,attempt_no,result,error_code,parser_version,created_at FROM app.webhook_processing_attempts WHERE event_id=$1 ORDER BY generation DESC,attempt_no DESC LIMIT 100", id)
		if e != nil {
			return nil, e
		}
		facts, e := rows(ctx, tx, "SELECT event_class,waba_id,phone_id,uncertain_identity FROM app.webhook_facts WHERE event_id=$1 ORDER BY id LIMIT 100", id)
		return map[string]any{"id": id, "state": state, "generation": generation, "payload_retained": retained, "attempts": attempts, "facts": facts}, e
	}
	queries := map[string]string{
		"/meta/connection":        "SELECT id,external_id,callback_key,configured_waba_id,graph_version,credential_state,last_sync_at,last_error,created_at FROM app.meta_apps",
		"/meta/wabas":             "SELECT id,external_id,name,timezone_id,subscribed,lifecycle,graph_version,synced_at FROM app.wabas",
		"/meta/phone-numbers":     "SELECT id,waba_id,external_id,name,display_number,quality,platform,code_verification_status,lifecycle,graph_version,synced_at FROM app.phone_numbers",
		"/meta/business-profiles": "SELECT id,phone_id,fields,graph_version,synced_at FROM app.business_profiles",
		"/meta/sync-runs":         "SELECT id,app_id,state,attempt_count,error_code,graph_version,created_at,finished_at FROM app.asset_sync_runs",
		"/meta/webhook-events":    "SELECT id,app_id,received_at,state,event_class,attempt_count,last_error,request_id,parser_version,processed_at FROM app.webhook_events",
	}
	q, ok := queries[rt.Path]
	if !ok {
		return nil, identity.ErrNotFound
	}
	cursor := r.URL.Query().Get("cursor")
	var after uuid.UUID
	if cursor != "" {
		var e error
		after, e = uuid.Parse(cursor)
		if e != nil {
			return nil, identity.ErrValidation
		}
	}
	// Cursor must name an accessible row, preventing cross-tenant cursor disclosure.
	if after != uuid.Nil {
		var exists bool
		table := map[string]string{"/meta/connection": "meta_apps", "/meta/wabas": "wabas", "/meta/phone-numbers": "phone_numbers", "/meta/business-profiles": "business_profiles", "/meta/sync-runs": "asset_sync_runs", "/meta/webhook-events": "webhook_events"}[rt.Path]
		if e := tx.QueryRow(ctx, "SELECT EXISTS(SELECT FROM app."+table+" WHERE id=$1)", after).Scan(&exists); e != nil {
			return nil, e
		}
		if !exists {
			return nil, identity.ErrNotFound
		}
	}
	items, e := rows(ctx, tx, q+" WHERE id>$1 ORDER BY id LIMIT 101", after)
	if e != nil {
		return nil, e
	}
	more := len(items) > 100
	var next any
	if more {
		items = items[:100]
		next = items[99]["id"]
	}
	return map[string]any{"items": items, "has_more": more, "next_cursor": next}, nil
}
func (s *Service) RunWorker(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		if s.Config.Enabled {
			b, e := s.binding(ctx)
			if e == nil {
				for _, work := range []func(context.Context, binding) error{s.syncOne, s.processOne} {
					if e := work(ctx, b); e != nil && !errors.Is(e, pgx.ErrNoRows) && ctx.Err() == nil {
						slog.Error("Meta worker operation failed; inspect scoped job state")
					}
				}
				_ = s.scoped(ctx, b, func(tx pgx.Tx) error {
					_, e := tx.Exec(ctx, "UPDATE app.webhook_events SET raw_ciphertext=NULL WHERE retain_until<=now() AND raw_ciphertext IS NOT NULL")
					return e
				})
			} else if !errors.Is(e, pgx.ErrNoRows) && ctx.Err() != nil {
				return ctx.Err()
			}
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }
