package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"waba.local/control/internal/config"
	"waba.local/control/internal/database"
	"waba.local/control/internal/identity"
	"waba.local/control/internal/meta"
)

func run() int {
	c, e := config.Load()
	if e != nil {
		slog.Error("runtime configuration invalid")
		return 1
	}
	mc, e := config.LoadMeta(os.Getenv)
	if e != nil || !mc.Enabled {
		slog.Error("Meta configuration required")
		return 1
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pool, e := database.Open(ctx, string(c.DatabaseURL), c.PoolMax)
	if e != nil {
		slog.Error("database unavailable")
		return 1
	}
	defer pool.Close()
	if e = database.Ready(ctx, pool); e != nil {
		slog.Error("database readiness failed")
		return 1
	}
	auth, e := identity.Load(ctx, c.PublicOrigin, c.Environment == "production")
	if e != nil {
		slog.Error("identity configuration invalid")
		return 1
	}
	defer auth.Pool.Close()
	if e = meta.New(mc, pool, auth).RunWorker(ctx); e != nil {
		slog.Error("Meta worker stopped")
		return 1
	}
	return 0
}
func main() { os.Exit(run()) }
