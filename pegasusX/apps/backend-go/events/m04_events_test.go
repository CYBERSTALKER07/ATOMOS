package events

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestM04AuditAndGovernanceEventsContract(t *testing.T) {
	if EventAuditAnchorWritten != "audit.anchor.written" {
		t.Fatalf("EventAuditAnchorWritten = %q, want %q", EventAuditAnchorWritten, "audit.anchor.written")
	}
	if EventIAMRoleChanged != "iam.role.changed" {
		t.Fatalf("EventIAMRoleChanged = %q, want %q", EventIAMRoleChanged, "iam.role.changed")
	}
	if EventSodViolationDetected != "sod.violation.detected" {
		t.Fatalf("EventSodViolationDetected = %q, want %q", EventSodViolationDetected, "sod.violation.detected")
	}
	if AggregateAuditAnchor != "AuditChainAnchor" {
		t.Fatalf("AggregateAuditAnchor = %q, want %q", AggregateAuditAnchor, "AuditChainAnchor")
	}
	if AggregateSupplierRole != "SupplierUserRole" {
		t.Fatalf("AggregateSupplierRole = %q, want %q", AggregateSupplierRole, "SupplierUserRole")
	}
	if AggregateSodViolation != "SodViolation" {
		t.Fatalf("AggregateSodViolation = %q, want %q", AggregateSodViolation, "SodViolation")
	}

	anchorEv := AuditAnchorWrittenEvent{
		BaseEvent: BaseEvent{
			Type:      EventAuditAnchorWritten,
			Version:   1,
			Timestamp: "2026-11-16T23:55:00Z",
		},
		TenantID:    "sup-1",
		DayBucket:   "2026-11-16",
		HeadEventID: "evt-99",
		EventCount:  42,
		RootHash:    strings.Repeat("a", 64),
		WormURI:     "gs://pegasusx-worm/sup-1/2026-11-16.json",
		SignerKeyID: "kms-key-1",
		AnchoredBy:  "nightly-anchor-worker",
		AnchoredAt:  "2026-11-16T23:55:00Z",
	}

	rawAnchor, err := json.Marshal(anchorEv)
	if err != nil {
		t.Fatalf("marshal AuditAnchorWrittenEvent: %v", err)
	}
	var decAnchor map[string]any
	decA := json.NewDecoder(bytes.NewReader(rawAnchor))
	decA.UseNumber()
	if err := decA.Decode(&decAnchor); err != nil {
		t.Fatalf("unmarshal AuditAnchorWrittenEvent: %v", err)
	}
	for k, want := range map[string]any{
		"type":          EventAuditAnchorWritten,
		"tenant_id":     "sup-1",
		"day_bucket":    "2026-11-16",
		"head_event_id": "evt-99",
		"root_hash":     strings.Repeat("a", 64),
		"anchored_by":   "nightly-anchor-worker",
	} {
		if decAnchor[k] != want {
			t.Fatalf("anchor[%q] = %v, want %v", k, decAnchor[k], want)
		}
	}
	if decAnchor["event_count"] != json.Number("42") {
		t.Fatalf("anchor event_count = %v, want 42", decAnchor["event_count"])
	}

	roleEv := IAMRoleChangedEvent{
		BaseEvent: BaseEvent{
			Type:      EventIAMRoleChanged,
			Version:   1,
			Timestamp: "2026-11-16T23:55:00Z",
		},
		TenantID:   "sup-1",
		UserID:     "usr-2",
		RoleName:   "FINANCE_MANAGER",
		Action:     "ASSIGNED",
		AssignedBy: "usr-owner",
		TraceID:    "trace-123",
	}

	rawRole, err := json.Marshal(roleEv)
	if err != nil {
		t.Fatalf("marshal IAMRoleChangedEvent: %v", err)
	}
	var decRole map[string]any
	if err := json.Unmarshal(rawRole, &decRole); err != nil {
		t.Fatalf("unmarshal IAMRoleChangedEvent: %v", err)
	}
	if decRole["type"] != EventIAMRoleChanged || decRole["role_name"] != "FINANCE_MANAGER" || decRole["action"] != "ASSIGNED" {
		t.Fatalf("unexpected IAMRoleChangedEvent payload: %v", decRole)
	}

	sodEv := SodViolationDetectedEvent{
		BaseEvent: BaseEvent{
			Type:      EventSodViolationDetected,
			Version:   1,
			Timestamp: "2026-11-16T23:55:00Z",
		},
		TenantID:    "sup-1",
		ViolationID: "viol-1",
		ActorID:     "usr-1",
		Action:      "draft.approve",
		ObjectType:  "FinancialDraft",
		ObjectID:    "draft-7",
		MakerID:     "usr-1",
		RuleCode:    "SELF_APPROVAL_BLOCKED",
		RoutePath:   "/v1/supplier/financial-drafts/draft-7/approve",
		TraceID:     "trace-456",
	}

	rawSod, err := json.Marshal(sodEv)
	if err != nil {
		t.Fatalf("marshal SodViolationDetectedEvent: %v", err)
	}
	var decSod map[string]any
	if err := json.Unmarshal(rawSod, &decSod); err != nil {
		t.Fatalf("unmarshal SodViolationDetectedEvent: %v", err)
	}
	if decSod["type"] != EventSodViolationDetected || decSod["rule_code"] != "SELF_APPROVAL_BLOCKED" || decSod["maker_id"] != "usr-1" {
		t.Fatalf("unexpected SodViolationDetectedEvent payload: %v", decSod)
	}

	envelopePath := filepath.Clean(filepath.Join("..", "..", "..", "packages", "types", "src", "envelope.ts"))
	envelopeSrc, err := os.ReadFile(envelopePath)
	if err != nil {
		t.Fatalf("read %s: %v", envelopePath, err)
	}
	for _, ev := range []string{`"audit.anchor.written"`, `"iam.role.changed"`, `"sod.violation.detected"`} {
		if !strings.Contains(string(envelopeSrc), ev) {
			t.Fatalf("%s missing %s in EventType union", envelopePath, ev)
		}
	}
}
