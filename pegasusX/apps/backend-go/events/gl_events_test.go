package events

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGLEntryEventsContract(t *testing.T) {
	if EventGLEntryPosted != "gl.entry.posted" {
		t.Fatalf("EventGLEntryPosted = %q, want %q", EventGLEntryPosted, "gl.entry.posted")
	}
	if EventGLEntryReversed != "gl.entry.reversed" {
		t.Fatalf("EventGLEntryReversed = %q, want %q", EventGLEntryReversed, "gl.entry.reversed")
	}
	if AggregateJournalEntry != "JournalEntry" {
		t.Fatalf("AggregateJournalEntry = %q, want %q", AggregateJournalEntry, "JournalEntry")
	}

	posted := GLEntryPostedEvent{
		BaseEvent: BaseEvent{
			Type:      EventGLEntryPosted,
			Version:   1,
			Timestamp: "2026-11-16T16:00:00Z",
		},
		EntryID:          "je-001",
		EntityID:         "sup-1",
		PeriodID:         "2026-11",
		SourceType:       "ORDER_DELIVERED",
		SourceID:         "ord-1",
		Currency:         "UZS",
		TotalDebitMinor:  150000,
		TotalCreditMinor: 150000,
		Hash:             strings.Repeat("a", 64),
	}

	rawPosted, err := json.Marshal(posted)
	if err != nil {
		t.Fatalf("marshal GLEntryPostedEvent: %v", err)
	}
	var decPosted map[string]any
	if err := json.Unmarshal(rawPosted, &decPosted); err != nil {
		t.Fatalf("unmarshal GLEntryPostedEvent: %v", err)
	}
	for k, want := range map[string]any{
		"type":        EventGLEntryPosted,
		"entry_id":    "je-001",
		"entity_id":   "sup-1",
		"period_id":   "2026-11",
		"source_type": "ORDER_DELIVERED",
		"source_id":   "ord-1",
		"currency":    "UZS",
		"hash":        strings.Repeat("a", 64),
	} {
		if decPosted[k] != want {
			t.Fatalf("posted[%q] = %v, want %v", k, decPosted[k], want)
		}
	}
	if decPosted["total_debit_minor"] != float64(150000) || decPosted["total_credit_minor"] != float64(150000) {
		t.Fatalf("unexpected totals in posted event: %v", decPosted)
	}
	if _, hasRev := decPosted["reverses_entry_id"]; hasRev {
		t.Fatalf("expected reverses_entry_id omitted when empty on posted event, got %v", decPosted["reverses_entry_id"])
	}

	reversed := GLEntryReversedEvent{
		BaseEvent: BaseEvent{
			Type:      EventGLEntryReversed,
			Version:   1,
			Timestamp: "2026-11-16T16:05:00Z",
		},
		EntryID:          "je-002",
		EntityID:         "sup-1",
		PeriodID:         "2026-11",
		SourceType:       "REVERSAL",
		SourceID:         "je-001",
		Currency:         "UZS",
		TotalDebitMinor:  150000,
		TotalCreditMinor: 150000,
		Hash:             strings.Repeat("b", 64),
		ReversesEntryID:  "je-001",
	}

	rawReversed, err := json.Marshal(reversed)
	if err != nil {
		t.Fatalf("marshal GLEntryReversedEvent: %v", err)
	}
	var decReversed map[string]any
	if err := json.Unmarshal(rawReversed, &decReversed); err != nil {
		t.Fatalf("unmarshal GLEntryReversedEvent: %v", err)
	}
	if decReversed["type"] != EventGLEntryReversed || decReversed["reverses_entry_id"] != "je-001" {
		t.Fatalf("unexpected reversed event payload: %v", decReversed)
	}

	envelopePath := filepath.Clean(filepath.Join("..", "..", "..", "packages", "types", "src", "envelope.ts"))
	envelopeSrc, err := os.ReadFile(envelopePath)
	if err != nil {
		t.Fatalf("read %s: %v", envelopePath, err)
	}
	for _, ev := range []string{`"gl.entry.posted"`, `"gl.entry.reversed"`} {
		if !strings.Contains(string(envelopeSrc), ev) {
			t.Fatalf("%s missing %s in EventType union", envelopePath, ev)
		}
	}
}
