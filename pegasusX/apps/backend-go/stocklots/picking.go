package stocklots

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

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

	var drafts []pickTaskDraft
	for _, oid := range orderIDs {
		resRows, err := loadOrderLotReservations(ctx, txn, oid)
		if err != nil {
			return nil, err
		}
		if len(resRows) > 0 {
			for _, rr := range resRows {
				lot, err := txn.ReadRow(ctx, "StockLots", spanner.Key{rr.lotID},
					[]string{"ProductId", "LocationId", "QuantityReserved", "Status"})
				if err != nil {
					if spanner.ErrCode(err) == 5 {
						continue
					}
					return nil, err
				}
				var pid, lid, st string
				var reserved int64
				if err := lot.Columns(&pid, &lid, &reserved, &st); err != nil {
					return nil, err
				}
				if st != "AVAILABLE" && st != "QUARANTINE" {
					continue
				}
				qty := rr.qty
				if qty > reserved && reserved > 0 {
					qty = reserved
				}
				if qty <= 0 {
					continue
				}
				seq, zone, aisle := locationPickMeta(ctx, txn, warehouseID, lid)
				drafts = append(drafts, pickTaskDraft{
					OrderID: oid, ProductID: pid, LotID: rr.lotID, LocationID: lid, Qty: qty, Seq: seq,
					Zone: zone, Aisle: aisle,
				})
			}
			continue
		}
		lines, err := loadOrderLineQtys(ctx, txn, oid)
		if err != nil {
			return nil, err
		}
		// Lots off: bag-of-SKU tasks. Confirm does not deplete StockLots.
		if !EffectiveLots(ctx, warehouseID, supplierID) {
			drafts = append(drafts, skuPickDraftsFromLines(oid, lines)...)
			continue
		}
		// Lots on: FEFO-allocate from available lots for line items.
		for _, line := range lines {
			cands, err := loadAvailableLots(ctx, txn, supplierID, warehouseID, line.SKU)
			if err != nil {
				return nil, err
			}
			sort.SliceStable(cands, func(i, j int) bool {
				if cands[i].Expiry.Valid && cands[j].Expiry.Valid {
					ti := cands[i].Expiry.Date.In(time.UTC)
					tj := cands[j].Expiry.Date.In(time.UTC)
					if !ti.Equal(tj) {
						return ti.Before(tj)
					}
				}
				return cands[i].ReceivedAt.Before(cands[j].ReceivedAt)
			})
			need := line.Quantity
			for _, c := range cands {
				if need <= 0 {
					break
				}
				take := c.Available
				if take > need {
					take = need
				}
				if take <= 0 {
					continue
				}
				// Reserve for wave (same as order reserve) so confirm can deplete.
				if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("StockLots", map[string]any{
					"LotId":            c.LotID,
					"QuantityReserved": c.Reserved + take,
					"UpdatedAt":        spanner.CommitTimestamp,
				})}); err != nil {
					return nil, err
				}
				if err := txn.BufferWrite([]*spanner.Mutation{spanner.InsertOrUpdateMap("OrderLotReservations", map[string]any{
					"OrderId":   oid,
					"LotId":     c.LotID,
					"Quantity":  take,
					"CreatedAt": spanner.CommitTimestamp,
				})}); err != nil {
					return nil, err
				}
				lid := ""
				if locRow, err := txn.ReadRow(ctx, "StockLots", spanner.Key{c.LotID}, []string{"LocationId"}); err == nil {
					_ = locRow.Columns(&lid)
				}
				seq, zone, aisle := locationPickMeta(ctx, txn, warehouseID, lid)
				drafts = append(drafts, pickTaskDraft{
					OrderID: oid, ProductID: line.SKU, LotID: c.LotID, LocationID: lid, Qty: take, Seq: seq,
					Zone: zone, Aisle: aisle,
				})
				need -= take
				c.Reserved += take
				c.Available -= take
			}
			if need > 0 {
				return nil, fmt.Errorf("%w: sku %s short %d for order %s", ErrInventoryExhausted, line.SKU, need, oid)
			}
			if err := RollupInventoryV2InTxn(ctx, txn, supplierID, warehouseID, line.SKU); err != nil {
				return nil, err
			}
		}
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

