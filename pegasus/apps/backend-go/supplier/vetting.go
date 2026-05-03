package supplier

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"backend-go/auth"
	internalKafka "backend-go/kafka"
	"backend-go/outbox"
	"backend-go/telemetry"
	"backend-go/workers"
	"backend-go/ws"

	"cloud.google.com/go/spanner"
	kafkago "github.com/segmentio/kafka-go"
	"google.golang.org/api/iterator"
)

// ── ORDER VETTING ENGINE ────────────────────────────────────────────────────
// Suppliers review incoming orders, approve or reject them.
// Rejection triggers Kafka event for payment refund + inventory lock release.

type PendingOrder struct {
	OrderID      string `json:"order_id"`
	RetailerID   string `json:"retailer_id"`
	RetailerName string `json:"retailer_name"`
	SupplierId   string `json:"supplier_id"`
	Amount       int64  `json:"amount"`
	ItemCount    int64  `json:"item_count"`
	State        string `json:"state"`
	OrderSource  string `json:"order_source"`
	CreatedAt    string `json:"created_at"`
}

type VetOrderRequest struct {
	OrderID  string `json:"order_id"`
	Decision string `json:"decision"` // APPROVED | REJECTED
	Reason   string `json:"reason"`   // required for REJECTED
}

// RetailerPusher is a minimal interface for pushing real-time events to retailer devices.
// Implemented by ws.RetailerHub — defined here to avoid circular imports.
type RetailerPusher interface {
	PushToRetailer(retailerID string, payload interface{}) bool
}

type OrderVettingService struct {
	Client      *spanner.Client
	Producer    *kafkago.Writer
	RetailerHub RetailerPusher
}

func NewOrderVettingService(client *spanner.Client, producer *kafkago.Writer, rh RetailerPusher) *OrderVettingService {
	return &OrderVettingService{Client: client, Producer: producer, RetailerHub: rh}
}

