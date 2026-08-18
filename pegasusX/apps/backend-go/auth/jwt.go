// JWT HS256 sign/verify helpers used by the supplier portal session cookie.
//
// Scaffold: stdlib hmac/sha256 + base64url. Production may swap for a
// vetted library — the Issue/Parse contract stays the same.
package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
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
	JTI          string `json:"jti,omitempty"`
	Role         string `json:"role"`
	SupplierID   string `json:"supplier_id"`
	SupplierRole string `json:"supplier_role,omitempty"`
	HomeNodeType string `json:"home_node_type,omitempty"`
	HomeNodeID   string `json:"home_node_id,omitempty"`
	IsRegistered bool   `json:"is_registered"`
	IsConfigured bool   `json:"is_configured"`
	PhoneNumber  string `json:"phone_number,omitempty"`
	TokenUse     string `json:"token_use,omitempty"`

	// Retail OS multi-user (optional; additive for backward compatibility).
	RetailerOrgID    string   `json:"retailer_org_id,omitempty"`
	RetailerRole     string   `json:"retailer_role,omitempty"`
	RetailerUserID   string   `json:"retailer_user_id,omitempty"`
	LocationIDs      []string `json:"location_ids,omitempty"`
	ActiveLocationID string   `json:"active_location_id,omitempty"`
	CapabilityPacks  []string `json:"capability_packs,omitempty"`
	MFAVerified      bool     `json:"mfa_verified,omitempty"`
	MarketCode       string   `json:"market_code,omitempty"`
	HomeCell         string   `json:"home_cell,omitempty"`
}

// Issue returns a signed HS256 JWT for the given claims.
// GS-A1: every token carries market_code + home_cell (claim → profile → env).
func Issue(c Claims, opts IssueOptions) (string, error) {
	c = StampMarketClaims(c)
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
	jti := strings.TrimSpace(c.JTI)
	if jti == "" {
		jti = uuid.NewString()
	}
	h, _ := json.Marshal(jwtHeader{Alg: "HS256", Typ: "JWT"})
	p, _ := json.Marshal(jwtPayload{
		Sub:              c.Subject,
		Iss:              opts.Issuer,
		Iat:              now.Unix(),
		Exp:              now.Add(opts.TTL).Unix(),
		JTI:              jti,
		Role:             string(c.Role),
		SupplierID:       c.SupplierID,
		SupplierRole:     string(c.SupplierRole),
		HomeNodeType:     string(c.HomeNodeType),
		HomeNodeID:       c.HomeNodeID,
		IsRegistered:     c.IsRegistered,
		IsConfigured:     c.IsConfigured,
		PhoneNumber:      c.PhoneNumber,
		TokenUse:         c.TokenUse,
		RetailerOrgID:    c.RetailerOrgID,
		RetailerRole:     c.RetailerRole,
		RetailerUserID:   c.RetailerUserID,
		LocationIDs:      c.LocationIDs,
		ActiveLocationID: c.ActiveLocationID,
		CapabilityPacks:  c.CapabilityPacks,
		MFAVerified:      c.MFAVerified,
		MarketCode:       c.MarketCode,
		HomeCell:         c.HomeCell,
	})
	head := b64(h) + "." + b64(p)
	sig := sign(head, opts.Secret)
	return head + "." + sig, nil
}

// IssueWSTicket mints a short-lived WebSocket upgrade ticket (token_use=ws).
// Copies identity from the session claims and forces a fresh jti so logout of
// the parent session does not denylist this ticket by accident (ticket TTL is short).
func IssueWSTicket(session Claims, opts IssueOptions) (token string, expiresAt time.Time, err error) {
	if opts.TTL <= 0 {
		opts.TTL = 10 * time.Minute
	}
	if opts.Now == nil {
		opts.Now = func() time.Time { return time.Now().UTC() }
	}
	now := opts.Now()
	if IsWSTicket(session) || IsPendingOrgSelect(session) {
		return "", time.Time{}, errors.New("jwt: cannot mint websocket ticket from restricted token")
	}
	ticket := session
	ticket.TokenUse = TokenUseWS
	ticket.JTI = ""
	token, err = Issue(ticket, opts)
	if err != nil {
		return "", time.Time{}, err
	}
	return token, now.Add(opts.TTL), nil
}

// ParseBearerClaims extracts and validates Authorization: Bearer (incl. revocation).
func ParseBearerClaims(r *http.Request, secret string) (Claims, bool) {
	token := BearerToken(r)
	if token == "" {
		return Claims{}, false
	}
	claims, err := Parse(token, secret)
	if err != nil || tokenRevoked(r.Context(), claims) || IsWSTicket(claims) {
		return Claims{}, false
	}
	return claims, true
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
	var expAt time.Time
	if p.Exp > 0 {
		expAt = time.Unix(p.Exp, 0).UTC()
	}
	claims := Claims{
		Subject:          p.Sub,
		Role:             Role(p.Role),
		SupplierID:       p.SupplierID,
		SupplierRole:     Role(p.SupplierRole),
		HomeNodeType:     HomeNodeType(p.HomeNodeType),
		HomeNodeID:       p.HomeNodeID,
		IsRegistered:     p.IsRegistered,
		IsConfigured:     p.IsConfigured,
		PhoneNumber:      p.PhoneNumber,
		TokenUse:         p.TokenUse,
		JTI:              p.JTI,
		ExpiresAt:        expAt,
		RetailerOrgID:    p.RetailerOrgID,
		RetailerRole:     p.RetailerRole,
		RetailerUserID:   p.RetailerUserID,
		LocationIDs:      p.LocationIDs,
		ActiveLocationID: p.ActiveLocationID,
		CapabilityPacks:  p.CapabilityPacks,
		MFAVerified:      p.MFAVerified,
		MarketCode:       p.MarketCode,
		HomeCell:         p.HomeCell,
	}
	if err := rejectForeignCell(claims); err != nil {
		return Claims{}, err
	}
	return claims, nil
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
		if claims, err := Parse(c.Value, secret); err == nil && !tokenRevoked(r.Context(), claims) && !IsWSTicket(claims) {
			return r.WithContext(WithClaims(r.Context(), claims))
		}
	}
	if token := BearerToken(r); token != "" {
		if claims, err := Parse(token, secret); err == nil && !tokenRevoked(r.Context(), claims) && !IsWSTicket(claims) {
			return r.WithContext(WithClaims(r.Context(), claims))
		}
	}
	return r
}

func tokenRevoked(ctx context.Context, claims Claims) bool {
	revoked, err := checkTokenRevoked(ctx, claims)
	// Store errors fail closed so a Redis blip cannot revive a denylisted jti.
	return err != nil || revoked
}

func checkTokenRevoked(ctx context.Context, claims Claims) (revoked bool, err error) {
	jti := strings.TrimSpace(claims.JTI)
	if jti == "" {
		// Legacy tokens without jti cannot be denylisted; still accept until they expire.
		return false, nil
	}
	return GetRevocationStore().IsRevoked(ctx, jti)
}

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func sign(input, secret string) string {
	m := hmac.New(sha256.New, []byte(secret))
	_, _ = m.Write([]byte(input))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}
