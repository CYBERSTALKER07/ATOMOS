package platformadmin

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

const websocketSessionTTL = 15 * time.Minute

// Handlers expose PLATFORM_ADMIN tenant APIs.
type Handlers struct {
	Svc       *Service
	JWTSecret string
	JWTIssuer string
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
	return auth.ActorFromContext(r.Context())
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

// HandleWebSocketSession GET /v1/platform-admin/ws-session
// Mints a short-lived JWT for /v1/ws (token_use=ws, including mfa_verified).
func (h *Handlers) HandleWebSocketSession(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.FromContext(r.Context())
	if !ok || c.Role != auth.RolePlatformAdmin {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if strings.TrimSpace(h.JWTSecret) == "" {
		writeErr(w, http.StatusServiceUnavailable, "jwt_not_configured")
		return
	}
	token, expiresAt, err := auth.IssueWSTicket(c, auth.IssueOptions{
		Secret: h.JWTSecret,
		Issuer: h.JWTIssuer,
		TTL:    websocketSessionTTL,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "issue_websocket_token_failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"token":      token,
		"expires_at": expiresAt.UTC().Format(time.RFC3339Nano),
	})
}

// RegisterRoutes mounts PLATFORM_ADMIN-gated routes.
// Optional stepUp middleware enforces TOTP when MFA is enrolled/required.
func RegisterRoutes(r chi.Router, h *Handlers, stepUp ...func(http.Handler) http.Handler) {
	if h == nil || h.Svc == nil {
		return
	}
	r.Route("/v1/platform-admin", func(pr chi.Router) {
		pr.Use(auth.RequireRole(auth.RolePlatformAdmin))
		for _, mw := range stepUp {
			if mw != nil {
				pr.Use(mw)
			}
		}
		pr.Get("/tenants", h.HandleListTenants)
		pr.Get("/tenants/{tenantType}/{tenantID}", h.HandleGetTenant)
		pr.Post("/tenants/{tenantType}/{tenantID}/transition", h.HandleTransitionTenant)
		pr.Get("/audit", h.HandleListAudit)
		pr.Get("/ws-session", h.HandleWebSocketSession)
	})
}
