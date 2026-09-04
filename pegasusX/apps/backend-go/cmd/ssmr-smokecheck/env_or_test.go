package main

import (
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/seed"
)

func TestSmokeSupplierIDFallsBackWhenSandboxIdentityUnset(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "sandbox")
	t.Setenv("SSMR_SMOKE_SUPPLIER_ID", "")
	t.Setenv("SANDBOX_SMOKE_SUPPLIER_ID", "")
	if got := smokeSupplierID(); got != seed.DefaultSupplierID {
		t.Fatalf("smokeSupplierID()=%q want seed %q", got, seed.DefaultSupplierID)
	}
}

func TestSmokeSupplierIDPrefersSandboxEnv(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "sandbox")
	t.Setenv("SANDBOX_SMOKE_SUPPLIER_ID", "sup-from-env")
	t.Setenv("SSMR_SMOKE_SUPPLIER_ID", "")
	if got := smokeSupplierID(); got != "sup-from-env" {
		t.Fatalf("smokeSupplierID()=%q", got)
	}
}

func TestEnvOrSandboxPasswordStaysFailClosed(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "sandbox")
	t.Setenv("SSMR_SMOKE_SUPPLIER_PASSWORD", "")
	t.Setenv("SANDBOX_SMOKE_SUPPLIER_PASSWORD", "")
	if got := envOr("SSMR_SMOKE_SUPPLIER_PASSWORD", "SmokeTest!234"); got != "" {
		t.Fatalf("sandbox password must not invent fallback, got %q", got)
	}
}
