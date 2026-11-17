package credit

import (
	"context"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	DefaultOverdueSweeperInterval = 1 * time.Minute
	DefaultOverdueBatchSize       = 100
)

// StartOverdueCreditSweeper starts an autonomous background daemon that periodically
// sweeps active credit profiles, evaluates overdue grace limits, and enforces credit holds.
func StartOverdueCreditSweeper(svc *Service, interval time.Duration) {
	StartOverdueCreditSweeperContext(context.Background(), svc, interval)
}

// StartOverdueCreditSweeperContext starts the overdue credit sweeper with context cancellation and optional WaitGroup support.
func StartOverdueCreditSweeperContext(ctx context.Context, svc *Service, interval time.Duration, wg ...*sync.WaitGroup) {
	if svc == nil || svc.repo == nil {
		return
	}
	if interval <= 0 {
		interval = overdueSweeperInterval()
	}

	var waitGroup *sync.WaitGroup
	if len(wg) > 0 && wg[0] != nil {
		waitGroup = wg[0]
		waitGroup.Add(1)
	}

	go func() {
		if waitGroup != nil {
			defer waitGroup.Done()
		}
		// Initial jitter to avoid thundering herd on cluster boot
		jitter := time.Duration(rand.Int63n(int64(10 * time.Second)))
		select {
		case <-ctx.Done():
			return
		case <-time.After(jitter):
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sweepCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
				_, _ = svc.RunOverdueCreditSweeperOnce(sweepCtx)
				cancel()
			}
		}
	}()
}

// RunOverdueCreditSweeperOnce executes a full keyset-paginated pass of overdue credit evaluation and grace limit enforcement.
func (s *Service) RunOverdueCreditSweeperOnce(ctx context.Context) (int, error) {
	if s == nil || s.repo == nil {
		return 0, nil
	}
	batchSize := overdueBatchSize()
	frozenCount := 0
	tolerance := MaxDelinquencyTolerance()

	pager, hasPager := s.repo.(ProfilePager)
	cursor := ""
	for {
		var candidates []Profile
		var nextCursor string
		var err error
		if hasPager {
			candidates, nextCursor, err = pager.ListDelinquentCandidatesPage(ctx, batchSize, cursor)
		} else {
			candidates, err = s.repo.ListDelinquentCandidates(ctx, batchSize)
		}
		if err != nil {
			return frozenCount, err
		}

		for _, p := range candidates {
			if p.Status != StatusActive {
				continue
			}

			shouldFreeze := false
			reason := ""

			// Delinquency count reaches or exceeds tolerance.
			// RiskScore and RiskTier are informational telemetry only and never auto-freeze profiles.
			if p.DelinquencyCount >= tolerance {
				shouldFreeze = true
				reason = "delinquency_tolerance_exceeded"
			}

			if shouldFreeze {
				if err := s.FreezeProfile(ctx, p.RetailerID, p.SupplierID, reason); err == nil {
					frozenCount++
				}
			}
		}

		if !hasPager || nextCursor == "" || nextCursor == cursor || len(candidates) == 0 {
			break
		}
		cursor = nextCursor
	}

	return frozenCount, nil
}

func overdueSweeperInterval() time.Duration {
	if raw := strings.TrimSpace(os.Getenv("CREDIT_OVERDUE_SWEEPER_INTERVAL_SECONDS")); raw != "" {
		if sec, err := strconv.Atoi(raw); err == nil && sec > 0 {
			return time.Duration(sec) * time.Second
		}
	}
	return DefaultOverdueSweeperInterval
}

func overdueBatchSize() int {
	if raw := strings.TrimSpace(os.Getenv("CREDIT_OVERDUE_BATCH_SIZE")); raw != "" {
		if sz, err := strconv.Atoi(raw); err == nil && sz > 0 {
			return sz
		}
	}
	return DefaultOverdueBatchSize
}
