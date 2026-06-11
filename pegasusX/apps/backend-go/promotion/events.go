package promotion

import (
	"context"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

func emitPromotionChanged(
	ctx context.Context,
	buf *spannerTxnBuffer,
	p Promotion,
	action string,
) error {
	if buf == nil {
		return nil
	}
	return outbox.EmitJSON(ctx, buf, events.AggregatePromotion, p.PromotionID, events.TopicMain, events.PromotionEvent{
		BaseEvent: events.BaseEvent{
			Type:      events.EventPromotionChanged,
			Version:   p.Version,
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		},
		SupplierID:    p.SupplierID,
		PromotionID:   p.PromotionID,
		RetailerScope: string(p.RetailerScope),
		RetailerIDs:   p.RetailerIDs,
		Action:        action,
	})
}
