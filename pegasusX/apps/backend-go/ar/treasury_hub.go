package ar

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
)

// AgingSummaryResponse represents the aggregated AR health for a supplier.
type AgingSummaryResponse struct {
	SupplierID               string `json:"supplier_id"`
	Currency                 string `json:"currency"`
	TotalOpenMinor           int64  `json:"total_open_minor"`
	TotalOverdueMinor        int64  `json:"total_overdue_minor"`
	BucketCurrentMinor       int64  `json:"bucket_current_minor"`
	Bucket1To30Minor         int64  `json:"bucket_1_30_minor"`
	Bucket31To60Minor        int64  `json:"bucket_31_60_minor"`
	Bucket61To90Minor        int64  `json:"bucket_61_90_minor"`
	Bucket90PlusMinor        int64  `json:"bucket_90_plus_minor"`
	TotalInvoicesCount       int64  `json:"total_invoices_count"`
	DelinquentRetailersCount int64  `json:"delinquent_retailers_count"`
	HighRiskInvoiceCount     int64  `json:"high_risk_invoice_count"`
	ComputedAt               string `json:"computed_at"`
}

// DelinquencyLockStatus represents credit-order gate status for a retailer.
type DelinquencyLockStatus struct {
	RetailerID         string          `json:"retailer_id"`
	SupplierID         string          `json:"supplier_id"`
	IsLocked           bool            `json:"is_locked"`
	Reason             string          `json:"reason,omitempty"`
	OverdueAmountMinor int64           `json:"overdue_amount_minor"`
	OverdueCount       int64           `json:"overdue_count"`
	DisputedMinor      int64           `json:"disputed_minor,omitempty"`
	UndisputedMinor    int64           `json:"undisputed_minor,omitempty"`
	ActivePromise      *PaymentPromise `json:"active_promise,omitempty"`
	CheckedAt          string          `json:"checked_at"`
}

// RetailerPayInvoiceRequest is submitted to pay an open AR invoice.
type RetailerPayInvoiceRequest struct {
	AmountMinor      int64  `json:"amount_minor"`
	PaymentMethod    string `json:"payment_method"` // WALLET, CARD, BANK_TRANSFER
	PaymentReference string `json:"payment_reference,omitempty"`
}

// WriteOffRequest is submitted to write off uncollectible debt.
type WriteOffRequest struct {
	Reason string `json:"reason"`
}

// GetSupplierAgingSummary aggregates all open AR items for a supplier.
func (s *Service) GetSupplierAgingSummary(ctx context.Context, supplierID string) (*AgingSummaryResponse, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("ar service unavailable")
	}
	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" {
		return nil, fmt.Errorf("supplier_id required")
	}

	var invoices []Invoice
	if pager, ok := s.repo.(InvoicePager); ok {
		cursor := ""
		for {
			page, next, err := pager.ListBySupplierPage(ctx, supplierID, "", 200, cursor)
			if err != nil {
				return nil, err
			}
			invoices = append(invoices, page...)
			if next == "" || next == cursor || len(page) == 0 {
				break
			}
			cursor = next
		}
	} else {
		var err error
		invoices, err = s.repo.ListBySupplier(ctx, supplierID, "", 1000)
		if err != nil {
			return nil, err
		}
	}

	now := s.now().UTC()
	resp := &AgingSummaryResponse{
		SupplierID: supplierID,
		Currency:   "UZS",
		ComputedAt: now.Format(time.RFC3339),
	}

	delinquentRetailers := map[string]struct{}{}

	for _, inv := range invoices {
		if inv.Status == StatusPaid || inv.Status == StatusVoid {
			continue
		}
		resp.TotalInvoicesCount++
		resp.TotalOpenMinor += inv.BalanceMinor
		if inv.Currency != "" {
			resp.Currency = inv.Currency
		}

		bucket := AgingBucketFor(inv.DueAt, now)
		switch bucket {
		case AgingCurrent:
			resp.BucketCurrentMinor += inv.BalanceMinor
		case Aging1to30:
			resp.Bucket1To30Minor += inv.BalanceMinor
			resp.TotalOverdueMinor += inv.BalanceMinor
		case Aging31to60:
			resp.Bucket31To60Minor += inv.BalanceMinor
			resp.TotalOverdueMinor += inv.BalanceMinor
			resp.HighRiskInvoiceCount++
			delinquentRetailers[inv.RetailerID] = struct{}{}
		case Aging61to90:
			resp.Bucket61To90Minor += inv.BalanceMinor
			resp.TotalOverdueMinor += inv.BalanceMinor
			resp.HighRiskInvoiceCount++
			delinquentRetailers[inv.RetailerID] = struct{}{}
		case Aging90Plus:
			resp.Bucket90PlusMinor += inv.BalanceMinor
			resp.TotalOverdueMinor += inv.BalanceMinor
			resp.HighRiskInvoiceCount++
			delinquentRetailers[inv.RetailerID] = struct{}{}
		}

		if inv.DunningStep >= StepCreditHold {
			delinquentRetailers[inv.RetailerID] = struct{}{}
		}
	}

	resp.DelinquentRetailersCount = int64(len(delinquentRetailers))
	return resp, nil
}

