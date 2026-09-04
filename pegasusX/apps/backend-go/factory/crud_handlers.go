package factory

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

// HandleCreateFactory serves POST /v1/factories
func (s *Service) HandleCreateFactory(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		web.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if auth.RejectBodyScopeOverrides(w, r, body) {
		return
	}

	var req Factory
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

	if req.FactoryID == "" {
		req.FactoryID = uuid.New().String()
	}
	applyFactoryCreateDefaults(&req)
	if err := stampFactoryEntity(r.Context(), &req); err != nil {
		if writeMarketLaw(w, err) {
			return
		}
		web.JSONError(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	emit := func(buf outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), buf, events.AggregateFactory, req.FactoryID, events.TopicMain, events.FactoryEvent{
			BaseEvent:  events.BaseEvent{Type: events.EventFactoryCreated, Version: 1},
			FactoryID:  req.FactoryID,
			SupplierID: req.SupplierID,
		})
	}
	if err := s.repo.CreateFactory(r.Context(), req, emit); err != nil {
		web.JSONError(w, "failed to create factory: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if s.cache != nil {
		s.cache.Invalidate(r.Context(), factoryCacheKey(req.FactoryID), factoriesListCacheKey(req.SupplierID))
	}

	web.JSONResponse(w, http.StatusCreated, req)
}

func applyFactoryCreateDefaults(f *Factory) {
	if f == nil {
		return
	}
	if f.DailyOutputCapacity <= 0 {
		f.DailyOutputCapacity = DefaultDailyOutputCapacity
	}
}

// HandleGetFactory serves GET /v1/factories/{factoryId}
func (s *Service) HandleGetFactory(w http.ResponseWriter, r *http.Request) {
	factoryID := chi.URLParam(r, "factoryId")
	if factoryID == "" {
		web.JSONError(w, "missing factoryId", http.StatusBadRequest)
		return
	}

	if _, ok := auth.FromContext(r.Context()); !ok {
		web.JSONError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	f, err := s.repo.GetFactory(r.Context(), factoryID)
	if err != nil {
		web.JSONError(w, "failed to get factory: "+err.Error(), http.StatusInternalServerError)
		return
	}

	allowed := auth.EntitySupplierAllowed(r.Context(), f.SupplierID) ||
		auth.HomeNodeMatches(r.Context(), factoryID, auth.HomeNodeFactory)
	if !allowed {
		web.JSONError(w, "factory_not_found", http.StatusNotFound)
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

	body, err := io.ReadAll(r.Body)
	if err != nil {
		web.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if auth.RejectBodyScopeOverrides(w, r, body) {
		return
	}

	var req Factory
	if err := json.Unmarshal(body, &req); err != nil {
		web.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.FactoryID = factoryID

	supplierID, ok := auth.ResolveSupplierID(r.Context())
	if !ok || supplierID == "" {
		web.JSONError(w, "missing supplier scope", http.StatusUnauthorized)
		return
	}
	req.SupplierID = supplierID
	if err := stampFactoryEntity(r.Context(), &req); err != nil {
		if writeMarketLaw(w, err) {
			return
		}
		web.JSONError(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	emit := func(buf outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), buf, events.AggregateFactory, req.FactoryID, events.TopicMain, events.FactoryEvent{
			BaseEvent:  events.BaseEvent{Type: events.EventFactoryLocationUpdated, Version: 1},
			FactoryID:  req.FactoryID,
			SupplierID: req.SupplierID,
		})
	}
	if err := s.repo.UpdateFactory(r.Context(), req, emit); err != nil {
		web.JSONError(w, "failed to update factory: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if s.cache != nil {
		s.cache.Invalidate(r.Context(), factoryCacheKey(req.FactoryID), factoriesListCacheKey(req.SupplierID))
	}

	web.JSONResponse(w, http.StatusOK, req)
}

// HandleListFactories serves GET /v1/factories
func (s *Service) HandleListFactories(w http.ResponseWriter, r *http.Request) {
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

	factories, err := s.repo.ListFactories(r.Context(), supplierID, limit, offset)
	if err != nil {
		web.JSONError(w, "failed to list factories: "+err.Error(), http.StatusInternalServerError)
		return
	}

	web.JSONResponse(w, http.StatusOK, factories)
}

func factoryCacheKey(factoryID string) string {
	return "factory:" + factoryID
}

func factoriesListCacheKey(supplierID string) string {
	return "factories:list:" + supplierID
}
