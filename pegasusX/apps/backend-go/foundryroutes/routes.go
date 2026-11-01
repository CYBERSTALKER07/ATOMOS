// Package foundryroutes mounts HTTP endpoints for the PegasusX Foundry Semantic-Kinetic-Dynamic
// Triad, Role 7 (Depot Cashier & AML Vault), Role 9 (Fleet, Yard & DVIR Red-Tag), and the
// 12 Cross-Role Pre-Execution Edge-Case Guards.
package foundryroutes

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/foundry"
)

// Deps bundles dependencies for the Foundry HTTP router.
type Deps struct {
	Engine *foundry.Engine
}

// RegisterRoutes mounts all /v1/foundry/* endpoints onto the chi router.
func RegisterRoutes(r chi.Router, deps Deps) {
	eng := deps.Engine
	if eng == nil {
		eng = foundry.NewEngine()
	}
	h := &handler{eng: eng}

	r.Post("/v1/foundry/ontology/objects", h.handleUpsertObject)
	r.Get("/v1/foundry/ontology/objects/{cellID}/{objectType}/{objectID}", h.handleGetObject)
	r.Post("/v1/foundry/ontology/links", h.handleLinkObjects)
	r.Post("/v1/foundry/ontology/search-around", h.handleSearchAround)
	r.Post("/v1/foundry/ontology/lot-recall", h.handleLotRecall)

	r.Post("/v1/foundry/kinetic/actions/execute", h.handleExecuteAction)

	r.Post("/v1/foundry/scenarios/{scenarioID}/patches", h.handleApplyScenarioPatch)
	r.Get("/v1/foundry/scenarios/{scenarioID}/resolve/{cellID}/{objectType}/{objectID}", h.handleResolveScenarioObject)
	r.Post("/v1/foundry/scenarios/{scenarioID}/commit", h.handleCommitScenario)

	r.Post("/v1/foundry/cashier/sessions", h.handleStartCashierSession)
	r.Post("/v1/foundry/cashier/sessions/{sessionID}/blind-count", h.handleSubmitBlindCount)
	r.Post("/v1/foundry/cashier/reclassify-tender", h.handleReclassifyTenderRRN)
	r.Post("/v1/foundry/cashier/vault/cit-handover", h.handleCITHandover)

	r.Post("/v1/foundry/fleet/dvir", h.handleSubmitDVIR)
	r.Post("/v1/foundry/fleet/gate-events", h.handleGateEvent)
	r.Post("/v1/foundry/fleet/mid-route-rescue", h.handleMidRouteRescue)
	r.Post("/v1/foundry/fleet/predictive-reefer", h.handlePredictiveReefer)
	r.Get("/v1/foundry/fleet/vehicles/{vehicleID}/dispatch-check", h.handleDispatchCheck)

	r.Post("/v1/foundry/guards/split-tender-reconcile", h.handleSplitTenderReconcile)
	r.Post("/v1/foundry/guards/aml-check", h.handleAMLCheck)
	r.Post("/v1/foundry/guards/offline-lease/issue", h.handleIssueOfflineLease)
	r.Post("/v1/foundry/guards/offline-lease/validate", h.handleValidateOfflineLease)
	r.Post("/v1/foundry/guards/fiscal-preflight", h.handleFiscalPreFlight)
	r.Post("/v1/foundry/guards/slice-payout-eligibility", h.handleSlicePayoutEligibility)
	r.Post("/v1/foundry/guards/clawback-slice", h.handleClawbackSlice)
	r.Post("/v1/foundry/guards/load-3d-rti", h.handleLoad3DRTI)
	r.Post("/v1/foundry/guards/fx-corridor", h.handleFXCorridor)
	r.Post("/v1/foundry/guards/tote-state", h.handleRegisterToteState)
	r.Post("/v1/foundry/guards/tote-takeover", h.handleToteTakeover)
}