// CheckRetailerDelinquencyLock evaluates if a retailer is locked from new credit orders.
func (s *Service) CheckRetailerDelinquencyLock(ctx context.Context, retailerID, supplierID string) (*DelinquencyLockStatus, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("ar service unavailable")
	}
	retailerID = strings.TrimSpace(retailerID)
	supplierID = strings.TrimSpace(supplierID)
	if retailerID == "" {
		return nil, fmt.Errorf("retailer_id required")
	}

	var invoices []Invoice
	if pager, ok := s.repo.(InvoicePager); ok {
		cursor := ""
		for {
			page, next, err := pager.ListByRetailerPage(ctx, retailerID, "", 200, cursor)
			if err != nil {
				return nil, err
			}
			invoices = append(invoices, page...)
			if next == "" || next == cursor || len(page) == 0 {
				break
			}
			cursor = next
		}
	} else {
		var err error
		invoices, err = s.repo.ListByRetailer(ctx, retailerID, "", 200)
		if err != nil {
			return nil, err
		}
	}

	now := s.now().UTC()
	var openInvoices []Invoice
	var orderIDs []string
	for _, inv := range invoices {
		if supplierID != "" && inv.SupplierID != supplierID {
			continue
		}
		if inv.Status == StatusPaid || inv.Status == StatusVoid {
			continue
		}
		openInvoices = append(openInvoices, inv)
		if inv.OrderID != "" {
			orderIDs = append(orderIDs, inv.OrderID)
		}
	}

	disputes, err := s.lookupDisputes(ctx, orderIDs)
	if err != nil {
		return nil, err
	}
	activePromises, err := s.listActivePromisesForInvoices(ctx, openInvoices, now)
	if err != nil {
		return nil, err
	}

	var overdueAmount int64
	var overdueCount int64
	var totalDisputed, totalUndisputed int64
	var isLocked bool
	var lockReason string
	var activePromise *PaymentPromise

	for _, inv := range openInvoices {
		disputed, undisputed := clampDisputed(inv.BalanceMinor, disputes[inv.OrderID])
		totalDisputed += disputed
		totalUndisputed += undisputed

		// Disputed-only remainder is excluded from delinquency/auto-lock escalation.
		if undisputed <= 0 {
			continue
		}

		p, hasPromise := hasActiveUnexpiredPromise(activePromises, inv, now)
		if hasPromise && activePromise == nil {
			cp := p
			activePromise = &cp
		}

		graceEnd := inv.DueAt.AddDate(0, 0, int(inv.GracePeriodDays))
		if now.After(graceEnd) {
			overdueCount++
			overdueAmount += undisputed
			if !hasPromise && (inv.DunningStep >= StepCreditHold || now.Sub(inv.DueAt) >= 21*24*time.Hour) {
				isLocked = true
				lockReason = fmt.Sprintf("Delinquency lock: Invoice %s is %s (overdue %d days)", inv.InvoiceID, StepName(inv.DunningStep), int(now.Sub(inv.DueAt).Hours()/24))
			}
		}
	}

	return &DelinquencyLockStatus{
		RetailerID:         retailerID,
		SupplierID:         supplierID,
		IsLocked:           isLocked,
		Reason:             lockReason,
		OverdueAmountMinor: overdueAmount,
		OverdueCount:       overdueCount,
		DisputedMinor:      totalDisputed,
		UndisputedMinor:    totalUndisputed,
		ActivePromise:      activePromise,
		CheckedAt:          now.Format(time.RFC3339),
	}, nil
}

// WriteOffInvoice voids an uncollectible invoice and logs an audit ledger adjustment.
func (s *Service) WriteOffInvoice(ctx context.Context, invoiceID, approverID string, req WriteOffRequest) (*Invoice, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("ar service unavailable")
	}
	_ = approverID
	_ = req
	invoiceID = strings.TrimSpace(invoiceID)
	if invoiceID == "" {
		return nil, fmt.Errorf("invoice_id required")
	}

	inv, err := s.repo.GetByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if claims, ok := auth.FromContext(ctx); ok {
		if claims.Role != auth.RoleAdmin && claims.Role != auth.RoleWarehouseAdmin {
			return nil, ErrForbidden
		}
		sid := strings.TrimSpace(claims.SupplierID)
		if sid == "" || strings.TrimSpace(inv.SupplierID) != sid {
			return nil, ErrForbidden
		}
	}
	if inv.Status == StatusPaid || inv.Status == StatusVoid {
		return nil, fmt.Errorf("cannot write off invoice with status %s", inv.Status)
	}

	// Ensure the invoice status and balance are persisted using VoidInvoice.
	idemKey := fmt.Sprintf("writeoff:%s:%s:v%d", inv.SupplierID, invoiceID, inv.Version)
	if err := s.repo.VoidInvoice(ctx, invoiceID, idemKey); err != nil {
		return nil, err
	}

	inv.Status = StatusVoid
	inv.BalanceMinor = 0
	inv.UpdatedAt = s.now().UTC()

	return &inv, nil
}

