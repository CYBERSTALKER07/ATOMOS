package warehouse

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"
	"backend-go/proximity"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ─── Orders ───────────────────────────────────────────────────────────────────
// Warehouse-scoped order viewing and state management.

type OrderItem struct {
	OrderID       string  `json:"order_id"`
	RetailerID    string  `json:"retailer_id"`
	RetailerName  string  `json:"retailer_name"`
	State         string  `json:"state"`
	TotalAmount   int64   `json:"total_amount"`
	ItemCount     int64   `json:"item_count"`
	DriverID      string  `json:"driver_id,omitempty"`
	DriverName    string  `json:"driver_name,omitempty"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at,omitempty"`
	TotalVolumeVU float64 `json:"total_volume_vu,omitempty"`
}

type OrderDetail struct {
	OrderItem
	LineItems   []LineItem `json:"line_items"`
	RetailerLat float64    `json:"retailer_lat,omitempty"`
	RetailerLng float64    `json:"retailer_lng,omitempty"`
	Notes       string     `json:"notes,omitempty"`
}

type LineItem struct {
	SkuID       string  `json:"sku_id"`
	ProductName string  `json:"product_name"`
	Quantity    int64   `json:"quantity"`
	UnitPrice   int64   `json:"unit_price"`
	Status      string  `json:"status"`
	VolumeVU    float64 `json:"volume_vu,omitempty"`
}

// HandleOpsOrders — GET for /v1/warehouse/ops/orders
func HandleOpsOrders(spannerClient *spanner.Client, readRouter proximity.ReadRouter) http.HandlerFunc {
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

		sql := `SELECT o.OrderId, o.RetailerId, COALESCE(rt.StoreName, ''),
		               o.State, COALESCE(o.TotalAmount, 0),
		               (SELECT COUNT(*) FROM OrderLineItems li WHERE li.OrderId = o.OrderId),
		               COALESCE(o.DriverId, ''), COALESCE(d.Name, ''),
		               o.CreatedAt, COALESCE(o.UpdatedAt, o.CreatedAt)
		        FROM Orders o
		        LEFT JOIN Retailers rt ON o.RetailerId = rt.RetailerId
		        LEFT JOIN Drivers d ON o.DriverId = d.DriverId
		        WHERE o.SupplierId = @sid AND o.WarehouseId = @whId`

		params := map[string]interface{}{"sid": ops.SupplierID, "whId": ops.WarehouseID}

		// State filter
		if state := r.URL.Query().Get("state"); state != "" {
			sql += " AND o.State = @filterState"
			params["filterState"] = strings.ToUpper(state)
		}
		// Date range
		if from := r.URL.Query().Get("from"); from != "" {
			if t, err := time.Parse("2006-01-02", from); err == nil {
				sql += " AND o.CreatedAt >= @fromDate"
				params["fromDate"] = t
			}
		}
		if to := r.URL.Query().Get("to"); to != "" {
			if t, err := time.Parse("2006-01-02", to); err == nil {
				sql += " AND o.CreatedAt < @toDate"
				params["toDate"] = t.Add(24 * time.Hour)
			}
		}

		sql += " ORDER BY o.CreatedAt DESC LIMIT 200"

		readClient := warehouseReadClient(r.Context(), spannerClient, readRouter, ops.WarehouseID)
		stmt := spanner.Statement{SQL: sql, Params: params}
		iter := readClient.Single().Query(r.Context(), stmt)
		defer iter.Stop()

		var orders []OrderItem
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[WH ORDERS] list error: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			var o OrderItem
			var createdAt, updatedAt time.Time
			if err := row.Columns(&o.OrderID, &o.RetailerID, &o.RetailerName,
				&o.State, &o.TotalAmount, &o.ItemCount,
				&o.DriverID, &o.DriverName, &createdAt, &updatedAt); err != nil {
				log.Printf("[WH ORDERS] parse: %v", err)
				continue
			}
			o.CreatedAt = createdAt.Format(time.RFC3339)
			o.UpdatedAt = updatedAt.Format(time.RFC3339)
			orders = append(orders, o)
		}
		if orders == nil {
			orders = []OrderItem{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"orders": orders, "total": len(orders)})
	}
}

