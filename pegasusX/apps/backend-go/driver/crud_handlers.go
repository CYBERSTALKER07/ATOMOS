package driver

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/web"
)

// HandleCreateDriver serves POST /v1/drivers
func (s *Service) HandleCreateDriver(w http.ResponseWriter, r *http.Request) {
	var req Driver
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Phone == "" || req.SupplierID == "" {
		web.JSONError(w, "missing required fields", http.StatusBadRequest)
		return
	}

	if req.DriverID == "" {
		req.DriverID = uuid.New().String()
	}

	if err := s.repo.CreateDriver(r.Context(), req); err != nil {
		web.JSONError(w, "failed to create driver: "+err.Error(), http.StatusInternalServerError)
		return
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

	var req Driver
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.DriverID = driverID

	if err := s.repo.UpdateDriver(r.Context(), req); err != nil {
		web.JSONError(w, "failed to update driver: "+err.Error(), http.StatusInternalServerError)
		return
	}

	web.JSONResponse(w, http.StatusOK, req)
}

// HandleListDrivers serves GET /v1/drivers
func (s *Service) HandleListDrivers(w http.ResponseWriter, r *http.Request) {
	supplierID := r.URL.Query().Get("supplier_id")
	if supplierID == "" {
		web.JSONError(w, "missing supplier_id query parameter", http.StatusBadRequest)
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
	var req Vehicle
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.LicensePlate == "" || req.SupplierID == "" {
		web.JSONError(w, "missing required fields", http.StatusBadRequest)
		return
	}

	if req.VehicleID == "" {
		req.VehicleID = uuid.New().String()
	}

	if err := s.repo.CreateVehicle(r.Context(), req); err != nil {
		web.JSONError(w, "failed to create vehicle: "+err.Error(), http.StatusInternalServerError)
		return
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

	var req Vehicle
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.VehicleID = vehicleID

	if err := s.repo.UpdateVehicle(r.Context(), req); err != nil {
		web.JSONError(w, "failed to update vehicle: "+err.Error(), http.StatusInternalServerError)
		return
	}

	web.JSONResponse(w, http.StatusOK, req)
}

// HandleListVehicles serves GET /v1/vehicles
func (s *Service) HandleListVehicles(w http.ResponseWriter, r *http.Request) {
	supplierID := r.URL.Query().Get("supplier_id")
	if supplierID == "" {
		web.JSONError(w, "missing supplier_id query parameter", http.StatusBadRequest)
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
