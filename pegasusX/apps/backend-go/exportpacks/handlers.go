package exportpacks

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
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type createPackRequest struct {
	TenantID string `json:"tenant_id,omitempty"`
	PackType string `json:"pack_type"`
	EntityID string `json:"entity_id,omitempty"`
	PeriodID string `json:"period_id,omitempty"`
	FromDate string `json:"from_date,omitempty"`
	ToDate   string `json:"to_date,omitempty"`
	From     string `json:"from,omitempty"`
	To       string `json:"to,omitempty"`
	Format   string `json:"format,omitempty"`
	TTLHours int64  `json:"ttl_hours,omitempty"`
}

// HandleCreatePack serves POST /v1/finance/packs.
func (s *Service) HandleCreatePack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writePackError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writePackError(w, http.StatusServiceUnavailable, "export_packs_unavailable", "export packs service unavailable")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 128*1024))
	if err != nil {
		writePackError(w, http.StatusBadRequest, "read_body_failed", "unable to read request body")
		return
	}
	defer r.Body.Close()

	var req createPackRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writePackError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return
	}

	requestedEntity := firstNonEmpty(req.EntityID, req.TenantID, r.URL.Query().Get("entity"))
	tenantID, ok := resolvePackTenantScope(w, r, requestedEntity, false)
	if !ok {
		return
	}

	fromStr := firstNonEmpty(req.FromDate, req.From)
	toStr := firstNonEmpty(req.ToDate, req.To)
	var fromPtr, toPtr *time.Time
	if fromStr != "" {
		t, perr := parsePackTime(fromStr, false)
		if perr != nil {
			writePackError(w, http.StatusBadRequest, "invalid_from_date", "from_date must be RFC3339 or YYYY-MM-DD")
			return
		}
		fromPtr = &t
	}
	if toStr != "" {
		t, perr := parsePackTime(toStr, true)
		if perr != nil {
			writePackError(w, http.StatusBadRequest, "invalid_to_date", "to_date must be RFC3339 or YYYY-MM-DD")
			return
		}
		toPtr = &t
	}

	claims, _ := auth.FromContext(r.Context())
	pack, err := s.CreatePack(r.Context(), CreatePackInput{
		TenantID:  tenantID,
		PackType:  req.PackType,
		EntityID:  firstNonEmpty(req.EntityID, tenantID),
		PeriodID:  req.PeriodID,
		FromDate:  fromPtr,
		ToDate:    toPtr,
		Format:    req.Format,
		TTLHours:  req.TTLHours,
		ActorID:   auth.ActorFromContext(r.Context()),
		ActorRole: auth.EffectiveSupplierRole(claims),
		ActorType: claims.ActorType,
		TraceID:   firstNonEmpty(outbox.TraceIDFromContext(r.Context()), claims.TraceID),
		IP:        r.RemoteAddr,
		DeviceID:  strings.TrimSpace(r.Header.Get("X-Device-Id")),
	})
	if err != nil {
		writePackDomainError(w, err)
		return
	}
	writePackJSON(w, http.StatusCreated, pack)
}

// HandleListPacks serves GET /v1/finance/packs.
func (s *Service) HandleListPacks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writePackError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writePackError(w, http.StatusServiceUnavailable, "export_packs_unavailable", "export packs service unavailable")
		return
	}
	tenantID, ok := resolvePackTenantScope(w, r, r.URL.Query().Get("entity"), true)
	if !ok {
		return
	}
	limit := 50
	if rawLim := strings.TrimSpace(r.URL.Query().Get("limit")); rawLim != "" {
		if n, err := strconv.Atoi(rawLim); err == nil && n > 0 {
			limit = n
		}
	}
	packs, err := s.ListPacks(r.Context(), tenantID, strings.TrimSpace(r.URL.Query().Get("entity")), limit)
	if err != nil {
		writePackDomainError(w, err)
		return
	}
	writePackJSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"packs":     packs,
		"count":     len(packs),
	})
}

// HandleGetPack serves GET /v1/finance/packs/{packID}.
func (s *Service) HandleGetPack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writePackError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writePackError(w, http.StatusServiceUnavailable, "export_packs_unavailable", "export packs service unavailable")
		return
	}
	packID := strings.TrimSpace(chi.URLParam(r, "packID"))
	if packID == "" {
		writePackError(w, http.StatusBadRequest, "pack_id_required", "packID is required")
		return
	}
	tenantID, ok := resolvePackTenantScope(w, r, r.URL.Query().Get("entity"), true)
	if !ok {
		return
	}
	pack, err := s.GetPack(r.Context(), tenantID, packID)
	if err != nil {
		writePackDomainError(w, err)
		return
	}
	writePackJSON(w, http.StatusOK, pack)
}

