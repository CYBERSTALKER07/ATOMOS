package claims

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

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
		Repo:   repo,
		Orders: fakeOrders{ok: true, o: completedOrder(now)},
		Now:    func() time.Time { return now },
		NewID:  func() string { return "x1" },
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
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
		}{{EvidenceType: EvidencePhoto, URI: "https://cdn.example/p.jpg", MimeType: "image/jpeg"}},
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
