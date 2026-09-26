package demandroutes

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/demand"
)

type Deps struct {
	Service *demand.Service
}

func RegisterRoutes(r chi.Router, d Deps) {
	guard := auth.RequireRole(
		auth.RoleAdmin,
		auth.RolePlatformAdmin,
		auth.RoleWarehouseAdmin,
		auth.RoleWarehouse,
		auth.RoleFactoryAdmin,
		auth.RoleFactory,
	)
	r.With(guard).Get("/v1/demand/signals", handleListSignals(d.Service))
	r.With(guard).Post("/v1/demand/signals", handleCreateSignal(d.Service))
	r.Route("/v1/demand", func(r chi.Router) {
		r.With(guard).Get("/adjustments", handleGetAdjustments(d.Service))

		r.Route("/signals", func(r chi.Router) {
			r.With(guard).Get("/", handleListSignals(d.Service))
			r.With(guard).Post("/", handleCreateSignal(d.Service))

			r.Route("/{id}", func(r chi.Router) {
				r.With(guard).Get("/", handleGetSignal(d.Service))
				r.With(guard).Patch("/", handleUpdateSignal(d.Service))
				r.With(guard).Post("/deactivate", handleDeactivateSignal(d.Service))
			})
		})
	})
}

func handleGetAdjustments(s *demand.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		retailerId := r.URL.Query().Get("retailerId")
		if retailerId == "" {
			writeError(w, http.StatusBadRequest, "retailerId is required")
			return
		}

		fromStr := r.URL.Query().Get("from")
		toStr := r.URL.Query().Get("to")
		var from, to time.Time
		var err error

		if fromStr != "" {
			from, err = time.Parse("2006-01-02", fromStr)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid from date")
				return
			}
		} else {
			from = time.Now().AddDate(0, 0, -14)
		}

		if toStr != "" {
			to, err = time.Parse("2006-01-02", toStr)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid to date")
				return
			}
		} else {
			to = time.Now().AddDate(0, 0, 14)
		}

		adjs, err := s.GetAdjustments(ctx, retailerId, from, to)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"adjustments": adjs,
		})
	}
}

func handleListSignals(s *demand.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		
		filter := demand.SignalFilter{}
		if t := r.URL.Query().Get("type"); t != "" {
			st := demand.SignalType(t)
			filter.Type = &st
		}
		if scope := r.URL.Query().Get("scope"); scope != "" {
			filter.Scope = &scope
		}
		if activeStr := r.URL.Query().Get("active"); activeStr == "true" {
			active := true
			filter.Active = &active
		}

		signals, err := s.ListSignals(ctx, filter)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, signals)
	}
}

func handleGetSignal(s *demand.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		id := chi.URLParam(r, "id")

		sig, err := s.GetSignal(ctx, id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if sig == nil {
			writeError(w, http.StatusNotFound, "signal not found")
			return
		}

		writeJSON(w, http.StatusOK, sig)
	}
}

func handleCreateSignal(s *demand.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var req demand.CreateSignalRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request payload")
			return
		}

		if req.Multiplier < 0.5 || req.Multiplier > 2.5 {
			writeError(w, http.StatusBadRequest, "multiplier must be between 0.5 and 2.5")
			return
		}
		if !req.StartAt.Before(req.EndAt) {
			writeError(w, http.StatusBadRequest, "startAt must be before endAt")
			return
		}
		if req.Type != demand.SignalPromo && req.Type != demand.SignalEvent && req.Type != demand.SignalPayday && req.Type != demand.SignalEventDensity && req.Type != demand.SignalCompetitorPressure {
			writeError(w, http.StatusBadRequest, "unsupported signal type")
			return
		}

		// Simple pseudo-auth for 'createdBy'
		createdBy := r.Header.Get("X-User-Id")
		if createdBy == "" {
			createdBy = "system"
		}

		sig, err := s.CreateSignal(ctx, req, createdBy)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusCreated, sig)
	}
}

func handleUpdateSignal(s *demand.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		id := chi.URLParam(r, "id")

		sig, err := s.GetSignal(ctx, id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if sig == nil {
			writeError(w, http.StatusNotFound, "signal not found")
			return
		}

		var req demand.CreateSignalRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.Multiplier < 0.5 || req.Multiplier > 2.5 {
			writeError(w, http.StatusBadRequest, "multiplier must be between 0.5 and 2.5")
			return
		}
		if !req.StartAt.Before(req.EndAt) {
			writeError(w, http.StatusBadRequest, "startAt must be before endAt")
			return
		}

		sig.Type = req.Type
		sig.Scope = req.Scope
		sig.Sku = req.Sku
		sig.StartAt = req.StartAt
		sig.EndAt = req.EndAt
		sig.Multiplier = req.Multiplier
		sig.Meta = req.Meta

		if err := s.UpdateSignal(ctx, sig); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, sig)
	}
}

func handleDeactivateSignal(s *demand.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		id := chi.URLParam(r, "id")

		actor := r.Header.Get("X-User-Id")
		if actor == "" {
			actor = "system"
		}

		if err := s.DeactivateSignal(ctx, id, actor); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "deactivated"})
	}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
