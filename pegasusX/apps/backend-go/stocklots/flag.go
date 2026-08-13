// Package stocklots implements §8.7 WMS lots, picks, counts, and cold-chain helpers.
package stocklots

import (
	"context"
	"sync"
	"sync/atomic"
)

var lotsEnabled atomic.Bool
var pickWavesEnabled atomic.Bool
var cycleCountsEnabled atomic.Bool
var pickSShapeEnabled atomic.Bool
var sealSoftWarnEnabled atomic.Bool
var coldChainEnabled atomic.Bool
var loadLedgerEnabled atomic.Bool
var laborCapacityEnforce atomic.Bool

// FlagEvaluator is the dual-control / tenant override surface (featureflags.Evaluate).
// Env process bits remain the default; ACTIVE tenant overrides enable seal-class pilots.
type FlagEvaluator interface {
	Evaluate(ctx context.Context, flagKey, tenantType, tenantID string) (bool, string, error)
}

var (
	flagEvalMu sync.RWMutex
	flagEval   FlagEvaluator
)

// SetFlagEvaluator wires runtime tenant overrides for WMS_* flags (G2.A).
func SetFlagEvaluator(e FlagEvaluator) {
	flagEvalMu.Lock()
	flagEval = e
	flagEvalMu.Unlock()
}

