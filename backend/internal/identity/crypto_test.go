package identity

import (
	"bytes"
	"strings"
	"testing"
)

func TestPasswordAndEncryption(t *testing.T) {
	hash, e := HashPassword("synthetic-password-42")
	if e != nil {
		t.Fatal(e)
	}
	if valid, rehash := VerifyPassword(hash, "synthetic-password-42"); !valid || rehash {
		t.Fatal("password policy")
	}
	if valid, _ := VerifyPassword(hash, "wrong"); valid {
		t.Fatal("wrong password")
	}
	if _, e := HashPassword("short"); e == nil {
		t.Fatal("weak password")
	}
	for _, bad := range []string{"bad", strings.Replace(hash, "m=19456", "m=999999999", 1), strings.Replace(hash, "v=19", "v=18", 1)} {
		if ok, _ := VerifyPassword(bad, "synthetic-password-42"); ok {
			t.Fatal("unsafe hash accepted")
		}
	}
	s, e := New(nil, bytes.Repeat([]byte{3}, 32), "https://example.invalid", true)
	if e != nil {
		t.Fatal(e)
	}
	sealed, e := s.seal([]byte("synthetic-secret"), "totp:resource")
	if e != nil {
		t.Fatal(e)
	}
	if bytes.Contains(sealed, []byte("synthetic-secret")) {
		t.Fatal("plaintext ciphertext")
	}
	clear, e := s.open(sealed, "totp:resource")
	if e != nil || string(clear) != "synthetic-secret" {
		t.Fatal("round trip")
	}
	if _, e = s.open(sealed, "totp:other"); e == nil {
		t.Fatal("resource substitution")
	}
	s.Root = bytes.Repeat([]byte{4}, 32)
	if _, e = s.open(sealed, "totp:resource"); e == nil {
		t.Fatal("wrong root")
	}
}
func TestCanonicalEmail(t *testing.T) {
	email, e := CanonicalEmail("  PERSON+tag@Example.INVALID ")
	if e != nil || email != "person+tag@example.invalid" {
		t.Fatal("canonical email")
	}
	for _, bad := range []string{"bad", "Name <a@example.invalid>", "a@example.invalid\r\nBcc:x@y.invalid"} {
		if _, e = CanonicalEmail(bad); e == nil {
			t.Fatal("bad address accepted")
		}
	}
}

func TestPasswordLengthCountsUnicodeCharacters(t *testing.T) {
	if _, e := HashPassword(strings.Repeat("\u754c", 4)); e == nil {
		t.Fatal("four Unicode characters passed the twelve-character minimum")
	}
	password := strings.Repeat("\u754c", 100)
	hash, e := HashPassword(password)
	if e != nil {
		t.Fatal("valid Unicode password rejected")
	}
	if valid, _ := VerifyPassword(hash, password); !valid {
		t.Fatal("Unicode password did not round-trip")
	}
}
