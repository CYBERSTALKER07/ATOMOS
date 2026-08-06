package partner

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// TokenUsePartnerAccess discriminates partner OAuth access tokens from human session JWTs.
const TokenUsePartnerAccess = "partner_access"

const (
	defaultPartnerOAuthIssuer = "pegasusx-partner"
	defaultPartnerOAuthTTL    = 15 * time.Minute
	maxPartnerOAuthTTL        = 60 * time.Minute
)

// PartnerAccessClaims is the JWT payload for client_credentials access tokens.
type PartnerAccessClaims struct {
	Subject    string
	KeyID      string
	TenantType string
	TenantID   string
	Scopes     []string
	Issuer     string
	IssuedAt   time.Time
	ExpiresAt  time.Time
}

type partnerAccessPayload struct {
	Sub        string `json:"sub"`
	Iss        string `json:"iss,omitempty"`
	Exp        int64  `json:"exp"`
	Iat        int64  `json:"iat"`
	TokenUse   string `json:"token_use"`
	TenantType string `json:"tenant_type"`
	TenantID   string `json:"tenant_id"`
	Scope      string `json:"scope"`
	KeyID      string `json:"key_id"`
}

// ResolvePartnerJWTSecret returns PARTNER_JWT_SECRET, or a derived key from human JWT_SECRET
// so partner tokens never verify under SessionAuth (different secret material).
func ResolvePartnerJWTSecret(humanJWTSecret string) string {
	if v := strings.TrimSpace(os.Getenv("PARTNER_JWT_SECRET")); v != "" {
		return v
	}
	base := strings.TrimSpace(humanJWTSecret)
	if base == "" {
		base = strings.TrimSpace(os.Getenv("JWT_SECRET"))
	}
	if base == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(base))
	_, _ = mac.Write([]byte("pegasusx-partner-oauth-v1"))
	return hex.EncodeToString(mac.Sum(nil))
}

// PartnerOAuthIssuer returns the JWT iss claim for partner access tokens.
func PartnerOAuthIssuer() string {
	return partnerOAuthIssuer()
}

// PartnerOAuthTTL returns the access-token lifetime.
func PartnerOAuthTTL() time.Duration {
	return partnerOAuthTTL()
}

func partnerOAuthIssuer() string {
	if v := strings.TrimSpace(os.Getenv("PARTNER_JWT_ISSUER")); v != "" {
		return v
	}
	return defaultPartnerOAuthIssuer
}

func partnerOAuthTTL() time.Duration {
	if v := strings.TrimSpace(os.Getenv("PARTNER_OAUTH_TTL_SECONDS")); v != "" {
		sec, err := strconv.Atoi(v)
		if err == nil && sec > 0 {
			d := time.Duration(sec) * time.Second
			if d > maxPartnerOAuthTTL {
				return maxPartnerOAuthTTL
			}
			return d
		}
	}
	return defaultPartnerOAuthTTL
}

// IssuePartnerAccessToken signs a short-lived partner access JWT.
func IssuePartnerAccessToken(secret string, c PartnerAccessClaims, ttl time.Duration) (string, int64, error) {
	if strings.TrimSpace(secret) == "" {
		return "", 0, errors.New("partner_jwt_secret_unavailable")
	}
	if ttl <= 0 {
		ttl = defaultPartnerOAuthTTL
	}
	if ttl > maxPartnerOAuthTTL {
		ttl = maxPartnerOAuthTTL
	}
	now := time.Now().UTC()
	if c.Issuer == "" {
		c.Issuer = partnerOAuthIssuer()
	}
	keyID := strings.TrimSpace(c.KeyID)
	if keyID == "" {
		keyID = strings.TrimSpace(c.Subject)
	}
	payload, _ := json.Marshal(partnerAccessPayload{
		Sub:        keyID,
		Iss:        c.Issuer,
		Iat:        now.Unix(),
		Exp:        now.Add(ttl).Unix(),
		TokenUse:   TokenUsePartnerAccess,
		TenantType: c.TenantType,
		TenantID:   c.TenantID,
		Scope:      strings.Join(c.Scopes, " "),
		KeyID:      keyID,
	})
	header, _ := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	head := b64url(header) + "." + b64url(payload)
	sig := signHS256(head, secret)
	return head + "." + sig, int64(ttl.Seconds()), nil
}

// ParsePartnerAccessToken verifies signature, exp, and token_use=partner_access.
func ParsePartnerAccessToken(token, secret string) (PartnerAccessClaims, error) {
	if strings.TrimSpace(secret) == "" {
		return PartnerAccessClaims{}, errors.New("partner_jwt_secret_unavailable")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return PartnerAccessClaims{}, errors.New("invalid_token")
	}
	expected := signHS256(parts[0]+"."+parts[1], secret)
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return PartnerAccessClaims{}, errors.New("invalid_token")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return PartnerAccessClaims{}, errors.New("invalid_token")
	}
	var p partnerAccessPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return PartnerAccessClaims{}, errors.New("invalid_token")
	}
	if p.TokenUse != TokenUsePartnerAccess {
		return PartnerAccessClaims{}, errors.New("invalid_token_use")
	}
	if p.Exp > 0 && time.Now().UTC().Unix() > p.Exp {
		return PartnerAccessClaims{}, errors.New("token_expired")
	}
	keyID := strings.TrimSpace(p.KeyID)
	if keyID == "" {
		keyID = strings.TrimSpace(p.Sub)
	}
	return PartnerAccessClaims{
		Subject:    keyID,
		KeyID:      keyID,
		TenantType: p.TenantType,
		TenantID:   p.TenantID,
		Scopes:     splitScopes(p.Scope),
		Issuer:     p.Iss,
		IssuedAt:   time.Unix(p.Iat, 0).UTC(),
		ExpiresAt:  time.Unix(p.Exp, 0).UTC(),
	}, nil
}

func splitScopes(scope string) []string {
	fields := strings.Fields(strings.TrimSpace(scope))
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if f = strings.TrimSpace(f); f != "" {
			out = append(out, f)
		}
	}
	return out
}

// IntersectScopes returns requested scopes that the key allows (empty request → all key scopes).
func IntersectScopes(keyScopes, requested []string) ([]string, error) {
	if len(requested) == 0 {
		return append([]string(nil), keyScopes...), nil
	}
	out := make([]string, 0, len(requested))
	for _, need := range requested {
		need = strings.TrimSpace(need)
		if need == "" {
			continue
		}
		if !HasScope(keyScopes, need) {
			return nil, fmt.Errorf("invalid_scope")
		}
		out = append(out, need)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("invalid_scope")
	}
	return out, nil
}

func b64url(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func signHS256(head, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(head))
	return b64url(mac.Sum(nil))
}

func looksLikeJWT(token string) bool {
	parts := strings.Split(token, ".")
	return len(parts) == 3 && parts[0] != "" && parts[1] != "" && parts[2] != ""
}
