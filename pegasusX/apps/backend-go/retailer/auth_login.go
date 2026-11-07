package retailer

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"golang.org/x/crypto/bcrypt"
)

// HandleRetailerLogin authenticates retailers for native and desktop clients.
// POST /v1/auth/retailer/login  body: { "phone_number", "password" }
// Phase 1: staff accounts in RetailerUsers authenticate with phone+password;
// owners still bootstrap via EnsureOwnerUser.
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
	if idToken != "" && s.firebaseVerifier == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": auth.FirebaseLoginUnavailable})
		return
	}

	var ret Retailer
	var sessionUser *RetailerUser
	var ok bool

	if idToken != "" && s.firebaseVerifier != nil {
		claims, err := s.firebaseVerifier.VerifyIDToken(r.Context(), idToken)
		if err != nil {
			if s.log != nil {
				s.log.Warn("firebase token verification failed", "err", err)
			}
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_id_token"})
			return
		}
		if claims.PhoneNumber == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "phone_number_missing_in_token"})
			return
		}
		// Prefer staff/user row by phone, then shop row.
		if u, found, err := s.findRetailerUserByPhoneAny(r.Context(), claims.PhoneNumber); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "login_failed"})
			return
		} else if found {
			if !u.IsActive {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "user_inactive"})
				return
			}
			shop, shopOK, err := s.repo.GetRetailer(r.Context(), u.RetailerID)
			if err != nil || !shopOK {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "user_not_found"})
				return
			}
			ret = shop
			sessionUser = &u
			ok = true
		} else {
			ret, ok, err = s.resolveFirebaseLogin(r.Context(), claims.PhoneNumber)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "login_failed"})
				return
			}
			if !ok {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "user_not_found"})
				return
			}
		}
	} else {
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

		// Staff / owner: when multi-org flag on, resolve all matching memberships first.
		if s.multiOrgLoginEnabled() {
			if ms, matched, err := s.resolveAuthenticatedMemberships(r.Context(), phone, secret); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "login_failed"})
				return
			} else if matched != nil {
				if shouldOfferOrgPicker(ms) {
					if s.maybeWritePendingOrgSelect(w, phone, ms) {
						return
					}
				}
				// Single (or allowlist not multi): full JWT for matched user.
				u := *matched
				if s.repo != nil {
					if shop, shopOK, err := s.repo.GetRetailer(r.Context(), u.RetailerID); err == nil && shopOK {
						ret = shop
					} else {
						ret = Retailer{RetailerID: u.RetailerID, Phone: u.Phone, Name: u.Name, SupplierID: s.resolveSupplierScope(r.Context())}
					}
				} else {
					ret = Retailer{RetailerID: u.RetailerID, Phone: u.Phone, Name: u.Name, SupplierID: s.resolveSupplierScope(r.Context())}
				}
				sessionUser = &u
				ok = true
			}
		}

		// Legacy single-org path (flag off or multi path found nothing).
		if !ok {
			if u, found, err := s.findRetailerUserByPhoneAny(r.Context(), phone); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "login_failed"})
				return
			} else if found {
				if !u.IsActive {
					writeJSON(w, http.StatusForbidden, map[string]string{"error": "user_inactive"})
					return
				}
				if strings.TrimSpace(u.PasswordHash) != "" {
					if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(secret)); err != nil {
						// Fall through to demo owner path only for owners without match
						if !u.IsOwner {
							writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_credentials"})
							return
						}
					} else {
						if s.repo != nil {
							if shop, shopOK, err := s.repo.GetRetailer(r.Context(), u.RetailerID); err == nil && shopOK {
								ret = shop
							} else {
								ret = Retailer{RetailerID: u.RetailerID, Phone: u.Phone, Name: u.Name, SupplierID: s.resolveSupplierScope(r.Context())}
							}
						} else {
							ret = Retailer{RetailerID: u.RetailerID, Phone: u.Phone, Name: u.Name, SupplierID: s.resolveSupplierScope(r.Context())}
						}
						sessionUser = &u
						ok = true
					}
				}
			}
		}

		if !ok {
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
	}

	if s.jwtSecret == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "jwt_not_configured"})
		return
	}

	// C1.2 multi-org: when flag on and phone has 2+ memberships, return intermediate token.
	// Flag off / single membership → legacy full JWT path unchanged.
	if sessionUser != nil {
		if s.tryMultiOrgLoginResponse(w, r, sessionUser.Phone, sessionUser) {
			return
		}
		s.writeMobileAuthResponseForUser(w, ret, *sessionUser)
		return
	}
	if s.tryMultiOrgLoginResponse(w, r, ret.Phone, nil) {
		return
	}
	s.writeMobileAuthResponse(w, ret)
}

