package entityresolution

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/web"
)

type resolveRequestBody struct {
	EntityType    string `json:"entity_type"`
	EntityID      string `json:"entity_id"`
	Query         string `json:"query"`
	MaxCandidates int    `json:"max_candidates"`
}

type explainRequestBody struct {
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
}

func scopeSupplierID(r *http.Request) string {
	sid := strings.TrimSpace(auth.PreferTenantSupplierID(r.Context(), ""))
	if sid != "" {
		return sid
	}
	if id, ok := auth.ResolveSupplierID(r.Context()); ok {
		return strings.TrimSpace(id)
	}
	return ""
}

func HandleResolve(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			web.JSONError(w, "method_not_allowed", http.StatusMethodNotAllowed)
			return
		}
		supplierID := scopeSupplierID(r)
		if supplierID == "" {
			web.JSONError(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var req resolveRequestBody
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			web.JSONError(w, "invalid_json_body", http.StatusBadRequest)
			return
		}
		result, err := svc.Resolve(r.Context(), ResolveInput{
			SupplierID:    supplierID,
			EntityType:    req.EntityType,
			EntityID:      req.EntityID,
			Query:         req.Query,
			MaxCandidates: req.MaxCandidates,
		})
		if err != nil {
			writeServiceError(w, err)
			return
		}
		web.JSONResponse(w, http.StatusOK, result)
	}
}

func HandleExplain(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			web.JSONError(w, "method_not_allowed", http.StatusMethodNotAllowed)
			return
		}
		supplierID := scopeSupplierID(r)
		if supplierID == "" {
			web.JSONError(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var req explainRequestBody
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			web.JSONError(w, "invalid_json_body", http.StatusBadRequest)
			return
		}
		result, err := svc.Explain(r.Context(), ExplainInput{
			SupplierID: supplierID,
			EntityType: req.EntityType,
			EntityID:   req.EntityID,
		})
		if err != nil {
			writeServiceError(w, err)
			return
		}
		web.JSONResponse(w, http.StatusOK, result)
	}
}

func writeServiceError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrInvalidInput) {
		web.JSONError(w, "invalid_input", http.StatusBadRequest)
		return
	}
	if errors.Is(err, ErrNotFound) {
		web.JSONError(w, "entity_not_found", http.StatusNotFound)
		return
	}
	web.JSONError(w, "query_failed", http.StatusInternalServerError)
}
