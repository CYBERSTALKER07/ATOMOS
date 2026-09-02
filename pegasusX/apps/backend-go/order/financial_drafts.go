package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"google.golang.org/api/iterator"
)

// --- Request / Response types ---

type CreateDraftRequest struct {
	OrderID        string `json:"orderId"`
	ProposedAmount int64  `json:"proposedAmount"`
	ReasonCode     string `json:"reasonCode"`
	Notes          string `json:"notes"`
}

type ReviewDraftRequest struct {
	DraftID string `json:"draftId"`
	Action  string `json:"action"` // APPROVE, REJECT
}

type DraftResponse struct {
	DraftID string `json:"draftId,omitempty"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// --- Domain Logic ---

func (s *Service) CreateFinancialDraft(ctx context.Context, claims auth.Claims, req CreateDraftRequest) (DraftResponse, error) {
	orderID := strings.TrimSpace(req.OrderID)
	if orderID == "" {
		return DraftResponse{}, errors.New("order_id required")
	}
	if strings.TrimSpace(req.ReasonCode) == "" {
		return DraftResponse{}, errors.New("reason_code required")
	}

	current, ok, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return DraftResponse{}, err
	}
	if !ok {
		return DraftResponse{}, ErrOrderNotFound
	}

	// Supplier scoping: caller must belong to the order's supplier.
	if claims.SupplierID != "" && claims.SupplierID != current.SupplierID {
		return DraftResponse{}, ErrOrderForbidden
	}

	// Mutability Boundary: pre-dispatch orders should be edited directly.
	if current.Status == StatusPending || current.Status == StatusLoaded || current.Status == StatusScheduled {
		return DraftResponse{}, errors.New("order has not crossed the mutability boundary; edit it directly")
	}

	draftID := uuid.NewString()

	mut := spanner.Insert("DraftAdjustments",
		[]string{"DraftId", "SupplierId", "OrderId", "ProposedAmount", "ReasonCode", "OperatorId", "Status", "CreatedAt"},
		[]any{draftID, current.SupplierID, current.OrderID, req.ProposedAmount, strings.TrimSpace(req.ReasonCode), claims.Subject, "PENDING_APPROVAL", spanner.CommitTimestamp},
	)

	_, err = s.spannerClient.Apply(ctx, []*spanner.Mutation{mut})
	if err != nil {
		return DraftResponse{}, fmt.Errorf("create draft: %w", err)
	}
	return DraftResponse{DraftID: draftID, Status: "PENDING_APPROVAL"}, nil
}

// ReviewFinancialDraft approves or rejects a draft inside a guarded RW
// transaction. It reads the current status first (TOCTOU-safe) and verifies
// the caller's supplier scope before mutating.
func (s *Service) ReviewFinancialDraft(ctx context.Context, claims auth.Claims, req ReviewDraftRequest) (DraftResponse, error) {
	if claims.Role != auth.RoleAdmin {
		return DraftResponse{}, errors.New("unauthorized: finance/admin only")
	}

	draftID := strings.TrimSpace(req.DraftID)
	if draftID == "" {
		return DraftResponse{}, errors.New("draft_id required")
	}

	action := strings.ToUpper(strings.TrimSpace(req.Action))
	if action != "APPROVE" && action != "REJECT" {
		return DraftResponse{}, errors.New("action must be APPROVE or REJECT")
	}

	nextStatus := "APPROVED"
	if action == "REJECT" {
		nextStatus = "REJECTED"
	}

	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// 1. Read current draft inside the transaction (TOCTOU guard).
		row, readErr := txn.ReadRow(ctx, "DraftAdjustments",
			spanner.Key{draftID},
			[]string{"DraftId", "SupplierId", "Status"},
		)
		if readErr != nil {
			if readErr == iterator.Done {
				return errors.New("draft_not_found")
			}
			return fmt.Errorf("read draft: %w", readErr)
		}

		var existingDraftID, existingSupplierID, existingStatus string
		if err := row.Columns(&existingDraftID, &existingSupplierID, &existingStatus); err != nil {
			return fmt.Errorf("scan draft: %w", err)
		}

		// 2. Supplier scoping: admin can only review drafts for their supplier.
		if claims.SupplierID != "" && claims.SupplierID != existingSupplierID {
			return ErrOrderForbidden
		}

		// 3. Status guard: only PENDING_APPROVAL drafts can be reviewed.
		if existingStatus != "PENDING_APPROVAL" {
			return fmt.Errorf("draft already %s; cannot review", existingStatus)
		}

		// 4. Apply the review.
		mut := spanner.Update("DraftAdjustments",
			[]string{"DraftId", "Status", "ReviewedAt", "ReviewedBy"},
			[]any{draftID, nextStatus, spanner.CommitTimestamp, claims.Subject},
		)
		return txn.BufferWrite([]*spanner.Mutation{mut})
	})
	if err != nil {
		return DraftResponse{}, err
	}
	return DraftResponse{DraftID: draftID, Status: nextStatus}, nil
}

// --- HTTP Handlers ---

func (s *Service) HandleCreateFinancialDraft(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req CreateDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	resp, err := s.CreateFinancialDraft(r.Context(), claims, req)
	if err != nil {
		s.writeOrderMutationError(w, "create draft failed", req.OrderID, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (s *Service) HandleReviewFinancialDraft(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req ReviewDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	resp, err := s.ReviewFinancialDraft(r.Context(), claims, req)
	if err != nil {
		s.writeOrderMutationError(w, "review draft failed", req.DraftID, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
