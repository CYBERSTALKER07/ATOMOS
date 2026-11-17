package stocklots

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

// PickValuationArgs describes a confirmed pick depletion passed to the optional inventory valuation hook.
type PickValuationArgs struct {
	EntityID       string
	WarehouseID    string
	ProductID      string
	LotID          string
	WaveID         string
	TaskID         string
	OrderID        string
	Quantity       int64
	PickerID       string
	IdempotencyKey string
}

// PickValuationHook depletes cost layers FEFO and posts COGS GL entries atomically inside confirmPickTaskCoreInTxn.
type PickValuationHook func(ctx context.Context, txn *spanner.ReadWriteTransaction, args PickValuationArgs) error

var (
	pickValuationHookMu sync.RWMutex
	pickValuationHook   PickValuationHook
)

// SetPickValuationHook registers an optional transactional FEFO COGS hook for pick confirmation.
func SetPickValuationHook(fn PickValuationHook) {
	pickValuationHookMu.Lock()
	defer pickValuationHookMu.Unlock()
	pickValuationHook = fn
}

func getPickValuationHook() PickValuationHook {
	pickValuationHookMu.RLock()
	defer pickValuationHookMu.RUnlock()
	return pickValuationHook
}

// PickWaveView is the API DTO for a pick wave.
type PickWaveView struct {
	WaveID      string         `json:"wave_id"`
	WarehouseID string         `json:"warehouse_id"`
	SupplierID  string         `json:"supplier_id"`
	ManifestID  string         `json:"manifest_id"`
	Strategy    string         `json:"strategy"`
	Status      string         `json:"status"`
	CreatedAt   string         `json:"created_at,omitempty"`
	ReadyAt     string         `json:"ready_at,omitempty"`
	Tasks       []PickTaskView `json:"tasks,omitempty"`
}

// PickTaskView is one pick line.
type PickTaskView struct {
	TaskID            string `json:"task_id"`
	OrderID           string `json:"order_id"`
	ProductID         string `json:"product_id"`
	LotID             string `json:"lot_id"`
	LocationID        string `json:"location_id"`
	QuantityRequested int64  `json:"quantity_requested"`
	QuantityPicked    int64  `json:"quantity_picked"`
	PickerID          string `json:"picker_id,omitempty"`
	Status            string `json:"status"`
	PickSequence      int64  `json:"pick_sequence"`
}

type pickTaskDraft struct {
	OrderID    string
	ProductID  string
	LotID      string
	LocationID string
	Qty        int64
	Seq        int64
	Zone       string
	Aisle      string
	StopRank   int64 // higher = pick earlier when LIFO (last delivery first)
}

// CreatePickWaveInTxn builds a MANIFEST strategy wave from a DRAFT/LOADING manifest.
func CreatePickWaveInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID, warehouseID, manifestID string) (*PickWaveView, error) {
	supplierID = strings.TrimSpace(supplierID)
	warehouseID = strings.TrimSpace(warehouseID)
	manifestID = strings.TrimSpace(manifestID)
	if supplierID == "" || warehouseID == "" || manifestID == "" {
		return nil, fmt.Errorf("supplier_id, warehouse_id, manifest_id required")
	}

	mRow, err := txn.ReadRow(ctx, "SupplierTruckManifests", spanner.Key{manifestID},
		[]string{"SupplierId", "WarehouseId", "State", "PickWaveId"})
	if err != nil {
		if spanner.ErrCode(err) == 5 {
			return nil, fmt.Errorf("manifest_not_found")
		}
		return nil, err
	}
	var mSupplier, mWarehouse, state string
	var existingWave spanner.NullString
	if err := mRow.Columns(&mSupplier, &mWarehouse, &state, &existingWave); err != nil {
		return nil, err
	}
	if existingWave.Valid && strings.TrimSpace(existingWave.StringVal) != "" {
		return nil, fmt.Errorf("pick_wave_exists")
	}
	state = strings.ToUpper(strings.TrimSpace(state))
	if state != "DRAFT" && state != "LOADING" {
		return nil, fmt.Errorf("manifest_not_pickable")
	}
	if strings.TrimSpace(mWarehouse) != "" {
		warehouseID = strings.TrimSpace(mWarehouse)
	}

	existIter := txn.Query(ctx, spanner.Statement{
		SQL:    `SELECT WaveId FROM PickWaves WHERE ManifestId = @mid LIMIT 1`,
		Params: map[string]any{"mid": manifestID},
	})
	existRow, existErr := existIter.Next()
	existIter.Stop()
	if existErr == nil && existRow != nil {
		return nil, fmt.Errorf("pick_wave_exists")
	}
	if existErr != nil && existErr != iterator.Done {
		return nil, existErr
	}

	orderIDs, err := loadManifestOrderIDs(ctx, txn, manifestID)
	if err != nil {
		return nil, err
	}
	if len(orderIDs) == 0 {
		return nil, fmt.Errorf("manifest_has_no_orders")
	}

	resByOrder, err := loadOrderLotReservationsBatch(ctx, txn, orderIDs)
	if err != nil {
		return nil, err
	}

	var uniqueLotIDs []string
	lotSeen := make(map[string]struct{})
	var fallbackOrderIDs []string
	for _, oid := range orderIDs {
		resRows := resByOrder[oid]
		if len(resRows) == 0 {
			fallbackOrderIDs = append(fallbackOrderIDs, oid)
			continue
		}
		for _, rr := range resRows {
			if _, ok := lotSeen[rr.lotID]; !ok {
				lotSeen[rr.lotID] = struct{}{}
				uniqueLotIDs = append(uniqueLotIDs, rr.lotID)
			}
		}
	}

	lotsByID, err := loadStockLotsBatch(ctx, txn, uniqueLotIDs)
	if err != nil {
		return nil, err
	}

	var uniqueLocIDs []string
	locSeen := make(map[string]struct{})
	for _, lot := range lotsByID {
		if lot.status != "AVAILABLE" {
			continue
		}
		if lid := strings.TrimSpace(lot.locationID); lid != "" {
			if _, ok := locSeen[lid]; !ok {
				locSeen[lid] = struct{}{}
				uniqueLocIDs = append(uniqueLocIDs, lid)
			}
		}
	}
	locMetaByID := locationPickMetaBatch(ctx, txn, warehouseID, uniqueLocIDs)

	linesByOrder, err := loadOrderLineQtysBatch(ctx, txn, fallbackOrderIDs)
	if err != nil {
		return nil, err
	}

	var drafts []pickTaskDraft
	for _, oid := range orderIDs {
		resRows := resByOrder[oid]
		if len(resRows) > 0 {
			for _, rr := range resRows {
				lot, ok := lotsByID[rr.lotID]
				if !ok {
					continue
				}
				if NormalizeLotStatus(lot.status) != "AVAILABLE" {
					continue
				}
				qty := rr.qty
				if qty > lot.reserved && lot.reserved > 0 {
					qty = lot.reserved
				}
				if qty <= 0 {
					continue
				}
				meta := locMetaByID[lot.locationID]
				drafts = append(drafts, pickTaskDraft{
					OrderID: oid, ProductID: lot.productID, LotID: rr.lotID, LocationID: lot.locationID,
					Qty: qty, Seq: meta.seq, Zone: meta.zone, Aisle: meta.aisle,
				})
			}
			continue
		}
		// Lots off: bag-of-SKU tasks. Confirm does not deplete StockLots.
		drafts = append(drafts, skuPickDraftsFromLines(oid, linesByOrder[oid])...)
	}

	if len(drafts) == 0 {
		return nil, fmt.Errorf("no_pick_tasks")
	}
	applyStopRanks(ctx, txn, manifestID, drafts)
	if PickSShapeEnabled() {
		sortSShapePickTaskDrafts(drafts)
	} else {
		sortPickTaskDrafts(drafts)
	}

	waveID := uuid.NewString()
	if err := txn.BufferWrite([]*spanner.Mutation{spanner.InsertMap("PickWaves", map[string]any{
		"WaveId":      waveID,
		"WarehouseId": warehouseID,
		"SupplierId":  supplierID,
		"ManifestId":  manifestID,
		"Strategy":    "MANIFEST",
		"Status":      "OPEN",
		"CreatedAt":   spanner.CommitTimestamp,
		"UpdatedAt":   spanner.CommitTimestamp,
	})}); err != nil {
		return nil, err
	}
	if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("SupplierTruckManifests", map[string]any{
		"ManifestId": manifestID,
		"PickWaveId": waveID,
		"UpdatedAt":  spanner.CommitTimestamp,
	})}); err != nil {
		return nil, err
	}

	tasks := make([]PickTaskView, 0, len(drafts))
	for i, d := range drafts {
		taskID := uuid.NewString()
		seq := d.Seq
		if seq == 0 {
			seq = int64(i + 1)
		}
		if err := txn.BufferWrite([]*spanner.Mutation{spanner.InsertMap("PickTasks", map[string]any{
			"WaveId":            waveID,
			"TaskId":            taskID,
			"OrderId":           d.OrderID,
			"ProductId":         d.ProductID,
			"LotId":             d.LotID,
			"LocationId":        d.LocationID,
			"QuantityRequested": d.Qty,
			"QuantityPicked":    int64(0),
			"Status":            "PENDING",
			"PickSequence":      seq,
			"CreatedAt":         spanner.CommitTimestamp,
			"UpdatedAt":         spanner.CommitTimestamp,
		})}); err != nil {
			return nil, err
		}
		tasks = append(tasks, PickTaskView{
			TaskID: taskID, OrderID: d.OrderID, ProductID: d.ProductID, LotID: d.LotID,
			LocationID: d.LocationID, QuantityRequested: d.Qty, Status: "PENDING", PickSequence: seq,
		})
	}
	return &PickWaveView{
		WaveID: waveID, WarehouseID: warehouseID, SupplierID: supplierID,
		ManifestID: manifestID, Strategy: "MANIFEST", Status: "OPEN", Tasks: tasks,
	}, nil
}

