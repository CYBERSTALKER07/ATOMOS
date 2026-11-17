package payment

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestDoorstepShortfallReferenceIDs_IncludesLegacyAndCurrent(t *testing.T) {
	refs := doorstepShortfallReferenceIDs("ord-123")
	if len(refs) != 2 {
		t.Fatalf("len(refs) = %d, want 2 (%v)", len(refs), refs)
	}
	if refs[0] != "jentry_shortfall_ord-123" {
		t.Fatalf("refs[0] = %q, want jentry_shortfall_ord-123", refs[0])
	}
	if refs[1] != "jentry_doorstep_shortfall_ord-123" {
		t.Fatalf("refs[1] = %q, want jentry_doorstep_shortfall_ord-123", refs[1])
	}
}

func TestWriteDoorstepShortfall_FailsOnNilTransaction(t *testing.T) {
	err := WriteDoorstepShortfall(context.Background(), nil, "ord-1", "sup-1", "drv-1", "UZS", 10000, 7000, time.Now())
	if err == nil || !strings.Contains(err.Error(), "nil transaction") {
		t.Fatalf("expected nil transaction error, got %v", err)
	}
}
