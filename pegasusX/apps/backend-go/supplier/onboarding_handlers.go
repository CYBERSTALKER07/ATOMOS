package supplier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/dispatch"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"golang.org/x/crypto/bcrypt"
)

var (
	errOrgMemberPhoneExists    = errors.New("supplier_org_member_phone_exists")
	errDriverPhoneExists       = errors.New("supplier_driver_phone_exists")
	errVehiclePlateExists      = errors.New("supplier_vehicle_plate_exists")
	errFleetVehicleMismatch    = errors.New("supplier_fleet_vehicle_mismatch")
	errUnsupportedSupplierRole = errors.New("unsupported_supplier_role")
	errMissingNodeAssignment   = errors.New("missing_node_assignment")
	errInvalidNodeAssignment   = errors.New("invalid_node_assignment")
	errInvalidFleetHomeNode    = errors.New("invalid_fleet_home_node")
)

type SupplierOrgMember struct {
	UserID              string    `json:"user_id"`
	SupplierID          string    `json:"supplier_id"`
	Name                string    `json:"name"`
	Email               string    `json:"email,omitempty"`
	Phone               string    `json:"phone"`
	SupplierRole        auth.Role `json:"supplier_role"`
	AssignedWarehouseID string    `json:"assigned_warehouse_id,omitempty"`
	AssignedFactoryID   string    `json:"assigned_factory_id,omitempty"`
	IsActive            bool      `json:"is_active"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type SupplierFleetDriver struct {
	DriverID     string            `json:"driver_id"`
	SupplierID   string            `json:"supplier_id"`
	Name         string            `json:"name"`
	Phone        string            `json:"phone"`
	HomeNodeType auth.HomeNodeType `json:"home_node_type"`
	HomeNodeID   string            `json:"home_node_id"`
	VehicleID    string            `json:"vehicle_id,omitempty"`
	IsActive     bool              `json:"is_active"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

type SupplierFleetVehicle struct {
	VehicleID    string            `json:"vehicle_id"`
	SupplierID   string            `json:"supplier_id"`
	Label        string            `json:"label,omitempty"`
	LicensePlate string            `json:"license_plate"`
	HomeNodeType auth.HomeNodeType `json:"home_node_type"`
	HomeNodeID   string            `json:"home_node_id"`
	VehicleClass string            `json:"vehicle_class"`
	MaxVolumeVU  float64           `json:"max_volume_vu"`
	IsActive     bool              `json:"is_active"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

type CreateOrgMemberParams struct {
	UserID              string
	SupplierID          string
	Name                string
	Email               string
	Phone               string
	PasswordHash        string
	SupplierRole        auth.Role
	AssignedWarehouseID string
	AssignedFactoryID   string
	IsActive            bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type CreateFleetDriverParams struct {
	DriverID     string
	SupplierID   string
	Name         string
	Phone        string
	PinHash      string
	HomeNodeType auth.HomeNodeType
	HomeNodeID   string
	VehicleID    string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type CreateFleetVehicleParams struct {
	VehicleID    string
	SupplierID   string
	Label        string
	LicensePlate string
	HomeNodeType auth.HomeNodeType
	HomeNodeID   string
	VehicleClass string
	MaxVolumeVU  float64
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type orgMemberCreateRequest struct {
	Name                string `json:"name"`
	Email               string `json:"email,omitempty"`
	Phone               string `json:"phone"`
	Password            string `json:"password"`
	SupplierRole        string `json:"supplier_role"`
	AssignedWarehouseID string `json:"assigned_warehouse_id,omitempty"`
	AssignedFactoryID   string `json:"assigned_factory_id,omitempty"`
	IsActive            *bool  `json:"is_active,omitempty"`
}

type fleetDriverCreateRequest struct {
	Name         string `json:"name"`
	Phone        string `json:"phone"`
	Pin          string `json:"pin"`
	HomeNodeType string `json:"home_node_type"`
	HomeNodeID   string `json:"home_node_id"`
	VehicleID    string `json:"vehicle_id,omitempty"`
	IsActive     *bool  `json:"is_active,omitempty"`
}

type fleetVehicleCreateRequest struct {
	Label        string  `json:"label,omitempty"`
	LicensePlate string  `json:"license_plate"`
	HomeNodeType string  `json:"home_node_type"`
	HomeNodeID   string  `json:"home_node_id"`
	VehicleClass string  `json:"vehicle_class,omitempty"`
	MaxVolumeVU  float64 `json:"max_volume_vu,omitempty"`
	IsActive     *bool   `json:"is_active,omitempty"`
}

type supplierOrgMembersResponse struct {
	SupplierID string              `json:"supplier_id"`
	Items      []SupplierOrgMember `json:"items"`
	UpdatedAt  string              `json:"updated_at"`
}

type supplierFleetDriversResponse struct {
	SupplierID string                `json:"supplier_id"`
	Items      []SupplierFleetDriver `json:"items"`
	UpdatedAt  string                `json:"updated_at"`
}

type supplierFleetVehiclesResponse struct {
	SupplierID string                 `json:"supplier_id"`
	Items      []SupplierFleetVehicle `json:"items"`
	UpdatedAt  string                 `json:"updated_at"`
}

type topologyLookup struct {
	warehouses map[string]struct{}
	factories  map[string]struct{}
}

func (s *Service) HandleOrgMembers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleOrgMembersGet(w, r)
	case http.MethodPost:
		s.handleOrgMembersPost(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) HandleFleetDrivers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleFleetDriversGet(w, r)
	case http.MethodPost:
		s.handleFleetDriversPost(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) HandleFleetVehicles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleFleetVehiclesGet(w, r)
	case http.MethodPost:
		s.handleFleetVehiclesPost(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) handleOrgMembersGet(w http.ResponseWriter, r *http.Request) {
	resp, err := s.loadOrgMembersResponse(r.Context(), s.scopedSupplierID(r))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supplier_org_members_failed"})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Service) handleOrgMembersPost(w http.ResponseWriter, r *http.Request) {
	body, ok := readMutationBody(w, r, 32*1024)
	if !ok {
		return
	}
	key, handled := s.guardMutationReplay(w, r, body)
	if handled {
		return
	}

	var req orgMemberCreateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	topology, err := s.repo.GetTopology(r.Context(), s.scopedSupplierID(r))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supplier_topology_failed"})
		return
	}
	sid := s.scopedSupplierID(r)
	params, err := buildOrgMemberParams(req, sid, topology, s.now())
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "hash_supplier_org_member_password_failed"})
		return
	}
	params.PasswordHash = string(hash)

	if err := s.repo.CreateOrgMember(r.Context(), params, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateSupplier, sid, events.TopicMain, events.SupplierEvent{
			BaseEvent:           events.BaseEvent{Type: events.EventSupplierMemberAdded, Timestamp: params.UpdatedAt.Format(time.RFC3339Nano)},
			SupplierID:          sid,
			UserID:              params.UserID,
			SupplierRole:        string(params.SupplierRole),
			AssignedWarehouseID: params.AssignedWarehouseID,
			AssignedFactoryID:   params.AssignedFactoryID,
			Action:              "ORG_MEMBER_CREATED",
		})
	}); err != nil {
		switch {
		case errors.Is(err, errOrgMemberPhoneExists):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "supplier_org_member_phone_exists"})
		default:
			slog.Error("persist supplier org member failed", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "persist_supplier_org_member_failed"})
		}
		return
	}
	if s.cache != nil {
		s.cache.Invalidate(r.Context(), supplierCacheKey(s.scopedSupplierID(r)))
	}
	resp, err := s.loadOrgMembersResponse(r.Context(), sid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supplier_org_members_failed"})
		return
	}
	respBytes, _ := json.Marshal(resp)
	s.storeMutationReplay(r.Context(), key, body, http.StatusCreated, respBytes)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(respBytes)
}

func (s *Service) handleFleetDriversGet(w http.ResponseWriter, r *http.Request) {
	resp, err := s.loadFleetDriversResponse(r.Context(), s.scopedSupplierID(r))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supplier_fleet_drivers_failed"})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Service) handleFleetDriversPost(w http.ResponseWriter, r *http.Request) {
	body, ok := readMutationBody(w, r, 32*1024)
	if !ok {
		return
	}
	key, handled := s.guardMutationReplay(w, r, body)
	if handled {
		return
	}

	var req fleetDriverCreateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	topology, err := s.repo.GetTopology(r.Context(), s.scopedSupplierID(r))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supplier_topology_failed"})
		return
	}
	sid := s.scopedSupplierID(r)
	params, err := buildFleetDriverParams(req, sid, topology, s.now())
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Pin), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "hash_supplier_driver_pin_failed"})
		return
	}
	params.PinHash = string(hash)

	if err := s.repo.CreateFleetDriver(r.Context(), params, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateDriver, params.DriverID, events.TopicMain, events.DriverEvent{
			BaseEvent:    events.BaseEvent{Type: events.EventDriverCreated, Timestamp: params.UpdatedAt.Format(time.RFC3339Nano)},
			SupplierID:   sid,
			DriverID:     params.DriverID,
			HomeNodeType: string(params.HomeNodeType),
			HomeNodeID:   params.HomeNodeID,
		})
	}); err != nil {
		switch {
		case errors.Is(err, errDriverPhoneExists):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "supplier_driver_phone_exists"})
		case errors.Is(err, errFleetVehicleMismatch):
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "supplier_fleet_vehicle_mismatch"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "persist_supplier_driver_failed"})
		}
		return
	}
	if s.cache != nil {
		s.cache.Invalidate(r.Context(), supplierCacheKey(s.scopedSupplierID(r)))
	}
	resp, err := s.loadFleetDriversResponse(r.Context(), sid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supplier_fleet_drivers_failed"})
		return
	}
	respBytes, _ := json.Marshal(resp)
	s.storeMutationReplay(r.Context(), key, body, http.StatusCreated, respBytes)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(respBytes)
}

func (s *Service) handleFleetVehiclesGet(w http.ResponseWriter, r *http.Request) {
	resp, err := s.loadFleetVehiclesResponse(r.Context(), s.scopedSupplierID(r))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supplier_fleet_vehicles_failed"})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Service) handleFleetVehiclesPost(w http.ResponseWriter, r *http.Request) {
	body, ok := readMutationBody(w, r, 32*1024)
	if !ok {
		return
	}
	key, handled := s.guardMutationReplay(w, r, body)
	if handled {
		return
	}

	var req fleetVehicleCreateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	topology, err := s.repo.GetTopology(r.Context(), s.scopedSupplierID(r))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supplier_topology_failed"})
		return
	}
	sid := s.scopedSupplierID(r)
	params, err := buildFleetVehicleParams(req, sid, topology, s.now())
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}

	if err := s.repo.CreateFleetVehicle(r.Context(), params, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateVehicle, params.VehicleID, events.TopicMain, events.VehicleEvent{
			BaseEvent:    events.BaseEvent{Type: events.EventVehicleCreated, Timestamp: params.UpdatedAt.Format(time.RFC3339Nano)},
			SupplierID:   sid,
			VehicleID:    params.VehicleID,
			HomeNodeType: string(params.HomeNodeType),
			HomeNodeID:   params.HomeNodeID,
		})
	}); err != nil {
		switch {
		case errors.Is(err, errVehiclePlateExists):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "supplier_vehicle_plate_exists"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "persist_supplier_vehicle_failed"})
		}
		return
	}
	if s.cache != nil {
		s.cache.Invalidate(r.Context(), supplierCacheKey(s.scopedSupplierID(r)))
	}
	resp, err := s.loadFleetVehiclesResponse(r.Context(), sid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supplier_fleet_vehicles_failed"})
		return
	}
	respBytes, _ := json.Marshal(resp)
	s.storeMutationReplay(r.Context(), key, body, http.StatusCreated, respBytes)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(respBytes)
}

func (s *Service) loadOrgMembersResponse(ctx context.Context, supplierID string) (supplierOrgMembersResponse, error) {
	items, err := s.repo.ListOrgMembers(ctx, supplierID)
	if err != nil {
		return supplierOrgMembersResponse{}, err
	}
	return supplierOrgMembersResponse{
		SupplierID: supplierID,
		Items:      items,
		UpdatedAt:  s.now().Format(time.RFC3339Nano),
	}, nil
}

func (s *Service) loadFleetDriversResponse(ctx context.Context, supplierID string) (supplierFleetDriversResponse, error) {
	items, err := s.repo.ListFleetDrivers(ctx, supplierID)
	if err != nil {
		return supplierFleetDriversResponse{}, err
	}
	return supplierFleetDriversResponse{
		SupplierID: supplierID,
		Items:      items,
		UpdatedAt:  s.now().Format(time.RFC3339Nano),
	}, nil
}

func (s *Service) loadFleetVehiclesResponse(ctx context.Context, supplierID string) (supplierFleetVehiclesResponse, error) {
	items, err := s.repo.ListFleetVehicles(ctx, supplierID)
	if err != nil {
		return supplierFleetVehiclesResponse{}, err
	}
	return supplierFleetVehiclesResponse{
		SupplierID: supplierID,
		Items:      items,
		UpdatedAt:  s.now().Format(time.RFC3339Nano),
	}, nil
}

func readMutationBody(w http.ResponseWriter, r *http.Request, limit int64) ([]byte, bool) {
	body, err := io.ReadAll(io.LimitReader(r.Body, limit))
	defer r.Body.Close()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_failed"})
		return nil, false
	}
	return body, true
}

func idempotencyKeyFromRequest(r *http.Request) string {
	if key := strings.TrimSpace(r.Header.Get("Idempotency-Key")); key != "" {
		return key
	}
	return strings.TrimSpace(r.Header.Get("X-Idempotency-Key"))
}

func (s *Service) guardMutationReplay(w http.ResponseWriter, r *http.Request, body []byte) (string, bool) {
	key := idempotencyKeyFromRequest(r)
	if key == "" || s.idem == nil {
		return "", false
	}
	hash := sha256Hex(body)
	rec, hit, err := idempotency.Guard(r.Context(), s.idem, key, hash)
	switch {
	case errors.Is(err, idempotency.ErrConflict):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "idempotency_key_payload_mismatch"})
		return "", true
	case err != nil:
		s.log.Warn("onboarding idempotency guard failed", "err", err)
		return key, false
	case hit:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(rec.StatusCode)
		_, _ = w.Write(rec.Response)
		return "", true
	default:
		return key, false
	}
}

func (s *Service) storeMutationReplay(ctx context.Context, key string, body []byte, statusCode int, response []byte) {
	if key == "" || s.idem == nil {
		return
	}
	_ = s.idem.Save(ctx, key, idempotency.Record{
		BodyHash:   sha256Hex(body),
		StatusCode: statusCode,
		Response:   response,
		StoredAt:   s.now(),
	}, 24*time.Hour)
}

func buildOrgMemberParams(req orgMemberCreateRequest, supplierID string, topology SupplierTopology, now time.Time) (CreateOrgMemberParams, error) {
	lookup := newTopologyLookup(topology)
	name := strings.TrimSpace(req.Name)
	if len(name) < 2 {
		return CreateOrgMemberParams{}, fmt.Errorf("supplier_org_member_name_required")
	}
	phone := strings.TrimSpace(req.Phone)
	if len(phone) < 7 {
		return CreateOrgMemberParams{}, fmt.Errorf("supplier_org_member_phone_required")
	}
	if len(strings.TrimSpace(req.Password)) < 8 {
		return CreateOrgMemberParams{}, fmt.Errorf("supplier_org_member_password_too_short")
	}
	role := auth.Role(strings.ToUpper(strings.TrimSpace(req.SupplierRole)))
	assignedWarehouseID := strings.TrimSpace(req.AssignedWarehouseID)
	assignedFactoryID := strings.TrimSpace(req.AssignedFactoryID)
	if err := validateOrgRoleAssignment(role, assignedWarehouseID, assignedFactoryID, lookup); err != nil {
		return CreateOrgMemberParams{}, err
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	trimmedEmail := strings.TrimSpace(req.Email)
	if trimmedEmail != "" && !strings.Contains(trimmedEmail, "@") {
		return CreateOrgMemberParams{}, fmt.Errorf("supplier_org_member_email_invalid")
	}
	return CreateOrgMemberParams{
		UserID:              uuid.NewString(),
		SupplierID:          supplierID,
		Name:                name,
		Email:               trimmedEmail,
		Phone:               phone,
		SupplierRole:        role,
		AssignedWarehouseID: assignedWarehouseID,
		AssignedFactoryID:   assignedFactoryID,
		IsActive:            isActive,
		CreatedAt:           now.UTC(),
		UpdatedAt:           now.UTC(),
	}, nil
}

func buildFleetDriverParams(req fleetDriverCreateRequest, supplierID string, topology SupplierTopology, now time.Time) (CreateFleetDriverParams, error) {
	lookup := newTopologyLookup(topology)
	name := strings.TrimSpace(req.Name)
	if len(name) < 2 {
		return CreateFleetDriverParams{}, fmt.Errorf("supplier_driver_name_required")
	}
	phone := strings.TrimSpace(req.Phone)
	if len(phone) < 7 {
		return CreateFleetDriverParams{}, fmt.Errorf("supplier_driver_phone_required")
	}
	pin := strings.TrimSpace(req.Pin)
	if len(pin) < 4 {
		return CreateFleetDriverParams{}, fmt.Errorf("supplier_driver_pin_too_short")
	}
	homeNodeType, homeNodeID, err := validateHomeNode(strings.TrimSpace(req.HomeNodeType), strings.TrimSpace(req.HomeNodeID), lookup)
	if err != nil {
		return CreateFleetDriverParams{}, err
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	return CreateFleetDriverParams{
		DriverID:     uuid.NewString(),
		SupplierID:   supplierID,
		Name:         name,
		Phone:        phone,
		HomeNodeType: homeNodeType,
		HomeNodeID:   homeNodeID,
		VehicleID:    strings.TrimSpace(req.VehicleID),
		IsActive:     isActive,
		CreatedAt:    now.UTC(),
		UpdatedAt:    now.UTC(),
	}, nil
}

func buildFleetVehicleParams(req fleetVehicleCreateRequest, supplierID string, topology SupplierTopology, now time.Time) (CreateFleetVehicleParams, error) {
	lookup := newTopologyLookup(topology)
	licensePlate := strings.ToUpper(strings.TrimSpace(req.LicensePlate))
	if len(licensePlate) < 3 {
		return CreateFleetVehicleParams{}, fmt.Errorf("supplier_vehicle_license_plate_required")
	}
	homeNodeType, homeNodeID, err := validateHomeNode(strings.TrimSpace(req.HomeNodeType), strings.TrimSpace(req.HomeNodeID), lookup)
	if err != nil {
		return CreateFleetVehicleParams{}, err
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	vehicleClass := dispatch.ResolveVehicleClass(req.VehicleClass)
	maxVolumeVU := dispatch.ResolveMaxVolumeVU(vehicleClass, req.MaxVolumeVU)
	return CreateFleetVehicleParams{
		VehicleID:    uuid.NewString(),
		SupplierID:   supplierID,
		Label:        strings.TrimSpace(req.Label),
		LicensePlate: licensePlate,
		HomeNodeType: homeNodeType,
		HomeNodeID:   homeNodeID,
		VehicleClass: vehicleClass,
		MaxVolumeVU:  maxVolumeVU,
		IsActive:     isActive,
		CreatedAt:    now.UTC(),
		UpdatedAt:    now.UTC(),
	}, nil
}

func newTopologyLookup(topology SupplierTopology) topologyLookup {
	lookup := topologyLookup{
		warehouses: make(map[string]struct{}, len(topology.Warehouses)),
		factories:  make(map[string]struct{}, len(topology.Factories)),
	}
	for _, warehouse := range topology.Warehouses {
		lookup.warehouses[strings.TrimSpace(warehouse.WarehouseID)] = struct{}{}
	}
	for _, factory := range topology.Factories {
		lookup.factories[strings.TrimSpace(factory.FactoryID)] = struct{}{}
	}
	return lookup
}

func validateOrgRoleAssignment(role auth.Role, warehouseID string, factoryID string, lookup topologyLookup) error {
	switch role {
	case auth.RoleAdmin:
		if warehouseID != "" || factoryID != "" {
			return fmt.Errorf("supplier_org_member_admin_cannot_bind_node")
		}
		return nil
	case auth.RoleWarehouseAdmin:
		if warehouseID == "" || factoryID != "" {
			return errMissingNodeAssignment
		}
		if _, ok := lookup.warehouses[warehouseID]; !ok {
			return errInvalidNodeAssignment
		}
		return nil
	case auth.RoleFactoryAdmin:
		if factoryID == "" || warehouseID != "" {
			return errMissingNodeAssignment
		}
		if _, ok := lookup.factories[factoryID]; !ok {
			return errInvalidNodeAssignment
		}
		return nil
	case auth.RolePayload:
		if (warehouseID == "" && factoryID == "") || (warehouseID != "" && factoryID != "") {
			return errMissingNodeAssignment
		}
		if warehouseID != "" {
			if _, ok := lookup.warehouses[warehouseID]; !ok {
				return errInvalidNodeAssignment
			}
			return nil
		}
		if _, ok := lookup.factories[factoryID]; !ok {
			return errInvalidNodeAssignment
		}
		return nil
	default:
		return errUnsupportedSupplierRole
	}
}

func validateHomeNode(homeNodeType string, homeNodeID string, lookup topologyLookup) (auth.HomeNodeType, string, error) {
	switch auth.HomeNodeType(strings.ToUpper(homeNodeType)) {
	case auth.HomeNodeWarehouse:
		if homeNodeID == "" {
			return "", "", errInvalidFleetHomeNode
		}
		if _, ok := lookup.warehouses[homeNodeID]; !ok {
			return "", "", errInvalidFleetHomeNode
		}
		return auth.HomeNodeWarehouse, homeNodeID, nil
	case auth.HomeNodeFactory:
		if homeNodeID == "" {
			return "", "", errInvalidFleetHomeNode
		}
		if _, ok := lookup.factories[homeNodeID]; !ok {
			return "", "", errInvalidFleetHomeNode
		}
		return auth.HomeNodeFactory, homeNodeID, nil
	default:
		return "", "", errInvalidFleetHomeNode
	}
}
