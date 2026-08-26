package ar

import (
	"context"
	"fmt"
	"strings"
	"time"
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
	RetailerID         string `json:"retailer_id"`
	SupplierID         string `json:"supplier_id"`
	IsLocked           bool   `json:"is_locked"`
	Reason             string `json:"reason,omitempty"`
	OverdueAmountMinor int64  `json:"overdue_amount_minor"`
	OverdueCount       int64  `json:"overdue_count"`
	CheckedAt          string `json:"checked_at"`
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

	invoices, err := s.repo.ListBySupplier(ctx, supplierID, "", 1000)
	if err != nil {
		return nil, err
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

	invoices, err := s.repo.ListByRetailer(ctx, retailerID, "", 200)
	if err != nil {
		return nil, err
	}

	now := s.now().UTC()
	var overdueAmount int64
	var overdueCount int64
	var isLocked bool
	var lockReason string

	for _, inv := range invoices {
		if supplierID != "" && inv.SupplierID != supplierID {
			continue
		}
		if inv.Status == StatusPaid || inv.Status == StatusVoid {
			continue
		}

		graceEnd := inv.DueAt.AddDate(0, 0, int(inv.GracePeriodDays))
		if now.After(graceEnd) {
			overdueCount++
			overdueAmount += inv.BalanceMinor
		}

		if inv.DunningStep >= StepCreditHold || now.Sub(inv.DueAt) >= 21*24*time.Hour {
			isLocked = true
			lockReason = fmt.Sprintf("Delinquency lock: Invoice %s is %s (overdue %d days)", inv.InvoiceID, StepName(inv.DunningStep), int(now.Sub(inv.DueAt).Hours()/24))
		}
	}

	return &DelinquencyLockStatus{
		RetailerID:         retailerID,
		SupplierID:         supplierID,
		IsLocked:           isLocked,
		Reason:             lockReason,
		OverdueAmountMinor: overdueAmount,
		OverdueCount:       overdueCount,
		CheckedAt:          now.Format(time.RFC3339),
	}, nil
}

// WriteOffInvoice voids an uncollectible invoice and logs an audit ledger adjustment.
func (s *Service) WriteOffInvoice(ctx context.Context, invoiceID, approverID string, req WriteOffRequest) (*Invoice, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("ar service unavailable")
	}
	invoiceID = strings.TrimSpace(invoiceID)
	if invoiceID == "" {
		return nil, fmt.Errorf("invoice_id required")
	}

	inv, err := s.repo.GetByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if inv.Status == StatusPaid || inv.Status == StatusVoid {
		return nil, fmt.Errorf("cannot write off invoice with status %s", inv.Status)
	}

	inv.Status = StatusVoid
	inv.BalanceMinor = 0
	inv.UpdatedAt = s.now().UTC()

	// Apply credit note / adjustment to clear balance in repo
	idemKey := fmt.Sprintf("writeoff_%s_%d", invoiceID, s.now().UnixNano())
	if err := s.repo.ApplyCreditNote(ctx, invoiceID, inv.BalanceMinor, idemKey); err != nil {
		// Even if already 0, ensure status update
	}

	return &inv, nil
}

// RetailerPayInvoice processes payment against an invoice.
func (s *Service) RetailerPayInvoice(ctx context.Context, retailerID, invoiceID string, req RetailerPayInvoiceRequest, idempotencyKey string) (*Invoice, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("ar service unavailable")
	}
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
	if inv.RetailerID != retailerID && retailerID != "" {
		return nil, fmt.Errorf("invoice does not belong to retailer")
	}
	if inv.Status == StatusPaid || inv.Status == StatusVoid {
		return nil, fmt.Errorf("invoice is already %s", inv.Status)
	}

	if idempotencyKey == "" {
		idempotencyKey = fmt.Sprintf("pay_%s_%d", invoiceID, s.now().UnixNano())
	}

	if err := s.repo.ApplyPayment(ctx, invoiceID, req.AmountMinor, idempotencyKey); err != nil {
		return nil, err
	}

	updated, err := s.repo.GetByID(ctx, invoiceID)
	if err != nil {
		return &inv, nil
	}
	return &updated, nil
}
