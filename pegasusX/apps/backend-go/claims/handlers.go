package claims

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

type Service struct {
	Repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{Repo: repo}
}

type CreateClaimRequest struct {
	Reason               string `json:"reason"`
	RequestedAmountMinor int64  `json:"requested_amount_minor"`
	Notes                string `json:"notes"`
}

func (s *Service) HandleCreateClaim(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	
	var req CreateClaimRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	claimsCtx, ok := auth.FromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	claim := &Claim{
		ClaimId:              uuid.New().String(),
		OrderId:              orderID,
		RetailerId:           claimsCtx.Subject,
		SupplierId:           claimsCtx.SupplierID,
		Status:               ClaimStatusPending,
		Reason:               req.Reason,
		RequestedAmountMinor: req.RequestedAmountMinor,
		Notes:                &req.Notes, // Passing address because it's a pointer now
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	if err := s.Repo.SaveClaim(r.Context(), claim, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(claim)
}