// HandleOpsOrderDetail — GET /v1/warehouse/ops/orders/{id}
func HandleOpsOrderDetail(spannerClient *spanner.Client, readRouter proximity.ReadRouter) http.HandlerFunc {
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

		parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
		orderID := parts[len(parts)-1]
		if orderID == "" || orderID == "orders" {
			http.Error(w, "order_id required", http.StatusBadRequest)
			return
		}

		// Order header
		stmt := spanner.Statement{
			SQL: `SELECT o.OrderId, o.RetailerId, COALESCE(rt.StoreName, ''),
			             o.State, COALESCE(o.TotalAmount, 0),
			             (SELECT COUNT(*) FROM OrderLineItems li WHERE li.OrderId = o.OrderId),
			             COALESCE(o.DriverId, ''), COALESCE(d.Name, ''),
			             o.CreatedAt, COALESCE(o.UpdatedAt, o.CreatedAt),
			             COALESCE(rt.Latitude, 0), COALESCE(rt.Longitude, 0),
			             COALESCE(o.Notes, '')
			      FROM Orders o
			      LEFT JOIN Retailers rt ON o.RetailerId = rt.RetailerId
			      LEFT JOIN Drivers d ON o.DriverId = d.DriverId
			      WHERE o.OrderId = @oid AND o.SupplierId = @sid AND o.WarehouseId = @whId`,
			Params: map[string]interface{}{"oid": orderID, "sid": ops.SupplierID, "whId": ops.WarehouseID},
		}
		readClient := warehouseReadClient(r.Context(), spannerClient, readRouter, ops.WarehouseID)
		iter := readClient.Single().Query(r.Context(), stmt)
		defer iter.Stop()

		row, err := iter.Next()
		if err == iterator.Done {
			http.Error(w, `{"error":"order not found"}`, http.StatusNotFound)
			return
		}
		if err != nil {
			log.Printf("[WH ORDERS] detail error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		var o OrderDetail
		var createdAt, updatedAt time.Time
		if err := row.Columns(&o.OrderID, &o.RetailerID, &o.RetailerName,
			&o.State, &o.TotalAmount, &o.ItemCount,
			&o.DriverID, &o.DriverName, &createdAt, &updatedAt,
			&o.RetailerLat, &o.RetailerLng, &o.Notes); err != nil {
			log.Printf("[WH ORDERS] detail parse: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		o.CreatedAt = createdAt.Format(time.RFC3339)
		o.UpdatedAt = updatedAt.Format(time.RFC3339)

		// Line items
		liStmt := spanner.Statement{
			SQL: `SELECT li.SkuId, COALESCE(sp.Name, ''), li.Quantity,
			             COALESCE(li.UnitPrice, 0), COALESCE(li.Status, 'PENDING'),
			             COALESCE(sp.VolumeVU, 0)
			      FROM OrderLineItems li
			      LEFT JOIN SupplierProducts sp ON li.SkuId = sp.SkuId
			      WHERE li.OrderId = @oid`,
			Params: map[string]interface{}{"oid": orderID},
		}
		liIter := readClient.Single().Query(r.Context(), liStmt)
		defer liIter.Stop()
		for {
			liRow, err := liIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				break
			}
			var li LineItem
			if err := liRow.Columns(&li.SkuID, &li.ProductName, &li.Quantity,
				&li.UnitPrice, &li.Status, &li.VolumeVU); err == nil {
				o.LineItems = append(o.LineItems, li)
			}
		}
		if o.LineItems == nil {
			o.LineItems = []LineItem{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(o)
	}
}

func warehouseReadClient(ctx context.Context, primary *spanner.Client, readRouter proximity.ReadRouter, warehouseID string) *spanner.Client {
	if primary == nil {
		return nil
	}
	if readRouter == nil || warehouseID == "" {
		return primary
	}

	row, err := primary.Single().ReadRow(ctx, "Warehouses", spanner.Key{warehouseID}, []string{"Lat", "Lng"})
	if err != nil {
		return primary
	}

	var lat, lng spanner.NullFloat64
	if row.Columns(&lat, &lng) != nil || !lat.Valid || !lng.Valid {
		return primary
	}

	return proximity.ReadClientForRetailer(primary, readRouter, lat.Float64, lng.Float64)
}
