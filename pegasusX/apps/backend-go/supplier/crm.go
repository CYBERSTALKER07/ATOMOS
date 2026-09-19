package supplier

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"google.golang.org/api/iterator"
)

type crmRetailer struct {
	RetailerID    string `json:"retailer_id"`
	RetailerName  string `json:"retailer_name"`
	Phone         string `json:"phone,omitempty"`
	Email         string `json:"email,omitempty"`
	Lifetime      int64  `json:"lifetime"`
	OrderCount    int64  `json:"order_count"`
	LastOrderDate string `json:"last_order_date,omitempty"`
	Status        string `json:"status"`
}

type crmOrderLine struct {
	SKU         string `json:"sku,omitempty"`
	ProductName string `json:"product_name,omitempty"`
	Qty         int64  `json:"qty"`
	AmountMinor int64  `json:"amount_minor,omitempty"`
}

type crmOrder struct {
	OrderID   string         `json:"order_id"`
	State     string         `json:"state"`
	Amount    int64          `json:"amount"`
	ItemCount int64          `json:"item_count"`
	CreatedAt string         `json:"created_at"`
	Lines     []crmOrderLine `json:"lines,omitempty"`
}

type crmRetailerDetail struct {
	crmRetailer
	Orders []crmOrder `json:"orders"`
}

const crmActiveWindow = 30 * 24 * time.Hour

func crmStatus(lastOrder, now time.Time) string {
	if lastOrder.IsZero() {
		return "INACTIVE"
	}
	if !lastOrder.Before(now.Add(-crmActiveWindow)) {
		return "ACTIVE"
	}
	return "INACTIVE"
}

func countCRMLineItems(raw []byte) int64 {
	if len(raw) == 0 {
		return 0
	}
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return 0
	}
	return int64(len(items))
}

func parseCRMLines(raw []byte) []crmOrderLine {
	if len(raw) == 0 {
		return nil
	}
	var items []struct {
		SKU            string `json:"sku"`
		ProductID      string `json:"product_id"`
		ProductName    string `json:"product_name"`
		Name           string `json:"name"`
		Qty            int64  `json:"qty"`
		Quantity       int64  `json:"quantity"`
		UnitPrice      int64  `json:"unit_price"`
		LineTotal      int64  `json:"line_total"`
		UnitPriceMinor int64  `json:"unit_price_minor"`
	}
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil
	}
	out := make([]crmOrderLine, 0, len(items))
	for _, it := range items {
		qty := it.Qty
		if qty == 0 {
			qty = it.Quantity
		}
		name := strings.TrimSpace(it.ProductName)
		if name == "" {
			name = strings.TrimSpace(it.Name)
		}
		sku := strings.TrimSpace(it.SKU)
		if sku == "" {
			sku = strings.TrimSpace(it.ProductID)
		}
		amount := it.LineTotal
		if amount == 0 && it.UnitPriceMinor != 0 {
			amount = it.UnitPriceMinor * qty
		}
		if amount == 0 && it.UnitPrice != 0 {
			amount = it.UnitPrice * qty
		}
		out = append(out, crmOrderLine{SKU: sku, ProductName: name, Qty: qty, AmountMinor: amount})
	}
	return out
}

func (s *Service) crmNow() time.Time {
	if s != nil && s.now != nil {
		return s.now()
	}
	return time.Now().UTC()
}

// HandleCRMRetailers serves GET /v1/supplier/crm/retailers (supplier-grain rollup).
func (s *Service) HandleCRMRetailers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if s == nil || s.portalSpanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "crm_unavailable"})
		return
	}
	supplierID := s.scopedSupplierID(r)
	if strings.TrimSpace(supplierID) == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	rows, err := s.listCRMRetailers(r.Context(), supplierID)
	if err != nil {
		if s.log != nil {
			s.log.WarnContext(r.Context(), "supplier crm list failed", "err", err, "supplier_id", supplierID)
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "query_failed"})
		return
	}
	if rows == nil {
		rows = []crmRetailer{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"retailers": rows})
}

