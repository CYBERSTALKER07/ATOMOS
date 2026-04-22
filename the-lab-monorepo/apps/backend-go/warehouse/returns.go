package warehouse

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"backend-go/auth"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ─── Returns ──────────────────────────────────────────────────────────────────
// Warehouse-scoped returns / rejected items.

type ReturnItem struct {
	OrderID      string `json:"order_id"`
	SkuID        string `json:"sku_id"`
	ProductName  string `json:"product_name"`
	Quantity     int64  `json:"quantity"`
	RetailerName string `json:"retailer_name"`
	DriverName   string `json:"driver_name,omitempty"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
}

// HandleOpsReturns — GET for /v1/warehouse/ops/returns
func HandleOpsReturns(spannerClient *spanner.Client) http.HandlerFunc {
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
			SQL: `SELECT li.OrderId, li.SkuId, COALESCE(sp.Name, ''), li.Quantity,
			             COALESCE(ret.StoreName, ''), COALESCE(d.Name, ''),
			             li.Status, o.CreatedAt
			      FROM OrderLineItems li
			      JOIN Orders o ON li.OrderId = o.OrderId
			      LEFT JOIN SupplierProducts sp ON li.SkuId = sp.SkuId
			      LEFT JOIN Retailers ret ON o.RetailerId = ret.RetailerId
			      LEFT JOIN Drivers d ON o.DriverId = d.DriverId
			      WHERE o.SupplierId = @sid AND o.WarehouseId = @whId
			        AND li.Status IN ('REJECTED_DAMAGED','REJECTED_WRONG','RETURNED')
			      ORDER BY o.CreatedAt DESC
			      LIMIT 200`,
			Params: map[string]interface{}{"sid": ops.SupplierID, "whId": ops.WarehouseID},
		}

		iter := spannerClient.Single().Query(r.Context(), stmt)
		defer iter.Stop()

		var returns []ReturnItem
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[WH RETURNS] list error: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			var ri ReturnItem
			var createdAt time.Time
			if err := row.Columns(&ri.OrderID, &ri.SkuID, &ri.ProductName, &ri.Quantity,
				&ri.RetailerName, &ri.DriverName, &ri.Status, &createdAt); err != nil {
				log.Printf("[WH RETURNS] parse: %v", err)
				continue
			}
			ri.CreatedAt = createdAt.Format(time.RFC3339)
			returns = append(returns, ri)
		}
		if returns == nil {
			returns = []ReturnItem{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"returns": returns, "total": len(returns)})
	}
}
