package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/retailer"
)

// ── Scaffold in-memory order repository ────────────────────────────────────
// Production replaces this with a Spanner-backed implementation that runs
// CreateOrder inside a ReadWriteTransaction and writes both the Orders row
// and the OutboxEvents row atomically.

type inMemoryOrderRepo struct {
	mu             sync.RWMutex
	byID           map[string]order.Order
	outboxAppender OutboxAppender
	windows        order.ReceivingWindowReader
}

func NewOrderRepo(outboxAppender OutboxAppender, windows order.ReceivingWindowReader) *inMemoryOrderRepo {
	return &inMemoryOrderRepo{
		byID:           make(map[string]order.Order),
		outboxAppender: outboxAppender,
		windows:        windows,
	}
}

// RetailerReceivingWindowAdapter reads receiving windows from a retailer repository.
type RetailerReceivingWindowAdapter struct {
	Repo retailer.Repository
}

func (a *RetailerReceivingWindowAdapter) GetReceivingWindows(ctx context.Context, retailerID string) (string, string, error) {
	if a == nil || a.Repo == nil {
		return "", "", nil
	}
	ret, ok, err := a.Repo.GetRetailer(ctx, retailerID)
	if err != nil {
		return "", "", err
	}
	if !ok {
		return "", "", nil
	}
	return ret.ReceivingWindowOpen, ret.ReceivingWindowClose, nil
}

func (r *inMemoryOrderRepo) CreateOrder(ctx context.Context, o *order.Order, emit func(outbox.TxnBuffer) error, _ order.StockReservationOpts) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if o == nil {
		return fmt.Errorf("create order: nil aggregate")
	}
	if _, exists := r.byID[o.OrderID]; exists {
		return fmt.Errorf("order_id collision: %s", o.OrderID)
	}
	if r.windows != nil {
		open, closeWindow, err := r.windows.GetReceivingWindows(ctx, o.RetailerID)
		if err != nil {
			return err
		}
		if err := order.SnapshotReceivingWindowsOnOrder(o, open, closeWindow); err != nil {
			return err
		}
	}
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	r.byID[o.OrderID] = *o
	return nil
}

func (r *inMemoryOrderRepo) GetOrder(_ context.Context, orderID string) (order.Order, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	o, ok := r.byID[orderID]
	return o, ok, nil
}

func (r *inMemoryOrderRepo) GetOrderTxn(_ context.Context, _ *spanner.ReadWriteTransaction, orderID string) (order.Order, bool, error) {
	return r.GetOrder(context.Background(), orderID)
}

func (r *inMemoryOrderRepo) GetFiscalByReceiptID(_ context.Context, receiptID string) (order.FiscalReceiptRow, bool, error) {
	if r == nil {
		return order.FiscalReceiptRow{}, false, nil
	}
	receiptID = strings.TrimSpace(receiptID)
	if receiptID == "" {
		return order.FiscalReceiptRow{}, false, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, o := range r.byID {
		if strings.TrimSpace(o.LatestFiscalReceiptID) != receiptID {
			continue
		}
		return order.FiscalReceiptRow{
			OrderID:         o.OrderID,
			AttemptID:       o.LatestFiscalAttemptID,
			SupplierID:      o.SupplierID,
			RetailerID:      o.RetailerID,
			Provider:        order.FiscalProviderPegasus,
			Status:          order.FiscalAttemptSuccess,
			FiscalReceiptID: receiptID,
			AmountMinor:     o.TotalMinor,
			Currency:        o.Currency,
		}, true, nil
	}
	return order.FiscalReceiptRow{}, false, nil
}

func (r *inMemoryOrderRepo) GetFiscalAttempt(_ context.Context, orderID, attemptID string) (order.FiscalReceiptRow, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	o, ok := r.byID[orderID]
	if !ok {
		return order.FiscalReceiptRow{}, false, nil
	}
	for _, fr := range o.PendingFiscalReceipts {
		if fr.AttemptID == attemptID {
			return fr, true, nil
		}
	}
	if o.FiscalReceiptUpdate != nil && o.FiscalReceiptUpdate.AttemptID == attemptID {
		return *o.FiscalReceiptUpdate, true, nil
	}
	return order.FiscalReceiptRow{}, false, nil
}

func (r *inMemoryOrderRepo) CountFiscalAttemptsByStatus(_ context.Context, orderID, status string) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	o, ok := r.byID[orderID]
	if !ok {
		return 0, nil
	}
	var n int64
	for _, fr := range o.PendingFiscalReceipts {
		if fr.Status == status {
			n++
		}
	}
	if o.FiscalReceiptUpdate != nil && o.FiscalReceiptUpdate.Status == status {
		n++
	}
	return n, nil
}

