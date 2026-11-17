package claims

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/storage"
)

func defaultTestEvidenceInspector() storage.EvidenceInspector {
	return func(_ context.Context, uri string) (storage.EvidenceInspection, error) {
		return storage.InspectEvidenceStream(bytes.NewReader([]byte("test-photo:"+strings.TrimSpace(uri))), uri, 7001)
	}
}

func mapTestEvidenceInspector(blobs map[string][]byte) storage.EvidenceInspector {
	return func(_ context.Context, uri string) (storage.EvidenceInspection, error) {
		payload, ok := blobs[strings.TrimSpace(uri)]
		if !ok {
			return storage.EvidenceInspection{}, errors.New("object_not_found")
		}
		return storage.InspectEvidenceStream(bytes.NewReader(payload), uri, 8001)
	}
}

type fakeOrders struct {
	o  OrderSnapshot
	ok bool
}

func (f fakeOrders) GetOrder(context.Context, string) (OrderSnapshot, bool, error) {
	return f.o, f.ok, nil
}

type fakeSettler struct {
	last ClaimSettlement
	res  SettlementResult
	err  error
}

func (f *fakeSettler) SettleClaimChargeback(_ context.Context, in ClaimSettlement) (SettlementResult, error) {
	f.last = in
	if f.err != nil {
		return SettlementResult{}, f.err
	}
	if f.res.AmountMinor == 0 {
		f.res.AmountMinor = in.AmountMinor
		f.res.Currency = in.Currency
		f.res.Mode = "LEDGER_ONLY"
		f.res.ChargebackID = "cb_test"
	}
	return f.res, nil
}

func completedOrder(now time.Time) OrderSnapshot {
	return OrderSnapshot{
		OrderID:    "ord-1",
		SupplierID: "sup-1",
		RetailerID: "ret-1",
		Status:     OrderStatusCompleted,
		UpdatedAt:  now.Add(-1 * time.Hour),
		Currency:   "UZS",
		TotalMinor: 10000,
		LineItems: []OrderLine{
			{SKU: "sku-1", Quantity: 5, UnitPriceMinor: 2000},
		},
	}
}

func TestFileRetailerClaimRequiresPhotoForDamage(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository()
	svc := NewService(Config{
		Repo:   repo,
		Orders: fakeOrders{ok: true, o: completedOrder(now)},
		Now:    func() time.Time { return now },
		NewID:  func() string { return "x1" },
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	_, err := svc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeDamaged,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 1}},
	})
	if err != ErrEvidenceRequired {
		t.Fatalf("got %v want ErrEvidenceRequired", err)
	}
}

func TestFileRetailerClaimSucceedsWithPhotoAndPricesFromOrder(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository()
	svc := NewService(Config{
		Repo:              repo,
		Orders:            fakeOrders{ok: true, o: completedOrder(now)},
		EvidenceInspector: defaultTestEvidenceInspector(),
		Now:               func() time.Time { return now },
		NewID:             func() string { return "x1" },
		Log:               slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	c, err := svc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeConcealedDamage,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 2, Reason: "DAMAGED"}},
		Evidences: []struct {
			EvidenceType EvidenceType `json:"evidence_type"`
			URI          string       `json:"uri"`
			MimeType     string       `json:"mime_type"`
		}{{EvidenceType: EvidencePhoto, URI: "https://storage.googleapis.com/pegasusx-test/evidence/p.jpg", MimeType: "image/jpeg"}},
	})
	if err != nil {
		t.Fatalf("FileRetailerClaim: %v", err)
	}
	if c.Status != StatusOpen || c.ClaimType != ClaimTypeConcealedDamage {
		t.Fatalf("unexpected claim: %+v", c)
	}
	if c.AmountMinor != 4000 { // 2 * 2000
		t.Fatalf("amount_minor=%d want 4000 (priced from order)", c.AmountMinor)
	}
	if len(c.LineItems) != 1 || c.LineItems[0].UnitPriceMinor != 2000 {
		t.Fatalf("line pricing: %+v", c.LineItems)
	}
}

type fakeStoreStock struct {
	holds    int
	resolves []string
	holdErr  error
}

func (f *fakeStoreStock) HoldForClaim(_ context.Context, _, _, _ string, lines []ClaimLine, _ string) error {
	f.holds++
	if f.holdErr != nil {
		return f.holdErr
	}
	if len(lines) == 0 {
		return fmt.Errorf("empty lines")
	}
	return nil
}

func (f *fakeStoreStock) ResolveClaimStock(_ context.Context, _, _ string, _ []ClaimLine, disposition, _ string) error {
	f.resolves = append(f.resolves, disposition)
	return nil
}

