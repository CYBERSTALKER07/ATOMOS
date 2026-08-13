package payout

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"os"
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
	r.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/supplier/payouts/rail", h.HandleRailInfo)
	r.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/supplier/payouts/batches", h.HandleGenerate)
	r.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/supplier/payouts/batches/{batchID}/export", h.HandleExport)
	r.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/supplier/payouts/batches/{batchID}/dispatch", h.HandleDispatch)
	r.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/supplier/payouts/batches/{batchID}/mark-paid", h.HandleMarkPaid)
	// Settlement webhook from a live rail. Authenticated by the rail's shared
	// secret header, not a user role (machine-to-machine).
	r.Post("/v1/webhooks/payouts/settlement", h.HandleSettlementWebhook)
}

// HandleRailInfo serves GET /v1/supplier/payouts/rail — G1.D honesty surface.
func (h *Handlers) HandleRailInfo(w http.ResponseWriter, r *http.Request) {
	info := h.Svc.RailInfo()
	writeJSON(w, http.StatusOK, info)
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
		// SupplierID in body is ignored for authorization (B1 M-P0-14).
		// Kept for backward-compatible clients; tenant comes from JWT/context.
		SupplierID     string `json:"supplier_id"`
		PeriodStart    string `json:"period_start"` // YYYY-MM-DD
		PeriodEnd      string `json:"period_end"`
		IdempotencyKey string `json:"idempotency_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	// Scope from claims / TenantContext only — never trust body.supplier_id for auth.
	supplierID := auth.PreferTenantSupplierID(r.Context(), "")
	if supplierID == "" {
		supplierID = strings.TrimSpace(claims.SupplierID)
	}
	if supplierID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "supplier_scope_required"})
		return
	}
	start, err1 := time.Parse("2006-01-02", strings.TrimSpace(body.PeriodStart))
	end, err2 := time.Parse("2006-01-02", strings.TrimSpace(body.PeriodEnd))
	if err1 != nil || err2 != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "period_start/period_end must be YYYY-MM-DD"})
		return
	}
	b, err := h.Svc.GenerateBatch(r.Context(), supplierID, start, end, claims.Subject, body.IdempotencyKey)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrNothingPayable) {
			status = http.StatusConflict
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	// G1.D: always advertise rail workflow so UI never implies live bank rails.
	writeJSON(w, http.StatusOK, map[string]any{
		"batch": b,
		"rail":  h.Svc.RailInfo(),
	})
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
	// G1.D: MarkPaid is the bank-file settlement confirmation (human after bank).
	info := h.Svc.RailInfo()
	writeJSON(w, http.StatusOK, map[string]any{
		"status":   StatusPaid,
		"batch_id": batchID,
		"rail":     info,
		"message":  "Batch marked PAID after bank-file settlement (not a live rail webhook)",
	})
}

// HandleDispatch submits a batch to the configured rail. Default is the
// bank-file dry-run; a live rail requires live=true and dispatches funds.
func (h *Handlers) HandleDispatch(w http.ResponseWriter, r *http.Request) {
	batchID := strings.TrimSpace(chi.URLParam(r, "batchID"))
	var body struct {
		Live bool `json:"live"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body) // empty body => dry-run
	b, err := h.Svc.SubmitForDispatch(r.Context(), batchID, body.Live)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrBatchNotFound):
			status = http.StatusNotFound
		case errors.Is(err, ErrBankDetailsMissing):
			status = http.StatusConflict
		case errors.Is(err, ErrNoLiveRail):
			// G1.D: never look like live money moved when only bank-file exists.
			status = http.StatusConflict
			writeJSON(w, status, map[string]any{
				"error":   "no_live_rail",
				"code":    "no_live_rail",
				"rail":    h.Svc.RailInfo(),
				"message": "No live payout rail: export CSV, process at bank, then POST .../mark-paid",
			})
			return
		case strings.Contains(err.Error(), "not dispatchable"):
			status = http.StatusConflict
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"batch": b,
		"rail":  h.Svc.RailInfo(),
	})
}

// HandleSettlementWebhook receives a rail's settlement confirmation and flips
// the batch to PAID. Machine-to-machine; verify the rail's shared secret.
func (h *Handlers) HandleSettlementWebhook(w http.ResponseWriter, r *http.Request) {
	if !verifyRailSecret(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid rail secret"})
		return
	}
	var body struct {
		BatchID string `json:"batch_id"`
		RailRef string `json:"rail_reference"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if err := h.Svc.ConfirmSettlement(r.Context(), body.BatchID, body.RailRef); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrBatchNotFound) {
			status = http.StatusNotFound
		} else if strings.Contains(err.Error(), "must be SUBMITTED") {
			status = http.StatusConflict
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": StatusPaid})
}

// verifyRailSecret checks the shared-secret header. When no secret is
// configured the endpoint is disabled (fail-closed) — a settlement webhook
// must never be reachable unauthenticated.
func verifyRailSecret(r *http.Request) bool {
	want := strings.TrimSpace(os.Getenv("PAYOUT_RAIL_WEBHOOK_SECRET"))
	if want == "" {
		return false
	}
	got := strings.TrimSpace(r.Header.Get("X-Payout-Rail-Secret"))
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}
