package ontology

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// RebuildRequest is the optional JSON payload for POST /v1/ontology/rebuild.
type RebuildRequest struct {
	CellID     string `json:"cell_id,omitempty"`
	SupplierID string `json:"supplier_id,omitempty"`
	EntityID   string `json:"entity_id,omitempty"`
}

// HandleListTypes serves GET /v1/ontology/types.
func (s *Service) HandleListTypes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeOntologyError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil {
		writeOntologyError(w, http.StatusServiceUnavailable, "ontology_unavailable", "ontology service unavailable")
		return
	}
	writeOntologyJSON(w, http.StatusOK, s.ListTypes())
}

// HandleListObjects serves GET /v1/ontology/objects.
func (s *Service) HandleListObjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeOntologyError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeOntologyError(w, http.StatusServiceUnavailable, "ontology_unavailable", "ontology repository unavailable")
		return
	}

	q := r.URL.Query()
	requestedSup := firstNonEmpty(q.Get("supplier_id"), q.Get("entity"), q.Get("tenant_id"))
	supID, ok := resolveOntologyTenantScope(w, r, requestedSup, true)
	if !ok {
		return
	}

	objType := strings.TrimSpace(firstNonEmpty(q.Get("type"), q.Get("object_type")))
	if objType != "" {
		if _, known := LookupType(objType); !known {
			writeOntologyError(w, http.StatusBadRequest, "unknown_object_type", "unknown v1 object_type")
			return
		}
	}

	limit, err := parseBoundedInt(q.Get("limit"), 50, 1, 200)
	if err != nil {
		writeOntologyError(w, http.StatusBadRequest, "invalid_limit", err.Error())
		return
	}
	offset, err := parseBoundedInt(q.Get("offset"), 0, 0, 100000)
	if err != nil {
		writeOntologyError(w, http.StatusBadRequest, "invalid_offset", err.Error())
		return
	}

	res, err := s.ListObjects(r.Context(), ListObjectsFilter{
		CellID:     s.resolveCellID(q.Get("cell_id")),
		SupplierID: supID,
		ObjectType: objType,
		Search:     strings.TrimSpace(firstNonEmpty(q.Get("search"), q.Get("q"))),
		Marking:    strings.TrimSpace(q.Get("marking")),
		Limit:      limit,
		Offset:     offset,
		ViewerRole: viewerRoleFromRequest(r),
	})
	if err != nil {
		writeOntologyDomainError(w, err)
		return
	}
	writeOntologyJSON(w, http.StatusOK, res)
}

// HandleGetNeighbors serves GET /v1/ontology/objects/{type}/{id}/neighbors.
func (s *Service) HandleGetNeighbors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeOntologyError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeOntologyError(w, http.StatusServiceUnavailable, "ontology_unavailable", "ontology repository unavailable")
		return
	}

	objType := strings.TrimSpace(chi.URLParam(r, "type"))
	objID := strings.TrimSpace(chi.URLParam(r, "id"))
	if objType == "" || objID == "" {
		writeOntologyError(w, http.StatusBadRequest, "invalid_identity", "object type and id path parameters are required")
		return
	}
	if _, known := LookupType(objType); !known {
		writeOntologyError(w, http.StatusBadRequest, "unknown_object_type", "unknown v1 object_type")
		return
	}

	q := r.URL.Query()
	requestedSup := firstNonEmpty(q.Get("supplier_id"), q.Get("entity"), q.Get("tenant_id"))
	supID, ok := resolveOntologyTenantScope(w, r, requestedSup, true)
	if !ok {
		return
	}

	depth, err := parseBoundedInt(q.Get("depth"), 2, 1, 4)
	if err != nil {
		writeOntologyError(w, http.StatusBadRequest, "invalid_depth", err.Error())
		return
	}

	var linkTypes []string
	for _, rawList := range append(q["link_type"], q["link_types"]...) {
		for _, part := range strings.Split(rawList, ",") {
			lt := strings.TrimSpace(part)
			if lt == "" {
				continue
			}
			if !IsValidLinkType(lt) {
				writeOntologyError(w, http.StatusBadRequest, "unknown_link_type", "unknown v1 link_type: "+lt)
				return
			}
			linkTypes = append(linkTypes, lt)
		}
	}

	res, err := s.GetNeighbors(
		r.Context(),
		s.resolveCellID(q.Get("cell_id")),
		supID,
		objType,
		objID,
		depth,
		linkTypes,
		viewerRoleFromRequest(r),
	)
	if err != nil {
		writeOntologyDomainError(w, err)
		return
	}
	writeOntologyJSON(w, http.StatusOK, res)
}

