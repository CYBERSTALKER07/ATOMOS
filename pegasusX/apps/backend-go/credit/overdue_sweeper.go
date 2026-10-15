package credit

import (
	"context"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultOverdueSweeperInterval = 1 * time.Minute
	DefaultOverdueBatchSize       = 100
)

// StartOverdueCreditSweeper starts an autonomous background daemon that periodically
// sweeps active credit profiles, evaluates overdue grace limits, and enforces credit holds.
func StartOverdueCreditSweeper(svc *Service, interval time.Duration) {
	if svc == nil || svc.repo == nil {
		return
	}
	if interval <= 0 {
		interval = overdueSweeperInterval()
	}

	go func() {
		// Initial jitter to avoid thundering herd on cluster boot
		jitter := time.Duration(rand.Int63n(int64(10 * time.Second)))
		time.Sleep(jitter)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			_, _ = svc.RunOverdueCreditSweeperOnce(ctx)
			cancel()
		}
	}()
}

// RunOverdueCreditSweeperOnce executes a single pass of overdue credit evaluation and grace limit enforcement.
func (s *Service) RunOverdueCreditSweeperOnce(ctx context.Context) (int, error) {
	if s == nil || s.repo == nil {
		return 0, nil
	}
	batchSize := overdueBatchSize()
	candidates, err := s.repo.ListDelinquentCandidates(ctx, batchSize)
	if err != nil {
		return 0, err
	}

	frozenCount := 0
	tolerance := MaxDelinquencyTolerance()
	autoHoldMax := ScoreAutoHoldMax()
	autoHoldEnabled := ScoreAutoHoldEnabled()

	for _, p := range candidates {
		if p.Status != StatusActive {
			continue
		}

		shouldFreeze := false
		reason := ""

		// 1) Delinquency count reaches or exceeds tolerance
		if p.DelinquencyCount >= tolerance {
			shouldFreeze = true
			reason = "delinquency_tolerance_exceeded"
		}

		// 2) Critical risk score (or RiskTierBlock) auto-hold
		if !shouldFreeze && autoHoldEnabled && p.RiskScore > 0 && p.RiskScore <= autoHoldMax {
			shouldFreeze = true
			reason = "critical_risk_score_auto_hold"
		}
		if !shouldFreeze && p.RiskTier == RiskTierBlock {
			shouldFreeze = true
			reason = "risk_tier_block_auto_hold"
		}

		// 3) Evaluated score under live scoring
		if !shouldFreeze && autoHoldEnabled && ScoringEnabled() {
			sc, scErr := s.EvaluateProfileScore(ctx, p)
			if scErr == nil {
				if sc.RiskTier == RiskTierBlock || (sc.Score > 0 && sc.Score <= autoHoldMax) {
					shouldFreeze = true
					reason = "evaluated_risk_tier_block"
				}
			}
		}

		if shouldFreeze {
			if err := s.FreezeProfile(ctx, p.RetailerID, p.SupplierID, reason); err == nil {
				frozenCount++
			}
		}
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
