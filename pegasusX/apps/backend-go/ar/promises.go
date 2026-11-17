package ar

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

const (
	PromiseStatusActive     = "ACTIVE"
	PromiseStatusFulfilled  = "FULFILLED"
	PromiseStatusBreached   = "BREACHED"
	PromiseStatusCancelled  = "CANCELLED"
	PromiseApprovedBySystem = "SYSTEM_POLICY"

	defaultMaxPromiseDays = 7
)

var (
	ErrQuarterlyPromiseLimitReached = errors.New("quarterly_promise_limit_reached")
	ErrPromisePrepayLocked          = errors.New("prepay_only_locked")
	ErrPromiseProfileFrozen         = errors.New("credit_profile_frozen")
	ErrPromiseInvalidExtension      = errors.New("invalid_extension_days")
	ErrPromiseNoBalance             = errors.New("no_undisputed_balance")
)

// QuarterlyLimitError carries the conflicting QuarterKey for HTTP 409 formatting.
type QuarterlyLimitError struct {
	QuarterKey string
	Message    string
}

func (e *QuarterlyLimitError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("quarterly_promise_limit_reached (%s)", e.QuarterKey)
}

func (e *QuarterlyLimitError) Unwrap() error {
	return ErrQuarterlyPromiseLimitReached
}

