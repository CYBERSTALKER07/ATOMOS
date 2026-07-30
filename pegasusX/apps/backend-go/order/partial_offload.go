package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// OffloadStatus on a line after partial delivery.
const (
	OffloadStatusFull     = "FULL"
	OffloadStatusPartial  = "PARTIAL"
	OffloadStatusNone     = "NONE"
	OffloadStatusReturned = "RETURNED"
)

// OffloadReason codes for line-level partials.
const (
	OffloadReasonDamaged     = "DAMAGED"
	OffloadReasonMissing     = "MISSING"
	OffloadReasonShopRefused = "SHOP_REFUSED"
	OffloadReasonCapacity    = "CAPACITY"
	OffloadReasonOther       = "OTHER"
)

var (
	ErrPartialQtyMismatch   = errors.New("partial_qty_mismatch")
	ErrPartialInvalidStatus = errors.New("partial_invalid_status")
	ErrPartialEmptyLines    = errors.New("partial_empty_lines")
	ErrPartialUnknownSKU    = errors.New("partial_unknown_sku")
)

// PartialOffloadLine is one line in POST /v1/delivery/partial-offload.
type PartialOffloadLine struct {
	OrderLineID  string `json:"orderLineId"`
	DeliveredQty int64  `json:"deliveredQty"`
	RemainingQty int64  `json:"remainingQty"`
	Reason       string `json:"reason,omitempty"` // DAMAGED|MISSING|SHOP_REFUSED|CAPACITY|OTHER
}

// PartialOffloadRequest is the driver wire shape (offline-capable + idempotent).
type PartialOffloadRequest struct {
	Lines    []PartialOffloadLine `json:"lines"`
	Location DriverTelemetry      `json:"location"`
}

// PartialOffloadResponse is returned after successful apply.
type PartialOffloadResponse struct {
	OrderID         string `json:"order_id"`
	PartialDelivery bool   `json:"partial_delivery"`
	DeliveredMinor  int64  `json:"delivered_minor"`
	RemainingMinor  int64  `json:"remaining_minor"`
	Currency        string `json:"currency"`
	Status          string `json:"status"`
	Message         string `json:"message"`
}

// ApplyPartialOffloadLines validates qty math and returns updated lines + totals.
// Pure: DeliveredQty + RemainingQty must equal original Quantity for each touched line.
// Untouched lines keep original qty as fully delivered when fullOffloadDefault is true;
// when false, untouched lines are left unchanged (no offload fields).
func ApplyPartialOffloadLines(current []LineItem, updates []PartialOffloadLine, fullOffloadDefault bool) ([]LineItem, int64, int64, error) {
	if len(updates) == 0 {
		return nil, 0, 0, ErrPartialEmptyLines
	}
	byLineID := make(map[string]PartialOffloadLine, len(updates))
	for _, u := range updates {
		id := strings.TrimSpace(u.OrderLineID)
		if id == "" {
			return nil, 0, 0, fmt.Errorf("%w: empty orderLineId", ErrPartialUnknownSKU)
		}
		if u.DeliveredQty < 0 || u.RemainingQty < 0 {
			return nil, 0, 0, fmt.Errorf("%w: negative qty on %s", ErrPartialQtyMismatch, id)
		}
		if reason := strings.ToUpper(strings.TrimSpace(u.Reason)); reason != "" {
			switch reason {
			case OffloadReasonDamaged, OffloadReasonMissing, OffloadReasonShopRefused, OffloadReasonCapacity, OffloadReasonOther:
			default:
				return nil, 0, 0, fmt.Errorf("invalid offload reason %q on %s", u.Reason, id)
			}
			u.Reason = reason
		}
		byLineID[id] = u
	}

	out := make([]LineItem, 0, len(current))
	var deliveredMinor, remainingMinor int64
	partialAny := false

	for _, line := range current {
		u, touched := byLineID[line.SKU]
		if !touched {
			if fullOffloadDefault {
				// Treat as fully delivered.
				line.DeliveredQty = line.Quantity
				line.RemainingQty = 0
				line.OffloadStatus = OffloadStatusFull
				deliveredMinor += line.UnitPrice * line.Quantity
			} else {
				// Preserve prior partial state if any.
				if line.DeliveredQty > 0 || line.RemainingQty > 0 {
					deliveredMinor += line.UnitPrice * line.DeliveredQty
					remainingMinor += line.UnitPrice * line.RemainingQty
					if line.OffloadStatus == OffloadStatusPartial || line.RemainingQty > 0 {
						partialAny = true
					}
				} else {
					deliveredMinor += line.UnitPrice * line.Quantity
				}
			}
			out = append(out, line)
			continue
		}
		delete(byLineID, line.SKU)
		if u.DeliveredQty+u.RemainingQty != line.Quantity {
			return nil, 0, 0, fmt.Errorf("%w: sku %s delivered(%d)+remaining(%d) != qty(%d)",
				ErrPartialQtyMismatch, line.SKU, u.DeliveredQty, u.RemainingQty, line.Quantity)
		}
		line.DeliveredQty = u.DeliveredQty
		line.RemainingQty = u.RemainingQty
		line.OffloadReason = u.Reason
		switch {
		case u.RemainingQty == 0 && u.DeliveredQty == line.Quantity:
			line.OffloadStatus = OffloadStatusFull
		case u.DeliveredQty == 0:
			line.OffloadStatus = OffloadStatusNone
			partialAny = true
		default:
			line.OffloadStatus = OffloadStatusPartial
			partialAny = true
		}
		deliveredMinor += line.UnitPrice * line.DeliveredQty
		remainingMinor += line.UnitPrice * line.RemainingQty
		out = append(out, line)
	}

	if len(byLineID) > 0 {
		for id := range byLineID {
			return nil, 0, 0, fmt.Errorf("%w: %s", ErrPartialUnknownSKU, id)
		}
	}
	_ = partialAny
	return out, deliveredMinor, remainingMinor, nil
}

