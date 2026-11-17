package migration

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrInvalidTransition indicates a disallowed project, bundle, or session state transition.
	ErrInvalidTransition = errors.New("invalid_migration_state_transition")
	// ErrDependencyNotSatisfied indicates an upstream pack must apply before a downstream pack can run.
	ErrDependencyNotSatisfied = errors.New("migration_pack_dependency_not_satisfied")
	// ErrConflictingInventoryMode indicates both stock and lots packs exist concurrently for the same warehouse/project.
	ErrConflictingInventoryMode = errors.New("conflicting_stock_and_lots_packs")
)

var projectTransitions = map[ProjectStatus]map[ProjectStatus]bool{
	ProjectStatusDraft: {
		ProjectStatusStaging: true,
		ProjectStatusFailed:  true,
	},
	ProjectStatusStaging: {
		ProjectStatusDryRunReady: true,
		ProjectStatusFailed:      true,
	},
	ProjectStatusDryRunReady: {
		ProjectStatusStaging:        true,
		ProjectStatusReadyToCutover: true,
		ProjectStatusFailed:         true,
	},
	ProjectStatusReadyToCutover: {
		ProjectStatusCutoverComplete: true,
		ProjectStatusStaging:         true,
		ProjectStatusFailed:          true,
	},
	ProjectStatusFailed: {
		ProjectStatusStaging: true,
	},
}

var bundleTransitions = map[BundleStatus]map[BundleStatus]bool{
	BundleStatusCreated: {
		BundleStatusUploading: true,
		BundleStatusFailed:    true,
	},
	BundleStatusUploading: {
		BundleStatusClassifying: true,
		BundleStatusFailed:      true,
	},
	BundleStatusClassifying: {
		BundleStatusSorted:      true,
		BundleStatusNeedsReview: true,
		BundleStatusFailed:      true,
	},
	BundleStatusNeedsReview: {
		BundleStatusSorted:      true,
		BundleStatusNeedsReview: true,
		BundleStatusClassifying: true,
		BundleStatusFailed:      true,
	},
	BundleStatusSorted: {
		BundleStatusSorted:      true,
		BundleStatusNeedsReview: true,
		BundleStatusClassifying: true,
	},
	BundleStatusFailed: {
		BundleStatusUploading:   true,
		BundleStatusClassifying: true,
	},
}

var sessionTransitions = map[SessionStatus]map[SessionStatus]bool{
	SessionStatusInitialized: {
		SessionStatusUploaded:    true,
		SessionStatusDiscovering: true,
		SessionStatusFailed:      true,
	},
	SessionStatusUploaded: {
		SessionStatusDiscovering: true,
		SessionStatusFailed:      true,
	},
	SessionStatusDiscovering: {
		SessionStatusDiscovered:      true,
		SessionStatusMappingRequired: true,
		SessionStatusMappingProposed: true,
		SessionStatusFailed:          true,
	},
	SessionStatusMappingRequired: {
		SessionStatusMappingProposed: true,
		SessionStatusDiscovered:      true,
		SessionStatusFailed:          true,
	},
	SessionStatusMappingProposed: {
		SessionStatusMappingRequired: true,
		SessionStatusDiscovered:      true,
		SessionStatusFailed:          true,
	},
	SessionStatusDiscovered: {
		SessionStatusValidating:      true,
		SessionStatusDryRunning:      true,
		SessionStatusDryRunReady:     true,
		SessionStatusMappingRequired: true,
		SessionStatusApproved:        true, // backward-compatible single-session flow
		SessionStatusFailed:          true,
	},
	SessionStatusValidating: {
		SessionStatusDryRunReady:     true,
		SessionStatusMappingRequired: true,
		SessionStatusFailed:          true,
	},
	SessionStatusDryRunReady: {
		SessionStatusDryRunning:      true,
		SessionStatusApproved:        true,
		SessionStatusMappingRequired: true,
		SessionStatusFailed:          true,
	},
	SessionStatusDryRunning: {
		SessionStatusDryRunReady: true,
		SessionStatusFailed:      true,
	},
	SessionStatusApproved: {
		SessionStatusSnapshotting: true,
		SessionStatusApplying:     true,
		SessionStatusDryRunReady:  true,
		SessionStatusFailed:       true,
	},
	SessionStatusSnapshotting: {
		SessionStatusApplying: true,
		SessionStatusFailed:   true,
	},
	SessionStatusApplying: {
		SessionStatusApplied: true,
		SessionStatusFailed:  true,
	},
	SessionStatusApplied: {
		SessionStatusRollingBack:     true,
		SessionStatusCutoverComplete: true,
	},
	SessionStatusRollingBack: {
		SessionStatusRolledBack: true,
		SessionStatusFailed:     true,
	},
	SessionStatusFailed: {
		SessionStatusApplying:    true, // chunk resume from cursor
		SessionStatusRollingBack: true, // revert partial committed chunks
		SessionStatusDiscovering: true, // retry discovery
	},
}

