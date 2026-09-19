package factory

import "testing"

func TestPlanPredictiveTransfersSource(t *testing.T) {
	breached := []breachedSKU{{
		SupplierID: "s1", WarehouseID: "w1", ProductID: "p1",
		Deficit: 10, UnitVU: 1,
	}}
	got := PlanPredictiveTransfers(NetworkModeBalanced, breached, func(_, _, _, _ string) (string, error) {
		return "fac-1", nil
	}, nil)
	if len(got) != 1 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].Source != TransferSourcePredicted {
		t.Fatalf("source=%s", got[0].Source)
	}
}

func TestPlanPredictiveTransfersManualOnly(t *testing.T) {
	got := PlanPredictiveTransfers(NetworkModeManualOnly, []breachedSKU{{Deficit: 5, WarehouseID: "w"}}, nil, nil)
	if len(got) != 0 {
		t.Fatalf("manual-only must skip, got %d", len(got))
	}
}

func TestProjectedDeficit(t *testing.T) {
	if projectedDeficit(100, 20, 10) != 0 {
		t.Fatal("still above safety")
	}
	if d := projectedDeficit(30, 20, 20); d != 10 {
		t.Fatalf("deficit=%d", d)
	}
}

func TestApplyFactoryCreateDefaults(t *testing.T) {
	f := Factory{Name: "A"}
	applyFactoryCreateDefaults(&f)
	if f.DailyOutputCapacity != DefaultDailyOutputCapacity {
		t.Fatalf("capacity=%d", f.DailyOutputCapacity)
	}
	f.DailyOutputCapacity = 1200
	applyFactoryCreateDefaults(&f)
	if f.DailyOutputCapacity != 1200 {
		t.Fatal("must not overwrite explicit capacity")
	}
}
