package ar

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

const (
	// LedgerEntryAdvanceCredit records overpayment routed to ArAdvanceBalances.
	LedgerEntryAdvanceCredit = "ADVANCE_CREDIT"
	// advanceLedgerInvoiceSentinel satisfies ArLedgerEntries.InvoiceId STRING(36) NOT NULL for unallocated advance credits.
	advanceLedgerInvoiceSentinel = "ADVANCE"

	EventARAdvanceCredited       = "AR_ADVANCE_BALANCE_CREDITED"
	EventARPaymentPromiseCreated = "AR_PAYMENT_PROMISE_CREATED"
	EventARPaymentPromiseStatus  = "AR_PAYMENT_PROMISE_STATUS_CHANGED"
)

var (
	ErrInvoiceNotFound       = errors.New("invoice_not_found")
	ErrInvoiceAlreadySettled = errors.New("invoice_already_settled")
	ErrInvalidFIFORequest    = errors.New("invalid_fifo_request")
)

// PayFIFORequest applies an oldest-first AR payment (or earmarked single invoice paydown)
// across open undisputed balances, routing any overpayment into ArAdvanceBalances.
type PayFIFORequest struct {
	SupplierID       string `json:"supplier_id"`
	RetailerID       string `json:"retailer_id,omitempty"`
	AmountMinor      int64  `json:"amount_minor"`
	Currency         string `json:"currency,omitempty"`
	PaymentMethod    string `json:"payment_method"`
	PaymentReference string `json:"payment_reference,omitempty"`
	EarmarkInvoiceID string `json:"earmark_invoice_id,omitempty"`
	IdempotencyKey   string `json:"idempotency_key,omitempty"`
}

// PayFIFOAllocation describes how much of a FIFO payment was applied to one invoice.
type PayFIFOAllocation struct {
	InvoiceID             string `json:"invoice_id"`
	OrderID               string `json:"order_id"`
	AppliedMinor          int64  `json:"applied_minor"`
	RemainingBalanceMinor int64  `json:"remaining_balance_minor"`
	Status                string `json:"status"`
}

// PayFIFOResponse is returned by ApplyPaymentFIFO and POST /v1/retailer/ar/pay-fifo.
type PayFIFOResponse struct {
	SupplierID          string              `json:"supplier_id"`
	RetailerID          string              `json:"retailer_id"`
	Currency            string              `json:"currency"`
	PaidMinor           int64               `json:"paid_minor"`
	AdvanceAddedMinor   int64               `json:"advance_added_minor"`
	AdvanceBalanceMinor int64               `json:"advance_balance_minor"`
	Allocations         []PayFIFOAllocation `json:"allocations"`
	IdempotencyKey      string              `json:"idempotency_key,omitempty"`
}

