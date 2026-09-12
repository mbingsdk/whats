package identity

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"strings"
	"time"
)

const permissionSQL = `SELECT EXISTS(
 SELECT FROM app.member_roles mr JOIN app.role_permissions rp ON rp.organization_id=mr.organization_id AND rp.role_id=mr.role_id
 WHERE mr.member_id=$1 AND rp.permission_key=$2 AND (mr.expires_at IS NULL OR mr.expires_at>now())
 AND (mr.scope_kind='ORG' OR (mr.scope_kind='TEAM' AND mr.team_id=$3 AND EXISTS(SELECT FROM app.teams active_team WHERE active_team.id=$3 AND active_team.archived_at IS NULL)) OR (mr.scope_kind='SELF' AND $4::uuid=$1))
 UNION ALL SELECT FROM app.team_members tm JOIN app.teams t ON t.organization_id=tm.organization_id AND t.id=tm.team_id
 JOIN app.team_roles tr ON tr.organization_id=t.organization_id AND tr.team_id=t.id
 JOIN app.role_permissions rp ON rp.organization_id=tr.organization_id AND rp.role_id=tr.role_id
 WHERE tm.member_id=$1 AND t.id=$3 AND t.archived_at IS NULL AND rp.permission_key=$2)`

func permission(ctx context.Context, tx pgx.Tx, member uuid.UUID, key string, team, subject *uuid.UUID) (bool, error) {
	var allowed bool
	e := tx.QueryRow(ctx, permissionSQL, member, key, team, subject).Scan(&allowed)
	return allowed, e
}
func require(ctx context.Context, tx pgx.Tx, member uuid.UUID, key string, team, subject *uuid.UUID) error {
	yes, e := permission(ctx, tx, member, key, team, subject)
	if e != nil {
		return e
	}
	if !yes {
		return ErrPermission
	}
	return nil
}
func roleKeys(ctx context.Context, tx pgx.Tx, role uuid.UUID) ([]string, error) {
	var exists bool
	if e := tx.QueryRow(ctx, "SELECT EXISTS(SELECT FROM app.roles WHERE id=$1)", role).Scan(&exists); e != nil {
		return nil, e
	}
	if !exists {
		return nil, ErrNotFound
	}
	rows, e := tx.Query(ctx, "SELECT permission_key FROM app.role_permissions WHERE role_id=$1", role)
	if e != nil {
		return nil, e
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}
func ceiling(ctx context.Context, tx pgx.Tx, member uuid.UUID, roles []uuid.UUID, team *uuid.UUID) error {
	if len(roles) > 20 {
		return ErrValidation
	}
	for _, role := range roles {
		keys, e := roleKeys(ctx, tx, role)
		if e != nil {
			return e
		}
		for _, key := range keys {
			if e = require(ctx, tx, member, key, team, nil); e != nil {
				return e
			}
		}
	}
	return nil
}
func ownerInvariant(ctx context.Context, tx pgx.Tx) error {
	var count int
	e := tx.QueryRow(ctx, `SELECT count(DISTINCT m.id) FROM app.organization_members m
 JOIN app.users u ON u.id=m.user_id
 JOIN app.member_roles mr ON mr.organization_id=m.organization_id AND mr.member_id=m.id
 JOIN app.role_permissions rp ON rp.organization_id=mr.organization_id AND rp.role_id=mr.role_id
 WHERE m.status='ACTIVE' AND u.global_status='ACTIVE' AND mr.scope_kind='ORG'
 AND (mr.expires_at IS NULL OR mr.expires_at>now()) AND rp.permission_key='owners.manage'`).Scan(&count)
	if e != nil {
		return e
	}
	if count < 1 {
		return ErrOwner
	}
	return nil
}
func checkRevision(ctx context.Context, tx pgx.Tx, table string, target uuid.UUID, revision int64) error {
	// Table names are internal constants at call sites, never request text.
	var actual int64
	if e := tx.QueryRow(ctx, "SELECT revision FROM app."+table+" WHERE id=$1 FOR UPDATE", target).Scan(&actual); e != nil {
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return e
	}
	if revision < 1 || actual != revision {
		return ErrConflict
	}
	return nil
}
func list(ctx context.Context, tx pgx.Tx, query string, args ...any) (Result, error) {
	rows, e := tx.Query(ctx, query, args...)
	if e != nil {
		return Result{}, e
	}
	items, e := pgx.CollectRows(rows, pgx.RowToMap)
	return stringResult(map[string]any{"items": jsonRows(items), "has_more": false, "next_cursor": nil}), e
}
func (s *Service) organization(ctx context.Context, tx pgx.Tx, v Session, action string, in Input, meta Meta) (Result, error) {
	org := in.OrganizationID
	if org == uuid.Nil && v.OrganizationID != nil {
		org = *v.OrganizationID
	}
	if org == uuid.Nil {
		return Result{}, ErrPermission
	}
	write := !strings.HasSuffix(action, ".list") && action != "org.get" && action != "org.member.get"
	fn := "app.authorize_membership"
	if write {
		fn += "_write"
	}
	var member *uuid.UUID
	if e := tx.QueryRow(ctx, "SELECT "+fn+"($1,$2)", v.UserID, org).Scan(&member); e != nil {
		return Result{}, e
	}
	if member == nil {
		return Result{}, ErrPermission
	}
	if e := scope(ctx, tx, org); e != nil {
		return Result{}, e
	}
	if v.OrganizationID != nil && *v.OrganizationID == org {
		var revision int64
		if e := tx.QueryRow(ctx, "SELECT access_revision FROM app.organization_members WHERE id=$1", member).Scan(&revision); e != nil {
			return Result{}, e
		}
		if revision != v.AccessRevision {
			return Result{}, ErrUnauthenticated
		}
	}
	if write {
		if !v.Verified {
			return Result{}, ErrPermission
		}
		if e := s.recent(v); e != nil {
			return Result{}, e
		}
		if s.Production && !v.MFA {
			return Result{}, ErrReauth
		}
	}
	ok := stringResult(map[string]any{"ok": true})
	needed := map[string]string{
		"org.get": "organization.view", "org.members.list": "members.view", "org.member.get": "members.view",
		"org.invitations.list": "members.invite", "org.invite": "members.invite", "org.invite.revoke": "members.invite", "org.invite.resend": "members.invite",
		"org.member.deactivate": "members.manage", "org.member.reactivate": "members.manage",
		"org.teams.list": "teams.view", "org.team.create": "teams.manage", "org.team.update": "teams.manage", "org.team.archive": "teams.manage",
		"org.team.members.list": "teams.view", "org.team.member.add": "teams.manage", "org.team.member.remove": "teams.manage",
		"org.roles.list": "roles.view", "org.role.create": "roles.manage", "org.role.update": "roles.manage",
		"org.member.roles": "roles.assign", "org.team.roles": "roles.assign", "org.audit.list": "audit.view"}
	key, exists := needed[action]
	if !exists {
		return Result{}, ErrNotFound
	}
	var team *uuid.UUID
	if strings.HasPrefix(action, "org.team.") && action != "org.team.create" {
		team = &in.ID
	}
	var subject *uuid.UUID
	if action == "org.member.get" {
		subject = &in.ID
	}
	if e := require(ctx, tx, *member, key, team, subject); e != nil && action != "org.teams.list" {
		return Result{}, e
	}
	switch action {
	case "org.get":
		var name, timezone string
		var revision int64
		if e := tx.QueryRow(ctx, "SELECT name,timezone,revision FROM app.organizations WHERE id=$1", org).Scan(&name, &timezone, &revision); e != nil {
			return Result{}, e
		}
		grants := []string{}
		for _, p := range []string{"organization.view", "members.view", "members.invite", "members.manage", "teams.view", "teams.manage", "roles.view", "roles.manage", "roles.assign", "owners.manage", "audit.view", "meta.view", "meta.manage", "webhooks.view", "webhooks.payload.view", "webhooks.replay"} {
			yes, e := permission(ctx, tx, *member, p, nil, nil)
			if e != nil {
				return Result{}, e
			}
			if yes {
				grants = append(grants, p)
			}
		}
		rows, e := tx.Query(ctx, "SELECT id FROM app.teams WHERE archived_at IS NULL")
		if e != nil {
			return Result{}, e
		}
		ids, e := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
		if e != nil {
			return Result{}, e
		}
		scoped := []map[string]any{}
		for _, tid := range ids {
			view, e := permission(ctx, tx, *member, "teams.view", &tid, nil)
			if e != nil {
				return Result{}, e
			}
			if !view {
				continue
			}
			keys := []string{}
			for _, key := range []string{"teams.view", "teams.manage", "roles.assign"} {
				yes, e := permission(ctx, tx, *member, key, &tid, nil)
				if e != nil {
					return Result{}, e
				}
				if yes {
					keys = append(keys, key)
				}
			}
			scoped = append(scoped, map[string]any{"id": tid, "permissions": keys})
		}
		return stringResult(map[string]any{"id": org, "name": name, "timezone": timezone, "revision": revision, "permissions": grants, "member_id": member, "scoped_teams": scoped}), nil
	case "org.members.list":
		return s.page(ctx, tx, in, action, v.UserID, org, `SELECT m.id,m.organization_id,u.display_name,u.canonical_email,m.status,m.revision,m.access_revision,
 coalesce((SELECT jsonb_agg(jsonb_build_object('role_id',mr.role_id,'scope',mr.scope_kind,'team_id',mr.team_id)) FROM app.member_roles mr WHERE mr.member_id=m.id),'[]'::jsonb) AS grants
 FROM app.organization_members m JOIN app.users u ON u.id=m.user_id ORDER BY m.id LIMIT 100`)
	case "org.member.get":
		return s.page(ctx, tx, in, action, v.UserID, org, "SELECT m.id,m.organization_id,u.display_name,u.canonical_email,m.status,m.revision,m.access_revision FROM app.organization_members m JOIN app.users u ON u.id=m.user_id WHERE m.id=$1", in.ID)
	case "org.teams.list":
		predicate := strings.ReplaceAll(strings.ReplaceAll(permissionSQL, "$3", "t.id"), "$4", "NULL::uuid")
		return s.page(ctx, tx, in, action, v.UserID, org, "SELECT id,organization_id,name,revision,archived_at FROM app.teams t WHERE ("+predicate+") ORDER BY id", *member, "teams.view")
	case "org.roles.list":
		return s.page(ctx, tx, in, action, v.UserID, org, "SELECT r.id,r.organization_id,r.name,r.revision,coalesce(jsonb_agg(rp.permission_key) FILTER(WHERE rp.permission_key IS NOT NULL),'[]'::jsonb) AS permissions FROM app.roles r LEFT JOIN app.role_permissions rp ON rp.role_id=r.id GROUP BY r.id ORDER BY r.id LIMIT 100")
	case "org.invitations.list":
		return s.page(ctx, tx, in, action, v.UserID, org, "SELECT i.id,i.organization_id,i.canonical_email,(SELECT d.state FROM app.mail_deliveries d WHERE d.organization_id=i.organization_id AND d.invitation_id=i.id ORDER BY d.created_at DESC,d.id DESC LIMIT 1) AS delivery_state,CASE WHEN i.state='PENDING' AND i.expires_at<=now() THEN 'EXPIRED' ELSE i.state END AS state,i.expires_at,i.revision FROM app.invitations i ORDER BY i.id DESC LIMIT 100")
	case "org.team.members.list":
		return s.page(ctx, tx, in, action, v.UserID, org, "SELECT tm.member_id AS id,tm.member_id,u.display_name,m.status FROM app.team_members tm JOIN app.organization_members m ON m.id=tm.member_id JOIN app.users u ON u.id=m.user_id WHERE tm.team_id=$1 ORDER BY tm.member_id LIMIT 100", in.ID)
	case "org.audit.list":
		return s.page(ctx, tx, in, action, v.UserID, org, "SELECT id,actor_user_id,actor_member_id,actor_kind,actor_label_snapshot,action,resource_id,resource_revision,change_digest,request_id,created_at FROM app.audit_log ORDER BY created_at DESC,id DESC LIMIT 100")
	case "org.invite":
		email, e := CanonicalEmail(in.Email)
		if e != nil {
			return Result{}, e
		}
		if len(in.RoleIDs) == 0 {
			return Result{}, ErrValidation
		}
		if e = ceiling(ctx, tx, *member, in.RoleIDs, nil); e != nil {
			return Result{}, e
		}
		if e = exec(ctx, tx, "UPDATE app.invitations SET state='EXPIRED',revision=revision+1 WHERE canonical_email=$1 AND state='PENDING' AND expires_at<=$2", email, s.Now()); e != nil {
			return Result{}, e
		}
		inv, token := id(), randomToken()
		deadline := s.Now().Add(72 * time.Hour)
		if e = exec(ctx, tx, "INSERT INTO app.invitations(id,organization_id,canonical_email,token_digest,inviter_member_id,expires_at) VALUES($1,$2,$3,$4,$5,$6)", inv, org, email, digest(token), member, deadline); e != nil {
			return Result{}, e
		}
		for _, role := range in.RoleIDs {
			if e = exec(ctx, tx, "INSERT INTO app.invitation_roles(id,organization_id,invitation_id,role_id) VALUES($1,$2,$3,$4)", id(), org, inv, role); e != nil {
				return Result{}, e
			}
		}
		if e = s.enqueue(ctx, tx, uuid.Nil, &org, &inv, nil, "INVITATION", inv, email, token, deadline); e != nil {
			return Result{}, e
		}
		in.ID = inv
		ok.Data = map[string]any{"id": inv, "state": "PENDING", "revision": 1}
	case "org.invite.revoke", "org.invite.resend":
		if e := checkRevision(ctx, tx, "invitations", in.ID, in.Revision); e != nil {
			return Result{}, e
		}
		var email, state string
		if e := tx.QueryRow(ctx, "SELECT canonical_email,state FROM app.invitations WHERE id=$1", in.ID).Scan(&email, &state); e != nil {
			return Result{}, e
		}
		if state != "PENDING" {
			return Result{}, ErrConflict
		}
		if e := exec(ctx, tx, "UPDATE app.mail_deliveries SET state='CANCELLED',payload_ciphertext=NULL,terminal_at=$2 WHERE invitation_id=$1 AND state IN ('QUEUED','RETRY_WAIT')", in.ID, s.Now()); e != nil {
			return Result{}, e
		}
		if action == "org.invite.revoke" {
			if e := exec(ctx, tx, "UPDATE app.invitations SET state='REVOKED',revision=revision+1 WHERE id=$1", in.ID); e != nil {
				return Result{}, e
			}
		} else {
			roles, e := invitationRoles(ctx, tx, in.ID)
			if e != nil {
				return Result{}, e
			}
			if e = ceiling(ctx, tx, *member, roles, nil); e != nil {
				return Result{}, e
			}
			token := randomToken()
			deadline := s.Now().Add(72 * time.Hour)
			if e = exec(ctx, tx, "UPDATE app.invitations SET token_digest=$2,expires_at=$3,inviter_member_id=$4,revision=revision+1 WHERE id=$1", in.ID, digest(token), deadline, member); e != nil {
				return Result{}, e
			}
			if e = s.enqueue(ctx, tx, uuid.Nil, &org, &in.ID, nil, "INVITATION", id(), email, token, deadline); e != nil {
				return Result{}, e
			}
		}
	case "org.member.deactivate", "org.member.reactivate":
		if e := checkRevision(ctx, tx, "organization_members", in.ID, in.Revision); e != nil {
			return Result{}, e
		}
		roles, e := memberRoles(ctx, tx, in.ID)
		if e != nil {
			return Result{}, e
		}
		if e = ceiling(ctx, tx, *member, roles, nil); e != nil {
			return Result{}, e
		}
		active := action == "org.member.reactivate"
		status := "DEACTIVATED"
		if active {
			status = "ACTIVE"
		}
		if e = exec(ctx, tx, "UPDATE app.organization_members SET status=$2,access_revision=access_revision+1,revision=revision+1,deactivated_at=$3 WHERE id=$1", in.ID, status, optionalTime(!active, s.Now())); e != nil {
			return Result{}, e
		}
		if !active {
			if e = exec(ctx, tx, "DELETE FROM app.team_members WHERE member_id=$1", in.ID); e != nil {
				return Result{}, e
			}
		}
		if e = ownerInvariant(ctx, tx); e != nil {
			return Result{}, e
		}
	case "org.team.create":
		if strings.TrimSpace(in.Name) == "" || len(in.Name) > 100 {
			return Result{}, ErrValidation
		}
		in.ID = id()
		if e := exec(ctx, tx, "INSERT INTO app.teams(id,organization_id,name) VALUES($1,$2,$3)", in.ID, org, in.Name); e != nil {
			return Result{}, e
		}
		ok.Data = map[string]any{"id": in.ID, "revision": 1}
	case "org.team.update", "org.team.archive", "org.team.member.add", "org.team.member.remove", "org.team.roles":
		if e := checkRevision(ctx, tx, "teams", in.ID, in.Revision); e != nil {
			return Result{}, e
		}
		rows, e := tx.Query(ctx, "SELECT role_id FROM app.team_roles WHERE team_id=$1", in.ID)
		if e != nil {
			return Result{}, e
		}
		roles, e := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
		if e != nil {
			return Result{}, e
		}
		if e = ceiling(ctx, tx, *member, roles, &in.ID); e != nil {
			return Result{}, e
		}
		switch action {
		case "org.team.update":
			if strings.TrimSpace(in.Name) == "" || len(in.Name) > 100 {
				return Result{}, ErrValidation
			}
			e = exec(ctx, tx, "UPDATE app.teams SET name=$2 WHERE id=$1", in.ID, in.Name)
		case "org.team.archive":
			e = exec(ctx, tx, "UPDATE app.teams SET archived_at=$2 WHERE id=$1", in.ID, s.Now())
		case "org.team.member.add":
			var active bool
			e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT FROM app.organization_members WHERE id=$1 AND status='ACTIVE') AND EXISTS(SELECT FROM app.teams WHERE id=$2 AND archived_at IS NULL)", in.MemberID, in.ID).Scan(&active)
			if e == nil && !active {
				return Result{}, ErrNotFound
			}
			if e == nil {
				e = exec(ctx, tx, "INSERT INTO app.team_members(id,organization_id,team_id,member_id) VALUES($1,$2,$3,$4) ON CONFLICT(organization_id,team_id,member_id) DO NOTHING", id(), org, in.ID, in.MemberID)
			}
		case "org.team.member.remove":
			e = exec(ctx, tx, "DELETE FROM app.team_members WHERE team_id=$1 AND member_id=$2", in.ID, in.MemberID)
		case "org.team.roles":
			if e = ceiling(ctx, tx, *member, in.RoleIDs, &in.ID); e != nil {
				return Result{}, e
			}
			if e = exec(ctx, tx, "DELETE FROM app.team_roles WHERE team_id=$1", in.ID); e != nil {
				return Result{}, e
			}
			for _, role := range in.RoleIDs {
				if e = exec(ctx, tx, "INSERT INTO app.team_roles(id,organization_id,team_id,role_id) VALUES($1,$2,$3,$4)", id(), org, in.ID, role); e != nil {
					return Result{}, e
				}
			}
		}
		if e != nil {
			return Result{}, e
		}
		if e = exec(ctx, tx, "UPDATE app.teams SET revision=revision+1 WHERE id=$1", in.ID); e != nil {
			return Result{}, e
		}
	case "org.role.create", "org.role.update":
		if strings.TrimSpace(in.Name) == "" || len(in.Name) > 100 || len(in.Permissions) > 30 {
			return Result{}, ErrValidation
		}
		if action == "org.role.update" {
			if e := checkRevision(ctx, tx, "roles", in.ID, in.Revision); e != nil {
				return Result{}, e
			}
			if e := ceiling(ctx, tx, *member, []uuid.UUID{in.ID}, nil); e != nil {
				return Result{}, e
			}
		} else {
			in.ID = id()
		}
		for _, key := range in.Permissions {
			if e := require(ctx, tx, *member, key, nil, nil); e != nil {
				return Result{}, e
			}
		}
		if action == "org.role.create" {
			if e := exec(ctx, tx, "INSERT INTO app.roles(id,organization_id,name) VALUES($1,$2,$3)", in.ID, org, in.Name); e != nil {
				return Result{}, e
			}
		} else {
			if e := exec(ctx, tx, "UPDATE app.roles SET name=$2,revision=revision+1 WHERE id=$1", in.ID, in.Name); e != nil {
				return Result{}, e
			}
			if e := exec(ctx, tx, "DELETE FROM app.role_permissions WHERE role_id=$1", in.ID); e != nil {
				return Result{}, e
			}
		}
		for _, key := range in.Permissions {
			if e := exec(ctx, tx, "INSERT INTO app.role_permissions(id,organization_id,role_id,permission_key) VALUES($1,$2,$3,$4)", id(), org, in.ID, key); e != nil {
				return Result{}, e
			}
		}
		if e := ownerInvariant(ctx, tx); e != nil {
			return Result{}, e
		}
		ok.Data = map[string]any{"id": in.ID}
	case "org.member.roles":
		if e := checkRevision(ctx, tx, "organization_members", in.ID, in.Revision); e != nil {
			return Result{}, e
		}
		old, e := memberRoles(ctx, tx, in.ID)
		if e != nil {
			return Result{}, e
		}
		if e = ceiling(ctx, tx, *member, old, nil); e != nil {
			return Result{}, e
		}
		grants := in.Grants
		if len(grants) > 0 && len(in.RoleIDs) > 0 {
			return Result{}, ErrValidation
		}
		if len(grants) == 0 {
			if in.Scope == "" {
				in.Scope = "ORG"
			}
			for _, role := range in.RoleIDs {
				grants = append(grants, Grant{role, in.Scope, in.TeamID})
			}
		}
		if len(grants) > 20 {
			return Result{}, ErrValidation
		}
		for _, grant := range grants {
			if grant.Scope != "ORG" && grant.Scope != "TEAM" && grant.Scope != "SELF" {
				return Result{}, ErrValidation
			}
			if (grant.Scope == "TEAM") != (grant.TeamID != nil) {
				return Result{}, ErrValidation
			}
			if e = ceiling(ctx, tx, *member, []uuid.UUID{grant.RoleID}, grant.TeamID); e != nil {
				return Result{}, e
			}
		}
		if e = exec(ctx, tx, "DELETE FROM app.member_roles WHERE member_id=$1", in.ID); e != nil {
			return Result{}, e
		}
		for _, grant := range grants {
			if e = exec(ctx, tx, "INSERT INTO app.member_roles(id,organization_id,member_id,role_id,scope_kind,team_id) VALUES($1,$2,$3,$4,$5,$6)", id(), org, in.ID, grant.RoleID, grant.Scope, grant.TeamID); e != nil {
				return Result{}, e
			}
		}
		if e = ownerInvariant(ctx, tx); e != nil {
			return Result{}, e
		}
		if e = exec(ctx, tx, "UPDATE app.organization_members SET access_revision=access_revision+1,revision=revision+1 WHERE id=$1", in.ID); e != nil {
			return Result{}, e
		}
	}
	if write {
		changed, _ := json.Marshal(struct {
			Action string
			Input  Input
		}{action, in})
		meta.ChangeDigest = digest(string(changed))
		meta.ResourceRevision = in.Revision + 1
		// Access revision invalidates organization-bound sessions without acquiring
		// another user's session locks beneath the organization barrier.
		query := ""
		switch action {
		case "org.role.update":
			query = "UPDATE app.organization_members SET access_revision=access_revision+1 WHERE id IN (SELECT member_id FROM app.member_roles WHERE role_id=$1 UNION SELECT tm.member_id FROM app.team_members tm JOIN app.team_roles tr ON tr.team_id=tm.team_id WHERE tr.role_id=$1)"
		case "org.team.roles", "org.team.archive", "org.team.member.add", "org.team.member.remove":
			query = "UPDATE app.organization_members SET access_revision=access_revision+1 WHERE id IN (SELECT member_id FROM app.team_members WHERE team_id=$1 UNION SELECT member_id FROM app.member_roles WHERE team_id=$1) OR id=$2"
		}
		if query != "" {
			args := []any{in.ID}
			if strings.HasPrefix(action, "org.team.") {
				args = append(args, in.MemberID)
			}
			if e := exec(ctx, tx, query, args...); e != nil {
				return Result{}, e
			}
		}
		var actorRevision int64
		if e := tx.QueryRow(ctx, "SELECT access_revision FROM app.organization_members WHERE id=$1", member).Scan(&actorRevision); e != nil {
			return Result{}, e
		}
		var sessionRevision int64
		if e := tx.QueryRow(ctx, "SELECT access_revision FROM app.sessions WHERE id=$1", v.ID).Scan(&sessionRevision); e != nil {
			return Result{}, e
		}
		if actorRevision != sessionRevision && v.OrganizationID != nil && *v.OrganizationID == org {
			if action == "org.member.deactivate" && in.ID == *member {
				ok.Cookie = "clear"
			} else {
				rotated, e := s.rotate(ctx, tx, v, v.MFA)
				if e != nil {
					return Result{}, e
				}
				ok.Cookie, ok.CSRF = rotated.Cookie, rotated.CSRF
			}
		}
		if e := s.audit(ctx, tx, v.UserID, &org, member, action, in.ID, meta); e != nil {
			return Result{}, e
		}
	}
	return ok, nil
}
func memberRoles(ctx context.Context, tx pgx.Tx, member uuid.UUID) ([]uuid.UUID, error) {
	rows, e := tx.Query(ctx, "SELECT DISTINCT role_id FROM app.member_roles WHERE member_id=$1", member)
	if e != nil {
		return nil, e
	}
	return pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
}
func invitationRoles(ctx context.Context, tx pgx.Tx, inv uuid.UUID) ([]uuid.UUID, error) {
	rows, e := tx.Query(ctx, "SELECT role_id FROM app.invitation_roles WHERE invitation_id=$1", inv)
	if e != nil {
		return nil, e
	}
	return pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
}
func (s *Service) accept(ctx context.Context, tx pgx.Tx, in Input, meta Meta) (Result, error) {
	var org *uuid.UUID
	if e := tx.QueryRow(ctx, "SELECT app.invitation_organization($1)", digest(in.Token)).Scan(&org); e != nil {
		return Result{}, e
	}
	if org == nil {
		return Result{}, ErrToken
	}
	if e := scope(ctx, tx, *org); e != nil {
		return Result{}, e
	}
	var inv, inviter uuid.UUID
	var email string
	if e := tx.QueryRow(ctx, "SELECT id,canonical_email,inviter_member_id FROM app.invitations WHERE token_digest=$1", digest(in.Token)).Scan(&inv, &email, &inviter); e != nil {
		return Result{}, ErrToken
	}
	if e := exec(ctx, tx, "SELECT pg_advisory_xact_lock(hashtextextended($1,721403))", email); e != nil {
		return Result{}, e
	}
	var inviterUser uuid.UUID
	if e := tx.QueryRow(ctx, "SELECT user_id FROM app.organization_members WHERE id=$1", inviter).Scan(&inviterUser); e != nil {
		return Result{}, e
	}
	locked, e := tx.Query(ctx, "SELECT id FROM app.users WHERE canonical_email=$1 OR id=$2 ORDER BY id FOR UPDATE", email, inviterUser)
	if e != nil {
		return Result{}, e
	}
	if _, e = pgx.CollectRows(locked, pgx.RowTo[uuid.UUID]); e != nil {
		return Result{}, e
	}
	var user uuid.UUID
	e = tx.QueryRow(ctx, "SELECT id FROM app.users WHERE canonical_email=$1", email).Scan(&user)
	if errors.Is(e, pgx.ErrNoRows) {
		hash, e := HashPassword(in.Password)
		if e != nil {
			return Result{}, e
		}
		if strings.TrimSpace(in.Name) == "" || len(in.Name) > 100 {
			return Result{}, ErrValidation
		}
		user = id()
		if e = exec(ctx, tx, "INSERT INTO app.users(id,canonical_email,display_name,verified_at) VALUES($1,$2,$3,$4)", user, email, in.Name, s.Now()); e != nil {
			return Result{}, e
		}
		if e = exec(ctx, tx, "INSERT INTO app.password_identities(user_id,password_hash) VALUES($1,$2)", user, hash); e != nil {
			return Result{}, e
		}
	} else if e != nil {
		return Result{}, e
	} else {
		v, e := s.authenticate(ctx, tx, meta.SessionToken)
		if e != nil || v.UserID != user || in.Password != "" || !csrfEqual(digest(meta.CSRF), v.CSRF) {
			return Result{}, ErrUnauthenticated
		}
	}
	var authorized *uuid.UUID
	if e = tx.QueryRow(ctx, "SELECT app.authorize_membership_write($1,$2)", inviterUser, org).Scan(&authorized); e != nil {
		return Result{}, e
	}
	if authorized == nil {
		return Result{}, ErrPermission
	}
	if e = require(ctx, tx, inviter, "members.invite", nil, nil); e != nil {
		return Result{}, e
	}
	roles, e := invitationRoles(ctx, tx, inv)
	if e != nil {
		return Result{}, e
	}
	if e = ceiling(ctx, tx, inviter, roles, nil); e != nil {
		return Result{}, e
	}
	var valid bool
	if e = tx.QueryRow(ctx, "SELECT state='PENDING' AND token_digest=$2 AND expires_at>$3 FROM app.invitations WHERE id=$1 FOR UPDATE", inv, digest(in.Token), s.Now()).Scan(&valid); e != nil {
		return Result{}, e
	}
	if !valid {
		return Result{}, ErrToken
	}
	member := id()
	if e = exec(ctx, tx, "INSERT INTO app.organization_members(id,organization_id,user_id,activated_at) VALUES($1,$2,$3,$4)", member, org, user, s.Now()); e != nil {
		return Result{}, e
	}
	for _, role := range roles {
		if e = exec(ctx, tx, "INSERT INTO app.member_roles(id,organization_id,member_id,role_id,scope_kind) VALUES($1,$2,$3,$4,'ORG')", id(), org, member, role); e != nil {
			return Result{}, e
		}
	}
	if e = exec(ctx, tx, "UPDATE app.invitations SET state='ACCEPTED',accepted_member_id=$2,revision=revision+1 WHERE id=$1", inv, member); e != nil {
		return Result{}, e
	}
	if e = s.audit(ctx, tx, user, org, &member, "invitation.accepted", inv, meta); e != nil {
		return Result{}, e
	}
	return stringResult(map[string]any{"ok": true, "organization_id": org}), nil
}

// pgx dynamic rows decode UUIDs as [16]byte. The public API always uses strings,
// and timestamp values are serialized in UTC regardless of the local test timezone.
func jsonRows(rows []map[string]any) []map[string]any {
	for _, row := range rows {
		for key, value := range row {
			switch v := value.(type) {
			case [16]byte:
				row[key] = uuid.UUID(v).String()
			case time.Time:
				row[key] = v.UTC()
			}
		}
	}
	return rows
}
