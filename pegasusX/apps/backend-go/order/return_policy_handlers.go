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

// callerSupplierID is the tenant for this request. Warehouse-ops scope wins,
// then JWT claims, then tenant context. Query and body never supply it.
func callerSupplierID(r *http.Request) string {
	if ops := auth.GetWarehouseOps(r.Context()); ops != nil && strings.TrimSpace(ops.SupplierID) != "" {
		return strings.TrimSpace(ops.SupplierID)
	}
	if sid, ok := auth.ResolveSupplierID(r.Context()); ok {
		return sid
	}
	if t, ok := auth.TenantFromContext(r.Context()); ok {
		return strings.TrimSpace(t.SupplierID)
	}
	return ""
}

// pinnedWarehouseID is the warehouse the token is allowed to operate.
// A query warehouse_id may name a warehouse only when the token has no pin
// (supplier ADMIN without a home node). A pin plus a different query id is rejected.
func pinnedWarehouseID(r *http.Request, claims auth.Claims) (string, string) {
	pinned := strings.TrimSpace(auth.EffectiveWarehouseID(r.Context()))
	if pinned == "" {
		pinned = strings.TrimSpace(claims.HomeNodeID)
	}
	queryID := strings.TrimSpace(r.URL.Query().Get("warehouse_id"))
	if pinned != "" && queryID != "" && queryID != pinned {
		return "", "warehouse_scope_forbidden"
	}
	if pinned != "" {
		return pinned, ""
	}
	if queryID == "" {
		return "", "warehouse_id_required"
	}
	return queryID, ""
}

// HandleWarehouseReturnPolicy serves GET/PUT /v1/warehouse/return-policy.
// SupplierId always comes from the token. Body and query supplier_id values
// that name another tenant are rejected.
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
	warehouseID, whCode := pinnedWarehouseID(r, claims)
	if whCode != "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": whCode})
		return
	}
	supplierID := callerSupplierID(r)
	if supplierID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "supplier_id_required"})
		return
	}
	if q := strings.TrimSpace(r.URL.Query().Get("supplier_id")); q != "" && q != supplierID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "supplier_scope_forbidden"})
		return
	}

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
		if got := strings.TrimSpace(body.SupplierID); got != "" && got != supplierID {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "supplier_scope_forbidden"})
			return
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
