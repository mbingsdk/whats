package config

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestConfigValidation(t *testing.T) {
	base := map[string]string{"DATABASE_URL": "postgres://runtime:synthetic@127.0.0.1:55432/waba?sslmode=disable"}
	tests := []struct {
		name  string
		patch map[string]string
		valid bool
	}{
		{"development", nil, true}, {"missing database", map[string]string{"DATABASE_URL": ""}, false},
		{"bad environment", map[string]string{"APP_ENV": "prod"}, false},
		{"invalid pool", map[string]string{"DB_POOL_MAX": "0"}, false},
		{"invalid timeout", map[string]string{"HTTP_READ_TIMEOUT": "-1s"}, false},
		{"bad origin", map[string]string{"PUBLIC_ORIGIN": "https://user:secret@example.test/path"}, false},
		{"production requires https", map[string]string{"APP_ENV": "production"}, false},
		{"production requires verified database", map[string]string{"APP_ENV": "production", "PUBLIC_ORIGIN": "https://app.example.test"}, false},
		{"smtp partial", map[string]string{"SMTP_HOST": "smtp.example.test"}, false},
		{"smtp insecure", map[string]string{"SMTP_HOST": "smtp.example.test", "SMTP_SENDER": "ops@example.test", "SMTP_USERNAME": "test", "SMTP_PASSWORD": "synthetic", "SMTP_TLS_MODE": "none"}, false},
		{"smtp configured", map[string]string{"SMTP_HOST": "smtp.example.test", "SMTP_SENDER": "ops@example.test", "SMTP_USERNAME": "test", "SMTP_PASSWORD": "synthetic", "SMTP_TLS_MODE": "starttls"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := map[string]string{}
			for k, v := range base {
				m[k] = v
			}
			for k, v := range tt.patch {
				m[k] = v
			}
			_, err := Parse(func(k string) string { return m[k] })
			if (err == nil) != tt.valid {
				t.Fatalf("valid=%v, err=%v", tt.valid, err)
			}
			if err != nil && strings.Contains(err.Error(), "synthetic") {
				t.Fatal("secret leaked")
			}
		})
	}
}
func TestSecretLogging(t *testing.T) {
	var b bytes.Buffer
	l := slog.New(slog.NewJSONHandler(&b, nil))
	l.Info("test", "credential", Secret("private-test-value"))
	if strings.Contains(b.String(), "private-test-value") {
		t.Fatal("secret logged")
	}
}