// PaymentPromise represents one ArPaymentPromises row.
type PaymentPromise struct {
	PromiseID     string    `json:"promise_id"`
	RetailerID    string    `json:"retailer_id"`
	SupplierID    string    `json:"supplier_id"`
	InvoiceID     string    `json:"invoice_id,omitempty"`
	QuarterKey    string    `json:"quarter_key"`
	PromisedAt    time.Time `json:"promised_at"`
	DueAt         time.Time `json:"due_at"`
	PromisedMinor int64     `json:"promised_minor"`
	Status        string    `json:"status"`
	ApprovedBy    string    `json:"approved_by"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// StatementInvoice extends Invoice with live disputed vs undisputed splits.
type StatementInvoice struct {
	InvoiceID       string    `json:"invoice_id"`
	SupplierID      string    `json:"supplier_id"`
	RetailerID      string    `json:"retailer_id"`
	OrderID         string    `json:"order_id"`
	Status          string    `json:"status"`
	PrincipalMinor  int64     `json:"principal_minor"`
	BalanceMinor    int64     `json:"balance_minor"`
	DisputedMinor   int64     `json:"disputed_minor"`
	UndisputedMinor int64     `json:"undisputed_minor"`
	Currency        string    `json:"currency"`
	CreditLeaveAt   time.Time `json:"credit_leave_at"`
	DueAt           time.Time `json:"due_at"`
	TermsDays       int64     `json:"terms_days"`
	GracePeriodDays int64     `json:"grace_period_days"`
	AgingBucket     string    `json:"aging_bucket,omitempty"`
	LastDunnedAt    time.Time `json:"last_dunned_at,omitempty"`
	DunningStep     int64     `json:"dunning_step"`
	Version         int64     `json:"version"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// StatementResponse is the unified self-serve AR statement for retailers.
type StatementResponse struct {
	RetailerID           string             `json:"retailer_id"`
	SupplierID           string             `json:"supplier_id"`
	Currency             string             `json:"currency"`
	TotalOpenMinor       int64              `json:"total_open_minor"`
	TotalDisputedMinor   int64              `json:"total_disputed_minor"`
	TotalUndisputedMinor int64              `json:"total_undisputed_minor"`
	AdvanceBalanceMinor  int64              `json:"advance_balance_minor"`
	CurrentQuarterKey    string             `json:"current_quarter_key"`
	SelfServePromiseUsed bool               `json:"self_serve_promise_used"`
	MaxExtensionDays     int                `json:"max_extension_days"`
	Invoices             []StatementInvoice `json:"invoices"`
	Promises             []PaymentPromise   `json:"promises"`
	LedgerEntries        []LedgerEntry      `json:"ledger_entries"`
	NextCursor           string             `json:"next_cursor,omitempty"`
}

// MaxPromiseDays returns AR_MAX_PROMISE_DAYS (default 7).
func MaxPromiseDays() int {
	raw := strings.TrimSpace(os.Getenv("AR_MAX_PROMISE_DAYS"))
	if raw == "" {
		return defaultMaxPromiseDays
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultMaxPromiseDays
	}
	return n
}

func resolveLocation(tzName string) *time.Location {
	tz := strings.TrimSpace(tzName)
	if tz == "" {
		tz = "Asia/Tashkent"
	}
	if loc, err := time.LoadLocation(tz); err == nil {
		return loc
	}
	if tz == "Asia/Tashkent" {
		return time.FixedZone("Asia/Tashkent", 5*3600)
	}
	if loc, err := time.LoadLocation("Asia/Tashkent"); err == nil {
		return loc
	}
	return time.FixedZone("Asia/Tashkent", 5*3600)
}

// QuarterKey formats now in tzName (default Asia/Tashkent) as "YYYY-Q1".."YYYY-Q4".
func QuarterKey(now time.Time, tzName string) string {
	loc := resolveLocation(tzName)
	local := now.In(loc)
	q := (int(local.Month())-1)/3 + 1
	return fmt.Sprintf("%04d-Q%d", local.Year(), q)
}

func isBlockedProfileStatus(st string) bool {
	switch strings.ToUpper(strings.TrimSpace(st)) {
	case "FROZEN", "BLACKLISTED", "CLOSED":
		return true
	default:
		return false
	}
}

func hasActiveUnexpiredPromise(promises []PaymentPromise, inv Invoice, now time.Time) (PaymentPromise, bool) {
	for _, p := range promises {
		if p.Status != PromiseStatusActive || p.DueAt.Before(now) {
			continue
		}
		if p.RetailerID != inv.RetailerID || p.SupplierID != inv.SupplierID {
			continue
		}
		if p.InvoiceID == inv.InvoiceID || p.InvoiceID == "" {
			return p, true
		}
	}
	return PaymentPromise{}, false
}

func (s *Service) resolveSupplierTimezone(ctx context.Context, supplierID string) string {
	if s == nil || supplierID == "" {
		return "Asia/Tashkent"
	}
	if sr, ok := s.repo.(*SpannerRepository); ok && sr.client != nil {
		row, err := sr.client.Single().ReadRow(ctx, "SupplierCreditPrograms", spanner.Key{supplierID}, []string{"Timezone"})
		if err == nil {
			var tz spanner.NullString
			if colErr := row.Columns(&tz); colErr == nil && tz.Valid && strings.TrimSpace(tz.StringVal) != "" {
				return strings.TrimSpace(tz.StringVal)
			}
		}
	}
	return "Asia/Tashkent"
}

func computePromiseDueAt(now time.Time, anchorInvoices []Invoice, extensionDays int) time.Time {
	base := now.UTC()
	for _, inv := range anchorInvoices {
		graceEnd := inv.DueAt.UTC().AddDate(0, 0, int(inv.GracePeriodDays))
		if graceEnd.After(base) {
			base = graceEnd
		}
	}
	return base.AddDate(0, 0, extensionDays)
}

func (s *Service) authorizeRetailerAction(ctx context.Context, retailerID, supplierID string) (string, string, error) {
	retailerID = strings.TrimSpace(retailerID)
	supplierID = strings.TrimSpace(supplierID)
	if claims, ok := auth.FromContext(ctx); ok {
		switch claims.Role {
		case auth.RoleRetailer:
			resolved := auth.ResolveRetailerOrgID(claims)
			if resolved == "" {
				return "", "", ErrForbidden
			}
			if retailerID != "" && retailerID != resolved {
				return "", "", ErrForbidden
			}
			retailerID = resolved
		case auth.RoleAdmin:
			if sid := strings.TrimSpace(claims.SupplierID); sid != "" {
				if supplierID != "" && supplierID != sid {
					return "", "", ErrForbidden
				}
				supplierID = sid
			}
		default:
			return "", "", ErrForbidden
		}
	}
	return retailerID, supplierID, nil
}

// CreateRetailerPromise creates a 1-per-quarter self-serve payment promise (1..AR_MAX_PROMISE_DAYS).
func (s *Service) CreateRetailerPromise(ctx context.Context, retailerID, supplierID, invoiceID string, extensionDays int) (*PaymentPromise, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("ar service unavailable")
	}
	retailerID, supplierID, err := s.authorizeRetailerAction(ctx, retailerID, supplierID)
	if err != nil {
		return nil, err
	}
	invoiceID = strings.TrimSpace(invoiceID)
	if retailerID == "" || supplierID == "" {
		return nil, fmt.Errorf("retailer_id and supplier_id required")
	}
	maxDays := MaxPromiseDays()
	if extensionDays < 1 || extensionDays > maxDays {
		return nil, fmt.Errorf("%w: must be between 1 and %d", ErrPromiseInvalidExtension, maxDays)
	}

	now := s.now().UTC()
	qKey := QuarterKey(now, s.resolveSupplierTimezone(ctx, supplierID))

	return s.createPromise(ctx, retailerID, supplierID, invoiceID, qKey, extensionDays, PromiseApprovedBySystem, true, now)
}

