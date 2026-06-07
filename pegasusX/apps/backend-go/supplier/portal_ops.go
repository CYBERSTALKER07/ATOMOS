package supplier

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch"
	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch/plan"
	"github.com/pegasusx/pegasusx/apps/backend-go/manifest"
)

// SupplierManifestRow is a supplier-scoped manifest queue projection.
type SupplierManifestRow struct {
	ManifestID    string  `json:"manifest_id"`
	Status        string  `json:"status"`
	State         string  `json:"state"`
	OrdersCount   int     `json:"orders_count"`
	DriverID      string  `json:"driver_id,omitempty"`
	DriverName    string  `json:"driver_name"`
	VehicleID     string  `json:"vehicle_id,omitempty"`
	VehiclePlate  string  `json:"vehicle_plate,omitempty"`
	TruckID       string  `json:"truck_id,omitempty"`
	TotalVu       int64   `json:"total_vu"`
	TotalVolumeVU float64 `json:"total_volume_vu"`
	MaxVolumeVU   float64 `json:"max_volume_vu"`
	StopCount     int     `json:"stop_count"`
	UpdatedAt     string  `json:"updated_at"`
}

// SupplierSupplyLaneRow is a warehouse-centric lane summary for the supplier portal.
type SupplierSupplyLaneRow struct {
	LaneID      string  `json:"lane_id"`
	Name        string  `json:"name"`
	WarehouseID string  `json:"warehouse_id"`
	H3Cells     int     `json:"h3_cells"`
	Drivers     int     `json:"drivers"`
	OrdersToday int     `json:"orders_today"`
	Capacity    int     `json:"capacity"`
	Utilization float64 `json:"utilization_pct"`
}

// SupplierActivityEvent is a supplier-portal activity feed row derived from order transitions.
type SupplierActivityEvent struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Timestamp   string `json:"timestamp"`
	Description string `json:"description"`
	OrderID     string `json:"order_id,omitempty"`
	ManifestID  string `json:"manifest_id,omitempty"`
}

// SupplierDispatchPreview is the supplier-scoped dispatch planning snapshot.
type SupplierDispatchPreview struct {
	UndispatchedOrders     []map[string]any `json:"undispatched_orders"`
	AvailableDrivers       []map[string]any `json:"available_drivers"`
	UnavailableDrivers     []map[string]any `json:"unavailable_drivers"`
	PendingCount           int              `json:"pending_count"`
	AvailableCount         int              `json:"available_driver_count"`
	WindowConstrainedCount int              `json:"window_constrained_count"`
	ProposedRoutes         []map[string]any `json:"proposed_routes,omitempty"`
	OptimizerSource        string           `json:"optimizer_source,omitempty"`
	OptimizerWarnings      []string         `json:"optimizer_warnings,omitempty"`
}

// SupplierExceptionRow is an operational exception surfaced to the supplier queue.
type SupplierExceptionRow struct {
	OrderID    string `json:"order_id"`
	Kind       string `json:"kind"`
	Status     string `json:"status"`
	RetailerID string `json:"retailer_id,omitempty"`
	Note       string `json:"note,omitempty"`
	ManifestID string `json:"manifest_id,omitempty"`
	UpdatedAt  string `json:"updated_at"`
}

type supplierDashboardDetail struct {
	supplierDashboardResponse
	OrdersByStatus         map[string]int          `json:"orders_by_status"`
	TodayRevenueMinor      int64                   `json:"today_revenue_minor"`
	Currency               string                  `json:"currency"`
	ActiveDrivers          int                     `json:"active_drivers"`
	TotalDrivers           int                     `json:"total_drivers"`
	RetailersOrderedToday  int                     `json:"retailers_ordered_today"`
	TotalRetailers         int                     `json:"total_retailers"`
	DeliveryCompletionRate float64                 `json:"delivery_completion_rate_pct"`
	FleetVuUsed            int64                   `json:"fleet_vu_used"`
	FleetVuTotal           int64                   `json:"fleet_vu_total"`
	RecentManifests        []SupplierManifestRow   `json:"recent_manifests"`
	ActivityEvents         []SupplierActivityEvent `json:"activity_events"`
}

