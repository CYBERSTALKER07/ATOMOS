package order

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"backend-go/auth"
	"backend-go/cache"
	"backend-go/hotspot"
	kafkaEvents "backend-go/kafka"
	"backend-go/outbox"
	"backend-go/payment"
	"backend-go/proximity"
	"backend-go/spannerx"
	"backend-go/telemetry"
	"backend-go/vault"

	"cloud.google.com/go/spanner"
	"github.com/segmentio/kafka-go"
	"google.golang.org/api/iterator"
)

var (
	ErrKafkaProducerTimeout = errors.New("kafka producer timeout")
	ErrAlreadyProcessed     = errors.New("order is already processed")
)

// topicLogisticsEvents is the canonical destination topic for domain events
// emitted from the order package through the transactional outbox.
const topicLogisticsEvents = "lab-logistics-events"

// ErrStateConflict is returned when a state mutation is blocked by a concurrent
// write (e.g., Supplier dispatches while Retailer cancels). HTTP handlers should
// surface this as 409 Conflict so the frontend can revert optimistic UI.
type ErrStateConflict struct {
	OrderID      string
	CurrentState string
	AttemptedOp  string
}

func (e *ErrStateConflict) Error() string {
	return fmt.Sprintf("state conflict on %s: current state is %s, cannot %s", e.OrderID, e.CurrentState, e.AttemptedOp)
}

// ErrVersionConflict is returned when an optimistic concurrency check fails —
// another transaction mutated the order between client read and write.
type ErrVersionConflict struct {
	OrderID         string
	ExpectedVersion int64
	ActualVersion   int64
}

func (e *ErrVersionConflict) Error() string {
	return fmt.Sprintf("version conflict on %s: expected v%d, found v%d — refresh required", e.OrderID, e.ExpectedVersion, e.ActualVersion)
}

// ErrFreezeLock is returned when an order is locked for physical dispatch and
// cannot be mutated until the lock expires.
type ErrFreezeLock struct {
	OrderID     string
	LockedUntil time.Time
}

func (e *ErrFreezeLock) Error() string {
	return fmt.Sprintf("order %s is locked for physical dispatch until %s", e.OrderID, e.LockedUntil.Format(time.RFC3339))
}

type Location struct {
	Latitude  float64
	Longitude float64
}

type Order struct {
	ID             string             `json:"order_id"`
	RetailerID     string             `json:"retailer_id"`
	SupplierID     string             `json:"supplier_id,omitempty"`
	SupplierName   string             `json:"supplier_name,omitempty"`
	Amount         int64              `json:"amount"`
	Currency       string             `json:"currency"`
	PaymentGateway string             `json:"payment_gateway"`
	State          string             `json:"state"`
	RouteID        spanner.NullString `json:"route_id"`
	OrderSource    spanner.NullString `json:"order_source"`
	AutoConfirmAt  spanner.NullTime   `json:"auto_confirm_at"`
	DeliverBefore  spanner.NullTime   `json:"deliver_before"`
	DeliveryToken  spanner.NullString `json:"delivery_token"`
	CreatedAt      time.Time          `json:"created_at"`
	Items          []LineItem         `json:"items,omitempty"`
}

type LineItem struct {
	LineItemID string `json:"line_item_id"`
	OrderID    string `json:"order_id"`
	SkuID      string `json:"sku_id"`
	SkuName    string `json:"sku_name,omitempty"`
	Quantity   int64  `json:"quantity"`
	UnitPrice  int64  `json:"unit_price"`
	Currency   string `json:"currency"`
	Status     string `json:"status"`
}

// OrderCompletedEvent, FleetDispatchedEvent, PayloadSealedEvent — canonical
// definitions live in kafka/events.go. Use kafkaEvents.* types.

type PayloadSealRequest struct {
	OrderID         string `json:"order_id"`
	TerminalID      string `json:"terminal_id"`
	ManifestCleared bool   `json:"manifest_cleared"`
}

type DispatchFleetRequest struct {
	OrderIds []string `json:"order_ids"`
	RouteId  string   `json:"route_id"`
}

// ReassignOrderRequest moves one or more orders from their current truck to a new one.
type ReassignOrderRequest struct {
	OrderIds   []string `json:"order_ids"`
	NewRouteId string   `json:"new_route_id"`
}

// ReassignConflict provides structured reasons when reassignment is blocked.
type ReassignConflict struct {
	OrderID string `json:"order_id"`
	Reason  string `json:"reason"`
}

// OrderReassignedEvent — canonical definition lives in kafka/events.go.

// CapacityInfo holds volume data for a truck used in validation responses.
type CapacityInfo struct {
	RouteID          string  `json:"route_id"`
	MaxVolumeVU      float64 `json:"max_volume_vu"`
	UsedVolumeVU     float64 `json:"used_volume_vu"`
	FreeVolumeVU     float64 `json:"free_volume_vu"`
	PendingReturnsVU float64 `json:"pending_returns_vu"`
}

type CreateOrderRequest struct {
	RetailerID     string  `json:"retailer_id"`
	Amount         int64   `json:"amount"`
	Currency       string  `json:"currency"`
	PaymentGateway string  `json:"payment_gateway"`
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	// AI Empathy Engine fields (optional)
	OrderSource   string `json:"order_source"`    // defaults to "MANUAL"
	State         string `json:"state"`           // defaults to "PENDING"
	AutoConfirmAt string `json:"auto_confirm_at"` // ISO8601, optional
	// Temporal Matrix (optional)
	DeliverBefore         string `json:"deliver_before"`          // ISO8601, optional hard deadline
	RequestedDeliveryDate string `json:"requested_delivery_date"` // ISO8601, optional → SCHEDULED state
	// Phase VII: Freight Surcharge
	DeliveryFee            int64  `json:"delivery_fee"`             // minor currency, computed by pricing engine
	FulfillmentWarehouseId string `json:"fulfillment_warehouse_id"` // which warehouse ships this order
}

type ActiveMission struct {
	OrderID            string  `json:"order_id"`
	State              string  `json:"state"`
	TargetLat          float64 `json:"target_lat"`
	TargetLng          float64 `json:"target_lng"`
	Amount             int64   `json:"amount"`
	Currency           string  `json:"currency"`
	Gateway            string  `json:"gateway"`
	RouteID            string  `json:"route_id"`
	SupplierID         string  `json:"supplier_id"`
	EstimatedArrivalAt *string `json:"estimated_arrival_at,omitempty"`
}

type OrderService struct {
	Producer     *kafka.Writer
	Cache        *cache.Cache
	Client       *spanner.Client
	ReadRouter   proximity.ReadRouter           // Optional H3-aware read router for geo-scoped reads.
	Vault        *vault.Service                 // Per-supplier credential vault (nil = ENV-only fallback)
	SessionSvc   *payment.SessionService        // Payment session engine (nil = legacy mode)
	CardTokenSvc *payment.CardTokenService      // Saved card token CRUD (nil = tokenization disabled)
	DirectClient *payment.GlobalPayDirectClient // Global Pay Payments Service Public (nil = disabled)
	FeeBP        int64                          // Platform fee in basis points (0 = zero-fee era)
}

// feeBasisPoints returns the configured platform fee. Zero-safe: defaults to 0.
func (s *OrderService) feeBasisPoints() int64 {
	return s.FeeBP
}

// getDistance calculates Haversine distance between two coordinates in meters
func getDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371e3 // Earth radius in meters
	// ... (implementation truncated for brevity if needed, but keeping full is better)
	toRad := func(deg float64) float64 { return deg * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLon := toRad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

// parseWKTPoint parses a Spanner GEOGRAPHY string into a Location
func parseWKTPoint(wkt string) (Location, error) {
	if !strings.HasPrefix(wkt, "POINT(") || !strings.HasSuffix(wkt, ")") {
		return Location{}, fmt.Errorf("invalid WKT format: %s", wkt)
	}
	content := wkt[6 : len(wkt)-1]
	parts := strings.Fields(content)
	if len(parts) != 2 {
		return Location{}, fmt.Errorf("invalid coordinate count in WKT: %s", wkt)
	}
	lon, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return Location{}, fmt.Errorf("invalid longitude: %w", err)
	}
	lat, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return Location{}, fmt.Errorf("invalid latitude: %w", err)
	}
	return Location{Latitude: lat, Longitude: lon}, nil
}

// GenerateSecureToken creates a random 16-character hex string for the QR code
func GenerateSecureToken() string {
	bytes := make([]byte, 8) // 8 bytes = 16 hex characters
	if _, err := rand.Read(bytes); err != nil {
		return "FALLBACK_TOK_0000" // Fallback in case of crypto failure
	}
	return hex.EncodeToString(bytes)
}

// Generate a unique ID for predictions
func GeneratePredictionId() string {
	return hotspot.NewPredictionID()
}

func (s *OrderService) SavePrediction(ctx context.Context, retailerId string, amount int64, triggerDate string, warehouseId string) error {
	predId := GeneratePredictionId()

	parsedTrigger, err := time.Parse(time.RFC1123, triggerDate)
	if err != nil {
		// Try RFC3339 as well, since both formats are common
		parsedTrigger, err = time.Parse(time.RFC3339, triggerDate)
		if err != nil {
			return fmt.Errorf("invalid trigger date format: %v", err)
		}
	}

	m := spanner.Insert("AIPredictions",
		[]string{"PredictionId", "RetailerId", "PredictedAmount", "TriggerDate", "TriggerShard", "Status", "WarehouseId"},
		[]interface{}{
			predId,
			retailerId,
			amount,
			spanner.NullTime{Time: parsedTrigger, Valid: true},
			hotspot.ShardForKey(predId),
			"WAITING", // Starts in the waiting state
			warehouseId,
		},
	)

	_, err = s.Client.Apply(ctx, []*spanner.Mutation{m})
	if err == nil {
		fmt.Printf("[PREDICTION SAVED] %s will trigger on %s for %d\n", retailerId, triggerDate, amount)
	}
	return err
}

// PredictionItem represents a single SKU forecast within a prediction.
type PredictionItem struct {
	SkuID    string `json:"sku_id"`
	Quantity int64  `json:"quantity"`
	Price    int64  `json:"price"`
}

// SavePredictionWithItems saves a prediction with SKU-level line items.
// Status is DORMANT when auto-order is globally off, WAITING when on.
func (s *OrderService) SavePredictionWithItems(ctx context.Context, retailerId string, amount int64, triggerDate string, items []PredictionItem, status string, warehouseId string) error {
	predId := GeneratePredictionId()

	parsedTrigger, err := time.Parse(time.RFC1123, triggerDate)
	if err != nil {
		parsedTrigger, err = time.Parse(time.RFC3339, triggerDate)
		if err != nil {
			return fmt.Errorf("invalid trigger date format: %v", err)
		}
	}

	if status == "" {
		status = "WAITING"
	}

	_, err = s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := []*spanner.Mutation{
			spanner.Insert("AIPredictions",
				[]string{"PredictionId", "RetailerId", "PredictedAmount", "TriggerDate", "TriggerShard", "Status", "CreatedAt", "WarehouseId"},
				[]interface{}{predId, retailerId, amount, spanner.NullTime{Time: parsedTrigger, Valid: true}, hotspot.ShardForKey(predId), status, spanner.CommitTimestamp, warehouseId},
			),
		}

		for _, item := range items {
			itemId := hotspot.NewPredictionItemID()
			mutations = append(mutations, spanner.Insert("AIPredictionItems",
				[]string{"PredictionId", "PredictionItemId", "SkuId", "PredictedQuantity", "UnitPrice", "CreatedAt"},
				[]interface{}{predId, itemId, item.SkuID, item.Quantity, item.Price, spanner.CommitTimestamp},
			))
		}

		return txn.BufferWrite(mutations)
	})

	if err == nil {
		fmt.Printf("[PREDICTION SAVED] %s will trigger on %s for %d (%d items, status=%s)\n",
			retailerId, triggerDate, amount, len(items), status)
	}
	return err
}

// HandleLineItemHistory returns past order line items for a retailer,
// used by the AI Worker to compute SKU-level purchase frequency.
// GET /v1/orders/line-items/history?retailer_id=X&since=ISO8601
func HandleLineItemHistory(client *spanner.Client, readRouter proximity.ReadRouter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		retailerID := r.URL.Query().Get("retailer_id")
		if retailerID == "" {
			http.Error(w, `{"error":"retailer_id required"}`, http.StatusBadRequest)
			return
		}

		sinceParam := r.URL.Query().Get("since")
		warehouseParam := r.URL.Query().Get("warehouse_id")

		sql := `SELECT oli.SkuId, oli.Quantity, oli.UnitPrice, o.CreatedAt,
		               COALESCE(sp.MinimumOrderQty, 1), COALESCE(sp.StepSize, 1),
		               COALESCE(sp.ProductId, ''), COALESCE(sp.CategoryId, ''), COALESCE(sp.SupplierId, ''),
		               COALESCE(o.WarehouseId, '')
		        FROM OrderLineItems oli
		        JOIN Orders o ON o.OrderId = oli.OrderId
		        LEFT JOIN SupplierProducts sp ON sp.SkuId = oli.SkuId
		        WHERE o.RetailerId = @rid AND o.State = 'COMPLETED'`
		params := map[string]interface{}{"rid": retailerID}

		if warehouseParam != "" {
			sql += ` AND o.WarehouseId = @wid`
			params["wid"] = warehouseParam
		}

		if sinceParam != "" {
			parsed, parseErr := time.Parse(time.RFC3339, sinceParam)
			if parseErr == nil {
				sql += ` AND o.CreatedAt >= @since`
				params["since"] = parsed
			}
		}

		sql += ` ORDER BY o.CreatedAt DESC LIMIT 500`

		readClient := client
		if readRouter != nil {
			row, err := client.Single().ReadRow(r.Context(), "Retailers", spanner.Key{retailerID}, []string{"Latitude", "Longitude"})
			if err == nil {
				var lat, lng spanner.NullFloat64
				if row.Columns(&lat, &lng) == nil && lat.Valid && lng.Valid {
					readClient = proximity.ReadClientForRetailer(client, readRouter, lat.Float64, lng.Float64)
				}
			}
		}

		stmt := spanner.Statement{SQL: sql, Params: params}
		iter := spannerx.StaleQuery(r.Context(), readClient, stmt)
		defer iter.Stop()

		type HistoryItem struct {
			SkuID           string `json:"skuId"`
			ProductID       string `json:"productId"`
			CategoryID      string `json:"categoryId"`
			SupplierID      string `json:"supplierId"`
			WarehouseId     string `json:"warehouseId"`
			Quantity        int64  `json:"quantity"`
			UnitPrice       int64  `json:"unitPrice"`
			OrderDate       string `json:"orderDate"`
			MinimumOrderQty int64  `json:"minimumOrderQty"`
			StepSize        int64  `json:"stepSize"`
		}

		var items []HistoryItem
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			var h HistoryItem
			var createdAt time.Time
			if row.Columns(&h.SkuID, &h.Quantity, &h.UnitPrice, &createdAt, &h.MinimumOrderQty, &h.StepSize, &h.ProductID, &h.CategoryID, &h.SupplierID, &h.WarehouseId) == nil {
				h.OrderDate = createdAt.Format(time.RFC3339)
				items = append(items, h)
			}
		}

		if items == nil {
			items = []HistoryItem{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, req CreateOrderRequest) (string, error) {
	orderID := hotspot.NewOrderID()
	scheduleShard := hotspot.ShardForKey(orderID)

	// Format to WKT POINT(lon lat)
	wkt := fmt.Sprintf("POINT(%f %f)", req.Longitude, req.Latitude)

	_, err := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// 1. Dynamic state — AI orders arrive as PENDING_REVIEW
		initialState := "PENDING"
		if req.State != "" {
			initialState = req.State
		}

		// 2. OrderSource — default to MANUAL for human-placed orders
		orderSource := spanner.NullString{StringVal: "MANUAL", Valid: true}
		if req.OrderSource != "" {
			orderSource = spanner.NullString{StringVal: req.OrderSource, Valid: true}
		}

		// 3. AutoConfirmAt — parse ISO8601 timestamp if provided
		var autoConfirm spanner.NullTime
		if req.AutoConfirmAt != "" {
			parsedTime, parseErr := time.Parse(time.RFC3339, req.AutoConfirmAt)
			if parseErr == nil {
				autoConfirm = spanner.NullTime{Time: parsedTime, Valid: true}
			} else {
				fmt.Printf("[WARNING] Failed to parse AutoConfirmAt timestamp: %v\n", parseErr)
			}
		}

		// 4. DeliverBefore — parse ISO8601 hard deadline if provided
		var deliverBefore spanner.NullTime
		if req.DeliverBefore != "" {
			parsedTime, parseErr := time.Parse(time.RFC3339, req.DeliverBefore)
			if parseErr == nil {
				deliverBefore = spanner.NullTime{Time: parsedTime, Valid: true}
			} else {
				fmt.Printf("[WARNING] Failed to parse DeliverBefore timestamp: %v\n", parseErr)
			}
		}

		// 5. RequestedDeliveryDate — if set and >= 4 calendar days away (Tashkent TZ), auto-SCHEDULE
		var requestedDD spanner.NullTime
		if req.RequestedDeliveryDate != "" {
			parsedTime, parseErr := time.Parse(time.RFC3339, req.RequestedDeliveryDate)
			if parseErr == nil {
				requestedDD = spanner.NullTime{Time: parsedTime, Valid: true}
				// Preorder threshold: 4 calendar days from today (Tashkent TZ)
				nowTKT := proximity.TashkentNow()
				todayMidnight := time.Date(nowTKT.Year(), nowTKT.Month(), nowTKT.Day(), 0, 0, 0, 0, proximity.TashkentLocation)
				preorderCutoff := todayMidnight.AddDate(0, 0, 4)
				deliveryTKT := parsedTime.In(proximity.TashkentLocation)
				if !deliveryTKT.Before(preorderCutoff) {
					initialState = "SCHEDULED"
				}
			} else {
				fmt.Printf("[WARNING] Failed to parse RequestedDeliveryDate: %v\n", parseErr)
			}
		}

		// Fulfillment warehouse (Phase VII)
		var fulfillmentWH spanner.NullString
		if req.FulfillmentWarehouseId != "" {
			fulfillmentWH = spanner.NullString{StringVal: req.FulfillmentWarehouseId, Valid: true}
		}

		mut := spanner.Insert("Orders",
			[]string{"OrderId", "RetailerId", "Amount", "Currency", "PaymentGateway", "State", "ShopLocation", "RouteId", "OrderSource", "AutoConfirmAt", "DeliverBefore", "RequestedDeliveryDate", "ScheduleShard", "DeliveryToken", "DeliveryFee", "FulfillmentWarehouseId", "Version", "CreatedAt"},
			[]interface{}{orderID, req.RetailerID, req.Amount, "UZS", req.PaymentGateway, initialState, wkt, spanner.NullString{Valid: false}, orderSource, autoConfirm, deliverBefore, requestedDD, scheduleShard, spanner.NullString{Valid: false}, req.DeliveryFee, fulfillmentWH, int64(1), spanner.CommitTimestamp},
		)
		return txn.BufferWrite([]*spanner.Mutation{mut})
	})

	if err != nil {
		return "", fmt.Errorf("failed to execute create order transaction: %w", err)
	}

	return orderID, nil
}

