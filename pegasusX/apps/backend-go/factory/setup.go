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
)

type factorySetupRequest struct {
	Name         string  `json:"name"`
	FactoryName  string  `json:"factoryName"`
	Address      string  `json:"address"`
	PlaceID      string  `json:"place_id"`
	Lat          float64 `json:"lat"`
	Lng          float64 `json:"lng"`
	FacilityType string  `json:"facilityType"`
}

// HandleFactorySetup serves POST /v1/factory/setup
// Creates a factory for self-serve onboarding or updates location on an existing assigned factory.
func (s *Service) HandleFactorySetup(w http.ResponseWriter, r *http.Request) {
	if s.spannerClient == nil {
		web.JSONError(w, "Spanner not configured", http.StatusServiceUnavailable)
		return
	}
	if s.jwtSecret == "" {
		web.JSONError(w, "jwt_not_configured", http.StatusServiceUnavailable)
		return
	}

	claims, ok := s.parseSetupClaims(r)
	if !ok || strings.TrimSpace(claims.Subject) == "" {
		web.JSONError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req factorySetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.JSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if strings.TrimSpace(req.Address) == "" {
		web.JSONError(w, "address is required", http.StatusBadRequest)
		return
	}
	if req.Lat < -90 || req.Lat > 90 || req.Lng < -180 || req.Lng > 180 {
		web.JSONError(w, "coordinates_out_of_range", http.StatusBadRequest)
		return
	}

	now := s.now().UTC()
	factoryID := strings.TrimSpace(claims.HomeNodeID)
	supplierID := strings.TrimSpace(claims.SupplierID)
	if supplierID == "" {
		supplierID = s.supplierID
	}

	mutations := make([]*spanner.Mutation, 0, 2)

	if factoryID == "" {
		name := strings.TrimSpace(req.Name)
		if name == "" {
			name = strings.TrimSpace(req.FactoryName)
		}
		if name == "" {
			web.JSONError(w, "name is required", http.StatusBadRequest)
			return
		}
		factoryID = "fac-" + uuid.NewString()[:8]
		mutations = append(mutations, spanner.Insert("Factories",
			[]string{"FactoryId", "SupplierId", "Name", "Address", "PlaceId", "Lat", "Lng", "IsActive", "CreatedAt", "UpdatedAt"},
			[]any{
				factoryID,
				supplierID,
				name,
				strings.TrimSpace(req.Address),
				factoryNullableString(strings.TrimSpace(req.PlaceID)),
				req.Lat,
				req.Lng,
				true,
				now,
				now,
			},
		))
		mutations = append(mutations, spanner.UpdateMap("SupplierUsers", map[string]any{
			"UserId":            claims.Subject,
			"AssignedFactoryId": factoryID,
			"UpdatedAt":         now,
		}))
	} else {
		update := map[string]any{
			"FactoryId": factoryID,
			"Address":   strings.TrimSpace(req.Address),
			"PlaceId":   factoryNullableString(strings.TrimSpace(req.PlaceID)),
			"Lat":       req.Lat,
			"Lng":       req.Lng,
			"UpdatedAt": now,
		}
		if name := strings.TrimSpace(req.Name); name != "" {
			update["Name"] = name
		} else if name := strings.TrimSpace(req.FactoryName); name != "" {
			update["Name"] = name
		}
		mutations = append(mutations, spanner.UpdateMap("Factories", update))
	}

	if _, err := s.spannerClient.Apply(r.Context(), mutations); err != nil {
		s.log.ErrorContext(r.Context(), "failed to complete factory setup", "err", err)
		web.JSONError(w, "Failed to complete factory setup", http.StatusInternalServerError)
		return
	}

	isConfigured := s.factoryIsConfigured(r.Context(), factoryID)
	staffRole := string(claims.SupplierRole)
	if staffRole == "" {
		staffRole = string(auth.RoleFactory)
	}
	jwtClaims := auth.Claims{
		Subject:      claims.Subject,
		Role:         auth.RoleFactory,
		SupplierID:   supplierID,
		SupplierRole: auth.Role(staffRole),
		HomeNodeType: auth.HomeNodeFactory,
		HomeNodeID:   factoryID,
		IsConfigured: isConfigured,
		PhoneNumber:  claims.PhoneNumber,
	}
	if strings.EqualFold(staffRole, string(auth.RoleFactoryAdmin)) {
		jwtClaims.SupplierRole = auth.RoleFactoryAdmin
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
		"factory_id":    factoryID,
		"supplier_id":   supplierID,
		"token":         token,
		"refresh_token": refresh,
		"is_configured": isConfigured,
	})
}

func (s *Service) parseSetupClaims(r *http.Request) (auth.Claims, bool) {
	if claims, ok := auth.FromContext(r.Context()); ok {
		return claims, true
	}
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(authHeader, "Bearer ") || s.jwtSecret == "" {
		return auth.Claims{}, false
	}
	token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	claims, err := auth.Parse(token, s.jwtSecret)
	if err != nil {
		return auth.Claims{}, false
	}
	return claims, true
}
