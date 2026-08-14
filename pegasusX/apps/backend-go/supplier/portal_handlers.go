package supplier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/gs1"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/telemetry"
)

// InventoryItem is the wire/storage shape for supplier inventory rows.
type InventoryItem struct {
	SKU         string `json:"sku"`
	ProductName string `json:"product_name"`
	Quantity    int64  `json:"quantity"`
	UpdatedAt   string `json:"updated_at"`
}

// InventoryPatchRequest is the mutation payload for PATCH /v1/supplier/inventory.
type InventoryPatchRequest struct {
	SKUID         string `json:"sku_id,omitempty"`
	SKU           string `json:"sku,omitempty"`
	ProductName   string `json:"product_name,omitempty"`
	Quantity      *int64 `json:"quantity,omitempty"`
	QuantityDelta *int64 `json:"quantity_delta,omitempty"`
	Reason        string `json:"reason,omitempty"`
}

// InventoryAuditEntry tracks additive inventory mutations for supplier audit UI.
type InventoryAuditEntry struct {
	SKU         string `json:"sku"`
	ProductName string `json:"product_name"`
	Before      int64  `json:"before"`
	After       int64  `json:"after"`
	Delta       int64  `json:"delta"`
	Reason      string `json:"reason"`
	Timestamp   string `json:"timestamp"`
}

// SupplierOrderLocation is the supplier-safe driver last-location projection.
type SupplierOrderLocation struct {
	DriverID          string   `json:"driver_id"`
	SupplierID        string   `json:"supplier_id"`
	Lat               float64  `json:"lat"`
	Lng               float64  `json:"lng"`
	Latitude          float64  `json:"latitude"`
	Longitude         float64  `json:"longitude"`
	Velocity          *float64 `json:"velocity,omitempty"`
	Heading           *float64 `json:"heading,omitempty"`
	ReportedAt        string   `json:"reported_at"`
	ReceivedAt        string   `json:"received_at"`
	StaleAfterSeconds int      `json:"stale_after_seconds"`
}

// SupplierOrder is the lightweight supplier vetting/order queue shape.
type SupplierOrder struct {
	OrderID               string                 `json:"order_id"`
	SupplierID            string                 `json:"supplier_id,omitempty"`
	RetailerID            string                 `json:"retailer_id"`
	WarehouseID           string                 `json:"warehouse_id,omitempty"`
	DriverID              string                 `json:"driver_id,omitempty"`
	VehicleID             string                 `json:"vehicle_id,omitempty"`
	RouteID               string                 `json:"route_id,omitempty"`
	ManifestID            string                 `json:"manifest_id,omitempty"`
	Status                string                 `json:"status"`
	TrackingStatus        string                 `json:"tracking_status,omitempty"`
	Decision              string                 `json:"decision,omitempty"`
	Note                  string                 `json:"note,omitempty"`
	TotalMinor            int64                  `json:"total_minor"`
	Currency              string                 `json:"currency"`
	LiveLocationAvailable bool                   `json:"live_location_available"`
	DriverLocation        *SupplierOrderLocation `json:"driver_location,omitempty"`
	CreatedAt             string                 `json:"created_at"`
	UpdatedAt             string                 `json:"updated_at"`
}

type supplierOrderReader interface {
	ListOrders(ctx context.Context, supplierID string, limit, offset int) ([]SupplierOrder, error)
	CountOrders(ctx context.Context, supplierID string) (int, error)
}

type supplierOrderLocationLookup struct {
	location telemetry.DriverLocation
	found    bool
}

type configureRequest struct {
	LegalName string `json:"legal_name,omitempty"`
}

type supplierProfileResponse struct {
	SupplierID       string   `json:"supplier_id"`
	LegalName        string   `json:"legal_name"`
	ContactName      string   `json:"contact_name"`
	Email            string   `json:"email"`
	Phone            string   `json:"phone"`
	Country          string   `json:"country"`
	Currency         string   `json:"currency"`
	Categories       []string `json:"categories"`
	IsRegistered     bool     `json:"is_registered"`
	IsConfigured     bool     `json:"is_configured"`
	SelectedGateways []string `json:"selected_gateways"`
	PaymentAcceptor  string   `json:"payment_acceptor"`
	Gln              string   `json:"gln,omitempty"`
	Gs1CompanyPrefix string   `json:"gs1_company_prefix,omitempty"`
	UpdatedAt        string   `json:"updated_at"`
}

type supplierProfileUpdateRequest struct {
	LegalName        string   `json:"legal_name,omitempty"`
	ContactName      string   `json:"contact_name,omitempty"`
	Email            string   `json:"email,omitempty"`
	Phone            string   `json:"phone,omitempty"`
	Categories       []string `json:"categories,omitempty"`
	Gln              *string  `json:"gln,omitempty"`
	Gs1CompanyPrefix *string  `json:"gs1_company_prefix,omitempty"`
}

type supplierDashboardResponse struct {
	SupplierID    string `json:"supplier_id"`
	IsConfigured  bool   `json:"is_configured"`
	InventorySKUs int    `json:"inventory_skus"`
	PendingOrders int    `json:"pending_orders"`
	UpdatedAt     string `json:"updated_at"`
}