// HandleDispatchPreview serves GET/POST /v1/supplier/dispatch/preview.
func (s *Service) HandleDispatchPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	sid := s.scopedSupplierID(r)
	warehouseFilter := auth.EffectiveWarehouseID(r.Context())
	if warehouseFilter == "" {
		warehouseFilter = strings.TrimSpace(r.URL.Query().Get("warehouse_id"))
	}
	preview, err := s.buildSupplierDispatchPreview(r.Context(), sid, warehouseFilter)
	if err != nil {
		s.log.WarnContext(r.Context(), "supplier dispatch preview failed", "supplier_id", sid, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "dispatch_preview_failed"})
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

// HandleActivity serves GET /v1/supplier/activity.
func (s *Service) HandleActivity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	sid := s.scopedSupplierID(r)
	limit, _ := parseListPagination(r, 20, 50)
	orders, err := s.listSupplierOrders(r.Context(), sid, "", "")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_activity_failed"})
		return
	}
	events := buildSupplierActivityEvents(orders, limit)
	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}

// HandleManifests serves GET /v1/supplier/manifests.
func (s *Service) HandleManifests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	sid := s.scopedSupplierID(r)
	rows, err := s.listSupplierManifests(r.Context(), sid)
	if err != nil {
		s.log.WarnContext(r.Context(), "supplier manifests load failed", "supplier_id", sid, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supplier_manifests_failed"})
		return
	}
	wire := make([]manifest.Wire, len(rows))
	for i := range rows {
		wire[i] = manifest.FromPortalRow(manifest.PortalRow{
			ManifestID:   rows[i].ManifestID,
			Status:       rows[i].Status,
			OrdersCount:  rows[i].OrdersCount,
			DriverID:     rows[i].DriverID,
			DriverName:   rows[i].DriverName,
			VehiclePlate: rows[i].VehiclePlate,
			VehicleID:    rows[i].VehicleID,
			TotalVu:      rows[i].TotalVu,
			UpdatedAt:    rows[i].UpdatedAt,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"manifests": wire})
}

// HandleSupplyLanes serves GET /v1/supplier/supply-lanes.
func (s *Service) HandleSupplyLanes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	sid := s.scopedSupplierID(r)
	lanes, err := s.listSupplierSupplyLanes(r.Context(), sid)
	if err != nil {
		s.log.WarnContext(r.Context(), "supplier supply lanes load failed", "supplier_id", sid, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supply_lanes_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"lanes": lanes})
}

// HandleExceptions serves GET /v1/supplier/exceptions.
func (s *Service) HandleExceptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	sid := s.scopedSupplierID(r)
	rows, err := s.listSupplierExceptions(r.Context(), sid)
	if err != nil {
		s.log.WarnContext(r.Context(), "supplier exceptions load failed", "supplier_id", sid, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supplier_exceptions_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"exceptions": rows})
}

func (s *Service) buildSupplierDashboardDetail(ctx context.Context, sid string, base supplierDashboardResponse) (supplierDashboardDetail, error) {
	orders, err := s.listSupplierOrders(ctx, sid, "", "")
	if err != nil {
		return supplierDashboardDetail{}, err
	}

	profile, _, _ := s.repo.GetProfile(ctx, sid)
	fleetMaxVU := int64(profile.FleetMaxVU)
	if fleetMaxVU <= 0 {
		fleetMaxVU = 1
	}

	manifests, err := s.aggregateManifests(ctx, sid, orders)
	if err != nil {
		return supplierDashboardDetail{}, err
	}

	metrics := aggregateOrderMetrics(orders, s.now())
	drivers, vehicles := s.fleetCounts(ctx, sid)

	detail := supplierDashboardDetail{
		supplierDashboardResponse: base,
		OrdersByStatus:            metrics.ordersByStatus,
		TodayRevenueMinor:         metrics.todayRevenueMinor,
		Currency:                  s.currency,
		ActiveDrivers:             metrics.activeDrivers,
		TotalDrivers:              drivers,
		RetailersOrderedToday:     metrics.retailersToday,
		TotalRetailers:            metrics.totalRetailers,
		DeliveryCompletionRate:    metrics.completionRate,
		FleetVuUsed:               metrics.fleetVuUsed,
		FleetVuTotal:              fleetMaxVU,
		RecentManifests:           manifests,
		ActivityEvents:            buildSupplierActivityEvents(orders, 20),
	}
	if detail.ActiveDrivers == 0 && drivers > 0 {
		detail.ActiveDrivers = metrics.activeDrivers
	}
	if detail.TotalDrivers == 0 {
		detail.TotalDrivers = drivers
	}
	_ = vehicles

	if s.earningsLookup != nil {
		resp, err := s.earningsLookup(ctx, sid, s.currency, s.now())
		if err == nil && resp.TodayMinor > 0 {
			detail.TodayRevenueMinor = resp.TodayMinor
			if strings.TrimSpace(resp.Currency) != "" {
				detail.Currency = resp.Currency
			}
		}
	}

	return detail, nil
}