// ValidateProjectTransition verifies a MigrationProjects status transition.
func ValidateProjectTransition(from, to ProjectStatus) error {
	if from == to {
		return nil
	}
	allowed, ok := projectTransitions[from]
	if !ok || !allowed[to] {
		return fmt.Errorf("%w: project %s -> %s", ErrInvalidTransition, from, to)
	}
	return nil
}

// ValidateBundleTransition verifies a MigrationBundles status transition.
func ValidateBundleTransition(from, to BundleStatus) error {
	if from == to {
		return nil
	}
	allowed, ok := bundleTransitions[from]
	if !ok || !allowed[to] {
		return fmt.Errorf("%w: bundle %s -> %s", ErrInvalidTransition, from, to)
	}
	return nil
}

// ValidateSessionTransition verifies a SupplierImportSessions status transition.
func ValidateSessionTransition(from, to SessionStatus) error {
	nf := NormalizeSessionStatus(string(from))
	nt := NormalizeSessionStatus(string(to))
	if nf == nt {
		return nil
	}
	allowed, ok := sessionTransitions[nf]
	if !ok || !allowed[nt] {
		return fmt.Errorf("%w: session %s -> %s", ErrInvalidTransition, nf, nt)
	}
	return nil
}

// TopologicalPackOrder defines the canonical execution sequence across the 7 entity domains.
// Stock and Lots share the inventory tier (mutually exclusive for a given warehouse mode).
var TopologicalPackOrder = []Pack{
	PackWarehouses,
	PackProductsPrices,
	PackCustomers,
	PackStock,
	PackLots,
	PackHistoricalOrders,
	PackNasiyaAR,
	PackOpenOrders,
}

// DirectPackDependencies defines direct upstream prerequisites within the project DAG.
// Absent upstream packs are skipped unless structurally required; when present, an upstream pack
// must satisfy stage completion before its downstream dependent runs.
var DirectPackDependencies = map[Pack][]Pack{
	PackWarehouses:       nil,
	PackProductsPrices:   {PackWarehouses},
	PackCustomers:        {PackProductsPrices},
	PackStock:            {PackWarehouses, PackProductsPrices},
	PackLots:             {PackWarehouses, PackProductsPrices},
	PackHistoricalOrders: {PackWarehouses, PackProductsPrices, PackCustomers},
	PackNasiyaAR:         {PackCustomers, PackHistoricalOrders},
	PackOpenOrders:       {PackWarehouses, PackProductsPrices, PackCustomers, PackStock, PackLots, PackNasiyaAR},
}

