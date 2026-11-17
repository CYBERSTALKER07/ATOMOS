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

// FlagApprovalDecision reports whether a CEL approval request was created/linked for a money flag override.
type FlagApprovalDecision struct {
	Required          bool
	Approved          bool
	ApprovalRequestID string
	PayloadHash       string
}

// FlagApprovalBridge connects money-affecting feature flag changes to the M1.2 CEL Approval Engine.
type FlagApprovalBridge interface {
	EnsureMoneyFlagRequest(
		ctx context.Context,
		tenantType, tenantID, flagKey string,
		enabled bool,
		reason, actorID, actorType, actorRole string,
	) (FlagApprovalDecision, error)
	ApproveMoneyFlagRequest(
		ctx context.Context,
		tenantType, tenantID, flagKey, actorID, actorType, actorRole string,
		approvalID, comment, totpCode string,
		mfaVerified, breakGlass bool,
		breakGlassReason string,
	) (FlagApprovalDecision, error)
}

// Handlers expose flag evaluate/override APIs.
type Handlers struct {
	Svc            *Service
	Audit          FlagAuditor
	ApprovalBridge FlagApprovalBridge
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
	var approvalRequestID string
	if MoneyAffectingFlags[strings.ToUpper(flag)] {
		status = StatusPending
		if h.ApprovalBridge != nil {
			claims, _ := auth.FromContext(r.Context())
			dec, bErr := h.ApprovalBridge.EnsureMoneyFlagRequest(
				r.Context(), req.TenantType, req.TenantID, strings.ToUpper(flag), req.Enabled, req.Reason,
				actor, auth.EffectiveActorType(claims), auth.EffectiveSupplierRole(claims),
			)
			if bErr != nil {
				writeErr(w, http.StatusForbidden, bErr.Error())
				return
			}
			approvalRequestID = dec.ApprovalRequestID
		}
	}
	if h.Audit != nil {
		detail, _ := json.Marshal(map[string]any{
			"flag_key":            strings.ToUpper(flag),
			"tenant_type":         req.TenantType,
			"tenant_id":           req.TenantID,
			"enabled":             req.Enabled,
			"reason":              req.Reason,
			"status":              status,
			"actor":               actor,
			"approval_request_id": approvalRequestID,
		})
		if err := h.Audit.RecordFlagAudit(r.Context(), actor, auditActionOverrideSet, req.TenantType, req.TenantID, string(detail)); err != nil {
			writeErr(w, http.StatusInternalServerError, "audit_failed")
			return
		}
	}
	resp := map[string]any{"ok": true, "status": status}
	if approvalRequestID != "" {
		resp["approval_request_id"] = approvalRequestID
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleApproveOverride POST /v1/platform-admin/flags/{flagKey}/approve
// Dual control: a different PLATFORM_ADMIN activates a PENDING money flag.
func (h *Handlers) HandleApproveOverride(w http.ResponseWriter, r *http.Request) {
	flag := chi.URLParam(r, "flagKey")
	var req struct {
		TenantType       string `json:"tenant_type"`
		TenantID         string `json:"tenant_id"`
		ApprovalID       string `json:"approval_id,omitempty"`
		Comment          string `json:"comment,omitempty"`
		TOTPCode         string `json:"totp_code,omitempty"`
		BreakGlass       bool   `json:"break_glass,omitempty"`
		BreakGlassReason string `json:"break_glass_reason,omitempty"`
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
	var approvalRequestID string
	if h.ApprovalBridge != nil {
		claims, _ := auth.FromContext(r.Context())
		dec, bErr := h.ApprovalBridge.ApproveMoneyFlagRequest(
			r.Context(), req.TenantType, req.TenantID, strings.ToUpper(flag), approver,
			auth.EffectiveActorType(claims), auth.EffectiveSupplierRole(claims),
			strings.TrimSpace(req.ApprovalID), req.Comment, req.TOTPCode,
			claims.MFAVerified, req.BreakGlass, req.BreakGlassReason,
		)
		if bErr != nil {
			_ = h.Svc.RevertApproveToPending(r.Context(), flag, req.TenantType, req.TenantID)
			code := http.StatusForbidden
			if strings.Contains(strings.ToLower(bErr.Error()), "hash") {
				code = http.StatusConflict
			}
			writeErr(w, code, bErr.Error())
			return
		}
		approvalRequestID = dec.ApprovalRequestID
		if dec.Required && !dec.Approved {
			_ = h.Svc.RevertApproveToPending(r.Context(), flag, req.TenantType, req.TenantID)
			writeJSON(w, http.StatusAccepted, map[string]any{
				"ok":                  false,
				"status":              StatusPending,
				"code":                "approval_required",
				"approval_request_id": dec.ApprovalRequestID,
				"payload_hash":        dec.PayloadHash,
			})
			return
		}
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
			"flag_key":            strings.ToUpper(flag),
			"tenant_type":         req.TenantType,
			"tenant_id":           req.TenantID,
			"approver":            approver,
			"approval_request_id": approvalRequestID,
		})
		if err := h.Audit.RecordFlagAudit(r.Context(), approver, action, req.TenantType, req.TenantID, string(detail)); err != nil {
			_ = h.Svc.RevertApproveToPending(r.Context(), flag, req.TenantType, req.TenantID)
			writeErr(w, http.StatusInternalServerError, "audit_failed")
			return
		}
	}
	resp := map[string]any{"ok": true, "status": StatusActive}
	if approvalRequestID != "" {
		resp["approval_request_id"] = approvalRequestID
	}
	writeJSON(w, http.StatusOK, resp)
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