func (s *Service) listSupplierManifests(ctx context.Context, supplierID string) ([]SupplierManifestRow, error) {
	orders, err := s.listSupplierOrders(ctx, supplierID, "", "")
	if err != nil {
		return nil, err
	}
	return s.aggregateManifests(ctx, supplierID, orders)
}

func (s *Service) listSupplierSupplyLanes(ctx context.Context, supplierID string) ([]SupplierSupplyLaneRow, error) {
	topology, err := s.repo.GetTopology(ctx, supplierID)
	if err != nil {
		return nil, err
	}
	orders, err := s.listSupplierOrders(ctx, supplierID, "", "")
	if err != nil {
		return nil, err
	}
	drivers, err := s.repo.ListFleetDrivers(ctx, supplierID)
	if err != nil {
		return nil, err
	}

	dayStart := time.Date(s.now().Year(), s.now().Month(), s.now().Day(), 0, 0, 0, 0, time.UTC)
	ordersByWarehouse := map[string]int{}
	for _, order := range orders {
		createdAt, err := time.Parse(time.RFC3339Nano, order.CreatedAt)
		if err != nil || createdAt.Before(dayStart) {
			continue
		}
		wid := strings.TrimSpace(order.WarehouseID)
		if wid == "" {
			continue
		}
		ordersByWarehouse[wid]++
	}

	driversByWarehouse := map[string]int{}
	for _, driver := range drivers {
		if driver.HomeNodeType != auth.HomeNodeWarehouse {
			continue
		}
		nodeID := strings.TrimSpace(driver.HomeNodeID)
		if nodeID == "" {
			continue
		}
		driversByWarehouse[nodeID]++
	}

	profile, _, _ := s.repo.GetProfile(ctx, supplierID)
	capacity := profile.FleetMaxVU
	if capacity <= 0 {
		capacity = 1000
	}

	lanes := make([]SupplierSupplyLaneRow, 0, len(topology.Warehouses))
	for _, wh := range topology.Warehouses {
		if !wh.IsActive {
			continue
		}
		wid := strings.TrimSpace(wh.WarehouseID)
		today := ordersByWarehouse[wid]
		util := float64(today) / float64(capacity) * 100
		if util > 100 {
			util = 100
		}
		h3Cells := int(wh.CoverageRadiusKm * 8)
		if h3Cells < 1 {
			h3Cells = 1
		}
		lanes = append(lanes, SupplierSupplyLaneRow{
			LaneID:      wid,
			Name:        wh.Name,
			WarehouseID: wid,
			H3Cells:     h3Cells,
			Drivers:     driversByWarehouse[wid],
			OrdersToday: today,
			Capacity:    capacity,
			Utilization: util,
		})
	}
	sort.Slice(lanes, func(i, j int) bool { return lanes[i].OrdersToday > lanes[j].OrdersToday })
	return lanes, nil
}

func (s *Service) listSupplierExceptions(ctx context.Context, supplierID string) ([]SupplierExceptionRow, error) {
	orders, err := s.listSupplierOrders(ctx, supplierID, "", "")
	if err != nil {
		return nil, err
	}
	rows := make([]SupplierExceptionRow, 0)
	for _, order := range orders {
		status := strings.ToUpper(strings.TrimSpace(order.Status))
		decision := strings.ToUpper(strings.TrimSpace(order.Decision))
		kind := ""
		switch {
		case status == "CANCELLED" || status == "REJECTED":
			kind = "ORDER_CANCELLED"
		case decision == "REJECTED":
			kind = "VET_REJECTED"
		case strings.EqualFold(status, "AWAITING_REVIEW"):
			kind = "AWAITING_REVIEW"
		case strings.TrimSpace(order.Note) != "" && status != "COMPLETED":
			kind = "OPERATOR_NOTE"
		default:
			continue
		}
		rows = append(rows, SupplierExceptionRow{
			OrderID:    order.OrderID,
			Kind:       kind,
			Status:     order.Status,
			RetailerID: order.RetailerID,
			Note:       order.Note,
			ManifestID: order.ManifestID,
			UpdatedAt:  order.UpdatedAt,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].UpdatedAt > rows[j].UpdatedAt })
	if len(rows) > 100 {
		rows = rows[:100]
	}
	return rows, nil
}

