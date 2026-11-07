package credit

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"strings"
	"time"
)

// Credit scoring v1 (G3.B) — FieldAssist-class collections signals, not ML.
//
// Score is 0–100 (higher = healthier). Weights sum to 1.0:
//
//	utilization   0.35  — open balance / limit (0 util → 100)
//	delinquency   0.25  — DelinquencyCount steps (each −15, floor 0)
//	DPD           0.25  — max days past due on open AR (or 0 if no AR feed)
//	pay_velocity  0.15  — cleared reservations / recent activity proxy
//
// Flag: CREDIT_SCORING_ENABLED (default true). Auto-hold when score ≤ threshold
// and CREDIT_SCORE_AUTO_HOLD_ENABLED=true (threshold default 25).

const (
	weightUtilBps      = 3500 // /10000
	weightDelinquency  = 2500
	weightDPD          = 2500
	weightPayVelocity  = 1500
	defaultAutoHoldMax = int64(25)
)

// ScoringEnabled reports whether risk scoring product is active.
func ScoringEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("CREDIT_SCORING_ENABLED")))
	if v == "" {
		return true // G3 default-on algorithm (honest numbers from profile/AR)
	}
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// ScoreAutoHoldEnabled gates auto FROZEN when score is critically low.
func ScoreAutoHoldEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("CREDIT_SCORE_AUTO_HOLD_ENABLED")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// ScoreAutoHoldMax is the inclusive max score that triggers auto-hold when enabled.
func ScoreAutoHoldMax() int64 {
	raw := strings.TrimSpace(os.Getenv("CREDIT_SCORE_AUTO_HOLD_MAX"))
	if raw == "" {
		return defaultAutoHoldMax
	}
	var n int64
	for _, c := range raw {
		if c < '0' || c > '9' {
			return defaultAutoHoldMax
		}
		n = n*10 + int64(c-'0')
	}
	return n
}

// ScoreMetrics are optional AR/payment inputs for scoring (fail-soft to profile-only).
type ScoreMetrics struct {
	MaxDaysPastDue     int64 // open AR max DPD
	PaymentsLast90d    int64 // successful pay-downs / clears
	ExpectedPayments90 int64 // expected cadence proxy (orders or invoice count)
}

// ComputeRiskScore derives RetailerCreditScore + risk tier from profile + metrics.
func ComputeRiskScore(p Profile, m ScoreMetrics, now time.Time) RetailerCreditScore {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	utilComp := utilComponent(p)
	delinqComp := delinquencyComponent(p.DelinquencyCount)
	dpdComp := dpdComponent(m.MaxDaysPastDue)
	velComp := velocityComponent(m.PaymentsLast90d, m.ExpectedPayments90, p)

	score := (utilComp*weightUtilBps + delinqComp*weightDelinquency + dpdComp*weightDPD + velComp*weightPayVelocity) / 10000
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	// Status hard floors
	switch p.Status {
	case StatusBlacklisted:
		score = 0
	case StatusFrozen:
		if score > 30 {
			score = 30
		}
	case StatusClosed:
		if score > 40 {
			score = 40
		}
	}

	tier := tierFromScore(score, p.Status)
	factors, _ := json.Marshal(map[string]any{
		"version":            "g3_v1",
		"weights":            map[string]int{"utilization_bps": weightUtilBps, "delinquency_bps": weightDelinquency, "dpd_bps": weightDPD, "pay_velocity_bps": weightPayVelocity},
		"utilization_score":  utilComp,
		"delinquency_score":  delinqComp,
		"dpd_score":          dpdComp,
		"pay_velocity_score": velComp,
		"max_dpd":            m.MaxDaysPastDue,
		"payments_90d":       m.PaymentsLast90d,
		"expected_90d":       m.ExpectedPayments90,
		"balance_minor":      p.CurrentBalanceMinor,
		"limit_minor":        p.CreditLimitMinor,
		"delinquency_count":  p.DelinquencyCount,
		"status":             p.Status,
	})
	suggested := suggestedLimit(p, score)
	return RetailerCreditScore{
		RetailerID:          p.RetailerID,
		Score:               score,
		RiskTier:            tier,
		SuggestedLimitMinor: suggested,
		FactorsJSON:         string(factors),
		WindowStart:         now.Add(-90 * 24 * time.Hour),
		WindowEnd:           now,
		ComputedAt:          now,
	}
}

