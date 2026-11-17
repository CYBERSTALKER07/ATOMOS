package ar

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

// InvoiceBalanceAdjustment describes the pure-integer balance and principal update for an invoice.
type InvoiceBalanceAdjustment struct {
	NewPrincipalMinor int64
	NewBalanceMinor   int64
	BalanceDeltaMinor int64
	NewStatus         string
}

// ScaleInt64Ratio computes amt * part / total using exact integer arithmetic with
// half-up remainder rounding (bounded within <= total/2, zero float64).
func ScaleInt64Ratio(amt, part, total int64) (int64, error) {
	if amt < 0 || part < 0 || total <= 0 {
		return 0, fmt.Errorf("ar: invalid scale inputs amt=%d part=%d total=%d", amt, part, total)
	}
	if amt == 0 || part == 0 {
		return 0, nil
	}
	if part == total {
		return amt, nil
	}
	num := new(big.Int).Mul(big.NewInt(amt), big.NewInt(part))
	den := big.NewInt(total)
	q, r := new(big.Int), new(big.Int)
	q.QuoRem(num, den, r)

	// Half-up remainder bound: increment quotient iff 2*rem >= total.
	twoRem := new(big.Int).Lsh(r, 1)
	if twoRem.Cmp(den) >= 0 {
		q.Add(q, big.NewInt(1))
	}
	if !q.IsInt64() {
		return 0, fmt.Errorf("ar: scaled amount overflows int64")
	}
	return q.Int64(), nil
}

// ComputeInvoiceAdjustment calculates updated principal, remaining balance, and status
// after an order's total changes from prevOrderTotalMinor to newOrderTotalMinor.
func ComputeInvoiceAdjustment(inv Invoice, prevOrderTotalMinor, newOrderTotalMinor int64) (InvoiceBalanceAdjustment, bool, error) {
	if newOrderTotalMinor < 0 {
		return InvoiceBalanceAdjustment{}, false, fmt.Errorf("ar: negative order total %d", newOrderTotalMinor)
	}
	if inv.Status == StatusVoid || inv.Status == StatusPaid || inv.BalanceMinor <= 0 {
		return InvoiceBalanceAdjustment{}, false, nil
	}
	if prevOrderTotalMinor == newOrderTotalMinor {
		return InvoiceBalanceAdjustment{}, false, nil
	}

	var newPrincipal int64
	if prevOrderTotalMinor > 0 {
		scaled, err := ScaleInt64Ratio(inv.PrincipalMinor, newOrderTotalMinor, prevOrderTotalMinor)
		if err != nil {
			return InvoiceBalanceAdjustment{}, false, err
		}
		if newOrderTotalMinor < prevOrderTotalMinor && scaled > inv.PrincipalMinor {
			scaled = inv.PrincipalMinor
		}
		newPrincipal = scaled
	} else {
		newPrincipal = newOrderTotalMinor
	}
	if newPrincipal < 0 {
		newPrincipal = 0
	}

	paidMinor := inv.PrincipalMinor - inv.BalanceMinor
	if paidMinor < 0 {
		paidMinor = 0
	}
	newBalance := newPrincipal - paidMinor
	if newBalance < 0 {
		newBalance = 0
	}
	if newPrincipal == inv.PrincipalMinor && newBalance == inv.BalanceMinor {
		return InvoiceBalanceAdjustment{}, false, nil
	}

	newStatus := StatusOpen
	switch {
	case newBalance == 0 && newPrincipal == 0 && paidMinor == 0:
		newStatus = StatusVoid
	case newBalance == 0:
		newStatus = StatusPaid
	case paidMinor > 0:
		newStatus = StatusPartial
	}

	return InvoiceBalanceAdjustment{
		NewPrincipalMinor: newPrincipal,
		NewBalanceMinor:   newBalance,
		BalanceDeltaMinor: newBalance - inv.BalanceMinor,
		NewStatus:         newStatus,
	}, true, nil
}

