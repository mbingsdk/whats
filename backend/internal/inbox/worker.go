package inbox

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"log/slog"
	"time"
	"waba.local/control/internal/meta"
)

type claimed struct {
	ID, Org, Session, Conversation, Actor, Phone, Message, Authorization, Lease uuid.UUID
	Text, Hash                                                                  string
}

func (s *Service) workerScope(ctx context.Context, fn func(pgx.Tx, uuid.UUID) error) error {
	if s.Meta == nil || !s.Meta.Config.Enabled {
		return pgx.ErrNoRows
	}
	var org uuid.UUID
	if e := s.Auth.Pool.QueryRow(ctx, "SELECT organization_id FROM app.meta_binding($1)", s.Meta.Config.AppID).Scan(&org); e != nil {
		return e
	}
	tx, e := s.Auth.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer rollback(tx)
	if _, e = tx.Exec(ctx, "SELECT set_config('app.organization_id',$1,true)", org.String()); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "SET LOCAL lock_timeout='3s'"); e != nil {
		return e
	}
	if e = fn(tx, org); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s *Service) DispatchOnce(ctx context.Context) error {
	var job claimed
	e := s.workerScope(ctx, func(tx pgx.Tx, org uuid.UUID) error {
		job.Org = org
		job.Lease = newID()
		e := tx.QueryRow(ctx, `SELECT id,session_id,conversation_id,actor_member_id,phone_id,message_id,authorization_id,text_body,scope_hash
 FROM app.outbound_intents WHERE state='QUEUED' OR (state='RESERVED' AND claim_until<now())
 ORDER BY created_at,id FOR UPDATE SKIP LOCKED LIMIT 1`).Scan(&job.ID, &job.Session, &job.Conversation, &job.Actor, &job.Phone, &job.Message, &job.Authorization, &job.Text, &job.Hash)
		if e != nil {
			return e
		}
		_, e = tx.Exec(ctx, "UPDATE app.outbound_intents SET state='RESERVED',claim_token=$2,claim_until=now()+interval '1 minute' WHERE id=$1", job.ID, job.Lease)
		return e
	})
	if e != nil {
		return e
	}
	var target, phone string
	var authorized bool
	e = s.workerScope(ctx, func(tx pgx.Tx, org uuid.UUID) error {
		if org != job.Org {
			return Error("WORKER_SCOPE_CHANGED")
		}
		v, member, authErr := s.Auth.LockDomainSession(ctx, tx, job.Session, org)
		deny := ""
		if authErr != nil || member != job.Actor {
			deny = "ACTOR_AUTHORIZATION_REVOKED"
		}
		if deny == "" {
			c, e := conversation(ctx, tx, job.Conversation, member, "messages.send", true)
			if e != nil {
				deny = "PERMISSION_DENIED"
			} else {
				var assignment, policyRev int64
				var expectedHash string
				var expires time.Time
				var consumed *uuid.UUID
				var policy uuid.UUID
				e = tx.QueryRow(ctx, "SELECT assignment_revision,policy_revision,scope_hash,expires_at,consumed_intent_id,policy_id FROM app.pricing_authorizations WHERE id=$1 FOR SHARE", job.Authorization).Scan(&assignment, &policyRev, &expectedHash, &expires, &consumed, &policy)
				if e != nil {
					return e
				}
				in := sendInput{Conversation: c.ID, Assignment: assignment, Text: job.Text, Type: "TEXT", Category: "SERVICE"}
				d, currentPolicy, currentRev, now, e := s.evaluate(ctx, tx, v, member, c, in)
				if e != nil {
					return e
				}
				switch {
				case !d.Allowed:
					deny = d.Code
				case !expires.After(now):
					deny = "PRICING_AUTHORIZATION_EXPIRED"
				case currentPolicy != policy || currentRev != policyRev:
					deny = "PRICING_POLICY_CHANGED"
				case consumed == nil || *consumed != job.ID || expectedHash != job.Hash || scopeHash(v, c, in) != job.Hash:
					deny = "INTENT_SCOPE_CHANGED"
				case !s.Live.Enabled || c.Identity != s.Live.Recipient:
					deny = "CONTROLLED_TEST_RECIPIENT_REQUIRED"
				default:
					target, phone = c.Identity, c.PhoneExternal
				}
			}
		}
		var state string
		var lease *uuid.UUID
		if e := tx.QueryRow(ctx, "SELECT state,claim_token FROM app.outbound_intents WHERE id=$1 FOR UPDATE", job.ID).Scan(&state, &lease); e != nil {
			return e
		}
		if state != "RESERVED" || lease == nil || *lease != job.Lease {
			return nil
		}
		if deny == "" {
			result, e := tx.Exec(ctx, "INSERT INTO app.test_send_slots(id,organization_id,phone_id,recipient_id,intent_id) SELECT $1,$2,$3,recipient_id,$4 FROM app.conversations WHERE id=$5 ON CONFLICT(organization_id,phone_id) DO NOTHING", newID(), org, job.Phone, job.ID, job.Conversation)
			if e != nil {
				return e
			}
			if result.RowsAffected() == 0 {
				deny = "CONTROLLED_TEST_ALREADY_ATTEMPTED"
			}
		}
		if deny != "" {
			if _, e := tx.Exec(ctx, "UPDATE app.outbound_intents SET state='BLOCKED',error_code=$2,finished_at=now() WHERE id=$1", job.ID, deny); e != nil {
				return e
			}
			if _, e := tx.Exec(ctx, "UPDATE app.messages SET processing_state='BLOCKED',error_code=$2 WHERE id=$1", job.Message, deny); e != nil {
				return e
			}
			if _, e := tx.Exec(ctx, "INSERT INTO app.audit_log(id,organization_id,actor_kind,action,resource_id,request_id) VALUES($1,$2,'SYSTEM','inbox.dispatch.BLOCKED',$3,'inbox-worker')", newID(), org, job.ID); e != nil {
				return e
			}
			return event(ctx, tx, org, job.Conversation, "message.status_changed")
		}
		if _, e := tx.Exec(ctx, "INSERT INTO app.send_attempts(id,organization_id,intent_id,state) VALUES($1,$2,$3,'DISPATCHING')", newID(), org, job.ID); e != nil {
			return e
		}
		if _, e := tx.Exec(ctx, "UPDATE app.outbound_intents SET state='DISPATCHING',dispatch_started_at=clock_timestamp() WHERE id=$1", job.ID); e != nil {
			return e
		}
		if e := audit(ctx, tx, org, v.UserID, job.ID, "inbox.dispatch.authorized"); e != nil {
			return e
		}
		authorized = true
		return nil
	})
	if e != nil || !authorized {
		return e
	}
	// Locks/transactions are released before exactly one network request.
	requestCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	outcome := s.Sender.SendText(requestCtx, phone, target, job.Text, job.ID.String())
	cancel()
	finishCtx, finishCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer finishCancel()
	return s.finishDispatch(finishCtx, job, outcome)
}
func (s *Service) finishDispatch(ctx context.Context, job claimed, outcome meta.SendOutcome) error {
	return s.workerScope(ctx, func(tx pgx.Tx, org uuid.UUID) error {
		if org != job.Org {
			return Error("WORKER_SCOPE_CHANGED")
		}
		state := outcome.State
		messageState := state
		if state == "REJECTED" {
			state = "FAILED"
			messageState = "FAILED"
		}
		if state != "FAILED" && state != "ACCEPTED" {
			state = "UNCERTAIN"
			messageState = "UNCERTAIN"
		}
		result, e := tx.Exec(ctx, "UPDATE app.outbound_intents SET state=$2,error_code=nullif($3,''),provider_id=nullif($4,''),finished_at=now() WHERE id=$1 AND state='DISPATCHING' AND claim_token=$5", job.ID, state, outcome.Code, outcome.ProviderID, job.Lease)
		if e != nil {
			return e
		}
		if result.RowsAffected() == 0 {
			return nil
		}
		if _, e = tx.Exec(ctx, "UPDATE app.send_attempts SET state=$2,provider_id=nullif($3,''),error_code=nullif($4,''),finished_at=now() WHERE intent_id=$1 AND state='DISPATCHING'", job.ID, outcome.State, outcome.ProviderID, outcome.Code); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, "UPDATE app.messages SET provider_id=nullif($2,''),delivery_state=$3,error_code=nullif($4,''),revision=revision+1 WHERE id=$1", job.Message, outcome.ProviderID, messageState, outcome.Code); e != nil {
			return e
		}
		if state == "ACCEPTED" {
			if _, e = tx.Exec(ctx, "UPDATE app.conversations SET last_outbound_at=now(),last_activity_at=greatest(last_activity_at,now()),revision=revision+1 WHERE id=$1", job.Conversation); e != nil {
				return e
			}
			if e = correlateStatus(ctx, tx, org, job.Phone, outcome.ProviderID); e != nil {
				return e
			}
		}
		if _, e = tx.Exec(ctx, "INSERT INTO app.audit_log(id,organization_id,actor_kind,action,resource_id,request_id) VALUES($1,$2,'SYSTEM',$3,$4,'inbox-worker')", newID(), org, "inbox.dispatch."+state, job.ID); e != nil {
			return e
		}
		return event(ctx, tx, org, job.Conversation, "message.status_changed")
	})
}
func (s *Service) recoverUncertain(ctx context.Context) error {
	return s.workerScope(ctx, func(tx pgx.Tx, org uuid.UUID) error {
		stale, e := rows(ctx, tx, "UPDATE app.outbound_intents SET state='UNCERTAIN',error_code='DISPATCH_UNCERTAIN',finished_at=now() WHERE state='DISPATCHING' AND dispatch_started_at<now()-interval '1 minute' RETURNING id,message_id,conversation_id")
		if e != nil {
			return e
		}
		for _, item := range stale {
			if _, e = tx.Exec(ctx, "UPDATE app.send_attempts SET state='UNCERTAIN',error_code='DISPATCH_UNCERTAIN',finished_at=now() WHERE intent_id=$1 AND state='DISPATCHING'", item["id"]); e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, "UPDATE app.messages SET delivery_state='UNCERTAIN',error_code='DISPATCH_UNCERTAIN' WHERE id=$1", item["message_id"]); e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, "INSERT INTO app.audit_log(id,organization_id,actor_kind,action,resource_id,request_id) VALUES($1,$2,'SYSTEM','inbox.dispatch.recovered_uncertain',$3,'inbox-worker')", newID(), org, item["id"]); e != nil {
				return e
			}
			cid, e := uuid.Parse(item["conversation_id"].(string))
			if e != nil {
				return e
			}
			if e = event(ctx, tx, org, cid, "message.status_changed"); e != nil {
				return e
			}
		}
		if _, e = tx.Exec(ctx, "UPDATE app.conversations SET status='OPEN',snoozed_until=NULL,revision=revision+1 WHERE status='SNOOZED' AND snoozed_until<=now()"); e != nil {
			return e
		}
		_, e = tx.Exec(ctx, "DELETE FROM app.inbox_presence WHERE expires_at<now()-interval '1 minute'")
		return e
	})
}
func (s *Service) RunWorker(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		for _, fn := range []func(context.Context) error{s.MaterializeOnce, s.recoverUncertain, s.DispatchOnce} {
			if e := fn(ctx); e != nil && !errors.Is(e, pgx.ErrNoRows) && ctx.Err() == nil {
				slog.Error("Inbox work failed; inspect scoped evidence")
			}
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
