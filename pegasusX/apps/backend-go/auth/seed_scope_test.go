package auth

import (
	"os"
	"strings"
	"testing"
)

func TestEnsureDemoScopeLinksSeedsFactoryDispatchFleet(t *testing.T) {
	body, err := os.ReadFile("seed_scope.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(body)
	if !strings.Contains(src, `"VehicleId":    "veh_factory_1"`) {
		t.Fatal("factory AUTO dispatch requires Vehicles.veh_factory_1 assigned to drv_factory_1")
	}
	if !strings.Contains(src, `"DriverId":     "drv_factory_1"`) {
		t.Fatal("missing drv_factory_1 seed")
	}
}

func TestEnsureDemoScopeLinks_SandboxRequiresPassword(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "sandbox")
	t.Setenv("SANDBOX_SMOKE_SUPPLIER_PASSWORD", "")
	t.Setenv("SSMR_SMOKE_SUPPLIER_PASSWORD", "")
	t.Setenv("SANDBOX_SMOKE_WAREHOUSE_ID", "wh-test")
	t.Setenv("SSMR_SMOKE_WAREHOUSE_ID", "")
	t.Setenv("WAREHOUSE_DEMO_ID", "")
	err := EnsureDemoScopeLinks(t.Context(), nil, "sup-1")
	if err != nil {
		t.Fatalf("nil client must no-op, got %v", err)
	}
}

func TestDemoWarehouseID_SandboxNoSilentDefault(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "sandbox")
	t.Setenv("SANDBOX_SMOKE_WAREHOUSE_ID", "")
	t.Setenv("SSMR_SMOKE_WAREHOUSE_ID", "")
	t.Setenv("WAREHOUSE_DEMO_ID", "")
	if got := demoWarehouseID(); got != "" {
		t.Fatalf("sandbox must not invent warehouse id, got %q", got)
	}
	t.Setenv("SANDBOX_SMOKE_WAREHOUSE_ID", "wh-from-env")
	if got := demoWarehouseID(); got != "wh-from-env" {
		t.Fatalf("got %q", got)
	}
}

func TestEnsureDemoScopeLinks_ProductionNoOp(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	if err := EnsureDemoScopeLinks(t.Context(), nil, "sup-1"); err != nil {
		t.Fatalf("production nil client: %v", err)
	}
}

func TestSeedSupplierCurrency_EmptyUsesPack(t *testing.T) {
	t.Setenv("SEED_SUPPLIER_CURRENCY", "")
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	if got := seedSupplierCurrency(); got != "UZS" {
		t.Fatalf("got %q want UZS from pack", got)
	}
}

func TestSeedSupplierCurrency_EnvWins(t *testing.T) {
	t.Setenv("SEED_SUPPLIER_CURRENCY", "eur")
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	if got := seedSupplierCurrency(); got != "EUR" {
		t.Fatalf("got %q want EUR from env", got)
	}
}

func TestSeedSupplierCurrency_PlannedDoesNotInvent(t *testing.T) {
	t.Setenv("SEED_SUPPLIER_CURRENCY", "")
	t.Setenv("DEFAULT_MARKET_CODE", "EU")
	if got := seedSupplierCurrency(); got != "" {
		t.Fatalf("planned pack must not invent UZS, got %q", got)
	}
}