// EvaluateProjectDAG checks pack mode exclusivity (stock vs lots) and computes runnable/blocked packs
// across all child sessions currently attached to a project.
func EvaluateProjectDAG(sessions []Session) (DAGStatus, error) {
	hasPack := make(map[Pack]bool)
	packStatuses := make(map[Pack][]SessionStatus)

	for _, s := range sessions {
		if s.Pack == "" {
			continue
		}
		st := NormalizeSessionStatus(string(s.Status))
		if st == SessionStatusRolledBack {
			continue
		}
		hasPack[s.Pack] = true
		packStatuses[s.Pack] = append(packStatuses[s.Pack], st)
	}

	if hasPack[PackStock] && hasPack[PackLots] {
		return DAGStatus{}, fmt.Errorf("%w: choose either 'stock' (lots off) or 'lots' (lots on)", ErrConflictingInventoryMode)
	}

	out := DAGStatus{
		Order:     append([]Pack(nil), TopologicalPackOrder...),
		BlockedBy: make(map[Pack][]Pack),
	}

	for _, p := range TopologicalPackOrder {
		if !hasPack[p] {
			continue
		}
		blockers := findBlockingParentsForApply(p, hasPack, packStatuses)
		if len(blockers) > 0 {
			out.BlockedBy[p] = blockers
			continue
		}
		if !allSessionsInPackTerminalApplied(packStatuses[p]) {
			out.RunnableNow = append(out.RunnableNow, p)
		}
	}

	return out, nil
}

// ValidatePackApplyAllowed verifies that all present upstream dependencies of targetPack
// are already APPLIED or CUTOVER_COMPLETE.
func ValidatePackApplyAllowed(targetPack Pack, sessions []Session) error {
	dag, err := EvaluateProjectDAG(sessions)
	if err != nil {
		return err
	}
	if blockers := dag.BlockedBy[targetPack]; len(blockers) > 0 {
		names := make([]string, 0, len(blockers))
		for _, b := range blockers {
			names = append(names, string(b))
		}
		return fmt.Errorf("%w: pack %s blocked by un-applied %s", ErrDependencyNotSatisfied, targetPack, strings.Join(names, ", "))
	}
	return nil
}

// ValidatePackDryRunAllowed verifies that no conflicting inventory mode exists and that every
// present upstream dependency is at least staged/discovered (or applied) before cross-pack overlay evaluation.
func ValidatePackDryRunAllowed(targetPack Pack, sessions []Session) error {
	hasPack := make(map[Pack]bool)
	packStatuses := make(map[Pack][]SessionStatus)
	for _, s := range sessions {
		if s.Pack == "" {
			continue
		}
		st := NormalizeSessionStatus(string(s.Status))
		if st == SessionStatusRolledBack {
			continue
		}
		hasPack[s.Pack] = true
		packStatuses[s.Pack] = append(packStatuses[s.Pack], st)
	}
	if hasPack[PackStock] && hasPack[PackLots] {
		return fmt.Errorf("%w: choose either 'stock' or 'lots'", ErrConflictingInventoryMode)
	}
	for _, parent := range DirectPackDependencies[targetPack] {
		if !hasPack[parent] {
			continue
		}
		if !allSessionsAtLeastStaged(packStatuses[parent]) {
			return fmt.Errorf("%w: upstream pack %s must finish discovery before dry-run of %s", ErrDependencyNotSatisfied, parent, targetPack)
		}
	}
	return nil
}

func findBlockingParentsForApply(target Pack, hasPack map[Pack]bool, statuses map[Pack][]SessionStatus) []Pack {
	var blockers []Pack
	for _, parent := range DirectPackDependencies[target] {
		if !hasPack[parent] {
			continue
		}
		if !allSessionsInPackTerminalApplied(statuses[parent]) {
			blockers = append(blockers, parent)
		}
	}
	return blockers
}

func allSessionsInPackTerminalApplied(list []SessionStatus) bool {
	if len(list) == 0 {
		return false
	}
	for _, st := range list {
		switch st {
		case SessionStatusApplied, SessionStatusCutoverComplete:
		default:
			return false
		}
	}
	return true
}

func allSessionsAtLeastStaged(list []SessionStatus) bool {
	if len(list) == 0 {
		return false
	}
	for _, st := range list {
		switch st {
		case SessionStatusMappingRequired, SessionStatusMappingProposed,
			SessionStatusDiscovered, SessionStatusValidating, SessionStatusDryRunReady,
			SessionStatusDryRunning, SessionStatusApproved, SessionStatusSnapshotting,
			SessionStatusApplying, SessionStatusApplied, SessionStatusCutoverComplete:
		default:
			return false
		}
	}
	return true
}
