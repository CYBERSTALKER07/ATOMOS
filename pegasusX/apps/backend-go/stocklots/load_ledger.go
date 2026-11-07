package stocklots

import (
	"context"
	"fmt"
	"os"
	"strings" 
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// Load line statuses (G2.B).
const (
	LoadLineOpen             = "OPEN"
	LoadLineComplete         = "COMPLETE"
	LoadLineVarianceApproved = "VARIANCE_APPROVED"
)

// LoadLine is one required/scanned SKU line on a delivery truck manifest.
type LoadLine struct {
	ManifestID  string `json:"manifest_id"`
	OrderID     string `json:"order_id"`
	LineItemID  string `json:"line_item_id"`
	SkuID       string `json:"sku_id"`
	RequiredQty int64  `json:"required_qty"`
	ScannedQty  int64  `json:"scanned_qty"`
	Status      string `json:"status"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

// LoadLineSeed seeds a ledger row at start-loading / inject.
type LoadLineSeed struct {
	OrderID     string
	LineItemID  string
	SkuID       string
	RequiredQty int64
}


func memoryBlocked() error {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("PEGASUSX_ENV")), "production") ||
		strings.EqualFold(strings.TrimSpace(os.Getenv("PEGASUSX_ENV")), "sandbox") ||
		strings.EqualFold(strings.TrimSpace(os.Getenv("REQUIRE_INFRA_ADAPTERS")), "true") {
		return fmt.Errorf("in-memory load ledger blocked in production/infra mode")
	}
	return nil
}

// memoryLoadLedger is used when Spanner is unavailable (tests / demo overlay).
var (
	memLoadMu   sync.RWMutex
	memLoadRows = map[string]LoadLine{} // key: manifest|order|line
)

func loadKey(manifestID, orderID, lineID string) string {
	return manifestID + "|" + orderID + "|" + lineID
}

// SeedLoadLedgerInTxn upserts required qty lines for a manifest (Spanner path).
func SeedLoadLedgerInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, manifestID string, seeds []LoadLineSeed) error {
	manifestID = strings.TrimSpace(manifestID)
	if manifestID == "" || len(seeds) == 0 {
		return nil
	}
	now := time.Now().UTC()
	var muts []*spanner.Mutation
	for _, s := range seeds {
		lineID := strings.TrimSpace(s.LineItemID)
		if lineID == "" {
			lineID = strings.TrimSpace(s.SkuID)
		}
		if lineID == "" || strings.TrimSpace(s.OrderID) == "" {
			continue
		}
		req := s.RequiredQty
		if req < 0 {
			req = 0
		}
		status := LoadLineOpen
		if req == 0 {
			status = LoadLineComplete
		}
		muts = append(muts, spanner.InsertOrUpdateMap("ManifestLoadLines", map[string]interface{}{
			"ManifestId":  manifestID,
			"OrderId":     strings.TrimSpace(s.OrderID),
			"LineItemId":  lineID,
			"SkuId":       strings.TrimSpace(s.SkuID),
			"RequiredQty": req,
			"ScannedQty":  int64(0),
			"Status":      status,
			"UpdatedAt":   now,
		}))
	}
	if len(muts) == 0 {
		return nil
	}
	return txn.BufferWrite(muts)
}

// SeedLoadLedgerMemory seeds the in-process ledger (demo / tests).
func SeedLoadLedgerMemory(manifestID string, seeds []LoadLineSeed) {
	memLoadMu.Lock()
	defer memLoadMu.Unlock()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, s := range seeds {
		lineID := strings.TrimSpace(s.LineItemID)
		if lineID == "" {
			lineID = strings.TrimSpace(s.SkuID)
		}
		if lineID == "" || strings.TrimSpace(s.OrderID) == "" {
			continue
		}
		req := s.RequiredQty
		if req < 0 {
			req = 0
		}
		status := LoadLineOpen
		if req == 0 {
			status = LoadLineComplete
		}
		k := loadKey(manifestID, s.OrderID, lineID)
		memLoadRows[k] = LoadLine{
			ManifestID:  manifestID,
			OrderID:     strings.TrimSpace(s.OrderID),
			LineItemID:  lineID,
			SkuID:       strings.TrimSpace(s.SkuID),
			RequiredQty: req,
			ScannedQty:  0,
			Status:      status,
			UpdatedAt:   now,
		}
	}
}

// ScanLoadLineMemory increments scanned qty for a line (memory path).
func ScanLoadLineMemory(manifestID, orderID, lineOrSku string, delta int64) (*LoadLine, error) {
	if err := memoryBlocked(); err != nil {
		return nil, err
	}
	if delta <= 0 {
		delta = 1
	}
	memLoadMu.Lock()
	defer memLoadMu.Unlock()
	var hit *LoadLine
	var hitKey string
	for k, row := range memLoadRows {
		if row.ManifestID != manifestID {
			continue
		}
		if orderID != "" && row.OrderID != orderID {
			continue
		}
		if row.LineItemID == lineOrSku || row.SkuID == lineOrSku {
			cp := row
			hit = &cp
			hitKey = k
			break
		}
	}
	if hit == nil {
		return nil, fmt.Errorf("load_line_not_found")
	}
	hit.ScannedQty += delta
	if hit.ScannedQty >= hit.RequiredQty || hit.Status == LoadLineVarianceApproved {
		if hit.Status != LoadLineVarianceApproved {
			hit.Status = LoadLineComplete
		}
	} else {
		hit.Status = LoadLineOpen
	}
	hit.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	memLoadRows[hitKey] = *hit
	return hit, nil
}

// ApproveLoadVarianceMemory marks a line variance-approved.
func ApproveLoadVarianceMemory(manifestID, orderID, lineID string) (*LoadLine, error) {
	if err := memoryBlocked(); err != nil {
		return nil, err
	}
	memLoadMu.Lock()
	defer memLoadMu.Unlock()
	k := loadKey(manifestID, orderID, lineID)
	row, ok := memLoadRows[k]
	if !ok {
		// fuzzy by sku
		for key, r := range memLoadRows {
			if r.ManifestID == manifestID && r.OrderID == orderID && (r.LineItemID == lineID || r.SkuID == lineID) {
				k = key
				row = r
				ok = true
				break
			}
		}
	}
	if !ok {
		return nil, fmt.Errorf("load_line_not_found")
	}
	row.Status = LoadLineVarianceApproved
	row.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	memLoadRows[k] = row
	return &row, nil
}

// ListLoadLedgerMemory returns memory lines for a manifest.
func ListLoadLedgerMemory(manifestID string) []LoadLine {
	if err := memoryBlocked(); err != nil {
		return nil
	}
	memLoadMu.RLock()
	defer memLoadMu.RUnlock()
	out := make([]LoadLine, 0)
	for _, row := range memLoadRows {
		if row.ManifestID == manifestID {
			out = append(out, row)
		}
	}
	return out
}

// ResetLoadLedgerMemory clears memory ledger (tests).
func ResetLoadLedgerMemory() {
	memLoadMu.Lock()
	memLoadRows = map[string]LoadLine{}
	memLoadMu.Unlock()
}

// ListLoadLedger lists Spanner ledger lines for a manifest.
func ListLoadLedger(ctx context.Context, client *spanner.Client, manifestID string) ([]LoadLine, error) {
	if client == nil {
		return ListLoadLedgerMemory(manifestID), nil
	}
	manifestID = strings.TrimSpace(manifestID)
	stmt := spanner.Statement{
		SQL: `SELECT ManifestId, OrderId, LineItemId, SkuId, RequiredQty, ScannedQty, Status, UpdatedAt
		      FROM ManifestLoadLines WHERE ManifestId = @mid`,
		Params: map[string]interface{}{"mid": manifestID},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	var out []LoadLine
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			// Table missing → empty (migration not applied); fail open only for list.
			if strings.Contains(err.Error(), "ManifestLoadLines") || spanner.ErrCode(err) == 5 {
				return ListLoadLedgerMemory(manifestID), nil
			}
			return nil, err
		}
		var m, o, l, sku, status string
		var req, scan int64
		var updated time.Time
		if err := row.Columns(&m, &o, &l, &sku, &req, &scan, &status, &updated); err != nil {
			return nil, err
		}
		out = append(out, LoadLine{
			ManifestID: m, OrderID: o, LineItemID: l, SkuID: sku,
			RequiredQty: req, ScannedQty: scan, Status: status,
			UpdatedAt: updated.UTC().Format(time.RFC3339Nano),
		})
	}
	return out, nil
}

// ScanLoadLineInTxn increments scanned qty on a Spanner ledger line.
func ScanLoadLineInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, manifestID, orderID, lineOrSku string, delta int64) (*LoadLine, error) {
	if delta <= 0 {
		delta = 1
	}
	manifestID = strings.TrimSpace(manifestID)
	lineOrSku = strings.TrimSpace(lineOrSku)
	// Prefer exact line key when order known; else scan match by sku/line.
	stmt := spanner.Statement{
		SQL: `SELECT ManifestId, OrderId, LineItemId, SkuId, RequiredQty, ScannedQty, Status
		      FROM ManifestLoadLines WHERE ManifestId = @mid`,
		Params: map[string]interface{}{"mid": manifestID},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()
	var found *LoadLine
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var m, o, l, sku, status string
		var req, scan int64
		if err := row.Columns(&m, &o, &l, &sku, &req, &scan, &status); err != nil {
			return nil, err
		}
		if orderID != "" && o != orderID {
			continue
		}
		if l == lineOrSku || sku == lineOrSku {
			found = &LoadLine{ManifestID: m, OrderID: o, LineItemID: l, SkuID: sku, RequiredQty: req, ScannedQty: scan, Status: status}
			break
		}
	}
	if found == nil {
		return nil, fmt.Errorf("load_line_not_found")
	}
	found.ScannedQty += delta
	if found.Status != LoadLineVarianceApproved {
		if found.ScannedQty >= found.RequiredQty {
			found.Status = LoadLineComplete
		} else {
			found.Status = LoadLineOpen
		}
	}
	now := time.Now().UTC()
	if err := txn.BufferWrite([]*spanner.Mutation{
		spanner.UpdateMap("ManifestLoadLines", map[string]interface{}{
			"ManifestId":  found.ManifestID,
			"OrderId":     found.OrderID,
			"LineItemId":  found.LineItemID,
			"SkuId":       found.SkuID,
			"RequiredQty": found.RequiredQty,
			"ScannedQty":  found.ScannedQty,
			"Status":      found.Status,
			"UpdatedAt":   now,
		}),
	}); err != nil {
		return nil, err
	}
	found.UpdatedAt = now.Format(time.RFC3339Nano)
	return found, nil
}

// ApproveLoadVarianceInTxn marks VARIANCE_APPROVED so seal may proceed short.
func ApproveLoadVarianceInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, manifestID, orderID, lineID string) (*LoadLine, error) {
	row, err := txn.ReadRow(ctx, "ManifestLoadLines", spanner.Key{manifestID, orderID, lineID},
		[]string{"ManifestId", "OrderId", "LineItemId", "SkuId", "RequiredQty", "ScannedQty", "Status"})
	if err != nil {
		return nil, fmt.Errorf("load_line_not_found")
	}
	var m, o, l, sku, status string
	var req, scan int64
	if err := row.Columns(&m, &o, &l, &sku, &req, &scan, &status); err != nil {
		return nil, err
	}
	status = LoadLineVarianceApproved
	now := time.Now().UTC()
	if err := txn.BufferWrite([]*spanner.Mutation{
		spanner.UpdateMap("ManifestLoadLines", map[string]interface{}{
			"ManifestId": m, "OrderId": o, "LineItemId": l, "SkuId": sku,
			"RequiredQty": req, "ScannedQty": scan, "Status": status, "UpdatedAt": now,
		}),
	}); err != nil {
		return nil, err
	}
	return &LoadLine{ManifestID: m, OrderID: o, LineItemID: l, SkuID: sku, RequiredQty: req, ScannedQty: scan, Status: status, UpdatedAt: now.Format(time.RFC3339Nano)}, nil
}

// AssertLoadLedgerReady blocks seal when ledger is enabled and any line is incomplete
// without variance approval. Empty ledger (not seeded) → incomplete when flag on.
func AssertLoadLedgerReady(ctx context.Context, client *spanner.Client, manifestID string) error {
	manifestID = strings.TrimSpace(manifestID)
	supplierID, warehouseID := "", ""
	if client != nil && manifestID != "" {
		supplierID, warehouseID = manifestScope(ctx, client, manifestID)
	}
	if !EffectiveLoadLedger(ctx, warehouseID, supplierID) {
		return nil
	}
	if manifestID == "" {
		return ErrLoadLedgerIncomplete
	}
	lines, err := ListLoadLedger(ctx, client, manifestID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrLoadLedgerIncomplete, err)
	}
	if len(lines) == 0 {
		// Memory-only seed may exist without Spanner table.
		lines = ListLoadLedgerMemory(manifestID)
	}
	if len(lines) == 0 {
		return fmt.Errorf("%w: no_lines", ErrLoadLedgerIncomplete)
	}
	for _, ln := range lines {
		st := strings.ToUpper(strings.TrimSpace(ln.Status))
		if st == LoadLineComplete || st == LoadLineVarianceApproved {
			continue
		}
		if ln.ScannedQty >= ln.RequiredQty {
			continue
		}
		return fmt.Errorf("%w: order=%s line=%s scanned=%d required=%d",
			ErrLoadLedgerIncomplete, ln.OrderID, ln.LineItemID, ln.ScannedQty, ln.RequiredQty)
	}
	return nil
}
