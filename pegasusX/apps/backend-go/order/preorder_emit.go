package order

import (
	"context"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

func emitPreorderEvent(ctx context.Context, txn outbox.TxnBuffer, eventType string, o Order, actorRole, actorID string) error {
	return outbox.EmitJSON(ctx, txn, events.AggregateOrder, o.OrderID, events.TopicMain, events.OrderEvent{
		BaseEvent:             events.BaseEvent{Type: eventType, Timestamp: o.UpdatedAt.Format(time.RFC3339Nano)},
		OrderID:               o.OrderID,
		SupplierID:            o.SupplierID,
		RetailerID:            o.RetailerID,
		WarehouseID:           o.WarehouseID,
		Status:                string(o.Status),
		OrderSource:           string(o.Source),
		ConfirmationStatus:    string(o.ConfirmationStatus),
		RequestedDeliveryDate: formatOptionalRFC3339(o.RequestedDeliveryDate),
		TotalMinor:            o.TotalMinor,
		Currency:              o.Currency,
		ActorRole:             actorRole,
		ActorID:               actorID,
	})
}
