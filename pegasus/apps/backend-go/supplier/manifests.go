package supplier

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"backend-go/auth"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ── WAREHOUSE PICKING MANIFESTS ─────────────────────────────────────────────
// Aggregated daily pick lists: groups all LOADED (approved) orders by SKU
// so warehouse staff can batch-pick efficiently. Supports JSON + CSV export.

type ManifestLine struct {
	SkuID       string `json:"sku_id"`
	ProductName string `json:"product_name"`
	TotalQty    int64  `json:"total_qty"`
	OrderCount  int64  `json:"order_count"`
}

type ManifestOrder struct {
	OrderID      string `json:"order_id"`
	RetailerName string `json:"retailer_name"`
	ItemCount    int64  `json:"item_count"`
	State        string `json:"state"`
	CreatedAt    string `json:"created_at"`
}

func HandleManifests(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		supplierId := claims.ResolveSupplierID()

		// Date parameter: default to today
		dateStr := r.URL.Query().Get("date")
		if dateStr == "" {
			dateStr = time.Now().Format("2006-01-02")
		}
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			http.Error(w, `{"error":"invalid date format, use YYYY-MM-DD"}`, http.StatusBadRequest)
			return
		}
		dayStart := date.Truncate(24 * time.Hour)
		dayEnd := dayStart.Add(24 * time.Hour)

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		// Aggregate line items from LOADED orders for this supplier+date
		sql := `SELECT li.SkuId, sp.Name, SUM(li.Quantity) as TotalQty, COUNT(DISTINCT li.OrderId) as OrderCount
		        FROM OrderLineItems li
		        JOIN Orders o ON li.OrderId = o.OrderId
		        JOIN SupplierProducts sp ON li.SkuId = sp.SkuId
		        WHERE o.SupplierId = @sid
		          AND o.State IN ('LOADED', 'EN_ROUTE', 'IN_TRANSIT')
		          AND o.CreatedAt >= @dayStart AND o.CreatedAt < @dayEnd`

		params := map[string]interface{}{
			"sid":      supplierId,
			"dayStart": dayStart,
			"dayEnd":   dayEnd,
		}

		// Apply warehouse scope if present
		if whID := auth.EffectiveWarehouseID(r.Context()); whID != "" {
			sql += " AND o.WarehouseId = @warehouseId"
			params["warehouseId"] = whID
		}

		sql += ` GROUP BY li.SkuId, sp.Name
		        ORDER BY TotalQty DESC`

		stmt := spanner.Statement{
			SQL:    sql,
			Params: params,
		}
		iter := client.Single().Query(ctx, stmt)
		defer iter.Stop()

		var lines []ManifestLine
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[MANIFESTS] Query error: %v", err)
				http.Error(w, `{"error":"query_failed"}`, http.StatusInternalServerError)
				return
			}
			var l ManifestLine
			if err := row.Columns(&l.SkuID, &l.ProductName, &l.TotalQty, &l.OrderCount); err != nil {
				log.Printf("[MANIFESTS] Row parse error: %v", err)
				http.Error(w, `{"error":"parse_failed"}`, http.StatusInternalServerError)
				return
			}
			lines = append(lines, l)
		}
		if lines == nil {
			lines = []ManifestLine{}
		}

		// CSV export when requested
		accept := r.Header.Get("Accept")
		if accept == "text/csv" || r.URL.Query().Get("format") == "csv" {
			w.Header().Set("Content-Type", "text/csv")
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=pick-list-%s.csv", dateStr))
			writer := csv.NewWriter(w)
			writer.Write([]string{"SKU ID", "Product Name", "Total Qty", "Order Count"})
			for _, l := range lines {
				writer.Write([]string{
					l.SkuID,
					l.ProductName,
					fmt.Sprintf("%d", l.TotalQty),
					fmt.Sprintf("%d", l.OrderCount),
				})
			}
			writer.Flush()
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"date":  dateStr,
			"lines": lines,
		})
	}
}

// HandleManifestOrders — GET /v1/supplier/manifests/orders: individual orders for manifest context
func HandleManifestOrders(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		supplierId := claims.ResolveSupplierID()

		dateStr := r.URL.Query().Get("date")
		if dateStr == "" {
			dateStr = time.Now().Format("2006-01-02")
		}
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			http.Error(w, `{"error":"invalid date format"}`, http.StatusBadRequest)
			return
		}
		dayStart := date.Truncate(24 * time.Hour)
		dayEnd := dayStart.Add(24 * time.Hour)

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		sql := `SELECT o.OrderId, COALESCE(ret.Name, 'Unknown'), o.State, o.CreatedAt,
		               (SELECT COUNT(*) FROM OrderLineItems li WHERE li.OrderId = o.OrderId)
		        FROM Orders o
		        LEFT JOIN Retailers ret ON o.RetailerId = ret.RetailerId
		        WHERE o.SupplierId = @sid
		          AND o.State IN ('LOADED', 'EN_ROUTE', 'IN_TRANSIT')
		          AND o.CreatedAt >= @dayStart AND o.CreatedAt < @dayEnd`

		params := map[string]interface{}{
			"sid":      supplierId,
			"dayStart": dayStart,
			"dayEnd":   dayEnd,
		}

		// Apply warehouse scope if present
		if whID := auth.EffectiveWarehouseID(r.Context()); whID != "" {
			sql += " AND o.WarehouseId = @warehouseId"
			params["warehouseId"] = whID
		}

		sql += " ORDER BY o.CreatedAt ASC"

		stmt := spanner.Statement{
			SQL:    sql,
			Params: params,
		}
		iter := client.Single().Query(ctx, stmt)
		defer iter.Stop()

		var orders []ManifestOrder
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[MANIFESTS] orders query error: %v", err)
				http.Error(w, `{"error":"query_failed"}`, http.StatusInternalServerError)
				return
			}
			var o ManifestOrder
			var createdAt spanner.NullTime
			if err := row.Columns(&o.OrderID, &o.RetailerName, &o.State, &createdAt, &o.ItemCount); err != nil {
				log.Printf("[MANIFESTS] orders parse error: %v", err)
				http.Error(w, `{"error":"parse_failed"}`, http.StatusInternalServerError)
				return
			}
			if createdAt.Valid {
				o.CreatedAt = createdAt.Time.Format(time.RFC3339)
			}
			orders = append(orders, o)
		}
		if orders == nil {
			orders = []ManifestOrder{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": orders})
	}
}
