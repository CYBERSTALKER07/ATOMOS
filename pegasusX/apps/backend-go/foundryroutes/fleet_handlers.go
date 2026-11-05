package foundryroutes

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/foundry"
)

type fleetHandler struct {
	eng *foundry.Engine
}

func newFleetHandler(eng *foundry.Engine) *fleetHandler {
	return &fleetHandler{eng: eng}
}

func (h *fleetHandler) mountRoutes(r chi.Router) {
	r.Post("/v1/foundry/fleet/dvir", h.handleSubmitDVIR)
	r.Post("/v1/foundry/fleet/gate-events", h.handleGateEvent)
	r.Post("/v1/foundry/fleet/mid-route-rescue", h.handleMidRouteRescue)
	r.Post("/v1/foundry/fleet/predictive-reefer", h.handlePredictiveReefer)
	r.Get("/v1/foundry/fleet/vehicles/{vehicleID}/dispatch-check", h.handleDispatchCheck)
}

func (h *fleetHandler) handleSubmitDVIR(w http.ResponseWriter, r *http.Request) {
	var req foundry.FleetDVIRInspection
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
		return
	}
	res, err := h.eng.SubmitDVIRInspection(r.Context(), req)
	if err != nil {
		writeError(w, err)
		return
	}
	if res.RedTagApplied {
		writeJSON(w, http.StatusConflict, map[string]any{
			"code":                  "VEHICLE_GROUNDED_RED_TAG",
			"error":                 "vehicle red-tagged: grounded due to critical defects",
			"inspection_id":         res.InspectionID,
			"vehicle_id":            res.VehicleID,
			"critical_defect_found": res.CriticalDefectFound,
			"red_tag_applied":       res.RedTagApplied,
			"failed_points":         res.FailedPoints,
			"status":                res.Status,
			"inspection":            res,
		})
		return
	}
	writeJSON(w, http.StatusCreated, res)
}

func (h *fleetHandler) handleGateEvent(w http.ResponseWriter, r *http.Request) {
	var req foundry.YardGateCheckEvent
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
		return
	}
	ev, err := h.eng.RecordGateCheckEvent(r.Context(), req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, ev)
}

func (h *fleetHandler) handleMidRouteRescue(w http.ResponseWriter, r *http.Request) {
	var req foundry.MidRouteRescueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
		return
	}
	res, err := h.eng.ExecuteMidRouteBreakdownRescue(r.Context(), req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *fleetHandler) handlePredictiveReefer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		VehicleID                string  `json:"vehicle_id"`
		CoolingRateDegPerMin     float64 `json:"cooling_rate_deg_per_min"`
		MinRequiredRateDegPerMin float64 `json:"min_required_rate_deg_per_min"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
		return
	}
	hold, err := h.eng.EvaluatePredictiveReeferHealth(r.Context(), req.VehicleID, req.CoolingRateDegPerMin, req.MinRequiredRateDegPerMin)
	if err != nil {
		writeError(w, err)
		return
	}
	status := http.StatusOK
	if hold {
		status = http.StatusConflict
	}
	writeJSON(w, status, map[string]any{
		"vehicle_id":                   strings.TrimSpace(req.VehicleID),
		"pre_failure_maintenance_hold": hold,
	})
}

func (h *fleetHandler) handleDispatchCheck(w http.ResponseWriter, r *http.Request) {
	vehicleID := chi.URLParam(r, "vehicleID")
	if err := h.eng.AssertVehicleDispatchAuthorized(r.Context(), vehicleID); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"vehicle_id":          vehicleID,
		"dispatch_authorized": true,
	})
}
