package partner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// TokenResponse is an RFC 6749-style access token response.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	Scope       string `json:"scope,omitempty"`
}

// IssueClientCredentials validates the partner API key as an OAuth client and issues a JWT.
func (s *Service) IssueClientCredentials(ctx context.Context, clientID, clientSecret, scopeCSV string) (TokenResponse, error) {
	if s == nil || s.keys == nil {
		return TokenResponse{}, fmt.Errorf("oauth_unavailable")
	}
	secret := strings.TrimSpace(s.oauthJWTSecret)
	if secret == "" {
		return TokenResponse{}, fmt.Errorf("partner_jwt_secret_unavailable")
	}
	k, err := s.resolveOAuthClient(ctx, clientID, clientSecret)
	if err != nil {
		return TokenResponse{}, err
	}
	if k.Status != KeyStatusActive {
		return TokenResponse{}, fmt.Errorf("unauthorized_client")
	}
	if k.ExpiresAt != nil && time.Now().UTC().After(*k.ExpiresAt) {
		return TokenResponse{}, fmt.Errorf("unauthorized_client")
	}
	requested := splitScopes(strings.ReplaceAll(scopeCSV, ",", " "))
	granted, err := IntersectScopes(k.Scopes, requested)
	if err != nil {
		return TokenResponse{}, err
	}
	ttl := s.oauthTTL
	if ttl <= 0 {
		ttl = partnerOAuthTTL()
	}
	token, expiresIn, err := IssuePartnerAccessToken(secret, PartnerAccessClaims{
		Subject:    k.KeyID,
		KeyID:      k.KeyID,
		TenantType: k.TenantType,
		TenantID:   k.TenantID,
		Scopes:     granted,
		Issuer:     firstNonEmpty(s.oauthJWTIssuer, partnerOAuthIssuer()),
	}, ttl)
	if err != nil {
		return TokenResponse{}, err
	}
	_ = s.keys.TouchLastUsed(ctx, k.KeyID)
	return TokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
		Scope:       strings.Join(granted, " "),
	}, nil
}

func (s *Service) resolveOAuthClient(ctx context.Context, clientID, clientSecret string) (ApiKey, error) {
	clientID = strings.TrimSpace(clientID)
	clientSecret = strings.TrimSpace(clientSecret)
	if clientSecret == "" {
		return ApiKey{}, fmt.Errorf("invalid_client")
	}
	var k ApiKey
	var found bool
	var err error
	if clientID != "" {
		k, found, err = s.keys.GetByID(ctx, clientID)
		if err != nil {
			return ApiKey{}, err
		}
		if !found {
			k, found, err = s.keys.GetByPrefix(ctx, clientID)
			if err != nil {
				return ApiKey{}, err
			}
		}
	}
	if !found {
		if px, ok := ParseBearerKey(clientSecret); ok {
			k, found, err = s.keys.GetByPrefix(ctx, px)
			if err != nil {
				return ApiKey{}, err
			}
		}
	}
	if !found {
		return ApiKey{}, fmt.Errorf("invalid_client")
	}
	if !VerifyAPIKey(clientSecret, k.KeyHash) {
		return ApiKey{}, fmt.Errorf("invalid_client")
	}
	return k, nil
}

// HandleOAuthToken POST /partner/v1/oauth/token (no RequirePartner).
func (h *Handlers) HandleOAuthToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writePartnerError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	grant, clientID, clientSecret, scope, err := parseOAuthTokenRequest(r)
	if err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if grant != "client_credentials" {
		writeOAuthError(w, http.StatusBadRequest, "unsupported_grant_type", "only client_credentials is supported")
		return
	}
	tok, err := h.Svc.IssueClientCredentials(r.Context(), clientID, clientSecret, scope)
	if err != nil {
		switch err.Error() {
		case "invalid_client", "unauthorized_client":
			writeOAuthError(w, http.StatusUnauthorized, err.Error(), "client authentication failed")
		case "invalid_scope":
			writeOAuthError(w, http.StatusBadRequest, "invalid_scope", "requested scope exceeds client grant")
		case "partner_jwt_secret_unavailable", "oauth_unavailable":
			writeOAuthError(w, http.StatusServiceUnavailable, "temporarily_unavailable", err.Error())
		default:
			writeOAuthError(w, http.StatusBadRequest, "invalid_request", err.Error())
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(tok)
}

func parseOAuthTokenRequest(r *http.Request) (grant, clientID, clientSecret, scope string, err error) {
	ct := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	body, readErr := io.ReadAll(io.LimitReader(r.Body, maxBody))
	if readErr != nil {
		return "", "", "", "", fmt.Errorf("read_body_error")
	}
	if strings.Contains(ct, "application/json") {
		var req struct {
			GrantType    string `json:"grant_type"`
			ClientID     string `json:"client_id"`
			ClientSecret string `json:"client_secret"`
			Scope        string `json:"scope"`
		}
		if len(body) > 0 {
			if uErr := json.Unmarshal(body, &req); uErr != nil {
				return "", "", "", "", fmt.Errorf("invalid_json")
			}
		}
		return strings.TrimSpace(req.GrantType), strings.TrimSpace(req.ClientID),
			strings.TrimSpace(req.ClientSecret), strings.TrimSpace(req.Scope), nil
	}
	// Default: form-urlencoded (RFC 6749)
	vals, pErr := url.ParseQuery(string(body))
	if pErr != nil {
		return "", "", "", "", fmt.Errorf("invalid_form")
	}
	// Also accept Basic auth for client_id:client_secret
	clientID = strings.TrimSpace(vals.Get("client_id"))
	clientSecret = strings.TrimSpace(vals.Get("client_secret"))
	if u, p, ok := r.BasicAuth(); ok {
		if clientID == "" {
			clientID = u
		}
		if clientSecret == "" {
			clientSecret = p
		}
	}
	return strings.TrimSpace(vals.Get("grant_type")), clientID, clientSecret, strings.TrimSpace(vals.Get("scope")), nil
}

func writeOAuthError(w http.ResponseWriter, status int, code, desc string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":             code,
		"error_description": desc,
	})
}
