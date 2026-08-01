package retailer

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// strings used for location filtering in me handler

// MeResponse is GET /v1/retailer/me (documented shape; handler encodes map for home_surface).
type MeResponse struct {
	UserID        string       `json:"user_id"`
	RetailerID    string       `json:"retailer_id"`
	RetailerOrgID string       `json:"retailer_org_id"`
	RetailerRole  string       `json:"retailer_role"`
	Name          string       `json:"name"`
	Phone         string       `json:"phone,omitempty"`
	IsOwner       bool         `json:"is_owner"`
	IsConfigured  bool         `json:"is_configured"`
	Permissions   []string     `json:"permissions"`
	Capabilities  []string     `json:"capabilities"`
	Packs         []PackStatus `json:"packs"`
	LocationIDs   []string     `json:"location_ids,omitempty"`
	HomeSurface   string       `json:"home_surface,omitempty"`
}

// PackStatus is a catalog pack with enable state for this org.
type PackStatus struct {
	PackMeta
	Enabled bool           `json:"enabled"`
	Config  map[string]any `json:"config,omitempty"`
}

// HandleMe serves GET /v1/retailer/me — identity, packs, permissions.
func (s *Service) HandleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleRetailer {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	orgID := auth.ResolveRetailerOrgID(claims)
	userID := auth.ResolveRetailerUserID(claims)
	role := auth.EffectiveRetailerRole(claims)

	ret, found, err := s.repo.GetRetailer(r.Context(), orgID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_failed"})
		return
	}
	name := ""
	phone := ""
	if found {
		name = coalesceRetailerName(ret.Name)
		phone = ret.Phone
	}

	// Prefer durable user row when present.
	isOwner := role == "OWNER"
	if s.spannerClient != nil {
		if u, okU, errU := s.findRetailerUserByID(r.Context(), userID); errU == nil && okU {
			name = coalesceRetailerName(u.Name)
			if name == "" {
				name = coalesceRetailerName(ret.Name)
			}
			phone = u.Phone
			role = strings.ToUpper(u.RetailerRole)
			isOwner = u.IsOwner || role == "OWNER"
			orgID = u.RetailerID
		}
	}

	enabled, err := s.LoadEnabledPacks(r.Context(), orgID)
	if err != nil {
		s.log.Warn("load packs failed", "err", err, "retailer_id", orgID)
		enabled = EnabledSet{}.WithCORE()
	}

	packs := make([]PackStatus, 0, len(PackCatalog))
	for _, meta := range PackCatalog {
		st := PackStatus{PackMeta: meta, Enabled: enabled.Has(meta.ID)}
		if cfg, err := s.LoadPackConfig(r.Context(), orgID, meta.ID); err == nil && len(cfg) > 0 {
			st.Config = cfg
		}
		packs = append(packs, st)
	}

	perms := auth.ListRetailerPerms(auth.Claims{Role: auth.RoleRetailer, RetailerRole: role})
	sort.Strings(perms)

	// Phase 2 locations
	if _, err := s.EnsurePrimaryLocation(r.Context(), orgID); err != nil && s.log != nil {
		s.log.Warn("me ensure primary location", "err", err)
	}
	locIDs := claims.LocationIDs
	if len(locIDs) == 0 {
		locIDs, _ = s.listUserLocationIDs(r.Context(), userID)
	}
	activeLoc := strings.TrimSpace(claims.ActiveLocationID)
	var locations []LocationDTO
	if all, err := s.listLocations(r.Context(), orgID); err == nil {
		for _, loc := range all {
			if !loc.IsActive {
				continue
			}
			// Scope non-owner staff with binds.
			if role != "OWNER" && role != "ADMIN" && len(locIDs) > 0 && !containsString(locIDs, loc.LocationID) {
				continue
			}
			locations = append(locations, dtoFromLocation(loc))
			if activeLoc == "" && loc.IsPrimary {
				activeLoc = loc.LocationID
			}
		}
		if activeLoc == "" && len(locations) > 0 {
			activeLoc = locations[0].LocationID
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user_id":            userID,
		"retailer_id":        orgID,
		"retailer_org_id":    orgID,
		"retailer_role":      role,
		"name":               name,
		"phone":              phone,
		"is_owner":           isOwner,
		"is_configured":      found && retailerProfileConfigured(ret),
		"permissions":        perms,
		"capabilities":       enabled.List(),
		"packs":              packs,
		"location_ids":       locIDs,
		"active_location_id": activeLoc,
		"locations":          locations,
		"home_surface":       roleHomeSurface(role),
	})
}

