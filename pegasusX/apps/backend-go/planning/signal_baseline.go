package planning

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type signalBaselinePayload struct {
	ProductID   string  `json:"product_id"`
	Units       int64   `json:"units"`
	BaselineQty int64   `json:"baseline_qty"`
	Confidence  float64 `json:"confidence"`
	WarehouseID string  `json:"warehouse_id"`
}

func baselineFromSignalIngest(supplierID string, in SignalIngestInput, now time.Time) (BaselineWriteInput, bool) {
	if supplierID == "" {
		return BaselineWriteInput{}, false
	}
	var payload signalBaselinePayload
	if len(in.Payload) > 0 {
		_ = json.Unmarshal(in.Payload, &payload)
	}
	warehouseID := strings.TrimSpace(in.WarehouseID)
	if warehouseID == "" {
		warehouseID = strings.TrimSpace(payload.WarehouseID)
	}
	productID := strings.TrimSpace(payload.ProductID)
	qty := payload.BaselineQty
	if qty <= 0 {
		qty = payload.Units
	}
	if warehouseID == "" || productID == "" || qty <= 0 {
		return BaselineWriteInput{}, false
	}
	conf := payload.Confidence
	if conf <= 0 {
		conf = 0.65
	}
	source := strings.TrimSpace(in.Source)
	if source == "" {
		source = "signal_ingest"
	}
	day := now.UTC().Truncate(24 * time.Hour)
	return BaselineWriteInput{
		SupplierID:     supplierID,
		WarehouseID:    warehouseID,
		ProductID:      productID,
		ForecastDate:   day,
		BaselineQty:    qty,
		Confidence:     conf,
		Source:         source,
		BaselineSource: "signal_ingest",
	}, true
}

type planningOutboxBuf struct {
	events []outbox.Event
}

func (b *planningOutboxBuf) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

// ProjectSignal persists an ingested planning signal. The Kafka worker uses this
// path and must not emit another copy of the event it just consumed.
func (s *Service) ProjectSignal(ctx context.Context, supplierID string, in SignalIngestInput) (string, error) {
	return s.projectSignal(ctx, supplierID, in, false)
}

// ProjectSignalWithOutbox persists the signal and the ingest event in one transaction.
// HTTP ingest uses this so a Kafka outage cannot drop the signal.
func (s *Service) ProjectSignalWithOutbox(ctx context.Context, supplierID string, in SignalIngestInput) (string, error) {
	return s.projectSignal(ctx, supplierID, in, true)
}

func (s *Service) projectSignal(ctx context.Context, supplierID string, in SignalIngestInput, emit bool) (string, error) {
	if s == nil || s.Spanner == nil {
		return "", errors.New("planning unavailable")
	}
	signalID := strings.TrimSpace(in.SignalID)
	if signalID == "" {
		signalID = uuid.NewString()
		in.SignalID = signalID
	}
	if strings.TrimSpace(in.Source) == "" {
		return "", errors.New("source_required")
	}
	now := s.Now()
	_, err := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		muts := []*spanner.Mutation{spanner.InsertOrUpdateMap("PlanningSignalProjections", map[string]any{
			"SupplierId":  supplierID,
			"SignalId":    signalID,
			"Source":      strings.TrimSpace(in.Source),
			"PayloadJson": string(in.Payload),
			"IngestedAt":  spanner.CommitTimestamp,
		})}
		if emit {
			buf := &planningOutboxBuf{}
			if err := outbox.EmitJSON(ctx, buf, "PlanningSignal", signalID, events.TopicPlanningSignalIngest, map[string]any{
				"type":         events.EventPlanningSignalIngest,
				"supplier_id":  supplierID,
				"signal_id":    signalID,
				"source":       strings.TrimSpace(in.Source),
				"warehouse_id": strings.TrimSpace(in.WarehouseID),
				"retailer_id":  strings.TrimSpace(in.RetailerID),
				"payload":      json.RawMessage(in.Payload),
				"timestamp":    now.UTC().Format(time.RFC3339Nano),
				"v":            1,
			}); err != nil {
				return err
			}
			for _, e := range buf.events {
				muts = append(muts, spanner.InsertOrUpdateMap("OutboxEvents", outbox.EventRowMap(e)))
			}
		}
		return txn.BufferWrite(muts)
	})
	if err != nil {
		return "", err
	}
	if baseline, ok := baselineFromSignalIngest(supplierID, in, now); ok {
		if err := WriteBaselineWithOutbox(ctx, s.Spanner, now, baseline); err != nil {
			return "", err
		}
	}
	return signalID, nil
}
