package entityresolution

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"backend-go/auth"
	"backend-go/pkg/httputil"
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

// HandleResolve serves POST /v1/supplier/entity-resolution/resolve.
func HandleResolve(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httputil.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed")
			return
		}

		supplierID, ok := resolveSupplierScope(r)
		if !ok {
			httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		var req resolveRequestBody
		if err := decodeJSONBody(r, &req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "invalid_json_body")
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
			handleServiceError(w, r, "resolve", supplierID, req.EntityType, req.EntityID, err)
			return
		}

		httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"status": "ok",
			"result": result,
		})
	}
}

// HandleExplain serves POST /v1/supplier/entity-resolution/explain.
func HandleExplain(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httputil.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed")
			return
		}

		supplierID, ok := resolveSupplierScope(r)
		if !ok {
			httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		var req explainRequestBody
		if err := decodeJSONBody(r, &req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "invalid_json_body")
			return
		}

		result, err := svc.Explain(r.Context(), ExplainInput{
			SupplierID: supplierID,
			EntityType: req.EntityType,
			EntityID:   req.EntityID,
		})
		if err != nil {
			handleServiceError(w, r, "explain", supplierID, req.EntityType, req.EntityID, err)
			return
		}

		httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"status": "ok",
			"result": result,
		})
	}
}

func resolveSupplierScope(r *http.Request) (string, bool) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims == nil {
		return "", false
	}
	supplierID := strings.TrimSpace(claims.ResolveSupplierID())
	if supplierID == "" {
		return "", false
	}
	return supplierID, true
}

func decodeJSONBody(r *http.Request, out interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}
	if decoder.More() {
		return errors.New("multiple JSON objects are not supported")
	}
	return nil
}

func handleServiceError(w http.ResponseWriter, r *http.Request, operation, supplierID, entityType, entityID string, err error) {
	status := http.StatusInternalServerError
	errorCode := "entity_resolution_failed"
	switch {
	case errors.Is(err, ErrInvalidInput):
		status = http.StatusBadRequest
		errorCode = "invalid_request"
	case errors.Is(err, ErrNotFound):
		status = http.StatusNotFound
		errorCode = "entity_not_found"
	}

	slog.ErrorContext(r.Context(), "entity resolution request failed",
		"operation", operation,
		"supplier_id", supplierID,
		"entity_type", strings.ToUpper(strings.TrimSpace(entityType)),
		"entity_id", strings.TrimSpace(entityID),
		"status", status,
		"error", err,
	)
	httputil.WriteError(w, status, errorCode)
}