func TestFileClaimHoldsStoreStockForPhysicalTypes(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	stock := &fakeStoreStock{}
	svc := NewService(Config{
		Repo:              NewMemoryRepository(),
		Orders:            fakeOrders{ok: true, o: completedOrder(now)},
		StoreStock:        stock,
		EvidenceInspector: defaultTestEvidenceInspector(),
		Now:               func() time.Time { return now },
		NewID:             func() string { return "hold1" },
		Log:               slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	_, err := svc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeConcealedDamage,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 1}},
		Evidences: []struct {
			EvidenceType EvidenceType `json:"evidence_type"`
			URI          string       `json:"uri"`
			MimeType     string       `json:"mime_type"`
		}{{EvidenceType: EvidencePhoto, URI: "https://storage.googleapis.com/pegasusx-test/evidence/p.jpg"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if stock.holds != 1 {
		t.Fatalf("holds=%d want 1", stock.holds)
	}
	// MISSING should not hold store stock
	stock.holds = 0
	_, err = svc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeMissing,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if stock.holds != 0 {
		t.Fatalf("MISSING should not hold store stock, holds=%d", stock.holds)
	}
}

func TestApproveRejectResolvesStoreStock(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	stock := &fakeStoreStock{}
	repo := NewMemoryRepository()
	svc := NewService(Config{
		Repo:              repo,
		Orders:            fakeOrders{ok: true, o: completedOrder(now)},
		Settler:           &fakeSettler{},
		StoreStock:        stock,
		EvidenceInspector: defaultTestEvidenceInspector(),
		Now:               func() time.Time { return now },
		NewID:             func() string { return "res1" },
		Log:               slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	c, err := svc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeDamaged,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 1}},
		Evidences: []struct {
			EvidenceType EvidenceType `json:"evidence_type"`
			URI          string       `json:"uri"`
			MimeType     string       `json:"mime_type"`
		}{{EvidenceType: EvidencePhoto, URI: "https://storage.googleapis.com/pegasusx-test/evidence/p.jpg"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = svc.ApproveClaim(context.Background(), auth.Claims{
		Subject: "admin-1", Role: auth.RoleAdmin,
	}, c.ClaimID, ApproveClaimRequest{ResolutionNote: "ok"})
	if err != nil {
		t.Fatal(err)
	}
	if len(stock.resolves) != 1 || stock.resolves[0] != "WASTE" {
		t.Fatalf("resolves=%v want WASTE", stock.resolves)
	}

	// Reject path
	stock.resolves = nil
	c2, err := svc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeDamaged,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 1}},
		Evidences: []struct {
			EvidenceType EvidenceType `json:"evidence_type"`
			URI          string       `json:"uri"`
			MimeType     string       `json:"mime_type"`
		}{{EvidenceType: EvidencePhoto, URI: "https://storage.googleapis.com/pegasusx-test/evidence/p2.jpg"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.RejectClaim(context.Background(), auth.Claims{
		Subject: "admin-1", Role: auth.RoleAdmin,
	}, c2.ClaimID, RejectClaimRequest{ResolutionNote: "nope"})
	if err != nil {
		t.Fatal(err)
	}
	if len(stock.resolves) != 1 || stock.resolves[0] != "RESTORE" {
		t.Fatalf("resolves=%v want RESTORE", stock.resolves)
	}
}

func TestFileRetailerClaimWindowExpired(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	o := completedOrder(now)
	o.UpdatedAt = now.Add(-72 * time.Hour)
	svc := NewService(Config{
		Repo:   NewMemoryRepository(),
		Orders: fakeOrders{ok: true, o: o},
		Now:    func() time.Time { return now },
		Window: 48 * time.Hour,
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	_, err := svc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeMissing,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 1}},
	})
	if err != ErrClaimWindowExpired {
		t.Fatalf("got %v want window expired", err)
	}
}

func TestFileRetailerClaim_SnapshotEndsAtExpired(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	o := completedOrder(now)
	o.UpdatedAt = now.Add(-2 * time.Hour) // open under legacy 48h
	ends := now.Add(-30 * time.Minute)
	o.ClaimWindowHours = 1
	o.ClaimWindowEndsAt = &ends
	o.ClaimWindowPolicySource = "SUPPLIER"
	svc := NewService(Config{
		Repo:   NewMemoryRepository(),
		Orders: fakeOrders{ok: true, o: o},
		Now:    func() time.Time { return now },
		Window: 48 * time.Hour,
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	_, err := svc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeMissing,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 1}},
	})
	if err != ErrClaimWindowExpired {
		t.Fatalf("got %v want window expired via snapshot", err)
	}
}

