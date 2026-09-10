package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
	"waba.local/control/internal/config"
	"waba.local/control/internal/identity"
)

func run() int {
	c, e := config.Load()
	if e != nil {
		slog.Error("mail configuration invalid")
		return 1
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	s, e := identity.Load(ctx, c.PublicOrigin, c.Environment == "production")
	if e != nil {
		slog.Error("identity configuration invalid")
		return 1
	}
	defer s.Pool.Close()
	sender := identity.SMTPTransport{Config: c.SMTP}
	for ctx.Err() == nil {
		worked, e := s.DeliverOne(ctx, sender)
		if e != nil {
			slog.Error("mail worker paused; inspect protected configuration and sanitized delivery status")
		}
		delay := time.Second
		if worked && e == nil {
			delay = 50 * time.Millisecond
		}
		if e != nil {
			delay = time.Minute
		}
		select {
		case <-ctx.Done():
			return 0
		case <-time.After(delay):
		}
	}
	return 0
}
func main() { os.Exit(run()) }
