package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const SchemaVersion int64 = 1

var ErrAccessDenied = errors.New("organization access denied")

func Open(ctx context.Context, dsn string, max int32) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, errors.New("invalid database configuration")
	}
	cfg.MaxConns = max
	cfg.MinConns = 0
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.ConnConfig.ConnectTimeout = 3 * time.Second
	// Never attach a query tracer that logs SQL parameters.
	cfg.ConnConfig.RuntimeParams["application_name"] = "waba-api"
	return pgxpool.NewWithConfig(ctx, cfg)
}
func Ready(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return errors.New("database unavailable")
	}
	var version int64
	if err := pool.QueryRow(ctx, "SELECT coalesce(max(version_id),0) FROM public.goose_db_version WHERE is_applied").Scan(&version); err != nil {
		return fmt.Errorf("schema check: %w", err)
	}
	if version != SchemaVersion {
		return errors.New("incompatible schema")
	}
	var unsafe bool
	if err := pool.QueryRow(ctx, `SELECT r.rolsuper OR r.rolbypassrls OR r.rolcreaterole OR r.rolcreatedb OR has_schema_privilege(current_user,'app','CREATE') OR
 EXISTS(SELECT 1 FROM pg_class c WHERE c.relnamespace='app'::regnamespace AND pg_has_role(r.oid,c.relowner,'USAGE'))
 FROM pg_roles r WHERE r.rolname=current_user`).Scan(&unsafe); err != nil {
		return fmt.Errorf("role check: %w", err)
	}
	if unsafe {
		return errors.New("unsafe runtime database role")
	}
	return nil
}

// InOrganization accepts a server-authenticated user ID, not an HTTP-provided identity.
// Membership is checked under row locks before installing transaction-local scope.
// No tenant HTTP endpoints are exposed in Sprint 0.
func InOrganization(ctx context.Context, pool *pgxpool.Pool, userID, orgID uuid.UUID, fn func(pgx.Tx) error) error {
	if userID == uuid.Nil || orgID == uuid.Nil {
		return ErrAccessDenied
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin organization transaction: %w", err)
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = tx.Rollback(cleanup)
	}()
	var memberID *uuid.UUID
	if err = tx.QueryRow(ctx, "SELECT app.authorize_membership($1,$2)", userID, orgID).Scan(&memberID); err != nil {
		return fmt.Errorf("authorize membership: %w", err)
	}
	if memberID == nil {
		return ErrAccessDenied
	}
	if _, err = tx.Exec(ctx, "SELECT set_config('app.organization_id',$1,true)", orgID.String()); err != nil {
		return fmt.Errorf("set organization context: %w", err)
	}
	if err = fn(tx); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit organization transaction: %w", err)
	}
	return nil
}
