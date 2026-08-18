package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	h3 "github.com/uber/h3-go/v4"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/promotion"
)

const orderH3Resolution = 7

// UnifiedCheckoutLineItem is one cart row from retailer clients.
type UnifiedCheckoutLineItem struct {
	SkuID      string `json:"sku_id"`
	Quantity   int64  `json:"quantity"`
	UnitPrice  int64  `json:"unit_price"`
	SupplierID string `json:"supplier_id,omitempty"`
}

// UnifiedCheckoutRequest is POST /v1/checkout/unified when the body carries cart items.
type UnifiedCheckoutRequest struct {
	RetailerID            string                    `json:"retailer_id"`
	PaymentGateway        string                    `json:"payment_gateway"`
	Latitude              float64                   `json:"latitude"`
	Longitude             float64                   `json:"longitude"`
	Items                 []UnifiedCheckoutLineItem `json:"items"`
	DeliveryMode          string                    `json:"delivery_mode,omitempty"`
	RequestedDeliveryDate string                    `json:"requested_delivery_date,omitempty"`
	DeliverBefore         string                    `json:"deliver_before,omitempty"`
	DeliveryPriority      string                    `json:"delivery_priority,omitempty"`
	CheckoutPolicyToken   string                    `json:"checkout_policy_token,omitempty"`
	// Currency optional; same rules as CreateRequest.Currency.
	Currency string `json:"currency,omitempty"`
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
	Status               string                    `json:"status"`
	InvoiceID            string                    `json:"invoice_id"`
	Total                int64                     `json:"total"`
	Currency             string                    `json:"currency"`
	MarketCode           string                    `json:"market_code,omitempty"`
	ParentOrderID        string                    `json:"parent_order_id,omitempty"`
	SupplierOrders       []SupplierOrderResult     `json:"supplier_orders"`
	BackorderedItemCount int                       `json:"backordered_item_count,omitempty"`
	StockWarnings        []StockWarning            `json:"stock_warnings,omitempty"`
	BackorderOrderID     string                    `json:"backorder_order_id,omitempty"`
	SupplierErrors       []SupplierCheckoutFailure `json:"supplier_errors,omitempty"`
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
	o, found, err := s.loadOrderForRequest(ctx, orderID)
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
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	// B3 M-P0-4: multi-user JWT — org is tenant for RetailerId on orders.
	retailerID := auth.ResolveRetailerOrgID(claims)
	if retailerID == "" {
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

	if req.RetailerID != "" && req.RetailerID != retailerID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "retailer_id_mismatch"})
		return
	}

	resp, err := s.UnifiedCheckout(r.Context(), retailerID, req)
	if err != nil {
		s.log.Warn("unified checkout failed", "retailer_id", retailerID, "err", err)
		var multiErr *MultiSupplierCheckoutError
		if errors.As(err, &multiErr) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"error":           multiErr.Error(),
				"supplier_errors": multiErr.Failures,
			})
			return
		}
		switch {
		case errors.Is(err, ErrZoneMiss):
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": ErrZoneMiss.Error()})
		case errors.Is(err, ErrServiceabilityUnavailable):
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": ErrServiceabilityUnavailable.Error()})
		case errors.Is(err, ErrInventoryExhausted):
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": ErrInventoryExhausted.Error()})
		case errors.Is(err, ErrCurrencyNotAllowed):
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": ErrCurrencyNotAllowed.Error(), "code": ErrCurrencyNotAllowed.Error()})
		case errors.Is(err, auth.ErrMarketPackUnknown), errors.Is(err, auth.ErrMarketPackNotShipped),
			errors.Is(err, auth.ErrPackGatewayForbidden), errors.Is(err, auth.ErrPackCurrencyMismatch),
			errors.Is(err, auth.ErrGeographyIncomplete), errors.Is(err, auth.ErrCrossMarketDeferred):
			st, code := auth.CheckoutPackHTTPStatus(err)
			writeJSON(w, st, map[string]string{"error": code})
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

// UnifiedCheckout creates one or more supplier-scoped child orders from the retailer cart.
// When MULTI_SUPPLIER_CHECKOUT_ENABLED, always creates a ParentOrders rollup (including N=1).
func (s *Service) applyCheckoutPack(ctx context.Context, req *UnifiedCheckoutRequest) (auth.MarketPack, error) {
	pack, err := auth.CheckoutPackFromContext(ctx)
	if err != nil {
		return auth.MarketPack{}, err
	}
	ccy, err := auth.ResolveCheckoutCurrency(pack, req.Currency)
	if err != nil {
		return auth.MarketPack{}, err
	}
	req.Currency = ccy
	if gw := strings.TrimSpace(req.PaymentGateway); gw != "" {
		if err := auth.AssertPackPSP(pack, gw); err != nil {
			return auth.MarketPack{}, err
		}
	}
	return pack, nil
}

func (s *Service) UnifiedCheckout(ctx context.Context, retailerID string, req UnifiedCheckoutRequest) (UnifiedCheckoutResponse, error) {
	if retailerID == "" {
		return UnifiedCheckoutResponse{}, errors.New("retailer_id required from session")
	}
	if len(req.Items) == 0 {
		return UnifiedCheckoutResponse{}, errors.New("items must not be empty")
	}
	pack, err := s.applyCheckoutPack(ctx, &req)
	if err != nil {
		return UnifiedCheckoutResponse{}, err
	}

	store, err := s.resolveCheckoutStore(ctx, retailerID, req.Latitude, req.Longitude)
	if err != nil {
		return UnifiedCheckoutResponse{}, err
	}
	lat, lng := store.Lat, store.Lng
	h3Cell, err := h3CellFromLatLng(lat, lng)
	if err != nil {
		return UnifiedCheckoutResponse{}, fmt.Errorf("derive h3 cell: %w", err)
	}

	if MultiSupplierCheckoutEnabled() {
		// Do not quote under the JWT trading-partner tenant for mixed carts —
		// each Create leg re-quotes under that supplier's TenantContext.
		draft := make([]LineItem, 0, len(req.Items))
		for i, item := range req.Items {
			sku := strings.TrimSpace(item.SkuID)
			if sku == "" || item.Quantity <= 0 {
				return UnifiedCheckoutResponse{}, fmt.Errorf("items[%d] requires sku_id and positive quantity", i)
			}
			if item.UnitPrice < 0 {
				return UnifiedCheckoutResponse{}, fmt.Errorf("items[%d].unit_price must be >= 0", i)
			}
			draft = append(draft, LineItem{
				SKU:       sku,
				Quantity:  item.Quantity,
				UnitPrice: item.UnitPrice,
			})
		}
		return s.unifiedCheckoutMultiSupplier(ctx, retailerID, req, draft, h3Cell, lat, lng, pack)
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
		Currency:              req.Currency,
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
		Status:     "ok",
		InvoiceID:  invoiceID,
		Total:      created.TotalMinor,
		Currency:   created.Currency,
		MarketCode: pack.Code,
		SupplierOrders: []SupplierOrderResult{{
			OrderID:      created.OrderID,
			SupplierID:   s.resolveSupplierScope(ctx),
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

func (s *Service) unifiedCheckoutMultiSupplier(
	ctx context.Context,
	retailerID string,
	req UnifiedCheckoutRequest,
	lineItems []LineItem,
	h3Cell string,
	lat, lng float64,
	pack auth.MarketPack,
) (UnifiedCheckoutResponse, error) {
	groups, err := s.groupCheckoutLinesBySupplier(ctx, req.Items, lineItems)
	if err != nil {
		return UnifiedCheckoutResponse{}, err
	}
	if len(groups) == 0 {
		return UnifiedCheckoutResponse{}, errors.New("items must not be empty")
	}
	if err := assertChildSuppliersSameMarket(ctx, pack, groups); err != nil {
		return UnifiedCheckoutResponse{}, err
	}

	parentID := strings.Replace(s.newID(), "ord_", "par_", 1)
	currency := strings.TrimSpace(req.Currency)
	if currency == "" {
		currency = pack.CurrencyCode
	}
	if currency == "" {
		currency = s.currency
	}
	if err := s.insertParentOrder(ctx, parentID, retailerID, currency, len(groups)); err != nil {
		return UnifiedCheckoutResponse{}, err
	}

	supplierName := strings.TrimSpace(s.supplierName)
	if supplierName == "" {
		supplierName = "Supplier"
	}

	createdOrders := make([]Order, 0, len(groups))
	supplierOrders := make([]SupplierOrderResult, 0, len(groups))
	var (
		totalMinor           int64
		backorderedItemCount int
		stockWarnings        []StockWarning
		backorderOrderID     string
		failures             []SupplierCheckoutFailure
	)

	for _, g := range groups {
		legCtx := createCtxForSupplier(ctx, g.SupplierID)
		created, createErr := s.Create(legCtx, retailerID, CreateRequest{
			LineItems:             g.Lines,
			H3Cell:                h3Cell,
			Lat:                   lat,
			Lng:                   lng,
			DeliveryMode:          req.DeliveryMode,
			RequestedDeliveryDate: req.RequestedDeliveryDate,
			DeliverBefore:         req.DeliverBefore,
			DeliveryPriority:      req.DeliveryPriority,
			CheckoutPolicyToken:   req.CheckoutPolicyToken,
			Currency:              req.Currency,
			SupplierID:            g.SupplierID,
			ParentOrderID:         parentID,
		})
		if createErr != nil {
			failures = append(failures, SupplierCheckoutFailure{
				SupplierID: g.SupplierID,
				Error:      createErr.Error(),
			})
			s.compensateParentCheckout(ctx, parentID, createdOrders)
			return UnifiedCheckoutResponse{}, &MultiSupplierCheckoutError{
				Failures: failures,
				Message:  fmt.Sprintf("multi_supplier_checkout_failed: supplier %s: %v", g.SupplierID, createErr),
			}
		}
		createdOrders = append(createdOrders, Order{
			OrderID:       created.OrderID,
			SupplierID:    g.SupplierID,
			RetailerID:    retailerID,
			ParentOrderID: parentID,
			Status:        created.Status,
			TotalMinor:    created.TotalMinor,
			Currency:      created.Currency,
		})
		supplierOrders = append(supplierOrders, SupplierOrderResult{
			OrderID:      created.OrderID,
			SupplierID:   g.SupplierID,
			SupplierName: supplierName,
			Total:        created.TotalMinor,
			Currency:     created.Currency,
			ItemCount:    g.ItemCount,
		})
		totalMinor += created.TotalMinor
		if created.Currency != "" {
			currency = created.Currency
		}
		backorderedItemCount += created.BackorderedItemCount
		stockWarnings = append(stockWarnings, created.StockWarnings...)
		if backorderOrderID == "" {
			backorderOrderID = created.BackorderOrderID
		}
	}

	if err := s.updateParentOrderTotals(ctx, parentID, parentStatusPending, currency, totalMinor, len(supplierOrders)); err != nil {
		s.log.Warn("parent order confirm update failed", "parent_order_id", parentID, "err", err)
	}

	invoiceID := strings.Replace(s.newID(), "ord_", "inv_", 1)
	return UnifiedCheckoutResponse{
		Status:               "ok",
		InvoiceID:            invoiceID,
		Total:                totalMinor,
		Currency:             currency,
		MarketCode:           pack.Code,
		ParentOrderID:        parentID,
		SupplierOrders:       supplierOrders,
		BackorderedItemCount: backorderedItemCount,
		StockWarnings:        stockWarnings,
		BackorderOrderID:     backorderOrderID,
	}, nil
}

func (s *Service) resolveRetailerCoordinates(ctx context.Context, retailerID string, lat, lng float64) (float64, float64, error) {
	store, err := s.resolveCheckoutStore(ctx, retailerID, lat, lng)
	if err != nil {
		return 0, 0, err
	}
	return store.Lat, store.Lng, nil
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
	// Client unit_price is a hint only. Enterprise path loads catalog PriceMinor
	// (and optional promos/overrides) so checkout cannot underpay.
	draft := make([]LineItem, 0, len(items))
	for i, item := range items {
		sku := strings.TrimSpace(item.SkuID)
		if sku == "" || item.Quantity <= 0 {
			return nil, fmt.Errorf("items[%d] requires sku_id and positive quantity", i)
		}
		if item.UnitPrice < 0 {
			return nil, fmt.Errorf("items[%d].unit_price must be >= 0", i)
		}
		draft = append(draft, LineItem{
			SKU:       sku,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice, // overwritten by catalog when Spanner is available
		})
	}

	// Production / SSMR: Spanner catalog is the price authority (same as Create).
	if s.spannerClient != nil {
		normalized, _, err := s.normalizeAndQuoteLineItems(ctx, draft, nil)
		if err != nil {
			return nil, err
		}
		if s.promotions != nil {
			normalized, err = s.applyPromotionsToLines(ctx, retailerID, normalized)
			if err != nil {
				return nil, err
			}
		}
		return s.enrichLineItemVolumes(ctx, normalized)
	}

	// No Spanner: promotion service can still rewrite list prices when its repo is wired.
	if s.promotions != nil {
		inputs := make([]promotion.LineInput, 0, len(draft))
		for _, line := range draft {
			inputs = append(inputs, promotion.LineInput{
				ProductID: line.SKU,
				Quantity:  line.Quantity,
				UnitPrice: line.UnitPrice,
				Currency:  s.currency,
			})
		}
		quote, err := s.promotions.QuoteCheckout(ctx, s.resolveSupplierScope(ctx), retailerID, inputs)
		if err != nil {
			return nil, err
		}
		lineItems := make([]LineItem, 0, len(quote.Lines))
		for _, line := range quote.Lines {
			if line.UnitPrice < 0 {
				return nil, fmt.Errorf("catalog price missing for sku %s", line.ProductID)
			}
			lineItems = append(lineItems, LineItem{
				SKU:         line.ProductID,
				Name:        line.ProductID,
				Quantity:    line.Quantity,
				UnitPrice:   line.UnitPrice,
				PromotionID: line.PromotionID,
			})
		}
		return s.enrichLineItemVolumes(ctx, lineItems)
	}

	// Scaffold / unit-test path only (no Spanner, no promotions). Client prices
	// remain so offline tests can exercise order create without a catalog.
	return s.enrichLineItemVolumes(ctx, draft)
}

// applyPromotionsToLines re-quotes already catalog-priced lines so discounts
// never apply on top of client-supplied unit prices.
func (s *Service) applyPromotionsToLines(ctx context.Context, retailerID string, lines []LineItem) ([]LineItem, error) {
	if s.promotions == nil || len(lines) == 0 {
		return lines, nil
	}
	inputs := make([]promotion.LineInput, 0, len(lines))
	for _, line := range lines {
		inputs = append(inputs, promotion.LineInput{
			ProductID: line.SKU,
			Quantity:  line.Quantity,
			UnitPrice: line.UnitPrice,
			Currency:  s.currency,
		})
	}
	quote, err := s.promotions.QuoteCheckout(ctx, s.resolveSupplierScope(ctx), retailerID, inputs)
	if err != nil {
		return nil, err
	}
	bySKU := make(map[string]LineItem, len(lines))
	for _, line := range lines {
		bySKU[line.SKU] = line
	}
	out := make([]LineItem, 0, len(quote.Lines))
	for _, q := range quote.Lines {
		base := bySKU[q.ProductID]
		base.SKU = q.ProductID
		base.Quantity = q.Quantity
		base.UnitPrice = q.UnitPrice
		base.PromotionID = q.PromotionID
		if base.Name == "" {
			base.Name = q.ProductID
		}
		out = append(out, base)
	}
	return out, nil
}
