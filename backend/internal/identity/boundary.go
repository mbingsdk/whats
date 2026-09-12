package identity

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"net/http"
)

// Authorized runs a bounded local operation while membership/revision locks remain held.
// Callers must not perform network I/O inside fn. The organization always comes from the session.
func (s *Service) Authorized(w http.ResponseWriter, r *http.Request, key string, write bool, fn func(context.Context, pgx.Tx, Session) (any, error)) {
	token := ""
	if c, e := r.Cookie("waba_session"); e == nil {
		token = c.Value
	}
	if write {
		c, e := r.Cookie("waba_csrf")
		v := r.Header.Get("X-CSRF-Token")
		if r.Header.Get("Origin") != s.Origin || e != nil || len(v) != 43 || !csrfEqual(v, c.Value) {
			fail(w, r, ErrPermission)
			return
		}
	}
	tx, e := s.Pool.Begin(r.Context())
	if e != nil {
		fail(w, r, e)
		return
	}
	defer rollback(tx)
	ctx := r.Context()
	v, e := s.authenticate(ctx, tx, token)
	if e != nil {
		fail(w, r, e)
		return
	}
	if v.OrganizationID == nil {
		fail(w, r, ErrPermission)
		return
	}
	if write && (!v.Verified || !csrfEqual(digest(r.Header.Get("X-CSRF-Token")), v.CSRF)) {
		fail(w, r, ErrPermission)
		return
	}
	if write {
		if e = s.recent(v); e != nil {
			fail(w, r, e)
			return
		}
		if s.Production && !v.MFA {
			fail(w, r, ErrReauth)
			return
		}
	}
	fnName := "app.authorize_membership"
	if write {
		fnName += "_write"
	}
	var member *uuid.UUID
	if e = tx.QueryRow(ctx, "SELECT "+fnName+"($1,$2)", v.UserID, v.OrganizationID).Scan(&member); e != nil {
		fail(w, r, e)
		return
	}
	if member == nil {
		fail(w, r, ErrPermission)
		return
	}
	if e = scope(ctx, tx, *v.OrganizationID); e != nil {
		fail(w, r, e)
		return
	}
	if e = require(ctx, tx, *member, key, nil, nil); e != nil {
		fail(w, r, e)
		return
	}
	data, e := fn(ctx, tx, v)
	if e != nil {
		fail(w, r, e)
		return
	}
	if e = tx.Commit(ctx); e != nil {
		fail(w, r, e)
		return
	}
	jsonResponse(w, 200, data)
}

// SealEvidence/OpenEvidence share the established envelope format; purpose is caller-specific.
func (s *Service) SealEvidence(b []byte, aad string) ([]byte, error) { return s.seal(b, aad) }
func (s *Service) OpenEvidence(b []byte, aad string) ([]byte, error) { return s.open(b, aad) }
