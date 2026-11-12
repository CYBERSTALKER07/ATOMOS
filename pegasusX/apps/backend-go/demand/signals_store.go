package demand

import (
	"context"

	"cloud.google.com/go/spanner"
)

// UpsertSignals idempotently writes demand signals. Weather ingestion no longer calls this.
func (s *Service) UpsertSignals(ctx context.Context, signals []DemandSignal) error {
	if s == nil || s.spanner == nil || len(signals) == 0 {
		return nil
	}
	muts := make([]*spanner.Mutation, 0, len(signals))
	for _, sig := range signals {
		var sku spanner.NullString
		if sig.Sku != nil {
			sku = spanner.NullString{StringVal: *sig.Sku, Valid: true}
		}
		var meta spanner.NullJSON
		if len(sig.Meta) > 0 {
			meta = spanner.NullJSON{Value: string(sig.Meta), Valid: true}
		}
		muts = append(muts, spanner.InsertOrUpdateMap("DemandSignals", map[string]any{
			"SignalId":   sig.SignalId,
			"Type":       string(sig.Type),
			"Scope":      sig.Scope,
			"Sku":        sku,
			"StartAt":    sig.StartAt,
			"EndAt":      sig.EndAt,
			"Multiplier": sig.Multiplier,
			"Meta":       meta,
			"CreatedAt":  sig.CreatedAt,
			"CreatedBy":  sig.CreatedBy,
			"SupplierId": ResolveSupplierID(sig.SupplierId),
		}))
	}

	const batch = 500
	for i := 0; i < len(muts); i += batch {
		end := i + batch
		if end > len(muts) {
			end = len(muts)
		}
		chunk := muts[i:end]
		if _, err := s.spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			return txn.BufferWrite(chunk)
		}); err != nil {
			return err
		}
	}
	return nil
}