// LedgerEntry models an immutable ArLedgerEntries row.
type LedgerEntry struct {
	EntryID        string    `json:"entry_id"`
	InvoiceID      string    `json:"invoice_id"`
	SupplierID     string    `json:"supplier_id"`
	RetailerID     string    `json:"retailer_id"`
	EntryType      string    `json:"entry_type"`
	AmountMinor    int64     `json:"amount_minor"`
	IdempotencyKey string    `json:"idempotency_key"`
	RefOrderID     string    `json:"ref_order_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// AdvanceBalance models an ArAdvanceBalances row for (RetailerId, SupplierId).
type AdvanceBalance struct {
	RetailerID   string    `json:"retailer_id"`
	SupplierID   string    `json:"supplier_id"`
	BalanceMinor int64     `json:"balance_minor"`
	Currency     string    `json:"currency"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func fifoKeyBase(idemKey string) string {
	if len(idemKey) <= 80 {
		return idemKey
	}
	h := sha256.Sum256([]byte(idemKey))
	return hex.EncodeToString(h[:])[:48]
}

func fifoIdempotencyPrefix(idemKey string) string {
	return "fifo:" + fifoKeyBase(idemKey) + ":"
}

func fifoIdempotencyPrefixEnd(idemKey string) string {
	return "fifo:" + fifoKeyBase(idemKey) + ";"
}

func sliceFIFOIdempotencyKey(idemKey, invoiceID string) string {
	raw := fifoIdempotencyPrefix(idemKey) + invoiceID
	if len(raw) <= 128 {
		return raw
	}
	return raw[:128]
}

func advanceFIFOIdempotencyKey(idemKey string) string {
	return sliceFIFOIdempotencyKey(idemKey, "advance")
}

// ApplyPaymentFIFO allocates amount_minor either to earmark_invoice_id or oldest-first
// across undisputed OPEN/PARTIAL invoices for (retailer_id, supplier_id), routing excess
// into ArAdvanceBalances and fulfilling satisfied payment promises atomically.
func (s *Service) ApplyPaymentFIFO(ctx context.Context, req PayFIFORequest) (*PayFIFOResponse, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("ar service unavailable")
	}
	req.SupplierID = strings.TrimSpace(req.SupplierID)
	req.RetailerID = strings.TrimSpace(req.RetailerID)
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	req.PaymentMethod = strings.ToUpper(strings.TrimSpace(req.PaymentMethod))
	req.PaymentReference = strings.TrimSpace(req.PaymentReference)
	req.EarmarkInvoiceID = strings.TrimSpace(req.EarmarkInvoiceID)
	req.IdempotencyKey = strings.TrimSpace(req.IdempotencyKey)

	if claims, ok := auth.FromContext(ctx); ok {
		switch claims.Role {
		case auth.RoleRetailer:
			resolved := auth.ResolveRetailerOrgID(claims)
			if resolved == "" {
				return nil, ErrForbidden
			}
			if req.RetailerID != "" && req.RetailerID != resolved {
				return nil, fmt.Errorf("%w: retailer mismatch", ErrForbidden)
			}
			req.RetailerID = resolved
		case auth.RoleAdmin:
			if sid := strings.TrimSpace(claims.SupplierID); sid != "" {
				if req.SupplierID != "" && req.SupplierID != sid {
					return nil, fmt.Errorf("%w: supplier mismatch", ErrForbidden)
				}
				req.SupplierID = sid
			}
		default:
			return nil, ErrForbidden
		}
	}

	if req.SupplierID == "" {
		return nil, fmt.Errorf("%w: supplier_id required", ErrInvalidFIFORequest)
	}
	if req.RetailerID == "" {
		return nil, fmt.Errorf("%w: retailer_id required", ErrInvalidFIFORequest)
	}
	if req.AmountMinor <= 0 {
		return nil, fmt.Errorf("%w: amount_minor must be positive", ErrInvalidFIFORequest)
	}
	if req.PaymentMethod == "" {
		req.PaymentMethod = "WALLET"
	}
	if req.Currency == "" {
		req.Currency = "UZS"
	}
	if req.IdempotencyKey == "" {
		req.IdempotencyKey = "ff_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	}

	switch repo := s.repo.(type) {
	case *SpannerRepository:
		return repo.applyPaymentFIFO(ctx, s, req)
	case *MemoryRepository:
		return repo.applyPaymentFIFO(ctx, s, req)
	default:
		return nil, fmt.Errorf("repository does not support FIFO AR payments")
	}
}

// GetAdvanceBalance returns the stored advance overpayment credit for (retailerID, supplierID).
func (s *Service) GetAdvanceBalance(ctx context.Context, retailerID, supplierID string) (AdvanceBalance, error) {
	if s == nil || s.repo == nil {
		return AdvanceBalance{}, fmt.Errorf("ar service unavailable")
	}
	retailerID = strings.TrimSpace(retailerID)
	supplierID = strings.TrimSpace(supplierID)
	if retailerID == "" || supplierID == "" {
		return AdvanceBalance{RetailerID: retailerID, SupplierID: supplierID, Currency: "UZS"}, nil
	}
	switch repo := s.repo.(type) {
	case *SpannerRepository:
		return repo.getAdvanceBalance(ctx, retailerID, supplierID)
	case *MemoryRepository:
		return repo.getAdvanceBalance(retailerID, supplierID), nil
	default:
		return AdvanceBalance{RetailerID: retailerID, SupplierID: supplierID, Currency: "UZS"}, nil
	}
}

// ListLedgerEntries returns AR ledger entries for (retailerID, supplierID) ordered newest first.
func (s *Service) ListLedgerEntries(ctx context.Context, retailerID, supplierID string, limit int) ([]LedgerEntry, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("ar service unavailable")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	switch repo := s.repo.(type) {
	case *SpannerRepository:
		return repo.listLedgerEntries(ctx, retailerID, supplierID, limit)
	case *MemoryRepository:
		return repo.listLedgerEntries(retailerID, supplierID, limit), nil
	default:
		return []LedgerEntry{}, nil
	}
}

func (r *MemoryRepository) getAdvanceBalance(retailerID, supplierID string) AdvanceBalance {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.advanceBalances != nil {
		if bal, ok := r.advanceBalances[relKey(retailerID, supplierID)]; ok {
			return bal
		}
	}
	return AdvanceBalance{RetailerID: retailerID, SupplierID: supplierID, Currency: "UZS"}
}

func (r *MemoryRepository) listLedgerEntries(retailerID, supplierID string, limit int) []LedgerEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]LedgerEntry, 0)
	for _, e := range r.ledgerEntries {
		if retailerID != "" && e.RetailerID != retailerID {
			continue
		}
		if supplierID != "" && e.SupplierID != supplierID {
			continue
		}
		out = append(out, e)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.After(out[j].CreatedAt)
		}
		return out[i].EntryID > out[j].EntryID
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func (r *MemoryRepository) collectOpenOrderIDsLocked(retailerID, supplierID string) []string {
	var ids []string
	for _, inv := range r.byID {
		if inv.RetailerID == retailerID && inv.SupplierID == supplierID && (inv.Status == StatusOpen || inv.Status == StatusPartial) {
			if inv.OrderID != "" {
				ids = append(ids, inv.OrderID)
			}
		}
	}
	return ids
}

func clonePayFIFOResponse(in PayFIFOResponse) *PayFIFOResponse {
	cp := in
	if in.Allocations == nil {
		cp.Allocations = []PayFIFOAllocation{}
	} else {
		cp.Allocations = append([]PayFIFOAllocation(nil), in.Allocations...)
	}
	return &cp
}

