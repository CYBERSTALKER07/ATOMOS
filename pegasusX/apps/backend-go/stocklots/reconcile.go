package stocklots

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
)

// InventoryReconcileReport compares V2 vs lot sums.
type InventoryReconcileReport struct {
	WarehouseID string                  `json:"warehouse_id"`
	SupplierID  string                  `json:"supplier_id"`
	Matched     int                     `json:"matched"`
	Mismatches  []InventoryReconcileRow `json:"mismatches"`
}

// InventoryReconcileRow is one SKU drift.
type InventoryReconcileRow struct {
	ProductID    string `json:"product_id"`
	V2OnHand     int64  `json:"v2_on_hand"`
	LotsOnHand   int64  `json:"lots_on_hand"`
	V2Reserved   int64  `json:"v2_reserved"`
	LotsReserved int64  `json:"lots_reserved"`
	MissingV2    bool   `json:"missing_v2,omitempty"`
}

// InventoryRepairResult summarizes mutations applied by RepairInventoryV2.
type InventoryRepairResult struct {
	WarehouseID               string                   `json:"warehouse_id"`
	SupplierID                string                   `json:"supplier_id"`
	ReleasedOrderReservations int                      `json:"released_order_reservations"`
	LotsAdjusted              int                      `json:"lots_adjusted"`
	ProductsRolledUp          int                      `json:"products_rolled_up"`
	Before                    InventoryReconcileReport `json:"before"`
	After                     InventoryReconcileReport `json:"after"`
}

// ReconcileInventoryV2 asserts SupplierInventoryV2 ≡ sum AVAILABLE lots,
// including orphan AVAILABLE StockLots that have no SupplierInventoryV2 row.
func ReconcileInventoryV2(ctx context.Context, client *spanner.Client, supplierID, warehouseID string) (*InventoryReconcileReport, error) {
	if client == nil {
		return nil, fmt.Errorf("spanner unavailable")
	}
	return ReconcileInventoryV2WithReader(ctx, client.Single().WithTimestampBound(spanner.StrongRead()), supplierID, warehouseID)
}

// ReconcileInventoryV2WithReader executes ReconcileInventoryV2 against a caller-supplied
// Spanner reader (such as a shared ReadOnlyTransaction snapshot).
func ReconcileInventoryV2WithReader(
	ctx context.Context,
	rdr interface {
		Query(context.Context, spanner.Statement) *spanner.RowIterator
	},
	supplierID, warehouseID string,
) (*InventoryReconcileReport, error) {
	if rdr == nil {
		return nil, fmt.Errorf("spanner unavailable")
	}
	supplierID = strings.TrimSpace(supplierID)
	warehouseID = strings.TrimSpace(warehouseID)
	iter := rdr.Query(ctx, spanner.Statement{
		SQL: `SELECT COALESCE(v.ProductId, l.ProductId) AS ProductId,
		             v.ProductId IS NOT NULL AS HasV2,
		             COALESCE(v.QuantityOnHand, 0) AS V2OnHand,
		             COALESCE(v.QuantityReserved, 0) AS V2Reserved,
		             COALESCE(l.qoh, 0) AS LotsOnHand,
		             COALESCE(l.qr, 0) AS LotsReserved
		      FROM (
		        SELECT ProductId, QuantityOnHand, QuantityReserved
		        FROM SupplierInventoryV2
		        WHERE SupplierId = @sid AND WarehouseId = @wid
		      ) v
		      FULL OUTER JOIN (
		        SELECT ProductId, SUM(QuantityOnHand) AS qoh, SUM(QuantityReserved) AS qr
		        FROM StockLots
		        WHERE SupplierId = @sid AND WarehouseId = @wid AND Status = 'AVAILABLE'
		        GROUP BY ProductId
		      ) l ON l.ProductId = v.ProductId
		      ORDER BY ProductId`,
		Params: map[string]any{"sid": supplierID, "wid": warehouseID},
	})
	defer iter.Stop()
	rep := &InventoryReconcileReport{WarehouseID: warehouseID, SupplierID: supplierID, Mismatches: []InventoryReconcileRow{}}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var pid string
		var hasV2 bool
		var v2q, v2r, lq, lr int64
		if err := row.Columns(&pid, &hasV2, &v2q, &v2r, &lq, &lr); err != nil {
			return nil, err
		}
		if hasV2 && v2q == lq && v2r == lr {
			rep.Matched++
			continue
		}
		rep.Mismatches = append(rep.Mismatches, InventoryReconcileRow{
			ProductID:    pid,
			V2OnHand:     v2q,
			LotsOnHand:   lq,
			V2Reserved:   v2r,
			LotsReserved: lr,
			MissingV2:    !hasV2,
		})
	}
	return rep, nil
}