type SupplierEarningsResponse struct {
	Currency        string `json:"currency"`
	TodayMinor      int64  `json:"today_minor"`
	WeekMinor       int64  `json:"week_minor"`
	MonthMinor      int64  `json:"month_minor"`
	AuthoritySource string `json:"authority_source,omitempty"`
	Authoritative   bool   `json:"authoritative"`
	UpdatedAt       string `json:"updated_at,omitempty"`
}

type supplierPricingRuleResponse struct {
	SupplierID          string `json:"supplier_id"`
	BaseMarkupBps       int64  `json:"base_markup_bps"`
	RetailerDiscountBps int64  `json:"retailer_discount_bps"`
	MinMarginBps        int64  `json:"min_margin_bps"`
	Currency            string `json:"currency"`
	RuleVersion         int64  `json:"rule_version"`
	UpdatedAt           string `json:"updated_at"`
}

type supplierPricingRuleUpdateRequest struct {
	BaseMarkupBps       *int64 `json:"base_markup_bps,omitempty"`
	RetailerDiscountBps *int64 `json:"retailer_discount_bps,omitempty"`
	MinMarginBps        *int64 `json:"min_margin_bps,omitempty"`
	Currency            string `json:"currency,omitempty"`
}

type topologyWarehouseInput struct {
	WarehouseID           string                    `json:"warehouse_id,omitempty"`
	Name                  string                    `json:"name"`
	Lat                   float64                   `json:"lat"`
	Lng                   float64                   `json:"lng"`
	Address               string                    `json:"address,omitempty"`
	PlaceID               string                    `json:"place_id,omitempty"`
	CoverageRadiusKm      *float64                  `json:"coverage_radius_km,omitempty"`
	IsActive              *bool                     `json:"is_active,omitempty"`
	IsOnShift             *bool                     `json:"is_on_shift,omitempty"`
	TransferMode          string                    `json:"transfer_mode,omitempty"`
	CoLocateWithFactoryID string                    `json:"co_locate_with_factory_id,omitempty"`
	PrimaryFactoryID      string                    `json:"primary_factory_id,omitempty"`
	SecondaryFactoryID    string                    `json:"secondary_factory_id,omitempty"`
	AssignedFactoryIDs    []string                  `json:"assigned_factory_ids,omitempty"`
	CountryCode           string                    `json:"country_code,omitempty"`
	CoverageCities        []order.CoverageCity      `json:"coverage_cities,omitempty"`
	DefaultOutOfStockPolicy string                  `json:"default_out_of_stock_policy,omitempty"`
	OperatingSchedule     json.RawMessage           `json:"operating_schedule,omitempty"`
	InitialInventory      []topologyInventorySeed   `json:"initial_inventory,omitempty"`
}

type topologyInventorySeed struct {
	ProductID string `json:"product_id"`
	Quantity  int64  `json:"quantity"`
}

type topologyFactoryInput struct {
	FactoryID   string  `json:"factory_id,omitempty"`
	Name        string  `json:"name"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
	Address     string  `json:"address,omitempty"`
	PlaceID     string  `json:"place_id,omitempty"`
	CountryCode string  `json:"country_code,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
}

type topologyUpdateRequest struct {
	Warehouses []topologyWarehouseInput `json:"warehouses"`
	Factories  []topologyFactoryInput   `json:"factories"`
}

type vetOrderRequest struct {
	OrderID    string `json:"order_id"`
	RetailerID string `json:"retailer_id,omitempty"`
	Decision   string `json:"decision"`
	Note       string `json:"note,omitempty"`
	TotalMinor int64  `json:"total_minor,omitempty"`
	Currency   string `json:"currency,omitempty"`
}

const maxPricingBps = 10000

func seedsFromTopologyInput(rows []topologyInventorySeed) []InventorySeed {
	if len(rows) == 0 {
		return nil
	}
	out := make([]InventorySeed, 0, len(rows))
	for _, row := range rows {
		pid := strings.TrimSpace(row.ProductID)
		if pid == "" || row.Quantity <= 0 {
			continue
		}
		out = append(out, InventorySeed{ProductID: pid, Quantity: row.Quantity})
	}
	return out
}

// HandleConfigure marks onboarding as configured/registered and supports the
// supplier portal completion handoff.
func (s *Service) HandleConfigure(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	body, ok := readMutationBody(w, r, 32*1024)
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
			s.releaseMutationReplay(r.Context(), r)
		}
	}()

	var req configureRequest
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
	}

	sid := s.scopedSupplierID(r)

	current, found, err := s.repo.GetProfile(r.Context(), sid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supplier_failed"})
		return
	}
	if !found {
		current = Profile{SupplierID: sid, Country: s.country, Currency: s.currency}
	}
	now := s.now()
	if strings.TrimSpace(req.LegalName) != "" {
		current.LegalName = strings.TrimSpace(req.LegalName)
	}
	current.IsRegistered = true
	current.UpdatedAt = now
	if err := s.repo.UpdateProfile(r.Context(), current, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateSupplier, sid, events.TopicMain, events.SupplierEvent{
			BaseEvent:    events.BaseEvent{Type: events.EventSupplierProfileUpdated, Timestamp: now.Format(time.RFC3339Nano)},
			SupplierID:   sid,
			LegalName:    current.LegalName,
			ContactName:  current.ContactName,
			Email:        current.Email,
			Phone:        current.Phone,
			Country:      current.Country,
			Categories:   current.Categories,
			IsRegistered: current.IsRegistered,
			IsConfigured: current.IsConfigured,
			Action:       "CONFIGURE",
		})
	}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "persist_supplier_failed"})
		return
	}
	if s.cache != nil {
		s.cache.Invalidate(r.Context(), supplierCacheKey(s.scopedSupplierID(r)))
	}
	resp := map[string]any{
		"supplier_id":   sid,
		"is_registered": current.IsRegistered,
		"is_configured": current.IsConfigured,
		"completed_at":  now.Format(time.RFC3339Nano),
	}
	respBytes, _ := json.Marshal(resp)
	idemCommitted = true
	s.saveMutationReplay(r.Context(), key, body, http.StatusOK, respBytes)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}