// HandleCapabilitiesList serves GET /v1/retailer/capabilities.
func (s *Service) HandleCapabilitiesList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	enabled, err := s.LoadEnabledPacks(r.Context(), orgID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_failed"})
		return
	}
	packs := make([]PackStatus, 0, len(PackCatalog))
	for _, meta := range PackCatalog {
		st := PackStatus{PackMeta: meta, Enabled: enabled.Has(meta.ID)}
		if cfg, err := s.LoadPackConfig(r.Context(), orgID, meta.ID); err == nil && len(cfg) > 0 {
			st.Config = cfg
		}
		packs = append(packs, st)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"retailer_id":  orgID,
		"capabilities": enabled.List(),
		"packs":        packs,
	})
}

// HandleCapabilityEnable serves POST /v1/retailer/capabilities/{packID}/enable
// body: { "accept_soft_deps": bool, "enable_deps": bool, "config": {} }
func (s *Service) HandleCapabilityEnable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || !auth.HasRetailerPerm(claims, auth.PermCapManage) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "permission": auth.PermCapManage})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	packID := NormalizePackID(chi.URLParam(r, "packID"))
	if !KnownPack(packID) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown_pack"})
		return
	}

	var req struct {
		AcceptSoftDeps bool           `json:"accept_soft_deps"`
		EnableDeps     bool           `json:"enable_deps"`
		Config         map[string]any `json:"config"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Config == nil {
		req.Config = map[string]any{}
	}

	enabled, err := s.LoadEnabledPacks(r.Context(), orgID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_failed"})
		return
	}

	eval := EvaluateEnable(enabled, packID, req.AcceptSoftDeps)
	if eval.Status == "BLOCKED" {
		if req.EnableDeps && len(eval.MissingHard) > 0 {
			// enable hard deps + pack
			for _, dep := range eval.WouldEnable {
				if dep == PackCORE {
					continue
				}
				if err := s.SetPackEnabled(r.Context(), orgID, dep, auth.ResolveRetailerUserID(claims), true, map[string]any{}); err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "enable_failed", "detail": err.Error()})
					return
				}
			}
			// re-load
			enabled, _ = s.LoadEnabledPacks(r.Context(), orgID)
			writeJSON(w, http.StatusOK, map[string]any{
				"status":       "OK",
				"enabled":      true,
				"pack_id":      packID,
				"capabilities": enabled.List(),
				"evaluation":   eval,
			})
			return
		}
		writeJSON(w, http.StatusConflict, eval)
		return
	}
	if eval.Status == "WARN" && !req.AcceptSoftDeps {
		writeJSON(w, http.StatusConflict, eval)
		return
	}

	if err := s.SetPackEnabled(r.Context(), orgID, packID, auth.ResolveRetailerUserID(claims), true, req.Config); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "enable_failed", "detail": err.Error()})
		return
	}
	enabled, _ = s.LoadEnabledPacks(r.Context(), orgID)
	writeJSON(w, http.StatusOK, map[string]any{
		"status":       "OK",
		"enabled":      true,
		"pack_id":      packID,
		"capabilities": enabled.List(),
		"evaluation":   eval,
	})
}

// HandleCapabilityDisable serves POST /v1/retailer/capabilities/{packID}/disable
func (s *Service) HandleCapabilityDisable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || !auth.HasRetailerPerm(claims, auth.PermCapManage) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "permission": auth.PermCapManage})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	packID := NormalizePackID(chi.URLParam(r, "packID"))
	enabled, err := s.LoadEnabledPacks(r.Context(), orgID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_failed"})
		return
	}
	eval := EvaluateDisable(enabled, packID)
	if eval.Status == "BLOCKED" {
		writeJSON(w, http.StatusConflict, eval)
		return
	}
	if err := s.SetPackEnabled(r.Context(), orgID, packID, auth.ResolveRetailerUserID(claims), false, map[string]any{}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "disable_failed", "detail": err.Error()})
		return
	}
	enabled, _ = s.LoadEnabledPacks(r.Context(), orgID)
	writeJSON(w, http.StatusOK, map[string]any{
		"status":       "OK",
		"enabled":      false,
		"pack_id":      packID,
		"capabilities": enabled.List(),
		"evaluation":   eval,
	})
}
