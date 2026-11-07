package retailer

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/gs1"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

// RetailerLocation is a store branch under a retailer org.
type RetailerLocation struct {
	LocationID           string
	RetailerID           string
	Name                 string
	DeliveryAddress      string
	PlaceID              string
	Lat                  float64
	Lng                  float64
	H3Cell               string
	CountryCode          string
	ReceivingWindowOpen  string
	ReceivingWindowClose string
	Gln                  string
	IsPrimary            bool
	IsActive             bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// LocationDTO is the wire shape.
type LocationDTO struct {
	LocationID           string  `json:"location_id"`
	RetailerID           string  `json:"retailer_id"`
	Name                 string  `json:"name"`
	DeliveryAddress      string  `json:"delivery_address,omitempty"`
	PlaceID              string  `json:"place_id,omitempty"`
	Lat                  float64 `json:"lat,omitempty"`
	Lng                  float64 `json:"lng,omitempty"`
	H3Cell               string  `json:"h3_cell,omitempty"`
	CountryCode          string  `json:"country_code,omitempty"`
	ReceivingWindowOpen  string  `json:"receiving_window_open,omitempty"`
	ReceivingWindowClose string  `json:"receiving_window_close,omitempty"`
	Gln                  string  `json:"gln,omitempty"`
	IsPrimary            bool    `json:"is_primary"`
	IsActive             bool    `json:"is_active"`
	CreatedAt            string  `json:"created_at,omitempty"`
	UpdatedAt            string  `json:"updated_at,omitempty"`
}

type locationsResponse struct {
	RetailerID       string        `json:"retailer_id"`
	ActiveLocationID string        `json:"active_location_id,omitempty"`
	Items            []LocationDTO `json:"items"`
}

type locationCreateRequest struct {
	Name                 string  `json:"name"`
	DeliveryAddress      string  `json:"delivery_address"`
	PlaceID              string  `json:"place_id"`
	Lat                  float64 `json:"lat"`
	Lng                  float64 `json:"lng"`
	H3Cell               string  `json:"h3_cell"`
	CountryCode          string  `json:"country_code"`
	ReceivingWindowOpen  string  `json:"receiving_window_open"`
	ReceivingWindowClose string  `json:"receiving_window_close"`
	IsPrimary            *bool   `json:"is_primary,omitempty"`
}

type locationUpdateRequest struct {
	Name                 *string  `json:"name,omitempty"`
	DeliveryAddress      *string  `json:"delivery_address,omitempty"`
	PlaceID              *string  `json:"place_id,omitempty"`
	Lat                  *float64 `json:"lat,omitempty"`
	Lng                  *float64 `json:"lng,omitempty"`
	H3Cell               *string  `json:"h3_cell,omitempty"`
	CountryCode          *string  `json:"country_code,omitempty"`
	ReceivingWindowOpen  *string  `json:"receiving_window_open,omitempty"`
	ReceivingWindowClose *string  `json:"receiving_window_close,omitempty"`
	Gln                  *string  `json:"gln,omitempty"`
	IsActive             *bool    `json:"is_active,omitempty"`
}

// HandleLocations serves GET/POST /v1/retailer/locations.
func (s *Service) HandleLocations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleLocationsGet(w, r)
	case http.MethodPost:
		s.handleLocationsPost(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

// HandleLocationByID serves PATCH /v1/retailer/locations/{locationID}.
func (s *Service) HandleLocationByID(w http.ResponseWriter, r *http.Request) {
	locID := strings.TrimSpace(chi.URLParam(r, "locationID"))
	if locID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "location_id_required"})
		return
	}
	switch r.Method {
	case http.MethodPatch, http.MethodPut:
		s.handleLocationPatch(w, r, locID)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

// HandleLocationSetPrimary serves POST /v1/retailer/locations/{locationID}/set-primary.
func (s *Service) HandleLocationSetPrimary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || !auth.HasRetailerPerm(claims, auth.PermLocationManage) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "permission": auth.PermLocationManage})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	locID := strings.TrimSpace(chi.URLParam(r, "locationID"))
	if locID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "location_id_required"})
		return
	}
	if err := s.setPrimaryLocation(r.Context(), orgID, locID); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	s.writeLocationsResponse(w, r, orgID, claims)
}

