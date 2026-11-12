package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/fxrates"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/services/billing"
	segkafka "github.com/segmentio/kafka-go"
)

// BillingTierWorker consumes ORDER_FINALIZED events and routes them to the MeterWorker.
type BillingTierWorker struct {
	MeterWorker       *billing.MeterWorker
	Fx                *fxrates.Service
	OperatingCurrency string
	Now               func() time.Time
}

// NewBillingTierWorker initializes a new BillingTierWorker.
// Empty operating currency reads the shipped pack — never invents UZS.
func NewBillingTierWorker(meterWorker *billing.MeterWorker) *BillingTierWorker {
	op := ""
	if c, err := auth.CurrencyFromContext(context.Background(), ""); err == nil {
		op = c
	}
	return &BillingTierWorker{
		MeterWorker:       meterWorker,
		OperatingCurrency: op,
		Now:               func() time.Time { return time.Now().UTC() },
	}
}

// WithFx wires ConvertMinor into operating currency before metering.
func (w *BillingTierWorker) WithFx(fx *fxrates.Service, operatingCurrency string) *BillingTierWorker {
	if w == nil {
		return nil
	}
	w.Fx = fx
	if op := strings.TrimSpace(operatingCurrency); op != "" {
		w.OperatingCurrency = op
	}
	return w
}

// HandleEvent adapts BillingTierWorker to the Kafka EventHandler signature.
func (w *BillingTierWorker) HandleEvent(ctx context.Context, msg segkafka.Message) error {
	return w.HandleMessage(ctx, msg.Value)
}

// orderFinalizedBillingEvent matches the live ORDER_FINALIZED emit shape from order/service.go
// plus legacy amount / total_minor fields.
type orderFinalizedBillingEvent struct {
	Type              string          `json:"type"`
	OrderID           string          `json:"order_id"`
	SupplierID        string          `json:"supplier_id"`
	AmountMinor       int64           `json:"amount_minor"`
	TotalMinor        int64           `json:"total_minor"`
	LegacyAmountMinor int64           `json:"-"`
	RawAmount         json.RawMessage `json:"amount"`
	Currency          string          `json:"currency"`
	Total             struct {
		Amount   int64  `json:"amount"`
		Currency string `json:"currency"`
	} `json:"total"`
}

// parseLegacyDecimalAmountMinor parses a JSON numeric/string major-unit token (e.g. 450.50, 19.99, 0.29)
// into 2-decimal int64 minor units using pure integer/string arithmetic (zero float64).
func parseLegacyDecimalAmountMinor(raw json.RawMessage) (int64, bool) {
	s := strings.TrimSpace(string(raw))
	s = strings.Trim(s, `"`)
	s = strings.TrimSpace(s)
	if s == "" || s == "null" || strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[") || strings.HasPrefix(s, "-") {
		return 0, false
	}
	parts := strings.Split(s, ".")
	if len(parts) > 2 {
		return 0, false
	}
	wholeStr := parts[0]
	if wholeStr == "" {
		wholeStr = "0"
	}
	whole, err := strconv.ParseInt(wholeStr, 10, 64)
	if err != nil || whole < 0 {
		return 0, false
	}
	var fracMinor int64
	if len(parts) == 2 {
		fracStr := parts[1]
		for i := 0; i < len(fracStr); i++ {
			if fracStr[i] < '0' || fracStr[i] > '9' {
				return 0, false
			}
		}
		padded := (fracStr + "000")[:3]
		cents, err := strconv.ParseInt(padded[:2], 10, 64)
		if err != nil {
			return 0, false
		}
		if padded[2] >= '5' {
			cents++
		}
		fracMinor = cents
	}
	return whole*100 + fracMinor, true
}

func decodeOrderFinalizedBillingEvent(msg []byte) (orderFinalizedBillingEvent, int64, error) {
	var event orderFinalizedBillingEvent
	if err := json.Unmarshal(msg, &event); err != nil {
		return event, 0, err
	}
	if len(event.RawAmount) > 0 {
		if legacyMinor, ok := parseLegacyDecimalAmountMinor(event.RawAmount); ok {
			event.LegacyAmountMinor = legacyMinor
		} else {
			var obj struct {
				Amount   int64  `json:"amount"`
				Currency string `json:"currency"`
			}
			if err := json.Unmarshal(event.RawAmount, &obj); err == nil {
				if event.AmountMinor == 0 && obj.Amount > 0 {
					event.AmountMinor = obj.Amount
				}
				if event.Currency == "" && obj.Currency != "" {
					event.Currency = obj.Currency
				}
			}
		}
	}
	minor := billing.ResolveMeterAmountMinor(event.AmountMinor, event.Total.Amount, event.TotalMinor, event.LegacyAmountMinor)
	return event, minor, nil
}

// HandleMessage processes incoming Kafka messages for the billing tier worker.
func (w *BillingTierWorker) HandleMessage(ctx context.Context, msg []byte) error {
	if w == nil || w.MeterWorker == nil {
		return nil
	}
	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(msg, &envelope); err != nil {
		slog.WarnContext(ctx, "failed to unmarshal billing event", "err", err)
		return err
	}
	if envelope.Type != events.EventOrderFinalized {
		return nil
	}

	event, minor, err := decodeOrderFinalizedBillingEvent(msg)
	if err != nil {
		slog.WarnContext(ctx, "failed to unmarshal billing event", "err", err)
		return err
	}
	orderID := strings.TrimSpace(event.OrderID)
	supplierID := strings.TrimSpace(event.SupplierID)
	if orderID == "" || supplierID == "" {
		slog.DebugContext(ctx, "billing ORDER_FINALIZED missing order_id or supplier_id; skipping")
		return nil
	}

	operating := fxrates.NormalizeCurrency(w.OperatingCurrency)
	if operating == "" {
		if c, err := auth.CurrencyFromContext(ctx, supplierID); err == nil {
			operating = fxrates.NormalizeCurrency(c)
		}
	}
	if operating == "" {
		slog.DebugContext(ctx, "billing ORDER_FINALIZED empty operating currency; skipping", "order_id", orderID)
		return nil
	}
	source := fxrates.NormalizeCurrency(event.Total.Currency)
	if source == "" {
		source = fxrates.NormalizeCurrency(event.Currency)
	}
	if source == "" {
		source = operating
	}

	if minor <= 0 {
		slog.DebugContext(ctx, "billing ORDER_FINALIZED non-positive amount; skipping", "order_id", orderID)
		return nil
	}

	now := time.Now().UTC()
	if w.Now != nil {
		now = w.Now()
	}

	if source != operating {
		if w.Fx == nil {
			slog.DebugContext(ctx, "billing FX skip: no converter", "source", source, "operating", operating, "order_id", orderID)
			return nil
		}
		converted, err := w.Fx.ConvertMinor(ctx, source, operating, minor, now)
		if errors.Is(err, fxrates.ErrRateMissing) {
			slog.DebugContext(ctx, "billing FX skip missing rate", "source", source, "operating", operating, "order_id", orderID, "minor", minor)
			return nil
		}
		if err != nil {
			return err
		}
		slog.InfoContext(ctx, "billing FX converted", "order_id", orderID, "source", source, "operating", operating, "from_minor", minor, "to_minor", converted)
		minor = converted
	}

	return w.MeterWorker.ProcessOrderFinalized(ctx, orderID, minor, supplierID)
}