// resolvePickConfirmQty turns an optional picked qty into a concrete quantity:
// nil defaults to requested; explicit 0 remains 0 (SHORT); negative values fail.
func resolvePickConfirmQty(requested int64, qty *int64) (int64, error) {
	if qty == nil {
		if requested < 0 {
			return 0, fmt.Errorf("quantity_requested_negative")
		}
		return requested, nil
	}
	if *qty < 0 {
		return 0, fmt.Errorf("quantity_picked_negative")
	}
	resolved := *qty
	if requested >= 0 && resolved > requested {
		resolved = requested
	}
	return resolved, nil
}

// ConfirmPickTaskInTxn confirms a task qty (0 = explicit SHORT without lot depletion) and may advance wave to READY_TO_SEAL.
func ConfirmPickTaskInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, waveID, taskID, pickerID string, quantityPicked int64) (*PickWaveView, error) {
	if quantityPicked < 0 {
		return nil, fmt.Errorf("quantity_picked_negative")
	}
	view, _, err := confirmPickTaskCoreInTxn(ctx, txn, waveID, taskID, pickerID, &quantityPicked)
	return view, err
}

// ConfirmPickTaskOptionalInTxn confirms a task using an optional quantity pointer (nil = requested, 0 = explicit SHORT).
func ConfirmPickTaskOptionalInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, waveID, taskID, pickerID string, quantityPicked *int64) (*PickWaveView, int64, error) {
	return confirmPickTaskCoreInTxn(ctx, txn, waveID, taskID, pickerID, quantityPicked)
}

func confirmPickTaskCoreInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, waveID, taskID, pickerID string, quantityPickedOpt *int64) (*PickWaveView, int64, error) {
	if quantityPickedOpt != nil && *quantityPickedOpt < 0 {
		return nil, 0, fmt.Errorf("quantity_picked_negative")
	}
	waveID = strings.TrimSpace(waveID)
	taskID = strings.TrimSpace(taskID)
	taskRow, err := txn.ReadRow(ctx, "PickTasks", spanner.Key{waveID, taskID},
		[]string{"OrderId", "ProductId", "LotId", "LocationId", "QuantityRequested", "QuantityPicked", "Status", "PickSequence"})
	if err != nil {
		return nil, 0, err
	}
	var orderID, productID, lotID, locationID, status string
	var requested, picked, seq int64
	if err := taskRow.Columns(&orderID, &productID, &lotID, &locationID, &requested, &picked, &status, &seq); err != nil {
		return nil, 0, err
	}
	status = strings.ToUpper(strings.TrimSpace(status))
	if status == "CONFIRMED" || status == "SHORT_WAIVED" {
		view, err := GetPickWaveInTxn(ctx, txn, waveID, true)
		return view, picked, err
	}
	quantityPicked, err := resolvePickConfirmQty(requested, quantityPickedOpt)
	if err != nil {
		return nil, 0, err
	}
	newStatus := pickTaskStatusForQty(requested, quantityPicked)
	depleteQty := quantityPicked
	if status == "SHORT" {
		depleteQty = quantityPicked - picked
	}

	if depleteQty > 0 && shouldDepleteLot(lotID) {
		lotRow, err := txn.ReadRow(ctx, "StockLots", spanner.Key{lotID},
			[]string{"SupplierId", "WarehouseId", "ProductId", "QuantityOnHand", "QuantityReserved", "Status"})
		if err != nil {
			return nil, 0, err
		}
		var sid, wid, pid, lotStatus string
		var qoh, qr int64
		if err := lotRow.Columns(&sid, &wid, &pid, &qoh, &qr, &lotStatus); err != nil {
			return nil, 0, err
		}
		if NormalizeLotStatus(lotStatus) != "AVAILABLE" {
			return nil, 0, fmt.Errorf("cannot pick from lot %s in status %s", lotID, lotStatus)
		}
		nextQoH, nextQR, depleted := applyLotDepletion(qoh, qr, depleteQty)
		lotStatusOut := lotStatus
		if depleted {
			lotStatusOut = "DEPLETED"
		}
		// Compute InventoryV2 rollup using the post-depletion lot snapshot BEFORE buffering
		// StockLots writes because Spanner txn.Query does not observe same-txn BufferWrite.
		overrides := map[string]StockLotSnapshot{
			lotID: {
				SupplierID:       sid,
				WarehouseID:      wid,
				ProductID:        pid,
				QuantityOnHand:   nextQoH,
				QuantityReserved: nextQR,
				Status:           lotStatusOut,
			},
		}
		if err := RollupInventoryV2WithLotOverridesInTxn(ctx, txn, sid, wid, pid, overrides); err != nil {
			return nil, 0, err
		}
		if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("StockLots", map[string]any{
			"LotId":            lotID,
			"QuantityOnHand":   nextQoH,
			"QuantityReserved": nextQR,
			"Status":           lotStatusOut,
			"UpdatedAt":        spanner.CommitTimestamp,
		})}); err != nil {
			return nil, 0, err
		}
		if hook := getPickValuationHook(); hook != nil {
			if hErr := hook(ctx, txn, PickValuationArgs{
				EntityID:       sid,
				WarehouseID:    wid,
				ProductID:      pid,
				LotID:          lotID,
				WaveID:         waveID,
				TaskID:         taskID,
				OrderID:        orderID,
				Quantity:       depleteQty,
				PickerID:       strings.TrimSpace(pickerID),
				IdempotencyKey: fmt.Sprintf("pick:%s:%s:%d:%d", waveID, taskID, picked, quantityPicked),
			}); hErr != nil {
				return nil, 0, hErr
			}
		}
	}

	// Read current wave & task snapshot BEFORE buffering task/wave mutations so readiness
	// evaluation and returned PickWaveView are consistent in a single atomic transaction.
	view, err := GetPickWaveInTxn(ctx, txn, waveID, true)
	if err != nil {
		return nil, 0, err
	}

	cols := map[string]any{
		"WaveId":         waveID,
		"TaskId":         taskID,
		"QuantityPicked": quantityPicked,
		"Status":         newStatus,
		"UpdatedAt":      spanner.CommitTimestamp,
	}
	if strings.TrimSpace(pickerID) != "" {
		cols["PickerId"] = strings.TrimSpace(pickerID)
	}
	if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("PickTasks", cols)}); err != nil {
		return nil, 0, err
	}

	if view != nil {
		for i := range view.Tasks {
			if view.Tasks[i].TaskID == taskID {
				view.Tasks[i].Status = newStatus
				view.Tasks[i].QuantityPicked = quantityPicked
				if strings.TrimSpace(pickerID) != "" {
					view.Tasks[i].PickerID = strings.TrimSpace(pickerID)
				}
			}
		}
		nextWaveStatus, readyAt := nextPickWaveStatusAfterTasks(view.Status, view.Tasks, time.Now().UTC())
		if nextWaveStatus != "" && !strings.EqualFold(view.Status, nextWaveStatus) {
			waveUpdate := map[string]any{
				"WaveId":    waveID,
				"Status":    nextWaveStatus,
				"UpdatedAt": spanner.CommitTimestamp,
			}
			if readyAt != nil {
				waveUpdate["ReadyAt"] = *readyAt
				view.ReadyAt = readyAt.UTC().Format(time.RFC3339)
			}
			if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("PickWaves", waveUpdate)}); err != nil {
				return nil, 0, err
			}
			view.Status = nextWaveStatus
		}
	}
	return view, quantityPicked, nil
}

// WaivedShortItem describes one line item reduced or cut during a wave short waive.
type WaivedShortItem struct {
	SKU            string `json:"sku"`
	OriginalQty    int64  `json:"original_qty"`
	PickedQty      int64  `json:"picked_qty"`
	ShortedQty     int64  `json:"shorted_qty"`
	UnitPriceMinor int64  `json:"unit_price_minor"`
	ShortedMinor   int64  `json:"shorted_minor"`
}

type orderWaiveLinesResult struct {
	UpdatedJSON       []byte
	SubtotalMinor     int64
	NewTotalMinor     int64
	NextOriginalTotal int64
	ShortedMinor      int64
	ShortedItems      []WaivedShortItem
}

func jsonInt64(v any) (int64, error) {
	if v == nil {
		return 0, nil
	}
	switch n := v.(type) {
	case json.Number:
		if strings.TrimSpace(n.String()) == "" {
			return 0, nil
		}
		return n.Int64()
	case int64:
		return n, nil
	case int:
		return int64(n), nil
	default:
		return 0, fmt.Errorf("expected integer json value, got %T", v)
	}
}