// HandleSwitchLocation serves POST /v1/auth/retailer/switch-location
// body: { "location_id": "..." } — re-issues JWT with active_location_id.
func (s *Service) HandleSwitchLocation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if s.jwtSecret == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "jwt_not_configured"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleRetailer {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	orgID := auth.ResolveRetailerOrgID(claims)
	var req struct {
		LocationID string `json:"location_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	locID := strings.TrimSpace(req.LocationID)
	if locID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "location_id_required"})
		return
	}
	loc, found, err := s.getLocation(r.Context(), locID)
	if err != nil || !found || loc.RetailerID != orgID || !loc.IsActive {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "location_not_found"})
		return
	}
	// Staff with scoped locations must be bound.
	role := auth.EffectiveRetailerRole(claims)
	if role != "OWNER" && role != "ADMIN" {
		bound, _ := s.listUserLocationIDs(r.Context(), auth.ResolveRetailerUserID(claims))
		if len(bound) > 0 && !containsString(bound, locID) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "location_not_in_scope"})
			return
		}
	}
	claims.ActiveLocationID = locID
	if packs, err := s.LoadEnabledPacks(r.Context(), orgID); err == nil {
		claims.CapabilityPacks = packs.List()
	}
	token, err := auth.Issue(claims, auth.IssueOptions{Secret: s.jwtSecret, Issuer: s.jwtIssuer, TTL: 24 * time.Hour})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "issue_token_failed"})
		return
	}
	refresh, err := auth.Issue(claims, auth.IssueOptions{Secret: s.jwtSecret, Issuer: s.jwtIssuer, TTL: 7 * 24 * time.Hour})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "issue_refresh_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"token":              token,
		"refresh_token":      refresh,
		"active_location_id": locID,
		"location":           dtoFromLocation(loc),
	})
}

// HandleMemberLocations serves PUT /v1/retailer/org/members/{userID}/locations
// body: { "location_ids": ["..."] }
func (s *Service) HandleMemberLocations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || !auth.HasRetailerPerm(claims, auth.PermStaffManage) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "permission": auth.PermStaffManage})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	userID := strings.TrimSpace(chi.URLParam(r, "userID"))
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "user_id_required"})
		return
	}
	target, found, err := s.findRetailerUserByID(r.Context(), userID)
	if err != nil || !found || target.RetailerID != orgID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "retailer_org_member_not_found"})
		return
	}
	var req struct {
		LocationIDs []string `json:"location_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	// Validate locations belong to org.
	for _, id := range req.LocationIDs {
		loc, okLoc, err := s.getLocation(r.Context(), strings.TrimSpace(id))
		if err != nil || !okLoc || loc.RetailerID != orgID {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "invalid_location_id", "location_id": id})
			return
		}
	}
	if err := s.replaceUserLocations(r.Context(), orgID, userID, req.LocationIDs); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "bind_locations_failed"})
		return
	}
	ids, _ := s.listUserLocationIDs(r.Context(), userID)
	writeJSON(w, http.StatusOK, map[string]any{
		"user_id":      userID,
		"location_ids": ids,
	})
}

func (s *Service) handleLocationsGet(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	// Bootstrap primary from shop profile when empty.
	if _, err := s.EnsurePrimaryLocation(r.Context(), orgID); err != nil && s.log != nil {
		s.log.Warn("ensure primary location failed", "err", err, "retailer_id", orgID)
	}
	s.writeLocationsResponse(w, r, orgID, claims)
}

