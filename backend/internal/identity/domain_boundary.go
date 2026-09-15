package identity

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"net/http"
)

// LockDomainSession accepts an internal persisted session ID, never an HTTP
// user/organization identity shortcut. Shared locks conflict with revocation.
func (s *Service) LockDomainSession(ctx context.Context, tx pgx.Tx, sid, org uuid.UUID) (Session, uuid.UUID, error) {
	var v Session
	var user uuid.UUID
	if e := tx.QueryRow(ctx, "SELECT user_id FROM app.sessions WHERE id=$1", sid).Scan(&user); e != nil {
		return v, uuid.Nil, ErrUnauthenticated
	}
	var revision int64
	if e := tx.QueryRow(ctx, "SELECT auth_revision FROM app.users WHERE id=$1 AND global_status='ACTIVE' FOR SHARE", user).Scan(&revision); e != nil {
		return v, uuid.Nil, ErrUnauthenticated
	}
	var member *uuid.UUID
	if e := tx.QueryRow(ctx, "SELECT app.authorize_membership($1,$2)", user, org).Scan(&member); e != nil {
		return v, uuid.Nil, e
	}
	if member == nil {
		return v, uuid.Nil, ErrPermission
	}
	if e := scope(ctx, tx, org); e != nil {
		return v, uuid.Nil, e
	}
	e := tx.QueryRow(ctx, `SELECT s.id,s.user_id,s.organization_id,u.canonical_email,u.display_name,u.verified_at IS NOT NULL,
 (s.mfa_at IS NOT NULL AND EXISTS(SELECT FROM app.auth_factors f WHERE f.user_id=u.id AND confirmed_at IS NOT NULL)),s.reauth_at,s.csrf_digest,s.expires_at,s.access_revision
 FROM app.sessions s JOIN app.users u ON u.id=s.user_id
 JOIN app.organization_members m ON m.user_id=s.user_id AND m.organization_id=s.organization_id
 WHERE s.id=$1 AND s.organization_id=$2 AND s.auth_revision=$3 AND s.access_revision=m.access_revision
 AND s.revoked_at IS NULL AND s.expires_at>clock_timestamp() AND s.idle_expires_at>clock_timestamp()
 FOR SHARE OF s`, sid, org, revision).Scan(&v.ID, &v.UserID, &v.OrganizationID, &v.Email, &v.Name, &v.Verified, &v.MFA, &v.ReauthAt, &v.CSRF, &v.ExpiresAt, &v.AccessRevision)
	if e != nil {
		return v, uuid.Nil, ErrUnauthenticated
	}
	return v, *member, nil
}

// Each REST command/SSE read revalidates the session. Ordinary replies do not
// need sensitive reauthentication; registry publication and settings do.
func (s *Service) DomainTransaction(r *http.Request, sensitive bool, fn func(context.Context, pgx.Tx, Session, uuid.UUID) (any, error)) (any, error) {
	ctx := r.Context()
	cookie, e := r.Cookie("waba_session")
	if e != nil || cookie.Value == "" || len(cookie.Value) > 128 {
		return nil, ErrUnauthenticated
	}
	write := r.Method != "GET"
	if write {
		csrf, e := r.Cookie("waba_csrf")
		token := r.Header.Get("X-CSRF-Token")
		if e != nil || r.Header.Get("Origin") != s.Origin || len(token) != 43 || !csrfEqual(csrf.Value, token) {
			return nil, ErrPermission
		}
	}
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return nil, e
	}
	defer rollback(tx)
	if _, e = tx.Exec(ctx, "SET LOCAL lock_timeout='3s'"); e != nil {
		return nil, e
	}
	var sid uuid.UUID
	var org *uuid.UUID
	if e = tx.QueryRow(ctx, "SELECT id,organization_id FROM app.sessions WHERE token_digest=$1", digest(cookie.Value)).Scan(&sid, &org); e != nil || org == nil {
		return nil, ErrUnauthenticated
	}
	v, member, e := s.LockDomainSession(ctx, tx, sid, *org)
	if e != nil {
		return nil, e
	}
	if write && (!v.Verified || !csrfEqual(digest(r.Header.Get("X-CSRF-Token")), v.CSRF)) {
		return nil, ErrPermission
	}
	if sensitive {
		if e = s.recent(v); e != nil {
			return nil, e
		}
		if !v.MFA {
			return nil, ErrReauth
		}
	}
	result, e := fn(ctx, tx, v, member)
	if e != nil {
		return nil, e
	}
	if e = tx.Commit(ctx); e != nil {
		return nil, e
	}
	return result, nil
}
func WriteDomainResult(w http.ResponseWriter, r *http.Request, value any, e error) {
	if e != nil {
		fail(w, r, e)
		return
	}
	jsonResponse(w, 200, value)
}