// GrantSupplierPromise allows Supplier Finance / Admin to grant a payment promise without consuming self-serve quota rules.
func (s *Service) GrantSupplierPromise(ctx context.Context, supplierID, retailerID, invoiceID string, extensionDays int, approverID string) (*PaymentPromise, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("ar service unavailable")
	}
	supplierID = strings.TrimSpace(supplierID)
	retailerID = strings.TrimSpace(retailerID)
	invoiceID = strings.TrimSpace(invoiceID)
	approverID = strings.TrimSpace(approverID)

	if claims, ok := auth.FromContext(ctx); ok {
		switch claims.Role {
		case auth.RoleAdmin, auth.RoleWarehouseAdmin:
			if sid := strings.TrimSpace(claims.SupplierID); sid != "" {
				if supplierID != "" && supplierID != sid {
					return nil, ErrForbidden
				}
				supplierID = sid
			}
			if approverID == "" {
				approverID = strings.TrimSpace(claims.Subject)
			}
		default:
			return nil, ErrForbidden
		}
	}
	if supplierID == "" || retailerID == "" {
		return nil, fmt.Errorf("supplier_id and retailer_id required")
	}
	if approverID == "" {
		approverID = "SUPPLIER_ADMIN"
	}
	if extensionDays < 1 || extensionDays > 90 {
		return nil, fmt.Errorf("%w: must be between 1 and 90", ErrPromiseInvalidExtension)
	}

	now := s.now().UTC()
	qKey := QuarterKey(now, s.resolveSupplierTimezone(ctx, supplierID))

	return s.createPromise(ctx, retailerID, supplierID, invoiceID, qKey, extensionDays, approverID, false, now)
}

func (s *Service) createPromise(
	ctx context.Context,
	retailerID, supplierID, invoiceID, qKey string,
	extensionDays int,
	approvedBy string,
	selfServe bool,
	now time.Time,
) (*PaymentPromise, error) {
	switch repo := s.repo.(type) {
	case *SpannerRepository:
		return repo.createPromise(ctx, s, retailerID, supplierID, invoiceID, qKey, extensionDays, approvedBy, selfServe, now)
	case *MemoryRepository:
		return repo.createPromise(ctx, s, retailerID, supplierID, invoiceID, qKey, extensionDays, approvedBy, selfServe, now)
	default:
		return nil, fmt.Errorf("repository does not support payment promises")
	}
}

// ListPromises returns promises scoped to retailerID and/or supplierID ordered newest first.
func (s *Service) ListPromises(ctx context.Context, retailerID, supplierID string, limit int) ([]PaymentPromise, error) {
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
		return repo.listPromises(ctx, retailerID, supplierID, limit)
	case *MemoryRepository:
		return repo.listPromises(retailerID, supplierID, limit), nil
	default:
		return []PaymentPromise{}, nil
	}
}

// IsSelfServePromiseUsed returns whether the retailer has already used their self-serve promise for qKey.
func (s *Service) IsSelfServePromiseUsed(ctx context.Context, retailerID, supplierID, qKey string) (bool, error) {
	if s == nil || s.repo == nil || retailerID == "" || supplierID == "" || qKey == "" {
		return false, nil
	}
	switch repo := s.repo.(type) {
	case *SpannerRepository:
		return repo.isSelfServePromiseUsed(ctx, retailerID, supplierID, qKey)
	case *MemoryRepository:
		return repo.isSelfServePromiseUsed(retailerID, supplierID, qKey), nil
	default:
		return false, nil
	}
}

