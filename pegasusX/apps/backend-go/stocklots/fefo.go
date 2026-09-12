package stocklots

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ErrInventoryExhausted mirrors order.ErrInventoryExhausted messaging for lot paths.
var ErrInventoryExhausted = fmt.Errorf("inventory exhausted")

// LineQty is a SKU quantity for FEFO reserve/release.
type LineQty struct {
	SKU      string
	Quantity int64
}

type lotCandidate struct {
	LotID      string
	Available  int64
	Expiry     spanner.NullDate
	ReceivedAt time.Time
	QoH        int64
	Reserved   int64
}

// ReserveFEFOInTxn allocates lots FEFO (perishable) or FIFO (non-perishable) and rolls up V2.
func ReserveFEFOInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	supplierID, warehouseID, orderID, retailerID string,
	expectedDelivery time.Time,
	lines []LineQty,
) error {
	if strings.TrimSpace(warehouseID) == "" || len(lines) == 0 {
		return nil
	}
	supplierID = strings.TrimSpace(supplierID)
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return fmt.Errorf("lot reserve: order_id required when WMS lots enabled")
	}
	if expectedDelivery.IsZero() {
		expectedDelivery = time.Now().UTC().Add(24 * time.Hour)
	}

	skuQty := map[string]int64{}
	for _, l := range lines {
		sku := strings.TrimSpace(l.SKU)
		if sku == "" || l.Quantity <= 0 {
			continue
		}
		skuQty[sku] += l.Quantity
	}

	minShelfRetailer, _ := loadRetailerMinShelf(ctx, txn, retailerID)

	for sku, need := range skuQty {
		perishable, productMin, err := loadProductShelfMeta(ctx, txn, sku)
		if err != nil {
			return err
		}
		minDays := productMin
		if minShelfRetailer > 0 {
			minDays = minShelfRetailer
		}

		cands, err := loadAvailableLots(ctx, txn, supplierID, warehouseID, sku)
		if err != nil {
			return err
		}
		cands = filterShelfLife(cands, perishable, expectedDelivery, minDays)
		if perishable {
			sort.SliceStable(cands, func(i, j int) bool {
				if !cands[i].Expiry.Valid && !cands[j].Expiry.Valid {
					return cands[i].ReceivedAt.Before(cands[j].ReceivedAt)
				}
				if !cands[i].Expiry.Valid {
					return false
				}
				if !cands[j].Expiry.Valid {
					return true
				}
				ti := cands[i].Expiry.Date.In(time.UTC)
				tj := cands[j].Expiry.Date.In(time.UTC)
				if ti.Equal(tj) {
					return cands[i].ReceivedAt.Before(cands[j].ReceivedAt)
				}
				return ti.Before(tj)
			})
		} else {
			sort.SliceStable(cands, func(i, j int) bool {
				return cands[i].ReceivedAt.Before(cands[j].ReceivedAt)
			})
		}

		remaining := need
		for _, c := range cands {
			if remaining <= 0 {
				break
			}
			take := c.Available
			if take > remaining {
				take = remaining
			}
			if take <= 0 {
				continue
			}
			newReserved := c.Reserved + take
			status := "AVAILABLE"
			if c.QoH-newReserved <= 0 && c.QoH <= newReserved {
				// keep AVAILABLE until depleted on pick/ship; reserved can equal QoH
			}
			if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("StockLots", map[string]any{
				"LotId":            c.LotID,
				"QuantityReserved": newReserved,
				"Status":           status,
				"UpdatedAt":        spanner.CommitTimestamp,
			})}); err != nil {
				return err
			}
			if err := txn.BufferWrite([]*spanner.Mutation{spanner.InsertOrUpdateMap("OrderLotReservations", map[string]any{
				"OrderId":   orderID,
				"LotId":     c.LotID,
				"Quantity":  take,
				"CreatedAt": spanner.CommitTimestamp,
			})}); err != nil {
				return err
			}
			remaining -= take
			c.Reserved = newReserved
		}
		if remaining > 0 {
			return fmt.Errorf("%w: sku %s has insufficient shelf-life-legal lots (short %d)", ErrInventoryExhausted, sku, remaining)
		}
		if err := RollupInventoryV2InTxn(ctx, txn, supplierID, warehouseID, sku); err != nil {
			return err
		}
	}
	return nil
}

