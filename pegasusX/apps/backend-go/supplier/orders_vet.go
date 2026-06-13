package supplier

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// VetOrderParams captures a supplier vet decision for a queued order.
type VetOrderParams struct {
	OrderID    string
	Decision   string
	Note       string
	DecidedBy  string
	ActorRole  string
}

// ErrOrderNotFound is returned when the order id does not exist for the supplier.
var ErrOrderNotFound = errors.New("order not found")

// ErrVetForbidden is returned when the order belongs to another supplier.
var ErrVetForbidden = errors.New("order forbidden")

// ErrInvalidVetState is returned when the order is not awaiting supplier vet.
var ErrInvalidVetState = errors.New("order not awaiting vet")

// ErrOrderAlreadyAssigned is returned when dispatch bindings already exist.
var ErrOrderAlreadyAssigned = errors.New("order already assigned")

// ErrPaymentNotCleared is returned when APPROVED is requested before payment clears.
var ErrPaymentNotCleared = errors.New("payment not cleared")

type supplierOrderVetter interface {
	VetOrder(ctx context.Context, supplierID string, params VetOrderParams) (SupplierOrder, error)
}

// VetOrder persists APPROVED/REJECTED supplier decisions on Orders via Spanner.
func (r *SpannerRepository) VetOrder(ctx context.Context, supplierID string, params VetOrderParams) (SupplierOrder, error) {
	if r == nil || r.client == nil {
		return SupplierOrder{}, fmt.Errorf("spanner supplier repository: nil client")
	}
	supplierID = strings.TrimSpace(supplierID)
	orderID := strings.TrimSpace(params.OrderID)
	decision := strings.ToUpper(strings.TrimSpace(params.Decision))
	if supplierID == "" || orderID == "" {
		return SupplierOrder{}, fmt.Errorf("vet order: supplier_id and order_id required")
	}
	if decision != "APPROVED" && decision != "REJECTED" {
		return SupplierOrder{}, fmt.Errorf("vet order: decision must be APPROVED or REJECTED")
	}

	var result SupplierOrder
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "Orders", spanner.Key{orderID}, []string{
			"OrderId", "SupplierId", "RetailerId",
			"WarehouseId", "DriverId", "VehicleId",
			"RouteId", "ManifestId",
			"Status", "ConfirmationStatus", "TotalMinor", "Currency", "Version",
			"CreatedAt", "UpdatedAt",
		})
		if err != nil {
			if errors.Is(err, spanner.ErrRowNotFound) {
				return ErrOrderNotFound
			}
			return fmt.Errorf("read order %s: %w", orderID, err)
		}

		var (
			current       SupplierOrder
			confirmation  string
			version       int64
			warehouseID   spanner.NullString
			driverID      spanner.NullString
			vehicleID     spanner.NullString
			routeID       spanner.NullString
			manifestID    spanner.NullString
			createdAt     time.Time
			updatedAt     time.Time
		)
		if err := row.Columns(
			&current.OrderID,
			&current.SupplierID,
			&current.RetailerID,
			&warehouseID,
			&driverID,
			&vehicleID,
			&routeID,
			&manifestID,
			&current.Status,
			&confirmation,
			&current.TotalMinor,
			&current.Currency,
			&version,
			&createdAt,
			&updatedAt,
		); err != nil {
			return fmt.Errorf("scan order %s: %w", orderID, err)
		}
		current.WarehouseID = warehouseID.StringVal
		current.DriverID = driverID.StringVal
		current.VehicleID = vehicleID.StringVal
		current.RouteID = routeID.StringVal
		current.ManifestID = manifestID.StringVal
		if current.SupplierID != supplierID {
			return ErrVetForbidden
		}
		if !strings.EqualFold(confirmation, "PENDING") && !strings.EqualFold(confirmation, "DRAFT") {
			return ErrInvalidVetState
		}
		if strings.TrimSpace(current.DriverID) != "" ||
			strings.TrimSpace(current.RouteID) != "" ||
			strings.TrimSpace(current.ManifestID) != "" {
			return ErrOrderAlreadyAssigned
		}
		if decision == "APPROVED" {
			cleared, err := orderPaymentClearedInTxn(ctx, txn, orderID)
			if err != nil {
				return fmt.Errorf("payment clearance check %s: %w", orderID, err)
			}
			if !cleared {
				return ErrPaymentNotCleared
			}
		}

		previousStatus := current.Status
		nextConfirmation := "CONFIRMED"
		nextStatus := current.Status
		if strings.TrimSpace(nextStatus) == "" {
			nextStatus = "PENDING"
		}
		if decision == "REJECTED" {
			nextConfirmation = "REJECTED"
			nextStatus = "CANCELLED"
		}
		decisionAt := time.Now().UTC()
		nextVersion := version + 1

		buf := &spannerTxnBuffer{}
		reason := strings.TrimSpace(params.Note)
		if reason == "" && decision == "REJECTED" {
			reason = "supplier_rejected"
		}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, orderID, events.TopicMain, events.OrderEvent{
			BaseEvent:          events.BaseEvent{Type: events.EventOrderStatusChanged, Version: nextVersion, Timestamp: decisionAt.Format(time.RFC3339Nano)},
			OrderID:            orderID,
			SupplierID:         supplierID,
			RetailerID:         current.RetailerID,
			PreviousStatus:     previousStatus,
			Status:             nextStatus,
			ConfirmationStatus: nextConfirmation,
			Reason:             reason,
			ActorRole:          strings.TrimSpace(params.ActorRole),
			ActorID:            strings.TrimSpace(params.DecidedBy),
			TotalMinor:         current.TotalMinor,
			Currency:           current.Currency,
			Version:            nextVersion,
		}); err != nil {
			return fmt.Errorf("emit vet order event %s: %w", orderID, err)
		}

		mutations := []*spanner.Mutation{
			spanner.UpdateMap("Orders", map[string]any{
				"OrderId":            orderID,
				"ConfirmationStatus": nextConfirmation,
				"Status":             nextStatus,
				"DecisionAt":         decisionAt,
				"DecisionBy":         strings.TrimSpace(params.DecidedBy),
				"AutoConfirmAt":      nil,
				"Version":            nextVersion,
				"UpdatedAt":          spanner.CommitTimestamp,
			}),
		}
		for _, e := range buf.events {
			createdAt := e.CreatedAt.UTC()
			if createdAt.IsZero() {
				createdAt = time.Now().UTC()
			}
			row := map[string]any{
				"EventId":       e.EventID,
				"AggregateType": e.AggregateType,
				"AggregateId":   e.AggregateID,
				"TopicName":     e.TopicName,
				"Payload":       e.Payload,
				"CreatedAt":     createdAt,
				"PublishedAt":   nil,
			}
			if e.PublishedAt != nil {
				row["PublishedAt"] = e.PublishedAt.UTC()
			}
			mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxEvents", row))
		}
		if err := txn.BufferWrite(mutations); err != nil {
			return fmt.Errorf("buffer vet order %s: %w", orderID, err)
		}

		current.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
		current.UpdatedAt = decisionAt.Format(time.RFC3339Nano)
		current.Decision = decision
		current.Note = strings.TrimSpace(params.Note)
		applySupplierOrderPresentation(&current, nextConfirmation, nextStatus)
		result = current
		return nil
	})
	if err != nil {
		return SupplierOrder{}, err
	}
	return result, nil
}