func (s *OrderService) ListOrders(ctx context.Context, routeId string, state string, retailerId string) ([]Order, error) {
	return s.ListOrdersPaginated(ctx, routeId, state, retailerId, 100, 0)
}

func (s *OrderService) ListOrdersPaginated(ctx context.Context, routeId string, state string, retailerId string, limit int, offset int64) ([]Order, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}

	sql := `SELECT o.OrderId, o.RetailerId, o.Amount, o.Currency, o.PaymentGateway, o.State, o.RouteId,
	               o.OrderSource, o.AutoConfirmAt, o.DeliverBefore, o.DeliveryToken, o.CreatedAt,
	               o.SupplierId, COALESCE(s.Name, '') AS SupplierName
	        FROM Orders o
	        LEFT JOIN Suppliers s ON o.SupplierId = s.SupplierId
	        WHERE 1=1`
	params := map[string]interface{}{
		"limit":  int64(limit),
		"offset": offset,
	}

	if routeId != "" {
		sql += " AND RouteId = @routeId"
		params["routeId"] = routeId
	}
	if state != "" {
		sql += " AND State = @state"
		params["state"] = state
	}
	if retailerId != "" {
		sql += " AND RetailerId = @retailerId"
		params["retailerId"] = retailerId
	}
	sql += " ORDER BY CreatedAt DESC LIMIT @limit OFFSET @offset"

	stmt := spanner.Statement{SQL: sql, Params: params}

	iter := s.Client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var orders []Order
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to query orders: %w", err)
		}

		var id, retailerId, stateVal string
		var amount spanner.NullInt64
		var currency spanner.NullString
		var gateway spanner.NullString
		var routeIdVal spanner.NullString
		var orderSource spanner.NullString
		var autoConfirmAt spanner.NullTime
		var deliverBefore spanner.NullTime
		var deliveryToken spanner.NullString
		var createdAt spanner.NullTime
		var supplierID spanner.NullString
		var supplierName spanner.NullString

		if err := row.Columns(&id, &retailerId, &amount, &currency, &gateway, &stateVal, &routeIdVal, &orderSource, &autoConfirmAt, &deliverBefore, &deliveryToken, &createdAt, &supplierID, &supplierName); err != nil {
			return nil, fmt.Errorf("failed to parse order row: %w", err)
		}

		orders = append(orders, Order{
			ID:             id,
			RetailerID:     retailerId,
			SupplierID:     supplierID.StringVal,
			SupplierName:   supplierName.StringVal,
			Amount:         amount.Int64,
			Currency:       currency.StringVal,
			PaymentGateway: gateway.StringVal,
			State:          stateVal,
			RouteID:        routeIdVal,
			OrderSource:    orderSource,
			AutoConfirmAt:  autoConfirmAt,
			DeliverBefore:  deliverBefore,
			DeliveryToken:  deliveryToken,
			CreatedAt:      createdAt.Time,
		})
	}

	if orders == nil {
		orders = []Order{}
	}

	// Hydrate line items for each order
	if len(orders) > 0 {
		orderIDs := make([]string, len(orders))
		for i, o := range orders {
			orderIDs[i] = o.ID
		}
		itemStmt := spanner.Statement{
			SQL: `SELECT li.LineItemId, li.OrderId, li.SkuId, COALESCE(sp.Name, li.SkuId) AS SkuName, li.Quantity, li.UnitPrice, li.Currency, li.Status
				  FROM OrderLineItems li
				  LEFT JOIN SupplierProducts sp ON li.SkuId = sp.SkuId
				  WHERE li.OrderId IN UNNEST(@orderIds)`,
			Params: map[string]interface{}{"orderIds": orderIDs},
		}
		itemIter := s.Client.Single().Query(ctx, itemStmt)
		defer itemIter.Stop()

		itemMap := map[string][]LineItem{}
		for {
			row, err := itemIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				break // non-fatal: return orders without items
			}
			var li LineItem
			if err := row.Columns(&li.LineItemID, &li.OrderID, &li.SkuID, &li.SkuName, &li.Quantity, &li.UnitPrice, &li.Currency, &li.Status); err != nil {
				continue
			}
			itemMap[li.OrderID] = append(itemMap[li.OrderID], li)
		}
		for i := range orders {
			if items, ok := itemMap[orders[i].ID]; ok {
				orders[i].Items = items
			}
		}
	}

	return orders, nil
}

// ListRoutes returns distinct truck/route IDs that have orders assigned
func (s *OrderService) ListRoutes(ctx context.Context) ([]string, error) {
	stmt := spanner.Statement{SQL: `SELECT DISTINCT RouteId FROM Orders WHERE RouteId IS NOT NULL ORDER BY RouteId`}
	iter := s.Client.Single().Query(ctx, stmt)
	defer iter.Stop()
	var routes []string
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to query routes: %w", err)
		}
		var routeId string
		if err := row.Columns(&routeId); err != nil {
			continue
		}
		routes = append(routes, routeId)
	}
	if routes == nil {
		routes = []string{}
	}
	return routes, nil
}

// SaveDeviceToken persists the Expo/FCM push token for a specific retailer device
func (s *OrderService) SaveDeviceToken(ctx context.Context, retailerID string, deviceToken string) error {
	_, err := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: `UPDATE Retailers SET DeviceToken = @token WHERE RetailerId = @id`,
			Params: map[string]interface{}{
				"token": deviceToken,
				"id":    retailerID,
			},
		}
		numRows, err := txn.Update(ctx, stmt)
		if err != nil {
			return err
		}
		if numRows == 0 {
			return fmt.Errorf("retailer %s not found", retailerID)
		}
		return nil
	})
	return err
}

func (s *OrderService) GetActiveFleet(ctx context.Context, supplierID, routeId string) ([]ActiveMission, error) {
	sql := "SELECT OrderId, State, ShopLocation, Amount, PaymentGateway, RouteId, SupplierId, EstimatedArrivalAt FROM Orders WHERE State IN ('PENDING', 'EN_ROUTE')"

	params := map[string]interface{}{}
	if supplierID != "" {
		sql += " AND SupplierId = @supplierId"
		params["supplierId"] = supplierID
	}
	if routeId != "" {
		sql += " AND RouteId = @routeId"
		params["routeId"] = routeId
	}

	stmt := spanner.Statement{SQL: sql, Params: params}

	iter := s.Client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var fleet []ActiveMission
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to query active fleet: %w", err)
		}

		var orderID, state string
		var shopLocation spanner.NullString
		var amount spanner.NullInt64
		var gateway spanner.NullString
		var routeId spanner.NullString
		var sid spanner.NullString
		var etaAt spanner.NullTime

		if err := row.Columns(&orderID, &state, &shopLocation, &amount, &gateway, &routeId, &sid, &etaAt); err != nil {
			return nil, fmt.Errorf("failed to parse fleet row: %w", err)
		}

		var targetLat, targetLng float64
		if shopLocation.Valid {
			if loc, err := parseWKTPoint(shopLocation.StringVal); err == nil {
				targetLat = loc.Latitude
				targetLng = loc.Longitude
			}
		}

		m := ActiveMission{
			OrderID:    orderID,
			State:      state,
			TargetLat:  targetLat,
			TargetLng:  targetLng,
			Amount:     amount.Int64,
			Currency:   "UZS",
			Gateway:    gateway.StringVal,
			RouteID:    routeId.StringVal,
			SupplierID: sid.StringVal,
		}
		if etaAt.Valid {
			s := etaAt.Time.Format(time.RFC3339)
			m.EstimatedArrivalAt = &s
		}
		fleet = append(fleet, m)
	}

	if fleet == nil {
		fleet = []ActiveMission{}
	}

	return fleet, nil
}

func (s *OrderService) SealPayload(ctx context.Context, req PayloadSealRequest) (string, error) {
	if !req.ManifestCleared {
		return "", fmt.Errorf("bad request: manifest not cleared")
	}

	// JIT: Generate the delivery token at dispatch time, not at checkout
	deliveryToken := GenerateSecureToken()
	var retailerID, supplierID string

	_, err := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// 1. Fetch current order state + retailer + supplier for WS push and notifications
		stmt := spanner.Statement{
			SQL: `SELECT State, RetailerId, SupplierId FROM Orders WHERE OrderId = @orderId LIMIT 1`,
			Params: map[string]interface{}{
				"orderId": req.OrderID,
			},
		}

		iter := txn.Query(ctx, stmt)
		row, err := iter.Next()
		iter.Stop()
		if err == iterator.Done {
			return fmt.Errorf("order %s not found", req.OrderID)
		}
		if err != nil {
			return fmt.Errorf("failed to query order state: %w", err)
		}

		var state string
		var rid, sid spanner.NullString
		if err := row.Columns(&state, &rid, &sid); err != nil {
			return fmt.Errorf("failed to parse order state: %w", err)
		}
		if rid.Valid {
			retailerID = rid.StringVal
		}
		if sid.Valid {
			supplierID = sid.StringVal
		}

		// 2. Payload sealing is only valid for warehouse-cleared LOADED orders.
		if state != "LOADED" {
			return fmt.Errorf("order %s must be LOADED to seal payload (current state: %s)", req.OrderID, state)
		}

		// 3. Attach the JIT delivery token AND advance state to DISPATCHED.
		updateStmt := spanner.Statement{
			SQL: `UPDATE Orders SET DeliveryToken = @token, State = 'DISPATCHED' WHERE OrderId = @orderId`,
			Params: map[string]interface{}{
				"orderId": req.OrderID,
				"token":   deliveryToken,
			},
		}

		if _, err := txn.Update(ctx, updateStmt); err != nil {
			return fmt.Errorf("failed to seal payload and advance to DISPATCHED: %w", err)
		}

		now := time.Now().UTC()
		traceID := telemetry.TraceIDFromContext(ctx)

		if err := outbox.EmitJSON(txn, "Order", req.OrderID, kafkaEvents.EventPayloadSealed, topicLogisticsEvents, kafkaEvents.PayloadSealedEvent{
			OrderID:       req.OrderID,
			TerminalID:    req.TerminalID,
			DeliveryToken: deliveryToken,
			Timestamp:     now,
		}, traceID); err != nil {
			return fmt.Errorf("outbox emit payload sealed: %w", err)
		}

		if err := outbox.EmitJSON(txn, "Order", req.OrderID, kafkaEvents.EventOrderStatusChanged, topicLogisticsEvents, kafkaEvents.OrderStatusChangedEvent{
			OrderID:    req.OrderID,
			RetailerID: retailerID,
			SupplierID: supplierID,
			OldState:   "LOADED",
			NewState:   "DISPATCHED",
			Timestamp:  now,
		}, traceID); err != nil {
			return fmt.Errorf("outbox emit order status changed (sealed): %w", err)
		}

		return nil
	})

	// Cache the delivery token in Redis with 4-hour TTL for fast validation.
	if err == nil {
		if cache.Client != nil {
			ttlCtx, ttlCancel := context.WithTimeout(context.Background(), 2*time.Second)
			if setErr := cache.Client.Set(ttlCtx, cache.PrefixDeliveryToken+req.OrderID, deliveryToken, cache.TTLDeliveryToken).Err(); setErr != nil {
				slog.Error("order.delivery_token_redis_set_failed", "order_id", req.OrderID, "err", setErr)
			}
			ttlCancel()
		}
	}

	return retailerID, err
}

// MarkArrived transitions an order from IN_TRANSIT → ARRIVED.
// Returns the owning supplierID for WebSocket push notification.
func (s *OrderService) MarkArrived(ctx context.Context, orderID string) (string, error) {
	var supplierID, retailerID, driverID string
	_, err := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "Orders", spanner.Key{orderID}, []string{"State", "Version", "SupplierId", "RetailerId", "DriverId"})
		if err != nil {
			return fmt.Errorf("order %s not found: %w", orderID, err)
		}

		var state string
		var version int64
		var sid, did spanner.NullString
		if err := row.Columns(&state, &version, &sid, &retailerID, &did); err != nil {
			return err
		}
		if sid.Valid {
			supplierID = sid.StringVal
		}
		if did.Valid {
			driverID = did.StringVal
		}

		if state != "IN_TRANSIT" {
			return fmt.Errorf("order %s must be IN_TRANSIT to mark arrived (current: %s)", orderID, state)
		}

		mut := spanner.Update("Orders",
			[]string{"OrderId", "State", "Version"},
			[]interface{}{orderID, "ARRIVED", version + 1},
		)
		if err := txn.BufferWrite([]*spanner.Mutation{mut}); err != nil {
			return err
		}

		now := time.Now().UTC()
		traceID := telemetry.TraceIDFromContext(ctx)

		if err := outbox.EmitJSON(txn, "Order", orderID, kafkaEvents.EventDriverArrived, topicLogisticsEvents, kafkaEvents.DriverArrivedEvent{
			OrderID:    orderID,
			RetailerID: retailerID,
			DriverID:   driverID,
			SupplierID: supplierID,
			Timestamp:  now,
		}, traceID); err != nil {
			return fmt.Errorf("outbox emit driver arrived: %w", err)
		}

		if err := outbox.EmitJSON(txn, "Order", orderID, kafkaEvents.EventOrderStatusChanged, topicLogisticsEvents, kafkaEvents.OrderStatusChangedEvent{
			OrderID:    orderID,
			RetailerID: retailerID,
			SupplierID: supplierID,
			OldState:   "IN_TRANSIT",
			NewState:   "ARRIVED",
			Timestamp:  now,
		}, traceID); err != nil {
			return fmt.Errorf("outbox emit order status changed (arrived): %w", err)
		}

		return nil
	})

	return supplierID, err
}

// RefreshDeliveryTokenTTL extends the Redis TTL on a delivery token.
// Called on DRIVER_APPROACHING to keep the token alive for the delivery window.
func (s *OrderService) RefreshDeliveryTokenTTL(ctx context.Context, orderID string) {
	if cache.Client == nil {
		return
	}
	ttlCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := cache.Client.Expire(ttlCtx, cache.PrefixDeliveryToken+orderID, cache.TTLDeliveryToken).Err(); err != nil {
		slog.Error("order.delivery_token_refresh_failed", "order_id", orderID, "err", err)
	}
}

// InvalidateDeliveryToken removes the Redis-cached delivery token.
// Called on ORDER_COMPLETED or ORDER_CANCELLED to prevent stale token reuse.
func (s *OrderService) InvalidateDeliveryToken(ctx context.Context, orderID string) {
	if cache.Client == nil {
		return
	}
	ttlCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := cache.Client.Del(ttlCtx, cache.PrefixDeliveryToken+orderID).Err(); err != nil {
		slog.Error("order.delivery_token_invalidation_failed", "order_id", orderID, "err", err)
	}
}

// AssignRoute performs a strict read-before-write transaction assigning a RouteId to one or more orders.
// ADMIN OVERRIDE: also forces State → PENDING and wipes AutoConfirmAt.
// CONCURRENCY GUARD: reads current state + Version + LockedUntil inside the RW transaction.
// OCC: each order's Version is bumped; if any concurrent mutation raced, rowCount=0 → ErrVersionConflict.
// FREEZE LOCK: dispatch WRITES LockedUntil (30 min from now) to prevent mutations during physical delivery.
func (s *OrderService) AssignRoute(ctx context.Context, orderIds []string, routeId string) error {
	_, err := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var eventSupplierID, eventDriverID string

		// Step 1: Read current state + OCC fields of ALL orders inside the serialised transaction
		stmt := spanner.Statement{
			SQL:    `SELECT OrderId, State, RouteId, Version, LockedUntil, SupplierId, DriverId FROM Orders WHERE OrderId IN UNNEST(@ids)`,
			Params: map[string]interface{}{"ids": orderIds},
		}
		iter := txn.Query(ctx, stmt)
		defer iter.Stop()

		type orderSnap struct {
			state      string
			route      spanner.NullString
			version    int64
			locked     spanner.NullTime
			supplierID string
			driverID   string
		}
		found := map[string]orderSnap{}
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return fmt.Errorf("read orders for dispatch: %w", err)
			}

			var id, state string
			var existingRoute, supplierCol, driverCol spanner.NullString
			var version int64
			var lockedUntil spanner.NullTime
			if err := row.Columns(&id, &state, &existingRoute, &version, &lockedUntil, &supplierCol, &driverCol); err != nil {
				return fmt.Errorf("parse order row: %w", err)
			}
			snap := orderSnap{state: state, route: existingRoute, version: version, locked: lockedUntil}
			if supplierCol.Valid {
				snap.supplierID = supplierCol.StringVal
			}
			if driverCol.Valid {
				snap.driverID = driverCol.StringVal
			}
			found[id] = snap

			// Freeze lock check — if a prior dispatch freeze is still active, reject
			if lockedUntil.Valid && time.Now().Before(lockedUntil.Time) {
				return &ErrFreezeLock{OrderID: id, LockedUntil: lockedUntil.Time}
			}

			// Guard: only PENDING or PENDING_REVIEW (unrouted) orders can be dispatched
			switch state {
			case "CANCELLED":
				return &ErrStateConflict{OrderID: id, CurrentState: state, AttemptedOp: "dispatch"}
			case "IN_TRANSIT", "ARRIVED", "COMPLETED":
				return &ErrStateConflict{OrderID: id, CurrentState: state, AttemptedOp: "dispatch"}
			}
		}

		// Verify all requested orders exist
		for _, id := range orderIds {
			if _, ok := found[id]; !ok {
				return fmt.Errorf("order %s not found", id)
			}
		}

		// Step 2: All guards passed — write route assignment with version-guarded DML
		freezeUntil := time.Now().Add(30 * time.Minute)
		for _, id := range orderIds {
			snap := found[id]
			newVersion := snap.version + 1
			rowCount, err := txn.Update(ctx, spanner.Statement{
				SQL: `UPDATE Orders
				      SET RouteId = @routeId,
				          State = 'PENDING',
				          AutoConfirmAt = NULL,
				          Version = @newVersion,
				          LockedUntil = @freezeUntil
				      WHERE OrderId = @orderId AND Version = @version`,
				Params: map[string]interface{}{
					"routeId":     spanner.NullString{StringVal: routeId, Valid: true},
					"newVersion":  newVersion,
					"freezeUntil": freezeUntil,
					"orderId":     id,
					"version":     snap.version,
				},
			})
			if err != nil {
				return fmt.Errorf("dispatch order %s: %w", id, err)
			}
			if rowCount == 0 {
				return &ErrVersionConflict{OrderID: id, ExpectedVersion: snap.version, ActualVersion: -1}
			}
		}

		// Capture metadata for post-commit notification events
		for _, snap := range found {
			if snap.supplierID != "" && eventSupplierID == "" {
				eventSupplierID = snap.supplierID
			}
			if snap.driverID != "" && eventDriverID == "" {
				eventDriverID = snap.driverID
			}
		}

		now := time.Now().UTC()
		traceID := telemetry.TraceIDFromContext(ctx)

		if err := outbox.EmitJSON(txn, "Route", routeId, kafkaEvents.EventFleetDispatched, topicLogisticsEvents, kafkaEvents.FleetDispatchedEvent{
			RouteID:   routeId,
			OrderIDs:  orderIds,
			Timestamp: now,
		}, traceID); err != nil {
			return fmt.Errorf("outbox emit fleet dispatched: %w", err)
		}

		if err := outbox.EmitJSON(txn, "Route", routeId, kafkaEvents.EventOrderDispatched, topicLogisticsEvents, kafkaEvents.OrderDispatchedEvent{
			RouteID:    routeId,
			OrderIDs:   orderIds,
			DriverID:   eventDriverID,
			SupplierID: eventSupplierID,
			Timestamp:  now,
		}, traceID); err != nil {
			return fmt.Errorf("outbox emit order dispatched: %w", err)
		}

		if err := outbox.EmitJSON(txn, "Route", routeId, kafkaEvents.EventPayloadReadyToSeal, topicLogisticsEvents, kafkaEvents.PayloadReadyToSealEvent{
			RouteID:    routeId,
			OrderIDs:   orderIds,
			SupplierID: eventSupplierID,
			Timestamp:  now,
		}, traceID); err != nil {
			return fmt.Errorf("outbox emit payload ready to seal: %w", err)
		}

		return nil
	})

	return err
}