// GetRetailerStatement returns the unified statement for retailerID (optionally filtered by supplierID).
func (s *Service) GetRetailerStatement(ctx context.Context, retailerID, supplierID, status string, limit int, cursor string) (*StatementResponse, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("ar service unavailable")
	}
	retailerID, supplierID, err := s.authorizeRetailerAction(ctx, retailerID, supplierID)
	if err != nil {
		return nil, err
	}
	if retailerID == "" {
		return nil, fmt.Errorf("retailer_id required")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	// Collect all open/partial invoices to compute accurate relationship summary totals.
	allOpen, err := s.listAllInvoicesForRetailer(ctx, retailerID, StatusOpen+","+StatusPartial)
	if err != nil {
		return nil, err
	}
	var filteredOpen []Invoice
	var orderIDs []string
	currency := "UZS"
	for _, inv := range allOpen {
		if supplierID != "" && inv.SupplierID != supplierID {
			continue
		}
		if currency == "UZS" && inv.Currency != "" {
			currency = inv.Currency
		}
		filteredOpen = append(filteredOpen, inv)
		if inv.OrderID != "" {
			orderIDs = append(orderIDs, inv.OrderID)
		}
	}

	filterStatus := strings.TrimSpace(status)
	if filterStatus == "" {
		filterStatus = StatusOpen + "," + StatusPartial
	}
	var pageInvoices []Invoice
	var nextCursor string
	if supplierID == "" {
		pageInvoices, nextCursor, err = s.ListRetailerInvoicesPage(ctx, retailerID, filterStatus, limit, cursor)
		if err != nil {
			return nil, err
		}
	} else {
		// Filter strictly to supplierID before cursor pagination over the sorted set.
		allRequested, err := s.listAllInvoicesForRetailer(ctx, retailerID, filterStatus)
		if err != nil {
			return nil, err
		}
		var supplierScoped []Invoice
		for _, inv := range allRequested {
			if inv.SupplierID == supplierID {
				supplierScoped = append(supplierScoped, inv)
				if inv.OrderID != "" {
					orderIDs = append(orderIDs, inv.OrderID)
				}
			}
		}
		pageInvoices, nextCursor, err = paginateMemoryInvoices(supplierScoped, limit, cursor)
		if err != nil {
			return nil, err
		}
	}
	for _, inv := range pageInvoices {
		if inv.OrderID != "" {
			orderIDs = append(orderIDs, inv.OrderID)
		}
	}

	disputes, err := s.lookupDisputes(ctx, orderIDs)
	if err != nil {
		return nil, err
	}

	var totalOpen, totalDisputed, totalUndisputed int64
	for _, inv := range filteredOpen {
		totalOpen += inv.BalanceMinor
		d, u := clampDisputed(inv.BalanceMinor, disputes[inv.OrderID])
		totalDisputed += d
		totalUndisputed += u
	}

	stmtInvoices := make([]StatementInvoice, 0, len(pageInvoices))
	for _, inv := range pageInvoices {
		d, u := clampDisputed(inv.BalanceMinor, disputes[inv.OrderID])
		stmtInvoices = append(stmtInvoices, StatementInvoice{
			InvoiceID:       inv.InvoiceID,
			SupplierID:      inv.SupplierID,
			RetailerID:      inv.RetailerID,
			OrderID:         inv.OrderID,
			Status:          inv.Status,
			PrincipalMinor:  inv.PrincipalMinor,
			BalanceMinor:    inv.BalanceMinor,
			DisputedMinor:   d,
			UndisputedMinor: u,
			Currency:        inv.Currency,
			CreditLeaveAt:   inv.CreditLeaveAt,
			DueAt:           inv.DueAt,
			TermsDays:       inv.TermsDays,
			GracePeriodDays: inv.GracePeriodDays,
			AgingBucket:     inv.AgingBucket,
			LastDunnedAt:    inv.LastDunnedAt,
			DunningStep:     inv.DunningStep,
			Version:         inv.Version,
			CreatedAt:       inv.CreatedAt,
			UpdatedAt:       inv.UpdatedAt,
		})
	}

	var advMinor int64
	if supplierID != "" {
		adv, err := s.GetAdvanceBalance(ctx, retailerID, supplierID)
		if err != nil {
			return nil, err
		}
		advMinor = adv.BalanceMinor
		if adv.Currency != "" && len(filteredOpen) == 0 {
			currency = adv.Currency
		}
	} else {
		advMinor, err = s.sumAdvanceBalancesForRetailer(ctx, retailerID)
		if err != nil {
			return nil, err
		}
	}

	promises, err := s.ListPromises(ctx, retailerID, supplierID, 50)
	if err != nil {
		return nil, err
	}
	if promises == nil {
		promises = []PaymentPromise{}
	}

	ledger, err := s.ListLedgerEntries(ctx, retailerID, supplierID, 50)
	if err != nil {
		return nil, err
	}
	if ledger == nil {
		ledger = []LedgerEntry{}
	}

	qKey := QuarterKey(s.now().UTC(), s.resolveSupplierTimezone(ctx, supplierID))
	used, err := s.IsSelfServePromiseUsed(ctx, retailerID, supplierID, qKey)
	if err != nil {
		return nil, err
	}
	if !used && supplierID == "" {
		for _, p := range promises {
			if p.QuarterKey == qKey && p.ApprovedBy == PromiseApprovedBySystem {
				used = true
				break
			}
		}
	}

	return &StatementResponse{
		RetailerID:           retailerID,
		SupplierID:           supplierID,
		Currency:             currency,
		TotalOpenMinor:       totalOpen,
		TotalDisputedMinor:   totalDisputed,
		TotalUndisputedMinor: totalUndisputed,
		AdvanceBalanceMinor:  advMinor,
		CurrentQuarterKey:    qKey,
		SelfServePromiseUsed: used,
		MaxExtensionDays:     MaxPromiseDays(),
		Invoices:             stmtInvoices,
		Promises:             promises,
		LedgerEntries:        ledger,
		NextCursor:           nextCursor,
	}, nil
}

func (s *Service) listAllInvoicesForRetailer(ctx context.Context, retailerID, status string) ([]Invoice, error) {
	pager, hasPager := s.repo.(InvoicePager)
	if !hasPager {
		return s.repo.ListByRetailer(ctx, retailerID, status, 200)
	}
	var all []Invoice
	cursor := ""
	for {
		page, next, err := pager.ListByRetailerPage(ctx, retailerID, status, 200, cursor)
		if err != nil {
			return nil, err
		}
		all = append(all, page...)
		if next == "" || next == cursor || len(page) == 0 {
			break
		}
		cursor = next
	}
	return all, nil
}

