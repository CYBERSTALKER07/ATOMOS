package replenishment

import "testing"

func TestClassifyUrgencyMEI(t *testing.T) {
	if got := classifyUrgency(1.0, 2.0); got != "CRITICAL" {
		t.Fatalf("expected CRITICAL got %s", got)
	}
	if got := classifyUrgency(3.5, 2.0); got != "WARNING" {
		t.Fatalf("expected WARNING got %s", got)
	}
	if got := classifyUrgency(5.0, 2.0); got != "STABLE" {
		t.Fatalf("expected STABLE got %s", got)
	}
}

func TestTouchlessEligible(t *testing.T) {
	p := defaultPolicy("sup-1")
	if !TouchlessEligible(p, "STABLE", "PREDICTIVE_PUSH", 10, 0, 0.90) {
		t.Fatal("predictive push should be eligible")
	}
	if TouchlessEligible(p, "CRITICAL", "PREDICTIVE_PUSH", 10, 0, 0.90) {
		t.Fatal("critical should not be touchless")
	}
	if TouchlessEligible(p, "STABLE", "LOW_STOCK", 600, 0, 0.90) {
		t.Fatal("should respect daily cap")
	}
	p.MinConfidenceScore = 1.0
	if TouchlessEligible(p, "STABLE", "PREDICTIVE_PUSH", 10, 0, 0.99) {
		t.Fatal("MinConfidenceScore=1.0 must auto-approve nothing below 1.0")
	}
}
