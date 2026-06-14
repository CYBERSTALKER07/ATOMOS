// JWT HS256 sign/verify helpers used by the supplier portal session cookie.
//
// Scaffold: stdlib hmac/sha256 + base64url. Production may swap for a
// vetted library — the Issue/Parse contract stays the same.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// CookieName is the canonical session cookie. The supplier-portal middleware
// reads this exact name; do not rename without updating both sides.
const CookieName = "supplier_jwt"

// ErrInvalidToken is returned when signature or structure fails verification.
var ErrInvalidToken = errors.New("invalid token")

// IssueOptions controls token shape.
type IssueOptions struct {
	Secret string
	Issuer string
	TTL    time.Duration
	Now    func() time.Time
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

type jwtPayload struct {
	Sub          string `json:"sub"`
	Iss          string `json:"iss,omitempty"`
	Exp          int64  `json:"exp"`
	Iat          int64  `json:"iat"`
	Role         string `json:"role"`
	SupplierID   string `json:"supplier_id"`
	SupplierRole string `json:"supplier_role,omitempty"`
	HomeNodeType string `json:"home_node_type,omitempty"`
	HomeNodeID   string `json:"home_node_id,omitempty"`
	IsRegistered bool   `json:"is_registered"`
	IsConfigured bool   `json:"is_configured"`
}

// Issue returns a signed HS256 JWT for the given claims.
func Issue(c Claims, opts IssueOptions) (string, error) {
	if opts.Secret == "" {
		return "", errors.New("jwt: empty secret")
	}
	if opts.TTL <= 0 {
		opts.TTL = 24 * time.Hour
	}
	if opts.Now == nil {
		opts.Now = func() time.Time { return time.Now().UTC() }
	}
	now := opts.Now()
	h, _ := json.Marshal(jwtHeader{Alg: "HS256", Typ: "JWT"})
	p, _ := json.Marshal(jwtPayload{
		Sub:          c.Subject,
		Iss:          opts.Issuer,
		Iat:          now.Unix(),
		Exp:          now.Add(opts.TTL).Unix(),
		Role:         string(c.Role),
		SupplierID:   c.SupplierID,
		SupplierRole: string(c.SupplierRole),
		HomeNodeType: string(c.HomeNodeType),
		HomeNodeID:   c.HomeNodeID,
		IsRegistered: c.IsRegistered,
		IsConfigured: c.IsConfigured,
	})
	head := b64(h) + "." + b64(p)
	sig := sign(head, opts.Secret)
	return head + "." + sig, nil
}

// Parse verifies signature + exp and returns the embedded Claims.
func Parse(token, secret string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrInvalidToken
	}
	expected := sign(parts[0]+"."+parts[1], secret)
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return Claims{}, ErrInvalidToken
	}
	pb, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, fmt.Errorf("jwt: payload decode: %w", err)
	}
	var p jwtPayload
	if err := json.Unmarshal(pb, &p); err != nil {
		return Claims{}, fmt.Errorf("jwt: payload json: %w", err)
	}
	if p.Exp > 0 && time.Now().UTC().Unix() > p.Exp {
		return Claims{}, fmt.Errorf("jwt: %w (expired)", ErrInvalidToken)
	}
	return Claims{
		Subject:      p.Sub,
		Role:         Role(p.Role),
		SupplierID:   p.SupplierID,
		SupplierRole: Role(p.SupplierRole),
		HomeNodeType: HomeNodeType(p.HomeNodeType),
		HomeNodeID:   p.HomeNodeID,
		IsRegistered: p.IsRegistered,
		IsConfigured: p.IsConfigured,
	}, nil
}

// SetSessionCookie writes the supplier portal cookie. SameSite=Lax + HttpOnly.
// Secure is left to caller — production should toggle when behind HTTPS.
func SetSessionCookie(w http.ResponseWriter, token string, ttl time.Duration, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// CookieAuth is a permissive middleware: it attaches Claims to the request
// context when a valid session cookie is present, and passes through silently
// otherwise. RequireRole performs the actual gating.
func CookieAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = attachSessionClaims(r, secret)
			next.ServeHTTP(w, r)
		})
	}
}

// SessionAuth attaches Claims from the supplier session cookie or a Bearer JWT.
// Used for local SSMR smoke and native clients that send Authorization headers.
func SessionAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = attachSessionClaims(r, secret)
			next.ServeHTTP(w, r)
		})
	}
}

func attachSessionClaims(r *http.Request, secret string) *http.Request {
	if c, err := r.Cookie(CookieName); err == nil && c.Value != "" {
		if claims, err := Parse(c.Value, secret); err == nil {
			return r.WithContext(WithClaims(r.Context(), claims))
		}
	}
	if token := BearerToken(r); token != "" {
		if claims, err := Parse(token, secret); err == nil {
			return r.WithContext(WithClaims(r.Context(), claims))
		}
	}
	return r
}

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func sign(input, secret string) string {
	m := hmac.New(sha256.New, []byte(secret))
	_, _ = m.Write([]byte(input))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}
