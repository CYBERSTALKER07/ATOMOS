package etaroutes

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/eta"
)

type Deps struct {
	Service *eta.Service
}

func RegisterRoutes(r chi.Router, d Deps) {
	r.Route("/v1/etas", func(r chi.Router) {
		r.Get("/route/{routeId}", handleGetRouteETAs(d.Service))
		r.Get("/stop/{stopId}", handleGetStopETA(d.Service))
		r.Post("/recalculate", handleRecalculate(d.Service))
	})
}

func handleGetRouteETAs(s *eta.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		routeId := chi.URLParam(r, "routeId")
		if routeId == "" {
			writeError(w, http.StatusBadRequest, "routeId is required")
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
		stopId := chi.URLParam(r, "stopId")
		if stopId == "" {
			writeError(w, http.StatusBadRequest, "stopId is required")
			return
		}
		eta, err := s.GetStopETA(r.Context(), stopId)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to get stop eta")
			return
		}
		if eta == nil {
			writeError(w, http.StatusNotFound, "stop eta not found")
			return
		}
		writeJSON(w, http.StatusOK, eta)
	}
}

func handleRecalculate(s *eta.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req eta.RecalculateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		
		if req.Now.IsZero() {
			req.Now = time.Now().UTC()
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