// PublishEvent emits a best-effort domain event through the transactional outbox.
// Always call in a goroutine (go s.PublishEvent(...)) to avoid blocking the HTTP path.
func (s *OrderService) PublishEvent(ctx context.Context, eventType string, payload interface{}) {
	if s == nil || s.Client == nil {
		slog.Warn("order publish event skipped: spanner unavailable", "event_type", eventType)
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	eventCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()

	aggregateID := fmt.Sprintf("%s:%d", eventType, time.Now().UTC().UnixNano())
	_, err := s.Client.ReadWriteTransaction(eventCtx, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		return outbox.EmitJSON(txn, "OrderEvent", aggregateID, eventType, topicLogisticsEvents, payload, telemetry.TraceIDFromContext(txCtx))
	})
	if err != nil {
		slog.ErrorContext(eventCtx, "order publish event failed", "event_type", eventType, "err", err)
	}
}

// ─── Cancellation Firewall ────────────────────────────────────────────────────

type ErrCancelForbidden struct{ Reason string }

func (e *ErrCancelForbidden) Error() string {
	return fmt.Sprintf("Access Denied: %s", e.Reason)
}

type CancelOrderRequest struct {
	OrderID    string `json:"order_id"`
	RetailerID string `json:"retailer_id"` // Must match the order owner
	Version    int64  `json:"version"`     // Optimistic concurrency — client must supply
}

// CancelOrder enforces the cancellation firewall with optimistic concurrency
// control and freeze-lock awareness:
//  1. Read State, Version, LockedUntil, RouteId, RetailerId
//  2. Evaluate Freeze Lock — locked orders cannot be cancelled
//  3. OCC check — version must match client expectation
//  4. State + Route firewall — only PENDING/PENDING_REVIEW without route
//  5. Execute mutation, bump Version
func (s *OrderService) CancelOrder(ctx context.Context, req CancelOrderRequest) error {
	var wasAuthorized bool
	var orderGateway string
	var cancelledOrderID string

	_, err := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "Orders", spanner.Key{req.OrderID},
			[]string{"State", "Version", "LockedUntil", "RouteId", "RetailerId", "PaymentStatus", "PaymentGateway"})
		if err != nil {
			return fmt.Errorf("order %s not found: %w", req.OrderID, err)
		}

		var state, retailerId string
		var version int64
		var lockedUntil spanner.NullTime
		var routeId spanner.NullString
		var paymentStatusNull, gatewayNull spanner.NullString
		if err := row.Columns(&state, &version, &lockedUntil, &routeId, &retailerId, &paymentStatusNull, &gatewayNull); err != nil {
			return fmt.Errorf("failed to parse order row: %w", err)
		}

		// 1. Ownership check
		if retailerId != req.RetailerID {
			return &ErrCancelForbidden{Reason: "You do not own this order."}
		}

		// 2. Freeze Lock — order is locked for physical dispatch
		if lockedUntil.Valid && time.Now().Before(lockedUntil.Time) {
			return &ErrFreezeLock{OrderID: req.OrderID, LockedUntil: lockedUntil.Time}
		}

		// 3. Optimistic Concurrency — version mismatch means another txn won
		if req.Version != 0 && version != req.Version {
			return &ErrVersionConflict{OrderID: req.OrderID, ExpectedVersion: req.Version, ActualVersion: version}
		}

		// 4. State firewall — cancellation is only permitted for PENDING orders.
		// Once a supplier has acknowledged (PENDING_REVIEW) or the order is routed,
		// the retailer can no longer cancel.
		if state != "PENDING" {
			return &ErrStateConflict{OrderID: req.OrderID, CurrentState: state, AttemptedOp: "cancel"}
		}

		// 5. Route firewall
		if routeId.Valid && routeId.StringVal != "" {
			return &ErrCancelForbidden{Reason: "Admin has already routed this payload to a truck."}
		}

		// Track authorization state for post-commit void.
		if paymentStatusNull.Valid && paymentStatusNull.StringVal == "AUTHORIZED" {
			wasAuthorized = true
			orderGateway = gatewayNull.StringVal
			cancelledOrderID = req.OrderID
		}

		// 6. Execute cancellation + bump version + clear PaymentStatus if authorized
		updateSQL := `UPDATE Orders SET State = 'CANCELLED', AutoConfirmAt = NULL, Version = @newVersion WHERE OrderId = @orderId AND Version = @version`
		if wasAuthorized {
			updateSQL = `UPDATE Orders SET State = 'CANCELLED', AutoConfirmAt = NULL, PaymentStatus = 'CANCELLED', Version = @newVersion WHERE OrderId = @orderId AND Version = @version`
		}
		updateStmt := spanner.Statement{
			SQL:    updateSQL,
			Params: map[string]interface{}{"orderId": req.OrderID, "version": version, "newVersion": version + 1},
		}
		rowCount, updateErr := txn.Update(ctx, updateStmt)
		if updateErr != nil {
			return fmt.Errorf("failed to cancel order: %w", updateErr)
		}
		if rowCount == 0 {
			return &ErrVersionConflict{OrderID: req.OrderID, ExpectedVersion: version, ActualVersion: -1}
		}

		if err := outbox.EmitJSON(txn, "Order", req.OrderID, kafkaEvents.EventOrderCancelled, topicLogisticsEvents, map[string]interface{}{
			"order_id":  req.OrderID,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}, telemetry.TraceIDFromContext(ctx)); err != nil {
			return fmt.Errorf("outbox emit order cancelled: %w", err)
		}

		return nil
	})

	if err == nil {
		// Void the GP authorization hold if the order was AUTHORIZED at checkout.
		if wasAuthorized && orderGateway == "GLOBAL_PAY" && s.DirectClient != nil {
			go s.voidAuthorizationForOrder(context.Background(), cancelledOrderID)
		}
	}

	return err
}

// voidAuthorizationForOrder releases a Global Pay authorization hold by looking up
// the AUTHORIZED PaymentSession for the order and calling VoidAuthorization.
// Best-effort: failures are logged but do not propagate (hold expires naturally).
func (s *OrderService) voidAuthorizationForOrder(ctx context.Context, orderID string) {
	if s.SessionSvc == nil {
		return
	}
	session, sessErr := s.SessionSvc.GetActiveSessionByOrder(ctx, orderID)
	if sessErr != nil || session == nil || session.AuthorizationID == "" {
		slog.Info("order.void_auth_no_active_session", "order_id", orderID)
		return
	}

	var merchantID, serviceID, secretKey string
	if s.Vault != nil {
		cfg, vaultErr := s.Vault.GetDecryptedConfigByOrder(ctx, orderID, "GLOBAL_PAY")
		if vaultErr == nil {
			merchantID = cfg.MerchantId
			serviceID = cfg.ServiceId
			secretKey = cfg.SecretKey
		}
	}
	creds, credErr := payment.ResolveGlobalPayCredentials(merchantID, serviceID, secretKey)
	if credErr != nil {
		slog.Error("order.void_auth_credentials_failed", "order_id", orderID, "err", credErr)
		return
	}

	if voidErr := s.DirectClient.VoidAuthorization(ctx, creds, session.AuthorizationID); voidErr != nil {
		slog.Warn("order.void_auth_failed", "order_id", orderID, "authorization_id", session.AuthorizationID, "err", voidErr)
		return
	}

	// Mark session as CANCELLED.
	_, updateErr := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("PaymentSessions",
				[]string{"SessionId", "Status"},
				[]interface{}{session.SessionID, "CANCELLED"},
			),
		})
	})
	if updateErr != nil {
		slog.Error("order.void_auth_session_cancel_failed", "session_id", session.SessionID, "err", updateErr)
	} else {
		slog.Info("order.void_auth_completed", "order_id", orderID, "session_id", session.SessionID)
	}
}

func (s *OrderService) CompleteDeliveryWithToken(ctx context.Context, orderId string, scannedToken string, driverLat, driverLng float64) (string, error) {
	var retailerId string
	var supplierID string
	var warehouseId string

	_, err := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// 1. Lock the row and read the true token + shop location
		row, err := txn.ReadRow(ctx, "Orders", spanner.Key{orderId}, []string{"State", "DeliveryToken", "RetailerId", "ShopLocation", "SupplierId", "WarehouseId", "Amount", "Currency", "ManifestId"})
		if err != nil {
			return err
		}

		var state string
		var trueToken spanner.NullString
		var shopLocation spanner.NullString
		var sid spanner.NullString
		var wid spanner.NullString
		var amount spanner.NullInt64
		var currency spanner.NullString
		var manifestIDNull spanner.NullString
		if err := row.Columns(&state, &trueToken, &retailerId, &shopLocation, &sid, &wid, &amount, &currency, &manifestIDNull); err != nil {
			return err
		}
		if sid.Valid {
			supplierID = sid.StringVal
		}
		if wid.Valid {
			warehouseId = wid.StringVal
		}

		// 2. The Physical Firewall
		if state == "COMPLETED" {
			return fmt.Errorf("Order is already completed")
		}
		if state != "ARRIVED" {
			return fmt.Errorf("order %s must be ARRIVED before QR completion (current state: %s)", orderId, state)
		}

		// 3. Geofence enforcement — driver must be within 500m of the shop
		if shopLocation.Valid && shopLocation.StringVal != "" {
			shopLoc, parseErr := parseWKTPoint(shopLocation.StringVal)
			if parseErr == nil {
				dist := getDistance(driverLat, driverLng, shopLoc.Latitude, shopLoc.Longitude)
				if dist > 500 {
					return fmt.Errorf("GEOFENCE_VIOLATION: driver is %.0fm from delivery point (max 500m)", dist)
				}
			}
		}

		// This is where the magic happens. Constant-time comparison prevents timing attacks.
		if !trueToken.Valid || subtle.ConstantTimeCompare([]byte(trueToken.StringVal), []byte(scannedToken)) != 1 {
			return fmt.Errorf("INVALID QR TOKEN: Cryptographic handshake failed. Delivery blocked.")
		}

		// 3. Token matches! Execute the handover.
		mut := spanner.Update("Orders", []string{"OrderId", "State"}, []interface{}{
			orderId,
			"COMPLETED",
		})
		if err := txn.BufferWrite([]*spanner.Mutation{mut}); err != nil {
			return err
		}

		// Emit ORDER_COMPLETED via transactional outbox — atomic with state transition.
		cur := "UZS"
		if currency.Valid && currency.StringVal != "" {
			cur = currency.StringVal
		}
		if err := outbox.EmitJSON(txn, "Order", orderId, kafkaEvents.EventOrderCompleted, kafkaEvents.TopicMain, kafkaEvents.OrderCompletedEvent{
			OrderID:     orderId,
			RetailerID:  retailerId,
			SupplierId:  supplierID,
			WarehouseId: warehouseId,
			Amount:      amount.Int64,
			Currency:    cur,
			Timestamp:   time.Now().UTC(),
		}, telemetry.TraceIDFromContext(ctx)); err != nil {
			return fmt.Errorf("outbox emit ORDER_COMPLETED: %w", err)
		}

		if err := outbox.EmitJSON(txn, "Order", orderId, kafkaEvents.EventOrderStatusChanged, kafkaEvents.TopicMain, kafkaEvents.OrderStatusChangedEvent{
			OrderID:    orderId,
			RetailerID: retailerId,
			SupplierID: supplierID,
			OldState:   "ARRIVED",
			NewState:   "COMPLETED",
			Timestamp:  time.Now().UTC(),
		}, telemetry.TraceIDFromContext(ctx)); err != nil {
			return fmt.Errorf("outbox emit ORDER_STATUS_CHANGED (arrived->completed): %w", err)
		}

		// LEO Phase V — manifest-completion rollup
		if manifestIDNull.Valid {
			if err := rollupManifestIfComplete(ctx, txn, manifestIDNull.StringVal, time.Now().UTC()); err != nil {
				return fmt.Errorf("manifest rollup failed for order %s: %w", orderId, err)
			}
		}

		return nil
	})

	// 4. Wake up the Intelligence Engine!
	if err == nil {
		// Decrement warehouse queue depth on successful completion
		cache.DecrementQueueDepth(context.Background(), warehouseId)
	}

	return supplierID, err
}

// ─── Phase 4: Supplier Dashboard Aggregation ──────────────────────────────────

// SupplierDashboardMetrics defines the exact JSON structure the Next.js dashboard needs
type SupplierDashboardMetrics struct {
	TotalPipeline    int64 `json:"total_pipeline"`
	PendingVolume    int64 `json:"pending_volume"`
	AIForecastVolume int64 `json:"ai_forecast_volume"`
}

// GetSupplierMetrics aggregates live data from Orders and AIPredictions
func (s *OrderService) GetSupplierMetrics(ctx context.Context) (*SupplierDashboardMetrics, error) {
	metrics := &SupplierDashboardMetrics{}

	// 1. Query the Present: Aggregate active amount and volume
	// We GROUP BY state to get the exact breakdown without pulling every row.
	stmt := spanner.Statement{
		SQL: `SELECT State, SUM(Amount), COUNT(OrderId)
		      FROM Orders
		      WHERE State IN ('PENDING', 'PENDING_REVIEW', 'LOADED')
		      GROUP BY State`,
	}
	iter := s.Client.Single().Query(ctx, stmt)
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to aggregate orders: %v", err)
		}

		var state string
		var totalAmount spanner.NullInt64
		var count int64
		row.Columns(&state, &totalAmount, &count)

		if state == "PENDING" || state == "LOADED" {
			// Active pipeline — orders ready or in dispatch queue
			metrics.TotalPipeline += totalAmount.Int64
			metrics.PendingVolume += count
		} else if state == "PENDING_REVIEW" {
			// AI-predicted, not yet confirmed — still counted as pending volume
			metrics.PendingVolume += count
		}
	}

	return metrics, nil
}

// ─── Phase 7.2: Retailer KYC Operations ──────────────────────────────────

type Retailer struct {
	RetailerId  string    `json:"retailer_id"`
	ShopName    string    `json:"shop_name"`
	OwnerName   string    `json:"owner_name"`  // Assuming Name from DDL maps to this
	StirTaxId   string    `json:"stir_tax_id"` // TaxIdentificationNumber from DDL
	Status      string    `json:"status"`      // e.g. "PENDING_KYC", "VERIFIED", "REJECTED"
	SubmittedAt time.Time `json:"submitted_at"`
}

// ListPendingRetailers fetches all retailers waiting for KYC clearance
func (s *OrderService) ListPendingRetailers(ctx context.Context) ([]Retailer, error) {
	stmt := spanner.Statement{
		SQL: `SELECT RetailerId, Name, TaxIdentificationNumber, Status, CreatedAt 
		      FROM Retailers 
		      WHERE Status = 'PENDING_KYC' 
		      ORDER BY CreatedAt ASC`,
	}

	iter := s.Client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var retailers []Retailer
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to query pending retailers: %w", err)
		}

		var id, name string
		var taxId, status spanner.NullString
		var createdAt spanner.NullTime

		if err := row.Columns(&id, &name, &taxId, &status, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to parse retailer row: %w", err)
		}

		retailers = append(retailers, Retailer{
			RetailerId:  id,
			ShopName:    name,
			OwnerName:   name, // Simple map for now based on DDL
			StirTaxId:   taxId.StringVal,
			Status:      status.StringVal,
			SubmittedAt: createdAt.Time,
		})
	}

	if retailers == nil {
		retailers = []Retailer{} // ensure JSON serializes to [] not null
	}
	return retailers, nil
}

// UpdateRetailerStatus changes the KYC status of a retailer (e.g. to VERIFIED or REJECTED)
func (s *OrderService) UpdateRetailerStatus(ctx context.Context, retailerId string, newStatus string) error {
	_, err := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mut := spanner.Update("Retailers",
			[]string{"RetailerId", "Status"},
			[]interface{}{retailerId, newStatus},
		)
		return txn.BufferWrite([]*spanner.Mutation{mut})
	})
	return err
}

// GetRetailerStatus fetches the KYC verification status for a given retailer
func (s *OrderService) GetRetailerStatus(ctx context.Context, retailerId string) (string, error) {
	stmt := spanner.Statement{
		SQL:    `SELECT Status FROM Retailers WHERE RetailerId = @id`,
		Params: map[string]interface{}{"id": retailerId},
	}

	iter := s.Client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return "", fmt.Errorf("retailer not found")
	}
	if err != nil {
		return "", fmt.Errorf("failed to query retailer status: %w", err)
	}

	var status spanner.NullString
	if err := row.Columns(&status); err != nil {
		return "", fmt.Errorf("failed to parse retailer status: %w", err)
	}

	return status.StringVal, nil
}

