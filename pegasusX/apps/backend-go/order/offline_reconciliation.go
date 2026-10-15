package order

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// ConflictResolution designates how a state divergence was reconciled.
type ConflictResolution string

const (
	ResolutionPhysicalCustodyWon ConflictResolution = "PHYSICAL_CUSTODY_WON"
	ResolutionIdempotentNoop     ConflictResolution = "IDEMPOTENT_NOOP"
	ResolutionNormalDelivery     ConflictResolution = "NORMAL_DELIVERY"
)

// OfflineDeliverySyncRequest is submitted by a driver device when connectivity restores.
type OfflineDeliverySyncRequest struct {
	OrderID              string    `json:"order_id"`
	DriverID             string    `json:"driver_id"`
	DeliveredAt          time.Time `json:"delivered_at"`
	ProofSignature       string    `json:"proof_signature,omitempty"`
	ProofPhotoURL        string    `json:"proof_photo_url,omitempty"`
	Notes                string    `json:"notes,omitempty"`
	PaymentMethod        string    `json:"payment_method,omitempty"` // CASH, CREDIT, PREPAID
	AmountCollectedMinor int64     `json:"amount_collected_minor,omitempty"`
}

// OfflineDeliverySyncResult returns the outcome of the reconciliation.
type OfflineDeliverySyncResult struct {
	OrderID     string             `json:"order_id"`
	Resolution  ConflictResolution `json:"resolution"`
	FinalStatus Status             `json:"final_status"`
	Disputed    bool               `json:"disputed"`
	DisputeNote string             `json:"dispute_note,omitempty"`
}

// ReconcileOfflineDelivery reconciles an offline driver delivery against the current order state,
// enforcing the Physical Custody Supremacy Invariant (PHYSICAL_DELIVERY > CONCURRENT_CANCEL).
func (s *Service) ReconcileOfflineDelivery(ctx context.Context, req OfflineDeliverySyncRequest) (OfflineDeliverySyncResult, error) {
	if strings.TrimSpace(req.OrderID) == "" {
		return OfflineDeliverySyncResult{}, errors.New("order_id required")
	}
	if req.DeliveredAt.IsZero() {
		req.DeliveredAt = s.now().UTC()
	}

	current, found, err := s.repo.GetOrder(ctx, req.OrderID)
	if err != nil {
		return OfflineDeliverySyncResult{}, fmt.Errorf("load order %s: %w", req.OrderID, err)
	}
	if !found {
		return OfflineDeliverySyncResult{}, fmt.Errorf("order %s: not found", req.OrderID)
	}

	// 1. Idempotent check: order is already past the offline delivery write.
	if offlineDeliveryAlreadyApplied(current.Status) {
		return OfflineDeliverySyncResult{
			OrderID:     req.OrderID,
			Resolution:  ResolutionIdempotentNoop,
			FinalStatus: current.Status,
			Disputed:    false,
		}, nil
	}

	// 2. Active order: walk only edges the status graph allows.
	if current.Status == StatusArrived || current.Status == StatusInTransit {
		previousStatus := current.Status
		targetStatus, err := offlineDeliveryTarget(previousStatus, req.PaymentMethod, req.AmountCollectedMinor)
		if err != nil {
			return OfflineDeliverySyncResult{}, fmt.Errorf("offline delivery %s: %w", req.OrderID, err)
		}

		current.Status = targetStatus
		current.UpdatedAt = s.now().UTC()
		if req.DriverID != "" && current.DriverID == "" {
			current.DriverID = req.DriverID
		}
		if req.Notes != "" {
			current.WarehouseNotes = strings.TrimSpace(current.WarehouseNotes + " | POD: " + req.Notes)
		}

		if err := s.repo.UpdateOrder(ctx, current, nil, func(txn outbox.TxnBuffer) error {
			return outbox.EmitJSON(ctx, txn, events.AggregateOrder, current.OrderID, events.TopicOrders, events.OrderEvent{
				BaseEvent:      events.BaseEvent{Type: events.EventOrderStatusChanged, Timestamp: current.UpdatedAt.Format(time.RFC3339Nano)},
				OrderID:        current.OrderID,
				SupplierID:     current.SupplierID,
				RetailerID:     current.RetailerID,
				DriverID:       current.DriverID,
				PreviousStatus: string(previousStatus),
				Status:         string(targetStatus),
				Reason:         "offline_delivery_sync",
			})
		}); err != nil {
			return OfflineDeliverySyncResult{}, fmt.Errorf("update delivered order %s: %w", req.OrderID, err)
		}

		return OfflineDeliverySyncResult{
			OrderID:     req.OrderID,
			Resolution:  ResolutionNormalDelivery,
			FinalStatus: targetStatus,
			Disputed:    false,
		}, nil
	}

	// 3. Conflict case: Order was cancelled or cancel-requested online while driver was offline!
	// Invariant: PHYSICAL_DELIVERY > CONCURRENT_CANCEL
	if current.Status == StatusCancelled || current.Status == StatusCancelRequested {
		disputeNote := fmt.Sprintf(
			"PHYSICAL_CUSTODY_WON: Driver %s completed offline delivery at %s with POD. Online cancellation superseded; converted to post-delivery dispute.",
			req.DriverID, req.DeliveredAt.Format(time.RFC3339),
		)
		previousStatus := current.Status
		if err := offlineCancelConflictTarget(previousStatus); err != nil {
			return OfflineDeliverySyncResult{}, fmt.Errorf("offline conflict %s: %w", req.OrderID, err)
		}

		current.Status = StatusReconciliationRequired
		current.UpdatedAt = s.now().UTC()
		if req.DriverID != "" && current.DriverID == "" {
			current.DriverID = req.DriverID
		}
		current.WarehouseNotes = strings.TrimSpace(current.WarehouseNotes + " | " + disputeNote)

		err := s.repo.UpdateOrder(ctx, current, nil, func(txn outbox.TxnBuffer) error {
			return outbox.EmitJSON(ctx, txn, events.AggregateOrder, current.OrderID, events.TopicExceptions, map[string]any{
				"type":            "ORDER_CONFLICT_RECONCILED",
				"order_id":        req.OrderID,
				"supplier_id":     current.SupplierID,
				"retailer_id":     current.RetailerID,
				"driver_id":       req.DriverID,
				"resolution":      string(ResolutionPhysicalCustodyWon),
				"prior_status":    string(previousStatus),
				"new_status":      string(StatusReconciliationRequired),
				"delivered_at":    req.DeliveredAt.Format(time.RFC3339),
				"proof_signature": req.ProofSignature,
				"proof_photo_url": req.ProofPhotoURL,
				"dispute_note":    disputeNote,
			})
		})
		if err != nil {
			return OfflineDeliverySyncResult{}, fmt.Errorf("reconcile conflicted order %s: %w", req.OrderID, err)
		}

		s.log.WarnContext(ctx, "offline physical delivery superseded online cancellation",
			"order_id", req.OrderID,
			"driver_id", req.DriverID,
			"delivered_at", req.DeliveredAt,
		)

		return OfflineDeliverySyncResult{
			OrderID:     req.OrderID,
			Resolution:  ResolutionPhysicalCustodyWon,
			FinalStatus: StatusReconciliationRequired,
			Disputed:    true,
			DisputeNote: disputeNote,
		}, nil
	}

	return OfflineDeliverySyncResult{}, fmt.Errorf("order %s in unhandled status %s for offline reconciliation", req.OrderID, current.Status)
}