func (s *Service) handleLocationsPost(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || !auth.HasRetailerPerm(claims, auth.PermLocationManage) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "permission": auth.PermLocationManage})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	body, okBody := readLimitedBody(w, r, 32*1024)
	if !okBody {
		return
	}
	if s.guardIdempotency(w, r, body) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	var req locationCreateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	name := strings.TrimSpace(req.Name)
	if len(name) < 1 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "location_name_required"})
		return
	}

	// Auto-enable LOCATIONS pack when creating a non-primary branch.
	enabled, _ := s.LoadEnabledPacks(r.Context(), orgID)
	if !enabled.Has(PackLOCATIONS) {
		_ = s.SetPackEnabled(r.Context(), orgID, PackLOCATIONS, auth.ResolveRetailerUserID(claims), true, map[string]any{})
	}

	// Ensure primary exists first.
	primary, _ := s.EnsurePrimaryLocation(r.Context(), orgID)

	country, h3, stampErr := stampRetailerOptionalCoords(r.Context(), req.Lat, req.Lng, req.CountryCode)
	if stampErr != nil {
		if writeMarketLaw(w, stampErr) {
			return
		}
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": stampErr.Error()})
		return
	}

	loc := RetailerLocation{
		LocationID:           s.newID(),
		RetailerID:           orgID,
		Name:                 name,
		DeliveryAddress:      strings.TrimSpace(req.DeliveryAddress),
		PlaceID:              strings.TrimSpace(req.PlaceID),
		Lat:                  req.Lat,
		Lng:                  req.Lng,
		H3Cell:               h3,
		CountryCode:          country,
		ReceivingWindowOpen:  strings.TrimSpace(req.ReceivingWindowOpen),
		ReceivingWindowClose: strings.TrimSpace(req.ReceivingWindowClose),
		IsPrimary:            false,
		IsActive:             true,
		CreatedAt:            s.now(),
		UpdatedAt:            s.now(),
	}
	if req.IsPrimary != nil && *req.IsPrimary {
		loc.IsPrimary = true
	}
	// If this is the very first location, mark primary.
	if primary.LocationID == "" || primary.LocationID == loc.LocationID {
		loc.IsPrimary = true
	}

	if err := s.insertLocation(r.Context(), loc); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "persist_location_failed", "detail": err.Error()})
		return
	}
	if loc.IsPrimary {
		_ = s.setPrimaryLocation(r.Context(), orgID, loc.LocationID)
	}

	resp := locationsResponse{RetailerID: orgID, Items: []LocationDTO{dtoFromLocation(loc)}}
	respBytes, _ := json.Marshal(resp)
	idemCommitted = true
	s.saveIdempotency(r.Context(), r, body, http.StatusCreated, respBytes)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(respBytes)
}

func (s *Service) handleLocationPatch(w http.ResponseWriter, r *http.Request, locID string) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || !auth.HasRetailerPerm(claims, auth.PermLocationManage) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "permission": auth.PermLocationManage})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	loc, found, err := s.getLocation(r.Context(), locID)
	if err != nil || !found || loc.RetailerID != orgID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "location_not_found"})
		return
	}
	var req locationUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if req.Name != nil {
		n := strings.TrimSpace(*req.Name)
		if n == "" {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "location_name_required"})
			return
		}
		loc.Name = n
	}
	if req.DeliveryAddress != nil {
		loc.DeliveryAddress = strings.TrimSpace(*req.DeliveryAddress)
	}
	if req.PlaceID != nil {
		loc.PlaceID = strings.TrimSpace(*req.PlaceID)
	}
	if req.Lat != nil {
		loc.Lat = *req.Lat
	}
	if req.Lng != nil {
		loc.Lng = *req.Lng
	}
	if req.CountryCode != nil {
		loc.CountryCode = strings.TrimSpace(*req.CountryCode)
	}
	country, cell, stampErr := stampRetailerOptionalCoords(r.Context(), loc.Lat, loc.Lng, loc.CountryCode)
	if stampErr != nil {
		if writeMarketLaw(w, stampErr) {
			return
		}
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": stampErr.Error()})
		return
	}
	loc.CountryCode = country
	if cell != "" {
		loc.H3Cell = cell
	} else if req.H3Cell != nil {
		loc.H3Cell = strings.TrimSpace(*req.H3Cell)
	}
	if req.ReceivingWindowOpen != nil {
		loc.ReceivingWindowOpen = strings.TrimSpace(*req.ReceivingWindowOpen)
	}
	if req.ReceivingWindowClose != nil {
		loc.ReceivingWindowClose = strings.TrimSpace(*req.ReceivingWindowClose)
	}
	if req.Gln != nil {
		raw := strings.TrimSpace(*req.Gln)
		if raw == "" {
			loc.Gln = ""
		} else {
			norm, err := gs1.NormalizeGLN(raw)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			loc.Gln = norm
		}
	}
	if req.IsActive != nil {
		if loc.IsPrimary && !*req.IsActive {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "cannot_deactivate_primary"})
			return
		}
		loc.IsActive = *req.IsActive
	}
	loc.UpdatedAt = s.now()
	if err := s.updateLocation(r.Context(), loc); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update_location_failed"})
		return
	}
	// If primary location geo changes, mirror to Retailers row for legacy delivery.
	if loc.IsPrimary {
		_ = s.mirrorPrimaryToRetailer(r.Context(), loc)
	}
	writeJSON(w, http.StatusOK, dtoFromLocation(loc))
}

