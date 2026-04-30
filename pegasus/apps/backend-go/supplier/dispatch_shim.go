package supplier

import "backend-go/dispatch"

// ═══════════════════════════════════════════════════════════════════════════════
// DISPATCH SHIM — Type aliases and constant re-exports binding supplier/ to
// the canonical dispatch/ package so shared domain vocabulary cannot drift.
//
// The supplier package historically declared its own copies of dispatch types
// (GeoOrder, DispatchRoute, SplitOrder, OrderChunk, etc.). Any divergence
// between the two became a silent data-shape bug waiting to happen. This file
// collapses the duplicates down to a single source of truth in dispatch/ while
// preserving the supplier.X name surface that the rest of the codebase uses.
//
// Note: VehicleMatch stays local because its Driver field embeds the
// unexported supplier.availableDriver struct — the in-memory adapter used by
// fetchAvailableDrivers. Consolidating that adapter is tracked separately.
// ═══════════════════════════════════════════════════════════════════════════════

// ── Shared domain types ─────────────────────────────────────────────────────

type (
	GeoOrder         = dispatch.GeoOrder
	DispatchRoute    = dispatch.DispatchRoute
	SplitOrder       = dispatch.SplitOrder
	OrderChunk       = dispatch.OrderChunk
	ManifestChunk    = dispatch.ManifestChunk
	ManifestGroup    = dispatch.ManifestGroup
	AssignmentResult = dispatch.AssignmentResult
)

// ── Shared dispatch constants ───────────────────────────────────────────────

const (
	TetrisBuffer            = dispatch.TetrisBuffer
	MaxDetourRadius         = dispatch.MaxDetourRadius
	MaxWaypointsPerManifest = dispatch.MaxWaypointsPerManifest
	VUDivisor               = dispatch.VUDivisor
)

// ── Driver status vocabulary ────────────────────────────────────────────────

const (
	DriverStatusIdle        = dispatch.DriverStatusIdle
	DriverStatusAvailable   = dispatch.DriverStatusAvailable
	DriverStatusLoading     = dispatch.DriverStatusLoading
	DriverStatusReady       = dispatch.DriverStatusReady
	DriverStatusInTransit   = dispatch.DriverStatusInTransit
	DriverStatusReturning   = dispatch.DriverStatusReturning
	DriverStatusMaintenance = dispatch.DriverStatusMaintenance
)