type handler struct {
	eng *foundry.Engine
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, err error) {
	var recErr *foundry.RecountRequiredError
	if errors.As(err, &recErr) {
		writeJSON(w, http.StatusConflict, map[string]any{
			"code":       "RECOUNT_REQUIRED",
			"error":      recErr.Error(),
			"session":    recErr.Session,
			"session_id": recErr.Session.SessionID,
			"status":     recErr.Session.Status,
		})
		return
	}

	var amlErr *foundry.AMLDailyCashLimitExceededError
	if errors.As(err, &amlErr) {
		writeJSON(w, http.StatusConflict, map[string]any{
			"code":                   "AML_DAILY_CASH_LIMIT_EXCEEDED",
			"error":                  amlErr.Error(),
			"legal_entity_tin":       amlErr.LegalEntityTIN,
			"current_exposure_minor": amlErr.CurrentExposureMinor,
			"attempted_cash_minor":   amlErr.AttemptedCashMinor,
			"ceiling_minor":          amlErr.CeilingMinor,
			"calendar_date":          amlErr.CalendarDate,
			"advice":                 amlErr.Advice,
		})
		return
	}

	var redTagErr *foundry.VehicleGroundedRedTagError
	if errors.As(err, &redTagErr) {
		writeJSON(w, http.StatusConflict, map[string]any{
			"code":             "VEHICLE_GROUNDED_RED_TAG",
			"error":            redTagErr.Error(),
			"inspection_id":    redTagErr.InspectionID,
			"vehicle_id":       redTagErr.VehicleID,
			"critical_defects": redTagErr.CriticalDefects,
			"reason":           redTagErr.Reason,
		})
		return
	}

	var toteErr *foundry.ToteTakeoverNotPermittedError
	if errors.As(err, &toteErr) {
		writeJSON(w, http.StatusConflict, map[string]any{
			"code":                   "TOTE_TAKEOVER_NOT_PERMITTED",
			"error":                  toteErr.Error(),
			"tote_id":                toteErr.ToteID,
			"current_picker_id":      toteErr.CurrentPickerID,
			"last_scan_heartbeat_at": toteErr.LastScanHeartbeatAt,
			"idle_duration_seconds":  int64(toteErr.IdleDuration / time.Second),
			"remaining_idle_seconds": toteErr.RemainingIdleSeconds,
		})
		return
	}

	if errors.Is(err, foundry.ErrTransactionMutationLimitExceeded) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"code":  "TRANSACTION_MUTATION_LIMIT_EXCEEDED",
			"error": err.Error(),
		})
		return
	}

	status := http.StatusBadRequest
	switch {
	case errors.Is(err, foundry.ErrObjectNotFound):
		status = http.StatusNotFound
	case errors.Is(err, foundry.ErrScenarioWatermarkDrift),
		errors.Is(err, foundry.ErrDuplicateBagSeal),
		errors.Is(err, foundry.ErrBlindRecountRequired),
		errors.Is(err, foundry.ErrAMLCeilingExceeded),
		errors.Is(err, foundry.ErrOfflineLeaseExceeded),
		errors.Is(err, foundry.ErrVehicleRedTagged),
		errors.Is(err, foundry.ErrGateOutboundSealInvalid),
		errors.Is(err, foundry.ErrFiscalPreFlightBlocked),
		errors.Is(err, foundry.ErrSlicePayoutUnfiscalized),
		errors.Is(err, foundry.ErrSliceImmutableAfterBatch),
		errors.Is(err, foundry.ErrCrushabilityViolation),
		errors.Is(err, foundry.ErrFXReconfirmationRequired),
		errors.Is(err, foundry.ErrToteHeartbeatActive):
		status = http.StatusConflict
	case errors.Is(err, foundry.ErrInvalidRRNProof),
		errors.Is(err, foundry.ErrPreconditionFailed),
		errors.Is(err, foundry.ErrInvalidObjectIdentity):
		status = http.StatusUnprocessableEntity
	}
	writeJSON(w, status, map[string]any{"error": err.Error()})
}

