package warehouse

import (
	"encoding/json"
	"log/slog"
	"net/http"
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
	IsActive    bool   `json:"is_active"`
	Environment string `json:"environment,omitempty"`
	Mode        string `json:"mode,omitempty"`
	MerchantID  string `json:"merchant_id,omitempty"`
	ServiceID   string `json:"service_id,omitempty"`
	LastUpdated string `json:"last_updated,omitempty"`
}

const opsPaymentConfigQuery = `SELECT ConfigId, GatewayName, MerchantId, ServiceId, IsActive, CreatedAt, UpdatedAt
	      FROM SupplierPaymentConfigs
	      WHERE SupplierId = @sid
	      ORDER BY GatewayName`

func buildOpsPaymentGatewayItem(configID string, gatewayName string, merchantID string, serviceID string, isActive bool, lastUpdated time.Time) PaymentGatewayItem {
	mode := ""
	if capability := vault.GetProviderCapability(gatewayName); capability != nil {
		mode = string(capability.OnboardingMode)
	}

	item := PaymentGatewayItem{
		ConfigID:    configID,
		GatewayID:   configID,
		GatewayName: gatewayName,
		Provider:    gatewayName,
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
			Params: map[string]interface{}{"sid": ops.SupplierID},
		}

		iter := spannerClient.Single().Query(r.Context(), stmt)
		defer iter.Stop()

		var gateways []PaymentGatewayItem
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
			var isActive bool
			var createdAt time.Time
			var updatedAt spanner.NullTime
			if err := row.Columns(&configID, &gatewayName, &merchantID, &serviceID,
				&isActive, &createdAt, &updatedAt); err != nil {
				slog.Error("warehouse.payment_config_row_parse_failed", "supplier_id", ops.SupplierID, "err", err)
				continue
			}
			lastUpdated := createdAt
			if updatedAt.Valid {
				lastUpdated = updatedAt.Time
			}
			gateways = append(gateways, buildOpsPaymentGatewayItem(
				configID,
				gatewayName,
				merchantID,
				serviceID.StringVal,
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
