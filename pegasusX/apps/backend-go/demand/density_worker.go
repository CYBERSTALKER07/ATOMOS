package demand

import (
	"context"
	"time"
)

// RunDensityWorker runs the event density aggregation logic periodically.
// Until a real event-source aggregation exists, the tick is a no-op (no mock
// H3 rows written to Spanner).
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
// Stubbed mock upserts were removed for prod-ready / no-mock policy.
func (s *Service) ComputeDensitySignals(ctx context.Context) error {
	_ = ctx
	return nil
}
