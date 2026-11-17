package foundryroutes

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/foundry"
)

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
