package controltower

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// Handlers serves control-tower playbook HTTP APIs.
type Handlers struct {
	Service    *Service
	SupplierID func(*http.Request) string
}

func NewHandlers(svc *Service, supplierID func(*http.Request) string) *Handlers {
	return &Handlers{Service: svc, SupplierID: supplierID}
}

func (h *Handlers) guardEnabled(w http.ResponseWriter) bool {
	if h.Service == nil || !h.Service.Enabled() {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "playbooks_disabled"})
		return false
	}
	return true
}

func (h *Handlers) HandlePlaybooks(w http.ResponseWriter, r *http.Request) {
	if !h.guardEnabled(w) {
		return
	}
	sid := h.SupplierID(r)
	switch r.Method {
	case http.MethodGet:
		rows, err := h.Service.ListPlaybooks(r.Context(), sid)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_playbooks_failed"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"playbooks": rows})
	case http.MethodPost:
		body, _ := io.ReadAll(io.LimitReader(r.Body, 64*1024))
		var in CreatePlaybookInput
		if err := json.Unmarshal(body, &in); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		claims, _ := auth.FromContext(r.Context())
		pb, err := h.Service.CreatePlaybook(r.Context(), sid, claims.Subject, in)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "create_playbook_failed"})
			return
		}
		writeJSON(w, http.StatusCreated, pb)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (h *Handlers) HandleDeactivatePlaybook(w http.ResponseWriter, r *http.Request) {
	if !h.guardEnabled(w) {
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	id := chi.URLParam(r, "id")
	if err := h.Service.DeactivatePlaybook(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "deactivate_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deactivated"})
}

func (h *Handlers) HandlePlaybookByID(w http.ResponseWriter, r *http.Request) {
	if !h.guardEnabled(w) {
		return
	}
	id := chi.URLParam(r, "id")
	switch r.Method {
	case http.MethodPatch:
		body, _ := io.ReadAll(io.LimitReader(r.Body, 64*1024))
		var in UpdatePlaybookInput
		if err := json.Unmarshal(body, &in); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		if err := h.Service.UpdatePlaybook(r.Context(), id, in); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update_playbook_failed"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (h *Handlers) HandleRuns(w http.ResponseWriter, r *http.Request) {
	if !h.guardEnabled(w) {
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	sid := h.SupplierID(r)
	status := r.URL.Query().Get("status")
	exceptionID := strings.TrimSpace(r.URL.Query().Get("exception_id"))
	var runs []PlaybookRun
	var err error
	if exceptionID != "" {
		runs, err = h.Service.ListRunsForException(r.Context(), exceptionID)
	} else {
		runs, err = h.Service.ListRuns(r.Context(), sid, status)
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_runs_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"runs": runs})
}

func (h *Handlers) HandleRunAction(w http.ResponseWriter, r *http.Request) {
	if !h.guardEnabled(w) {
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	runID := chi.URLParam(r, "id")
	action := chi.URLParam(r, "action")
	claims, _ := auth.FromContext(r.Context())
	actor := claims.Subject
	var err error
	switch action {
	case "approve":
		err = h.Service.ApproveRun(r.Context(), runID, actor)
	case "skip":
		err = h.Service.SkipRun(r.Context(), runID, actor)
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown_action"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handlers) HandleEvaluate(w http.ResponseWriter, r *http.Request) {
	if !h.guardEnabled(w) {
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	sid := h.SupplierID(r)
	var body struct {
		ExceptionIDs []string `json:"exception_ids"`
	}
	_ = json.NewDecoder(io.LimitReader(r.Body, 16*1024)).Decode(&body)
	if err := h.Service.Evaluate(r.Context(), sid, body.ExceptionIDs); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "evaluate_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "evaluated"})
}

// HandleScoredExceptions serves GET /v1/control-tower/exceptions/scored.
func (h *Handlers) HandleScoredExceptions(w http.ResponseWriter, r *http.Request) {
	if !h.guardEnabled(w) {
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	sid := h.SupplierID(r)
	limit := 50
	if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	rows, err := h.Service.ListOpenScored(r.Context(), sid, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_scored_exceptions_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"exceptions": rows})
}

// Worker periodically evaluates playbooks for suppliers with open exceptions.
type Worker struct {
	svc      *Service
	log      *slog.Logger
	interval time.Duration
}

func NewWorker(svc *Service, log *slog.Logger, interval time.Duration) *Worker {
	if log == nil {
		log = slog.Default()
	}
	if interval <= 0 {
		interval = 3 * time.Minute
	}
	return &Worker{svc: svc, log: log, interval: interval}
}

func (w *Worker) Run(ctx context.Context) {
	if w == nil || w.svc == nil || !w.svc.Enabled() {
		return
	}
	if err := w.svc.SeedIfEmpty(ctx); err != nil {
		w.log.Warn("playbook seed failed", "err", err)
	}
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *Worker) tick(ctx context.Context) {
	ids, err := w.svc.ListSupplierIDsWithOpenExceptions(ctx)
	if err != nil {
		w.log.Warn("playbook supplier list failed", "err", err)
		return
	}
	for _, sid := range ids {
		if err := w.svc.Evaluate(ctx, sid, nil); err != nil {
			w.log.Warn("playbook evaluate failed", "supplier_id", sid, "err", err)
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type AbortManifestRequest struct {
	ReasonCode    string `json:"reason_code"`
	OperatorNotes string `json:"operator_notes,omitempty"`
}

func (h *Handlers) HandleManifestAbort(w http.ResponseWriter, r *http.Request) {
	if !h.guardEnabled(w) {
		return
	}
	manifestID := chi.URLParam(r, "id")
	supplierID := h.SupplierID(r)
	if supplierID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	operatorID := claims.Subject

	var req AbortManifestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_payload"})
		return
	}

	if req.ReasonCode == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing_reason_code"})
		return
	}

	err := h.Service.AbortManifest(r.Context(), manifestID, supplierID, operatorID, req.ReasonCode, req.OperatorNotes)
	if err != nil {
		slog.Error("Failed to abort manifest", "err", err, "manifest_id", manifestID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "aborted"})
}