// HandleDownloadPack serves GET /v1/finance/packs/{packID}/download?format=&token=.
func (s *Service) HandleDownloadPack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writePackError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writePackError(w, http.StatusServiceUnavailable, "export_packs_unavailable", "export packs service unavailable")
		return
	}
	packID := strings.TrimSpace(chi.URLParam(r, "packID"))
	if packID == "" {
		writePackError(w, http.StatusBadRequest, "pack_id_required", "packID is required")
		return
	}

	claims, hasClaims := auth.FromContext(r.Context())
	qToken := strings.TrimSpace(r.URL.Query().Get("token"))
	reqEntity := strings.TrimSpace(r.URL.Query().Get("entity"))

	var tenantID string
	if hasClaims {
		var ok bool
		tenantID, ok = resolvePackTenantScope(w, r, reqEntity, true)
		if !ok {
			return
		}
		// When authenticated via Bearer/Cookie JWT and ?token= carries the JWT itself (not an opaque download token),
		// do not treat ?token= as the pack's opaque DownloadToken.
		if bearer := auth.BearerToken(r); bearer != "" && qToken == bearer {
			qToken = ""
		} else if c, cerr := r.Cookie(auth.CookieName); cerr == nil && qToken == c.Value {
			qToken = ""
		} else if strings.Count(qToken, ".") == 2 {
			// Also ignore ?token= when it was the JWT used to authenticate this request.
			qToken = ""
		}
	}

	res, err := s.DownloadPack(r.Context(), DownloadPackInput{
		TenantID:      tenantID,
		EntityID:      reqEntity,
		PackID:        packID,
		Format:        r.URL.Query().Get("format"),
		DownloadToken: qToken,
		ActorID:       auth.ActorFromContext(r.Context()),
		ActorRole:     auth.EffectiveSupplierRole(claims),
		ActorType:     claims.ActorType,
		TraceID:       firstNonEmpty(outbox.TraceIDFromContext(r.Context()), claims.TraceID),
		IP:            r.RemoteAddr,
		DeviceID:      strings.TrimSpace(r.Header.Get("X-Device-Id")),
	})
	if err != nil {
		writePackDomainError(w, err)
		return
	}

	w.Header().Set("Content-Type", res.ContentType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+sanitizeHeaderFilename(res.Filename)+`"`)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Atomos-Watermark", res.Watermark)
	w.Header().Set("X-Atomos-Pack-Sha256", res.Sha256Hex)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(res.Data)
}

func sanitizeHeaderFilename(name string) string {
	cleaned := strings.NewReplacer(`"`, "", "\r", "", "\n", "", "/", "_", "\\", "_").Replace(strings.TrimSpace(name))
	if cleaned == "" {
		return "export-pack.bin"
	}
	return cleaned
}

func resolvePackTenantScope(w http.ResponseWriter, r *http.Request, requestedEntity string, allowCrossTenantPlatformRead bool) (string, bool) {
	requested := strings.TrimSpace(requestedEntity)
	ctx := r.Context()

	var callerEntity string
	if t, ok := auth.TenantFromContext(ctx); ok && strings.TrimSpace(t.SupplierID) != "" {
		callerEntity = strings.TrimSpace(t.SupplierID)
	} else if !auth.TenantContextEnforced() {
		if sid, ok := auth.ResolveSupplierID(ctx); ok {
			callerEntity = strings.TrimSpace(sid)
		}
	}

	if auth.IsPlatformAdmin(ctx) {
		target := firstNonEmpty(requested, callerEntity)
		if target == "" {
			if allowCrossTenantPlatformRead {
				return "", true
			}
			writePackError(w, http.StatusBadRequest, "entity_required", "entity parameter is required for platform admin")
			return "", false
		}
		return target, true
	}

	if callerEntity == "" {
		writePackError(w, http.StatusUnauthorized, "tenant_required", "authenticated supplier tenant context is required")
		return "", false
	}
	if requested != "" && requested != callerEntity {
		writePackError(w, http.StatusForbidden, "entity_scope_mismatch", "requested entity does not match authenticated supplier tenant")
		return "", false
	}
	return callerEntity, true
}

func writePackDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrPackExpired):
		writePackError(w, http.StatusGone, "export_pack_expired", err.Error())
	case errors.Is(err, ErrInvalidDownloadToken), errors.Is(err, ErrForbidden):
		writePackError(w, http.StatusForbidden, "export_pack_forbidden", err.Error())
	case errors.Is(err, ErrNotFound):
		writePackError(w, http.StatusNotFound, "export_pack_not_found", err.Error())
	case errors.Is(err, ErrInvalidInput):
		writePackError(w, http.StatusBadRequest, "invalid_export_pack_input", err.Error())
	case errors.Is(err, ErrStorageUnavailable), errors.Is(err, ErrRepoUnavailable):
		writePackError(w, http.StatusServiceUnavailable, "export_packs_unavailable", err.Error())
	default:
		writePackError(w, http.StatusInternalServerError, "export_packs_internal_error", err.Error())
	}
}

func writePackJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writePackError(w http.ResponseWriter, status int, code, msg string) {
	writePackJSON(w, status, map[string]any{
		"error":   code,
		"code":    code,
		"message": msg,
	})
}

func parsePackTime(raw string, endOfDay bool) (time.Time, error) {
	s := strings.TrimSpace(raw)
	if len(s) == 10 {
		d, err := time.Parse("2006-01-02", s)
		if err != nil {
			return time.Time{}, err
		}
		if endOfDay {
			return d.UTC().Add(24*time.Hour - time.Nanosecond), nil
		}
		return d.UTC(), nil
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.UTC(), nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
