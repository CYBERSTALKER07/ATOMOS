package billing

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// Handlers expose billing operations (admin-triggered monthly run; a scheduler
// job hits this endpoint monthly).
type Handlers struct {
	Worker *InvoiceWorker
}

func RegisterRoutes(r chi.Router, h *Handlers, stepUp ...func(http.Handler) http.Handler) {
	if h == nil || h.Worker == nil {
		return
	}
	r.Route("/v1/admin/billing", func(br chi.Router) {
		br.Use(auth.RequireRole(auth.RolePlatformAdmin))
		for _, mw := range stepUp {
			if mw != nil {
				br.Use(mw)
			}
		}
		br.Post("/run-monthly", h.HandleRunMonthly)
		br.Get("/invoices", h.HandleListInvoices)
		br.Get("/fee-schedules", h.HandleListFeeSchedules)
	})
}

func (h *Handlers) HandleRunMonthly(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Month string `json:"month"` // YYYY-MM, default: previous full month
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	month := time.Now().UTC().AddDate(0, -1, 0)
	if m := strings.TrimSpace(body.Month); m != "" {
		parsed, err := time.Parse("2006-01", m)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "month must be YYYY-MM"})
			return
		}
		month = parsed
	}
	billed, err := h.Worker.GenerateMonthlyInvoices(r.Context(), month)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error(), "billed": billed})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"billed": billed, "month": month.Format("2006-01")})
}

func (h *Handlers) HandleListInvoices(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			limit = n
		}
	}
	invoices, err := h.Worker.ListPlatformInvoices(r.Context(), limit)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "billing_unavailable"})
		return
	}
	if invoices == nil {
		invoices = []PlatformInvoice{}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"invoices": invoices})
}

func (h *Handlers) HandleListFeeSchedules(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			limit = n
		}
	}
	schedules, err := h.Worker.ListFeeSchedules(r.Context(), limit)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "billing_unavailable"})
		return
	}
	if schedules == nil {
		schedules = []FeeScheduleRow{}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"fee_schedules": schedules})
}
