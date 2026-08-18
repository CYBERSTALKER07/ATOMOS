package factory

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/web"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
	"github.com/pegasusx/pegasusx/apps/backend-go/staffinvite"
)

// HandleFactoryRegister serves POST /v1/auth/factory/register
// Accepts legacy {name, phone, password} and portal {account, id_token} shapes.
func (s *Service) HandleFactoryRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name              string `json:"name"`
		Phone             string `json:"phone"`
		Password          string `json:"password"`
		AssignedFactoryID string `json:"assigned_factory_id"`
		InviteToken       string `json:"invite_token"`
		IDToken           string `json:"id_token"`
		Account           *struct {
			LegalName   string `json:"legalName"`
			ContactName string `json:"contactName"`
			Email       string `json:"email"`
			Country     string `json:"country"`
			Phone       string `json:"phone"`
		} `json:"account"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.JSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	name := strings.TrimSpace(req.Name)
	phone := strings.TrimSpace(req.Phone)
	password := strings.TrimSpace(req.Password)
	factoryID := strings.TrimSpace(req.AssignedFactoryID)
	country := ""

	if req.Account != nil {
		if name == "" {
			name = strings.TrimSpace(req.Account.ContactName)
		}
		if name == "" {
			name = strings.TrimSpace(req.Account.LegalName)
		}
		if phone == "" {
			phone = strings.TrimSpace(req.Account.Phone)
		}
		country = strings.ToUpper(strings.TrimSpace(req.Account.Country))
	}

	idToken := strings.TrimSpace(req.IDToken)
	if idToken != "" && s.firebaseVerifier == nil {
		web.JSONError(w, auth.FirebaseLoginUnavailable, http.StatusServiceUnavailable)
		return
	}
	if idToken != "" && s.firebaseVerifier != nil {
		fbClaims, err := s.firebaseVerifier.VerifyIDToken(r.Context(), idToken)
		if err != nil {
			s.log.Warn("factory register firebase token verification failed", "err", err)
			web.JSONError(w, "invalid_id_token", http.StatusUnauthorized)
			return
		}
		if fbClaims.PhoneNumber != "" {
			phone = fbClaims.PhoneNumber
		}
	}

	if name == "" || phone == "" {
		web.JSONError(w, "name and phone are required", http.StatusBadRequest)
		return
	}

	admin, _ := staffinvite.ParseBearer(r.Header.Get("Authorization"), s.jwtSecret)
	scope, err := staffinvite.ResolveRegister(staffinvite.ResolveOpts{
		Secret:         s.jwtSecret,
		InviteToken:    req.InviteToken,
		WantRole:       staffinvite.RoleFactory,
		RequestedNode:  factoryID,
		SeedSupplierID: s.seedSupplierID,
		Admin:          admin,
		Now:            s.now(),
		NodeOwned:      staffinvite.SpannerNodeOwned(s.spannerClient),
		Ctx:            r.Context(),
	})
	if err != nil {
		staffinvite.WriteError(w, err)
		return
	}
	factoryID = scope.NodeID
	supplierID := scope.SupplierID

	passwordHash, err := staffinvite.HashPassword(password)
	if err != nil {
		staffinvite.WriteError(w, err)
		return
	}
	if s.spannerClient == nil {
		web.JSONError(w, "Spanner not configured", http.StatusServiceUnavailable)
		return
	}

	userID := "usr-" + uuid.NewString()[:8]
	now := s.now().UTC()

	var assignedFactory spanner.NullString
	if factoryID != "" {
		assignedFactory = spanner.NullString{StringVal: factoryID, Valid: true}
	}

	m := spanner.Insert("SupplierUsers",
		[]string{"UserId", "SupplierId", "Name", "Phone", "PasswordHash", "SupplierRole", "AssignedFactoryId", "IsActive", "CreatedAt", "UpdatedAt"},
		[]any{userID, supplierID, name, phone, passwordHash, "FACTORY", assignedFactory, true, now, now},
	)
	muts := []*spanner.Mutation{m}
	if factoryID != "" {
		pack, packErr := auth.CheckoutPackFromContext(r.Context())
		if packErr != nil {
			if writeMarketLaw(w, packErr) {
				return
			}
			web.JSONError(w, packErr.Error(), http.StatusUnprocessableEntity)
			return
		}
		stamped, stampErr := proximity.ResolveNodeCountry(pack, country)
		if stampErr != nil {
			if writeMarketLaw(w, stampErr) {
				return
			}
			web.JSONError(w, stampErr.Error(), http.StatusUnprocessableEntity)
			return
		}
		muts = append(muts, spanner.UpdateMap("Factories", map[string]any{
			"FactoryId":   factoryID,
			"CountryCode": stamped,
			"UpdatedAt":   spanner.CommitTimestamp,
		}))
	}

	if _, err := s.spannerClient.Apply(r.Context(), muts); err != nil {
		s.log.ErrorContext(r.Context(), "failed to register factory user", "err", err)
		web.JSONError(w, "Failed to register factory user", http.StatusInternalServerError)
		return
	}

	if s.jwtSecret == "" {
		web.JSONError(w, "jwt_not_configured", http.StatusServiceUnavailable)
		return
	}

	isConfigured := s.factoryIsConfigured(r.Context(), factoryID)
	jwtClaims := auth.Claims{
		Subject:      userID,
		Role:         auth.RoleFactory,
		SupplierID:   supplierID,
		SupplierRole: auth.RoleFactory,
		HomeNodeType: auth.HomeNodeFactory,
		HomeNodeID:   factoryID,
		IsConfigured: isConfigured,
		PhoneNumber:  phone,
	}
	if factoryID == "" {
		jwtClaims.HomeNodeID = ""
	}

	token, err := auth.Issue(jwtClaims, auth.IssueOptions{Secret: s.jwtSecret, Issuer: s.jwtIssuer, TTL: 24 * time.Hour})
	if err != nil {
		web.JSONError(w, "issue_token_failed", http.StatusInternalServerError)
		return
	}
	refresh, err := auth.Issue(jwtClaims, auth.IssueOptions{Secret: s.jwtSecret, Issuer: s.jwtIssuer, TTL: 7 * 24 * time.Hour})
	if err != nil {
		web.JSONError(w, "issue_refresh_failed", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"user_id":             userID,
		"name":                name,
		"phone":               phone,
		"supplier_role":       "FACTORY",
		"assigned_factory_id": factoryID,
		"token":               token,
		"refresh_token":       refresh,
		"is_configured":       isConfigured,
	})
}
