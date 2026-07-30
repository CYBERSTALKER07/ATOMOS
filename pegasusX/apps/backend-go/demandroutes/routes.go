package demandroutes

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/demand"
)

type Deps struct {
	Service *demand.Service
}

func RegisterRoutes(r chi.Router, d Deps) {
	r.Route("/v1/demand", func(r chi.Router) {
		r.Get("/adjustments", handleGetAdjustments(d.Service))
		r.Get("/signals", handleGetSignals(d.Service))
		r.Post("/signals", handleCreateSignal(d.Service))
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

func handleGetSignals(s *demand.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		signals, err := s.GetSignals(ctx)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"signals": signals,
		})
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

		if req.Type == "" || req.Scope == "" || req.Multiplier <= 0 {
			writeError(w, http.StatusBadRequest, "missing required fields or invalid multiplier")
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

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
