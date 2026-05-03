package retailerroutes

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"google.golang.org/api/iterator"

	"backend-go/auth"
	"backend-go/cache"
	"backend-go/order"
)

var errRetailerOrderAccessForbidden = errors.New(`{"error":"forbidden: cannot access another retailer's orders"}`)

type mobileLineItem struct {
	ID          string `json:"id"`
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	VariantID   string `json:"variant_id,omitempty"`
	VariantSize string `json:"variant_size,omitempty"`
	Quantity    int64  `json:"quantity"`
	UnitPrice   int64  `json:"unit_price"`
	TotalPrice  int64  `json:"total_price"`
}

type mobileOrder struct {
	OrderID           string           `json:"order_id"`
	RetailerID        string           `json:"retailer_id"`
	SupplierID        string           `json:"supplier_id,omitempty"`
	SupplierName      string           `json:"supplier_name,omitempty"`
	State             string           `json:"state"`
	Amount            int64            `json:"amount"`
	OrderSource       string           `json:"order_source,omitempty"`
	CreatedAt         string           `json:"created_at"`
	UpdatedAt         string           `json:"updated_at,omitempty"`
	EstimatedDelivery string           `json:"estimated_delivery,omitempty"`
	DeliveryToken     string           `json:"delivery_token,omitempty"`
	Items             []mobileLineItem `json:"items"`
}

type trackingRow struct {
	OrderID       string
	SupplierID    string
	DriverID      string
	State         string
	Amount        int64
	DeliveryToken string
	OrderSource   string
	CreatedAt     time.Time
	WarehouseID   string
}

type trackingItem struct {
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	Quantity    int64  `json:"quantity"`
	UnitPrice   int64  `json:"unit_price"`
	LineTotal   int64  `json:"line_total"`
}

type trackingOrder struct {
	OrderID         string         `json:"order_id"`
	SupplierID      string         `json:"supplier_id"`
	SupplierName    string         `json:"supplier_name"`
	WarehouseID     string         `json:"warehouse_id,omitempty"`
	WarehouseName   string         `json:"warehouse_name,omitempty"`
	DriverID        string         `json:"driver_id,omitempty"`
	State           string         `json:"state"`
	TotalAmount     int64          `json:"total_amount"`
	OrderSource     string         `json:"order_source,omitempty"`
	DriverLatitude  *float64       `json:"driver_latitude"`
	DriverLongitude *float64       `json:"driver_longitude"`
	IsApproaching   bool           `json:"is_approaching"`
	DeliveryToken   string         `json:"delivery_token,omitempty"`
	CreatedAt       string         `json:"created_at"`
	Items           []trackingItem `json:"items"`
}

func handleRetailerOrders(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		retailerID := chi.URLParam(r, "retailerID")
		if retailerID == "" {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		if err := authorizeRetailerOrders(r, retailerID); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}

		orders, err := d.Order.ListOrders(r.Context(), "", "", retailerID)
		if err != nil {
			log.Printf("Failed to list orders for retailer %s: %v", retailerID, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mapMobileOrders(orders)); err != nil {
			log.Printf("Failed to write orders response payload: %v", err)
		}
	}
}

