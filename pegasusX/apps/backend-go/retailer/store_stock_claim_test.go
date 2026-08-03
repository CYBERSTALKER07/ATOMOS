package retailer

import (
	"context"
	"testing"
	"time"
)

func TestHoldForClaimAndResolve(t *testing.T) {
	t.Parallel()
	n := 0
	svc := NewService(ServiceConfig{
		Now:   func() time.Time { return time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC) },
		NewID: func() string { n++; return "id-" + string(rune('A'+n%26)) },
	})
	ctx := context.Background()
	// Seed FLOOR stock via adjust
	loc, err := svc.EnsurePrimaryLocation(ctx, "ret-claim")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.applyDelta(ctx, "ret-claim", loc.LocationID, BinFloor, "SKU-Q", 5, MoveReceive, "ORDER", "ord-1", "u1", "seed"); err != nil {
		t.Fatal(err)
	}
	if err := svc.HoldForClaim(ctx, "ret-claim", "clm-1", "ord-1", []ClaimHoldLine{{SKU: "SKU-Q", Qty: 3}}, "u1"); err != nil {
		t.Fatal(err)
	}
	if got := svc.onHandInBin(ctx, loc.LocationID, BinFloor, "SKU-Q"); got != 2 {
		t.Fatalf("floor=%d want 2", got)
	}
	if got := svc.onHandInBin(ctx, loc.LocationID, BinQuarantine, "SKU-Q"); got != 3 {
		t.Fatalf("quarantine=%d want 3", got)
	}
	// Approve → RETURN removes quarantine
	if err := svc.ResolveClaimStock(ctx, "ret-claim", "clm-1", []ClaimHoldLine{{SKU: "SKU-Q", Qty: 3}}, ClaimStockReturn, "admin"); err != nil {
		t.Fatal(err)
	}
	if got := svc.onHandInBin(ctx, loc.LocationID, BinQuarantine, "SKU-Q"); got != 0 {
		t.Fatalf("quarantine after return=%d want 0", got)
	}

	// Restore path
	_ = svc.applyDelta(ctx, "ret-claim", loc.LocationID, BinFloor, "SKU-R", 2, MoveReceive, "ORDER", "ord-2", "u1", "seed")
	_ = svc.HoldForClaim(ctx, "ret-claim", "clm-2", "ord-2", []ClaimHoldLine{{SKU: "SKU-R", Qty: 2}}, "u1")
	_ = svc.ResolveClaimStock(ctx, "ret-claim", "clm-2", []ClaimHoldLine{{SKU: "SKU-R", Qty: 2}}, ClaimStockRestore, "admin")
	if got := svc.onHandInBin(ctx, loc.LocationID, BinFloor, "SKU-R"); got != 2 {
		t.Fatalf("floor after restore=%d want 2", got)
	}
}