func (s *Service) writeLocationsResponse(w http.ResponseWriter, r *http.Request, orgID string, claims auth.Claims) {
	items, err := s.listLocations(r.Context(), orgID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_locations_failed"})
		return
	}
	// Scope for non-owner staff with binds.
	role := auth.EffectiveRetailerRole(claims)
	if role != "OWNER" && role != "ADMIN" {
		bound, _ := s.listUserLocationIDs(r.Context(), auth.ResolveRetailerUserID(claims))
		if len(bound) > 0 {
			set := map[string]bool{}
			for _, id := range bound {
				set[id] = true
			}
			filtered := items[:0]
			for _, it := range items {
				if set[it.LocationID] {
					filtered = append(filtered, it)
				}
			}
			items = filtered
		}
	}
	active := strings.TrimSpace(claims.ActiveLocationID)
	if active == "" {
		for _, it := range items {
			if it.IsPrimary {
				active = it.LocationID
				break
			}
		}
		if active == "" && len(items) > 0 {
			active = items[0].LocationID
		}
	}
	dtos := make([]LocationDTO, 0, len(items))
	for _, it := range items {
		dtos = append(dtos, dtoFromLocation(it))
	}
	writeJSON(w, http.StatusOK, locationsResponse{
		RetailerID:       orgID,
		ActiveLocationID: active,
		Items:            dtos,
	})
}

// EnsurePrimaryLocation creates a primary location from the Retailers row when missing.
func (s *Service) EnsurePrimaryLocation(ctx context.Context, retailerID string) (RetailerLocation, error) {
	items, err := s.listLocations(ctx, retailerID)
	if err != nil {
		return RetailerLocation{}, err
	}
	for _, it := range items {
		if it.IsPrimary && it.IsActive {
			return it, nil
		}
	}
	if len(items) > 0 {
		// Promote first active.
		for _, it := range items {
			if it.IsActive {
				_ = s.setPrimaryLocation(ctx, retailerID, it.LocationID)
				it.IsPrimary = true
				return it, nil
			}
		}
	}
	var ret Retailer
	if s.repo != nil {
		found := false
		var err error
		ret, found, err = s.repo.GetRetailer(ctx, retailerID)
		if err != nil {
			return RetailerLocation{}, err
		}
		if !found {
			// Memory/test path: synthesize minimal primary without Spanner retailer row.
			ret = Retailer{RetailerID: retailerID, Name: "Primary store"}
		}
	} else {
		ret = Retailer{RetailerID: retailerID, Name: "Primary store"}
	}
	name := coalesceRetailerName(ret.Name)
	if name == "" {
		name = "Primary store"
	}
	country, cell, stampErr := stampRetailerOptionalCoords(ctx, ret.Lat, ret.Lng, ret.CountryCode)
	if stampErr != nil {
		return RetailerLocation{}, stampErr
	}
	h3 := ret.H3Cell
	if cell != "" {
		h3 = cell
	}
	loc := RetailerLocation{
		LocationID:           s.newID(),
		RetailerID:           retailerID,
		Name:                 name,
		DeliveryAddress:      ret.DeliveryAddress,
		PlaceID:              ret.PlaceID,
		Lat:                  ret.Lat,
		Lng:                  ret.Lng,
		H3Cell:               h3,
		CountryCode:          country,
		ReceivingWindowOpen:  ret.ReceivingWindowOpen,
		ReceivingWindowClose: ret.ReceivingWindowClose,
		IsPrimary:            true,
		IsActive:             true,
		CreatedAt:            s.now(),
		UpdatedAt:            s.now(),
	}
	if err := s.insertLocation(ctx, loc); err != nil {
		// race: re-list
		if items2, err2 := s.listLocations(ctx, retailerID); err2 == nil && len(items2) > 0 {
			return items2[0], nil
		}
		return RetailerLocation{}, err
	}
	return loc, nil
}