func handleRetailerTracking(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		rows, supplierIDs, driverIDs, warehouseIDs, orderIDs, err := loadTrackingRows(r, d.Spanner, claims.UserID)
		if err != nil {
			log.Printf("[TRACKING] Failed to query active orders for retailer %s: %v", claims.UserID, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if len(rows) == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"orders":[]}`))
			return
		}

		supplierNames := loadNameMap(r, d.Spanner, "Suppliers", "SupplierId", "Name", "sids", supplierIDs)
		warehouseNames := loadNameMap(r, d.Spanner, "Warehouses", "WarehouseId", "Name", "wids", warehouseIDs)
		driverPositions := loadDriverPositions(r, driverIDs)
		approachingSet := loadApproachingOrders(r, orderIDs)
		orderItems := loadTrackingItems(r, d.Spanner, orderIDs)

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"orders": buildTrackingOrders(rows, supplierNames, warehouseNames, driverPositions, approachingSet, orderItems),
		}); err != nil {
			log.Printf("[TRACKING] Failed to write response: %v", err)
		}
	}
}

func authorizeRetailerOrders(r *http.Request, retailerID string) error {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if ok && claims != nil && claims.Role == "RETAILER" && claims.UserID != retailerID {
		return errRetailerOrderAccessForbidden
	}
	return nil
}

func mapMobileOrders(orders []order.Order) []mobileOrder {
	result := make([]mobileOrder, 0, len(orders))
	for _, current := range orders {
		mapped := mobileOrder{
			OrderID:      current.ID,
			RetailerID:   current.RetailerID,
			SupplierID:   current.SupplierID,
			SupplierName: current.SupplierName,
			State:        current.State,
			Amount:       current.Amount,
			CreatedAt:    current.CreatedAt.Format(time.RFC3339),
			Items:        make([]mobileLineItem, 0, len(current.Items)),
		}
		if current.OrderSource.Valid {
			mapped.OrderSource = current.OrderSource.StringVal
		}
		if current.DeliverBefore.Valid {
			mapped.EstimatedDelivery = current.DeliverBefore.Time.Format(time.RFC3339)
		}
		if current.DeliveryToken.Valid {
			mapped.DeliveryToken = current.DeliveryToken.StringVal
		}
		for _, item := range current.Items {
			mapped.Items = append(mapped.Items, mobileLineItem{
				ID:          item.LineItemID,
				ProductID:   item.SkuID,
				ProductName: item.SkuName,
				Quantity:    item.Quantity,
				UnitPrice:   item.UnitPrice,
				TotalPrice:  item.Quantity * item.UnitPrice,
			})
		}
		result = append(result, mapped)
	}
	return result
}

func loadTrackingRows(r *http.Request, client *spanner.Client, retailerID string) ([]trackingRow, []string, []string, []string, []string, error) {
	stmt := spanner.Statement{
		SQL: `SELECT OrderId, SupplierId, DriverId, State, Amount, DeliveryToken, OrderSource, CreatedAt,
		             COALESCE(WarehouseId, '') AS WarehouseId
		      FROM Orders
		      WHERE RetailerId = @retailerId
		        AND State IN ('PENDING', 'LOADED', 'DISPATCHED', 'IN_TRANSIT', 'ARRIVING', 'ARRIVED')
		      ORDER BY CreatedAt DESC
		      LIMIT 50`,
		Params: map[string]interface{}{"retailerId": retailerID},
	}

	iter := client.Single().Query(r.Context(), stmt)
	defer iter.Stop()

	rows := make([]trackingRow, 0, 16)
	supplierSet := map[string]bool{}
	driverSet := map[string]bool{}
	warehouseSet := map[string]bool{}
	orderIDs := make([]string, 0, 16)

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return rows, keysOf(supplierSet), keysOf(driverSet), keysOf(warehouseSet), orderIDs, nil
		}
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}

		current, err := decodeTrackingRow(row)
		if err != nil {
			log.Printf("[TRACKING] Column parse failed: %v", err)
			continue
		}
		rows = append(rows, current)
		orderIDs = append(orderIDs, current.OrderID)
		if current.SupplierID != "" {
			supplierSet[current.SupplierID] = true
		}
		if current.DriverID != "" {
			driverSet[current.DriverID] = true
		}
		if current.WarehouseID != "" {
			warehouseSet[current.WarehouseID] = true
		}
	}
}

func decodeTrackingRow(row *spanner.Row) (trackingRow, error) {
	var orderID, stateValue string
	var supplierID, driverID, deliveryToken, orderSource spanner.NullString
	var amount spanner.NullInt64
	var createdAt spanner.NullTime
	var warehouseID string

	if err := row.Columns(&orderID, &supplierID, &driverID, &stateValue, &amount, &deliveryToken, &orderSource, &createdAt, &warehouseID); err != nil {
		return trackingRow{}, err
	}

	return trackingRow{
		OrderID:       orderID,
		SupplierID:    supplierID.StringVal,
		DriverID:      driverID.StringVal,
		State:         stateValue,
		Amount:        amount.Int64,
		DeliveryToken: deliveryToken.StringVal,
		OrderSource:   orderSource.StringVal,
		CreatedAt:     createdAt.Time,
		WarehouseID:   warehouseID,
	}, nil
}

func loadNameMap(r *http.Request, client *spanner.Client, table string, idColumn string, nameColumn string, paramName string, ids []string) map[string]string {
	names := map[string]string{}
	if len(ids) == 0 {
		return names
	}

	stmt := spanner.Statement{
		SQL:    "SELECT " + idColumn + ", " + nameColumn + " FROM " + table + " WHERE " + idColumn + " IN UNNEST(@" + paramName + ")",
		Params: map[string]interface{}{paramName: ids},
	}
	iter := client.Single().Query(r.Context(), stmt)
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return names
		}
		if err != nil {
			return names
		}
		var id, name string
		if err := row.Columns(&id, &name); err == nil {
			names[id] = name
		}
	}
}

func loadDriverPositions(r *http.Request, driverIDs []string) map[string][2]float64 {
	positions := map[string][2]float64{}
	if cache.Client == nil || len(driverIDs) == 0 {
		return positions
	}

	members := make([]string, len(driverIDs))
	for index, driverID := range driverIDs {
		members[index] = cache.DriverGeoMember(driverID)
	}
	geoPositions, err := cache.Client.GeoPos(r.Context(), cache.KeyGeoProximity, members...).Result()
	if err != nil {
		return positions
	}

	for index, driverID := range driverIDs {
		if index < len(geoPositions) && geoPositions[index] != nil {
			positions[driverID] = [2]float64{geoPositions[index].Latitude, geoPositions[index].Longitude}
		}
	}
	return positions
}

func loadApproachingOrders(r *http.Request, orderIDs []string) map[string]bool {
	approaching := map[string]bool{}
	if cache.Client == nil {
		return approaching
	}
	for _, orderID := range orderIDs {
		isApproaching, err := cache.Client.SIsMember(r.Context(), cache.KeyArrivingSet, orderID).Result()
		if err == nil && isApproaching {
			approaching[orderID] = true
		}
	}
	return approaching
}

func loadTrackingItems(r *http.Request, client *spanner.Client, orderIDs []string) map[string][]trackingItem {
	itemsByOrder := map[string][]trackingItem{}
	if len(orderIDs) == 0 {
		return itemsByOrder
	}

	stmt := spanner.Statement{
		SQL: `SELECT li.OrderId, li.SkuId, COALESCE(sp.Name, li.SkuId) AS SkuName, li.Quantity, li.UnitPrice
		      FROM OrderLineItems li
		      LEFT JOIN SupplierProducts sp ON li.SkuId = sp.SkuId
		      WHERE li.OrderId IN UNNEST(@orderIds)`,
		Params: map[string]interface{}{"orderIds": orderIDs},
	}
	iter := client.Single().Query(r.Context(), stmt)
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return itemsByOrder
		}
		if err != nil {
			return itemsByOrder
		}
		var orderID, skuID, skuName string
		var quantity, unitPrice int64
		if err := row.Columns(&orderID, &skuID, &skuName, &quantity, &unitPrice); err == nil {
			itemsByOrder[orderID] = append(itemsByOrder[orderID], trackingItem{
				ProductID:   skuID,
				ProductName: skuName,
				Quantity:    quantity,
				UnitPrice:   unitPrice,
				LineTotal:   quantity * unitPrice,
			})
		}
	}
}

func buildTrackingOrders(rows []trackingRow, supplierNames map[string]string, warehouseNames map[string]string, driverPositions map[string][2]float64, approachingSet map[string]bool, orderItems map[string][]trackingItem) []trackingOrder {
	orders := make([]trackingOrder, 0, len(rows))
	for _, current := range rows {
		mapped := trackingOrder{
			OrderID:       current.OrderID,
			SupplierID:    current.SupplierID,
			SupplierName:  supplierNames[current.SupplierID],
			WarehouseID:   current.WarehouseID,
			WarehouseName: warehouseNames[current.WarehouseID],
			DriverID:      current.DriverID,
			State:         current.State,
			TotalAmount:   current.Amount,
			OrderSource:   current.OrderSource,
			IsApproaching: approachingSet[current.OrderID],
			DeliveryToken: current.DeliveryToken,
			CreatedAt:     current.CreatedAt.Format(time.RFC3339),
			Items:         orderItems[current.OrderID],
		}
		if position, ok := driverPositions[current.DriverID]; ok {
			mapped.DriverLatitude = &position[0]
			mapped.DriverLongitude = &position[1]
		}
		if mapped.Items == nil {
			mapped.Items = []trackingItem{}
		}
		orders = append(orders, mapped)
	}
	return orders
}

func keysOf(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}