func TestApproveClaimSettlesChargeback(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository()
	settler := &fakeSettler{}
	svc := NewService(Config{
		Repo:    repo,
		Orders:  fakeOrders{ok: true, o: completedOrder(now)},
		Settler: settler,
		Now:     func() time.Time { return now },
		NewID:   func() string { return "x1" },
		Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	c, err := svc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeMissing,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	approved, settlement, err := svc.ApproveClaim(context.Background(), auth.Claims{
		Subject: "admin-1", Role: auth.RoleAdmin,
	}, c.ClaimID, ApproveClaimRequest{ResolutionNote: "ok"})
	if err != nil {
		t.Fatal(err)
	}
	if approved.Status != StatusResolved {
		t.Fatalf("status=%s", approved.Status)
	}
	if settlement.ChargebackID == "" || settlement.AmountMinor != 2000 {
		t.Fatalf("settlement=%+v", settlement)
	}
	if settler.last.OrderID != "ord-1" || settler.last.AmountMinor != 2000 {
		t.Fatalf("settler input=%+v", settler.last)
	}

	// Re-approve is idempotent — no second settlement call required for success.
	again, settle2, err := svc.ApproveClaim(context.Background(), auth.Claims{
		Subject: "admin-1", Role: auth.RoleAdmin,
	}, c.ClaimID, ApproveClaimRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if !settle2.Idempotent || again.Status != StatusResolved {
		t.Fatalf("replay=%+v settle=%+v", again, settle2)
	}
}

func TestApproveClaim_UnderReviewRetryAndRejectedValidation(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository()
	settler := &fakeSettler{err: errors.New("temporary gateway timeout")}
	svc := NewService(Config{
		Repo:    repo,
		Orders:  fakeOrders{ok: true, o: completedOrder(now)},
		Settler: settler,
		Now:     func() time.Time { return now },
		NewID:   func() string { return "x1" },
		Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	c, err := svc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeMissing,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}

	// First attempt fails during settlement, leaving claim in StatusUnderReview
	_, _, err = svc.ApproveClaim(context.Background(), auth.Claims{
		Subject: "admin-1", Role: auth.RoleAdmin,
	}, c.ClaimID, ApproveClaimRequest{ResolutionNote: "initial attempt"})
	if err == nil || !strings.Contains(err.Error(), "temporary gateway timeout") {
		t.Fatalf("expected settlement error, got %v", err)
	}

	loaded, ok, err := repo.GetClaim(context.Background(), c.ClaimID)
	if err != nil || !ok || loaded.Status != StatusUnderReview {
		t.Fatalf("expected claim in StatusUnderReview, got ok=%v, status=%s, err=%v", ok, loaded.Status, err)
	}

	// Fix settler and retry ApproveClaim on StatusUnderReview claim
	settler.err = nil
	approved, settlement, err := svc.ApproveClaim(context.Background(), auth.Claims{
		Subject: "admin-1", Role: auth.RoleAdmin,
	}, c.ClaimID, ApproveClaimRequest{ResolutionNote: "retry after timeout"})
	if err != nil {
		t.Fatalf("expected retry on StatusUnderReview to succeed, got %v", err)
	}
	if approved.Status != StatusResolved {
		t.Fatalf("expected status=%s, got %s", StatusResolved, approved.Status)
	}
	if settlement.ChargebackID == "" || settlement.AmountMinor != 2000 {
		t.Fatalf("unexpected settlement: %+v", settlement)
	}

	// Now file another claim and reject it
	c2, err := svc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeMissing,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.RejectClaim(context.Background(), auth.Claims{
		Subject: "admin-1", Role: auth.RoleAdmin,
	}, c2.ClaimID, RejectClaimRequest{ResolutionNote: "fraud"})
	if err != nil {
		t.Fatal(err)
	}

	// Approve on rejected claim must fail with ErrInvalidClaimState
	_, _, err = svc.ApproveClaim(context.Background(), auth.Claims{
		Subject: "admin-1", Role: auth.RoleAdmin,
	}, c2.ClaimID, ApproveClaimRequest{ResolutionNote: "re-approve"})
	if !errors.Is(err, ErrInvalidClaimState) {
		t.Fatalf("expected ErrInvalidClaimState for rejected claim, got %v", err)
	}
}

func TestApproveClaim_SupplierScopeForbidden(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository()
	svc := NewService(Config{
		Repo:    repo,
		Orders:  fakeOrders{ok: true, o: completedOrder(now)},
		Settler: &fakeSettler{},
		Now:     func() time.Time { return now },
		NewID:   func() string { return "x1" },
		Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	c, err := svc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeMissing,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = svc.ApproveClaim(context.Background(), auth.Claims{
		Subject: "admin-other", Role: auth.RoleAdmin, SupplierID: "sup-OTHER",
	}, c.ClaimID, ApproveClaimRequest{})
	if err != ErrForbidden {
		t.Fatalf("got %v want forbidden", err)
	}
}

func TestListSupplierClaims_Scoped(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository()
	svc := NewService(Config{
		Repo:   repo,
		Orders: fakeOrders{ok: true, o: completedOrder(now)},
		Now:    func() time.Time { return now },
		NewID:  func() string { return "x1" },
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	_, err := svc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeMissing,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	list, err := svc.ListSupplierClaims(context.Background(), auth.Claims{
		Subject: "admin", Role: auth.RoleAdmin, SupplierID: "sup-1",
	}, StatusOpen, 20)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v len=%d", err, len(list))
	}
	list, err = svc.ListSupplierClaims(context.Background(), auth.Claims{
		Subject: "admin", Role: auth.RoleAdmin, SupplierID: "sup-OTHER",
	}, StatusOpen, 20)
	if err != nil || len(list) != 0 {
		t.Fatalf("other supplier: %v len=%d", err, len(list))
	}
}

type fakeRL struct {
	calls int
	last  ReverseLogisticsInput
}

func (f *fakeRL) OpenFromClaim(_ context.Context, in ReverseLogisticsInput) error {
	f.calls++
	f.last = in
	return nil
}

func TestResolveSettlementFlags(t *testing.T) {
	skip, store := resolveSettlementFlags(ApproveClaimRequest{SettlementMode: "STORE_CREDIT"})
	if !skip || !store {
		t.Fatalf("store credit flags skip=%v store=%v", skip, store)
	}
	skip, store = resolveSettlementFlags(ApproveClaimRequest{SettlementMode: "GATEWAY_REFUND"})
	if skip || store {
		t.Fatalf("gateway flags skip=%v store=%v", skip, store)
	}
	skip, store = resolveSettlementFlags(ApproveClaimRequest{SkipGatewayRefund: true})
	if !skip || store {
		t.Fatalf("legacy skip flags skip=%v store=%v", skip, store)
	}
}

type fakeStoreCr struct {
	calls int
	amt   int64
}

func (f *fakeStoreCr) ClearBalance(_ context.Context, _, _ string, amountMinor int64, _ string) error {
	f.calls++
	f.amt = amountMinor
	return nil
}

func TestApproveClaim_StoreCreditMode(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository()
	settler := &fakeSettler{res: SettlementResult{ChargebackID: "chargeback_x1", AmountMinor: 100, Mode: "LEDGER_ONLY"}}
	sc := &fakeStoreCr{}
	svc := NewService(Config{
		Repo:    repo,
		Orders:  fakeOrders{ok: true, o: completedOrder(now)},
		Settler: settler,
		StoreCr: sc,
		Now:     func() time.Time { return now },
		NewID:   func() string { return "x1" },
		Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	c, err := svc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeMissing,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, settlement, err := svc.ApproveClaim(context.Background(), auth.Claims{
		Subject: "admin", Role: auth.RoleAdmin, SupplierID: "sup-1",
	}, c.ClaimID, ApproveClaimRequest{SettlementMode: "STORE_CREDIT"})
	if err != nil {
		t.Fatal(err)
	}
	if sc.calls != 1 {
		t.Fatalf("store credit calls=%d", sc.calls)
	}
	if settlement.Mode != "LEDGER_AND_STORE_CREDIT" {
		t.Fatalf("mode=%s", settlement.Mode)
	}
}

func TestFileRetailerClaim_OpensReverseLogisticsForDamage(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository()
	rl := &fakeRL{}
	svc := NewService(Config{
		Repo:              repo,
		Orders:            fakeOrders{ok: true, o: completedOrder(now)},
		RL:                rl,
		EvidenceInspector: defaultTestEvidenceInspector(),
		Now:               func() time.Time { return now },
		NewID:             func() string { return "x1" },
		Log:               slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	c, err := svc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeDamaged,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 1, Reason: "DAMAGED"}},
		Evidences: []struct {
			EvidenceType EvidenceType `json:"evidence_type"`
			URI          string       `json:"uri"`
			MimeType     string       `json:"mime_type"`
		}{{EvidenceType: EvidencePhoto, URI: "https://storage.googleapis.com/pegasusx-test/evidence/p.jpg", MimeType: "image/jpeg"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if rl.calls != 1 {
		t.Fatalf("rl calls=%d want 1", rl.calls)
	}
	if rl.last.ClaimID != c.ClaimID || rl.last.OrderID != "ord-1" {
		t.Fatalf("rl input %+v", rl.last)
	}
}

func TestFileRetailerClaim_WrongRetailerForbidden(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository()
	svc := NewService(Config{
		Repo:   repo,
		Orders: fakeOrders{ok: true, o: completedOrder(now)},
		Now:    func() time.Time { return now },
		NewID:  func() string { return "x1" },
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	_, err := svc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-OTHER", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeMissing,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 1}},
	})
	if err != ErrForbidden {
		t.Fatalf("got %v want forbidden", err)
	}
}

func TestFileRetailerClaim_TeamManagerCanFileCashierCannot(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	seq := 0
	svc := NewService(Config{
		Repo:   NewMemoryRepository(),
		Orders: fakeOrders{ok: true, o: completedOrder(now)},
		Now:    func() time.Time { return now },
		NewID: func() string {
			seq++
			return fmt.Sprintf("mgr_%d", seq)
		},
		Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	mgr := auth.Claims{
		Subject: "user-mgr", Role: auth.RoleRetailer,
		RetailerOrgID: "ret-1", RetailerUserID: "user-mgr", RetailerRole: "MANAGER",
	}
	_, err := svc.FileRetailerClaim(context.Background(), mgr, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeMissing,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 1}},
	})
	if err != nil {
		t.Fatalf("MANAGER file: %v", err)
	}
	cashier := auth.Claims{
		Subject: "user-cash", Role: auth.RoleRetailer,
		RetailerOrgID: "ret-1", RetailerUserID: "user-cash", RetailerRole: "CASHIER",
	}
	_, err = svc.FileRetailerClaim(context.Background(), cashier, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeMissing,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 1}},
	})
	if err != ErrForbidden {
		t.Fatalf("CASHIER got %v want forbidden", err)
	}
}

func TestFileRetailerClaim_RejectsPlaceholderEvidenceURI(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	svc := NewService(Config{
		Repo:   NewMemoryRepository(),
		Orders: fakeOrders{ok: true, o: completedOrder(now)},
		Now:    func() time.Time { return now },
		NewID:  func() string { return "x1" },
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	_, err := svc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeDamaged,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 1}},
		Evidences: []struct {
			EvidenceType EvidenceType `json:"evidence_type"`
			URI          string       `json:"uri"`
			MimeType     string       `json:"mime_type"`
		}{{EvidenceType: EvidencePhoto, URI: "https://placehold.co/400x400?text=x"}},
	})
	if !errors.Is(err, ErrInvalidEvidenceURI) {
		t.Fatalf("got %v want invalid_evidence_uri", err)
	}
}

func TestFileRetailerClaim_HoldFailureCompensatesOpenClaim(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	stock := &fakeStoreStock{holdErr: fmt.Errorf("location_unavailable")}
	repo := NewMemoryRepository()
	svc := NewService(Config{
		Repo:              repo,
		Orders:            fakeOrders{ok: true, o: completedOrder(now)},
		StoreStock:        stock,
		EvidenceInspector: defaultTestEvidenceInspector(),
		Now:               func() time.Time { return now },
		NewID:             func() string { return "holdfail1" },
		Log:               slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	_, err := svc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeConcealedDamage,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 1}},
		Evidences: []struct {
			EvidenceType EvidenceType `json:"evidence_type"`
			URI          string       `json:"uri"`
			MimeType     string       `json:"mime_type"`
		}{{EvidenceType: EvidencePhoto, URI: "https://storage.googleapis.com/pegasusx-test/evidence/p.jpg"}},
	})
	if !errors.Is(err, ErrClaimStockHoldFailed) {
		t.Fatalf("got %v want claim_stock_hold_failed", err)
	}
	got, ok, gerr := repo.GetClaim(context.Background(), "clm_holdfail1")
	if gerr != nil {
		t.Fatalf("GetClaim error: %v", gerr)
	}
	if ok {
		t.Fatalf("expected atomic rollback (no persisted claim), got ok=%v status=%s", ok, got.Status)
	}
}

func TestFileRetailerClaim_DamageRequiresPhoto(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository()
	svc := NewService(Config{
		Repo:   repo,
		Orders: fakeOrders{ok: true, o: completedOrder(now)},
		Now:    func() time.Time { return now },
		NewID:  func() string { return "x1" },
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	_, err := svc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeDamaged,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 1, Reason: "DAMAGED"}},
	})
	if err != ErrEvidenceRequired {
		t.Fatalf("got %v want evidence required", err)
	}
}

func TestListOrderClaims_RetailerOwnership(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository()
	svc := NewService(Config{
		Repo:   repo,
		Orders: fakeOrders{ok: true, o: completedOrder(now)},
		Now:    func() time.Time { return now },
		NewID:  func() string { return "x1" },
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	_, err := svc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeMissing,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	// Owner can list.
	list, err := svc.ListOrderClaims(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1")
	if err != nil || len(list) != 1 {
		t.Fatalf("owner list: %v len=%d", err, len(list))
	}
	// Other retailer forbidden.
	_, err = svc.ListOrderClaims(context.Background(), auth.Claims{
		Subject: "ret-other", Role: auth.RoleRetailer,
	}, "ord-1")
	if err != ErrForbidden {
		t.Fatalf("got %v want forbidden", err)
	}
	// Admin allowed.
	list, err = svc.ListOrderClaims(context.Background(), auth.Claims{
		Subject: "admin", Role: auth.RoleAdmin,
	}, "ord-1")
	if err != nil || len(list) != 1 {
		t.Fatalf("admin list: %v", err)
	}
}

func TestFileRetailerClaim_BlocksOverClaimAcrossClaims(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository()
	id := 0
	svc := NewService(Config{
		Repo:   repo,
		Orders: fakeOrders{ok: true, o: completedOrder(now)}, // sku-1 qty 5
		Now:    func() time.Time { return now },
		NewID: func() string {
			id++
			return fmt.Sprintf("x%d", id)
		},
		Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	_, err := svc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeMissing,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 4}},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeMissing,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 2}}, // only 1 remaining
	})
	if err == nil {
		t.Fatal("expected cumulative over-claim error")
	}
}