// ─── VECTOR B: ORDER AMEND (PARTIAL-QUANTITY RECONCILIATION) ──────────────

// AmendItemReq represents a single line-item partial settlement by the driver.
// AcceptedQty + RejectedQty must equal the original Quantity for that SKU.
type AmendItemReq struct {
	ProductId   string `json:"product_id"`
	AcceptedQty int64  `json:"accepted_qty"`
	RejectedQty int64  `json:"rejected_qty"`
	Reason      string `json:"reason"` // DAMAGED | MISSING | WRONG_ITEM | OTHER
}

type AmendOrderRequest struct {
	OrderID     string         `json:"order_id"`
	AmendmentID string         `json:"amendment_id"` // client-generated idempotency key
	Items       []AmendItemReq `json:"items"`
	DriverNotes string         `json:"driver_notes"`
}

type AmendOrderResponse struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	AdjustedTotal int64  `json:"adjusted_total"`
	AmendmentID   string `json:"amendment_id"`
	RetailerID    string `json:"retailer_id,omitempty"`
	DriverID      string `json:"driver_id,omitempty"`
	SupplierID    string `json:"supplier_id,omitempty"`
}

// OrderModifiedEvent — canonical definition lives in kafka/events.go.

// SupplierReturnEntry is inserted for each rejected item so the warehouse
// manager can process the write-off or return-to-stock.
type SupplierReturnEntry struct {
	ReturnID    string `json:"return_id"`
	OrderID     string `json:"order_id"`
	SkuID       string `json:"sku_id"`
	RejectedQty int64  `json:"rejected_qty"`
	Reason      string `json:"reason"`
	DriverNotes string `json:"driver_notes"`
}

// AmendOrder performs partial-quantity reconciliation inside a single
// Spanner ReadWriteTransaction:
//  1. Validates order state is IN_TRANSIT or ARRIVED
//  2. For each item: looks up OrderLineItems by OrderId + SkuId (product_id),
//     updates Quantity to AcceptedQty, recalculates UnitPrice * AcceptedQty
//  3. Inserts SupplierReturns rows for rejected quantities
//  4. Recalculates Orders.Amount = SUM of all accepted line totals
//  5. Transitions state to COMPLETED
//  6. Emits ORDER_MODIFIED to Kafka with the refund delta
func (s *OrderService) AmendOrder(ctx context.Context, req AmendOrderRequest) (*AmendOrderResponse, error) {
	if len(req.Items) == 0 {
		return nil, fmt.Errorf("amend request must contain at least one line item")
	}

	// Generate idempotency key if client did not provide one
	if req.AmendmentID == "" {
		req.AmendmentID = fmt.Sprintf("AMD-%s", GenerateSecureToken())
	}

	var newTotal int64
	var originalTotal int64
	var driverID string
	var retailerID string
	var supplierID string

	var orderVersion int64

	_, err := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// 1. Fetch order state, original amount, driver + OCC fields
		orderStmt := spanner.Statement{
			SQL:    `SELECT State, Amount, COALESCE(DriverId, ''), COALESCE(RetailerId, ''), COALESCE(SupplierId, ''), Version, LockedUntil FROM Orders WHERE OrderId = @id LIMIT 1`,
			Params: map[string]interface{}{"id": req.OrderID},
		}
		iter := txn.Query(ctx, orderStmt)
		row, err := iter.Next()
		iter.Stop()
		if err != nil {
			return fmt.Errorf("order not found: %w", err)
		}
		var state string
		var lockedUntil spanner.NullTime
		if err := row.Columns(&state, &originalTotal, &driverID, &retailerID, &supplierID, &orderVersion, &lockedUntil); err != nil {
			return fmt.Errorf("order column scan failed: %w", err)
		}

		// Freeze lock check
		if lockedUntil.Valid && time.Now().Before(lockedUntil.Time) {
			return &ErrFreezeLock{OrderID: req.OrderID, LockedUntil: lockedUntil.Time}
		}

		if state != "IN_TRANSIT" && state != "ARRIVED" {
			return fmt.Errorf("order %s cannot be amended from state %s (must be IN_TRANSIT or ARRIVED)", req.OrderID, state)
		}

		// 2. Process each item: lookup by OrderId + SkuId, partial qty update
		for _, item := range req.Items {
			// Fetch the existing line item
			liStmt := spanner.Statement{
				SQL: `SELECT LineItemId, Quantity, UnitPrice FROM OrderLineItems
				      WHERE OrderId = @oid AND SkuId = @skuId LIMIT 1`,
				Params: map[string]interface{}{
					"oid":   req.OrderID,
					"skuId": item.ProductId,
				},
			}
			liIter := txn.Query(ctx, liStmt)
			liRow, liErr := liIter.Next()
			liIter.Stop()
			if liErr != nil {
				return fmt.Errorf("line item not found for sku %s in order %s: %w", item.ProductId, req.OrderID, liErr)
			}

			var lineItemID string
			var origQty, unitPrice int64
			if err := liRow.Columns(&lineItemID, &origQty, &unitPrice); err != nil {
				return fmt.Errorf("line item column scan failed for sku %s: %w", item.ProductId, err)
			}

			// Validate quantities
			if item.AcceptedQty < 0 || item.RejectedQty < 0 {
				return fmt.Errorf("quantities cannot be negative for sku %s", item.ProductId)
			}
			if item.AcceptedQty+item.RejectedQty != origQty {
				return fmt.Errorf("accepted(%d) + rejected(%d) != original(%d) for sku %s",
					item.AcceptedQty, item.RejectedQty, origQty, item.ProductId)
			}

			// Determine status
			var newStatus string
			switch {
			case item.RejectedQty == 0:
				newStatus = "DELIVERED"
			case item.AcceptedQty == 0:
				newStatus = "REJECTED_DAMAGED"
			default:
				newStatus = "PARTIAL_DELIVERED"
			}

			// Update the line item: set accepted quantity, rejected quantity, and status
			updateStmt := spanner.Statement{
				SQL: `UPDATE OrderLineItems SET Quantity = @qty, RejectedQty = @rejQty, Status = @status
				      WHERE LineItemId = @lid`,
				Params: map[string]interface{}{
					"qty":    item.AcceptedQty,
					"rejQty": item.RejectedQty,
					"status": newStatus,
					"lid":    lineItemID,
				},
			}
			if _, err := txn.Update(ctx, updateStmt); err != nil {
				return fmt.Errorf("failed to update line item %s: %w", lineItemID, err)
			}

			// Insert SupplierReturns row for rejected quantity
			if item.RejectedQty > 0 {
				returnID := fmt.Sprintf("RET-%s", GenerateSecureToken())
				if err := txn.BufferWrite([]*spanner.Mutation{
					spanner.Insert("SupplierReturns",
						[]string{"ReturnId", "OrderId", "SkuId", "RejectedQty", "Reason", "DriverNotes", "CreatedAt"},
						[]interface{}{returnID, req.OrderID, item.ProductId, item.RejectedQty, item.Reason, req.DriverNotes, spanner.CommitTimestamp},
					),
				}); err != nil {
					return fmt.Errorf("insert supplier return %s: %w", returnID, err)
				}
			}
		}

		// 3. Recalculate total + VolumeVU in one pass.
		//    Volume uses the canonical fallback chain from supplier/dispatcher.go:
		//    SupplierProducts.VolumetricUnit → (LWH/5000) → PalletFootprint → 1.0.
		//    This is the Volumetric Guardian: every quantity mutation that lands
		//    here keeps Orders.VolumeVU in sync so the Phase 2 optimiser never
		//    plans against stale mass.
		totalStmt := spanner.Statement{
			SQL: `SELECT
			          SUM(li.UnitPrice * li.Quantity),
			          SUM(li.Quantity * COALESCE(
			              sp.VolumetricUnit,
			              (sp.LengthCM * sp.WidthCM * sp.HeightCM) / 5000.0,
			              sp.PalletFootprint,
			              1.0))
			      FROM OrderLineItems li
			      LEFT JOIN SupplierProducts sp ON sp.SkuId = li.SkuId
			      WHERE li.OrderId = @oid`,
			Params: map[string]interface{}{"oid": req.OrderID},
		}
		totalIter := txn.Query(ctx, totalStmt)
		totalRow, err := totalIter.Next()
		totalIter.Stop()
		if err != nil {
			return fmt.Errorf("total recalculation query failed: %w", err)
		}
		var nullTotal spanner.NullInt64
		var nullVolumeVU spanner.NullFloat64
		if err := totalRow.Columns(&nullTotal, &nullVolumeVU); err != nil {
			return fmt.Errorf("total column scan failed: %w", err)
		}
		if nullTotal.Valid {
			newTotal = nullTotal.Int64
		}
		newVolumeVU := 0.0
		if nullVolumeVU.Valid {
			newVolumeVU = nullVolumeVU.Float64
		}

		// 4. Update Orders: new total + new VolumeVU + version-guarded OCC + clear freeze lock (state stays ARRIVED)
		newVersion := orderVersion + 1
		rowCount, updateErr := txn.Update(ctx, spanner.Statement{
			SQL: `UPDATE Orders SET Amount = @total, VolumeVU = @volumeVU, Version = @newVersion, LockedUntil = NULL
			      WHERE OrderId = @id AND Version = @version`,
			Params: map[string]interface{}{
				"total":      newTotal,
				"volumeVU":   newVolumeVU,
				"id":         req.OrderID,
				"newVersion": newVersion,
				"version":    orderVersion,
			},
		})
		if updateErr != nil {
			return fmt.Errorf("failed to update order total: %w", updateErr)
		}
		if rowCount == 0 {
			return &ErrVersionConflict{OrderID: req.OrderID, ExpectedVersion: orderVersion, ActualVersion: -1}
		}

		// 4b. Propagate new amount to active PaymentSession (if one exists)
		sessStmt := spanner.Statement{
			SQL: `SELECT SessionId FROM PaymentSessions
			      WHERE OrderId = @oid AND Status IN ('CREATED', 'PENDING')
			      LIMIT 1`,
			Params: map[string]interface{}{"oid": req.OrderID},
		}
		sessIter := txn.Query(ctx, sessStmt)
		sessRow, sessErr := sessIter.Next()
		sessIter.Stop()
		if sessErr == nil {
			var sessionID string
			if colErr := sessRow.Columns(&sessionID); colErr == nil {
				if err := txn.BufferWrite([]*spanner.Mutation{
					spanner.Update("PaymentSessions",
						[]string{"SessionId", "LockedAmount", "UpdatedAt"},
						[]interface{}{sessionID, newTotal, spanner.CommitTimestamp},
					),
				}); err != nil {
					return fmt.Errorf("update payment session amount %s: %w", sessionID, err)
				}
			}
		}

		// 4c. Propagate new amount to MasterInvoice (if one exists and is still PENDING)
		invAmendStmt := spanner.Statement{
			SQL:    `SELECT InvoiceId FROM MasterInvoices WHERE OrderId = @oid AND State = 'PENDING' LIMIT 1`,
			Params: map[string]interface{}{"oid": req.OrderID},
		}
		invAmendIter := txn.Query(ctx, invAmendStmt)
		invAmendRow, invAmendErr := invAmendIter.Next()
		invAmendIter.Stop()
		if invAmendErr == nil {
			var invoiceID string
			if colErr := invAmendRow.Columns(&invoiceID); colErr == nil {
				if err := txn.BufferWrite([]*spanner.Mutation{
					spanner.Update("MasterInvoices",
						[]string{"InvoiceId", "Total"},
						[]interface{}{invoiceID, newTotal},
					),
				}); err != nil {
					return fmt.Errorf("update master invoice total %s: %w", invoiceID, err)
				}
			}
		}

		refunded := originalTotal - newTotal
		if refunded < 0 {
			refunded = 0
		}

		if err := outbox.EmitJSON(txn, "Order", req.OrderID, kafkaEvents.EventOrderModified, topicLogisticsEvents, kafkaEvents.OrderModifiedEvent{
			OrderID:     req.OrderID,
			AmendmentID: req.AmendmentID,
			DriverID:    driverID,
			SupplierID:  supplierID,
			RetailerID:  retailerID,
			NewAmount:   newTotal,
			Refunded:    refunded,
			Currency:    "UZS",
			Timestamp:   time.Now().UTC(),
		}, telemetry.TraceIDFromContext(ctx)); err != nil {
			return fmt.Errorf("outbox emit ORDER_MODIFIED: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	refunded := originalTotal - newTotal
	if refunded < 0 {
		refunded = 0
	}

	msg := "Order amended"
	if refunded > 0 {
		msg = "Order amended - refund applied"
	}

	return &AmendOrderResponse{
		Success:       true,
		Message:       msg,
		AdjustedTotal: newTotal,
		AmendmentID:   req.AmendmentID,
		RetailerID:    retailerID,
		DriverID:      driverID,
		SupplierID:    supplierID,
	}, nil
}

// HandleAmendOrder is the HTTP handler for POST /v1/order/amend.
// Drivers submit partial-quantity settlements here after last-mile delivery.
func (s *OrderService) HandleAmendOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req AmendOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.OrderID == "" || len(req.Items) == 0 {
		http.Error(w, "order_id and items are required", http.StatusBadRequest)
		return
	}

	resp, err := s.AmendOrder(r.Context(), req)
	if err != nil {
		// OCC version conflict → 409
		var versionConflict *ErrVersionConflict
		if errors.As(err, &versionConflict) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"error": versionConflict.Error()})
			return
		}
		// Freeze lock → 423 Locked
		var freezeLock *ErrFreezeLock
		if errors.As(err, &freezeLock) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(423)
			json.NewEncoder(w).Encode(map[string]string{"error": freezeLock.Error()})
			return
		}
		switch {
		case strings.Contains(err.Error(), "cannot be amended"):
			http.Error(w, err.Error(), http.StatusConflict)
		case strings.Contains(err.Error(), "not found"):
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, "internal error: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// ─── PER-SUPPLIER FULFILLMENT PAYMENT (STAGGERED PARTIAL CHARGE) ─────────────
//
// Multi-supplier order flow:
//   Checkout shattered one cart into N independent orders (one per supplier).
//   Each order travels independently: dispatch → driver → arrive → amend → offload.
//   Once a specific order reaches AWAITING_PAYMENT (after ConfirmOffload), THIS
//   method triggers the card charge for that ONE supplier's adjusted amount only.
//
// Sequence:  ARRIVED → AmendOrder (damages)
//            → ValidateQR → ConfirmOffload (→ AWAITING_PAYMENT)
//            → TriggerSupplierFulfillmentPayment (this method)
//
// Three-phase execution (no Spanner RW tx around external API calls):
//   Phase 1 — Snapshot read: validate state, recalculate line-item total, geofence
//   Phase 2 — External payment: saved card direct charge OR hosted checkout fallback
//   Phase 3 — Commit: OCC-guarded status update, settle session, cache invalidation, Kafka events

// FulfillmentPaymentResult is returned to the caller after triggering a per-supplier payment.
type FulfillmentPaymentResult struct {
	OrderID     string `json:"order_id"`
	SupplierID  string `json:"supplier_id"`
	RetailerID  string `json:"retailer_id"`
	Amount      int64  `json:"amount"` // Adjusted total (after driver edits)
	PaymentID   string `json:"payment_id,omitempty"`
	CheckoutURL string `json:"checkout_url,omitempty"` // Non-empty for hosted checkout / 3DS redirect
	SessionID   string `json:"session_id,omitempty"`
	Status      string `json:"status"` // PAID | 3DS_REQUIRED | CHECKOUT_REDIRECT
	Message     string `json:"message"`
}

// TriggerSupplierFulfillmentPayment triggers a secure card payment for ONE supplier's
// order after the driver has arrived and edited the order (damages/shortages).
//
// The amount charged is the live recalculated subtotal from OrderLineItems — which
// already reflects AmendOrder quantity adjustments — NOT the original order amount.
// Payment is split 95% supplier + 5% platform via ComputeSplitRecipients.
//
// Idempotent: if PaymentStatus is already PAID, returns the settled result without
// re-charging. Safe for driver/supplier retry after network failures.
func (s *OrderService) TriggerSupplierFulfillmentPayment(ctx context.Context, orderID string) (*FulfillmentPaymentResult, error) {

	// ────────────────────────────────────────────────────────────────────────
	// Phase 1: Snapshot Read — validate order state, geofence, and recalculate
	// ────────────────────────────────────────────────────────────────────────

	row, err := s.Client.Single().ReadRow(ctx, "Orders", spanner.Key{orderID},
		[]string{"State", "RetailerId", "SupplierId", "DriverId", "Amount",
			"PaymentStatus", "PaymentGateway", "Version"})
	if err != nil {
		return nil, fmt.Errorf("order %s not found: %w", orderID, err)
	}

	var state, retailerID, paymentStatus string
	var supplierIDNull, driverIDNull, gatewayNull spanner.NullString
	var amount, version int64
	if err := row.Columns(&state, &retailerID, &supplierIDNull, &driverIDNull,
		&amount, &paymentStatus, &gatewayNull, &version); err != nil {
		return nil, fmt.Errorf("failed to parse order %s: %w", orderID, err)
	}

	supplierID := supplierIDNull.StringVal
	driverID := driverIDNull.StringVal
	gateway := gatewayNull.StringVal

	// Idempotent: already settled → return success without re-charging
	if paymentStatus == "PAID" {
		return &FulfillmentPaymentResult{
			OrderID:    orderID,
			SupplierID: supplierID,
			RetailerID: retailerID,
			Amount:     amount,
			Status:     "PAID",
			Message:    "Payment already completed for this fulfillment",
		}, nil
	}

	// AUTHORIZED = funds already held at checkout via GP auth-capture path.
	// Capture happens automatically at CompleteOrder → Treasurer → GatewayWorker.
	// Skip manual charge; return status so the caller knows to proceed with delivery.
	if paymentStatus == "AUTHORIZED" {
		return &FulfillmentPaymentResult{
			OrderID:    orderID,
			SupplierID: supplierID,
			RetailerID: retailerID,
			Amount:     amount,
			Status:     "AUTHORIZED",
			Message:    "Funds held at checkout — capture occurs at order completion",
		}, nil
	}

	// Guard: ConfirmOffload must have been called first
	if state != "AWAITING_PAYMENT" {
		return nil, fmt.Errorf("order %s must be AWAITING_PAYMENT (current: %s) — call ConfirmOffload first", orderID, state)
	}

	if supplierID == "" {
		return nil, fmt.Errorf("order %s has no supplier assignment", orderID)
	}

	// Recalculate adjusted total from line items (reflects AmendOrder damage edits).
	// AmendOrder writes Quantity = AcceptedQty — so SUM(UnitPrice * Quantity) is the
	// post-edit adjusted subtotal for THIS supplier's order only.
	adjustedStmt := spanner.Statement{
		SQL:    `SELECT COALESCE(SUM(UnitPrice * Quantity), 0) FROM OrderLineItems WHERE OrderId = @oid`,
		Params: map[string]interface{}{"oid": orderID},
	}
	adjustedIter := s.Client.Single().Query(ctx, adjustedStmt)
	adjustedRow, adjustedErr := adjustedIter.Next()
	adjustedIter.Stop()
	if adjustedErr != nil {
		return nil, fmt.Errorf("failed to recalculate line items for order %s: %w", orderID, adjustedErr)
	}
	var adjustedAmount int64
	if err := adjustedRow.Columns(&adjustedAmount); err != nil {
		return nil, fmt.Errorf("sum parse error for order %s: %w", orderID, err)
	}
	if adjustedAmount <= 0 {
		return nil, fmt.Errorf("adjusted amount for order %s is %d — nothing to charge", orderID, adjustedAmount)
	}

	// Geofence re-check: driver must still be within 100m of the retailer.
	// Uses the same Redis GEO sorted set populated by proximity.ProcessPing.
	// Graceful degradation: if Redis is nil or geofence data is missing, we proceed
	// (the initial 100m check at ConfirmOffload already passed).
	if cache.Client != nil && driverID != "" {
		distCmd := cache.Client.GeoDist(ctx, cache.KeyGeoProximity, cache.DriverGeoMember(driverID), cache.RetailerGeoMember(retailerID), "m")
		if dist, geoErr := distCmd.Result(); geoErr == nil && dist > 100.0 {
			return nil, fmt.Errorf("GEOFENCE_VIOLATION: driver %s is %.0fm from retailer %s (max 100m)", driverID, dist, retailerID)
		}
		// geoErr != nil → key missing or Redis blip → proceed (graceful degradation)
	}

	// Resolve active payment session (created by ConfirmOffload)
	var sessionID string
	if s.SessionSvc != nil {
		activeSession, sessErr := s.SessionSvc.GetActiveSessionByOrder(ctx, orderID)
		if sessErr == nil && activeSession != nil {
			sessionID = activeSession.SessionID
			// Update session with recalculated amount if it drifted from amendment
			if activeSession.LockedAmount != adjustedAmount {
				slog.Warn("order.fulfillment_pay_amount_drift", "session_id", sessionID, "locked_amount", activeSession.LockedAmount, "adjusted_amount", adjustedAmount)
			}
		}
	}

	// ────────────────────────────────────────────────────────────────────────
	// Phase 2: External Payment — charge saved card or fall back to checkout
	// ────────────────────────────────────────────────────────────────────────

	// Resolve supplier credentials from vault (using order's actual gateway)
	if gateway == "" {
		gateway = "GLOBAL_PAY" // Default to Global Pay if not yet set
	}
	var merchantID, serviceID, secretKey, recipientID string
	if s.Vault != nil {
		cfg, vaultErr := s.Vault.GetDecryptedConfigByOrder(ctx, orderID, gateway)
		if vaultErr == nil {
			merchantID = cfg.MerchantId
			serviceID = cfg.ServiceId
			secretKey = cfg.SecretKey
			recipientID = cfg.RecipientId
		} else {
			slog.Warn("order.fulfillment_pay_vault_fallback", "order_id", orderID, "err", vaultErr)
		}
	}

	creds, credErr := payment.ResolveGlobalPayCredentials(merchantID, serviceID, secretKey)
	if credErr != nil {
		return nil, fmt.Errorf("payment credentials unavailable for order %s: %w", orderID, credErr)
	}

	// Supplier passthrough + platform fee split
	splitRecipients := payment.ComputeSplitRecipients(adjustedAmount, recipientID, s.feeBasisPoints())

	result := &FulfillmentPaymentResult{
		OrderID:    orderID,
		SupplierID: supplierID,
		RetailerID: retailerID,
		Amount:     adjustedAmount,
		SessionID:  sessionID,
	}

	// Try saved card → direct charge (no redirect needed)
	var directPaymentID string
	var directPaid bool

	if s.CardTokenSvc != nil && s.DirectClient != nil {
		savedCard, _ := s.CardTokenSvc.GetDefaultCard(ctx, retailerID, "GLOBAL_PAY")
		if savedCard != nil {
			slog.Info("order.fulfillment_pay_saved_card", "retailer_id", retailerID, "token_id", savedCard.TokenID, "amount", adjustedAmount)

			externalID := "fulfill-" + orderID
			if sessionID != "" {
				externalID = sessionID // Use session as idempotency key if available
			}

			initResult, initErr := s.DirectClient.InitPayment(ctx, creds, payment.DirectPaymentInitRequest{
				CardToken:  savedCard.ProviderCardToken,
				Amount:     adjustedAmount,
				OrderID:    orderID,
				SessionID:  sessionID,
				ExternalID: externalID,
				Recipients: splitRecipients,
			})
			if initErr != nil {
				slog.Warn("order.fulfillment_pay_direct_init_failed", "order_id", orderID, "err", initErr)
				// Fall through to hosted checkout below
			} else {
				directPaymentID = initResult.PaymentID

				if initResult.SecurityCheckURL != "" {
					// 3DS verification required — return URL for retailer
					result.PaymentID = directPaymentID
					result.CheckoutURL = initResult.SecurityCheckURL
					result.Status = "3DS_REQUIRED"
					result.Message = fmt.Sprintf("3D Secure verification required for %d", adjustedAmount)

					// Bind the direct payment to the session for webhook reconciliation
					if s.SessionSvc != nil && sessionID != "" {
						_ = s.SessionSvc.BindProviderCheckout(ctx, sessionID, "GLOBAL_PAY", "", initResult.SecurityCheckURL, directPaymentID, nil)
					}
					return result, nil
				}

				// No 3DS — perform charge immediately
				performResult, performErr := s.DirectClient.PerformPayment(ctx, creds, directPaymentID)
				if performErr != nil {
					slog.Warn("order.fulfillment_pay_direct_perform_failed", "order_id", orderID, "err", performErr)
					// Fall through to hosted checkout
				} else if performResult.Paid {
					directPaid = true
					// Continue to Phase 3 below
				} else {
					slog.Warn("order.fulfillment_pay_direct_unpaid", "order_id", orderID, "status", performResult.Status)
				}
			}
		}
	}

	// Hosted checkout fallback (no saved card, or direct charge failed)
	if !directPaid && directPaymentID == "" {
		paymentAccount, accountErr := s.lookupRetailerPaymentAccount(ctx, retailerID)
		if accountErr != nil {
			return nil, fmt.Errorf("retailer payment account lookup failed for order %s: %w", orderID, accountErr)
		}

		// Create attempt for the hosted session
		attemptID := ""
		if s.SessionSvc != nil && sessionID != "" {
			attempt, attemptErr := s.SessionSvc.CreateAttempt(ctx, sessionID, "GLOBAL_PAY")
			if attemptErr == nil {
				attemptID = attempt.AttemptID
			}
		}

		checkoutResult, checkoutErr := payment.CreateGlobalPayHostedCheckout(ctx, creds, payment.GlobalPayCheckoutRequest{
			OrderID:    orderID,
			InvoiceID:  fmt.Sprintf("INV-%s", GenerateSecureToken()),
			SessionID:  sessionID,
			AttemptID:  attemptID,
			Amount:     adjustedAmount,
			Account:    paymentAccount,
			Recipients: splitRecipients,
		})
		if checkoutErr != nil {
			if s.SessionSvc != nil && sessionID != "" {
				_ = s.SessionSvc.FailSession(ctx, sessionID, "FULFILLMENT_CHECKOUT_FAILED", checkoutErr.Error())
			}
			return nil, fmt.Errorf("hosted checkout creation failed for order %s: %w", orderID, checkoutErr)
		}

		result.CheckoutURL = checkoutResult.RedirectURL
		result.PaymentID = checkoutResult.ProviderReference
		result.Status = "CHECKOUT_REDIRECT"
		result.Message = fmt.Sprintf("Open checkout to pay %d for supplier %s", adjustedAmount, supplierID)

		// Bind checkout URL to session
		if s.SessionSvc != nil && sessionID != "" {
			_ = s.SessionSvc.BindProviderCheckout(ctx, sessionID, "GLOBAL_PAY", "", checkoutResult.RedirectURL, checkoutResult.ProviderReference, checkoutResult.ExpiresAt)
		}
		return result, nil
	}

	// If we got here with failed direct but had a PaymentID (3DS fallthrough shouldn't reach here)
	if !directPaid {
		// Direct init succeeded but perform failed — return hosted fallback
		// (Should not normally reach here, but defensive)
		result.Status = "CHECKOUT_REDIRECT"
		result.Message = "Direct payment failed — please complete via hosted checkout"
		return result, nil
	}

	// ────────────────────────────────────────────────────────────────────────
	// Phase 3: Commit — OCC-guarded state update, settle session, events
	// ────────────────────────────────────────────────────────────────────────

	_, commitErr := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Re-read version for OCC
		verRow, verErr := txn.ReadRow(ctx, "Orders", spanner.Key{orderID}, []string{"Version", "PaymentStatus"})
		if verErr != nil {
			return fmt.Errorf("order %s disappeared during payment: %w", orderID, verErr)
		}
		var currentVersion int64
		var currentPayStatus string
		if err := verRow.Columns(&currentVersion, &currentPayStatus); err != nil {
			return err
		}

		// Double-check idempotency inside transaction
		if currentPayStatus == "PAID" {
			return nil // Already settled by another request (race-safe)
		}

		newVersion := currentVersion + 1
		rowCount, updateErr := txn.Update(ctx, spanner.Statement{
			SQL: `UPDATE Orders SET PaymentStatus = 'PAID', Amount = @amount,
			      Version = @newVersion
			      WHERE OrderId = @id AND Version = @version`,
			Params: map[string]interface{}{
				"id":         orderID,
				"amount":     adjustedAmount,
				"newVersion": newVersion,
				"version":    currentVersion,
			},
		})
		if updateErr != nil {
			return fmt.Errorf("fulfillment payment commit failed: %w", updateErr)
		}
		if rowCount == 0 {
			return &ErrVersionConflict{OrderID: orderID, ExpectedVersion: currentVersion, ActualVersion: -1}
		}

		now := time.Now().UTC()
		traceID := telemetry.TraceIDFromContext(ctx)

		if err := outbox.EmitJSON(txn, "Order", orderID, kafkaEvents.EventFulfillmentPaymentCompleted, topicLogisticsEvents, map[string]interface{}{
			"order_id":    orderID,
			"supplier_id": supplierID,
			"retailer_id": retailerID,
			"driver_id":   driverID,
			"amount":      adjustedAmount,
			"payment_id":  directPaymentID,
			"gateway":     gateway,
			"timestamp":   now,
		}, traceID); err != nil {
			return fmt.Errorf("outbox emit FULFILLMENT_PAYMENT_COMPLETED: %w", err)
		}

		if err := outbox.EmitJSON(txn, "Order", orderID, kafkaEvents.EventFulfillmentPaid, topicLogisticsEvents, map[string]interface{}{
			"order_id":    orderID,
			"supplier_id": supplierID,
			"retailer_id": retailerID,
			"amount":      adjustedAmount,
			"timestamp":   now,
		}, traceID); err != nil {
			return fmt.Errorf("outbox emit FULFILLMENT_PAID: %w", err)
		}

		return nil
	})
	if commitErr != nil {
		// Payment was taken but Spanner update failed — log for manual reconciliation
		slog.Error("order.fulfillment_pay_commit_failed", "payment_id", directPaymentID, "order_id", orderID, "err", commitErr)
		return nil, fmt.Errorf("payment charged but status update failed — contact support (payment_id: %s): %w", directPaymentID, commitErr)
	}

	// Settle the payment session
	if s.SessionSvc != nil && sessionID != "" {
		if settleErr := s.SessionSvc.SettleSession(ctx, sessionID, directPaymentID); settleErr != nil {
			slog.Warn("order.fulfillment_pay_settle_failed", "session_id", sessionID, "payment_id", directPaymentID, "err", settleErr)
		}
	}

	// Redis cache invalidation — clear active orders cache for this retailer
	if cache.Client != nil {
		delCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		cache.Client.Del(delCtx, cache.PrefixActiveOrders+retailerID)
	}

	result.PaymentID = directPaymentID
	result.Status = "PAID"
	result.Message = fmt.Sprintf("Payment of %d completed for supplier %s via saved card", adjustedAmount, supplierID)

	slog.Info("order.fulfillment_pay_completed", "order_id", orderID, "amount", adjustedAmount, "retailer_id", retailerID, "supplier_id", supplierID, "payment_id", directPaymentID)

	return result, nil
}

