package audit

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// AnchorRequest is the optional JSON body for POST /v1/audit/anchors.
type AnchorRequest struct {
	EntityID  string `json:"entity_id,omitempty"`
	TenantID  string `json:"tenant_id,omitempty"`
	Date      string `json:"date,omitempty"`
	DayBucket string `json:"day_bucket,omitempty"`
}

// LegalHoldRequest is the JSON body for POST /v1/audit/legal-hold.
type LegalHoldRequest struct {
	EntityID   string `json:"entity_id,omitempty"`
	TenantID   string `json:"tenant_id,omitempty"`
	ObjectType string `json:"object_type"`
	ObjectID   string `json:"object_id"`
	Reason     string `json:"reason"`
	ExpiresAt  string `json:"expires_at,omitempty"`
}

// HandleListEvents serves GET /v1/audit/events.
func (s *Service) HandleListEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAuditError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeAuditError(w, http.StatusServiceUnavailable, "audit_unavailable", "audit repository unavailable")
		return
	}

	q := r.URL.Query()
	requestedEntity := firstNonEmpty(q.Get("entity"), q.Get("tenant_id"))
	tenantID, ok := resolveAuditTenantScope(w, r, requestedEntity, true)
	if !ok {
		return
	}

	fromTime, err := parseOptionalTimeParam(q.Get("from"), false)
	if err != nil {
		writeAuditError(w, http.StatusBadRequest, "invalid_from", err.Error())
		return
	}
	toTime, err := parseOptionalTimeParam(q.Get("to"), true)
	if err != nil {
		writeAuditError(w, http.StatusBadRequest, "invalid_to", err.Error())
		return
	}

	dayBucket := strings.TrimSpace(firstNonEmpty(q.Get("date"), q.Get("day_bucket")))
	if dayBucket != "" && !IsValidDayBucket(dayBucket) {
		writeAuditError(w, http.StatusBadRequest, "invalid_day_bucket", ErrInvalidDayBucket.Error())
		return
	}

	limit := parseLimitParam(q.Get("limit"), 100, 500)
	viewerRole := viewerRoleFromRequest(r)

	objectType := strings.TrimSpace(firstNonEmpty(q.Get("object"), q.Get("object_type")))
	objectID := strings.TrimSpace(q.Get("object_id"))
	includeLegacy := strings.EqualFold(strings.TrimSpace(q.Get("include_legacy")), "true") ||
		(objectType != "" && objectID != "")

	var actorType ActorType
	if rawAT := strings.TrimSpace(q.Get("actor_type")); rawAT != "" {
		actorType = NormalizeActorType(rawAT)
	}

	events, err := s.ListEvents(r.Context(), Filter{
		TenantID:      tenantID,
		ObjectType:    objectType,
		ObjectID:      objectID,
		ActorID:       strings.TrimSpace(firstNonEmpty(q.Get("actor"), q.Get("actor_id"))),
		ActorType:     actorType,
		Action:        strings.TrimSpace(q.Get("action")),
		DayBucket:     dayBucket,
		From:          fromTime,
		To:            toTime,
		Limit:         limit,
		IncludeLegacy: includeLegacy,
		ViewerRole:    viewerRole,
	})
	if err != nil {
		writeAuditDomainError(w, err)
		return
	}
	if events == nil {
		events = []AuditEvent{}
	}
	writeAuditJSON(w, http.StatusOK, map[string]any{
		"tenant_id":  tenantID,
		"pii_masked": !IsFinanceOrAdminRole(viewerRole),
		"count":      len(events),
		"events":     events,
	})
}

