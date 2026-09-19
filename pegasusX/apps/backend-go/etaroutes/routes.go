package etaroutes

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/eta"
)

type Deps struct {
	Service *eta.Service
}

func RegisterRoutes(r chi.Router, d Deps) {
	r.Route("/v1/etas", func(er chi.Router) {
		er.Group(func(rr chi.Router) {
			rr.Use(auth.RequireRole(
				auth.RoleAdmin,
				auth.RoleWarehouse,
				auth.RoleWarehouseAdmin,
				auth.RoleDriver,
				auth.RoleRetailer,
				auth.RolePayload,
				auth.RoleFactory,
				auth.RoleFactoryAdmin,
				auth.RolePlatformAdmin,
			))
			rr.Get("/route/{routeId}", handleGetRouteETAs(d.Service))
			rr.Get("/stop/{stopId}", handleGetStopETA(d.Service))
		})
		er.Group(func(wr chi.Router) {
			wr.Use(auth.RequireRole(
				auth.RoleAdmin,
				auth.RoleWarehouse,
				auth.RoleWarehouseAdmin,
				auth.RolePlatformAdmin,
			))
			wr.Post("/recalculate", handleRecalculate(d.Service))
		})
	})
}

func handleGetRouteETAs(s *eta.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s == nil {
			writeError(w, http.StatusServiceUnavailable, "eta_unavailable")
			return
		}
		routeId := chi.URLParam(r, "routeId")
		if routeId == "" {
			writeError(w, http.StatusBadRequest, "routeId is required")
			return
		}
		if err := s.AuthorizeRoute(r.Context(), routeId); err != nil {
			writeRouteAuthError(w, err)
			return
		}
		etas, err := s.GetRouteETAs(r.Context(), routeId)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to get route etas")
			return
		}
		writeJSON(w, http.StatusOK, etas)
	}
}

func handleGetStopETA(s *eta.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s == nil {
			writeError(w, http.StatusServiceUnavailable, "eta_unavailable")
			return
		}
		stopId := chi.URLParam(r, "stopId")
		if stopId == "" {
			writeError(w, http.StatusBadRequest, "stopId is required")
			return
		}
		row, err := s.GetStopETA(r.Context(), stopId)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to get stop eta")
			return
		}
		if row == nil {
			writeError(w, http.StatusNotFound, "stop eta not found")
			return
		}
		if err := s.AuthorizeRoute(r.Context(), row.RouteId); err != nil {
			writeRouteAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, row)
	}
}

func handleRecalculate(s *eta.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s == nil {
			writeError(w, http.StatusServiceUnavailable, "eta_unavailable")
			return
		}
		var req eta.RecalculateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.Now.IsZero() {
			req.Now = time.Now().UTC()
		}
		routeID := req.EffectiveRouteID()
		if routeID == "" {
			writeError(w, http.StatusBadRequest, "routeId is required")
			return
		}
		if err := s.AuthorizeRoute(r.Context(), routeID); err != nil {
			writeRouteAuthError(w, err)
			return
		}

		if err := s.RecalculateETAs(r.Context(), req); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to recalculate etas")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeRouteAuthError(w http.ResponseWriter, err error) {
	if errors.Is(err, eta.ErrRouteNotFound) {
		writeError(w, http.StatusNotFound, "route_not_found")
		return
	}
	writeError(w, http.StatusInternalServerError, "failed to authorize route")
}
