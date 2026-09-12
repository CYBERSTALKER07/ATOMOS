// Package auth provides Firebase ID token verification for login/register
// `id_token` bodies. HTTP/WS session is pegasus JWT (SessionAuth), not Firebase.
package auth

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultFirebaseCertsURL is Google's x509 endpoint for Firebase SecureToken.
	DefaultFirebaseCertsURL = "https://www.googleapis.com/robot/v1/metadata/x509/securetoken@system.gserviceaccount.com"
)

var (
	// ErrFirebaseTokenInvalid signals signature or structural JWT validation failure.
	ErrFirebaseTokenInvalid = errors.New("firebase token invalid")
)

// FirebaseLoginUnavailable is the JSON error when the client sent id_token but
// no verifier was wired (flag off or empty FIREBASE_PROJECT_ID).
const FirebaseLoginUnavailable = "firebase_login_unavailable"

// FirebaseVerifier verifies a Firebase ID token from login/register JSON `id_token`.
// It is not used as HTTP Authorization.
type FirebaseVerifier interface {
	VerifyIDToken(ctx context.Context, token string) (Claims, error)
}

// FirebaseTokenVerifierOptions controls verifier behavior.
type FirebaseTokenVerifierOptions struct {
	CertsURL   string
	HTTPClient *http.Client
	Now        func() time.Time
	ClockSkew  time.Duration
	CacheTTL   time.Duration
}

// FirebaseTokenVerifier validates Firebase ID tokens using cached Google certs.
type FirebaseTokenVerifier struct {
	projectID string
	issuer    string
	certsURL  string

	httpClient *http.Client
	now        func() time.Time
	clockSkew  time.Duration
	cacheTTL   time.Duration

	mu            sync.RWMutex
	certs         map[string]*rsa.PublicKey
	certsExpireAt time.Time
}

// NewFirebaseTokenVerifier constructs a verifier for a Firebase project.
func NewFirebaseTokenVerifier(projectID string, opts FirebaseTokenVerifierOptions) *FirebaseTokenVerifier {
	if opts.CertsURL == "" {
		opts.CertsURL = DefaultFirebaseCertsURL
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = &http.Client{Timeout: 5 * time.Second}
	}
	if opts.Now == nil {
		opts.Now = func() time.Time { return time.Now().UTC() }
	}
	if opts.ClockSkew <= 0 {
		opts.ClockSkew = 30 * time.Second
	}
	if opts.CacheTTL <= 0 {
		opts.CacheTTL = 1 * time.Hour
	}
	return &FirebaseTokenVerifier{
		projectID:  projectID,
		issuer:     "https://securetoken.google.com/" + projectID,
		certsURL:   opts.CertsURL,
		httpClient: opts.HTTPClient,
		now:        opts.Now,
		clockSkew:  opts.ClockSkew,
		cacheTTL:   opts.CacheTTL,
		certs:      make(map[string]*rsa.PublicKey),
	}
}

// VerifyIDToken verifies signature and standard Firebase claims, then maps
// custom claims into local auth.Claims.
func (v *FirebaseTokenVerifier) VerifyIDToken(ctx context.Context, token string) (Claims, error) {
	if v == nil || v.projectID == "" {
		return Claims{}, fmt.Errorf("firebase verifier disabled")
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrFirebaseTokenInvalid
	}

	hb, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Claims{}, fmt.Errorf("firebase header decode: %w", err)
	}
	var header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
		Kid string `json:"kid"`
	}
	if err := json.Unmarshal(hb, &header); err != nil {
		return Claims{}, fmt.Errorf("firebase header json: %w", err)
	}
	if header.Alg != "RS256" || header.Kid == "" {
		return Claims{}, ErrFirebaseTokenInvalid
	}

	pb, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, fmt.Errorf("firebase payload decode: %w", err)
	}
	var std struct {
		Sub string `json:"sub"`
		Aud string `json:"aud"`
		Iss string `json:"iss"`
		Exp int64  `json:"exp"`
		Iat int64  `json:"iat"`
	}
	if err := json.Unmarshal(pb, &std); err != nil {
		return Claims{}, fmt.Errorf("firebase payload json: %w", err)
	}

	if std.Aud != v.projectID || std.Iss != v.issuer || std.Sub == "" {
		return Claims{}, ErrFirebaseTokenInvalid
	}
	if len(std.Sub) > 128 {
		return Claims{}, ErrFirebaseTokenInvalid
	}

	now := v.now()
	if std.Exp == 0 || now.After(time.Unix(std.Exp, 0).Add(v.clockSkew)) {
		return Claims{}, ErrFirebaseTokenInvalid
	}
	if std.Iat > 0 && now.Before(time.Unix(std.Iat, 0).Add(-v.clockSkew)) {
		return Claims{}, ErrFirebaseTokenInvalid
	}

	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return Claims{}, fmt.Errorf("firebase signature decode: %w", err)
	}
	key, err := v.publicKeyForKID(ctx, header.Kid)
	if err != nil {
		return Claims{}, err
	}
	signed := parts[0] + "." + parts[1]
	digest := sha256.Sum256([]byte(signed))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], sig); err != nil {
		return Claims{}, ErrFirebaseTokenInvalid
	}

	var custom map[string]any
	if err := json.Unmarshal(pb, &custom); err != nil {
		return Claims{}, fmt.Errorf("firebase custom claims: %w", err)
	}

	subject := stringClaim(custom, "retailer_id")
	if subject == "" {
		subject = std.Sub
	}

	claims := Claims{
		Subject:      subject,
		Role:         Role(strings.ToUpper(stringClaim(custom, "role"))),
		SupplierID:   stringClaim(custom, "supplier_id"),
		SupplierRole: Role(strings.ToUpper(stringClaim(custom, "supplier_role"))),
		HomeNodeType: HomeNodeType(strings.ToUpper(stringClaim(custom, "home_node_type"))),
		HomeNodeID:   stringClaim(custom, "home_node_id"),
		IsConfigured: boolClaim(custom, "is_configured"),
		PhoneNumber:  stringClaim(custom, "phone_number"),
	}
	return claims, nil
}