// HandleObjectHistory serves GET /v1/audit/history/{objectType}/{objectId}.
func (s *Service) HandleObjectHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAuditError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeAuditError(w, http.StatusServiceUnavailable, "audit_unavailable", "audit repository unavailable")
		return
	}

	objectType := strings.TrimSpace(chi.URLParam(r, "objectType"))
	objectID := strings.TrimSpace(chi.URLParam(r, "objectId"))
	if objectType == "" || objectID == "" {
		writeAuditError(w, http.StatusBadRequest, "invalid_object", "objectType and objectId are required")
		return
	}

	q := r.URL.Query()
	requestedEntity := firstNonEmpty(q.Get("entity"), q.Get("tenant_id"))
	tenantID, ok := resolveAuditTenantScope(w, r, requestedEntity, true)
	if !ok {
		return
	}

	limit := parseLimitParam(q.Get("limit"), 100, 500)
	viewerRole := viewerRoleFromRequest(r)

	var events []AuditEvent
	var err error
	if tenantID != "" {
		events, err = s.GetObjectHistory(r.Context(), tenantID, objectType, objectID, viewerRole)
	} else {
		events, err = s.ListEvents(r.Context(), Filter{
			ObjectType:    objectType,
			ObjectID:      objectID,
			Limit:         limit,
			IncludeLegacy: true,
			ViewerRole:    viewerRole,
		})
	}
	if err != nil {
		writeAuditDomainError(w, err)
		return
	}
	if len(events) > limit {
		events = events[:limit]
	}
	if events == nil {
		events = []AuditEvent{}
	}
	writeAuditJSON(w, http.StatusOK, map[string]any{
		"tenant_id":   tenantID,
		"object_type": objectType,
		"object_id":   objectID,
		"pii_masked":  !IsFinanceOrAdminRole(viewerRole),
		"count":       len(events),
		"events":      events,
	})
}

// HandleVerifyDayChain serves GET /v1/audit/verify?date=YYYY-MM-DD&entity=.
func (s *Service) HandleVerifyDayChain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAuditError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeAuditError(w, http.StatusServiceUnavailable, "audit_unavailable", "audit repository unavailable")
		return
	}

	q := r.URL.Query()
	requestedEntity := firstNonEmpty(q.Get("entity"), q.Get("tenant_id"))
	tenantID, ok := resolveAuditTenantScope(w, r, requestedEntity, false)
	if !ok {
		return
	}

	dayBucket := strings.TrimSpace(firstNonEmpty(q.Get("date"), q.Get("day_bucket")))
	if dayBucket == "" {
		dayBucket = time.Now().UTC().Format("2006-01-02")
	} else if !IsValidDayBucket(dayBucket) {
		writeAuditError(w, http.StatusBadRequest, "invalid_day_bucket", ErrInvalidDayBucket.Error())
		return
	}

	res, err := s.VerifyDayChain(r.Context(), tenantID, dayBucket)
	if err != nil {
		writeAuditDomainError(w, err)
		return
	}
	writeAuditJSON(w, http.StatusOK, res)
}

// HandleListAnchors serves GET /v1/audit/anchors?entity=&date=&limit=.
func (s *Service) HandleListAnchors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAuditError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeAuditError(w, http.StatusServiceUnavailable, "audit_unavailable", "audit repository unavailable")
		return
	}

	q := r.URL.Query()
	requestedEntity := firstNonEmpty(q.Get("entity"), q.Get("tenant_id"))
	tenantID, ok := resolveAuditTenantScope(w, r, requestedEntity, true)
	if !ok {
		return
	}
	dayBucket := strings.TrimSpace(firstNonEmpty(q.Get("date"), q.Get("day_bucket")))
	if dayBucket != "" && !IsValidDayBucket(dayBucket) {
		writeAuditError(w, http.StatusBadRequest, "invalid_day_bucket", ErrInvalidDayBucket.Error())
		return
	}
	limit := parseLimitParam(q.Get("limit"), 100, 500)

	anchors, err := s.ListAnchors(r.Context(), tenantID, dayBucket, limit)
	if err != nil {
		writeAuditDomainError(w, err)
		return
	}
	if anchors == nil {
		anchors = []Anchor{}
	}
	writeAuditJSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"count":     len(anchors),
		"anchors":   anchors,
	})
}

