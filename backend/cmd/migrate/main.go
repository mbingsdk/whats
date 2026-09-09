package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
	"waba.local/control/internal/migrate"
)

func main() {
	dir := flag.String("dir", "../database/migrations", "reviewed migration directory")
	flag.Parse()
	dsn := os.Getenv("MIGRATION_DATABASE_URL")
	if path := os.Getenv("MIGRATION_DATABASE_URL_FILE"); path != "" {
		if dsn != "" {
			fmt.Fprintln(os.Stderr, "ambiguous migration credential source")
			os.Exit(1)
		}
		b, e := os.ReadFile(path)
		if e != nil {
			fmt.Fprintln(os.Stderr, "migration credential file unreadable")
			os.Exit(1)
		}
		dsn = strings.TrimSpace(string(b))
	}
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "migration database configuration required")
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	count, err := migrate.Up(ctx, dsn, os.DirFS(*dir))
	if err != nil {
		fmt.Fprintln(os.Stderr, "migration failed; inspect schema/role/connectivity using restricted diagnostics")
		os.Exit(1)
	}
	fmt.Printf("Applied %d migration(s)\n", count)
}