// ─── VALIDATE QR TOKEN (READ-ONLY QR CHECK) ──────────────────────────────────

// ValidateQRResponse is returned when QR token is valid.
type ValidateQRResponse struct {
	OrderID        string     `json:"order_id"`
	RetailerID     string     `json:"retailer_id"`
	Amount         int64      `json:"amount"`
	PaymentGateway string     `json:"payment_gateway"`
	State          string     `json:"state"`
	Items          []LineItem `json:"items"`
}

// ValidateQRToken checks the scanned token against the stored delivery token
// using constant-time comparison. Does NOT change state. Sets QRValidatedAt
// timestamp so subsequent endpoints know QR passed.
// Idempotent: if QRValidatedAt is already set and the token matches, returns
// success without re-writing (safe for network-retry rescans by the driver).
func (s *OrderService) ValidateQRToken(ctx context.Context, orderId string, scannedToken string) (*ValidateQRResponse, error) {
	var resp ValidateQRResponse

	// Fast path: check Redis-cached token first (avoids Spanner RW transaction on replay/retry)
	if cache.Client != nil {
		cachedToken, redisErr := cache.Client.Get(ctx, cache.PrefixDeliveryToken+orderId).Result()
		if redisErr == nil && cachedToken != "" {
			if subtle.ConstantTimeCompare([]byte(cachedToken), []byte(scannedToken)) != 1 {
				return nil, fmt.Errorf("INVALID QR TOKEN: Cryptographic handshake failed. Delivery blocked.")
			}
			// Token matches — proceed to Spanner for state check + QRValidatedAt stamp
		}
		// Redis miss or error → fall through to Spanner (graceful degradation)
	}

	_, err := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "Orders", spanner.Key{orderId},
			[]string{"State", "DeliveryToken", "RetailerId", "Amount", "PaymentGateway", "QRValidatedAt"})
		if err != nil {
			return fmt.Errorf("order %s not found: %w", orderId, err)
		}

		var state string
		var trueToken spanner.NullString
		var gateway spanner.NullString
		var qrValidatedAt spanner.NullTime
		if err := row.Columns(&state, &trueToken, &resp.RetailerID, &resp.Amount, &gateway, &qrValidatedAt); err != nil {
			return err
		}
		resp.PaymentGateway = gateway.StringVal

		if state != "ARRIVED" {
			return fmt.Errorf("order %s must be ARRIVED to validate QR (current state: %s)", orderId, state)
		}

		if !trueToken.Valid || subtle.ConstantTimeCompare([]byte(trueToken.StringVal), []byte(scannedToken)) != 1 {
			return fmt.Errorf("INVALID QR TOKEN: Cryptographic handshake failed. Delivery blocked.")
		}

		// Idempotency: if QR was already validated (e.g. network retry), skip the write
		if qrValidatedAt.Valid {
			resp.OrderID = orderId
			resp.State = state
			return nil
		}

		// Stamp QRValidatedAt — subsequent endpoints check this
		stmt := spanner.Statement{
			SQL:    `UPDATE Orders SET QRValidatedAt = CURRENT_TIMESTAMP() WHERE OrderId = @id`,
			Params: map[string]interface{}{"id": orderId},
		}
		if _, err := txn.Update(ctx, stmt); err != nil {
			return fmt.Errorf("failed to stamp QRValidatedAt: %w", err)
		}

		resp.OrderID = orderId
		resp.State = state
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Hydrate line items
	itemStmt := spanner.Statement{
		SQL: `SELECT li.LineItemId, li.OrderId, li.SkuId, COALESCE(sp.Name, li.SkuId) AS SkuName, li.Quantity, li.UnitPrice, li.Status
		      FROM OrderLineItems li
		      LEFT JOIN SupplierProducts sp ON li.SkuId = sp.SkuId
		      WHERE li.OrderId = @oid`,
		Params: map[string]interface{}{"oid": orderId},
	}
	iter := s.Client.Single().Query(ctx, itemStmt)
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			break
		}
		var li LineItem
		if err := row.Columns(&li.LineItemID, &li.OrderID, &li.SkuID, &li.SkuName, &li.Quantity, &li.UnitPrice, &li.Status); err != nil {
			continue
		}
		resp.Items = append(resp.Items, li)
	}
	if resp.Items == nil {
		resp.Items = []LineItem{}
	}

	return &resp, nil
}

// ─── CONFIRM OFFLOAD ──────────────────────────────────────────────────────────

// ConfirmOffloadResponse is returned after offload confirmation.
type ConfirmOffloadResponse struct {
	OrderID               string   `json:"order_id"`
	State                 string   `json:"state"`
	PaymentMethod         string   `json:"payment_method"`
	AvailableCardGateways []string `json:"available_card_gateways,omitempty"`
	Amount                int64    `json:"amount"`
	OriginalAmount        int64    `json:"original_amount"`
	InvoiceID             string   `json:"invoice_id,omitempty"` // GlobalPay invoice ID for retailer payment
	SessionID             string   `json:"session_id,omitempty"` // Payment session ID
	RetailerID            string   `json:"retailer_id"`
	SupplierID            string   `json:"supplier_id"`
	Message               string   `json:"message"`
}

// OffloadConfirmedEvent — canonical definition lives in kafka/events.go.

