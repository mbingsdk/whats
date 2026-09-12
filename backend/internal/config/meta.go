package config

import (
	"errors"
	"os"
	"regexp"
	"strings"
)

// Meta is disabled when all inputs are absent. Partial configuration fails closed.
type Meta struct {
	Enabled                                      bool
	Version, AppID, PortfolioID, WABAID, PhoneID string
	AccessToken, AppSecret, VerifyToken          Secret
}

func LoadMeta(get func(string) string) (Meta, error) {
	c := Meta{Version: get("META_GRAPH_VERSION"), AppID: get("META_APP_ID"), PortfolioID: get("META_BUSINESS_PORTFOLIO_ID"), WABAID: get("META_WABA_ID"), PhoneID: get("META_PHONE_NUMBER_ID")}
	keys := []string{"META_GRAPH_VERSION", "META_APP_ID", "META_BUSINESS_PORTFOLIO_ID", "META_WABA_ID", "META_PHONE_NUMBER_ID", "META_ACCESS_TOKEN_FILE", "META_APP_SECRET_FILE", "META_WEBHOOK_VERIFY_TOKEN_FILE"}
	for _, k := range keys {
		if get(k) != "" {
			c.Enabled = true
		}
	}
	if !c.Enabled {
		return c, nil
	}
	if c.Version != "v26.0" {
		return c, errors.New("META_GRAPH_VERSION must match the reviewed v26.0 contract")
	}
	numeric := regexp.MustCompile(`^[0-9]{1,32}$`)
	if !numeric.MatchString(c.AppID) || !numeric.MatchString(c.WABAID) {
		return c, errors.New("META_APP_ID and META_WABA_ID required")
	}
	for _, v := range []string{c.PortfolioID, c.PhoneID} {
		if v != "" && !numeric.MatchString(v) {
			return c, errors.New("Meta identifier invalid")
		}
	}
	for _, s := range []struct {
		k   string
		dst *Secret
	}{{"META_ACCESS_TOKEN", &c.AccessToken}, {"META_APP_SECRET", &c.AppSecret}, {"META_WEBHOOK_VERIFY_TOKEN", &c.VerifyToken}} {
		if get(s.k) != "" {
			return c, errors.New("Meta secrets require protected files")
		}
		b, e := os.ReadFile(get(s.k + "_FILE"))
		v := strings.TrimSpace(strings.TrimPrefix(string(b), "\ufeff"))
		if e != nil || len(v) < 16 || len(v) > 8192 || strings.ContainsAny(v, "\r\n\x00") {
			return c, errors.New(s.k + "_FILE unreadable or invalid")
		}
		*s.dst = Secret(v)
	}
	if c.AccessToken == c.AppSecret || c.VerifyToken == c.AppSecret || c.VerifyToken == c.AccessToken {
		return c, errors.New("Meta secrets must be distinct")
	}
	return c, nil
}
