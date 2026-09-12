package meta

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"time"
)

func (s *Service) syncOne(ctx context.Context, b binding) error {
	var run, lease uuid.UUID
	var count int
	e := s.scoped(ctx, b, func(tx pgx.Tx) error {
		e := tx.QueryRow(ctx, `SELECT id,attempt_count FROM app.asset_sync_runs WHERE app_id=$1 AND ((state='QUEUED' AND next_attempt_at<=now()) OR (state='RUNNING' AND lease_until<now())) ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT 1`, b.App).Scan(&run, &count)
		if e != nil {
			return e
		}
		lease = newID()
		count++
		_, e = tx.Exec(ctx, "UPDATE app.asset_sync_runs SET state='RUNNING',attempt_count=$2,lease_token=$3,lease_until=now()+interval '3 minutes' WHERE id=$1", run, count, lease)
		return e
	})
	if e != nil {
		return e
	}
	deadline, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	var snapshot Snapshot
	var fetchErr error
	if count > 8 {
		fetchErr = &GraphError{Kind: "ATTEMPTS_EXHAUSTED"}
	} else {
		snapshot, fetchErr = s.Graph.Snapshot(deadline, s.Config.WABAID, s.Config.AppID)
	}
	return s.scoped(ctx, b, func(tx pgx.Tx) error {
		var matches bool
		if e := tx.QueryRow(ctx, "SELECT lease_token=$2 AND lease_until>now() FROM app.asset_sync_runs WHERE id=$1 FOR UPDATE", run, lease).Scan(&matches); e != nil {
			return e
		}
		if !matches {
			return errors.New("sync lease lost")
		}
		if fetchErr != nil {
			state := "FAILED"
			var g *GraphError
			if errors.As(fetchErr, &g) && g.Retryable && count < 8 {
				state = "QUEUED"
			}
			credential := "DEGRADED"
			if errors.As(fetchErr, &g) && (g.Kind == "AUTHENTICATION" || g.Kind == "PERMISSION") {
				credential = "BLOCKED"
			}
			if _, e := tx.Exec(ctx, "UPDATE app.asset_sync_runs SET state=$2,error_code=$3,next_attempt_at=now()+$4::interval,finished_at=CASE WHEN $2='FAILED' THEN now() ELSE NULL END,lease_token=NULL,lease_until=NULL WHERE id=$1", run, state, errorCode(fetchErr), backoff(count)); e != nil {
				return e
			}
			if _, e := tx.Exec(ctx, "UPDATE app.meta_apps SET last_error=$2,credential_state=$3,updated_at=now() WHERE id=$1", b.App, errorCode(fetchErr), credential); e != nil {
				return e
			}
			return audit(ctx, tx, b.Org, nil, "meta.sync."+state, run, "meta-worker")
		}
		if e := s.persistSnapshot(ctx, tx, b, snapshot); e != nil {
			return e
		}
		if _, e := tx.Exec(ctx, "UPDATE app.asset_sync_runs SET state='SUCCEEDED',finished_at=now(),error_code=NULL,lease_token=NULL,lease_until=NULL WHERE id=$1", run); e != nil {
			return e
		}
		if _, e := tx.Exec(ctx, "UPDATE app.meta_apps SET last_sync_at=now(),last_error=NULL,credential_state='READ_ACCESS_VERIFIED',updated_at=now() WHERE id=$1", b.App); e != nil {
			return e
		}
		return audit(ctx, tx, b.Org, nil, "meta.sync.succeeded", run, "meta-worker")
	})
}
func backoff(count int) string {
	if count > 8 {
		count = 8
	}
	return (time.Duration(1<<uint(count)) * time.Second).String()
}
func (s *Service) persistSnapshot(ctx context.Context, tx pgx.Tx, b binding, v Snapshot) error {
	var waba uuid.UUID
	e := tx.QueryRow(ctx, `INSERT INTO app.wabas(id,organization_id,app_id,external_id,name,timezone_id,subscribed,lifecycle,graph_version) VALUES($1,$2,$3,$4,$5,$6,$7,'PRESENT',$8)
 ON CONFLICT(external_id) DO UPDATE SET name=excluded.name,timezone_id=excluded.timezone_id,subscribed=excluded.subscribed,lifecycle='PRESENT',graph_version=excluded.graph_version,synced_at=now() WHERE app.wabas.organization_id=excluded.organization_id AND app.wabas.app_id=excluded.app_id RETURNING id`, newID(), b.Org, b.App, v.WABA.ID, v.WABA.Name, v.WABA.TimezoneID, v.Subscribed, s.Config.Version).Scan(&waba)
	if e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "UPDATE app.phone_numbers SET lifecycle='NOT_OBSERVED' WHERE waba_id=$1", waba); e != nil {
		return e
	}
	for _, p := range v.Phones {
		var phone uuid.UUID
		e = tx.QueryRow(ctx, `INSERT INTO app.phone_numbers(id,organization_id,waba_id,external_id,name,display_number,quality,platform,code_verification_status,lifecycle,graph_version)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,'PRESENT',$10) ON CONFLICT(external_id) DO UPDATE SET name=excluded.name,display_number=excluded.display_number,quality=excluded.quality,platform=excluded.platform,code_verification_status=excluded.code_verification_status,lifecycle='PRESENT',graph_version=excluded.graph_version,synced_at=now() WHERE app.phone_numbers.organization_id=excluded.organization_id AND app.phone_numbers.waba_id=excluded.waba_id RETURNING id`, newID(), b.Org, waba, p.ID, p.Name, p.Number, p.Quality, p.Platform, p.CodeVerification, s.Config.Version).Scan(&phone)
		if e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `INSERT INTO app.business_profiles(id,organization_id,phone_id,fields,graph_version) VALUES($1,$2,$3,$4,$5) ON CONFLICT(organization_id,phone_id) DO UPDATE SET fields=excluded.fields,graph_version=excluded.graph_version,synced_at=now()`, newID(), b.Org, phone, v.Profiles[p.ID], s.Config.Version); e != nil {
			return e
		}
	}
	return nil
}
