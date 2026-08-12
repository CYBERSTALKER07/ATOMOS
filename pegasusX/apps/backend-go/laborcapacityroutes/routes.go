package laborcapacityroutes

import (
	"encoding/json"
	"net/http"
	"time"

	"cloud.google.com/go/civil"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/laborcapacity"
)

// Deps holds dependencies for labor capacity routes.
type Deps struct {
	Service *laborcapacity.Service
}

// RegisterRoutes registers labor capacity API endpoints (supplier/warehouse JWT).
func RegisterRoutes(r chi.Router, d Deps) {
	r.Route("/v1/labor-capacity", func(lr chi.Router) {
		lr.Use(auth.RequireRole(
			auth.RoleAdmin,
			auth.RoleWarehouseAdmin,
			auth.RoleWarehouse,
			auth.RolePlatformAdmin,
		))
		lr.Get("/driver-score/{driverId}", handleGetDriverScore(d.Service))
		lr.Get("/zone-capacity", handleGetZoneCapacity(d.Service))
		lr.Post("/driver-availability", handleSetDriverAvailability(d.Service))
	})
}

func handleGetDriverScore(s *laborcapacity.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		driverID := chi.URLParam(r, "driverId")
		if driverID == "" {
			writeError(w, http.StatusBadRequest, "driverId is required")
			return
		}
		score, err := s.GetDriverScore(r.Context(), driverID)
		if err != nil {
			writeError(w, http.StatusNotFound, "driver score not found")
			return
		}
		writeJSON(w, http.StatusOK, score)
	}
}

func handleGetZoneCapacity(s *laborcapacity.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dateStr := r.URL.Query().Get("date")
		if dateStr == "" {
			dateStr = civil.DateOf(time.Now().UTC()).String()
		}
		date, err := civil.ParseDate(dateStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid date format, use YYYY-MM-DD")
			return
		}

		zoneH3 := r.URL.Query().Get("zoneH3")
		if zoneH3 != "" {
			zc, err := s.GetZoneCapacity(r.Context(), zoneH3, date)
			if err != nil {
				writeError(w, http.StatusNotFound, "zone capacity not found")
				return
			}
			writeJSON(w, http.StatusOK, zc)
			return
		}

		zones, err := s.ListZoneCapacities(r.Context(), date)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"zones": zones})
	}
}

func handleSetDriverAvailability(s *laborcapacity.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req laborcapacity.SetAvailabilityRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request payload")
			return
		}
		if req.DriverId == "" || req.Date == "" || req.Status == "" {
			writeError(w, http.StatusBadRequest, "driverId, date, and status are required")
			return
		}
		if err := s.SetDriverAvailability(r.Context(), req); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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
