package warehouse

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

type warehouseSetupRequest struct {
	Name             string  `json:"name"`
	Address          string  `json:"address"`
	PlaceID          string  `json:"place_id"`
	Lat              float64 `json:"lat"`
	Lng              float64 `json:"lng"`
	CoverageRadiusKm float64 `json:"coverage_radius_km"`
	PrimaryFactoryID string  `json:"primary_factory_id"`
}

// HandleWarehouseSetup serves POST /v1/warehouse/setup
// Creates a warehouse for self-serve onboarding or updates location on an existing assigned warehouse.
func (s *Service) HandleWarehouseSetup(w http.ResponseWriter, r *http.Request) {
	if s.spannerClient == nil {
		web.JSONError(w, "Spanner not configured", http.StatusServiceUnavailable)
		return
	}
	if s.jwtSecret == "" {
		web.JSONError(w, "jwt_not_configured", http.StatusServiceUnavailable)
		return
	}

	claims, ok := s.warehouseClaimsFromRequest(r)
	if !ok || strings.TrimSpace(claims.Subject) == "" {
		web.JSONError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req warehouseSetupRequest
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
	if req.CoverageRadiusKm <= 0 {
		req.CoverageRadiusKm = 25.0
	}

	warehouseID := strings.TrimSpace(claims.HomeNodeID)
	primaryFactoryID := strings.TrimSpace(req.PrimaryFactoryID)

	if warehouseID != "" && primaryFactoryID != "" {
		if err := s.validateSupplyCycle(r.Context(), warehouseID, primaryFactoryID); err != nil {
			web.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	now := s.now().UTC()
	supplierID := strings.TrimSpace(claims.SupplierID)
	if supplierID == "" {
		supplierID = s.resolveSupplierScope(r.Context())
	}

	geo, err := stampWarehouseCoords(r.Context(), req.Lat, req.Lng, "")
	if err != nil {
		if writeMarketLaw(w, err) {
			return
		}
		web.JSONError(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	mutations := make([]*spanner.Mutation, 0, 2)

	if warehouseID == "" {
		name := strings.TrimSpace(req.Name)
		if name == "" {
			web.JSONError(w, "name is required", http.StatusBadRequest)
			return
		}
		warehouseID = "wh-" + uuid.NewString()[:8]
		mutations = append(mutations, spanner.Insert("Warehouses",
			[]string{"WarehouseId", "SupplierId", "Name", "Address", "PlaceId", "Lat", "Lng", "CoverageRadiusKm", "PrimaryFactoryId", "CountryCode", "H3Cell", "IsActive", "IsOnShift", "CreatedAt", "UpdatedAt"},
			[]any{
				warehouseID,
				supplierID,
				name,
				strings.TrimSpace(req.Address),
				nullableString(strings.TrimSpace(req.PlaceID)),
				req.Lat,
				req.Lng,
				req.CoverageRadiusKm,
				strings.TrimSpace(req.PrimaryFactoryID),
				geo.CountryCode,
				geo.H3Cell,
				true,
				false,
				now,
				now,
			},
		))
		mutations = append(mutations, spanner.UpdateMap("SupplierUsers", map[string]any{
			"UserId":              claims.Subject,
			"AssignedWarehouseId": warehouseID,
			"UpdatedAt":           now,
		}))
	} else {
		update := map[string]any{
			"WarehouseId": warehouseID,
			"Address":     strings.TrimSpace(req.Address),
			"PlaceId":     nullableString(strings.TrimSpace(req.PlaceID)),
			"Lat":         req.Lat,
			"Lng":         req.Lng,
			"CountryCode": geo.CountryCode,
			"H3Cell":      geo.H3Cell,
			"UpdatedAt":   now,
		}
		if name := strings.TrimSpace(req.Name); name != "" {
			update["Name"] = name
		}
		mutations = append(mutations, spanner.UpdateMap("Warehouses", update))
	}

	if _, err := s.spannerClient.Apply(r.Context(), mutations); err != nil {
		s.log.ErrorContext(r.Context(), "failed to complete warehouse setup", "err", err)
		web.JSONError(w, "Failed to complete warehouse setup", http.StatusInternalServerError)
		return
	}

	isConfigured := s.warehouseIsConfigured(r.Context(), warehouseID)
	jwtClaims := auth.Claims{
		Subject:      claims.Subject,
		Role:         auth.RoleWarehouse,
		SupplierID:   supplierID,
		SupplierRole: claims.SupplierRole,
		HomeNodeType: auth.HomeNodeWarehouse,
		HomeNodeID:   warehouseID,
		IsConfigured: isConfigured,
		PhoneNumber:  claims.PhoneNumber,
	}
	if jwtClaims.SupplierRole == "" {
		jwtClaims.SupplierRole = auth.RoleWarehouse
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
		"warehouse_id":  warehouseID,
		"supplier_id":   supplierID,
		"token":         token,
		"refresh_token": refresh,
		"is_configured": isConfigured,
	})
}

func (s *Service) warehouseClaimsFromRequest(r *http.Request) (auth.Claims, bool) {
	if claims, ok := auth.FromContext(r.Context()); ok {
		return claims, true
	}
	token := auth.BearerToken(r)
	if token == "" || s.jwtSecret == "" {
		return auth.Claims{}, false
	}
	claims, err := auth.Parse(token, s.jwtSecret)
	if err != nil {
		return auth.Claims{}, false
	}
	return claims, true
}
