package orgoidc

import (
	"context"
	"crypto/rsa"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// KeyFunc returns the IdP public key for an issuer (tests inject a static key).
type KeyFunc func(ctx context.Context, issuer string) (*rsa.PublicKey, error)

// Service attaches IdP config and exchanges an id_token for the cell HS256 JWT.
type Service struct {
	Store        Store
	Keys         KeyFunc
	JWTSecret    string
	JWTIssuer    string
	JWTTTL       time.Duration
	CookieSecure bool
	Now          func() time.Time
}

func (s *Service) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now()
	}
	return time.Now().UTC()
}

func (s *Service) ttl() time.Duration {
	if s != nil && s.JWTTTL > 0 {
		return s.JWTTTL
	}
	return 24 * time.Hour
}

// Attach upserts IdP settings for the caller's supplier. Secret is never stored.
func (s *Service) Attach(ctx context.Context, c Config) (Config, error) {
	c, err := normalizeConfig(c)
	if err != nil {
		return Config{}, err
	}
	c.Enabled = true
	if err := s.Store.Put(ctx, c); err != nil {
		return Config{}, err
	}
	return c, nil
}

// Get returns the attached row (may be disabled).
func (s *Service) Get(ctx context.Context, supplierID string) (Config, error) {
	c, ok, err := s.Store.Get(ctx, supplierID)
	if err != nil {
		return Config{}, err
	}
	if !ok {
		return Config{}, ErrNotConfigured
	}
	return c, nil
}

// Detach removes the IdP. Password login is unchanged.
func (s *Service) Detach(ctx context.Context, supplierID string) error {
	return s.Store.Delete(ctx, supplierID)
}

// Discovery is the public start payload. 404 when missing or disabled.
func (s *Service) Discovery(ctx context.Context, supplierID, nonce, state, redirectURI string) (map[string]any, error) {
	c, err := s.Get(ctx, supplierID)
	if err != nil {
		return nil, err
	}
	if !c.Enabled {
		return nil, ErrDisabled
	}
	if redir := redirectURI; redir != "" && c.RedirectURI != "" && redir != c.RedirectURI {
		return nil, ErrRedirectMismatch
	}
	return map[string]any{
		"enabled":            true,
		"supplier_id":        c.SupplierID,
		"issuer":             c.Issuer,
		"client_id":          c.ClientID,
		"redirect_uri":       firstNonEmpty(redirectURI, c.RedirectURI),
		"authorization_url":  c.AuthorizationURL(nonce, state, firstNonEmpty(redirectURI, c.RedirectURI)),
		"staff_jwt_unchanged": true,
	}, nil
}

// Exchange validates the IdP id_token and issues the existing HS256 ADMIN JWT.
func (s *Service) Exchange(ctx context.Context, supplierID, idToken, nonce string) (token, refresh string, err error) {
	c, err := s.Get(ctx, supplierID)
	if err != nil {
		return "", "", err
	}
	if !c.Enabled {
		return "", "", ErrDisabled
	}
	keys := s.Keys
	if keys == nil {
		keys = FetchJWKS
	}
	key, err := keys(ctx, c.Issuer)
	if err != nil || key == nil {
		return "", "", ErrInvalidToken
	}
	sub, _, err := VerifyIDToken(idToken, c, key, s.now(), nonce)
	if err != nil {
		return "", "", err
	}
	claims := auth.Claims{
		Subject:      sub,
		Role:         auth.RoleAdmin,
		SupplierID:   c.SupplierID,
		IsRegistered: true,
		IsConfigured: true,
	}
	token, err = auth.Issue(claims, auth.IssueOptions{
		Secret: s.JWTSecret,
		Issuer: s.JWTIssuer,
		TTL:    s.ttl(),
		Now:    s.now,
	})
	if err != nil {
		return "", "", err
	}
	refresh, err = auth.Issue(claims, auth.IssueOptions{
		Secret: s.JWTSecret,
		Issuer: s.JWTIssuer,
		TTL:    7 * 24 * time.Hour,
		Now:    s.now,
	})
	if err != nil {
		return token, "", err
	}
	return token, refresh, nil
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