// ConfirmOffload transitions ARRIVED → AWAITING_PAYMENT after QR validation.
// For GlobalPay: creates an invoice record so the retailer can pay.
// For Cash: driver collects cash.
func (s *OrderService) ConfirmOffload(ctx context.Context, orderId string) (*ConfirmOffloadResponse, error) {
	var resp ConfirmOffloadResponse
	var supplierId string

	_, err := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "Orders", spanner.Key{orderId},
			[]string{"State", "QRValidatedAt", "RetailerId", "Amount", "PaymentGateway", "SupplierId", "Version"})
		if err != nil {
			return fmt.Errorf("order %s not found: %w", orderId, err)
		}

		var state string
		var qrValidatedAt spanner.NullTime
		var gateway spanner.NullString
		var supplierIdNull spanner.NullString
		var version int64
		if err := row.Columns(&state, &qrValidatedAt, &resp.RetailerID, &resp.Amount, &gateway, &supplierIdNull, &version); err != nil {
			return err
		}
		if supplierIdNull.Valid {
			supplierId = supplierIdNull.StringVal
		}

		if state != "ARRIVED" {
			return fmt.Errorf("order %s must be ARRIVED to confirm offload (current: %s)", orderId, state)
		}
		if !qrValidatedAt.Valid {
			return fmt.Errorf("order %s has not been QR-validated — scan QR first", orderId)
		}

		resp.PaymentMethod = gateway.StringVal
		resp.OrderID = orderId

		// Compute original (pre-amendment) total: accepted qty + rejected qty at original unit price
		origStmt := spanner.Statement{
			SQL:    `SELECT COALESCE(SUM(UnitPrice * (Quantity + COALESCE(RejectedQty, 0))), 0) FROM OrderLineItems WHERE OrderId = @oid`,
			Params: map[string]interface{}{"oid": orderId},
		}
		origIter := txn.Query(ctx, origStmt)
		origRow, origErr := origIter.Next()
		origIter.Stop()
		if origErr == nil {
			var origTotal int64
			if scanErr := origRow.Columns(&origTotal); scanErr == nil {
				resp.OriginalAmount = origTotal
			}
		}
		// Fallback: if no rejections exist, original == current
		if resp.OriginalAmount == 0 {
			resp.OriginalAmount = resp.Amount
		}

		// Transition to AWAITING_PAYMENT and set PaymentStatus = PENDING
		newVersion := version + 1
		rowCount, err := txn.Update(ctx, spanner.Statement{
			SQL: `UPDATE Orders SET State = 'AWAITING_PAYMENT', PaymentStatus = 'PENDING', Version = @newVersion
			      WHERE OrderId = @id AND Version = @version`,
			Params: map[string]interface{}{
				"id":         orderId,
				"newVersion": newVersion,
				"version":    version,
			},
		})
		if err != nil {
			return fmt.Errorf("offload state transition failed: %w", err)
		}
		if rowCount == 0 {
			return &ErrVersionConflict{OrderID: orderId, ExpectedVersion: version, ActualVersion: -1}
		}

		// For GlobalPay: create a MasterInvoice so the webhook can settle it
		if strings.EqualFold(gateway.StringVal, "GLOBAL_PAY") {
			invoiceID := fmt.Sprintf("INV-%s", GenerateSecureToken())
			resp.InvoiceID = invoiceID
			if err := txn.BufferWrite([]*spanner.Mutation{
				spanner.Insert("MasterInvoices",
					[]string{"InvoiceId", "RetailerId", "Total", "State", "OrderId", "CreatedAt"},
					[]interface{}{invoiceID, resp.RetailerID, resp.Amount, "PENDING", orderId, spanner.CommitTimestamp},
				),
			}); err != nil {
				return fmt.Errorf("create master invoice: %w", err)
			}
		}

		now := time.Now().UTC()
		traceID := telemetry.TraceIDFromContext(ctx)

		if err := outbox.EmitJSON(txn, "Order", orderId, kafkaEvents.EventOffloadConfirmed, topicLogisticsEvents, kafkaEvents.OffloadConfirmedEvent{
			OrderID:        orderId,
			RetailerID:     resp.RetailerID,
			Amount:         resp.Amount,
			OriginalAmount: resp.OriginalAmount,
			PaymentMethod:  resp.PaymentMethod,
			Timestamp:      now,
		}, traceID); err != nil {
			return fmt.Errorf("outbox emit OFFLOAD_CONFIRMED: %w", err)
		}

		if err := outbox.EmitJSON(txn, "Order", orderId, kafkaEvents.EventOrderStatusChanged, topicLogisticsEvents, kafkaEvents.OrderStatusChangedEvent{
			OrderID:    orderId,
			RetailerID: resp.RetailerID,
			SupplierID: supplierId,
			OldState:   "ARRIVED",
			NewState:   "AWAITING_PAYMENT",
			Timestamp:  now,
		}, traceID); err != nil {
			return fmt.Errorf("outbox emit ORDER_STATUS_CHANGED (arrived->awaiting_payment): %w", err)
		}

		resp.State = "AWAITING_PAYMENT"
		return nil
	})
	if err != nil {
		return nil, err
	}
	resp.AvailableCardGateways = s.resolveAvailableCardGateways(ctx, supplierId, resp.PaymentMethod)
	resp.SupplierID = supplierId

	if strings.EqualFold(resp.PaymentMethod, "CASH") {
		resp.Message = fmt.Sprintf("Collect %d cash from retailer", resp.Amount)
	} else {
		resp.Message = "Payment request sent to retailer"
	}

	// Create a durable payment session for electronic payments
	if s.SessionSvc != nil && !strings.EqualFold(resp.PaymentMethod, "CASH") {
		session, sessionErr := s.SessionSvc.CreateSession(ctx, payment.CreateSessionRequest{
			OrderID:    orderId,
			RetailerID: resp.RetailerID,
			SupplierID: supplierId,
			Gateway:    strings.ToUpper(resp.PaymentMethod),
			Amount:     resp.Amount,
			InvoiceID:  resp.InvoiceID,
		})
		if sessionErr != nil {
			slog.Error("order.confirm_offload_session_failed", "order_id", orderId, "err", sessionErr)
		} else {
			resp.SessionID = session.SessionID
		}
	}

	return &resp, nil
}

func (s *OrderService) resolveAvailableCardGateways(ctx context.Context, supplierId, fallbackGateway string) []string {
	if s.Vault != nil && strings.TrimSpace(supplierId) != "" {
		gateways, err := s.Vault.ListActiveGatewayNames(ctx, supplierId)
		if err != nil {
			slog.Error("order.confirm_offload_list_gateways_failed", "supplier_id", supplierId, "err", err)
		} else if len(gateways) > 0 {
			return gateways
		}
	}
	if normalized := normalizeCardGateway(fallbackGateway); normalized != "" {
		return []string{normalized}
	}
	return nil
}

func normalizeCardGateway(gateway string) string {
	switch strings.ToUpper(strings.TrimSpace(gateway)) {
	case "CASH", "GLOBAL_PAY":
		return strings.ToUpper(strings.TrimSpace(gateway))
	default:
		return ""
	}
}

// ─── COMPLETE ORDER ───────────────────────────────────────────────────────────

// CompleteOrder finalizes the delivery after payment is settled (or cash collected).
// Guard: State must be AWAITING_PAYMENT.
// For GlobalPay: verifies the MasterInvoice is SETTLED before allowing completion.
// For Cash: trusts the driver's confirmation.
// Returns the owning supplierID for WebSocket push notification.
func (s *OrderService) CompleteOrder(ctx context.Context, orderId string) (string, error) {
	var retailerId string
	var supplierID string
	var warehouseId string

	_, err := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "Orders", spanner.Key{orderId},
			[]string{"State", "RetailerId", "PaymentGateway", "Version", "SupplierId", "WarehouseId", "ManifestId", "Amount", "Currency"})
		if err != nil {
			return fmt.Errorf("order %s not found: %w", orderId, err)
		}

		var state string
		var gateway spanner.NullString
		var version int64
		var sid spanner.NullString
		var wid spanner.NullString
		var manifestIDNull spanner.NullString
		var orderAmount spanner.NullInt64
		var orderCurrency spanner.NullString
		if err := row.Columns(&state, &retailerId, &gateway, &version, &sid, &wid, &manifestIDNull, &orderAmount, &orderCurrency); err != nil {
			return err
		}
		if sid.Valid {
			supplierID = sid.StringVal
		}
		if wid.Valid {
			warehouseId = wid.StringVal
		}

		if state == "COMPLETED" {
			return nil // Idempotent
		}
		if state != "AWAITING_PAYMENT" {
			return fmt.Errorf("order %s must be AWAITING_PAYMENT to complete (current: %s)", orderId, state)
		}

		// For electronic payments: verify payment is settled
		gw := strings.ToUpper(gateway.StringVal)
		if gw == "GLOBAL_PAY" || gw == "CASH" {
			settled := false

			// Primary check: payment session
			if s.SessionSvc != nil {
				activeSession, sessErr := s.SessionSvc.GetActiveSessionByOrder(ctx, orderId)
				if sessErr == nil && activeSession != nil && activeSession.Status == "SETTLED" {
					settled = true
				}
			}

			// Fallback: check MasterInvoice state
			if !settled {
				invStmt := spanner.Statement{
					SQL:    `SELECT State FROM MasterInvoices WHERE OrderId = @oid LIMIT 1`,
					Params: map[string]interface{}{"oid": orderId},
				}
				invIter := txn.Query(ctx, invStmt)
				invRow, invErr := invIter.Next()
				invIter.Stop()
				if invErr != nil {
					return fmt.Errorf("payment invoice not found for order %s", orderId)
				}
				var invState string
				if err := invRow.Columns(&invState); err != nil {
					return err
				}
				if invState == "SETTLED" {
					settled = true
				}
			}

			if !settled {
				return fmt.Errorf("payment not yet settled for order %s (gateway: %s)", orderId, gw)
			}
		}

		// Transition to COMPLETED
		newVersion := version + 1
		rowCount, err := txn.Update(ctx, spanner.Statement{
			SQL: `UPDATE Orders SET State = 'COMPLETED', PaymentStatus = 'PAID', Version = @newVersion, LockedUntil = NULL
			      WHERE OrderId = @id AND Version = @version`,
			Params: map[string]interface{}{
				"id":         orderId,
				"newVersion": newVersion,
				"version":    version,
			},
		})
		if err != nil {
			return fmt.Errorf("completion failed: %w", err)
		}
		if rowCount == 0 {
			return &ErrVersionConflict{OrderID: orderId, ExpectedVersion: version, ActualVersion: -1}
		}

		// LEO Phase V — manifest-completion rollup. Atomic with the order
		// completion: if this was the last non-COMPLETED stop on the manifest,
		// the parent SupplierTruckManifests row advances to COMPLETED and a
		// MANIFEST_COMPLETED event is emitted via outbox.
		cur := "UZS"
		if orderCurrency.Valid && orderCurrency.StringVal != "" {
			cur = orderCurrency.StringVal
		}
		if err := outbox.EmitJSON(txn, "Order", orderId, kafkaEvents.EventOrderCompleted, kafkaEvents.TopicMain, kafkaEvents.OrderCompletedEvent{
			OrderID:     orderId,
			RetailerID:  retailerId,
			SupplierId:  supplierID,
			WarehouseId: warehouseId,
			Amount:      orderAmount.Int64,
			Currency:    cur,
			Timestamp:   time.Now().UTC(),
		}, telemetry.TraceIDFromContext(ctx)); err != nil {
			return fmt.Errorf("outbox emit ORDER_COMPLETED: %w", err)
		}

		if err := outbox.EmitJSON(txn, "Order", orderId, kafkaEvents.EventOrderStatusChanged, kafkaEvents.TopicMain, kafkaEvents.OrderStatusChangedEvent{
			OrderID:    orderId,
			RetailerID: retailerId,
			SupplierID: supplierID,
			OldState:   "AWAITING_PAYMENT",
			NewState:   "COMPLETED",
			Timestamp:  time.Now().UTC(),
		}, telemetry.TraceIDFromContext(ctx)); err != nil {
			return fmt.Errorf("outbox emit ORDER_STATUS_CHANGED (awaiting_payment->completed): %w", err)
		}

		if manifestIDNull.Valid {
			if err := rollupManifestIfComplete(ctx, txn, manifestIDNull.StringVal, time.Now().UTC()); err != nil {
				return fmt.Errorf("manifest rollup failed for order %s: %w", orderId, err)
			}
		}

		return nil
	})

	if err == nil {
		// Decrement warehouse queue depth on successful completion
		cache.DecrementQueueDepth(context.Background(), warehouseId)
	}

	return supplierID, err
}

// ─── CARD CHECKOUT (Retailer selects a card gateway after offload) ────────────

// CardCheckoutResponse is returned when the retailer selects a card gateway.
type CardCheckoutResponse struct {
	OrderID    string `json:"order_id"`
	State      string `json:"state"`
	Amount     int64  `json:"amount"`
	Gateway    string `json:"gateway"`
	PaymentURL string `json:"payment_url"`
	InvoiceID  string `json:"invoice_id"`
	SessionID  string `json:"session_id,omitempty"`
	AttemptID  string `json:"attempt_id,omitempty"`
	AttemptNo  int64  `json:"attempt_no,omitempty"`
	RetailerID string `json:"retailer_id"`
	Message    string `json:"message"`
}

// CardCheckout transitions AWAITING_PAYMENT → keeps AWAITING_PAYMENT (no state change — webhook settles).
// Creates a MasterInvoice for the gateway and returns a hosted checkout URL.
func (s *OrderService) CardCheckout(ctx context.Context, orderId, gateway, callbackBaseURL string) (*CardCheckoutResponse, error) {
	gateway = strings.ToUpper(gateway)
	if gateway != "GLOBAL_PAY" && gateway != "CASH" {
		return nil, fmt.Errorf("unsupported card gateway: %s (supported: GLOBAL_PAY, CASH)", gateway)
	}

	var resp CardCheckoutResponse
	var retailerId string
	var supplierId string

	_, err := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "Orders", spanner.Key{orderId},
			[]string{"State", "RetailerId", "SupplierId", "Amount", "PaymentGateway", "Version"})
		if err != nil {
			return fmt.Errorf("order %s not found: %w", orderId, err)
		}

		var state string
		var supplierIdNull spanner.NullString
		var existingGateway spanner.NullString
		var version int64
		if err := row.Columns(&state, &retailerId, &supplierIdNull, &resp.Amount, &existingGateway, &version); err != nil {
			return err
		}
		if supplierIdNull.Valid {
			supplierId = supplierIdNull.StringVal
		}

		if state != "AWAITING_PAYMENT" {
			return fmt.Errorf("order %s must be AWAITING_PAYMENT to initiate card payment (current: %s)", orderId, state)
		}

		resp.RetailerID = retailerId

		newVersion := version + 1
		_, err = txn.Update(ctx, spanner.Statement{
			SQL: `UPDATE Orders SET PaymentGateway = @gw, PaymentStatus = 'AWAITING_GATEWAY_WEBHOOK', Version = @newVersion
			      WHERE OrderId = @id AND Version = @version`,
			Params: map[string]interface{}{
				"id":         orderId,
				"gw":         gateway,
				"newVersion": newVersion,
				"version":    version,
			},
		})
		if err != nil {
			return fmt.Errorf("card checkout update failed: %w", err)
		}

		invStmt := spanner.Statement{
			SQL:    `SELECT InvoiceId FROM MasterInvoices WHERE OrderId = @oid LIMIT 1`,
			Params: map[string]interface{}{"oid": orderId},
		}
		invIter := txn.Query(ctx, invStmt)
		invRow, invErr := invIter.Next()
		invIter.Stop()

		if invErr == nil {
			var existingInvID string
			if colErr := invRow.Columns(&existingInvID); colErr == nil {
				resp.InvoiceID = existingInvID
				return nil
			}
		}

		invoiceID := fmt.Sprintf("INV-%s", GenerateSecureToken())
		resp.InvoiceID = invoiceID
		txn.BufferWrite([]*spanner.Mutation{
			spanner.Insert("MasterInvoices",
				[]string{"InvoiceId", "RetailerId", "Total", "State", "OrderId", "PaymentMode", "CreatedAt"},
				[]interface{}{invoiceID, retailerId, resp.Amount, "PENDING", orderId, gateway, spanner.CommitTimestamp},
			),
		})

		resp.OrderID = orderId
		resp.State = "AWAITING_PAYMENT"
		resp.Gateway = gateway
		return nil
	})
	if err != nil {
		return nil, err
	}

	if s.SessionSvc != nil {
		activeSession, sessErr := s.SessionSvc.GetActiveSessionByOrder(ctx, orderId)
		if sessErr != nil {
			activeSession, sessErr = s.SessionSvc.CreateSession(ctx, payment.CreateSessionRequest{
				OrderID:    orderId,
				RetailerID: retailerId,
				SupplierID: supplierId,
				Gateway:    gateway,
				Amount:     resp.Amount,
				InvoiceID:  resp.InvoiceID,
			})
			if sessErr != nil {
				slog.Error("order.card_checkout_session_failed", "order_id", orderId, "err", sessErr)
				activeSession = nil
			}
		}
		if activeSession != nil {
			resp.SessionID = activeSession.SessionID
			attempt, attemptErr := s.SessionSvc.CreateAttempt(ctx, activeSession.SessionID, gateway)
			if attemptErr != nil {
				slog.Error("order.card_checkout_attempt_failed", "session_id", activeSession.SessionID, "err", attemptErr)
			} else {
				resp.AttemptID = attempt.AttemptID
				resp.AttemptNo = attempt.AttemptNo
			}
		}
	}

	var paymentURL string
	var providerReference string
	var expiresAt *time.Time
	var urlErr error
	var merchantID string
	var serviceID string
	var secretKey string
	var recipientID string
	if s.Vault != nil {
		cfg, vaultErr := s.Vault.GetDecryptedConfigByOrder(ctx, orderId, gateway)
		if vaultErr == nil {
			merchantID = cfg.MerchantId
			serviceID = cfg.ServiceId
			secretKey = cfg.SecretKey
			recipientID = cfg.RecipientId
		} else {
			slog.Warn("order.card_checkout_vault_fallback", "order_id", orderId, "gateway", gateway, "err", vaultErr)
		}
	}

	if gateway == "GLOBAL_PAY" {
		if resp.SessionID == "" {
			return nil, fmt.Errorf("global pay checkout requires a payment session")
		}
		creds, credErr := payment.ResolveGlobalPayCredentials(merchantID, serviceID, secretKey)
		if credErr != nil {
			return nil, credErr
		}

		// Compute split recipients if configured
		splitRecipients := payment.ComputeSplitRecipients(resp.Amount, recipientID, s.feeBasisPoints())

		// ── Dual-mode: check for saved card → direct charge, else hosted checkout ──
		var savedCard *payment.RetailerCardToken
		if s.CardTokenSvc != nil && s.DirectClient != nil {
			savedCard, _ = s.CardTokenSvc.GetDefaultCard(ctx, retailerId, "GLOBAL_PAY")
		}

		if savedCard != nil {
			// DIRECT GATEWAY PATH: charge the saved card immediately
			slog.Info("order.card_checkout_saved_card", "retailer_id", retailerId, "token_id", savedCard.TokenID)
			initResult, initErr := s.DirectClient.InitPayment(ctx, creds, payment.DirectPaymentInitRequest{
				CardToken:  savedCard.ProviderCardToken,
				Amount:     resp.Amount,
				OrderID:    orderId,
				SessionID:  resp.SessionID,
				ExternalID: resp.AttemptID,
				Recipients: splitRecipients,
			})
			if initErr != nil {
				slog.Warn("order.card_checkout_direct_init_failed", "order_id", orderId, "err", initErr)
				// Fall through to hosted checkout below
			} else {
				providerReference = initResult.PaymentID

				if initResult.SecurityCheckURL != "" {
					// 3DS required — return the URL for retailer to complete verification
					paymentURL = initResult.SecurityCheckURL
					resp.Message = fmt.Sprintf("3D Secure verification required for %d", resp.Amount)
				} else {
					// No 3DS — perform the charge immediately
					performResult, performErr := s.DirectClient.PerformPayment(ctx, creds, initResult.PaymentID)
					if performErr != nil {
						urlErr = performErr
					} else if performResult.Paid {
						// Payment complete — settle immediately
						if s.SessionSvc != nil && resp.SessionID != "" {
							if settleErr := s.SessionSvc.SettleSession(ctx, resp.SessionID, initResult.PaymentID); settleErr != nil {
								slog.Error("order.card_checkout_settle_failed", "session_id", resp.SessionID, "err", settleErr)
							}
						}
						resp.PaymentURL = ""
						resp.Message = fmt.Sprintf("Payment of %d completed via saved card", resp.Amount)
						resp.OrderID = orderId
						resp.State = "AWAITING_PAYMENT" // Webhook or reconciler will transition to COMPLETED
						resp.Gateway = gateway
						return &resp, nil
					} else {
						urlErr = fmt.Errorf("direct payment perform returned unpaid status: %s", performResult.Status)
					}
				}

				if urlErr == nil {
					// Bind the direct payment reference to the session
					if s.SessionSvc != nil && resp.SessionID != "" {
						if bindErr := s.SessionSvc.BindProviderCheckout(ctx, resp.SessionID, gateway, resp.InvoiceID, paymentURL, providerReference, nil); bindErr != nil {
							slog.Error("order.card_checkout_bind_direct_failed", "session_id", resp.SessionID, "err", bindErr)
						}
					}
					resp.PaymentURL = paymentURL
					resp.OrderID = orderId
					resp.State = "AWAITING_PAYMENT"
					resp.Gateway = gateway
					return &resp, nil
				}

				// Direct payment failed — fall through to hosted checkout
				slog.Warn("order.card_checkout_direct_failed", "order_id", orderId, "err", urlErr)
				urlErr = nil
			}
		}

		// HOSTED CHECKOUT FALLBACK (no saved card or direct payment failed)
		paymentAccount, accountErr := s.lookupRetailerPaymentAccount(ctx, retailerId)
		if accountErr != nil {
			return nil, fmt.Errorf("global pay account lookup failed: %w", accountErr)
		}
		checkoutReq := payment.GlobalPayCheckoutRequest{
			OrderID:         orderId,
			InvoiceID:       resp.InvoiceID,
			SessionID:       resp.SessionID,
			AttemptID:       resp.AttemptID,
			Amount:          resp.Amount,
			Account:         paymentAccount,
			CallbackBaseURL: callbackBaseURL,
			Recipients:      splitRecipients,
		}
		checkoutResult, checkoutErr := payment.CreateGlobalPayHostedCheckout(ctx, creds, checkoutReq)
		if checkoutErr != nil {
			urlErr = checkoutErr
			if s.SessionSvc != nil {
				if failErr := s.SessionSvc.FailSession(ctx, resp.SessionID, "GLOBAL_PAY_INIT_FAILED", checkoutErr.Error()); failErr != nil {
					slog.Error("order.card_checkout_session_fail_failed", "session_id", resp.SessionID, "err", failErr)
				}
			}
		} else {
			paymentURL = checkoutResult.RedirectURL
			providerReference = checkoutResult.ProviderReference
			expiresAt = checkoutResult.ExpiresAt
		}
	} else {
		if merchantID != "" || serviceID != "" {
			paymentURL, urlErr = payment.CheckoutURLWithCredentials(gateway, orderId, resp.Amount, merchantID, serviceID)
		} else {
			paymentURL, urlErr = payment.CheckoutURL(gateway, orderId, resp.Amount)
		}
	}
	if urlErr != nil {
		slog.Error("order.card_checkout_url_failed", "gateway", gateway, "order_id", orderId, "err", urlErr)
		resp.Message = fmt.Sprintf("Invoice created but deep-link URL unavailable: %v", urlErr)
	} else {
		resp.PaymentURL = paymentURL
		resp.Message = fmt.Sprintf("Open %s to pay %d", gateway, resp.Amount)
		if s.SessionSvc != nil && resp.SessionID != "" {
			if bindErr := s.SessionSvc.BindProviderCheckout(ctx, resp.SessionID, gateway, resp.InvoiceID, paymentURL, providerReference, expiresAt); bindErr != nil {
				slog.Error("order.card_checkout_bind_failed", "session_id", resp.SessionID, "err", bindErr)
			}
		}
	}

	resp.OrderID = orderId
	resp.State = "AWAITING_PAYMENT"
	resp.Gateway = gateway

	return &resp, nil
}