func (r *MemoryRepository) findLedgerEntriesForIdempotencyLocked(idemKey string) []LedgerEntry {
	prefix := fifoIdempotencyPrefix(idemKey)
	var found []LedgerEntry
	for _, e := range r.ledgerEntries {
		if e.IdempotencyKey == idemKey || strings.HasPrefix(e.IdempotencyKey, prefix) {
			found = append(found, e)
		}
	}
	return found
}

func (r *MemoryRepository) applyPaymentFIFO(ctx context.Context, svc *Service, req PayFIFORequest) (*PayFIFOResponse, error) {
	// Resolve disputes outside r.mu if a custom lookup callback is wired.
	r.mu.RLock()
	orderIDs := r.collectOpenOrderIDsLocked(req.RetailerID, req.SupplierID)
	if req.EarmarkInvoiceID != "" {
		if inv, ok := r.byID[req.EarmarkInvoiceID]; ok && inv.OrderID != "" {
			orderIDs = append(orderIDs, inv.OrderID)
		}
	}
	r.mu.RUnlock()

	disputes, err := svc.lookupDisputes(ctx, orderIDs)
	if err != nil {
		return nil, err
	}

	now := svc.now().UTC()
	var pendingClears []PayFIFOAllocation

	r.mu.Lock()
	r.ensureMapsLocked()

	fifoCacheKey := relKey(req.RetailerID, req.SupplierID) + "|" + req.IdempotencyKey
	if cached, ok := r.fifoResults[fifoCacheKey]; ok {
		r.mu.Unlock()
		return clonePayFIFOResponse(cached), nil
	}
	// Check exact key or slice keys in ledgerEntries / ledgerKeys too.
	if r.ledgerKeys[req.IdempotencyKey] {
		entries := r.findLedgerEntriesForIdempotencyLocked(req.IdempotencyKey)
		if len(entries) > 0 && (entries[0].RetailerID != req.RetailerID || entries[0].SupplierID != req.SupplierID) {
			r.mu.Unlock()
			return nil, ErrForbidden
		}
		resp := r.reconstructFIFOFromEntriesLocked(req, entries)
		r.fifoResults[fifoCacheKey] = *resp
		r.mu.Unlock()
		return resp, nil
	}
	if existingEntries := r.findLedgerEntriesForIdempotencyLocked(req.IdempotencyKey); len(existingEntries) > 0 {
		if existingEntries[0].RetailerID != req.RetailerID || existingEntries[0].SupplierID != req.SupplierID {
			r.mu.Unlock()
			return nil, ErrForbidden
		}
		resp := r.reconstructFIFOFromEntriesLocked(req, existingEntries)
		r.fifoResults[fifoCacheKey] = *resp
		r.mu.Unlock()
		return resp, nil
	}

	var candidates []Invoice
	normalizeCurrency := func(c string) string {
		c = strings.ToUpper(strings.TrimSpace(c))
		if c == "" {
			return "UZS"
		}
		return c
	}
	reqCurrency := normalizeCurrency(req.Currency)

	if req.EarmarkInvoiceID != "" {
		inv, ok := r.byID[req.EarmarkInvoiceID]
		if !ok {
			r.mu.Unlock()
			return nil, ErrInvoiceNotFound
		}
		if inv.RetailerID != req.RetailerID || inv.SupplierID != req.SupplierID {
			r.mu.Unlock()
			return nil, ErrForbidden
		}
		if inv.Status == StatusPaid || inv.Status == StatusVoid || inv.BalanceMinor <= 0 {
			r.mu.Unlock()
			return nil, ErrInvoiceAlreadySettled
		}
		invCurrency := normalizeCurrency(inv.Currency)
		if reqCurrency != invCurrency {
			r.mu.Unlock()
			return nil, fmt.Errorf("%w: currency mismatch (%s vs %s)", ErrInvalidFIFORequest, reqCurrency, invCurrency)
		}
		candidates = []Invoice{inv}
	} else {
		for _, inv := range r.byID {
			if inv.RetailerID != req.RetailerID || inv.SupplierID != req.SupplierID {
				continue
			}
			if inv.Status != StatusOpen && inv.Status != StatusPartial {
				continue
			}
			invCurrency := normalizeCurrency(inv.Currency)
			if invCurrency != reqCurrency {
				continue
			}
			candidates = append(candidates, inv)
		}
		sortInvoicesDueAsc(candidates)
	}

	remaining := req.AmountMinor
	allocations := make([]PayFIFOAllocation, 0, len(candidates))
	var paidMinor int64

	for _, inv := range candidates {
		if remaining <= 0 {
			break
		}
		eligibleCap := inv.BalanceMinor
		if req.EarmarkInvoiceID == "" {
			_, eligibleCap = clampDisputed(inv.BalanceMinor, disputes[inv.OrderID])
		}
		if eligibleCap <= 0 {
			continue
		}
		apply := remaining
		if apply > eligibleCap {
			apply = eligibleCap
		}
		sliceKey := sliceFIFOIdempotencyKey(req.IdempotencyKey, inv.InvoiceID)
		inv.BalanceMinor -= apply
		if inv.BalanceMinor < 0 {
			inv.BalanceMinor = 0
		}
		if inv.BalanceMinor == 0 {
			inv.Status = StatusPaid
		} else {
			inv.Status = StatusPartial
		}
		inv.Version++
		inv.UpdatedAt = now
		r.byID[inv.InvoiceID] = inv
		r.ledgerKeys[sliceKey] = true
		r.ledgerEntries = append(r.ledgerEntries, LedgerEntry{
			EntryID:        ledgerEntryID("pay", sliceKey),
			InvoiceID:      inv.InvoiceID,
			SupplierID:     req.SupplierID,
			RetailerID:     req.RetailerID,
			EntryType:      "PAYMENT",
			AmountMinor:    -apply,
			IdempotencyKey: sliceKey,
			RefOrderID:     inv.OrderID,
			CreatedAt:      now,
		})
		alloc := PayFIFOAllocation{
			InvoiceID:             inv.InvoiceID,
			OrderID:               inv.OrderID,
			AppliedMinor:          apply,
			RemainingBalanceMinor: inv.BalanceMinor,
			Status:                inv.Status,
		}
		allocations = append(allocations, alloc)
		if inv.OrderID != "" {
			pendingClears = append(pendingClears, alloc)
		}
		paidMinor += apply
		remaining -= apply
	}

	excessMinor := remaining
	advBal := r.advanceBalances[relKey(req.RetailerID, req.SupplierID)]
	if advBal.RetailerID == "" {
		advBal.RetailerID = req.RetailerID
		advBal.SupplierID = req.SupplierID
		advBal.Currency = req.Currency
	}
	if excessMinor > 0 {
		advKey := advanceFIFOIdempotencyKey(req.IdempotencyKey)
		advBal.BalanceMinor += excessMinor
		if advBal.Currency == "" {
			advBal.Currency = req.Currency
		}
		advBal.UpdatedAt = now
		r.advanceBalances[relKey(req.RetailerID, req.SupplierID)] = advBal
		r.ledgerKeys[advKey] = true
		r.ledgerEntries = append(r.ledgerEntries, LedgerEntry{
			EntryID:        ledgerEntryID("adv", advKey),
			InvoiceID:      advanceLedgerInvoiceSentinel,
			SupplierID:     req.SupplierID,
			RetailerID:     req.RetailerID,
			EntryType:      LedgerEntryAdvanceCredit,
			AmountMinor:    excessMinor,
			IdempotencyKey: advKey,
			CreatedAt:      now,
		})
	}
	r.ledgerKeys[req.IdempotencyKey] = true

	// Fulfill matching ACTIVE promises inside the same critical section.
	r.fulfillPromisesLocked(req.RetailerID, req.SupplierID, disputes, now)

	resp := PayFIFOResponse{
		SupplierID:          req.SupplierID,
		RetailerID:          req.RetailerID,
		Currency:            req.Currency,
		PaidMinor:           paidMinor,
		AdvanceAddedMinor:   excessMinor,
		AdvanceBalanceMinor: advBal.BalanceMinor,
		Allocations:         allocations,
		IdempotencyKey:      req.IdempotencyKey,
	}
	r.fifoResults[fifoCacheKey] = *clonePayFIFOResponse(resp)
	r.mu.Unlock()

	if svc.creditClearer != nil {
		for _, alloc := range pendingClears {
			if err := svc.creditClearer(ctx, nil, req.RetailerID, req.SupplierID, alloc.OrderID, alloc.AppliedMinor); err != nil {
				return nil, err
			}
		}
	}

	return clonePayFIFOResponse(resp), nil
}

