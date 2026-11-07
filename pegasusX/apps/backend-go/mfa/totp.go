package mfa

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	totpDigits     = 6
	totpPeriod     = 30
	totpSkewWindows = 1
	secretBytes    = 20
)

// GenerateSecret returns a new base32-encoded TOTP shared secret (no padding).
func GenerateSecret() (string, error) {
	buf := make([]byte, secretBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf), nil
}

// OTPAuthURL builds an otpauth:// URI for authenticator apps.
func OTPAuthURL(issuer, accountName, secret string) string {
	issuer = strings.TrimSpace(issuer)
	if issuer == "" {
		issuer = "PegasusX"
	}
	accountName = strings.TrimSpace(accountName)
	if accountName == "" {
		accountName = "platform-admin"
	}
	label := url.PathEscape(issuer + ":" + accountName)
	q := url.Values{}
	q.Set("secret", secret)
	q.Set("issuer", issuer)
	q.Set("algorithm", "SHA1")
	q.Set("digits", fmt.Sprintf("%d", totpDigits))
	q.Set("period", fmt.Sprintf("%d", totpPeriod))
	return "otpauth://totp/" + label + "?" + q.Encode()
}

// ValidateCode checks a TOTP code against secret with ±skew windows and returns the matching time step.
func ValidateCode(secret, code string, now time.Time) (bool, uint64) {
	code = strings.TrimSpace(code)
	if len(code) != totpDigits {
		return false, 0
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			return false, 0
		}
	}
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil || len(key) == 0 {
		return false, 0
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	counter := now.Unix() / totpPeriod
	for d := int64(-totpSkewWindows); d <= totpSkewWindows; d++ {
		step := uint64(counter + d)
		if hotp(key, step) == code {
			return true, step
		}
	}
	return false, 0
}

func hotp(key []byte, counter uint64) string {
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], counter)
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(msg[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	truncated := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	mod := uint32(1)
	for i := 0; i < totpDigits; i++ {
		mod *= 10
	}
	return fmt.Sprintf("%0*d", totpDigits, truncated%mod)
}
