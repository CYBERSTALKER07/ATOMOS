package warehouse

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"
	"backend-go/vault"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ─── Payment Config (read-only view of supplier's payment gateways) ──────────

type PaymentGatewayItem struct {
	ConfigID    string `json:"config_id,omitempty"`
	GatewayID   string `json:"gateway_id"`
	GatewayName string `json:"gateway_name"`
	Provider    string `json:"provider"`
	ConfigScope string `json:"config_scope,omitempty"`
	WarehouseID string `json:"warehouse_id,omitempty"`
	IsActive    bool   `json:"is_active"`
	Environment string `json:"environment,omitempty"`
	Mode        string `json:"mode,omitempty"`
	MerchantID  string `json:"merchant_id,omitempty"`
	ServiceID   string `json:"service_id,omitempty"`
	LastUpdated string `json:"last_updated,omitempty"`
}

const opsPaymentConfigQuery = `SELECT ConfigId, GatewayName, MerchantId, ServiceId, IsActive, CreatedAt, UpdatedAt, WarehouseId
	      FROM SupplierPaymentConfigs
	      WHERE SupplierId = @sid
	        AND (WarehouseId = @wid OR WarehouseId IS NULL)
	      ORDER BY GatewayName,
	               CASE WHEN WarehouseId = @wid THEN 0 ELSE 1 END,
	               UpdatedAt DESC,
	               CreatedAt DESC`

func buildOpsPaymentGatewayItem(configID string, gatewayName string, merchantID string, serviceID string, warehouseID string, isActive bool, lastUpdated time.Time) PaymentGatewayItem {
	mode := ""
	if capability := vault.GetProviderCapability(gatewayName); capability != nil {
		mode = string(capability.OnboardingMode)
	}
	configScope := "SUPPLIER_DEFAULT"
	if strings.TrimSpace(warehouseID) != "" {
		configScope = "WAREHOUSE"
	}

	item := PaymentGatewayItem{
		ConfigID:    configID,
		GatewayID:   configID,
		GatewayName: gatewayName,
		Provider:    gatewayName,
		ConfigScope: configScope,
		WarehouseID: warehouseID,
		IsActive:    isActive,
		Mode:        mode,
		MerchantID:  merchantID,
		ServiceID:   serviceID,
	}
	if !lastUpdated.IsZero() {
		item.LastUpdated = lastUpdated.UTC().Format(time.RFC3339)
	}
	return item
}

// HandleOpsPaymentConfig — GET for /v1/warehouse/ops/payment-config
func HandleOpsPaymentConfig(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		ops := auth.GetWarehouseOps(r.Context())
		if ops == nil {
			http.Error(w, "Warehouse scope required", http.StatusForbidden)
			return
		}

		stmt := spanner.Statement{
			SQL:    opsPaymentConfigQuery,
			Params: map[string]interface{}{"sid": ops.SupplierID, "wid": ops.WarehouseID},
		}

		iter := spannerClient.Single().Query(r.Context(), stmt)
		defer iter.Stop()

		var gateways []PaymentGatewayItem
		seenGateway := make(map[string]struct{})
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				slog.Error("warehouse.payment_config_list_failed", "supplier_id", ops.SupplierID, "err", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			var configID string
			var gatewayName string
			var merchantID string
			var serviceID spanner.NullString
			var warehouseID spanner.NullString
			var isActive bool
			var createdAt time.Time
			var updatedAt spanner.NullTime
			if err := row.Columns(&configID, &gatewayName, &merchantID, &serviceID,
				&isActive, &createdAt, &updatedAt, &warehouseID); err != nil {
				slog.Error("warehouse.payment_config_row_parse_failed", "supplier_id", ops.SupplierID, "err", err)
				continue
			}
			if _, already := seenGateway[gatewayName]; already {
				continue
			}
			seenGateway[gatewayName] = struct{}{}
			lastUpdated := createdAt
			if updatedAt.Valid {
				lastUpdated = updatedAt.Time
			}
			gateways = append(gateways, buildOpsPaymentGatewayItem(
				configID,
				gatewayName,
				merchantID,
				serviceID.StringVal,
				warehouseID.StringVal,
				isActive,
				lastUpdated,
			))
		}
		if gateways == nil {
			gateways = []PaymentGatewayItem{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"gateways": gateways, "total": len(gateways)})
	}
}
