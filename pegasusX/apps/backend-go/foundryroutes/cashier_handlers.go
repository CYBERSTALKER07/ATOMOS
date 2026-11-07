package foundryroutes

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/foundry"
)

type cashierHandler struct {
	eng *foundry.Engine
}

func newCashierHandler(eng *foundry.Engine) *cashierHandler {
	return &cashierHandler{eng: eng}
}

func (h *cashierHandler) mountRoutes(r chi.Router) {
	r.Post("/v1/foundry/cashier/sessions", h.handleStartCashierSession)
	r.Post("/v1/foundry/cashier/sessions/{sessionID}/blind-count", h.handleSubmitBlindCount)
	r.Post("/v1/foundry/cashier/reclassify-tender", h.handleReclassifyTenderRRN)
	r.Post("/v1/foundry/cashier/vault/cit-handover", h.handleCITHandover)
	r.Post("/v1/foundry/guards/split-tender-reconcile", h.handleSplitTenderReconcile)
	r.Post("/v1/foundry/guards/aml-check", h.handleAMLCheck)
}

func (h *cashierHandler) handleStartCashierSession(w http.ResponseWriter, r *http.Request) {
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

func (h *cashierHandler) handleSubmitBlindCount(w http.ResponseWriter, r *http.Request) {
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

func (h *cashierHandler) handleReclassifyTenderRRN(w http.ResponseWriter, r *http.Request) {
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

func (h *cashierHandler) handleCITHandover(w http.ResponseWriter, r *http.Request) {
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

func (h *cashierHandler) handleSplitTenderReconcile(w http.ResponseWriter, r *http.Request) {
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

func (h *cashierHandler) handleAMLCheck(w http.ResponseWriter, r *http.Request) {
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
