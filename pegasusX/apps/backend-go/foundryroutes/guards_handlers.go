package foundryroutes

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/foundry"
)

type guardsHandler struct {
	eng *foundry.Engine
}

func newGuardsHandler(eng *foundry.Engine) *guardsHandler {
	return &guardsHandler{eng: eng}
}

func (h *guardsHandler) mountRoutes(r chi.Router) {
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

func (h *guardsHandler) handleIssueOfflineLease(w http.ResponseWriter, r *http.Request) {
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

func (h *guardsHandler) handleValidateOfflineLease(w http.ResponseWriter, r *http.Request) {
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

func (h *guardsHandler) handleFiscalPreFlight(w http.ResponseWriter, r *http.Request) {
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

func (h *guardsHandler) handleSlicePayoutEligibility(w http.ResponseWriter, r *http.Request) {
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

func (h *guardsHandler) handleClawbackSlice(w http.ResponseWriter, r *http.Request) {
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

func (h *guardsHandler) handleLoad3DRTI(w http.ResponseWriter, r *http.Request) {
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

func (h *guardsHandler) handleFXCorridor(w http.ResponseWriter, r *http.Request) {
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

func (h *guardsHandler) handleRegisterToteState(w http.ResponseWriter, r *http.Request) {
	var req foundry.PickToteState
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON payload"})
		return
	}
	h.eng.RegisterPickToteState(req)
	writeJSON(w, http.StatusCreated, req)
}

func (h *guardsHandler) handleToteTakeover(w http.ResponseWriter, r *http.Request) {
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
