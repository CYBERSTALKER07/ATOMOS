package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"cloud.google.com/go/spanner"
	h3 "github.com/uber/h3-go/v4"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/promotion"
)

const orderH3Resolution = 7

// UnifiedCheckoutLineItem is one cart row from retailer clients.
type UnifiedCheckoutLineItem struct {
	SkuID     string `json:"sku_id"`
	Quantity  int64  `json:"quantity"`
	UnitPrice int64  `json:"unit_price"`
}

// UnifiedCheckoutRequest is POST /v1/checkout/unified when the body carries cart items.
type UnifiedCheckoutRequest struct {
	RetailerID     string                    `json:"retailer_id"`
	PaymentGateway string                    `json:"payment_gateway"`
	Latitude       float64                   `json:"latitude"`
	Longitude      float64                   `json:"longitude"`
	Items          []UnifiedCheckoutLineItem `json:"items"`
	DeliveryMode          string `json:"delivery_mode,omitempty"`
	RequestedDeliveryDate string `json:"requested_delivery_date,omitempty"`
	DeliverBefore         string `json:"deliver_before,omitempty"`
	DeliveryPriority      string `json:"delivery_priority,omitempty"`
	CheckoutPolicyToken   string `json:"checkout_policy_token,omitempty"`
}

// SupplierOrderResult is one supplier slice returned to clients.
type SupplierOrderResult struct {
	OrderID      string `json:"order_id"`
	SupplierID   string `json:"supplier_id"`
	SupplierName string `json:"supplier_name"`
	Total        int64  `json:"total"`
	Currency     string `json:"currency"`
	ItemCount    int    `json:"item_count"`
}

// UnifiedCheckoutResponse matches retailer desktop/Android/iOS contracts.
type UnifiedCheckoutResponse struct {
	Status               string                `json:"status"`
	InvoiceID            string                `json:"invoice_id"`
	Total                int64                 `json:"total"`
	Currency             string                `json:"currency"`
	SupplierOrders       []SupplierOrderResult `json:"supplier_orders"`
	BackorderedItemCount int                   `json:"backordered_item_count,omitempty"`
	StockWarnings        []StockWarning        `json:"stock_warnings,omitempty"`
	BackorderOrderID     string                `json:"backorder_order_id,omitempty"`
}

// CheckoutSnapshot returns order totals for payment initiation.
func (s *Service) CheckoutSnapshot(ctx context.Context, orderID, retailerID string) (totalMinor int64, currency string, err error) {
	ctxData, err := s.CheckoutOrderContext(ctx, orderID, retailerID)
	if err != nil {
		return 0, "", err
	}
	return ctxData.TotalMinor, ctxData.Currency, nil
}

// CheckoutOrderContext returns order totals and routing metadata for payment flows.
func (s *Service) CheckoutOrderContext(ctx context.Context, orderID, retailerID string) (CheckoutOrderContext, error) {
	orderID = strings.TrimSpace(orderID)
	retailerID = strings.TrimSpace(retailerID)
	if orderID == "" || retailerID == "" {
		return CheckoutOrderContext{}, errors.New("order_id and retailer_id required")
	}
	o, found, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return CheckoutOrderContext{}, fmt.Errorf("load order %s: %w", orderID, err)
	}
	if !found {
		return CheckoutOrderContext{}, ErrOrderNotFound
	}
	if o.RetailerID != retailerID {
		return CheckoutOrderContext{}, ErrOrderForbidden
	}
	if o.Status == StatusCompleted || o.Status == StatusCancelled {
		return CheckoutOrderContext{}, fmt.Errorf("order cannot be paid in status: %s", o.Status)
	}
	if o.Status == StatusBackordered {
		return CheckoutOrderContext{}, ErrBackorderPaymentDeferred
	}
	if !OrderPayableAtDelivery(o.Status) {
		return CheckoutOrderContext{}, ErrPaymentBeforeDelivery
	}
	return CheckoutOrderContext{
		TotalMinor:  o.TotalMinor,
		Currency:    o.Currency,
		SupplierID:  o.SupplierID,
		WarehouseID: o.WarehouseID,
	}, nil
}

// OrderPayableAtDelivery reports whether retailer card/cash checkout is allowed.
// Payment is collected after offload (ARRIVED) or while awaiting settlement.
func OrderPayableAtDelivery(status Status) bool {
	switch status {
	case StatusArrived, StatusAwaitingPayment:
		return true
	default:
		return false
	}
}

// HandleUnifiedCheckout serves POST /v1/checkout/unified for cart payloads.
func (s *Service) HandleUnifiedCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Subject == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotency(w, r, body) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	var req UnifiedCheckoutRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	if req.RetailerID != "" && req.RetailerID != claims.Subject {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "retailer_id_mismatch"})
		return
	}

	resp, err := s.UnifiedCheckout(r.Context(), claims.Subject, req)
	if err != nil {
		s.log.Warn("unified checkout failed", "retailer_id", claims.Subject, "err", err)
		switch {
		case errors.Is(err, ErrZoneMiss):
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": ErrZoneMiss.Error()})
		case errors.Is(err, ErrServiceabilityUnavailable):
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": ErrServiceabilityUnavailable.Error()})
		case errors.Is(err, ErrInventoryExhausted):
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": ErrInventoryExhausted.Error()})
		default:
			if raw, ok := MarshalInventoryCheckoutError(err); ok {
				writeJSONBytes(w, http.StatusConflict, raw)
				return
			}
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		}
		return
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, body, http.StatusCreated, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusCreated, respBytes)
}