func (r *MemoryRepository) reconstructFIFOFromEntriesLocked(req PayFIFORequest, entries []LedgerEntry) *PayFIFOResponse {
	allocations := make([]PayFIFOAllocation, 0, len(entries))
	var paidMinor, advAdded int64
	for _, e := range entries {
		switch e.EntryType {
		case "PAYMENT":
			applied := -e.AmountMinor
			if applied < 0 {
				applied = 0
			}
			inv := r.byID[e.InvoiceID]
			allocations = append(allocations, PayFIFOAllocation{
				InvoiceID:             e.InvoiceID,
				OrderID:               inv.OrderID,
				AppliedMinor:          applied,
				RemainingBalanceMinor: inv.BalanceMinor,
				Status:                inv.Status,
			})
			paidMinor += applied
		case LedgerEntryAdvanceCredit:
			if e.AmountMinor > 0 {
				advAdded += e.AmountMinor
			}
		}
	}
	adv := r.advanceBalances[relKey(req.RetailerID, req.SupplierID)]
	cur := req.Currency
	if cur == "" {
		cur = adv.Currency
	}
	if cur == "" {
		cur = "UZS"
	}
	return &PayFIFOResponse{
		SupplierID:          req.SupplierID,
		RetailerID:          req.RetailerID,
		Currency:            cur,
		PaidMinor:           paidMinor,
		AdvanceAddedMinor:   advAdded,
		AdvanceBalanceMinor: adv.BalanceMinor,
		Allocations:         allocations,
		IdempotencyKey:      req.IdempotencyKey,
	}
}