// HandleSupplierOrders — GET: list orders for this supplier with bucket-aware filtering
// Query params:
//
//	state=PENDING (single state, legacy compat)
//	states=PENDING,LOADED (comma-separated set)
//	bucket=today|scheduled|dispatched|history (shortcut)
//	route_id=<driverId> (filter by assigned truck)
//	date_from=2024-01-01 / date_to=2024-01-31 (delivery date range)
//	limit=50 / offset=0 (pagination)
func (s *OrderVettingService) HandleSupplierOrders(w http.ResponseWriter, r *http.Request) {
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

	q := r.URL.Query()

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	sql := `SELECT o.OrderId, o.RetailerId, COALESCE(ret.Name, 'Unknown'), o.SupplierId,
	               COALESCE(o.Amount, 0), o.State, COALESCE(o.OrderSource, 'MANUAL'), o.CreatedAt,
	               (SELECT COUNT(*) FROM OrderLineItems li WHERE li.OrderId = o.OrderId),
	               o.RouteId, o.RequestedDeliveryDate, o.PaymentGateway, o.PaymentStatus
	        FROM Orders o
	        LEFT JOIN Retailers ret ON o.RetailerId = ret.RetailerId
	        WHERE o.SupplierId = @sid`

	params := map[string]interface{}{"sid": supplierId}

	// Apply warehouse scope if present
	if whID := auth.EffectiveWarehouseID(r.Context()); whID != "" {
		sql += " AND o.WarehouseId = @warehouseId"
		params["warehouseId"] = whID
	}

	// Bucket shortcut expands into state + date filters
	bucket := q.Get("bucket")
	now := time.Now().UTC()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	todayEnd := todayStart.Add(24 * time.Hour)

	switch bucket {
	case "today":
		sql += ` AND o.State IN ('PENDING', 'LOADED') AND (o.RequestedDeliveryDate IS NULL OR o.RequestedDeliveryDate < @todayEnd)`
		params["todayEnd"] = todayEnd
	case "scheduled":
		sql += ` AND o.State = 'SCHEDULED'`
	case "dispatched":
		sql += ` AND o.State IN ('LOADED', 'IN_TRANSIT', 'ARRIVED', 'AWAITING_PAYMENT') AND o.RouteId IS NOT NULL`
	case "history":
		sql += ` AND o.State IN ('COMPLETED', 'CANCELLED')`
	default:
		// Fine-grained filters
		if states := q.Get("states"); states != "" {
			stateList := strings.Split(states, ",")
			sql += ` AND o.State IN UNNEST(@states)`
			params["states"] = stateList
		} else if state := q.Get("state"); state != "" && state != "ALL" {
			sql += ` AND o.State = @state`
			params["state"] = state
		}
	}

	// Route filter
	if routeId := q.Get("route_id"); routeId != "" {
		sql += ` AND o.RouteId = @routeId`
		params["routeId"] = routeId
	}

	// Date range on RequestedDeliveryDate
	if dateFrom := q.Get("date_from"); dateFrom != "" {
		if t, err := time.Parse("2006-01-02", dateFrom); err == nil {
			sql += ` AND o.RequestedDeliveryDate >= @dateFrom`
			params["dateFrom"] = t
		}
	}
	if dateTo := q.Get("date_to"); dateTo != "" {
		if t, err := time.Parse("2006-01-02", dateTo); err == nil {
			sql += ` AND o.RequestedDeliveryDate < @dateTo`
			params["dateTo"] = t.Add(24 * time.Hour)
		}
	}

	sql += ` ORDER BY o.CreatedAt DESC`

	// Pagination
	limit := 50
	offset := 0
	if l := q.Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 500 {
			limit = v
		}
	}
	if o := q.Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}
	sql += fmt.Sprintf(` LIMIT %d OFFSET %d`, limit+1, offset)

	stmt := spanner.Statement{SQL: sql, Params: params}
	iter := s.Client.Single().Query(ctx, stmt)
	defer iter.Stop()

	type EnrichedOrder struct {
		PendingOrder
		RouteId               string `json:"route_id,omitempty"`
		RequestedDeliveryDate string `json:"requested_delivery_date,omitempty"`
		PaymentGateway        string `json:"payment_gateway,omitempty"`
		PaymentStatus         string `json:"payment_status,omitempty"`
	}

	var orders []EnrichedOrder
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("[VETTING] Query error: %v", err)
			break
		}
		var o EnrichedOrder
		var createdAt spanner.NullTime
		var routeId spanner.NullString
		var deliveryDate spanner.NullDate
		var paymentGateway spanner.NullString
		var paymentStatus spanner.NullString
		if err := row.Columns(&o.OrderID, &o.RetailerID, &o.RetailerName, &o.SupplierId, &o.Amount, &o.State, &o.OrderSource, &createdAt, &o.ItemCount, &routeId, &deliveryDate, &paymentGateway, &paymentStatus); err != nil {
			log.Printf("[VETTING] Row parse error: %v", err)
			continue
		}
		if createdAt.Valid {
			o.CreatedAt = createdAt.Time.Format(time.RFC3339)
		}
		if routeId.Valid {
			o.RouteId = routeId.StringVal
		}
		if deliveryDate.Valid {
			o.RequestedDeliveryDate = deliveryDate.Date.String()
		}
		if paymentGateway.Valid {
			o.PaymentGateway = paymentGateway.StringVal
		}
		if paymentStatus.Valid {
			o.PaymentStatus = paymentStatus.StringVal
		}
		orders = append(orders, o)
	}
	if orders == nil {
		orders = []EnrichedOrder{}
	}
	hasMore := len(orders) > limit
	if hasMore {
		orders = orders[:limit]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":        orders,
		"limit":       limit,
		"offset":      offset,
		"has_more":    hasMore,
		"next_offset": offset + len(orders),
	})
}

