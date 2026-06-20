package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// CheckoutPreviewResponse is POST /v1/checkout/preview (dry-run inventory planning).
type CheckoutPreviewResponse struct {
	OK                    bool             `json:"ok"`
	Blocked               bool             `json:"blocked,omitempty"`
	Code                  string           `json:"code,omitempty"`
	Message               string           `json:"message,omitempty"`
	RejectedSKUs          []string         `json:"rejected_skus,omitempty"`
	OOSItems              []string         `json:"oos_items,omitempty"`
	Shortfall             map[string]int64 `json:"shortfall,omitempty"`
	StockWarnings         []StockWarning   `json:"stock_warnings,omitempty"`
	MaxQuantities         map[string]int64 `json:"max_quantities,omitempty"`
	OrderableQuantities   map[string]int64 `json:"orderable_quantities,omitempty"`
	LineErrors            map[string]string `json:"line_errors,omitempty"`
	BackorderItemCount    int              `json:"backordered_item_count,omitempty"`
	ShowStockCounts       bool             `json:"show_stock_counts,omitempty"`
	PreorderMinLeadDays   int64            `json:"preorder_min_lead_days,omitempty"`
	PreorderMaxLeadDays   int64            `json:"preorder_max_lead_days,omitempty"`
	OrderLineMinQuantity  *int64           `json:"order_line_min_quantity,omitempty"`
	OrderLineMaxQuantity  *int64           `json:"order_line_max_quantity,omitempty"`
	DeliveryFeeMinor      int64            `json:"delivery_fee_minor,omitempty"`
	DeliveryDistanceKm    float64          `json:"delivery_distance_km,omitempty"`
}

