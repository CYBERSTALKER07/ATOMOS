package featureflags

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// Handlers expose flag evaluate/override APIs.
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

// HandleEvaluate GET /v1/platform-admin/flags/{flagKey}?tenant_type=&tenant_id=
func (h *Handlers) HandleEvaluate(w http.ResponseWriter, r *http.Request) {
	flag := chi.URLParam(r, "flagKey")
	tt := r.URL.Query().Get("tenant_type")
	tid := r.URL.Query().Get("tenant_id")
	enabled, source, err := h.Svc.Evaluate(r.Context(), flag, tt, tid)
	if err != nil {
		writeErr(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"flag_key": strings.ToUpper(flag),
		"enabled":  enabled,
		"source":   source,
		"money_affecting": MoneyAffectingFlags[strings.ToUpper(flag)],
	})
}

type setOverrideRequest struct {
	TenantType string `json:"tenant_type"`
	TenantID   string `json:"tenant_id"`
	Enabled    bool   `json:"enabled"`
	Reason     string `json:"reason"`
}

// HandleSetOverride PUT /v1/platform-admin/flags/{flagKey}
func (h *Handlers) HandleSetOverride(w http.ResponseWriter, r *http.Request) {
	flag := chi.URLParam(r, "flagKey")
	var req setOverrideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json")
		return
	}
	actor := "unknown"
	if c, ok := auth.FromContext(r.Context()); ok {
		actor = c.Subject
		if actor == "" {
			actor = string(c.Role)
		}
	}
	err := h.Svc.SetOverride(r.Context(), Override{
		FlagKey:    flag,
		TenantType: req.TenantType,
		TenantID:   req.TenantID,
		Enabled:    req.Enabled,
		UpdatedBy:  actor,
		Reason:     req.Reason,
	})
	if err != nil {
		writeErr(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	status := StatusActive
	if MoneyAffectingFlags[strings.ToUpper(flag)] {
		status = StatusPending
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": status})
}

// HandleApproveOverride POST /v1/platform-admin/flags/{flagKey}/approve
// Dual control: a different PLATFORM_ADMIN activates a PENDING money flag.
func (h *Handlers) HandleApproveOverride(w http.ResponseWriter, r *http.Request) {
	flag := chi.URLParam(r, "flagKey")
	var req struct {
		TenantType string `json:"tenant_type"`
		TenantID   string `json:"tenant_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json")
		return
	}
	approver := "unknown"
	if c, ok := auth.FromContext(r.Context()); ok {
		approver = c.Subject
		if approver == "" {
			approver = string(c.Role)
		}
	}
	if err := h.Svc.ApproveOverride(r.Context(), flag, req.TenantType, req.TenantID, approver); err != nil {
		status := http.StatusUnprocessableEntity
		switch err.Error() {
		case "override_not_found":
			status = http.StatusNotFound
		case "approver_must_differ_from_setter", "override_not_pending", "not_a_money_flag":
			status = http.StatusConflict
		}
		writeErr(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": StatusActive})
}

// RegisterRoutes mounts flag routes under an already PLATFORM_ADMIN-gated router group
// or standalone with RequireRole.
func RegisterRoutes(r chi.Router, h *Handlers) {
	if h == nil || h.Svc == nil {
		return
	}
	r.Route("/v1/platform-admin/flags", func(fr chi.Router) {
		fr.Use(auth.RequireRole(auth.RolePlatformAdmin))
		fr.Get("/{flagKey}", h.HandleEvaluate)
		fr.Put("/{flagKey}", h.HandleSetOverride)
		fr.Post("/{flagKey}/approve", h.HandleApproveOverride)
	})
}
