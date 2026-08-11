package retailer

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/civil"
)

func newSoakTestService() *Service {
	s := &Service{
		now:   func() time.Time { return time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC) },
		newID: func() string { return "id-1" },
	}
	return s
}

func seedShadowProposals(s *Service, orgID string, n int) {
	day := civil.DateOf(s.now().UTC())
	s.shadowProposalsMem = map[string][]AutoOrderShadowProposal{}
	for i := 0; i < n; i++ {
		s.shadowProposalsMem[orgID] = append(s.shadowProposalsMem[orgID], AutoOrderShadowProposal{
			ProposalID:  "p" + string(rune('a'+i%26)),
			RetailerID:  orgID,
			SKU:         "sku-1",
			ProposedQty: 10,
			BucketDate:  day.String(),
			Status:      ShadowProposalStatusOpen,
		})
	}
}

func TestSoakGate_DisabledAllows(t *testing.T) {
	s := newSoakTestService()
	d := s.EvaluateSoakGate(context.Background(), "org1", SoakGateConfig{Disabled: true})
	if !d.Allowed {
		t.Fatalf("disabled gate should allow, got %+v", d)
	}
}

func TestSoakGate_InsufficientProposalsDenies(t *testing.T) {
	s := newSoakTestService()
	seedShadowProposals(s, "org1", 5) // below default 20
	d := s.EvaluateSoakGate(context.Background(), "org1", SoakGateConfig{MinProposals: 20, MaxWAPE: 0.30, MinUnmodified: 0.60})
	if d.Allowed {
		t.Fatalf("expected deny for insufficient proposals, got %+v", d)
	}
	found := false
	for _, r := range d.Reasons {
		if r == "insufficient_shadow_proposals" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected insufficient_shadow_proposals reason, got %v", d.Reasons)
	}
}

func TestSoakGate_EnoughProposalsButNoMatchedOrdersDenies(t *testing.T) {
	s := newSoakTestService()
	seedShadowProposals(s, "org1", 25) // enough, but no Orders in memory → no matches
	d := s.EvaluateSoakGate(context.Background(), "org1", SoakGateConfig{MinProposals: 20, MaxWAPE: 0.30, MinUnmodified: 0.60})
	if d.Allowed {
		t.Fatalf("expected deny when no matched orders, got %+v", d)
	}
	if d.Stats == nil || d.Stats.ProposalCount != 25 {
		t.Fatalf("expected 25 proposals in stats, got %+v", d.Stats)
	}
}

func TestPlaceAllowedForRetailer_RequiresFlagAndGate(t *testing.T) {
	s := newSoakTestService()
	seedShadowProposals(s, "org1", 25)
	// Flag off → false regardless of gate.
	s.autoOrderPlaceEnabled = false
	if s.placeAllowedForRetailer(context.Background(), "org1") {
		t.Fatal("place must be blocked when process flag is off")
	}
	// Flag on but no matched orders (env gate defaults) → false.
	s.autoOrderPlaceEnabled = true
	t.Setenv("AUTO_ORDER_SOAK_GATE_DISABLED", "")
	if s.placeAllowedForRetailer(context.Background(), "org1") {
		t.Fatal("place must be blocked when soak gate fails")
	}
	// Break-glass bypass → true.
	t.Setenv("AUTO_ORDER_SOAK_GATE_DISABLED", "true")
	if !s.placeAllowedForRetailer(context.Background(), "org1") {
		t.Fatal("place must be allowed when gate disabled and flag on")
	}
}
