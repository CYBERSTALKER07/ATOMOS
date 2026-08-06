package stocklots

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

// CycleCountView is the API DTO for a cycle count.
type CycleCountView struct {
	CountID     string `json:"count_id"`
	WarehouseID string `json:"warehouse_id"`
	LocationID  string `json:"location_id"`
	ProductID   string `json:"product_id"`
	LotID       string `json:"lot_id,omitempty"`
	ExpectedQty int64  `json:"expected_qty"`
	CountedQty  *int64 `json:"counted_qty,omitempty"`
	VarianceQty *int64 `json:"variance_qty,omitempty"`
	ReasonCode  string `json:"reason_code,omitempty"`
	Status      string `json:"status"`
	CountedBy   string `json:"counted_by,omitempty"`
	CountedAt   string `json:"counted_at,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
}

// InventoryAdjustmentView is the API DTO for an inventory adjustment stub.
type InventoryAdjustmentView struct {
	AdjustmentID string `json:"adjustment_id"`
	WarehouseID  string `json:"warehouse_id"`
	ProductID    string `json:"product_id"`
	LotID        string `json:"lot_id,omitempty"`
	CountID      string `json:"count_id,omitempty"`
	DeltaQty     int64  `json:"delta_qty"`
	ReasonCode   string `json:"reason_code,omitempty"`
	Status       string `json:"status"`
	ActorID      string `json:"actor_id,omitempty"`
	ApprovedBy   string `json:"approved_by,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
}

// CreateCycleCountRequest is the body for creating an OPEN count.
type CreateCycleCountRequest struct {
	LocationID  string
	ProductID   string
	LotID       string
	ExpectedQty *int64 // nil → sum AVAILABLE lot QoH at location+product
}

// varianceQty returns counted − expected.
func varianceQty(expected, counted int64) int64 {
	return counted - expected
}

// CreateCycleCountInTxn inserts an OPEN cycle count.
func CreateCycleCountInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, warehouseID string, req CreateCycleCountRequest) (*CycleCountView, error) {
	warehouseID = strings.TrimSpace(warehouseID)
	loc := strings.TrimSpace(req.LocationID)
	pid := strings.TrimSpace(req.ProductID)
	if warehouseID == "" || loc == "" || pid == "" {
		return nil, fmt.Errorf("warehouse_id, location_id, product_id required")
	}
	expected := int64(0)
	if req.ExpectedQty != nil {
		expected = *req.ExpectedQty
	} else {
		sum, err := sumAvailableQoH(ctx, txn, warehouseID, loc, pid, strings.TrimSpace(req.LotID))
		if err != nil {
			return nil, err
		}
		expected = sum
	}
	countID := uuid.NewString()
	cols := map[string]any{
		"CountId":     countID,
		"WarehouseId": warehouseID,
		"LocationId":  loc,
		"ProductId":   pid,
		"ExpectedQty": expected,
		"Status":      "OPEN",
		"CreatedAt":   spanner.CommitTimestamp,
		"UpdatedAt":   spanner.CommitTimestamp,
	}
	lotID := strings.TrimSpace(req.LotID)
	if lotID != "" {
		cols["LotId"] = lotID
	}
	if err := txn.BufferWrite([]*spanner.Mutation{spanner.InsertMap("CycleCounts", cols)}); err != nil {
		return nil, err
	}
	return &CycleCountView{
		CountID: countID, WarehouseID: warehouseID, LocationID: loc, ProductID: pid,
		LotID: lotID, ExpectedQty: expected, Status: "OPEN",
	}, nil
}

func sumAvailableQoH(ctx context.Context, txn *spanner.ReadWriteTransaction, warehouseID, locationID, productID, lotID string) (int64, error) {
	sql := `SELECT COALESCE(SUM(QuantityOnHand), 0) FROM StockLots
	        WHERE WarehouseId = @wid AND LocationId = @lid AND ProductId = @pid AND Status = 'AVAILABLE'`
	params := map[string]any{"wid": warehouseID, "lid": locationID, "pid": productID}
	if lotID != "" {
		sql += ` AND LotId = @lot`
		params["lot"] = lotID
	}
	iter := txn.Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var sum int64
	if err := row.Columns(&sum); err != nil {
		return 0, err
	}
	return sum, nil
}