// FirebaseAuth does not attach session claims. HTTP/WS SoT is pegasus JWT
// via SessionAuth. Firebase ID tokens are accepted only as login/register
// JSON `id_token`. Kept as an explicit pass-through so a remount cannot mint
// a second session from Authorization.
func FirebaseAuth(verifier FirebaseVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = verifier
			next.ServeHTTP(w, r)
		})
	}
}

func (v *FirebaseTokenVerifier) publicKeyForKID(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	cachedKey, cachedOK := v.certs[kid]
	fresh := v.now().Before(v.certsExpireAt)
	v.mu.RUnlock()
	if cachedOK && fresh {
		return cachedKey, nil
	}

	if err := v.refreshCerts(ctx); err != nil {
		if cachedOK {
			return cachedKey, nil
		}
		return nil, fmt.Errorf("firebase cert refresh: %w", err)
	}

	v.mu.RLock()
	defer v.mu.RUnlock()
	k, ok := v.certs[kid]
	if !ok {
		return nil, ErrFirebaseTokenInvalid
	}
	return k, nil
}

func (v *FirebaseTokenVerifier) refreshCerts(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.certsURL, nil)
	if err != nil {
		return err
	}
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status=%d", resp.StatusCode)
	}

	var raw map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return err
	}
	if len(raw) == 0 {
		return errors.New("empty cert set")
	}

	parsed := make(map[string]*rsa.PublicKey, len(raw))
	for kid, certPEM := range raw {
		pub, err := parseRSAPublicKeyFromCert(certPEM)
		if err != nil {
			return fmt.Errorf("kid=%s: %w", kid, err)
		}
		parsed[kid] = pub
	}

	expireAt := v.now().Add(v.cacheTTL)
	if maxAge, ok := maxAgeFromCacheControl(resp.Header.Get("Cache-Control")); ok {
		expireAt = v.now().Add(maxAge)
	}

	v.mu.Lock()
	v.certs = parsed
	v.certsExpireAt = expireAt
	v.mu.Unlock()
	return nil
}

func parseRSAPublicKeyFromCert(certPEM string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return nil, errors.New("invalid pem")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	pub, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("cert key is not rsa")
	}
	return pub, nil
}

func maxAgeFromCacheControl(cacheControl string) (time.Duration, bool) {
	for _, part := range strings.Split(cacheControl, ",") {
		part = strings.TrimSpace(part)
		if !strings.HasPrefix(part, "max-age=") {
			continue
		}
		seconds, err := strconv.Atoi(strings.TrimPrefix(part, "max-age="))
		if err != nil || seconds <= 0 {
			return 0, false
		}
		return time.Duration(seconds) * time.Second, true
	}
	return 0, false
}

func stringClaim(claims map[string]any, key string) string {
	v, ok := claims[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func boolClaim(claims map[string]any, key string) bool {
	v, ok := claims[key]
	if !ok {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return strings.EqualFold(strings.TrimSpace(t), "true")
	default:
		return false
	}
}