func TestFileRetailerClaim_EmptyCurrencyUsesPack(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	o := completedOrder(now)
	o.Currency = ""
	svc := NewService(Config{
		Repo:   NewMemoryRepository(),
		Orders: fakeOrders{ok: true, o: o},
		Now:    func() time.Time { return now },
		NewID:  func() string { return "ccy1" },
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	c, err := svc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeMissing,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 1}},
	})
	if err != nil {
		t.Fatalf("FileRetailerClaim: %v", err)
	}
	if c.Currency != "UZS" {
		t.Fatalf("currency=%q want UZS from pack", c.Currency)
	}
}



func TestFileRetailerClaim_EmptyCurrencyPlannedFailsClosed(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	o := completedOrder(now)
	o.Currency = ""
	svc := NewService(Config{
		Repo:   NewMemoryRepository(),
		Orders: fakeOrders{ok: true, o: o},
		Now:    func() time.Time { return now },
		NewID:  func() string { return "ccy2" },
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	ctx := auth.WithClaims(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer, MarketCode: "EU",
	})
	_, err := svc.FileRetailerClaim(ctx, auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeMissing,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 1}},
	})
	if err != auth.ErrMarketPackNotShipped {
		t.Fatalf("err=%v want %v", err, auth.ErrMarketPackNotShipped)
	}
}

