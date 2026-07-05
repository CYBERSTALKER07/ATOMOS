package driver

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/web"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// HandleCreateDriver serves POST /v1/drivers
func (s *Service) HandleCreateDriver(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		web.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if auth.RejectBodyScopeOverrides(w, r, body) {
		return
	}

	var req Driver
	if err := json.Unmarshal(body, &req); err != nil {
		web.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	supplierID, ok := auth.ResolveSupplierID(r.Context())
	if !ok || supplierID == "" {
		web.JSONError(w, "missing supplier scope", http.StatusUnauthorized)
		return
	}
	req.SupplierID = supplierID

	if req.Name == "" || req.Phone == "" {
		web.JSONError(w, "missing required fields", http.StatusBadRequest)
		return
	}

	if req.DriverID == "" {
		req.DriverID = uuid.New().String()
	}

	emit := func(buf outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), buf, events.AggregateDriver, req.DriverID, events.TopicMain, events.DriverEvent{
			BaseEvent:    events.BaseEvent{Type: events.EventDriverCreated, Version: 1},
			DriverID:     req.DriverID,
			SupplierID:   req.SupplierID,
			HomeNodeID:   req.HomeNodeID,
			HomeNodeType: req.HomeNodeType,
		})
	}
	if err := s.repo.CreateDriver(r.Context(), req, emit); err != nil {
		web.JSONError(w, "failed to create driver: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if s.cache != nil {
		s.cache.Invalidate(r.Context(), driverCacheKey(req.DriverID), driversListCacheKey(req.SupplierID))
	}

	web.JSONResponse(w, http.StatusCreated, req)
}

// HandleGetDriver serves GET /v1/drivers/{driverId}
func (s *Service) HandleGetDriver(w http.ResponseWriter, r *http.Request) {
	driverID := chi.URLParam(r, "driverId")
	if driverID == "" {
		web.JSONError(w, "missing driverId", http.StatusBadRequest)
		return
	}

	d, err := s.repo.GetDriver(r.Context(), driverID)
	if err != nil {
		web.JSONError(w, "failed to get driver: "+err.Error(), http.StatusInternalServerError)
		return
	}

	web.JSONResponse(w, http.StatusOK, d)
}

