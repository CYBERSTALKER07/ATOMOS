package returns

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	pegasuskafka "github.com/pegasusx/pegasusx/apps/backend-go/kafka"
	kafka "github.com/segmentio/kafka-go"
)

// EventConsumer opens reverse-logistics dock tickets from CLAIM / REVERSE events (G12).
type EventConsumer struct {
	svc *Service
	log *slog.Logger
}

// NewEventConsumer constructs the returns Kafka consumer.
func NewEventConsumer(svc *Service, log *slog.Logger) *EventConsumer {
	if log == nil {
		log = slog.Default()
	}
	return &EventConsumer{svc: svc, log: log}
}

// HandleEvent processes REVERSE_LOGISTICS_REQUIRED and damage CLAIM_FILED payloads.
func (c *EventConsumer) HandleEvent(ctx context.Context, msg kafka.Message) error {
	if c == nil || c.svc == nil {
		return nil
	}
	envelope, err := pegasuskafka.ParseEnvelope(msg.Value)
	if err != nil {
		c.log.Warn("returns consumer payload parsing failed", "err", err, "topic", msg.Topic)
		return nil
	}
	switch envelope.Type {
	case events.EventReverseLogisticsRequired, events.EventClaimFiled:
		return c.openFromClaimEvent(ctx, msg.Value, envelope.Type)
	default:
		return nil
	}
}

func (c *EventConsumer) openFromClaimEvent(ctx context.Context, raw []byte, eventType string) error {
	var payload struct {
		ClaimID     string `json:"claim_id"`
		OrderID     string `json:"order_id"`
		WarehouseID string `json:"warehouse_id"`
		SupplierID  string `json:"supplier_id"`
		DriverID    string `json:"driver_id"`
		ClaimType   string `json:"claim_type"`
		Source      string `json:"source"`
		LineItems   []struct {
			SKU      string `json:"sku"`
			Quantity int64  `json:"quantity"`
			Reason   string `json:"reason"`
		} `json:"line_items"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		c.log.Warn("returns consumer decode failed", "err", err, "type", eventType)
		return nil
	}
	if strings.TrimSpace(payload.OrderID) == "" || strings.TrimSpace(payload.ClaimID) == "" {
		return nil
	}
	// CLAIM_FILED without lines is notify-only; reverse open needs SKUs.
	if eventType == events.EventClaimFiled && len(payload.LineItems) == 0 {
		return nil
	}
	claimType := strings.ToUpper(strings.TrimSpace(payload.ClaimType))
	if eventType == events.EventClaimFiled {
		switch claimType {
		case "DAMAGED", "CONCEALED_DAMAGE", "TEMPERATURE", "TAMPER":
		default:
			return nil // MISSING/OTHER: typically no physical reverse
		}
	}
	lines := make([]TicketLine, 0, len(payload.LineItems))
	for _, li := range payload.LineItems {
		sku := strings.TrimSpace(li.SKU)
		if sku == "" || li.Quantity <= 0 {
			continue
		}
		reason := strings.TrimSpace(li.Reason)
		if reason == "" {
			reason = claimType
		}
		if reason == "" {
			reason = "DAMAGED"
		}
		lines = append(lines, TicketLine{SKU: sku, Quantity: li.Quantity, Reason: reason})
	}
	if len(lines) == 0 {
		return nil
	}
	source := strings.TrimSpace(payload.Source)
	if source == "" {
		source = "CLAIM"
	}
	_, err := c.svc.OpenTickets(ctx, OpenTicketsInput{
		OrderID:     payload.OrderID,
		WarehouseID: payload.WarehouseID,
		SupplierID:  payload.SupplierID,
		DriverID:    payload.DriverID,
		ClaimID:     payload.ClaimID,
		Source:      source,
		Note:        claimType,
		Lines:       lines,
	})
	if err != nil {
		c.log.ErrorContext(ctx, "returns consumer OpenTickets failed",
			"err", err, "claim_id", payload.ClaimID, "order_id", payload.OrderID, "type", eventType)
		return err // retry via Kafka
	}
	return nil
}
