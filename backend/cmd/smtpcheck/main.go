package main

import (
	"context"
	"fmt"
	"os"
	"waba.local/control/internal/config"
	"waba.local/control/internal/smtpprobe"
)

func main() {
	c, err := config.Load()
	if err == nil {
		err = smtpprobe.Probe(context.Background(), c.SMTP)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "SMTP configuration/transport/authentication check failed; inspect controlled relay diagnostics")
		os.Exit(1)
	}
	fmt.Println("SMTP TLS/authentication/NOOP accepted; no email sent; inbox delivery not verified.")
}
