package partner

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const keyPrefixLen = 8

// GenerateAPIKey creates a plaintext key pxk_<prefix>_<secret> and bcrypt hash.
func GenerateAPIKey() (plaintext, prefix, hash string, err error) {
	prefixBytes := make([]byte, 6)
	secretBytes := make([]byte, 24)
	if _, err = rand.Read(prefixBytes); err != nil {
		return "", "", "", err
	}
	if _, err = rand.Read(secretBytes); err != nil {
		return "", "", "", err
	}
	prefix = hex.EncodeToString(prefixBytes)[:keyPrefixLen]
	secret := base64.RawURLEncoding.EncodeToString(secretBytes)
	plaintext = "pxk_" + prefix + "_" + secret
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	if err != nil {
		return "", "", "", err
	}
	return plaintext, prefix, string(hashBytes), nil
}

// ParseBearerKey extracts prefix from pxk_<prefix>_<rest>.
func ParseBearerKey(token string) (prefix string, ok bool) {
	token = strings.TrimSpace(token)
	if !strings.HasPrefix(token, "pxk_") {
		return "", false
	}
	rest := strings.TrimPrefix(token, "pxk_")
	idx := strings.IndexByte(rest, '_')
	if idx < 4 {
		return "", false
	}
	prefix = rest[:idx]
	return prefix, true
}

// VerifyAPIKey checks plaintext against bcrypt hash.
func VerifyAPIKey(plaintext, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plaintext)) == nil
}

// HasScope reports whether principal includes scope (or wildcard "*").
func HasScope(scopes []string, need string) bool {
	need = strings.TrimSpace(need)
	for _, s := range scopes {
		s = strings.TrimSpace(s)
		if s == "*" || s == need {
			return true
		}
	}
	return false
}

// DefaultScopesForTenant returns starter scopes.
func DefaultScopesForTenant(tenantType string) []string {
	switch strings.ToUpper(strings.TrimSpace(tenantType)) {
	case TenantSupplier:
		return []string{ScopeOrdersRead, ScopeCatalogRead, ScopeInventoryRead, ScopeWebhooksManage, ScopeExportsRead}
	default:
		return []string{ScopeOrdersRead, ScopeOrdersWrite, ScopeCatalogRead, ScopeInventoryRead, ScopeWebhooksManage, ScopeExportsRead}
	}
}

// GenerateWebhookSecret returns a signing secret for HMAC.
func GenerateWebhookSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "whsec_" + base64.RawURLEncoding.EncodeToString(b), nil
}

// SignPayload returns hex HMAC-SHA256 of "timestamp.rawBody".
func SignPayload(secret string, timestampUnix int64, body []byte) string {
	msg := append([]byte(fmt.Sprintf("%d.", timestampUnix)), body...)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(msg)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifySignature checks HMAC with constant-time compare.
func VerifySignature(secret string, timestampUnix int64, body []byte, sigHex string) bool {
	want := SignPayload(secret, timestampUnix, body)
	return hmac.Equal([]byte(want), []byte(strings.TrimSpace(sigHex)))
}
