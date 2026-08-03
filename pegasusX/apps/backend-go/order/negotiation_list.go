package order

import (
	"encoding/json"
	"net/http"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/pkg/httppagination"
	"google.golang.org/api/iterator"
)

// NegotiationProposalDTO is the supplier-portal projection for a pending proposal.
type NegotiationProposalDTO struct {
	ProposalID string                    `json:"proposal_id"`
	OrderID    string                    `json:"order_id"`
	DriverID   string                    `json:"driver_id"`
	Items      []ProposedNegotiationItem `json:"items"`
	CreatedAt  time.Time                 `json:"created_at"`
	ExpiresAt  time.Time                 `json:"expires_at"`
}

// HandleListPendingNegotiations serves GET /v1/supplier/negotiations/pending.
func (s *Service) HandleListPendingNegotiations(w http.ResponseWriter, r *http.Request) {
	if quantityNegotiationDisabled() {
		writeJSON(w, http.StatusOK, map[string]any{"data": []NegotiationProposalDTO{}})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleAdmin {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if s.spannerClient == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "negotiation_unavailable"})
		return
	}

	supplierID, ok := auth.ResolveSupplierID(r.Context())
	if !ok {
		supplierID = s.supplierID
	}
	if supplierID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "supplier_scope_required"})
		return
	}

	ctx := r.Context()
	limit, offset := httppagination.ParseLimitOffset(r, 100, 500)
	stmt := spanner.Statement{
		SQL: `SELECT n.ProposalId, n.OrderId, n.DriverId, n.ProposedItems, n.CreatedAt, n.ExpiresAt
		      FROM NegotiationProposals n
		      JOIN Orders o ON n.OrderId = o.OrderId
		      WHERE o.SupplierId = @supplierId
		        AND n.Status = 'PENDING'
		      ORDER BY n.CreatedAt DESC
		      LIMIT @limit OFFSET @offset`,
		Params: map[string]any{
			"supplierId": supplierID,
			"limit":      int64(limit),
			"offset":     int64(offset),
		},
	}

	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()

	pending := make([]NegotiationProposalDTO, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			s.log.ErrorContext(ctx, "list pending negotiations failed", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_negotiations_failed"})
			return
		}

		var dto NegotiationProposalDTO
		var itemsJSON string
		if err := row.Columns(&dto.ProposalID, &dto.OrderID, &dto.DriverID, &itemsJSON, &dto.CreatedAt, &dto.ExpiresAt); err != nil {
			s.log.ErrorContext(ctx, "parse negotiation row failed", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "parse_negotiation_failed"})
			return
		}
		if itemsJSON != "" {
			if err := json.Unmarshal([]byte(itemsJSON), &dto.Items); err != nil {
				s.log.ErrorContext(ctx, "parse proposed items failed", "err", err, "proposal_id", dto.ProposalID)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "parse_negotiation_items_failed"})
				return
			}
		}
		pending = append(pending, dto)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data":     pending,
		"limit":    limit,
		"offset":   offset,
		"count":    len(pending),
		"has_more": len(pending) == limit,
	})
}