func (r *MemoryRepository) fulfillPromisesLocked(retailerID, supplierID string, disputes map[string]int64, now time.Time) {
	if len(r.promises) == 0 {
		return
	}
	var totalUndisputed int64
	for _, inv := range r.byID {
		if inv.RetailerID != retailerID || inv.SupplierID != supplierID {
			continue
		}
		if inv.Status != StatusOpen && inv.Status != StatusPartial {
			continue
		}
		_, und := clampDisputed(inv.BalanceMinor, disputes[inv.OrderID])
		totalUndisputed += und
	}

	for id, p := range r.promises {
		if p.RetailerID != retailerID || p.SupplierID != supplierID || p.Status != PromiseStatusActive {
			continue
		}
		fulfilled := totalUndisputed <= 0
		if !fulfilled && p.InvoiceID != "" {
			if inv, ok := r.byID[p.InvoiceID]; ok {
				_, und := clampDisputed(inv.BalanceMinor, disputes[inv.OrderID])
				if inv.BalanceMinor <= 0 || inv.Status == StatusPaid || inv.Status == StatusVoid || und <= 0 {
					fulfilled = true
				}
			}
		}
		if fulfilled {
			p.Status = PromiseStatusFulfilled
			p.UpdatedAt = now
			r.promises[id] = p
		}
	}
}

func (r *SpannerRepository) getAdvanceBalance(ctx context.Context, retailerID, supplierID string) (AdvanceBalance, error) {
	out := AdvanceBalance{RetailerID: retailerID, SupplierID: supplierID, Currency: "UZS"}
	if r == nil || r.client == nil {
		return out, nil
	}
	row, err := r.client.Single().ReadRow(ctx, "ArAdvanceBalances", spanner.Key{retailerID, supplierID},
		[]string{"BalanceMinor", "Currency", "UpdatedAt"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return out, nil
		}
		return AdvanceBalance{}, err
	}
	if err := row.Columns(&out.BalanceMinor, &out.Currency, &out.UpdatedAt); err != nil {
		return AdvanceBalance{}, err
	}
	return out, nil
}

func (r *SpannerRepository) listLedgerEntries(ctx context.Context, retailerID, supplierID string, limit int) ([]LedgerEntry, error) {
	if r == nil || r.client == nil {
		return []LedgerEntry{}, nil
	}
	var clauses []string
	params := map[string]any{"limit": int64(limit)}
	if retailerID != "" {
		clauses = append(clauses, "RetailerId = @rid")
		params["rid"] = retailerID
	}
	if supplierID != "" {
		clauses = append(clauses, "SupplierId = @sid")
		params["sid"] = supplierID
	}
	sql := `SELECT EntryId, InvoiceId, SupplierId, RetailerId, EntryType, AmountMinor, IdempotencyKey, IFNULL(RefOrderId, ''), CreatedAt FROM ArLedgerEntries`
	if len(clauses) > 0 {
		sql += " WHERE " + strings.Join(clauses, " AND ")
	}
	sql += " ORDER BY CreatedAt DESC, EntryId DESC LIMIT @limit"

	iter := r.client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	out := []LedgerEntry{}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		var e LedgerEntry
		if err := row.Columns(&e.EntryID, &e.InvoiceID, &e.SupplierID, &e.RetailerID, &e.EntryType, &e.AmountMinor, &e.IdempotencyKey, &e.RefOrderID, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
}

func queryExistingFIFOEntriesInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, idemKey string) ([]LedgerEntry, error) {
	iter := txn.Query(ctx, spanner.Statement{
		SQL: `SELECT EntryId, InvoiceId, SupplierId, RetailerId, EntryType, AmountMinor, IdempotencyKey, IFNULL(RefOrderId, ''), CreatedAt
			FROM ArLedgerEntries
			WHERE IdempotencyKey = @exact OR (IdempotencyKey >= @prefix AND IdempotencyKey < @prefixEnd)
			ORDER BY CreatedAt ASC, EntryId ASC`,
		Params: map[string]any{
			"exact":     idemKey,
			"prefix":    fifoIdempotencyPrefix(idemKey),
			"prefixEnd": fifoIdempotencyPrefixEnd(idemKey),
		},
	})
	defer iter.Stop()
	var entries []LedgerEntry
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return entries, nil
		}
		if err != nil {
			return nil, err
		}
		var e LedgerEntry
		if err := row.Columns(&e.EntryID, &e.InvoiceID, &e.SupplierID, &e.RetailerID, &e.EntryType, &e.AmountMinor, &e.IdempotencyKey, &e.RefOrderID, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
}

func readAdvanceBalanceInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, retailerID, supplierID string) (int64, string, error) {
	row, err := txn.ReadRow(ctx, "ArAdvanceBalances", spanner.Key{retailerID, supplierID}, []string{"BalanceMinor", "Currency"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return 0, "UZS", nil
		}
		return 0, "", err
	}
	var bal int64
	var cur string
	if err := row.Columns(&bal, &cur); err != nil {
		return 0, "", err
	}
	return bal, cur, nil
}

func listSupplierOpenInvoicesInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, retailerID, supplierID string, optionalCurrency ...string) ([]Invoice, error) {
	var targetCur string
	if len(optionalCurrency) > 0 && strings.TrimSpace(optionalCurrency[0]) != "" {
		targetCur = strings.ToUpper(strings.TrimSpace(optionalCurrency[0]))
	}

	iter := txn.Query(ctx, spanner.Statement{
		SQL: `SELECT ` + invoiceSelectColumns + `
			FROM ArInvoices
			WHERE RetailerId = @rid AND SupplierId = @sid AND Status IN ('OPEN', 'PARTIAL')
			ORDER BY DueAt ASC, InvoiceId ASC`,
		Params: map[string]any{"rid": retailerID, "sid": supplierID},
	})
	defer iter.Stop()
	var list []Invoice
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return list, nil
		}
		if err != nil {
			return nil, err
		}
		inv, err := scanInvoice(row)
		if err != nil {
			return nil, err
		}
		if targetCur != "" {
			invCur := strings.ToUpper(strings.TrimSpace(inv.Currency))
			if invCur == "" {
				invCur = "UZS"
			}
			if invCur != targetCur {
				continue
			}
		}
		list = append(list, inv)
	}
}

func (r *SpannerRepository) applyPaymentFIFO(ctx context.Context, svc *Service, req PayFIFORequest) (*PayFIFOResponse, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("spanner ar repository: nil client")
	}
	var result PayFIFOResponse
	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		existing, err := queryExistingFIFOEntriesInTxn(ctx, txn, req.IdempotencyKey)
		if err != nil {
			return err
		}
		if len(existing) > 0 {
			if existing[0].RetailerID != req.RetailerID || existing[0].SupplierID != req.SupplierID {
				return ErrForbidden
			}
			advBal, _, err := readAdvanceBalanceInTxn(ctx, txn, req.RetailerID, req.SupplierID)
			if err != nil {
				return err
			}
			allocations := make([]PayFIFOAllocation, 0, len(existing))
			var paidMinor, advAdded int64
			for _, e := range existing {
				switch e.EntryType {
				case "PAYMENT":
					applied := -e.AmountMinor
					if applied < 0 {
						applied = 0
					}
					row, err := txn.ReadRow(ctx, "ArInvoices", spanner.Key{e.InvoiceID}, []string{"OrderId", "BalanceMinor", "Status"})
					if err != nil {
						return err
					}
					var oid, st string
					var rem int64
					if err := row.Columns(&oid, &rem, &st); err != nil {
						return err
					}
					allocations = append(allocations, PayFIFOAllocation{
						InvoiceID:             e.InvoiceID,
						OrderID:               oid,
						AppliedMinor:          applied,
						RemainingBalanceMinor: rem,
						Status:                st,
					})
					paidMinor += applied
				case LedgerEntryAdvanceCredit:
					if e.AmountMinor > 0 {
						advAdded += e.AmountMinor
					}
				}
			}
			result = PayFIFOResponse{
				SupplierID:          req.SupplierID,
				RetailerID:          req.RetailerID,
				Currency:            req.Currency,
				PaidMinor:           paidMinor,
				AdvanceAddedMinor:   advAdded,
				AdvanceBalanceMinor: advBal,
				Allocations:         allocations,
				IdempotencyKey:      req.IdempotencyKey,
			}
			return nil
		}

		reqCurrency := strings.ToUpper(strings.TrimSpace(req.Currency))
		if reqCurrency == "" {
			reqCurrency = "UZS"
		}

		openList, err := listSupplierOpenInvoicesInTxn(ctx, txn, req.RetailerID, req.SupplierID, reqCurrency)
		if err != nil {
			return err
		}

		var candidates []Invoice
		var allOrderIDs []string
		for _, inv := range openList {
			if inv.OrderID != "" {
				allOrderIDs = append(allOrderIDs, inv.OrderID)
			}
		}

		if req.EarmarkInvoiceID != "" {
			var found *Invoice
			for i := range openList {
				if openList[i].InvoiceID == req.EarmarkInvoiceID {
					found = &openList[i]
					break
				}
			}
			if found == nil {
				row, err := txn.ReadRow(ctx, "ArInvoices", spanner.Key{req.EarmarkInvoiceID},
					[]string{"SupplierId", "RetailerId", "Status", "Currency"})
				if err != nil {
					if spanner.ErrCode(err) == codes.NotFound {
						return ErrInvoiceNotFound
					}
					return err
				}
				var sid, rid, st, cur string
				if err := row.Columns(&sid, &rid, &st, &cur); err != nil {
					return err
				}
				if sid != req.SupplierID || rid != req.RetailerID {
					return ErrForbidden
				}
				invCur := strings.ToUpper(strings.TrimSpace(cur))
				if invCur == "" {
					invCur = "UZS"
				}
				if reqCurrency != invCur {
					return fmt.Errorf("%w: currency mismatch (%s vs %s)", ErrInvalidFIFORequest, reqCurrency, invCur)
				}
				return ErrInvoiceAlreadySettled
			}
			invCur := strings.ToUpper(strings.TrimSpace(found.Currency))
			if invCur == "" {
				invCur = "UZS"
			}
			if reqCurrency != invCur {
				return fmt.Errorf("%w: currency mismatch (%s vs %s)", ErrInvalidFIFORequest, reqCurrency, invCur)
			}
			candidates = []Invoice{*found}
		} else {
			candidates = openList
		}

		var disputes map[string]int64
		if svc.disputeLookup != nil {
			disputes, err = svc.disputeLookup(ctx, dedupeNonEmptyIDs(allOrderIDs))
		} else {
			disputes, err = queryClaimsDisputeMap(ctx, txn, allOrderIDs)
		}
		if err != nil {
			return err
		}

		remaining := req.AmountMinor
		allocations := make([]PayFIFOAllocation, 0, len(candidates))
		updatedInvoices := make(map[string]Invoice, len(candidates))
		var paidMinor int64
		buf := outbox.NewSpannerTxnBuffer(txn)
		var mutations []*spanner.Mutation
		glInputs := make([]gl.JournalEntryInput, 0, len(candidates)+1)
		now := svc.now().UTC()
		ts := now.Format(time.RFC3339Nano)
		paymentMode := normalizeARPaymentMode(req.PaymentMethod)

		for _, inv := range candidates {
			if remaining <= 0 {
				break
			}
			capAmt := inv.BalanceMinor
			if req.EarmarkInvoiceID == "" {
				_, capAmt = clampDisputed(inv.BalanceMinor, disputes[inv.OrderID])
			}
			if capAmt <= 0 {
				continue
			}
			apply := remaining
			if apply > capAmt {
				apply = capAmt
			}
			newBal := inv.BalanceMinor - apply
			if newBal < 0 {
				newBal = 0
			}
			newStatus := StatusPartial
			eventType := events.EventARInvoicePayment
			if newBal == 0 {
				newStatus = StatusPaid
				eventType = events.EventARInvoiceSettled
			}
			sliceKey := sliceFIFOIdempotencyKey(req.IdempotencyKey, inv.InvoiceID)
			if err := outbox.EmitJSON(ctx, buf, events.AggregateARInvoice, inv.InvoiceID, events.TopicMain, events.ARInvoiceEvent{
				BaseEvent:    events.BaseEvent{Type: eventType, Timestamp: ts},
				InvoiceID:    inv.InvoiceID,
				SupplierID:   inv.SupplierID,
				RetailerID:   inv.RetailerID,
				OrderID:      inv.OrderID,
				AmountMinor:  apply,
				BalanceMinor: newBal,
				Status:       newStatus,
			}); err != nil {
				return err
			}
			mutations = append(mutations,
				spanner.UpdateMap("ArInvoices", map[string]any{
					"InvoiceId":    inv.InvoiceID,
					"BalanceMinor": newBal,
					"Status":       newStatus,
					"Version":      inv.Version + 1,
					"UpdatedAt":    spanner.CommitTimestamp,
				}),
				spanner.InsertMap("ArLedgerEntries", map[string]any{
					"EntryId":        ledgerEntryID("pay", sliceKey),
					"InvoiceId":      inv.InvoiceID,
					"SupplierId":     inv.SupplierID,
					"RetailerId":     inv.RetailerID,
					"EntryType":      "PAYMENT",
					"AmountMinor":    -apply,
					"IdempotencyKey": sliceKey,
					"RefOrderId":     inv.OrderID,
					"CreatedAt":      spanner.CommitTimestamp,
				}),
			)
			invCur := strings.ToUpper(strings.TrimSpace(inv.Currency))
			if invCur == "" {
				invCur = reqCurrency
			}
			if invCur == "" {
				invCur = "UZS"
			}
			var glIn gl.JournalEntryInput
			var glErr error
			if paymentMode == "WALLET" {
				glIn = buildWalletARPaymentGLInput(
					inv.SupplierID, inv.RetailerID, inv.OrderID, inv.InvoiceID, invCur,
					"gl:ar:fifo:"+sliceKey,
					fmt.Sprintf("FIFO AR payment %s invoice %s", strings.TrimSpace(req.PaymentReference), inv.InvoiceID),
					apply, now,
				)
			} else {
				glIn, glErr = gl.BuildARPaymentInput(gl.RuleEvalContext{
					EntityID:       inv.SupplierID,
					SupplierID:     inv.SupplierID,
					RetailerID:     inv.RetailerID,
					OrderID:        inv.OrderID,
					SourceID:       inv.InvoiceID,
					Currency:       invCur,
					PaymentMode:    paymentMode,
					GrossMinor:     apply,
					IdempotencyKey: "gl:ar:fifo:" + sliceKey,
					Memo:           fmt.Sprintf("FIFO AR payment %s invoice %s", strings.TrimSpace(req.PaymentReference), inv.InvoiceID),
					OccurredAt:     now,
				})
				if glErr != nil {
					return glErr
				}
			}
			glInputs = append(glInputs, glIn)
			inv.BalanceMinor = newBal
			inv.Status = newStatus
			updatedInvoices[inv.InvoiceID] = inv

			allocations = append(allocations, PayFIFOAllocation{
				InvoiceID:             inv.InvoiceID,
				OrderID:               inv.OrderID,
				AppliedMinor:          apply,
				RemainingBalanceMinor: newBal,
				Status:                newStatus,
			})
			if svc.creditClearer != nil && strings.TrimSpace(inv.OrderID) != "" {
				if err := svc.creditClearer(ctx, txn, req.RetailerID, req.SupplierID, inv.OrderID, apply); err != nil {
					return err
				}
			}
			paidMinor += apply
			remaining -= apply
		}

		curAdv, curAdvCurrency, err := readAdvanceBalanceInTxn(ctx, txn, req.RetailerID, req.SupplierID)
		if err != nil {
			return err
		}
		excessMinor := remaining
		newAdv := curAdv + excessMinor
		if excessMinor > 0 {
			advKey := advanceFIFOIdempotencyKey(req.IdempotencyKey)
			cur := req.Currency
			if cur == "" {
				cur = curAdvCurrency
			}
			if cur == "" {
				cur = "UZS"
			}
			if err := outbox.EmitJSON(ctx, buf, events.AggregateARInvoice, advanceLedgerInvoiceSentinel+":"+req.RetailerID+":"+req.SupplierID, events.TopicMain, events.ARInvoiceEvent{
				BaseEvent:    events.BaseEvent{Type: EventARAdvanceCredited, Timestamp: ts},
				InvoiceID:    advanceLedgerInvoiceSentinel,
				SupplierID:   req.SupplierID,
				RetailerID:   req.RetailerID,
				AmountMinor:  excessMinor,
				BalanceMinor: newAdv,
				Status:       LedgerEntryAdvanceCredit,
			}); err != nil {
				return err
			}
			mutations = append(mutations,
				spanner.InsertOrUpdateMap("ArAdvanceBalances", map[string]any{
					"RetailerId":   req.RetailerID,
					"SupplierId":   req.SupplierID,
					"BalanceMinor": newAdv,
					"Currency":     cur,
					"UpdatedAt":    spanner.CommitTimestamp,
				}),
				spanner.InsertMap("ArLedgerEntries", map[string]any{
					"EntryId":        ledgerEntryID("adv", advKey),
					"InvoiceId":      advanceLedgerInvoiceSentinel,
					"SupplierId":     req.SupplierID,
					"RetailerId":     req.RetailerID,
					"EntryType":      LedgerEntryAdvanceCredit,
					"AmountMinor":    excessMinor,
					"IdempotencyKey": advKey,
					"CreatedAt":      spanner.CommitTimestamp,
				}),
			)
			advGLIn, advErr := gl.EvaluateEventToJournalInput(ctx, gl.DefaultPostingRules(""), gl.RuleEventARAdvanceCredited, gl.RuleEvalContext{
				EntityID:       req.SupplierID,
				SupplierID:     req.SupplierID,
				RetailerID:     req.RetailerID,
				SourceID:       advanceLedgerInvoiceSentinel + ":" + req.RetailerID + ":" + req.SupplierID,
				Currency:       cur,
				PaymentMode:    paymentMode,
				GrossMinor:     excessMinor,
				IdempotencyKey: "gl:ar:adv:" + advKey,
				Memo:           fmt.Sprintf("FIFO overpayment advance credit %s", strings.TrimSpace(req.PaymentReference)),
				OccurredAt:     now,
			})
			if advErr != nil {
				return advErr
			}
			glInputs = append(glInputs, advGLIn)
		}

		// Merge updated invoice states back into openList to check promise fulfillment cleanly.
		postByID := make(map[string]Invoice, len(openList))
		var postTotalUndisputed int64
		for _, inv := range openList {
			if u, ok := updatedInvoices[inv.InvoiceID]; ok {
				inv = u
			}
			postByID[inv.InvoiceID] = inv
			if inv.Status == StatusOpen || inv.Status == StatusPartial {
				_, und := clampDisputed(inv.BalanceMinor, disputes[inv.OrderID])
				postTotalUndisputed += und
			}
		}
		if err := fulfillMatchingPromisesInTxn(ctx, txn, buf, req.RetailerID, req.SupplierID, postByID, disputes, postTotalUndisputed, ts, &mutations); err != nil {
			return err
		}

		if len(mutations) > 0 {
			if err := txn.BufferWrite(mutations); err != nil {
				return err
			}
		}
		if len(glInputs) > 0 {
			if _, err := gl.BufferJournalEntriesBatchInTxn(ctx, txn, buf, glInputs); err != nil {
				return err
			}
		}
		if err := buf.Flush(ctx); err != nil {
			return err
		}

		result = PayFIFOResponse{
			SupplierID:          req.SupplierID,
			RetailerID:          req.RetailerID,
			Currency:            req.Currency,
			PaidMinor:           paidMinor,
			AdvanceAddedMinor:   excessMinor,
			AdvanceBalanceMinor: newAdv,
			Allocations:         allocations,
			IdempotencyKey:      req.IdempotencyKey,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}
