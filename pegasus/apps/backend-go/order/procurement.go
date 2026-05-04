package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"backend-go/auth"
	"backend-go/cache"
	"backend-go/hotspot"
	kafkaEvents "backend-go/kafka"
	"backend-go/outbox"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

var (
	ErrProcurementEmptyItems    = errors.New("procurement items must not be empty")
	ErrProcurementInvalidItem   = errors.New("procurement item is invalid")
	ErrProcurementUnknownSKU    = errors.New("procurement sku is unavailable")
	ErrProcurementMixedSupplier = errors.New("procurement items must belong to one supplier")
	ErrProcurementRetailerGone  = errors.New("procurement retailer profile not found")
)

// ProcurementItem is a legacy retailer procurement line item.
type ProcurementItem struct {
	ProductID string `json:"product_id"`
	Quantity  int64  `json:"quantity"`
}

// ProcurementOrderRequest is the legacy procurement-order request body.
type ProcurementOrderRequest struct {
	RetailerID string            `json:"retailer_id"`
	Items      []ProcurementItem `json:"items"`
}

// ProcurementOrderResponse preserves the legacy response while adding fields
// expected by native clients that decode the result as an order summary.
type ProcurementOrderResponse struct {
	Status      string            `json:"status"`
	OrderID     string            `json:"order_id"`
	RetailerID  string            `json:"retailer_id"`
	SupplierID  string            `json:"supplier_id,omitempty"`
	State       string            `json:"state"`
	Amount      int64             `json:"amount"`
	Total       int64             `json:"total"`
	Currency    string            `json:"currency"`
	OrderSource string            `json:"order_source"`
	CreatedAt   string            `json:"created_at"`
	Items       []ProcurementItem `json:"items"`
}

type procurementProduct struct {
	SKU        string
	SupplierID string
	Price      int64
}

type procurementMutationInput struct {
	OrderID    string
	Request    ProcurementOrderRequest
	Products   map[string]procurementProduct
	SupplierID string
	Latitude   float64
	Longitude  float64
}