// HandleCheckoutPreview serves POST /v1/checkout/preview.
func (s *Service) HandleCheckoutPreview(w http.ResponseWriter, r *http.Request) {
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

	var req UnifiedCheckoutRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if req.RetailerID != "" && req.RetailerID != claims.Subject {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "retailer_id_mismatch"})
		return
	}

	resp, err := s.PreviewCheckout(r.Context(), claims.Subject, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrZoneMiss):
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": ErrZoneMiss.Error()})
		case errors.Is(err, ErrServiceabilityUnavailable):
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": ErrServiceabilityUnavailable.Error()})
		default:
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		}
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// PreviewCheckout dry-runs inventory policy without creating an order.
func (s *Service) PreviewCheckout(ctx context.Context, retailerID string, req UnifiedCheckoutRequest) (CheckoutPreviewResponse, error) {
	if retailerID == "" {
		return CheckoutPreviewResponse{}, errors.New("retailer_id required from session")
	}
	if len(req.Items) == 0 {
		return CheckoutPreviewResponse{}, errors.New("items must not be empty")
	}

	lat, lng, err := s.resolveRetailerCoordinates(ctx, retailerID, req.Latitude, req.Longitude)
	if err != nil {
		return CheckoutPreviewResponse{}, err
	}
	if _, err := h3CellFromLatLng(lat, lng); err != nil {
		return CheckoutPreviewResponse{}, err
	}

	lineItems, err := s.authoritativeCheckoutLines(ctx, retailerID, req.Items)
	if err != nil {
		return CheckoutPreviewResponse{}, err
	}

	if s.warehouse == nil {
		return CheckoutPreviewResponse{}, fmt.Errorf("%w: warehouse_resolver_unavailable", ErrServiceabilityUnavailable)
	}
	warehouseID, err := s.warehouse.ResolveNearestWarehouseID(ctx, s.supplierID, lat, lng)
	if err != nil {
		return CheckoutPreviewResponse{}, fmt.Errorf("%w: resolve nearest warehouse: %v", ErrServiceabilityUnavailable, err)
	}
	warehouseID = strings.TrimSpace(warehouseID)
	if warehouseID == "" {
		return CheckoutPreviewResponse{}, fmt.Errorf("%w: no_eligible_warehouse", ErrZoneMiss)
	}

	resp := CheckoutPreviewResponse{OK: true, MaxQuantities: map[string]int64{}, OrderableQuantities: map[string]int64{}}
	whPolicy, policyErr := LoadWarehouseOpsPolicy(ctx, s.spannerClient, warehouseID)
	if policyErr == nil {
		resp.ShowStockCounts = whPolicy.ShowStockCountsToRetailers
		resp.PreorderMinLeadDays = whPolicy.PreorderMinLeadDays
		resp.PreorderMaxLeadDays = whPolicy.PreorderMaxLeadDays
		resp.OrderLineMinQuantity = whPolicy.OrderLineMinQuantity
		resp.OrderLineMaxQuantity = whPolicy.OrderLineMaxQuantity
		feeMinor, distKm := ComputeOrderDeliveryFee(whPolicy, lat, lng)
		resp.DeliveryFeeMinor = feeMinor
		resp.DeliveryDistanceKm = distKm
		lineErrs := ValidateLineQuantities(lineItems, whPolicy)
		if packErrs := validatePackMultiples(ctx, s.spannerClient, lineItems); len(packErrs) > 0 {
			lineErrs = mergeLineErrors(lineErrs, packErrs)
		}
		if len(lineErrs) > 0 {
			return CheckoutPreviewResponse{
				OK:                    false,
				Blocked:               true,
				Code:                  ErrLineQuantityOutOfRange.Error(),
				Message:               "line quantity policy violated",
				LineErrors:            lineErrs,
				MaxQuantities:         resp.MaxQuantities,
				OrderableQuantities:   resp.OrderableQuantities,
				ShowStockCounts:       resp.ShowStockCounts,
				PreorderMinLeadDays:   resp.PreorderMinLeadDays,
				PreorderMaxLeadDays:   resp.PreorderMaxLeadDays,
				OrderLineMinQuantity:  resp.OrderLineMinQuantity,
				OrderLineMaxQuantity:  resp.OrderLineMaxQuantity,
				DeliveryFeeMinor:      resp.DeliveryFeeMinor,
				DeliveryDistanceKm:    resp.DeliveryDistanceKm,
			}, nil
		}
	} else if s.spannerClient != nil {
		resp.ShowStockCounts = loadWarehouseShowStockCounts(ctx, s.spannerClient, warehouseID)
	}
	if s.spannerClient != nil {
		if maxQty, err := loadAvailableQuantities(ctx, s.spannerClient, s.supplierID, warehouseID, lineItems); err == nil {
			resp.MaxQuantities = maxQty
			if policyErr == nil {
				orderable, stockLineErrs := ComputeOrderableQuantities(maxQty, whPolicy)
				resp.OrderableQuantities = orderable
				if len(stockLineErrs) > 0 {
					return CheckoutPreviewResponse{
						OK:                    false,
						Blocked:               true,
						Code:                  ErrLineQuantityOutOfRange.Error(),
						Message:               "insufficient stock for line minimum",
						LineErrors:            stockLineErrs,
						MaxQuantities:         resp.MaxQuantities,
						OrderableQuantities:   resp.OrderableQuantities,
						ShowStockCounts:       resp.ShowStockCounts,
						PreorderMinLeadDays:   resp.PreorderMinLeadDays,
						PreorderMaxLeadDays:   resp.PreorderMaxLeadDays,
						OrderLineMinQuantity:  resp.OrderLineMinQuantity,
						OrderLineMaxQuantity:  resp.OrderLineMaxQuantity,
						DeliveryFeeMinor:      resp.DeliveryFeeMinor,
						DeliveryDistanceKm:    resp.DeliveryDistanceKm,
					}, nil
				}
			} else {
				resp.OrderableQuantities = maxQty
			}
		}
		invPlan, err := PlanInventoryCheckout(ctx, s.spannerClient, s.supplierID, warehouseID, lineItems)
		if err != nil {
			var ice *InventoryCheckoutError
			if errors.As(err, &ice) {
				return CheckoutPreviewResponse{
					OK:                    false,
					Blocked:               true,
					Code:                  ice.Code,
					Message:               ice.Message,
					RejectedSKUs:          ice.RejectedSKUs,
					OOSItems:              ice.OOSItems,
					Shortfall:             ice.Shortfall,
					MaxQuantities:         resp.MaxQuantities,
					OrderableQuantities:   resp.OrderableQuantities,
					ShowStockCounts:       resp.ShowStockCounts,
					PreorderMinLeadDays:   resp.PreorderMinLeadDays,
					PreorderMaxLeadDays:   resp.PreorderMaxLeadDays,
					OrderLineMinQuantity:  resp.OrderLineMinQuantity,
					OrderLineMaxQuantity:  resp.OrderLineMaxQuantity,
					DeliveryFeeMinor:      resp.DeliveryFeeMinor,
					DeliveryDistanceKm:    resp.DeliveryDistanceKm,
				}, nil
			}
			return CheckoutPreviewResponse{}, err
		}
		resp.StockWarnings = invPlan.Warnings
		resp.BackorderItemCount = invPlan.BackorderCount
	}
	return resp, nil
}

func loadWarehouseShowStockCounts(ctx context.Context, client *spanner.Client, warehouseID string) bool {
	row, err := client.Single().ReadRow(ctx, "Warehouses", spanner.Key{warehouseID},
		[]string{"ShowStockCountsToRetailers"})
	if err != nil {
		return false
	}
	var show bool
	if err := row.Columns(&show); err != nil {
		return false
	}
	return show
}

func loadAvailableQuantities(ctx context.Context, client *spanner.Client, supplierID, warehouseID string, items []LineItem) (map[string]int64, error) {
	out := make(map[string]int64, len(items))
	for _, item := range items {
		sku := strings.TrimSpace(item.SKU)
		if sku == "" {
			continue
		}
		row, err := client.Single().ReadRow(ctx, "SupplierInventoryV2",
			spanner.Key{supplierID, warehouseID, sku},
			[]string{"QuantityOnHand", "QuantityReserved"})
		if err != nil {
			out[sku] = 0
			continue
		}
		var qoh, qr int64
		if err := row.Columns(&qoh, &qr); err != nil {
			out[sku] = 0
			continue
		}
		avail := qoh - qr
		if avail < 0 {
			avail = 0
		}
		out[sku] = avail
	}
	return out, nil
}
