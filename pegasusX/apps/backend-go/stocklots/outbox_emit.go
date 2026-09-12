package stocklots

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// emitWMSEvent buffers a WMS domain event onto the same Spanner RW transaction
// as the stock mutation (Wave B2 M-P0-3). Call after domain BufferWrite work,
// before the transaction returns.
func emitWMSEvent(ctx context.Context, txn *spanner.ReadWriteTransaction, eventType, warehouseID, supplierID, aggregateID string, extra map[string]any) error {
	if txn == nil || stringsEmpty(eventType) || stringsEmpty(warehouseID) {
		return nil
	}
	if stringsEmpty(aggregateID) {
		aggregateID = warehouseID
	}
	payload := map[string]any{
		"type":         eventType,
		"warehouse_id": warehouseID,
		"supplier_id":  supplierID,
		"timestamp":    time.Now().UTC().Format(time.RFC3339Nano),
	}
	for k, v := range extra {
		if k == "" || v == nil {
			continue
		}
		payload[k] = v
	}
	buf := outbox.NewSpannerTxnBuffer(txn)
	if err := outbox.EmitJSON(ctx, buf, events.AggregateWarehouse, aggregateID, events.TopicMain, payload); err != nil {
		return err
	}
	return buf.Flush(ctx)
}

func stringsEmpty(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] != ' ' && s[i] != '\t' {
			return false
		}
	}
	return true
}