func jsonStr(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func lineSKUFromMap(item map[string]any) string {
	for _, k := range []string{"sku", "sku_id", "product_id"} {
		if s := jsonStr(item[k]); s != "" {
			return s
		}
	}
	return ""
}

// applyWaivedShortsToLineItems reduces affected SKU line quantities by waivedBySKU using
// pure int64 arithmetic, cuts zero-qty lines, preserves arbitrary line metadata, and
// recomputes subtotal and order total (including delivery fee when subtotal > 0).
func applyWaivedShortsToLineItems(
	raw []byte,
	waivedBySKU map[string]int64,
	prevTotalMinor, origTotalMinor, deliveryFeeMinor int64,
) (orderWaiveLinesResult, error) {
	var items []map[string]any
	if len(bytes.TrimSpace(raw)) > 0 {
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.UseNumber()
		if err := dec.Decode(&items); err != nil {
			return orderWaiveLinesResult{}, fmt.Errorf("decode order line items: %w", err)
		}
	}

	remaining := make(map[string]int64, len(waivedBySKU))
	for sku, q := range waivedBySKU {
		if q > 0 {
			remaining[strings.TrimSpace(sku)] += q
		}
	}

	updated := make([]map[string]any, 0, len(items))
	var shorted []WaivedShortItem
	var subtotalMinor int64
	var shortedMinor int64

	for _, item := range items {
		sku := lineSKUFromMap(item)
		qty, err := jsonInt64(item["quantity"])
		if err != nil {
			return orderWaiveLinesResult{}, fmt.Errorf("invalid quantity for sku %q: %w", sku, err)
		}
		unitPrice, err := jsonInt64(item["unit_price_minor"])
		if err != nil {
			return orderWaiveLinesResult{}, fmt.Errorf("invalid unit_price_minor for sku %q: %w", sku, err)
		}

		if sku != "" && remaining[sku] > 0 && qty > 0 {
			cut := remaining[sku]
			if cut > qty {
				cut = qty
			}
			remaining[sku] -= cut
			newQty := qty - cut
			origQty := qty
			if prevOrig, err := jsonInt64(item["original_quantity"]); err == nil && prevOrig > 0 {
				origQty = prevOrig
			}
			prevShorted, _ := jsonInt64(item["shorted_qty"])
			lineShortMinor := cut * unitPrice
			shortedMinor += lineShortMinor
			shorted = append(shorted, WaivedShortItem{
				SKU:            sku,
				OriginalQty:    origQty,
				PickedQty:      newQty,
				ShortedQty:     cut,
				UnitPriceMinor: unitPrice,
				ShortedMinor:   lineShortMinor,
			})
			if newQty == 0 {
				// Cut 0-picked lines from active LineItemsJson; details remain in ShortedItems/outbox.
				continue
			}
			item["quantity"] = newQty
			item["original_quantity"] = origQty
			item["shorted_qty"] = prevShorted + cut
			subtotalMinor += newQty * unitPrice
			updated = append(updated, item)
			continue
		}

		if qty > 0 {
			subtotalMinor += qty * unitPrice
		}
		updated = append(updated, item)
	}

	newTotalMinor := subtotalMinor
	if subtotalMinor > 0 && deliveryFeeMinor > 0 {
		newTotalMinor = subtotalMinor + deliveryFeeMinor
	}
	nextOrigTotal := origTotalMinor
	if nextOrigTotal == 0 {
		nextOrigTotal = prevTotalMinor
	}

	encoded, err := json.Marshal(updated)
	if err != nil {
		return orderWaiveLinesResult{}, fmt.Errorf("encode updated line items: %w", err)
	}
	return orderWaiveLinesResult{
		UpdatedJSON:       encoded,
		SubtotalMinor:     subtotalMinor,
		NewTotalMinor:     newTotalMinor,
		NextOriginalTotal: nextOrigTotal,
		ShortedMinor:      shortedMinor,
		ShortedItems:      shorted,
	}, nil
}

// WaiveShortsInTxn marks SHORT tasks as SHORT_WAIVED, trims unpicked order lines and totals,
// releases unpicked lot/inventory reservations, adjusts credit holds/balances/AR invoices,
// emits outbox events in the same transaction, and advances to READY_TO_SEAL when eligible.
func WaiveShortsInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, waveID string) (*PickWaveView, error) {
	waveID = strings.TrimSpace(waveID)
	view, err := GetPickWaveInTxn(ctx, txn, waveID, true)
	if err != nil {
		return nil, err
	}

	type orderLotKey struct {
		orderID string
		lotID   string
	}
	type skuTuple struct {
		supplierID  string
		warehouseID string
		productID   string
	}

	var taskMuts []*spanner.Mutation
	waivedByOrderSKU := make(map[string]map[string]int64)
	waivedByOrderLot := make(map[orderLotKey]int64)
	pickedByOrderLot := make(map[orderLotKey]int64)
	waivedByLot := make(map[string]int64)
	waivedNonLotByTuple := make(map[skuTuple]int64)
	affectedOrderSet := make(map[string]struct{})
	var waivedTaskIDs []string

	for i := range view.Tasks {
		t := &view.Tasks[i]
		if shouldDepleteLot(t.LotID) && t.QuantityPicked > 0 {
			pickedByOrderLot[orderLotKey{orderID: t.OrderID, lotID: t.LotID}] += t.QuantityPicked
		}
		if !strings.EqualFold(strings.TrimSpace(t.Status), "SHORT") {
			continue
		}
		t.Status = "SHORT_WAIVED"
		waivedTaskIDs = append(waivedTaskIDs, t.TaskID)
		taskMuts = append(taskMuts, spanner.UpdateMap("PickTasks", map[string]any{
			"WaveId":    waveID,
			"TaskId":    t.TaskID,
			"Status":    "SHORT_WAIVED",
			"UpdatedAt": spanner.CommitTimestamp,
		}))

		shortQty := t.QuantityRequested - t.QuantityPicked
		if t.QuantityPicked < 0 || shortQty <= 0 {
			continue
		}
		oid := strings.TrimSpace(t.OrderID)
		if oid != "" {
			affectedOrderSet[oid] = struct{}{}
			if waivedByOrderSKU[oid] == nil {
				waivedByOrderSKU[oid] = make(map[string]int64)
			}
			waivedByOrderSKU[oid][t.ProductID] += shortQty
			if shouldDepleteLot(t.LotID) {
				waivedByOrderLot[orderLotKey{orderID: oid, lotID: t.LotID}] += shortQty
			}
		}
		if shouldDepleteLot(t.LotID) {
			waivedByLot[t.LotID] += shortQty
		} else if strings.TrimSpace(t.ProductID) != "" {
			waivedNonLotByTuple[skuTuple{
				supplierID:  view.SupplierID,
				warehouseID: view.WarehouseID,
				productID:   strings.TrimSpace(t.ProductID),
			}] += shortQty
		}
	}

	affectedOrderIDs := make([]string, 0, len(affectedOrderSet))
	for oid := range affectedOrderSet {
		affectedOrderIDs = append(affectedOrderIDs, oid)
	}
	sort.Strings(affectedOrderIDs)

	var domainMuts []*spanner.Mutation

	// 1. Release unpicked OrderLotReservations and StockLots.QuantityReserved + roll up SupplierInventoryV2.
	rolledUpTuples := make(map[skuTuple]bool)
	if len(affectedOrderIDs) > 0 && len(waivedByOrderLot) > 0 {
		resByOrder, err := loadOrderLotReservationsBatch(ctx, txn, affectedOrderIDs)
		if err != nil {
			return nil, fmt.Errorf("load order lot reservations: %w", err)
		}
		for _, oid := range affectedOrderIDs {
			for _, res := range resByOrder[oid] {
				key := orderLotKey{orderID: oid, lotID: res.lotID}
				if waivedByOrderLot[key] <= 0 {
					continue
				}
				resolved := pickedByOrderLot[key] + waivedByOrderLot[key]
				remaining := res.qty - resolved
				if remaining <= 0 {
					domainMuts = append(domainMuts, spanner.Delete("OrderLotReservations", spanner.Key{oid, res.lotID}))
				} else {
					domainMuts = append(domainMuts, spanner.UpdateMap("OrderLotReservations", map[string]any{
						"OrderId":  oid,
						"LotId":    res.lotID,
						"Quantity": remaining,
					}))
				}
			}
		}
	}

	if len(waivedByLot) > 0 {
		lotIDs := make([]string, 0, len(waivedByLot))
		for lid := range waivedByLot {
			lotIDs = append(lotIDs, lid)
		}
		sort.Strings(lotIDs)

		overridesByTuple := make(map[skuTuple]map[string]StockLotSnapshot)
		for _, lid := range lotIDs {
			row, err := txn.ReadRow(ctx, "StockLots", spanner.Key{lid},
				[]string{"SupplierId", "WarehouseId", "ProductId", "QuantityOnHand", "QuantityReserved", "Status"})
			if err != nil {
				return nil, fmt.Errorf("read stock lot %s: %w", lid, err)
			}
			var sid, wid, pid, lotStatus string
			var qoh, qr int64
			if err := row.Columns(&sid, &wid, &pid, &qoh, &qr, &lotStatus); err != nil {
				return nil, err
			}
			nextQR := qr - waivedByLot[lid]
			if nextQR < 0 {
				nextQR = 0
			}
			tKey := skuTuple{supplierID: sid, warehouseID: wid, productID: pid}
			if overridesByTuple[tKey] == nil {
				overridesByTuple[tKey] = make(map[string]StockLotSnapshot)
			}
			overridesByTuple[tKey][lid] = StockLotSnapshot{
				SupplierID:       sid,
				WarehouseID:      wid,
				ProductID:        pid,
				QuantityOnHand:   qoh,
				QuantityReserved: nextQR,
				Status:           lotStatus,
			}
			domainMuts = append(domainMuts, spanner.UpdateMap("StockLots", map[string]any{
				"LotId":            lid,
				"QuantityReserved": nextQR,
				"UpdatedAt":        spanner.CommitTimestamp,
			}))
		}

		tuples := make([]skuTuple, 0, len(overridesByTuple))
		for tk := range overridesByTuple {
			tuples = append(tuples, tk)
		}
		sort.Slice(tuples, func(i, j int) bool {
			if tuples[i].supplierID != tuples[j].supplierID {
				return tuples[i].supplierID < tuples[j].supplierID
			}
			if tuples[i].warehouseID != tuples[j].warehouseID {
				return tuples[i].warehouseID < tuples[j].warehouseID
			}
			return tuples[i].productID < tuples[j].productID
		})
		for _, tk := range tuples {
			if err := RollupInventoryV2WithLotOverridesInTxn(ctx, txn, tk.supplierID, tk.warehouseID, tk.productID, overridesByTuple[tk]); err != nil {
				return nil, err
			}
			rolledUpTuples[tk] = true
		}
	}

	if len(waivedNonLotByTuple) > 0 {
		nonLotTuples := make([]skuTuple, 0, len(waivedNonLotByTuple))
		for tk := range waivedNonLotByTuple {
			if !rolledUpTuples[tk] && tk.supplierID != "" && tk.warehouseID != "" && tk.productID != "" {
				nonLotTuples = append(nonLotTuples, tk)
			}
		}
		sort.Slice(nonLotTuples, func(i, j int) bool {
			if nonLotTuples[i].supplierID != nonLotTuples[j].supplierID {
				return nonLotTuples[i].supplierID < nonLotTuples[j].supplierID
			}
			if nonLotTuples[i].warehouseID != nonLotTuples[j].warehouseID {
				return nonLotTuples[i].warehouseID < nonLotTuples[j].warehouseID
			}
			return nonLotTuples[i].productID < nonLotTuples[j].productID
		})
		for _, tk := range nonLotTuples {
			row, err := txn.ReadRow(ctx, "SupplierInventoryV2", spanner.Key{tk.supplierID, tk.warehouseID, tk.productID},
				[]string{"QuantityReserved"})
			if err != nil {
				if spanner.ErrCode(err) == codes.NotFound {
					continue
				}
				return nil, fmt.Errorf("read SupplierInventoryV2 reserved: %w", err)
			}
			var qr int64
			if err := row.Columns(&qr); err != nil {
				return nil, err
			}
			nextQR := qr - waivedNonLotByTuple[tk]
			if nextQR < 0 {
				nextQR = 0
			}
			domainMuts = append(domainMuts, spanner.UpdateMap("SupplierInventoryV2", map[string]any{
				"SupplierId":       tk.supplierID,
				"WarehouseId":      tk.warehouseID,
				"ProductId":        tk.productID,
				"QuantityReserved": nextQR,
				"UpdatedAt":        spanner.CommitTimestamp,
			}))
		}
	}

	// 2. Reduce order lines, recompute totals, batch credit/AR adjustments, and emit outbox per affected order.
	orderNewTotals := make(map[string]int64, len(affectedOrderIDs))
	orderOutbox := outbox.NewSpannerTxnBuffer(txn)
	nowStr := time.Now().UTC().Format(time.RFC3339Nano)
	for _, oid := range affectedOrderIDs {
		ordRow, err := txn.ReadRow(ctx, "Orders", spanner.Key{oid}, []string{
			"SupplierId", "RetailerId", "WarehouseId", "Status",
			"LineItemsJson", "TotalMinor", "OriginalTotalMinor", "DeliveryFeeMinor", "Currency", "Version",
		})
		if err != nil {
			return nil, fmt.Errorf("read order %s: %w", oid, err)
		}
		var sid, rid, ordStatus, currency string
		var wid spanner.NullString
		var lineRaw []byte
		var prevTotal, origTotal, deliveryFee, version int64
		if err := ordRow.Columns(&sid, &rid, &wid, &ordStatus, &lineRaw, &prevTotal, &origTotal, &deliveryFee, &currency, &version); err != nil {
			return nil, err
		}

		res, err := applyWaivedShortsToLineItems(lineRaw, waivedByOrderSKU[oid], prevTotal, origTotal, deliveryFee)
		if err != nil {
			return nil, err
		}
		domainMuts = append(domainMuts, spanner.UpdateMap("Orders", map[string]any{
			"OrderId":            oid,
			"LineItemsJson":      res.UpdatedJSON,
			"TotalMinor":         res.NewTotalMinor,
			"OriginalTotalMinor": res.NextOriginalTotal,
			"Version":            version + 1,
			"UpdatedAt":          spanner.CommitTimestamp,
		}))

		if res.NewTotalMinor != prevTotal {
			orderNewTotals[oid] = res.NewTotalMinor
		}

		whVal := view.WarehouseID
		if wid.Valid && strings.TrimSpace(wid.StringVal) != "" {
			whVal = strings.TrimSpace(wid.StringVal)
		}
		if err := outbox.EmitJSON(ctx, orderOutbox, events.AggregateOrder, oid, events.TopicMain, map[string]any{
			"type":               events.EventOrderAmended,
			"order_id":           oid,
			"supplier_id":        sid,
			"retailer_id":        rid,
			"warehouse_id":       whVal,
			"wave_id":            waveID,
			"status":             ordStatus,
			"reason":             "wms_short_waived",
			"previous_total":     prevTotal,
			"subtotal_minor":     res.SubtotalMinor,
			"adjusted_total":     res.NewTotalMinor,
			"total_minor":        res.NewTotalMinor,
			"original_amount":    res.NextOriginalTotal,
			"short_waived_minor": prevTotal - res.NewTotalMinor,
			"currency":           currency,
			"shorted_items":      res.ShortedItems,
			"timestamp":          nowStr,
		}); err != nil {
			return nil, err
		}
	}

	if len(orderNewTotals) > 0 {
		if err := (&credit.SpannerRepository{}).AdjustReservesBatchInTxn(ctx, txn, orderNewTotals); err != nil {
			return nil, fmt.Errorf("adjust credit reserves for wave %s: %w", waveID, err)
		}
	}

	if len(taskMuts) > 0 {
		domainMuts = append(domainMuts, taskMuts...)
	}
	if len(domainMuts) > 0 {
		if err := txn.BufferWrite(domainMuts); err != nil {
			return nil, err
		}
	}
	if err := orderOutbox.Flush(ctx); err != nil {
		return nil, err
	}
	if len(waivedTaskIDs) > 0 {
		if err := emitWMSEvent(ctx, txn, events.EventWMSPickConfirmed, view.WarehouseID, view.SupplierID, waveID, map[string]any{
			"wave_id":         waveID,
			"action":          "SHORT_WAIVED",
			"waived_tasks":    waivedTaskIDs,
			"affected_orders": affectedOrderIDs,
		}); err != nil {
			return nil, err
		}
	}

	nextWaveStatus, readyAt := nextPickWaveStatusAfterTasks(view.Status, view.Tasks, time.Now().UTC())
	if nextWaveStatus == "READY_TO_SEAL" && !strings.EqualFold(view.Status, "READY_TO_SEAL") {
		waveUpdate := map[string]any{
			"WaveId":    waveID,
			"Status":    "READY_TO_SEAL",
			"UpdatedAt": spanner.CommitTimestamp,
		}
		if readyAt != nil {
			waveUpdate["ReadyAt"] = *readyAt
			view.ReadyAt = readyAt.UTC().Format(time.RFC3339)
		}
		if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("PickWaves", waveUpdate)}); err != nil {
			return nil, err
		}
		view.Status = "READY_TO_SEAL"
	}
	return view, nil
}

