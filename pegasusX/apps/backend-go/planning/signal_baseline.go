package planning

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
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

// ProjectSignal persists an ingested planning signal and optionally upserts demand baseline.
func (s *Service) ProjectSignal(ctx context.Context, supplierID string, in SignalIngestInput) error {
	if s == nil || s.Spanner == nil {
		return errors.New("planning unavailable")
	}
	signalID := strings.TrimSpace(in.SignalID)
	if signalID == "" {
		return errors.New("signal_id_required")
	}
	now := s.Now()
	_, err := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{spanner.InsertOrUpdateMap("PlanningSignalProjections", map[string]any{
			"SupplierId":  supplierID,
			"SignalId":    signalID,
			"Source":      strings.TrimSpace(in.Source),
			"PayloadJson": string(in.Payload),
			"IngestedAt":  spanner.CommitTimestamp,
		})})
	})
	if err != nil {
		return err
	}
	if baseline, ok := baselineFromSignalIngest(supplierID, in, now); ok {
		return WriteBaselineWithOutbox(ctx, s.Spanner, now, baseline)
	}
	return nil
}