// HandleProfile supports GET/PUT /v1/supplier/profile.
func (s *Service) HandleProfile(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetProfile(w, r)
	case http.MethodPut:
		s.handleUpdateProfile(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	sid := s.scopedSupplierID(r)
	current, found, err := s.repo.GetProfile(r.Context(), sid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supplier_failed"})
		return
	}
	if !found {
		current = Profile{SupplierID: sid, Country: s.country, Currency: s.currency}
	}
	writeJSON(w, http.StatusOK, s.buildSupplierProfileResponse(current))
}

func (s *Service) buildSupplierProfileResponse(current Profile) supplierProfileResponse {
	updated := ""
	if !current.UpdatedAt.IsZero() {
		updated = current.UpdatedAt.Format(time.RFC3339Nano)
	}
	return supplierProfileResponse{
		SupplierID:       current.SupplierID,
		LegalName:        current.LegalName,
		ContactName:      current.ContactName,
		Email:            current.Email,
		Phone:            current.Phone,
		Country:          current.Country,
		Currency:         current.Currency,
		Categories:       append([]string(nil), current.Categories...),
		IsRegistered:     current.IsRegistered,
		IsConfigured:     current.IsConfigured,
		SelectedGateways: append([]string(nil), current.SelectedGateways...),
		PaymentAcceptor:  normalizePaymentAcceptor(current.PaymentAcceptor),
		Gln:              current.Gln,
		Gs1CompanyPrefix: current.Gs1CompanyPrefix,
		UpdatedAt:        updated,
	}
}

func (s *Service) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	sid := s.scopedSupplierID(r)
	body, ok := readMutationBody(w, r, 32*1024)
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
			s.releaseMutationReplay(r.Context(), r)
		}
	}()

	var req supplierProfileUpdateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	current, found, err := s.repo.GetProfile(r.Context(), sid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supplier_failed"})
		return
	}
	if !found {
		current = Profile{SupplierID: sid, Country: s.country, Currency: s.currency}
	}
	if strings.TrimSpace(req.LegalName) != "" {
		current.LegalName = strings.TrimSpace(req.LegalName)
	}
	if strings.TrimSpace(req.ContactName) != "" {
		current.ContactName = strings.TrimSpace(req.ContactName)
	}
	if strings.TrimSpace(req.Email) != "" {
		current.Email = strings.TrimSpace(req.Email)
	}
	if strings.TrimSpace(req.Phone) != "" {
		current.Phone = strings.TrimSpace(req.Phone)
	}
	if len(req.Categories) > 0 {
		current.Categories = append([]string(nil), req.Categories...)
	}
	if req.Gln != nil {
		raw := strings.TrimSpace(*req.Gln)
		if raw == "" {
			current.Gln = ""
		} else {
			norm, err := gs1.NormalizeGLN(raw)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			current.Gln = norm
		}
	}
	if req.Gs1CompanyPrefix != nil {
		prefix := digitsOnlyCompanyPrefix(*req.Gs1CompanyPrefix)
		if prefix != "" && (len(prefix) < 7 || len(prefix) > 10) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_gs1_company_prefix"})
			return
		}
		current.Gs1CompanyPrefix = prefix
	}
	now := s.now()
	current.UpdatedAt = now
	if err := s.repo.UpdateProfile(r.Context(), current, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateSupplier, sid, events.TopicMain, events.SupplierEvent{
			BaseEvent:    events.BaseEvent{Type: events.EventSupplierProfileUpdated, Timestamp: now.Format(time.RFC3339Nano)},
			SupplierID:   sid,
			LegalName:    current.LegalName,
			ContactName:  current.ContactName,
			Email:        current.Email,
			Phone:        current.Phone,
			Country:      current.Country,
			Categories:   current.Categories,
			IsRegistered: current.IsRegistered,
			IsConfigured: current.IsConfigured,
			Action:       "PROFILE_UPDATED",
		})
	}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "persist_supplier_failed"})
		return
	}
	if s.cache != nil {
		s.cache.Invalidate(r.Context(), supplierCacheKey(s.scopedSupplierID(r)))
	}
	resp := s.buildSupplierProfileResponse(current)
	respBytes, _ := json.Marshal(resp)
	idemCommitted = true
	s.saveMutationReplay(r.Context(), key, body, http.StatusOK, respBytes)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}

