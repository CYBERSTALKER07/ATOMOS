package controltower

import (
	"testing"
	"time"
)

func TestComputeExceptionScore_ordering(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	high := Exception{
		Severity:        "HIGH",
		AmountMinor:     5_000_000,
		CreatedAt:       now.Add(-2 * time.Hour),
		RetailerSegment: "A",
	}
	low := Exception{
		Severity:    "LOW",
		AmountMinor: 100_000,
		CreatedAt:   now.Add(-10 * time.Minute),
	}
	highScore, _, _ := ComputeExceptionScore(high, now)
	lowScore, _, _ := ComputeExceptionScore(low, now)
	if highScore <= lowScore {
		t.Fatalf("expected high severity/large amount to score higher: high=%d low=%d", highScore, lowScore)
	}
}

func TestComputeExceptionScore_segmentBoost(t *testing.T) {
	now := time.Now()
	base := Exception{Severity: "MEDIUM", AmountMinor: 1_000_000, CreatedAt: now.Add(-30 * time.Minute)}
	segmentA := base
	segmentA.RetailerSegment = "A"
	scoreBase, _, _ := ComputeExceptionScore(base, now)
	scoreA, _, _ := ComputeExceptionScore(segmentA, now)
	if scoreA <= scoreBase {
		t.Fatalf("segment A should boost score: base=%d a=%d", scoreBase, scoreA)
	}
}

func TestComputeExceptionScore_freshCriticalOutscores48hLow(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	// Fresh CRITICAL exception (0 age, 0 amount, no segment boost)
	freshCritical := Exception{
		Severity:    "CRITICAL",
		AmountMinor: 0,
		CreatedAt:   now,
	}
	// 48-hour old LOW exception with max capped amount (10M tiyin) and Segment A boost
	oldMaxLow := Exception{
		Severity:        "LOW",
		AmountMinor:     scoreAmountCapMinor,
		CreatedAt:       now.Add(-48 * time.Hour),
		RetailerSegment: "A",
	}

	criticalScore, _, _ := ComputeExceptionScore(freshCritical, now)
	lowScore, _, _ := ComputeExceptionScore(oldMaxLow, now)

	if criticalScore <= lowScore {
		t.Fatalf("fresh CRITICAL exception (%d) must outscore 48h LOW exception with max amount and segment boost (%d)", criticalScore, lowScore)
	}
}

func TestMatchPlaybooks_priorityOrder(t *testing.T) {
	now := time.Now()
	ex := Exception{Type: "FISCAL_FAILED", Severity: "HIGH", CreatedAt: now.Add(-1 * time.Hour)}
	playbooks := []Playbook{
		{PlaybookID: "low", Name: "Low", Priority: 10, IsActive: true, MatchRules: MatchRules{Types: []string{"FISCAL_FAILED"}}},
		{PlaybookID: "high", Name: "High", Priority: 90, IsActive: true, MatchRules: MatchRules{Types: []string{"FISCAL_FAILED"}}},
	}
	matched := MatchPlaybooks(ex, playbooks, now)
	if len(matched) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matched))
	}
	if matched[0].PlaybookID != "high" {
		t.Fatalf("expected high priority first, got %s", matched[0].PlaybookID)
	}
}
