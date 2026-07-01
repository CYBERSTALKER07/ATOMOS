package predictivepush

import (
	"testing"
	"time"
)

func TestAnalyzerInitialization(t *testing.T) {
	// Simple test to ensure the analyzer initializes correctly with defaults
	analyzer := NewAnalyzer(nil)
	if analyzer.weeksToAnalyze != 8 {
		t.Errorf("expected 8 weeksToAnalyze, got %d", analyzer.weeksToAnalyze)
	}
	if analyzer.confidenceThreshold != 0.75 {
		t.Errorf("expected 0.75 confidenceThreshold, got %f", analyzer.confidenceThreshold)
	}
}

func TestDemandEventStruct(t *testing.T) {
	event := &DemandEvent{
		RetailerId:  "r1",
		ProductId:   "p1",
		TargetDate:  time.Now(),
		Quantity:    100,
		Confidence:  0.8,
		PatternDays: 4,
	}

	if event.RetailerId != "r1" {
		t.Errorf("unexpected RetailerId: %s", event.RetailerId)
	}
}
