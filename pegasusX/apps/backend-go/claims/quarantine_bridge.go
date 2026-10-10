package claims

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	pegasuskafka "github.com/pegasusx/pegasusx/apps/backend-go/kafka"
	kafka "github.com/segmentio/kafka-go"
)

type QuarantineBridge struct {
	service *Service
	log     *slog.Logger
}

func NewQuarantineBridge(service *Service, log *slog.Logger) *QuarantineBridge {
	return &QuarantineBridge{
		service: service,
		log:     log,
	}
}

func (b *QuarantineBridge) HandleEvent(ctx context.Context, msg kafka.Message) error {
	envelope, err := pegasuskafka.ParseEnvelope(msg.Value)
	if err != nil {
		b.log.ErrorContext(ctx, "failed to parse event envelope", "err", err)
		return nil
	}
	if envelope.Type == events.EventReceivingVarianceReported {
		var payload struct {
			OrderID    string `json:"order_id"`
			RetailerID string `json:"retailer_id"`
			Sku        string `json:"sku"`
			Qty        int64  `json:"qty"`
			Condition  string `json:"condition"`
			ReportedBy string `json:"reported_by"`
		}
		if err := json.Unmarshal(msg.Value, &payload); err != nil {
			return nil
		}
		if payload.Condition == "DAMAGED" && payload.Qty > 0 {
			o, ok, err := b.service.OrderLookup().GetOrder(ctx, payload.OrderID)
			if err != nil || !ok {
				b.log.ErrorContext(ctx, "failed to load order for variance", "order", payload.OrderID)
				return nil
			}
			
			// Auto-file claim from receiving variance
			_, err = b.service.CreateFromDriverException(ctx, o, payload.ReportedBy, ClaimTypeDamaged, []ClaimLine{
				{SKU: payload.Sku, Quantity: payload.Qty, Reason: "Auto-filed from receiving variance"},
			}, nil, "Receiving Damage Variance")
			if err != nil {
				b.log.ErrorContext(ctx, "failed to auto-file claim from receiving variance", "err", err)
			}
		}
	}
	return nil
}