// RepairInventoryV2 transactionally frees stranded terminal OrderLotReservations
// on StockLots and recomputes SupplierInventoryV2 from AVAILABLE StockLots.
func RepairInventoryV2(ctx context.Context, client *spanner.Client, supplierID, warehouseID string) (*InventoryRepairResult, error) {
	if client == nil {
		return nil, fmt.Errorf("spanner unavailable")
	}
	supplierID = strings.TrimSpace(supplierID)
	warehouseID = strings.TrimSpace(warehouseID)
	if supplierID == "" || warehouseID == "" {
		return nil, fmt.Errorf("supplier_id and warehouse_id required")
	}

	before, err := ReconcileInventoryV2(ctx, client, supplierID, warehouseID)
	if err != nil {
		return nil, err
	}

	res := &InventoryRepairResult{
		WarehouseID: warehouseID,
		SupplierID:  supplierID,
		Before:      *before,
	}

	err = spannerutils.RunReadWriteTransaction(ctx, client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		released, lotsAdj, overridesByProduct, muts, rErr := releaseStrandedTerminalLotReservationsInTxn(ctx, txn, supplierID, warehouseID)
		if rErr != nil {
			return rErr
		}
		res.ReleasedOrderReservations = released
		res.LotsAdjusted = lotsAdj
		if len(muts) > 0 {
			if err := txn.BufferWrite(muts); err != nil {
				return fmt.Errorf("buffer lot repair: %w", err)
			}
		}

		pids, v2Set, pErr := collectWarehouseProductIDsInTxn(ctx, txn, supplierID, warehouseID)
		if pErr != nil {
			return pErr
		}
		mismatchSet := make(map[string]bool, len(before.Mismatches))
		for _, m := range before.Mismatches {
			mismatchSet[m.ProductID] = true
		}

		rolled := 0
		for _, pid := range pids {
			ov := overridesByProduct[pid]
			if !mismatchSet[pid] && len(ov) == 0 && v2Set[pid] {
				continue
			}
			if err := RollupInventoryV2WithLotOverridesInTxn(ctx, txn, supplierID, warehouseID, pid, ov); err != nil {
				return err
			}
			rolled++
		}
		res.ProductsRolledUp = rolled
		return nil
	})
	if err != nil {
		return nil, err
	}

	after, err := ReconcileInventoryV2(ctx, client, supplierID, warehouseID)
	if err != nil {
		return nil, err
	}
	res.After = *after
	return res, nil
}

func releaseStrandedTerminalLotReservationsInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	supplierID, warehouseID string,
) (releasedRows int, lotsAdjusted int, overridesByProduct map[string]map[string]StockLotSnapshot, muts []*spanner.Mutation, err error) {
	stmt := spanner.Statement{
		SQL: `SELECT r.OrderId, r.LotId, r.Quantity, l.ProductId, l.QuantityOnHand, l.QuantityReserved, l.Status
		      FROM OrderLotReservations r
		      JOIN Orders o ON o.OrderId = r.OrderId
		      JOIN StockLots l ON l.LotId = r.LotId
		      WHERE l.SupplierId = @sid AND l.WarehouseId = @wid
		        AND o.Status IN ('CANCELLED', 'COMPLETED', 'REJECTED')`,
		Params: map[string]any{"sid": supplierID, "wid": warehouseID},
	}
	type lotMeta struct {
		productID string
		qoh       int64
		qr        int64
		status    string
		release   int64
	}
	byLot := make(map[string]*lotMeta)
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()
	for {
		row, qErr := iter.Next()
		if qErr == iterator.Done {
			break
		}
		if qErr != nil {
			return 0, 0, nil, nil, fmt.Errorf("query stranded lot reservations: %w", qErr)
		}
		var oid, lid, pid, st string
		var qty, qoh, qr int64
		if cErr := row.Columns(&oid, &lid, &qty, &pid, &qoh, &qr, &st); cErr != nil {
			return 0, 0, nil, nil, cErr
		}
		muts = append(muts, spanner.Delete("OrderLotReservations", spanner.Key{oid, lid}))
		releasedRows++
		m, ok := byLot[lid]
		if !ok {
			m = &lotMeta{productID: pid, qoh: qoh, qr: qr, status: st}
			byLot[lid] = m
		}
		m.release += qty
	}

	overridesByProduct = make(map[string]map[string]StockLotSnapshot)
	lotIDs := make([]string, 0, len(byLot))
	for lid := range byLot {
		lotIDs = append(lotIDs, lid)
	}
	sort.Strings(lotIDs)
	for _, lid := range lotIDs {
		m := byLot[lid]
		nextQR := m.qr - m.release
		if nextQR < 0 {
			nextQR = 0
		}
		if nextQR != m.qr {
			muts = append(muts, spanner.UpdateMap("StockLots", map[string]any{
				"LotId":            lid,
				"QuantityReserved": nextQR,
				"UpdatedAt":        spanner.CommitTimestamp,
			}))
			lotsAdjusted++
		}
		if overridesByProduct[m.productID] == nil {
			overridesByProduct[m.productID] = make(map[string]StockLotSnapshot)
		}
		overridesByProduct[m.productID][lid] = StockLotSnapshot{
			SupplierID:       supplierID,
			WarehouseID:      warehouseID,
			ProductID:        m.productID,
			QuantityOnHand:   m.qoh,
			QuantityReserved: nextQR,
			Status:           m.status,
		}
	}
	return releasedRows, lotsAdjusted, overridesByProduct, muts, nil
}

func collectWarehouseProductIDsInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID, warehouseID string) ([]string, map[string]bool, error) {
	seen := make(map[string]struct{})
	v2Set := make(map[string]bool)
	for i, sql := range []string{
		`SELECT ProductId FROM SupplierInventoryV2 WHERE SupplierId = @sid AND WarehouseId = @wid`,
		`SELECT DISTINCT ProductId FROM StockLots WHERE SupplierId = @sid AND WarehouseId = @wid AND Status = 'AVAILABLE'`,
	} {
		iter := txn.Query(ctx, spanner.Statement{
			SQL:    sql,
			Params: map[string]any{"sid": supplierID, "wid": warehouseID},
		})
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				iter.Stop()
				return nil, nil, fmt.Errorf("collect warehouse product ids: %w", err)
			}
			var pid string
			if err := row.Columns(&pid); err != nil {
				iter.Stop()
				return nil, nil, err
			}
			pid = strings.TrimSpace(pid)
			if pid == "" {
				continue
			}
			seen[pid] = struct{}{}
			if i == 0 {
				v2Set[pid] = true
			}
		}
		iter.Stop()
	}
	pids := make([]string, 0, len(seen))
	for pid := range seen {
		pids = append(pids, pid)
	}
	sort.Strings(pids)
	return pids, v2Set, nil
}