// RetailerPayInvoice processes payment against an invoice.
func (s *Service) RetailerPayInvoice(ctx context.Context, retailerID, invoiceID string, req RetailerPayInvoiceRequest, idempotencyKey string) (*Invoice, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("ar service unavailable")
	}
	retailerID = strings.TrimSpace(retailerID)
	invoiceID = strings.TrimSpace(invoiceID)
	if invoiceID == "" {
		return nil, fmt.Errorf("invoice_id required")
	}
	if req.AmountMinor <= 0 {
		return nil, fmt.Errorf("payment amount must be positive")
	}

	inv, err := s.repo.GetByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if claims, ok := auth.FromContext(ctx); ok {
		switch claims.Role {
		case auth.RoleAdmin:
			sid := strings.TrimSpace(claims.SupplierID)
			if sid == "" || strings.TrimSpace(inv.SupplierID) != sid {
				return nil, fmt.Errorf("%w: invoice does not belong to supplier", ErrForbidden)
			}
			if retailerID != "" && inv.RetailerID != retailerID {
				return nil, fmt.Errorf("%w: invoice does not belong to retailer", ErrForbidden)
			}
		case auth.RoleRetailer:
			resolvedOrg := auth.ResolveRetailerOrgID(claims)
			if resolvedOrg == "" || inv.RetailerID != resolvedOrg {
				return nil, fmt.Errorf("%w: invoice does not belong to retailer", ErrForbidden)
			}
			if retailerID != "" && retailerID != resolvedOrg {
				return nil, fmt.Errorf("%w: invoice does not belong to retailer", ErrForbidden)
			}
		default:
			return nil, ErrForbidden
		}
	} else {
		if retailerID == "" || inv.RetailerID != retailerID {
			return nil, fmt.Errorf("%w: invoice does not belong to retailer", ErrForbidden)
		}
	}
	if inv.Status == StatusPaid || inv.Status == StatusVoid {
		return nil, fmt.Errorf("invoice is already %s", inv.Status)
	}

	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" {
		idempotencyKey = "pay_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	} else {
		idempotencyKey = fmt.Sprintf("pay:%s:%s:%s:%s", inv.SupplierID, inv.RetailerID, invoiceID, idempotencyKey)
	}

	if err := s.applySingleInvoicePayment(ctx, inv, req.AmountMinor, req.PaymentMethod, idempotencyKey); err != nil {
		return nil, err
	}

	updated, err := s.repo.GetByID(ctx, invoiceID)
	if err != nil {
		return &inv, nil
	}
	return &updated, nil
}

