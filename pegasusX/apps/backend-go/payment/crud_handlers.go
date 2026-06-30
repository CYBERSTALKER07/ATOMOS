package payment

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/web"
)

// HandleCreatePayer serves POST /v1/payers
func (s *Service) HandleCreatePayer(w http.ResponseWriter, r *http.Request) {
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
		req.PayerID = uuid.New().String()
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

	payers, err := s.repo.ListPayers(r.Context(), limit, offset)
	if err != nil {
		web.JSONError(w, "failed to list payers: "+err.Error(), http.StatusInternalServerError)
		return
	}

	web.JSONResponse(w, http.StatusOK, payers)
}
