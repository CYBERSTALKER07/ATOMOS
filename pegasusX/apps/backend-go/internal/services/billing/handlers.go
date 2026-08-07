package billing

import (
	"encoding/json"
	"net/http"
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

func RegisterRoutes(r chi.Router, h *Handlers) {
	if h == nil || h.Worker == nil {
		return
	}
	r.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/admin/billing/run-monthly", h.HandleRunMonthly)
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
