package identity

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"io"
	"net"
	"net/smtp"
	"net/textproto"
	"strconv"
	"strings"
	"time"
	"waba.local/control/internal/config"
)

type MailPayload struct{ Recipient, Token, Purpose string }
type Mail struct {
	ID                       uuid.UUID
	Recipient, Subject, Body string
}

// Sender is the external SMTP effect boundary; integration tests substitute a local server.
type Sender interface {
	Send(context.Context, Mail) error
}
type SMTPTransport struct {
	Config    config.SMTP
	TLSConfig *tls.Config
}
type MailError struct {
	Code             string
	Permanent, Pause bool
}

func (e *MailError) Error() string { return e.Code }
func (t SMTPTransport) Send(ctx context.Context, m Mail) error {
	c := t.Config
	if c.Host == "" || c.Username == "" || c.Password == "" || (c.TLSMode != "tls" && c.TLSMode != "starttls") {
		return &MailError{Code: "SMTP_CONFIG", Pause: true}
	}
	ctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()
	conn, e := (&net.Dialer{}).DialContext(ctx, "tcp", net.JoinHostPort(c.Host, strconv.Itoa(c.Port)))
	if e != nil {
		return &MailError{Code: "SMTP_CONNECT"}
	}
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	deadline, _ := ctx.Deadline()
	_ = conn.SetDeadline(deadline)
	cfg := &tls.Config{ServerName: c.Host, MinVersion: tls.VersionTLS12}
	if t.TLSConfig != nil {
		cfg = t.TLSConfig.Clone()
		cfg.ServerName = c.Host
		cfg.MinVersion = tls.VersionTLS12
	}
	if cfg.InsecureSkipVerify {
		return &MailError{Code: "SMTP_TLS", Pause: true}
	}
	if c.TLSMode == "tls" {
		secure := tls.Client(conn, cfg)
		if e = secure.HandshakeContext(ctx); e != nil {
			return &MailError{Code: "SMTP_TLS", Pause: true}
		}
		conn = secure
	}
	client, e := smtp.NewClient(conn, c.Host)
	if e != nil {
		return &MailError{Code: "SMTP_GREETING"}
	}
	defer client.Close()
	if c.TLSMode == "starttls" {
		if yes, _ := client.Extension("STARTTLS"); !yes {
			return &MailError{Code: "SMTP_TLS", Pause: true}
		}
		if e = client.StartTLS(cfg); e != nil {
			return &MailError{Code: "SMTP_TLS", Pause: true}
		}
	}
	if e = client.Auth(smtp.PlainAuth("", c.Username, string(c.Password), c.Host)); e != nil {
		return &MailError{Code: "SMTP_AUTH", Pause: true}
	}
	classify := func(e error) error {
		var protocol *textproto.Error
		permanent := errors.As(e, &protocol) && protocol.Code >= 500
		return &MailError{Code: "SMTP_REJECTED", Permanent: permanent}
	}
	if strings.ContainsAny(m.Recipient+c.Sender+m.Subject, "\r\n") {
		return &MailError{Code: "MAIL_INVALID", Permanent: true}
	}
	if e = client.Mail(c.Sender); e != nil {
		return classify(e)
	}
	if e = client.Rcpt(m.Recipient); e != nil {
		return classify(e)
	}
	writer, e := client.Data()
	if e != nil {
		return classify(e)
	}
	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMessage-ID: <%s@waba-control.invalid>\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n", c.Sender, m.Recipient, m.Subject, m.ID, m.Body)
	if _, e = io.WriteString(writer, message); e != nil {
		return &MailError{Code: "SMTP_UNKNOWN"}
	}
	if e = writer.Close(); e != nil {
		return &MailError{Code: "SMTP_UNKNOWN"}
	}
	_ = client.Quit() // DATA completion already established acceptance.
	return nil
}
func (s *Service) mailValid(ctx context.Context, tx pgx.Tx, mid uuid.UUID, p MailPayload) (bool, error) {
	var challenge, invitation, org *uuid.UUID
	if e := tx.QueryRow(ctx, "SELECT challenge_id,invitation_id,organization_id FROM app.mail_deliveries WHERE id=$1", mid).Scan(&challenge, &invitation, &org); e != nil {
		return false, e
	}
	var valid bool
	if challenge != nil {
		e := tx.QueryRow(ctx, "SELECT EXISTS(SELECT FROM app.auth_challenges WHERE id=$1 AND token_digest=$2 AND consumed_at IS NULL AND revoked_at IS NULL AND expires_at>$3)", challenge, digest(p.Token), s.Now()).Scan(&valid)
		return valid, e
	}
	if invitation != nil {
		if org == nil {
			return false, nil
		}
		if e := scope(ctx, tx, *org); e != nil {
			return false, e
		}
		e := tx.QueryRow(ctx, "SELECT EXISTS(SELECT FROM app.invitations WHERE id=$1 AND token_digest=$2 AND state='PENDING' AND expires_at>$3)", invitation, digest(p.Token), s.Now()).Scan(&valid)
		return valid, e
	}
	return p.Purpose == "SECURITY_NOTICE", nil
}