func utilComponent(p Profile) int64 {
	if p.CreditLimitMinor <= 0 {
		if p.CurrentBalanceMinor > 0 {
			return 20
		}
		return 70
	}
	// utilization 0 → 100, 100% → 0, over-limit → 0
	util := float64(p.CurrentBalanceMinor+p.ReservedMinor) / float64(p.CreditLimitMinor)
	if util < 0 {
		util = 0
	}
	s := 100.0 - util*100.0
	if s < 0 {
		s = 0
	}
	if s > 100 {
		s = 100
	}
	return int64(math.Round(s))
}

func delinquencyComponent(count int64) int64 {
	s := 100 - count*15
	if s < 0 {
		return 0
	}
	return s
}

func dpdComponent(maxDPD int64) int64 {
	switch {
	case maxDPD <= 0:
		return 100
	case maxDPD <= 3:
		return 90
	case maxDPD <= 7:
		return 75
	case maxDPD <= 14:
		return 55
	case maxDPD <= 21:
		return 35
	case maxDPD <= 30:
		return 15
	default:
		return 0
	}
}

func velocityComponent(payments, expected int64, p Profile) int64 {
	if expected <= 0 {
		// No cadence data: neutral-high if no delinquency, else mid.
		if p.DelinquencyCount == 0 && p.CurrentBalanceMinor == 0 {
			return 85
		}
		if p.DelinquencyCount == 0 {
			return 60
		}
		return 35
	}
	ratio := float64(payments) / float64(expected)
	if ratio > 1.2 {
		ratio = 1.2
	}
	s := ratio * 100.0 / 1.2
	if s > 100 {
		s = 100
	}
	return int64(math.Round(s))
}

func tierFromScore(score int64, status Status) RiskTier {
	if status == StatusBlacklisted {
		return RiskTierBlock
	}
	switch {
	case score >= 75:
		return RiskTierLow
	case score >= 50:
		return RiskTierMedium
	case score >= 25:
		return RiskTierHigh
	default:
		return RiskTierBlock
	}
}

func suggestedLimit(p Profile, score int64) int64 {
	base := p.CreditLimitMinor
	if base <= 0 {
		base = 1_000_000 // 10k UZS units placeholder minor — desk treats as suggestion only
	}
	// Scale limit by score/100 with floors.
	scaled := base * score / 100
	if score >= 75 && scaled < base {
		scaled = base
	}
	if score < 25 {
		return 0
	}
	return scaled
}

// ApplyScoreToProfile stamps RiskScore / RiskTier / LastEvaluatedAt from a computed score.
func ApplyScoreToProfile(p *Profile, sc RetailerCreditScore) {
	if p == nil {
		return
	}
	p.RiskScore = sc.Score
	p.RiskTier = sc.RiskTier
	p.LastEvaluatedAt = sc.ComputedAt
}

// ScoreMetricsProvider loads AR/payment metrics for scoring (optional).
type ScoreMetricsProvider interface {
	LoadScoreMetrics(ctx context.Context, retailerID, supplierID string) (ScoreMetrics, error)
}

// EvaluateProfileScore computes score for one profile using optional metrics provider.
func (s *Service) EvaluateProfileScore(ctx context.Context, p Profile) (RetailerCreditScore, error) {
	if !ScoringEnabled() {
		return RetailerCreditScore{RetailerID: p.RetailerID, Score: p.RiskScore, RiskTier: p.RiskTier, ComputedAt: time.Now().UTC()}, nil
	}
	var m ScoreMetrics
	if s != nil && s.scoreMetrics != nil {
		loaded, err := s.scoreMetrics.LoadScoreMetrics(ctx, p.RetailerID, p.SupplierID)
		if err == nil {
			m = loaded
		}
	}
	now := time.Now().UTC()
	if s != nil && s.now != nil {
		now = s.now()
	}
	return ComputeRiskScore(p, m, now), nil
}

// SetScoreMetricsProvider wires AR/payment metrics for scoring.
func (s *Service) SetScoreMetricsProvider(p ScoreMetricsProvider) {
	if s == nil {
		return
	}
	s.scoreMetrics = p
}