func (s *OrderService) lookupRetailerPaymentAccount(ctx context.Context, retailerID string) (string, error) {
	row, err := s.Client.Single().ReadRow(ctx, "Retailers", spanner.Key{retailerID}, []string{"Phone"})
	if err != nil {
		return "", err
	}
	var phone spanner.NullString
	if err := row.Columns(&phone); err != nil {
		return "", err
	}
	if phone.Valid && strings.TrimSpace(phone.StringVal) != "" {
		return strings.TrimSpace(phone.StringVal), nil
	}
	return retailerID, nil
}

// LookupRetailerPhone is the exported version of lookupRetailerPaymentAccount
// for use by card management endpoints in main.go.
func (s *OrderService) LookupRetailerPhone(ctx context.Context, retailerID string) (string, error) {
	return s.lookupRetailerPaymentAccount(ctx, retailerID)
}

// ─── CASH CHECKOUT (Retailer selects cash after offload) ──────────────────────

// CashCheckoutResponse is returned when the retailer selects cash as the payment method.
type CashCheckoutResponse struct {
	OrderID    string `json:"order_id"`
	State      string `json:"state"`
	Amount     int64  `json:"amount"`
	DriverID   string `json:"driver_id,omitempty"`
	RetailerID string `json:"retailer_id"`
	Message    string `json:"message"`
}

// CashCheckout transitions AWAITING_PAYMENT → PENDING_CASH_COLLECTION.
// Called by the retailer after offload to confirm they will pay cash.
// Creates a cash-custody MasterInvoice record for reconciliation.
func (s *OrderService) CashCheckout(ctx context.Context, orderId string) (*CashCheckoutResponse, error) {
	var resp CashCheckoutResponse
	var retailerId, supplierID string

	_, err := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "Orders", spanner.Key{orderId},
			[]string{"State", "RetailerId", "Amount", "DriverId", "Version", "SupplierId"})
		if err != nil {
			return fmt.Errorf("order %s not found: %w", orderId, err)
		}

		var state string
		var driverId, sid spanner.NullString
		var version int64
		if err := row.Columns(&state, &retailerId, &resp.Amount, &driverId, &version, &sid); err != nil {
			return err
		}

		resp.RetailerID = retailerId
		if driverId.Valid {
			resp.DriverID = driverId.StringVal
		}
		if sid.Valid {
			supplierID = sid.StringVal
		}

		if state == "PENDING_CASH_COLLECTION" {
			resp.OrderID = orderId
			resp.State = "PENDING_CASH_COLLECTION"
			resp.Message = fmt.Sprintf("Cash collection of %d is already pending", resp.Amount)
			return nil // Idempotent
		}
		if state != "AWAITING_PAYMENT" {
			return fmt.Errorf("order %s must be AWAITING_PAYMENT to select cash (current: %s)", orderId, state)
		}

		// Transition to PENDING_CASH_COLLECTION
		newVersion := version + 1
		rowCount, err := txn.Update(ctx, spanner.Statement{
			SQL: `UPDATE Orders SET State = 'PENDING_CASH_COLLECTION', PaymentGateway = 'CASH', PaymentStatus = 'PENDING_CASH', Version = @newVersion
			      WHERE OrderId = @id AND Version = @version`,
			Params: map[string]interface{}{
				"id":         orderId,
				"newVersion": newVersion,
				"version":    version,
			},
		})
		if err != nil {
			return fmt.Errorf("cash checkout transition failed: %w", err)
		}
		if rowCount == 0 {
			return &ErrVersionConflict{OrderID: orderId, ExpectedVersion: version, ActualVersion: -1}
		}

		// Create a cash-custody MasterInvoice for reconciliation tracking
		invoiceID := fmt.Sprintf("INV-%s", GenerateSecureToken())
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.Insert("MasterInvoices",
				[]string{"InvoiceId", "RetailerId", "Total", "State", "OrderId", "PaymentMode", "CustodyStatus", "CreatedAt"},
				[]interface{}{invoiceID, retailerId, resp.Amount, "PENDING", orderId, "CASH", "PENDING", spanner.CommitTimestamp},
			),
		}); err != nil {
			return fmt.Errorf("create cash custody invoice: %w", err)
		}

		now := time.Now().UTC()
		traceID := telemetry.TraceIDFromContext(ctx)

		if err := outbox.EmitJSON(txn, "Order", orderId, kafkaEvents.EventCashCollectionRequired, topicLogisticsEvents, map[string]interface{}{
			"order_id":    orderId,
			"retailer_id": retailerId,
			"amount":      resp.Amount,
			"timestamp":   now,
		}, traceID); err != nil {
			return fmt.Errorf("outbox emit CASH_COLLECTION_REQUIRED: %w", err)
		}

		if err := outbox.EmitJSON(txn, "Order", orderId, kafkaEvents.EventOrderStatusChanged, topicLogisticsEvents, kafkaEvents.OrderStatusChangedEvent{
			OrderID:    orderId,
			RetailerID: retailerId,
			SupplierID: supplierID,
			OldState:   "AWAITING_PAYMENT",
			NewState:   "PENDING_CASH_COLLECTION",
			Timestamp:  now,
		}, traceID); err != nil {
			return fmt.Errorf("outbox emit ORDER_STATUS_CHANGED (awaiting_payment->pending_cash_collection): %w", err)
		}

		resp.OrderID = orderId
		resp.State = "PENDING_CASH_COLLECTION"
		return nil
	})
	if err != nil {
		return nil, err
	}

	resp.Message = fmt.Sprintf("Cash collection of %d is now pending — driver will collect", resp.Amount)

	return &resp, nil
}

// ─── COLLECT CASH (Driver confirms geofenced cash collection) ─────────────────

