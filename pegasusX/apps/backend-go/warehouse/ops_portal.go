package warehouse

import (
	"encoding/json"
	"net/http"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch"
	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch/plan"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/manifest"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// PortalDriver is the warehouse ops driver read model.
type PortalDriver struct {
	DriverID                 string  `json:"driver_id"`
	Name                     string  `json:"name"`
	Phone                    string  `json:"phone"`
	TruckStatus              string  `json:"truck_status"`
	IsActive                 bool    `json:"is_active"`
	VehicleID                string  `json:"vehicle_id,omitempty"`
	VehicleClass             string  `json:"vehicle_class,omitempty"`
	MaxVolumeVU              float64 `json:"max_volume_vu,omitempty"`
	VehicleIsActive          bool    `json:"vehicle_is_active,omitempty"`
	VehicleUnavailableReason string  `json:"vehicle_unavailable_reason,omitempty"`
}

// PortalVehicle is the warehouse ops vehicle read model.
type PortalVehicle struct {
	VehicleID          string  `json:"vehicle_id"`
	Label              string  `json:"label"`
	LicensePlate       string  `json:"license_plate"`
	VehicleClass       string  `json:"vehicle_class"`
	MaxVolumeVU        float64 `json:"max_volume_vu,omitempty"`
	IsActive           bool    `json:"is_active"`
	UnavailableReason  string  `json:"unavailable_reason,omitempty"`
	AssignedDriverID   string  `json:"assigned_driver_id,omitempty"`
	AssignedDriverName string  `json:"assigned_driver_name,omitempty"`
}

type portalStaff struct {
	StaffID string `json:"staff_id"`
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Role    string `json:"role"`
}

type portalProduct struct {
	ProductID string `json:"product_id"`
	Name      string `json:"name"`
	SKU       string `json:"sku_id"`
	Category  string `json:"category"`
	PriceUZS  int    `json:"price_uzs"`
	IsActive  bool   `json:"is_active"`
}

type portalManifest struct {
	ManifestID   string `json:"manifest_id"`
	DriverName   string `json:"driver_name"`
	VehicleLabel string `json:"vehicle_label"`
	StopCount    int    `json:"stop_count"`
	CreatedAt    string `json:"created_at"`
}

type portalRetailer struct {
	RetailerID   string `json:"retailer_id"`
	BusinessName string `json:"business_name"`
	OrderCount   int64  `json:"order_count"`
	RevenueUZS   int64  `json:"revenue_uzs"`
	LastOrderAt  string `json:"last_order_at,omitempty"`
}

type portalReturn struct {
	ReturnID    string `json:"return_id"`
	OrderID     string `json:"order_id"`
	ProductName string `json:"product_name"`
	Quantity    int    `json:"quantity"`
	Status      string `json:"status"`
}

type portalOrder struct {
	OrderID      string `json:"order_id"`
	RetailerName string `json:"retailer_name"`
	State        string `json:"state"`
	TotalUZS     int    `json:"total_uzs"`
}

func (s *Service) ensurePortalSeed() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.portalSeeded {
		return
	}
	now := s.now().Format("2006-01-02T15:04:05Z")
	s.orders = []OrderRow{
		{OrderID: "ord-wh-1", RetailerID: "ret-1", Status: "PENDING", TotalMinor: 12500000, Currency: s.currency, UpdatedAt: now},
		{OrderID: "ord-wh-2", RetailerID: "ret-2", Status: "IN_TRANSIT", TotalMinor: 8900000, Currency: s.currency, UpdatedAt: now},
	}
	s.drivers = []PortalDriver{
		{DriverID: "drv-1", Name: "Jamshid R.", Phone: "+998901111001", TruckStatus: "AVAILABLE", IsActive: true, VehicleID: "veh-1", VehicleClass: "CLASS_A", MaxVolumeVU: 50, VehicleIsActive: true},
		{DriverID: "drv-2", Name: "Dilnoza K.", Phone: "+998901111002", TruckStatus: "IN_TRANSIT", IsActive: true, VehicleID: "veh-2", VehicleClass: "CLASS_C", MaxVolumeVU: 400, VehicleIsActive: true},
	}
	s.vehicles = []PortalVehicle{
		{VehicleID: "veh-1", Label: "Van 12", LicensePlate: "01A111AA", VehicleClass: "CLASS_A", MaxVolumeVU: 50, IsActive: true, AssignedDriverID: "drv-1", AssignedDriverName: "Jamshid R."},
		{VehicleID: "veh-2", Label: "Truck 08", LicensePlate: "01B222BB", VehicleClass: "CLASS_C", MaxVolumeVU: 400, IsActive: true, AssignedDriverID: "drv-2", AssignedDriverName: "Dilnoza K."},
	}
	s.staff = []portalStaff{{StaffID: "stf-1", Name: "Ops Lead", Phone: "+998901000088", Role: "WAREHOUSE_ADMIN"}}
	s.products = []portalProduct{
		{ProductID: "prod-1", Name: "Mineral Water 1.5L", SKU: "SKU-1001", Category: "Beverages", PriceUZS: 12000, IsActive: true},
		{ProductID: "prod-2", Name: "Sunflower Oil 1L", SKU: "SKU-2002", Category: "Grocery", PriceUZS: 28000, IsActive: true},
	}
	s.manifests = []portalManifest{{ManifestID: "mf-1", DriverName: "Jamshid R.", VehicleLabel: "Van 12", StopCount: 6, CreatedAt: now}}
	s.retailers = []portalRetailer{
		{RetailerID: "ret-1", BusinessName: "Corner Shop 12", OrderCount: 42, RevenueUZS: 128000000, LastOrderAt: now},
		{RetailerID: "ret-2", BusinessName: "Family Market", OrderCount: 18, RevenueUZS: 56000000, LastOrderAt: now},
	}
	s.returns = []portalReturn{{ReturnID: "retn-1", OrderID: "ord-wh-1", ProductName: "Mineral Water 1.5L", Quantity: 2, Status: "PENDING"}}
	s.portalSeeded = true
}

