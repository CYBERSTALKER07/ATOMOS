// Package stocklots implements §8.7 WMS lots, picks, counts, and cold-chain helpers.
package stocklots

import "sync/atomic"

var lotsEnabled atomic.Bool
var pickWavesEnabled atomic.Bool
var cycleCountsEnabled atomic.Bool
var pickSShapeEnabled atomic.Bool
var sealSoftWarnEnabled atomic.Bool
var coldChainEnabled atomic.Bool

// SetLotsEnabled toggles WMS lot/FEFO paths (WMS_LOTS_ENABLED).
func SetLotsEnabled(v bool) { lotsEnabled.Store(v) }

// LotsEnabled reports whether lot-level inventory is active.
func LotsEnabled() bool { return lotsEnabled.Load() }

// SetPickWavesEnabled toggles pick-wave APIs + seal gate (WMS_PICK_WAVES_ENABLED).
func SetPickWavesEnabled(v bool) { pickWavesEnabled.Store(v) }

// PickWavesEnabled reports whether pick waves / seal gate are active.
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
