package fxrates

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// Handlers serves admin FX rate read/write.
type Handlers struct {
	Repo Repository
	Svc  *Service
	Now  func() time.Time
}

// NewHandlers wires admin FX endpoints.
func NewHandlers(repo Repository) *Handlers {
	return &Handlers{
		Repo: repo,
		Svc:  NewService(repo),
		Now:  func() time.Time { return time.Now().UTC() },
	}
}

// RegisterAdminRoutes mounts GET/PUT /v1/admin/fx-rates.
// GET: supplier ADMIN or PLATFORM_ADMIN. PUT: PLATFORM_ADMIN only (cross-tenant rates).
func RegisterAdminRoutes(r chi.Router, h *Handlers) {
	if h == nil || h.Repo == nil {
		return
	}
	r.With(auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin)).Get("/v1/admin/fx-rates", h.HandleList)
	r.With(auth.RequireRole(auth.RolePlatformAdmin)).Put("/v1/admin/fx-rates", h.HandleUpsert)
}

// RegisterSupplierRoutes mounts GET /v1/supplier/fx-rates (supplier portal ADMIN session, read-only).
func RegisterSupplierRoutes(r chi.Router, h *Handlers) {
	if h == nil || h.Repo == nil {
		return
	}
	r.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/supplier/fx-rates", h.HandleList)
}

type rateDTO struct {
	RateID        string `json:"rate_id"`
	BaseCurrency  string `json:"base_currency"`
	QuoteCurrency string `json:"quote_currency"`
	RateScaled    int64  `json:"rate_scaled"`
	Scale         int64  `json:"scale"`
	EffectiveAt   string `json:"effective_at"`
	Source        string `json:"source"`
}

func rateToDTO(r Rate) rateDTO {
	return rateDTO{
		RateID:        r.RateID,
		BaseCurrency:  r.BaseCurrency,
		QuoteCurrency: r.QuoteCurrency,
		RateScaled:    r.RateScaled,
		Scale:         r.Scale,
		EffectiveAt:   r.EffectiveAt.UTC().Format(time.RFC3339Nano),
		Source:        r.Source,
	}
}

// HandleList GET /v1/admin/fx-rates — latest rates per pair.
func (h *Handlers) HandleList(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if q := strings.TrimSpace(r.URL.Query().Get("limit")); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n > 0 {
			limit = n
		}
	}
	rates, err := h.Repo.ListLatest(r.Context(), limit)
	if err != nil {
		writeFXError(w, http.StatusInternalServerError, "fx_list_failed", err.Error())
		return
	}
	out := make([]rateDTO, 0, len(rates))
	for _, rate := range rates {
		out = append(out, rateToDTO(rate))
	}
	writeFXJSON(w, http.StatusOK, map[string]any{"rates": out})
}

// HandleUpsert PUT /v1/admin/fx-rates — platform-admin rate upsert.
func (h *Handlers) HandleUpsert(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 32*1024))
	if err != nil {
		writeFXError(w, http.StatusBadRequest, "read_body_failed", "unable to read body")
		return
	}
	defer r.Body.Close()

	var req struct {
		BaseCurrency  string `json:"base_currency"`
		QuoteCurrency string `json:"quote_currency"`
		RateScaled    int64  `json:"rate_scaled"`
		Scale         int64  `json:"scale"`
		EffectiveAt   string `json:"effective_at"`
		Source        string `json:"source"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeFXError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return
	}
	base := NormalizeCurrency(req.BaseCurrency)
	quote := NormalizeCurrency(req.QuoteCurrency)
	if len(base) != 3 || len(quote) != 3 {
		writeFXError(w, http.StatusUnprocessableEntity, "invalid_currency", "base_currency and quote_currency must be ISO-4217 codes")
		return
	}
	if req.RateScaled <= 0 {
		writeFXError(w, http.StatusUnprocessableEntity, "invalid_rate", "rate_scaled must be > 0")
		return
	}
	scale := req.Scale
	if scale <= 0 {
		scale = DefaultScale
	}
	at := h.Now()
	if strings.TrimSpace(req.EffectiveAt) != "" {
		parsed, err := time.Parse(time.RFC3339Nano, req.EffectiveAt)
		if err != nil {
			parsed, err = time.Parse(time.RFC3339, req.EffectiveAt)
		}
		if err != nil {
			writeFXError(w, http.StatusUnprocessableEntity, "invalid_effective_at", "effective_at must be RFC3339")
			return
		}
		at = parsed.UTC()
	}
	source := strings.ToUpper(strings.TrimSpace(req.Source))
	if source == "" {
		source = "ADMIN"
	}
	rate := Rate{
		BaseCurrency:  base,
		QuoteCurrency: quote,
		RateScaled:    req.RateScaled,
		Scale:         scale,
		EffectiveAt:   at,
		Source:        source,
	}
	if err := h.Repo.Upsert(r.Context(), rate); err != nil {
		writeFXError(w, http.StatusInternalServerError, "fx_upsert_failed", err.Error())
		return
	}
	writeFXJSON(w, http.StatusOK, map[string]any{"ok": true, "rate": rateToDTO(rate)})
}

func writeFXJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeFXError(w http.ResponseWriter, status int, code, message string) {
	writeFXJSON(w, status, map[string]string{"code": code, "error": message})
}