func (s *Service) handleOpsDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	whID := warehouseIDFromRequest(r)

	var active, pending, onRoute, idle, totalDrivers, totalVehicles int64
	var lowStock int64

	if s.opsOrders != nil {
		orders, err := s.opsOrders(r.Context(), whID, 200)
		if err == nil {
			for _, o := range orders {
				switch strings.ToUpper(o.Status) {
				case "PENDING", "LOADED":
					pending++
					active++
				case "IN_TRANSIT", "ARRIVED":
					active++
				}
			}
		}
	} else {
		s.ensurePortalSeed()
		s.mu.RLock()
		for _, o := range s.orders {
			switch strings.ToUpper(o.Status) {
			case "PENDING", "LOADED":
				pending++
				active++
			case "IN_TRANSIT", "ARRIVED":
				active++
			}
		}
		s.mu.RUnlock()
	}

	if s.opsDrivers != nil {
		drivers, err := s.opsDrivers(r.Context(), whID)
		if err == nil {
			totalDrivers = int64(len(drivers))
			for _, d := range drivers {
				if !d.IsActive {
					continue
				}
				if strings.EqualFold(d.TruckStatus, "IN_TRANSIT") {
					onRoute++
				} else {
					idle++
				}
			}
		}
	} else {
		s.mu.RLock()
		totalDrivers = int64(len(s.drivers))
		for _, d := range s.drivers {
			if !d.IsActive {
				continue
			}
			if strings.EqualFold(d.TruckStatus, "IN_TRANSIT") {
				onRoute++
			} else {
				idle++
			}
		}
		s.mu.RUnlock()
	}

	if s.opsVehicles != nil {
		vehicles, err := s.opsVehicles(r.Context(), whID)
		if err == nil {
			totalVehicles = int64(len(vehicles))
		}
	} else {
		s.mu.RLock()
		totalVehicles = int64(len(s.vehicles))
		s.mu.RUnlock()
	}

	inventoryList, _ := s.repo.GetInventoryList(r.Context(), "wh-1")
	for _, row := range inventoryList {
		if row.Quantity < 20 {
			lowStock++
		}
	}
	s.mu.RLock()
	staffCount := int64(len(s.staff))
	s.mu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"warehouse_id":     whID,
		"active_orders":    active,
		"completed_today":  int64(0),
		"pending_dispatch": pending,
		"drivers_on_route": onRoute,
		"drivers_idle":     idle,
		"total_drivers":    totalDrivers,
		"total_vehicles":   totalVehicles,
		"today_revenue":    int64(0),
		"low_stock_count":  lowStock,
		"total_staff":      staffCount,
		"fleet_status": []map[string]any{
			{"status": "AVAILABLE", "count": idle},
			{"status": "IN_TRANSIT", "count": onRoute},
		},
	})
}

