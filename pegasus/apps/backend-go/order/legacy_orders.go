package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"
	apperrors "backend-go/errors"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

var errLegacyOrderForbidden = errors.New("order outside caller scope")

type legacyOrderItemResponse struct {
	LineItemID  string `json:"line_item_id"`
	OrderID     string `json:"order_id"`
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	SkuID       string `json:"sku_id"`
	SkuName     string `json:"sku_name"`
	Quantity    int64  `json:"quantity"`
	UnitPrice   int64  `json:"unit_price"`
	LineTotal   int64  `json:"line_total"`
	Status      string `json:"status"`
}

type legacyOrderDetailResponse struct {
	ID                   string                    `json:"id"`
	OrderID              string                    `json:"order_id"`
	RetailerID           string                    `json:"retailer_id"`
	RetailerName         string                    `json:"retailer_name"`
	SupplierID           string                    `json:"supplier_id"`
	DriverID             string                    `json:"driver_id,omitempty"`
	State                string                    `json:"state"`
	TotalAmount          int64                     `json:"total_amount"`
	Amount               int64                     `json:"amount"`
	DeliveryAddress      string                    `json:"delivery_address"`
	Latitude             float64                   `json:"latitude"`
	Longitude            float64                   `json:"longitude"`
	QRToken              string                    `json:"qr_token"`
	DeliveryToken        string                    `json:"delivery_token"`
	PaymentGateway       string                    `json:"payment_gateway"`
	PaymentStatus        string                    `json:"payment_status"`
	CreatedAt            string                    `json:"created_at"`
	UpdatedAt            string                    `json:"updated_at"`
	EstimatedArrivalAt   *string                   `json:"estimated_arrival_at,omitempty"`
	EstimatedDurationSec *int64                    `json:"eta_duration_sec,omitempty"`
	EstimatedDistanceM   *int64                    `json:"eta_distance_m,omitempty"`
	RouteID              *string                   `json:"route_id"`
	SequenceIndex        int64                     `json:"sequence_index"`
	OrderSource          *string                   `json:"order_source"`
	AutoConfirmAt        *string                   `json:"auto_confirm_at"`
	DeliverBefore        *string                   `json:"deliver_before"`
	Version              int64                     `json:"version"`
	Items                []legacyOrderItemResponse `json:"items"`
}

type legacyOrderScope struct {
	clause string
	params map[string]interface{}
}

// HandleLegacyOrdersPath serves the legacy /v1/orders/* surface used by the
// driver and retailer clients: GET /v1/orders/{id} and PATCH /v1/orders/{id}/status|state.
func HandleLegacyOrdersPath(svc *OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			svc.handleLegacyOrderDetail(w, r)
		case http.MethodPatch:
			svc.handleLegacyOrderStatusPatch(w, r)
		default:
			apperrors.MethodNotAllowed(w, r)
		}
	}
}

func (s *OrderService) handleLegacyOrderDetail(w http.ResponseWriter, r *http.Request) {
	claims, _ := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	parts := legacyOrderPathParts(r.URL.Path)
	if claims == nil {
		apperrors.Unauthorized(w, r, "authentication required")
		return
	}
	if len(parts) != 1 || parts[0] == "" {
		apperrors.NotFound(w, r, "order detail endpoint not found")
		return
	}

	orderDetail, err := s.GetLegacyOrderDetail(r.Context(), claims, parts[0])
	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(orderDetail)
		return
	}
	if errors.Is(err, errLegacyOrderForbidden) {
		apperrors.Forbidden(w, r, "order is outside caller scope")
		return
	}
	if errors.Is(err, spanner.ErrRowNotFound) {
		apperrors.NotFound(w, r, fmt.Sprintf("order %s not found", parts[0]))
		return
	}

	slog.ErrorContext(r.Context(), "legacy_orders.detail_failed",
		"trace_id", telemetry.TraceIDFromContext(r.Context()),
		"order_id", parts[0],
		"role", claims.Role,
		"actor_id", claims.UserID,
		"error", err.Error())
	apperrors.InternalError(w, r, "failed to load order detail")
	return
}