// ConfirmPickTaskInTxn confirms a task qty and may advance wave to READY_TO_SEAL.
func ConfirmPickTaskInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, waveID, taskID, pickerID string, quantityPicked int64) (*PickWaveView, error) {
	waveID = strings.TrimSpace(waveID)
	taskID = strings.TrimSpace(taskID)
	taskRow, err := txn.ReadRow(ctx, "PickTasks", spanner.Key{waveID, taskID},
		[]string{"OrderId", "ProductId", "LotId", "LocationId", "QuantityRequested", "QuantityPicked", "Status", "PickSequence"})
	if err != nil {
		return nil, err
	}
	var orderID, productID, lotID, locationID, status string
	var requested, picked, seq int64
	if err := taskRow.Columns(&orderID, &productID, &lotID, &locationID, &requested, &picked, &status, &seq); err != nil {
		return nil, err
	}
	status = strings.ToUpper(strings.TrimSpace(status))
	if status == "CONFIRMED" || status == "SHORT_WAIVED" {
		return GetPickWaveInTxn(ctx, txn, waveID, true)
	}
	if quantityPicked <= 0 {
		quantityPicked = requested
	}
	newStatus := "CONFIRMED"
	if quantityPicked < requested {
		newStatus = "SHORT"
	}
	if quantityPicked > requested {
		quantityPicked = requested
	}

	if shouldDepleteLot(lotID) {
		lotRow, err := txn.ReadRow(ctx, "StockLots", spanner.Key{lotID},
			[]string{"SupplierId", "WarehouseId", "ProductId", "QuantityOnHand", "QuantityReserved", "Status"})
		if err != nil {
			return nil, err
		}
		var sid, wid, pid, lotStatus string
		var qoh, qr int64
		if err := lotRow.Columns(&sid, &wid, &pid, &qoh, &qr, &lotStatus); err != nil {
			return nil, err
		}
		nextQoH, nextQR, depleted := applyLotDepletion(qoh, qr, quantityPicked)
		lotStatusOut := lotStatus
		if depleted {
			lotStatusOut = "DEPLETED"
		}
		if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("StockLots", map[string]any{
			"LotId":            lotID,
			"QuantityOnHand":   nextQoH,
			"QuantityReserved": nextQR,
			"Status":           lotStatusOut,
			"UpdatedAt":        spanner.CommitTimestamp,
		})}); err != nil {
			return nil, err
		}
		if err := RollupInventoryV2InTxn(ctx, txn, sid, wid, pid); err != nil {
			return nil, err
		}
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
		return nil, err
	}

	waveRow, err := txn.ReadRow(ctx, "PickWaves", spanner.Key{waveID}, []string{"Status"})
	if err != nil {
		return nil, err
	}
	var waveStatus string
	_ = waveRow.Columns(&waveStatus)
	if strings.EqualFold(waveStatus, "OPEN") {
		if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("PickWaves", map[string]any{
			"WaveId":    waveID,
			"Status":    "PICKING",
			"UpdatedAt": spanner.CommitTimestamp,
		})}); err != nil {
			return nil, err
		}
	}
	if err := maybeMarkWaveReadyInTxn(ctx, txn, waveID, taskID, newStatus); err != nil {
		return nil, err
	}
	view, err := GetPickWaveInTxn(ctx, txn, waveID, true)
	if err != nil {
		return nil, err
	}
	if view != nil {
		for i := range view.Tasks {
			if view.Tasks[i].TaskID == taskID {
				view.Tasks[i].Status = newStatus
				view.Tasks[i].QuantityPicked = quantityPicked
			}
		}
		if waveReadyFromTasks(view.Tasks) {
			view.Status = "READY_TO_SEAL"
		} else if strings.EqualFold(view.Status, "OPEN") {
			view.Status = "PICKING"
		}
	}
	return view, nil
}

// WaiveShortsInTxn marks SHORT tasks as SHORT_WAIVED and advances to READY_TO_SEAL when eligible.
func WaiveShortsInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, waveID string) (*PickWaveView, error) {
	waveID = strings.TrimSpace(waveID)
	iter := txn.Query(ctx, spanner.Statement{
		SQL:    `SELECT TaskId FROM PickTasks WHERE WaveId = @wid AND Status = 'SHORT'`,
		Params: map[string]any{"wid": waveID},
	})
	defer iter.Stop()
	var taskIDs []string
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var tid string
		if err := row.Columns(&tid); err != nil {
			return nil, err
		}
		taskIDs = append(taskIDs, tid)
	}
	for _, tid := range taskIDs {
		if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("PickTasks", map[string]any{
			"WaveId":    waveID,
			"TaskId":    tid,
			"Status":    "SHORT_WAIVED",
			"UpdatedAt": spanner.CommitTimestamp,
		})}); err != nil {
			return nil, err
		}
	}
	if err := maybeMarkWaveReadyInTxn(ctx, txn, waveID, "", ""); err != nil {
		return nil, err
	}
	return GetPickWaveInTxn(ctx, txn, waveID, true)
}

func maybeMarkWaveReadyInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, waveID, overrideTaskID, overrideStatus string) error {
	iter := txn.Query(ctx, spanner.Statement{
		SQL:    `SELECT TaskId, Status FROM PickTasks@{FORCE_INDEX=_BASE_TABLE} WHERE WaveId = @wid`,
		Params: map[string]any{"wid": waveID},
	})
	defer iter.Stop()
	var tasks []PickTaskView
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		var tid, st string
		if err := row.Columns(&tid, &st); err != nil {
			return err
		}
		if overrideTaskID != "" && tid == overrideTaskID && strings.TrimSpace(overrideStatus) != "" {
			st = overrideStatus
		}
		tasks = append(tasks, PickTaskView{TaskID: tid, Status: st})
	}
	if !waveReadyFromTasks(tasks) {
		return nil
	}
	return txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("PickWaves", map[string]any{
		"WaveId":    waveID,
		"Status":    "READY_TO_SEAL",
		"ReadyAt":   time.Now().UTC(),
		"UpdatedAt": spanner.CommitTimestamp,
	})})
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
	iter := client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
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
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT TaskId, OrderId, ProductId, LotId, LocationId, QuantityRequested, QuantityPicked, PickerId, Status, PickSequence
		      FROM PickTasks WHERE WaveId = @wid ORDER BY PickSequence, TaskId`,
		Params: map[string]any{"wid": waveID},
	})
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

// pickTaskStatusForQty returns CONFIRMED or SHORT for a confirm body.
func pickTaskStatusForQty(requested, quantityPicked int64) string {
	if quantityPicked <= 0 {
		quantityPicked = requested
	}
	if quantityPicked < requested {
		return "SHORT"
	}
	return "CONFIRMED"
}