func (s *Service) handleOpsInventory(w http.ResponseWriter, r *http.Request) {
	s.ensurePortalSeed()
	switch r.Method {
	case http.MethodGet:
		lowOnly := strings.EqualFold(r.URL.Query().Get("low_stock"), "true")
		inventoryList, err := s.repo.GetInventoryList(r.Context(), "wh-1")
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed_to_fetch_inventory"})
			return
		}
		items := make([]map[string]any, 0, len(inventoryList))
		for sku, row := range inventoryList {
			qty := row.Quantity
			isLow := qty < 20
			if lowOnly && !isLow {
				continue
			}
			items = append(items, map[string]any{
				"product_id":   sku,
				"sku_id":       row.SKU,
				"product_name": row.ProductName,
				"quantity":     qty,
				"is_low_stock": isLow,
				"last_updated": row.UpdatedAt,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"inventory": items, "items": items})
	case http.MethodPatch:
		var body struct {
			ProductID string `json:"product_id"`
			Quantity  int64  `json:"quantity"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		defer r.Body.Close()
		err := s.repo.UpdateInventoryQuantity(r.Context(), "wh-1", body.ProductID, body.Quantity, func(buf outbox.TxnBuffer) error {
			// omit event emission for simple patch if none is required
			return nil
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed_to_update_inventory"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) handleOpsOrders(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/warehouse/ops/orders")
	path = strings.Trim(path, "/")
	if path != "" {
		s.handleOpsOrderDetail(w, r, path)
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	stateFilter := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("state")))
	whID := warehouseIDFromRequest(r)

	if s.opsOrders != nil {
		rows, err := s.opsOrders(r.Context(), whID, 200)
		if err == nil {
			orders := make([]portalOrder, 0, len(rows))
			for _, row := range rows {
				state := strings.ToUpper(row.Status)
				if stateFilter != "" && state != stateFilter {
					continue
				}
				orders = append(orders, portalOrder{
					OrderID:      row.OrderID,
					RetailerName: "Retailer " + row.RetailerID,
					State:        state,
					TotalUZS:     int(row.TotalMinor / 100),
				})
			}
			writeJSON(w, http.StatusOK, map[string]any{"orders": orders})
			return
		}
		s.log.WarnContext(r.Context(), "ops orders query failed, falling back", "err", err)
	}

	s.ensurePortalSeed()
	s.mu.RLock()
	orders := make([]portalOrder, 0, len(s.orders))
	for _, row := range s.orders {
		state := strings.ToUpper(row.Status)
		if stateFilter != "" && state != stateFilter {
			continue
		}
		orders = append(orders, portalOrder{
			OrderID:      row.OrderID,
			RetailerName: "Retailer " + row.RetailerID,
			State:        state,
			TotalUZS:     int(row.TotalMinor / 100),
		})
	}
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{"orders": orders})
}

func (s *Service) handleOpsOrderDetail(w http.ResponseWriter, r *http.Request, orderID string) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, row := range s.orders {
		if row.OrderID != orderID {
			continue
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"order_id":      row.OrderID,
			"retailer_name": "Retailer " + row.RetailerID,
			"state":         strings.ToUpper(row.Status),
			"total_uzs":     int(row.TotalMinor / 100),
			"line_items": []map[string]any{
				{"product_id": "prod-1", "product_name": "Mineral Water 1.5L", "quantity": 10, "unit_price": 12000},
			},
		})
		return
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
}

func (s *Service) handleOpsDispatchPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	whID := warehouseIDFromRequest(r)

	undispatched := make([]map[string]any, 0)
	windowConstrained := 0
	dispatchRows := make([]dispatch.DispatchableOrder, 0)
	if s.spannerClient != nil && strings.TrimSpace(s.supplierID) != "" {
		repo := dispatch.NewRepository(s.spannerClient)
		rows, err := repo.FetchDispatchable(r.Context(), dispatch.FetchParams{
			SupplierID:  s.supplierID,
			WarehouseID: whID,
		})
		if err == nil {
			dispatchRows = rows
			preview := dispatch.BuildPreview(rows)
			undispatched = make([]map[string]any, 0, len(preview.UndispatchedOrders))
			for _, row := range preview.UndispatchedOrders {
				totalMinor, _ := row["total_minor"].(int64)
				undispatched = append(undispatched, map[string]any{
					"order_id":               row["order_id"],
					"retailer_id":            row["retailer_id"],
					"retailer_name":          row["retailer_name"],
					"total_uzs":              int(totalMinor / 100),
					"total_minor":            totalMinor,
					"currency":               row["currency"],
					"receiving_window_open":  row["receiving_window_open"],
					"receiving_window_close": row["receiving_window_close"],
					"has_receiving_window":   row["has_receiving_window"],
					"volume_vu":              row["volume_vu"],
				})
			}
			windowConstrained = preview.WindowConstrained
		}
	} else if s.opsOrders != nil {
		rows, err := s.opsOrders(r.Context(), whID, 200)
		if err == nil {
			for _, o := range rows {
				if strings.EqualFold(o.Status, "PENDING") || strings.EqualFold(o.Status, "LOADED") {
					undispatched = append(undispatched, map[string]any{
						"order_id":      o.OrderID,
						"retailer_name": "Retailer " + o.RetailerID,
						"total_uzs":     int(o.TotalMinor / 100),
						"item_count":    3,
					})
				}
			}
		}
	} else {
		s.ensurePortalSeed()
		s.mu.RLock()
		for _, o := range s.orders {
			if strings.EqualFold(o.Status, "PENDING") {
				undispatched = append(undispatched, map[string]any{
					"order_id":      o.OrderID,
					"retailer_name": "Retailer " + o.RetailerID,
					"total_uzs":     int(o.TotalMinor / 100),
					"item_count":    3,
				})
			}
		}
		s.mu.RUnlock()
	}

	available := make([]map[string]any, 0)
	unavailable := make([]map[string]any, 0)
	var solveDrivers []PortalDriver
	if s.opsDrivers != nil {
		drivers, err := s.opsDrivers(r.Context(), whID)
		if err == nil {
			solveDrivers = drivers
			for _, d := range drivers {
				entry := map[string]any{
					"driver_id":     d.DriverID,
					"name":          d.Name,
					"truck_status":  d.TruckStatus,
					"vehicle_id":    d.VehicleID,
					"vehicle_class": d.VehicleClass,
					"max_volume_vu": d.MaxVolumeVU,
				}
				if strings.EqualFold(d.TruckStatus, "AVAILABLE") {
					available = append(available, entry)
				} else {
					entry["unavailable_reason"] = d.TruckStatus
					unavailable = append(unavailable, entry)
				}
			}
		}
	} else {
		s.ensurePortalSeed()
		s.mu.RLock()
		solveDrivers = append([]PortalDriver(nil), s.drivers...)
		for _, d := range s.drivers {
			entry := map[string]any{
				"driver_id":     d.DriverID,
				"name":          d.Name,
				"truck_status":  d.TruckStatus,
				"vehicle_id":    d.VehicleID,
				"vehicle_class": d.VehicleClass,
				"max_volume_vu": d.MaxVolumeVU,
			}
			if strings.EqualFold(d.TruckStatus, "AVAILABLE") {
				available = append(available, entry)
			} else {
				entry["unavailable_reason"] = d.TruckStatus
				unavailable = append(unavailable, entry)
			}
		}
		s.mu.RUnlock()
	}

	response := map[string]any{
		"preview_ready":            true,
		"undispatched_orders":      undispatched,
		"available_drivers":        available,
		"unavailable_drivers":      unavailable,
		"window_constrained_count": windowConstrained,
	}
	if len(dispatchRows) > 0 && len(solveDrivers) > 0 {
		driverInputs := make([]dispatch.FleetDriverInput, 0, len(solveDrivers))
		for _, driver := range solveDrivers {
			if !strings.EqualFold(driver.TruckStatus, "AVAILABLE") {
				continue
			}
			driverInputs = append(driverInputs, dispatch.FleetDriverInput{
				DriverID:     driver.DriverID,
				DriverName:   driver.Name,
				VehicleID:    driver.VehicleID,
				VehicleClass: driver.VehicleClass,
				MaxVolumeVU:  driver.MaxVolumeVU,
				IsActive:     driver.IsActive,
				TruckStatus:  driver.TruckStatus,
				HomeNodeID:   whID,
			})
		}
		fleet := dispatch.BuildAvailableFleet(driverInputs, nil)
		depot := dispatch.ResolveDepot(r.Context(), s.spannerClient, whID, dispatch.DepotCoords{
			Lat: s.fallbackDepotLat,
			Lng: s.fallbackDepotLng,
		})
		job := plan.BuildSolveJob(r.Context(), s.supplierID, whID, depot, dispatchRows, fleet)
		solve := plan.RunSolvePreview(r.Context(), s.optimizerClient, s.planCounters, job)
		if len(solve.ProposedRoutes) > 0 {
			response["proposed_routes"] = solve.ProposedRoutes
		}
		if solve.OptimizerSource != "" {
			response["optimizer_source"] = solve.OptimizerSource
		}
		if len(solve.OptimizerWarnings) > 0 {
			response["optimizer_warnings"] = solve.OptimizerWarnings
		}
	}
	writeJSON(w, http.StatusOK, response)
}

type DispatchExecuteResult struct {
	Status           string                 `json:"status"`
	SupplierID       string                 `json:"supplier_id"`
	WarehouseID      string                 `json:"warehouse_id,omitempty"`
	ManifestsCreated int                    `json:"manifests_created"`
	OrdersAssigned   int                    `json:"orders_assigned"`
	OptimizerSource  string                 `json:"optimizer_source,omitempty"`
	Warnings         []string               `json:"warnings,omitempty"`
	Manifests        []DispatchExecuteRoute `json:"manifests"`
	Orphans          []string               `json:"orphan_order_ids,omitempty"`
}

type DispatchExecuteRoute struct {
	ManifestID string   `json:"manifest_id"`
	RouteID    string   `json:"route_id"`
	DriverID   string   `json:"driver_id"`
	VehicleID  string   `json:"vehicle_id,omitempty"`
	OrderIDs   []string `json:"order_ids"`
	VolumeVU   float64  `json:"volume_vu"`
	MaxVolume  float64  `json:"max_volume_vu"`
}

func (s *Service) handleOpsDispatchExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if s.spannerClient == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "dispatch_unavailable"})
		return
	}
	whID := warehouseIDFromRequest(r)
	sid := s.supplierID

	var req struct {
		Mode   string                 `json:"mode"`
		Routes []DispatchExecuteRoute `json:"routes"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
		r.Body.Close()
	}

	out := DispatchExecuteResult{
		Status:      "no_op",
		SupplierID:  sid,
		WarehouseID: whID,
		Manifests:   []DispatchExecuteRoute{},
	}

	repo := dispatch.NewRepository(s.spannerClient)
	rows, err := repo.FetchDispatchable(r.Context(), dispatch.FetchParams{
		SupplierID:  sid,
		WarehouseID: whID,
		StrongRead:  true,
	})
	if err != nil {
		s.log.ErrorContext(r.Context(), "dispatch execute failed", "warehouse_id", whID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "dispatch_execute_failed"})
		return
	}
	if len(rows) == 0 {
		writeJSON(w, http.StatusOK, out)
		return
	}

	var solveDrivers []PortalDriver
	if s.opsDrivers != nil {
		drivers, err := s.opsDrivers(r.Context(), whID)
		if err == nil {
			solveDrivers = drivers
		}
	} else {
		s.mu.RLock()
		solveDrivers = append([]PortalDriver(nil), s.drivers...)
		s.mu.RUnlock()
	}

	var driverInputs []dispatch.FleetDriverInput
	vehicleByDriver := make(map[string]string)
	for _, driver := range solveDrivers {
		if !strings.EqualFold(driver.TruckStatus, "AVAILABLE") {
			continue
		}
		driverInputs = append(driverInputs, dispatch.FleetDriverInput{
			DriverID:     driver.DriverID,
			DriverName:   driver.Name,
			VehicleID:    driver.VehicleID,
			VehicleClass: driver.VehicleClass,
			MaxVolumeVU:  driver.MaxVolumeVU,
			IsActive:     driver.IsActive,
			TruckStatus:  driver.TruckStatus,
			HomeNodeID:   whID,
		})
		vehicleByDriver[strings.TrimSpace(driver.DriverID)] = strings.TrimSpace(driver.VehicleID)
	}

	var assignment *dispatch.AssignmentResult
	var source string

	if strings.ToUpper(req.Mode) == "MANUAL" {
		source = "manual"
		assignment = &dispatch.AssignmentResult{}
		
		orderMap := make(map[string]dispatch.DispatchableOrder)
		for _, row := range rows {
			orderMap[row.OrderID] = row
		}

		for _, mr := range req.Routes {
			route := dispatch.DispatchRoute{
				DriverID: mr.DriverID,
			}
			for _, d := range solveDrivers {
				if d.DriverID == mr.DriverID {
					route.MaxVolume = d.MaxVolumeVU
					break
				}
			}
			for _, oid := range mr.OrderIDs {
				if o, ok := orderMap[oid]; ok {
					route.Orders = append(route.Orders, o.ToGeo())
					route.LoadedVolume += o.VolumeVU
				}
			}
			if len(route.Orders) > 0 {
				assignment.Routes = append(assignment.Routes, route)
			}
		}
	} else {
		fleet := dispatch.BuildAvailableFleet(driverInputs, nil)
		if len(fleet) == 0 {
			out.Warnings = append(out.Warnings, "no_available_drivers")
			writeJSON(w, http.StatusOK, out)
			return
		}

		depot := dispatch.ResolveDepot(r.Context(), s.spannerClient, whID, dispatch.DepotCoords{
			Lat: s.fallbackDepotLat,
			Lng: s.fallbackDepotLng,
		})
		job := plan.BuildSolveJob(r.Context(), sid, whID, depot, rows, fleet)
		assignment, source, err = plan.OptimizeAndValidate(r.Context(), s.optimizerClient, job)
		if err != nil {
			s.log.ErrorContext(r.Context(), "dispatch execute optimize failed", "warehouse_id", whID, "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "dispatch_execute_failed"})
			return
		}
	}
	out.OptimizerSource = source
	if assignment != nil {
		out.Warnings = append(out.Warnings, assignment.Warnings...)
		for _, orphan := range assignment.Orphans {
			out.Orphans = append(out.Orphans, orphan.OrderID)
		}
	}
	if assignment == nil || len(assignment.Routes) == 0 {
		writeJSON(w, http.StatusOK, out)
		return
	}

	now := s.now().UTC()
	batch := &manifest.SupplierWriteBatch{}
	committed := make([]DispatchExecuteRoute, 0, len(assignment.Routes))
	type pendingEvent struct {
		aggregateType string
		aggregateID   string
		payload       any
	}
	queued := make([]pendingEvent, 0, len(rows)+len(assignment.Routes))

	for _, route := range assignment.Routes {
		driverID := strings.TrimSpace(route.DriverID)
		if driverID == "" || len(route.Orders) == 0 {
			continue
		}
		manifestID := uuid.NewString()
		routeID := uuid.NewString()
		vehicleID := strings.TrimSpace(vehicleByDriver[driverID])
		sealedAt := now

		batch.Manifests = append(batch.Manifests, manifest.SupplierTruckRow{
			ManifestID:    manifestID,
			SupplierID:    sid,
			WarehouseID:   whID,
			RouteID:       routeID,
			TruckID:       vehicleID,
			DriverID:      driverID,
			State:         "SEALED",
			TotalVolumeVU: route.LoadedVolume,
			MaxVolumeVU:   route.MaxVolume,
			StopCount:     int64(len(route.Orders)),
			SealedAt:      &sealedAt,
			CreatedAt:     now,
		})

		orderIDs := make([]string, 0, len(route.Orders))
		for idx, stop := range route.Orders {
			orderID := strings.TrimSpace(stop.OrderID)
			if orderID == "" {
				continue
			}
			batch.Orders = append(batch.Orders, manifest.SupplierManifestOrderRow{
				ManifestID:    manifestID,
				OrderID:       orderID,
				SequenceIndex: int64(idx),
				LoadingOrder:  int64(idx),
				VolumeVU:      stop.Volume,
				State:         "LOADED",
				UpdatedAt:     now,
			})
			batch.OrderPatches = append(batch.OrderPatches, manifest.OrderPatch{
				OrderID:    orderID,
				Status:     "LOADED",
				ManifestID: manifestID,
				DriverID:   driverID,
				VehicleID:  vehicleID,
				RouteID:    routeID,
				UpdatedAt:  now,
			})
			queued = append(queued, pendingEvent{
				aggregateType: events.AggregateOrder,
				aggregateID:   orderID,
				payload: events.OrderEvent{
					BaseEvent:   events.BaseEvent{Type: events.EventOrderAssigned},
					OrderID:     orderID,
					SupplierID:  sid,
					RetailerID:  stop.RetailerID,
					WarehouseID: whID,
					DriverID:    driverID,
					VehicleID:   vehicleID,
					RouteID:     routeID,
					ManifestID:  manifestID,
					Status:      "LOADED",
				},
			})
			orderIDs = append(orderIDs, orderID)
		}

		queued = append(queued,
			pendingEvent{
				aggregateType: events.AggregateRoute,
				aggregateID:   routeID,
				payload: events.RouteEvent{
					BaseEvent:   events.BaseEvent{Type: events.EventRouteCreated},
					RouteID:     routeID,
					ManifestID:  manifestID,
					SupplierID:  sid,
					WarehouseID: whID,
					DriverID:    driverID,
					VehicleID:   vehicleID,
					OrderIDs:    orderIDs,
				},
			},
			pendingEvent{
				aggregateType: events.AggregateManifest,
				aggregateID:   manifestID,
				payload: events.ManifestEvent{
					BaseEvent:   events.BaseEvent{Type: events.EventManifestSealed},
					ManifestID:  manifestID,
					RouteID:     routeID,
					SupplierID:  sid,
					WarehouseID: whID,
					DriverID:    driverID,
					StopCount:   int64(len(orderIDs)),
				},
			},
		)

		committed = append(committed, DispatchExecuteRoute{
			ManifestID: manifestID,
			RouteID:    routeID,
			DriverID:   driverID,
			VehicleID:  vehicleID,
			OrderIDs:   orderIDs,
			VolumeVU:   route.LoadedVolume,
			MaxVolume:  route.MaxVolume,
		})
		out.OrdersAssigned += len(orderIDs)
	}

	if len(committed) == 0 {
		writeJSON(w, http.StatusOK, out)
		return
	}

	store := manifest.NewStore(s.spannerClient)
	if err := store.CommitSupplier(r.Context(), batch, func(buf outbox.TxnBuffer) error {
		for _, evt := range queued {
			if err := outbox.EmitJSON(r.Context(), buf, evt.aggregateType, evt.aggregateID, events.TopicMain, evt.payload); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		s.log.ErrorContext(r.Context(), "dispatch execute commit failed", "warehouse_id", whID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "dispatch_execute_failed"})
		return
	}

	out.Status = "dispatched"
	out.ManifestsCreated = len(committed)
	out.Manifests = committed

	s.log.InfoContext(r.Context(), "dispatch executed",
		"warehouse_id", whID,
		"manifests", out.ManifestsCreated,
		"orders_assigned", out.OrdersAssigned,
		"optimizer_source", out.OptimizerSource,
	)
	writeJSON(w, http.StatusOK, out)
}

func (s *Service) HandleOpsDrivers(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/warehouse/ops/drivers")
	path = strings.Trim(path, "/")
	if parts := strings.Split(path, "/"); len(parts) == 2 && parts[1] == "assign-vehicle" {
		s.handleAssignVehicle(w, r, parts[0])
		return
	}
	if path != "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		whID := warehouseIDFromRequest(r)
		if s.opsDrivers != nil {
			drivers, err := s.opsDrivers(r.Context(), whID)
			if err == nil {
				writeJSON(w, http.StatusOK, map[string]any{"drivers": drivers})
				return
			}
			s.log.WarnContext(r.Context(), "ops drivers query failed, falling back", "err", err)
		}
		s.ensurePortalSeed()
		s.mu.RLock()
		drivers := append([]PortalDriver(nil), s.drivers...)
		s.mu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"drivers": drivers})
	case http.MethodPost:
		var req struct {
			Name  string `json:"name"`
			Phone string `json:"phone"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		defer r.Body.Close()
		driverID := "drv-" + uuid.NewString()[:8]
		
		if s.spannerClient != nil {
			now := s.now().UTC()
			m := spanner.Insert("Drivers",
				[]string{"DriverId", "Name", "Phone", "PinHash", "SupplierId", "HomeNodeType", "HomeNodeId", "IsActive", "CreatedAt", "UpdatedAt"},
				[]any{driverID, strings.TrimSpace(req.Name), strings.TrimSpace(req.Phone), "4321", s.supplierID, "WAREHOUSE", warehouseIDFromRequest(r), true, now, now},
			)
			if _, err := s.spannerClient.Apply(r.Context(), []*spanner.Mutation{m}); err != nil {
				s.log.ErrorContext(r.Context(), "failed to create driver", "err", err)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed_to_create_driver"})
				return
			}
		} else {
			driver := PortalDriver{
				DriverID:    driverID,
				Name:        strings.TrimSpace(req.Name),
				Phone:       strings.TrimSpace(req.Phone),
				TruckStatus: "AVAILABLE",
				IsActive:    true,
			}
			s.mu.Lock()
			s.drivers = append(s.drivers, driver)
			s.mu.Unlock()
		}
		
		writeJSON(w, http.StatusCreated, map[string]any{
			"driver_id": driverID,
			"pin":       "4321",
		})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) handleAssignVehicle(w http.ResponseWriter, r *http.Request, driverID string) {
	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	var req struct {
		VehicleID string `json:"vehicle_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	defer r.Body.Close()
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.drivers {
		if s.drivers[i].DriverID != driverID {
			continue
		}
		s.drivers[i].VehicleID = strings.TrimSpace(req.VehicleID)
		writeJSON(w, http.StatusOK, map[string]string{"status": "assigned", "driver_id": driverID})
		return
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "driver_not_found"})
}

