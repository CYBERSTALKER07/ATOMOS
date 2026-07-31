package controltower

import (
	"testing"
	"time"
)

func TestMatchRules_TypeAndSeverity(t *testing.T) {
	rules := MatchRules{Types: []string{"BUYER_REJECTED"}, Severities: []string{"HIGH"}}
	ex := Exception{Type: "BUYER_EHF_REJECTION", Severity: "HIGH", CreatedAt: time.Now()}
	if !rules.MatchesException(ex, time.Now()) {
		t.Fatal("expected match for aliased buyer reject type")
	}
	ex.Severity = "LOW"
	if rules.MatchesException(ex, time.Now()) {
		t.Fatal("expected severity mismatch")
	}
}

func TestMatchRules_AmountThreshold(t *testing.T) {
	rules := MatchRules{MinAmountMinor: 500000}
	ex := Exception{AmountMinor: 400000, CreatedAt: time.Now()}
	if rules.MatchesException(ex, time.Now()) {
		t.Fatal("expected amount below threshold to fail")
	}
	ex.AmountMinor = 600000
	if !rules.MatchesException(ex, time.Now()) {
		t.Fatal("expected amount above threshold to match")
	}
}

func TestMatchRules_SegmentAndAge(t *testing.T) {
	now := time.Now()
	rules := MatchRules{RetailerSegments: []string{"A"}, MinAgeMinutes: 10, MaxAgeMinutes: 120}
	ex := Exception{RetailerSegment: "A", CreatedAt: now.Add(-30 * time.Minute)}
	if !rules.MatchesException(ex, now) {
		t.Fatal("expected segment and age match")
	}
	ex.CreatedAt = now.Add(-5 * time.Minute)
	if rules.MatchesException(ex, now) {
		t.Fatal("expected too young exception to fail min age")
	}
}

func TestPlaybookPriorityFirstMatch(t *testing.T) {
	high := Playbook{PlaybookID: "high", Priority: 100, MatchRules: MatchRules{Types: []string{"CASH_SHORT"}}}
	low := Playbook{PlaybookID: "low", Priority: 10, MatchRules: MatchRules{Types: []string{"CASH_SHORT"}}}
	playbooks := []Playbook{low, high}
	// simulate sort
	if playbooks[0].Priority < playbooks[1].Priority {
		playbooks[0], playbooks[1] = playbooks[1], playbooks[0]
	}
	ex := Exception{Type: "CASH_SHORTFALL", CreatedAt: time.Now()}
	matched := ""
	for _, pb := range playbooks {
		if pb.MatchRules.MatchesException(ex, time.Now()) {
			matched = pb.PlaybookID
			break
		}
	}
	if matched != "high" {
		t.Fatalf("first match want high priority playbook, got %s", matched)
	}
}

func TestIsAutoSafeAction(t *testing.T) {
	if !IsAutoSafeAction("NOTIFY") {
		t.Fatal("NOTIFY should be auto-safe")
	}
	if IsAutoSafeAction("CREATE_CREDIT_NOTE") {
		t.Fatal("CREATE_CREDIT_NOTE should not be auto-safe")
	}
}
