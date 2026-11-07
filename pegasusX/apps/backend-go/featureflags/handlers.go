package featureflags

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// AuditActionAutoOrderPlace is written to PlatformAdminAudit when
// AUTO_ORDER_PLACE_ENABLED is dual-control approved (place-flip evidence trail).
const AuditActionAutoOrderPlace = "FLAG_AUTO_ORDER_PLACE"

// AuditActionAutoOrderSoakGate is written when AUTO_ORDER_SOAK_GATE_DISABLED
// is dual-control approved (break-glass soak bypass trail — P2-7).
const AuditActionAutoOrderSoakGate = "FLAG_AUTO_ORDER_SOAK_GATE"

const (
	auditActionOverrideSet     = "FLAG_OVERRIDE_SET"
	auditActionOverrideApprove = "FLAG_OVERRIDE_APPROVE"
)

// FlagAuditor records dual-control flag approvals for money-affecting flags.
type FlagAuditor interface {
	RecordFlagAudit(ctx context.Context, actor, action, tenantType, tenantID, detailJSON string) error
}

// Handlers expose flag evaluate/override APIs.
type Handlers struct {
	Svc   *Service
	Audit FlagAuditor
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
		"flag_key":        strings.ToUpper(flag),
		"enabled":         enabled,
		"source":          source,
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
	actor := auth.ActorFromContext(r.Context())
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
	if h.Audit != nil {
		detail, _ := json.Marshal(map[string]any{
			"flag_key":    strings.ToUpper(flag),
			"tenant_type": req.TenantType,
			"tenant_id":   req.TenantID,
			"enabled":     req.Enabled,
			"reason":      req.Reason,
			"status":      status,
			"actor":       actor,
		})
		if err := h.Audit.RecordFlagAudit(r.Context(), actor, auditActionOverrideSet, req.TenantType, req.TenantID, string(detail)); err != nil {
			writeErr(w, http.StatusInternalServerError, "audit_failed")
			return
		}
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
	approver := auth.ActorFromContext(r.Context())
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
	// B5 M-P0-11: money-flag ACTIVE without durable audit is fail-closed —
	// compensate back to PENDING when audit cannot be recorded.
	if h.Audit != nil {
		action := auditActionOverrideApprove
		if strings.EqualFold(flag, "AUTO_ORDER_PLACE_ENABLED") {
			action = AuditActionAutoOrderPlace
		} else if strings.EqualFold(flag, "AUTO_ORDER_SOAK_GATE_DISABLED") {
			action = AuditActionAutoOrderSoakGate
		}
		detail, _ := json.Marshal(map[string]any{
			"flag_key":    strings.ToUpper(flag),
			"tenant_type": req.TenantType,
			"tenant_id":   req.TenantID,
			"approver":    approver,
		})
		if err := h.Audit.RecordFlagAudit(r.Context(), approver, action, req.TenantType, req.TenantID, string(detail)); err != nil {
			_ = h.Svc.RevertApproveToPending(r.Context(), flag, req.TenantType, req.TenantID)
			writeErr(w, http.StatusInternalServerError, "audit_failed")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": StatusActive})
}

// RegisterRoutes mounts flag routes under an already PLATFORM_ADMIN-gated router group
// or standalone with RequireRole. Optional stepUp enforces TOTP when MFA is enrolled/required.
func RegisterRoutes(r chi.Router, h *Handlers, stepUp ...func(http.Handler) http.Handler) {
	if h == nil || h.Svc == nil {
		return
	}
	r.Route("/v1/platform-admin/flags", func(fr chi.Router) {
		fr.Use(auth.RequireRole(auth.RolePlatformAdmin))
		for _, mw := range stepUp {
			if mw != nil {
				fr.Use(mw)
			}
		}
		fr.Get("/", h.HandleListPending)
		fr.Get("/{flagKey}", h.HandleEvaluate)
		fr.Put("/{flagKey}", h.HandleSetOverride)
		fr.Post("/{flagKey}/approve", h.HandleApproveOverride)
	})
}

// HandleListPending GET /v1/platform-admin/flags — PENDING dual-control overrides.
func (h *Handlers) HandleListPending(w http.ResponseWriter, r *http.Request) {
	if h.Svc == nil {
		writeErr(w, http.StatusServiceUnavailable, "unavailable")
		return
	}
	rows, err := h.Svc.ListPending(r.Context(), 100)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if rows == nil {
		rows = []Override{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items":     rows,
		"count":     len(rows),
		"available": true,
		"status":    StatusPending,
	})
}