func TestCreateFromDriverException_EmptyCurrencyUsesPack(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	o := completedOrder(now)
	o.Currency = ""
	svc := NewService(Config{
		Repo:   NewMemoryRepository(),
		Orders: fakeOrders{ok: true, o: o},
		Now:    func() time.Time { return now },
		NewID:  func() string { return "drv1" },
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	c, err := svc.CreateFromDriverException(context.Background(), o, "drv-1", ClaimTypeMissing, []ClaimLine{{SKU: "sku-1", Quantity: 1}}, nil, "note")
	if err != nil {
		t.Fatalf("CreateFromDriverException: %v", err)
	}
	if c.Currency != "UZS" {
		t.Fatalf("currency=%q want UZS from pack", c.Currency)
	}
}

func TestFileRetailerClaim_ReusedPhotoEvidenceRefused(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository()
	svc := NewService(Config{
		Repo:              repo,
		Orders:            fakeOrders{ok: true, o: completedOrder(now)},
		EvidenceInspector: defaultTestEvidenceInspector(),
		Now:               func() time.Time { return now },
		NewID:             func() string { return "c1" },
		Log:               slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	photoURI := "https://storage.googleapis.com/pegasusx-test/evidence/damaged_box.jpg"

	// First claim with photo succeeds
	_, err := svc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeConcealedDamage,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 1, Reason: "DAMAGED"}},
		Evidences: []struct {
			EvidenceType EvidenceType `json:"evidence_type"`
			URI          string       `json:"uri"`
			MimeType     string       `json:"mime_type"`
		}{{EvidenceType: EvidencePhoto, URI: photoURI, MimeType: "image/jpeg"}},
	})
	if err != nil {
		t.Fatalf("first claim should succeed: %v", err)
	}

	// Second claim with the EXACT SAME photo URI must be refused per B2B Operating Model:
	// "A reused photo is refused."
	svc2 := NewService(Config{
		Repo:              repo,
		Orders:            fakeOrders{ok: true, o: completedOrder(now)},
		EvidenceInspector: defaultTestEvidenceInspector(),
		Now:               func() time.Time { return now },
		NewID:             func() string { return "c2" },
		Log:               slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	_, err = svc2.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeConcealedDamage,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 1, Reason: "DAMAGED"}},
		Evidences: []struct {
			EvidenceType EvidenceType `json:"evidence_type"`
			URI          string       `json:"uri"`
			MimeType     string       `json:"mime_type"`
		}{{EvidenceType: EvidencePhoto, URI: photoURI, MimeType: "image/jpeg"}},
	})
	if !errors.Is(err, ErrReusedPhotoEvidence) {
		t.Fatalf("want ErrReusedPhotoEvidence, got %v", err)
	}
}

