package retailer

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// HandleRetailerLogin authenticates retailers for native and desktop clients.
// POST /v1/auth/retailer/login  body: { "phone_number", "password" }
func (s *Service) HandleRetailerLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	defer r.Body.Close()

	var req struct {
		PhoneNumber string `json:"phone_number"`
		Phone       string `json:"phone"`
		Password    string `json:"password"`
		PIN         string `json:"pin"`
		IDToken     string `json:"id_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	idToken := strings.TrimSpace(req.IDToken)

	var ret Retailer
	var ok bool

	if idToken != "" && s.firebaseVerifier != nil {
		claims, err := s.firebaseVerifier.VerifyIDToken(r.Context(), idToken)
		if err != nil {
			s.log.Warn("firebase token verification failed", "err", err)
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_id_token"})
			return
		}
		if claims.PhoneNumber == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "phone_number_missing_in_token"})
			return
		}
		ret, ok, err = s.resolveFirebaseLogin(r.Context(), claims.PhoneNumber)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "login_failed"})
			return
		}
		if !ok {
			// Unregistered phone numbers are BLOCKED pending admin approval.
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "user_not_found"})
			return
		}
	} else {
		// Fallback to demo login if IDToken is not provided.
		phone := strings.TrimSpace(req.PhoneNumber)
		if phone == "" {
			phone = strings.TrimSpace(req.Phone)
		}
		secret := strings.TrimSpace(req.Password)
		if secret == "" {
			secret = strings.TrimSpace(req.PIN)
		}
		if phone == "" || secret == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "phone_and_password_required"})
			return
		}

		var err error
		ret, ok, err = s.resolveRetailerLogin(r.Context(), phone, secret)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "login_failed"})
			return
		}
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "user_not_found"})
			return
		}
	}

	if s.jwtSecret == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "jwt_not_configured"})
		return
	}

	s.writeMobileAuthResponse(w, ret)
}

// HandleRetailerRefresh re-issues tokens from a refresh JWT.
// POST /v1/auth/retailer/refresh
func (s *Service) HandleRetailerRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	defer r.Body.Close()
	if s.jwtSecret == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "jwt_not_configured"})
		return
	}

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	refresh := strings.TrimSpace(req.RefreshToken)
	if refresh == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "refresh_token_required"})
		return
	}

	claims, err := auth.Parse(refresh, s.jwtSecret)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_refresh_token"})
		return
	}
	if claims.Role != auth.RoleRetailer {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "invalid_refresh_scope"})
		return
	}

	token, err := auth.Issue(claims, auth.IssueOptions{Secret: s.jwtSecret, Issuer: s.jwtIssuer, TTL: 24 * time.Hour})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "issue_token_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"token":         token,
		"refresh_token": refresh,
	})
}

func (s *Service) resolveRetailerLogin(ctx context.Context, phone, secret string) (Retailer, bool, error) {
	expectPhone := strings.TrimSpace(os.Getenv("RETAILER_DEMO_PHONE"))
	if expectPhone == "" {
		expectPhone = "+998901000077"
	}
	expectSecret := strings.TrimSpace(os.Getenv("RETAILER_DEMO_PASSWORD"))
	if expectSecret == "" {
		expectSecret = strings.TrimSpace(os.Getenv("RETAILER_DEMO_PIN"))
	}
	if expectSecret == "" {
		expectSecret = "1234"
	}

	if phone == expectPhone && secret == expectSecret {
		if ret, found, err := s.repo.FindByPhone(ctx, phone); err != nil {
			return Retailer{}, false, err
		} else if found {
			return ret, true, nil
		}
		reg, err := s.Register(ctx, RegisterRequest{
			Phone: phone,
			Name:  demoRetailerStoreName(),
			Lat:   41.311081,
			Lng:   69.240562,
		})
		if err != nil {
			return Retailer{}, false, err
		}
		return Retailer{
			RetailerID: reg.RetailerID,
			Phone:      reg.Phone,
			Name:       demoRetailerStoreName(),
			SupplierID: s.supplierID,
		}, true, nil
	}

	if ret, found, err := s.repo.FindByPhone(ctx, phone); err != nil {
		return Retailer{}, false, err
	} else if found && secret == expectSecret {
		return ret, true, nil
	}
	return Retailer{}, false, nil
}

func (s *Service) resolveFirebaseLogin(ctx context.Context, phone string) (Retailer, bool, error) {
	if ret, found, err := s.repo.FindByPhone(ctx, phone); err != nil {
		return Retailer{}, false, err
	} else if found {
		return ret, true, nil
	}
	return Retailer{}, false, nil
}

func demoRetailerStoreName() string {
	if name := strings.TrimSpace(os.Getenv("RETAILER_DEMO_STORE")); name != "" {
		return name
	}
	return "PegasusX Demo Store"
}

func retailerProfileConfigured(ret Retailer) bool {
	return strings.TrimSpace(ret.Name) != "" && (ret.Lat != 0 || ret.Lng != 0)
}

func (s *Service) marshalMobileAuthResponse(ret Retailer) (int, []byte) {
	isConfigured := retailerProfileConfigured(ret)
	claims := auth.Claims{
		Subject:      ret.RetailerID,
		Role:         auth.RoleRetailer,
		SupplierID:   ret.SupplierID,
		IsConfigured: isConfigured,
	}
	if claims.SupplierID == "" {
		claims.SupplierID = s.supplierID
	}
	token, err := auth.Issue(claims, auth.IssueOptions{Secret: s.jwtSecret, Issuer: s.jwtIssuer, TTL: 24 * time.Hour})
	if err != nil {
		b, _ := json.Marshal(map[string]string{"error": "issue_token_failed"})
		return http.StatusInternalServerError, b
	}
	refresh, err := auth.Issue(claims, auth.IssueOptions{Secret: s.jwtSecret, Issuer: s.jwtIssuer, TTL: 7 * 24 * time.Hour})
	if err != nil {
		b, _ := json.Marshal(map[string]string{"error": "issue_refresh_failed"})
		return http.StatusInternalServerError, b
	}
	respMap := map[string]any{
		"token":         token,
		"refresh_token": refresh,
		"is_configured": isConfigured,
		"user": map[string]any{
			"id":         ret.RetailerID,
			"name":       coalesceRetailerName(ret.Name),
			"company":    coalesceRetailerName(ret.Name),
			"email":      "",
			"avatar_url": nil,
		},
	}
	if fbToken, err := auth.MintCustomToken(context.Background(), ret.RetailerID, map[string]interface{}{
		"role":        string(auth.RoleRetailer),
		"retailer_id": ret.RetailerID,
		"supplier_id": claims.SupplierID,
	}); err == nil && fbToken != "" {
		respMap["firebase_token"] = fbToken
		respMap["firebaseToken"] = fbToken
	}
	b, err := json.Marshal(respMap)
	if err != nil {
		b, _ = json.Marshal(map[string]string{"error": "marshal_failed"})
		return http.StatusInternalServerError, b
	}
	return http.StatusOK, b
}

func (s *Service) writeMobileAuthResponse(w http.ResponseWriter, ret Retailer) {
	status, body := s.marshalMobileAuthResponse(ret)
	writeJSONBytes(w, status, body)
}

func coalesceRetailerName(name string) string {
	if strings.TrimSpace(name) != "" {
		return strings.TrimSpace(name)
	}
	return demoRetailerStoreName()
}
