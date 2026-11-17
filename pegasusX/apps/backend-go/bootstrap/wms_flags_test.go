package bootstrap

import (
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
)

func TestConfigureWMSFlagsFromEnv(t *testing.T) {
	t.Cleanup(func() {
		stocklots.SetLotsEnabled(false)
		stocklots.SetPickWavesEnabled(false)
		stocklots.SetCycleCountsEnabled(false)
		stocklots.SetPickSShapeEnabled(false)
		stocklots.SetSealSoftWarnEnabled(false)
		stocklots.SetColdChainEnabled(false)
		stocklots.SetLoadLedgerEnabled(false)
		stocklots.SetLaborCapacityEnforce(false)
	})

	t.Setenv("WMS_LOTS_ENABLED", "true")
	t.Setenv("WMS_PICK_WAVES_ENABLED", "TRUE")
	t.Setenv("WMS_CYCLE_COUNTS_ENABLED", " true ")
	t.Setenv("WMS_PICK_SSHAPE_ENABLED", "true")
	t.Setenv("WMS_SEAL_SOFT_WARN", "true")
	t.Setenv("WMS_COLD_CHAIN_ENABLED", "true")
	t.Setenv("PAYLOAD_LOAD_LEDGER_ENABLED", "true")
	t.Setenv("LABOR_CAPACITY_ENFORCE", "true")

	configureWMSFlagsFromEnv()

	if !stocklots.LotsEnabled() {
		t.Fatal("expected stocklots.LotsEnabled() to be true when WMS_LOTS_ENABLED=true")
	}
	if !stocklots.PickWavesEnabled() || !stocklots.CycleCountsEnabled() ||
		!stocklots.PickSShapeEnabled() || !stocklots.SealSoftWarnEnabled() ||
		!stocklots.ColdChainEnabled() || !stocklots.LoadLedgerEnabled() ||
		!stocklots.LaborCapacityEnforce() {
		t.Fatal("expected all WMS process flags to reflect true env values")
	}

	t.Setenv("WMS_LOTS_ENABLED", "false")
	configureWMSFlagsFromEnv()
	if stocklots.LotsEnabled() {
		t.Fatal("expected stocklots.LotsEnabled() to clear to false when WMS_LOTS_ENABLED=false")
	}
}