func TestListOrderClaims_MultiUserRetailerOrgID(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository()
	svc := NewService(Config{
		Repo:   repo,
		Orders: fakeOrders{ok: true, o: completedOrder(now)}, // RetailerID: "ret-1"
		Now:    func() time.Time { return now },
		NewID:  func() string { return "mu1" },
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	_, err := svc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeMissing,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Staff member in the same RetailerOrgID with a distinct Subject must be allowed.
	list, err := svc.ListOrderClaims(context.Background(), auth.Claims{
		Subject:       "staff-buyer-99",
		Role:          auth.RoleRetailer,
		RetailerOrgID: "ret-1",
	}, "ord-1")
	if err != nil || len(list) != 1 {
		t.Fatalf("multi-user org member err=%v len=%d", err, len(list))
	}

	// Forged Subject == "ret-1" but wrong RetailerOrgID == "ret-evil" must be forbidden.
	_, err = svc.ListOrderClaims(context.Background(), auth.Claims{
		Subject:       "ret-1",
		Role:          auth.RoleRetailer,
		RetailerOrgID: "ret-evil",
	}, "ord-1")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for mismatched RetailerOrgID, got %v", err)
	}
}

func TestListSupplierClaimsPage_CursorPagination(t *testing.T) {
	now := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository()
	svc := NewService(Config{
		Repo: repo,
		Now:  func() time.Time { return now },
		Log:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	// Seed 5 claims (including two with identical CreatedAt to verify ClaimID tie-breaking)
	seed := []Claim{
		{ClaimID: "clm_a", OrderID: "ord-p", SupplierID: "sup-1", Status: StatusOpen, CreatedAt: now},
		{ClaimID: "clm_b", OrderID: "ord-p", SupplierID: "sup-1", Status: StatusOpen, CreatedAt: now},
		{ClaimID: "clm_c", OrderID: "ord-p", SupplierID: "sup-1", Status: StatusOpen, CreatedAt: now.Add(time.Minute)},
		{ClaimID: "clm_d", OrderID: "ord-p", SupplierID: "sup-1", Status: StatusOpen, CreatedAt: now.Add(2 * time.Minute)},
		{ClaimID: "clm_e", OrderID: "ord-p", SupplierID: "sup-1", Status: StatusOpen, CreatedAt: now.Add(3 * time.Minute)},
		{ClaimID: "clm_other", OrderID: "ord-x", SupplierID: "sup-2", Status: StatusOpen, CreatedAt: now.Add(4 * time.Minute)},
	}
	for _, c := range seed {
		if err := repo.CreateClaim(context.Background(), c, nil); err != nil {
			t.Fatal(err)
		}
	}

	actor := auth.Claims{Subject: "admin-1", Role: auth.RoleAdmin, SupplierID: "sup-1"}
	seen := make([]string, 0, 5)
	cursor := ""
	for pageNum := 0; pageNum < 5; pageNum++ {
		page, next, err := svc.ListSupplierClaimsPage(context.Background(), actor, StatusOpen, 2, cursor)
		if err != nil {
			t.Fatalf("page %d err=%v", pageNum, err)
		}
		for _, c := range page {
			seen = append(seen, c.ClaimID)
		}
		if next == "" {
			break
		}
		cursor = next
	}
	want := []string{"clm_e", "clm_d", "clm_c", "clm_b", "clm_a"}
	if fmt.Sprint(seen) != fmt.Sprint(want) {
		t.Fatalf("got %v want %v", seen, want)
	}
	if _, _, err := svc.ListSupplierClaimsPage(context.Background(), actor, StatusOpen, 2, "not-valid-base64!!!"); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("expected ErrInvalidCursor, got %v", err)
	}
}

func TestNewService_DefaultIDFormatAndLength(t *testing.T) {
	svc := NewService(Config{
		Repo: NewMemoryRepository(),
		Log:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	id1 := svc.newID()
	id2 := svc.newID()
	if id1 == id2 || len(id1) != 32 {
		t.Fatalf("expected distinct 32-char UUID hex IDs, got %q and %q", id1, id2)
	}
}





type failingDedupeRepo struct {
	*MemoryRepository
	dedupeErr error
}

func (r *failingDedupeRepo) IsEvidenceReused(context.Context, string, string, string) (bool, error) {
	return false, r.dedupeErr
}

func (r *failingDedupeRepo) IsEvidenceURIReused(context.Context, string) (bool, error) {
	return false, r.dedupeErr
}

func TestFileRetailerClaim_FailClosedWithoutStorageClient(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	prevClient := storage.Client
	storage.Client = nil
	t.Cleanup(func() { storage.Client = prevClient })

	repo := NewMemoryRepository()
	svc := NewService(Config{
		Repo:   repo,
		Orders: fakeOrders{ok: true, o: completedOrder(now)},
		Now:    func() time.Time { return now },
		NewID:  func() string { return "nofailopen" },
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	_, err := svc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeDamaged,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 1}},
		Evidences: []struct {
			EvidenceType EvidenceType `json:"evidence_type"`
			URI          string       `json:"uri"`
			MimeType     string       `json:"mime_type"`
		}{{EvidenceType: EvidencePhoto, URI: "https://storage.googleapis.com/pegasusx-test/evidence/p.jpg"}},
	})
	if !errors.Is(err, storage.ErrMediaStorageUnavailable) {
		t.Fatalf("want ErrMediaStorageUnavailable when GCS client nil, got %v", err)
	}
}

func TestFileRetailerClaim_BoundedStreamOver10MiBRejected(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	oversized := make([]byte, storage.MaxEvidenceUploadBytes+1)
	uri := "https://storage.googleapis.com/pegasusx-test/evidence/oversized.jpg"
	repo := NewMemoryRepository()
	svc := NewService(Config{
		Repo:              repo,
		Orders:            fakeOrders{ok: true, o: completedOrder(now)},
		EvidenceInspector: mapTestEvidenceInspector(map[string][]byte{uri: oversized}),
		Now:               func() time.Time { return now },
		NewID:             func() string { return "huge1" },
		Log:               slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	_, err := svc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeDamaged,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 1}},
		Evidences: []struct {
			EvidenceType EvidenceType `json:"evidence_type"`
			URI          string       `json:"uri"`
			MimeType     string       `json:"mime_type"`
		}{{EvidenceType: EvidencePhoto, URI: uri}},
	})
	if !errors.Is(err, storage.ErrInvalidUploadSize) {
		t.Fatalf("want ErrInvalidUploadSize for >10MiB photo, got %v", err)
	}
	if _, ok, _ := repo.GetClaim(context.Background(), "clm_huge1"); ok {
		t.Fatal("oversized evidence must fail before creating any claim")
	}
}

func TestFileRetailerClaim_ReusedContentHashDifferentURIRefused_AndHTTP409(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	identicalPhotoBytes := []byte("exact-identical-damage-jpeg-payload")
	uri1 := "https://storage.googleapis.com/pegasusx-test/evidence/claim1.jpg"
	uri2 := "https://storage.googleapis.com/pegasusx-test/evidence/renamed/claim2.jpg"

	repo := NewMemoryRepository()
	idSeq := 0
	svc := NewService(Config{
		Repo:   repo,
		Orders: fakeOrders{ok: true, o: completedOrder(now)},
		EvidenceInspector: mapTestEvidenceInspector(map[string][]byte{
			uri1: identicalPhotoBytes,
			uri2: identicalPhotoBytes,
		}),
		Now:   func() time.Time { return now },
		NewID: func() string { idSeq++; return fmt.Sprintf("csha%d", idSeq) },
		Log:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	c1, err := svc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeDamaged,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 1}},
		Evidences: []struct {
			EvidenceType EvidenceType `json:"evidence_type"`
			URI          string       `json:"uri"`
			MimeType     string       `json:"mime_type"`
		}{{EvidenceType: EvidencePhoto, URI: uri1}},
	})
	if err != nil {
		t.Fatalf("first claim failed: %v", err)
	}
	if len(c1.Evidences) != 1 || len(c1.Evidences[0].ContentSha256) != 64 || len(c1.Evidences[0].UriSha256) != 64 || c1.Evidences[0].ObjectGeneration != 8001 {
		t.Fatalf("expected persisted sha256 hashes + generation, got %+v", c1.Evidences)
	}

	// Second claim uploads identical bytes under a DIFFERENT GCS path -> ContentSha256 collision refused!
	_, err = svc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeConcealedDamage,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 1}},
		Evidences: []struct {
			EvidenceType EvidenceType `json:"evidence_type"`
			URI          string       `json:"uri"`
			MimeType     string       `json:"mime_type"`
		}{{EvidenceType: EvidencePhoto, URI: uri2}},
	})
	if !errors.Is(err, ErrReusedPhotoEvidence) {
		t.Fatalf("want ErrReusedPhotoEvidence on ContentSha256 match across different URIs, got %v", err)
	}

	// HTTP error mapping returns 409 Conflict for ErrReusedPhotoEvidence.
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/orders/ord-1/claims", nil)
	writeClaimError(rr, req, ErrReusedPhotoEvidence)
	if rr.Code != http.StatusConflict {
		t.Fatalf("writeClaimError(ErrReusedPhotoEvidence) status = %d, want 409", rr.Code)
	}

	// Repository dedupe error must fail closed (never swallow DB error).
	failSvc := NewService(Config{
		Repo:              &failingDedupeRepo{MemoryRepository: repo, dedupeErr: errors.New("spanner table error")},
		Orders:            fakeOrders{ok: true, o: completedOrder(now)},
		EvidenceInspector: mapTestEvidenceInspector(map[string][]byte{uri2: []byte("distinct-bytes")}),
		Now:               func() time.Time { return now },
		Log:               slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	_, err = failSvc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeDamaged,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 1}},
		Evidences: []struct {
			EvidenceType EvidenceType `json:"evidence_type"`
			URI          string       `json:"uri"`
			MimeType     string       `json:"mime_type"`
		}{{EvidenceType: EvidencePhoto, URI: uri2}},
	})
	if err == nil || !strings.Contains(err.Error(), "spanner table error") {
		t.Fatalf("expected fail-closed dedupe error propagation, got %v", err)
	}
}

