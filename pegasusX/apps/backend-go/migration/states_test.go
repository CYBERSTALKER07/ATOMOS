package migration

import (
	"errors"
	"testing"
	"time"
)

func TestStateTransitions(t *testing.T) {
	if err := ValidateProjectTransition(ProjectStatusDraft, ProjectStatusStaging); err != nil {
		t.Fatalf("DRAFT -> STAGING failed: %v", err)
	}
	if err := ValidateProjectTransition(ProjectStatusDraft, ProjectStatusCutoverComplete); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("DRAFT -> CUTOVER_COMPLETE should fail, got %v", err)
	}

	if err := ValidateBundleTransition(BundleStatusUploading, BundleStatusClassifying); err != nil {
		t.Fatalf("UPLOADING -> CLASSIFYING failed: %v", err)
	}
	if err := ValidateBundleTransition(BundleStatusCreated, BundleStatusSorted); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("CREATED -> SORTED should fail, got %v", err)
	}

	for _, pair := range [][2]SessionStatus{
		{SessionStatusInitialized, SessionStatusUploaded},
		{SessionStatusUploaded, SessionStatusDiscovering},
		{SessionStatusDiscovering, SessionStatusMappingProposed},
		{SessionStatusMappingProposed, SessionStatusDiscovered},
		{SessionStatusDiscovered, SessionStatusDryRunReady},
		{SessionStatusDryRunReady, SessionStatusApproved},
		{SessionStatusApproved, SessionStatusSnapshotting},
		{SessionStatusSnapshotting, SessionStatusApplying},
		{SessionStatusApplying, SessionStatusApplied},
		{SessionStatusApplied, SessionStatusRollingBack},
		{SessionStatusRollingBack, SessionStatusRolledBack},
		{SessionStatusApplied, SessionStatusCutoverComplete},
		{SessionStatusFailed, SessionStatusApplying},
		{"initialized", "uploaded"},
	} {
		if err := ValidateSessionTransition(pair[0], pair[1]); err != nil {
			t.Fatalf("expected valid transition %s -> %s, got %v", pair[0], pair[1], err)
		}
	}

	if err := ValidateSessionTransition(SessionStatusUploaded, SessionStatusApproved); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("UPLOADED -> APPROVED must fail, got %v", err)
	}
}

func TestEvaluateProjectDAG_DependenciesAndExclusivity(t *testing.T) {
	sessions := []Session{
		{SessionID: "s1", Pack: PackWarehouses, Status: SessionStatusApplied},
		{SessionID: "s2", Pack: PackProductsPrices, Status: SessionStatusDryRunReady},
		{SessionID: "s3", Pack: PackStock, Status: SessionStatusDiscovered},
	}

	dag, err := EvaluateProjectDAG(sessions)
	if err != nil {
		t.Fatalf("EvaluateProjectDAG: %v", err)
	}
	if len(dag.BlockedBy[PackStock]) == 0 || dag.BlockedBy[PackStock][0] != PackProductsPrices {
		t.Fatalf("expected stock blocked by products_prices, got %+v", dag.BlockedBy)
	}
	if err := ValidatePackApplyAllowed(PackStock, sessions); !errors.Is(err, ErrDependencyNotSatisfied) {
		t.Fatalf("expected ErrDependencyNotSatisfied for stock before products_prices applied, got %v", err)
	}
	// Dry-run is allowed across staged virtual packs!
	if err := ValidatePackDryRunAllowed(PackStock, sessions); err != nil {
		t.Fatalf("expected dry-run overlay to be permitted when upstream is DRY_RUN_READY, got %v", err)
	}

	// Upstream sessions that finished staging (MAPPING_PROPOSED / MAPPING_REQUIRED) also unblock cross-pack dry-run,
	// while still strictly blocking apply until upstream reaches APPLIED/CUTOVER_COMPLETE.
	stagedUpstream := []Session{
		{SessionID: "s-cust", Pack: PackCustomers, Status: SessionStatusMappingProposed},
		{SessionID: "s-ord", Pack: PackHistoricalOrders, Status: SessionStatusMappingRequired},
		{SessionID: "s-ar", Pack: PackNasiyaAR, Status: SessionStatusDiscovered},
	}
	if err := ValidatePackDryRunAllowed(PackNasiyaAR, stagedUpstream); err != nil {
		t.Fatalf("expected dry-run overlay permitted when upstream is MAPPING_PROPOSED/MAPPING_REQUIRED, got %v", err)
	}
	if err := ValidatePackApplyAllowed(PackNasiyaAR, stagedUpstream); !errors.Is(err, ErrDependencyNotSatisfied) {
		t.Fatalf("expected ErrDependencyNotSatisfied for apply before upstream applied, got %v", err)
	}
	// Pre-staging sessions (INITIALIZED/UPLOADED/DISCOVERING) still block downstream dry-run.
	preStagedUpstream := []Session{
		{SessionID: "s-cust", Pack: PackCustomers, Status: SessionStatusDiscovering},
		{SessionID: "s-ar", Pack: PackNasiyaAR, Status: SessionStatusDiscovered},
	}
	if err := ValidatePackDryRunAllowed(PackNasiyaAR, preStagedUpstream); !errors.Is(err, ErrDependencyNotSatisfied) {
		t.Fatalf("expected ErrDependencyNotSatisfied while upstream is DISCOVERING, got %v", err)
	}

	// Conflicting stock + lots in same project must fail.
	sessionsConflict := append(sessions, Session{SessionID: "s4", Pack: PackLots, Status: SessionStatusDiscovered})
	if _, err := EvaluateProjectDAG(sessionsConflict); !errors.Is(err, ErrConflictingInventoryMode) {
		t.Fatalf("expected ErrConflictingInventoryMode, got %v", err)
	}
}

func TestWarehouseLockSemantics(t *testing.T) {
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	active := &WarehouseLock{
		SupplierID:  "sup-1",
		WarehouseID: "wh-1",
		SessionID:   "sess-1",
		ExpiresAt:   now.Add(5 * time.Minute),
	}
	if canAcquireLock(active, "sess-2", now) {
		t.Fatal("other session must not steal active lease")
	}
	if !canAcquireLock(active, "sess-1", now) {
		t.Fatal("owning session must be re-entrant")
	}
	if !canAcquireLock(active, "sess-2", now.Add(6*time.Minute)) {
		t.Fatal("expired lock must be evictable")
	}
}
