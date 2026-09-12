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

// GenerateAPIKey creates a plaintext live key pxk_<prefix>_<secret> and bcrypt hash.
func GenerateAPIKey() (plaintext, prefix, hash string, err error) {
	return generateAPIKeyWithScheme("pxk_")
}

// GenerateSandboxAPIKey creates a plaintext sandbox key pxs_<prefix>_<secret>.
func GenerateSandboxAPIKey() (plaintext, prefix, hash string, err error) {
	return generateAPIKeyWithScheme("pxs_")
}

func generateAPIKeyWithScheme(scheme string) (plaintext, prefix, hash string, err error) {
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
	plaintext = scheme + prefix + "_" + secret
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	if err != nil {
		return "", "", "", err
	}
	return plaintext, prefix, string(hashBytes), nil
}

// ParseBearerKey extracts prefix from pxk_<prefix>_<rest> or pxs_<prefix>_<rest>.
func ParseBearerKey(token string) (prefix string, ok bool) {
	token = strings.TrimSpace(token)
	var rest string
	switch {
	case strings.HasPrefix(token, "pxk_"):
		rest = strings.TrimPrefix(token, "pxk_")
	case strings.HasPrefix(token, "pxs_"):
		rest = strings.TrimPrefix(token, "pxs_")
	default:
		return "", false
	}
	idx := strings.IndexByte(rest, '_')
	if idx < 4 {
		return "", false
	}
	prefix = rest[:idx]
	return prefix, true
}

// IsSandboxKey reports whether the plaintext uses the sandbox scheme.
func IsSandboxKey(token string) bool {
	return strings.HasPrefix(strings.TrimSpace(token), "pxs_")
}

// IsSandboxRateClass reports sandbox via RateLimitClass (persisted marker).
func IsSandboxRateClass(class string) bool {
	return strings.EqualFold(strings.TrimSpace(class), RateLimitSandbox)
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
		return []string{
			ScopeOrdersRead, ScopeCatalogRead, ScopeCatalogWrite,
			ScopeInventoryRead, ScopeInventoryWrite, ScopeWebhooksManage, ScopeExportsRead, ScopeDemandWrite,
		}
	default:
		return []string{
			ScopeOrdersRead, ScopeOrdersWrite, ScopeCatalogRead, ScopeInventoryRead,
			ScopeWebhooksManage, ScopeExportsRead, ScopeDemandWrite,
		}
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
