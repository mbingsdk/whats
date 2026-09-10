package identity

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

func CanonicalEmail(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if len(s) > 254 || strings.ContainsAny(s, "\r\n") {
		return "", ErrValidation
	}
	a, e := mail.ParseAddress(s)
	if e != nil || a.Address != s || !strings.Contains(s, "@") {
		return "", ErrValidation
	}
	// No provider-specific dot/plus rewriting; local-part case-insensitivity is company policy.
	return s, nil
}
func randomToken() string { return base64.RawURLEncoding.EncodeToString(randomBytes(32)) }
func randomBytes(n int) []byte {
	b := make([]byte, n)
	if _, e := rand.Read(b); e != nil {
		panic("entropy unavailable")
	}
	return b
}
func digest(s string) string {
	b := sha256.Sum256([]byte(s))
	return base64.RawURLEncoding.EncodeToString(b[:])
}
func HashPassword(password string) (string, error) {
	if !utf8.ValidString(password) || utf8.RuneCountInString(password) < 12 || utf8.RuneCountInString(password) > 256 {
		return "", ErrValidation
	}
	salt := randomBytes(16)
	hash := argon2.IDKey([]byte(password), salt, 2, 19456, 1, 32)
	return fmt.Sprintf("$argon2id$v=19$m=19456,t=2,p=1$%s$%s", base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(hash)), nil
}
func VerifyPassword(encoded, password string) (bool, bool) {
	if !utf8.ValidString(password) || len(password) > 1024 || utf8.RuneCountInString(password) > 256 {
		return false, false
	}
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return false, false
	}
	var memory, iterations uint32
	var threads uint8
	if _, e := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &threads); e != nil || memory < 8192 || memory > 131072 || iterations < 1 || iterations > 6 || threads < 1 || threads > 4 {
		return false, false
	}
	salt, e := base64.RawStdEncoding.DecodeString(parts[4])
	if e != nil || len(salt) != 16 {
		return false, false
	}
	expected, e := base64.RawStdEncoding.DecodeString(parts[5])
	if e != nil || len(expected) != 32 {
		return false, false
	}
	actual := argon2.IDKey([]byte(password), salt, iterations, memory, threads, 32)
	return subtle.ConstantTimeCompare(actual, expected) == 1, memory != 19456 || iterations != 2 || threads != 1
}

type envelope struct {
	Version          int
	Wrapped, Payload []byte
}

func aead(key []byte) (cipher.AEAD, error) {
	b, e := aes.NewCipher(key)
	if e != nil {
		return nil, e
	}
	return cipher.NewGCM(b)
}
func sealWith(key, plain, associated []byte) ([]byte, error) {
	a, e := aead(key)
	if e != nil {
		return nil, e
	}
	nonce := randomBytes(a.NonceSize())
	return a.Seal(nonce, nonce, plain, associated), nil
}
func openWith(key, data, associated []byte) ([]byte, error) {
	a, e := aead(key)
	if e != nil || len(data) < a.NonceSize() {
		return nil, errors.New("invalid encrypted value")
	}
	n := a.NonceSize()
	return a.Open(nil, data[:n], data[n:], associated)
}
func (s *Service) seal(plain []byte, associated string) ([]byte, error) {
	key := randomBytes(32)
	wrapped, e := sealWith(s.Root, key, []byte(associated+":key:v1"))
	if e != nil {
		return nil, e
	}
	payload, e := sealWith(key, plain, []byte(associated+":data:v1"))
	if e != nil {
		return nil, e
	}
	return json.Marshal(envelope{1, wrapped, payload})
}
func (s *Service) open(data []byte, associated string) ([]byte, error) {
	var v envelope
	if json.Unmarshal(data, &v) != nil || v.Version != 1 {
		return nil, errors.New("invalid encrypted value")
	}
	key, e := openWith(s.Root, v.Wrapped, []byte(associated+":key:v1"))
	if e != nil {
		return nil, e
	}
	return openWith(key, v.Payload, []byte(associated+":data:v1"))
}
