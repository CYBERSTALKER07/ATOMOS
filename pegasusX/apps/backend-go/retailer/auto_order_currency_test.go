package retailer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

type memCartRepo struct {
	mu    sync.Mutex
	items []CartItem
}

func (m *memCartRepo) ListByRetailer(_ context.Context, retailerID, supplierID string) ([]CartItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []CartItem
	for _, it := range m.items {
		if it.RetailerID == retailerID && it.SupplierID == supplierID {
			out = append(out, it)
		}
	}
	return out, nil
}

func (m *memCartRepo) ListByRetailerAll(_ context.Context, retailerID string) ([]CartItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []CartItem
	for _, it := range m.items {
		if it.RetailerID == retailerID {
			out = append(out, it)
		}
	}
	return out, nil
}

func (m *memCartRepo) UpsertItems(_ context.Context, items []CartItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items = append(m.items, items...)
	return nil
}

func (m *memCartRepo) ClearCart(_ context.Context, retailerID, supplierID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	kept := m.items[:0]
	for _, it := range m.items {
		if it.RetailerID == retailerID && it.SupplierID == supplierID {
			continue
		}
		kept = append(kept, it)
	}
	m.items = kept
	return nil
}

func (m *memCartRepo) ClearCartAll(_ context.Context, retailerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	kept := m.items[:0]
	for _, it := range m.items {
		if it.RetailerID == retailerID {
			continue
		}
		kept = append(kept, it)
	}
	m.items = kept
	return nil
}

func TestAutoOrderCartCurrency_EmptyUsesPack(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	if got := autoOrderCartCurrency(context.Background(), "sup-uz"); got != "UZS" {
		t.Fatalf("currency=%q want UZS from shipped pack", got)
	}
}

func TestAutoOrderCartCurrency_PlannedDoesNotInvent(t *testing.T) {
	ctx := auth.WithClaims(context.Background(), auth.Claims{MarketCode: "EU", SupplierID: "sup-eu"})
	if got := autoOrderCartCurrency(ctx, "sup-eu"); got != "" {
		t.Fatalf("planned pack must not invent UZS, got %q", got)
	}
}

func TestAutoOrderDraftCart_EmptyCurrencyUsesPack(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	cart := &memCartRepo{}
	n := 0
	svc := NewService(ServiceConfig{
		Now:      time.Now,
		NewID:    func() string { n++; return "ao-ccy-" + string(rune('A'+n%26)) },
		CartRepo: cart,
	})
	_ = svc.saveAutoOrderDurable(t.Context(), "ret-ccy-uz", "o", AutoOrderSettings{GlobalEnabled: true})
	svc.SeedAutoOrderCandidates("ret-ccy-uz", []AutoOrderCandidate{
		{SKU: "MILK", ProductID: "MILK", SupplierID: "sup-ccy", Qty: 2},
	})
	run := svc.RunAutoOrderForRetailer(t.Context(), "ret-ccy-uz", AutoOrderModeDraft)
	if run.DraftLines != 1 || run.Status != "OK" {
		t.Fatalf("run=%+v", run)
	}
	items, err := cart.ListByRetailerAll(t.Context(), "ret-ccy-uz")
	if err != nil || len(items) != 1 {
		t.Fatalf("cart err=%v items=%+v", err, items)
	}
	if items[0].Currency != "UZS" {
		t.Fatalf("currency=%q want UZS from pack", items[0].Currency)
	}
}

func TestAutoOrderDraftCart_PlannedDoesNotInvent(t *testing.T) {
	cart := &memCartRepo{}
	n := 0
	svc := NewService(ServiceConfig{
		Now:      time.Now,
		NewID:    func() string { n++; return "ao-eu-" + string(rune('A'+n%26)) },
		CartRepo: cart,
	})
	_ = svc.saveAutoOrderDurable(t.Context(), "ret-ccy-eu", "o", AutoOrderSettings{GlobalEnabled: true})
	svc.SeedAutoOrderCandidates("ret-ccy-eu", []AutoOrderCandidate{
		{SKU: "EAU", ProductID: "EAU", SupplierID: "sup-eu", Qty: 1},
	})
	ctx := auth.WithClaims(t.Context(), auth.Claims{MarketCode: "EU", SupplierID: "sup-eu"})
	run := svc.RunAutoOrderForRetailer(ctx, "ret-ccy-eu", AutoOrderModeDraft)
	if run.DraftLines != 1 || run.Status != "OK" {
		t.Fatalf("run=%+v", run)
	}
	items, err := cart.ListByRetailerAll(t.Context(), "ret-ccy-eu")
	if err != nil || len(items) != 1 {
		t.Fatalf("cart err=%v items=%+v", err, items)
	}
	if items[0].Currency != "" {
		t.Fatalf("planned pack must not invent UZS, got %q", items[0].Currency)
	}
}

func TestAutoOrderPlaceFallbackCart_EmptyCurrencyUsesPack(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	cart := &memCartRepo{}
	n := 0
	svc := NewService(ServiceConfig{
		Now:      time.Now,
		NewID:    func() string { n++; return "ao-fb-" + string(rune('A'+n%26)) },
		CartRepo: cart,
		// Place stays off: no OrderCreator, AutoOrderPlaceEnabled default false.
	})
	_ = svc.saveAutoOrderDurable(t.Context(), "ret-ccy-fb", "o", AutoOrderSettings{GlobalEnabled: true})
	svc.SeedAutoOrderCandidates("ret-ccy-fb", []AutoOrderCandidate{
		{SKU: "BREAD", ProductID: "BREAD", SupplierID: "sup-fb", Qty: 1},
	})
	run := svc.RunAutoOrderForRetailer(t.Context(), "ret-ccy-fb", AutoOrderModePlace)
	if run.DraftLines != 1 {
		t.Fatalf("place-unavailable fallback should draft, got %+v", run)
	}
	if run.PlacedLines != 0 {
		t.Fatalf("place must stay off, placed=%d run=%+v", run.PlacedLines, run)
	}
	items, err := cart.ListByRetailerAll(t.Context(), "ret-ccy-fb")
	if err != nil || len(items) != 1 {
		t.Fatalf("cart err=%v items=%+v", err, items)
	}
	if items[0].Currency != "UZS" {
		t.Fatalf("fallback currency=%q want UZS from pack", items[0].Currency)
	}
}