func (s *Service) HandleOpsVehicles(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/warehouse/ops/vehicles")
	path = strings.Trim(path, "/")
	if path != "" {
		s.handleOpsVehiclePatch(w, r, path)
		return
	}
	switch r.Method {
	case http.MethodGet:
		whID := warehouseIDFromRequest(r)
		if s.opsVehicles != nil {
			vehicles, err := s.opsVehicles(r.Context(), whID)
			if err == nil {
				writeJSON(w, http.StatusOK, map[string]any{"vehicles": vehicles})
				return
			}
			s.log.WarnContext(r.Context(), "ops vehicles query failed, falling back", "err", err)
		}
		s.ensurePortalSeed()
		s.mu.RLock()
		vehicles := append([]PortalVehicle(nil), s.vehicles...)
		s.mu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"vehicles": vehicles})
	case http.MethodPost:
		var req struct {
			Label        string `json:"label"`
			LicensePlate string `json:"license_plate"`
			VehicleClass string `json:"vehicle_class"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		defer r.Body.Close()
		vehicleID := "veh-" + uuid.NewString()[:8]
		
		if s.spannerClient != nil {
			now := s.now().UTC()
			m := spanner.Insert("Vehicles",
				[]string{"VehicleId", "Label", "LicensePlate", "VehicleClass", "SupplierId", "HomeNodeType", "HomeNodeId", "IsActive", "MaxVolumeVU", "CreatedAt", "UpdatedAt"},
				[]any{vehicleID, strings.TrimSpace(req.Label), strings.TrimSpace(req.LicensePlate), strings.TrimSpace(req.VehicleClass), s.supplierID, "WAREHOUSE", warehouseIDFromRequest(r), true, 150.0, now, now},
			)
			if _, err := s.spannerClient.Apply(r.Context(), []*spanner.Mutation{m}); err != nil {
				s.log.ErrorContext(r.Context(), "failed to create vehicle", "err", err)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed_to_create_vehicle"})
				return
			}
		} else {
			vehicle := PortalVehicle{
				VehicleID:    vehicleID,
				Label:        req.Label,
				LicensePlate: req.LicensePlate,
				VehicleClass: req.VehicleClass,
				IsActive:     true,
			}
			s.mu.Lock()
			s.vehicles = append(s.vehicles, vehicle)
			s.mu.Unlock()
		}
		
		writeJSON(w, http.StatusCreated, map[string]any{
			"vehicle_id": vehicleID,
			"label": req.Label,
			"license_plate": req.LicensePlate,
			"vehicle_class": req.VehicleClass,
			"is_active": true,
		})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) handleOpsVehiclePatch(w http.ResponseWriter, r *http.Request, vehicleID string) {
	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	var req struct {
		IsActive          bool   `json:"is_active"`
		UnavailableReason string `json:"unavailable_reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	defer r.Body.Close()
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.vehicles {
		if s.vehicles[i].VehicleID != vehicleID {
			continue
		}
		s.vehicles[i].IsActive = req.IsActive
		s.vehicles[i].UnavailableReason = strings.TrimSpace(req.UnavailableReason)
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated", "vehicle_id": vehicleID})
		return
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "vehicle_not_found"})
}

