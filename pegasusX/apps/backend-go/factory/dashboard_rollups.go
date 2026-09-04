package factory

import (
	"context"
	"strings"
	"time"
)

// Live factory-transfer dictionary. Spec lists CREATED/APPROVED/PENDING/
// ASSIGNED/LOADING/DISPATCHED/CANCELLED; the extra keys are states the
// service actually writes (rebalance + warehouse handoff). Do not add
// last-mile retailer order statuses here.
var factoryTransferStates = []string{
	"CREATED",
	"APPROVED",
	"PENDING",
	"ASSIGNED",
	"LOADING",
	"DISPATCHED",
	"IN_TRANSIT",
	"ARRIVED",
	"RECEIVED",
	"CANCELLED",
	"REASSIGNED",
}

var factoryManifestStates = []string{
	manifestStateDraft,
	manifestStateLoading,
	manifestStateSealed,
	manifestStateDispatched,
	manifestStateCompleted,
	manifestStateCancelled,
}

var factoryVehicleStates = []string{"READY", "AVAILABLE", "UNAVAILABLE"}

var factoryDriverDuty = []string{"ON_SHIFT", "OFF_SHIFT"}

var factorySLAStatuses = []string{
	SLAStatusOnTime,
	SLAStatusAtRisk,
	SLAStatusBreached,
	SLAStatusMet,
	SLAStatusNA,
}

var factoryQCResults = []string{"PASS", "FAIL", "MISSING"}

// FactoryDashboardQuery injects Spanner-backed rows for tests and the live
// dashboard path. Returning a snapshot labels source=spanner.
type FactoryDashboardQuery func(ctx context.Context, factoryID, supplierID string) (factoryDashboardSnapshot, error)

type factoryDashboardSnapshot struct {
	Transfers   []TransferRow
	Manifests   []ManifestRow
	Vehicles    []FleetVehicle
	Drivers     []FleetDriver
	Exceptions  []ManifestException
	Staff       []StaffRow
	Requests    []SupplyRequest
	QC          map[string]string
	QCAvailable bool
}

func emptyCountMap(keys []string) map[string]int {
	out := make(map[string]int, len(keys))
	for _, key := range keys {
		out[key] = 0
	}
	return out
}

func canonicalizeFactoryTransfer(state string) string {
	return strings.ToUpper(strings.TrimSpace(state))
}

func canonicalizeFactoryManifest(state string) string {
	return strings.ToUpper(strings.TrimSpace(state))
}

func canonicalizeFactoryVehicle(state string) string {
	key := strings.ToUpper(strings.TrimSpace(state))
	if key == "READY" || key == "AVAILABLE" {
		return key
	}
	return "UNAVAILABLE"
}

func countFactoryTransfers(rows []TransferRow) (counts map[string]int, pending, loading, dispatched int64) {
	counts = emptyCountMap(factoryTransferStates)
	for _, row := range rows {
		key := canonicalizeFactoryTransfer(row.State)
		if _, ok := counts[key]; ok {
			counts[key]++
		}
		switch key {
		case "CREATED", "APPROVED", "PENDING":
			pending++
		case "LOADING", "ASSIGNED":
			loading++
		case "DISPATCHED":
			dispatched++
		}
	}
	return counts, pending, loading, dispatched
}

func countFactoryManifests(rows []ManifestRow) (counts map[string]int, active, dispatched int64) {
	counts = emptyCountMap(factoryManifestStates)
	for _, row := range rows {
		key := canonicalizeFactoryManifest(row.State)
		if _, ok := counts[key]; ok {
			counts[key]++
		}
		switch key {
		case manifestStateDraft, manifestStateLoading:
			active++
		case manifestStateDispatched:
			dispatched++
		}
	}
	return counts, active, dispatched
}

func countFactoryVehicles(rows []FleetVehicle) (counts map[string]int, available int64) {
	counts = emptyCountMap(factoryVehicleStates)
	for _, row := range rows {
		key := canonicalizeFactoryVehicle(row.State)
		if _, ok := counts[key]; ok {
			counts[key]++
		}
		if key == "READY" || key == "AVAILABLE" {
			available++
		}
	}
	return counts, available
}

func countFactoryDriverDuty(rows []FleetDriver) (counts map[string]int, onShift int64) {
	counts = emptyCountMap(factoryDriverDuty)
	for _, row := range rows {
		if row.OnShift {
			counts["ON_SHIFT"]++
			onShift++
			continue
		}
		counts["OFF_SHIFT"]++
	}
	return counts, onShift
}

func countFactorySLA(rows []SupplyRequest, now time.Time) map[string]int {
	counts := emptyCountMap(factorySLAStatuses)
	for _, row := range rows {
		createdAt, _ := time.Parse(time.RFC3339Nano, row.CreatedAt)
		if createdAt.IsZero() {
			createdAt, _ = time.Parse(time.RFC3339, row.CreatedAt)
		}
		var due time.Time
		if strings.TrimSpace(row.RequestedDeliveryDate) != "" {
			due, _ = time.Parse(time.RFC3339Nano, row.RequestedDeliveryDate)
			if due.IsZero() {
				due, _ = time.Parse(time.RFC3339, row.RequestedDeliveryDate)
			}
		}
		eval := EvaluateSLA(row.Status, createdAt, due, now)
		if _, ok := counts[eval.Status]; ok {
			counts[eval.Status]++
		}
	}
	return counts
}