// HandleCRMRetailerDetail serves GET /v1/supplier/crm/retailers/{retailerId}.
func (s *Service) HandleCRMRetailerDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	retailerID := strings.TrimSpace(chi.URLParam(r, "retailerId"))
	if retailerID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "retailer_id_required"})
		return
	}
	if s == nil || s.portalSpanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "crm_unavailable"})
		return
	}
	supplierID := s.scopedSupplierID(r)
	detail, err := s.loadCRMRetailerDetail(r.Context(), supplierID, retailerID)
	if err != nil {
		if s.log != nil {
			s.log.WarnContext(r.Context(), "supplier crm detail failed", "err", err, "retailer_id", retailerID)
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "query_failed"})
		return
	}
	if detail == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "retailer_not_found"})
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Service) listCRMRetailers(ctx context.Context, supplierID string) ([]crmRetailer, error) {
	sql := `SELECT o.RetailerId,
	               COALESCE(r.Name, o.RetailerId) AS retailer_name,
	               COALESCE(r.Phone, '') AS phone,
	               COALESCE(r.Email, '') AS email,
	               COALESCE(SUM(o.TotalMinor), 0) AS lifetime,
	               COUNT(DISTINCT o.OrderId) AS order_count,
	               MAX(o.CreatedAt) AS last_order
	        FROM Orders o
	        LEFT JOIN Retailers r ON o.RetailerId = r.RetailerId
	        WHERE o.SupplierId = @sid
	          AND o.Status != 'CANCELLED'
	        GROUP BY o.RetailerId, r.Name, r.Phone, r.Email
	        ORDER BY lifetime DESC
	        LIMIT 200`
	iter := s.portalSpanner.Single().Query(ctx, spanner.Statement{
		SQL:    sql,
		Params: map[string]any{"sid": supplierID},
	})
	defer iter.Stop()
	now := s.crmNow()
	out := make([]crmRetailer, 0, 16)
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		rec, err := scanCRMRetailer(row, now)
		if err != nil {
			continue
		}
		out = append(out, rec)
	}
	return out, nil
}

func (s *Service) loadCRMRetailerDetail(ctx context.Context, supplierID, retailerID string) (*crmRetailerDetail, error) {
	sql := `SELECT o.RetailerId,
	               COALESCE(r.Name, o.RetailerId) AS retailer_name,
	               COALESCE(r.Phone, '') AS phone,
	               COALESCE(r.Email, '') AS email,
	               COALESCE(SUM(o.TotalMinor), 0) AS lifetime,
	               COUNT(DISTINCT o.OrderId) AS order_count,
	               MAX(o.CreatedAt) AS last_order
	        FROM Orders o
	        LEFT JOIN Retailers r ON o.RetailerId = r.RetailerId
	        WHERE o.SupplierId = @sid
	          AND o.RetailerId = @rid
	          AND o.Status != 'CANCELLED'
	        GROUP BY o.RetailerId, r.Name, r.Phone, r.Email`
	iter := s.portalSpanner.Single().Query(ctx, spanner.Statement{
		SQL:    sql,
		Params: map[string]any{"sid": supplierID, "rid": retailerID},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	summary, err := scanCRMRetailer(row, s.crmNow())
	if err != nil {
		return nil, err
	}
	orders, err := s.listCRMOrders(ctx, supplierID, retailerID)
	if err != nil {
		return nil, err
	}
	if orders == nil {
		orders = []crmOrder{}
	}
	return &crmRetailerDetail{crmRetailer: summary, Orders: orders}, nil
}

func (s *Service) listCRMOrders(ctx context.Context, supplierID, retailerID string) ([]crmOrder, error) {
	iter := s.portalSpanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT OrderId, Status, TotalMinor, LineItemsJson, CreatedAt
		      FROM Orders
		      WHERE SupplierId = @sid AND RetailerId = @rid AND Status != 'CANCELLED'
		      ORDER BY CreatedAt DESC
		      LIMIT 50`,
		Params: map[string]any{"sid": supplierID, "rid": retailerID},
	})
	defer iter.Stop()
	out := make([]crmOrder, 0, 16)
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var rec crmOrder
		var raw []byte
		var created time.Time
		if err := row.Columns(&rec.OrderID, &rec.State, &rec.Amount, &raw, &created); err != nil {
			continue
		}
		rec.ItemCount = countCRMLineItems(raw)
		rec.Lines = parseCRMLines(raw)
		if rec.ItemCount == 0 {
			rec.ItemCount = int64(len(rec.Lines))
		}
		rec.CreatedAt = created.UTC().Format(time.RFC3339)
		out = append(out, rec)
	}
	return out, nil
}

func scanCRMRetailer(row *spanner.Row, now time.Time) (crmRetailer, error) {
	var rec crmRetailer
	var last spanner.NullTime
	if err := row.Columns(&rec.RetailerID, &rec.RetailerName, &rec.Phone, &rec.Email, &rec.Lifetime, &rec.OrderCount, &last); err != nil {
		return crmRetailer{}, err
	}
	if last.Valid {
		rec.LastOrderDate = last.Time.UTC().Format("2006-01-02")
		rec.Status = crmStatus(last.Time.UTC(), now.UTC())
	} else {
		rec.Status = "INACTIVE"
	}
	return rec, nil
}
