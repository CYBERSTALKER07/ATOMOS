package globalproducts

import (
	"context"
	"time"
)

// StartMatchWorker periodically reprocesses PENDING fuzzy queue items that lack
// a candidate (noop for review-only rows). Exact matching is synchronous on write;
// this worker is a soak/retry hook for future auto-resolution heuristics.
func (s *Service) StartMatchWorker(ctx context.Context, interval time.Duration) {
	if !Enabled() || s.repo == nil {
		return
	}
	if interval <= 0 {
		interval = 2 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			items, err := s.repo.ListMatchQueue(ctx, StatusPending, 50)
			if err != nil {
				s.log.ErrorContext(ctx, "globalproducts match worker list failed", "err", err)
				continue
			}
			for _, item := range items {
				if item.CandidateGlobalProductID == "" {
					continue
				}
				// Leave ambiguous items for human resolve; log depth for ops.
				s.log.InfoContext(ctx, "globalproducts match queue pending",
					"queue_id", item.QueueID, "product_id", item.ProductID, "score", item.Score)
			}
		}
	}
}