func TestSpannerRepository_IsEvidenceURIReused_FailClosed(t *testing.T) {
	repo := NewSpannerRepository(nil)
	if reused, err := repo.IsEvidenceURIReused(context.Background(), "https://storage.googleapis.com/b/o.jpg"); err == nil || reused {
		t.Fatalf("expected fail-closed error for nil spanner client in IsEvidenceURIReused, got reused=%v err=%v", reused, err)
	}
	if reused, err := repo.IsEvidenceReused(context.Background(), "https://storage.googleapis.com/b/o.jpg", "u", "c"); err == nil || reused {
		t.Fatalf("expected fail-closed error for nil spanner client in IsEvidenceReused, got reused=%v err=%v", reused, err)
	}
}

func TestFileRetailerClaim_DHashOffAndEscalationMetadata(t *testing.T) {
	t.Setenv("CLAIM_AUTO_APPROVE_MAX_MINOR", "0")
	t.Setenv("CLAIM_EVIDENCE_DHASH_ENABLED", "true")
	if EvidenceDHashEnabled() {
		t.Fatal("dHash MUST stay off while CLAIM_AUTO_APPROVE_MAX_MINOR == 0")
	}

	t.Setenv("CLAIM_ESCALATION_THRESHOLD_MINOR", "3500")
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository()
	idSeq := 0
	svc := NewService(Config{
		Repo:              repo,
		Orders:            fakeOrders{ok: true, o: completedOrder(now)},
		EvidenceInspector: defaultTestEvidenceInspector(),
		Now:               func() time.Time { return now },
		NewID:             func() string { idSeq++; return fmt.Sprintf("esc%d", idSeq) },
		Log:               slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	// High amount triggers escalation metadata even when dHash is off.
	c1, err := svc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeMissing,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 2}}, // 4000 >= 3500 threshold
	})
	if err != nil {
		t.Fatalf("FileRetailerClaim: %v", err)
	}
	if c1.Escalation == nil || !c1.Escalation.EscalationRequired || c1.Escalation.DHashStatus != "off" {
		t.Fatalf("expected escalation required with dhash off, got %+v", c1.Escalation)
	}

	// Repeat claim on same order stays escalated and blocks auto-approve even if CLAIM_AUTO_APPROVE_MAX_MINOR > 0.
	t.Setenv("CLAIM_AUTO_APPROVE_MAX_MINOR", "50000")
	t.Setenv("CLAIM_ESCALATION_THRESHOLD_MINOR", "0")
	c2, err := svc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeMissing,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 1}}, // second claim on ord-1
	})
	if err != nil {
		t.Fatalf("second claim on order: %v", err)
	}
	if c2.Escalation == nil || !c2.Escalation.EscalationRequired || c2.Status != StatusOpen {
		t.Fatalf("expected repeat order claim to escalate and block auto-approve (status=%s esc=%+v)", c2.Status, c2.Escalation)
	}
}

func TestFileRetailerClaim_SingleTxnAutoApproveAtomic(t *testing.T) {
	t.Setenv("CLAIM_AUTO_APPROVE_MAX_MINOR", "25000")
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	stock := &fakeStoreStock{}
	repo := NewMemoryRepository()
	svc := NewService(Config{
		Repo:              repo,
		Orders:            fakeOrders{ok: true, o: completedOrder(now)},
		StoreStock:        stock,
		EvidenceInspector: defaultTestEvidenceInspector(),
		Now:               func() time.Time { return now },
		NewID:             func() string { return "auto1" },
		Log:               slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	c, err := svc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeDamaged,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 1}},
		Evidences: []struct {
			EvidenceType EvidenceType `json:"evidence_type"`
			URI          string       `json:"uri"`
			MimeType     string       `json:"mime_type"`
		}{{EvidenceType: EvidencePhoto, URI: "https://storage.googleapis.com/pegasusx-test/evidence/a1.jpg"}},
	})
	if err != nil {
		t.Fatalf("single-txn auto-approve failed: %v", err)
	}
	if c.Status != StatusApproved || c.ResolvedBy != "system:auto_approve" {
		t.Fatalf("expected APPROVED by system:auto_approve, got status=%s by=%q", c.Status, c.ResolvedBy)
	}
	if stock.holds != 1 {
		t.Fatalf("expected 1 StoreStock hold, got %d", stock.holds)
	}

	// If StoreStock hold fails alongside auto-approve, zero claim is committed.
	badStock := &fakeStoreStock{holdErr: errors.New("quarantine_unavailable")}
	repo2 := NewMemoryRepository()
	svc2 := NewService(Config{
		Repo:              repo2,
		Orders:            fakeOrders{ok: true, o: completedOrder(now)},
		StoreStock:        badStock,
		EvidenceInspector: defaultTestEvidenceInspector(),
		Now:               func() time.Time { return now },
		NewID:             func() string { return "autofail" },
		Log:               slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if _, err := svc2.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1", FileClaimRequest{
		ClaimType: ClaimTypeDamaged,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 1}},
		Evidences: []struct {
			EvidenceType EvidenceType `json:"evidence_type"`
			URI          string       `json:"uri"`
			MimeType     string       `json:"mime_type"`
		}{{EvidenceType: EvidencePhoto, URI: "https://storage.googleapis.com/pegasusx-test/evidence/a2.jpg"}},
	}); !errors.Is(err, ErrClaimStockHoldFailed) {
		t.Fatalf("want ErrClaimStockHoldFailed, got %v", err)
	}
	if _, ok, _ := repo2.GetClaim(context.Background(), "clm_autofail"); ok {
		t.Fatal("expected zero claim committed when atomic hold fails")
	}
}

