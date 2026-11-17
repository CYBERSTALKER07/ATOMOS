package foundryroutes

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/foundry"
)

func isAllowedFleetRole(role auth.Role) bool {
	switch role {
	case auth.RoleWarehouse, auth.RoleWarehouseAdmin, auth.RoleAdmin, auth.RolePayload, auth.RoleDriver, auth.RolePlatformAdmin:
		return true
	default:
		return false
	}
}

type fleetHandler struct {
	eng *foundry.Engine
}

func newFleetHandler(eng *foundry.Engine) *fleetHandler {
	return &fleetHandler{eng: eng}
}

func (h *fleetHandler) requireFleetRole(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.FromContext(r.Context())
		if !ok || !isAllowedFleetRole(claims.Role) {
			writeJSON(w, http.StatusForbidden, map[string]any{"error": "forbidden: insufficient fleet role"})
			return
		}
		next(w, r)
	}
}

func (h *fleetHandler) mountRoutes(r chi.Router) {
	r.Post("/v1/foundry/fleet/dvir", h.requireFleetRole(h.handleSubmitDVIR))
	r.Post("/v1/foundry/fleet/gate-events", h.requireFleetRole(h.handleGateEvent))
	r.Post("/v1/foundry/fleet/mid-route-rescue", h.requireFleetRole(h.handleMidRouteRescue))
	r.Get("/v1/foundry/fleet/vehicles/{vehicleID}/dispatch-check", h.requireFleetRole(h.handleDispatchCheck))
}

func (h *fleetHandler) handleSubmitDVIR(w http.ResponseWriter, r *http.Request) {
	var req foundry.FleetDVIRInspection
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
		return
	}
	sid, ok := bindSupplierID(w, r, req.SupplierID)
	if !ok {
		return
	}
	req.SupplierID = sid
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
	sid, ok := bindSupplierID(w, r, req.SupplierID)
	if !ok {
		return
	}
	req.SupplierID = sid
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
	sid, ok := bindSupplierID(w, r, req.SupplierID)
	if !ok {
		return
	}
	req.SupplierID = sid
	res, err := h.eng.ExecuteMidRouteBreakdownRescue(r.Context(), req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
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