// ResolveActiveLocation returns the active location for delivery/checkout bind.
func (s *Service) ResolveActiveLocation(ctx context.Context, claims auth.Claims) (RetailerLocation, bool, error) {
	orgID := auth.ResolveRetailerOrgID(claims)
	if orgID == "" {
		return RetailerLocation{}, false, nil
	}
	if id := strings.TrimSpace(claims.ActiveLocationID); id != "" {
		loc, ok, err := s.getLocation(ctx, id)
		if err != nil {
			return RetailerLocation{}, false, err
		}
		if ok && loc.RetailerID == orgID && loc.IsActive {
			return loc, true, nil
		}
	}
	loc, err := s.EnsurePrimaryLocation(ctx, orgID)
	if err != nil {
		return RetailerLocation{}, false, err
	}
	return loc, true, nil
}

func (s *Service) listLocations(ctx context.Context, retailerID string) ([]RetailerLocation, error) {
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		return append([]RetailerLocation(nil), s.locationsByRetailer[retailerID]...), nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT LocationId, RetailerId, Name, IFNULL(DeliveryAddress, ''), IFNULL(PlaceId, ''),
			IFNULL(Lat, 0), IFNULL(Lng, 0), IFNULL(H3Cell, ''), IFNULL(CountryCode, ''),
			IFNULL(ReceivingWindowOpen, ''), IFNULL(ReceivingWindowClose, ''),
			IFNULL(Gln, ''),
			IsPrimary, IsActive, CreatedAt, UpdatedAt
			FROM RetailerLocations@{FORCE_INDEX=Idx_RetailerLocations_ByRetailer}
			WHERE RetailerId = @rid
			ORDER BY IsPrimary DESC, UpdatedAt DESC`,
		Params: map[string]any{"rid": retailerID},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	var out []RetailerLocation
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		loc, err := decodeLocationRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, loc)
	}
	return out, nil
}

func (s *Service) getLocation(ctx context.Context, locationID string) (RetailerLocation, bool, error) {
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		for _, list := range s.locationsByRetailer {
			for _, loc := range list {
				if loc.LocationID == locationID {
					return loc, true, nil
				}
			}
		}
		return RetailerLocation{}, false, nil
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "RetailerLocations", spanner.Key{locationID},
		[]string{"LocationId", "RetailerId", "Name", "DeliveryAddress", "PlaceId", "Lat", "Lng", "H3Cell", "CountryCode",
			"ReceivingWindowOpen", "ReceivingWindowClose", "Gln", "IsPrimary", "IsActive", "CreatedAt", "UpdatedAt"})
	if err != nil {
		if isNotFound(err) {
			return RetailerLocation{}, false, nil
		}
		return RetailerLocation{}, false, err
	}
	loc, err := decodeLocationRow(row)
	if err != nil {
		return RetailerLocation{}, false, err
	}
	return loc, true, nil
}

func decodeLocationRow(row *spanner.Row) (RetailerLocation, error) {
	var loc RetailerLocation
	var addr, place, h3, country, open, close, gln spanner.NullString
	var lat, lng spanner.NullFloat64
	var created, updated time.Time
	// Try flexible column decode.
	var name string
	err := row.Columns(
		&loc.LocationID, &loc.RetailerID, &name,
		&addr, &place, &lat, &lng, &h3, &country, &open, &close, &gln,
		&loc.IsPrimary, &loc.IsActive, &created, &updated,
	)
	if err != nil {
		// Query path uses IFNULL strings/floats
		var addrS, placeS, h3S, countryS, openS, closeS, glnS string
		var latF, lngF float64
		if err2 := row.Columns(
			&loc.LocationID, &loc.RetailerID, &name,
			&addrS, &placeS, &latF, &lngF, &h3S, &countryS, &openS, &closeS, &glnS,
			&loc.IsPrimary, &loc.IsActive, &created, &updated,
		); err2 != nil {
			return RetailerLocation{}, err
		}
		loc.Name = name
		loc.DeliveryAddress = addrS
		loc.PlaceID = placeS
		loc.Lat = latF
		loc.Lng = lngF
		loc.H3Cell = h3S
		loc.CountryCode = countryS
		loc.ReceivingWindowOpen = openS
		loc.ReceivingWindowClose = closeS
		loc.Gln = glnS
		loc.CreatedAt = created
		loc.UpdatedAt = updated
		return loc, nil
	}
	loc.Name = name
	if addr.Valid {
		loc.DeliveryAddress = addr.StringVal
	}
	if place.Valid {
		loc.PlaceID = place.StringVal
	}
	if lat.Valid {
		loc.Lat = lat.Float64
	}
	if lng.Valid {
		loc.Lng = lng.Float64
	}
	if h3.Valid {
		loc.H3Cell = h3.StringVal
	}
	if country.Valid {
		loc.CountryCode = country.StringVal
	}
	if open.Valid {
		loc.ReceivingWindowOpen = open.StringVal
	}
	if close.Valid {
		loc.ReceivingWindowClose = close.StringVal
	}
	if gln.Valid {
		loc.Gln = gln.StringVal
	}
	loc.CreatedAt = created
	loc.UpdatedAt = updated
	return loc, nil
}

func (s *Service) insertLocation(ctx context.Context, loc RetailerLocation) error {
	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.locationsByRetailer == nil {
			s.locationsByRetailer = map[string][]RetailerLocation{}
		}
		s.locationsByRetailer[loc.RetailerID] = append(s.locationsByRetailer[loc.RetailerID], loc)
		return nil
	}
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row := map[string]any{
			"LocationId": loc.LocationID,
			"RetailerId": loc.RetailerID,
			"Name":       loc.Name,
			"IsPrimary":  loc.IsPrimary,
			"IsActive":   loc.IsActive,
			"CreatedAt":  spanner.CommitTimestamp,
			"UpdatedAt":  spanner.CommitTimestamp,
		}
		if loc.DeliveryAddress != "" {
			row["DeliveryAddress"] = loc.DeliveryAddress
		}
		if loc.PlaceID != "" {
			row["PlaceId"] = loc.PlaceID
		}
		if loc.Lat != 0 || loc.Lng != 0 {
			row["Lat"] = loc.Lat
			row["Lng"] = loc.Lng
		}
		if loc.H3Cell != "" {
			row["H3Cell"] = loc.H3Cell
		}
		if loc.CountryCode != "" {
			row["CountryCode"] = loc.CountryCode
		}
		if loc.ReceivingWindowOpen != "" {
			row["ReceivingWindowOpen"] = loc.ReceivingWindowOpen
		}
		if loc.ReceivingWindowClose != "" {
			row["ReceivingWindowClose"] = loc.ReceivingWindowClose
		}
		if loc.Gln != "" {
			row["Gln"] = loc.Gln
		}
		if err := txn.BufferWrite([]*spanner.Mutation{spanner.InsertMap("RetailerLocations", row)}); err != nil {
			return err
		}
		buf := &spannerTxnBuffer{}
		payload := map[string]any{
			"type":        events.EventRetailerLocationCreated,
			"timestamp":   s.now().Format(time.RFC3339Nano),
			"retailer_id": loc.RetailerID,
			"location_id": loc.LocationID,
			"name":        loc.Name,
			"is_primary":  loc.IsPrimary,
		}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateRetailer, loc.RetailerID, events.TopicMain, payload); err != nil {
			return err
		}
		return buf.Flush(txn)
	})
	return err
}

func (s *Service) updateLocation(ctx context.Context, loc RetailerLocation) error {
	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		list := s.locationsByRetailer[loc.RetailerID]
		for i := range list {
			if list[i].LocationID == loc.LocationID {
				list[i] = loc
				s.locationsByRetailer[loc.RetailerID] = list
				return nil
			}
		}
		return errors.New("location_not_found")
	}
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		_, err := txn.Update(ctx, spanner.Statement{
			SQL: `UPDATE RetailerLocations SET
				Name = @name,
				DeliveryAddress = @addr,
				PlaceId = @place,
				Lat = @lat,
				Lng = @lng,
				H3Cell = @h3,
				CountryCode = @country,
				ReceivingWindowOpen = @open,
				ReceivingWindowClose = @close,
				Gln = @gln,
				IsActive = @active,
				IsPrimary = @primary,
				UpdatedAt = PENDING_COMMIT_TIMESTAMP()
				WHERE LocationId = @id`,
			Params: map[string]any{
				"name":    loc.Name,
				"addr":    nullableStr(loc.DeliveryAddress),
				"place":   nullableStr(loc.PlaceID),
				"lat":     loc.Lat,
				"lng":     loc.Lng,
				"h3":      nullableStr(loc.H3Cell),
				"country": nullableStr(loc.CountryCode),
				"open":    nullableStr(loc.ReceivingWindowOpen),
				"close":   nullableStr(loc.ReceivingWindowClose),
				"gln":     nullableStr(loc.Gln),
				"active":  loc.IsActive,
				"primary": loc.IsPrimary,
				"id":      loc.LocationID,
			},
		})
		if err != nil {
			return err
		}
		buf := &spannerTxnBuffer{}
		payload := map[string]any{
			"type":        events.EventRetailerLocationUpdated,
			"timestamp":   s.now().Format(time.RFC3339Nano),
			"retailer_id": loc.RetailerID,
			"location_id": loc.LocationID,
		}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateRetailer, loc.RetailerID, events.TopicMain, payload); err != nil {
			return err
		}
		return buf.Flush(txn)
	})
	return err
}

func nullableStr(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func (s *Service) setPrimaryLocation(ctx context.Context, retailerID, locationID string) error {
	loc, found, err := s.getLocation(ctx, locationID)
	if err != nil {
		return err
	}
	if !found || loc.RetailerID != retailerID {
		return errors.New("location_not_found")
	}
	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		list := s.locationsByRetailer[retailerID]
		for i := range list {
			list[i].IsPrimary = list[i].LocationID == locationID
		}
		s.locationsByRetailer[retailerID] = list
		return s.mirrorPrimaryToRetailer(ctx, loc)
	}
	_, err = s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Clear other primaries then set this one.
		if _, err := txn.Update(ctx, spanner.Statement{
			SQL: `UPDATE RetailerLocations SET IsPrimary = FALSE, UpdatedAt = PENDING_COMMIT_TIMESTAMP()
				WHERE RetailerId = @rid AND IsPrimary = TRUE`,
			Params: map[string]any{"rid": retailerID},
		}); err != nil {
			return err
		}
		_, err := txn.Update(ctx, spanner.Statement{
			SQL: `UPDATE RetailerLocations SET IsPrimary = TRUE, IsActive = TRUE, UpdatedAt = PENDING_COMMIT_TIMESTAMP()
				WHERE LocationId = @id`,
			Params: map[string]any{"id": locationID},
		})
		return err
	})
	if err != nil {
		return err
	}
	loc.IsPrimary = true
	return s.mirrorPrimaryToRetailer(ctx, loc)
}

func (s *Service) mirrorPrimaryToRetailer(ctx context.Context, loc RetailerLocation) error {
	if s.repo == nil {
		return nil
	}
	ret, found, err := s.repo.GetRetailer(ctx, loc.RetailerID)
	if err != nil || !found {
		return err
	}
	ret.DeliveryAddress = loc.DeliveryAddress
	ret.PlaceID = loc.PlaceID
	ret.Lat = loc.Lat
	ret.Lng = loc.Lng
	ret.H3Cell = loc.H3Cell
	ret.CountryCode = loc.CountryCode
	ret.ReceivingWindowOpen = loc.ReceivingWindowOpen
	ret.ReceivingWindowClose = loc.ReceivingWindowClose
	ret.UpdatedAt = s.now()
	return s.repo.UpdateRetailer(ctx, ret, nil)
}

func (s *Service) listUserLocationIDs(ctx context.Context, userID string) ([]string, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, nil
	}
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		return append([]string(nil), s.userLocations[userID]...), nil
	}
	stmt := spanner.Statement{
		SQL:    `SELECT LocationId FROM RetailerUserLocations WHERE UserId = @uid`,
		Params: map[string]any{"uid": userID},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	var out []string
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var id string
		if err := row.Columns(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}

func (s *Service) replaceUserLocations(ctx context.Context, retailerID, userID string, locationIDs []string) error {
	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.userLocations == nil {
			s.userLocations = map[string][]string{}
		}
		s.userLocations[userID] = append([]string(nil), locationIDs...)
		return nil
	}
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Delete existing binds.
		if _, err := txn.Update(ctx, spanner.Statement{
			SQL:    `DELETE FROM RetailerUserLocations WHERE UserId = @uid`,
			Params: map[string]any{"uid": userID},
		}); err != nil {
			return err
		}
		var muts []*spanner.Mutation
		for _, id := range locationIDs {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			muts = append(muts, spanner.InsertMap("RetailerUserLocations", map[string]any{
				"UserId":     userID,
				"LocationId": id,
				"RetailerId": retailerID,
				"CreatedAt":  spanner.CommitTimestamp,
			}))
		}
		if len(muts) == 0 {
			return nil
		}
		return txn.BufferWrite(muts)
	})
	return err
}

func dtoFromLocation(loc RetailerLocation) LocationDTO {
	return LocationDTO{
		LocationID:           loc.LocationID,
		RetailerID:           loc.RetailerID,
		Name:                 loc.Name,
		DeliveryAddress:      loc.DeliveryAddress,
		PlaceID:              loc.PlaceID,
		Lat:                  loc.Lat,
		Lng:                  loc.Lng,
		H3Cell:               loc.H3Cell,
		CountryCode:          loc.CountryCode,
		ReceivingWindowOpen:  loc.ReceivingWindowOpen,
		ReceivingWindowClose: loc.ReceivingWindowClose,
		Gln:                  loc.Gln,
		IsPrimary:            loc.IsPrimary,
		IsActive:             loc.IsActive,
		CreatedAt:            formatTimeOpt(loc.CreatedAt),
		UpdatedAt:            formatTimeOpt(loc.UpdatedAt),
	}
}

func containsString(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}

// DeliverySnapshotFromActiveLocation projects active location onto profile-like fields for checkout bind.
func (s *Service) DeliverySnapshotFromActiveLocation(ctx context.Context, claims auth.Claims) (address string, lat, lng float64, h3, open, close string, locationID string, ok bool) {
	loc, found, err := s.ResolveActiveLocation(ctx, claims)
	if err != nil || !found {
		return "", 0, 0, "", "", "", "", false
	}
	return loc.DeliveryAddress, loc.Lat, loc.Lng, loc.H3Cell, loc.ReceivingWindowOpen, loc.ReceivingWindowClose, loc.LocationID, true
}