func (s *OrderService) handleLegacyOrderStatusPatch(w http.ResponseWriter, r *http.Request) {
	orderID, ok := legacyOrderPatchTarget(r.URL.Path)
	if !ok {
		apperrors.NotFound(w, r, "order status endpoint not found")
		return
	}

	var req struct {
		Status string `json:"status"`
		State  string `json:"state"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperrors.BadRequest(w, r, "invalid JSON body")
		return
	}
	status := req.Status
	if status == "" {
		status = req.State
	}
	if status == "" {
		apperrors.BadRequest(w, r, "status is required")
		return
	}

	_, err := s.Client.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: `UPDATE Orders SET State = @state WHERE OrderId = @orderId`,
			Params: map[string]interface{}{
				"state":   status,
				"orderId": orderID,
			},
		}
		_, err := txn.Update(ctx, stmt)
		return err
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "legacy_orders.patch_failed",
			"trace_id", telemetry.TraceIDFromContext(r.Context()),
			"order_id", orderID,
			"status", status,
			"error", err.Error())
		apperrors.InternalError(w, r, "failed to patch order state")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(fmt.Sprintf(`{"status":"SUCCESS","message":"Order %s patched to %s"}`, orderID, status)))
}

// GetLegacyOrderDetail returns a superset payload compatible with the native
// driver clients and retailer desktop's selected-order detail request.
func (s *OrderService) GetLegacyOrderDetail(ctx context.Context, claims *auth.PegasusClaims, orderID string) (*legacyOrderDetailResponse, error) {
	scope, err := legacyOrderScopeForClaims(claims)
	if err != nil {
		return nil, err
	}

	orderDetail, supplierID, err := s.fetchLegacyOrderHeader(ctx, orderID, scope)
	if err != nil {
		return nil, err
	}
	items, err := s.fetchLegacyOrderItems(ctx, orderID, supplierID)
	if err != nil {
		return nil, err
	}
	orderDetail.Items = items
	return orderDetail, nil
}

func (s *OrderService) fetchLegacyOrderHeader(ctx context.Context, orderID string, scope legacyOrderScope) (*legacyOrderDetailResponse, string, error) {
	params := map[string]interface{}{"orderId": orderID}
	for key, value := range scope.params {
		params[key] = value
	}
	stmt := spanner.Statement{
		SQL: `SELECT o.OrderId,
		             o.RetailerId,
		             COALESCE(r.ShopName, r.Name, o.RetailerId) AS RetailerName,
		             COALESCE(o.SupplierId, '') AS SupplierId,
		             COALESCE(o.DriverId, '') AS DriverId,
		             o.State,
		             COALESCE(o.Amount, 0) AS TotalAmount,
		             COALESCE(r.ShopLocation, '') AS DeliveryAddress,
		             COALESCE(r.Latitude, 0) AS Latitude,
		             COALESCE(r.Longitude, 0) AS Longitude,
		             COALESCE(o.DeliveryToken, '') AS DeliveryToken,
		             COALESCE(o.PaymentGateway, '') AS PaymentGateway,
		             COALESCE(o.PaymentStatus, '') AS PaymentStatus,
		             o.CreatedAt,
		             o.EstimatedArrivalAt,
		             o.EstimatedDurationSec,
		             o.EstimatedDistanceM,
		             o.RouteId,
		             COALESCE(o.SequenceIndex, 0) AS SequenceIndex,
		             o.OrderSource,
		             o.AutoConfirmAt,
		             o.DeliverBefore,
		             COALESCE(o.Version, 0) AS Version
		      FROM Orders o
		      LEFT JOIN Retailers r ON o.RetailerId = r.RetailerId
		      WHERE o.OrderId = @orderId AND ` + scope.clause,
		Params: params,
	}

	iter := s.Client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			return nil, "", spanner.ErrRowNotFound
		}
		return nil, "", fmt.Errorf("query order detail %s: %w", orderID, err)
	}

	return scanLegacyOrderHeader(row)
}

func scanLegacyOrderHeader(row *spanner.Row) (*legacyOrderDetailResponse, string, error) {
	var orderID, retailerID, retailerName, supplierID, driverID string
	var state, deliveryAddress, deliveryToken, paymentGateway, paymentStatus string
	var totalAmount, sequenceIndex, version int64
	var latitude, longitude float64
	var createdAt spanner.NullTime
	var estimatedArrivalAt, autoConfirmAt, deliverBefore spanner.NullTime
	var estimatedDurationSec, estimatedDistanceM spanner.NullInt64
	var routeID, orderSource spanner.NullString
	if err := row.Columns(
		&orderID, &retailerID, &retailerName, &supplierID, &driverID, &state,
		&totalAmount, &deliveryAddress, &latitude, &longitude, &deliveryToken,
		&paymentGateway, &paymentStatus, &createdAt, &estimatedArrivalAt,
		&estimatedDurationSec, &estimatedDistanceM, &routeID, &sequenceIndex,
		&orderSource, &autoConfirmAt, &deliverBefore, &version,
	); err != nil {
		return nil, "", fmt.Errorf("scan order detail: %w", err)
	}
	createdAtText := ""
	if createdAt.Valid {
		createdAtText = createdAt.Time.Format(time.RFC3339)
	}

	response := &legacyOrderDetailResponse{
		ID:              orderID,
		OrderID:         orderID,
		RetailerID:      retailerID,
		RetailerName:    retailerName,
		SupplierID:      supplierID,
		DriverID:        driverID,
		State:           state,
		TotalAmount:     totalAmount,
		Amount:          totalAmount,
		DeliveryAddress: deliveryAddress,
		Latitude:        latitude,
		Longitude:       longitude,
		QRToken:         deliveryToken,
		DeliveryToken:   deliveryToken,
		PaymentGateway:  paymentGateway,
		PaymentStatus:   paymentStatus,
		CreatedAt:       createdAtText,
		UpdatedAt:       createdAtText,
		RouteID:         nullableString(routeID),
		SequenceIndex:   sequenceIndex,
		OrderSource:     nullableString(orderSource),
		AutoConfirmAt:   nullableTime(autoConfirmAt),
		DeliverBefore:   nullableTime(deliverBefore),
		Version:         version,
		Items:           []legacyOrderItemResponse{},
	}
	if estimatedArrivalAt.Valid {
		formatted := estimatedArrivalAt.Time.Format(time.RFC3339)
		response.EstimatedArrivalAt = &formatted
	}
	if estimatedDurationSec.Valid {
		value := estimatedDurationSec.Int64
		response.EstimatedDurationSec = &value
	}
	if estimatedDistanceM.Valid {
		value := estimatedDistanceM.Int64
		response.EstimatedDistanceM = &value
	}
	return response, supplierID, nil
}

func (s *OrderService) fetchLegacyOrderItems(ctx context.Context, orderID string, supplierID string) ([]legacyOrderItemResponse, error) {
	query := `SELECT li.LineItemId,
	                 li.OrderId,
	                 li.SkuId,
	                 COALESCE(sp.Name, li.SkuId) AS ProductName,
	                 li.Quantity,
	                 COALESCE(li.UnitPrice, 0) AS UnitPrice,
	                 COALESCE(li.Status, 'PENDING') AS Status
	          FROM OrderLineItems li
	          LEFT JOIN SupplierProducts sp ON li.SkuId = sp.SkuId`
	params := map[string]interface{}{"orderId": orderID}
	if supplierID != "" {
		query += ` AND sp.SupplierId = @supplierId`
		params["supplierId"] = supplierID
	}
	query += ` WHERE li.OrderId = @orderId ORDER BY li.LineItemId`

	iter := s.Client.Single().Query(ctx, spanner.Statement{SQL: query, Params: params})
	defer iter.Stop()

	items := make([]legacyOrderItemResponse, 0, 4)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return items, nil
		}
		if err != nil {
			return nil, fmt.Errorf("query order items %s: %w", orderID, err)
		}

		var lineItemID, itemOrderID, skuID, productName, status string
		var quantity, unitPrice int64
		if err := row.Columns(&lineItemID, &itemOrderID, &skuID, &productName, &quantity, &unitPrice, &status); err != nil {
			return nil, fmt.Errorf("scan order item %s: %w", orderID, err)
		}
		items = append(items, legacyOrderItemResponse{
			LineItemID:  lineItemID,
			OrderID:     itemOrderID,
			ProductID:   skuID,
			ProductName: productName,
			SkuID:       skuID,
			SkuName:     productName,
			Quantity:    quantity,
			UnitPrice:   unitPrice,
			LineTotal:   quantity * unitPrice,
			Status:      status,
		})
	}
}

func legacyOrderScopeForClaims(claims *auth.PegasusClaims) (legacyOrderScope, error) {
	if claims == nil {
		return legacyOrderScope{}, errLegacyOrderForbidden
	}

	switch claims.Role {
	case "DRIVER":
		return legacyOrderScope{clause: `o.DriverId = @actorId`, params: map[string]interface{}{"actorId": claims.UserID}}, nil
	case "RETAILER":
		return legacyOrderScope{clause: `o.RetailerId = @actorId`, params: map[string]interface{}{"actorId": claims.UserID}}, nil
	case "ADMIN", "SUPPLIER":
		supplierID := claims.ResolveSupplierID()
		if supplierID == "" {
			return legacyOrderScope{}, errLegacyOrderForbidden
		}
		return legacyOrderScope{clause: `o.SupplierId = @supplierId`, params: map[string]interface{}{"supplierId": supplierID}}, nil
	default:
		return legacyOrderScope{}, errLegacyOrderForbidden
	}
}

func legacyOrderPathParts(path string) []string {
	trimmed := strings.Trim(strings.TrimPrefix(path, "/v1/orders/"), "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func legacyOrderPatchTarget(path string) (string, bool) {
	parts := legacyOrderPathParts(path)
	if len(parts) != 2 || parts[0] == "" {
		return "", false
	}
	switch parts[1] {
	case "status", "state":
		return parts[0], true
	default:
		return "", false
	}
}

func nullableString(value spanner.NullString) *string {
	if !value.Valid || value.StringVal == "" {
		return nil
	}
	result := value.StringVal
	return &result
}

func nullableTime(value spanner.NullTime) *string {
	if !value.Valid {
		return nil
	}
	formatted := value.Time.Format(time.RFC3339)
	return &formatted
}