func (h *handler) handleUpsertObject(w http.ResponseWriter, r *http.Request) {
	var req foundry.OntologyObject
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
		return
	}
	obj, err := h.eng.UpsertObject(r.Context(), req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, obj)
}

func (h *handler) handleGetObject(w http.ResponseWriter, r *http.Request) {
	cellID := chi.URLParam(r, "cellID")
	objectType := chi.URLParam(r, "objectType")
	objectID := chi.URLParam(r, "objectID")
	obj, err := h.eng.GetObject(r.Context(), cellID, objectType, objectID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, obj)
}

func (h *handler) handleLinkObjects(w http.ResponseWriter, r *http.Request) {
	var req foundry.OntologyGraphEdge
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
		return
	}
	edge, err := h.eng.LinkObjects(r.Context(), req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, edge)
}

func (h *handler) handleSearchAround(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CellID           string   `json:"cell_id"`
		StartObjectType  string   `json:"start_object_type"`
		StartObjectIDs   []string `json:"start_object_ids"`
		AllowedLinkTypes []string `json:"allowed_link_types"`
		MaxHops          int      `json:"max_hops"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
		return
	}
	objects, err := h.eng.SearchAround(r.Context(), req.CellID, req.StartObjectType, req.StartObjectIDs, req.AllowedLinkTypes, req.MaxHops)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"objects": objects})
}

func (h *handler) handleLotRecall(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CellID           string `json:"cell_id"`
		SupplierID       string `json:"supplier_id"`
		RawMaterialLotID string `json:"raw_material_lot_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
		return
	}
	report, err := h.eng.TriggerContaminatedLotRecall(r.Context(), req.CellID, req.SupplierID, req.RawMaterialLotID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (h *handler) handleExecuteAction(w http.ResponseWriter, r *http.Request) {
	var req foundry.ActionExecutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
		return
	}
	rec, err := h.eng.ExecuteKineticAction(r.Context(), req, func(ctx context.Context) (map[string]any, error) {
		return req.Parameters, nil
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

func (h *handler) handleApplyScenarioPatch(w http.ResponseWriter, r *http.Request) {
	scenarioID := chi.URLParam(r, "scenarioID")
	var req struct {
		CellID          string         `json:"cell_id"`
		SupplierID      string         `json:"supplier_id"`
		ObjectType      string         `json:"object_type"`
		ObjectID        string         `json:"object_id"`
		Operation       string         `json:"operation"`
		PatchProperties map[string]any `json:"patch_properties"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
		return
	}
	patch, err := h.eng.ApplyScenarioPatch(r.Context(), req.CellID, scenarioID, req.SupplierID, req.ObjectType, req.ObjectID, req.Operation, req.PatchProperties)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, patch)
}

func (h *handler) handleResolveScenarioObject(w http.ResponseWriter, r *http.Request) {
	scenarioID := chi.URLParam(r, "scenarioID")
	cellID := chi.URLParam(r, "cellID")
	objectType := chi.URLParam(r, "objectType")
	objectID := chi.URLParam(r, "objectID")
	obj, err := h.eng.ResolveObjectState(r.Context(), cellID, scenarioID, objectType, objectID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, obj)
}

func (h *handler) handleCommitScenario(w http.ResponseWriter, r *http.Request) {
	scenarioID := chi.URLParam(r, "scenarioID")
	res, err := h.eng.CommitScenario(r.Context(), scenarioID)
	if err != nil {
		if errors.Is(err, foundry.ErrScenarioWatermarkDrift) {
			writeJSON(w, http.StatusConflict, res)
			return
		}
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *handler) handleStartCashierSession(w http.ResponseWriter, r *http.Request) {
	var req foundry.StartCashierSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
		return
	}
	view, err := h.eng.StartBlindCountSession(r.Context(), req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

func (h *handler) handleSubmitBlindCount(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")
	var req struct {
		DenominationsUZS        map[int64]int64 `json:"denominations_uzs"`
		ToleranceThresholdMinor int64           `json:"tolerance_threshold_minor"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
		return
	}
	view, err := h.eng.SubmitBlindDenominationCount(r.Context(), sessionID, req.DenominationsUZS, req.ToleranceThresholdMinor)
	if err != nil {
		if errors.Is(err, foundry.ErrBlindRecountRequired) {
			writeJSON(w, http.StatusConflict, map[string]any{
				"code":       "RECOUNT_REQUIRED",
				"error":      err.Error(),
				"session":    view,
				"session_id": view.SessionID,
				"status":     view.Status,
			})
			return
		}
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (h *handler) handleReclassifyTenderRRN(w http.ResponseWriter, r *http.Request) {
	var req foundry.RRNTenderReclassificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
		return
	}
	view, err := h.eng.ReclassifyCashToCardByRRN(r.Context(), req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (h *handler) handleCITHandover(w http.ResponseWriter, r *http.Request) {
	var req foundry.CITHandoverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
		return
	}
	entry, err := h.eng.ExecuteCITArmoredHandover(r.Context(), req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, entry)
}

func (h *handler) handleSubmitDVIR(w http.ResponseWriter, r *http.Request) {
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

func (h *handler) handleGateEvent(w http.ResponseWriter, r *http.Request) {
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

func (h *handler) handleMidRouteRescue(w http.ResponseWriter, r *http.Request) {
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

func (h *handler) handleSplitTenderReconcile(w http.ResponseWriter, r *http.Request) {
	var req foundry.SplitTenderRejectionInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
		return
	}
	res, err := h.eng.ReconcileSplitTenderPartialRejection(r.Context(), req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *handler) handleAMLCheck(w http.ResponseWriter, r *http.Request) {
	var req struct {
		LegalEntityTIN string `json:"legal_entity_tin"`
		SupplierID     string `json:"supplier_id"`
		CalendarDate   string `json:"calendar_date"`
		CashMinor      int64  `json:"cash_minor"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
		return
	}
	accum, err := h.eng.CheckAndRecordAMLCashExposure(r.Context(), req.LegalEntityTIN, req.SupplierID, req.CalendarDate, req.CashMinor)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"legal_entity_tin":       strings.TrimSpace(req.LegalEntityTIN),
		"accumulated_cash_minor": accum,
		"ceiling_minor":          h.eng.AMLCashCeilingMinor(),
	})
}

func (h *handler) handleFiscalPreFlight(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Lines         []foundry.FiscalLineItemCheck `json:"lines"`
		CertExpiresAt time.Time                     `json:"cert_expires_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
		return
	}
	if err := h.eng.ValidatePreWaveFiscalReadiness(r.Context(), req.Lines, req.CertExpiresAt); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"fiscal_preflight_ok": true})
}

func (h *handler) handleClawbackSlice(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OriginalSlice            foundry.SettlementSliceRecord `json:"original_slice"`
		ApprovedDamageClaimMinor int64                         `json:"approved_damage_claim_minor"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
		return
	}
	slice, err := h.eng.CreatePostSettlementClawbackSlice(r.Context(), req.OriginalSlice, req.ApprovedDamageClaimMinor)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, slice)
}

func (h *handler) handlePredictiveReefer(w http.ResponseWriter, r *http.Request) {
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
		"vehicle_id":                    strings.TrimSpace(req.VehicleID),
		"pre_failure_maintenance_hold": hold,
	})
}

func (h *handler) handleDispatchCheck(w http.ResponseWriter, r *http.Request) {
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

func (h *handler) handleIssueOfflineLease(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OrderID              string `json:"order_id"`
		DriverID             string `json:"driver_id"`
		SupplierID           string `json:"supplier_id"`
		LegalEntityTIN       string `json:"legal_entity_tin"`
		CalendarDate         string `json:"calendar_date"`
		RequestedCashMinor   int64  `json:"requested_cash_minor"`
		RequestedCreditMinor int64  `json:"requested_credit_minor"`
		TTLMinutes           int64  `json:"ttl_minutes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
		return
	}
	ttl := time.Duration(req.TTLMinutes) * time.Minute
	lease, err := h.eng.IssueOfflineQuotaLease(
		r.Context(),
		req.OrderID,
		req.DriverID,
		req.SupplierID,
		req.LegalEntityTIN,
		req.CalendarDate,
		req.RequestedCashMinor,
		req.RequestedCreditMinor,
		ttl,
	)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, lease)
}

func (h *handler) handleValidateOfflineLease(w http.ResponseWriter, r *http.Request) {
	var req struct {
		LeaseID              string `json:"lease_id"`
		CollectedCashMinor   int64  `json:"collected_cash_minor"`
		CollectedCreditMinor int64  `json:"collected_credit_minor"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
		return
	}
	if err := h.eng.ValidateOfflineLeaseCollection(r.Context(), req.LeaseID, req.CollectedCashMinor, req.CollectedCreditMinor); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"lease_valid": true})
}

func (h *handler) handleSlicePayoutEligibility(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SliceID     string `json:"slice_id"`
		FiscalState string `json:"fiscal_state"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
		return
	}
	if err := h.eng.ValidateSlicePayoutEligibility(r.Context(), req.SliceID, req.FiscalState); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"payout_eligible": true})
}

func (h *handler) handleLoad3DRTI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StackLayers             []foundry.LoadStackLayer `json:"stack_layers"`
		VehicleCapacityDm3      int64                    `json:"vehicle_capacity_dm3"`
		AccountID               string                   `json:"account_id"`
		SupplierID              string                   `json:"supplier_id"`
		ContainerSKU            string                   `json:"container_sku"`
		DeliveredFullContainers int64                    `json:"delivered_full_containers"`
		PickedUpEmptyContainers int64                    `json:"picked_up_empty_containers"`
		UnitContainerVolumeDm3  int64                    `json:"unit_container_volume_dm3"`
		UnitDepositMinor        int64                    `json:"unit_deposit_minor"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
		return
	}
	bal, err := h.eng.Validate3DLoadAndRTIReturns(
		r.Context(),
		req.StackLayers,
		req.VehicleCapacityDm3,
		req.AccountID,
		req.SupplierID,
		req.ContainerSKU,
		req.DeliveredFullContainers,
		req.PickedUpEmptyContainers,
		req.UnitContainerVolumeDm3,
		req.UnitDepositMinor,
	)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bal)
}

func (h *handler) handleFXCorridor(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FXPolicy             string `json:"fx_policy"`
		CheckoutPriceMinor   int64  `json:"checkout_price_minor"`
		CheckoutFXRateMicros int64  `json:"checkout_fx_rate_micros"`
		CurrentFXRateMicros  int64  `json:"current_fx_rate_micros"`
		MaxCorridorBps       int64  `json:"max_corridor_bps"`
		WaveAlreadyReleased  bool   `json:"wave_already_released"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
		return
	}
	effectivePrice, err := h.eng.EvaluateFXCorridorBeforeWave(
		r.Context(),
		req.FXPolicy,
		req.CheckoutPriceMinor,
		req.CheckoutFXRateMicros,
		req.CurrentFXRateMicros,
		req.MaxCorridorBps,
		req.WaveAlreadyReleased,
	)
	if err != nil {
		if errors.Is(err, foundry.ErrFXReconfirmationRequired) {
			writeJSON(w, http.StatusConflict, map[string]any{
				"error":                 err.Error(),
				"effective_price_minor": effectivePrice,
			})
			return
		}
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"effective_price_minor": effectivePrice})
}

func (h *handler) handleRegisterToteState(w http.ResponseWriter, r *http.Request) {
	var req foundry.PickToteState
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
		return
	}
	h.eng.RegisterPickToteState(req)
	writeJSON(w, http.StatusCreated, req)
}

func (h *handler) handleToteTakeover(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ToteID      string `json:"tote_id"`
		NewPickerID string `json:"new_picker_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
		return
	}
	state, err := h.eng.TakeoverAbandonedPickTote(r.Context(), req.ToteID, req.NewPickerID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

