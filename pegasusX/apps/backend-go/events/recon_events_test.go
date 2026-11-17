package events

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReconEventsContract(t *testing.T) {
	if EventReconRunCompleted != "recon.run.completed" {
		t.Fatalf("EventReconRunCompleted = %q, want %q", EventReconRunCompleted, "recon.run.completed")
	}
	if EventReconBreakOpened != "recon.break.opened" {
		t.Fatalf("EventReconBreakOpened = %q, want %q", EventReconBreakOpened, "recon.break.opened")
	}
	if EventReconBreakResolved != "recon.break.resolved" {
		t.Fatalf("EventReconBreakResolved = %q, want %q", EventReconBreakResolved, "recon.break.resolved")
	}
	if AggregateReconRun != "ReconciliationRun" {
		t.Fatalf("AggregateReconRun = %q, want %q", AggregateReconRun, "ReconciliationRun")
	}
	if AggregateReconBreak != "ReconciliationBreak" {
		t.Fatalf("AggregateReconBreak = %q, want %q", AggregateReconBreak, "ReconciliationBreak")
	}

	runEv := ReconRunCompletedEvent{
		BaseEvent: BaseEvent{
			Type:      EventReconRunCompleted,
			Version:   1,
			Timestamp: "2026-11-16T18:00:00Z",
		},
		RunID:         "run-001",
		EntityID:      "sup-1",
		PeriodID:      "2026-11",
		Mode:          "FOUR_WAY",
		Status:        "BREAKS_FOUND",
		BreakCount:    2,
		ExposureMinor: 450000,
		Currency:      "UZS",
		TriggeredBy:   "nightly-worker",
		StartedAt:     "2026-11-16T17:59:55Z",
		CompletedAt:   "2026-11-16T18:00:00Z",
	}

	rawRun, err := json.Marshal(runEv)
	if err != nil {
		t.Fatalf("marshal ReconRunCompletedEvent: %v", err)
	}
	var decRun map[string]any
	dec := json.NewDecoder(bytes.NewReader(rawRun))
	dec.UseNumber()
	if err := dec.Decode(&decRun); err != nil {
		t.Fatalf("unmarshal ReconRunCompletedEvent: %v", err)
	}
	for k, want := range map[string]any{
		"type":         EventReconRunCompleted,
		"run_id":       "run-001",
		"entity_id":    "sup-1",
		"period_id":    "2026-11",
		"mode":         "FOUR_WAY",
		"status":       "BREAKS_FOUND",
		"currency":     "UZS",
		"triggered_by": "nightly-worker",
	} {
		if decRun[k] != want {
			t.Fatalf("run[%q] = %v, want %v", k, decRun[k], want)
		}
	}
	if decRun["break_count"] != json.Number("2") || decRun["exposure_minor"] != json.Number("450000") {
		t.Fatalf("unexpected counts/minor units in run event: %v", decRun)
	}

	openedEv := ReconBreakOpenedEvent{
		BaseEvent: BaseEvent{
			Type:      EventReconBreakOpened,
			Version:   1,
			Timestamp: "2026-11-16T18:00:00Z",
		},
		BreakID:                 "brk-001",
		RunID:                   "run-001",
		EntityID:                "sup-1",
		Dimension:               "ORDERS_AR",
		ObjectType:              "Order",
		ObjectID:                "ord-123",
		ExpectedMinor:           1000000,
		ActualMinor:             750000,
		DeltaMinor:              -250000,
		Currency:                "UZS",
		Severity:                "HIGH",
		Status:                  "OPEN",
		OwnerID:                 "finops-lead",
		SlaDueAt:                "2026-11-17T18:00:00Z",
		ControlTowerExceptionID: "exc-99",
	}

	rawOpened, err := json.Marshal(openedEv)
	if err != nil {
		t.Fatalf("marshal ReconBreakOpenedEvent: %v", err)
	}
	var decOpened map[string]any
	decOp := json.NewDecoder(bytes.NewReader(rawOpened))
	decOp.UseNumber()
	if err := decOp.Decode(&decOpened); err != nil {
		t.Fatalf("unmarshal ReconBreakOpenedEvent: %v", err)
	}
	if decOpened["type"] != EventReconBreakOpened || decOpened["break_id"] != "brk-001" || decOpened["delta_minor"] != json.Number("-250000") {
		t.Fatalf("unexpected opened break payload: %v", decOpened)
	}

	resolvedEv := ReconBreakResolvedEvent{
		BaseEvent: BaseEvent{
			Type:      EventReconBreakResolved,
			Version:   1,
			Timestamp: "2026-11-16T18:10:00Z",
		},
		BreakID:        "brk-001",
		RunID:          "run-001",
		EntityID:       "sup-1",
		Dimension:      "ORDERS_AR",
		ObjectType:     "Order",
		ObjectID:       "ord-123",
		DeltaMinor:     -250000,
		Currency:       "UZS",
		Status:         "RESOLVED",
		ResolvedBy:     "admin-1",
		ResolutionNote: "matched manual adjustment",
	}

	rawResolved, err := json.Marshal(resolvedEv)
	if err != nil {
		t.Fatalf("marshal ReconBreakResolvedEvent: %v", err)
	}
	var decResolved map[string]any
	if err := json.Unmarshal(rawResolved, &decResolved); err != nil {
		t.Fatalf("unmarshal ReconBreakResolvedEvent: %v", err)
	}
	if decResolved["type"] != EventReconBreakResolved || decResolved["status"] != "RESOLVED" || decResolved["resolved_by"] != "admin-1" {
		t.Fatalf("unexpected resolved break payload: %v", decResolved)
	}
	if _, hasCT := decResolved["control_tower_exception_id"]; hasCT {
		t.Fatalf("expected empty control_tower_exception_id omitted, got %v", decResolved["control_tower_exception_id"])
	}

	envelopePath := filepath.Clean(filepath.Join("..", "..", "..", "packages", "types", "src", "envelope.ts"))
	envelopeSrc, err := os.ReadFile(envelopePath)
	if err != nil {
		t.Fatalf("read %s: %v", envelopePath, err)
	}
	for _, ev := range []string{`"recon.run.completed"`, `"recon.break.opened"`, `"recon.break.resolved"`} {
		if !strings.Contains(string(envelopeSrc), ev) {
			t.Fatalf("%s missing %s in EventType union", envelopePath, ev)
		}
	}
}
