package warehouse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/manifest"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/pkg/httppagination"
	"google.golang.org/api/iterator"
)

// warehouseDispatchExecuteManifestState is written on execute; payloader seals before depart.
const warehouseDispatchExecuteManifestState = "DRAFT"

// PortalDriver is the warehouse ops driver read model.
type PortalDriver struct {
	DriverID                 string  `json:"driver_id"`
	Name                     string  `json:"name"`
	Phone                    string  `json:"phone"`
	TruckStatus              string  `json:"truck_status"`
	IsActive                 bool    `json:"is_active"`
	OnShift                  bool    `json:"on_shift,omitempty"`
	UnavailableReason        string  `json:"unavailable_reason,omitempty"`
	VehicleID                string  `json:"vehicle_id,omitempty"`
	VehicleLabel             string  `json:"vehicle_label,omitempty"`
	VehicleClass             string  `json:"vehicle_class,omitempty"`
	MaxVolumeVU              float64 `json:"max_volume_vu,omitempty"`
	VehicleIsActive          bool    `json:"vehicle_is_active,omitempty"`
	VehicleUnavailableReason string  `json:"vehicle_unavailable_reason,omitempty"`
	VehicleUnavailableNote   string  `json:"vehicle_unavailable_note,omitempty"`
	UnavailableNote          string  `json:"unavailable_note,omitempty"`
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
	UnavailableNote    string  `json:"unavailable_note,omitempty"`
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
	RetailerID    string `json:"retailer_id"`
	BusinessName  string `json:"business_name"`
	TotalOrders   int64  `json:"total_orders"`
	TotalRevenue  int64  `json:"total_revenue"`
	LastOrderDate string `json:"last_order_date,omitempty"`
}

