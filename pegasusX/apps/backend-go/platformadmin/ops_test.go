package platformadmin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOutboxSummaryWire_DeadLetterCountOmittedWhenUnavailable(t *testing.T) {
	m := outboxSummaryWire(0, false, 0, false, 0, "")
	if _, ok := m["dead_letter_count"]; ok {
		t.Fatalf("unavailable must not emit dead_letter_count=0: %+v", m)
	}
	if _, ok := m["unpublished_count"]; ok {
		t.Fatalf("unpublished unavailable must not emit unpublished_count=0: %+v", m)
	}
	if m["dead_letter_available"] != false || m["unpublished_available"] != false {
		t.Fatalf("available flags: %+v", m)
	}
}

func TestOutboxSummaryWire_ZeroIsRealEmpty(t *testing.T) {
	m := outboxSummaryWire(3, true, 0, true, 10, "2026-08-16T00:00:00Z")
	if m["dead_letter_count"] != int64(0) {
		t.Fatalf("empty table must report 0, got %+v", m["dead_letter_count"])
	}
	if m["unpublished_count"] != int64(3) {
		t.Fatalf("unpublished_count=%v", m["unpublished_count"])
	}
	if m["dead_letter_available"] != true {
		t.Fatal("zero must still be available")
	}
}

func TestDeadLettersListWire_PageCountIsNotTableCount(t *testing.T) {
	items := []map[string]any{{"event_id": "a"}, {"event_id": "b"}}
	m := deadLettersListWire(items, 2, 7, true, "ok")
	if m["page_count"] != 2 {
		t.Fatalf("page_count=%v", m["page_count"])
	}
	if m["dead_letter_count"] != int64(7) {
		t.Fatalf("dead_letter_count must be COUNT(*)=7, not len(page); got %v", m["dead_letter_count"])
	}
	if _, ok := m["count"]; ok {
		t.Fatalf("legacy count=len(page) must not be the KPI key: %+v", m)
	}
}

func TestDeadLettersListWire_UnavailableOmitsCount(t *testing.T) {
	m := deadLettersListWire(nil, 0, 0, false, "OutboxDeadLetters table not applied")
	if _, ok := m["dead_letter_count"]; ok {
		t.Fatalf("unavailable must not invent 0: %+v", m)
	}
	if m["available"] != false {
		t.Fatal("available")
	}
}

func TestHandleOutboxDeadLetters_NoSpannerIsUnavailableNotZero(t *testing.T) {
	h := &Handlers{Svc: NewService(NewMemoryRepository()), Ops: &OpsDeps{}}
	req := httptest.NewRequest(http.MethodGet, "/v1/platform-admin/ops/outbox/dead-letters", nil)
	rr := httptest.NewRecorder()
	h.HandleOutboxDeadLetters(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["available"] != false {
		t.Fatalf("want available=false got %+v", body)
	}
	if _, ok := body["dead_letter_count"]; ok {
		t.Fatalf("no-spanner must not emit dead_letter_count: %+v", body)
	}
}

func TestHandleOutboxSummary_NoStoreOmitsCounts(t *testing.T) {
	h := &Handlers{Svc: NewService(NewMemoryRepository())}
	req := httptest.NewRequest(http.MethodGet, "/v1/platform-admin/ops/outbox/summary", nil)
	rr := httptest.NewRecorder()
	h.HandleOutboxSummary(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["available"] != false || body["dead_letter_available"] != false {
		t.Fatalf("want both unavailable: %+v", body)
	}
	if _, ok := body["dead_letter_count"]; ok {
		t.Fatalf("must not invent dead_letter_count: %+v", body)
	}
}
