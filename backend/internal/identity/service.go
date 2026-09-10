// Package identity owns global authentication and scoped company access.
// It uses a dedicated non-owner identity runtime role; no migration credentials.
package identity

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/url"
	"os"
	"strings"
	"time"
)

var (
	ErrValidation      = errors.New("VALIDATION_FAILED")
	ErrAuthentication  = errors.New("AUTHENTICATION_FAILED")
	ErrUnauthenticated = errors.New("UNAUTHENTICATED")
	ErrPermission      = errors.New("PERMISSION_DENIED")
	ErrReauth          = errors.New("REAUTHENTICATION_REQUIRED")
	ErrToken           = errors.New("TOKEN_INVALID_OR_EXPIRED")
	ErrConflict        = errors.New("REVISION_CONFLICT")
	ErrNotFound        = errors.New("RESOURCE_NOT_FOUND")
	ErrRate            = errors.New("RATE_LIMITED")
	ErrOwner           = errors.New("LAST_ACTIVE_OWNER")
	ErrInitialized     = errors.New("ALREADY_INITIALIZED")
)

type Service struct {
	Pool       *pgxpool.Pool
	Root       []byte
	Origin     string
	Production bool
	Now        func() time.Time
	dummy      string
}
type Session struct {
	ID             uuid.UUID  `json:"id"`
	UserID         uuid.UUID  `json:"user_id"`
	OrganizationID *uuid.UUID `json:"organization_id"`
	Email          string     `json:"email"`
	Name           string     `json:"name"`
	Verified       bool       `json:"verified"`
	MFA            bool       `json:"mfa_enabled"`
	AccessRevision int64      `json:"-"`
	ExpiresAt      time.Time  `json:"-"`
	ReauthAt       time.Time  `json:"-"`
	CSRF           string     `json:"-"`
}
type Grant struct {
	RoleID uuid.UUID  `json:"role_id"`
	Scope  string     `json:"scope"`
	TeamID *uuid.UUID `json:"team_id"`
}
type Input struct {
	Grants         []Grant     `json:"grants"`
	Cursor         string      `json:"-"`
	Email          string      `json:"email"`
	Password       string      `json:"password"`
	Code           string      `json:"code"`
	Token          string      `json:"token"`
	Name           string      `json:"name"`
	Slug           string      `json:"slug"`
	Timezone       string      `json:"timezone"`
	OrganizationID uuid.UUID   `json:"organization_id"`
	ID             uuid.UUID   `json:"id"`
	MemberID       uuid.UUID   `json:"member_id"`
	RoleIDs        []uuid.UUID `json:"role_ids"`
	Permissions    []string    `json:"permissions"`
	Scope          string      `json:"scope"`
	TeamID         *uuid.UUID  `json:"team_id"`
	Revision       int64       `json:"revision"`
}
type Result struct {
	Data   any
	Cookie string
	CSRF   string
}
type Meta struct {
	RequestID, IP, CSRF, SessionToken, ChangeDigest string
	ResourceRevision                                int64
}