// HandleUpdateDriver serves PUT /v1/drivers/{driverId}
func (s *Service) HandleUpdateDriver(w http.ResponseWriter, r *http.Request) {
	driverID := chi.URLParam(r, "driverId")
	if driverID == "" {
		web.JSONError(w, "missing driverId", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		web.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if auth.RejectBodyScopeOverrides(w, r, body) {
		return
	}

	var req Driver
	if err := json.Unmarshal(body, &req); err != nil {
		web.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.DriverID = driverID

	supplierID, ok := auth.ResolveSupplierID(r.Context())
	if !ok || supplierID == "" {
		web.JSONError(w, "missing supplier scope", http.StatusUnauthorized)
		return
	}
	req.SupplierID = supplierID

	emit := func(buf outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), buf, events.AggregateDriver, req.DriverID, events.TopicMain, events.DriverEvent{
			BaseEvent:    events.BaseEvent{Type: events.EventDriverAvailabilityChanged, Version: 1},
			DriverID:     req.DriverID,
			SupplierID:   req.SupplierID,
			HomeNodeID:   req.HomeNodeID,
			HomeNodeType: req.HomeNodeType,
			Available:    req.IsActive,
			OnShift:      req.OnShift,
		})
	}
	if err := s.repo.UpdateDriver(r.Context(), req, emit); err != nil {
		web.JSONError(w, "failed to update driver: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if s.cache != nil {
		s.cache.Invalidate(r.Context(), driverCacheKey(req.DriverID), driversListCacheKey(req.SupplierID))
	}

	web.JSONResponse(w, http.StatusOK, req)
}

// HandleListDrivers serves GET /v1/drivers
func (s *Service) HandleListDrivers(w http.ResponseWriter, r *http.Request) {
	supplierID, ok := auth.ResolveSupplierID(r.Context())
	if !ok || supplierID == "" {
		web.JSONError(w, "missing supplier scope", http.StatusUnauthorized)
		return
	}

	if q := r.URL.Query().Get("supplier_id"); q != "" && q != supplierID {
		web.JSONError(w, "access denied: supplier scope violation", http.StatusForbidden)
		return
	}

	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	drivers, err := s.repo.ListDrivers(r.Context(), supplierID, limit, offset)
	if err != nil {
		web.JSONError(w, "failed to list drivers: "+err.Error(), http.StatusInternalServerError)
		return
	}

	web.JSONResponse(w, http.StatusOK, drivers)
}

// HandleCreateVehicle serves POST /v1/vehicles
func (s *Service) HandleCreateVehicle(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		web.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if auth.RejectBodyScopeOverrides(w, r, body) {
		return
	}

	var req Vehicle
	if err := json.Unmarshal(body, &req); err != nil {
		web.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	supplierID, ok := auth.ResolveSupplierID(r.Context())
	if !ok || supplierID == "" {
		web.JSONError(w, "missing supplier scope", http.StatusUnauthorized)
		return
	}
	req.SupplierID = supplierID

	if req.LicensePlate == "" {
		web.JSONError(w, "missing required fields", http.StatusBadRequest)
		return
	}

	if req.VehicleID == "" {
		req.VehicleID = uuid.New().String()
	}

	emit := func(buf outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), buf, events.AggregateVehicle, req.VehicleID, events.TopicMain, events.VehicleEvent{
			BaseEvent:    events.BaseEvent{Type: events.EventVehicleCreated, Version: 1},
			VehicleID:    req.VehicleID,
			SupplierID:   req.SupplierID,
			HomeNodeID:   req.HomeNodeID,
			HomeNodeType: req.HomeNodeType,
		})
	}
	if err := s.repo.CreateVehicle(r.Context(), req, emit); err != nil {
		web.JSONError(w, "failed to create vehicle: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if s.cache != nil {
		s.cache.Invalidate(r.Context(), vehicleCacheKey(req.VehicleID), vehiclesListCacheKey(req.SupplierID))
	}

	web.JSONResponse(w, http.StatusCreated, req)
}

// HandleGetVehicle serves GET /v1/vehicles/{vehicleId}
func (s *Service) HandleGetVehicle(w http.ResponseWriter, r *http.Request) {
	vehicleID := chi.URLParam(r, "vehicleId")
	if vehicleID == "" {
		web.JSONError(w, "missing vehicleId", http.StatusBadRequest)
		return
	}

	v, err := s.repo.GetVehicle(r.Context(), vehicleID)
	if err != nil {
		web.JSONError(w, "failed to get vehicle: "+err.Error(), http.StatusInternalServerError)
		return
	}

	web.JSONResponse(w, http.StatusOK, v)
}

// HandleUpdateVehicle serves PUT /v1/vehicles/{vehicleId}
func (s *Service) HandleUpdateVehicle(w http.ResponseWriter, r *http.Request) {
	vehicleID := chi.URLParam(r, "vehicleId")
	if vehicleID == "" {
		web.JSONError(w, "missing vehicleId", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		web.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if auth.RejectBodyScopeOverrides(w, r, body) {
		return
	}

	var req Vehicle
	if err := json.Unmarshal(body, &req); err != nil {
		web.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.VehicleID = vehicleID

	supplierID, ok := auth.ResolveSupplierID(r.Context())
	if !ok || supplierID == "" {
		web.JSONError(w, "missing supplier scope", http.StatusUnauthorized)
		return
	}
	req.SupplierID = supplierID

	emit := func(buf outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), buf, events.AggregateVehicle, req.VehicleID, events.TopicMain, events.VehicleEvent{
			BaseEvent:    events.BaseEvent{Type: events.EventVehicleAvailabilityChanged, Version: 1},
			VehicleID:    req.VehicleID,
			SupplierID:   req.SupplierID,
			HomeNodeID:   req.HomeNodeID,
			HomeNodeType: req.HomeNodeType,
			IsActive:     req.IsActive,
		})
	}
	if err := s.repo.UpdateVehicle(r.Context(), req, emit); err != nil {
		web.JSONError(w, "failed to update vehicle: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if s.cache != nil {
		s.cache.Invalidate(r.Context(), vehicleCacheKey(req.VehicleID), vehiclesListCacheKey(req.SupplierID))
	}

	web.JSONResponse(w, http.StatusOK, req)
}

// HandleListVehicles serves GET /v1/vehicles
func (s *Service) HandleListVehicles(w http.ResponseWriter, r *http.Request) {
	supplierID, ok := auth.ResolveSupplierID(r.Context())
	if !ok || supplierID == "" {
		web.JSONError(w, "missing supplier scope", http.StatusUnauthorized)
		return
	}

	if q := r.URL.Query().Get("supplier_id"); q != "" && q != supplierID {
		web.JSONError(w, "access denied: supplier scope violation", http.StatusForbidden)
		return
	}

	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	vehicles, err := s.repo.ListVehicles(r.Context(), supplierID, limit, offset)
	if err != nil {
		web.JSONError(w, "failed to list vehicles: "+err.Error(), http.StatusInternalServerError)
		return
	}

	web.JSONResponse(w, http.StatusOK, vehicles)
}

func driverCacheKey(driverID string) string {
	return "driver:" + driverID
}

func driversListCacheKey(supplierID string) string {
	return "drivers:list:" + supplierID
}

func vehicleCacheKey(vehicleID string) string {
	return "vehicle:" + vehicleID
}

func vehiclesListCacheKey(supplierID string) string {
	return "vehicles:list:" + supplierID
}