// ReleaseLotReservationsInTxn reverses OrderLotReservations for an order and rolls up.
func ReleaseLotReservationsInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID, warehouseID, orderID string) error {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return nil
	}
	stmt := spanner.Statement{
		SQL:    `SELECT LotId, Quantity FROM OrderLotReservations WHERE OrderId = @oid`,
		Params: map[string]any{"oid": orderID},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()

	type resRow struct {
		lotID string
		qty   int64
	}
	var rows []resRow
	productTouched := map[string]struct{}{}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		var lotID string
		var qty int64
		if err := row.Columns(&lotID, &qty); err != nil {
			return err
		}
		rows = append(rows, resRow{lotID, qty})
	}

	for _, r := range rows {
		lotRow, err := txn.ReadRow(ctx, "StockLots", spanner.Key{r.lotID},
			[]string{"SupplierId", "WarehouseId", "ProductId", "QuantityReserved", "QuantityOnHand", "Status"})
		if err != nil {
			if spanner.ErrCode(err) == 5 {
				continue
			}
			return err
		}
		var sid, wid, pid, status string
		var reserved, qoh int64
		if err := lotRow.Columns(&sid, &wid, &pid, &reserved, &qoh, &status); err != nil {
			return err
		}
		next := reserved - r.qty
		if next < 0 {
			next = 0
		}
		if status == "DEPLETED" && qoh-next > 0 {
			status = "AVAILABLE"
		}
		if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("StockLots", map[string]any{
			"LotId":            r.lotID,
			"QuantityReserved": next,
			"Status":           status,
			"UpdatedAt":        spanner.CommitTimestamp,
		})}); err != nil {
			return err
		}
		if err := txn.BufferWrite([]*spanner.Mutation{spanner.Delete("OrderLotReservations", spanner.Key{orderID, r.lotID})}); err != nil {
			return err
		}
		productTouched[pid] = struct{}{}
		if supplierID == "" {
			supplierID = sid
		}
		if warehouseID == "" {
			warehouseID = wid
		}
	}
	for pid := range productTouched {
		if err := RollupInventoryV2InTxn(ctx, txn, supplierID, warehouseID, pid); err != nil {
			return err
		}
	}
	return nil
}

func loadAvailableLots(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID, warehouseID, productID string) ([]lotCandidate, error) {
	stmt := spanner.Statement{
		SQL: `SELECT LotId, QuantityOnHand, QuantityReserved, ExpiryDate, ReceivedAt
		      FROM StockLots
		      WHERE SupplierId = @sid AND WarehouseId = @wid AND ProductId = @pid
		        AND Status = 'AVAILABLE'`,
		Params: map[string]any{
			"sid": supplierID,
			"wid": warehouseID,
			"pid": productID,
		},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()
	var out []lotCandidate
	now := time.Now().UTC().Truncate(24 * time.Hour)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var lotID string
		var qoh, qr int64
		var expiry spanner.NullDate
		var received time.Time
		if err := row.Columns(&lotID, &qoh, &qr, &expiry, &received); err != nil {
			return nil, err
		}
		if expiry.Valid {
			ed := expiry.Date.In(time.UTC)
			if !ed.After(now) {
				continue
			}
		}
		avail := qoh - qr
		if avail <= 0 {
			continue
		}
		out = append(out, lotCandidate{
			LotID: lotID, Available: avail, Expiry: expiry, ReceivedAt: received, QoH: qoh, Reserved: qr,
		})
	}
	return out, nil
}

func filterShelfLife(cands []lotCandidate, perishable bool, expectedDelivery time.Time, minDays int64) []lotCandidate {
	if !perishable || minDays <= 0 {
		return cands
	}
	cutoff := expectedDelivery.UTC().Truncate(24 * time.Hour).AddDate(0, 0, int(minDays))
	var out []lotCandidate
	for _, c := range cands {
		if !c.Expiry.Valid {
			continue
		}
		ed := c.Expiry.Date.In(time.UTC)
		if ed.Before(cutoff) {
			continue
		}
		out = append(out, c)
	}
	return out
}

func loadProductShelfMeta(ctx context.Context, txn *spanner.ReadWriteTransaction, productID string) (perishable bool, minDays int64, err error) {
	row, err := txn.ReadRow(ctx, "Products", spanner.Key{productID}, []string{"IsPerishable", "MinShelfLifeDays"})
	if err != nil {
		if spanner.ErrCode(err) == 5 {
			return false, 0, nil
		}
		return false, 0, err
	}
	var isPerishable bool
	var min spanner.NullInt64
	if err := row.Columns(&isPerishable, &min); err != nil {
		return false, 0, err
	}
	if min.Valid {
		minDays = min.Int64
	}
	return isPerishable, minDays, nil
}

func loadRetailerMinShelf(ctx context.Context, txn *spanner.ReadWriteTransaction, retailerID string) (int64, error) {
	retailerID = strings.TrimSpace(retailerID)
	if retailerID == "" {
		return 0, nil
	}
	row, err := txn.ReadRow(ctx, "Retailers", spanner.Key{retailerID}, []string{"MinShelfLifeDays"})
	if err != nil {
		if spanner.ErrCode(err) == 5 {
			return 0, nil
		}
		return 0, err
	}
	var min spanner.NullInt64
	if err := row.Columns(&min); err != nil {
		return 0, err
	}
	if min.Valid {
		return min.Int64, nil
	}
	return 0, nil
}
