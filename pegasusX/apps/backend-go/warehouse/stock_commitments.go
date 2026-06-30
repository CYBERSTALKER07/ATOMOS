package warehouse

import (
	"context"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
)

// StockCommitmentOrder is a single order consuming SKU capacity.
type StockCommitmentOrder struct {
	OrderID               string `json:"order_id"`
	RetailerID            string `json:"retailer_id"`
	Status                string `json:"status"`
	OrderSource           string `json:"order_source"`
	RequestedDeliveryDate string `json:"requested_delivery_date,omitempty"`
	DeliverBefore         string `json:"deliver_before,omitempty"`
	DeliveryPriority      string `json:"delivery_priority,omitempty"`
	Quantity              int64  `json:"quantity"`
	PreorderBadge         string `json:"preorder_badge,omitempty"`
}

// StockCommitmentSKU aggregates reserved demand for one SKU.
type StockCommitmentSKU struct {
	SKUID             string                 `json:"sku_id"`
	Name              string                 `json:"name"`
	ImageURL          string                 `json:"image_url,omitempty"`
	OnHand            int64                  `json:"on_hand"`
	AvailableQty      int64                  `json:"available_qty"`
	ReservedASAP      int64                  `json:"reserved_asap"`
	ReservedScheduled int64                  `json:"reserved_scheduled"`
	DeficitQty        int64                  `json:"deficit_qty"`
	Orders            []StockCommitmentOrder `json:"orders,omitempty"`
}

// OrderStockReader lists orders for stock projection.
type OrderStockReader interface {
	ListOrdersForStockCommitment(ctx context.Context, warehouseID string, limit int) ([]order.Order, error)
}

// SetOrderStockReader wires order stock aggregation for commitment APIs.
func (s *Service) SetOrderStockReader(r OrderStockReader) {
	s.orderStock = r
}

// HandleStockCommitments serves GET /v1/warehouse/ops/stock-commitments.
func (s *Service) HandleStockCommitments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	whID := warehouseIDFromRequest(r)
	if whID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_scope_required"})
		return
	}
	skuFilter := strings.TrimSpace(r.URL.Query().Get("sku_id"))
	if skuFilter != "" {
		s.handleStockCommitmentDetail(w, r, whID, skuFilter)
		return
	}
	items, err := s.buildStockCommitments(r.Context(), whID, "")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "stock_commitments_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "skus": items})
}

// HandleStockCommitmentBySKU serves GET /v1/warehouse/ops/stock-commitments/{skuId}.
func (s *Service) HandleStockCommitmentBySKU(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	whID := warehouseIDFromRequest(r)
	skuID := strings.TrimSpace(chi.URLParam(r, "skuId"))
	if whID == "" || skuID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "warehouse_and_sku_required"})
		return
	}
	s.handleStockCommitmentDetail(w, r, whID, skuID)
}

func (s *Service) handleStockCommitmentDetail(w http.ResponseWriter, r *http.Request, whID, skuID string) {
	items, err := s.buildStockCommitments(r.Context(), whID, skuID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "stock_commitments_failed"})
		return
	}
	if len(items) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "sku_not_found"})
		return
	}
	writeJSON(w, http.StatusOK, items[0])
}

func (s *Service) buildStockCommitments(ctx context.Context, warehouseID, skuFilter string) ([]StockCommitmentSKU, error) {
	if s.spannerClient != nil {
		txn := s.spannerClient.ReadOnlyTransaction()
		defer txn.Close()
		ctx = spannerutils.WithReadOnlyTransaction(ctx, txn)
	}
	onHand, err := s.inventoryLevelsByWarehouse(ctx, warehouseID)
	if err != nil {
		return nil, err
	}
	var orders []order.Order
	if s.orderStock != nil {
		orders, err = s.orderStock.ListOrdersForStockCommitment(ctx, warehouseID, 500)
		if err != nil {
			return nil, err
		}
	}
	type agg struct {
		asap      int64
		scheduled int64
		orders    []StockCommitmentOrder
	}
	bySKU := map[string]*agg{}
	for _, o := range orders {
		for _, li := range o.LineItems {
			sku := strings.TrimSpace(li.SKU)
			if sku == "" {
				continue
			}
			if skuFilter != "" && sku != skuFilter {
				continue
			}
			a := bySKU[sku]
			if a == nil {
				a = &agg{}
				bySKU[sku] = a
			}
			qty := li.Quantity
			if order.IsScheduledPreorder(o) {
				a.scheduled += qty
			} else {
				a.asap += qty
			}
			a.orders = append(a.orders, StockCommitmentOrder{
				OrderID:               o.OrderID,
				RetailerID:            o.RetailerID,
				Status:                string(o.Status),
				OrderSource:           string(o.Source),
				RequestedDeliveryDate: formatTimePtr(o.RequestedDeliveryDate),
				DeliverBefore:         formatTimePtr(o.DeliverBefore),
				DeliveryPriority:      string(o.DeliveryPriority),
				Quantity:              qty,
				PreorderBadge:         preorderBadge(o),
			})
		}
	}
	if skuFilter != "" {
		if _, ok := bySKU[skuFilter]; !ok {
			bySKU[skuFilter] = &agg{}
		}
	}
	meta := s.productMetaByIDs(ctx, mapKeys(bySKU))
	out := make([]StockCommitmentSKU, 0, len(bySKU))
	for sku, a := range bySKU {
		qoh := onHand[sku]
		reserved := a.asap + a.scheduled
		available := qoh - reserved
		if available < 0 {
			available = 0
		}
		deficit := int64(0)
		if reserved > qoh {
			deficit = reserved - qoh
		}
		row := StockCommitmentSKU{
			SKUID:             sku,
			OnHand:            qoh,
			AvailableQty:      available,
			ReservedASAP:      a.asap,
			ReservedScheduled: a.scheduled,
			DeficitQty:        deficit,
			Orders:            a.orders,
		}
		if m, ok := meta[sku]; ok {
			row.Name = m.name
			row.ImageURL = m.imageURL
		}
		out = append(out, row)
	}
	return out, nil
}

type productMeta struct {
	name     string
	imageURL string
}

func mapKeys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func (s *Service) productMetaByIDs(ctx context.Context, productIDs []string) map[string]productMeta {
	out := map[string]productMeta{}
	if s.spannerClient == nil || len(productIDs) == 0 || s.supplierID == "" {
		return out
	}
	stmt := spanner.Statement{
		SQL: `SELECT ProductId, Name, COALESCE(ImageURL, '') FROM Products
		      WHERE SupplierId = @sid AND ProductId IN UNNEST(@ids)`,
		Params: map[string]any{"sid": s.supplierID, "ids": productIDs},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return out
		}
		var id, name, image string
		if err := row.Columns(&id, &name, &image); err != nil {
			continue
		}
		out[id] = productMeta{name: name, imageURL: image}
	}
	return out
}

func preorderBadge(o order.Order) string {
	if order.IsScheduledPreorder(o) {
		return "Pre-order"
	}
	if o.DeliverBefore != nil {
		return "Deliver by"
	}
	if o.DeliveryPriority == order.DeliveryPriorityExpress {
		return "Express"
	}
	if o.Status == order.StatusAutoAccepted {
		return "Confirmed for dispatch"
	}
	return ""
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
