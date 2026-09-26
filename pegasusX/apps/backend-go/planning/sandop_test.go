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

func TestSAndOPSnapshotLabelsEnvDefaultCapacity(t *testing.T) {
	svc := &Service{}
	out, err := svc.GetSAndOP(t.Context(), "sup-1")
	if err == nil {
		t.Fatal("expected planning unavailable without Spanner")
	}
	if out.CapacitySource != "env_default" {
		t.Fatalf("capacity_source=%q want env_default", out.CapacitySource)
	}
	if out.CapacityModel != "production_lines" {
		t.Fatalf("capacity_model=%q want production_lines", out.CapacityModel)
	}
}

func TestSopFactoryCapacityColumnWinsEnv(t *testing.T) {
	units, src := sopFactoryCapacity(1200, 2, 700, 7)
	if src != "factories_column" || units != 1200*7 {
		t.Fatalf("got units=%d src=%s", units, src)
	}
	units, src = sopFactoryCapacity(0, 2, 700, 7)
	if src != "env_default" || units != 2*700*7 {
		t.Fatalf("env fallback units=%d src=%s", units, src)
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
