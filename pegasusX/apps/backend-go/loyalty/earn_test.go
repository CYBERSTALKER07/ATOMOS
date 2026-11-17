package loyalty

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestPointsFor(t *testing.T) {
	// 1,000,000 tiyin = 10,000 UZS at 100 bps (1%) = 100 points
	if got := PointsFor(1_000_000, 100); got != 100 {
		t.Fatalf("expected 100 points for 1,000,000 tiyin at 100 bps, got %d", got)
	}
	// 10,000 tiyin = 100 UZS at 100 bps = 1 point
	if got := PointsFor(10_000, 100); got != 1 {
		t.Fatalf("expected 1 point for 10,000 tiyin at 100 bps, got %d", got)
	}
	// < 10,000 tiyin at 100 bps = 0 points
	if got := PointsFor(9_999, 100); got != 0 {
		t.Fatalf("expected 0 points for 9,999 tiyin at 100 bps, got %d", got)
	}
	if got := PointsFor(1, 100); got != 0 {
		t.Fatalf("floor toward zero, got %d", got)
	}
	if got := PointsFor(0, 100); got != 0 {
		t.Fatalf("zero amount, got %d", got)
	}
	if got := PointsFor(-5000, 100); got != 0 {
		t.Fatalf("negative amount, got %d", got)
	}
	if got := PointsFor(1_000_000, 0); got != 0 {
		t.Fatalf("zero earnBps, got %d", got)
	}
	if got := PointsFor(1_000_000, -100); got != 0 {
		t.Fatalf("negative earnBps, got %d", got)
	}
}

func TestClawbackBalance(t *testing.T) {
	// Normal deduction within available balance
	life, avail := ClawbackBalance(500, 300, 100)
	if life != 400 || avail != 200 {
		t.Fatalf("expected 400, 200, got %d, %d", life, avail)
	}

	// Clawback exceeds available points: clamps at 0
	life, avail = ClawbackBalance(500, 50, 100)
	if life != 400 || avail != 0 {
		t.Fatalf("expected 400, 0, got %d, %d", life, avail)
	}

	// Clawback exceeds both: clamps at 0
	life, avail = ClawbackBalance(50, 50, 100)
	if life != 0 || avail != 0 {
		t.Fatalf("expected 0, 0, got %d, %d", life, avail)
	}

	// Negative or zero clawback: no change
	life, avail = ClawbackBalance(500, 300, 0)
	if life != 500 || avail != 300 {
		t.Fatalf("expected 500, 300, got %d, %d", life, avail)
	}
	life, avail = ClawbackBalance(500, 300, -50)
	if life != 500 || avail != 300 {
		t.Fatalf("expected 500, 300, got %d, %d", life, avail)
	}
}

func TestDeriveClawbackLedgerID_PartialRefunds(t *testing.T) {
	// Case 1: Specific refundID provided
	ledgerID, orderID := deriveClawbackLedgerID("ord-100", "ref-xyz", 25, 250000)
	if ledgerID != "clawback:ord-100:ref-xyz" {
		t.Fatalf("want clawback:ord-100:ref-xyz, got %s", ledgerID)
	}
	if orderID != "ord-100#clawback:ref-xyz" {
		t.Fatalf("want ord-100#clawback:ref-xyz, got %s", orderID)
	}

	// Case 2: Empty refundID - first partial refund
	ledger1, _ := deriveClawbackLedgerID("ord-100", "", 10, 100000)
	if ledger1 != "clawback:ord-100:pts10:amt100000" {
		t.Fatalf("want clawback:ord-100:pts10:amt100000, got %s", ledger1)
	}

	// Case 3: Empty refundID - second partial refund with different amount
	ledger2, _ := deriveClawbackLedgerID("ord-100", "", 20, 200000)
	if ledger2 != "clawback:ord-100:pts20:amt200000" {
		t.Fatalf("want clawback:ord-100:pts20:amt200000, got %s", ledger2)
	}

	// Keys must be distinct so subsequent partial refunds are never dropped
	if ledger1 == ledger2 {
		t.Fatal("partial refund ledger IDs must not collide when amounts differ")
	}
}

func TestTierFor(t *testing.T) {
	cur, next := TierFor(0, DefaultTiers)
	if cur.Name != "STANDARD" || next == nil || next.Name != "SILVER" {
		t.Fatalf("got %+v %+v", cur, next)
	}
	cur, next = TierFor(50000, DefaultTiers)
	if cur.Name != "SILVER" || next == nil || next.Name != "GOLD" {
		t.Fatalf("got %+v %+v", cur, next)
	}
	cur, next = TierFor(200000, DefaultTiers)
	if cur.Name != "GOLD" || next != nil {
		t.Fatalf("got %+v %+v", cur, next)
	}
}

func TestHandleSupplierProgram_Validation(t *testing.T) {
	h := &Handlers{}

	// Helper to send PATCH request
	doPatch := func(body any) *httptest.ResponseRecorder {
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPatch, "/v1/supplier/loyalty/program", bytes.NewReader(raw))
		req = req.WithContext(auth.WithClaims(context.Background(), auth.Claims{
			Subject:    "test-admin",
			Role:       auth.RoleAdmin,
			SupplierID: "sup-1",
		}))
		rr := httptest.NewRecorder()
		h.HandleSupplierProgram(rr, req)
		return rr
	}

	// 1. Negative earn_bps -> 400 Bad Request
	rr := doPatch(map[string]any{
		"earn_bps": -1,
		"reason":   "negative earn bps",
	})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for negative earn_bps, got %d", rr.Code)
	}

	// 2. earn_bps > 10000 -> 400 Bad Request
	rr = doPatch(map[string]any{
		"earn_bps": 10001,
		"reason":   "earn bps too large",
	})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for earn_bps > 10000, got %d", rr.Code)
	}

	// 3. Missing reason -> 400 Bad Request
	rr = doPatch(map[string]any{
		"earn_bps": 100,
		"reason":   "   ",
	})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing reason, got %d", rr.Code)
	}

	// 4. Negative tier min_points -> 400 Bad Request
	rr = doPatch(map[string]any{
		"earn_bps": 100,
		"reason":   "negative min points",
		"tiers": []Tier{
			{Name: "STANDARD", MinPoints: -10},
		},
	})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for negative tier min points, got %d", rr.Code)
	}

	// 5. Decreasing tier min_points -> 400 Bad Request
	rr = doPatch(map[string]any{
		"earn_bps": 100,
		"reason":   "decreasing tiers",
		"tiers": []Tier{
			{Name: "SILVER", MinPoints: 1000},
			{Name: "GOLD", MinPoints: 500},
		},
	})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for decreasing tiers, got %d", rr.Code)
	}
}