// SubmitCycleCountInTxn records counted qty, variance, and a PENDING adjustment when variance ≠ 0.
func SubmitCycleCountInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, countID, actorID string, countedQty int64, reasonCode string) (*CycleCountView, error) {
	countID = strings.TrimSpace(countID)
	row, err := txn.ReadRow(ctx, "CycleCounts", spanner.Key{countID},
		[]string{"WarehouseId", "LocationId", "ProductId", "LotId", "ExpectedQty", "Status"})
	if err != nil {
		return nil, err
	}
	var wid, loc, pid, status string
	var lot spanner.NullString
	var expected int64
	if err := row.Columns(&wid, &loc, &pid, &lot, &expected, &status); err != nil {
		return nil, err
	}
	status = strings.ToUpper(strings.TrimSpace(status))
	if status != "OPEN" {
		return nil, fmt.Errorf("count_not_open")
	}
	variance := varianceQty(expected, countedQty)
	now := time.Now().UTC()
	cols := map[string]any{
		"CountId":     countID,
		"CountedQty":  countedQty,
		"VarianceQty": variance,
		"Status":      "SUBMITTED",
		"CountedAt":   now,
		"UpdatedAt":   spanner.CommitTimestamp,
	}
	if strings.TrimSpace(actorID) != "" {
		cols["CountedBy"] = strings.TrimSpace(actorID)
	}
	if rc := strings.TrimSpace(reasonCode); rc != "" {
		cols["ReasonCode"] = rc
	}
	if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("CycleCounts", cols)}); err != nil {
		return nil, err
	}
	if variance != 0 {
		adjID := uuid.NewString()
		adj := map[string]any{
			"AdjustmentId": adjID,
			"WarehouseId":  wid,
			"ProductId":    pid,
			"CountId":      countID,
			"DeltaQty":     variance,
			"Status":       "PENDING",
			"CreatedAt":    spanner.CommitTimestamp,
			"UpdatedAt":    spanner.CommitTimestamp,
		}
		if lot.Valid && strings.TrimSpace(lot.StringVal) != "" {
			adj["LotId"] = strings.TrimSpace(lot.StringVal)
		}
		if strings.TrimSpace(actorID) != "" {
			adj["ActorId"] = strings.TrimSpace(actorID)
		}
		if rc := strings.TrimSpace(reasonCode); rc != "" {
			adj["ReasonCode"] = rc
		} else {
			adj["ReasonCode"] = "CYCLE_COUNT_VARIANCE"
		}
		if err := txn.BufferWrite([]*spanner.Mutation{spanner.InsertMap("InventoryAdjustments", adj)}); err != nil {
			return nil, err
		}
	}
	return GetCycleCountInTxn(ctx, txn, countID)
}

// ApproveAdjustmentInTxn marks PENDING → APPROVED and applies DeltaQty to StockLots + V2 roll-up.
func ApproveAdjustmentInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, adjustmentID, approvedBy string) (*InventoryAdjustmentView, error) {
	adjustmentID = strings.TrimSpace(adjustmentID)
	row, err := txn.ReadRow(ctx, "InventoryAdjustments", spanner.Key{adjustmentID},
		[]string{"WarehouseId", "ProductId", "LotId", "CountId", "DeltaQty", "ReasonCode", "Status", "ActorId"})
	if err != nil {
		return nil, err
	}
	var wid, pid, status string
	var lot, countID, reason, actor spanner.NullString
	var delta int64
	if err := row.Columns(&wid, &pid, &lot, &countID, &delta, &reason, &status, &actor); err != nil {
		return nil, err
	}
	if !strings.EqualFold(strings.TrimSpace(status), "PENDING") {
		return nil, fmt.Errorf("adjustment_not_pending")
	}
	supplierID, err := applyAdjustmentDeltaInTxn(ctx, txn, wid, pid, lot, delta)
	if err != nil {
		return nil, err
	}
	if err := RollupInventoryV2InTxn(ctx, txn, supplierID, wid, pid); err != nil {
		return nil, err
	}
	cols := map[string]any{
		"AdjustmentId": adjustmentID,
		"Status":       "APPROVED",
		"UpdatedAt":    spanner.CommitTimestamp,
	}
	if ab := strings.TrimSpace(approvedBy); ab != "" {
		cols["ApprovedBy"] = ab
	}
	if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("InventoryAdjustments", cols)}); err != nil {
		return nil, err
	}
	view := &InventoryAdjustmentView{
		AdjustmentID: adjustmentID, WarehouseID: wid, ProductID: pid,
		DeltaQty: delta, Status: "APPROVED", ApprovedBy: strings.TrimSpace(approvedBy),
	}
	if lot.Valid {
		view.LotID = lot.StringVal
	}
	if countID.Valid {
		view.CountID = countID.StringVal
	}
	if reason.Valid {
		view.ReasonCode = reason.StringVal
	}
	if actor.Valid {
		view.ActorID = actor.StringVal
	}
	return view, nil
}