// AdjustInvoiceBalanceForOrderInTxn updates an open/partial AR invoice for orderID in the
// active Spanner RW transaction, inserts a deterministic ArLedgerEntries row, and emits
// an ARInvoiceEvent via transactional outbox.
func AdjustInvoiceBalanceForOrderInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	orderID string,
	prevOrderTotalMinor, newOrderTotalMinor int64,
	idempotencyKey string,
) (int64, error) {
	if txn == nil {
		return 0, fmt.Errorf("nil spanner txn")
	}
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return 0, fmt.Errorf("order_id required")
	}
	if prevOrderTotalMinor < 0 || newOrderTotalMinor < 0 {
		return 0, fmt.Errorf("ar: invalid order totals prev=%d new=%d", prevOrderTotalMinor, newOrderTotalMinor)
	}
	if prevOrderTotalMinor == newOrderTotalMinor {
		return 0, nil
	}

	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" {
		idempotencyKey = fmt.Sprintf("ar-adj:%s:%d->%d", orderID, prevOrderTotalMinor, newOrderTotalMinor)
	}

	idemIter := txn.Query(ctx, spanner.Statement{
		SQL:    `SELECT EntryId FROM ArLedgerEntries WHERE IdempotencyKey = @k LIMIT 1`,
		Params: map[string]any{"k": idempotencyKey},
	})
	_, qerr := idemIter.Next()
	idemIter.Stop()
	if qerr == nil {
		return 0, nil
	}
	if qerr != iterator.Done {
		return 0, qerr
	}

	inv, found, err := getInvoiceByOrderInTxn(ctx, txn, orderID)
	if err != nil {
		return 0, err
	}
	if !found {
		return 0, nil
	}

	adj, changed, err := ComputeInvoiceAdjustment(inv, prevOrderTotalMinor, newOrderTotalMinor)
	if err != nil || !changed {
		return 0, err
	}

	entryType := "ADJUSTMENT"
	absDelta := adj.BalanceDeltaMinor
	if adj.BalanceDeltaMinor < 0 {
		entryType = "CREDIT_NOTE"
		absDelta = -adj.BalanceDeltaMinor
	}
	eventType := events.EventARInvoicePayment
	if adj.NewBalanceMinor == 0 {
		eventType = events.EventARInvoiceSettled
	} else if adj.BalanceDeltaMinor > 0 {
		eventType = events.EventARInvoiceOpened
	}

	now := time.Now().UTC()
	buf := outbox.NewSpannerTxnBuffer(txn)
	if err := outbox.EmitJSON(ctx, buf, events.AggregateARInvoice, inv.InvoiceID, events.TopicMain, events.ARInvoiceEvent{
		BaseEvent:      events.BaseEvent{Type: eventType, Timestamp: now.Format(time.RFC3339Nano)},
		InvoiceID:      inv.InvoiceID,
		SupplierID:     inv.SupplierID,
		RetailerID:     inv.RetailerID,
		OrderID:        inv.OrderID,
		PrincipalMinor: adj.NewPrincipalMinor,
		AmountMinor:    absDelta,
		BalanceMinor:   adj.NewBalanceMinor,
		Status:         adj.NewStatus,
	}); err != nil {
		return 0, err
	}

	if err := txn.BufferWrite([]*spanner.Mutation{
		spanner.UpdateMap("ArInvoices", map[string]any{
			"InvoiceId":      inv.InvoiceID,
			"PrincipalMinor": adj.NewPrincipalMinor,
			"BalanceMinor":   adj.NewBalanceMinor,
			"Status":         adj.NewStatus,
			"Version":        inv.Version + 1,
			"UpdatedAt":      spanner.CommitTimestamp,
		}),
		spanner.InsertMap("ArLedgerEntries", map[string]any{
			"EntryId":        ledgerEntryID("adj", idempotencyKey),
			"InvoiceId":      inv.InvoiceID,
			"SupplierId":     inv.SupplierID,
			"RetailerId":     inv.RetailerID,
			"EntryType":      entryType,
			"AmountMinor":    adj.BalanceDeltaMinor,
			"IdempotencyKey": idempotencyKey,
			"RefOrderId":     inv.OrderID,
			"CreatedAt":      spanner.CommitTimestamp,
		}),
	}); err != nil {
		return 0, err
	}
	cur := strings.ToUpper(strings.TrimSpace(inv.Currency))
	if cur == "" {
		cur = "UZS"
	}
	if adj.BalanceDeltaMinor < 0 {
		if _, err := gl.PostEventInTxn(ctx, txn, buf, gl.RuleEventARAdjustWaive, gl.RuleEvalContext{
			EntityID:       inv.SupplierID,
			SupplierID:     inv.SupplierID,
			RetailerID:     inv.RetailerID,
			OrderID:        inv.OrderID,
			SourceID:       inv.InvoiceID,
			Currency:       cur,
			GrossMinor:     absDelta,
			Action:         "WAIVE",
			IdempotencyKey: "gl:ar:adj:" + idempotencyKey,
			Memo:           "AR balance waive/adjustment " + inv.InvoiceID,
			OccurredAt:     now,
		}); err != nil {
			return 0, err
		}
	} else if adj.BalanceDeltaMinor > 0 {
		if _, err := gl.BufferJournalEntryInTxn(ctx, txn, buf, gl.JournalEntryInput{
			EntityID:       inv.SupplierID,
			SourceType:     "AR_ADJUSTMENT",
			SourceID:       inv.InvoiceID,
			OrderID:        inv.OrderID,
			Memo:           "AR upward balance adjustment " + inv.InvoiceID,
			Currency:       cur,
			IdempotencyKey: "gl:ar:adj:" + idempotencyKey,
			OccurredAt:     now,
			Postings: []gl.PostingInput{
				{
					AccountCode: gl.AccountAccountsReceivable,
					Direction:   gl.DirectionDebit,
					AmountMinor: absDelta,
					Currency:    cur,
					PartyType:   "RETAILER",
					PartyID:     inv.RetailerID,
					Description: "Upward AR balance amendment",
				},
				{
					AccountCode: gl.AccountSalesRevenue,
					Direction:   gl.DirectionCredit,
					AmountMinor: absDelta,
					Currency:    cur,
					Description: "Upward AR sales adjustment",
				},
			},
		}); err != nil {
			return 0, err
		}
	}
	if err := buf.Flush(ctx); err != nil {
		return 0, err
	}
	return adj.BalanceDeltaMinor, nil
}

// AdjustByOrder applies ComputeInvoiceAdjustment in MemoryRepository for unit testing.
func (r *MemoryRepository) AdjustByOrder(_ context.Context, orderID string, prevOrderTotalMinor, newOrderTotalMinor int64, idempotencyKey string) (Invoice, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if idempotencyKey != "" && r.ledgerKeys[idempotencyKey] {
		return Invoice{}, false, nil
	}
	id, ok := r.byOrder[orderID]
	if !ok {
		return Invoice{}, false, nil
	}
	inv := r.byID[id]
	adj, changed, err := ComputeInvoiceAdjustment(inv, prevOrderTotalMinor, newOrderTotalMinor)
	if err != nil || !changed {
		return inv, false, err
	}
	if idempotencyKey != "" {
		r.ledgerKeys[idempotencyKey] = true
	}
	inv.PrincipalMinor = adj.NewPrincipalMinor
	inv.BalanceMinor = adj.NewBalanceMinor
	inv.Status = adj.NewStatus
	inv.Version++
	r.byID[id] = inv
	return inv, true, nil
}
