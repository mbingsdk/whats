// Package smtpprobe checks SMTP transport/authentication without sending email.
package smtpprobe

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/smtp"
	"strconv"
	"waba.local/control/internal/config"
)

func Probe(ctx context.Context, c config.SMTP) error {
	if c.Host == "" || c.Port < 1 || c.Port > 65535 || c.Timeout <= 0 || c.Username == "" || c.Password == "" || (c.TLSMode != "tls" && c.TLSMode != "starttls") {
		return errors.New("SMTP configuration incomplete")
	}
	return probe(ctx, c, &tls.Config{ServerName: c.Host, MinVersion: tls.VersionTLS12})
}
func probe(ctx context.Context, c config.SMTP, tlsConfig *tls.Config) error {
	ctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", net.JoinHostPort(c.Host, strconv.Itoa(c.Port)))
	if err != nil {
		return errors.New("SMTP connection failed")
	}
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	deadline, _ := ctx.Deadline()
	_ = conn.SetDeadline(deadline)
	if c.TLSMode == "tls" {
		secure := tls.Client(conn, tlsConfig)
		if err = secure.HandshakeContext(ctx); err != nil {
			return errors.New("SMTP TLS verification failed")
		}
		conn = secure
	}
	client, err := smtp.NewClient(conn, c.Host)
	if err != nil {
		return errors.New("SMTP greeting failed")
	}
	defer client.Close()
	if c.TLSMode == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return errors.New("SMTP STARTTLS required")
		}
		if err = client.StartTLS(tlsConfig); err != nil {
			return errors.New("SMTP TLS verification failed")
		}
	}
	if err = client.Auth(smtp.PlainAuth("", c.Username, string(c.Password), c.Host)); err != nil {
		return errors.New("SMTP authentication failed")
	}
	if err = client.Noop(); err != nil {
		return errors.New("SMTP probe failed")
	}
	if err = client.Quit(); err != nil {
		return errors.New("SMTP probe completion failed")
	}
	return nil
}
