package main

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"os"
	"strings"
	"time"
	"waba.local/control/internal/identity"
)

func run() int {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	service, e := identity.Load(ctx, os.Getenv("PUBLIC_ORIGIN"), os.Getenv("APP_ENV") == "production")
	if e != nil {
		fmt.Fprintln(os.Stderr, "Bootstrap configuration unavailable.")
		return 1
	}
	defer service.Pool.Close()
	password, e := os.ReadFile(os.Getenv("BOOTSTRAP_PASSWORD_FILE"))
	if e != nil {
		fmt.Fprintln(os.Stderr, "BOOTSTRAP_PASSWORD_FILE required.")
		return 1
	}
	in := identity.Input{Email: os.Getenv("BOOTSTRAP_EMAIL"), Password: strings.TrimRight(string(password), "\r\n"), Name: os.Getenv("BOOTSTRAP_NAME"), Slug: os.Getenv("BOOTSTRAP_ORGANIZATION_SLUG"), Timezone: os.Getenv("BOOTSTRAP_TIMEZONE")}
	if value := os.Getenv("BOOTSTRAP_ORGANIZATION_ID"); value != "" {
		org, e := uuid.Parse(value)
		if e != nil {
			fmt.Fprintln(os.Stderr, "Invalid bootstrap organization ID.")
			return 1
		}
		in.OrganizationID = org
	}
	if service.Origin == "" {
		fmt.Fprintln(os.Stderr, "PUBLIC_ORIGIN required.")
		return 1
	}
	if e = service.Bootstrap(ctx, in); e != nil {
		if e == identity.ErrInitialized {
			fmt.Println("Already initialized; no owner created.")
			return 0
		}
		fmt.Fprintln(os.Stderr, "Bootstrap failed; verify protected input and database configuration.")
		return 1
	}
	fmt.Println("Owner initialized. Complete email verification and MFA enrollment.")
	return 0
}
func main() { os.Exit(run()) }