// applyAdjustmentDeltaInTxn credits/debits lot QoH by delta (counted − expected).
func applyAdjustmentDeltaInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, warehouseID, productID string, lot spanner.NullString, delta int64) (supplierID string, err error) {
	if delta == 0 {
		if lot.Valid && strings.TrimSpace(lot.StringVal) != "" {
			row, err := txn.ReadRow(ctx, "StockLots", spanner.Key{strings.TrimSpace(lot.StringVal)}, []string{"SupplierId"})
			if err != nil {
				return "", err
			}
			_ = row.Columns(&supplierID)
			return supplierID, nil
		}
		return "", fmt.Errorf("zero_delta_no_lot")
	}
	lotID := ""
	if lot.Valid {
		lotID = strings.TrimSpace(lot.StringVal)
	}
	if lotID == "" {
		// Pick any AVAILABLE lot for the product in the warehouse.
		iter := txn.Query(ctx, spanner.Statement{
			SQL: `SELECT LotId, SupplierId, QuantityOnHand, Status FROM StockLots
			      WHERE WarehouseId = @wid AND ProductId = @pid AND Status = 'AVAILABLE'
			      ORDER BY ExpiryDate ASC NULLS LAST, ReceivedAt ASC LIMIT 1`,
			Params: map[string]any{"wid": warehouseID, "pid": productID},
		})
		defer iter.Stop()
		row, err := iter.Next()
		if err == iterator.Done {
			return "", fmt.Errorf("no_available_lot_for_adjustment")
		}
		if err != nil {
			return "", err
		}
		var qoh int64
		var st string
		if err := row.Columns(&lotID, &supplierID, &qoh, &st); err != nil {
			return "", err
		}
		return applyLotDelta(ctx, txn, lotID, supplierID, qoh, delta)
	}
	row, err := txn.ReadRow(ctx, "StockLots", spanner.Key{lotID},
		[]string{"SupplierId", "QuantityOnHand", "Status"})
	if err != nil {
		return "", err
	}
	var qoh int64
	var st string
	if err := row.Columns(&supplierID, &qoh, &st); err != nil {
		return "", err
	}
	return applyLotDelta(ctx, txn, lotID, supplierID, qoh, delta)
}

func applyLotDelta(ctx context.Context, txn *spanner.ReadWriteTransaction, lotID, supplierID string, qoh, delta int64) (string, error) {
	_ = ctx
	next := qoh + delta
	status := "AVAILABLE"
	if next < 0 {
		next = 0
	}
	if next == 0 {
		status = "DEPLETED"
	}
	if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("StockLots", map[string]any{
		"LotId":          lotID,
		"QuantityOnHand": next,
		"Status":         status,
		"UpdatedAt":      spanner.CommitTimestamp,
	})}); err != nil {
		return "", err
	}
	return supplierID, nil
}

// RejectAdjustmentInTxn marks PENDING → REJECTED without lot mutation.
func RejectAdjustmentInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, adjustmentID, actorID string) (*InventoryAdjustmentView, error) {
	adjustmentID = strings.TrimSpace(adjustmentID)
	row, err := txn.ReadRow(ctx, "InventoryAdjustments", spanner.Key{adjustmentID},
		[]string{"WarehouseId", "ProductId", "LotId", "CountId", "DeltaQty", "ReasonCode", "Status", "ActorId"})
	if err != nil {
		return nil, err
	}
	var wid, pid, status string
	var lot, countID, reason, actor spanner.NullString
	var delta int64
	if err := row.Columns(&wid, &pid, &lot, &countID, &delta, &reason, &status, &actor); err != nil {
		return nil, err
	}
	if !strings.EqualFold(strings.TrimSpace(status), "PENDING") {
		return nil, fmt.Errorf("adjustment_not_pending")
	}
	cols := map[string]any{
		"AdjustmentId": adjustmentID,
		"Status":       "REJECTED",
		"UpdatedAt":    spanner.CommitTimestamp,
	}
	if ab := strings.TrimSpace(actorID); ab != "" {
		cols["ApprovedBy"] = ab
	}
	if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("InventoryAdjustments", cols)}); err != nil {
		return nil, err
	}
	view := &InventoryAdjustmentView{
		AdjustmentID: adjustmentID, WarehouseID: wid, ProductID: pid,
		DeltaQty: delta, Status: "REJECTED", ApprovedBy: strings.TrimSpace(actorID),
	}
	if lot.Valid {
		view.LotID = lot.StringVal
	}
	if countID.Valid {
		view.CountID = countID.StringVal
	}
	if reason.Valid {
		view.ReasonCode = reason.StringVal
	}
	return view, nil
}