func TestApproveClaim_UnderReviewResidualBlocksConcurrentExceedingGross(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository()
	settler := &fakeSettler{err: errors.New("temporary gateway timeout")}
	var idCounter int64

	// Order delivered gross is 6,000 minor (discounted basket with 2 distinct SKUs)
	order := OrderSnapshot{
		OrderID:    "ord-concurrent",
		SupplierID: "sup-1",
		RetailerID: "ret-1",
		Status:     OrderStatusCompleted,
		UpdatedAt:  now.Add(-1 * time.Hour),
		Currency:   "UZS",
		TotalMinor: 6000,
		LineItems: []OrderLine{
			{SKU: "sku-1", Quantity: 4, UnitPriceMinor: 1000},
			{SKU: "sku-2", Quantity: 4, UnitPriceMinor: 1000},
		},
	}

	svc := NewService(Config{
		Repo:    repo,
		Orders:  fakeOrders{ok: true, o: order},
		Settler: settler,
		Now:     func() time.Time { return now },
		NewID: func() string {
			idCounter++
			return fmt.Sprintf("clm-id-%d", idCounter)
		},
		Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	// 1. File Claim 1 for sku-1 (4 items = 4,000 minor)
	c1, err := svc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-concurrent", FileClaimRequest{
		ClaimType: ClaimTypeMissing,
		LineItems: []ClaimLine{{SKU: "sku-1", Quantity: 4}},
	})
	if err != nil {
		t.Fatalf("File Claim 1: %v", err)
	}

	// 2. File Claim 2 for sku-2 (4 items = 4,000 minor)
	c2, err := svc.FileRetailerClaim(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-concurrent", FileClaimRequest{
		ClaimType: ClaimTypeMissing,
		LineItems: []ClaimLine{{SKU: "sku-2", Quantity: 4}},
	})
	if err != nil {
		t.Fatalf("File Claim 2: %v", err)
	}

	// 3. Attempt to approve Claim 1: settler fails, leaving Claim 1 in StatusUnderReview
	_, _, err = svc.ApproveClaim(context.Background(), auth.Claims{
		Subject: "admin-1", Role: auth.RoleAdmin,
	}, c1.ClaimID, ApproveClaimRequest{ResolutionNote: "initial attempt"})
	if err == nil {
		t.Fatal("expected settlement error")
	}

	loaded1, _, _ := repo.GetClaim(context.Background(), c1.ClaimID)
	if loaded1.Status != StatusUnderReview {
		t.Fatalf("expected Claim 1 in StatusUnderReview, got %s", loaded1.Status)
	}

	// 4. Check GetRemainingClaimable on the order:
	// Total: 6,000; Under Review: 4,000; Remaining: 2,000
	rem, err := svc.GetRemainingClaimable(context.Background(), "ord-concurrent")
	if err != nil {
		t.Fatalf("GetRemainingClaimable: %v", err)
	}
	if rem.AlreadyClaimedMinor != 4000 || rem.RemainingClaimableMinor != 2000 {
		t.Fatalf("expected already claimed 4000, remaining 2000, got already=%d remaining=%d",
			rem.AlreadyClaimedMinor, rem.RemainingClaimableMinor)
	}

	// 5. Attempt to approve Claim 2 for 4,000 minor:
	// MUST FAIL because 4,000 > remaining 2,000!
	_, _, err = svc.ApproveClaim(context.Background(), auth.Claims{
		Subject: "admin-1", Role: auth.RoleAdmin,
	}, c2.ClaimID, ApproveClaimRequest{ResolutionNote: "claim 2 attempt"})
	if !errors.Is(err, ErrPricingFailed) {
		t.Fatalf("expected ErrPricingFailed on concurrent claim exceeding remaining gross, got %v", err)
	}

	// 6. Claim 1 retry should succeed (not blocking itself) once settler is healthy
	settler.err = nil
	resolved1, _, err := svc.ApproveClaim(context.Background(), auth.Claims{
		Subject: "admin-1", Role: auth.RoleAdmin,
	}, c1.ClaimID, ApproveClaimRequest{ResolutionNote: "claim 1 retry"})
	if err != nil {
		t.Fatalf("expected Claim 1 retry to succeed, got %v", err)
	}
	if resolved1.Status != StatusResolved {
		t.Fatalf("expected Claim 1 StatusResolved, got %s", resolved1.Status)
	}

	// 7. Claim 2 with requested partial amount 2,000 should now succeed within remaining 2,000
	approved2, _, err := svc.ApproveClaim(context.Background(), auth.Claims{
		Subject: "admin-1", Role: auth.RoleAdmin,
	}, c2.ClaimID, ApproveClaimRequest{AmountMinor: 2000, ResolutionNote: "claim 2 partial approve"})
	if err != nil {
		t.Fatalf("expected partial approve of 2000 to succeed, got %v", err)
	}
	if approved2.Status != StatusResolved {
		t.Fatalf("expected Claim 2 StatusResolved, got %s", approved2.Status)
	}
}