func nextPickWaveStatusAfterTasks(currentWaveStatus string, tasks []PickTaskView, now time.Time) (string, *time.Time) {
	if waveReadyFromTasks(tasks) {
		t := now.UTC()
		return "READY_TO_SEAL", &t
	}
	if strings.EqualFold(strings.TrimSpace(currentWaveStatus), "OPEN") {
		return "PICKING", nil
	}
	return currentWaveStatus, nil
}

func waveReadyFromTasks(tasks []PickTaskView) bool {
	if len(tasks) == 0 {
		return false
	}
	for _, t := range tasks {
		st := strings.ToUpper(strings.TrimSpace(t.Status))
		switch st {
		case "CONFIRMED", "SHORT_WAIVED":
		default:
			return false
		}
	}
	return true
}

// GetPickWave loads a wave (optionally with tasks).
func GetPickWave(ctx context.Context, client *spanner.Client, waveID string, withTasks bool) (*PickWaveView, error) {
	if client == nil {
		return nil, fmt.Errorf("spanner required")
	}
	row, err := client.Single().ReadRow(ctx, "PickWaves", spanner.Key{strings.TrimSpace(waveID)},
		[]string{"WaveId", "WarehouseId", "SupplierId", "ManifestId", "Strategy", "Status", "CreatedAt", "ReadyAt"})
	if err != nil {
		return nil, err
	}
	v, err := scanWave(row)
	if err != nil {
		return nil, err
	}
	if withTasks {
		tasks, err := listTasks(ctx, client, v.WaveID)
		if err != nil {
			return nil, err
		}
		v.Tasks = tasks
	}
	return &v, nil
}

