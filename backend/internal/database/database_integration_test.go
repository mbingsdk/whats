//go:build integration

package database

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"waba.local/control/internal/config"
	"waba.local/control/internal/httpapi"
	"waba.local/control/internal/migrate"
)

func newID(t *testing.T) uuid.UUID {
	t.Helper()
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	if id.Version() != 7 {
		t.Fatal("UUIDv7 required")
	}
	return id
}
func TestPostgresFoundation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	adminURL := os.Getenv("TEST_ADMIN_DATABASE_URL")
	if adminURL == "" {
		t.Fatal("TEST_ADMIN_DATABASE_URL required; integration tests never silently skip")
	}
	admin, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		t.Fatal("test admin connection failed")
	}
	defer admin.Close(context.Background())
	dbName := "waba_test_" + strings.ReplaceAll(newID(t).String(), "-", "")
	quoted := pgx.Identifier{dbName}.Sanitize()
	if _, err = admin.Exec(ctx, "CREATE DATABASE "+quoted+" OWNER waba_migrator"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if _, e := admin.Exec(cleanup, "DROP DATABASE "+quoted+" WITH (FORCE)"); e != nil {
			t.Error("test database cleanup failed")
		}
	}()
	setDB := func(raw string) string {
		u, e := url.Parse(raw)
		if e != nil {
			t.Fatal("invalid test connection")
		}
		u.Path = "/" + dbName
		return u.String()
	}
	migrationDSN := setDB(os.Getenv("TEST_MIGRATION_DATABASE_URL"))
	runtimeDSN := setDB(os.Getenv("TEST_RUNTIME_DATABASE_URL"))
	setup, err := pgx.Connect(ctx, setDB(adminURL))
	if err != nil {
		t.Fatal(err)
	}
	defer setup.Close(context.Background())
	if _, err = setup.Exec(ctx, "REVOKE ALL ON DATABASE "+quoted+" FROM PUBLIC; GRANT CONNECT ON DATABASE "+quoted+" TO waba_migrator,waba_runtime,waba_identity"); err != nil {
		t.Fatal(err)
	}
	files := os.DirFS(filepath.Join("..", "..", "..", "database", "migrations"))
	t.Run("clean_and_repeated_migration", func(t *testing.T) {
		count, e := migrate.Up(ctx, migrationDSN, files)
		if e != nil {
			t.Fatal(e)
		}
		if count != 3 {
			t.Fatalf("got %d initial migrations", count)
		}
		count, e = migrate.Up(ctx, migrationDSN, files)
		if e != nil || count != 0 {
			t.Fatalf("repeat count=%d err=%v", count, e)
		}
	})
	// Setup alone uses the isolated container administrator; every assertion below uses runtime.
	orgA, orgB, userA, userB, memberA, memberB := newID(t), newID(t), newID(t), newID(t), newID(t), newID(t)
	for _, v := range []struct {
		o, u, m uuid.UUID
		label   string
	}{{orgA, userA, memberA, "a"}, {orgB, userB, memberB, "b"}} {
		if _, err = setup.Exec(ctx, "INSERT INTO app.users(id,canonical_email,display_name) VALUES($1,$2,$3)", v.u, v.label+"@example.invalid", "SYNTHETIC "+v.label); err != nil {
			t.Fatal(err)
		}
		if _, err = setup.Exec(ctx, "INSERT INTO app.organizations(id,name,slug,timezone) VALUES($1,$2,$2,'UTC')", v.o, "synthetic-"+v.label); err != nil {
			t.Fatal(err)
		}
		if _, err = setup.Exec(ctx, "INSERT INTO app.organization_members(id,organization_id,user_id) VALUES($1,$2,$3)", v.m, v.o, v.u); err != nil {
			t.Fatal(err)
		}
	}
	pool, err := Open(ctx, runtimeDSN, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	assertSQLDenied := func(t *testing.T, sql string, args ...any) {
		t.Helper()
		tx, e := pool.Begin(ctx)
		if e != nil {
			t.Fatal(e)
		}
		defer tx.Rollback(ctx)
		_, e = tx.Exec(ctx, sql, args...)
		var pe *pgconn.PgError
		if !errors.As(e, &pe) || pe.Code != "42501" {
			t.Fatalf("expected SQL privilege denial, got %v", e)
		}
	}
	t.Run("runtime_privileges", func(t *testing.T) {
		var unsafe bool
		err := pool.QueryRow(ctx, `SELECT rolsuper OR rolbypassrls OR rolcreaterole OR rolcreatedb OR
  pg_has_role(current_user,'waba_migrator','MEMBER') OR pg_has_role(current_user,'waba_authorizer','MEMBER')
  FROM pg_roles WHERE rolname=current_user`).Scan(&unsafe)
		if err != nil || unsafe {
			t.Fatalf("unsafe role: %v %v", unsafe, err)
		}
		if err = Ready(ctx, pool); err != nil {
			t.Fatal(err)
		}
		assertSQLDenied(t, "ALTER TABLE app.organization_members DISABLE ROW LEVEL SECURITY")
		assertSQLDenied(t, "SET ROLE waba_migrator")
		assertSQLDenied(t, "SET ROLE waba_authorizer")
		assertSQLDenied(t, "SELECT * FROM app.users")
		assertSQLDenied(t, "SET LOCAL row_security=off; SELECT * FROM app.organization_members")
	})
	for _, v := range []struct {
		name                    string
		org, user, own, foreign uuid.UUID
	}{{"tenant_a", orgA, userA, memberA, memberB}, {"tenant_b", orgB, userB, memberB, memberA}} {
		t.Run(v.name, func(t *testing.T) {
			err := InOrganization(ctx, pool, v.user, v.org, func(tx pgx.Tx) error {
				var count int
				if e := tx.QueryRow(ctx, "SELECT count(*) FROM app.organization_members").Scan(&count); e != nil {
					return e
				}
				if count != 1 {
					return fmt.Errorf("read leaked: %d", count)
				}
				var id uuid.UUID
				if e := tx.QueryRow(ctx, "SELECT id FROM app.organization_members").Scan(&id); e != nil {
					return e
				}
				if id != v.own {
					return errors.New("foreign member visible")
				}
				tag, e := tx.Exec(ctx, "UPDATE app.organization_members SET access_revision=access_revision+1 WHERE id=$1", v.foreign)
				if e != nil {
					return e
				}
				if tag.RowsAffected() != 0 {
					return errors.New("foreign update succeeded")
				}
				tag, e = tx.Exec(ctx, "DELETE FROM app.organization_members WHERE id=$1", v.foreign)
				if e != nil {
					return e
				}
				if tag.RowsAffected() != 0 {
					return errors.New("foreign delete succeeded")
				}
				tag, e = tx.Exec(ctx, "UPDATE app.organization_members SET access_revision=access_revision+1 WHERE id=$1", v.own)
				if e != nil {
					return e
				}
				if tag.RowsAffected() != 1 {
					return errors.New("own update failed")
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
	t.Run("membership_checked_before_context", func(t *testing.T) {
		called := false
		err := InOrganization(ctx, pool, userA, orgB, func(pgx.Tx) error { called = true; return nil })
		if !errors.Is(err, ErrAccessDenied) || called {
			t.Fatalf("untrusted organization bypass: %v", err)
		}
	})
	t.Run("cross_tenant_insert_and_move", func(t *testing.T) {
		for _, sql := range []string{
			"INSERT INTO app.organization_members(id,organization_id,user_id) VALUES($1,$2,$3)",
			"UPDATE app.organization_members SET organization_id=$2 WHERE id=$1",
		} {
			id := newID(t)
			if strings.HasPrefix(sql, "UPDATE") {
				id = memberA
			}
			err := InOrganization(ctx, pool, userA, orgA, func(tx pgx.Tx) error {
				args := []any{id, orgB}
				if strings.HasPrefix(sql, "INSERT") {
					args = append(args, userA)
				}
				_, e := tx.Exec(ctx, sql, args...)
				return e
			})
			var pe *pgconn.PgError
			if !errors.As(err, &pe) || pe.Code != "42501" {
				t.Fatalf("expected WITH CHECK failure: %v", err)
			}
		}
	})
	t.Run("missing_context_and_pool_reuse", func(t *testing.T) {
		var pid int
		if e := pool.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&pid); e != nil {
			t.Fatal(e)
		}
		for _, rollback := range []bool{false, true} {
			sentinel := errors.New("rollback requested")
			err := InOrganization(ctx, pool, userA, orgA, func(tx pgx.Tx) error {
				var inner int
				if e := tx.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&inner); e != nil {
					return e
				}
				if inner != pid {
					return errors.New("test did not reuse connection")
				}
				if rollback {
					return sentinel
				}
				return nil
			})
			if err != nil && !errors.Is(err, sentinel) {
				t.Fatal(err)
			}
			var count int
			var setting string
			if e := pool.QueryRow(ctx, "SELECT coalesce(current_setting('app.organization_id',true),''),(SELECT count(*) FROM app.organization_members)").Scan(&setting, &count); e != nil {
				t.Fatal(e)
			}
			if setting != "" || count != 0 {
				t.Fatalf("sticky context: %q, rows=%d", setting, count)
			}
		}
	})
	t.Run("readiness_real_database", func(t *testing.T) {
		handler := httpapi.Handler(config.Config{MaxBodyBytes: 1024}, slog.New(slog.NewJSONHandler(io.Discard, nil)), func(c context.Context) error { return Ready(c, pool) })
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", "/readyz", nil))
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		deadConfig, e := pgxpool.ParseConfig(runtimeDSN)
		if e != nil {
			t.Fatal(e)
		}
		deadConfig.ConnConfig.Port = 1
		deadConfig.ConnConfig.ConnectTimeout = 100 * time.Millisecond
		dead, e := pgxpool.NewWithConfig(ctx, deadConfig)
		if e != nil {
			t.Fatal(e)
		}
		defer dead.Close()
		handler = httpapi.Handler(config.Config{MaxBodyBytes: 1024}, slog.New(slog.NewJSONHandler(io.Discard, nil)), func(c context.Context) error { return Ready(c, dead) })
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", "/readyz", nil))
		if w.Code != 503 {
			t.Fatal(w.Code)
		}
	})
	t.Run("cancelled_transaction_does_not_leak", func(t *testing.T) {
		cancelled, stop := context.WithCancel(ctx)
		stop()
		if e := InOrganization(cancelled, pool, userA, orgA, func(pgx.Tx) error { t.Fatal("cancelled callback ran"); return nil }); e == nil {
			t.Fatal("cancellation ignored")
		}
		var count int
		if e := pool.QueryRow(ctx, "SELECT count(*) FROM app.organization_members").Scan(&count); e != nil || count != 0 {
			t.Fatalf("after cancellation rows=%d err=%v", count, e)
		}
	})
}