func (s *Service) HandleOpsStaff(w http.ResponseWriter, r *http.Request) {
	s.ensurePortalSeed()
	if r.Method == http.MethodGet {
		whID := warehouseIDFromRequest(r)
		if s.spannerClient != nil && whID != "" {
			stmt := spanner.Statement{
				SQL:    `SELECT UserId, Name, Role FROM SupplierUsers WHERE HomeNodeId = @wh_id AND HomeNodeType = "WAREHOUSE"`,
				Params: map[string]any{"wh_id": whID},
			}
			iter := s.spannerClient.Single().Query(r.Context(), stmt)
			defer iter.Stop()
			var staff []portalStaff
			for {
				row, err := iter.Next()
				if err != nil {
					break
				}
				var st portalStaff
				if err := row.Columns(&st.StaffID, &st.Name, &st.Role); err == nil {
					staff = append(staff, st)
				}
			}
			if len(staff) > 0 {
				writeJSON(w, http.StatusOK, map[string]any{"staff": staff})
				return
			}
		}

		s.mu.RLock()
		staff := append([]portalStaff(nil), s.staff...)
		s.mu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"staff": staff})
		return
	}
	if r.Method == http.MethodPost {
		var req struct {
			Name  string `json:"name"`
			Phone string `json:"phone"`
			Role  string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		defer r.Body.Close()
		staffID := "stf-" + uuid.NewString()[:8]
		
		if s.spannerClient != nil {
			now := s.now().UTC()
			m := spanner.Insert("SupplierUsers",
				[]string{"UserId", "SupplierId", "Phone", "Name", "PasswordHash", "SupplierRole", "AssignedWarehouseId", "IsActive", "CreatedAt", "UpdatedAt"},
				[]any{staffID, s.supplierID, strings.TrimSpace(req.Phone), strings.TrimSpace(req.Name), "password_hash", strings.TrimSpace(req.Role), warehouseIDFromRequest(r), true, now, now},
			)
			if _, err := s.spannerClient.Apply(r.Context(), []*spanner.Mutation{m}); err != nil {
				s.log.ErrorContext(r.Context(), "failed to create staff", "err", err)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed_to_create_staff"})
				return
			}
		} else {
			row := portalStaff{StaffID: staffID, Name: req.Name, Phone: req.Phone, Role: req.Role}
			s.mu.Lock()
			s.staff = append(s.staff, row)
			s.mu.Unlock()
		}
		
		writeJSON(w, http.StatusCreated, map[string]any{"staff_id": staffID, "pin": "5678"})
		return
	}
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
}