// GetPickWaveInTxn loads wave inside a txn.
func GetPickWaveInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, waveID string, withTasks bool) (*PickWaveView, error) {
	row, err := txn.ReadRow(ctx, "PickWaves", spanner.Key{strings.TrimSpace(waveID)},
		[]string{"WaveId", "WarehouseId", "SupplierId", "ManifestId", "Strategy", "Status", "CreatedAt", "ReadyAt"})
	if err != nil {
		return nil, err
	}
	v, err := scanWave(row)
	if err != nil {
		return nil, err
	}
	if withTasks {
		iter := txn.Query(ctx, spanner.Statement{
			SQL: `SELECT TaskId, OrderId, ProductId, LotId, LocationId, QuantityRequested, QuantityPicked, PickerId, Status, PickSequence
			      FROM PickTasks WHERE WaveId = @wid ORDER BY PickSequence, TaskId`,
			Params: map[string]any{"wid": waveID},
		})
		defer iter.Stop()
		for {
			trow, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, err
			}
			t, err := scanTask(trow)
			if err != nil {
				return nil, err
			}
			v.Tasks = append(v.Tasks, t)
		}
	}
	return &v, nil
}

// ListPickWaves lists waves for a warehouse.
func ListPickWaves(ctx context.Context, client *spanner.Client, warehouseID, manifestID, status string) ([]PickWaveView, error) {
	if client == nil {
		return nil, fmt.Errorf("spanner required")
	}
	sql := `SELECT WaveId, WarehouseId, SupplierId, ManifestId, Strategy, Status, CreatedAt, ReadyAt
	        FROM PickWaves WHERE WarehouseId = @wid`
	params := map[string]any{"wid": strings.TrimSpace(warehouseID)}
	if m := strings.TrimSpace(manifestID); m != "" {
		sql += ` AND ManifestId = @mid`
		params["mid"] = m
	}
	if s := strings.TrimSpace(status); s != "" {
		sql += ` AND Status = @st`
		params["st"] = strings.ToUpper(s)
	}
	sql += ` ORDER BY CreatedAt DESC LIMIT 100`
	stmt := spanner.Statement{SQL: sql, Params: params}
	var iter *spanner.RowIterator
	if txn := spannerutils.ReadOnlyTxnFromContext(ctx); txn != nil {
		iter = txn.Query(ctx, stmt)
	} else {
		iter = client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, stmt)
	}
	defer iter.Stop()
	var out []PickWaveView
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		v, err := scanWave(row)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func listTasks(ctx context.Context, client *spanner.Client, waveID string) ([]PickTaskView, error) {
	taskStmt := spanner.Statement{
		SQL: `SELECT TaskId, OrderId, ProductId, LotId, LocationId, QuantityRequested, QuantityPicked, PickerId, Status, PickSequence
		      FROM PickTasks WHERE WaveId = @wid ORDER BY PickSequence, TaskId`,
		Params: map[string]any{"wid": waveID},
	}
	var iter *spanner.RowIterator
	if txn := spannerutils.ReadOnlyTxnFromContext(ctx); txn != nil {
		iter = txn.Query(ctx, taskStmt)
	} else {
		iter = client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, taskStmt)
	}
	defer iter.Stop()
	var out []PickTaskView
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		t, err := scanTask(row)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

