package factory

import "testing"

func TestPlanPullTransfers_WritesOnSafetyBreach(t *testing.T) {
	breached := []breachedSKU{{
		SupplierID:  "sup-1",
		WarehouseID: "wh-1",
		ProductID:   "sku-1",
		CurrentQty:  2,
		SafetyLevel: 10,
		Deficit:     8,
		UnitVU:      2,
	}}
	pick := func(supplierID, warehouseID, productID, mode string) (string, error) {
		if mode != NetworkModeSpeed {
			t.Fatalf("mode=%s", mode)
		}
		return "fac-opt", nil
	}
	got := PlanPullTransfers(NetworkModeSpeed, "CRON", breached, pick, nil)
	if len(got) != 1 {
		t.Fatalf("transfers=%d want 1", len(got))
	}
	if got[0].FactoryID != "fac-opt" || got[0].Source != TransferSourceThreshold {
		t.Fatalf("got %+v", got[0])
	}
	if got[0].State != TransferStateCreated {
		t.Fatalf("state=%s want CREATED", got[0].State)
	}
	if got[0].TotalVU != 16 {
		t.Fatalf("vu=%v want 16", got[0].TotalVU)
	}
}

func TestPlanPullTransfers_ManualRunStillSystemThreshold(t *testing.T) {
	breached := []breachedSKU{{
		SupplierID: "sup-1", WarehouseID: "wh-1", ProductID: "sku-1", Deficit: 4, UnitVU: 1,
	}}
	got := PlanPullTransfers(NetworkModeSpeed, "MANUAL", breached,
		func(_, _, _, _ string) (string, error) { return "fac-1", nil },
		nil,
	)
	if len(got) != 1 || got[0].Source != TransferSourceThreshold {
		t.Fatalf("manual pull-matrix must still be SYSTEM_THRESHOLD, got %+v", got)
	}
}

func TestPlanPullTransfers_ManualOnlyEmpty(t *testing.T) {
	breached := []breachedSKU{{
		SupplierID: "sup-1", WarehouseID: "wh-1", ProductID: "sku-1", Deficit: 5, UnitVU: 1,
	}}
	got := PlanPullTransfers(NetworkModeManualOnly, "CRON", breached, func(_, _, _, _ string) (string, error) {
		t.Fatal("picker must not run in MANUAL_ONLY")
		return "fac", nil
	}, nil)
	if len(got) != 0 {
		t.Fatalf("want empty, got %d", len(got))
	}
}

func TestPlanPullTransfers_LockDeniedSkipped(t *testing.T) {
	breached := []breachedSKU{{
		SupplierID: "sup-1", WarehouseID: "wh-1", ProductID: "sku-1", Deficit: 5, UnitVU: 1,
	}}
	got := PlanPullTransfers(NetworkModeSpeed, "CRON", breached,
		func(_, _, _, _ string) (string, error) { return "fac-1", nil },
		func(_, _, _, _ string) bool { return false },
	)
	if len(got) != 0 {
		t.Fatalf("want skip on lock deny, got %d", len(got))
	}
}

func TestSafetyDeficit(t *testing.T) {
	if safetyDeficit(12, 10) != 0 {
		t.Fatal("above safety is not a breach")
	}
	if safetyDeficit(3, 10) != 7 {
		t.Fatal("deficit")
	}
}
