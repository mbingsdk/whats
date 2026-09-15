package inbox

import (
	"errors"
	"os"
	"strings"
	"time"
)

func LoadLive(get func(string) string) (LiveConfig, error) {
	var c LiveConfig
	enabled := get("INBOX_LIVE_ACCEPTANCE_ENABLED")
	if enabled != "" && enabled != "false" && enabled != "true" {
		return c, errors.New("INBOX_LIVE_ACCEPTANCE_ENABLED must be true or false")
	}
	c.Enabled = enabled == "true"
	if get("INBOX_TEST_RECIPIENT") != "" {
		return c, errors.New("controlled TEST recipient requires a protected file")
	}
	path := get("INBOX_TEST_RECIPIENT_FILE")
	if path != "" {
		b, e := os.ReadFile(path)
		if e != nil {
			return c, errors.New("controlled TEST recipient file unavailable")
		}
		c.Recipient = strings.TrimPrefix(strings.TrimSpace(strings.TrimPrefix(string(b), "\ufeff")), "+")
		if !identityDigits(c.Recipient) {
			return c, errors.New("controlled TEST recipient invalid")
		}
	}
	if c.Enabled && c.Recipient == "" {
		return c, errors.New("explicit controlled TEST recipient required")
	}
	if raw := get("INBOX_LIVE_ACCEPTANCE_NOT_BEFORE"); raw != "" {
		var e error
		c.AcceptanceNotBefore, e = time.Parse(time.RFC3339, raw)
		if e != nil {
			return c, errors.New("INBOX_LIVE_ACCEPTANCE_NOT_BEFORE must be RFC3339")
		}
	}
	if c.Enabled && c.AcceptanceNotBefore.IsZero() {
		return c, errors.New("explicit acceptance start timestamp required")
	}
	// There is no operator override for unverified delivery/pricing coverage.
	// The reviewed Direct Send TTL contract does not establish a Service text delivery bound.
	return c, nil
}