// HandleRebuild serves POST /v1/ontology/rebuild.
func (s *Service) HandleRebuild(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeOntologyError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeOntologyError(w, http.StatusServiceUnavailable, "ontology_unavailable", "ontology repository unavailable")
		return
	}

	var req RebuildRequest
	if r.Body != nil {
		raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			writeOntologyError(w, http.StatusBadRequest, "invalid_body", "failed to read request body")
			return
		}
		if len(strings.TrimSpace(string(raw))) > 0 {
			if err := json.Unmarshal(raw, &req); err != nil {
				writeOntologyError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
				return
			}
		}
	}

	q := r.URL.Query()
	requestedSup := firstNonEmpty(req.SupplierID, req.EntityID, q.Get("supplier_id"), q.Get("entity"), q.Get("tenant_id"))
	supID, ok := resolveOntologyTenantScope(w, r, requestedSup, true)
	if !ok {
		return
	}

	cellID := s.resolveCellID(firstNonEmpty(req.CellID, q.Get("cell_id")))
	auditTarget := firstNonEmpty(supID, "ALL")
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   supID,
		ObjectType: "OntologyRebuild",
		ObjectID:   auditTarget,
		ReasonCode: "ONTOLOGY_REBUILD",
	})

	res, err := s.RebuildFromSpanner(r.Context(), cellID, supID)
	if err != nil {
		writeOntologyDomainError(w, err)
		return
	}
	writeOntologyJSON(w, http.StatusOK, res)
}

func resolveOntologyTenantScope(w http.ResponseWriter, r *http.Request, requestedSup string, allowCrossTenantPlatformAdmin bool) (string, bool) {
	requested := strings.TrimSpace(requestedSup)
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
			writeOntologyError(w, http.StatusBadRequest, "supplier_required", "supplier_id is required")
			return "", false
		}
		return target, true
	}

	if callerTenant == "" {
		writeOntologyError(w, http.StatusUnauthorized, "tenant_required", "authenticated supplier tenant context is required")
		return "", false
	}
	if requested != "" && requested != callerTenant {
		writeOntologyError(w, http.StatusForbidden, "supplier_scope_mismatch", "requested supplier_id does not match authenticated supplier tenant")
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

func parseBoundedInt(raw string, def, minVal, maxVal int) (int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return def, nil
	}
	v, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, errors.New("must be an integer")
	}
	if v < minVal || v > maxVal {
		return def, nil
	}
	return v, nil
}

func writeOntologyDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrObjectNotFound):
		writeOntologyError(w, http.StatusNotFound, "ontology_object_not_found", err.Error())
	case errors.Is(err, ErrInvalidIdentity), errors.Is(err, ErrUnknownObjectType),
		errors.Is(err, ErrUnknownLinkType), errors.Is(err, ErrSchemaValidation), errors.Is(err, ErrFloatForbidden):
		writeOntologyError(w, http.StatusBadRequest, "invalid_ontology_request", err.Error())
	case errors.Is(err, ErrRepoUnavailable):
		writeOntologyError(w, http.StatusServiceUnavailable, "ontology_unavailable", err.Error())
	default:
		writeOntologyError(w, http.StatusInternalServerError, "ontology_internal_error", err.Error())
	}
}

func writeOntologyError(w http.ResponseWriter, status int, code, msg string) {
	writeOntologyJSON(w, status, map[string]string{
		"error":   code,
		"code":    code,
		"message": msg,
	})
}

func writeOntologyJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