func countFactoryQC(requests []SupplyRequest, results map[string]string, available bool) (map[string]int, bool) {
	counts := emptyCountMap(factoryQCResults)
	if !available {
		return counts, false
	}
	for _, row := range requests {
		key := strings.ToUpper(strings.TrimSpace(results[row.RequestID]))
		switch key {
		case "PASS", "FAIL":
			counts[key]++
		default:
			counts["MISSING"]++
		}
	}
	return counts, true
}

func exceptionSummaries(rows []ManifestException) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for i := range rows {
		if i >= 8 {
			break
		}
		out = append(out, map[string]any{
			"exception_id": rows[i].ExceptionID,
			"manifest_id":  rows[i].ManifestID,
			"transfer_id":  rows[i].TransferID,
			"reason":       rows[i].Reason,
			"escalated":    rows[i].Escalated,
		})
	}
	return out
}

func snapshotFromMemoryLocked(s *Service) factoryDashboardSnapshot {
	return factoryDashboardSnapshot{
		Transfers:   append([]TransferRow(nil), s.transfers...),
		Manifests:   append([]ManifestRow(nil), s.manifests...),
		Vehicles:    append([]FleetVehicle(nil), s.fleetVehicles...),
		Drivers:     append([]FleetDriver(nil), s.fleetDrivers...),
		Exceptions:  append([]ManifestException(nil), s.manifestExceptions...),
		Staff:       append([]StaffRow(nil), s.staff...),
		Requests:    append([]SupplyRequest(nil), s.supplyRequests...),
		QCAvailable: false,
	}
}

func buildFactoryDashboard(snap factoryDashboardSnapshot, source, supplierID, factoryID string, now time.Time) map[string]any {
	transfers, pending, loading, dispatchedTransfers := countFactoryTransfers(snap.Transfers)
	manifests, activeManifests, dispatchedManifests := countFactoryManifests(snap.Manifests)
	vehicles, vehiclesAvailable := countFactoryVehicles(snap.Vehicles)
	duty, onShift := countFactoryDriverDuty(snap.Drivers)
	qc, qcAvailable := countFactoryQC(snap.Requests, snap.QC, snap.QCAvailable)
	bayTransfers := int64(transfers["LOADING"])
	bayManifests := int64(manifests[manifestStateLoading])

	return map[string]any{
		"source":                source,
		"plane":                 "factory_trucks",
		"pending_transfers":     pending,
		"loading_transfers":     loading,
		"active_manifests":      activeManifests,
		"dispatched_today":      dispatchedManifests,
		"dispatched_transfers":  dispatchedTransfers,
		"vehicles_total":        int64(len(snap.Vehicles)),
		"vehicles_available":    vehiclesAvailable,
		"staff_on_shift":        onShift,
		"staff_total":           int64(len(snap.Staff)),
		"critical_insights":     int64(len(snap.Exceptions)),
		"transfers_by_state":    transfers,
		"manifests_by_state":    manifests,
		"vehicles_by_state":     vehicles,
		"driver_duty":           duty,
		"sla_by_status":         countFactorySLA(snap.Requests, now),
		"qc_by_result":          qc,
		"qc_available":          qcAvailable,
		"bay_loading_transfers": bayTransfers,
		"bay_loading_manifests": bayManifests,
		"exceptions":            exceptionSummaries(snap.Exceptions),
		"supplier_id":           supplierID,
		"factory_id":            factoryID,
		"updated_at":            now.Format(time.RFC3339Nano),
	}
}

func (s *Service) loadDashboardSnapshot(ctx context.Context) (factoryDashboardSnapshot, string, error) {
	fid := strings.TrimSpace(s.resolveFactoryNode(ctx))
	sid := strings.TrimSpace(s.resolveSupplierScope(ctx))

	if s.dashboardQuery != nil {
		snap, err := s.dashboardQuery(ctx, fid, sid)
		if err != nil {
			return factoryDashboardSnapshot{}, "", err
		}
		return snap, "spanner", nil
	}

	if s.spannerClient != nil {
		snap, err := s.loadDashboardFromSpanner(ctx)
		if err == nil {
			return snap, "spanner", nil
		}
		s.log.WarnContext(ctx, "factory dashboard spanner failed", "err", err)
		if !s.portalSeedEnabled() {
			return factoryDashboardSnapshot{}, "empty", nil
		}
	}

	if s.portalSeedEnabled() {
		s.mu.Lock()
		s.ensureDemoDataLocked()
		snap := snapshotFromMemoryLocked(s)
		s.mu.Unlock()
		return snap, "memory", nil
	}

	return factoryDashboardSnapshot{}, "empty", nil
}

func (s *Service) loadDashboardFromSpanner(ctx context.Context) (factoryDashboardSnapshot, error) {
	transfers, err := s.loadFactoryTransfersFromSpanner(ctx)
	if err != nil {
		return factoryDashboardSnapshot{}, err
	}
	manifests, err := s.loadFactoryManifestsFromSpanner(ctx)
	if err != nil {
		return factoryDashboardSnapshot{}, err
	}
	snap := factoryDashboardSnapshot{
		Transfers: transfers,
		Manifests: manifests,
	}
	if backend := s.exceptionBackend(); backend != nil {
		if rows, listErr := backend.List(ctx, s.resolveSupplierScope(ctx), s.resolveFactoryNode(ctx)); listErr == nil {
			snap.Exceptions = rows
		}
	}
	if rows, staffErr := s.loadFactoryStaffFromSpanner(ctx); staffErr == nil {
		snap.Staff = rows
	}
	if rows, reqErr := s.listSupplyRequestsFromSpanner(ctx); reqErr == nil {
		snap.Requests = rows
	}
	return snap, nil
}
