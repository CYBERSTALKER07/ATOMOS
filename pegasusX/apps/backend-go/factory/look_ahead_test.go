package factory

import "testing"

func TestShadowDeficit_TriggersWhenSafeToday(t *testing.T) {
	// stock 100 > typical safety, but 7d demand 100 → buffered 115 → deficit 15
	got := ShadowDeficit(100, 100)
	if got != 15 {
		t.Fatalf("deficit=%d want 15", got)
	}
}

func TestShadowDeficit_ZeroWhenCovered(t *testing.T) {
	if ShadowDeficit(10, 20) != 0 {
		t.Fatal("stock covers buffered demand")
	}
	if ShadowDeficit(0, 0) != 0 {
		t.Fatal("no future demand")
	}
}

func TestLookAheadConfirmationMapping(t *testing.T) {
	if !lookAheadConfirmed("CONFIRMED") || !lookAheadConfirmed("AUTO_CONFIRMED") {
		t.Fatal("committed statuses")
	}
	if lookAheadConfirmed("PENDING") || lookAheadConfirmed("DRAFT") || lookAheadConfirmed("REJECTED") {
		t.Fatal("X has no LOCKED; do not treat PENDING confirmation as committed")
	}
}

func TestSplitClassCVolumes(t *testing.T) {
	got := SplitClassCVolumes(850)
	if len(got) != 3 || got[0] != 400 || got[1] != 400 || got[2] != 50 {
		t.Fatalf("got %v", got)
	}
}

func TestPlanLookAheadTransfers_ManualOnlyEmpty(t *testing.T) {
	entries := []shadowDemandEntry{{
		SupplierID: "sup-1", WarehouseID: "wh-1", ProductID: "sku-1", ShadowDeficit: 5, UnitVU: 1,
	}}
	got := PlanLookAheadTransfers(NetworkModeManualOnly, entries,
		func(_, _, _, _ string) (string, error) {
			t.Fatal("picker must not run in MANUAL_ONLY")
			return "fac", nil
		},
		nil,
	)
	if len(got) != 0 {
		t.Fatalf("want empty, got %d", len(got))
	}
}

func TestParsePlanningLineItems_SKUAndProductID(t *testing.T) {
	got := parsePlanningLineItems([]byte(`[{"sku":"a","quantity":2},{"product_id":"b","quantity":3}]`))
	if len(got) != 2 || got[0].skuID() != "a" || got[1].skuID() != "b" {
		t.Fatalf("got %+v", got)
	}
}

func TestPlanLookAheadTransfers_SplitsAndPicks(t *testing.T) {
	entries := []shadowDemandEntry{{
		SupplierID: "sup-1", WarehouseID: "wh-1", ProductID: "sku-1",
		ShadowDeficit: 2, UnitVU: 250,
	}}
	got := PlanLookAheadTransfers(NetworkModeSpeed, entries,
		func(_, _, _, _ string) (string, error) { return "fac-a", nil },
		nil,
	)
	if len(got) != 2 {
		t.Fatalf("want 2 class-C splits, got %d", len(got))
	}
	if got[0].FactoryID != "fac-a" || got[0].Source != TransferSourceThreshold {
		t.Fatalf("%+v", got[0])
	}
}
