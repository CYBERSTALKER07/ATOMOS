package demand

import (
	"context"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
)

// RunDensityWorker runs the event density aggregation logic periodically.
func (s *Service) RunDensityWorker(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.ComputeDensitySignals(ctx); err != nil {
				// log error
			}
		}
	}
}

// ComputeDensitySignals computes and upserts event density signals.
func (s *Service) ComputeDensitySignals(ctx context.Context) error {
	now := time.Now().UTC()
	today := civil.DateOf(now)

	// Stubbed aggregation logic
	type eventDensity struct {
		ZoneH3       string
		Date         civil.Date
		DensityScore float64
		EventsJson   string
		ComputedAt   time.Time
	}

	mockData := []eventDensity{
		{
			ZoneH3:       "8928308280fffff",
			Date:         today,
			DensityScore: 0.85,
			EventsJson:   `{"events": ["concert", "festival"]}`,
			ComputedAt:   now,
		},
		{
			ZoneH3:       "8928308280ffffe",
			Date:         today,
			DensityScore: 0.20,
			EventsJson:   `{"events": ["local_market"]}`,
			ComputedAt:   now,
		},
	}

	var mutations []*spanner.Mutation
	for _, data := range mockData {
		mutations = append(mutations, spanner.InsertOrUpdateMap("EventDensitySignals", map[string]any{
			"ZoneH3":       data.ZoneH3,
			"Date":         spanner.NullDate{Valid: true, Date: data.Date},
			"DensityScore": data.DensityScore,
			"EventsJson":   data.EventsJson,
			"ComputedAt":   data.ComputedAt,
		}))
	}

	_, err := s.spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite(mutations)
	})
	return err
}
