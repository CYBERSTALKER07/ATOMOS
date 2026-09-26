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

	// 1. Idempotent check: order is already marked delivered or completed
	if current.Status == StatusCompleted || current.Status == StatusDeliveredOnCredit || current.Status == StatusPendingCashCollection {
		return OfflineDeliverySyncResult{
			OrderID:     req.OrderID,
			Resolution:  ResolutionIdempotentNoop,
			FinalStatus: current.Status,
			Disputed:    false,
		}, nil
	}

	// 2. Active order: standard physical delivery completion
	if current.Status == StatusArrived || current.Status == StatusInTransit {
		targetStatus := StatusDeliveredOnCredit
		if strings.EqualFold(req.PaymentMethod, "CASH") && req.AmountCollectedMinor > 0 {
			targetStatus = StatusPendingCashCollection
		} else if strings.EqualFold(req.PaymentMethod, "PREPAID") {
			targetStatus = StatusCompleted
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
				BaseEvent:  events.BaseEvent{Type: events.EventOrderStatusChanged, Timestamp: current.UpdatedAt.Format(time.RFC3339Nano)},
				OrderID:    current.OrderID,
				SupplierID: current.SupplierID,
				RetailerID: current.RetailerID,
				DriverID:   current.DriverID,
				Status:     string(targetStatus),
				Reason:     "offline_delivery_sync",
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

		current.Status = StatusDeliveredOnCredit
		current.UpdatedAt = s.now().UTC()
		if req.DriverID != "" && current.DriverID == "" {
			current.DriverID = req.DriverID
		}
		current.WarehouseNotes = strings.TrimSpace(current.WarehouseNotes + " | " + disputeNote)

		err := s.repo.UpdateOrder(ctx, current, nil, func(txn outbox.TxnBuffer) error {
			return outbox.EmitJSON(ctx, txn, events.AggregateOrder, current.OrderID, events.TopicExceptions, map[string]any{
				"type":                 "ORDER_CONFLICT_RECONCILED",
				"order_id":             req.OrderID,
				"supplier_id":          current.SupplierID,
				"retailer_id":          current.RetailerID,
				"driver_id":            req.DriverID,
				"resolution":           string(ResolutionPhysicalCustodyWon),
				"prior_status":         string(StatusCancelled),
				"new_status":           string(StatusDeliveredOnCredit),
				"delivered_at":         req.DeliveredAt.Format(time.RFC3339),
				"proof_signature":      req.ProofSignature,
				"proof_photo_url":      req.ProofPhotoURL,
				"dispute_note":         disputeNote,
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
			FinalStatus: StatusDeliveredOnCredit,
			Disputed:    true,
			DisputeNote: disputeNote,
		}, nil
	}

	return OfflineDeliverySyncResult{}, fmt.Errorf("order %s in unhandled status %s for offline reconciliation", req.OrderID, current.Status)
}