func (s *Service) applySingleInvoicePayment(ctx context.Context, inv Invoice, amountMinor int64, paymentMode, idempotencyKey string) error {
	mode := strings.TrimSpace(paymentMode)
	if mode == "" {
		mode = inferARPaymentModeFromKey(idempotencyKey)
	}
	switch repo := s.repo.(type) {
	case *SpannerRepository:
		if repo == nil || repo.client == nil {
			return fmt.Errorf("spanner ar repository: nil client")
		}
		return spannerutils.RunReadWriteTransaction(ctx, repo.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			iter := txn.Query(ctx, spanner.Statement{
				SQL:    `SELECT EntryId FROM ArLedgerEntries WHERE IdempotencyKey = @k LIMIT 1`,
				Params: map[string]any{"k": idempotencyKey},
			})
			_, qerr := iter.Next()
			iter.Stop()
			if qerr == nil {
				return nil // already applied
			}
			if qerr != iterator.Done {
				return qerr
			}

			// Load current balance before applyPaymentWithModeInTxn for exact credit-clear clamping.
			row, err := txn.ReadRow(ctx, "ArInvoices", spanner.Key{inv.InvoiceID}, []string{"BalanceMinor", "Status", "OrderId"})
			if err != nil {
				return err
			}
			var curBal int64
			var curStatus, orderID string
			if err := row.Columns(&curBal, &curStatus, &orderID); err != nil {
				return err
			}
			if curStatus == StatusPaid || curStatus == StatusVoid || curBal <= 0 {
				return fmt.Errorf("invoice is already %s", curStatus)
			}
			applied := amountMinor
			if applied > curBal {
				applied = curBal
			}
			if applied < 0 {
				applied = 0
			}

			if err := applyPaymentWithModeInTxn(ctx, txn, inv.InvoiceID, amountMinor, mode, idempotencyKey); err != nil {
				return err
			}
			if s.creditClearer != nil && strings.TrimSpace(orderID) != "" && applied > 0 {
				if err := s.creditClearer(ctx, txn, inv.RetailerID, inv.SupplierID, orderID, applied); err != nil {
					return err
				}
			}

			openList, err := listSupplierOpenInvoicesInTxn(ctx, txn, inv.RetailerID, inv.SupplierID)
			if err != nil {
				return err
			}
			orderIDs := make([]string, 0, len(openList))
			for _, o := range openList {
				if o.OrderID != "" {
					orderIDs = append(orderIDs, o.OrderID)
				}
			}
			var disputes map[string]int64
			if s.disputeLookup != nil {
				disputes, err = s.disputeLookup(ctx, dedupeNonEmptyIDs(orderIDs))
			} else {
				disputes, err = queryClaimsDisputeMap(ctx, txn, orderIDs)
			}
			if err != nil {
				return err
			}

			newBal := curBal - amountMinor
			if newBal < 0 {
				newBal = 0
			}
			newSt := StatusPartial
			if newBal == 0 {
				newSt = StatusPaid
			}
			postInv := inv
			postInv.BalanceMinor = newBal
			postInv.Status = newSt

			postByID := map[string]Invoice{inv.InvoiceID: postInv}
			var postTotalUndisputed int64
			for _, o := range openList {
				if o.InvoiceID == inv.InvoiceID {
					o = postInv
				}
				postByID[o.InvoiceID] = o
				if o.Status == StatusOpen || o.Status == StatusPartial {
					_, und := clampDisputed(o.BalanceMinor, disputes[o.OrderID])
					postTotalUndisputed += und
				}
			}

			buf := outbox.NewSpannerTxnBuffer(txn)
			var mutations []*spanner.Mutation
			ts := s.now().UTC().Format(time.RFC3339Nano)
			if err := fulfillMatchingPromisesInTxn(ctx, txn, buf, inv.RetailerID, inv.SupplierID, postByID, disputes, postTotalUndisputed, ts, &mutations); err != nil {
				return err
			}
			if len(mutations) > 0 {
				if err := txn.BufferWrite(mutations); err != nil {
					return err
				}
			}
			return buf.Flush(ctx)
		})

	case *MemoryRepository:
		repo.mu.RLock()
		alreadyApplied := repo.ledgerKeys[idempotencyKey]
		orderIDs := repo.collectOpenOrderIDsLocked(inv.RetailerID, inv.SupplierID)
		repo.mu.RUnlock()
		if alreadyApplied {
			return nil
		}

		disputes, err := s.lookupDisputes(ctx, orderIDs)
		if err != nil {
			return err
		}
		applied := amountMinor
		if applied > inv.BalanceMinor {
			applied = inv.BalanceMinor
		}
		if applied < 0 {
			applied = 0
		}
		if mw, ok := s.repo.(interface {
			ApplyPaymentWithMode(context.Context, string, int64, string, string) error
		}); ok {
			if err := mw.ApplyPaymentWithMode(ctx, inv.InvoiceID, amountMinor, mode, idempotencyKey); err != nil {
				return err
			}
		} else if err := s.repo.ApplyPayment(ctx, inv.InvoiceID, amountMinor, idempotencyKey); err != nil {
			return err
		}
		repo.mu.Lock()
		repo.fulfillPromisesLocked(inv.RetailerID, inv.SupplierID, disputes, s.now().UTC())
		repo.mu.Unlock()
		if s.creditClearer != nil && strings.TrimSpace(inv.OrderID) != "" && applied > 0 {
			return s.creditClearer(ctx, nil, inv.RetailerID, inv.SupplierID, inv.OrderID, applied)
		}
		return nil

	default:
		applied := amountMinor
		if applied > inv.BalanceMinor {
			applied = inv.BalanceMinor
		}
		if applied < 0 {
			applied = 0
		}
		if mw, ok := s.repo.(interface {
			ApplyPaymentWithMode(context.Context, string, int64, string, string) error
		}); ok {
			if err := mw.ApplyPaymentWithMode(ctx, inv.InvoiceID, amountMinor, mode, idempotencyKey); err != nil {
				return err
			}
		} else if err := s.repo.ApplyPayment(ctx, inv.InvoiceID, amountMinor, idempotencyKey); err != nil {
			return err
		}
		if s.creditClearer != nil && strings.TrimSpace(inv.OrderID) != "" && applied > 0 {
			return s.creditClearer(ctx, nil, inv.RetailerID, inv.SupplierID, inv.OrderID, applied)
		}
		return nil
	}
}