// tryMultiOrgLoginResponse writes pending_org_select when multi-org applies.
// passwordAlreadyVerified is the session user when password path already matched one row.
func (s *Service) tryMultiOrgLoginResponse(w http.ResponseWriter, r *http.Request, phone string, passwordMatched *RetailerUser) bool {
	phone = strings.TrimSpace(phone)
	if phone == "" || !s.multiOrgLoginEnabled() {
		return false
	}
	ms, err := s.ListMembershipsByPhone(r.Context(), phone)
	if err != nil || len(ms) <= 1 {
		// Also try dual-read from all users if memberships sparse but multi user rows exist.
		if passwordMatched != nil {
			users, uerr := s.listRetailerUsersByPhone(r.Context(), phone)
			if uerr == nil && len(users) > 1 {
				var synth []RetailerMembership
				for _, u := range users {
					if !u.IsActive {
						continue
					}
					synth = append(synth, RetailerMembership{
						UserID: u.UserID, RetailerID: u.RetailerID, RetailerRole: u.RetailerRole,
						IsActive: true, Phone: u.Phone, Name: u.Name,
					})
				}
				if shouldOfferOrgPicker(synth) {
					return s.maybeWritePendingOrgSelect(w, phone, synth)
				}
			}
		}
		return false
	}
	active := make([]RetailerMembership, 0, len(ms))
	for _, m := range ms {
		if m.IsActive {
			active = append(active, m)
		}
	}
	if !shouldOfferOrgPicker(active) {
		return false
	}
	return s.maybeWritePendingOrgSelect(w, phone, active)
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
	if auth.IsPendingOrgSelect(claims) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_refresh_token", "code": "PENDING_ORG_EXPIRED"})
		return
	}
	// Refresh capability snapshot when possible.
	if org := auth.ResolveRetailerOrgID(claims); org != "" {
		if packs, err := s.LoadEnabledPacks(r.Context(), org); err == nil {
			claims.CapabilityPacks = packs.List()
		}
		if claims.RetailerOrgID == "" {
			claims.RetailerOrgID = org
		}
		if claims.RetailerRole == "" {
			claims.RetailerRole = "OWNER"
		}
		if claims.RetailerUserID == "" {
			claims.RetailerUserID = auth.ResolveRetailerUserID(claims)
		}
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
	expectSecret := demoRetailerSecret()
	if expectSecret != "" {
		expectPhone := strings.TrimSpace(os.Getenv("RETAILER_DEMO_PHONE"))
		if expectPhone == "" {
			expectPhone = "+998901000077"
		}
		if phone == expectPhone && secret == expectSecret {
			if ret, found, err := s.repo.FindByPhone(ctx, phone); err != nil {
				return Retailer{}, false, err
			} else if found {
				return ret, true, nil
			}
			reg, err := s.Register(ctx, RegisterRequest{
				Phone:      phone,
				Name:       demoRetailerStoreName(),
				SupplierID: s.seedSupplierID,
				Lat:        41.311081,
				Lng:        69.240562,
				H3Cell:     "8928308280fffff",
			})
			if err != nil {
				return Retailer{}, false, err
			}
			return Retailer{
				RetailerID: reg.RetailerID,
				Phone:      reg.Phone,
				Name:       demoRetailerStoreName(),
				SupplierID: reg.SupplierID,
			}, true, nil
		}
		if ret, found, err := s.repo.FindByPhone(ctx, phone); err != nil {
			return Retailer{}, false, err
		} else if found && secret == expectSecret {
			return ret, true, nil
		}
		return Retailer{}, false, nil
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
	// Phase 0: bootstrap OWNER user so JWT Subject is the person id.
	owner, err := s.EnsureOwnerUser(context.Background(), ret)
	if err != nil {
		if s.log != nil {
			s.log.Warn("owner bootstrap failed; falling back to legacy subject", "retailer_id", ret.RetailerID, "err", err)
		}
		owner = RetailerUser{
			UserID:       ret.RetailerID,
			RetailerID:   ret.RetailerID,
			Phone:        ret.Phone,
			Name:         coalesceRetailerName(ret.Name),
			RetailerRole: "OWNER",
			IsOwner:      true,
			IsActive:     true,
		}
	}
	return s.marshalMobileAuthResponseForUser(ret, owner)
}

func (s *Service) marshalMobileAuthResponseForUser(ret Retailer, user RetailerUser) (int, []byte) {
	isConfigured := retailerProfileConfigured(ret)
	packList := []string{PackCORE}
	if packs, err := s.LoadEnabledPacks(context.Background(), ret.RetailerID); err == nil {
		packList = packs.List()
	}

	role := strings.ToUpper(strings.TrimSpace(user.RetailerRole))
	if role == "" {
		if user.IsOwner {
			role = "OWNER"
		} else {
			role = "VIEWER"
		}
	}

	// Phase 2: ensure primary location + attach scope/active branch to JWT.
	var locIDs []string
	var activeLoc string
	if primary, err := s.EnsurePrimaryLocation(context.Background(), ret.RetailerID); err == nil {
		activeLoc = primary.LocationID
	}
	if bound, err := s.listUserLocationIDs(context.Background(), user.UserID); err == nil && len(bound) > 0 {
		locIDs = bound
		if !containsString(bound, activeLoc) {
			activeLoc = bound[0]
		}
	}

	claims := auth.Claims{
		Subject:          user.UserID,
		Role:             auth.RoleRetailer,
		SupplierID:       ret.SupplierID,
		IsConfigured:     isConfigured,
		RetailerOrgID:    ret.RetailerID,
		RetailerRole:     role,
		RetailerUserID:   user.UserID,
		CapabilityPacks:  packList,
		PhoneNumber:      user.Phone,
		LocationIDs:      locIDs,
		ActiveLocationID: activeLoc,
	}
	if claims.SupplierID == "" {
		claims.SupplierID = s.seedSupplierID
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
	// Home surface hint for clients (role home).
	home := roleHomeSurface(role)
	respMap := map[string]any{
		"token":              token,
		"token_type":         "full",
		"refresh_token":      refresh,
		"is_configured":      isConfigured,
		"retailer_id":        ret.RetailerID,
		"retailer_org_id":    ret.RetailerID,
		"user_id":            user.UserID,
		"retailer_role":      role,
		"capabilities":       packList,
		"home_surface":       home,
		"permissions":        auth.ListRetailerPerms(claims),
		"active_location_id": activeLoc,
		"location_ids":       locIDs,
		"user": map[string]any{
			"id":          user.UserID,
			"retailer_id": ret.RetailerID,
			"name":        coalesceRetailerName(user.Name),
			"company":     coalesceRetailerName(ret.Name),
			"email":       "",
			"avatar_url":  nil,
			"role":        role,
		},
	}
	if fbToken, err := auth.MintCustomToken(context.Background(), user.UserID, map[string]interface{}{
		"role":             string(auth.RoleRetailer),
		"retailer_id":      ret.RetailerID,
		"retailer_org_id":  ret.RetailerID,
		"retailer_user_id": user.UserID,
		"retailer_role":    role,
		"supplier_id":      claims.SupplierID,
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

func roleHomeSurface(role string) string {
	switch strings.ToUpper(role) {
	case "CASHIER":
		return "pos"
	case "RECEIVER", "STOCK_CLERK":
		return "dock"
	case "BUYER":
		return "catalog"
	case "VIEWER":
		return "insights"
	default:
		return "dashboard"
	}
}

func (s *Service) writeMobileAuthResponse(w http.ResponseWriter, ret Retailer) {
	status, body := s.marshalMobileAuthResponse(ret)
	writeJSONBytes(w, status, body)
}

func (s *Service) writeMobileAuthResponseForUser(w http.ResponseWriter, ret Retailer, user RetailerUser) {
	status, body := s.marshalMobileAuthResponseForUser(ret, user)
	writeJSONBytes(w, status, body)
}

func coalesceRetailerName(name string) string {
	if strings.TrimSpace(name) != "" {
		return strings.TrimSpace(name)
	}
	return demoRetailerStoreName()
}
