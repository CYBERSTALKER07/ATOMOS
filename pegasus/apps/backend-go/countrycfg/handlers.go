package countrycfg

import (
	"encoding/json"
	"net/http"
	"strings"

	"backend-go/auth"
)

// HandleCountryConfigs exposes admin CRUD for country-level operational config.
// GET  /v1/admin/country-configs        -> list all active configs
// PUT  /v1/admin/country-configs        -> upsert config from JSON body
func HandleCountryConfigs(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			configs, err := svc.ListAllConfigs(r.Context())
			if err != nil {
				http.Error(w, `{"error":"failed_to_list_country_configs"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "SUCCESS",
				"data":   configs,
			})

		case http.MethodPut:
			var cfg CountryConfig
			if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
				http.Error(w, `{"error":"invalid_json"}`, http.StatusBadRequest)
				return
			}
			if strings.TrimSpace(cfg.CountryCode) == "" {
				http.Error(w, `{"error":"country_code_required"}`, http.StatusBadRequest)
				return
			}
			if err := svc.UpsertConfig(r.Context(), &cfg); err != nil {
				http.Error(w, `{"error":"failed_to_upsert_country_config"}`, http.StatusInternalServerError)
				return
			}
			updated := svc.GetConfig(r.Context(), cfg.CountryCode)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "SUCCESS",
				"data":   updated,
			})

		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

// HandleCountryConfigByCode returns a single country config by URL suffix.
// GET /v1/admin/country-configs/{code}
func HandleCountryConfigByCode(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		code := strings.TrimPrefix(r.URL.Path, "/v1/admin/country-configs/")
		code = strings.TrimSpace(code)
		if code == "" || strings.Contains(code, "/") {
			http.Error(w, `{"error":"invalid_country_code"}`, http.StatusBadRequest)
			return
		}

		cfg := svc.GetConfig(r.Context(), strings.ToUpper(code))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "SUCCESS",
			"data":   cfg,
		})
	}
}

// HandleSupplierCountryOverrides manages the authenticated supplier's country overrides.
// GET /v1/supplier/country-overrides         -> list all overrides set by this supplier
// PUT /v1/supplier/country-overrides         -> upsert an override (body: SupplierOverride JSON)
func HandleSupplierCountryOverrides(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if !ok || claims == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		supplierID := claims.ResolveSupplierID()

		switch r.Method {
		case http.MethodGet:
			overrides, err := svc.ListSupplierOverrides(r.Context(), supplierID)
			if err != nil {
				http.Error(w, `{"error":"failed_to_list_overrides"}`, http.StatusInternalServerError)
				return
			}
			if overrides == nil {
				overrides = []*SupplierOverride{}
			}
			// Enrich with base country config so the frontend can show effective values.
			type enriched struct {
				Override  *SupplierOverride `json:"override"`
				Effective *CountryConfig    `json:"effective"`
			}
			out := make([]enriched, 0, len(overrides))
			for _, o := range overrides {
				eff := svc.GetEffectiveConfig(r.Context(), o.SupplierId, o.CountryCode)
				out = append(out, enriched{Override: o, Effective: eff})
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "SUCCESS",
				"data":   out,
			})

		case http.MethodPut:
			var o SupplierOverride
			if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
				http.Error(w, `{"error":"invalid_json"}`, http.StatusBadRequest)
				return
			}
			// Force the supplier ID to match the authenticated caller — prevent IDOR.
			o.SupplierId = supplierID
			if strings.TrimSpace(o.CountryCode) == "" {
				http.Error(w, `{"error":"country_code_required"}`, http.StatusBadRequest)
				return
			}
			o.CountryCode = strings.ToUpper(strings.TrimSpace(o.CountryCode))
			if err := svc.UpsertSupplierOverride(r.Context(), &o); err != nil {
				http.Error(w, `{"error":"failed_to_upsert_override"}`, http.StatusInternalServerError)
				return
			}
			eff := svc.GetEffectiveConfig(r.Context(), supplierID, o.CountryCode)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":    "SUCCESS",
				"override":  o,
				"effective": eff,
			})

		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

// HandleSupplierCountryOverrideByCode manages a single supplier override by country code.
// GET    /v1/supplier/country-overrides/{code}  -> get override + effective config
// DELETE /v1/supplier/country-overrides/{code}  -> remove override (revert to platform defaults)
func HandleSupplierCountryOverrideByCode(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if !ok || claims == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		supplierID := claims.ResolveSupplierID()

		code := strings.TrimPrefix(r.URL.Path, "/v1/supplier/country-overrides/")
		code = strings.ToUpper(strings.TrimSpace(code))
		if code == "" || strings.Contains(code, "/") {
			http.Error(w, `{"error":"invalid_country_code"}`, http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodGet:
			o, _ := svc.GetSupplierOverride(r.Context(), supplierID, code)
			eff := svc.GetEffectiveConfig(r.Context(), supplierID, code)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":    "SUCCESS",
				"override":  o,
				"effective": eff,
			})

		case http.MethodDelete:
			if err := svc.DeleteSupplierOverride(r.Context(), supplierID, code); err != nil {
				http.Error(w, `{"error":"failed_to_delete_override"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"status": "DELETED"})

		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}
