package retailer

import "testing"

func TestReceivableQty_ExcludesRemaining(t *testing.T) {
	// order 10, delivered 7, remaining 3 → receive max 7 (delivered already net of residual)
	got := ReceivableQty(10, 7, 3, 0, 0)
	if got != 7 {
		t.Fatalf("got %d want 7", got)
	}
	// When only ordered+remaining (no delivered_qty): 10 - 3 = 7
	got = ReceivableQty(10, 0, 3, 0, 0)
	if got != 7 {
		t.Fatalf("got %d want 7", got)
	}
	// Open claim qty further reduces receivable
	got = ReceivableQty(10, 10, 0, 0, 2)
	if got != 8 {
		t.Fatalf("got %d want 8", got)
	}
	if ReceivableQty(5, 0, 9, 0, 0) != 0 {
		t.Fatal("must not go negative")
	}
}