type orderMetricsAggregate struct {
	ordersByStatus    map[string]int
	todayRevenueMinor int64
	activeDrivers     int
	retailersToday    int
	totalRetailers    int
	completionRate    float64
	fleetVuUsed       int64
}

func aggregateOrderMetrics(orders []SupplierOrder, now time.Time) orderMetricsAggregate {
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	statuses := map[string]int{
		"PENDING": 0, "LOADED": 0, "IN_TRANSIT": 0, "ARRIVED": 0, "COMPLETED": 0, "CANCELLED": 0,
	}
	retailersToday := map[string]struct{}{}
	allRetailers := map[string]struct{}{}
	activeDrivers := map[string]struct{}{}
	var todayRevenue int64
	completedToday := 0
	attemptedToday := 0

	for _, order := range orders {
		status := strings.ToUpper(strings.TrimSpace(order.Status))
		if _, ok := statuses[status]; ok {
			statuses[status]++
		}
		if strings.TrimSpace(order.RetailerID) != "" {
			allRetailers[order.RetailerID] = struct{}{}
		}
		updatedAt, err := time.Parse(time.RFC3339Nano, order.UpdatedAt)
		if err != nil {
			continue
		}
		if !updatedAt.Before(dayStart) {
			attemptedToday++
			if status == "COMPLETED" {
				completedToday++
				todayRevenue += order.TotalMinor
			}
			createdAt, err := time.Parse(time.RFC3339Nano, order.CreatedAt)
			if err == nil && !createdAt.Before(dayStart) && strings.TrimSpace(order.RetailerID) != "" {
				retailersToday[order.RetailerID] = struct{}{}
			}
		}
		if status == "LOADED" || status == "IN_TRANSIT" || status == "ARRIVED" {
			if driverID := strings.TrimSpace(order.DriverID); driverID != "" {
				activeDrivers[driverID] = struct{}{}
			}
		}
	}

	completion := 0.0
	if attemptedToday > 0 {
		completion = float64(completedToday) / float64(attemptedToday) * 100
	}

	return orderMetricsAggregate{
		ordersByStatus:    statuses,
		todayRevenueMinor: todayRevenue,
		activeDrivers:     len(activeDrivers),
		retailersToday:    len(retailersToday),
		totalRetailers:    len(allRetailers),
		completionRate:    completion,
		fleetVuUsed:       int64(len(orders)) * 10,
	}
}

