package driver

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

// SyncEscalationItem captures one local offline driver mutation that exhausted automatic retries.
type SyncEscalationItem struct {
	ID                 string `json:"id"`
	Endpoint           string `json:"endpoint"`
	OrderID            string `json:"order_id,omitempty"`
	PayloadJSON        string `json:"payload_json,omitempty"`
	LastError          string `json:"last_error,omitempty"`
	AttemptCount       int    `json:"attempt_count,omitempty"`
	ClientTimestampISO string `json:"client_timestamp_iso,omitempty"`
}

// SyncEscalationRequest is POST /v1/driver/sync-escalation input.
type SyncEscalationRequest struct {
	Items []SyncEscalationItem `json:"items"`
}

// SyncEscalationRecord persists one escalated offline driver item for ops support review.
type SyncEscalationRecord struct {
	EscalationID string               `json:"escalation_id"`
	TicketID     string               `json:"ticket_id"`
	DriverID     string               `json:"driver_id"`
	SupplierID   string               `json:"supplier_id,omitempty"`
	Item         SyncEscalationItem   `json:"item"`
	CreatedAt    string               `json:"created_at"`
}

// HandleSyncEscalation serves POST /v1/driver/sync-escalation.
// Persists immutable support-review tickets so dead local mutations are never silently dropped.
func (s *Service) HandleSyncEscalation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	driverID := driverIDFromRequest(r)
	if driverID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	body, err := readLimitedBody(r, 256*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	var req SyncEscalationRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if len(req.Items) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "items_required"})
		return
	}

	now := s.now().UTC()
	escalationID := uuid.NewString()
	supplierID := s.resolveSupplierScope(r.Context())

	records := make([]SyncEscalationRecord, 0, len(req.Items))
	muts := make([]*spanner.Mutation, 0, len(req.Items))
	for _, it := range req.Items {
		ticketID := uuid.NewString()
		orderID := strings.TrimSpace(it.OrderID)
		if orderID == "" {
			orderID = escalationID
		}
		rec := SyncEscalationRecord{
			EscalationID: escalationID,
			TicketID:     ticketID,
			DriverID:     driverID,
			SupplierID:   supplierID,
			Item:         it,
			CreatedAt:    now.Format(time.RFC3339Nano),
		}
		records = append(records, rec)

		if s.spanner != nil {
			payloadMap := map[string]any{
				"escalation_id":        escalationID,
				"driver_id":            driverID,
				"supplier_id":          supplierID,
				"item_id":              strings.TrimSpace(it.ID),
				"endpoint":             strings.TrimSpace(it.Endpoint),
				"order_id":             strings.TrimSpace(it.OrderID),
				"payload_json":         it.PayloadJSON,
				"last_error":           it.LastError,
				"attempt_count":        it.AttemptCount,
				"client_timestamp_iso": it.ClientTimestampISO,
				"handler_group":        "ROUTE_AND_CLOSED_DOOR",
			}
			rawPayload, mErr := json.Marshal(payloadMap)
			if mErr != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "marshal_escalation_failed"})
				return
			}
			title := "Driver offline mutation escalated: " + strings.TrimSpace(it.Endpoint)
			if len(title) > 250 {
				title = title[:250]
			}
			muts = append(muts, spanner.InsertMap("ExceptionTickets", map[string]any{
				"TicketId":     ticketID,
				"Type":         "DRIVER_SYNC_ESCALATION",
				"OrderId":      orderID,
				"EhfId":        spanner.NullString{StringVal: escalationID, Valid: true},
				"Severity":     "HIGH",
				"Status":       "OPEN",
				"Title":        title,
				"Description":  spanner.NullString{StringVal: strings.TrimSpace(it.LastError), Valid: strings.TrimSpace(it.LastError) != ""},
				"AssignedRole": spanner.NullString{StringVal: "SUPPORT", Valid: true},
				"CreatedAt":    now,
				"CreatedBy":    spanner.NullString{StringVal: driverID, Valid: true},
				"Payload":      spanner.NullJSON{Value: json.RawMessage(rawPayload), Valid: true},
			}))
		}
	}

	if s.spanner != nil {
		if _, err := s.spanner.ReadWriteTransaction(r.Context(), func(_ context.Context, txn *spanner.ReadWriteTransaction) error {
			return txn.BufferWrite(muts)
		}); err != nil {
			s.log.ErrorContext(r.Context(), "persist driver sync escalation failed", "err", err, "driver_id", driverID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "sync_escalation_persist_failed"})
			return
		}
	} else {
		s.mu.Lock()
		s.syncEscalations = append(s.syncEscalations, records...)
		s.mu.Unlock()
	}

	if s.supplierHub != nil && supplierID != "" {
		alertBytes, mErr := json.Marshal(map[string]any{
			"type":          "DRIVER_SYNC_ESCALATION",
			"escalation_id": escalationID,
			"driver_id":     driverID,
			"count":         len(records),
		})
		if mErr == nil {
			s.supplierHub.Broadcast(r.Context(), "supplier:"+supplierID, alertBytes)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":            true,
		"escalated":     len(records),
		"escalation_id": escalationID,
	})
}

// ListSyncEscalations returns in-memory escalated sync records for unit tests.
func (s *Service) ListSyncEscalations() []SyncEscalationRecord {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]SyncEscalationRecord, len(s.syncEscalations))
	copy(out, s.syncEscalations)
	return out
}

func (s *Service) queryRetryQueuedFiscalOrders(ctx context.Context, driverID string) ([]string, error) {
	if s == nil || s.spanner == nil || strings.TrimSpace(driverID) == "" {
		return nil, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT o.OrderId
		      FROM Orders o
		      WHERE o.DriverId = @did
		        AND (o.Status IN ('FISCALIZING', 'FISCAL_FAILED') OR o.FiscalStatus IN ('FAILED', 'FISCAL_RETRY_QUEUED', 'DEAD_LETTER'))
		        AND (EXISTS(SELECT 1 FROM OrderPaymentLegs l WHERE l.OrderId = o.OrderId AND l.Status = 'CAPTURED' AND l.Method != 'CREDIT' AND l.IdempotencyKey NOT LIKE 'credit-leave-%')
		             OR EXISTS(SELECT 1 FROM OrderDeliveryProofs p WHERE p.OrderId = o.OrderId AND p.ProofType = 'CASH_COLLECTION_GEOFENCE')
		             OR EXISTS(SELECT 1 FROM OrderFiscalReceipts r WHERE r.OrderId = o.OrderId AND r.Status = 'SUCCESS'))
		        AND EXISTS(SELECT 1 FROM OrderDeliveryProofs p
		                   WHERE p.OrderId = o.OrderId
		                     AND p.ProofType IN ('QR_HANDOFF', 'OFFLINE_CRYPTOGRAPHIC_POD')
		                     AND (COALESCE(p.QRTokenHash, '') != '' OR COALESCE(p.ScannedTokenHash, '') != ''))
		      ORDER BY o.UpdatedAt DESC
		      LIMIT 50`,
		Params: map[string]any{"did": strings.TrimSpace(driverID)},
	}
	iter := s.spanner.Single().Query(ctx, stmt)
	defer iter.Stop()
	var ids []string
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var oid string
		if err := row.Columns(&oid); err != nil {
			return nil, err
		}
		oid = strings.TrimSpace(oid)
		if oid != "" {
			ids = append(ids, oid)
		}
	}
	return ids, nil
}
