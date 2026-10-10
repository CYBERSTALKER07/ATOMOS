package payload

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
)

func itoa(i int) string { return strconv.Itoa(i) }

// seedLoadLedgerForManifest builds required_qty lines from in-memory orders + Spanner line items.
func (s *Service) seedLoadLedgerForManifest(ctx context.Context, manifestID string) {
	manifestID = strings.TrimSpace(manifestID)
	if manifestID == "" {
		return
	}
	s.mu.Lock()
	orders := append([]ManifestOrder(nil), s.manifestOrders[manifestID]...)
	lineItems := s.orderLineItemsLocked()
	s.mu.Unlock()

	seeds := make([]stocklots.LoadLineSeed, 0)
	for _, mo := range orders {
		items := lineItems[mo.OrderID]
		if len(items) == 0 {
			// Placeholder line so ledger is non-empty when flag on and orders lack SKUs.
			seeds = append(seeds, stocklots.LoadLineSeed{
				OrderID: mo.OrderID, LineItemID: mo.OrderID + ":all", SkuID: "unknown", RequiredQty: 1,
			})
			continue
		}
		for i, it := range items {
			sku := strings.TrimSpace(it.SKUID)
			lineID := mo.OrderID + ":" + sku
			if sku == "" {
				lineID = mo.OrderID + ":" + itoa(i)
				sku = "sku_" + itoa(i)
			}
			qty := it.Quantity
			if qty <= 0 {
				qty = 1
			}
			seeds = append(seeds, stocklots.LoadLineSeed{
				OrderID: mo.OrderID, LineItemID: lineID, SkuID: sku, RequiredQty: qty,
			})
		}
	}
	if len(seeds) == 0 {
		return
	}
	stocklots.SeedLoadLedgerMemory(manifestID, seeds)
	client := s.spannerClient()
	if client == nil {
		return
	}
	_ = spannerutils.RunReadWriteTransaction(ctx, client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return stocklots.SeedLoadLedgerInTxn(ctx, txn, manifestID, seeds)
	})
}

// HandleLoadLedger GET /v1/payloader/manifests/{manifestID}/load-ledger
func (s *Service) HandleLoadLedger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	manifestID := manifestIDParam(r)
	if manifestID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "manifest_id_required"})
		return
	}
	enabled := stocklots.EffectiveLoadLedger(r.Context(), s.resolveWarehouseScope(r.Context()), s.resolveSupplierScope(r.Context()))
	lines, err := stocklots.ListLoadLedger(r.Context(), s.spannerClient(), manifestID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if len(lines) == 0 {
		lines = stocklots.ListLoadLedgerMemory(manifestID)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"manifest_id":          manifestID,
		"load_ledger_enabled":  enabled,
		"lines":                lines,
		"complete":             loadLedgerComplete(lines),
	})
}

func loadLedgerComplete(lines []stocklots.LoadLine) bool {
	if len(lines) == 0 {
		return false
	}
	for _, ln := range lines {
		st := strings.ToUpper(strings.TrimSpace(ln.Status))
		if st == stocklots.LoadLineComplete || st == stocklots.LoadLineVarianceApproved {
			continue
		}
		if ln.ScannedQty >= ln.RequiredQty {
			continue
		}
		return false
	}
	return true
}

// HandleLoadLedgerScan POST /v1/payloader/manifests/{manifestID}/load-ledger/scan
func (s *Service) HandleLoadLedgerScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	manifestID := manifestIDParam(r)
	if manifestID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "manifest_id_required"})
		return
	}
	body, err := readLimitedBody(r, 8*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	var req struct {
		OrderID    string `json:"order_id"`
		LineItemID string `json:"line_item_id"`
		SkuID      string `json:"sku_id"`
		Quantity   int64  `json:"quantity"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	key := strings.TrimSpace(req.LineItemID)
	if key == "" {
		key = strings.TrimSpace(req.SkuID)
	}
	if key == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "line_or_sku_required"})
		return
	}
	delta := req.Quantity
	if delta <= 0 {
		delta = 1
	}
	client := s.spannerClient()
	var line *stocklots.LoadLine
	if client != nil {
		err = spannerutils.RunReadWriteTransaction(r.Context(), client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			var e error
			line, e = stocklots.ScanLoadLineInTxn(ctx, txn, manifestID, req.OrderID, key, delta)
			return e
		})
		if err != nil && strings.Contains(err.Error(), "load_line_not_found") {
			// Fall back to memory.
			line, err = stocklots.ScanLoadLineMemory(manifestID, req.OrderID, key, delta)
		}
	} else {
		line, err = stocklots.ScanLoadLineMemory(manifestID, req.OrderID, key, delta)
	}
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"line": line, "status": "scanned"})
}

// HandleLoadLedgerVarianceApprove POST .../load-ledger/variance/approve
func (s *Service) HandleLoadLedgerVarianceApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	manifestID := manifestIDParam(r)
	if manifestID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "manifest_id_required"})
		return
	}
	body, err := readLimitedBody(r, 8*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	var req struct {
		OrderID    string `json:"order_id"`
		LineItemID string `json:"line_item_id"`
		Reason     string `json:"reason"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if strings.TrimSpace(req.OrderID) == "" || strings.TrimSpace(req.LineItemID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_and_line_required"})
		return
	}
	client := s.spannerClient()
	var line *stocklots.LoadLine
	if client != nil {
		err = spannerutils.RunReadWriteTransaction(r.Context(), client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			var e error
			line, e = stocklots.ApproveLoadVarianceInTxn(ctx, txn, manifestID, req.OrderID, req.LineItemID)
			return e
		})
		if err != nil {
			line, err = stocklots.ApproveLoadVarianceMemory(manifestID, req.OrderID, req.LineItemID)
		}
	} else {
		line, err = stocklots.ApproveLoadVarianceMemory(manifestID, req.OrderID, req.LineItemID)
	}
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"line":   line,
		"status": "variance_approved",
		"reason": strings.TrimSpace(req.Reason),
	})
}

