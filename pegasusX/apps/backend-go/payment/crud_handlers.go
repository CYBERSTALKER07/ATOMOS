package payment

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/web"
)

// HandleCreatePayer serves POST /v1/payers
// Role-gated: only retailers (self) and admins may create payer profiles;
// an explicit payer_id must belong to the caller (fail-closed ownership).
func (s *Service) HandleCreatePayer(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Subject == "" {
		web.JSONError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.Role != auth.RoleRetailer && claims.Role != auth.RoleAdmin {
		web.JSONError(w, "forbidden: only retailers or admins can create payers", http.StatusForbidden)
		return
	}
	var req Payer
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Email == "" {
		web.JSONError(w, "missing required fields: name, email", http.StatusBadRequest)
		return
	}

	if req.PayerID == "" {
		req.PayerID = claims.Subject
	} else if claims.Role != auth.RoleAdmin && req.PayerID != claims.Subject {
		web.JSONError(w, "forbidden: cannot create another payer profile", http.StatusForbidden)
		return
	}

	if err := s.repo.CreatePayer(r.Context(), req); err != nil {
		web.JSONError(w, "failed to create payer: "+err.Error(), http.StatusInternalServerError)
		return
	}

	web.JSONResponse(w, http.StatusCreated, req)
}

// HandleGetPayer serves GET /v1/payers/{payerId}
func (s *Service) HandleGetPayer(w http.ResponseWriter, r *http.Request) {
	payerID := chi.URLParam(r, "payerId")
	if payerID == "" {
		web.JSONError(w, "missing payerId", http.StatusBadRequest)
		return
	}

	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Subject == "" {
		web.JSONError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.Role != auth.RoleAdmin && claims.Subject != payerID {
		web.JSONError(w, "forbidden: cannot access another payer profile", http.StatusForbidden)
		return
	}

	p, err := s.repo.GetPayer(r.Context(), payerID)
	if err != nil {
		web.JSONError(w, "failed to get payer: "+err.Error(), http.StatusInternalServerError)
		return
	}

	web.JSONResponse(w, http.StatusOK, p)
}

// HandleUpdatePayer serves PUT /v1/payers/{payerId}
func (s *Service) HandleUpdatePayer(w http.ResponseWriter, r *http.Request) {
	payerID := chi.URLParam(r, "payerId")
	if payerID == "" {
		web.JSONError(w, "missing payerId", http.StatusBadRequest)
		return
	}

	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Subject == "" {
		web.JSONError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.Role != auth.RoleAdmin && claims.Subject != payerID {
		web.JSONError(w, "forbidden: cannot update another payer profile", http.StatusForbidden)
		return
	}

	var req Payer
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.PayerID = payerID

	if err := s.repo.UpdatePayer(r.Context(), req); err != nil {
		web.JSONError(w, "failed to update payer: "+err.Error(), http.StatusInternalServerError)
		return
	}

	web.JSONResponse(w, http.StatusOK, req)
}

// HandleListPayers serves GET /v1/payers
func (s *Service) HandleListPayers(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Subject == "" {
		web.JSONError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Retailers (and non-platform roles) may only see their own payer profile.
	// Platform admin may list globally; supplier ADMIN is not a global payers desk.
	if claims.Role != auth.RolePlatformAdmin {
		if claims.Role != auth.RoleRetailer && claims.Role != auth.RoleAdmin {
			web.JSONError(w, "forbidden", http.StatusForbidden)
			return
		}
		p, err := s.repo.GetPayer(r.Context(), claims.Subject)
		if err != nil {
			web.JSONResponse(w, http.StatusOK, []Payer{})
			return
		}
		web.JSONResponse(w, http.StatusOK, []Payer{p})
		return
	}

	payers, err := s.repo.ListPayers(r.Context(), limit, offset)
	if err != nil {
		web.JSONError(w, "failed to list payers: "+err.Error(), http.StatusInternalServerError)
		return
	}

	web.JSONResponse(w, http.StatusOK, payers)
}