// HandleWriteAnchor serves POST /v1/audit/anchors.
func (s *Service) HandleWriteAnchor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAuditError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeAuditError(w, http.StatusServiceUnavailable, "audit_unavailable", "audit repository unavailable")
		return
	}

	var req AnchorRequest
	if r.Body != nil {
		body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
		if err != nil {
			writeAuditError(w, http.StatusBadRequest, "read_body_failed", "unable to read request body")
			return
		}
		defer r.Body.Close()
		if len(strings.TrimSpace(string(body))) > 0 {
			if err := json.Unmarshal(body, &req); err != nil {
				writeAuditError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
				return
			}
		}
	}

	requestedEntity := firstNonEmpty(req.EntityID, req.TenantID, r.URL.Query().Get("entity"), r.URL.Query().Get("tenant_id"))
	tenantID, ok := resolveAuditTenantScope(w, r, requestedEntity, false)
	if !ok {
		return
	}

	dayBucket := strings.TrimSpace(firstNonEmpty(req.Date, req.DayBucket, r.URL.Query().Get("date")))
	if dayBucket == "" {
		dayBucket = time.Now().UTC().Format("2006-01-02")
	} else if !IsValidDayBucket(dayBucket) {
		writeAuditError(w, http.StatusBadRequest, "invalid_day_bucket", ErrInvalidDayBucket.Error())
		return
	}

	anchor, err := s.WriteDailyAnchor(r.Context(), tenantID, dayBucket, auth.ActorFromContext(r.Context()), nil)
	if err != nil {
		writeAuditDomainError(w, err)
		return
	}
	SetMutationContext(r.Context(), MutationContext{
		TenantID:   tenantID,
		ObjectType: "audit_anchor",
		ObjectID:   tenantID + ":" + dayBucket,
		ReasonCode: "DAILY_ANCHOR",
	})
	writeAuditJSON(w, http.StatusCreated, anchor)
}

// HandleCreateLegalHold serves POST /v1/audit/legal-hold.
func (s *Service) HandleCreateLegalHold(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAuditError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeAuditError(w, http.StatusServiceUnavailable, "audit_unavailable", "audit repository unavailable")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 128*1024))
	if err != nil {
		writeAuditError(w, http.StatusBadRequest, "read_body_failed", "unable to read request body")
		return
	}
	defer r.Body.Close()

	var req LegalHoldRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeAuditError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return
	}

	requestedEntity := firstNonEmpty(req.EntityID, req.TenantID, r.URL.Query().Get("entity"), r.URL.Query().Get("tenant_id"))
	tenantID, ok := resolveAuditTenantScope(w, r, requestedEntity, false)
	if !ok {
		return
	}

	var expiresAt *time.Time
	if rawExp := strings.TrimSpace(req.ExpiresAt); rawExp != "" {
		t, err := time.Parse(time.RFC3339Nano, rawExp)
		if err != nil {
			writeAuditError(w, http.StatusBadRequest, "invalid_expires_at", "expires_at must be RFC3339")
			return
		}
		utc := t.UTC()
		expiresAt = &utc
	}

	hold, err := s.CreateLegalHold(r.Context(), LegalHoldInput{
		TenantID:   tenantID,
		ObjectType: req.ObjectType,
		ObjectID:   req.ObjectID,
		Reason:     req.Reason,
		CreatedBy:  auth.ActorFromContext(r.Context()),
		ExpiresAt:  expiresAt,
	})
	if err != nil {
		writeAuditDomainError(w, err)
		return
	}
	SetMutationContext(r.Context(), MutationContext{
		TenantID:   tenantID,
		ObjectType: hold.ObjectType,
		ObjectID:   hold.ObjectID,
		ReasonCode: "LEGAL_HOLD",
	})
	writeAuditJSON(w, http.StatusCreated, hold)
}

