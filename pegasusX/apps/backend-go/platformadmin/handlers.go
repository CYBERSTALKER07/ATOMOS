package platformadmin

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// Handlers expose PLATFORM_ADMIN tenant APIs.
type Handlers struct {
	Svc *Service
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func actorSubject(r *http.Request) string {
	if c, ok := auth.FromContext(r.Context()); ok {
		if c.Subject != "" {
			return c.Subject
		}
		return string(c.Role)
	}
	return "unknown"
}

// HandleListTenants GET /v1/platform-admin/tenants
func (h *Handlers) HandleListTenants(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	lim, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	tenants, err := h.Svc.List(r.Context(), status, lim)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tenants": tenants})
}

// HandleGetTenant GET /v1/platform-admin/tenants/{tenantType}/{tenantID}
func (h *Handlers) HandleGetTenant(w http.ResponseWriter, r *http.Request) {
	tt := chi.URLParam(r, "tenantType")
	id := chi.URLParam(r, "tenantID")
	t, ok, err := h.Svc.Get(r.Context(), tt, id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		writeErr(w, http.StatusNotFound, "tenant_not_found")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

type transitionRequest struct {
	Status   string `json:"status"`
	KybNotes string `json:"kyb_notes"`
}

// HandleTransitionTenant POST /v1/platform-admin/tenants/{tenantType}/{tenantID}/transition
func (h *Handlers) HandleTransitionTenant(w http.ResponseWriter, r *http.Request) {
	tt := chi.URLParam(r, "tenantType")
	id := chi.URLParam(r, "tenantID")
	var req transitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json")
		return
	}
	t, err := h.Svc.Transition(r.Context(), actorSubject(r), tt, id, req.Status, req.KybNotes)
	if err != nil {
		msg := err.Error()
		status := http.StatusUnprocessableEntity
		if strings.HasPrefix(msg, "illegal_transition") || msg == "invalid_status" {
			status = http.StatusConflict
		}
		writeErr(w, status, msg)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// HandleListAudit GET /v1/platform-admin/audit
func (h *Handlers) HandleListAudit(w http.ResponseWriter, r *http.Request) {
	lim, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	rows, err := h.Svc.ListAudit(r.Context(), lim)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"audit": rows})
}

// RegisterRoutes mounts PLATFORM_ADMIN-gated routes.
func RegisterRoutes(r chi.Router, h *Handlers) {
	if h == nil || h.Svc == nil {
		return
	}
	r.Route("/v1/platform-admin", func(pr chi.Router) {
		pr.Use(auth.RequireRole(auth.RolePlatformAdmin))
		pr.Get("/tenants", h.HandleListTenants)
		pr.Get("/tenants/{tenantType}/{tenantID}", h.HandleGetTenant)
		pr.Post("/tenants/{tenantType}/{tenantID}/transition", h.HandleTransitionTenant)
		pr.Get("/audit", h.HandleListAudit)
	})
}