// HandlePartialOffload is POST /v1/delivery/partial-offload.
// Records line-level delivered/remaining; fiscal/payment later use delivered portion only.
func (s *Service) HandlePartialOffload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleDriver {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	body, err := readLimitedBody(r, 256*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotency(w, r, body) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	var req PartialOffloadRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	orderID := chi.URLParam(r, "orderId")
	if orderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id_required"})
		return
	}
	if err := req.Location.Validate(100.0); err != nil {
		writeJSON(w, http.StatusPreconditionFailed, map[string]string{"error": err.Error()})
		return
	}

	ctx := r.Context()
	now := s.now()

	current, found, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "order_load_failed"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
		return
	}
	if strings.TrimSpace(current.DriverID) != strings.TrimSpace(claims.Subject) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	switch current.Status {
	case StatusArrived, StatusShopClosedPending, StatusAwaitingPayment, StatusPendingCashCollection:
	default:
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": fmt.Sprintf("%v: status %s", ErrPartialInvalidStatus, current.Status),
		})
		return
	}
	// Fiscal hard-gate: no partial after money path enters fiscal.
	if strings.HasPrefix(strings.ToUpper(current.FiscalStatus), "FISCAL") ||
		current.Status == StatusFiscalizing || current.Status == StatusFiscalFailed {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "partial_blocked_fiscal"})
		return
	}

	updated, deliveredMinor, remainingMinor, err := ApplyPartialOffloadLines(current.LineItems, req.Lines, true)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	current.LineItems = updated
	current.PartialDelivery = remainingMinor > 0 || anyPartialLine(updated)
	// Money path uses delivered portion; keep TotalMinor as delivered for settlement.
	// OriginalTotalMinor preserves the pre-partial economic footprint when still zero.
	if current.OriginalTotalMinor <= 0 {
		current.OriginalTotalMinor = current.TotalMinor
	}
	current.TotalMinor = deliveredMinor
	current.UpdatedAt = now.UTC()

	// Remaining qty → reverse logistics (SupplierReturns quarantine), same path as amend.
	pendingReturns := make([]SupplierReturn, 0)
	for _, line := range updated {
		if line.RemainingQty <= 0 {
			continue
		}
		reason := strings.TrimSpace(line.OffloadReason)
		if reason == "" {
			reason = OffloadReasonOther
		}
		// Map offload reasons onto amend/return vocabulary where needed.
		switch reason {
		case OffloadReasonShopRefused, OffloadReasonCapacity:
			reason = "OTHER"
		}
		pendingReturns = append(pendingReturns, SupplierReturn{
			ReturnID:    s.newID(),
			SKU:         line.SKU,
			RejectedQty: line.RemainingQty,
			Reason:      reason,
			DriverNotes: "partial_offload:" + line.OffloadReason,
			ManifestID:  strings.TrimSpace(current.ManifestID),
			DriverID:    strings.TrimSpace(claims.Subject),
			WarehouseID: strings.TrimSpace(current.WarehouseID),
		})
	}
	current.PendingSupplierReturns = pendingReturns

	err = s.repo.UpdateOrderWithTxn(ctx, current, nil, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := s.ensureProximityUnlocked(ctx, txn, &current, req.Location.ToLocation(), TransitionOpts{
			Actor:  claims.Subject,
			Reason: "partial_offload",
		}); err != nil {
			return err
		}
		return nil
	}, func(txn outbox.TxnBuffer) error {
		if err := outbox.EmitJSON(ctx, txn, events.AggregateOrder, current.OrderID, events.TopicMain, events.OrderEvent{
			BaseEvent:  events.BaseEvent{Type: events.EventPartialOffload, Timestamp: now.UTC().Format(time.RFC3339Nano)},
			OrderID:    current.OrderID,
			DriverID:   claims.Subject,
			RetailerID: current.RetailerID,
			SupplierID: current.SupplierID,
			Status:     string(current.Status),
		}); err != nil {
			return err
		}

		for _, ret := range pendingReturns {
			if err := outbox.EmitJSON(ctx, txn, events.AggregateOrder, current.OrderID, events.TopicMain, map[string]any{
				"type":            events.EventSupplierReturnCreated,
				"order_id":        current.OrderID,
				"return_id":       ret.ReturnID,
				"sku":             ret.SKU,
				"rejected_qty":    ret.RejectedQty,
				"reason":          ret.Reason,
				"driver_id":       claims.Subject,
				"supplier_id":     current.SupplierID,
				"retailer_id":     current.RetailerID,
				"timestamp":       now.UTC().Format(time.RFC3339Nano),
			}); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		s.log.ErrorContext(ctx, "partial offload failed", "order_id", orderID, "err", err)
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}

	// Best-effort flag on Orders.PartialDelivery when Spanner columns exist.
	if s.spannerClient != nil {
		_, _ = s.spannerClient.Apply(ctx, []*spanner.Mutation{
			spanner.UpdateMap("Orders", map[string]any{
				"OrderId":         current.OrderID,
				"PartialDelivery": current.PartialDelivery,
				"UpdatedAt":       now.UTC(),
			}),
		})
	}

	s.invalidateOrderCache(ctx, orderID)
	s.broadcastShopClosed(ctx, current.SupplierID, current.RetailerID, claims.Subject, events.OrderEvent{
		BaseEvent:  events.BaseEvent{Type: events.EventPartialOffload, Timestamp: now.UTC().Format(time.RFC3339Nano)},
		OrderID:    current.OrderID,
		DriverID:   claims.Subject,
		RetailerID: current.RetailerID,
		SupplierID: current.SupplierID,
	})

	var proxUnlocked bool
	var proxMethod string
	if current.ProximityUnlockedAt != nil {
		proxUnlocked = true
		proxMethod = current.ProximityMethod
	}

	resp := driverEndpointResponse{
		OrderID:           current.OrderID,
		Status:            string(current.Status),
		ProximityUnlocked: proxUnlocked,
		ProximityMethod:   proxMethod,
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(ctx, r, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}

func anyPartialLine(lines []LineItem) bool {
	for _, l := range lines {
		if l.OffloadStatus == OffloadStatusPartial || l.OffloadStatus == OffloadStatusNone || l.RemainingQty > 0 {
			return true
		}
	}
	return false
}