type portalReturnItem struct {
	LineItemID  string `json:"line_item_id"`
	OrderID     string `json:"order_id"`
	ProductName string `json:"product_name"`
	Quantity    int    `json:"quantity"`
	Status      string `json:"status"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

type portalOrder struct {
	OrderID      string `json:"order_id"`
	RetailerName string `json:"retailer_name"`
	State        string `json:"state"`
	TotalUZS     int    `json:"total_uzs"`
}

func (s *Service) ensurePortalSeed() {
	if !s.portalSeedEnabled() {
		return
	}
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
		{RetailerID: "ret-1", BusinessName: "Corner Shop 12", TotalOrders: 42, TotalRevenue: 128000000, LastOrderDate: now},
		{RetailerID: "ret-2", BusinessName: "Family Market", TotalOrders: 18, TotalRevenue: 56000000, LastOrderDate: now},
	}
	s.returns = []portalReturnItem{{LineItemID: "retn-1", OrderID: "ord-wh-1", ProductName: "Mineral Water 1.5L", Quantity: 2, Status: "PENDING", UpdatedAt: now}}
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

	if s.repo != nil && whID != "" {
		inventoryList, _ := s.repo.GetInventoryList(r.Context(), whID)
		for _, row := range inventoryList {
			if row.Quantity < 20 {
				lowStock++
			}
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
	whID := warehouseIDFromRequest(r)
	if whID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_scope_required"})
		return
	}
	if s.repo == nil {
		s.ensurePortalSeed()
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "inventory_unavailable"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		lowOnly := strings.EqualFold(r.URL.Query().Get("low_stock"), "true")
		inventoryList, err := s.repo.GetInventoryList(r.Context(), whID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed_to_fetch_inventory"})
			return
		}
		items := make([]map[string]any, 0, len(inventoryList))
		for sku, row := range inventoryList {
			qty := row.Quantity
			threshold := row.ReorderThreshold
			if threshold <= 0 {
				threshold = 20
			}
			isLow := qty <= threshold
			if lowOnly && !isLow {
				continue
			}
			items = append(items, map[string]any{
				"product_id":         sku,
				"sku_id":             row.SKU,
				"product_name":       row.ProductName,
				"quantity":           qty,
				"quantity_on_hand":   row.QuantityOnHand,
				"reorder_threshold":  threshold,
				"out_of_stock_policy": row.OutOfStockPolicy,
				"effective_policy":   row.EffectivePolicy,
				"accepts_backorder":  row.EffectivePolicy == OutOfStockPolicyAcceptBackorder,
				"is_low_stock":       isLow,
				"last_updated":       row.UpdatedAt,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"inventory": items, "items": items})
	case http.MethodPatch:
		body, ok := readMutationBody(w, r, 64*1024)
		if !ok {
			return
		}
		key, handled := s.guardMutationReplay(w, r, body)
		if handled {
			return
		}
		idemCommitted := false
		defer func() {
			if !idemCommitted {
				s.releaseMutationReplay(r.Context(), key)
			}
		}()

		var patchBody struct {
			ProductID string `json:"product_id"`
			Quantity  int64  `json:"quantity"`
		}
		if err := json.Unmarshal(body, &patchBody); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		err := s.repo.UpdateInventoryQuantity(r.Context(), whID, patchBody.ProductID, patchBody.Quantity, func(buf outbox.TxnBuffer) error {
			return nil
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed_to_update_inventory"})
			return
		}
		resp := map[string]string{"status": "updated"}
		respBytes, _ := json.Marshal(resp)
		s.storeMutationReplay(r.Context(), key, body, http.StatusOK, respBytes)
		idemCommitted = true
		writeJSON(w, http.StatusOK, resp)
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
	whID := warehouseIDFromRequest(r)
	if detail, ok := s.loadOpsOrderDetailFromSpanner(r.Context(), whID, orderID); ok {
		writeJSON(w, http.StatusOK, detail)
		return
	}
	if s.opsOrders != nil {
		rows, err := s.opsOrders(r.Context(), whID, 200)
		if err == nil {
			for _, row := range rows {
				if row.OrderID != orderID {
					continue
				}
				writeJSON(w, http.StatusOK, map[string]any{
					"order_id":      row.OrderID,
					"retailer_name": "Retailer " + row.RetailerID,
					"state":         strings.ToUpper(row.Status),
					"total_uzs":     int(row.TotalMinor / 100),
					"line_items":    []map[string]any{},
				})
				return
			}
		}
	}
	s.ensurePortalSeed()
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

func (s *Service) loadOpsOrderDetailFromSpanner(ctx context.Context, warehouseID, orderID string) (map[string]any, bool) {
	if s.spannerClient == nil || strings.TrimSpace(orderID) == "" {
		return nil, false
	}
	stmt := spanner.Statement{
		SQL: `SELECT o.OrderId, o.RetailerId, o.Status, o.TotalMinor, o.LineItemsJson, COALESCE(r.Name, '')
		      FROM Orders o
		      LEFT JOIN Retailers r ON o.RetailerId = r.RetailerId
		      WHERE o.OrderId = @orderId
		        AND (@warehouseId = '' OR o.WarehouseId = @warehouseId)
		      LIMIT 1`,
		Params: map[string]any{
			"orderId":     orderID,
			"warehouseId": strings.TrimSpace(warehouseID),
		},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return nil, false
	}
	if err != nil {
		s.log.WarnContext(ctx, "ops order detail query failed", "order_id", orderID, "err", err)
		return nil, false
	}

	var (
		retailerID   string
		status       string
		totalMinor   int64
		lineItemsRaw []byte
		retailerName string
	)
	if err := row.Columns(&orderID, &retailerID, &status, &totalMinor, &lineItemsRaw, &retailerName); err != nil {
		s.log.WarnContext(ctx, "ops order detail scan failed", "order_id", orderID, "err", err)
		return nil, false
	}
	if strings.TrimSpace(retailerName) == "" {
		retailerName = "Retailer " + retailerID
	}

	lineItems := make([]map[string]any, 0)
	if len(lineItemsRaw) > 0 {
		var parsed []order.LineItem
		if err := json.Unmarshal(lineItemsRaw, &parsed); err == nil {
			for _, item := range parsed {
				lineItems = append(lineItems, map[string]any{
					"product_id":   item.SKU,
					"product_name": item.Name,
					"quantity":     item.Quantity,
					"unit_price":   int(item.UnitPrice / 100),
				})
			}
		}
	}

	return map[string]any{
		"order_id":      orderID,
		"retailer_name": retailerName,
		"state":         strings.ToUpper(status),
		"total_uzs":     int(totalMinor / 100),
		"line_items":    lineItems,
	}, true
}

func (s *Service) handleOpsDispatchPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	whID := warehouseIDFromRequest(r)
	limit, offset := httppagination.ParseLimitOffset(r, 300, 5000)

	var previewBody struct {
		OrderIDs []string `json:"order_ids"`
	}
	if r.Method == http.MethodPost && r.Body != nil {
		body, _ := io.ReadAll(io.LimitReader(r.Body, 16*1024))
		_ = json.Unmarshal(body, &previewBody)
	}

	undispatched := make([]map[string]any, 0)
	windowConstrained := 0
	dispatchRows := make([]dispatch.DispatchableOrder, 0)
	if s.spannerClient != nil && strings.TrimSpace(s.supplierID) != "" {
		repo := dispatch.NewRepository(s.spannerClient)
		rows, err := repo.FetchDispatchable(r.Context(), dispatch.FetchParams{
			SupplierID:  s.supplierID,
			WarehouseID: whID,
			Limit:       limit,
			Offset:      offset,
		})
		if err == nil {
			dispatchRows = rows
			dispatchRows = filterDispatchRowsByOrderIDs(dispatchRows, previewBody.OrderIDs)
			preview := dispatch.BuildPreview(dispatchRows)
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
	sid := s.resolveDispatchSupplierID(r.Context(), whID)
	if s.opsDrivers != nil {
		drivers, err := s.opsDrivers(r.Context(), whID)
		if err == nil {
			solveDrivers = drivers
		}
	} else {
		s.ensurePortalSeed()
		s.mu.RLock()
		solveDrivers = append([]PortalDriver(nil), s.drivers...)
		s.mu.RUnlock()
	}
	fleetCtx := fleetDispatchContext{InTransit: map[string]bool{}, TopOff: map[string]manifest.DriverManifestCapacity{}}
	if len(solveDrivers) > 0 {
		var err error
		fleetCtx, err = s.loadFleetDispatchContext(r.Context(), sid, whID, collectWarehouseDriverIDs(solveDrivers))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "dispatch_preview_failed"})
			return
		}
	}
	dispatchRows = filterDispatchRowsByOrderIDs(dispatchRows, previewBody.OrderIDs)
	for _, d := range solveDrivers {
		entry := driverPreviewEntry(d, fleetCtx)
		truckStatus, isUnavailable, reason := warehouseDriverAvailability(d, fleetCtx)
		entry["truck_status"] = truckStatus
		if isUnavailable {
			entry["unavailable_reason"] = reason
			unavailable = append(unavailable, entry)
		} else {
			available = append(available, entry)
		}
	}

	response := map[string]any{
		"preview_ready":              true,
		"undispatched_orders":        undispatched,
		"available_drivers":          available,
		"unavailable_drivers":        unavailable,
		"window_constrained_count":   windowConstrained,
		"fleet_effective_capacity_vu": fleetEffectiveCapacityVU(solveDrivers, fleetCtx),
	}
	if len(previewBody.OrderIDs) > 0 {
		response["selected_orders_volume_vu"] = sumOrderVolumeVU(dispatchRows)
	}
	if len(dispatchRows) > 0 && len(solveDrivers) > 0 {
		planMeta, _ := s.solveDispatchPreview(r.Context(), whID, dispatchRows, fleetCtx, solveDrivers, previewBody.OrderIDs)
		for k, v := range planMeta {
			response[k] = v
		}
	}
	w.Header().Set("X-Page-Limit", strconv.Itoa(limit))
	w.Header().Set("X-Page-Offset", strconv.Itoa(offset))
	w.Header().Set("X-Page-Has-More", strconv.FormatBool(len(undispatched) == limit))
	writeJSON(w, http.StatusOK, response)
}

type DispatchExecuteResult struct {
	Status           string                      `json:"status"`
	SupplierID       string                      `json:"supplier_id"`
	WarehouseID      string                      `json:"warehouse_id,omitempty"`
	ManifestsCreated int                         `json:"manifests_created"`
	OrdersAssigned   int                         `json:"orders_assigned"`
	OptimizerSource  string                      `json:"optimizer_source,omitempty"`
	Warnings         []string                    `json:"warnings,omitempty"`
	CapacityWarnings []DispatchCapacityWarning   `json:"capacity_warnings,omitempty"`
	Manifests        []DispatchExecuteRoute      `json:"manifests"`
	Orphans          []string                    `json:"orphan_order_ids,omitempty"`
}

type DispatchCapacityWarning struct {
	DriverID                  string   `json:"driver_id"`
	LoadedVU                  float64  `json:"loaded_vu"`
	MaxVolumeVU               float64  `json:"max_volume_vu"`
	EffectiveMaxVU            float64  `json:"effective_max_vu"`
	ExcessVU                  float64  `json:"excess_vu,omitempty"`
	SuggestedUnselectOrderIDs []string `json:"suggested_unselect_order_ids,omitempty"`
	SuggestedDeferOrderIDs    []string `json:"suggested_defer_order_ids,omitempty"`
	FleetEffectiveCapacityVU  float64  `json:"fleet_effective_capacity_vu,omitempty"`
	RequestedVolumeVU         float64  `json:"requested_volume_vu,omitempty"`
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
	if !s.requireMutationIdempotencyKey(w, r) {
		return
	}
	if s.spannerClient == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "dispatch_unavailable"})
		return
	}
	body, ok := readMutationBody(w, r, 64*1024)
	if !ok {
		return
	}
	key, handled := s.guardMutationReplay(w, r, body)
	if handled {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseMutationReplay(r.Context(), key)
		}
	}()

	whID := warehouseIDFromRequest(r)
	sid := s.resolveDispatchSupplierID(r.Context(), whID)

	var req struct {
		Mode          string                 `json:"mode"`
		Routes        []DispatchExecuteRoute `json:"routes"`
		OrderIDs      []string               `json:"order_ids"`
		ForceCapacity bool                   `json:"force_capacity"`
		AcceptPartial bool                   `json:"accept_partial"`
		PlanFingerprint string               `json:"plan_fingerprint"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	out, err := s.ExecuteDispatch(r.Context(), DispatchExecuteRequest{
		WarehouseID:     whID,
		SupplierID:      sid,
		Mode:            req.Mode,
		Routes:          req.Routes,
		OrderIDs:        req.OrderIDs,
		ForceCapacity:   req.ForceCapacity,
		AcceptPartial:   req.AcceptPartial,
		PlanFingerprint: req.PlanFingerprint,
	})
	if err != nil {
		s.log.ErrorContext(r.Context(), "dispatch execute failed", "warehouse_id", whID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "dispatch_execute_failed"})
		return
	}

	if out.Status == "dispatched" {
		s.log.InfoContext(r.Context(), "dispatch executed",
			"warehouse_id", whID,
			"manifests", out.ManifestsCreated,
			"orders_assigned", out.OrdersAssigned,
			"optimizer_source", out.OptimizerSource,
		)
		s.broadcastWarehouseEvent(r.Context(), whID, map[string]any{
			"type":              "DISPATCH_COMMITTED",
			"trace_id":          outbox.TraceIDFromContext(r.Context()),
			"warehouse_id":      whID,
			"manifests_created": out.ManifestsCreated,
			"orders_assigned":   out.OrdersAssigned,
			"optimizer_source":  out.OptimizerSource,
			"timestamp":         s.now().UTC().Format(time.RFC3339Nano),
		})
	}
	if encoded, err := json.Marshal(out); err == nil {
		s.storeMutationReplay(r.Context(), key, body, http.StatusOK, encoded)
		idemCommitted = true
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Service) handleOpsDispatchSettings(w http.ResponseWriter, r *http.Request) {
	whID := warehouseIDFromRequest(r)
	if whID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "warehouse_id_required"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		enabled, err := s.repo.GetAutoDispatch(r.Context(), whID)
		if err != nil {
			s.log.ErrorContext(r.Context(), "failed to get auto_dispatch_enabled", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "fetch_failed"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"warehouse_id":          whID,
			"auto_dispatch_enabled": enabled,
		})

	case http.MethodPatch:
		var payload struct {
			AutoDispatchEnabled *bool `json:"auto_dispatch_enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		defer r.Body.Close()

		if payload.AutoDispatchEnabled != nil {
			err := s.repo.UpdateAutoDispatch(r.Context(), whID, *payload.AutoDispatchEnabled, func(buf outbox.TxnBuffer) error {
				eventPayload := events.WarehouseEvent{
					BaseEvent:   events.BaseEvent{Type: "WAREHOUSE_DISPATCH_SETTINGS_UPDATED"},
					WarehouseID: whID,
					SupplierID:  s.supplierID,
				}
				return outbox.EmitJSON(r.Context(), buf, events.AggregateWarehouse, whID, events.TopicMain, eventPayload)
			})
			if err != nil {
				s.log.ErrorContext(r.Context(), "failed to update auto_dispatch_enabled", "err", err)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update_failed"})
				return
			}
			s.broadcastWarehouseEvent(r.Context(), whID, map[string]any{
				"type":                  "WAREHOUSE_DISPATCH_SETTINGS_UPDATED",
				"warehouse_id":          whID,
				"auto_dispatch_enabled": *payload.AutoDispatchEnabled,
			})
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
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
		body, ok := readMutationBody(w, r, 64*1024)
		if !ok {
			return
		}
		key, handled := s.guardMutationReplay(w, r, body)
		if handled {
			return
		}
		idemCommitted := false
		defer func() {
			if !idemCommitted {
				s.releaseMutationReplay(r.Context(), key)
			}
		}()

		var req struct {
			Name  string `json:"name"`
			Phone string `json:"phone"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		driverID := "drv-" + uuid.NewString()[:8]

		if s.spannerClient != nil {
			now := s.now().UTC()
			whID := warehouseIDFromRequest(r)
			if err := s.createOpsDriverSpanner(r.Context(), opsDriverCreateParams{
				DriverID:    driverID,
				Name:        strings.TrimSpace(req.Name),
				Phone:       strings.TrimSpace(req.Phone),
				PinHash:     "4321",
				SupplierID:  strings.TrimSpace(s.supplierID),
				WarehouseID: whID,
				CreatedAt:   now,
			}); err != nil {
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

		resp := map[string]any{
			"driver_id": driverID,
			"pin":       "4321",
		}
		respBytes, _ := json.Marshal(resp)
		s.storeMutationReplay(r.Context(), key, body, http.StatusCreated, respBytes)
		idemCommitted = true
		writeJSON(w, http.StatusCreated, resp)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) handleAssignVehicle(w http.ResponseWriter, r *http.Request, driverID string) {
	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	body, ok := readMutationBody(w, r, 64*1024)
	if !ok {
		return
	}
	key, handled := s.guardMutationReplay(w, r, body)
	if handled {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseMutationReplay(r.Context(), key)
		}
	}()

	var req struct {
		VehicleID string `json:"vehicle_id"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	vehicleID := strings.TrimSpace(req.VehicleID)
	whID := warehouseIDFromRequest(r)
	if s.spannerClient != nil {
		if err := s.persistDriverVehicleAssignment(r.Context(), whID, driverID, vehicleID); err != nil {
			var fleetErr *FleetMutationError
			if errors.As(err, &fleetErr) {
				writeJSON(w, fleetErr.StatusCode, map[string]string{"error": fleetErr.Code, "message": fleetErr.Message})
				return
			}
			if strings.Contains(err.Error(), "driver_not_found") {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "driver_not_found"})
				return
			}
			s.log.ErrorContext(r.Context(), "assign vehicle failed", "driver_id", driverID, "vehicle_id", vehicleID, "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "assign_vehicle_failed"})
			return
		}
		resp := map[string]string{"status": "assigned", "driver_id": driverID, "vehicle_id": vehicleID}
		respBytes, _ := json.Marshal(resp)
		s.storeMutationReplay(r.Context(), key, body, http.StatusOK, respBytes)
		idemCommitted = true
		writeJSON(w, http.StatusOK, resp)
		return
	}

	s.ensurePortalSeed()
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.drivers {
		if s.drivers[i].DriverID != driverID {
			continue
		}
		s.drivers[i].VehicleID = vehicleID
		resp := map[string]string{"status": "assigned", "driver_id": driverID, "vehicle_id": vehicleID}
		respBytes, _ := json.Marshal(resp)
		s.storeMutationReplay(r.Context(), key, body, http.StatusOK, respBytes)
		idemCommitted = true
		writeJSON(w, http.StatusOK, resp)
		return
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "driver_not_found"})
}

func (s *Service) persistDriverVehicleAssignment(ctx context.Context, warehouseID, driverID, vehicleID string) error {
	if s.spannerClient == nil {
		return fmt.Errorf("spanner_not_configured")
	}
	sid := strings.TrimSpace(s.supplierID)
	now := s.now().UTC()
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		state, err := readDriverAssignmentState(ctx, txn, sid, warehouseID, driverID)
		if err != nil {
			if errors.Is(err, errDriverNotFound) {
				return fmt.Errorf("driver_not_found")
			}
			return err
		}
		activeDriverOrders, err := countActiveOrdersForDriver(ctx, txn, driverID)
		if err != nil {
			return err
		}
		if err := driverAssignmentGuard(state, activeDriverOrders); err != nil {
			return err
		}
		if vehicleID != "" {
			if activeVehicleOrders, err := countActiveOrdersForVehicle(ctx, txn, vehicleID); err != nil {
				return err
			} else if activeVehicleOrders > 0 {
				return &FleetMutationError{
					StatusCode: http.StatusConflict,
					Code:       "vehicle_active_orders",
					Message:    fmt.Sprintf("vehicle %s has active orders and cannot be reassigned", vehicleID),
				}
			}
			conflict, err := readDriverByVehicle(ctx, txn, sid, warehouseID, vehicleID, driverID)
			if err != nil {
				return err
			}
			if conflict != nil {
				conflictOrders, err := countActiveOrdersForDriver(ctx, txn, conflict.DriverID)
				if err != nil {
					return err
				}
				if guardErr := driverAssignmentGuard(*conflict, conflictOrders); guardErr != nil {
					return &FleetMutationError{
						StatusCode: http.StatusConflict,
						Code:       "vehicle_driver_active",
						Message:    fmt.Sprintf("vehicle %s is assigned to active driver %s", vehicleID, conflict.DriverID),
					}
				}
			}
		}

		homeNodeID := state.HomeNodeID
		if homeNodeID == "" {
			homeNodeID = warehouseID
		}

		mutations := []*spanner.Mutation{
			spanner.UpdateMap("Drivers", map[string]any{
				"DriverId":  driverID,
				"VehicleId": nullableWarehouseString(vehicleID),
				"UpdatedAt": now,
			}),
		}
		if vehicleID != "" {
			clearStmt := spanner.Statement{
				SQL: `SELECT DriverId FROM Drivers@{FORCE_INDEX=Idx_Drivers_ByHomeNode}
				      WHERE HomeNodeType = 'WAREHOUSE' AND HomeNodeId = @wid AND VehicleId = @vid AND DriverId != @driverId`,
				Params: map[string]any{"wid": homeNodeID, "vid": vehicleID, "driverId": driverID},
			}
			iter := txn.Query(ctx, clearStmt)
			defer iter.Stop()
			for {
				row, err := iter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					return err
				}
				var otherDriverID string
				if err := row.Columns(&otherDriverID); err != nil {
					return err
				}
				mutations = append(mutations, spanner.UpdateMap("Drivers", map[string]any{
					"DriverId":  otherDriverID,
					"VehicleId": spanner.NullString{},
					"UpdatedAt": now,
				}))
			}
		}
		return txn.BufferWrite(mutations)
	})
	return err
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
		body, ok := readMutationBody(w, r, 64*1024)
		if !ok {
			return
		}
		key, handled := s.guardMutationReplay(w, r, body)
		if handled {
			return
		}
		idemCommitted := false
		defer func() {
			if !idemCommitted {
				s.releaseMutationReplay(r.Context(), key)
			}
		}()

		var req struct {
			Label        string  `json:"label"`
			LicensePlate string  `json:"license_plate"`
			VehicleClass string  `json:"vehicle_class"`
			MaxVolumeVU  float64 `json:"max_volume_vu"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		vehicleID := "veh-" + uuid.NewString()[:8]

		if s.spannerClient != nil {
			now := s.now().UTC()
			whID := warehouseIDFromRequest(r)
			maxVU := resolveVehicleMaxVU(req.VehicleClass, req.MaxVolumeVU)
			if err := s.createOpsVehicleSpanner(r.Context(), opsVehicleCreateParams{
				VehicleID:    vehicleID,
				Label:        strings.TrimSpace(req.Label),
				LicensePlate: strings.TrimSpace(req.LicensePlate),
				VehicleClass: strings.TrimSpace(req.VehicleClass),
				MaxVolumeVU:  maxVU,
				SupplierID:   strings.TrimSpace(s.supplierID),
				WarehouseID:  whID,
				CreatedAt:    now,
			}); err != nil {
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

		resp := map[string]any{
			"vehicle_id":    vehicleID,
			"label":         req.Label,
			"license_plate": req.LicensePlate,
			"vehicle_class": req.VehicleClass,
			"max_volume_vu": resolveVehicleMaxVU(req.VehicleClass, req.MaxVolumeVU),
			"is_active":     true,
		}
		respBytes, _ := json.Marshal(resp)
		s.storeMutationReplay(r.Context(), key, body, http.StatusCreated, respBytes)
		idemCommitted = true
		writeJSON(w, http.StatusCreated, resp)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) handleOpsVehiclePatch(w http.ResponseWriter, r *http.Request, vehicleID string) {
	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	body, ok := readMutationBody(w, r, 64*1024)
	if !ok {
		return
	}
	key, handled := s.guardMutationReplay(w, r, body)
	if handled {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseMutationReplay(r.Context(), key)
		}
	}()

	var req struct {
		IsActive          bool   `json:"is_active"`
		UnavailableReason string `json:"unavailable_reason"`
		UnavailableNote   string `json:"unavailable_note"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	whID := warehouseIDFromRequest(r)
	if whID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_scope_required"})
		return
	}
	now := time.Now().UTC()
	if s.spannerClient != nil {
		if err := s.patchOpsVehicleSpanner(r.Context(), opsVehiclePatchParams{
			VehicleID:         vehicleID,
			WarehouseID:       whID,
			SupplierID:        s.supplierID,
			IsActive:          req.IsActive,
			UnavailableReason: req.UnavailableReason,
			UnavailableNote:   req.UnavailableNote,
			UpdatedAt:         now,
		}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "vehicle_update_failed"})
			return
		}
		idemCommitted = true
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated", "vehicle_id": vehicleID})
		return
	}

	s.ensurePortalSeed()
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.vehicles {
		if s.vehicles[i].VehicleID != vehicleID {
			continue
		}
		s.vehicles[i].IsActive = req.IsActive
		reason := ""
		note := ""
		if !req.IsActive {
			reason = normalizeWarehouseVehicleReason(req.UnavailableReason)
			note = strings.TrimSpace(req.UnavailableNote)
			if reason == VehicleReasonOther && note == "" {
				note = strings.TrimSpace(req.UnavailableReason)
			}
		}
		s.vehicles[i].UnavailableReason = reason
		s.vehicles[i].UnavailableNote = note
		idemCommitted = true
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
				SQL: `SELECT UserId, Name, Phone, SupplierRole
				      FROM SupplierUsers
				      WHERE AssignedWarehouseId = @wh_id
				        AND IsActive = true
				        AND SupplierRole IN ('WAREHOUSE', 'WAREHOUSE_ADMIN', 'WAREHOUSE_STAFF', 'PAYLOADER')`,
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
				if err := row.Columns(&st.StaffID, &st.Name, &st.Phone, &st.Role); err == nil {
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
		body, ok := readMutationBody(w, r, 64*1024)
		if !ok {
			return
		}
		key, handled := s.guardMutationReplay(w, r, body)
		if handled {
			return
		}
		idemCommitted := false
		defer func() {
			if !idemCommitted {
				s.releaseMutationReplay(r.Context(), key)
			}
		}()

		var req struct {
			Name  string `json:"name"`
			Phone string `json:"phone"`
			Role  string `json:"role"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
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

		resp := map[string]any{"staff_id": staffID, "pin": "5678"}
		respBytes, _ := json.Marshal(resp)
		s.storeMutationReplay(r.Context(), key, body, http.StatusCreated, respBytes)
		idemCommitted = true
		writeJSON(w, http.StatusCreated, resp)
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
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	whID := warehouseIDFromRequest(r)
	if s.spannerClient != nil && !s.portalSeedEnabled() {
		retailers, err := s.loadWarehouseCRMFromSpanner(r.Context(), whID)
		if err != nil {
			s.log.WarnContext(r.Context(), "warehouse crm query failed", "err", err, "warehouse_id", whID)
			writeJSON(w, http.StatusOK, map[string]any{"retailers": []portalRetailer{}})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"retailers": retailers})
		return
	}
	s.ensurePortalSeed()
	s.mu.RLock()
	retailers := append([]portalRetailer(nil), s.retailers...)
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{"retailers": retailers})
}

func (s *Service) HandleOpsReturns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if s.spannerClient != nil && !s.portalSeedEnabled() {
		writeJSON(w, http.StatusOK, map[string]any{"items": []portalReturnItem{}})
		return
	}
	s.ensurePortalSeed()
	s.mu.RLock()
	items := append([]portalReturnItem(nil), s.returns...)
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
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

	if s.spannerClient != nil {
		supplierID := s.resolveAnalyticsSupplierID(r)
		payload, err := s.loadOpsAnalytics(r.Context(), whID, period, supplierID)
		if err == nil {
			writeJSON(w, http.StatusOK, payload)
			return
		}
		if s.log != nil {
			s.log.WarnContext(r.Context(), "warehouse analytics spanner read failed, using fallback", "warehouse_id", whID, "err", err)
		}
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
		} else if s.log != nil {
			s.log.WarnContext(r.Context(), "warehouse analytics query failed", "err", err, "warehouse_id", whID)
		}
	}

	var avgOrderValue float64
	if completedOrders > 0 {
		avgOrderValue = float64(totalRevenue) / float64(completedOrders)
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
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	whID := warehouseIDFromRequest(r)
	view := strings.TrimSpace(r.URL.Query().Get("view"))
	if view == "" {
		view = "overview"
	}
	if s.spannerClient != nil && !s.portalSeedEnabled() {
		if view == "invoices" {
			writeJSON(w, http.StatusOK, map[string]any{"invoices": []any{}})
			return
		}
		writeJSON(w, http.StatusOK, s.loadWarehouseTreasuryOverview(r.Context(), whID))
		return
	}
	s.ensurePortalSeed()
	if view == "invoices" {
		writeJSON(w, http.StatusOK, map[string]any{
			"invoices": []map[string]any{
				{"invoice_id": "inv-1", "status": "PAID", "amount_uzs": 45000000, "payout_uzs": 42750000},
			},
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"total_invoiced":    int64(120000000),
		"total_paid":        int64(98000000),
		"total_outstanding": int64(22000000),
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
	factoryID := strings.TrimSpace(req.FactoryID)
	priority := strings.TrimSpace(req.Priority)
	if priority == "" {
		priority = "NORMAL"
	}
	totalVU := req.TotalVolumeVU
	if totalVU <= 0 && req.ProjectedUnits > 0 {
		totalVU = float64(req.ProjectedUnits)
	}
	items := make([]map[string]any, 0, len(req.Items))
	for _, item := range req.Items {
		items = append(items, map[string]any{
			"item_id":             item.ItemID,
			"product_id":          item.ProductID,
			"requested_quantity":  item.RequestedQuantity,
			"recommended_qty":     item.RecommendedQty,
			"unit_volume_vu":      item.UnitVolumeVU,
		})
	}
	payload := map[string]any{
		"request_id":              req.RequestID,
		"warehouse_id":            req.WarehouseID,
		"factory_id":              factoryID,
		"supplier_id":             req.SupplierID,
		"state":                   state,
		"priority":                priority,
		"notes":                   req.Notes,
		"region_id":               req.RegionID,
		"total_volume_vu":         totalVU,
		"projected_units":         req.ProjectedUnits,
		"item_count":              len(req.Items),
		"items":                   items,
		"transfer_order_id":       strings.TrimSpace(req.LinkedTransferID),
		"created_by":              req.RequestedBy,
		"created_at":              req.CreatedAt,
		"updated_at":              req.UpdatedAt,
	}
	if strings.TrimSpace(req.RequestedDeliveryDate) != "" {
		payload["requested_delivery_date"] = req.RequestedDeliveryDate
	}
	return payload
}

func (s *Service) loadWarehouseCRMFromSpanner(ctx context.Context, warehouseID string) ([]portalRetailer, error) {
	if s.spannerClient == nil {
		return nil, fmt.Errorf("spanner_not_configured")
	}
	warehouseID = strings.TrimSpace(warehouseID)
	if warehouseID == "" {
		return []portalRetailer{}, nil
	}

	stmt := spanner.Statement{
		SQL: `SELECT o.RetailerId, COALESCE(r.Name, ''), COUNT(*), COALESCE(SUM(o.TotalMinor), 0), MAX(o.UpdatedAt)
		      FROM Orders@{FORCE_INDEX=Idx_Orders_ByWarehouseCreated} o
		      LEFT JOIN Retailers r ON o.RetailerId = r.RetailerId
		      WHERE o.WarehouseId = @warehouseId
		      GROUP BY o.RetailerId, r.Name
		      ORDER BY MAX(o.UpdatedAt) DESC
		      LIMIT 100`,
		Params: map[string]any{"warehouseId": warehouseID},
	}
	iter := s.spannerClient.Single().WithTimestampBound(spanner.MaxStaleness(15*time.Second)).Query(ctx, stmt)
	defer iter.Stop()

	retailers := make([]portalRetailer, 0, 16)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("warehouse crm query: %w", err)
		}
		var rec portalRetailer
		var lastOrder spanner.NullTime
		if err := row.Columns(&rec.RetailerID, &rec.BusinessName, &rec.TotalOrders, &rec.TotalRevenue, &lastOrder); err != nil {
			return nil, fmt.Errorf("warehouse crm scan: %w", err)
		}
		if lastOrder.Valid {
			rec.LastOrderDate = lastOrder.Time.UTC().Format(time.RFC3339)
		}
		retailers = append(retailers, rec)
	}
	return retailers, nil
}

func (s *Service) loadWarehouseTreasuryOverview(ctx context.Context, warehouseID string) map[string]any {
	var totalInvoiced, totalPaid, totalOutstanding int64
	if s.analyticsQuery != nil && strings.TrimSpace(warehouseID) != "" {
		counts, err := s.analyticsQuery(ctx, warehouseID)
		if err == nil {
			totalInvoiced = counts.TotalRevenue
			totalPaid = counts.TotalRevenue
		} else {
			s.log.WarnContext(ctx, "treasury analytics query failed", "err", err, "warehouse_id", warehouseID)
		}
	}
	if s.spannerClient != nil && strings.TrimSpace(warehouseID) != "" {
		stmt := spanner.Statement{
			SQL: `SELECT COALESCE(SUM(TotalMinor), 0)
			      FROM Orders@{FORCE_INDEX=Idx_Orders_ByWarehouseCreated}
			      WHERE WarehouseId = @warehouseId
			        AND Status IN ('PENDING', 'LOADED', 'IN_TRANSIT', 'ARRIVED')`,
			Params: map[string]any{"warehouseId": warehouseID},
		}
		iter := s.spannerClient.Single().WithTimestampBound(spanner.MaxStaleness(15*time.Second)).Query(ctx, stmt)
		defer iter.Stop()
		row, err := iter.Next()
		if err == nil {
			_ = row.Columns(&totalOutstanding)
		}
	}
	return map[string]any{
		"total_invoiced":    totalInvoiced,
		"total_paid":        totalPaid,
		"total_outstanding": totalOutstanding,
	}
}
