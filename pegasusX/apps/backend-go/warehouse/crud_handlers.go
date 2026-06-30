package warehouse

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/web"
)

// HandleCreateWarehouse serves POST /v1/warehouses
func (s *Service) HandleCreateWarehouse(w http.ResponseWriter, r *http.Request) {
	var req Warehouse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.SupplierID == "" {
		web.JSONError(w, "missing required fields: name, supplier_id", http.StatusBadRequest)
		return
	}

	if req.WarehouseID == "" {
		req.WarehouseID = uuid.New().String()
	}

	if req.TransferMode == "" {
		req.TransferMode = "TRUCK"
	}
	
	if req.DefaultOutOfStockPolicy == "" {
		req.DefaultOutOfStockPolicy = "REJECT"
	}

	if err := s.repo.CreateWarehouse(r.Context(), req); err != nil {
		web.JSONError(w, "failed to create warehouse: "+err.Error(), http.StatusInternalServerError)
		return
	}

	web.JSONResponse(w, http.StatusCreated, req)
}

// HandleGetWarehouse serves GET /v1/warehouses/{warehouseId}
func (s *Service) HandleGetWarehouse(w http.ResponseWriter, r *http.Request) {
	warehouseID := chi.URLParam(r, "warehouseId")
	if warehouseID == "" {
		web.JSONError(w, "missing warehouseId", http.StatusBadRequest)
		return
	}

	wh, err := s.repo.GetWarehouse(r.Context(), warehouseID)
	if err != nil {
		web.JSONError(w, "failed to get warehouse: "+err.Error(), http.StatusInternalServerError)
		return
	}

	web.JSONResponse(w, http.StatusOK, wh)
}

// HandleUpdateWarehouse serves PUT /v1/warehouses/{warehouseId}
func (s *Service) HandleUpdateWarehouse(w http.ResponseWriter, r *http.Request) {
	warehouseID := chi.URLParam(r, "warehouseId")
	if warehouseID == "" {
		web.JSONError(w, "missing warehouseId", http.StatusBadRequest)
		return
	}

	var req Warehouse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.WarehouseID = warehouseID

	if err := s.repo.UpdateWarehouse(r.Context(), req); err != nil {
		web.JSONError(w, "failed to update warehouse: "+err.Error(), http.StatusInternalServerError)
		return
	}

	web.JSONResponse(w, http.StatusOK, req)
}

// HandleListWarehouses serves GET /v1/warehouses
func (s *Service) HandleListWarehouses(w http.ResponseWriter, r *http.Request) {
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

	warehouses, err := s.repo.ListWarehouses(r.Context(), supplierID, limit, offset)
	if err != nil {
		web.JSONError(w, "failed to list warehouses: "+err.Error(), http.StatusInternalServerError)
		return
	}

	web.JSONResponse(w, http.StatusOK, warehouses)
}
