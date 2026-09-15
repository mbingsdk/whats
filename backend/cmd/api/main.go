package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"waba.local/control/internal/config"
	"waba.local/control/internal/database"
	"waba.local/control/internal/httpapi"
	"waba.local/control/internal/identity"
	"waba.local/control/internal/inbox"
	"waba.local/control/internal/meta"
)

func run() int {
	c, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: c.LogLevel}))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pool, err := database.Open(ctx, string(c.DatabaseURL), c.PoolMax)
	if err != nil {
		logger.Error("database configuration invalid")
		return 1
	}
	defer pool.Close()
	auth, err := identity.Load(ctx, c.PublicOrigin, c.Environment == "production")
	if err != nil {
		logger.Error("identity configuration invalid")
		return 1
	}
	defer auth.Pool.Close()
	mc, err := config.LoadMeta(os.Getenv)
	if err != nil {
		logger.Error("Meta configuration invalid")
		return 1
	}
	metaService := meta.New(mc, pool, auth)
	live, err := inbox.LoadLive(os.Getenv)
	if err != nil {
		logger.Error("Inbox live acceptance configuration invalid")
		return 1
	}
	inboxService := inbox.New(auth, metaService, live)
	handler := httpapi.Handler(c, logger, func(ctx context.Context) error {
		if e := database.Ready(ctx, pool); e != nil {
			return e
		}
		return auth.Pool.Ping(ctx)
	}, inboxService.Handler(metaService.Handler(auth.Handler())))
	listener, err := net.Listen("tcp", c.Listen)
	if err != nil {
		logger.Error("HTTP listen failed")
		return 1
	}
	logger.Info("API starting", slog.String("environment", c.Environment), slog.Int64("schema_version", database.SchemaVersion))
	if err = httpapi.Serve(ctx, httpapi.Server(c, handler), listener, c.ShutdownTimeout); err != nil {
		logger.Error("HTTP shutdown failed")
		return 1
	}
	logger.Info("API stopped")
	return 0
}
func main() { os.Exit(run()) }
