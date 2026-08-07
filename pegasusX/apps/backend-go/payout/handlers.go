package payout

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// Handlers expose payout batch management on the supplier portal (ADMIN).
type Handlers struct {
	Svc *Service
}

func RegisterRoutes(r chi.Router, h *Handlers) {
	if h == nil || h.Svc == nil {
		return
	}
	r.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/supplier/payouts/batches", h.HandleGenerate)
	r.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/supplier/payouts/batches/{batchID}/export", h.HandleExport)
	r.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/supplier/payouts/batches/{batchID}/mark-paid", h.HandleMarkPaid)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *Handlers) HandleGenerate(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var body struct {
		SupplierID     string `json:"supplier_id"`
		PeriodStart    string `json:"period_start"` // YYYY-MM-DD
		PeriodEnd      string `json:"period_end"`
		IdempotencyKey string `json:"idempotency_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	start, err1 := time.Parse("2006-01-02", strings.TrimSpace(body.PeriodStart))
	end, err2 := time.Parse("2006-01-02", strings.TrimSpace(body.PeriodEnd))
	if err1 != nil || err2 != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "period_start/period_end must be YYYY-MM-DD"})
		return
	}
	b, err := h.Svc.GenerateBatch(r.Context(), body.SupplierID, start, end, claims.Subject, body.IdempotencyKey)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrNothingPayable) {
			status = http.StatusConflict
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (h *Handlers) HandleExport(w http.ResponseWriter, r *http.Request) {
	batchID := strings.TrimSpace(chi.URLParam(r, "batchID"))
	raw, b, err := h.Svc.ExportBankFile(r.Context(), batchID)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrBatchNotFound):
			status = http.StatusNotFound
		case errors.Is(err, ErrBankDetailsMissing):
			status = http.StatusConflict
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=payout-"+b.BatchID+".csv")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

func (h *Handlers) HandleMarkPaid(w http.ResponseWriter, r *http.Request) {
	batchID := strings.TrimSpace(chi.URLParam(r, "batchID"))
	if err := h.Svc.MarkPaid(r.Context(), batchID); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrBatchNotFound) {
			status = http.StatusNotFound
		} else if strings.Contains(err.Error(), "must be EXPORTED") {
			status = http.StatusConflict
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": StatusPaid})
}