// InventoryAccuracyKPI is 1 − Σ|variance| / Σexpected over SUBMITTED counts.
type InventoryAccuracyKPI struct {
	WarehouseID       string  `json:"warehouse_id"`
	InventoryAccuracy float64 `json:"inventory_accuracy"`
	SubmittedCounts   int64   `json:"submitted_counts"`
	SumExpected       int64   `json:"sum_expected"`
	SumAbsVariance    int64   `json:"sum_abs_variance"`
}

// ComputeInventoryAccuracy returns the ops KPI for a warehouse.
func ComputeInventoryAccuracy(ctx context.Context, client *spanner.Client, warehouseID string) (*InventoryAccuracyKPI, error) {
	if client == nil {
		return nil, fmt.Errorf("spanner unavailable")
	}
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT ExpectedQty, VarianceQty FROM CycleCounts
		      WHERE WarehouseId = @wid AND Status = 'SUBMITTED'`,
		Params: map[string]any{"wid": warehouseID},
	})
	defer iter.Stop()
	var sumExp, sumAbs, n int64
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var exp int64
		var v spanner.NullInt64
		if err := row.Columns(&exp, &v); err != nil {
			return nil, err
		}
		n++
		sumExp += exp
		if v.Valid {
			d := v.Int64
			if d < 0 {
				d = -d
			}
			sumAbs += d
		}
	}
	acc := 1.0
	if sumExp > 0 {
		acc = 1.0 - float64(sumAbs)/float64(sumExp)
		if acc < 0 {
			acc = 0
		}
	}
	return &InventoryAccuracyKPI{
		WarehouseID: warehouseID, InventoryAccuracy: acc,
		SubmittedCounts: n, SumExpected: sumExp, SumAbsVariance: sumAbs,
	}, nil
}

// EnqueueABCCountsInTxn creates OPEN counts for top-A SKUs by on-hand (simple ABC stub: top 20%).
func EnqueueABCCountsInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, warehouseID, supplierID string, limit int) ([]CycleCountView, error) {
	warehouseID = strings.TrimSpace(warehouseID)
	if warehouseID == "" {
		return nil, fmt.Errorf("warehouse_id required")
	}
	if limit <= 0 {
		limit = 20
	}
	iter := txn.Query(ctx, spanner.Statement{
		SQL: `SELECT ProductId, LocationId, SUM(QuantityOnHand) AS q
		      FROM StockLots
		      WHERE WarehouseId = @wid AND Status = 'AVAILABLE' AND QuantityOnHand > 0
		      GROUP BY ProductId, LocationId
		      ORDER BY q DESC
		      LIMIT @lim`,
		Params: map[string]any{"wid": warehouseID, "lim": int64(limit)},
	})
	defer iter.Stop()
	var created []CycleCountView
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var pid, lid string
		var q int64
		if err := row.Columns(&pid, &lid, &q); err != nil {
			return nil, err
		}
		exp := q
		v, err := CreateCycleCountInTxn(ctx, txn, warehouseID, CreateCycleCountRequest{
			LocationID: lid, ProductID: pid, ExpectedQty: &exp,
		})
		if err != nil {
			return nil, err
		}
		created = append(created, *v)
	}
	_ = supplierID
	if created == nil {
		created = []CycleCountView{}
	}
	return created, nil
}

// ListCycleCounts returns counts for a warehouse, optional status filter.
func ListCycleCounts(ctx context.Context, client *spanner.Client, warehouseID, status string) ([]CycleCountView, error) {
	if client == nil {
		return nil, fmt.Errorf("spanner unavailable")
	}
	sql := `SELECT CountId, WarehouseId, LocationId, ProductId, LotId, ExpectedQty, CountedQty, VarianceQty,
	               ReasonCode, Status, CountedBy, CountedAt, CreatedAt
	        FROM CycleCounts WHERE WarehouseId = @wid`
	params := map[string]any{"wid": warehouseID}
	if s := strings.TrimSpace(status); s != "" {
		sql += ` AND Status = @st`
		params["st"] = strings.ToUpper(s)
	}
	sql += ` ORDER BY CreatedAt DESC LIMIT 200`
	iter := client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	var out []CycleCountView
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		v, err := scanCycleCount(row)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	if out == nil {
		out = []CycleCountView{}
	}
	return out, nil
}

// GetCycleCount loads one count.
func GetCycleCount(ctx context.Context, client *spanner.Client, countID string) (*CycleCountView, error) {
	if client == nil {
		return nil, fmt.Errorf("spanner unavailable")
	}
	row, err := client.Single().ReadRow(ctx, "CycleCounts", spanner.Key{strings.TrimSpace(countID)},
		[]string{"CountId", "WarehouseId", "LocationId", "ProductId", "LotId", "ExpectedQty", "CountedQty",
			"VarianceQty", "ReasonCode", "Status", "CountedBy", "CountedAt", "CreatedAt"})
	if err != nil {
		return nil, err
	}
	return scanCycleCount(row)
}

func GetCycleCountInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, countID string) (*CycleCountView, error) {
	row, err := txn.ReadRow(ctx, "CycleCounts", spanner.Key{strings.TrimSpace(countID)},
		[]string{"CountId", "WarehouseId", "LocationId", "ProductId", "LotId", "ExpectedQty", "CountedQty",
			"VarianceQty", "ReasonCode", "Status", "CountedBy", "CountedAt", "CreatedAt"})
	if err != nil {
		return nil, err
	}
	return scanCycleCount(row)
}

func scanCycleCount(row *spanner.Row) (*CycleCountView, error) {
	var v CycleCountView
	var lot, reason, countedBy spanner.NullString
	var counted, variance spanner.NullInt64
	var countedAt, created spanner.NullTime
	if err := row.Columns(&v.CountID, &v.WarehouseID, &v.LocationID, &v.ProductID, &lot,
		&v.ExpectedQty, &counted, &variance, &reason, &v.Status, &countedBy, &countedAt, &created); err != nil {
		return nil, err
	}
	if lot.Valid {
		v.LotID = lot.StringVal
	}
	if counted.Valid {
		q := counted.Int64
		v.CountedQty = &q
	}
	if variance.Valid {
		q := variance.Int64
		v.VarianceQty = &q
	}
	if reason.Valid {
		v.ReasonCode = reason.StringVal
	}
	if countedBy.Valid {
		v.CountedBy = countedBy.StringVal
	}
	if countedAt.Valid {
		v.CountedAt = countedAt.Time.UTC().Format(time.RFC3339Nano)
	}
	if created.Valid {
		v.CreatedAt = created.Time.UTC().Format(time.RFC3339Nano)
	}
	return &v, nil
}

// ListInventoryAdjustments lists adjustments for a warehouse.
func ListInventoryAdjustments(ctx context.Context, client *spanner.Client, warehouseID, status string) ([]InventoryAdjustmentView, error) {
	if client == nil {
		return nil, fmt.Errorf("spanner unavailable")
	}
	sql := `SELECT AdjustmentId, WarehouseId, ProductId, LotId, CountId, DeltaQty, ReasonCode, Status, ActorId, ApprovedBy, CreatedAt
	        FROM InventoryAdjustments WHERE WarehouseId = @wid`
	params := map[string]any{"wid": warehouseID}
	if s := strings.TrimSpace(status); s != "" {
		sql += ` AND Status = @st`
		params["st"] = strings.ToUpper(s)
	}
	sql += ` ORDER BY CreatedAt DESC LIMIT 200`
	iter := client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	var out []InventoryAdjustmentView
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		v, err := scanAdjustment(row)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	if out == nil {
		out = []InventoryAdjustmentView{}
	}
	return out, nil
}

func scanAdjustment(row *spanner.Row) (*InventoryAdjustmentView, error) {
	var v InventoryAdjustmentView
	var lot, countID, reason, actor, approved spanner.NullString
	var created spanner.NullTime
	if err := row.Columns(&v.AdjustmentID, &v.WarehouseID, &v.ProductID, &lot, &countID,
		&v.DeltaQty, &reason, &v.Status, &actor, &approved, &created); err != nil {
		return nil, err
	}
	if lot.Valid {
		v.LotID = lot.StringVal
	}
	if countID.Valid {
		v.CountID = countID.StringVal
	}
	if reason.Valid {
		v.ReasonCode = reason.StringVal
	}
	if actor.Valid {
		v.ActorID = actor.StringVal
	}
	if approved.Valid {
		v.ApprovedBy = approved.StringVal
	}
	if created.Valid {
		v.CreatedAt = created.Time.UTC().Format(time.RFC3339Nano)
	}
	return &v, nil
}