func (s *Service) HandleOpsProducts(w http.ResponseWriter, r *http.Request) {
	s.ensurePortalSeed()
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	s.mu.RLock()
	products := append([]portalProduct(nil), s.products...)
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{"products": products})
}

func (s *Service) HandleOpsManifests(w http.ResponseWriter, r *http.Request) {
	s.ensurePortalSeed()
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	s.mu.RLock()
	manifests := append([]portalManifest(nil), s.manifests...)
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{"manifests": manifests})
}

func (s *Service) HandleOpsCRM(w http.ResponseWriter, r *http.Request) {
	s.ensurePortalSeed()
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	s.mu.RLock()
	retailers := append([]portalRetailer(nil), s.retailers...)
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{"retailers": retailers})
}

func (s *Service) HandleOpsReturns(w http.ResponseWriter, r *http.Request) {
	s.ensurePortalSeed()
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	s.mu.RLock()
	returns := append([]portalReturn(nil), s.returns...)
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{"returns": returns})
}

func (s *Service) HandleOpsAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	whID := warehouseIDFromRequest(r)
	period := strings.TrimSpace(r.URL.Query().Get("period"))
	if period == "" {
		period = "7d"
	}

	var totalOrders, completedOrders, cancelledOrders int64
	var totalRevenue int64
	if s.analyticsQuery != nil {
		counts, err := s.analyticsQuery(r.Context(), whID)
		if err == nil {
			totalOrders = counts.TotalOrders
			completedOrders = counts.CompletedOrders
			cancelledOrders = counts.CancelledOrders
			totalRevenue = counts.TotalRevenue
		} else {
			s.log.WarnContext(r.Context(), "warehouse analytics query failed", "err", err, "warehouse_id", whID)
		}
	}

	var avgOrderValue float64
	if totalOrders > 0 {
		avgOrderValue = float64(totalRevenue) / float64(totalOrders)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"warehouse_id":         whID,
		"period":               period,
		"total_orders":         totalOrders,
		"total_revenue":        totalRevenue,
		"completed_orders":     completedOrders,
		"cancelled_orders":     cancelledOrders,
		"avg_order_value":      avgOrderValue,
		"top_products":         []any{},
		"daily_breakdown":      []any{},
		"fleet_utilization":    map[string]any{"utilization_pct": 0},
		"import_freshness":     map[string]any{"applied_rows_30d": 0, "applied_skus_30d": 0, "quantity_delta_30d": 0},
		"import_anomaly_queue": map[string]any{"open_rows_30d": 0},
	})
}

