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

// HandleFactorySetup serves POST /v1/factory/setup
func (s *Service) HandleFactorySetup(w http.ResponseWriter, r *http.Request) {
	if s.spannerClient == nil {
		web.JSONError(w, "Spanner not configured", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		Name              string  `json:"name"`
		FactoryName       string  `json:"factoryName"`
		Address           string  `json:"address"`
		City              string  `json:"city"`
		PostalCode        string  `json:"postalCode"`
		PlaceID           string  `json:"place_id"`
		Lat               float64 `json:"lat"`
		Lng               float64 `json:"lng"`
		FacilityType      string  `json:"facilityType"`
		TotalCapacitySqM  int     `json:"totalCapacitySqM"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.JSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = strings.TrimSpace(req.FactoryName)
	}
	if name == "" {
		web.JSONError(w, "name is required", http.StatusBadRequest)
		return
	}

	address := strings.TrimSpace(req.Address)
	if city := strings.TrimSpace(req.City); city != "" {
		if address != "" {
			address = address + ", " + city
		} else {
			address = city
		}
	}
	if postal := strings.TrimSpace(req.PostalCode); postal != "" && address != "" {
		address = address + " " + postal
	}

	facID := "fac-" + uuid.NewString()[:8]
	now := s.now().UTC()

	m := spanner.Insert("Factories",
		[]string{"FactoryId", "SupplierId", "Name", "Address", "PlaceId", "Lat", "Lng", "IsActive", "CreatedAt", "UpdatedAt"},
		[]any{facID, s.supplierID, name, factoryNullableString(address), factoryNullableString(strings.TrimSpace(req.PlaceID)), req.Lat, req.Lng, true, now, now},
	)

	mutations := []*spanner.Mutation{m}
	var staffUserID string
	var staffPhone string
	var staffRole string
	if claims, ok := s.parseSetupClaims(r); ok && strings.TrimSpace(claims.Subject) != "" {
		staffUserID = strings.TrimSpace(claims.Subject)
		staffPhone = strings.TrimSpace(claims.PhoneNumber)
		staffRole = string(claims.SupplierRole)
		if staffRole == "" {
			staffRole = string(auth.RoleFactory)
		}
		mutations = append(mutations, spanner.UpdateMap("SupplierUsers", map[string]any{
			"UserId":            staffUserID,
			"AssignedFactoryId": facID,
			"UpdatedAt":         now,
		}))
	}

	if _, err := s.spannerClient.Apply(r.Context(), mutations); err != nil {
		s.log.ErrorContext(r.Context(), "failed to create factory", "err", err)
		web.JSONError(w, "Failed to create factory", http.StatusInternalServerError)
		return
	}

	resp := map[string]any{
		"factory_id":  facID,
		"supplier_id": s.supplierID,
		"name":        name,
		"address":     address,
		"lat":         req.Lat,
		"lng":         req.Lng,
		"is_configured": staffUserID != "",
	}

	if staffUserID != "" && s.jwtSecret != "" {
		jwtClaims := auth.Claims{
			Subject:      staffUserID,
			Role:         auth.RoleFactory,
			SupplierID:   s.supplierID,
			SupplierRole: auth.Role(staffRole),
			HomeNodeType: auth.HomeNodeFactory,
			HomeNodeID:   facID,
			IsConfigured: true,
			PhoneNumber:  staffPhone,
		}
		if strings.EqualFold(staffRole, string(auth.RoleFactoryAdmin)) {
			jwtClaims.SupplierRole = auth.RoleFactoryAdmin
		}
		token, err := auth.Issue(jwtClaims, auth.IssueOptions{Secret: s.jwtSecret, Issuer: s.jwtIssuer, TTL: 24 * time.Hour})
		if err == nil {
			refresh, refreshErr := auth.Issue(jwtClaims, auth.IssueOptions{Secret: s.jwtSecret, Issuer: s.jwtIssuer, TTL: 7 * 24 * time.Hour})
			if refreshErr == nil {
				resp["token"] = token
				resp["refresh_token"] = refresh
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
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