func New(pool *pgxpool.Pool, root []byte, origin string, production bool) (*Service, error) {
	parsed, e := url.Parse(origin)
	if e != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") || (parsed.Scheme != "https" && parsed.Scheme != "http") || (production && parsed.Scheme != "https") {
		return nil, errors.New("valid public origin required")
	}
	if len(root) != 32 {
		return nil, errors.New("identity root key must contain 32 bytes")
	}
	dummy, e := HashPassword(randomToken())
	if e != nil {
		return nil, e
	}
	return &Service{Pool: pool, Root: root, Origin: strings.TrimRight(origin, "/"), Production: production, Now: time.Now, dummy: dummy}, nil
}
func Load(ctx context.Context, origin string, production bool) (*Service, error) {
	path := os.Getenv("IDENTITY_DATABASE_URL_FILE")
	raw, e := os.ReadFile(path)
	if e != nil {
		return nil, errors.New("IDENTITY_DATABASE_URL_FILE required")
	}
	key, e := os.ReadFile(os.Getenv("IDENTITY_ROOT_KEY_FILE"))
	if e != nil {
		return nil, errors.New("IDENTITY_ROOT_KEY_FILE required")
	}
	root, e := base64.StdEncoding.DecodeString(strings.TrimSpace(string(key)))
	if e != nil {
		return nil, errors.New("identity root key invalid")
	}
	cfg, e := pgxpool.ParseConfig(strings.TrimSpace(string(raw)))
	if e != nil {
		return nil, errors.New("identity database configuration invalid")
	}
	if cfg.ConnConfig.User != "waba_identity" {
		return nil, errors.New("identity runtime role required")
	}
	if production && (cfg.ConnConfig.TLSConfig == nil || cfg.ConnConfig.TLSConfig.InsecureSkipVerify || len(cfg.ConnConfig.Fallbacks) != 0) {
		return nil, errors.New("identity database requires verified TLS")
	}
	cfg.MaxConns = 10
	pool, e := pgxpool.NewWithConfig(ctx, cfg)
	if e != nil {
		return nil, errors.New("identity database unavailable")
	}
	if e = pool.Ping(ctx); e != nil {
		pool.Close()
		return nil, errors.New("identity database unavailable")
	}
	service, e := New(pool, root, origin, production)
	if e != nil {
		pool.Close()
	}
	return service, e
}
func id() uuid.UUID {
	v, e := uuid.NewV7()
	if e != nil {
		panic("entropy unavailable")
	}
	return v
}
func rollback(tx pgx.Tx) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = tx.Rollback(ctx)
}
func exec(ctx context.Context, tx pgx.Tx, sql string, args ...any) error {
	_, e := tx.Exec(ctx, sql, args...)
	return e
}
func (s *Service) audit(ctx context.Context, tx pgx.Tx, user uuid.UUID, org *uuid.UUID, member *uuid.UUID, action string, resource uuid.UUID, meta Meta) error {
	if org == nil {
		return exec(ctx, tx, "INSERT INTO app.identity_audit(id,user_id,action,resource_id,request_id) VALUES($1,$2,$3,$4,$5)", id(), nullable(user), action, nullable(resource), meta.RequestID)
	}
	return exec(ctx, tx, "INSERT INTO app.audit_log(id,organization_id,actor_user_id,actor_member_id,action,resource_id,request_id,change_digest,resource_revision,actor_label_snapshot) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,(SELECT display_name FROM app.users WHERE id=$3))", id(), org, nullable(user), member, action, nullable(resource), meta.RequestID, meta.ChangeDigest, meta.ResourceRevision)
}
func nullable(v uuid.UUID) any {
	if v == uuid.Nil {
		return nil
	}
	return v
}
func scope(ctx context.Context, tx pgx.Tx, org uuid.UUID) error {
	return exec(ctx, tx, "SELECT set_config('app.organization_id',$1,true)", org.String())
}
func (s *Service) limit(ctx context.Context, key string, max int) error {
	// Serialized bounded table: per-account and direct-peer IP digests, 15 minute windows.
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer rollback(tx)
	if e = exec(ctx, tx, "SELECT pg_advisory_xact_lock(721402)"); e != nil {
		return e
	}
	if e = exec(ctx, tx, "DELETE FROM app.login_limits WHERE expires_at < $1", s.Now()); e != nil {
		return e
	}
	var count int
	if e = tx.QueryRow(ctx, "SELECT count(*) FROM app.login_limits").Scan(&count); e != nil {
		return e
	}
	if count >= 10000 {
		return ErrRate
	}
	var attempts int
	e = tx.QueryRow(ctx, `INSERT INTO app.login_limits(key,window_at,attempts,expires_at) VALUES($1,$2,1,$3)
 ON CONFLICT(key) DO UPDATE SET attempts=app.login_limits.attempts+1 RETURNING attempts`, digest(key), s.Now(), s.Now().Add(15*time.Minute)).Scan(&attempts)
	if e != nil {
		return e
	}
	if e = tx.Commit(ctx); e != nil {
		return e
	}
	if attempts > max {
		return ErrRate
	}
	return nil
}
func (s *Service) authenticate(ctx context.Context, tx pgx.Tx, token string) (Session, error) {
	var v Session
	var user uuid.UUID
	if token == "" || len(token) > 128 {
		return v, ErrUnauthenticated
	}
	if e := tx.QueryRow(ctx, "SELECT user_id FROM app.sessions WHERE token_digest=$1", digest(token)).Scan(&user); e != nil {
		return v, ErrUnauthenticated
	}
	var revision int64
	if e := tx.QueryRow(ctx, "SELECT auth_revision FROM app.users WHERE id=$1 AND global_status='ACTIVE' FOR UPDATE", user).Scan(&revision); e != nil {
		return v, ErrUnauthenticated
	}
	e := tx.QueryRow(ctx, `SELECT s.id,s.user_id,s.organization_id,u.canonical_email,u.display_name,u.verified_at IS NOT NULL,
 EXISTS(SELECT FROM app.auth_factors f WHERE f.user_id=u.id AND confirmed_at IS NOT NULL),s.reauth_at,s.csrf_digest,s.expires_at,s.access_revision
 FROM app.sessions s JOIN app.users u ON u.id=s.user_id WHERE token_digest=$1 AND s.auth_revision=$2 AND s.revoked_at IS NULL AND s.expires_at>$3 AND s.idle_expires_at>$3 AND (s.organization_id IS NULL OR EXISTS(SELECT FROM app.user_organizations(s.user_id) d WHERE d.id=s.organization_id AND d.access_revision=s.access_revision)) FOR UPDATE OF s`, digest(token), revision, s.Now()).Scan(&v.ID, &v.UserID, &v.OrganizationID, &v.Email, &v.Name, &v.Verified, &v.MFA, &v.ReauthAt, &v.CSRF, &v.ExpiresAt, &v.AccessRevision)
	if e != nil {
		return v, ErrUnauthenticated
	}
	e = exec(ctx, tx, "UPDATE app.sessions SET idle_expires_at=least(expires_at,$2) WHERE id=$1", v.ID, s.Now().Add(30*time.Minute))
	return v, e
}
func (s *Service) session(ctx context.Context, tx pgx.Tx, user uuid.UUID, mfa bool) (Result, error) {
	token, csrf := randomToken(), randomToken()
	sid := id()
	e := exec(ctx, tx, `INSERT INTO app.sessions(id,user_id,token_digest,csrf_digest,auth_revision,expires_at,idle_expires_at,reauth_at,mfa_at)
 SELECT $1,id,$2,$3,auth_revision,$4,$5,$6,$7 FROM app.users WHERE id=$8 AND global_status='ACTIVE'`, sid, digest(token), digest(csrf), s.Now().Add(12*time.Hour), s.Now().Add(30*time.Minute), s.Now(), optionalTime(mfa, s.Now()), user)
	if e != nil {
		return Result{}, e
	}
	rows, e := tx.Query(ctx, "SELECT id FROM app.user_organizations($1)", user)
	if e != nil {
		return Result{}, e
	}
	orgs, e := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if e != nil {
		return Result{}, e
	}
	if len(orgs) == 1 {
		if e = exec(ctx, tx, "UPDATE app.sessions SET organization_id=$2,access_revision=(SELECT access_revision FROM app.user_organizations($3) WHERE id=$2) WHERE id=$1", sid, orgs[0], user); e != nil {
			return Result{}, e
		}
	}
	return Result{Data: map[string]any{"authenticated": true, "csrf_token": csrf}, Cookie: token, CSRF: csrf}, e
}
func optionalTime(ok bool, t time.Time) any {
	if ok {
		return t
	}
	return nil
}
func (s *Service) Run(ctx context.Context, action, token string, in Input, meta Meta) (Result, error) {
	meta.SessionToken = token
	if strings.HasPrefix(action, "public.") {
		if e := s.limit(ctx, "ip:"+meta.IP, 100); e != nil {
			return Result{}, e
		}
		key := "account:" + strings.ToLower(strings.TrimSpace(in.Email))
		if action != "public.login" && action != "public.reset.request" && action != "public.verify.request" {
			key = "proof:" + digest(in.Token)
		}
		if e := s.limit(ctx, key, 20); e != nil {
			return Result{}, e
		}
	}
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return Result{}, e
	}
	defer rollback(tx)
	_ = exec(ctx, tx, "SET LOCAL lock_timeout='3s'")
	var v Session
	if !strings.HasPrefix(action, "public.") {
		v, e = s.authenticate(ctx, tx, token)
		if e != nil {
			return Result{}, e
		}
		readOnly := action == "session" || action == "sessions" || action == "org.get" || action == "org.member.get" || strings.HasSuffix(action, ".list")
		if !readOnly && (meta.CSRF == "" || !csrfEqual(digest(meta.CSRF), v.CSRF)) {
			return Result{}, ErrPermission
		}
		if !readOnly {
			if e = s.limit(ctx, "security:"+v.UserID.String(), 60); e != nil {
				return Result{}, e
			}
		}
	}
	var result Result
	if strings.HasPrefix(action, "org.") {
		result, e = s.organization(ctx, tx, v, action, in, meta)
	} else {
		result, e = s.identity(ctx, tx, v, action, in, meta)
	}
	if e != nil {
		// Failed authentication/TOTP attempts are durable, including attempt counters/audit.
		if (action == "public.login" || action == "public.mfa") && (errors.Is(e, ErrAuthentication) || errors.Is(e, ErrToken)) {
			if ce := tx.Commit(ctx); ce != nil {
				return Result{}, ce
			}
		}
		return Result{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return Result{}, e
	}
	return result, nil
}
func (s *Service) recent(v Session) error {
	if s.Now().Sub(v.ReauthAt) > 5*time.Minute {
		return ErrReauth
	}
	return nil
}
func stringResult(data any) Result { return Result{Data: data} }
func invalid(action string) error  { return fmt.Errorf("%w: %s", ErrValidation, action) }
