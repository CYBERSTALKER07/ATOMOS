package warehouse

import (
	"context"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"google.golang.org/api/iterator"
)

// OpsExceptionRow is one triage item on GET /v1/warehouse/ops/exceptions.
type OpsExceptionRow struct {
	ExceptionID   string                     `json:"exception_id,omitempty"`
	Kind          string                     `json:"kind"`
	OrderID       string                     `json:"order_id,omitempty"`
	ManifestID    string                     `json:"manifest_id,omitempty"`
	Reason        string                     `json:"reason,omitempty"`
	Status        string                     `json:"status,omitempty"`
	UpdatedAt     string                     `json:"updated_at,omitempty"`
	DeliveryExpectation *order.DeliveryExpectation `json:"delivery_expectation,omitempty"`
}

// HandleOpsExceptions serves GET /v1/warehouse/ops/exceptions.
func (s *Service) HandleOpsExceptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	whID := warehouseIDFromRequest(r)
	rows, err := s.listOpsExceptions(r.Context(), whID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "exceptions_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"exceptions": rows})
}

func (s *Service) listOpsExceptions(ctx context.Context, warehouseID string) ([]OpsExceptionRow, error) {
	if s.spannerClient == nil || warehouseID == "" {
		return []OpsExceptionRow{}, nil
	}
	now := s.now().UTC()
	out := make([]OpsExceptionRow, 0, 32)
	manifestRows, err := s.listManifestExceptions(ctx, warehouseID)
	if err != nil {
		return nil, err
	}
	out = append(out, manifestRows...)
	delayedRows, err := s.listDelayedOrderExceptions(ctx, warehouseID, now)
	if err != nil {
		return nil, err
	}
	out = append(out, delayedRows...)
	lockRows := s.listDispatchLockExceptions(ctx, warehouseID)
	out = append(out, lockRows...)
	return out, nil
}

func (s *Service) listManifestExceptions(ctx context.Context, warehouseID string) ([]OpsExceptionRow, error) {
	sid := strings.TrimSpace(s.supplierID)
	stmt := spanner.Statement{
		SQL: `SELECT e.ExceptionId, e.OrderId, e.ManifestId, e.Reason, e.CreatedAt
		      FROM ManifestExceptions e
		      JOIN Orders o ON o.OrderId = e.OrderId
		      WHERE o.WarehouseId = @wh
		        AND e.ResolvedAt IS NULL
		      ORDER BY e.CreatedAt DESC
		      LIMIT 100`,
		Params: map[string]any{"wh": warehouseID, "sid": sid},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	out := make([]OpsExceptionRow, 0, 16)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		var exceptionID, orderID, manifestID, reason string
		var createdAt time.Time
		if err := row.Columns(&exceptionID, &orderID, &manifestID, &reason, &createdAt); err != nil {
			continue
		}
		out = append(out, OpsExceptionRow{
			ExceptionID: exceptionID,
			Kind:        "manifest_exception",
			OrderID:     orderID,
			ManifestID:  manifestID,
			Reason:      reason,
			UpdatedAt:   createdAt.UTC().Format(time.RFC3339Nano),
		})
	}
}

func (s *Service) listDelayedOrderExceptions(ctx context.Context, warehouseID string, now time.Time) ([]OpsExceptionRow, error) {
	stmt := spanner.Statement{
		SQL: `SELECT OrderId, Status, Source, ConfirmationStatus, DeliveryPriority,
		             DeliverBefore, RequestedDeliveryDate, ProposedDeliveryDate,
		             ReceivingWindowOpen, ReceivingWindowClose, UpdatedAt
		      FROM Orders
		      WHERE WarehouseId = @wh
		        AND Status = 'DELAYED'
		      ORDER BY UpdatedAt DESC
		      LIMIT 50`,
		Params: map[string]any{"wh": warehouseID},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	out := make([]OpsExceptionRow, 0, 8)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		var (
			orderID, status, source, confirmation, priority string
			deliverBefore, requested, proposed              spanner.NullTime
			windowOpen, windowClose                         spanner.NullString
			updatedAt                                       time.Time
		)
		if err := row.Columns(&orderID, &status, &source, &confirmation, &priority,
			&deliverBefore, &requested, &proposed, &windowOpen, &windowClose, &updatedAt); err != nil {
			continue
		}
		o := order.Order{
			OrderID:              orderID,
			Status:               order.Status(status),
			Source:               order.OrderSource(source),
			ConfirmationStatus:   order.ConfirmationStatus(confirmation),
			DeliveryPriority:     order.DeliveryPriority(priority),
			ReceivingWindowOpen:  windowOpen.StringVal,
			ReceivingWindowClose: windowClose.StringVal,
		}
		if deliverBefore.Valid {
			t := deliverBefore.Time
			o.DeliverBefore = &t
		}
		if requested.Valid {
			t := requested.Time
			o.RequestedDeliveryDate = &t
		}
		if proposed.Valid {
			t := proposed.Time
			o.ProposedDeliveryDate = &t
		}
		exp := order.ComputeDeliveryExpectation(now, o)
		out = append(out, OpsExceptionRow{
			Kind:                "delayed_order",
			OrderID:             orderID,
			Status:              status,
			Reason:              exp.DelayReason,
			UpdatedAt:           updatedAt.UTC().Format(time.RFC3339Nano),
			DeliveryExpectation: &exp,
		})
	}
}

func (s *Service) listDispatchLockExceptions(ctx context.Context, warehouseID string) []OpsExceptionRow {
	locks := s.loadDispatchLocks(ctx, warehouseID)
	out := make([]OpsExceptionRow, 0, len(locks))
	for _, lock := range locks {
		out = append(out, OpsExceptionRow{
			Kind:      "dispatch_lock",
			OrderID:   lock.EntityID,
			Reason:    lock.Reason,
			Status:    lock.EntityType,
			UpdatedAt: lock.CreatedAt,
		})
	}
	return out
}
