package supplier

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

type supplierDispatchScopeBody struct {
	WarehouseID string `json:"warehouse_id"`
}

// resolveSupplierDispatchWarehouseID resolves warehouse scope for supplier dispatch.
// Precedence: JWT ops scope, query warehouse_id, JSON body warehouse_id (POST only).
func resolveSupplierDispatchWarehouseID(r *http.Request) string {
	if id := strings.TrimSpace(auth.EffectiveWarehouseID(r.Context())); id != "" {
		return id
	}
	if id := strings.TrimSpace(r.URL.Query().Get("warehouse_id")); id != "" {
		return id
	}
	if r.Body == nil {
		return ""
	}
	var body supplierDispatchScopeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return ""
	}
	return strings.TrimSpace(body.WarehouseID)
}
