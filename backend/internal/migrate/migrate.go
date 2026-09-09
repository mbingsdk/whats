package migrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

func Up(ctx context.Context, dsn string, files fs.FS) (int, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return 0, errors.New("invalid migration database configuration")
	}
	defer db.Close()
	db.SetMaxOpenConns(2)
	var role string
	var elevated bool
	if err = db.QueryRowContext(ctx, "SELECT current_user,rolsuper OR rolbypassrls FROM pg_roles WHERE rolname=current_user").Scan(&role, &elevated); err != nil {
		return 0, fmt.Errorf("migration connection: %w", err)
	}
	if role != "waba_migrator" || elevated {
		return 0, errors.New("dedicated non-superuser waba_migrator required")
	}
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return 0, err
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, db, files, goose.WithSessionLocker(locker))
	if err != nil {
		return 0, err
	}
	results, err := provider.Up(ctx)
	return len(results), err
}
