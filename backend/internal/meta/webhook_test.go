package meta

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"strings"
	"testing"
)

func signature(b []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	_, _ = h.Write(b)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}
func TestChallengeFailClosed(t *testing.T) {
	q := url.Values{"hub.mode": {"subscribe"}, "hub.challenge": {"12345"}, "hub.verify_token": {"synthetic-verify-token"}}
	c, ok := verifyChallenge(q.Encode(), "synthetic-verify-token")
	if !ok || c != "12345" {
		t.Fatal("valid challenge rejected")
	}
	for _, key := range []string{"hub.mode", "hub.challenge", "hub.verify_token"} {
		copy := url.Values{}
		for k, v := range q {
			copy[k] = append([]string{}, v...)
		}
		copy.Del(key)
		if _, ok := verifyChallenge(copy.Encode(), "synthetic-verify-token"); ok {
			t.Fatal("missing accepted")
		}
		copy[key] = []string{"wrong"}
		if key != "hub.challenge" {
			if _, ok := verifyChallenge(copy.Encode(), "synthetic-verify-token"); ok {
				t.Fatal("wrong accepted")
			}
		}
		copy[key] = []string{"subscribe", "subscribe"}
		if _, ok := verifyChallenge(copy.Encode(), "synthetic-verify-token"); ok {
			t.Fatal("duplicate accepted")
		}
	}
	if _, ok := verifyChallenge(q.Encode()+"&bad=%zz", "synthetic-verify-token"); ok {
		t.Fatal("malformed query")
	}
}
func TestRawSignature(t *testing.T) {
	raw := []byte(`{"object":"whatsapp_business_account"}`)
	sig := signature(raw, "synthetic-app-secret")
	if !validSignature(raw, []string{sig}, "synthetic-app-secret") {
		t.Fatal("valid")
	}
	for _, v := range [][]string{nil, {""}, {"sha1=" + sig[7:]}, {"sha256=xyz"}, {sig, sig}, {signature(raw, "synthetic-verify-token")}} {
		if validSignature(raw, v, "synthetic-app-secret") {
			t.Fatal("invalid signature accepted")
		}
	}
	if validSignature([]byte(`{ "object":"whatsapp_business_account"}`), []string{sig}, "synthetic-app-secret") {
		t.Fatal("re-marshaled bytes accepted")
	}
}

const inbound = `{"object":"whatsapp_business_account","entry":[{"id":"222","changes":[{"field":"messages","value":{"metadata":{"phone_number_id":"333"},"messages":[{"id":"wamid.synthetic","text":{"body":"PRIVATE TEXT"}}]}}]}]}`

func TestClassifierIdempotencyUnknownAndRedaction(t *testing.T) {
	facts, state := classify([]byte(inbound))
	if state != "PROCESSED" || len(facts) != 1 || facts[0].Class != "INBOUND_MESSAGE" {
		t.Fatal(state)
	}
	var data any
	_ = json.Unmarshal([]byte(inbound), &data)
	changed, _ := json.MarshalIndent(data, "", " ")
	again, _ := classify(changed)
	if facts[0].Key != again[0].Key {
		t.Fatal("semantic identity unstable")
	}
	unknown := strings.Replace(inbound, `"field":"messages"`, `"field":"new_future_field"`, 1)
	f, s := classify([]byte(unknown))
	if s != "UNKNOWN" || !f[0].Uncertain {
		t.Fatal("unknown classification")
	}
	redacted, _ := json.Marshal(redactPayload([]byte(inbound)))
	for _, secret := range []string{"PRIVATE TEXT", "wamid.synthetic", "333"} {
		if strings.Contains(string(redacted), secret) {
			t.Fatal("PII leaked")
		}
	}
	_, s = classify([]byte(`{"entry":[{"id":"222","changes":[]}]}`))
	if s != "INVALID" {
		t.Fatal("invalid classification")
	}
}
