// Package orgoidc is GS-I: OIDC attached to one supplier org.
// Staff/driver keep HS256. This package never wraps the process router.
package orgoidc

import (
	"errors"
	"net/url"
	"strings"
)

var (
	ErrNotConfigured     = errors.New("oidc_not_configured")
	ErrDisabled          = errors.New("oidc_disabled")
	ErrInvalidIssuer     = errors.New("oidc_invalid_issuer")
	ErrInvalidToken      = errors.New("oidc_invalid_id_token")
	ErrIssuerMismatch    = errors.New("oidc_issuer_mismatch")
	ErrAudienceMismatch  = errors.New("oidc_audience_mismatch")
	ErrNonceMismatch     = errors.New("oidc_nonce_mismatch")
	ErrMissingSubject    = errors.New("oidc_missing_subject")
	ErrRedirectMismatch  = errors.New("oidc_redirect_mismatch")
	ErrSupplierRequired  = errors.New("supplier_id_required")
)

// Config is the public IdP row for one supplier. No secrets.
type Config struct {
	SupplierID              string `json:"supplier_id"`
	Issuer                  string `json:"issuer"`
	ClientID                string `json:"client_id"`
	Audience                string `json:"audience,omitempty"`
	AuthorizationEndpoint   string `json:"authorization_endpoint,omitempty"`
	RedirectURI             string   `json:"redirect_uri,omitempty"`
	AdminEmails             []string `json:"admin_emails,omitempty"`
	Enabled                 bool     `json:"enabled"`
}

func (c Config) audience() string {
	if a := strings.TrimSpace(c.Audience); a != "" {
		return a
	}
	return strings.TrimSpace(c.ClientID)
}

func (c Config) authorizeURL() string {
	if ep := strings.TrimSpace(c.AuthorizationEndpoint); ep != "" {
		return strings.TrimRight(ep, "/")
	}
	return strings.TrimRight(strings.TrimSpace(c.Issuer), "/") + "/authorize"
}

func normalizeConfig(c Config) (Config, error) {
	c.SupplierID = strings.TrimSpace(c.SupplierID)
	c.Issuer = strings.TrimRight(strings.TrimSpace(c.Issuer), "/")
	c.ClientID = strings.TrimSpace(c.ClientID)
	c.Audience = strings.TrimSpace(c.Audience)
	c.AuthorizationEndpoint = strings.TrimSpace(c.AuthorizationEndpoint)
	c.RedirectURI = strings.TrimSpace(c.RedirectURI)
	if c.SupplierID == "" {
		return Config{}, ErrSupplierRequired
	}
	if c.ClientID == "" {
		return Config{}, errors.New("oidc_client_id_required")
	}
	if err := validateIssuer(c.Issuer); err != nil {
		return Config{}, err
	}
	return c, nil
}

func validateIssuer(issuer string) error {
	u, err := url.Parse(issuer)
	if err != nil || u.Host == "" {
		return ErrInvalidIssuer
	}
	if strings.EqualFold(u.Scheme, "https") {
		return nil
	}
	// Tests / local IdP only.
	if strings.EqualFold(u.Scheme, "http") && (u.Hostname() == "127.0.0.1" || u.Hostname() == "localhost") {
		return nil
	}
	return ErrInvalidIssuer
}

// AuthorizationURL is the browser start URL (implicit id_token). PKCE/code later.
func (c Config) AuthorizationURL(nonce, state, redirectURI string) string {
	redir := strings.TrimSpace(redirectURI)
	if redir == "" {
		redir = c.RedirectURI
	}
	q := url.Values{}
	q.Set("client_id", c.ClientID)
	q.Set("response_type", "id_token")
	q.Set("response_mode", "fragment")
	q.Set("scope", "openid email profile")
	if redir != "" {
		q.Set("redirect_uri", redir)
	}
	if strings.TrimSpace(nonce) != "" {
		q.Set("nonce", nonce)
	}
	if strings.TrimSpace(state) != "" {
		q.Set("state", state)
	}
	return c.authorizeURL() + "?" + q.Encode()
}
