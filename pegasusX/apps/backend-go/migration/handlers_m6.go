package migration

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
)

type applySessionRequest struct {
	ApplyMode string `json:"apply_mode,omitempty"`
	ChunkSize int    `json:"chunk_size,omitempty"`
}

// HandleApplySession executes chunked atomic entity apply for an approved (or resumable) session.
func (h *Handlers) HandleApplySession(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "", "apply", "")
	if !ok {
		return
	}
	sessionID := strings.TrimSpace(chi.URLParam(r, "sessionId"))
	if sessionID == "" {
		writeMigrationError(w, http.StatusBadRequest, "session_id_required", "sessionId is required")
		return
	}

	var req applySessionRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeMigrationError(w, http.StatusBadRequest, "invalid_json", "malformed apply request body")
			return
		}
	}

	sess, err := h.repo.GetSession(r.Context(), claims.SupplierID, sessionID)
	if err != nil {
		writeRepoError(w, err)
		return
	}

	action := "apply"
	effectiveMode := NormalizeApplyMode(sess.ApplyMode)
	if strings.TrimSpace(req.ApplyMode) != "" {
		effectiveMode = NormalizeApplyMode(req.ApplyMode)
	}
	if (sess.Pack == PackLots || sess.Pack == PackStock) && effectiveMode == ApplyModeSetOnHand {
		action = "set_on_hand"
	}
	claims, ok = h.authorize(w, r, string(sess.Pack), action, claims.SupplierID)
	if !ok {
		return
	}
	if h.applyStore == nil {
		writeMigrationError(w, http.StatusServiceUnavailable, "apply_store_unavailable", "migration apply store is not configured")
		return
	}

	res, err := ApplySession(r.Context(), h.repo, h.applyStore, ApplyOptions{
		SupplierID:   claims.SupplierID,
		SessionID:    sessionID,
		ApplyMode:    req.ApplyMode,
		ChunkSize:    req.ChunkSize,
		Now:          h.now(),
		ProgressEmit: h.progressEmit,
	})
	if err != nil {
		writeRepoError(w, err)
		return
	}
	writeMigrationJSON(w, http.StatusOK, res)
}

type releaseOpenOrderItemRequest struct {
	OrderID      string `json:"order_id"`
	TargetStatus string `json:"target_status,omitempty"`
}

type releaseOpenOrdersRequest struct {
	Items        []releaseOpenOrderItemRequest `json:"items,omitempty"`
	OrderIDs     []string                      `json:"order_ids,omitempty"`
	TargetStatus string                        `json:"target_status,omitempty"`
}

// HandleReleaseOpenOrders reserves inventory/FEFO lots and transitions held MIGRATION_OPEN orders to PENDING/SCHEDULED.
func (h *Handlers) HandleReleaseOpenOrders(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, string(PackOpenOrders), "release_open_orders", "")
	if !ok {
		return
	}

	projectID := strings.TrimSpace(chi.URLParam(r, "projectId"))
	sessionID := strings.TrimSpace(chi.URLParam(r, "sessionId"))
	if projectID != "" {
		if _, err := h.repo.GetProject(r.Context(), claims.SupplierID, projectID); err != nil {
			writeRepoError(w, err)
			return
		}
	}
	if sessionID != "" {
		if _, err := h.repo.GetSession(r.Context(), claims.SupplierID, sessionID); err != nil {
			writeRepoError(w, err)
			return
		}
	}

	var req releaseOpenOrdersRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeMigrationError(w, http.StatusBadRequest, "invalid_json", "malformed open-order release request body")
			return
		}
	}

	defaultTarget, err := parseReleaseTargetStatus(req.TargetStatus)
	if err != nil {
		writeMigrationError(w, http.StatusBadRequest, "invalid_target_status", err.Error())
		return
	}

	var items []order.ReleaseOpenOrderItem
	for _, it := range req.Items {
		oid := strings.TrimSpace(it.OrderID)
		if oid == "" {
			continue
		}
		target := defaultTarget
		if strings.TrimSpace(it.TargetStatus) != "" {
			target, err = parseReleaseTargetStatus(it.TargetStatus)
			if err != nil {
				writeMigrationError(w, http.StatusBadRequest, "invalid_target_status", err.Error())
				return
			}
		}
		items = append(items, order.ReleaseOpenOrderItem{OrderID: oid, TargetStatus: target})
	}
	for _, rawID := range req.OrderIDs {
		if oid := strings.TrimSpace(rawID); oid != "" {
			items = append(items, order.ReleaseOpenOrderItem{OrderID: oid, TargetStatus: defaultTarget})
		}
	}

	if h.openOrderReleaser == nil {
		writeMigrationError(w, http.StatusServiceUnavailable, "open_order_releaser_unavailable", "open-order releaser is not configured")
		return
	}

	res, err := h.openOrderReleaser.ReleaseOpenOrders(r.Context(), claims.SupplierID, items, auth.ActorLabel(claims), h.now())
	if err != nil {
		writeRepoError(w, err)
		return
	}
	if len(res.ReleasedOrderIDs) == 0 && len(res.Errors) > 0 {
		writeMigrationJSON(w, http.StatusConflict, map[string]any{
			"error":   "open_orders_release_failed",
			"message": "one or more open orders failed inventory/FEFO reservation",
			"result":  res,
		})
		return
	}
	writeMigrationJSON(w, http.StatusOK, res)
}

func parseReleaseTargetStatus(raw string) (order.Status, error) {
	cleaned := strings.ToUpper(strings.TrimSpace(raw))
	if cleaned == "" {
		return "", nil
	}
	switch order.Status(cleaned) {
	case order.StatusPending, order.StatusScheduled:
		return order.Status(cleaned), nil
	default:
		return "", order.ErrInvalidStatusTransition
	}
}

// HandleCutoverProject verifies the final preflight checklist and permanently transitions project + sessions to CUTOVER_COMPLETE.
func (h *Handlers) HandleCutoverProject(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "", "cutover", "")
	if !ok {
		return
	}
	projectID := strings.TrimSpace(chi.URLParam(r, "projectId"))
	if projectID == "" {
		writeMigrationError(w, http.StatusBadRequest, "project_id_required", "projectId is required")
		return
	}

	res, err := ExecuteProjectCutover(r.Context(), h.repo, CutoverOptions{
		SupplierID:           claims.SupplierID,
		ProjectID:            projectID,
		ActorID:              auth.ActorLabel(claims),
		Now:                  h.now(),
		SoliqHealthPing:      h.soliqHealthPing,
		ReleaseLotsGateCheck: h.releaseLotsGateCheck,
	})
	if errors.Is(err, ErrCutoverChecklistFailed) {
		writeMigrationJSON(w, http.StatusConflict, map[string]any{
			"error":     "cutover_checklist_failed",
			"message":   err.Error(),
			"checklist": res.Checklist,
			"project":   res.Project,
		})
		return
	}
	if err != nil {
		writeRepoError(w, err)
		return
	}
	writeMigrationJSON(w, http.StatusOK, res)
}