// HandleTopology supports GET/PUT /v1/supplier/topology.
func (s *Service) HandleTopology(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleTopologyGet(w, r)
	case http.MethodPut:
		s.handleTopologyPut(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) handleTopologyGet(w http.ResponseWriter, r *http.Request) {
	sid := s.scopedSupplierID(r)
	topology, err := s.repo.GetTopology(r.Context(), sid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supplier_topology_failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"supplier_id": sid,
		"warehouses":  topology.Warehouses,
		"factories":   topology.Factories,
		"updated_at":  s.now().Format(time.RFC3339Nano),
	})
}

func (s *Service) handleTopologyPut(w http.ResponseWriter, r *http.Request) {
	sid := s.scopedSupplierID(r)
	body, ok := readMutationBody(w, r, 512*1024)
	if !ok {
		return
	}
	idemKey, handled := s.guardMutationReplay(w, r, body)
	if handled {
		return
	}

	var req topologyUpdateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	if len(req.Warehouses) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "warehouses_required"})
		return
	}

	topology := SupplierTopology{
		Warehouses: make([]WarehouseNode, 0, len(req.Warehouses)),
		Factories:  make([]FactoryNode, 0, len(req.Factories)),
	}

	for i, wh := range req.Warehouses {
		name := strings.TrimSpace(wh.Name)
		if name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("warehouses[%d].name_required", i)})
			return
		}
		if wh.Lat < -90 || wh.Lat > 90 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("warehouses[%d].lat_out_of_range", i)})
			return
		}
		if wh.Lng < -180 || wh.Lng > 180 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("warehouses[%d].lng_out_of_range", i)})
			return
		}

		coverage := defaultCoverageRadiusKm
		if wh.CoverageRadiusKm != nil && *wh.CoverageRadiusKm > 0 {
			coverage = *wh.CoverageRadiusKm
		}
		isActive := true
		if wh.IsActive != nil {
			isActive = *wh.IsActive
		}
		isOnShift := true
		if wh.IsOnShift != nil {
			isOnShift = *wh.IsOnShift
		}

		topology.Warehouses = append(topology.Warehouses, WarehouseNode{
			WarehouseID:             strings.TrimSpace(wh.WarehouseID),
			Name:                    name,
			Lat:                     wh.Lat,
			Lng:                     wh.Lng,
			Address:                 strings.TrimSpace(wh.Address),
			PlaceID:                 strings.TrimSpace(wh.PlaceID),
			CoverageRadiusKm:        coverage,
			TransferMode:            normalizeTransferMode(wh.TransferMode),
			CoLocateWithFactoryID:   strings.TrimSpace(wh.CoLocateWithFactoryID),
			PrimaryFactoryID:        strings.TrimSpace(wh.PrimaryFactoryID),
			SecondaryFactoryID:      strings.TrimSpace(wh.SecondaryFactoryID),
			AssignedFactoryIDs:      append([]string(nil), wh.AssignedFactoryIDs...),
			CountryCode:             strings.ToUpper(strings.TrimSpace(wh.CountryCode)),
			CoverageCities:          append([]order.CoverageCity(nil), wh.CoverageCities...),
			IsActive:                isActive,
			IsOnShift:               isOnShift,
			DefaultOutOfStockPolicy: strings.TrimSpace(wh.DefaultOutOfStockPolicy),
			OperatingSchedule:       string(wh.OperatingSchedule),
			InitialInventory:        seedsFromTopologyInput(wh.InitialInventory),
		})
	}

	for i, fc := range req.Factories {
		name := strings.TrimSpace(fc.Name)
		if name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("factories[%d].name_required", i)})
			return
		}
		if fc.Lat < -90 || fc.Lat > 90 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("factories[%d].lat_out_of_range", i)})
			return
		}
		if fc.Lng < -180 || fc.Lng > 180 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("factories[%d].lng_out_of_range", i)})
			return
		}

		isActive := true
		if fc.IsActive != nil {
			isActive = *fc.IsActive
		}

		topology.Factories = append(topology.Factories, FactoryNode{
			FactoryID:   strings.TrimSpace(fc.FactoryID),
			Name:        name,
			Lat:         fc.Lat,
			Lng:         fc.Lng,
			Address:     strings.TrimSpace(fc.Address),
			PlaceID:     strings.TrimSpace(fc.PlaceID),
			CountryCode: strings.ToUpper(strings.TrimSpace(fc.CountryCode)),
			IsActive:    isActive,
		})
	}

	current, found, err := s.repo.GetProfile(r.Context(), sid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supplier_failed"})
		return
	}
	if !found {
		current = Profile{SupplierID: sid, Country: s.country, Currency: s.currency, IsRegistered: true}
	}

	now := s.now()
	if err := s.repo.ReplaceTopology(r.Context(), sid, topology, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateSupplier, sid, events.TopicMain, events.SupplierEvent{
			BaseEvent:    events.BaseEvent{Type: events.EventSupplierProfileUpdated, Timestamp: now.Format(time.RFC3339Nano)},
			SupplierID:   sid,
			LegalName:    current.LegalName,
			ContactName:  current.ContactName,
			Email:        current.Email,
			Phone:        current.Phone,
			Country:      current.Country,
			Categories:   current.Categories,
			IsRegistered: current.IsRegistered,
			IsConfigured: current.IsConfigured,
			Action:       "TOPOLOGY_UPDATED",
		})
	}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "persist_supplier_topology_failed"})
		return
	}

	if s.cache != nil {
		s.cache.Invalidate(r.Context(), supplierCacheKey(s.scopedSupplierID(r)))
	}

	savedTopology, err := s.repo.GetTopology(r.Context(), sid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supplier_topology_failed"})
		return
	}
	resp := map[string]any{
		"supplier_id": sid,
		"warehouses":  savedTopology.Warehouses,
		"factories":   savedTopology.Factories,
		"updated_at":  s.now().Format(time.RFC3339Nano),
	}
	respBytes, _ := json.Marshal(resp)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
	s.storeMutationReplay(r.Context(), idemKey, body, http.StatusOK, respBytes)
}