func applySupplierOrderPresentation(order *SupplierOrder, confirmationStatus, status string) {
	if order == nil {
		return
	}
	order.Status = strings.TrimSpace(status)
	switch strings.ToUpper(strings.TrimSpace(confirmationStatus)) {
	case "CONFIRMED", "AUTO_CONFIRMED":
		order.Decision = "APPROVED"
	case "REJECTED":
		order.Decision = "REJECTED"
		if strings.EqualFold(order.Status, "CANCELLED") {
			order.Status = "REJECTED"
		}
	}
	if strings.EqualFold(order.Status, "PENDING") && strings.EqualFold(confirmationStatus, "PENDING") {
		order.Status = "AWAITING_REVIEW"
	}
	order.TrackingStatus = "unassigned"
	if strings.TrimSpace(order.DriverID) != "" && strings.TrimSpace(order.RouteID) != "" {
		order.TrackingStatus = "assigned"
	}
}

func orderPaymentClearedInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID string) (bool, error) {
	stmt := spanner.Statement{
		SQL: `SELECT
		        EXISTS (
		          SELECT 1
		          FROM PaymentLedgerEntries ple
		          WHERE ple.OrderId = @orderId
		            AND ple.EntryType IN UNNEST(@clearedEntryTypes)
		        ) AS ledger_cleared,
		        EXISTS (
		          SELECT 1
		          FROM PaymentSessions ps
		          WHERE ps.OrderId = @orderId
		            AND ps.Status IN UNNEST(@clearedSessionStatuses)
		        ) AS session_cleared`,
		Params: map[string]any{
			"orderId":                orderID,
			"clearedEntryTypes":      []string{"WEBHOOK_PAID", "CASH_COLLECTED", "SETTLEMENT_CREDIT"},
			"clearedSessionStatuses": []string{"PAID", "CAPTURED", "SETTLED", "SUCCESS", "AUTHORIZED"},
		},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return false, fmt.Errorf("query payment clearance: %w", err)
	}
	var ledgerCleared, sessionCleared bool
	if err := row.Columns(&ledgerCleared, &sessionCleared); err != nil {
		return false, fmt.Errorf("scan payment clearance: %w", err)
	}
	return ledgerCleared || sessionCleared, nil
}
