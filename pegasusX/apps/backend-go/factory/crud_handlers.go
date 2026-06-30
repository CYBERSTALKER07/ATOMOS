package factory

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/web"
)

// HandleCreateFactory serves POST /v1/factories
func (s *Service) HandleCreateFactory(w http.ResponseWriter, r *http.Request) {
	var req Factory
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.SupplierID == "" {
		web.JSONError(w, "missing required fields: name, supplier_id", http.StatusBadRequest)
		return
	}

	if req.FactoryID == "" {
		req.FactoryID = uuid.New().String()
	}

	if err := s.repo.CreateFactory(r.Context(), req); err != nil {
		web.JSONError(w, "failed to create factory: "+err.Error(), http.StatusInternalServerError)
		return
	}

	web.JSONResponse(w, http.StatusCreated, req)
}

// HandleGetFactory serves GET /v1/factories/{factoryId}
func (s *Service) HandleGetFactory(w http.ResponseWriter, r *http.Request) {
	factoryID := chi.URLParam(r, "factoryId")
	if factoryID == "" {
		web.JSONError(w, "missing factoryId", http.StatusBadRequest)
		return
	}

	f, err := s.repo.GetFactory(r.Context(), factoryID)
	if err != nil {
		web.JSONError(w, "failed to get factory: "+err.Error(), http.StatusInternalServerError)
		return
	}

	web.JSONResponse(w, http.StatusOK, f)
}

// HandleUpdateFactory serves PUT /v1/factories/{factoryId}
func (s *Service) HandleUpdateFactory(w http.ResponseWriter, r *http.Request) {
	factoryID := chi.URLParam(r, "factoryId")
	if factoryID == "" {
		web.JSONError(w, "missing factoryId", http.StatusBadRequest)
		return
	}

	var req Factory
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.FactoryID = factoryID

	if err := s.repo.UpdateFactory(r.Context(), req); err != nil {
		web.JSONError(w, "failed to update factory: "+err.Error(), http.StatusInternalServerError)
		return
	}

	web.JSONResponse(w, http.StatusOK, req)
}

// HandleListFactories serves GET /v1/factories
func (s *Service) HandleListFactories(w http.ResponseWriter, r *http.Request) {
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

	factories, err := s.repo.ListFactories(r.Context(), supplierID, limit, offset)
	if err != nil {
		web.JSONError(w, "failed to list factories: "+err.Error(), http.StatusInternalServerError)
		return
	}

	web.JSONResponse(w, http.StatusOK, factories)
}