// HandlePricingRules supports GET/PATCH /v1/supplier/pricing/rules.
func (s *Service) HandlePricingRules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handlePricingRuleGet(w, r)
	case http.MethodPatch:
		s.handlePricingRulePatch(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) handlePricingRuleGet(w http.ResponseWriter, r *http.Request) {
	sid := s.scopedSupplierID(r)
	rule, found, err := s.repo.GetPricingRule(r.Context(), sid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supplier_pricing_failed"})
		return
	}

	if !found {
		currency := s.currency
		if current, profileFound, profileErr := s.repo.GetProfile(r.Context(), sid); profileErr == nil && profileFound {
			if strings.TrimSpace(current.Currency) != "" {
				currency = current.Currency
			}
		}
		writeJSON(w, http.StatusOK, supplierPricingRuleResponse{
			SupplierID:          sid,
			BaseMarkupBps:       0,
			RetailerDiscountBps: 0,
			MinMarginBps:        0,
			Currency:            currency,
			RuleVersion:         0,
			UpdatedAt:           "",
		})
		return
	}

	updatedAt := ""
	if !rule.UpdatedAt.IsZero() {
		updatedAt = rule.UpdatedAt.Format(time.RFC3339Nano)
	}
	writeJSON(w, http.StatusOK, supplierPricingRuleResponse{
		SupplierID:          rule.SupplierID,
		BaseMarkupBps:       rule.BaseMarkupBps,
		RetailerDiscountBps: rule.RetailerDiscountBps,
		MinMarginBps:        rule.MinMarginBps,
		Currency:            rule.Currency,
		RuleVersion:         rule.RuleVersion,
		UpdatedAt:           updatedAt,
	})
}

func (s *Service) buildPricingRuleResponse(rule SupplierPricingRule) supplierPricingRuleResponse {
	updatedAt := ""
	if !rule.UpdatedAt.IsZero() {
		updatedAt = rule.UpdatedAt.Format(time.RFC3339Nano)
	}
	return supplierPricingRuleResponse{
		SupplierID:          rule.SupplierID,
		BaseMarkupBps:       rule.BaseMarkupBps,
		RetailerDiscountBps: rule.RetailerDiscountBps,
		MinMarginBps:        rule.MinMarginBps,
		Currency:            rule.Currency,
		RuleVersion:         rule.RuleVersion,
		UpdatedAt:           updatedAt,
	}
}

