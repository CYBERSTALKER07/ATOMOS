package warehouse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
	"golang.org/x/crypto/bcrypt"
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
	IsOnline                 bool    `json:"is_online"`
	OnShift                  bool    `json:"on_shift,omitempty"`
	UnavailableReason        string  `json:"unavailable_reason,omitempty"`
	VehicleID                string  `json:"vehicle_id,omitempty"`
	VehicleLabel             string  `json:"vehicle_label,omitempty"`
	LicensePlate             string  `json:"license_plate,omitempty"`
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
		{DriverID: "drv-1", Name: "Jamshid R.", Phone: "+998901111001", TruckStatus: "AVAILABLE", IsActive: true, OnShift: true, VehicleID: "veh-1", VehicleClass: "CLASS_A", MaxVolumeVU: 50, VehicleIsActive: true},
		{DriverID: "drv-2", Name: "Dilnoza K.", Phone: "+998901111002", TruckStatus: "IN_TRANSIT", IsActive: true, OnShift: true, VehicleID: "veh-2", VehicleClass: "CLASS_C", MaxVolumeVU: 400, VehicleIsActive: true},
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

	var active, pending, totalDrivers, totalVehicles int64
	var lowStock int64
	ordersByStatus := emptyWarehouseOrderStatusCounts()
	truckDuty := emptyWarehouseTruckDutyCounts()
	holdReasons := []map[string]any{}
	var drivers []PortalDriver
	var vehicles []PortalVehicle

	if s.opsOrders != nil {
		orders, err := s.opsOrders(r.Context(), whID, 200)
		if err == nil {
			ordersByStatus, active, pending = countWarehouseOrdersByStatus(orders)
		}
	} else {
		s.ensurePortalSeed()
		s.mu.RLock()
		ordersByStatus, active, pending = countWarehouseOrdersByStatus(s.orders)
		s.mu.RUnlock()
	}

	if s.opsDrivers != nil {
		rows, err := s.opsDrivers(r.Context(), whID)
		if err == nil {
			drivers = rows
		}
	} else {
		s.mu.RLock()
		drivers = append([]PortalDriver(nil), s.drivers...)
		s.mu.RUnlock()
	}
	totalDrivers = int64(len(drivers))
	truckDuty = countWarehouseTruckDuty(drivers)
	onRoute := int64(truckDuty["IN_TRANSIT"])
	idle := int64(truckDuty["AVAILABLE"])

	if s.opsVehicles != nil {
		rows, err := s.opsVehicles(r.Context(), whID)
		if err == nil {
			vehicles = rows
		}
	} else {
		s.mu.RLock()
		vehicles = append([]PortalVehicle(nil), s.vehicles...)
		s.mu.RUnlock()
	}
	totalVehicles = int64(len(vehicles))
	holdReasons = collectHoldReasons(drivers, vehicles)
	_, demandSource := s.productDemandForecast(r.Context(), whID, 7)

	if s.repo != nil && whID != "" {
		inventoryList, _ := s.repo.GetInventoryList(r.Context(), whID, InventoryListOptions{})
		for _, row := range inventoryList {
			if row.Quantity < 20 {
				lowStock++
			}
		}
	}
	s.mu.RLock()
	staffCount := int64(len(s.staff))
	s.mu.RUnlock()

	payload := map[string]any{
		"warehouse_id":              whID,
		"active_orders":             active,
		"pending_dispatch":          pending,
		"drivers_on_route":          onRoute,
		"drivers_idle":              idle,
		"total_drivers":             totalDrivers,
		"total_vehicles":            totalVehicles,
		"low_stock_count":           lowStock,
		"total_staff":               staffCount,
		"completed_today_available": false,
		"today_revenue_available":   false,
		"history_available":         false,
		"currency":                  s.responseCurrency(r.Context()),
		"orders_by_status":          ordersByStatus,
		"truck_duty":                truckDuty,
		"hold_reasons":              holdReasons,
		"demand_source":             demandSource,
		"fleet_status":              fleetStatusFromDuty(truckDuty),
	}
	key := cache.DashboardKey("warehouse", whID)
	body, err := cache.LoadDashboard(s.cache, r.Context(), key, func(context.Context) ([]byte, error) {
		return json.Marshal(payload)
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "dashboard_unavailable"})
		return
	}
	cache.WriteJSONWithETag(w, r, body)
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
		fresh := strings.EqualFold(r.URL.Query().Get("fresh"), "1") || strings.EqualFold(r.URL.Query().Get("fresh"), "true")
		limit := parseInventoryQueryInt(r.URL.Query().Get("limit"), 0, 500)
		offset := parseInventoryQueryInt(r.URL.Query().Get("offset"), 0, 100000)
		inventoryList, err := s.repo.GetInventoryList(r.Context(), whID, InventoryListOptions{
			Fresh:  fresh,
			Limit:  limit,
			Offset: offset,
		})
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
				"product_id":          sku,
				"sku_id":              row.SKU,
				"product_name":        row.ProductName,
				"quantity":            qty,
				"quantity_on_hand":    row.QuantityOnHand,
				"reorder_threshold":   threshold,
				"out_of_stock_policy": row.OutOfStockPolicy,
				"effective_policy":    row.EffectivePolicy,
				"accepts_backorder":   row.EffectivePolicy == OutOfStockPolicyAcceptBackorder,
				"is_low_stock":        isLow,
				"last_updated":        row.UpdatedAt,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"inventory":    items,
			"items":        items,
			"limit":        limit,
			"offset":       offset,
			"total":        len(items),
			"lots_enabled": stocklots.LotsEnabled(),
		})
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
			// B2 M-P0-3: stock absolute set must leave the bus.
			return outbox.EmitJSON(r.Context(), buf, events.AggregateWarehouse, whID, events.TopicMain, map[string]any{
				"type":         events.EventInventoryQuantityUpdated,
				"warehouse_id": whID,
				"product_id":   patchBody.ProductID,
				"quantity":     patchBody.Quantity,
				"timestamp":    time.Now().UTC().Format(time.RFC3339Nano),
			})
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
		plainPin, err := generateOpsDriverPIN(4)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "pin_generate_failed"})
			return
		}
		pinHash, err := bcrypt.GenerateFromPassword([]byte(plainPin), bcrypt.DefaultCost)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "pin_hash_failed"})
			return
		}
		role := strings.TrimSpace(req.Role)
		if role == "" {
			role = "WAREHOUSE_STAFF"
		}

		if s.spannerClient != nil {
			now := s.now().UTC()
			m := spanner.Insert("SupplierUsers",
				[]string{"UserId", "SupplierId", "Phone", "Name", "PasswordHash", "SupplierRole", "AssignedWarehouseId", "IsActive", "CreatedAt", "UpdatedAt"},
				[]any{staffID, s.resolveSupplierScope(r.Context()), strings.TrimSpace(req.Phone), strings.TrimSpace(req.Name), string(pinHash), role, warehouseIDFromRequest(r), true, now, now},
			)
			if _, err := s.spannerClient.Apply(r.Context(), []*spanner.Mutation{m}); err != nil {
				s.log.ErrorContext(r.Context(), "failed to create staff", "err", err)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed_to_create_staff"})
				return
			}
		} else {
			row := portalStaff{StaffID: staffID, Name: req.Name, Phone: req.Phone, Role: role}
			s.mu.Lock()
			s.staff = append(s.staff, row)
			s.mu.Unlock()
		}

		resp := map[string]any{"staff_id": staffID, "pin": plainPin}
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
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	whID := warehouseIDFromRequest(r)
	if s.spannerClient != nil && !s.portalSeedEnabled() {
		manifests, err := s.loadWarehouseManifestsFromSpanner(r.Context(), whID)
		if err != nil {
			s.log.WarnContext(r.Context(), "warehouse manifests query failed", "err", err, "warehouse_id", whID)
			writeJSON(w, http.StatusOK, map[string]any{"manifests": []portalManifest{}})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"manifests": manifests})
		return
	}
	s.ensurePortalSeed()
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
			payload["currency"] = s.responseCurrency(r.Context())
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
		"warehouse_id":              whID,
		"period":                    period,
		"currency":                  s.responseCurrency(r.Context()),
		"total_orders":              totalOrders,
		"total_revenue":             totalRevenue,
		"completed_orders":          completedOrders,
		"cancelled_orders":          cancelledOrders,
		"avg_order_value":           avgOrderValue,
		"top_products_available":    false,
		"daily_breakdown_available": false,
		"fleet_utilization":         map[string]any{"utilization_pct": 0},
		"import_freshness":          map[string]any{"applied_rows_30d": 0, "applied_skus_30d": 0, "quantity_delta_30d": 0},
		"import_anomaly_queue":      map[string]any{"open_rows_30d": 0},
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
	// Fail-closed honesty: [] only when the warehouse-scoped query is empty.
	if view == "invoices" {
		sid := s.resolveAnalyticsSupplierID(r)
		rows, err := s.loadWarehouseTreasuryInvoices(r.Context(), whID, sid)
		if err != nil {
			if s.log != nil {
				s.log.WarnContext(r.Context(), "warehouse treasury invoices read failed", "warehouse_id", whID, "err", err)
			}
			writeTreasuryInvoicesUnavailable(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"invoices": rows})
		return
	}
	if s.spannerClient != nil && !s.portalSeedEnabled() {
		writeJSON(w, http.StatusOK, s.loadWarehouseTreasuryOverview(r.Context(), whID))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"total_invoiced":    int64(0),
		"total_paid":        int64(0),
		"total_outstanding": int64(0),
		"currency":          s.responseCurrency(r.Context()),
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

		warehouseID := warehouseIDFromRequest(r)
		if warehouseID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "warehouse_id required"})
			return
		}
		if s.repo == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "repository_unavailable"})
			return
		}

		rows, err := s.repo.ListSupplyRequests(r.Context(), warehouseID, 200)
		if err != nil {
			s.log.Warn("warehouse supply request cancel load failed", "warehouse_id", warehouseID, "request_id", id, "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supply_requests_failed"})
			return
		}
		var target SupplyRequest
		var found bool
		for _, req := range rows {
			if req.RequestID == id {
				target = req
				found = true
				break
			}
		}
		if !found {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "request_not_found"})
			return
		}

		state := strings.ToUpper(strings.TrimSpace(target.State))
		if state == "" {
			state = strings.ToUpper(strings.TrimSpace(target.Status))
		}
		if state == "CANCELLED" {
			writeJSON(w, http.StatusOK, map[string]any{"request_id": id, "state": "CANCELLED"})
			return
		}
		if state == "FULFILLED" || state == "RECEIVED" {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "cannot_cancel_terminal_state"})
			return
		}

		nowTS := s.now().UTC().Format(time.RFC3339Nano)
		target.State = "CANCELLED"
		target.Status = "CANCELLED"
		target.UpdatedAt = nowTS

		if err := s.repo.UpdateSupplyRequestStatus(r.Context(), id, "CANCELLED", func(txn outbox.TxnBuffer) error {
			return outbox.EmitJSON(r.Context(), txn, events.AggregateWarehouse, id, events.TopicMain, events.WarehouseEvent{
				BaseEvent:   events.BaseEvent{Type: events.EventSupplyRequestUpdate, Timestamp: nowTS},
				RequestID:   id,
				WarehouseID: warehouseID,
				SupplierID:  s.resolveSupplierScope(r.Context()),
				FactoryID:   target.FactoryID,
				Status:      "CANCELLED",
			})
		}); err != nil {
			s.log.Warn("warehouse supply request cancel failed", "warehouse_id", warehouseID, "request_id", id, "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "cancel_supply_request_failed"})
			return
		}

		if s.cache != nil {
			s.cache.Invalidate(r.Context(), warehouseSupplyRequestsKey(s.resolveSupplierScope(r.Context()), warehouseID))
		}
		s.broadcastSupplyRequestUpdate(r.Context(), warehouseID, target)
		s.log.Info("warehouse supply request cancelled", "supplier_id", s.resolveSupplierScope(r.Context()), "warehouse_id", warehouseID, "request_id", id)
		writeJSON(w, http.StatusOK, map[string]any{"request_id": id, "state": "CANCELLED"})
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
			"item_id":            item.ItemID,
			"product_id":         item.ProductID,
			"requested_quantity": item.RequestedQuantity,
			"recommended_qty":    item.RecommendedQty,
			"unit_volume_vu":     item.UnitVolumeVU,
		})
	}
	payload := map[string]any{
		"request_id":        req.RequestID,
		"warehouse_id":      req.WarehouseID,
		"factory_id":        factoryID,
		"supplier_id":       req.SupplierID,
		"state":             state,
		"priority":          priority,
		"notes":             req.Notes,
		"region_id":         req.RegionID,
		"total_volume_vu":   totalVU,
		"projected_units":   req.ProjectedUnits,
		"item_count":        len(req.Items),
		"items":             items,
		"transfer_order_id": strings.TrimSpace(req.LinkedTransferID),
		"created_by":        req.RequestedBy,
		"created_at":        req.CreatedAt,
		"updated_at":        req.UpdatedAt,
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
		"currency":          s.responseCurrency(ctx),
	}
}

func parseInventoryQueryInt(raw string, fallback, max int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 0 {
		return fallback
	}
	if max > 0 && v > max {
		return max
	}
	return v
}
