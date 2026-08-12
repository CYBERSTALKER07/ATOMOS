package warehouse

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

// HandleCreateWarehouse serves POST /v1/warehouses
func (s *Service) HandleCreateWarehouse(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		web.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if auth.RejectBodyScopeOverrides(w, r, body) {
		return
	}

	var req Warehouse
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

	if req.Name == "" {
		web.JSONError(w, "missing required fields: name", http.StatusBadRequest)
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

	emit := func(buf outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), buf, events.AggregateWarehouse, req.WarehouseID, events.TopicMain, events.WarehouseEvent{
			BaseEvent:   events.BaseEvent{Type: events.EventWarehouseCreated, Version: 1},
			WarehouseID: req.WarehouseID,
			SupplierID:  req.SupplierID,
		})
	}
	if err := s.repo.CreateWarehouse(r.Context(), req, emit); err != nil {
		web.JSONError(w, "failed to create warehouse: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if s.cache != nil {
		s.cache.Invalidate(r.Context(), warehouseCacheKey(req.WarehouseID), warehousesListCacheKey(req.SupplierID))
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

	if _, ok := auth.FromContext(r.Context()); !ok {
		web.JSONError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	wh, err := s.repo.GetWarehouse(r.Context(), warehouseID)
	if err != nil {
		web.JSONError(w, "failed to get warehouse: "+err.Error(), http.StatusInternalServerError)
		return
	}

	allowed := auth.EntitySupplierAllowed(r.Context(), wh.SupplierID) ||
		auth.HomeNodeMatches(r.Context(), warehouseID, auth.HomeNodeWarehouse)
	if !allowed {
		web.JSONError(w, "warehouse_not_found", http.StatusNotFound)
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

	body, err := io.ReadAll(r.Body)
	if err != nil {
		web.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if auth.RejectBodyScopeOverrides(w, r, body) {
		return
	}

	var req Warehouse
	if err := json.Unmarshal(body, &req); err != nil {
		web.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.WarehouseID = warehouseID

	supplierID, ok := auth.ResolveSupplierID(r.Context())
	if !ok || supplierID == "" {
		web.JSONError(w, "missing supplier scope", http.StatusUnauthorized)
		return
	}
	req.SupplierID = supplierID

	emit := func(buf outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), buf, events.AggregateWarehouse, req.WarehouseID, events.TopicMain, events.WarehouseEvent{
			BaseEvent:   events.BaseEvent{Type: events.EventWarehouseLocationUpdated, Version: 1},
			WarehouseID: req.WarehouseID,
			SupplierID:  req.SupplierID,
		})
	}
	if err := s.repo.UpdateWarehouse(r.Context(), req, emit); err != nil {
		web.JSONError(w, "failed to update warehouse: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if s.cache != nil {
		s.cache.Invalidate(r.Context(), warehouseCacheKey(req.WarehouseID), warehousesListCacheKey(req.SupplierID))
	}

	web.JSONResponse(w, http.StatusOK, req)
}

// HandleListWarehouses serves GET /v1/warehouses
func (s *Service) HandleListWarehouses(w http.ResponseWriter, r *http.Request) {
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

	warehouses, err := s.repo.ListWarehouses(r.Context(), supplierID, limit, offset)
	if err != nil {
		web.JSONError(w, "failed to list warehouses: "+err.Error(), http.StatusInternalServerError)
		return
	}

	web.JSONResponse(w, http.StatusOK, warehouses)
}

func warehouseCacheKey(warehouseID string) string {
	return "warehouse:" + warehouseID
}

func warehousesListCacheKey(supplierID string) string {
	return "warehouses:list:" + supplierID
}
