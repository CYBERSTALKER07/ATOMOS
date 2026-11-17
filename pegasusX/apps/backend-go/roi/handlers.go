package roi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

type captureBaselineHTTPReq struct {
	TenantID               string         `json:"tenant_id,omitempty"`
	WindowDays             int64          `json:"window_days,omitempty"`
	BaselineArMinor        *int64         `json:"baseline_ar_minor,omitempty"`
	BaselineBadDebtMinor   *int64         `json:"baseline_bad_debt_minor,omitempty"`
	BaselineExpiredMinor   *int64         `json:"baseline_expired_minor,omitempty"`
	BaselineStockoutMinor  int64          `json:"baseline_stockout_minor,omitempty"`
	BaselineLogisticsMinor int64          `json:"baseline_logistics_minor,omitempty"`
	BaselineLabourHours    int64          `json:"baseline_labour_hours,omitempty"`
	SurveyHourlyWageMinor  int64          `json:"survey_hourly_wage_minor,omitempty"`
	SurveyNotes            string         `json:"survey_notes,omitempty"`
	Survey                 map[string]any `json:"survey,omitempty"`
	Currency               string         `json:"currency,omitempty"`
}

type recordEventHTTPReq struct {
	TenantID        string `json:"tenant_id,omitempty"`
	Category        string `json:"category"`
	SavingsMinor    int64  `json:"savings_minor"`
	Currency        string `json:"currency,omitempty"`
	Methodology     string `json:"methodology,omitempty"`
	ConfidenceBps   int64  `json:"confidence_bps,omitempty"`
	IsEstimate      *bool  `json:"is_estimate,omitempty"`
	HoldoutControl  bool   `json:"holdout_control,omitempty"`
	ProposalID      string `json:"proposal_id,omitempty"`
	EvidenceRefType string `json:"evidence_ref_type,omitempty"`
	EvidenceRefID   string `json:"evidence_ref_id,omitempty"`
	Notes           string `json:"notes,omitempty"`
	IdempotencyKey  string `json:"idempotency_key,omitempty"`
	OccurredAt      string `json:"occurred_at,omitempty"`
}

// HandleGetBaseline serves GET /v1/roi/baseline.
func (s *Service) HandleGetBaseline(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "roi_unavailable", "roi service unavailable")
		return
	}
	tenantID, ok := resolveRoiTenantScope(w, r, r.URL.Query().Get("tenant_id"))
	if !ok {
		return
	}
	b, found, err := s.GetLatestBaseline(r.Context(), tenantID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if !found {
		writeJSON(w, http.StatusOK, map[string]any{
			"tenant_id": tenantID,
			"baseline":  nil,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"baseline":  b,
	})
}

// HandleCaptureBaseline serves POST /v1/roi/baseline.
func (s *Service) HandleCaptureBaseline(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "roi_unavailable", "roi service unavailable")
		return
	}
	if !requireNonAuditorNonAgent(w, r) {
		return
	}
	var req captureBaselineHTTPReq
	if !decodeJSONBody(w, r, &req, true) {
		return
	}
	tenantID, ok := resolveRoiTenantScope(w, r, coalesceNonEmpty(req.TenantID, r.URL.Query().Get("tenant_id")))
	if !ok {
		return
	}
	b, err := s.CaptureBaseline(r.Context(), CaptureBaselineInput{
		TenantID:               tenantID,
		WindowDays:             req.WindowDays,
		BaselineArMinor:        req.BaselineArMinor,
		BaselineBadDebtMinor:   req.BaselineBadDebtMinor,
		BaselineExpiredMinor:   req.BaselineExpiredMinor,
		BaselineStockoutMinor:  req.BaselineStockoutMinor,
		BaselineLogisticsMinor: req.BaselineLogisticsMinor,
		BaselineLabourHours:    req.BaselineLabourHours,
		SurveyHourlyWageMinor:  req.SurveyHourlyWageMinor,
		SurveyNotes:            req.SurveyNotes,
		Survey:                 req.Survey,
		Currency:               req.Currency,
		CapturedBy:             auth.ActorFromContext(r.Context()),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   tenantID,
		Action:     "roi.baseline.capture",
		ObjectType: "RoiBaseline",
		ObjectID:   b.BaselineID,
	})
	writeJSON(w, http.StatusCreated, map[string]any{
		"tenant_id": tenantID,
		"baseline":  b,
	})
}

// HandleGetSummary serves GET /v1/roi/summary?days=90.
func (s *Service) HandleGetSummary(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "roi_unavailable", "roi service unavailable")
		return
	}
	tenantID, ok := resolveRoiTenantScope(w, r, r.URL.Query().Get("tenant_id"))
	if !ok {
		return
	}
	days := parseDaysParam(r.URL.Query().Get("days"), 90)
	sum, err := s.GetSummary(r.Context(), tenantID, days)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sum)
}