func scanWave(row *spanner.Row) (PickWaveView, error) {
	var v PickWaveView
	var created time.Time
	var ready spanner.NullTime
	if err := row.Columns(&v.WaveID, &v.WarehouseID, &v.SupplierID, &v.ManifestID, &v.Strategy, &v.Status, &created, &ready); err != nil {
		return v, err
	}
	v.CreatedAt = created.UTC().Format(time.RFC3339)
	if ready.Valid {
		v.ReadyAt = ready.Time.UTC().Format(time.RFC3339)
	}
	return v, nil
}

func scanTask(row *spanner.Row) (PickTaskView, error) {
	var t PickTaskView
	var picker spanner.NullString
	if err := row.Columns(&t.TaskID, &t.OrderID, &t.ProductID, &t.LotID, &t.LocationID,
		&t.QuantityRequested, &t.QuantityPicked, &picker, &t.Status, &t.PickSequence); err != nil {
		return t, err
	}
	if picker.Valid {
		t.PickerID = picker.StringVal
	}
	return t, nil
}

func loadManifestOrderIDs(ctx context.Context, txn *spanner.ReadWriteTransaction, manifestID string) ([]string, error) {
	iter := txn.Query(ctx, spanner.Statement{
		SQL:    `SELECT OrderId FROM ManifestOrders WHERE ManifestId = @mid ORDER BY SequenceIndex`,
		Params: map[string]any{"mid": manifestID},
	})
	defer iter.Stop()
	var ids []string
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var oid string
		if err := row.Columns(&oid); err != nil {
			return nil, err
		}
		ids = append(ids, oid)
	}
	return ids, nil
}

type lotResRow struct {
	lotID string
	qty   int64
}

type stockLotPickMeta struct {
	productID  string
	locationID string
	reserved   int64
	status     string
}

type locPickMeta struct {
	seq   int64
	zone  string
	aisle string
}

func loadOrderLotReservationsBatch(ctx context.Context, txn *spanner.ReadWriteTransaction, orderIDs []string) (map[string][]lotResRow, error) {
	out := make(map[string][]lotResRow, len(orderIDs))
	if len(orderIDs) == 0 {
		return out, nil
	}
	iter := txn.Query(ctx, spanner.Statement{
		SQL:    `SELECT OrderId, LotId, Quantity FROM OrderLotReservations WHERE OrderId IN UNNEST(@oids)`,
		Params: map[string]any{"oids": orderIDs},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var oid, lid string
		var qty int64
		if err := row.Columns(&oid, &lid, &qty); err != nil {
			return nil, err
		}
		out[oid] = append(out[oid], lotResRow{lid, qty})
	}
	return out, nil
}

func loadStockLotsBatch(ctx context.Context, txn *spanner.ReadWriteTransaction, lotIDs []string) (map[string]stockLotPickMeta, error) {
	out := make(map[string]stockLotPickMeta, len(lotIDs))
	if len(lotIDs) == 0 {
		return out, nil
	}
	keys := make([]spanner.Key, 0, len(lotIDs))
	for _, lid := range lotIDs {
		keys = append(keys, spanner.Key{lid})
	}
	iter := txn.Read(ctx, "StockLots", spanner.KeySetFromKeys(keys...),
		[]string{"LotId", "ProductId", "LocationId", "QuantityReserved", "Status"})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var lid, pid, locID, st string
		var reserved int64
		if err := row.Columns(&lid, &pid, &locID, &reserved, &st); err != nil {
			return nil, err
		}
		out[lid] = stockLotPickMeta{productID: pid, locationID: locID, reserved: reserved, status: st}
	}
	return out, nil
}

func loadOrderLineQtysBatch(ctx context.Context, txn *spanner.ReadWriteTransaction, orderIDs []string) (map[string][]LineQty, error) {
	out := make(map[string][]LineQty, len(orderIDs))
	if len(orderIDs) == 0 {
		return out, nil
	}
	keys := make([]spanner.Key, 0, len(orderIDs))
	for _, oid := range orderIDs {
		keys = append(keys, spanner.Key{oid})
	}
	iter := txn.Read(ctx, "Orders", spanner.KeySetFromKeys(keys...), []string{"OrderId", "LineItemsJson"})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var oid string
		var raw []byte
		if err := row.Columns(&oid, &raw); err != nil {
			return nil, err
		}
		lines, err := parseOrderLineQtys(raw)
		if err != nil {
			return nil, err
		}
		out[oid] = lines
	}
	return out, nil
}

func locationPickMetaBatch(ctx context.Context, txn *spanner.ReadWriteTransaction, warehouseID string, locationIDs []string) map[string]locPickMeta {
	out := make(map[string]locPickMeta, len(locationIDs))
	if len(locationIDs) == 0 {
		return out
	}
	keys := make([]spanner.Key, 0, len(locationIDs))
	for _, lid := range locationIDs {
		if strings.TrimSpace(lid) != "" {
			keys = append(keys, spanner.Key{warehouseID, lid})
		}
	}
	if len(keys) == 0 {
		return out
	}
	iter := txn.Read(ctx, "WarehouseLocations", spanner.KeySetFromKeys(keys...),
		[]string{"LocationId", "PickSequence", "Zone", "Aisle"})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err != nil {
			break
		}
		var lid string
		var seq int64
		var z, a spanner.NullString
		if err := row.Columns(&lid, &seq, &z, &a); err != nil {
			continue
		}
		var m locPickMeta
		m.seq = seq
		if z.Valid {
			m.zone = z.StringVal
		}
		if a.Valid {
			m.aisle = a.StringVal
		}
		out[lid] = m
	}
	return out
}