func offlineDeliveryAlreadyApplied(status Status) bool {
	switch status {
	case StatusCompleted, StatusDeliveredOnCredit, StatusPendingCashCollection, StatusAwaitingPayment, StatusFiscalizing, StatusReconciliationRequired:
		return true
	default:
		return false
	}
}

// offlineDeliveryTarget walks IN_TRANSIT through ARRIVED, then the payment edge
// the graph already allows. Prepaid stops at FISCALIZING so Soliq still runs.
func offlineDeliveryTarget(from Status, method string, amountMinor int64) (Status, error) {
	hops := []Status{StatusDeliveredOnCredit}
	switch {
	case strings.EqualFold(method, "CASH") && amountMinor > 0:
		hops = []Status{StatusPendingCashCollection}
	case strings.EqualFold(method, "PREPAID"):
		hops = []Status{StatusAwaitingPayment, StatusFiscalizing}
	}
	if from == StatusInTransit {
		hops = append([]Status{StatusArrived}, hops...)
	}
	cursor := from
	for _, next := range hops {
		if err := ValidateStatusTransition(string(cursor), string(next), TransitionOpts{Reason: "offline_delivery_sync"}); err != nil {
			return "", err
		}
		cursor = next
	}
	return cursor, nil
}

// offlineCancelConflictTarget checks the graph path into RECONCILIATION_REQUIRED.
// CANCEL_REQUESTED must pass through CANCELLED; both edges already exist.
func offlineCancelConflictTarget(from Status) error {
	cursor := from
	if cursor == StatusCancelRequested {
		if err := ValidateStatusTransition(string(cursor), string(StatusCancelled), TransitionOpts{Reason: "offline_physical_custody"}); err != nil {
			return err
		}
		cursor = StatusCancelled
	}
	return ValidateStatusTransition(string(cursor), string(StatusReconciliationRequired), TransitionOpts{Reason: "offline_physical_custody"})
}