// DeliverOne commits a fenced claim before SMTP. SMTP can duplicate after ambiguity;
// retries reuse the same logical delivery/link, never renew its challenge.
func (s *Service) DeliverOne(ctx context.Context, sender Sender) (bool, error) {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return false, e
	}
	defer rollback(tx)
	if e = exec(ctx, tx, `UPDATE app.mail_attempts SET result='UNKNOWN',error_code='WORKER_LEASE_EXPIRED',finished_at=$1
 WHERE finished_at IS NULL AND delivery_id IN (SELECT id FROM app.mail_deliveries WHERE state='SENDING' AND lease_until<$1)`, s.Now()); e != nil {
		return false, e
	}
	if e = exec(ctx, tx, "UPDATE app.mail_deliveries SET state='RETRY_WAIT',next_attempt_at=$1,lease_token=NULL WHERE state='SENDING' AND lease_until<$1", s.Now()); e != nil {
		return false, e
	}
	var mid uuid.UUID
	var data []byte
	var deadline time.Time
	var attempts int
	e = tx.QueryRow(ctx, `SELECT id,payload_ciphertext,deadline_at,attempt_count FROM app.mail_deliveries WHERE state IN ('QUEUED','RETRY_WAIT') AND next_attempt_at<=$1 ORDER BY next_attempt_at,id FOR UPDATE SKIP LOCKED LIMIT 1`, s.Now()).Scan(&mid, &data, &deadline, &attempts)
	if errors.Is(e, pgx.ErrNoRows) {
		return false, tx.Commit(ctx)
	}
	if e != nil {
		return false, e
	}
	terminal := func(state string) (bool, error) {
		if e := exec(ctx, tx, "UPDATE app.mail_deliveries SET state=$2,payload_ciphertext=NULL,terminal_at=$3 WHERE id=$1", mid, state, s.Now()); e != nil {
			return true, e
		}
		return true, tx.Commit(ctx)
	}
	if !s.Now().Before(deadline) {
		return terminal("EXPIRED")
	}
	if attempts >= 8 {
		return terminal("FAILED")
	}
	plain, e := s.open(data, "mail:"+mid.String())
	if e != nil {
		return terminal("FAILED")
	}
	var payload MailPayload
	if json.Unmarshal(plain, &payload) != nil {
		return terminal("FAILED")
	}
	valid, e := s.mailValid(ctx, tx, mid, payload)
	if e != nil {
		return true, e
	}
	if !valid {
		return terminal("CANCELLED")
	}
	lease := id()
	attempts++
	if e = exec(ctx, tx, "UPDATE app.mail_deliveries SET state='SENDING',attempt_count=$2,lease_token=$3,lease_until=$4 WHERE id=$1", mid, attempts, lease, s.Now().Add(2*time.Minute)); e != nil {
		return true, e
	}
	if e = exec(ctx, tx, "INSERT INTO app.mail_attempts(id,delivery_id,attempt_no) VALUES($1,$2,$3)", id(), mid, attempts); e != nil {
		return true, e
	}
	if e = tx.Commit(ctx); e != nil {
		return true, e
	}
	// Recheck immediately before the network effect, outside the claim transaction.
	pre, e := s.Pool.Begin(ctx)
	if e != nil {
		return true, e
	}
	valid, e = s.mailValid(ctx, pre, mid, payload)
	rollback(pre)
	if e != nil {
		return true, e
	}
	state, result, code := "SMTP_ACCEPTED", "SMTP_ACCEPTED", ""
	var sendErr error
	if !valid || !s.Now().Before(deadline) {
		state, result, code = "CANCELLED", "REJECTED", "SOURCE_OBSOLETE"
	} else {
		route := "/verify-email"
		subject := "Verify your email"
		switch payload.Purpose {
		case "INVITATION":
			route = "/accept-invitation"
			subject = "Company invitation"
		case "PASSWORD_RESET":
			route = "/reset-password"
			subject = "Reset your password"
		case "SECURITY_NOTICE":
			subject = "Account security change"
		}
		body := "A security setting on your WABA Control account changed. Contact your company administrator if this was not you."
		if payload.Token != "" {
			body = subject + "\n\n" + s.Origin + route + "#token=" + payload.Token + "\n\nThis link expires and can be used once."
		}
		sendContext, stopSend := context.WithTimeout(ctx, 90*time.Second)
		sendErr = sender.Send(sendContext, Mail{mid, payload.Recipient, subject, body})
		stopSend()
		if sendErr != nil {
			state, result, code = "RETRY_WAIT", "UNKNOWN", "SMTP_UNKNOWN"
			var detail *MailError
			if errors.As(sendErr, &detail) {
				code = detail.Code
				if detail.Permanent {
					state, result = "FAILED", "REJECTED"
				}
			}
			if attempts >= 8 {
				state = "FAILED"
			}
		}
	}
	finish, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	done, e := s.Pool.Begin(finish)
	if e != nil {
		return true, e
	}
	defer rollback(done)
	var current uuid.UUID
	if e = done.QueryRow(finish, "SELECT lease_token FROM app.mail_deliveries WHERE id=$1 AND state='SENDING' FOR UPDATE", mid).Scan(&current); e != nil {
		return true, e
	}
	if current != lease {
		return true, errors.New("mail lease lost")
	}
	delay := time.Duration(1<<uint(attempts))*time.Minute + time.Duration(randomBytes(1)[0])*time.Second
	if e = exec(finish, done, "UPDATE app.mail_attempts SET result=$3,error_code=$4,finished_at=$5 WHERE delivery_id=$1 AND attempt_no=$2 AND finished_at IS NULL", mid, attempts, result, code, s.Now()); e != nil {
		return true, e
	}
	if e = exec(finish, done, `UPDATE app.mail_deliveries SET state=$2,lease_token=NULL,lease_until=NULL,next_attempt_at=$3,
 payload_ciphertext=CASE WHEN $2='RETRY_WAIT' THEN payload_ciphertext ELSE NULL END,
 terminal_at=CASE WHEN $2='RETRY_WAIT' THEN NULL ELSE $4::timestamptz END WHERE id=$1`, mid, state, s.Now().Add(delay), s.Now()); e != nil {
		return true, e
	}
	if e = done.Commit(finish); e != nil {
		return true, e
	}
	var detail *MailError
	if errors.As(sendErr, &detail) && detail.Pause {
		return true, detail
	}
	return true, nil
}
