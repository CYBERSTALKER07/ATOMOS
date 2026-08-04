package returns

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	kafka "github.com/segmentio/kafka-go"
)

func TestEventConsumer_ReverseOpensTickets(t *testing.T) {
	svc := &Service{
		log: slog.New(slog.NewTextHandler(io.Discard, nil)),
		// spanner nil → OpenTickets returns nil,nil — still exercises decode path.
	}
	c := NewEventConsumer(svc, svc.log)
	payload, _ := json.Marshal(map[string]any{
		"type":         events.EventReverseLogisticsRequired,
		"claim_id":     "clm-1",
		"order_id":     "ord-1",
		"warehouse_id": "wh-1",
		"supplier_id":  "sup-1",
		"claim_type":   "CONCEALED_DAMAGE",
		"source":       "RETAILER_CLAIM",
		"line_items": []map[string]any{
			{"sku": "SKU-1", "quantity": 2, "reason": "CONCEALED_DAMAGE"},
		},
	})
	if err := c.HandleEvent(context.Background(), kafka.Message{Value: payload}); err != nil {
		t.Fatalf("handle: %v", err)
	}
}

func TestEventConsumer_ClaimFiledMissingSkipped(t *testing.T) {
	svc := &Service{log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	c := NewEventConsumer(svc, svc.log)
	payload, _ := json.Marshal(map[string]any{
		"type":       events.EventClaimFiled,
		"claim_id":   "clm-2",
		"order_id":   "ord-2",
		"claim_type": "MISSING",
		"line_items": []map[string]any{
			{"sku": "SKU-1", "quantity": 1, "reason": "MISSING"},
		},
	})
	if err := c.HandleEvent(context.Background(), kafka.Message{Value: payload}); err != nil {
		t.Fatalf("handle: %v", err)
	}
}
