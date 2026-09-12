package identity

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"strings"
	"time"
)

func (s *Service) Bootstrap(ctx context.Context, in Input) error {
	email, e := CanonicalEmail(in.Email)
	if e != nil {
		return e
	}
	hash, e := HashPassword(in.Password)
	if e != nil {
		return e
	}
	if strings.TrimSpace(in.Name) == "" || len(in.Name) > 100 || in.Slug == "" {
		return ErrValidation
	}
	if in.Timezone == "" {
		in.Timezone = "UTC"
	}
	if _, e = time.LoadLocation(in.Timezone); e != nil {
		return ErrValidation
	}
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer rollback(tx)
	if e = exec(ctx, tx, "SELECT pg_advisory_xact_lock(721401)"); e != nil {
		return e
	}
	var initialized bool
	if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT FROM app.identity_bootstrap)").Scan(&initialized); e != nil {
		return e
	}
	if initialized {
		return ErrInitialized
	}
	org := in.OrganizationID
	if org == uuid.Nil {
		org = id()
	}
	user, member := id(), id()
	if e = scope(ctx, tx, org); e != nil {
		return e
	}
	if e = exec(ctx, tx, "INSERT INTO app.organizations(id,name,slug,timezone) VALUES($1,$2,$3,$4) ON CONFLICT(id) DO NOTHING", org, in.Name, in.Slug, in.Timezone); e != nil {
		return e
	}
	if e = exec(ctx, tx, "INSERT INTO app.users(id,canonical_email,display_name) VALUES($1,$2,$3)", user, email, in.Name); e != nil {
		return e
	}
	if e = exec(ctx, tx, "INSERT INTO app.password_identities(user_id,password_hash) VALUES($1,$2)", user, hash); e != nil {
		return e
	}
	if e = exec(ctx, tx, "INSERT INTO app.organization_members(id,organization_id,user_id,activated_at) VALUES($1,$2,$3,now())", member, org, user); e != nil {
		return e
	}
	keys := []string{"organization.view", "members.view", "members.invite", "members.manage", "teams.view", "teams.manage", "roles.view", "roles.manage", "roles.assign", "owners.manage", "audit.view", "meta.view", "meta.manage", "webhooks.view", "webhooks.payload.view", "webhooks.replay"}
	for _, preset := range []string{"Owner", "Admin", "Agent"} {
		role := id()
		if e = exec(ctx, tx, "INSERT INTO app.roles(id,organization_id,name) VALUES($1,$2,$3)", role, org, preset); e != nil {
			return e
		}
		for _, key := range keys {
			if preset == "Admin" && key == "owners.manage" {
				continue
			}
			if preset == "Agent" && key != "organization.view" {
				continue
			}
			if e = exec(ctx, tx, "INSERT INTO app.role_permissions(id,organization_id,role_id,permission_key) VALUES($1,$2,$3,$4)", id(), org, role, key); e != nil {
				return e
			}
		}
		if preset == "Owner" {
			if e = exec(ctx, tx, "INSERT INTO app.member_roles(id,organization_id,member_id,role_id,scope_kind) VALUES($1,$2,$3,$4,'ORG')", id(), org, member, role); e != nil {
				return e
			}
		}
	}
	if e = exec(ctx, tx, "INSERT INTO app.identity_bootstrap(singleton,organization_id,user_id) VALUES(true,$1,$2)", org, user); e != nil {
		return e
	}
	if e = s.audit(ctx, tx, user, &org, &member, "owner.bootstrap", user, Meta{RequestID: "operator-bootstrap"}); e != nil {
		return e
	}
	if _, e = s.challenge(ctx, tx, user, email, "EMAIL_VERIFY", 24*time.Hour); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s *Service) challenge(ctx context.Context, tx pgx.Tx, user uuid.UUID, email, kind string, lifetime time.Duration) (string, error) {
	if e := exec(ctx, tx, "UPDATE app.auth_challenges SET revoked_at=$3 WHERE user_id=$1 AND kind=$2 AND consumed_at IS NULL AND revoked_at IS NULL", user, kind, s.Now()); e != nil {
		return "", e
	}
	if e := exec(ctx, tx, `UPDATE app.mail_deliveries SET state='CANCELLED',payload_ciphertext=NULL,terminal_at=$2 WHERE challenge_id IN
 (SELECT id FROM app.auth_challenges WHERE user_id=$1 AND revoked_at IS NOT NULL) AND state IN ('QUEUED','RETRY_WAIT')`, user, s.Now()); e != nil {
		return "", e
	}
	token, cid := randomToken(), id()
	deadline := s.Now().Add(lifetime)
	if e := exec(ctx, tx, "INSERT INTO app.auth_challenges(id,user_id,kind,token_digest,expires_at) VALUES($1,$2,$3,$4,$5)", cid, user, kind, digest(token), deadline); e != nil {
		return "", e
	}
	if kind != "LOGIN_MFA" {
		if e := s.enqueue(ctx, tx, user, nil, nil, &cid, kind, cid, email, token, deadline); e != nil {
			return "", e
		}
	}
	return token, nil
}
func (s *Service) enqueue(ctx context.Context, tx pgx.Tx, user uuid.UUID, org, invitation, challenge *uuid.UUID, purpose string, source uuid.UUID, email, token string, deadline time.Time) error {
	if cap := s.Now().Add(24 * time.Hour); deadline.After(cap) {
		deadline = cap
	}
	mid := id()
	payload, _ := json.Marshal(MailPayload{Recipient: email, Token: token, Purpose: purpose})
	encrypted, e := s.seal(payload, "mail:"+mid.String())
	if e != nil {
		return e
	}
	return exec(ctx, tx, `INSERT INTO app.mail_deliveries(id,user_id,organization_id,invitation_id,challenge_id,purpose,source_id,payload_ciphertext,deadline_at)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT(purpose,source_id) DO NOTHING`, mid, nullable(user), org, invitation, challenge, purpose, source, encrypted, deadline)
}
func (s *Service) notice(ctx context.Context, tx pgx.Tx, user uuid.UUID, email string) error {
	return s.enqueue(ctx, tx, user, nil, nil, nil, "SECURITY_NOTICE", id(), email, "", s.Now().Add(24*time.Hour))
}
func (s *Service) checkFactor(ctx context.Context, tx pgx.Tx, user uuid.UUID, code string) error {
	var encrypted []byte
	var last int64
	e := tx.QueryRow(ctx, "SELECT secret_ciphertext,last_step FROM app.auth_factors WHERE user_id=$1 AND confirmed_at IS NOT NULL FOR UPDATE", user).Scan(&encrypted, &last)
	if e != nil {
		return ErrAuthentication
	}
	secret, e := s.open(encrypted, "totp:"+user.String())
	if e != nil {
		return e
	}
	step := s.Now().Unix() / 30
	for _, offset := range []int64{-1, 0, 1} {
		candidate := step + offset
		if candidate <= last {
			continue
		}
		valid, err := totp.ValidateCustom(code, string(secret), time.Unix(candidate*30, 0), totp.ValidateOpts{Period: 30, Skew: 0, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1})
		if err == nil && valid {
			return exec(ctx, tx, "UPDATE app.auth_factors SET last_step=$2 WHERE user_id=$1", user, candidate)
		}
	}
	result, e := tx.Exec(ctx, "UPDATE app.auth_recovery_codes SET consumed_at=$3 WHERE user_id=$1 AND digest=$2 AND consumed_at IS NULL", user, digest(code), s.Now())
	if e != nil {
		return e
	}
	if result.RowsAffected() == 1 {
		return s.audit(ctx, tx, user, nil, nil, "mfa.recovery.used", user, Meta{})
	}
	return ErrAuthentication
}
func (s *Service) recovery(ctx context.Context, tx pgx.Tx, user uuid.UUID) ([]string, error) {
	if e := exec(ctx, tx, "DELETE FROM app.auth_recovery_codes WHERE user_id=$1", user); e != nil {
		return nil, e
	}
	codes := make([]string, 10)
	for i := range codes {
		codes[i] = randomToken()
		if e := exec(ctx, tx, "INSERT INTO app.auth_recovery_codes(id,user_id,digest) VALUES($1,$2,$3)", id(), user, digest(codes[i])); e != nil {
			return nil, e
		}
	}
	return codes, nil
}
func (s *Service) rotate(ctx context.Context, tx pgx.Tx, v Session, mfa bool) (Result, error) {
	if e := exec(ctx, tx, "UPDATE app.sessions SET revoked_at=$2 WHERE id=$1", v.ID, s.Now()); e != nil {
		return Result{}, e
	}
	r, e := s.session(ctx, tx, v.UserID, mfa)
	if e == nil && v.OrganizationID != nil {
		e = exec(ctx, tx, "UPDATE app.sessions SET organization_id=$2,access_revision=(SELECT access_revision FROM app.user_organizations($3) WHERE id=$2) WHERE token_digest=$1", digest(r.Cookie), v.OrganizationID, v.UserID)
	}
	if e == nil {
		e = exec(ctx, tx, "UPDATE app.sessions SET expires_at=$2,idle_expires_at=least($2::timestamptz,$3::timestamptz),reauth_at=$4 WHERE token_digest=$1", digest(r.Cookie), v.ExpiresAt, s.Now().Add(30*time.Minute), v.ReauthAt)
	}
	return r, e
}
func (s *Service) identity(ctx context.Context, tx pgx.Tx, v Session, action string, in Input, meta Meta) (Result, error) {
	ok := stringResult(map[string]any{"ok": true})
	switch action {
	case "public.login":
		email, e := CanonicalEmail(in.Email)
		if e != nil {
			email = ""
		}
		var user uuid.UUID
		var hash, emailDB string
		var mfa bool
		e = tx.QueryRow(ctx, `SELECT u.id,p.password_hash,u.canonical_email,EXISTS(SELECT FROM app.auth_factors f WHERE f.user_id=u.id AND confirmed_at IS NOT NULL)
 FROM app.users u JOIN app.password_identities p ON p.user_id=u.id WHERE canonical_email=$1 AND global_status='ACTIVE' FOR UPDATE OF u`, email).Scan(&user, &hash, &emailDB, &mfa)
		if e != nil {
			hash = s.dummy
		}
		valid, rehash := VerifyPassword(hash, in.Password)
		if e != nil || !valid {
			if err := s.audit(ctx, tx, uuid.Nil, nil, nil, "login.failed", uuid.Nil, meta); err != nil {
				return Result{}, err
			}
			return Result{}, ErrAuthentication
		}
		if rehash {
			newHash, e := HashPassword(in.Password)
			if e != nil {
				return Result{}, e
			}
			if e = exec(ctx, tx, "UPDATE app.password_identities SET password_hash=$2 WHERE user_id=$1", user, newHash); e != nil {
				return Result{}, e
			}
		}
		if mfa {
			proof, e := s.challenge(ctx, tx, user, emailDB, "LOGIN_MFA", 5*time.Minute)
			return stringResult(map[string]any{"mfa_required": true, "challenge": proof}), e
		}
		if e = s.audit(ctx, tx, user, nil, nil, "login.succeeded", user, meta); e != nil {
			return Result{}, e
		}
		return s.session(ctx, tx, user, false)
	case "public.mfa", "public.reset.complete", "public.verify.complete":
		kind := "LOGIN_MFA"
		if action == "public.reset.complete" {
			kind = "PASSWORD_RESET"
		}
		if action == "public.verify.complete" {
			kind = "EMAIL_VERIFY"
		}
		var user, cid uuid.UUID
		var email string
		e := tx.QueryRow(ctx, "SELECT user_id FROM app.auth_challenges WHERE token_digest=$1 AND kind=$2", digest(in.Token), kind).Scan(&user)
		if e != nil {
			return Result{}, ErrToken
		}
		if e = tx.QueryRow(ctx, "SELECT canonical_email FROM app.users WHERE id=$1 AND global_status='ACTIVE' FOR UPDATE", user).Scan(&email); e != nil {
			return Result{}, ErrToken
		}
		e = tx.QueryRow(ctx, "SELECT id FROM app.auth_challenges WHERE user_id=$1 AND token_digest=$2 AND kind=$3 AND expires_at>$4 AND consumed_at IS NULL AND revoked_at IS NULL AND attempts<8 FOR UPDATE", user, digest(in.Token), kind, s.Now()).Scan(&cid)
		if e != nil {
			return Result{}, ErrToken
		}
		if e = exec(ctx, tx, "UPDATE app.auth_challenges SET attempts=attempts+1 WHERE id=$1", cid); e != nil {
			return Result{}, e
		}
		if kind == "LOGIN_MFA" {
			if e = s.checkFactor(ctx, tx, user, in.Code); e != nil {
				return Result{}, e
			}
		}
		if kind == "PASSWORD_RESET" {
			hash, e := HashPassword(in.Password)
			if e != nil {
				return Result{}, e
			}
			if e = exec(ctx, tx, "UPDATE app.password_identities SET password_hash=$2,changed_at=$3 WHERE user_id=$1", user, hash, s.Now()); e != nil {
				return Result{}, e
			}
			if e = exec(ctx, tx, "UPDATE app.users SET auth_revision=auth_revision+1 WHERE id=$1", user); e != nil {
				return Result{}, e
			}
			if e = exec(ctx, tx, "UPDATE app.sessions SET revoked_at=$2 WHERE user_id=$1 AND revoked_at IS NULL", user, s.Now()); e != nil {
				return Result{}, e
			}
			if e = exec(ctx, tx, "UPDATE app.auth_challenges SET revoked_at=$2 WHERE user_id=$1 AND kind='LOGIN_MFA' AND consumed_at IS NULL", user, s.Now()); e != nil {
				return Result{}, e
			}
			if e = s.notice(ctx, tx, user, email); e != nil {
				return Result{}, e
			}
		}
		if kind == "EMAIL_VERIFY" {
			if e = exec(ctx, tx, "UPDATE app.users SET verified_at=$2 WHERE id=$1", user, s.Now()); e != nil {
				return Result{}, e
			}
		}
		if e = exec(ctx, tx, "UPDATE app.auth_challenges SET consumed_at=$2 WHERE id=$1", cid, s.Now()); e != nil {
			return Result{}, e
		}
		if e = s.audit(ctx, tx, user, nil, nil, action, user, meta); e != nil {
			return Result{}, e
		}
		if kind == "LOGIN_MFA" {
			return s.session(ctx, tx, user, true)
		}
		return ok, nil
	case "public.reset.request", "public.verify.request":
		email, e := CanonicalEmail(in.Email)
		if e != nil {
			return ok, nil
		}
		var user uuid.UUID
		var verified bool
		e = tx.QueryRow(ctx, "SELECT id,verified_at IS NOT NULL FROM app.users WHERE canonical_email=$1 AND global_status='ACTIVE' FOR UPDATE", email).Scan(&user, &verified)
		if errors.Is(e, pgx.ErrNoRows) {
			return ok, nil
		}
		if e != nil {
			return Result{}, e
		}
		kind, lifetime := "PASSWORD_RESET", 30*time.Minute
		if action == "public.verify.request" {
			if verified {
				return ok, nil
			}
			kind, lifetime = "EMAIL_VERIFY", 24*time.Hour
		}
		_, e = s.challenge(ctx, tx, user, email, kind, lifetime)
		return ok, e
	case "public.invitation.accept":
		return s.accept(ctx, tx, in, meta)
	case "session":
		rows, e := tx.Query(ctx, "SELECT id,name,member_id FROM app.user_organizations($1)", v.UserID)
		if e != nil {
			return Result{}, e
		}
		items, e := pgx.CollectRows(rows, pgx.RowToMap)
		if e != nil {
			return Result{}, e
		}
		return stringResult(map[string]any{"user": v, "organizations": jsonRows(items)}), nil
	case "organization.select":
		var member *uuid.UUID
		if e := tx.QueryRow(ctx, "SELECT app.authorize_membership($1,$2)", v.UserID, in.OrganizationID).Scan(&member); e != nil {
			return Result{}, e
		}
		if member == nil {
			return Result{}, ErrPermission
		}
		v.OrganizationID = &in.OrganizationID
		return s.rotate(ctx, tx, v, v.MFA)
	case "logout":
		e := exec(ctx, tx, "UPDATE app.sessions SET revoked_at=$2 WHERE id=$1", v.ID, s.Now())
		if e != nil {
			return Result{}, e
		}
		e = s.audit(ctx, tx, v.UserID, nil, nil, "logout", v.ID, meta)
		ok.Cookie = "clear"
		return ok, e
	case "sessions":
		return s.page(ctx, tx, in, action, v.UserID, uuid.Nil, "SELECT id,organization_id,created_at,expires_at,idle_expires_at,(id=$2) AS current FROM app.sessions s WHERE user_id=$1 AND revoked_at IS NULL AND expires_at>$3 AND idle_expires_at>$3 AND auth_revision=(SELECT auth_revision FROM app.users WHERE id=$1) AND (organization_id IS NULL OR EXISTS(SELECT FROM app.user_organizations($1) d WHERE d.id=s.organization_id AND d.access_revision=s.access_revision)) ORDER BY id DESC", v.UserID, v.ID, s.Now())
	case "session.revoke", "sessions.revoke":
		if e := s.recent(v); e != nil {
			return Result{}, e
		}
		query := "UPDATE app.sessions SET revoked_at=$3 WHERE user_id=$1 AND id=$2"
		target := in.ID
		if action == "sessions.revoke" {
			query = "UPDATE app.sessions SET revoked_at=$3 WHERE user_id=$1 AND id<>$2"
			target = v.ID
		}
		result, e := tx.Exec(ctx, query, v.UserID, target, s.Now())
		if e != nil {
			return Result{}, e
		}
		if action == "session.revoke" && result.RowsAffected() == 0 {
			return Result{}, ErrNotFound
		}
		if target == v.ID && action == "session.revoke" {
			ok.Cookie = "clear"
		}
		return ok, s.audit(ctx, tx, v.UserID, nil, nil, action, target, meta)
	case "reauth":
		var hash string
		if e := tx.QueryRow(ctx, "SELECT password_hash FROM app.password_identities WHERE user_id=$1", v.UserID).Scan(&hash); e != nil {
			return Result{}, e
		}
		valid, _ := VerifyPassword(hash, in.Password)
		if !valid {
			return Result{}, ErrAuthentication
		}
		if v.MFA {
			if e := s.checkFactor(ctx, tx, v.UserID, in.Code); e != nil {
				return Result{}, e
			}
		}
		v.ReauthAt = s.Now()
		r, e := s.rotate(ctx, tx, v, v.MFA)
		if e != nil {
			return Result{}, e
		}
		return r, s.audit(ctx, tx, v.UserID, nil, nil, "reauthenticated", v.UserID, meta)
	case "mfa.enroll":
		if e := s.recent(v); e != nil {
			return Result{}, e
		}
		if v.MFA {
			return Result{}, ErrConflict
		}
		key, e := totp.Generate(totp.GenerateOpts{Issuer: "WABA Control", AccountName: v.Email, SecretSize: 20})
		if e != nil {
			return Result{}, e
		}
		encrypted, e := s.seal([]byte(key.Secret()), "totp:"+v.UserID.String())
		if e != nil {
			return Result{}, e
		}
		e = exec(ctx, tx, "INSERT INTO app.auth_factors(user_id,secret_ciphertext) VALUES($1,$2) ON CONFLICT(user_id) DO UPDATE SET secret_ciphertext=$2,last_step=-1 WHERE app.auth_factors.confirmed_at IS NULL", v.UserID, encrypted)
		return stringResult(map[string]any{"secret": key.Secret(), "uri": key.URL()}), e
	case "mfa.confirm":
		if e := s.recent(v); e != nil {
			return Result{}, e
		}
		if v.MFA {
			return Result{}, ErrConflict
		}
		var data []byte
		if e := tx.QueryRow(ctx, "SELECT secret_ciphertext FROM app.auth_factors WHERE user_id=$1 AND confirmed_at IS NULL FOR UPDATE", v.UserID).Scan(&data); e != nil {
			return Result{}, ErrNotFound
		}
		secret, e := s.open(data, "totp:"+v.UserID.String())
		if e != nil {
			return Result{}, e
		}
		valid, e := totp.ValidateCustom(in.Code, string(secret), s.Now(), totp.ValidateOpts{Period: 30, Skew: 0, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1})
		if e != nil || !valid {
			return Result{}, ErrAuthentication
		}
		if e = exec(ctx, tx, "UPDATE app.auth_factors SET confirmed_at=$2,last_step=$3 WHERE user_id=$1", v.UserID, s.Now(), s.Now().Unix()/30); e != nil {
			return Result{}, e
		}
		codes, e := s.recovery(ctx, tx, v.UserID)
		if e != nil {
			return Result{}, e
		}
		if e = exec(ctx, tx, "UPDATE app.users SET auth_revision=auth_revision+1 WHERE id=$1", v.UserID); e != nil {
			return Result{}, e
		}
		if e = exec(ctx, tx, "UPDATE app.auth_challenges SET revoked_at=$2 WHERE user_id=$1 AND kind='LOGIN_MFA' AND consumed_at IS NULL", v.UserID, s.Now()); e != nil {
			return Result{}, e
		}
		r, e := s.rotate(ctx, tx, v, true)
		if e != nil {
			return Result{}, e
		}
		r.Data = map[string]any{"recovery_codes": codes, "csrf_token": r.CSRF}
		if e = s.notice(ctx, tx, v.UserID, v.Email); e != nil {
			return Result{}, e
		}
		return r, s.audit(ctx, tx, v.UserID, nil, nil, "mfa.enabled", v.UserID, meta)
	case "mfa.disable", "mfa.recovery":
		if e := s.recent(v); e != nil {
			return Result{}, e
		}
		if !v.MFA {
			return Result{}, ErrConflict
		}
		if action == "mfa.disable" {
			if e := exec(ctx, tx, "DELETE FROM app.auth_factors WHERE user_id=$1", v.UserID); e != nil {
				return Result{}, e
			}
			if e := exec(ctx, tx, "DELETE FROM app.auth_recovery_codes WHERE user_id=$1", v.UserID); e != nil {
				return Result{}, e
			}
		} else {
			codes, e := s.recovery(ctx, tx, v.UserID)
			if e != nil {
				return Result{}, e
			}
			ok.Data = map[string]any{"recovery_codes": codes}
		}
		if e := exec(ctx, tx, "UPDATE app.users SET auth_revision=auth_revision+1 WHERE id=$1", v.UserID); e != nil {
			return Result{}, e
		}
		if e := exec(ctx, tx, "UPDATE app.auth_challenges SET revoked_at=$2 WHERE user_id=$1 AND kind='LOGIN_MFA' AND consumed_at IS NULL", v.UserID, s.Now()); e != nil {
			return Result{}, e
		}
		r, e := s.rotate(ctx, tx, v, action != "mfa.disable")
		if e != nil {
			return Result{}, e
		}
		r.Data = map[string]any{"result": ok.Data, "csrf_token": r.CSRF}
		if e = s.notice(ctx, tx, v.UserID, v.Email); e != nil {
			return Result{}, e
		}
		return r, s.audit(ctx, tx, v.UserID, nil, nil, action, v.UserID, meta)
	}
	return Result{}, ErrNotFound
}
func csrfEqual(a, b string) bool { return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1 }
