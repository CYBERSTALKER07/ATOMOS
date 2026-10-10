package orgoidc

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotConfigured), errors.Is(err, ErrDisabled):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
	case errors.Is(err, ErrInvalidIssuer), errors.Is(err, ErrSupplierRequired),
		errors.Is(err, ErrRedirectMismatch):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	case errors.Is(err, ErrInvalidToken), errors.Is(err, ErrIssuerMismatch),
		errors.Is(err, ErrAudienceMismatch), errors.Is(err, ErrNonceMismatch),
		errors.Is(err, ErrMissingSubject):
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
	default:
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
	}
}

// HandleDiscovery GET /v1/auth/oidc/discovery?supplier_id=
func (s *Service) HandleDiscovery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	sid := strings.TrimSpace(r.URL.Query().Get("supplier_id"))
	body, err := s.Discovery(r.Context(), sid, r.URL.Query().Get("nonce"), r.URL.Query().Get("state"), r.URL.Query().Get("redirect_uri"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, body)
}

type exchangeReq struct {
	SupplierID string `json:"supplier_id"`
	IDToken    string `json:"id_token"`
	Nonce      string `json:"nonce"`
}

// HandleExchange POST /v1/auth/oidc/exchange
func (s *Service) HandleExchange(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	var req exchangeReq
	if err := json.NewDecoder(io.LimitReader(r.Body, 32*1024)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	token, refresh, err := s.Exchange(r.Context(), req.SupplierID, req.IDToken, req.Nonce)
	if err != nil {
		writeErr(w, err)
		return
	}
	auth.SetSessionCookie(w, token, s.ttl(), s.CookieSecure)
	writeJSON(w, http.StatusOK, map[string]any{
		"token":         token,
		"refresh_token": refresh,
		"token_type":    "bearer",
		"role":          string(auth.RoleAdmin),
		"supplier_id":   strings.TrimSpace(req.SupplierID),
	})
}

// HandleGet GET /v1/supplier/oidc
func (s *Service) HandleGet(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || strings.TrimSpace(claims.SupplierID) == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	c, err := s.Get(r.Context(), claims.SupplierID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

type attachReq struct {
	Issuer                string `json:"issuer"`
	ClientID              string `json:"client_id"`
	Audience              string `json:"audience"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	RedirectURI           string `json:"redirect_uri"`
}

// HandlePut PUT /v1/supplier/oidc
func (s *Service) HandlePut(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || strings.TrimSpace(claims.SupplierID) == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var req attachReq
	if err := json.NewDecoder(io.LimitReader(r.Body, 16*1024)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	c, err := s.Attach(r.Context(), Config{
		SupplierID:            claims.SupplierID,
		Issuer:                req.Issuer,
		ClientID:              req.ClientID,
		Audience:              req.Audience,
		AuthorizationEndpoint: req.AuthorizationEndpoint,
		RedirectURI:           req.RedirectURI,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// HandleDelete DELETE /v1/supplier/oidc
func (s *Service) HandleDelete(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || strings.TrimSpace(claims.SupplierID) == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if err := s.Detach(r.Context(), claims.SupplierID); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"detached": true})
}
