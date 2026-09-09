package config

import (
	"errors"
	"log/slog"
	"net"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Secret prevents accidental structured logging of credentials.
type Secret string

func (Secret) LogValue() slog.Value { return slog.StringValue("[REDACTED]") }
func (Secret) String() string       { return "[REDACTED]" }

type SMTP struct {
	Host     string
	Port     int
	TLSMode  string
	Sender   string
	Username string
	Password Secret
	Timeout  time.Duration
}
type Config struct {
	Environment, Listen, PublicOrigin                       string
	DatabaseURL                                             Secret
	PoolMax                                                 int32
	LogLevel                                                slog.Level
	ReadTimeout, WriteTimeout, IdleTimeout, ShutdownTimeout time.Duration
	MaxHeaderBytes                                          int
	MaxBodyBytes                                            int64
	SMTP                                                    SMTP
}

func Load() (Config, error) { return Parse(os.Getenv) }
func Parse(get func(string) string) (Config, error) {
	c := Config{Environment: get("APP_ENV"), Listen: get("HTTP_ADDR"), PublicOrigin: get("PUBLIC_ORIGIN")}
	if c.Environment == "" {
		c.Environment = "development"
	}
	if c.Environment != "development" && c.Environment != "test" && c.Environment != "production" {
		return c, errors.New("APP_ENV must be development, test or production")
	}
	production := c.Environment == "production"
	if c.Listen == "" {
		c.Listen = "127.0.0.1:8080"
	}
	if _, _, err := net.SplitHostPort(c.Listen); err != nil {
		return c, errors.New("HTTP_ADDR must contain host and port")
	}
	if c.PublicOrigin == "" && !production {
		c.PublicOrigin = "http://localhost:3000"
	}
	u, err := url.Parse(c.PublicOrigin)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || (u.Scheme != "https" && u.Scheme != "http") {
		return c, errors.New("PUBLIC_ORIGIN must be an http(s) origin")
	}
	if production && u.Scheme != "https" {
		return c, errors.New("PUBLIC_ORIGIN requires https in production")
	}
	raw, err := readSecret(get, "DATABASE_URL")
	if err != nil || raw == "" {
		return c, errors.New("DATABASE_URL or DATABASE_URL_FILE is required and must be readable")
	}
	d, err := url.Parse(raw)
	if err != nil || (d.Scheme != "postgres" && d.Scheme != "postgresql") || d.Hostname() == "" || d.Path == "" {
		return c, errors.New("DATABASE_URL must be a PostgreSQL URL")
	}
	if production && d.Query().Get("sslmode") != "verify-full" {
		return c, errors.New("DATABASE_URL requires sslmode=verify-full in production")
	}
	c.DatabaseURL = Secret(raw)
	number := func(name string, def, min, max int) (int, error) {
		v := get(name)
		if v == "" {
			return def, nil
		}
		n, e := strconv.Atoi(v)
		if e != nil || n < min || n > max {
			return 0, errors.New(name + " out of range")
		}
		return n, nil
	}
	n, err := number("DB_POOL_MAX", 10, 1, 100)
	if err != nil {
		return c, err
	}
	c.PoolMax = int32(n)
	c.MaxHeaderBytes, err = number("HTTP_MAX_HEADER_BYTES", 16384, 1024, 1048576)
	if err != nil {
		return c, err
	}
	n, err = number("HTTP_MAX_BODY_BYTES", 1048576, 1, 16777216)
	if err != nil {
		return c, err
	}
	c.MaxBodyBytes = int64(n)
	if level := get("LOG_LEVEL"); level != "" {
		if err = c.LogLevel.UnmarshalText([]byte(level)); err != nil {
			return c, errors.New("LOG_LEVEL invalid")
		}
	}
	duration := func(name string, def time.Duration) (time.Duration, error) {
		if get(name) == "" {
			return def, nil
		}
		v, e := time.ParseDuration(get(name))
		if e != nil || v <= 0 || v > 5*time.Minute {
			return 0, errors.New(name + " out of range")
		}
		return v, nil
	}
	for _, v := range []struct {
		name string
		dst  *time.Duration
		def  time.Duration
	}{
		{"HTTP_READ_TIMEOUT", &c.ReadTimeout, 10 * time.Second}, {"HTTP_WRITE_TIMEOUT", &c.WriteTimeout, 15 * time.Second},
		{"HTTP_IDLE_TIMEOUT", &c.IdleTimeout, 60 * time.Second}, {"SHUTDOWN_TIMEOUT", &c.ShutdownTimeout, 10 * time.Second},
		{"SMTP_TIMEOUT", &c.SMTP.Timeout, 10 * time.Second},
	} {
		*v.dst, err = duration(v.name, v.def)
		if err != nil {
			return c, err
		}
	}
	c.SMTP.Host = get("SMTP_HOST")
	c.SMTP.TLSMode = get("SMTP_TLS_MODE")
	c.SMTP.Sender = get("SMTP_SENDER")
	c.SMTP.Username = get("SMTP_USERNAME")
	c.SMTP.Port, err = number("SMTP_PORT", 587, 1, 65535)
	if err != nil {
		return c, err
	}
	password, err := readSecret(get, "SMTP_PASSWORD")
	if err != nil {
		return c, errors.New("SMTP_PASSWORD source invalid")
	}
	c.SMTP.Password = Secret(password)
	anySMTP := c.SMTP.Host != "" || c.SMTP.Sender != "" || c.SMTP.Username != "" || password != "" || c.SMTP.TLSMode != "" || get("SMTP_PORT") != ""
	if production || anySMTP {
		if c.SMTP.Host == "" || strings.ContainsAny(c.SMTP.Host, " /\r\n:") {
			return c, errors.New("SMTP_HOST must be a hostname")
		}
		if c.SMTP.TLSMode != "starttls" && c.SMTP.TLSMode != "tls" {
			return c, errors.New("SMTP_TLS_MODE must be starttls or tls")
		}
		if address, e := mail.ParseAddress(c.SMTP.Sender); e != nil || address.Address != c.SMTP.Sender {
			return c, errors.New("SMTP_SENDER must be a mailbox address")
		}
		if c.SMTP.Username == "" || password == "" {
			return c, errors.New("authenticated SMTP credentials required")
		}
		if production && get("SMTP_PASSWORD_FILE") == "" {
			return c, errors.New("production SMTP credentials require SMTP_PASSWORD_FILE")
		}
	}
	return c, nil
}
func readSecret(get func(string) string, key string) (string, error) {
	value, path := get(key), get(key+"_FILE")
	if value != "" && path != "" {
		return "", errors.New("ambiguous secret source")
	}
	if path != "" {
		b, e := os.ReadFile(path)
		if e != nil {
			return "", errors.New("secret file unreadable")
		}
		value = strings.TrimSpace(string(b))
	}
	return value, nil
}