// HandleCreateProcurementOrder creates a claim-scoped legacy procurement order.
func HandleCreateProcurementOrder(service *OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil || claims.Role != "RETAILER" || claims.UserID == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req ProcurementOrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"Invalid JSON body"}`, http.StatusBadRequest)
			return
		}
		if req.RetailerID != "" && req.RetailerID != claims.UserID {
			http.Error(w, `{"error":"forbidden: cannot create orders for another retailer"}`, http.StatusForbidden)
			return
		}
		req.RetailerID = claims.UserID

		response, err := service.CreateProcurementOrder(r.Context(), req)
		if err != nil {
			writeProcurementError(w, r, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			slog.ErrorContext(r.Context(), "procurement write response failed", "err", err, "retailer_id", req.RetailerID)
		}
	}
}

// CreateProcurementOrder creates the legacy retailer procurement order atomically.
func (s *OrderService) CreateProcurementOrder(ctx context.Context, req ProcurementOrderRequest) (ProcurementOrderResponse, error) {
	if len(req.Items) == 0 {
		return ProcurementOrderResponse{}, ErrProcurementEmptyItems
	}
	if err := validateProcurementItems(req.Items); err != nil {
		return ProcurementOrderResponse{}, err
	}

	orderID := hotspot.NewOrderID()
	createdAt := time.Now().UTC()
	response := ProcurementOrderResponse{}

	_, err := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		latitude, longitude, err := readProcurementRetailerLocation(ctx, txn, req.RetailerID)
		if err != nil {
			return err
		}
		products, supplierID, err := readProcurementProducts(ctx, txn, req.Items)
		if err != nil {
			return err
		}

		mutations, totalAmount := buildProcurementMutations(procurementMutationInput{
			OrderID:    orderID,
			Request:    req,
			Products:   products,
			SupplierID: supplierID,
			Latitude:   latitude,
			Longitude:  longitude,
		})
		if err := txn.BufferWrite(mutations); err != nil {
			return fmt.Errorf("buffer procurement order writes: %w", err)
		}
		event := kafkaEvents.OrderCreatedEvent{
			OrderID:    orderID,
			SupplierID: supplierID,
			RetailerID: req.RetailerID,
			Total:      totalAmount,
			Currency:   "UZS",
			Timestamp:  createdAt,
		}
		if err := outbox.EmitJSON(txn, "Order", orderID, kafkaEvents.EventOrderCreated, kafkaEvents.TopicMain, event, telemetry.TraceIDFromContext(ctx)); err != nil {
			return fmt.Errorf("emit procurement order event: %w", err)
		}

		response = ProcurementOrderResponse{
			Status:      "PROCUREMENT_AUTHORIZED",
			OrderID:     orderID,
			RetailerID:  req.RetailerID,
			SupplierID:  supplierID,
			State:       "PENDING",
			Amount:      totalAmount,
			Total:       totalAmount,
			Currency:    "UZS",
			OrderSource: "PROCUREMENT",
			CreatedAt:   createdAt.Format(time.RFC3339),
			Items:       req.Items,
		}
		return nil
	})
	if err != nil {
		return ProcurementOrderResponse{}, fmt.Errorf("create procurement order: %w", err)
	}
	if s.Cache != nil {
		s.Cache.Invalidate(ctx, cache.PrefixActiveOrders+req.RetailerID)
	}
	return response, nil
}

func buildProcurementMutations(input procurementMutationInput) ([]*spanner.Mutation, int64) {
	totalAmount := int64(0)
	lineMutations := make([]*spanner.Mutation, 0, len(input.Request.Items))
	for _, item := range input.Request.Items {
		product := input.Products[item.ProductID]
		totalAmount += product.Price * item.Quantity
		lineMutations = append(lineMutations, spanner.Insert("OrderLineItems",
			[]string{"LineItemId", "OrderId", "SkuId", "Quantity", "UnitPrice", "Currency", "Status"},
			[]interface{}{fmt.Sprintf("LI-%s", GenerateSecureToken()), input.OrderID, item.ProductID, item.Quantity, product.Price, "UZS", "PENDING"},
		))
	}

	orderMutation := spanner.Insert("Orders",
		[]string{"OrderId", "RetailerId", "SupplierId", "Amount", "Currency", "PaymentGateway", "State", "ShopLocation", "RouteId", "OrderSource", "ScheduleShard", "DeliveryToken", "Version", "CreatedAt"},
		[]interface{}{input.OrderID, input.Request.RetailerID, input.SupplierID, totalAmount, "UZS", "PENDING", "PENDING", fmt.Sprintf("POINT(%f %f)", input.Longitude, input.Latitude), spanner.NullString{Valid: false}, spanner.NullString{StringVal: "PROCUREMENT", Valid: true}, hotspot.ShardForKey(input.OrderID), spanner.NullString{Valid: false}, int64(1), spanner.CommitTimestamp},
	)
	return append([]*spanner.Mutation{orderMutation}, lineMutations...), totalAmount
}

func validateProcurementItems(items []ProcurementItem) error {
	for _, item := range items {
		if item.ProductID == "" || item.Quantity <= 0 {
			return ErrProcurementInvalidItem
		}
	}
	return nil
}

func readProcurementRetailerLocation(ctx context.Context, txn *spanner.ReadWriteTransaction, retailerID string) (float64, float64, error) {
	row, err := txn.ReadRow(ctx, "Retailers", spanner.Key{retailerID}, []string{"Latitude", "Longitude"})
	if errors.Is(err, spanner.ErrRowNotFound) {
		return 0, 0, ErrProcurementRetailerGone
	}
	if err != nil {
		return 0, 0, fmt.Errorf("read retailer location: %w", err)
	}

	var latitude, longitude spanner.NullFloat64
	if err := row.Columns(&latitude, &longitude); err != nil {
		return 0, 0, fmt.Errorf("decode retailer location: %w", err)
	}
	return latitude.Float64, longitude.Float64, nil
}

func readProcurementProducts(ctx context.Context, txn *spanner.ReadWriteTransaction, items []ProcurementItem) (map[string]procurementProduct, string, error) {
	productIDs := make([]string, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		if !seen[item.ProductID] {
			productIDs = append(productIDs, item.ProductID)
			seen[item.ProductID] = true
		}
	}

	stmt := spanner.Statement{
		SQL:    `SELECT SkuId, SupplierId, BasePrice FROM SupplierProducts WHERE SkuId IN UNNEST(@ids) AND IsActive = TRUE`,
		Params: map[string]interface{}{"ids": productIDs},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()

	products := make(map[string]procurementProduct, len(productIDs))
	supplierID := ""
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, "", fmt.Errorf("query procurement products: %w", err)
		}
		var product procurementProduct
		if err := row.Columns(&product.SKU, &product.SupplierID, &product.Price); err != nil {
			return nil, "", fmt.Errorf("decode procurement product: %w", err)
		}
		if supplierID == "" {
			supplierID = product.SupplierID
		}
		if supplierID != product.SupplierID {
			return nil, "", ErrProcurementMixedSupplier
		}
		products[product.SKU] = product
	}
	for _, productID := range productIDs {
		if _, ok := products[productID]; !ok {
			return nil, "", fmt.Errorf("%w: %s", ErrProcurementUnknownSKU, productID)
		}
	}
	return products, supplierID, nil
}

func writeProcurementError(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusInternalServerError
	message := `{"error":"Order creation failed"}`
	switch {
	case errors.Is(err, ErrProcurementEmptyItems):
		status = http.StatusUnprocessableEntity
		message = `{"error":"items must not be empty"}`
	case errors.Is(err, ErrProcurementInvalidItem):
		status = http.StatusUnprocessableEntity
		message = `{"error":"each item requires product_id and positive quantity"}`
	case errors.Is(err, ErrProcurementUnknownSKU):
		status = http.StatusUnprocessableEntity
		message = `{"error":"one or more products are unavailable"}`
	case errors.Is(err, ErrProcurementMixedSupplier):
		status = http.StatusUnprocessableEntity
		message = `{"error":"legacy procurement orders must contain products from one supplier"}`
	case errors.Is(err, ErrProcurementRetailerGone):
		status = http.StatusNotFound
		message = `{"error":"retailer profile not found"}`
	default:
		slog.ErrorContext(r.Context(), "procurement order create failed", "err", err)
	}
	http.Error(w, message, status)
}
