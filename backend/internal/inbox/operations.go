package inbox

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"net/http"
	"strings"
	"time"
	"waba.local/control/internal/identity"
)

type command struct {
	Revision           int64       `json:"revision"`
	AssignmentRevision int64       `json:"assignment_revision"`
	Team               *uuid.UUID  `json:"team_id"`
	Member             *uuid.UUID  `json:"member_id"`
	Status             string      `json:"status"`
	Priority           string      `json:"priority"`
	Handoff            string      `json:"handoff_state"`
	Snooze             *time.Time  `json:"snoozed_until"`
	Body               string      `json:"body"`
	Mentions           []uuid.UUID `json:"mentions"`
	Watermark          int64       `json:"watermark"`
	ManualUnread       bool        `json:"manual_unread"`
	Enabled            *bool       `json:"sending_enabled"`
}

func (s *Service) operation(ctx context.Context, tx pgx.Tx, v identity.Session, member uuid.UUID, r *http.Request, rt route) (any, error) {
	org := *v.OrganizationID
	switch rt.Path {
	case "/pricing", "/rate-cards/imports", "/rate-cards/imports/{id}/{action}", "/budgets":
		return s.registry(ctx, tx, v, member, r, rt)
	case "/pricing/preflight":
		return s.preflight(ctx, tx, v, member, r)
	case "/outbound-intents":
		if r.Method == "GET" {
			key := r.URL.Query().Get("client_key")
			if len(key) < 16 || len(key) > 128 {
				return nil, identity.ErrValidation
			}
			return rows(ctx, tx, `SELECT i.id,i.conversation_id,i.state,i.error_code,i.provider_id,i.created_at,i.finished_at FROM app.outbound_intents i JOIN app.conversations c ON c.organization_id=i.organization_id AND c.id=i.conversation_id WHERE i.actor_member_id=$1 AND i.session_id=$2 AND i.client_key=$3 AND app.inbox_permission($1,'inbox.view',c.assigned_team_id,c.assigned_member_id)`, member, v.ID, key)
		}
		return s.submit(ctx, tx, v, member, r)
	case "/outbound-intents/{id}":
		id, e := pathID(r)
		if e != nil {
			return nil, e
		}
		result, e := rows(ctx, tx, `SELECT i.id,i.conversation_id,i.state,i.error_code,i.provider_id,i.created_at,i.finished_at FROM app.outbound_intents i JOIN app.conversations c ON c.organization_id=i.organization_id AND c.id=i.conversation_id WHERE i.id=$1 AND app.inbox_permission($2,'inbox.view',c.assigned_team_id,c.assigned_member_id)`, id, member)
		if e == nil && len(result) == 0 {
			e = identity.ErrNotFound
		}
		return result, e
	case "/inbox":
		var canView bool
		if e := tx.QueryRow(ctx, "SELECT app.inbox_permission($1,'inbox.view',NULL,$1) OR EXISTS(SELECT FROM app.teams WHERE archived_at IS NULL AND app.inbox_permission($1,'inbox.view',id,NULL))", member).Scan(&canView); e != nil {
			return nil, e
		}
		if !canView {
			return nil, identity.ErrPermission
		}
		settings, e := rows(ctx, tx, "SELECT sending_enabled,billing_currency_state,billing_currency,policy_revision FROM app.inbox_settings")
		if e != nil {
			return nil, e
		}
		teams, e := rows(ctx, tx, "SELECT id,name FROM app.teams WHERE archived_at IS NULL AND app.inbox_permission($1,'inbox.view',id,NULL) ORDER BY name", member)
		if e != nil {
			return nil, e
		}
		members, e := rows(ctx, tx, `SELECT DISTINCT m.id,u.display_name AS name,tm.team_id FROM app.organization_members m JOIN app.users u ON u.id=m.user_id LEFT JOIN app.team_members tm ON tm.member_id=m.id AND tm.organization_id=m.organization_id WHERE m.status='ACTIVE' AND u.global_status='ACTIVE' AND (app.inbox_permission($1,'inbox.view',NULL,NULL) OR app.inbox_permission($1,'inbox.view',tm.team_id,m.id)) ORDER BY name`, member)
		if e != nil {
			return nil, e
		}
		return map[string]any{"member_id": member, "session_id": v.ID, "settings": settings, "teams": teams, "members": members, "live_test_configured": s.Live.Enabled && identityDigits(s.Live.Recipient), "paid_authority": "CLOSED", "template_send": "DISABLED"}, nil
	case "/inbox/settings":
		if e := require(ctx, tx, member, "sending.manage", nil, nil); e != nil {
			return nil, e
		}
		var in command
		if e := decode(r, &in); e != nil {
			return nil, e
		}
		if in.Enabled == nil {
			return nil, identity.ErrValidation
		}
		result, e := tx.Exec(ctx, "UPDATE app.inbox_settings SET sending_enabled=$1,policy_revision=policy_revision+1 WHERE policy_revision=$2", *in.Enabled, in.Revision)
		if e != nil {
			return nil, e
		}
		if result.RowsAffected() != 1 {
			return nil, identity.ErrConflict
		}
		return map[string]any{"saved": true}, audit(ctx, tx, org, v.UserID, org, "inbox.sending_policy.changed")
	case "/conversations":
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if len(query) > 200 {
			return nil, identity.ErrValidation
		}
		state := r.URL.Query().Get("status")
		if state != "" && state != "OPEN" && state != "RESOLVED" && state != "SNOOZED" {
			return nil, identity.ErrValidation
		}
		priority := r.URL.Query().Get("priority")
		if priority != "" && priority != "LOW" && priority != "NORMAL" && priority != "HIGH" && priority != "URGENT" {
			return nil, identity.ErrValidation
		}
		unread := r.URL.Query().Get("unread")
		if unread != "" && unread != "true" {
			return nil, identity.ErrValidation
		}
		var team, assignee, phone *uuid.UUID
		for key, target := range map[string]**uuid.UUID{"team_id": &team, "member_id": &assignee, "phone_id": &phone} {
			if raw := r.URL.Query().Get(key); raw != "" {
				id, e := uuid.Parse(raw)
				if e != nil {
					return nil, identity.ErrValidation
				}
				*target = &id
			}
		}
		var cursor *uuid.UUID
		if raw := r.URL.Query().Get("cursor"); raw != "" {
			id, e := uuid.Parse(raw)
			if e != nil {
				return nil, identity.ErrValidation
			}
			if _, e = conversation(ctx, tx, id, member, "", false); e != nil {
				return nil, e
			}
			cursor = &id
		}
		result, e := rows(ctx, tx, `SELECT c.id,c.phone_id,p.display_number AS sending_phone,r.display_name,r.identity_value,c.status,c.priority,c.assigned_team_id,c.assigned_member_id,c.handoff_state,c.handoff_epoch,c.assignment_revision,c.revision,c.last_activity_at,c.last_inbound_at,c.last_outbound_at,c.window_expires_at,c.window_confidence,c.inbound_sequence,
 greatest(c.inbound_sequence-coalesce(rd.inbound_watermark,0),0) AS unread_count,coalesce(rd.manual_unread,false) AS manual_unread,clock_timestamp() AS server_now
 FROM app.conversations c JOIN app.phone_numbers p ON p.organization_id=c.organization_id AND p.id=c.phone_id
 JOIN app.recipient_identities r ON r.organization_id=c.organization_id AND r.id=c.recipient_id
 LEFT JOIN app.conversation_reads rd ON rd.organization_id=c.organization_id AND rd.conversation_id=c.id AND rd.member_id=$1
 WHERE app.inbox_permission($1,'inbox.view',c.assigned_team_id,c.assigned_member_id)
 AND ($2='' OR c.status=$2) AND ($3='' OR r.identity_value ILIKE '%'||$3||'%' OR r.display_name ILIKE '%'||$3||'%' OR EXISTS(SELECT FROM app.messages m WHERE m.conversation_id=c.id AND m.organization_id=c.organization_id AND to_tsvector('simple',m.text_body) @@ plainto_tsquery('simple',$3)))
 AND ($5='' OR c.priority=$5) AND ($6='' OR c.inbound_sequence>coalesce(rd.inbound_watermark,0) OR coalesce(rd.manual_unread,false))
 AND ($7::uuid IS NULL OR c.assigned_team_id=$7) AND ($8::uuid IS NULL OR c.assigned_member_id=$8) AND ($9::uuid IS NULL OR c.phone_id=$9)
 AND ($4::uuid IS NULL OR (c.last_activity_at,c.id)<(SELECT last_activity_at,id FROM app.conversations WHERE id=$4))
 ORDER BY c.last_activity_at DESC,c.id DESC LIMIT 101`, member, state, query, cursor, priority, unread, team, assignee, phone)
		if e != nil {
			return nil, e
		}
		more := len(result) > 100
		var next any
		if more {
			result = result[:100]
			next = result[99]["id"]
		}
		return map[string]any{"items": result, "has_more": more, "next_cursor": next}, nil
	}
	if rt.Path == "/notes/{id}" || rt.Path == "/notes/{id}/redact" {
		return s.noteChange(ctx, tx, v, member, r, rt)
	}
	id, e := pathID(r)
	if e != nil {
		return nil, e
	}
	key := ""
	switch rt.Path {
	case "/conversations/{id}/assignment":
		key = "conversations.assign"
	case "/conversations/{id}/notes":
		key = "notes.write"
	case "/conversations/{id}":
		if r.Method == "PATCH" {
			key = "conversations.manage"
		}
	}
	c, e := conversation(ctx, tx, id, member, key, r.Method != "GET" && rt.Path != "/conversations/{id}/presence")
	if e != nil {
		return nil, e
	}
	if rt.Path == "/conversations/{id}" && r.Method == "GET" {
		var before *uuid.UUID
		if raw := r.URL.Query().Get("before"); raw != "" {
			cursor, e := uuid.Parse(raw)
			if e != nil {
				return nil, identity.ErrValidation
			}
			var exists bool
			if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT FROM app.messages WHERE id=$1 AND conversation_id=$2)", cursor, id).Scan(&exists); e != nil {
				return nil, e
			}
			if !exists {
				return nil, identity.ErrNotFound
			}
			before = &cursor
		}
		messages, e := rows(ctx, tx, "SELECT id,provider_id,direction,message_type,content,text_body,context_provider_id,provider_at,received_at,inbound_sequence,delivery_state,processing_state,error_code,revision FROM app.messages WHERE conversation_id=$1 AND ($2::uuid IS NULL OR (received_at,id)<(SELECT received_at,id FROM app.messages WHERE id=$2 AND conversation_id=$1)) ORDER BY received_at DESC,id DESC LIMIT 101", id, before)
		if e != nil {
			return nil, e
		}
		more := len(messages) > 100
		var messageCursor any
		if more {
			messages = messages[:100]
			messageCursor = messages[99]["id"]
		}
		notes, e := rows(ctx, tx, "SELECT n.id,n.author_member_id,n.body,n.revision,n.created_at,n.updated_at,n.redacted_at,EXISTS(SELECT FROM app.note_mentions m WHERE m.note_id=n.id AND m.organization_id=n.organization_id AND m.member_id=$2) AS mentioned_me FROM app.internal_notes n WHERE n.conversation_id=$1 ORDER BY n.created_at DESC LIMIT 100", id, member)
		if e != nil {
			return nil, e
		}
		presence, e := rows(ctx, tx, "SELECT p.member_id,u.display_name AS name,p.expires_at FROM app.inbox_presence p JOIN app.organization_members m ON m.organization_id=p.organization_id AND m.id=p.member_id JOIN app.users u ON u.id=m.user_id WHERE p.conversation_id=$1 AND p.expires_at>now() AND p.member_id<>$2 AND m.status='ACTIVE' AND app.inbox_permission(p.member_id,'inbox.view',$3,$4)", id, member, c.Team, c.Member)
		if e != nil {
			return nil, e
		}
		details, e := rows(ctx, tx, "SELECT c.*,p.display_number AS sending_phone,r.identity_value,r.display_name,clock_timestamp() AS server_now FROM app.conversations c JOIN app.phone_numbers p ON p.organization_id=c.organization_id AND p.id=c.phone_id JOIN app.recipient_identities r ON r.organization_id=c.organization_id AND r.id=c.recipient_id WHERE c.id=$1", id)
		if e != nil {
			return nil, e
		}
		granted := []string{}
		for _, permission := range []string{"inbox.view", "messages.send", "conversations.assign", "conversations.manage", "notes.write", "notes.redact"} {
			if require(ctx, tx, member, permission, c.Team, c.Member) == nil {
				granted = append(granted, permission)
			}
		}
		return map[string]any{"conversation": details, "messages": messages, "notes": notes, "presence": presence, "permissions": granted, "message_has_more": more, "message_next_cursor": messageCursor}, nil
	}
	var in command
	if e = decode(r, &in); e != nil {
		return nil, e
	}
	switch rt.Path {
	case "/conversations/{id}/read":
		if in.Watermark < 0 || in.Watermark > c.InboundSequence {
			return nil, identity.ErrValidation
		}
		_, e = tx.Exec(ctx, "INSERT INTO app.conversation_reads(id,organization_id,conversation_id,member_id,inbound_watermark,manual_unread) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(organization_id,conversation_id,member_id) DO UPDATE SET inbound_watermark=greatest(app.conversation_reads.inbound_watermark,excluded.inbound_watermark),manual_unread=excluded.manual_unread,updated_at=now()", newID(), org, id, member, in.Watermark, in.ManualUnread)
	case "/conversations/{id}/presence":
		_, e = tx.Exec(ctx, "INSERT INTO app.inbox_presence(id,organization_id,conversation_id,member_id,expires_at) VALUES($1,$2,$3,$4,now()+interval '20 seconds') ON CONFLICT(organization_id,conversation_id,member_id) DO UPDATE SET expires_at=excluded.expires_at", newID(), org, id, member)
		// Presence is advisory and expiring, never a dispatch lock or durable event.
		return map[string]any{"saved": e == nil}, e
	case "/conversations/{id}/assignment":
		if in.AssignmentRevision != c.Assignment {
			return nil, Error("ASSIGNMENT_CHANGED")
		}
		if in.Team != nil {
			if e = require(ctx, tx, member, "conversations.assign", in.Team, in.Member); e != nil {
				return nil, e
			}
			var ok bool
			if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT FROM app.teams WHERE id=$1 AND archived_at IS NULL)", in.Team).Scan(&ok); e != nil {
				return nil, e
			}
			if !ok {
				return nil, identity.ErrValidation
			}
		}
		if in.Member != nil {
			var ok bool
			if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT FROM app.organization_members m JOIN app.users u ON u.id=m.user_id WHERE m.id=$1 AND m.status='ACTIVE' AND u.global_status='ACTIVE' AND ($2::uuid IS NULL OR EXISTS(SELECT FROM app.team_members tm WHERE tm.member_id=m.id AND tm.team_id=$2)) AND app.inbox_permission(m.id,'inbox.view',$2,m.id))", in.Member, in.Team).Scan(&ok); e != nil {
				return nil, e
			}
			if !ok {
				return nil, identity.ErrValidation
			}
		}
		_, e = tx.Exec(ctx, "UPDATE app.conversations SET assigned_team_id=$2,assigned_member_id=$3,assignment_revision=assignment_revision+1,revision=revision+1 WHERE id=$1", id, in.Team, in.Member)
	case "/conversations/{id}":
		if in.Revision != c.Revision {
			return nil, identity.ErrConflict
		}
		if in.Status != "" && in.Status != "OPEN" && in.Status != "RESOLVED" && in.Status != "SNOOZED" {
			return nil, identity.ErrValidation
		}
		if in.Priority != "" && in.Priority != "LOW" && in.Priority != "NORMAL" && in.Priority != "HIGH" && in.Priority != "URGENT" {
			return nil, identity.ErrValidation
		}
		if in.Handoff != "" && in.Handoff != "HUMAN" && in.Handoff != "WAITING_AGENT" {
			return nil, identity.ErrValidation
		}
		if in.Status == "SNOOZED" && (in.Snooze == nil || !in.Snooze.After(time.Now()) || in.Snooze.After(time.Now().Add(30*24*time.Hour))) {
			return nil, identity.ErrValidation
		}
		_, e = tx.Exec(ctx, `UPDATE app.conversations SET status=coalesce(nullif($2,''),status),priority=coalesce(nullif($3,''),priority),
 handoff_epoch=handoff_epoch+CASE WHEN $4<>'' AND $4<>handoff_state THEN 1 ELSE 0 END,handoff_state=coalesce(nullif($4,''),handoff_state),
 snoozed_until=CASE WHEN $2='' THEN snoozed_until WHEN $2='SNOOZED' THEN $5 ELSE NULL END,revision=revision+1 WHERE id=$1`, id, in.Status, in.Priority, in.Handoff, in.Snooze)
	case "/conversations/{id}/notes":
		if !validText(in.Body, 8000) || len(in.Mentions) > 20 {
			return nil, identity.ErrValidation
		}
		nid := newID()
		if _, e = tx.Exec(ctx, "INSERT INTO app.internal_notes(id,organization_id,conversation_id,author_member_id,body) VALUES($1,$2,$3,$4,$5)", nid, org, id, member, in.Body); e != nil {
			return nil, e
		}
		for _, target := range in.Mentions {
			var active bool
			if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT FROM app.organization_members m JOIN app.users u ON u.id=m.user_id WHERE m.id=$1 AND m.status='ACTIVE' AND u.global_status='ACTIVE')", target).Scan(&active); e != nil {
				return nil, e
			}
			if !active {
				return nil, identity.ErrValidation
			}

			if e = require(ctx, tx, target, "inbox.view", c.Team, c.Member); e != nil {
				return nil, identity.ErrValidation
			}
			if _, e = tx.Exec(ctx, "INSERT INTO app.note_mentions(id,organization_id,note_id,member_id) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING", newID(), org, nid, target); e != nil {
				return nil, e
			}
		}
		if e = audit(ctx, tx, org, v.UserID, nid, "inbox.note.created"); e != nil {
			return nil, e
		}
		return map[string]any{"id": nid}, event(ctx, tx, org, id, "conversation.updated")
	default:
		return nil, identity.ErrNotFound
	}
	if e != nil {
		return nil, e
	}
	if rt.Path != "/conversations/{id}/read" {
		if e = audit(ctx, tx, org, v.UserID, id, "inbox.conversation.changed"); e != nil {
			return nil, e
		}
	}
	return map[string]any{"saved": true}, event(ctx, tx, org, id, "conversation.updated")
}
func (s *Service) noteChange(ctx context.Context, tx pgx.Tx, v identity.Session, member uuid.UUID, r *http.Request, rt route) (any, error) {
	id, e := pathID(r)
	if e != nil {
		return nil, e
	}
	var conv, author uuid.UUID
	var revision int64
	var redacted *time.Time
	if e = tx.QueryRow(ctx, "SELECT conversation_id,author_member_id,revision,redacted_at FROM app.internal_notes WHERE id=$1", id).Scan(&conv, &author, &revision, &redacted); errors.Is(e, pgx.ErrNoRows) {
		return nil, identity.ErrNotFound
	} else if e != nil {
		return nil, e
	}
	key := "notes.write"
	if rt.Path == "/notes/{id}/redact" {
		key = "notes.redact"
	}
	if _, e = conversation(ctx, tx, conv, member, key, true); e != nil {
		return nil, e
	}
	if e = tx.QueryRow(ctx, "SELECT revision,redacted_at FROM app.internal_notes WHERE id=$1 FOR UPDATE", id).Scan(&revision, &redacted); e != nil {
		return nil, e
	}
	var in command
	if e = decode(r, &in); e != nil {
		return nil, e
	}
	if revision != in.Revision || redacted != nil {
		return nil, identity.ErrConflict
	}
	if key == "notes.write" && (member != author || !validText(in.Body, 8000)) {
		return nil, identity.ErrPermission
	}
	if key == "notes.redact" {
		_, e = tx.Exec(ctx, "UPDATE app.internal_notes SET body='',redacted_at=now(),updated_at=now(),revision=revision+1 WHERE id=$1", id)
	} else {
		_, e = tx.Exec(ctx, "UPDATE app.internal_notes SET body=$2,updated_at=now(),revision=revision+1 WHERE id=$1", id, in.Body)
	}
	if e != nil {
		return nil, e
	}
	if e = audit(ctx, tx, *v.OrganizationID, v.UserID, id, "inbox."+key); e != nil {
		return nil, e
	}
	return map[string]any{"saved": true}, event(ctx, tx, *v.OrganizationID, conv, "conversation.updated")
}
