package warehouse

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type opsDriverCreateParams struct {
	DriverID     string
	Name         string
	Phone        string
	PinHash      string
	SupplierID   string
	WarehouseID  string
	CreatedAt    time.Time
}

type opsVehicleCreateParams struct {
	VehicleID    string
	Label        string
	LicensePlate string
	VehicleClass string
	MaxVolumeVU  float64
	SupplierID   string
	WarehouseID  string
	CreatedAt    time.Time
}

func (s *Service) createOpsDriverSpanner(ctx context.Context, params opsDriverCreateParams) error {
	if s.spannerClient == nil {
		return fmt.Errorf("spanner_not_configured")
	}
	buf := &spannerTxnBuffer{}
	emit := func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateDriver, params.DriverID, events.TopicMain, events.DriverEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventDriverCreated,
				Timestamp: params.CreatedAt.Format(time.RFC3339Nano),
			},
			SupplierID:   params.SupplierID,
			DriverID:     params.DriverID,
			HomeNodeType: "WAREHOUSE",
			HomeNodeID:   params.WarehouseID,
		})
	}
	if err := emit(buf); err != nil {
		return err
	}

	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(_ context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := []*spanner.Mutation{spanner.InsertMap("Drivers", map[string]any{
			"DriverId":     params.DriverID,
			"Name":         params.Name,
			"Phone":        params.Phone,
			"PinHash":      params.PinHash,
			"SupplierId":   params.SupplierID,
			"HomeNodeType": "WAREHOUSE",
			"HomeNodeId":   params.WarehouseID,
			"IsActive":     true,
			"OnShift":      true,
			"CreatedAt":    params.CreatedAt,
			"UpdatedAt":    params.CreatedAt,
		})}
		mutations = append(mutations, outboxMutations(buf.events)...)
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return fmt.Errorf("create ops driver: %w", err)
	}
	return nil
}

func (s *Service) createOpsVehicleSpanner(ctx context.Context, params opsVehicleCreateParams) error {
	if s.spannerClient == nil {
		return fmt.Errorf("spanner_not_configured")
	}
	buf := &spannerTxnBuffer{}
	emit := func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateVehicle, params.VehicleID, events.TopicMain, events.VehicleEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventVehicleCreated,
				Timestamp: params.CreatedAt.Format(time.RFC3339Nano),
			},
			SupplierID:   params.SupplierID,
			VehicleID:    params.VehicleID,
			HomeNodeType: "WAREHOUSE",
			HomeNodeID:   params.WarehouseID,
		})
	}
	if err := emit(buf); err != nil {
		return err
	}

	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(_ context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := []*spanner.Mutation{spanner.InsertMap("Vehicles", map[string]any{
			"VehicleId":    params.VehicleID,
			"Label":        params.Label,
			"LicensePlate": params.LicensePlate,
			"VehicleClass": params.VehicleClass,
			"SupplierId":   params.SupplierID,
			"HomeNodeType": "WAREHOUSE",
			"HomeNodeId":   params.WarehouseID,
			"IsActive":     true,
			"MaxVolumeVU":  params.MaxVolumeVU,
			"CreatedAt":    params.CreatedAt,
			"UpdatedAt":    params.CreatedAt,
		})}
		mutations = append(mutations, outboxMutations(buf.events)...)
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return fmt.Errorf("create ops vehicle: %w", err)
	}
	return nil
}

func warehouseActorID(ctx context.Context) string {
	if ops := auth.GetWarehouseOps(ctx); ops != nil && strings.TrimSpace(ops.Subject) != "" {
		return strings.TrimSpace(ops.Subject)
	}
	if claims, ok := auth.FromContext(ctx); ok && strings.TrimSpace(claims.Subject) != "" {
		return strings.TrimSpace(claims.Subject)
	}
	return "warehouse_ops"
}

type opsVehiclePatchParams struct {
	VehicleID         string
	WarehouseID       string
	SupplierID        string
	IsActive          bool
	UnavailableReason string
	UnavailableNote   string
	UpdatedAt         time.Time
}

func (s *Service) patchOpsVehicleSpanner(ctx context.Context, params opsVehiclePatchParams) error {
	if s.spannerClient == nil {
		return fmt.Errorf("spanner_not_configured")
	}
	reason := ""
	note := ""
	if !params.IsActive {
		reason = normalizeWarehouseVehicleReason(params.UnavailableReason)
		note = strings.TrimSpace(params.UnavailableNote)
		if reason == VehicleReasonOther && note == "" {
			note = strings.TrimSpace(params.UnavailableReason)
		}
	}
	buf := &spannerTxnBuffer{}
	event := events.VehicleEvent{
		BaseEvent: events.BaseEvent{
			Type:      events.EventVehicleAvailabilityChanged,
			Timestamp: params.UpdatedAt.Format(time.RFC3339Nano),
		},
		VehicleID:         params.VehicleID,
		SupplierID:        params.SupplierID,
		HomeNodeType:      "WAREHOUSE",
		HomeNodeID:        params.WarehouseID,
		IsActive:          params.IsActive,
		UnavailableReason: reason,
		UnavailableNote:   note,
	}
	if err := outbox.EmitJSON(ctx, buf, events.AggregateVehicle, params.VehicleID, events.TopicMain, event); err != nil {
		return err
	}
	row := map[string]any{
		"VehicleId":    params.VehicleID,
		"IsActive":     params.IsActive,
		"UpdatedAt":    params.UpdatedAt,
		"UnavailableReason": spanner.NullString{},
		"UnavailableNote":     spanner.NullString{},
	}
	if reason != "" {
		row["UnavailableReason"] = reason
	}
	if note != "" {
		row["UnavailableNote"] = note
	}
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(_ context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := []*spanner.Mutation{spanner.UpdateMap("Vehicles", row)}
		mutations = append(mutations, outboxMutations(buf.events)...)
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return fmt.Errorf("patch ops vehicle: %w", err)
	}
	s.broadcastWarehouseEvent(ctx, params.WarehouseID, map[string]any{
		"type":               events.EventVehicleAvailabilityChanged,
		"vehicle_id":         params.VehicleID,
		"warehouse_id":       params.WarehouseID,
		"is_active":          params.IsActive,
		"unavailable_reason": reason,
		"unavailable_note":   note,
		"timestamp":          params.UpdatedAt.Format(time.RFC3339Nano),
	})
	s.invalidateDispatchPlanCache(ctx, params.WarehouseID)
	return nil
}