// HandleListLegalHolds serves GET /v1/audit/legal-holds.
func (s *Service) HandleListLegalHolds(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAuditError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeAuditError(w, http.StatusServiceUnavailable, "audit_unavailable", "audit repository unavailable")
		return
	}

	q := r.URL.Query()
	requestedEntity := firstNonEmpty(q.Get("entity"), q.Get("tenant_id"))
	tenantID, ok := resolveAuditTenantScope(w, r, requestedEntity, true)
	if !ok {
		return
	}

	limit := parseLimitParam(q.Get("limit"), 100, 500)
	holds, err := s.ListLegalHolds(
		r.Context(),
		tenantID,
		strings.TrimSpace(firstNonEmpty(q.Get("object"), q.Get("object_type"))),
		strings.TrimSpace(q.Get("object_id")),
		strings.TrimSpace(q.Get("status")),
		limit,
	)
	if err != nil {
		writeAuditDomainError(w, err)
		return
	}
	if holds == nil {
		holds = []LegalHold{}
	}
	writeAuditJSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"count":     len(holds),
		"holds":     holds,
	})
}

func resolveAuditTenantScope(w http.ResponseWriter, r *http.Request, requestedEntity string, allowCrossTenantPlatformAdmin bool) (string, bool) {
	requested := strings.TrimSpace(requestedEntity)
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
		target := firstNonEmpty(requested, callerTenant)
		if target == "" && !allowCrossTenantPlatformAdmin {
			writeAuditError(w, http.StatusBadRequest, "entity_required", "entity query/body parameter is required for platform admin")
			return "", false
		}
		return target, true
	}

	if callerTenant == "" {
		writeAuditError(w, http.StatusUnauthorized, "tenant_required", "authenticated supplier tenant context is required")
		return "", false
	}
	if requested != "" && requested != callerTenant {
		writeAuditError(w, http.StatusForbidden, "entity_scope_mismatch", "requested entity does not match authenticated supplier tenant")
		return "", false
	}
	return callerTenant, true
}

func viewerRoleFromRequest(r *http.Request) string {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		return ""
	}
	return auth.EffectiveSupplierRole(claims)
}

func parseOptionalTimeParam(raw string, endOfDay bool) (*time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	if len(trimmed) == 10 && IsValidDayBucket(trimmed) {
		t, _ := time.Parse("2006-01-02", trimmed)
		if endOfDay {
			t = t.Add(24*time.Hour - time.Nanosecond)
		}
		utc := t.UTC()
		return &utc, nil
	}
	t, err := time.Parse(time.RFC3339Nano, trimmed)
	if err != nil {
		return nil, errors.New("timestamp must be YYYY-MM-DD or RFC3339")
	}
	utc := t.UTC()
	return &utc, nil
}

func parseLimitParam(raw string, def, max int) int {
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || v <= 0 {
		return def
	}
	if v > max {
		return max
	}
	return v
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if t := strings.TrimSpace(v); t != "" {
			return t
		}
	}
	return ""
}

func writeAuditDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidAuditInput), errors.Is(err, ErrInvalidDayBucket):
		writeAuditError(w, http.StatusBadRequest, "invalid_input", err.Error())
	case errors.Is(err, ErrChainBroken):
		writeAuditError(w, http.StatusConflict, "audit_chain_broken", err.Error())
	case errors.Is(err, ErrAnchorWormRequired), errors.Is(err, ErrRepoUnavailable):
		writeAuditError(w, http.StatusServiceUnavailable, "audit_storage_unavailable", err.Error())
	case errors.Is(err, ErrHoldNotFound):
		writeAuditError(w, http.StatusNotFound, "legal_hold_not_found", err.Error())
	default:
		writeAuditError(w, http.StatusInternalServerError, "audit_internal_error", err.Error())
	}
}

func writeAuditError(w http.ResponseWriter, status int, code, msg string) {
	writeAuditJSON(w, status, map[string]string{
		"error":   code,
		"code":    code,
		"message": msg,
	})
}

func writeAuditJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