func (s *Service) aggregateManifests(ctx context.Context, supplierID string, orders []SupplierOrder) ([]SupplierManifestRow, error) {
	type manifestAcc struct {
		row      SupplierManifestRow
		statuses map[string]int
	}
	byManifest := map[string]*manifestAcc{}
	unassigned := make([]SupplierOrder, 0)

	driverNames := map[string]string{}
	if drivers, err := s.repo.ListFleetDrivers(ctx, supplierID); err == nil {
		for _, d := range drivers {
			driverNames[d.DriverID] = strings.TrimSpace(d.Name)
		}
	}
	vehiclePlates := map[string]string{}
	if vehicles, err := s.repo.ListFleetVehicles(ctx, supplierID); err == nil {
		for _, v := range vehicles {
			vehiclePlates[v.VehicleID] = strings.TrimSpace(v.LicensePlate)
		}
	}

	for _, order := range orders {
		mid := strings.TrimSpace(order.ManifestID)
		if mid == "" {
			unassigned = append(unassigned, order)
			continue
		}
		acc, ok := byManifest[mid]
		if !ok {
			acc = &manifestAcc{
				row: SupplierManifestRow{
					ManifestID: mid,
					Status:     "DRAFT",
					DriverID:   strings.TrimSpace(order.DriverID),
				},
				statuses: map[string]int{},
			}
			byManifest[mid] = acc
		}
		acc.row.OrdersCount++
		acc.row.TotalVu += order.TotalMinor / 1000
		if order.UpdatedAt > acc.row.UpdatedAt {
			acc.row.UpdatedAt = order.UpdatedAt
		}
		status := strings.ToUpper(strings.TrimSpace(order.Status))
		acc.statuses[status]++
		if strings.TrimSpace(order.DriverID) != "" {
			acc.row.DriverID = order.DriverID
		}
		vid := strings.TrimSpace(order.VehicleID)
		if vid != "" {
			acc.row.VehicleID = vid
			acc.row.TruckID = vid
		}
		if plate := vehiclePlates[vid]; plate != "" {
			acc.row.VehiclePlate = plate
		}
	}

	rows := make([]SupplierManifestRow, 0, len(byManifest))
	for _, acc := range byManifest {
		acc.row.Status = manifestStatusFromOrders(acc.statuses)
		acc.row.State = acc.row.Status
		acc.row.StopCount = acc.row.OrdersCount
		acc.row.TotalVolumeVU = float64(acc.row.TotalVu)
		if name := driverNames[acc.row.DriverID]; name != "" {
			acc.row.DriverName = name
		} else if acc.row.DriverID != "" {
			acc.row.DriverName = acc.row.DriverID
		} else {
			acc.row.DriverName = "Unassigned"
		}
		rows = append(rows, acc.row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].UpdatedAt > rows[j].UpdatedAt })

	if len(unassigned) > 0 {
		rows = append(rows, SupplierManifestRow{
			ManifestID:  "unassigned",
			Status:      "DRAFT",
			OrdersCount: len(unassigned),
			DriverName:  "Unassigned",
			UpdatedAt:   unassigned[0].UpdatedAt,
		})
	}
	return rows, nil
}

func manifestStatusFromOrders(statuses map[string]int) string {
	if statuses["IN_TRANSIT"]+statuses["ARRIVED"]+statuses["COMPLETED"] > 0 {
		return "DISPATCHED"
	}
	if statuses["LOADED"] > 0 {
		return "LOADING"
	}
	return "DRAFT"
}

