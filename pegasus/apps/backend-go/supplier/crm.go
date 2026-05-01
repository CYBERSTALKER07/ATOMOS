package supplier

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ── Supplier CRM: Retailer Intelligence ──

type CRMRetailer struct {
	RetailerID    string `json:"retailer_id"`
	RetailerName  string `json:"retailer_name"`
	Phone         string `json:"phone,omitempty"`
	Email         string `json:"email,omitempty"`
	Lifetime   int64  `json:"lifetime"`
	OrderCount    int64  `json:"order_count"`
	LastOrderDate string `json:"last_order_date,omitempty"`
	Status        string `json:"status"`
}

type CRMRetailerDetail struct {
	CRMRetailer
	Orders []CRMOrder `json:"orders"`
}

type CRMOrder struct {
	OrderID   string `json:"order_id"`
	State     string `json:"state"`
	Amount int64  `json:"amount"`
	ItemCount int64  `json:"item_count"`
	CreatedAt string `json:"created_at"`
}

// HandleCRMRetailers returns all retailers who have ordered from the authenticated supplier.
func HandleCRMRetailers(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil || claims.UserID == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		supplierID := claims.ResolveSupplierID()

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		sql := `SELECT 
					o.RetailerId as retailer_id,
					COALESCE(rt.Name, o.RetailerId) as retailer_name,
					COALESCE(rt.Phone, '') as phone,
					COALESCE(CAST(SUM(o.Amount) AS INT64), 0) as lifetime,
					COUNT(DISTINCT o.OrderId) as order_count,
					MAX(FORMAT_TIMESTAMP('%Y-%m-%d', o.CreatedAt)) as last_order_date,
					CASE WHEN MAX(o.CreatedAt) >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 30 DAY) THEN 'ACTIVE' ELSE 'INACTIVE' END as status
				FROM Orders o
				LEFT JOIN Retailers rt ON o.RetailerId = rt.RetailerId
				WHERE o.SupplierId = @supplierId
				  AND o.State IN ('COMPLETED', 'ARRIVED', 'IN_TRANSIT', 'LOADED', 'PENDING', 'ARRIVING', 'AWAITING_PAYMENT', 'PENDING_CASH_COLLECTION')`
		params := map[string]interface{}{
			"supplierId": supplierID,
		}

		// Apply warehouse scope if present
		if whID := auth.EffectiveWarehouseID(r.Context()); whID != "" {
			sql += " AND o.WarehouseId = @warehouseId"
			params["warehouseId"] = whID
		}

		sql += ` GROUP BY o.RetailerId, rt.Name, rt.Phone
				ORDER BY lifetime DESC`

		stmt := spanner.Statement{
			SQL:    sql,
			Params: params,
		}

		iter := client.Single().Query(ctx, stmt)
		defer iter.Stop()

		var retailers []CRMRetailer
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[CRM] query error: %v", err)
				http.Error(w, `{"error":"query_failed"}`, http.StatusInternalServerError)
				return
			}
			var cr CRMRetailer
			if err := row.Columns(&cr.RetailerID, &cr.RetailerName, &cr.Phone, &cr.Lifetime, &cr.OrderCount, &cr.LastOrderDate, &cr.Status); err != nil {
				log.Printf("[CRM] scan error: %v", err)
				continue
			}
			retailers = append(retailers, cr)
		}

		if retailers == nil {
			retailers = []CRMRetailer{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(retailers)
	}
}

// HandleCRMRetailerDetail returns a single retailer's full order history for this supplier.
// Route: GET /v1/supplier/crm/retailers/{retailerId}
func HandleCRMRetailerDetail(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil || claims.UserID == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		supplierID := claims.ResolveSupplierID()

		// Parse retailer ID from path: /v1/supplier/crm/retailers/{retailerId}
		path := strings.TrimPrefix(r.URL.Path, "/v1/supplier/crm/retailers/")
		if path == "" || strings.Contains(path, "/") {
			http.Error(w, `{"error":"retailer_id required"}`, http.StatusBadRequest)
			return
		}
		retailerID := path

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		// 1. Retailer summary
		summarySql := `SELECT 
					o.RetailerId,
					COALESCE(rt.Name, o.RetailerId) as retailer_name,
					COALESCE(rt.Phone, '') as phone,
					COALESCE(CAST(SUM(o.Amount) AS INT64), 0) as lifetime,
					COUNT(DISTINCT o.OrderId) as order_count,
					MAX(FORMAT_TIMESTAMP('%Y-%m-%d', o.CreatedAt)) as last_order_date,
					CASE WHEN MAX(o.CreatedAt) >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 30 DAY) THEN 'ACTIVE' ELSE 'INACTIVE' END as status
				FROM Orders o
				LEFT JOIN Retailers rt ON o.RetailerId = rt.RetailerId
				WHERE o.SupplierId = @supplierId
				  AND o.RetailerId = @retailerId`
		summaryParams := map[string]interface{}{
			"supplierId": supplierID,
			"retailerId": retailerID,
		}

		// Apply warehouse scope if present
		if whID := auth.EffectiveWarehouseID(r.Context()); whID != "" {
			summarySql += " AND o.WarehouseId = @warehouseId"
			summaryParams["warehouseId"] = whID
		}

		summarySql += ` GROUP BY o.RetailerId, rt.Name, rt.Phone`

		summaryStmt := spanner.Statement{
			SQL:    summarySql,
			Params: summaryParams,
		}

		summaryIter := client.Single().Query(ctx, summaryStmt)
		defer summaryIter.Stop()

		detail := CRMRetailerDetail{}
		row, err := summaryIter.Next()
		if err != nil {
			http.Error(w, `{"error":"retailer_not_found"}`, http.StatusNotFound)
			return
		}
		if err := row.Columns(&detail.RetailerID, &detail.RetailerName, &detail.Phone, &detail.Lifetime, &detail.OrderCount, &detail.LastOrderDate, &detail.Status); err != nil {
			http.Error(w, `{"error":"parse_failed"}`, http.StatusInternalServerError)
			return
		}

		// 2. Order history for this retailer (only orders containing this supplier's products)
		ordersSql := `SELECT
					o.OrderId,
					o.State,
					COALESCE(o.Amount, 0) as amount,
					(SELECT COUNT(*) FROM OrderLineItems oli WHERE oli.OrderId = o.OrderId) as item_count,
					FORMAT_TIMESTAMP('%Y-%m-%dT%H:%M:%SZ', o.CreatedAt) as created_at
				FROM Orders o
				WHERE o.RetailerId = @retailerId
				  AND o.SupplierId = @supplierId`
		ordersParams := map[string]interface{}{
			"retailerId": retailerID,
			"supplierId": supplierID,
		}

		// Apply warehouse scope if present
		if whID := auth.EffectiveWarehouseID(r.Context()); whID != "" {
			ordersSql += " AND o.WarehouseId = @warehouseId"
			ordersParams["warehouseId"] = whID
		}

		ordersSql += ` ORDER BY o.CreatedAt DESC
				LIMIT 50`

		ordersStmt := spanner.Statement{
			SQL:    ordersSql,
			Params: ordersParams,
		}

		ordersIter := client.Single().Query(ctx, ordersStmt)
		defer ordersIter.Stop()
		for {
			oRow, err := ordersIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				break
			}
			var co CRMOrder
			if err := oRow.Columns(&co.OrderID, &co.State, &co.Amount, &co.ItemCount, &co.CreatedAt); err != nil {
				continue
			}
			detail.Orders = append(detail.Orders, co)
		}

		if detail.Orders == nil {
			detail.Orders = []CRMOrder{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(detail)
	}
}
