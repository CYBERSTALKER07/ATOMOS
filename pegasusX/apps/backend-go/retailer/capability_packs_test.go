package retailer

import "testing"

func TestEvaluateEnable_POSRequiresStock(t *testing.T) {
	enabled := EnabledSet{}.WithCORE()
	res := EvaluateEnable(enabled, PackPOS, false)
	if res.Status != "BLOCKED" {
		t.Fatalf("expected BLOCKED, got %s (%s)", res.Status, res.Message)
	}
	if len(res.MissingHard) == 0 || res.MissingHard[0] != PackSTORESTOCK {
		t.Fatalf("expected missing STORE_STOCK, got %v", res.MissingHard)
	}
}

func TestEvaluateEnable_POSWithStockWarnsShifts(t *testing.T) {
	enabled := EnabledSet{PackSTORESTOCK: true}.WithCORE()
	res := EvaluateEnable(enabled, PackPOS, false)
	if res.Status != "WARN" {
		t.Fatalf("expected WARN, got %s", res.Status)
	}
	// soft: TEAM and SHIFTS
	if len(res.MissingSoft) == 0 {
		t.Fatalf("expected soft deps")
	}
	res2 := EvaluateEnable(enabled, PackPOS, true)
	if res2.Status != "OK" {
		t.Fatalf("accept soft should OK, got %s", res2.Status)
	}
}

func TestEvaluateDisable_BlocksDependents(t *testing.T) {
	enabled := EnabledSet{PackSTORESTOCK: true, PackPOS: true}.WithCORE()
	res := EvaluateDisable(enabled, PackSTORESTOCK)
	if res.Status != "BLOCKED" {
		t.Fatalf("expected BLOCKED when POS depends on stock, got %s", res.Status)
	}
}

func TestEvaluateDisable_CORE(t *testing.T) {
	res := EvaluateDisable(EnabledSet{}.WithCORE(), PackCORE)
	if res.Status != "BLOCKED" {
		t.Fatalf("cannot disable CORE")
	}
}

func TestEvaluateEnable_AlwaysOnCORE(t *testing.T) {
	res := EvaluateEnable(nil, PackCORE, false)
	if res.Status != "OK" {
		t.Fatalf("CORE always OK")
	}
}
