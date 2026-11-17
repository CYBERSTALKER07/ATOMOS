package credit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// Credit scoring v1/v2 — explainable scorecard + calibrated PD + honest thin-file baseline.
//
// Score is 0–100 (higher = healthier). Weights sum to 10000 bps:
//
//	utilization   3500 bps  — open balance / limit (0 util → 100)
//	delinquency   2500 bps  — DelinquencyCount steps (each −15, floor 0)
//	DPD           2500 bps  — max days past due on open AR (or 0 if no AR feed)
//	pay_velocity  1500 bps  — cleared reservations / recent activity proxy
//
// Flag: CREDIT_SCORING_ENABLED (default true).

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

// ScoreAutoHoldEnabled always returns false: risk score and RiskTier are purely
// informational telemetry and never block orders or auto-freeze credit profiles.
func ScoreAutoHoldEnabled() bool {
	return false
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

// ScoreMetrics are optional AR/payment/program inputs for scoring (fail-soft to profile-only).
type ScoreMetrics struct {
	MaxDaysPastDue        int64 // open AR max DPD (undisputed)
	PaymentsLast90d       int64 // successful pay-downs / clears
	ExpectedPayments90    int64 // expected cadence proxy (orders or invoice count)
	StarterLimitMinor     int64 // supplier program or terms starter limit
	PaidVolume90dMinor    int64 // settled AR / cleared orders over last 90d
	AvgOrderMinor         int64 // average historical order amount
	OrderCount90d         int64 // non-cancelled orders in last 90d
	DisputedInvoicesCount int64 // supplier-specific disputed open invoices
	DisputedAmountMinor   int64 // disputed open AR amount
	TenureDays            int64 // days since relationship/retailer first activity
}

// ComputeRiskScore derives RetailerCreditScore + risk tier + reason codes from profile + metrics.
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
	suggested, limitSource := suggestedLimitWithReason(p, m, score)
	reasons := buildScorecardReasonCodes(p, m, utilComp, delinqComp, dpdComp, velComp, limitSource, suggested)
	reasonsJSON, _ := json.Marshal(reasons)

	factors, _ := json.Marshal(map[string]any{
		"version":                 "credit_v2",
		"weights":                 map[string]int{"utilization_bps": weightUtilBps, "delinquency_bps": weightDelinquency, "dpd_bps": weightDPD, "pay_velocity_bps": weightPayVelocity},
		"utilization_score":       utilComp,
		"delinquency_score":       delinqComp,
		"dpd_score":               dpdComp,
		"pay_velocity_score":      velComp,
		"max_dpd":                 m.MaxDaysPastDue,
		"payments_90d":            m.PaymentsLast90d,
		"expected_90d":            m.ExpectedPayments90,
		"balance_minor":           p.CurrentBalanceMinor,
		"limit_minor":             p.CreditLimitMinor,
		"starter_limit_minor":     m.StarterLimitMinor,
		"paid_volume_90d_minor":   m.PaidVolume90dMinor,
		"avg_order_minor":         m.AvgOrderMinor,
		"order_count_90d":         m.OrderCount90d,
		"disputed_invoices_count": m.DisputedInvoicesCount,
		"limit_baseline_source":   limitSource,
		"delinquency_count":       p.DelinquencyCount,
		"status":                  p.Status,
		"reason_codes":            reasons,
	})
	_ = reasonsJSON
	return RetailerCreditScore{
		RetailerID:          p.RetailerID,
		Score:               score,
		RiskTier:            tier,
		SuggestedLimitMinor: suggested,
		FactorsJSON:         string(factors),
		ReasonCodes:         reasons,
		ReasonCodesJSON:     string(reasonsJSON),
		PdBps:               calibratedPdBpsFromScore(score, p.Status),
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
	exposure := p.CurrentBalanceMinor + p.ReservedMinor
	if exposure <= 0 {
		return 100
	}
	utilPct := (exposure * 100) / p.CreditLimitMinor
	s := 100 - utilPct
	if s < 0 {
		return 0
	}
	if s > 100 {
		return 100
	}
	return s
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
	// ratio capped at 12000 bps (1.2x), mapped to 0..100
	ratioBps := (payments * 10000) / expected
	if ratioBps > 12000 {
		ratioBps = 12000
	}
	if ratioBps < 0 {
		ratioBps = 0
	}
	s := (ratioBps * 100) / 12000
	if s > 100 {
		return 100
	}
	return s
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

// resolveBaselineLimitMinor derives an honest credit limit baseline from profile,
// program starter limits, or observed 90d paid AR/order volume (never hardcoded 1_000_000).
func resolveBaselineLimitMinor(p Profile, m ScoreMetrics) (int64, string) {
	if p.CreditLimitMinor > 0 {
		return p.CreditLimitMinor, "EXISTING_PROFILE_LIMIT"
	}
	if m.StarterLimitMinor > 0 {
		return m.StarterLimitMinor, "PROGRAM_STARTER_LIMIT"
	}
	if m.PaidVolume90dMinor > 0 {
		base := m.PaidVolume90dMinor / 3
		if m.AvgOrderMinor > 0 && m.AvgOrderMinor*2 > base {
			base = m.AvgOrderMinor * 2
		}
		if base > 0 {
			return base, "OBSERVED_PAID_VOLUME"
		}
	}
	if m.AvgOrderMinor > 0 && m.OrderCount90d >= 2 {
		return m.AvgOrderMinor, "OBSERVED_ORDER_HISTORY"
	}
	return 0, "THIN_FILE"
}

func suggestedLimitWithReason(p Profile, m ScoreMetrics, score int64) (int64, string) {
	base, source := resolveBaselineLimitMinor(p, m)
	if score < 25 || p.Status == StatusBlacklisted {
		return 0, source
	}
	if base <= 0 {
		return 0, source
	}
	scaled := (base * score) / 100
	if score >= 75 && scaled < base {
		scaled = base
	}
	return scaled, source
}

func buildScorecardReasonCodes(
	p Profile,
	m ScoreMetrics,
	utilComp, delinqComp, dpdComp, velComp int64,
	limitSource string,
	suggestedLimit int64,
) []ReasonCode {
	reasons := make([]ReasonCode, 0, 8)
	if p.Status == StatusBlacklisted {
		reasons = append(reasons, ReasonCode{
			Code: "STATUS_BLACKLISTED", Label: "Account blacklisted", ImpactBps: -10000,
			Detail: "status=BLACKLISTED", Source: "SCORECARD",
		})
	} else if p.Status == StatusFrozen {
		reasons = append(reasons, ReasonCode{
			Code: "STATUS_FROZEN", Label: "Credit profile on hold", ImpactBps: -3000,
			Detail: "status=FROZEN", Source: "SCORECARD",
		})
	}
	if p.PrepayOnlyLocked {
		reasons = append(reasons, ReasonCode{
			Code: "PREPAY_ONLY_LOCKED", Label: "Prepay-only lock active", ImpactBps: -2000,
			Detail: "prepay_only_locked=true", Source: "SCORECARD",
		})
	}
	switch limitSource {
	case "THIN_FILE":
		reasons = append(reasons, ReasonCode{
			Code: "THIN_FILE_NO_HISTORY_OR_PROGRAM_LIMIT", Label: "Thin credit file (no prior limit or paid AR volume)", ImpactBps: 0,
			Detail: fmt.Sprintf("suggested_limit_minor=%d", suggestedLimit), Source: "SCORECARD",
		})
	case "PROGRAM_STARTER_LIMIT":
		reasons = append(reasons, ReasonCode{
			Code: "PROGRAM_STARTER_LIMIT", Label: "Baseline anchored to supplier credit program starter limit", ImpactBps: 500,
			Detail: fmt.Sprintf("starter_limit_minor=%d", m.StarterLimitMinor), Source: "SCORECARD",
		})
	case "OBSERVED_PAID_VOLUME", "OBSERVED_ORDER_HISTORY":
		reasons = append(reasons, ReasonCode{
			Code: limitSource, Label: "Baseline derived from observed 90-day settled order/AR history", ImpactBps: 1000,
			Detail: fmt.Sprintf("paid_90d_minor=%d orders_90d=%d", m.PaidVolume90dMinor, m.OrderCount90d), Source: "SCORECARD",
		})
	}
	if p.CreditLimitMinor > 0 {
		utilBps := ((p.CurrentBalanceMinor + p.ReservedMinor) * 10000) / p.CreditLimitMinor
		if utilComp < 40 {
			reasons = append(reasons, ReasonCode{
				Code: "UTILIZATION_HIGH", Label: "High credit limit utilization", ImpactBps: -int64((100 - utilComp) * 35),
				Detail: fmt.Sprintf("utilization_bps=%d", utilBps), Source: "SCORECARD",
			})
		} else if utilComp >= 80 {
			reasons = append(reasons, ReasonCode{
				Code: "UTILIZATION_HEALTHY", Label: "Low credit limit utilization", ImpactBps: int64(utilComp * 35),
				Detail: fmt.Sprintf("utilization_bps=%d", utilBps), Source: "SCORECARD",
			})
		}
	}
	if p.DelinquencyCount > 0 {
		reasons = append(reasons, ReasonCode{
			Code: "DELINQUENCY_HISTORY", Label: "Prior delinquency events recorded", ImpactBps: -int64(p.DelinquencyCount * 1500),
			Detail: fmt.Sprintf("delinquency_count=%d component=%d", p.DelinquencyCount, delinqComp), Source: "SCORECARD",
		})
	}
	if m.MaxDaysPastDue > 0 {
		code := "DPD_MODERATE"
		if m.MaxDaysPastDue > 14 {
			code = "DPD_SEVERE"
		}
		reasons = append(reasons, ReasonCode{
			Code: code, Label: "Open overdue receivables detected", ImpactBps: -int64((100 - dpdComp) * 25),
			Detail: fmt.Sprintf("max_dpd_days=%d", m.MaxDaysPastDue), Source: "SCORECARD",
		})
	}
	if m.ExpectedPayments90 > 0 {
		if velComp >= 75 {
			reasons = append(reasons, ReasonCode{
				Code: "PAY_VELOCITY_STRONG", Label: "Consistent on-time repayment cadence", ImpactBps: int64(velComp * 15),
				Detail: fmt.Sprintf("payments_90d=%d expected_90d=%d", m.PaymentsLast90d, m.ExpectedPayments90), Source: "SCORECARD",
			})
		} else {
			reasons = append(reasons, ReasonCode{
				Code: "PAY_VELOCITY_WEAK", Label: "Slow repayment velocity vs expected order cadence", ImpactBps: -int64((100 - velComp) * 15),
				Detail: fmt.Sprintf("payments_90d=%d expected_90d=%d", m.PaymentsLast90d, m.ExpectedPayments90), Source: "SCORECARD",
			})
		}
	}
	if m.DisputedInvoicesCount > 0 {
		reasons = append(reasons, ReasonCode{
			Code: "DISPUTED_INVOICES_PRESENT", Label: "Supplier-specific disputed invoices tracked separately", ImpactBps: 0,
			Detail: fmt.Sprintf("disputed_count=%d disputed_minor=%d", m.DisputedInvoicesCount, m.DisputedAmountMinor), Source: "SCORECARD",
		})
	}
	return reasons
}

// calibratedPdBpsFromScore converts 0..100 health score to calibrated annual PD in basis points.
func calibratedPdBpsFromScore(score int64, status Status) int64 {
	if status == StatusBlacklisted || score <= 0 {
		return 9500
	}
	switch {
	case score >= 90:
		return 100
	case score >= 75:
		return 400 - ((score-75)*300)/15
	case score >= 50:
		return 1500 - ((score-50)*1100)/25
	case score >= 25:
		return 4500 - ((score-25)*3000)/25
	default:
		return 9500 - (score*5000)/25
	}
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