func (s *Service) HandleOpsTreasury(w http.ResponseWriter, r *http.Request) {
	s.ensurePortalSeed()
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	view := strings.TrimSpace(r.URL.Query().Get("view"))
	if view == "invoices" {
		writeJSON(w, http.StatusOK, map[string]any{
			"invoices": []map[string]any{
				{"invoice_id": "inv-1", "status": "PAID", "amount_uzs": 45000000, "payout_uzs": 42750000},
			},
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"invoiced_uzs":    120000000,
		"paid_uzs":        98000000,
		"outstanding_uzs": 22000000,
	})
}

func (s *Service) HandleOpsPaymentConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"gateways": []map[string]any{
			{"provider": "GLOBAL_PAY", "mode": "SANDBOX", "is_active": true},
			{"provider": "CASH", "mode": "LIVE", "is_active": true},
		},
	})
}

func (s *Service) HandleSupplyRequestByID(w http.ResponseWriter, r *http.Request) {
	warehouseID := warehouseIDFromRequest(r)
	id := strings.TrimPrefix(r.URL.Path, "/v1/warehouse/supply-requests/")
	id = strings.Trim(id, "/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request_id_required"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		if s.repo == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "request_not_found"})
			return
		}
		rows, err := s.repo.ListSupplyRequests(r.Context(), warehouseID, 200)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supply_requests_failed"})
			return
		}
		for _, req := range rows {
			if req.RequestID == id {
				writeJSON(w, http.StatusOK, supplyRequestIOSPayload(req))
				return
			}
		}
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "request_not_found"})
	case http.MethodPatch:
		var body struct {
			Action string `json:"action"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		defer r.Body.Close()
		if !strings.EqualFold(body.Action, "CANCEL") {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "unsupported_action"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"request_id": id, "state": "CANCELLED"})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func supplyRequestIOSPayload(req SupplyRequest) map[string]any {
	state := strings.TrimSpace(req.State)
	if state == "" {
		state = strings.TrimSpace(req.Status)
	}
	return map[string]any{
		"request_id":      req.RequestID,
		"warehouse_id":    req.WarehouseID,
		"factory_id":      "fc-demo-1",
		"supplier_id":     req.SupplierID,
		"state":           state,
		"priority":        "NORMAL",
		"total_volume_vu": 0,
		"notes":           "",
		"created_by":      req.RequestedBy,
		"created_at":      req.CreatedAt,
		"updated_at":      req.UpdatedAt,
	}
}