func (s *Service) handlePricingRulePatch(w http.ResponseWriter, r *http.Request) {
	sid := s.scopedSupplierID(r)
	body, ok := readMutationBody(w, r, 16*1024)
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
			s.releaseMutationReplay(r.Context(), r)
		}
	}()

	var req supplierPricingRuleUpdateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	if req.BaseMarkupBps == nil && req.RetailerDiscountBps == nil && req.MinMarginBps == nil && strings.TrimSpace(req.Currency) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no_pricing_fields_provided"})
		return
	}

	currentProfile, profileFound, err := s.repo.GetProfile(r.Context(), sid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supplier_failed"})
		return
	}
	if !profileFound {
		currentProfile = Profile{SupplierID: sid, Country: s.country, Currency: s.currency}
	}

	rule, found, err := s.repo.GetPricingRule(r.Context(), sid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supplier_pricing_failed"})
		return
	}
	if !found {
		rule = SupplierPricingRule{
			SupplierID: sid,
			Currency:   currentProfile.Currency,
		}
		if strings.TrimSpace(rule.Currency) == "" {
			rule.Currency = s.currency
		}
	}

	if req.BaseMarkupBps != nil {
		rule.BaseMarkupBps = *req.BaseMarkupBps
	}
	if req.RetailerDiscountBps != nil {
		rule.RetailerDiscountBps = *req.RetailerDiscountBps
	}
	if req.MinMarginBps != nil {
		rule.MinMarginBps = *req.MinMarginBps
	}
	if strings.TrimSpace(req.Currency) != "" {
		rule.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	}

	if rule.BaseMarkupBps < 0 || rule.BaseMarkupBps > maxPricingBps {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "base_markup_bps_out_of_range"})
		return
	}
	if rule.RetailerDiscountBps < 0 || rule.RetailerDiscountBps > maxPricingBps {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "retailer_discount_bps_out_of_range"})
		return
	}
	if rule.RetailerDiscountBps > rule.BaseMarkupBps {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "retailer_discount_exceeds_base_markup"})
		return
	}
	if rule.MinMarginBps < 0 || rule.MinMarginBps > maxPricingBps {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "min_margin_bps_out_of_range"})
		return
	}
	if rule.MinMarginBps > rule.BaseMarkupBps {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "min_margin_exceeds_base_markup"})
		return
	}
	if len(strings.TrimSpace(rule.Currency)) != 3 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "currency_must_be_iso4217"})
		return
	}

	updatedBy := "supplier_portal"
	if claims, ok := auth.FromContext(r.Context()); ok && strings.TrimSpace(claims.Subject) != "" {
		updatedBy = strings.TrimSpace(claims.Subject)
	}
	if strings.TrimSpace(updatedBy) == "" {
		updatedBy = "supplier_portal"
	}

	rule.SupplierID = sid
	rule.UpdatedBy = updatedBy
	rule.UpdatedAt = s.now()

	if err := s.repo.UpsertPricingRule(r.Context(), rule, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateSupplier, sid, events.TopicMain, events.SupplierEvent{
			BaseEvent:    events.BaseEvent{Type: events.EventSupplierProfileUpdated, Timestamp: rule.UpdatedAt.Format(time.RFC3339Nano)},
			SupplierID:   sid,
			LegalName:    currentProfile.LegalName,
			ContactName:  currentProfile.ContactName,
			Email:        currentProfile.Email,
			Phone:        currentProfile.Phone,
			Country:      currentProfile.Country,
			Categories:   currentProfile.Categories,
			IsRegistered: currentProfile.IsRegistered,
			IsConfigured: currentProfile.IsConfigured,
			Action:       "PRICING_RULES_UPDATED",
		})
	}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "persist_supplier_pricing_failed"})
		return
	}

	if updated, ok, err := s.repo.GetPricingRule(r.Context(), sid); err == nil && ok {
		rule = updated
	}

	if s.cache != nil {
		s.cache.Invalidate(r.Context(), supplierCacheKey(s.scopedSupplierID(r)))
	}

	resp := s.buildPricingRuleResponse(rule)
	respBytes, _ := json.Marshal(resp)
	idemCommitted = true
	s.saveMutationReplay(r.Context(), key, body, http.StatusOK, respBytes)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}

// HandleDashboard returns supplier-facing operational counters.
func (s *Service) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	sid := s.scopedSupplierID(r)
	current, found, _ := s.repo.GetProfile(r.Context(), sid)
	if !found {
		current = Profile{SupplierID: sid, Country: s.country, Currency: s.currency}
	}
	invCount := 0
	if s.inventorySvc != nil {
		if levels, err := s.inventorySvc.ListBySupplier(r.Context(), sid); err == nil {
			invCount = len(levels)
		}
	}

	pending := 0
	if s.dashboardQuery != nil {
		counts, err := s.dashboardQuery(r.Context(), sid)
		if err == nil {
			pending = counts.PendingOrders
		} else {
			s.log.WarnContext(r.Context(), "dashboard count query failed, falling back", "err", err)
		}
	}

	base := supplierDashboardResponse{
		SupplierID:    sid,
		IsConfigured:  current.IsConfigured,
		InventorySKUs: invCount,
		PendingOrders: pending,
		UpdatedAt:     s.now().Format(time.RFC3339Nano),
	}
	detail, err := s.buildSupplierDashboardDetail(r.Context(), sid, base)
	if err != nil {
		s.log.WarnContext(r.Context(), "supplier dashboard detail failed", "supplier_id", sid, "err", err)
		writeJSON(w, http.StatusOK, base)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

// HandleEarnings returns ledger-backed supplier earnings summaries.
func (s *Service) HandleEarnings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	sid := s.scopedSupplierID(r)
	now := s.now()
	if s.earningsLookup == nil {
		// G3.D honesty: machine-readable unavailable (not empty success).
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error":   "earnings_unavailable",
			"code":    "earnings_lookup_unwired",
			"message": "Supplier earnings authority is not wired in this process; use ledger/settlement views",
		})
		return
	}
	resp, err := s.earningsLookup(r.Context(), sid, s.currency, now)
	if err != nil {
		s.log.WarnContext(r.Context(), "supplier earnings authority lookup failed", "err", err, "supplier_id", sid)
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error":   "earnings_unavailable",
			"code":    "earnings_lookup_failed",
			"message": "Could not load earnings authority; treasury page may use ledger fallback",
		})
		return
	}
	if strings.TrimSpace(resp.Currency) == "" {
		resp.Currency = s.currency
	}
	if strings.TrimSpace(resp.UpdatedAt) == "" {
		resp.UpdatedAt = now.Format(time.RFC3339Nano)
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleInventory supports GET/PATCH /v1/supplier/inventory.
func (s *Service) HandleInventory(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleInventoryList(w, r)
	case http.MethodPatch:
		s.handleInventoryPatch(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) handleInventoryList(w http.ResponseWriter, r *http.Request) {
	supplierID := s.scopedSupplierID(r)
	if s.portalSpanner != nil {
		levels, err := s.listSupplierInventoryV2(r.Context(), supplierID)
		if err != nil {
			s.log.ErrorContext(r.Context(), "list supplier inventory v2 failed", "err", err, "supplier_id", supplierID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": levels})
		return
	}
	if s.inventorySvc == nil {
		writeJSON(w, http.StatusOK, map[string]any{"items": []any{}})
		return
	}
	levels, err := s.inventorySvc.ListBySupplier(r.Context(), supplierID)
	if err != nil {
		s.log.ErrorContext(r.Context(), "list inventory failed", "err", err, "supplier_id", supplierID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": levels})
}

func (s *Service) handleInventoryPatch(w http.ResponseWriter, r *http.Request) {
	body, ok := readMutationBody(w, r, 16*1024)
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
			s.releaseMutationReplay(r.Context(), r)
		}
	}()

	var req InventoryPatchRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	invID := strings.TrimSpace(req.SKUID)
	if invID == "" {
		invID = strings.TrimSpace(req.SKU)
	}
	if invID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "inventory_id_required"})
		return
	}
	if req.QuantityDelta == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "quantity_delta_required"})
		return
	}
	if s.inventorySvc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "inventory_unavailable"})
		return
	}
	version := int64(0)
	if req.Quantity != nil {
		version = *req.Quantity
	}
	if err := s.inventorySvc.AdjustStock(r.Context(), invID, *req.QuantityDelta, version); err != nil {
		s.log.ErrorContext(r.Context(), "adjust stock failed", "err", err, "inventory_id", invID)
		writeJSON(w, http.StatusConflict, map[string]string{"error": "version_conflict_or_internal"})
		return
	}
	if s.cache != nil {
		s.cache.Invalidate(r.Context(), "supplier:inventory:"+s.scopedSupplierID(r))
	}
	respBytes, _ := json.Marshal(map[string]string{"status": "ok"})
	idemCommitted = true
	s.saveMutationReplay(r.Context(), key, body, http.StatusOK, respBytes)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}