func (r *inMemoryOrderRepo) ListRetailerOrders(_ context.Context, retailerID string, limit int) ([]order.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 {
		limit = 25
	}
	items := make([]order.Order, 0, limit)
	for _, orderRecord := range r.byID {
		if orderRecord.RetailerID != strings.TrimSpace(retailerID) {
			continue
		}
		items = append(items, orderRecord)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (r *inMemoryOrderRepo) ListWarehouseOrdersByDeliveryWindow(_ context.Context, warehouseID string, from, to time.Time, limit int) ([]order.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 {
		limit = 200
	}
	items := make([]order.Order, 0, limit)
	for _, orderRecord := range r.byID {
		if orderRecord.WarehouseID != strings.TrimSpace(warehouseID) || orderRecord.RequestedDeliveryDate == nil {
			continue
		}
		requested := orderRecord.RequestedDeliveryDate.UTC()
		if requested.Before(from.UTC()) || !requested.Before(to.UTC()) {
			continue
		}
		items = append(items, orderRecord)
	}
	sort.Slice(items, func(i, j int) bool {
		left := items[i].RequestedDeliveryDate
		right := items[j].RequestedDeliveryDate
		if left == nil || right == nil {
			return items[i].UpdatedAt.After(items[j].UpdatedAt)
		}
		if left.Equal(*right) {
			return items[i].UpdatedAt.After(items[j].UpdatedAt)
		}
		return left.After(*right)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (r *inMemoryOrderRepo) ListDueAutoConfirmOrders(_ context.Context, before time.Time, limit int) ([]order.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 {
		limit = 50
	}
	items := make([]order.Order, 0, limit)
	for _, orderRecord := range r.byID {
		if orderRecord.AutoConfirmAt == nil || orderRecord.Source != order.OrderSourceAIPreorder || orderRecord.ConfirmationStatus != order.ConfirmationStatusPending {
			continue
		}
		if orderRecord.AutoConfirmAt.After(before.UTC()) {
			continue
		}
		items = append(items, orderRecord)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].AutoConfirmAt.Before(*items[j].AutoConfirmAt)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}
func (r *inMemoryOrderRepo) ListManifestOrders(_ context.Context, manifestID string) ([]order.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []order.Order
	for _, o := range r.byID {
		if o.ManifestID == manifestID {
			out = append(out, o)
		}
	}
	return out, nil
}

func (r *inMemoryOrderRepo) ListWarehousePreorders(_ context.Context, warehouseID string, limit, offset int) ([]order.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 {
		limit = 50
	}
	var out []order.Order
	for _, o := range r.byID {
		if o.WarehouseID != warehouseID || o.Source != order.OrderSourceManualPreorder {
			continue
		}
		if o.Status != order.StatusScheduled && o.Status != order.StatusAutoAccepted {
			continue
		}
		out = append(out, o)
	}
	sort.Slice(out, func(i, j int) bool {
		var di, dj time.Time
		if out[i].RequestedDeliveryDate != nil {
			di = *out[i].RequestedDeliveryDate
		}
		if out[j].RequestedDeliveryDate != nil {
			dj = *out[j].RequestedDeliveryDate
		}
		return di.Before(dj)
	})
	if offset >= len(out) {
		return nil, nil
	}
	end := offset + limit
	if end > len(out) {
		end = len(out)
	}
	return out[offset:end], nil
}

func (r *inMemoryOrderRepo) ListOrdersForStockCommitment(_ context.Context, warehouseID string, limit int) ([]order.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 {
		limit = 500
	}
	var out []order.Order
	for _, o := range r.byID {
		if o.WarehouseID != warehouseID {
			continue
		}
		switch o.Status {
		case order.StatusPending, order.StatusScheduled, order.StatusAutoAccepted,
			order.StatusLoaded, order.StatusInTransit, order.StatusDelayed:
			out = append(out, o)
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *inMemoryOrderRepo) ClearBackorder(ctx context.Context, id string, emit func(outbox.TxnBuffer) error, _ order.StockReservationOpts) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	o, exists := r.byID[id]
	if !exists {
		return fmt.Errorf("order not found")
	}
	o.Status = order.StatusPending
	r.byID[id] = o
	return nil
}

func (r *inMemoryOrderRepo) ListBackorderedOrders(ctx context.Context, limit int) ([]order.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []order.Order
	for _, o := range r.byID {
		if o.Status == order.StatusBackordered {
			out = append(out, o)
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *inMemoryOrderRepo) CreateConditionReport(ctx context.Context, report order.ConditionReport, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *inMemoryOrderRepo) ListConditionReports(ctx context.Context, orderID string) ([]order.ConditionReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return nil, nil
}

func (r *inMemoryOrderRepo) ListOrdersByStatus(ctx context.Context, supplierID, status string, limit int) ([]order.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []order.Order
	for _, o := range r.byID {
		if o.SupplierID == supplierID && string(o.Status) == status {
			out = append(out, o)
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *inMemoryOrderRepo) UpdateOrder(ctx context.Context, o order.Order, _ []order.DeliveryProofArtifact, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.byID[o.OrderID]; !exists {
		return fmt.Errorf("order not found: %s", o.OrderID)
	}
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	r.byID[o.OrderID] = o
	return nil
}

func (r *inMemoryOrderRepo) UpdateOrderWithTxn(ctx context.Context, o order.Order, proofs []order.DeliveryProofArtifact, inTxn func(context.Context, *spanner.ReadWriteTransaction) error, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.byID[o.OrderID]; !exists {
		return fmt.Errorf("order not found: %s", o.OrderID)
	}

	o.UpdatedAt = time.Now().UTC()
	r.byID[o.OrderID] = o

	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	
	if inTxn != nil {
		if err := inTxn(ctx, nil); err != nil {
			return err
		}
	}

	return nil
}

func (r *inMemoryOrderRepo) FindSiblingDriversForOrder(ctx context.Context, orderID string) ([]string, error) {
	return nil, nil
}

func (r *inMemoryOrderRepo) FindPendingBuyerAcceptance(ctx context.Context, limit int) ([]*order.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var res []*order.Order
	for _, o := range r.byID {
		if o.BuyerAcceptanceStatus == "PENDING" {
			copy := o
			res = append(res, &copy)
			if len(res) >= limit {
				break
			}
		}
	}
	return res, nil
}
