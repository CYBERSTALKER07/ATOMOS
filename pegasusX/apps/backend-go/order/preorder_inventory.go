package order

import (
	"context"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// reservationCreditForLines sums reserved qty per SKU for an existing order.
func reservationCreditForLines(lines []LineItem) map[string]int64 {
	credit := make(map[string]int64)
	for _, item := range lines {
		sku := strings.TrimSpace(item.SKU)
		if sku == "" || item.Quantity <= 0 {
			continue
		}
		credit[sku] += item.Quantity
	}
	return credit
}

// applyPreorderInventoryGuard plans checkout for new lines and returns fulfillable lines only.
func (s *Service) applyPreorderInventoryGuard(
	ctx context.Context,
	supplierID, warehouseID string,
	prevLines, requestedLines []LineItem,
) ([]LineItem, InventoryPlan, error) {
	if s == nil || s.spannerClient == nil || len(requestedLines) == 0 {
		return requestedLines, InventoryPlan{Fulfillable: append([]LineItem(nil), requestedLines...)}, nil
	}
	credit := reservationCreditForLines(prevLines)
	plan, err := PlanInventoryCheckoutWithCredit(ctx, s.spannerClient, supplierID, warehouseID, requestedLines, "", credit)
	if err != nil {
		return nil, plan, err
	}
	if len(plan.Fulfillable) == 0 && len(plan.Backorder) == 0 {
		return nil, plan, ErrInventoryExhausted
	}
	return plan.Fulfillable, plan, nil
}

// reconcilePreorderReservationsInTxn releases prior reservations and reserves fulfillable lines.
func reconcilePreorderReservationsInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	supplierID, warehouseID string,
	source OrderSource,
	prevLines, fulfillableLines []LineItem,
) error {
	if err := ReleaseReservationsInTxn(ctx, txn, supplierID, warehouseID, source, prevLines); err != nil {
		return err
	}
	if len(fulfillableLines) == 0 {
		return nil
	}
	return ReserveLineItemsInTxn(ctx, txn, supplierID, warehouseID, fulfillableLines)
}

func totalMinorForLines(lines []LineItem) int64 {
	var total int64
	for _, li := range lines {
		total += li.Quantity * li.UnitPrice
	}
	return total
}

// updatePreorderLines applies inventory guard and persists line changes with reservation reconcile.
func (s *Service) updatePreorderLines(
	ctx context.Context,
	current Order,
	requestedLines []LineItem,
	emit func(outbox.TxnBuffer, Order) error,
) (Order, error) {
	prevLines := append([]LineItem(nil), current.LineItems...)
	fulfillable, _, err := s.applyPreorderInventoryGuard(ctx, current.SupplierID, current.WarehouseID, prevLines, requestedLines)
	if err != nil {
		return current, err
	}
	current.LineItems = fulfillable
	current.TotalMinor = totalMinorForLines(fulfillable)
	err = s.repo.UpdateOrderWithTxn(ctx, current, nil, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return reconcilePreorderReservationsInTxn(ctx, txn, current.SupplierID, current.WarehouseID, current.Source, prevLines, fulfillable)
	}, func(txn outbox.TxnBuffer) error {
		if emit != nil {
			return emit(txn, current)
		}
		return nil
	})
	return current, err
}
