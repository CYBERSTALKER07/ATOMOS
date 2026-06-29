package supplier

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

type retailerOverridePreviewRequest struct {
	RetailerID    string `json:"retailer_id"`
	ProductID     string `json:"product_id"`
	SkuID         string `json:"sku_id"`
	ProposedPrice int64  `json:"proposed_price"`
}

// RetailerOverridePreview is a read-only impact summary before saving an override.
type RetailerOverridePreview struct {
	RetailersOnSKUCount  int      `json:"retailers_on_sku_count"`
	ActiveOverrideCount  int      `json:"active_override_count"`
	CatalogListPrice     int64    `json:"catalog_list_price"`
	MarginDeltaPerUnit   int64    `json:"margin_delta_per_unit"`
	MarginEstimateLabel  string   `json:"margin_estimate_label"`
	AffectedRetailerIDs  []string `json:"affected_retailer_ids,omitempty"`
}

// HandleRetailerPricingOverridePreview serves POST /v1/supplier/pricing/retailer-overrides/preview.
func (s *Service) HandleRetailerPricingOverridePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if s.portalSpanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "pricing_unavailable"})
		return
	}

	body, ok := readMutationBody(w, r, 16*1024)
	if !ok {
		return
	}
	var req retailerOverridePreviewRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	productID := strings.TrimSpace(req.ProductID)
	if productID == "" {
		productID = strings.TrimSpace(req.SkuID)
	}
	if productID == "" || req.ProposedPrice <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "product_id_and_positive_price_required"})
		return
	}

	sid := s.scopedSupplierID(r)
	preview, err := s.buildRetailerOverridePreview(r.Context(), sid, strings.TrimSpace(req.RetailerID), productID, req.ProposedPrice)
	if err != nil {
		if err == errProductNotFound {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "product_not_found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "preview_failed"})
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func (s *Service) buildRetailerOverridePreview(ctx context.Context, supplierID, retailerID, productID string, proposedPrice int64) (RetailerOverridePreview, error) {
	if err := s.ensureSupplierProduct(ctx, supplierID, productID); err != nil {
		return RetailerOverridePreview{}, err
	}

	listPrice, err := s.productListPrice(ctx, productID)
	if err != nil {
		return RetailerOverridePreview{}, err
	}

	overrideRetailers, overrideCount, err := s.overrideRetailersForProduct(ctx, supplierID, productID)
	if err != nil {
		return RetailerOverridePreview{}, err
	}

	orderRetailers, err := s.recentOrderRetailersForProduct(ctx, supplierID, productID, 30)
	if err != nil {
		return RetailerOverridePreview{}, err
	}

	retailerSet := map[string]struct{}{}
	for id := range overrideRetailers {
		retailerSet[id] = struct{}{}
	}
	for id := range orderRetailers {
		retailerSet[id] = struct{}{}
	}
	if retailerID != "" {
		retailerSet[retailerID] = struct{}{}
	}

	affected := make([]string, 0, len(retailerSet))
	for id := range retailerSet {
		affected = append(affected, id)
	}
	if len(affected) > 20 {
		affected = affected[:20]
	}

	delta := proposedPrice - listPrice
	return RetailerOverridePreview{
		RetailersOnSKUCount: len(retailerSet),
		ActiveOverrideCount: overrideCount,
		CatalogListPrice:    listPrice,
		MarginDeltaPerUnit:  delta,
		MarginEstimateLabel: "estimated vs catalog list price",
		AffectedRetailerIDs: affected,
	}, nil
}

func (s *Service) productListPrice(ctx context.Context, productID string) (int64, error) {
	row, err := s.portalSpanner.Single().ReadRow(ctx, "Products", spanner.Key{productID}, []string{"PriceMinor"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return 0, errProductNotFound
		}
		return 0, err
	}
	var price int64
	if err := row.Columns(&price); err != nil {
		return 0, err
	}
	return price, nil
}

func (s *Service) overrideRetailersForProduct(ctx context.Context, supplierID, productID string) (map[string]struct{}, int, error) {
	stmt := spanner.Statement{
		SQL: `SELECT RetailerId FROM RetailerPricingOverrides
		      WHERE SupplierId = @sid AND ProductId = @pid AND IsActive = true`,
		Params: map[string]any{"sid": supplierID, "pid": productID},
	}
	iter := s.portalSpanner.Single().Query(ctx, stmt)
	defer iter.Stop()
	out := map[string]struct{}{}
	count := 0
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return out, count, nil
		}
		if err != nil {
			return nil, 0, err
		}
		var retailerID string
		if err := row.Columns(&retailerID); err != nil {
			continue
		}
		count++
		if id := strings.TrimSpace(retailerID); id != "" {
			out[id] = struct{}{}
		}
	}
}

func (s *Service) recentOrderRetailersForProduct(ctx context.Context, supplierID, productID string, days int) (map[string]struct{}, error) {
	since := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
	stmt := spanner.Statement{
		SQL: `SELECT RetailerId, LineItemsJson
		      FROM Orders@{FORCE_INDEX=Idx_Orders_BySupplierUpdated}
		      WHERE SupplierId = @sid AND UpdatedAt >= @since
		      LIMIT 300`,
		Params: map[string]any{"sid": supplierID, "since": since},
	}
	iter := s.portalSpanner.Single().Query(ctx, stmt)
	defer iter.Stop()
	out := map[string]struct{}{}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		var retailerID string
		var lineItems []byte
		if err := row.Columns(&retailerID, &lineItems); err != nil {
			continue
		}
		if !lineItemsContainProduct(lineItems, productID) {
			continue
		}
		if id := strings.TrimSpace(retailerID); id != "" {
			out[id] = struct{}{}
		}
	}
}

func lineItemsContainProduct(raw []byte, productID string) bool {
	if len(raw) == 0 {
		return false
	}
	var items []struct {
		ProductID string `json:"product_id"`
		SkuID     string `json:"sku_id"`
	}
	if err := json.Unmarshal(raw, &items); err != nil {
		return false
	}
	for _, item := range items {
		if item.ProductID == productID || item.SkuID == productID {
			return true
		}
	}
	return false
}
