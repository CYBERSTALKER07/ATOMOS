package factory

import (
	"os"
	"strings"
)

const (
	FlagFactoryPlanning = "FACTORY_PLANNING_ENABLED"
	FlagFactoryBatcher  = "FACTORY_BATCHER_ENABLED"

	NetworkModeSpeed      = "SPEED"
	NetworkModeEconomy    = "ECONOMY"
	NetworkModeBalanced   = "BALANCED"
	NetworkModeLowCarbon  = "LOW_CARBON"
	NetworkModeManualOnly = "MANUAL_ONLY"

	TransferSourceThreshold = "SYSTEM_THRESHOLD"
	TransferSourcePredicted = "SYSTEM_PREDICTED"
	TransferSourceManual    = "MANUAL_EMERGENCY"
	TransferSourceLookAhead = "SYSTEM_THRESHOLD" // look-ahead still system-owned for kill switch
	TransferStateCreated    = "CREATED"
	TransferStateApproved   = "APPROVED"
	TransferStateLoading    = "LOADING"
	TransferStateCancelled  = "CANCELLED"

	DispatchAlgoPickN   = "pick_n_created_v1"
	DispatchAlgoBatcher = "ffd_nn_lifo_v1"
	OptimizerHeuristic  = "HEURISTIC"

	// DefaultDailyOutputCapacity matches SOP_FACTORY_DAILY_UNITS (700) so new
	// factories populate Factories.DailyOutputCapacity and S&OP can use
	// capacity_source=factories_column instead of env_default.
	DefaultDailyOutputCapacity int64 = 700
)

// PlanningEnabled is FACTORY_PLANNING_ENABLED (default false).
func PlanningEnabled() bool {
	return envFlagOn(FlagFactoryPlanning)
}

// BatcherEnabled is FACTORY_BATCHER_ENABLED (default false).
// POST /v1/factory/dispatch no longer uses this as the engine gate: when Spanner
// is present the warehouse-class solver (OR-Tools then H3 BinPack) always runs.
// The flag remains for planning cron / PackFFDNNLIFO callers only.
func BatcherEnabled() bool {
	return envFlagOn(FlagFactoryBatcher)
}

func envFlagOn(key string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func validNetworkMode(mode string) bool {
	switch strings.ToUpper(strings.TrimSpace(mode)) {
	case NetworkModeSpeed, NetworkModeEconomy, NetworkModeBalanced, NetworkModeLowCarbon, NetworkModeManualOnly:
		return true
	default:
		return false
	}
}

func normalizeNetworkMode(mode string) string {
	m := strings.ToUpper(strings.TrimSpace(mode))
	if m == "" {
		return NetworkModeBalanced
	}
	return m
}