// HandleListEvents serves GET /v1/roi/events?category=&limit=.
func (s *Service) HandleListEvents(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "roi_unavailable", "roi service unavailable")
		return
	}
	tenantID, ok := resolveRoiTenantScope(w, r, r.URL.Query().Get("tenant_id"))
	if !ok {
		return
	}
	limit := 100
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	list, err := s.ListEvents(r.Context(), EventsFilter{
		TenantID: tenantID,
		Category: r.URL.Query().Get("category"),
		Limit:    limit,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if list == nil {
		list = []RoiEvent{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"events":    list,
		"count":     len(list),
	})
}

// HandleRecordEvent serves POST /v1/roi/events.
func (s *Service) HandleRecordEvent(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "roi_unavailable", "roi service unavailable")
		return
	}
	if !requireNonAuditorNonAgent(w, r) {
		return
	}
	var req recordEventHTTPReq
	if !decodeJSONBody(w, r, &req, false) {
		return
	}
	tenantID, ok := resolveRoiTenantScope(w, r, coalesceNonEmpty(req.TenantID, r.URL.Query().Get("tenant_id")))
	if !ok {
		return
	}
	var occurredAt time.Time
	if raw := strings.TrimSpace(req.OccurredAt); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid_request", "occurred_at must be RFC3339 timestamp")
			return
		}
		occurredAt = parsed
	}
	idem := coalesceNonEmpty(req.IdempotencyKey, r.Header.Get("Idempotency-Key"))
	ev, err := s.RecordEvent(r.Context(), RecordEventInput{
		TenantID:        tenantID,
		Category:        req.Category,
		SavingsMinor:    req.SavingsMinor,
		Currency:        req.Currency,
		Methodology:     req.Methodology,
		ConfidenceBps:   req.ConfidenceBps,
		IsEstimate:      req.IsEstimate,
		HoldoutControl:  req.HoldoutControl,
		ProposalID:      req.ProposalID,
		EvidenceRefType: req.EvidenceRefType,
		EvidenceRefID:   req.EvidenceRefID,
		Notes:           req.Notes,
		RecordedBy:      auth.ActorFromContext(r.Context()),
		IdempotencyKey:  idem,
		OccurredAt:      occurredAt,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   tenantID,
		Action:     "roi.event.record",
		ObjectType: "RoiEvent",
		ObjectID:   ev.EventID,
	})
	writeJSON(w, http.StatusCreated, map[string]any{
		"tenant_id": tenantID,
		"event":     ev,
	})
}

// HandleExportReport serves GET /v1/roi/report?format=json|csv&days=90.
func (s *Service) HandleExportReport(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "roi_unavailable", "roi service unavailable")
		return
	}
	tenantID, ok := resolveRoiTenantScope(w, r, r.URL.Query().Get("tenant_id"))
	if !ok {
		return
	}
	days := parseDaysParam(r.URL.Query().Get("days"), 90)
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		format = "json"
	}
	switch format {
	case "csv":
		csvBytes, err := s.ExportQBRCSV(r.Context(), tenantID, days)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="roi-qbr-`+tenantID+`.csv"`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(csvBytes)
	case "json":
		rep, err := s.ExportQBR(r.Context(), tenantID, days)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, rep)
	default:
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "format must be json or csv")
	}
}

func parseDaysParam(raw string, def int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

func requireNonAuditorNonAgent(w http.ResponseWriter, r *http.Request) bool {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		return true
	}
	if auth.IsReadOnlyAuditor(claims) {
		writeJSONError(w, http.StatusForbidden, "auditor_read_only", "AUDITOR role is strictly read-only")
		return false
	}
	if auth.IsAgentActor(claims) {
		writeJSONError(w, http.StatusForbidden, "agent_mutation_forbidden", "AGENT actor cannot directly mutate ROI records via HTTP")
		return false
	}
	return true
}

func resolveRoiTenantScope(w http.ResponseWriter, r *http.Request, requested string) (string, bool) {
	req := strings.TrimSpace(requested)
	ctx := r.Context()

	var callerTenant string
	if t, ok := auth.TenantFromContext(ctx); ok && strings.TrimSpace(t.SupplierID) != "" {
		callerTenant = strings.TrimSpace(t.SupplierID)
	} else if !auth.TenantContextEnforced() {
		if sid, ok := auth.ResolveSupplierID(ctx); ok {
			callerTenant = strings.TrimSpace(sid)
		}
	}

	if auth.IsPlatformAdmin(ctx) {
		target := coalesceNonEmpty(req, callerTenant)
		if target == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_required", "tenant_id parameter is required for platform admin")
			return "", false
		}
		return target, true
	}

	if callerTenant == "" {
		writeJSONError(w, http.StatusUnauthorized, "tenant_required", "authenticated supplier tenant context is required")
		return "", false
	}
	if req != "" && req != callerTenant {
		writeJSONError(w, http.StatusForbidden, "tenant_scope_mismatch", "requested tenant_id does not match authenticated supplier tenant")
		return "", false
	}
	return callerTenant, true
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any, allowEmpty bool) bool {
	if r.Body == nil {
		if allowEmpty {
			return true
		}
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "request body is required")
		return false
	}
	defer r.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(r.Body, 512*1024))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "read_body_failed", "unable to read request body")
		return false
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		if allowEmpty {
			return true
		}
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "request body is required")
		return false
	}
	var generic any
	gdec := json.NewDecoder(bytes.NewReader(raw))
	gdec.UseNumber()
	if err := gdec.Decode(&generic); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return false
	}
	if err := validateNoFloats(generic); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return false
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(dst); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return false
	}
	return true
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrFloatForbidden), errors.Is(err, ErrInvalidInput):
		writeJSONError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, ErrForbidden):
		writeJSONError(w, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, ErrRepoUnavailable):
		writeJSONError(w, http.StatusServiceUnavailable, "roi_unavailable", err.Error())
	default:
		writeJSONError(w, http.StatusInternalServerError, "roi_internal_error", err.Error())
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error":   message,
		"code":    code,
		"message": message,
	})
}

func coalesceNonEmpty(vals ...string) string {
	for _, v := range vals {
		if t := strings.TrimSpace(v); t != "" {
			return t
		}
	}
	return ""
}
