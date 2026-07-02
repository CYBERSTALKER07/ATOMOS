package supplier

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

type replenishmentTraceRow struct {
	InsightID     string `json:"insight_id"`
	WarehouseID   string `json:"warehouse_id"`
	WarehouseName string `json:"warehouse_name,omitempty"`
	ProductID     string `json:"product_id"`
	ProductName   string `json:"product_name,omitempty"`
	Status        string `json:"status"`
	ReasonCode    string `json:"reason_code,omitempty"`
	TransferID    string `json:"transfer_id,omitempty"`
	TransferState string `json:"transfer_state,omitempty"`
	CreatedAt     string `json:"created_at"`
	LinkedAt      string `json:"linked_at,omitempty"`
}

type replenishmentTraceabilityResponse struct {
	Rows        []replenishmentTraceRow `json:"rows"`
	GeneratedAt string                  `json:"generated_at"`
}

// HandleReplenishmentTraceability serves GET /v1/supplier/replenishment/traceability.
func (s *Service) HandleReplenishmentTraceability(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if s.portalSpanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "traceability_unavailable"})
		return
	}
	sid := s.scopedSupplierID(r)
	rows, err := s.listReplenishmentTraceability(r.Context(), sid, 50)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "traceability_load_failed"})
		return
	}
	writeJSON(w, http.StatusOK, replenishmentTraceabilityResponse{
		Rows:        rows,
		GeneratedAt: s.now().UTC().Format(time.RFC3339Nano),
	})
}

func (s *Service) listReplenishmentTraceability(ctx context.Context, supplierID string, limit int) ([]replenishmentTraceRow, error) {
	if limit < 1 {
		limit = 50
	}
	stmt := spanner.Statement{
		SQL: `SELECT ri.InsightId, ri.WarehouseId, COALESCE(w.Name, ri.WarehouseId),
		             ri.ProductId, COALESCE(p.Name, ri.ProductId),
		             ri.Status, ri.ReasonCode, ri.CreatedAt,
		             ft.TransferId, ft.State, ft.CreatedAt
		      FROM ReplenishmentInsights ri
		      LEFT JOIN FactoryInternalTransfers ft ON ft.SourceInsightId = ri.InsightId
		      LEFT JOIN Warehouses w ON ri.WarehouseId = w.WarehouseId
		      LEFT JOIN Products p ON ri.ProductId = p.ProductId
		      WHERE ri.SupplierId = @sid
		      ORDER BY ri.CreatedAt DESC
		      LIMIT @lim`,
		Params: map[string]any{
			"sid": supplierID,
			"lim": int64(limit),
		},
	}
	iter := s.portalSpanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	out := make([]replenishmentTraceRow, 0)
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query replenishment traceability: %w", err)
		}
		var item replenishmentTraceRow
		var createdAt time.Time
		var transferID spanner.NullString
		var transferState spanner.NullString
		var linkedAt spanner.NullTime
		if err := row.Columns(
			&item.InsightID, &item.WarehouseID, &item.WarehouseName,
			&item.ProductID, &item.ProductName,
			&item.Status, &item.ReasonCode, &createdAt,
			&transferID, &transferState, &linkedAt,
		); err != nil {
			return nil, fmt.Errorf("scan replenishment traceability: %w", err)
		}
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
		if transferID.Valid {
			item.TransferID = transferID.StringVal
		}
		if transferState.Valid {
			item.TransferState = transferState.StringVal
		}
		if linkedAt.Valid {
			item.LinkedAt = linkedAt.Time.UTC().Format(time.RFC3339Nano)
		}
		out = append(out, item)
	}
	return out, nil
}
