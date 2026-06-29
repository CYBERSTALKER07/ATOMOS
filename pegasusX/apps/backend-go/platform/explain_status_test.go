package platform

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExplainForCode_known(t *testing.T) {
	ex := ExplainForCode("zone_miss")
	if ex == nil {
		t.Fatal("expected explain for zone_miss")
	}
	if ex.Title == "" || ex.Summary == "" {
		t.Fatalf("incomplete explain: %+v", ex)
	}
}

func TestExplainForCode_preorder_edit_locked_alias(t *testing.T) {
	ex := ExplainForError(errors.New("invalid status transition: preorder edit locked within 2 days of delivery"))
	if ex == nil || ex.Code != "preorder_edit_locked" {
		t.Fatalf("expected preorder_edit_locked, got %+v", ex)
	}
}

func TestAttachExplainToMap(t *testing.T) {
	body := map[string]any{"manifest_id": "mf_x", "status": "not_found"}
	AttachExplainToMap(body, "manifest_not_found")
	explainRaw := body["explain"]
	explain, ok := explainRaw.(StatusExplain)
	if !ok {
		if ptr, okPtr := explainRaw.(*StatusExplain); okPtr {
			explain = *ptr
		} else {
			t.Fatalf("expected explain object, got %#v", explainRaw)
		}
	}
	if explain.Code != "manifest_not_found" {
		t.Fatalf("expected manifest_not_found, got %q", explain.Code)
	}
}

func TestAttachExplain_adds_field(t *testing.T) {
	body := map[string]any{"error": "warehouse_scope_required"}
	AttachExplain(body, nil)
	if body["explain"] == nil {
		t.Fatal("expected explain attached")
	}
}

func TestExplainForCode_gate_and_seal(t *testing.T) {
	for _, code := range []string{"AWAITING_PAYLOAD_SEAL", "manifest_seal_failed", "manifest_not_sealable"} {
		ex := ExplainForCode(code)
		if ex == nil || ex.Title == "" || ex.Summary == "" {
			t.Fatalf("expected catalog entry for %s, got %#v", code, ex)
		}
	}
}

func TestWriteErrorWithExplain(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteErrorWithExplain(rr, http.StatusForbidden, "geofence_violation", nil)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d", rr.Code)
	}
	if !containsAll(rr.Body.String(), `"error"`, `"explain"`, `"geofence_violation"`) {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !contains(s, p) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
