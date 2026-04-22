package warehouse

import (
	"encoding/json"
	"log"
	"net/http"

	"backend-go/auth"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ─── Payment Config (read-only view of supplier's payment gateways) ──────────

type PaymentGatewayItem struct {
	GatewayID   string `json:"gateway_id"`
	Provider    string `json:"provider"` // CLICK | PAYME | GLOBAL_PAY
	IsActive    bool   `json:"is_active"`
	Environment string `json:"environment"` // SANDBOX | PRODUCTION
	MerchantID  string `json:"merchant_id,omitempty"`
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
			SQL: `SELECT GatewayId, Provider, IsActive, Environment, COALESCE(MerchantId, '')
			      FROM SupplierPaymentGateways
			      WHERE SupplierId = @sid
			      ORDER BY Provider`,
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
				log.Printf("[WH PAYMENT] list error: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			var g PaymentGatewayItem
			if err := row.Columns(&g.GatewayID, &g.Provider, &g.IsActive,
				&g.Environment, &g.MerchantID); err != nil {
				log.Printf("[WH PAYMENT] parse: %v", err)
				continue
			}
			gateways = append(gateways, g)
		}
		if gateways == nil {
			gateways = []PaymentGatewayItem{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"gateways": gateways, "total": len(gateways)})
	}
}