func (s *Service) sumAdvanceBalancesForRetailer(ctx context.Context, retailerID string) (int64, error) {
	switch repo := s.repo.(type) {
	case *SpannerRepository:
		if repo == nil || repo.client == nil {
			return 0, nil
		}
		iter := repo.client.Single().Query(ctx, spanner.Statement{
			SQL:    `SELECT IFNULL(SUM(BalanceMinor), 0) FROM ArAdvanceBalances WHERE RetailerId = @rid`,
			Params: map[string]any{"rid": retailerID},
		})
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
	case *MemoryRepository:
		repo.mu.RLock()
		defer repo.mu.RUnlock()
		var sum int64
		for _, bal := range repo.advanceBalances {
			if bal.RetailerID == retailerID {
				sum += bal.BalanceMinor
			}
		}
		return sum, nil
	default:
		return 0, nil
	}
}

func (s *Service) listActivePromisesForInvoices(ctx context.Context, invoices []Invoice, now time.Time) ([]PaymentPromise, error) {
	if s == nil || s.repo == nil || len(invoices) == 0 {
		return nil, nil
	}
	switch repo := s.repo.(type) {
	case *MemoryRepository:
		repo.mu.RLock()
		defer repo.mu.RUnlock()
		var out []PaymentPromise
		for _, p := range repo.promises {
			if p.Status == PromiseStatusActive && !p.DueAt.Before(now) {
				out = append(out, p)
			}
		}
		return out, nil
	case *SpannerRepository:
		if repo == nil || repo.client == nil {
			return nil, nil
		}
		rids := make([]string, 0, len(invoices))
		for _, inv := range invoices {
			rids = append(rids, inv.RetailerID)
		}
		rids = dedupeNonEmptyIDs(rids)
		if len(rids) == 0 {
			return nil, nil
		}
		iter := repo.client.Single().Query(ctx, spanner.Statement{
			SQL: `SELECT PromiseId, RetailerId, SupplierId, IFNULL(InvoiceId, ''), QuarterKey, PromisedAt, DueAt, PromisedMinor, Status, IFNULL(ApprovedBy, ''), CreatedAt, UpdatedAt
				FROM ArPaymentPromises
				WHERE Status = 'ACTIVE' AND DueAt >= @now AND RetailerId IN UNNEST(@rids)`,
			Params: map[string]any{"now": now.UTC(), "rids": rids},
		})
		defer iter.Stop()
		var out []PaymentPromise
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				return out, nil
			}
			if err != nil {
				return nil, err
			}
			p, err := scanPromise(row)
			if err != nil {
				return nil, err
			}
			out = append(out, p)
		}
	default:
		return nil, nil
	}
}

func scanPromise(row *spanner.Row) (PaymentPromise, error) {
	var p PaymentPromise
	if err := row.Columns(&p.PromiseID, &p.RetailerID, &p.SupplierID, &p.InvoiceID, &p.QuarterKey,
		&p.PromisedAt, &p.DueAt, &p.PromisedMinor, &p.Status, &p.ApprovedBy, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return PaymentPromise{}, err
	}
	return p, nil
}

func (r *MemoryRepository) isSelfServePromiseUsed(retailerID, supplierID, qKey string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if st, ok := r.creditProfiles[relKey(retailerID, supplierID)]; ok && st.ActiveQuarterKey == qKey {
		return true
	}
	for _, p := range r.promises {
		if p.RetailerID == retailerID && p.SupplierID == supplierID && p.QuarterKey == qKey {
			return true
		}
	}
	return false
}