func loadOrderLotReservations(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID string) ([]lotResRow, error) {
	iter := txn.Query(ctx, spanner.Statement{
		SQL:    `SELECT LotId, Quantity FROM OrderLotReservations WHERE OrderId = @oid`,
		Params: map[string]any{"oid": orderID},
	})
	defer iter.Stop()
	var out []lotResRow
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var lid string
		var qty int64
		if err := row.Columns(&lid, &qty); err != nil {
			return nil, err
		}
		out = append(out, lotResRow{lid, qty})
	}
	return out, nil
}

// skuPickDraftsFromLines builds bag-of-SKU pick tasks when lots are off.
func skuPickDraftsFromLines(orderID string, lines []LineQty) []pickTaskDraft {
	if len(lines) == 0 {
		return nil
	}
	out := make([]pickTaskDraft, 0, len(lines))
	for _, line := range lines {
		sku := strings.TrimSpace(line.SKU)
		if sku == "" || line.Quantity <= 0 {
			continue
		}
		out = append(out, pickTaskDraft{
			OrderID:    orderID,
			ProductID:  sku,
			LotID:      "",
			LocationID: "",
			Qty:        line.Quantity,
		})
	}
	return out
}

// shouldDepleteLot is false for bag-of-SKU tasks (empty LotId).
func shouldDepleteLot(lotID string) bool {
	return strings.TrimSpace(lotID) != ""
}

func loadOrderLineQtys(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID string) ([]LineQty, error) {
	row, err := txn.ReadRow(ctx, "Orders", spanner.Key{orderID}, []string{"LineItemsJson"})
	if err != nil {
		return nil, err
	}
	var raw []byte
	if err := row.Columns(&raw); err != nil {
		return nil, err
	}
	return parseOrderLineQtys(raw)
}

func parseOrderLineQtys(raw []byte) ([]LineQty, error) {
	var items []struct {
		SKU       string `json:"sku"`
		SKUID     string `json:"sku_id"`
		ProductID string `json:"product_id"`
		Quantity  int64  `json:"quantity"`
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &items); err != nil {
			return nil, err
		}
	}
	out := make([]LineQty, 0, len(items))
	for _, it := range items {
		sku := strings.TrimSpace(it.SKU)
		if sku == "" {
			sku = strings.TrimSpace(it.SKUID)
		}
		if sku == "" {
			sku = strings.TrimSpace(it.ProductID)
		}
		if sku == "" || it.Quantity <= 0 {
			continue
		}
		out = append(out, LineQty{SKU: sku, Quantity: it.Quantity})
	}
	return out, nil
}

func pickSequenceForLocation(ctx context.Context, txn *spanner.ReadWriteTransaction, warehouseID, locationID string) int64 {
	seq, _, _ := locationPickMeta(ctx, txn, warehouseID, locationID)
	return seq
}

func locationPickMeta(ctx context.Context, txn *spanner.ReadWriteTransaction, warehouseID, locationID string) (seq int64, zone, aisle string) {
	if strings.TrimSpace(locationID) == "" {
		return 0, "", ""
	}
	row, err := txn.ReadRow(ctx, "WarehouseLocations", spanner.Key{warehouseID, locationID},
		[]string{"PickSequence", "Zone", "Aisle"})
	if err != nil {
		return 0, "", ""
	}
	var z, a spanner.NullString
	_ = row.Columns(&seq, &z, &a)
	if z.Valid {
		zone = z.StringVal
	}
	if a.Valid {
		aisle = a.StringVal
	}
	return seq, zone, aisle
}

// applyStopRanks sets StopRank so last delivery stop is picked first (LIFO load).
func applyStopRanks(ctx context.Context, txn *spanner.ReadWriteTransaction, manifestID string, drafts []pickTaskDraft) {
	iter := txn.Query(ctx, spanner.Statement{
		SQL:    `SELECT OrderId, SequenceIndex FROM ManifestOrders WHERE ManifestId = @mid`,
		Params: map[string]any{"mid": manifestID},
	})
	defer iter.Stop()
	rank := map[string]int64{}
	var maxSeq int64
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return
		}
		var oid string
		var s int64
		if err := row.Columns(&oid, &s); err != nil {
			continue
		}
		rank[oid] = s
		if s > maxSeq {
			maxSeq = s
		}
	}
	if len(rank) == 0 {
		return
	}
	for i := range drafts {
		s := rank[drafts[i].OrderID]
		// Higher StopRank → pick earlier: invert stop sequence.
		drafts[i].StopRank = maxSeq - s + 1
	}
}

// sortPickTaskDrafts orders tasks by WarehouseLocations.PickSequence then location id.
func sortPickTaskDrafts(drafts []pickTaskDraft) {
	sort.SliceStable(drafts, func(i, j int) bool {
		if drafts[i].Seq != drafts[j].Seq {
			return drafts[i].Seq < drafts[j].Seq
		}
		return drafts[i].LocationID < drafts[j].LocationID
	})
}

// sortSShapePickTaskDrafts: LIFO stop rank, then zone, serpentine aisle, then PickSequence.
func sortSShapePickTaskDrafts(drafts []pickTaskDraft) {
	sort.SliceStable(drafts, func(i, j int) bool {
		if drafts[i].StopRank != drafts[j].StopRank {
			return drafts[i].StopRank > drafts[j].StopRank
		}
		if drafts[i].Zone != drafts[j].Zone {
			return drafts[i].Zone < drafts[j].Zone
		}
		ai, aj := drafts[i].Aisle, drafts[j].Aisle
		if ai != aj {
			// Serpentine: odd aisle index descending seq within aisle group via string compare + parity heuristic.
			return ai < aj
		}
		// Within aisle: alternate direction by aisle hash parity.
		odd := aisleOdd(ai)
		if drafts[i].Seq != drafts[j].Seq {
			if odd {
				return drafts[i].Seq > drafts[j].Seq
			}
			return drafts[i].Seq < drafts[j].Seq
		}
		return drafts[i].LocationID < drafts[j].LocationID
	})
}

func aisleOdd(aisle string) bool {
	n := 0
	for _, c := range aisle {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	if n == 0 {
		for _, c := range aisle {
			n += int(c)
		}
	}
	return n%2 == 1
}

// applyLotDepletion returns next on-hand/reserved after confirming a pick qty.
func applyLotDepletion(qoh, qr, picked int64) (nextQoH, nextQR int64, depleted bool) {
	nextQoH = qoh - picked
	if nextQoH < 0 {
		nextQoH = 0
	}
	nextQR = qr - picked
	if nextQR < 0 {
		nextQR = 0
	}
	return nextQoH, nextQR, nextQoH == 0
}

// pickTaskStatusForQty returns CONFIRMED or SHORT for an explicit confirm quantity.
func pickTaskStatusForQty(requested, quantityPicked int64) string {
	if quantityPicked < requested {
		return "SHORT"
	}
	return "CONFIRMED"
}