func (s *Service) buildSupplierDispatchPreview(ctx context.Context, supplierID, warehouseID string) (SupplierDispatchPreview, error) {
	undispatched := make([]map[string]any, 0)
	windowConstrained := 0
	dispatchRows := make([]dispatch.DispatchableOrder, 0)
	if s.portalSpanner != nil {
		repo := dispatch.NewRepository(s.portalSpanner)
		rows, err := repo.FetchDispatchable(ctx, dispatch.FetchParams{
			SupplierID:  supplierID,
			WarehouseID: warehouseID,
		})
		if err != nil {
			return SupplierDispatchPreview{}, err
		}
		dispatchRows = rows
		preview := dispatch.BuildPreview(rows)
		undispatched = preview.UndispatchedOrders
		windowConstrained = preview.WindowConstrained
	} else {
		orders, err := s.listSupplierOrders(ctx, supplierID, "", "")
		if err != nil {
			return SupplierDispatchPreview{}, err
		}
		for _, order := range orders {
			if warehouseID != "" && !strings.EqualFold(strings.TrimSpace(order.WarehouseID), warehouseID) {
				continue
			}
			if !strings.EqualFold(order.Status, "PENDING") && !strings.EqualFold(order.Status, "LOADED") {
				continue
			}
			undispatched = append(undispatched, map[string]any{
				"order_id":     order.OrderID,
				"retailer_id":  order.RetailerID,
				"warehouse_id": order.WarehouseID,
				"total_minor":  order.TotalMinor,
				"currency":     order.Currency,
			})
		}
	}

	available := make([]map[string]any, 0)
	unavailable := make([]map[string]any, 0)
	vehiclesByID := make(map[string]dispatch.VehicleSpec)
	if fleetVehicles, err := s.repo.ListFleetVehicles(ctx, supplierID); err == nil {
		for _, vehicle := range fleetVehicles {
			id, spec := dispatch.VehicleSpecIndex(vehicle.VehicleID, vehicle.VehicleClass, vehicle.MaxVolumeVU)
			vehiclesByID[id] = spec
		}
	}
	drivers, err := s.repo.ListFleetDrivers(ctx, supplierID)
	if err != nil {
		return SupplierDispatchPreview{}, err
	}
	for _, driver := range drivers {
		entry := map[string]any{
			"driver_id":  driver.DriverID,
			"name":       driver.Name,
			"vehicle_id": driver.VehicleID,
		}
		if spec, ok := vehiclesByID[strings.TrimSpace(driver.VehicleID)]; ok {
			entry["vehicle_class"] = spec.VehicleClass
			entry["max_volume_vu"] = spec.MaxVolumeVU
		}
		if driver.IsActive {
			entry["truck_status"] = "AVAILABLE"
			available = append(available, entry)
		} else {
			entry["truck_status"] = "UNAVAILABLE"
			entry["unavailable_reason"] = "INACTIVE"
			unavailable = append(unavailable, entry)
		}
	}

	out := SupplierDispatchPreview{
		UndispatchedOrders:     undispatched,
		AvailableDrivers:       available,
		UnavailableDrivers:     unavailable,
		PendingCount:           len(undispatched),
		AvailableCount:         len(available),
		WindowConstrainedCount: windowConstrained,
	}

	if len(dispatchRows) > 0 && len(available) > 0 {
		driverInputs := make([]dispatch.FleetDriverInput, 0, len(drivers))
		for _, driver := range drivers {
			if !driver.IsActive {
				continue
			}
			if warehouseID != "" && !strings.EqualFold(strings.TrimSpace(driver.HomeNodeID), warehouseID) {
				continue
			}
			driverInputs = append(driverInputs, dispatch.FleetDriverInput{
				DriverID:    driver.DriverID,
				DriverName:  driver.Name,
				VehicleID:   driver.VehicleID,
				IsActive:    driver.IsActive,
				TruckStatus: "AVAILABLE",
				HomeNodeID:  driver.HomeNodeID,
			})
		}
		fleet := dispatch.BuildAvailableFleet(driverInputs, vehiclesByID)
		homeNodeID := strings.TrimSpace(warehouseID)
		if homeNodeID == "" && len(driverInputs) > 0 {
			homeNodeID = driverInputs[0].HomeNodeID
		}
		depot := dispatch.ResolveDepot(ctx, s.portalSpanner, warehouseID, dispatch.DepotCoords{
			Lat: s.fallbackDepotLat,
			Lng: s.fallbackDepotLng,
		})
		job := plan.BuildSolveJob(ctx, supplierID, homeNodeID, depot, dispatchRows, fleet)
		solve := plan.RunSolvePreview(ctx, s.optimizerClient, s.planCounters, job)
		out.ProposedRoutes = solve.ProposedRoutes
		out.OptimizerSource = solve.OptimizerSource
		out.OptimizerWarnings = solve.OptimizerWarnings
	}

	return out, nil
}

func buildSupplierActivityEvents(orders []SupplierOrder, limit int) []SupplierActivityEvent {
	if limit <= 0 {
		limit = 20
	}
	sorted := append([]SupplierOrder(nil), orders...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].UpdatedAt > sorted[j].UpdatedAt })
	events := make([]SupplierActivityEvent, 0, limit)
	for _, order := range sorted {
		if len(events) >= limit {
			break
		}
		eventType := "ORDER_" + strings.ToUpper(strings.TrimSpace(order.Status))
		if eventType == "ORDER_" {
			eventType = "ORDER_UPDATED"
		}
		events = append(events, SupplierActivityEvent{
			ID:          "order-" + order.OrderID,
			Type:        eventType,
			Timestamp:   order.UpdatedAt,
			Description: fmt.Sprintf("Order %s · %s", order.OrderID, order.Status),
			OrderID:     order.OrderID,
			ManifestID:  order.ManifestID,
		})
	}
	return events
}

func (s *Service) fleetCounts(ctx context.Context, supplierID string) (totalDrivers int, totalVehicles int) {
	if drivers, err := s.repo.ListFleetDrivers(ctx, supplierID); err == nil {
		totalDrivers = len(drivers)
	}
	if vehicles, err := s.repo.ListFleetVehicles(ctx, supplierID); err == nil {
		totalVehicles = len(vehicles)
	}
	return totalDrivers, totalVehicles
}
