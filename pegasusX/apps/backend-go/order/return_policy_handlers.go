package order

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

type supplierReturnPolicyDTO struct {
	DefaultWindowHours         int64  `json:"default_window_hours"`
	ConcealedDamageWindowHours *int64 `json:"concealed_damage_window_hours,omitempty"`
	RequirePhoto               *bool  `json:"require_photo,omitempty"`
	AllowExpiredClaims         *bool  `json:"allow_expired_claims,omitempty"`
	PolicySourceHint           string `json:"policy_source_hint,omitempty"`
}

type warehouseReturnPolicyDTO struct {
	SupplierID                string `json:"supplier_id"`
	ReverseDockSLAHours       *int64 `json:"reverse_dock_sla_hours,omitempty"`
	RetailerFileWindowHours   *int64 `json:"retailer_file_window_hours,omitempty"`
	CanOverrideRetailerWindow bool   `json:"can_override_retailer_window"`
}

// HandleSupplierReturnPolicy serves GET/PUT /v1/supplier/return-policy.
func (s *Service) HandleSupplierReturnPolicy(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.returnPolicies == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "return_policy_unavailable"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	supplierID := strings.TrimSpace(claims.SupplierID)
	if supplierID == "" {
		if sid, ok := auth.ResolveSupplierID(r.Context()); ok {
			supplierID = sid
		}
	}
	if supplierID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "supplier_id_required"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		p, found, err := s.returnPolicies.GetSupplierReturnPolicy(r.Context(), supplierID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_failed"})
			return
		}
		if !found {
			h, src := claimWindowHoursFromEnv()
			reqPhoto := true
			allowExp := false
			writeJSON(w, http.StatusOK, supplierReturnPolicyDTO{
				DefaultWindowHours: h,
				RequirePhoto:       &reqPhoto,
				AllowExpiredClaims: &allowExp,
				PolicySourceHint:   src,
			})
			return
		}
		reqPhoto := p.RequirePhoto
		allowExp := p.AllowExpiredClaims
		writeJSON(w, http.StatusOK, supplierReturnPolicyDTO{
			DefaultWindowHours:         p.DefaultWindowHours,
			ConcealedDamageWindowHours: p.ConcealedDamageWindowHours,
			RequirePhoto:               &reqPhoto,
			AllowExpiredClaims:         &allowExp,
			PolicySourceHint:           ClaimWindowSourceSupplier,
		})
	case http.MethodPut:
		var body supplierReturnPolicyDTO
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		if body.DefaultWindowHours < minClaimWindowHours || body.DefaultWindowHours > maxClaimWindowHours {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_window_hours"})
			return
		}
		requirePhoto := true
		if body.RequirePhoto != nil {
			requirePhoto = *body.RequirePhoto
		}
		allowExpired := false
		if body.AllowExpiredClaims != nil {
			allowExpired = *body.AllowExpiredClaims
		}
		p := SupplierReturnPolicy{
			SupplierID:                 supplierID,
			DefaultWindowHours:         body.DefaultWindowHours,
			ConcealedDamageWindowHours: body.ConcealedDamageWindowHours,
			RequirePhoto:               requirePhoto,
			AllowExpiredClaims:         allowExpired,
			UpdatedByUserID:            claims.Subject,
		}
		if err := s.returnPolicies.UpsertSupplierReturnPolicy(r.Context(), p); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "save_failed"})
			return
		}
		reqPhoto := p.RequirePhoto
		allowExp := p.AllowExpiredClaims
		writeJSON(w, http.StatusOK, supplierReturnPolicyDTO{
			DefaultWindowHours:         p.DefaultWindowHours,
			ConcealedDamageWindowHours: p.ConcealedDamageWindowHours,
			RequirePhoto:               &reqPhoto,
			AllowExpiredClaims:         &allowExp,
			PolicySourceHint:           ClaimWindowSourceSupplier,
		})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

// HandleWarehouseReturnPolicy serves GET/PUT /v1/warehouse/return-policy.
func (s *Service) HandleWarehouseReturnPolicy(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.returnPolicies == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "return_policy_unavailable"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	warehouseID := strings.TrimSpace(auth.EffectiveWarehouseID(r.Context()))
	if warehouseID == "" {
		warehouseID = strings.TrimSpace(claims.HomeNodeID)
	}
	if q := strings.TrimSpace(r.URL.Query().Get("warehouse_id")); q != "" {
		warehouseID = q
	}
	if warehouseID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_id_required"})
		return
	}
	supplierID := strings.TrimSpace(r.URL.Query().Get("supplier_id"))

	switch r.Method {
	case http.MethodGet:
		p, found, err := s.returnPolicies.GetWarehouseReturnPolicy(r.Context(), warehouseID, supplierID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_failed"})
			return
		}
		if !found {
			writeJSON(w, http.StatusOK, warehouseReturnPolicyDTO{
				SupplierID:                supplierID,
				CanOverrideRetailerWindow: false,
			})
			return
		}
		writeJSON(w, http.StatusOK, warehouseReturnPolicyDTO{
			SupplierID:                p.SupplierID,
			ReverseDockSLAHours:       p.ReverseDockSLAHours,
			RetailerFileWindowHours:   p.RetailerFileWindowHours,
			CanOverrideRetailerWindow: p.CanOverrideRetailerWindow,
		})
	case http.MethodPut:
		var body warehouseReturnPolicyDTO
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		if body.SupplierID != "" {
			supplierID = strings.TrimSpace(body.SupplierID)
		}
		if body.RetailerFileWindowHours != nil {
			h := *body.RetailerFileWindowHours
			if h < minClaimWindowHours || h > maxClaimWindowHours {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_window_hours"})
				return
			}
		}
		p := WarehouseReturnPolicy{
			WarehouseID:               warehouseID,
			SupplierID:                supplierID,
			ReverseDockSLAHours:       body.ReverseDockSLAHours,
			RetailerFileWindowHours:   body.RetailerFileWindowHours,
			CanOverrideRetailerWindow: body.CanOverrideRetailerWindow,
			UpdatedByUserID:           claims.Subject,
		}
		if err := s.returnPolicies.UpsertWarehouseReturnPolicy(r.Context(), p); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "save_failed"})
			return
		}
		writeJSON(w, http.StatusOK, warehouseReturnPolicyDTO{
			SupplierID:                p.SupplierID,
			ReverseDockSLAHours:       p.ReverseDockSLAHours,
			RetailerFileWindowHours:   p.RetailerFileWindowHours,
			CanOverrideRetailerWindow: p.CanOverrideRetailerWindow,
		})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}
