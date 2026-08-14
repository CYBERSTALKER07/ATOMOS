package auth

import (
	"encoding/json"
	"net/http"
	"strings"
)

// HandleSession GET /v1/auth/session — any authenticated role.
// Returns identity + resolved MarketPack. Checkout still does not apply the pack
// until CheckoutReadsThis is true (GS-M).
func HandleSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAuthJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := FromContext(r.Context())
	if !ok || strings.TrimSpace(claims.Subject) == "" {
		writeAuthJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	code := EffectiveMarketCode(claims)
	pack, ok := ResolveMarketPack(code)
	if !ok {
		writeAuthJSON(w, http.StatusOK, map[string]any{
			"subject":     claims.Subject,
			"role":        claims.Role,
			"supplier_id": claims.SupplierID,
			"market_code": code,
			"home_cell":   EffectiveHomeCell(claims),
			"pack":        nil,
			"pack_error":  "unknown_market",
		})
		return
	}
	writeAuthJSON(w, http.StatusOK, map[string]any{
		"subject":           claims.Subject,
		"role":              claims.Role,
		"supplier_id":       claims.SupplierID,
		"retailer_org_id":   claims.RetailerOrgID,
		"home_node_type":    claims.HomeNodeType,
		"home_node_id":      claims.HomeNodeID,
		"is_registered":     claims.IsRegistered,
		"is_configured":     claims.IsConfigured,
		"market_code":       pack.Code,
		"home_cell":         EffectiveHomeCell(claims),
		"pack":              pack,
		"checkout_reads_this": pack.CheckoutReadsThis,
	})
}

// HandleListMarketPacks GET /v1/platform/market-packs — public catalog, no secrets.
func HandleListMarketPacks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAuthJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeAuthJSON(w, http.StatusOK, map[string]any{
		"items":           ListMarketPacks(),
		"default_code":    DefaultMarketCodeFromEnv(),
		"default_cell":    DefaultHomeCellFromEnv(),
		"checkout_note":   "checkout_reads_this is false until GS-M; UZ is the only shipped pack",
	})
}

// HandleGetMarketPack GET /v1/platform/market-packs/{code}
func HandleGetMarketPack(code string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeAuthJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		pack, ok := ResolveMarketPack(code)
		if !ok {
			writeAuthJSON(w, http.StatusNotFound, map[string]string{
				"error": "country_not_supported",
				"code":  NormalizeMarketCode(code),
			})
			return
		}
		writeAuthJSON(w, http.StatusOK, pack)
	}
}

func writeAuthJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
