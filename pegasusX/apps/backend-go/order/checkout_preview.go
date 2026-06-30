package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"google.golang.org/api/iterator"
)

// CheckoutPreviewResponse is POST /v1/checkout/preview (dry-run inventory planning).
type CheckoutPreviewResponse struct {
	OK                         bool              `json:"ok"`
	Blocked                    bool              `json:"blocked,omitempty"`
	Code                       string            `json:"code,omitempty"`
	Message                    string            `json:"message,omitempty"`
	RejectedSKUs               []string          `json:"rejected_skus,omitempty"`
	OOSItems                   []string          `json:"oos_items,omitempty"`
	Shortfall                  map[string]int64  `json:"shortfall,omitempty"`
	StockWarnings              []StockWarning    `json:"stock_warnings,omitempty"`
	MaxQuantities              map[string]int64  `json:"max_quantities,omitempty"`
	OrderableQuantities        map[string]int64  `json:"orderable_quantities,omitempty"`
	LineErrors                 map[string]string `json:"line_errors,omitempty"`
	BackorderItemCount         int               `json:"backordered_item_count,omitempty"`
	ShowStockCounts            bool              `json:"show_stock_counts,omitempty"`
	DefaultOutOfStockPolicy    string            `json:"default_out_of_stock_policy,omitempty"`
	CheckoutPolicyToken        string            `json:"checkout_policy_token,omitempty"`
	CheckoutPolicyExpiresAt    string            `json:"checkout_policy_expires_at,omitempty"`
	OrderAcceptanceOpen        bool              `json:"order_acceptance_open,omitempty"`
	OrderAcceptanceWindowLabel string            `json:"order_acceptance_window_label,omitempty"`
	NextOrderAcceptanceAt      *string           `json:"next_order_acceptance_at,omitempty"`
	PreorderMinLeadDays        int64             `json:"preorder_min_lead_days,omitempty"`
	PreorderMaxLeadDays        int64             `json:"preorder_max_lead_days,omitempty"`
	OrderLineMinQuantity       *int64            `json:"order_line_min_quantity,omitempty"`
	OrderLineMaxQuantity       *int64            `json:"order_line_max_quantity,omitempty"`
	DeliveryFeeMinor           int64             `json:"delivery_fee_minor,omitempty"`
	DeliveryDistanceKm         float64           `json:"delivery_distance_km,omitempty"`
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

	// Rate limit checkout preview by RetailerID to prevent scraping.
	if !s.previewRateLimiter.Allow(claims.Subject) {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate_limit_exceeded"})
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

func basePreviewResponse(resp CheckoutPreviewResponse) CheckoutPreviewResponse {
	if resp.MaxQuantities == nil {
		resp.MaxQuantities = map[string]int64{}
	}
	if resp.OrderableQuantities == nil {
		resp.OrderableQuantities = map[string]int64{}
	}
	return resp
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

	resp := basePreviewResponse(CheckoutPreviewResponse{OK: true})
	whPolicy, policyErr := LoadWarehouseOpsPolicy(ctx, s.spannerClient, warehouseID)
	if policyErr == nil {
		resp.ShowStockCounts = whPolicy.ShowStockCountsToRetailers
		resp.DefaultOutOfStockPolicy = whPolicy.DefaultOutOfStockPolicy
		resp.PreorderMinLeadDays = whPolicy.PreorderMinLeadDays
		resp.PreorderMaxLeadDays = whPolicy.PreorderMaxLeadDays
		resp.OrderLineMinQuantity = whPolicy.OrderLineMinQuantity
		resp.OrderLineMaxQuantity = whPolicy.OrderLineMaxQuantity
		feeMinor, distKm := ComputeOrderDeliveryFee(whPolicy, lat, lng)
		resp.DeliveryFeeMinor = feeMinor
		resp.DeliveryDistanceKm = distKm

		open, label, nextOpen, closedMsg := checkOrderAcceptanceGate(whPolicy, s.now())
		applyAcceptancePreviewFields(&resp, open, label, nextOpen)
		if !open {
			return basePreviewResponse(CheckoutPreviewResponse{
				OK:                         false,
				Blocked:                    true,
				Code:                       ErrOrderAcceptanceClosed.Error(),
				Message:                    closedMsg,
				ShowStockCounts:              resp.ShowStockCounts,
				DefaultOutOfStockPolicy:      resp.DefaultOutOfStockPolicy,
				OrderAcceptanceOpen:          false,
				OrderAcceptanceWindowLabel: label,
				NextOrderAcceptanceAt:      resp.NextOrderAcceptanceAt,
				PreorderMinLeadDays:        resp.PreorderMinLeadDays,
				PreorderMaxLeadDays:        resp.PreorderMaxLeadDays,
				OrderLineMinQuantity:       resp.OrderLineMinQuantity,
				OrderLineMaxQuantity:       resp.OrderLineMaxQuantity,
				DeliveryFeeMinor:           resp.DeliveryFeeMinor,
				DeliveryDistanceKm:         resp.DeliveryDistanceKm,
			}), nil
		}

		lineErrs := ValidateLineQuantities(lineItems, whPolicy)
		if packErrs := validatePackMultiples(ctx, s.spannerClient, lineItems); len(packErrs) > 0 {
			lineErrs = mergeLineErrors(lineErrs, packErrs)
		}
		if len(lineErrs) > 0 {
			return basePreviewResponse(CheckoutPreviewResponse{
				OK:                    false,
				Blocked:               true,
				Code:                  ErrLineQuantityOutOfRange.Error(),
				Message:               "line quantity policy violated",
				LineErrors:            lineErrs,
				ShowStockCounts:       resp.ShowStockCounts,
				DefaultOutOfStockPolicy: resp.DefaultOutOfStockPolicy,
				OrderAcceptanceOpen:   true,
				PreorderMinLeadDays:   resp.PreorderMinLeadDays,
				PreorderMaxLeadDays:   resp.PreorderMaxLeadDays,
				OrderLineMinQuantity:  resp.OrderLineMinQuantity,
				OrderLineMaxQuantity:  resp.OrderLineMaxQuantity,
				DeliveryFeeMinor:      resp.DeliveryFeeMinor,
				DeliveryDistanceKm:    resp.DeliveryDistanceKm,
			}), nil
		}

		if token, exp, err := IssueCheckoutPolicyToken(s.jwtSecret, snapshotFromWarehousePolicy(whPolicy, whPolicy.OperatingSchedule), s.now()); err == nil {
			resp.CheckoutPolicyToken = token
			resp.CheckoutPolicyExpiresAt = exp.Format(time.RFC3339)
		}
	} else if s.spannerClient != nil {
		resp.ShowStockCounts = loadWarehouseShowStockCounts(ctx, s.spannerClient, warehouseID)
	}

	if s.spannerClient != nil {
		warehouseDefault := whPolicy.DefaultOutOfStockPolicy
		if warehouseDefault == "" {
			warehouseDefault = outOfStockPolicyReject
		}
		maxQty, perSKUPolicy, err := loadAvailableQuantitiesAndPolicies(ctx, s.spannerClient, s.supplierID, warehouseID, lineItems, warehouseDefault)
		if err == nil {
			resp.MaxQuantities = maxQty
			if policyErr == nil {
				orderable, stockLineErrs := ComputeOrderableQuantitiesForPolicy(maxQty, perSKUPolicy, whPolicy)
				resp.OrderableQuantities = orderable
				if len(stockLineErrs) > 0 {
					return basePreviewResponse(CheckoutPreviewResponse{
						OK:                      false,
						Blocked:                 true,
						Code:                    ErrLineQuantityOutOfRange.Error(),
						Message:                 "insufficient stock for line minimum",
						LineErrors:              stockLineErrs,
						MaxQuantities:           resp.MaxQuantities,
						OrderableQuantities:     resp.OrderableQuantities,
						ShowStockCounts:         resp.ShowStockCounts,
						DefaultOutOfStockPolicy: resp.DefaultOutOfStockPolicy,
						CheckoutPolicyToken:     resp.CheckoutPolicyToken,
						CheckoutPolicyExpiresAt: resp.CheckoutPolicyExpiresAt,
						OrderAcceptanceOpen:     resp.OrderAcceptanceOpen,
						PreorderMinLeadDays:     resp.PreorderMinLeadDays,
						PreorderMaxLeadDays:     resp.PreorderMaxLeadDays,
						OrderLineMinQuantity:    resp.OrderLineMinQuantity,
						OrderLineMaxQuantity:    resp.OrderLineMaxQuantity,
						DeliveryFeeMinor:        resp.DeliveryFeeMinor,
						DeliveryDistanceKm:      resp.DeliveryDistanceKm,
					}), nil
				}
			} else {
				resp.OrderableQuantities = maxQty
			}
		}
		policyOverride, _ := s.resolveCheckoutPolicyOverride(whPolicy, req.CheckoutPolicyToken)
		invPlan, err := PlanInventoryCheckout(ctx, s.spannerClient, s.supplierID, warehouseID, lineItems, policyOverride)
		if err != nil {
			var ice *InventoryCheckoutError
			if errors.As(err, &ice) {
				return basePreviewResponse(CheckoutPreviewResponse{
					OK:                      false,
					Blocked:                 true,
					Code:                    ice.Code,
					Message:                 ice.Message,
					RejectedSKUs:            ice.RejectedSKUs,
					OOSItems:                ice.OOSItems,
					Shortfall:               ice.Shortfall,
					MaxQuantities:           resp.MaxQuantities,
					OrderableQuantities:     resp.OrderableQuantities,
					ShowStockCounts:         resp.ShowStockCounts,
					DefaultOutOfStockPolicy: resp.DefaultOutOfStockPolicy,
					CheckoutPolicyToken:     resp.CheckoutPolicyToken,
					CheckoutPolicyExpiresAt: resp.CheckoutPolicyExpiresAt,
					OrderAcceptanceOpen:     resp.OrderAcceptanceOpen,
					PreorderMinLeadDays:     resp.PreorderMinLeadDays,
					PreorderMaxLeadDays:     resp.PreorderMaxLeadDays,
					OrderLineMinQuantity:    resp.OrderLineMinQuantity,
					OrderLineMaxQuantity:    resp.OrderLineMaxQuantity,
					DeliveryFeeMinor:        resp.DeliveryFeeMinor,
					DeliveryDistanceKm:      resp.DeliveryDistanceKm,
				}), nil
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
	avail, _, err := loadAvailableQuantitiesAndPolicies(ctx, client, supplierID, warehouseID, items, outOfStockPolicyReject)
	return avail, err
}

func loadAvailableQuantitiesAndPolicies(
	ctx context.Context,
	client *spanner.Client,
	supplierID, warehouseID string,
	items []LineItem,
	warehouseDefault string,
) (map[string]int64, map[string]string, error) {
	out := make(map[string]int64, len(items))
	policies := make(map[string]string, len(items))
	skus := make(map[string]struct{}, len(items))
	for _, item := range items {
		sku := strings.TrimSpace(item.SKU)
		if sku == "" {
			continue
		}
		skus[sku] = struct{}{}
	}
	if len(skus) == 0 {
		return out, policies, nil
	}
	keys := make([]spanner.KeySet, 0, len(skus))
	for sku := range skus {
		keys = append(keys, spanner.Key{supplierID, warehouseID, sku})
	}
	iter := client.Single().Read(ctx, "SupplierInventoryV2", spanner.KeySets(keys...),
		[]string{"ProductId", "QuantityOnHand", "QuantityReserved", "OutOfStockPolicy"})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		var productID string
		var qoh, qr int64
		var policy spanner.NullString
		if err := row.Columns(&productID, &qoh, &qr, &policy); err != nil {
			return nil, nil, err
		}
		avail := qoh - qr
		if avail < 0 {
			avail = 0
		}
		out[productID] = avail
		policies[productID] = resolveOutOfStockPolicy(warehouseDefault, policy.StringVal)
	}
	for sku := range skus {
		if _, ok := out[sku]; !ok {
			out[sku] = 0
			policies[sku] = resolveOutOfStockPolicy(warehouseDefault, "")
		}
	}
	return out, policies, nil
}
