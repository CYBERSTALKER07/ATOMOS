package retailer

import (
	"testing"
	"time"
)

func TestNormalizeExecutionMode(t *testing.T) {
	cases := map[string]string{
		"":       AutoOrderModeDraft,
		"DRAFT":  AutoOrderModeDraft,
		"shadow": AutoOrderModeShadow,
		"off":    AutoOrderModeOff,
		"place":  AutoOrderModePlace,
		"bogus":  "",
	}
	for in, want := range cases {
		if got := NormalizeExecutionMode(in); got != want {
			t.Fatalf("NormalizeExecutionMode(%q)=%q want %q", in, got, want)
		}
	}
}

func TestCandidateAllowedPolicyMatrix(t *testing.T) {
	base := AutoOrderSettings{GlobalEnabled: true}
	c := AutoOrderCandidate{SKU: "sku-1", ProductID: "sku-1", SupplierID: "sup-1", CategoryID: "cat-1"}

	if !candidateAllowed(base, c) {
		t.Fatal("global on should allow")
	}

	// Supplier disable blocks under global on
	s := base
	s.SupplierOverrides = []SupplierOverride{{SupplierID: "sup-1", Enabled: false}}
	if candidateAllowed(s, c) {
		t.Fatal("supplier disable should block under global on")
	}

	// Category disable blocks
	s = base
	s.CategoryOverrides = []CategoryOverride{{CategoryID: "cat-1", Enabled: false}}
	if candidateAllowed(s, c) {
		t.Fatal("category disable should block under global on")
	}

	// Product disable blocks
	s = base
	s.ProductOverrides = []ProductOverride{{ProductID: "sku-1", Enabled: false}}
	if candidateAllowed(s, c) {
		t.Fatal("product disable should block")
	}

	// Variant most-specific wins over product disable
	s = base
	s.ProductOverrides = []ProductOverride{{ProductID: "sku-1", Enabled: false}}
	s.VariantOverrides = []VariantOverride{{VariantID: "sku-1", Enabled: true}}
	if !candidateAllowed(s, c) {
		t.Fatal("variant enable should win over product disable")
	}

	// Global off + supplier enable
	s = AutoOrderSettings{
		GlobalEnabled:     false,
		SupplierOverrides: []SupplierOverride{{SupplierID: "sup-1", Enabled: true}},
	}
	if !candidateAllowed(s, c) {
		t.Fatal("scoped supplier enable should allow when global off")
	}

	// Global off + no match
	s = AutoOrderSettings{GlobalEnabled: false}
	if candidateAllowed(s, c) {
		t.Fatal("global off with no override should deny")
	}
}

func TestComputeOrderUpToQty(t *testing.T) {
	qty, s, ok := ComputeOrderUpToQty(5, 10, 4, 1, 1)
	if !ok || qty != 9 { // S=10+4=14, need=9
		t.Fatalf("qty=%d S=%v ok=%v want qty=9", qty, s, ok)
	}
	qty, _, ok = ComputeOrderUpToQty(20, 10, 4, 1, 1)
	if ok || qty != 0 {
		t.Fatalf("above ROP should not order: qty=%d ok=%v", qty, ok)
	}
	qty, _, ok = ComputeOrderUpToQty(5, 10, 4, 1, 6)
	if !ok || qty%6 != 0 {
		t.Fatalf("pack round failed qty=%d", qty)
	}
}

func TestShadowRunWritesNoPlace(t *testing.T) {
	n := 0
	svc := NewService(ServiceConfig{
		Now: time.Now,
		NewID: func() string {
			n++
			return "sh-" + string(rune('A'+n%26))
		},
	})
	_ = svc.saveAutoOrderDurable(t.Context(), "ret-shadow", "o", AutoOrderSettings{
		GlobalEnabled: true,
		ExecutionMode: AutoOrderModeShadow,
	})
	svc.SeedAutoOrderCandidates("ret-shadow", []AutoOrderCandidate{
		{SKU: "sku-a", ProductID: "sku-a", SupplierID: "sup-a", Qty: 3, IP: 2, ReorderPoint: 5, OrderUpTo: 8},
	})
	run := svc.RunAutoOrderForRetailer(t.Context(), "ret-shadow", AutoOrderModeShadow)
	if run.Mode != AutoOrderModeShadow {
		t.Fatalf("mode=%s", run.Mode)
	}
	if run.PlacedLines != 0 {
		t.Fatalf("shadow must not place: placed=%d", run.PlacedLines)
	}
	if run.DraftLines < 1 {
		t.Fatalf("expected shadow proposals in draft_lines, got %d status=%s msg=%s", run.DraftLines, run.Status, run.Message)
	}
	props, err := svc.listShadowProposals(t.Context(), "ret-shadow", 10)
	if err != nil || len(props) < 1 {
		t.Fatalf("shadow proposals: err=%v len=%d", err, len(props))
	}
}