// HandleVetOrder — POST: supplier approves or rejects an order
func (s *OrderVettingService) HandleVetOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	supplierId := claims.ResolveSupplierID()
	adjustedBy := claims.UserID
	if adjustedBy == "" {
		adjustedBy = supplierId
	}

	var req VetOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}
	if req.OrderID == "" {
		http.Error(w, `{"error":"order_id required"}`, http.StatusBadRequest)
		return
	}
	if req.Decision != "APPROVED" && req.Decision != "REJECTED" {
		http.Error(w, `{"error":"decision must be APPROVED or REJECTED"}`, http.StatusBadRequest)
		return
	}
	if req.Decision == "REJECTED" && req.Reason == "" {
		http.Error(w, `{"error":"reason required for rejection"}`, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var newState string
	var retailerId string
	var amount int64
	var gateway string

	_, err := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Read current order state + verify supplier ownership
		row, err := txn.ReadRow(ctx, "Orders", spanner.Key{req.OrderID},
			[]string{"State", "SupplierId", "RetailerId", "Amount", "PaymentGateway"})
		if err != nil {
			return fmt.Errorf("order not found: %s", req.OrderID)
		}
		var currentState, orderSupplier string
		var nullAmount spanner.NullInt64
		var nullGateway spanner.NullString
		if err := row.Columns(&currentState, &orderSupplier, &retailerId, &nullAmount, &nullGateway); err != nil {
			return err
		}
		if orderSupplier != supplierId {
			return fmt.Errorf("access denied: order belongs to another supplier")
		}
		if currentState != "PENDING" {
			return fmt.Errorf("order %s is %s, not PENDING — cannot vet", req.OrderID, currentState)
		}

		if nullAmount.Valid {
			amount = nullAmount.Int64
		}
		if nullGateway.Valid {
			gateway = nullGateway.StringVal
		}

		if req.Decision == "APPROVED" {
			newState = "LOADED"
		} else {
			newState = "CANCELLED"
		}
		transitionAt := time.Now().UTC()

		// Update state
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("Orders",
				[]string{"OrderId", "State"},
				[]interface{}{req.OrderID, newState},
			),
		}); err != nil {
			return err
		}
		if err := outbox.EmitJSON(txn, "Order", req.OrderID, internalKafka.EventOrderStatusChanged, internalKafka.TopicMain, internalKafka.OrderStatusChangedEvent{
			OrderID:    req.OrderID,
			RetailerID: retailerId,
			SupplierID: supplierId,
			OldState:   currentState,
			NewState:   newState,
			Timestamp:  transitionAt,
		}, telemetry.TraceIDFromContext(ctx)); err != nil {
			return fmt.Errorf("emit order status changed: %w", err)
		}

		// If rejected, release inventory locks
		if req.Decision == "REJECTED" {
			lineStmt := spanner.Statement{
				SQL:    `SELECT SkuId, Quantity FROM OrderLineItems WHERE OrderId = @oid`,
				Params: map[string]interface{}{"oid": req.OrderID},
			}
			lineIter := txn.Query(ctx, lineStmt)
			defer lineIter.Stop()

			skuQtyMap := make(map[string]int64)
			for {
				lineRow, err := lineIter.Next()
				if err != nil {
					break
				}
				var skuId string
				var qty int64
				if err := lineRow.Columns(&skuId, &qty); err != nil {
					continue
				}
				skuQtyMap[skuId] += qty
			}

			for skuId, qty := range skuQtyMap {
				invRow, err := txn.ReadRow(ctx, "SupplierInventory", spanner.Key{skuId}, []string{"QuantityAvailable"})
				if err != nil {
					continue
				}
				var currentQty int64
				if err := invRow.Columns(&currentQty); err != nil {
					return err
				}
				auditEntry := newReturnRestockAuditEntry(skuId, supplierId, adjustedBy, currentQty, qty)
				if err := txn.BufferWrite([]*spanner.Mutation{
					spanner.Update("SupplierInventory",
						[]string{"ProductId", "QuantityAvailable", "UpdatedAt"},
						[]interface{}{skuId, currentQty + qty, spanner.CommitTimestamp},
					),
					auditEntry.Mutation(),
				}); err != nil {
					return err
				}
			}

			event := map[string]interface{}{
				"type":        ws.EventOrderRejectedBySupplier,
				"order_id":    req.OrderID,
				"retailer_id": retailerId,
				"supplier_id": supplierId,
				"amount":      amount,
				"gateway":     gateway,
				"reason":      req.Reason,
				"timestamp":   transitionAt.UnixMilli(),
			}
			return outbox.EmitJSON(txn, "Order", req.OrderID, ws.EventOrderRejectedBySupplier, internalKafka.TopicMain, event, telemetry.TraceIDFromContext(ctx))
		}

		return nil
	})

	if err != nil {
		log.Printf("[VETTING] VetOrder failed for %s: %v", req.OrderID, err)
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	// Push ORDER_STATE_CHANGED to supplier admin portal via WebSocket
	if newState == "LOADED" {
		workers.EventPool.Submit(func() { telemetry.FleetHub.BroadcastOrderStateChange(supplierId, req.OrderID, "LOADED", "") })
		// Push ORDER_STATUS_CHANGED (LOADED / "Approved") to retailer via WebSocket
		if s.RetailerHub != nil && retailerId != "" {
			workers.EventPool.Submit(func() {
				s.RetailerHub.PushToRetailer(retailerId, map[string]interface{}{
					"type":      ws.EventOrderStatusChanged,
					"order_id":  req.OrderID,
					"state":     "LOADED",
					"timestamp": time.Now().UTC().Format(time.RFC3339),
				})
			})
		}
	} else if newState == "CANCELLED" {
		workers.EventPool.Submit(func() { telemetry.FleetHub.BroadcastOrderStateChange(supplierId, req.OrderID, "CANCELLED", "") })
		// Push ORDER_STATUS_CHANGED (CANCELLED) to retailer via WebSocket
		if s.RetailerHub != nil && retailerId != "" {
			workers.EventPool.Submit(func() {
				s.RetailerHub.PushToRetailer(retailerId, map[string]interface{}{
					"type":      ws.EventOrderStatusChanged,
					"order_id":  req.OrderID,
					"state":     "CANCELLED",
					"timestamp": time.Now().UTC().Format(time.RFC3339),
				})
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    fmt.Sprintf("ORDER_%s", req.Decision),
		"order_id":  req.OrderID,
		"new_state": newState,
	})
}
