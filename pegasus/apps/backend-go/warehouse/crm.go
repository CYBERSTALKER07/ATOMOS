package warehouse

import (
	"encoding/json"
	"log"
	"net/http"

	"backend-go/auth"
	"backend-go/spannerx"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ─── CRM — Retailer Relationships ────────────────────────────────────────────
// Warehouse-scoped view of retailers who order from this warehouse.

type CRMRetailer struct {
	RetailerID    string `json:"retailer_id"`
	StoreName     string `json:"store_name"`
	ContactName   string `json:"contact_name,omitempty"`
	Phone         string `json:"phone,omitempty"`
	TotalOrders   int64  `json:"total_orders"`
	TotalRevenue  int64  `json:"total_revenue"`
	LastOrderDate string `json:"last_order_date,omitempty"`
	Address       string `json:"address,omitempty"`
}

// HandleOpsCRM — GET for /v1/warehouse/ops/crm
func HandleOpsCRM(spannerClient *spanner.Client) http.HandlerFunc {
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
			SQL: `SELECT rt.RetailerId, COALESCE(rt.StoreName, ''),
			             COALESCE(rt.ContactName, ''), COALESCE(rt.Phone, ''),
			             COUNT(o.OrderId) as total_orders,
			             COALESCE(SUM(CASE WHEN o.State = 'COMPLETED' THEN o.TotalAmount ELSE 0 END), 0) as revenue,
			             MAX(o.CreatedAt) as last_order,
			             COALESCE(rt.Address, '')
			      FROM Orders o
			      JOIN Retailers rt ON o.RetailerId = rt.RetailerId
			      WHERE o.SupplierId = @sid AND o.WarehouseId = @whId
			        AND o.State IN ('PENDING','LOADED','IN_TRANSIT','ARRIVED','COMPLETED','EN_ROUTE')
			      GROUP BY rt.RetailerId, rt.StoreName, rt.ContactName, rt.Phone, rt.Address
			      ORDER BY total_orders DESC
			      LIMIT 200`,
			Params: map[string]interface{}{"sid": ops.SupplierID, "whId": ops.WarehouseID},
		}

		iter := spannerx.StaleQuery(r.Context(), spannerClient, stmt)
		defer iter.Stop()

		var retailers []CRMRetailer
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[WH CRM] list error: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			var c CRMRetailer
			var lastOrder spanner.NullTime
			if err := row.Columns(&c.RetailerID, &c.StoreName, &c.ContactName,
				&c.Phone, &c.TotalOrders, &c.TotalRevenue, &lastOrder, &c.Address); err != nil {
				log.Printf("[WH CRM] parse: %v", err)
				continue
			}
			if lastOrder.Valid {
				c.LastOrderDate = lastOrder.Time.Format("2006-01-02")
			}
			retailers = append(retailers, c)
		}
		if retailers == nil {
			retailers = []CRMRetailer{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"retailers": retailers, "total": len(retailers)})
	}
}