// HandleInventoryAudit is not a product surface (no adjust/stocklot ledger reader).
// P1: 410 audit_unwired — never silent {entries:[]}.
func (s *Service) HandleInventoryAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	writeJSON(w, http.StatusGone, map[string]string{
		"error":   "audit_unwired",
		"message": "GET /v1/supplier/inventory/audit is not wired; use inventory list and adjust",
	})
}

// HandleOrders returns supplier queue entries for vetting/review.
func (s *Service) HandleOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	sid := s.scopedSupplierID(r)
	statusFilter := strings.TrimSpace(r.URL.Query().Get("status"))
	groupFilter := strings.TrimSpace(r.URL.Query().Get("filter"))
	limit, offset := parseListPagination(r, 25, 500)
	orders, total, err := s.listSupplierOrdersPage(r.Context(), sid, statusFilter, groupFilter, limit, offset)
	if err != nil {
		s.log.Warn("supplier orders load failed", "supplier_id", sid, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supplier_orders_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"orders": orders,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func parseListPagination(r *http.Request, defaultLimit, maxLimit int) (limit, offset int) {
	limit = defaultLimit
	offset = 0
	if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if v := strings.TrimSpace(r.URL.Query().Get("offset")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return limit, offset
}

func orderMatchesListFilter(order SupplierOrder, statusFilter, groupFilter string) bool {
	if statusFilter != "" {
		return strings.EqualFold(order.Status, statusFilter)
	}
	switch strings.ToUpper(groupFilter) {
	case "ACTIVE":
		st := strings.ToUpper(strings.TrimSpace(order.Status))
		return st != "COMPLETED" && st != "CANCELLED" && st != "REJECTED"
	case "COMPLETED":
		return strings.EqualFold(order.Status, "COMPLETED")
	case "CANCELLED":
		st := strings.ToUpper(strings.TrimSpace(order.Status))
		return st == "CANCELLED" || st == "REJECTED"
	case "RETURNS":
		st := strings.ToUpper(strings.TrimSpace(order.Status))
		dec := strings.ToUpper(strings.TrimSpace(order.Decision))
		return st == "CANCELLED" || st == "REJECTED" || dec == "REJECTED"
	default:
		return true
	}
}

func (s *Service) listSupplierOrdersPage(ctx context.Context, supplierID, statusFilter, groupFilter string, limit, offset int) ([]SupplierOrder, int, error) {
	all, err := s.listSupplierOrders(ctx, supplierID, statusFilter, groupFilter)
	if err != nil {
		return nil, 0, err
	}
	total := len(all)
	if offset >= total {
		return []SupplierOrder{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

func (s *Service) listSupplierOrders(ctx context.Context, supplierID, statusFilter, groupFilter string) ([]SupplierOrder, error) {
	reader, ok := s.repo.(supplierOrderReader)
	if !ok {
		return []SupplierOrder{}, nil
	}
	orders, err := reader.ListOrders(ctx, supplierID, 300, 0)
	if err != nil {
		return nil, err
	}
	for i := range orders {
		if strings.TrimSpace(orders[i].SupplierID) == "" {
			orders[i].SupplierID = supplierID
		}
	}

	filtered := make([]SupplierOrder, 0, len(orders))
	for _, order := range orders {
		if !orderMatchesListFilter(order, statusFilter, groupFilter) {
			continue
		}
		filtered = append(filtered, order)
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].UpdatedAt > filtered[j].UpdatedAt })
	return s.attachOrderLocations(ctx, supplierID, filtered), nil
}

func (s *Service) attachOrderLocations(ctx context.Context, supplierID string, orders []SupplierOrder) []SupplierOrder {
	if s.locations == nil || len(orders) == 0 {
		return orders
	}
	now := s.now()
	lookups := make(map[string]supplierOrderLocationLookup, len(orders))
	for i := range orders {
		orders[i].LiveLocationAvailable = false
		orders[i].DriverLocation = nil
		driverID := strings.TrimSpace(orders[i].DriverID)
		if driverID == "" {
			continue
		}
		lookup, ok := lookups[driverID]
		if !ok {
			location, found, err := s.locations.GetDriverLocation(ctx, driverID)
			if err != nil {
				s.log.Warn("supplier orders location read failed", "driver_id", driverID, "err", err)
				continue
			}
			lookup = supplierOrderLocationLookup{location: location, found: found}
			lookups[driverID] = lookup
		}
		if !lookup.found || !lookup.location.IsLive(now) {
			continue
		}
		orderSupplierID := strings.TrimSpace(orders[i].SupplierID)
		if orderSupplierID == "" {
			orderSupplierID = supplierID
		}
		if strings.TrimSpace(lookup.location.SupplierID) != orderSupplierID {
			continue
		}
		orders[i].LiveLocationAvailable = true
		orders[i].DriverLocation = supplierOrderLocationFromTelemetry(lookup.location)
	}
	return orders
}

func supplierOrderLocationFromTelemetry(location telemetry.DriverLocation) *SupplierOrderLocation {
	return &SupplierOrderLocation{
		DriverID:          location.DriverID,
		SupplierID:        location.SupplierID,
		Lat:               location.Lat,
		Lng:               location.Lng,
		Latitude:          location.Latitude,
		Longitude:         location.Longitude,
		Velocity:          location.Velocity,
		Heading:           location.Heading,
		ReportedAt:        location.ReportedAt.UTC().Format(time.RFC3339Nano),
		ReceivedAt:        location.ReceivedAt.UTC().Format(time.RFC3339Nano),
		StaleAfterSeconds: location.StaleAfterSeconds,
	}
}

// HandleVetOrder applies APPROVED/REJECTED decisions for supplier queue items.
// HandleVetOrder is deprecated — orders auto-confirm at create; kept for SSMR inventory tests.
func (s *Service) HandleVetOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	body, ok := readMutationBody(w, r, 32*1024)
	if !ok {
		return
	}
	key, handled := s.guardMutationReplay(w, r, body)
	if handled {
		return
	}

	var req vetOrderRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	req.OrderID = strings.TrimSpace(req.OrderID)
	req.Decision = strings.ToUpper(strings.TrimSpace(req.Decision))
	if req.OrderID == "" || (req.Decision != "APPROVED" && req.Decision != "REJECTED") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id and decision(APPROVED|REJECTED) required"})
		return
	}

	sid := s.scopedSupplierID(r)
	vetter, ok := s.repo.(supplierOrderVetter)
	if !ok {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "vet_unavailable"})
		return
	}

	decidedBy := sid
	actorRole := string(auth.RoleAdmin)
	if claims, ok := auth.FromContext(r.Context()); ok {
		if strings.TrimSpace(claims.Subject) != "" {
			decidedBy = strings.TrimSpace(claims.Subject)
		}
		if strings.TrimSpace(string(claims.Role)) != "" {
			actorRole = string(claims.Role)
		}
	}

	order, err := vetter.VetOrder(r.Context(), sid, VetOrderParams{
		OrderID:   req.OrderID,
		Decision:  req.Decision,
		Note:      strings.TrimSpace(req.Note),
		DecidedBy: decidedBy,
		ActorRole: actorRole,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrOrderNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
		case errors.Is(err, ErrVetForbidden):
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		case errors.Is(err, ErrInvalidVetState):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid_vet_state"})
		case errors.Is(err, ErrOrderAlreadyAssigned):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "order_already_assigned"})
		case errors.Is(err, ErrPaymentNotCleared):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "payment_not_cleared"})
		default:
			s.log.ErrorContext(r.Context(), "vet order failed", "order_id", req.OrderID, "supplier_id", sid, "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "vet_failed"})
		}
		return
	}

	if s.cache != nil {
		s.cache.Invalidate(r.Context(),
			supplierCacheKey(sid),
			fmt.Sprintf("orders:supplier:%s", sid),
			fmt.Sprintf("orders:retailer:%s", strings.TrimSpace(order.RetailerID)),
		)
		if req.Decision == "REJECTED" {
			s.cache.Invalidate(r.Context(), "catalog:products:"+sid)
		}
	}

	response := map[string]any{"order": order}
	if encoded, err := json.Marshal(response); err == nil {
		s.storeMutationReplay(r.Context(), key, body, http.StatusOK, encoded)
	}
	w.Header().Set("Deprecation", "true")
	w.Header().Set("Sunset", "Sun, 31 Dec 2026 23:59:59 GMT")
	writeJSON(w, http.StatusOK, response)
}

func digitsOnlyCompanyPrefix(raw string) string {
	var b strings.Builder
	for i := 0; i < len(raw); i++ {
		if raw[i] >= '0' && raw[i] <= '9' {
			b.WriteByte(raw[i])
		}
	}
	return b.String()
}
