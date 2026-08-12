package planning

import (
	"os"
	"testing"
)

func TestSandopUtilizationKeepsCapacitySeparate(t *testing.T) {
	// Demand must not be treated as capacity: util = demand/cap.
	pct, alert := sandopUtilization(9000, 7000, 10000)
	if pct < 128 || pct > 129 {
		t.Fatalf("expected ~128.57 utilization got %v", pct)
	}
	if !alert {
		t.Fatal("projected > factory capacity should alert")
	}

	pct, alert = sandopUtilization(5000, 7000, 10000)
	if pct < 71 || pct > 72 {
		t.Fatalf("expected ~71.4 utilization got %v", pct)
	}
	if alert {
		t.Fatal("demand under both caps should not alert")
	}

	pct, alert = sandopUtilization(8000, 10000, 7000)
	if !alert {
		t.Fatal("projected > warehouse inbound should alert")
	}
	if pct < 79 || pct > 81 {
		t.Fatalf("expected ~80 utilization got %v", pct)
	}
}

func TestSandopUtilizationNoDemandFallsBackToCapVsInbound(t *testing.T) {
	pct, alert := sandopUtilization(0, 8000, 5000)
	if pct != 160 {
		t.Fatalf("expected 160 got %v", pct)
	}
	if !alert {
		t.Fatal("factory > inbound should alert when no demand")
	}
}

func TestSopHorizonDays(t *testing.T) {
	t.Setenv("SOP_HORIZON_DAYS", "")
	if got := sopHorizonDays(); got != 7 {
		t.Fatalf("default horizon want 7 got %d", got)
	}
	t.Setenv("SOP_HORIZON_DAYS", "14")
	if got := sopHorizonDays(); got != 14 {
		t.Fatalf("want 14 got %d", got)
	}
	t.Setenv("SOP_HORIZON_DAYS", "28")
	if got := sopHorizonDays(); got != 28 {
		t.Fatalf("want 28 got %d", got)
	}
	t.Setenv("SOP_HORIZON_DAYS", "45")
	if got := sopHorizonDays(); got != 45 {
		t.Fatalf("want 45 got %d", got)
	}
	_ = os.Unsetenv("SOP_HORIZON_DAYS")
}