// CollectCashRequest is the body for POST /v1/deliveries/{delivery_id}/collect-cash.
type CollectCashRequest struct {
	OrderID   string  `json:"order_id"`
	DriverID  string  `json:"driver_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// CollectCashResponse is returned after successful cash collection.
type CollectCashResponse struct {
	OrderID    string  `json:"order_id"`
	State      string  `json:"state"`
	Amount     int64   `json:"amount"`
	DistanceM  float64 `json:"distance_m"`
	RetailerID string  `json:"retailer_id"`
	Message    string  `json:"message"`
}

// CollectCash transitions PENDING_CASH_COLLECTION → COMPLETED after GPS-validated cash collection.
// Validates the driver is within the retailer's geofence (500m threshold).
// Updates the cash-custody MasterInvoice with collection metadata.
func (s *OrderService) CollectCash(ctx context.Context, req CollectCashRequest) (*CollectCashResponse, error) {
	var resp CollectCashResponse
	var retailerId, supplierID, warehouseId string

	_, err := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "Orders", spanner.Key{req.OrderID},
			[]string{"State", "RetailerId", "Amount", "ShopLocation", "DriverId", "Version", "SupplierId", "WarehouseId", "ManifestId", "Currency"})
		if err != nil {
			return fmt.Errorf("order %s not found: %w", req.OrderID, err)
		}

		var state string
		var shopLoc spanner.NullString
		var driverId, sid, wid, manifestIDNull, orderCurrency spanner.NullString
		var version int64
		if err := row.Columns(&state, &retailerId, &resp.Amount, &shopLoc, &driverId, &version, &sid, &wid, &manifestIDNull, &orderCurrency); err != nil {
			return err
		}
		if sid.Valid {
			supplierID = sid.StringVal
		}
		if wid.Valid {
			warehouseId = wid.StringVal
		}

		if state == "COMPLETED" {
			resp.OrderID = req.OrderID
			resp.State = "COMPLETED"
			resp.Message = "Order already completed"
			return nil // Idempotent
		}
		if state != "PENDING_CASH_COLLECTION" {
			return fmt.Errorf("order %s must be PENDING_CASH_COLLECTION to collect cash (current: %s)", req.OrderID, state)
		}

		// Verify driver identity matches assigned driver
		if driverId.Valid && driverId.StringVal != "" && driverId.StringVal != req.DriverID {
			return fmt.Errorf("driver %s is not assigned to order %s", req.DriverID, req.OrderID)
		}

		// Geofence validation: driver must be within 500m of the retailer
		if shopLoc.Valid && shopLoc.StringVal != "" {
			retailerLoc, parseErr := parseWKTPoint(shopLoc.StringVal)
			if parseErr == nil {
				distance := getDistance(req.Latitude, req.Longitude, retailerLoc.Latitude, retailerLoc.Longitude)
				resp.DistanceM = distance
				if distance > 500 {
					return fmt.Errorf("driver is %.0fm from retailer (max 500m) — move closer to collect cash", distance)
				}
			}
		}

		// Transition to COMPLETED
		newVersion := version + 1
		rowCount, err := txn.Update(ctx, spanner.Statement{
			SQL: `UPDATE Orders SET State = 'COMPLETED', PaymentStatus = 'PAID', Version = @newVersion, LockedUntil = NULL
			      WHERE OrderId = @id AND Version = @version`,
			Params: map[string]interface{}{
				"id":         req.OrderID,
				"newVersion": newVersion,
				"version":    version,
			},
		})
		if err != nil {
			return fmt.Errorf("cash collection completion failed: %w", err)
		}
		if rowCount == 0 {
			return &ErrVersionConflict{OrderID: req.OrderID, ExpectedVersion: version, ActualVersion: -1}
		}

		// Update the cash-custody MasterInvoice with collection details
		invStmt := spanner.Statement{
			SQL:    `SELECT InvoiceId FROM MasterInvoices WHERE OrderId = @oid AND PaymentMode = 'CASH' LIMIT 1`,
			Params: map[string]interface{}{"oid": req.OrderID},
		}
		invIter := txn.Query(ctx, invStmt)
		invRow, invErr := invIter.Next()
		invIter.Stop()
		if invErr == nil {
			var invoiceId string
			if colErr := invRow.Columns(&invoiceId); colErr == nil {
				now := time.Now().UTC()
				if err := txn.BufferWrite([]*spanner.Mutation{
					spanner.Update("MasterInvoices",
						[]string{"InvoiceId", "State", "CollectorDriverId", "CollectedAt", "CollectionLat", "CollectionLng", "GeofenceDistanceM", "CustodyStatus"},
						[]interface{}{invoiceId, "SETTLED", req.DriverID, now, req.Latitude, req.Longitude, resp.DistanceM, "HELD_BY_DRIVER"},
					),
				}); err != nil {
					return fmt.Errorf("update cash custody invoice: %w", err)
				}
			}
		}

		resp.OrderID = req.OrderID
		resp.RetailerID = retailerId
		resp.State = "COMPLETED"

		// LEO Phase V — manifest-completion rollup atomic with cash collection.
		cashCur := "UZS"
		if orderCurrency.Valid && orderCurrency.StringVal != "" {
			cashCur = orderCurrency.StringVal
		}
		if err := outbox.EmitJSON(txn, "Order", req.OrderID, kafkaEvents.EventOrderCompleted, kafkaEvents.TopicMain, kafkaEvents.OrderCompletedEvent{
			OrderID:     req.OrderID,
			RetailerID:  retailerId,
			SupplierId:  supplierID,
			WarehouseId: warehouseId,
			Amount:      resp.Amount,
			Currency:    cashCur,
			Timestamp:   time.Now().UTC(),
		}, telemetry.TraceIDFromContext(ctx)); err != nil {
			return fmt.Errorf("outbox emit ORDER_COMPLETED: %w", err)
		}

		if err := outbox.EmitJSON(txn, "Order", req.OrderID, kafkaEvents.EventOrderStatusChanged, kafkaEvents.TopicMain, kafkaEvents.OrderStatusChangedEvent{
			OrderID:    req.OrderID,
			RetailerID: retailerId,
			SupplierID: supplierID,
			OldState:   "PENDING_CASH_COLLECTION",
			NewState:   "COMPLETED",
			Timestamp:  time.Now().UTC(),
		}, telemetry.TraceIDFromContext(ctx)); err != nil {
			return fmt.Errorf("outbox emit ORDER_STATUS_CHANGED (pending_cash_collection->completed): %w", err)
		}

		if manifestIDNull.Valid {
			if err := rollupManifestIfComplete(ctx, txn, manifestIDNull.StringVal, time.Now().UTC()); err != nil {
				return fmt.Errorf("manifest rollup failed for order %s: %w", req.OrderID, err)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	resp.Message = fmt.Sprintf("Cash of %d collected. Order complete.", resp.Amount)

	return &resp, nil
}

// ─── Reassignment Engine ─────────────────────────────────────────────────────

// ReassignRoute moves orders from their current truck to a different truck.
// Guards: freeze lock, state (only PENDING/LOADED with existing route allowed),
// sealed orders (DeliveryToken != NULL) are blocked, target truck capacity validated.
func (s *OrderService) ReassignRoute(ctx context.Context, orderIds []string, newRouteId string) ([]ReassignConflict, error) {
	var conflicts []ReassignConflict
	var oldRouteID string

	_, err := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// 1. Read all orders' current state
		stmt := spanner.Statement{
			SQL:    `SELECT OrderId, State, RouteId, Version, LockedUntil, DeliveryToken FROM Orders WHERE OrderId IN UNNEST(@ids)`,
			Params: map[string]interface{}{"ids": orderIds},
		}
		iter := txn.Query(ctx, stmt)
		defer iter.Stop()

		type snap struct {
			state   string
			route   spanner.NullString
			version int64
			locked  spanner.NullTime
			sealed  spanner.NullString
		}
		found := map[string]snap{}
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return fmt.Errorf("read orders for reassign: %w", err)
			}
			var id, state string
			var route, token spanner.NullString
			var version int64
			var locked spanner.NullTime
			if err := row.Columns(&id, &state, &route, &version, &locked, &token); err != nil {
				return fmt.Errorf("parse order: %w", err)
			}
			found[id] = snap{state: state, route: route, version: version, locked: locked, sealed: token}
		}

		// 2. Validate each order
		var validIds []string
		for _, id := range orderIds {
			sn, ok := found[id]
			if !ok {
				conflicts = append(conflicts, ReassignConflict{OrderID: id, Reason: "ORDER_NOT_FOUND"})
				continue
			}
			if sn.locked.Valid && time.Now().Before(sn.locked.Time) {
				conflicts = append(conflicts, ReassignConflict{OrderID: id, Reason: fmt.Sprintf("FREEZE_LOCKED_UNTIL_%s", sn.locked.Time.Format(time.RFC3339))})
				continue
			}
			if sn.sealed.Valid && sn.sealed.StringVal != "" {
				conflicts = append(conflicts, ReassignConflict{OrderID: id, Reason: "ALREADY_SEALED"})
				continue
			}
			switch sn.state {
			case "CANCELLED", "COMPLETED", "IN_TRANSIT", "ARRIVED", "AWAITING_PAYMENT":
				conflicts = append(conflicts, ReassignConflict{OrderID: id, Reason: fmt.Sprintf("STATE_%s_NOT_REASSIGNABLE", sn.state)})
				continue
			}
			if !sn.route.Valid || sn.route.StringVal == "" {
				conflicts = append(conflicts, ReassignConflict{OrderID: id, Reason: "NO_CURRENT_ROUTE"})
				continue
			}
			if sn.route.StringVal == newRouteId {
				conflicts = append(conflicts, ReassignConflict{OrderID: id, Reason: "ALREADY_ON_TARGET_TRUCK"})
				continue
			}
			if oldRouteID == "" {
				oldRouteID = sn.route.StringVal
			}
			validIds = append(validIds, id)
		}

		if len(validIds) == 0 {
			return nil // all blocked — conflicts populated
		}

		// 3. Capacity check on target truck
		capErr := s.validateTruckCapacity(ctx, txn, newRouteId, validIds)
		if capErr != nil {
			for _, id := range validIds {
				conflicts = append(conflicts, ReassignConflict{OrderID: id, Reason: capErr.Error()})
			}
			return nil
		}

		// 4. Reassign valid orders. Track only rows actually written so the
		//    outbox emit below covers exactly the committed mutations.
		freezeUntil := time.Now().Add(30 * time.Minute)
		var publishedIds []string
		for _, id := range validIds {
			sn := found[id]
			newVersion := sn.version + 1
			rowCount, err := txn.Update(ctx, spanner.Statement{
				SQL: `UPDATE Orders
				      SET RouteId = @newRouteId, Version = @newVersion, LockedUntil = @freezeUntil
				      WHERE OrderId = @orderId AND Version = @version`,
				Params: map[string]interface{}{
					"newRouteId":  spanner.NullString{StringVal: newRouteId, Valid: true},
					"newVersion":  newVersion,
					"freezeUntil": freezeUntil,
					"orderId":     id,
					"version":     sn.version,
				},
			})
			if err != nil {
				return fmt.Errorf("reassign order %s: %w", id, err)
			}
			if rowCount == 0 {
				conflicts = append(conflicts, ReassignConflict{OrderID: id, Reason: "VERSION_CONFLICT"})
				continue
			}
			publishedIds = append(publishedIds, id)
		}

		// 5. Outbox emit — ORDER_REASSIGNED. Atomic with the Orders UPDATEs:
		//    if the commit aborts the event disappears with the mutations.
		//    The Relay publishes asynchronously onto lab-logistics-events.
		//    RouteId == DriverId in this codebase (see validateTruckCapacity).
		if len(publishedIds) > 0 {
			if err := outbox.EmitJSON(txn, "Route", newRouteId, kafkaEvents.EventOrderReassigned, topicLogisticsEvents, map[string]interface{}{
				"type":          kafkaEvents.EventOrderReassigned,
				"order_ids":     publishedIds,
				"old_route_id":  oldRouteID,
				"new_route_id":  newRouteId,
				"old_driver_id": oldRouteID,
				"new_driver_id": newRouteId,
				"timestamp":     time.Now().UTC().Format(time.RFC3339),
			}, telemetry.TraceIDFromContext(ctx)); err != nil {
				return fmt.Errorf("emit ORDER_REASSIGNED outbox: %w", err)
			}
		}
		return nil
	})

	if err != nil {
		return conflicts, err
	}

	// Cache fan-out so optimistic Fleet-Hub / Supplier-Portal views drop
	// stale route membership on the next read. Keys are advisory — HTTP
	// caches adopting route:<id> conventions will refresh transparently.
	if oldRouteID != "" || newRouteId != "" {
		keys := make([]string, 0, 2)
		if oldRouteID != "" {
			keys = append(keys, "route:"+oldRouteID)
		}
		if newRouteId != "" {
			keys = append(keys, "route:"+newRouteId)
		}
		cache.Invalidate(ctx, keys...)
	}

	return conflicts, nil
}

// validateTruckCapacity checks if the target truck has enough volume for the given orders.
func (s *OrderService) validateTruckCapacity(ctx context.Context, txn *spanner.ReadWriteTransaction, routeId string, orderIds []string) error {
	// Get truck max capacity
	capStmt := spanner.Statement{
		SQL: `SELECT v.MaxVolumeVU
		      FROM Drivers d JOIN Vehicles v ON d.VehicleId = v.VehicleId
		      WHERE d.DriverId = @driverId AND d.IsActive = true AND v.IsActive = true`,
		Params: map[string]interface{}{"driverId": routeId},
	}
	capIter := txn.Query(ctx, capStmt)
	capRow, err := capIter.Next()
	capIter.Stop()
	if err != nil {
		return fmt.Errorf("TRUCK_NOT_FOUND")
	}
	var maxVU float64
	if err := capRow.Columns(&maxVU); err != nil {
		return fmt.Errorf("TRUCK_CAPACITY_READ_ERROR")
	}

	// Get current load on target truck (excluding the orders being moved)
	loadStmt := spanner.Statement{
		SQL: `SELECT COALESCE(SUM(li.Quantity * COALESCE(sp.VolumetricUnit, sp.PalletFootprint, 1.0)), 0)
		      FROM Orders o
		      JOIN OrderLineItems li ON o.OrderId = li.OrderId
		      LEFT JOIN SupplierProducts sp ON li.SkuId = sp.SkuId
		      WHERE o.RouteId = @routeId AND o.State IN ('PENDING', 'LOADED', 'SCHEDULED')
		        AND o.OrderId NOT IN UNNEST(@excludeIds)`,
		Params: map[string]interface{}{"routeId": routeId, "excludeIds": orderIds},
	}
	loadIter := txn.Query(ctx, loadStmt)
	loadRow, err := loadIter.Next()
	loadIter.Stop()
	var currentLoad float64
	if err == nil {
		loadRow.Columns(&currentLoad)
	}

	// Get volume of orders being moved
	moveStmt := spanner.Statement{
		SQL: `SELECT COALESCE(SUM(li.Quantity * COALESCE(sp.VolumetricUnit, sp.PalletFootprint, 1.0)), 0)
		      FROM OrderLineItems li
		      LEFT JOIN SupplierProducts sp ON li.SkuId = sp.SkuId
		      WHERE li.OrderId IN UNNEST(@ids)`,
		Params: map[string]interface{}{"ids": orderIds},
	}
	moveIter := txn.Query(ctx, moveStmt)
	moveRow, err := moveIter.Next()
	moveIter.Stop()
	var moveVolume float64
	if err == nil {
		moveRow.Columns(&moveVolume)
	}

	if currentLoad+moveVolume > maxVU {
		return fmt.Errorf("INSUFFICIENT_CAPACITY: need %.1f VU, truck has %.1f free of %.1f max",
			moveVolume, maxVU-currentLoad, maxVU)
	}
	return nil
}

// GetTruckCapacity returns capacity info for a truck.
func (s *OrderService) GetTruckCapacity(ctx context.Context, routeId string) (*CapacityInfo, error) {
	capStmt := spanner.Statement{
		SQL: `SELECT v.MaxVolumeVU
		      FROM Drivers d JOIN Vehicles v ON d.VehicleId = v.VehicleId
		      WHERE d.DriverId = @driverId AND d.IsActive = true AND v.IsActive = true`,
		Params: map[string]interface{}{"driverId": routeId},
	}
	capIter := s.Client.Single().Query(ctx, capStmt)
	capRow, err := capIter.Next()
	capIter.Stop()
	if err != nil {
		return nil, fmt.Errorf("truck %s not found", routeId)
	}
	var maxVU float64
	if err := capRow.Columns(&maxVU); err != nil {
		return nil, err
	}

	loadStmt := spanner.Statement{
		SQL: `SELECT COALESCE(SUM(li.Quantity * COALESCE(sp.VolumetricUnit, sp.PalletFootprint, 1.0)), 0)
		      FROM Orders o
		      JOIN OrderLineItems li ON o.OrderId = li.OrderId
		      LEFT JOIN SupplierProducts sp ON li.SkuId = sp.SkuId
		      WHERE o.RouteId = @routeId AND o.State IN ('PENDING', 'LOADED', 'SCHEDULED')`,
		Params: map[string]interface{}{"routeId": routeId},
	}
	loadIter := s.Client.Single().Query(ctx, loadStmt)
	loadRow, err := loadIter.Next()
	loadIter.Stop()
	var used float64
	if err == nil {
		loadRow.Columns(&used)
	}

	// Pending returns: rejected items physically still on truck (ReturnClearedAt IS NULL)
	returnsStmt := spanner.Statement{
		SQL: `SELECT COALESCE(SUM(li.RejectedQty * COALESCE(sp.VolumetricUnit, sp.PalletFootprint, 1.0)), 0)
		      FROM Orders o
		      JOIN OrderLineItems li ON o.OrderId = li.OrderId
		      LEFT JOIN SupplierProducts sp ON li.SkuId = sp.SkuId
		      WHERE o.RouteId = @routeId
		        AND li.RejectedQty > 0
		        AND li.ReturnClearedAt IS NULL`,
		Params: map[string]interface{}{"routeId": routeId},
	}
	returnsIter := s.Client.Single().Query(ctx, returnsStmt)
	returnsRow, err := returnsIter.Next()
	returnsIter.Stop()
	var pendingReturns float64
	if err == nil {
		returnsRow.Columns(&pendingReturns)
	}

	return &CapacityInfo{
		RouteID:          routeId,
		MaxVolumeVU:      maxVU,
		UsedVolumeVU:     used,
		FreeVolumeVU:     maxVU - used,
		PendingReturnsVU: pendingReturns,
	}, nil
}

// AssignRouteWithCapacity wraps AssignRoute with a pre-flight capacity check.
func (s *OrderService) AssignRouteWithCapacity(ctx context.Context, orderIds []string, routeId string) error {
	_, err := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := s.validateTruckCapacity(ctx, txn, routeId, orderIds); err != nil {
			return fmt.Errorf("capacity validation failed: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return s.AssignRoute(ctx, orderIds, routeId)
}

// ─── Active Fulfillment (Retailer View) ────────────────────────────────────────

// ActiveFulfillmentItem represents a single incoming delivery visible to the retailer.
type ActiveFulfillmentItem struct {
	OrderID        string `json:"order_id"`
	SupplierID     string `json:"supplier_id"`
	SupplierName   string `json:"supplier_name"`
	State          string `json:"state"`
	AdjustedAmount int64  `json:"adjusted_amount"` // SUM(UnitPrice * Quantity) in tiyins
	ItemCount      int    `json:"item_count"`
}

// ActiveFulfillments returns all orders approaching or awaiting payment for a given retailer.
// States: IN_TRANSIT, ARRIVED, AWAITING_PAYMENT — the "incoming deliveries" window.
func (s *OrderService) ActiveFulfillments(ctx context.Context, retailerID string) ([]ActiveFulfillmentItem, error) {
	stmt := spanner.Statement{
		SQL: `SELECT o.OrderId, o.SupplierId, COALESCE(s.Name, '') AS SupplierName, o.State,
		             COALESCE((SELECT SUM(li.UnitPrice * li.Quantity)
		                       FROM OrderLineItems li
		                       WHERE li.OrderId = o.OrderId), 0) AS AdjustedAmount,
		             COALESCE((SELECT COUNT(*)
		                       FROM OrderLineItems li2
		                       WHERE li2.OrderId = o.OrderId), 0) AS ItemCount
		      FROM Orders o
		      LEFT JOIN Suppliers s ON o.SupplierId = s.SupplierId
		      WHERE o.RetailerId = @retailerId
		        AND o.State IN ('IN_TRANSIT', 'ARRIVED', 'AWAITING_PAYMENT')
		      ORDER BY o.UpdatedAt DESC`,
		Params: map[string]interface{}{
			"retailerId": retailerID,
		},
	}

	iter := s.Client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var results []ActiveFulfillmentItem
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("active fulfillments query failed: %w", err)
		}

		var item ActiveFulfillmentItem
		var supplierID, supplierName spanner.NullString
		var adjustedAmount spanner.NullInt64
		var itemCount int64

		if err := row.Columns(&item.OrderID, &supplierID, &supplierName, &item.State, &adjustedAmount, &itemCount); err != nil {
			return nil, fmt.Errorf("column parse failed: %w", err)
		}

		item.SupplierID = supplierID.StringVal
		item.SupplierName = supplierName.StringVal
		item.AdjustedAmount = adjustedAmount.Int64
		item.ItemCount = int(itemCount)
		results = append(results, item)
	}

	return results, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Cash Shadow Recovery
// ═══════════════════════════════════════════════════════════════════════════════

// PendingCollection represents an order awaiting cash collection from the driver.
type PendingCollection struct {
	OrderID    string `json:"order_id"`
	RetailerID string `json:"retailer_id"`
	Amount     int64  `json:"amount"`
	State      string `json:"state"`
	UpdatedAt  string `json:"updated_at"`
}

// GetPendingCollections returns all PENDING_CASH_COLLECTION orders assigned to the
// given driver. Used for driver app reconnect / cash shadow recovery (F-2).
func (s *OrderService) GetPendingCollections(ctx context.Context, driverID string) ([]PendingCollection, error) {
	stmt := spanner.Statement{
		SQL: `SELECT OrderId, RetailerId, Amount, State, UpdatedAt
		       FROM Orders
		       WHERE DriverId = @driverID
		         AND State = 'PENDING_CASH_COLLECTION'
		       ORDER BY UpdatedAt DESC`,
		Params: map[string]interface{}{"driverID": driverID},
	}

	iter := s.Client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var results []PendingCollection
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("pending collections query failed: %w", err)
		}

		var item PendingCollection
		var retailerID spanner.NullString
		var updatedAt spanner.NullTime

		if err := row.Columns(&item.OrderID, &retailerID, &item.Amount, &item.State, &updatedAt); err != nil {
			return nil, fmt.Errorf("pending collection column parse: %w", err)
		}

		item.RetailerID = retailerID.StringVal
		if updatedAt.Valid {
			item.UpdatedAt = updatedAt.Time.Format("2006-01-02T15:04:05Z")
		}
		results = append(results, item)
	}

	return results, nil
}

// HandlePendingCollections returns an HTTP handler for GET /v1/driver/pending-collections.
// Returns all PENDING_CASH_COLLECTION orders assigned to the requesting driver.
func HandlePendingCollections(svc *OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if !ok || claims == nil || claims.UserID == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		results, err := svc.GetPendingCollections(ctx, claims.UserID)
		if err != nil {
			slog.Error("order.pending_collections_query_failed", "driver_id", claims.UserID, "err", err)
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"pending_collections": results,
			"count":               len(results),
		})
	}
}

// rollupManifestIfComplete is the LEO Phase V terminal-rollup hook. It is
// called from inside any RWTxn that just transitioned an Order to COMPLETED.
// If the order was the last non-COMPLETED stop on its manifest, the parent
// SupplierTruckManifests row is advanced DISPATCHED → COMPLETED and a
// MANIFEST_COMPLETED ManifestLifecycleEvent is emitted via the outbox in the
// same transaction. No-op when manifestID is empty (legacy non-LEO orders) or
// when remaining stops still have non-COMPLETED state. Idempotent: a manifest
// already in COMPLETED or CANCELLED is left alone.
func rollupManifestIfComplete(ctx context.Context, txn *spanner.ReadWriteTransaction, manifestID string, ts time.Time) error {
	if manifestID == "" {
		return nil
	}
	cntStmt := spanner.Statement{
		SQL:    `SELECT COUNT(1) FROM Orders WHERE ManifestId = @mid AND State NOT IN ('COMPLETED', 'CANCELLED', 'RETURNED', 'FAILED', 'REJECTED')`,
		Params: map[string]interface{}{"mid": manifestID},
	}
	cntIter := txn.Query(ctx, cntStmt)
	defer cntIter.Stop()
	cntRow, cntErr := cntIter.Next()
	if cntErr != nil {
		return cntErr
	}
	var remaining int64
	if err := cntRow.Columns(&remaining); err != nil {
		return err
	}
	if remaining > 0 {
		return nil
	}
	mRow, err := txn.ReadRow(ctx, "SupplierTruckManifests", spanner.Key{manifestID},
		[]string{"State", "SupplierId", "DriverId", "TruckId", "StopCount", "TotalVolumeVU", "MaxVolumeVU"})
	if err != nil {
		// Manifest not found — order is legacy non-LEO; non-fatal.
		return nil
	}
	var state, supplierID, driverID, truckID string
	var stopCount int64
	var totalVU, maxVU float64
	if err := mRow.Columns(&state, &supplierID, &driverID, &truckID, &stopCount, &totalVU, &maxVU); err != nil {
		return err
	}
	if state == "COMPLETED" || state == "CANCELLED" {
		return nil
	}
	if err := txn.BufferWrite([]*spanner.Mutation{
		spanner.Update("SupplierTruckManifests",
			[]string{"ManifestId", "State", "CompletedAt", "UpdatedAt"},
			[]interface{}{manifestID, "COMPLETED", spanner.CommitTimestamp, spanner.CommitTimestamp}),
	}); err != nil {
		return err
	}
	return outbox.EmitJSON(txn, "Manifest", manifestID,
		kafkaEvents.EventManifestCompleted, kafkaEvents.TopicMain,
		kafkaEvents.ManifestLifecycleEvent{
			ManifestID:  manifestID,
			SupplierId:  supplierID,
			DriverID:    driverID,
			TruckID:     truckID,
			State:       "COMPLETED",
			StopCount:   int(stopCount),
			VolumeVU:    totalVU,
			MaxVolumeVU: maxVU,
			Timestamp:   ts,
		}, telemetry.TraceIDFromContext(ctx))
}