// UnifiedCheckout creates one pegasusX-scoped order from the retailer cart.
func (s *Service) UnifiedCheckout(ctx context.Context, retailerID string, req UnifiedCheckoutRequest) (UnifiedCheckoutResponse, error) {
	if retailerID == "" {
		return UnifiedCheckoutResponse{}, errors.New("retailer_id required from session")
	}
	if len(req.Items) == 0 {
		return UnifiedCheckoutResponse{}, errors.New("items must not be empty")
	}

	lat, lng, err := s.resolveRetailerCoordinates(ctx, retailerID, req.Latitude, req.Longitude)
	if err != nil {
		return UnifiedCheckoutResponse{}, err
	}
	h3Cell, err := h3CellFromLatLng(lat, lng)
	if err != nil {
		return UnifiedCheckoutResponse{}, fmt.Errorf("derive h3 cell: %w", err)
	}

	lineItems, err := s.authoritativeCheckoutLines(ctx, retailerID, req.Items)
	if err != nil {
		return UnifiedCheckoutResponse{}, err
	}

	created, err := s.Create(ctx, retailerID, CreateRequest{
		LineItems:             lineItems,
		H3Cell:                h3Cell,
		Lat:                   lat,
		Lng:                   lng,
		DeliveryMode:          req.DeliveryMode,
		RequestedDeliveryDate: req.RequestedDeliveryDate,
		DeliverBefore:         req.DeliverBefore,
		DeliveryPriority:      req.DeliveryPriority,
		CheckoutPolicyToken:   req.CheckoutPolicyToken,
	})
	if err != nil {
		return UnifiedCheckoutResponse{}, err
	}

	supplierName := strings.TrimSpace(s.supplierName)
	if supplierName == "" {
		supplierName = "Supplier"
	}

	invoiceID := strings.Replace(s.newID(), "ord_", "inv_", 1)

	return UnifiedCheckoutResponse{
		Status:    "ok",
		InvoiceID: invoiceID,
		Total:     created.TotalMinor,
		Currency:  created.Currency,
		SupplierOrders: []SupplierOrderResult{{
			OrderID:      created.OrderID,
			SupplierID:   s.supplierID,
			SupplierName: supplierName,
			Total:        created.TotalMinor,
			Currency:     created.Currency,
			ItemCount:    len(lineItems),
		}},
		BackorderedItemCount: created.BackorderedItemCount,
		StockWarnings:        created.StockWarnings,
		BackorderOrderID:     created.BackorderOrderID,
	}, nil
}

func (s *Service) resolveRetailerCoordinates(ctx context.Context, retailerID string, lat, lng float64) (float64, float64, error) {
	if lat != 0 || lng != 0 {
		return lat, lng, nil
	}
	if s.spannerClient == nil {
		return 0, 0, errors.New("lat/lng required")
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "Retailers",
		spanner.Key{retailerID},
		[]string{"Latitude", "Longitude"},
	)
	if err != nil {
		return 0, 0, fmt.Errorf("load retailer coordinates: %w", err)
	}
	var storedLat, storedLng spanner.NullFloat64
	if err := row.Columns(&storedLat, &storedLng); err != nil {
		return 0, 0, fmt.Errorf("decode retailer coordinates: %w", err)
	}
	if storedLat.Valid && storedLng.Valid && (storedLat.Float64 != 0 || storedLng.Float64 != 0) {
		return storedLat.Float64, storedLng.Float64, nil
	}
	return 0, 0, errors.New("lat/lng required")
}

func h3CellFromLatLng(lat, lng float64) (string, error) {
	if lat == 0 && lng == 0 {
		return "", errors.New("lat/lng required")
	}
	cell, err := h3.LatLngToCell(h3.LatLng{Lat: lat, Lng: lng}, orderH3Resolution)
	if err != nil {
		return "", err
	}
	if !cell.IsValid() {
		return "", errors.New("invalid h3 cell")
	}
	return strings.ToLower(cell.String()), nil
}

func (s *Service) authoritativeCheckoutLines(
	ctx context.Context,
	retailerID string,
	items []UnifiedCheckoutLineItem,
) ([]LineItem, error) {
	inputs := make([]promotion.LineInput, 0, len(items))
	for i, item := range items {
		sku := strings.TrimSpace(item.SkuID)
		if sku == "" || item.Quantity <= 0 {
			return nil, fmt.Errorf("items[%d] requires sku_id and positive quantity", i)
		}
		if item.UnitPrice < 0 {
			return nil, fmt.Errorf("items[%d].unit_price must be >= 0", i)
		}
		inputs = append(inputs, promotion.LineInput{
			ProductID: sku,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
			Currency:  s.currency,
		})
	}
	if s.promotions != nil {
		quote, err := s.promotions.QuoteCheckout(ctx, s.supplierID, retailerID, inputs)
		if err != nil {
			return nil, err
		}
		lineItems := make([]LineItem, 0, len(quote.Lines))
		for _, line := range quote.Lines {
			lineItems = append(lineItems, LineItem{
				SKU:       line.ProductID,
				Name:      line.ProductID,
				Quantity:  line.Quantity,
				UnitPrice: line.UnitPrice,
			})
		}
		return s.enrichLineItemVolumes(ctx, lineItems)
	}
	lineItems := make([]LineItem, 0, len(inputs))
	for _, line := range inputs {
		lineItems = append(lineItems, LineItem{
			SKU:       line.ProductID,
			Name:      line.ProductID,
			Quantity:  line.Quantity,
			UnitPrice: line.UnitPrice,
		})
	}
	return s.enrichLineItemVolumes(ctx, lineItems)
}
