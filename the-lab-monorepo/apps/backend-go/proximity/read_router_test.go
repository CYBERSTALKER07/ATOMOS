package proximity

import (
	"math"
	"testing"

	"cloud.google.com/go/spanner"
)

type fakeReadRouter struct {
	forCalls int
	lastCell string
	forResp  *spanner.Client
}

func (f *fakeReadRouter) For(h3Cell string) *spanner.Client {
	f.forCalls++
	f.lastCell = h3Cell
	return f.forResp
}

func (f *fakeReadRouter) Primary() *spanner.Client { return nil }

func TestReadClientForCell_NoRouterUsesPrimary(t *testing.T) {
	primary := &spanner.Client{}
	got := ReadClientForCell(primary, nil, "872830828ffffff")
	if got != primary {
		t.Fatalf("expected primary client when router is nil")
	}
}

func TestReadClientForCell_NilPrimaryReturnsNil(t *testing.T) {
	router := &fakeReadRouter{forResp: &spanner.Client{}}
	got := ReadClientForCell(nil, router, "872830828ffffff")
	if got != nil {
		t.Fatalf("expected nil when primary client is nil")
	}
	if router.forCalls != 0 {
		t.Fatalf("expected router.For not to be called when primary is nil")
	}
}

func TestReadClientForCell_EmptyCellFallsBack(t *testing.T) {
	primary := &spanner.Client{}
	router := &fakeReadRouter{forResp: &spanner.Client{}}

	got := ReadClientForCell(primary, router, "")
	if got != primary {
		t.Fatalf("expected primary fallback for empty cell")
	}
	if router.forCalls != 0 {
		t.Fatalf("expected router.For not to be called for empty cell")
	}
}

func TestReadClientForRetailer_NoRouterUsesPrimary(t *testing.T) {
	primary := &spanner.Client{}
	got := ReadClientForRetailer(primary, nil, 41.2995, 69.2401)
	if got != primary {
		t.Fatalf("expected primary client when router is nil")
	}
}

func TestReadClientForRetailer_UsesRouterForValidCell(t *testing.T) {
	primary := &spanner.Client{}
	regional := &spanner.Client{}
	router := &fakeReadRouter{forResp: regional}

	got := ReadClientForRetailer(primary, router, 41.2995, 69.2401)
	if got != regional {
		t.Fatalf("expected regional client from router")
	}
	if router.forCalls != 1 {
		t.Fatalf("expected router.For to be called exactly once, got=%d", router.forCalls)
	}
	if router.lastCell == "" {
		t.Fatalf("expected non-empty H3 cell routed to For")
	}
}

func TestReadClientForRetailer_RouterNilResponseFallsBack(t *testing.T) {
	primary := &spanner.Client{}
	router := &fakeReadRouter{forResp: nil}

	got := ReadClientForRetailer(primary, router, 41.2995, 69.2401)
	if got != primary {
		t.Fatalf("expected primary fallback when router returns nil")
	}
}

func TestReadClientForRetailer_InvalidCoordinateFallsBack(t *testing.T) {
	primary := &spanner.Client{}
	router := &fakeReadRouter{forResp: &spanner.Client{}}

	got := ReadClientForRetailer(primary, router, math.NaN(), 69.2401)
	if got != primary {
		t.Fatalf("expected primary fallback for invalid coordinate")
	}
	if router.forCalls != 0 {
		t.Fatalf("expected router.For not to be called for invalid coordinate")
	}
}

func TestReadClientForRetailer_BackCompatWrapperMatches(t *testing.T) {
	primary := &spanner.Client{}
	regional := &spanner.Client{}
	router := &fakeReadRouter{forResp: regional}

	gotExported := ReadClientForRetailer(primary, router, 41.2995, 69.2401)
	gotCompat := readClientForRetailer(primary, router, 41.2995, 69.2401)

	if gotExported != gotCompat {
		t.Fatalf("compat wrapper mismatch: exported=%p compat=%p", gotExported, gotCompat)
	}
}
