package migration

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

type dryRunRequest struct {
	ApplyMode string `json:"apply_mode,omitempty"`
}

// HandleDryRunProject runs POST /v1/supplier/migrations/projects/{projectId}/dry-run.
// It simulates all staged packs in DAG topological order against Live Spanner U Valid Staged Preceding Packs without mutating live tables.
func (h *Handlers) HandleDryRunProject(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "", "write", "")
	if !ok {
		return
	}
	projectID := strings.TrimSpace(chi.URLParam(r, "projectId"))
	if projectID == "" {
		writeMigrationError(w, http.StatusBadRequest, "project_id_required", "projectId path param is required")
		return
	}
	var req dryRunRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
			writeMigrationError(w, http.StatusBadRequest, "invalid_json", "malformed dry-run request body")
			return
		}
	}
	rep, summary, err := EvaluateProjectDryRun(r.Context(), h.repo, DryRunOptions{
		SupplierID: claims.SupplierID,
		ProjectID:  projectID,
		ApplyMode:  firstNonEmpty(req.ApplyMode, r.URL.Query().Get("apply_mode")),
		Now:        h.now(),
		NewID:      h.newID,
		LiveReader: h.liveReader,
	})
	if err != nil {
		writeRepoError(w, err)
		return
	}
	writeMigrationJSON(w, http.StatusOK, map[string]any{
		"report":  rep,
		"summary": summary,
	})
}

// HandleDryRunSession runs POST /v1/supplier/migrations/sessions/{sessionId}/dry-run.
func (h *Handlers) HandleDryRunSession(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "", "write", "")
	if !ok {
		return
	}
	sessionID := strings.TrimSpace(chi.URLParam(r, "sessionId"))
	if sessionID == "" {
		writeMigrationError(w, http.StatusBadRequest, "session_id_required", "sessionId path param is required")
		return
	}
	var req dryRunRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
			writeMigrationError(w, http.StatusBadRequest, "invalid_json", "malformed dry-run request body")
			return
		}
	}
	rep, summary, err := EvaluateSessionDryRun(r.Context(), h.repo, DryRunOptions{
		SupplierID: claims.SupplierID,
		SessionID:  sessionID,
		ApplyMode:  firstNonEmpty(req.ApplyMode, r.URL.Query().Get("apply_mode")),
		Now:        h.now(),
		NewID:      h.newID,
		LiveReader: h.liveReader,
	})
	if err != nil {
		writeRepoError(w, err)
		return
	}
	writeMigrationJSON(w, http.StatusOK, map[string]any{
		"report":  rep,
		"summary": summary,
	})
}

// HandleGetReport runs GET /v1/supplier/migrations/reports/{reportId}.
func (h *Handlers) HandleGetReport(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "", "read", "")
	if !ok {
		return
	}
	reportID := strings.TrimSpace(chi.URLParam(r, "reportId"))
	if reportID == "" {
		writeMigrationError(w, http.StatusBadRequest, "report_id_required", "reportId path param is required")
		return
	}
	rep, err := h.repo.GetMigrationReport(r.Context(), claims.SupplierID, reportID)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	var summary DryRunSummary
	_ = json.Unmarshal(rep.SummaryJSON, &summary)
	writeMigrationJSON(w, http.StatusOK, map[string]any{
		"report":  rep,
		"summary": summary,
	})
}

// HandleApproveSession runs POST /v1/supplier/migrations/sessions/{sessionId}/approve.
func (h *Handlers) HandleApproveSession(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "", "approve", "")
	if !ok {
		return
	}
	sessionID := strings.TrimSpace(chi.URLParam(r, "sessionId"))
	if sessionID == "" {
		writeMigrationError(w, http.StatusBadRequest, "session_id_required", "sessionId path param is required")
		return
	}
	var req ApproveSessionRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
			writeMigrationError(w, http.StatusBadRequest, "invalid_json", "malformed approve request body")
			return
		}
	}
	sess, err := h.repo.GetSession(r.Context(), claims.SupplierID, sessionID)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	action := "approve"
	if NormalizeApplyMode(firstNonEmpty(req.ApplyMode, sess.ApplyMode)) == ApplyModeSetOnHand {
		action = "set_on_hand"
	}
	if _, ok := h.authorize(w, r, string(sess.Pack), action, claims.SupplierID); !ok {
		return
	}

	approved, err := ApproveSession(r.Context(), h.repo, claims.SupplierID, sessionID, req, h.now())
	if err != nil {
		writeRepoError(w, err)
		return
	}
	writeMigrationJSON(w, http.StatusOK, map[string]any{
		"session": approved,
	})
}

// HandleRollbackSession runs POST /v1/supplier/migrations/sessions/{sessionId}/rollback.
func (h *Handlers) HandleRollbackSession(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "", "rollback", "")
	if !ok {
		return
	}
	sessionID := strings.TrimSpace(chi.URLParam(r, "sessionId"))
	if sessionID == "" {
		writeMigrationError(w, http.StatusBadRequest, "session_id_required", "sessionId path param is required")
		return
	}
	res, err := RollbackSession(r.Context(), h.repo, claims.SupplierID, sessionID, h.now())
	if err != nil {
		writeRepoError(w, err)
		return
	}
	writeMigrationJSON(w, http.StatusOK, res)
}
