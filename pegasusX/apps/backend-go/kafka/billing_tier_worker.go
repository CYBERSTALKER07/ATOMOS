package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"log"
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
	Type        string  `json:"type"`
	OrderID     string  `json:"order_id"`
	SupplierID  string  `json:"supplier_id"`
	AmountMinor int64   `json:"amount_minor"`
	TotalMinor  int64   `json:"total_minor"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Total       struct {
		Amount   int64  `json:"amount"`
		Currency string `json:"currency"`
	} `json:"total"`
}

// HandleMessage processes incoming Kafka messages for the billing tier worker.
func (w *BillingTierWorker) HandleMessage(ctx context.Context, msg []byte) error {
	if w == nil || w.MeterWorker == nil {
		return nil
	}
	var event orderFinalizedBillingEvent
	if err := json.Unmarshal(msg, &event); err != nil {
		log.Printf("Failed to unmarshal billing event: %v", err)
		return err
	}

	if event.Type != events.EventOrderFinalized {
		return nil
	}
	orderID := strings.TrimSpace(event.OrderID)
	supplierID := strings.TrimSpace(event.SupplierID)
	if orderID == "" || supplierID == "" {
		log.Printf("billing ORDER_FINALIZED missing order_id or supplier_id; skipping")
		return nil
	}

	operating := fxrates.NormalizeCurrency(w.OperatingCurrency)
	if operating == "" {
		if c, err := auth.CurrencyFromContext(ctx, supplierID); err == nil {
			operating = fxrates.NormalizeCurrency(c)
		}
	}
	if operating == "" {
		log.Printf("billing ORDER_FINALIZED empty operating currency (planned/unknown pack); skipping orderID=%s", orderID)
		return nil
	}
	source := fxrates.NormalizeCurrency(event.Total.Currency)
	if source == "" {
		source = fxrates.NormalizeCurrency(event.Currency)
	}
	if source == "" {
		source = operating
	}

	minor := billing.ResolveMeterAmountMinor(event.AmountMinor, event.Total.Amount, event.TotalMinor, event.Amount)
	if minor <= 0 {
		log.Printf("billing ORDER_FINALIZED non-positive amount; skipping orderID=%s", orderID)
		return nil
	}

	now := time.Now().UTC()
	if w.Now != nil {
		now = w.Now()
	}

	if source != operating {
		if w.Fx == nil {
			log.Printf("billing FX skip: no converter source=%s operating=%s orderID=%s", source, operating, orderID)
			return nil
		}
		converted, err := w.Fx.ConvertMinor(ctx, source, operating, minor, now)
		if errors.Is(err, fxrates.ErrRateMissing) {
			log.Printf("billing FX skip missing rate source=%s operating=%s orderID=%s minor=%d", source, operating, orderID, minor)
			return nil
		}
		if err != nil {
			return err
		}
		log.Printf("billing FX converted orderID=%s source=%s→%s minor=%d→%d", orderID, source, operating, minor, converted)
		minor = converted
	}

	amount := billing.MinorToMajor(minor)
	return w.MeterWorker.ProcessOrderFinalized(ctx, orderID, amount, supplierID)
}