func effectiveFlag(ctx context.Context, flagKey string, processOn bool, tenantType, tenantID string) bool {
	if processOn {
		return true
	}
	flagEvalMu.RLock()
	e := flagEval
	flagEvalMu.RUnlock()
	if e == nil {
		return false
	}
	tenantID = trimSpace(tenantID)
	tenantType = trimSpace(tenantType)
	if tenantID == "" {
		return false
	}
	// Prefer warehouse scope, then supplier (callers pass primary first via Effective*).
	on, _, err := e.Evaluate(ctx, flagKey, tenantType, tenantID)
	return err == nil && on
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

// EffectivePickWaves reports pick/seal gate for warehouse or supplier seal-class.
func EffectivePickWaves(ctx context.Context, warehouseID, supplierID string) bool {
	if PickWavesEnabled() {
		return true
	}
	if warehouseID != "" && effectiveFlag(ctx, "WMS_PICK_WAVES_ENABLED", false, "WAREHOUSE", warehouseID) {
		return true
	}
	return effectiveFlag(ctx, "WMS_PICK_WAVES_ENABLED", false, "SUPPLIER", supplierID)
}

// EffectiveLots reports lot/FEFO for seal-class tenants.
func EffectiveLots(ctx context.Context, warehouseID, supplierID string) bool {
	if LotsEnabled() {
		return true
	}
	if warehouseID != "" && effectiveFlag(ctx, "WMS_LOTS_ENABLED", false, "WAREHOUSE", warehouseID) {
		return true
	}
	return effectiveFlag(ctx, "WMS_LOTS_ENABLED", false, "SUPPLIER", supplierID)
}

// EffectiveCycleCounts reports cycle-count APIs for seal-class tenants.
func EffectiveCycleCounts(ctx context.Context, warehouseID, supplierID string) bool {
	if CycleCountsEnabled() {
		return true
	}
	if warehouseID != "" && effectiveFlag(ctx, "WMS_CYCLE_COUNTS_ENABLED", false, "WAREHOUSE", warehouseID) {
		return true
	}
	return effectiveFlag(ctx, "WMS_CYCLE_COUNTS_ENABLED", false, "SUPPLIER", supplierID)
}

// EffectiveColdChain reports cold-chain for seal-class / chilled tenants.
func EffectiveColdChain(ctx context.Context, warehouseID, supplierID string) bool {
	if ColdChainEnabled() {
		return true
	}
	if warehouseID != "" && effectiveFlag(ctx, "WMS_COLD_CHAIN_ENABLED", false, "WAREHOUSE", warehouseID) {
		return true
	}
	return effectiveFlag(ctx, "WMS_COLD_CHAIN_ENABLED", false, "SUPPLIER", supplierID)
}

// EffectiveLoadLedger reports durable payload load ledger (G2.B).
func EffectiveLoadLedger(ctx context.Context, warehouseID, supplierID string) bool {
	if LoadLedgerEnabled() {
		return true
	}
	if warehouseID != "" && effectiveFlag(ctx, "PAYLOAD_LOAD_LEDGER_ENABLED", false, "WAREHOUSE", warehouseID) {
		return true
	}
	return effectiveFlag(ctx, "PAYLOAD_LOAD_LEDGER_ENABLED", false, "SUPPLIER", supplierID)
}

// EffectiveLaborCapacityEnforce reports hard labor refuse on dispatch (G2.C).
func EffectiveLaborCapacityEnforce(ctx context.Context, warehouseID, supplierID string) bool {
	if LaborCapacityEnforce() {
		return true
	}
	if warehouseID != "" && effectiveFlag(ctx, "LABOR_CAPACITY_ENFORCE", false, "WAREHOUSE", warehouseID) {
		return true
	}
	return effectiveFlag(ctx, "LABOR_CAPACITY_ENFORCE", false, "SUPPLIER", supplierID)
}

// SetLotsEnabled toggles WMS lot/FEFO paths (WMS_LOTS_ENABLED).
func SetLotsEnabled(v bool) { lotsEnabled.Store(v) }

// LotsEnabled reports whether lot-level inventory is active (process/env).
func LotsEnabled() bool { return lotsEnabled.Load() }

// SetPickWavesEnabled toggles pick-wave APIs + seal gate (WMS_PICK_WAVES_ENABLED).
func SetPickWavesEnabled(v bool) { pickWavesEnabled.Store(v) }

// PickWavesEnabled reports whether pick waves / seal gate are active (process/env).
func PickWavesEnabled() bool { return pickWavesEnabled.Load() }

// SetCycleCountsEnabled toggles cycle-count APIs (WMS_CYCLE_COUNTS_ENABLED).
func SetCycleCountsEnabled(v bool) { cycleCountsEnabled.Store(v) }

// CycleCountsEnabled reports whether cycle-count / adjustment APIs are active.
func CycleCountsEnabled() bool { return cycleCountsEnabled.Load() }

// SetPickSShapeEnabled toggles zone serpentine + LIFO task ordering (WMS_PICK_SSHAPE_ENABLED).
func SetPickSShapeEnabled(v bool) { pickSShapeEnabled.Store(v) }

// PickSShapeEnabled reports S-shape/LIFO pick ordering.
func PickSShapeEnabled() bool { return pickSShapeEnabled.Load() }

// SetSealSoftWarnEnabled allows seal when wave incomplete with a warning (WMS_SEAL_SOFT_WARN).
func SetSealSoftWarnEnabled(v bool) { sealSoftWarnEnabled.Store(v) }

// SealSoftWarnEnabled reports soft-warn seal mode.
func SealSoftWarnEnabled() bool { return sealSoftWarnEnabled.Load() }

// SetColdChainEnabled toggles temperature ingest + quarantine (WMS_COLD_CHAIN_ENABLED).
func SetColdChainEnabled(v bool) { coldChainEnabled.Store(v) }

// ColdChainEnabled reports cold-chain paths.
func ColdChainEnabled() bool { return coldChainEnabled.Load() }

// SetLoadLedgerEnabled toggles durable load ledger seal gate (PAYLOAD_LOAD_LEDGER_ENABLED).
func SetLoadLedgerEnabled(v bool) { loadLedgerEnabled.Store(v) }

// LoadLedgerEnabled reports process/env load ledger.
func LoadLedgerEnabled() bool { return loadLedgerEnabled.Load() }

// SetLaborCapacityEnforce toggles hard labor refuse on dispatch (LABOR_CAPACITY_ENFORCE).
func SetLaborCapacityEnforce(v bool) { laborCapacityEnforce.Store(v) }

// LaborCapacityEnforce reports process/env labor hard gate.
func LaborCapacityEnforce() bool { return laborCapacityEnforce.Load() }
