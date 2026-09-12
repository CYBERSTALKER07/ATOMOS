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

func TestSelectTransfersUnderCapital(t *testing.T) {
	cands := []meiTransferCandidate{
		{skuID: "warn", qty: 10, unitValueMinor: 1000, receiverDaysCover: 3, urgency: "WARNING"},     // 10000
		{skuID: "crit", qty: 5, unitValueMinor: 1000, receiverDaysCover: 1, urgency: "CRITICAL"},     // 5000
		{skuID: "crit2", qty: 10, unitValueMinor: 1000, receiverDaysCover: 0.5, urgency: "CRITICAL"}, // 10000
	}
	accepted, used, skipped := selectTransfersUnderCapital(cands, 15000)
	if len(accepted) != 2 || skipped != 1 || used != 15000 {
		t.Fatalf("accepted=%d used=%d skipped=%d", len(accepted), used, skipped)
	}
	if accepted[0].skuID != "crit2" || accepted[1].skuID != "crit" {
		t.Fatalf("order want crit2,crit got %s,%s", accepted[0].skuID, accepted[1].skuID)
	}

	accepted, used, skipped = selectTransfersUnderCapital(cands, 0)
	if len(accepted) != 3 || skipped != 0 || used != 25000 {
		t.Fatalf("unlimited: accepted=%d used=%d skipped=%d", len(accepted), used, skipped)
	}
}

func TestSelectTransfersCostAware_PrefersShortHaul(t *testing.T) {
	// Same CRITICAL urgency; short-haul should win under capital that only funds one transfer.
	cands := []meiTransferCandidate{
		{skuID: "long", qty: 5, unitValueMinor: 1000, urgency: "CRITICAL", transportCostKm: 200, receiverDaysCover: 1},
		{skuID: "short", qty: 5, unitValueMinor: 1000, urgency: "CRITICAL", transportCostKm: 10, receiverDaysCover: 1},
		{skuID: "mid", qty: 5, unitValueMinor: 1000, urgency: "WARNING", transportCostKm: 5, receiverDaysCover: 3},
	}
	accepted, used, skipped := selectTransfersCostAware(cands, 5000)
	if len(accepted) != 1 || skipped != 2 || used != 5000 {
		t.Fatalf("accepted=%d used=%d skipped=%d", len(accepted), used, skipped)
	}
	if accepted[0].skuID != "short" {
		t.Fatalf("want short-haul under capital bind, got %s (solver=%s)", accepted[0].skuID, MEIOSolverCostAwareV2)
	}
	// Unlimited: short (crit) before long (crit) before mid (warn) by bang-for-buck.
	accepted, _, _ = selectTransfersCostAware(cands, 0)
	if len(accepted) != 3 {
		t.Fatalf("unlimited len=%d", len(accepted))
	}
	if accepted[0].skuID != "short" {
		t.Fatalf("first want short got %s", accepted[0].skuID)
	}
}

func TestUrgencyWeightAndTransport(t *testing.T) {
	if urgencyWeight("CRITICAL") <= urgencyWeight("WARNING") {
		t.Fatal("CRITICAL weight must exceed WARNING")
	}
	if transportCostMinor(10) <= transportCostMinor(0) {
		t.Fatal("longer haul costs more")
	}
	if MEIOSolverCostAwareV2 != "cost_aware_v2" {
		t.Fatal("honesty label")
	}
}