func (r *MemoryRepository) listPromises(retailerID, supplierID string, limit int) []PaymentPromise {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]PaymentPromise, 0)
	for _, p := range r.promises {
		if retailerID != "" && p.RetailerID != retailerID {
			continue
		}
		if supplierID != "" && p.SupplierID != supplierID {
			continue
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.After(out[j].CreatedAt)
		}
		return out[i].PromiseID > out[j].PromiseID
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func (r *MemoryRepository) createPromise(
	ctx context.Context,
	svc *Service,
	retailerID, supplierID, invoiceID, qKey string,
	extensionDays int,
	approvedBy string,
	selfServe bool,
	now time.Time,
) (*PaymentPromise, error) {
	r.mu.RLock()
	orderIDs := r.collectOpenOrderIDsLocked(retailerID, supplierID)
	if invoiceID != "" {
		if inv, ok := r.byID[invoiceID]; ok && inv.OrderID != "" {
			orderIDs = append(orderIDs, inv.OrderID)
		}
	}
	r.mu.RUnlock()

	disputes, err := svc.lookupDisputes(ctx, orderIDs)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.ensureMapsLocked()

	if selfServe {
		if st, ok := r.creditProfiles[relKey(retailerID, supplierID)]; ok {
			if st.PrepayOnlyLocked {
				return nil, ErrPromisePrepayLocked
			}
			if isBlockedProfileStatus(st.Status) {
				return nil, ErrPromiseProfileFrozen
			}
			if st.ActiveQuarterKey == qKey {
				return nil, &QuarterlyLimitError{
					QuarterKey: qKey,
					Message:    fmt.Sprintf("Self-service promise already used in %s", qKey),
				}
			}
		}
		for _, p := range r.promises {
			if p.RetailerID == retailerID && p.SupplierID == supplierID && p.QuarterKey == qKey {
				return nil, &QuarterlyLimitError{
					QuarterKey: qKey,
					Message:    fmt.Sprintf("Self-service promise already used in %s", qKey),
				}
			}
		}
	}

	var anchorInvoices []Invoice
	var promisedMinor int64
	if invoiceID != "" {
		inv, ok := r.byID[invoiceID]
		if !ok {
			return nil, ErrInvoiceNotFound
		}
		if inv.RetailerID != retailerID || inv.SupplierID != supplierID {
			return nil, ErrForbidden
		}
		if inv.Status == StatusPaid || inv.Status == StatusVoid {
			return nil, ErrInvoiceAlreadySettled
		}
		_, und := clampDisputed(inv.BalanceMinor, disputes[inv.OrderID])
		if und <= 0 {
			return nil, ErrPromiseNoBalance
		}
		promisedMinor = und
		anchorInvoices = []Invoice{inv}
	} else {
		for _, inv := range r.byID {
			if inv.RetailerID != retailerID || inv.SupplierID != supplierID {
				continue
			}
			if inv.Status != StatusOpen && inv.Status != StatusPartial {
				continue
			}
			_, und := clampDisputed(inv.BalanceMinor, disputes[inv.OrderID])
			if und <= 0 {
				continue
			}
			promisedMinor += und
			anchorInvoices = append(anchorInvoices, inv)
		}
		if promisedMinor <= 0 {
			return nil, ErrPromiseNoBalance
		}
	}

	dueAt := computePromiseDueAt(now, anchorInvoices, extensionDays)
	promise := PaymentPromise{
		PromiseID:     "arp_" + strings.ReplaceAll(uuid.NewString(), "-", ""),
		RetailerID:    retailerID,
		SupplierID:    supplierID,
		InvoiceID:     invoiceID,
		QuarterKey:    qKey,
		PromisedAt:    now,
		DueAt:         dueAt,
		PromisedMinor: promisedMinor,
		Status:        PromiseStatusActive,
		ApprovedBy:    approvedBy,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	r.promises[promise.PromiseID] = promise
	if selfServe {
		st := r.creditProfiles[relKey(retailerID, supplierID)]
		st.ActiveQuarterKey = qKey
		r.creditProfiles[relKey(retailerID, supplierID)] = st
	}
	cp := promise
	return &cp, nil
}

func (r *SpannerRepository) isSelfServePromiseUsed(ctx context.Context, retailerID, supplierID, qKey string) (bool, error) {
	if r == nil || r.client == nil {
		return false, nil
	}
	row, err := r.client.Single().ReadRow(ctx, "RetailerCreditProfiles", spanner.Key{retailerID, supplierID}, []string{"ActiveQuarterKey"})
	if err == nil {
		var activeQ spanner.NullString
		if colErr := row.Columns(&activeQ); colErr == nil && activeQ.Valid && activeQ.StringVal == qKey {
			return true, nil
		}
	} else if spanner.ErrCode(err) != codes.NotFound {
		return false, err
	}
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT PromiseId FROM ArPaymentPromises
			WHERE RetailerId = @rid AND SupplierId = @sid AND QuarterKey = @qk LIMIT 1`,
		Params: map[string]any{"rid": retailerID, "sid": supplierID, "qk": qKey},
	})
	defer iter.Stop()
	_, qerr := iter.Next()
	if qerr == nil {
		return true, nil
	}
	if qerr == iterator.Done {
		return false, nil
	}
	return false, qerr
}

func (r *SpannerRepository) listPromises(ctx context.Context, retailerID, supplierID string, limit int) ([]PaymentPromise, error) {
	if r == nil || r.client == nil {
		return []PaymentPromise{}, nil
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
	sql := `SELECT PromiseId, RetailerId, SupplierId, IFNULL(InvoiceId, ''), QuarterKey, PromisedAt, DueAt, PromisedMinor, Status, IFNULL(ApprovedBy, ''), CreatedAt, UpdatedAt FROM ArPaymentPromises`
	if len(clauses) > 0 {
		sql += " WHERE " + strings.Join(clauses, " AND ")
	}
	sql += " ORDER BY CreatedAt DESC, PromiseId DESC LIMIT @limit"

	iter := r.client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	out := []PaymentPromise{}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		p, err := scanPromise(row)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
}

func (r *SpannerRepository) createPromise(
	ctx context.Context,
	svc *Service,
	retailerID, supplierID, invoiceID, qKey string,
	extensionDays int,
	approvedBy string,
	selfServe bool,
	now time.Time,
) (*PaymentPromise, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("spanner ar repository: nil client")
	}
	var created PaymentPromise
	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		profileFound := false
		profRow, err := txn.ReadRow(ctx, "RetailerCreditProfiles", spanner.Key{retailerID, supplierID},
			[]string{"Status", "PrepayOnlyLocked", "ActiveQuarterKey"})
		if err == nil {
			profileFound = true
			var status, activeQ spanner.NullString
			var prepayLocked spanner.NullBool
			if colErr := profRow.Columns(&status, &prepayLocked, &activeQ); colErr != nil {
				return colErr
			}
			if selfServe {
				if prepayLocked.Valid && prepayLocked.Bool {
					return ErrPromisePrepayLocked
				}
				if status.Valid && isBlockedProfileStatus(status.StringVal) {
					return ErrPromiseProfileFrozen
				}
				if activeQ.Valid && activeQ.StringVal == qKey {
					return &QuarterlyLimitError{
						QuarterKey: qKey,
						Message:    fmt.Sprintf("Self-service promise already used in %s", qKey),
					}
				}
			}
		} else if spanner.ErrCode(err) != codes.NotFound {
			return err
		}

		if selfServe {
			iter := txn.Query(ctx, spanner.Statement{
				SQL: `SELECT PromiseId FROM ArPaymentPromises
					WHERE RetailerId = @rid AND SupplierId = @sid AND QuarterKey = @qk LIMIT 1`,
				Params: map[string]any{"rid": retailerID, "sid": supplierID, "qk": qKey},
			})
			_, qerr := iter.Next()
			iter.Stop()
			if qerr == nil {
				return &QuarterlyLimitError{
					QuarterKey: qKey,
					Message:    fmt.Sprintf("Self-service promise already used in %s", qKey),
				}
			}
			if qerr != iterator.Done {
				return qerr
			}
		}

		openList, err := listSupplierOpenInvoicesInTxn(ctx, txn, retailerID, supplierID)
		if err != nil {
			return err
		}
		var orderIDs []string
		for _, inv := range openList {
			if inv.OrderID != "" {
				orderIDs = append(orderIDs, inv.OrderID)
			}
		}
		var disputes map[string]int64
		if svc.disputeLookup != nil {
			disputes, err = svc.disputeLookup(ctx, dedupeNonEmptyIDs(orderIDs))
		} else {
			disputes, err = queryClaimsDisputeMap(ctx, txn, orderIDs)
		}
		if err != nil {
			return err
		}

		var anchorInvoices []Invoice
		var promisedMinor int64
		if invoiceID != "" {
			var target *Invoice
			for i := range openList {
				if openList[i].InvoiceID == invoiceID {
					target = &openList[i]
					break
				}
			}
			if target == nil {
				invRow, invErr := txn.ReadRow(ctx, "ArInvoices", spanner.Key{invoiceID}, []string{"SupplierId", "RetailerId", "Status"})
				if invErr != nil {
					if spanner.ErrCode(invErr) == codes.NotFound {
						return ErrInvoiceNotFound
					}
					return invErr
				}
				var sid, rid, st string
				if colErr := invRow.Columns(&sid, &rid, &st); colErr != nil {
					return colErr
				}
				if sid != supplierID || rid != retailerID {
					return ErrForbidden
				}
				return ErrInvoiceAlreadySettled
			}
			_, und := clampDisputed(target.BalanceMinor, disputes[target.OrderID])
			if und <= 0 {
				return ErrPromiseNoBalance
			}
			promisedMinor = und
			anchorInvoices = []Invoice{*target}
		} else {
			for _, inv := range openList {
				_, und := clampDisputed(inv.BalanceMinor, disputes[inv.OrderID])
				if und <= 0 {
					continue
				}
				promisedMinor += und
				anchorInvoices = append(anchorInvoices, inv)
			}
			if promisedMinor <= 0 {
				return ErrPromiseNoBalance
			}
		}

		dueAt := computePromiseDueAt(now, anchorInvoices, extensionDays)
		promiseID := "arp_" + strings.ReplaceAll(uuid.NewString(), "-", "")
		invIDCol := spanner.NullString{StringVal: invoiceID, Valid: invoiceID != ""}

		mutations := []*spanner.Mutation{
			spanner.InsertOrUpdateMap("ArPaymentPromises", map[string]any{
				"PromiseId":     promiseID,
				"RetailerId":    retailerID,
				"SupplierId":    supplierID,
				"InvoiceId":     invIDCol,
				"QuarterKey":    qKey,
				"PromisedAt":    now,
				"DueAt":         dueAt,
				"PromisedMinor": promisedMinor,
				"Status":        PromiseStatusActive,
				"ApprovedBy":    approvedBy,
				"CreatedAt":     spanner.CommitTimestamp,
				"UpdatedAt":     spanner.CommitTimestamp,
			}),
		}
		if selfServe && profileFound {
			mutations = append(mutations, spanner.UpdateMap("RetailerCreditProfiles", map[string]any{
				"RetailerId":       retailerID,
				"SupplierId":       supplierID,
				"ActiveQuarterKey": qKey,
				"UpdatedAt":        spanner.CommitTimestamp,
			}))
		}

		buf := outbox.NewSpannerTxnBuffer(txn)
		if err := outbox.EmitJSON(ctx, buf, events.AggregateARInvoice, promiseID, events.TopicMain, map[string]any{
			"type":           EventARPaymentPromiseCreated,
			"timestamp":      now.Format(time.RFC3339Nano),
			"promise_id":     promiseID,
			"retailer_id":    retailerID,
			"supplier_id":    supplierID,
			"invoice_id":     invoiceID,
			"quarter_key":    qKey,
			"due_at":         dueAt.Format(time.RFC3339Nano),
			"promised_minor": promisedMinor,
			"approved_by":    approvedBy,
		}); err != nil {
			return err
		}
		if err := txn.BufferWrite(mutations); err != nil {
			return err
		}
		if err := buf.Flush(ctx); err != nil {
			return err
		}

		created = PaymentPromise{
			PromiseID:     promiseID,
			RetailerID:    retailerID,
			SupplierID:    supplierID,
			InvoiceID:     invoiceID,
			QuarterKey:    qKey,
			PromisedAt:    now,
			DueAt:         dueAt,
			PromisedMinor: promisedMinor,
			Status:        PromiseStatusActive,
			ApprovedBy:    approvedBy,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &created, nil
}

func fulfillMatchingPromisesInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	buf outbox.TxnBuffer,
	retailerID, supplierID string,
	postInvoices map[string]Invoice,
	disputes map[string]int64,
	postTotalUndisputed int64,
	ts string,
	mutations *[]*spanner.Mutation,
) error {
	iter := txn.Query(ctx, spanner.Statement{
		SQL: `SELECT PromiseId, RetailerId, SupplierId, IFNULL(InvoiceId, ''), QuarterKey, PromisedAt, DueAt, PromisedMinor, Status, IFNULL(ApprovedBy, ''), CreatedAt, UpdatedAt
			FROM ArPaymentPromises
			WHERE RetailerId = @rid AND SupplierId = @sid AND Status = 'ACTIVE'`,
		Params: map[string]any{"rid": retailerID, "sid": supplierID},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return nil
		}
		if err != nil {
			return err
		}
		p, err := scanPromise(row)
		if err != nil {
			return err
		}
		fulfilled := postTotalUndisputed <= 0
		if !fulfilled && p.InvoiceID != "" {
			if inv, ok := postInvoices[p.InvoiceID]; ok {
				_, und := clampDisputed(inv.BalanceMinor, disputes[inv.OrderID])
				if inv.BalanceMinor <= 0 || inv.Status == StatusPaid || inv.Status == StatusVoid || und <= 0 {
					fulfilled = true
				}
			} else {
				invRow, invErr := txn.ReadRow(ctx, "ArInvoices", spanner.Key{p.InvoiceID}, []string{"BalanceMinor", "Status", "OrderId"})
				if invErr == nil {
					var bal int64
					var st, oid string
					if colErr := invRow.Columns(&bal, &st, &oid); colErr == nil {
						_, und := clampDisputed(bal, disputes[oid])
						if bal <= 0 || st == StatusPaid || st == StatusVoid || und <= 0 {
							fulfilled = true
						}
					}
				}
			}
		}
		if !fulfilled {
			continue
		}
		*mutations = append(*mutations, spanner.UpdateMap("ArPaymentPromises", map[string]any{
			"PromiseId": p.PromiseID,
			"Status":    PromiseStatusFulfilled,
			"UpdatedAt": spanner.CommitTimestamp,
		}))
		if err := outbox.EmitJSON(ctx, buf, events.AggregateARInvoice, p.PromiseID, events.TopicMain, map[string]any{
			"type":        EventARPaymentPromiseStatus,
			"timestamp":   ts,
			"promise_id":  p.PromiseID,
			"retailer_id": retailerID,
			"supplier_id": supplierID,
			"invoice_id":  p.InvoiceID,
			"status":      PromiseStatusFulfilled,
		}); err != nil {
			return err
		}
	}
}

func encodePromiseCursor(dueAt time.Time, promiseID string) string {
	payload := dueAt.UTC().Format(time.RFC3339Nano) + "|" + promiseID
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func decodePromiseCursor(cursor string) (time.Time, string, bool, error) {
	cursor = strings.TrimSpace(cursor)
	if cursor == "" {
		return time.Time{}, "", false, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, "", false, ErrInvalidCursor
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
		return time.Time{}, "", false, ErrInvalidCursor
	}
	ts, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", false, ErrInvalidCursor
	}
	return ts.UTC(), parts[1], true, nil
}
