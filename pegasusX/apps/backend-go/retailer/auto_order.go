package retailer

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// AutoOrderSettings is the retailer auto-order configuration DTO.
type AutoOrderSettings struct {
	GlobalEnabled      bool               `json:"global_enabled"`
	AnalyticsStartDate *string            `json:"analytics_start_date,omitempty"`
	HasAnyHistory      bool               `json:"has_any_history"`
	SupplierOverrides  []SupplierOverride `json:"supplier_overrides"`
	CategoryOverrides  []CategoryOverride `json:"category_overrides"`
	ProductOverrides   []ProductOverride  `json:"product_overrides"`
	VariantOverrides   []VariantOverride  `json:"variant_overrides"`
}

// SupplierOverride toggles auto-order for one supplier.
type SupplierOverride struct {
	SupplierID string `json:"supplier_id"`
	Enabled    bool   `json:"enabled"`
}

// CategoryOverride toggles auto-order for one category.
type CategoryOverride struct {
	CategoryID string `json:"category_id"`
	Enabled    bool   `json:"enabled"`
}

// ProductOverride toggles auto-order for one product.
type ProductOverride struct {
	ProductID string `json:"product_id"`
	Enabled   bool   `json:"enabled"`
}

// VariantOverride toggles auto-order for one SKU variant.
type VariantOverride struct {
	VariantID string `json:"variant_id"`
	Enabled   bool   `json:"enabled"`
}

func (s *Service) autoOrderStore() map[string]*AutoOrderSettings {
	if s.autoOrderByRetailer == nil {
		s.autoOrderByRetailer = make(map[string]*AutoOrderSettings)
	}
	return s.autoOrderByRetailer
}

func (s *Service) getAutoOrderSettings(retailerID string) AutoOrderSettings {
	s.autoOrderMu.RLock()
	defer s.autoOrderMu.RUnlock()
	if row, ok := s.autoOrderStore()[retailerID]; ok && row != nil {
		return cloneAutoOrderSettings(*row)
	}
	return AutoOrderSettings{
		SupplierOverrides: []SupplierOverride{},
		CategoryOverrides: []CategoryOverride{},
		ProductOverrides:  []ProductOverride{},
		VariantOverrides:  []VariantOverride{},
	}
}

func (s *Service) saveAutoOrderSettings(retailerID string, settings AutoOrderSettings) {
	s.autoOrderMu.Lock()
	defer s.autoOrderMu.Unlock()
	copied := cloneAutoOrderSettings(settings)
	s.autoOrderStore()[retailerID] = &copied
}

func cloneAutoOrderSettings(in AutoOrderSettings) AutoOrderSettings {
	raw, _ := json.Marshal(in)
	var out AutoOrderSettings
	_ = json.Unmarshal(raw, &out)
	return out
}

// HandleAutoOrderSettings serves GET /v1/retailer/settings/auto-order.
func (s *Service) HandleAutoOrderSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	retailerID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.loadAutoOrderDurable(r.Context(), retailerID))
}

// HandleAutoOrderPatch serves PATCH auto-order settings endpoints.
func (s *Service) HandleAutoOrderPatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	retailerID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}

	settings := s.loadAutoOrderDurable(r.Context(), retailerID)
	path := r.URL.Path

	switch {
	case strings.HasSuffix(path, "/global"):
		var req struct {
			GlobalAutoOrderEnabled bool  `json:"global_auto_order_enabled"`
			UseHistory             *bool `json:"use_history"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		settings.GlobalEnabled = req.GlobalAutoOrderEnabled
		if req.UseHistory != nil && *req.UseHistory {
			settings.HasAnyHistory = true
		}
	case strings.Contains(path, "/supplier/"):
		supplierID := strings.TrimSpace(chi.URLParam(r, "supplierID"))
		if supplierID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "supplier_id_required"})
			return
		}
		enabled, err := decodeScopedAutoOrder(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		settings.SupplierOverrides = upsertSupplierOverride(settings.SupplierOverrides, supplierID, enabled)
	case strings.Contains(path, "/category/"):
		categoryID := strings.TrimSpace(chi.URLParam(r, "categoryID"))
		if categoryID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "category_id_required"})
			return
		}
		enabled, err := decodeScopedAutoOrder(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		settings.CategoryOverrides = upsertCategoryOverride(settings.CategoryOverrides, categoryID, enabled)
	case strings.Contains(path, "/product/"):
		productID := strings.TrimSpace(chi.URLParam(r, "productID"))
		if productID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "product_id_required"})
			return
		}
		enabled, err := decodeScopedAutoOrder(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		settings.ProductOverrides = upsertProductOverride(settings.ProductOverrides, productID, enabled)
	case strings.Contains(path, "/variant/"):
		variantID := strings.TrimSpace(chi.URLParam(r, "variantID"))
		if variantID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "variant_id_required"})
			return
		}
		enabled, err := decodeScopedAutoOrder(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		settings.VariantOverrides = upsertVariantOverride(settings.VariantOverrides, variantID, enabled)
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown_auto_order_scope"})
		return
	}

	actor := ""
	if claims, ok := auth.FromContext(r.Context()); ok {
		actor = auth.ResolveRetailerUserID(claims)
	}
	if err := s.saveAutoOrderDurable(r.Context(), retailerID, actor, settings); err != nil {
		s.log.Warn("auto-order durable save failed", "err", err, "retailer_id", retailerID)
		// memory already updated inside saveAutoOrderDurable before Spanner write
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func decodeScopedAutoOrder(r *http.Request) (bool, error) {
	var req struct {
		AutoOrderEnabled *bool `json:"auto_order_enabled"`
		Enabled          *bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return false, errInvalidJSON("invalid_json")
	}
	if req.AutoOrderEnabled != nil {
		return *req.AutoOrderEnabled, nil
	}
	if req.Enabled != nil {
		return *req.Enabled, nil
	}
	return false, errInvalidJSON("enabled_required")
}

type invalidJSONError string

func (e invalidJSONError) Error() string { return string(e) }

func errInvalidJSON(msg string) error { return invalidJSONError(msg) }

func upsertSupplierOverride(rows []SupplierOverride, id string, enabled bool) []SupplierOverride {
	for i := range rows {
		if rows[i].SupplierID == id {
			rows[i].Enabled = enabled
			return rows
		}
	}
	return append(rows, SupplierOverride{SupplierID: id, Enabled: enabled})
}

func upsertCategoryOverride(rows []CategoryOverride, id string, enabled bool) []CategoryOverride {
	for i := range rows {
		if rows[i].CategoryID == id {
			rows[i].Enabled = enabled
			return rows
		}
	}
	return append(rows, CategoryOverride{CategoryID: id, Enabled: enabled})
}

func upsertProductOverride(rows []ProductOverride, id string, enabled bool) []ProductOverride {
	for i := range rows {
		if rows[i].ProductID == id {
			rows[i].Enabled = enabled
			return rows
		}
	}
	return append(rows, ProductOverride{ProductID: id, Enabled: enabled})
}

func upsertVariantOverride(rows []VariantOverride, id string, enabled bool) []VariantOverride {
	for i := range rows {
		if rows[i].VariantID == id {
			rows[i].Enabled = enabled
			return rows
		}
	}
	return append(rows, VariantOverride{VariantID: id, Enabled: enabled})
}
