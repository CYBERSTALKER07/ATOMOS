package events

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWave3AControlAccountingEventsContract(t *testing.T) {
	wantEvents := map[string]string{
		"EventPeriodSoftClosed":    EventPeriodSoftClosed,
		"EventPeriodClosed":        EventPeriodClosed,
		"EventPeriodReopened":      EventPeriodReopened,
		"EventApprovalRequested":   EventApprovalRequested,
		"EventApprovalDecided":     EventApprovalDecided,
		"EventInventoryReceived":   EventInventoryReceived,
		"EventInventoryIssued":     EventInventoryIssued,
		"EventInventoryExpiryRisk": EventInventoryExpiryRisk,
		"EventEsfSubmitted":        EventEsfSubmitted,
		"EventEsfAccepted":         EventEsfAccepted,
		"EventEsfRejected":         EventEsfRejected,
		"EventEsfCancelled":        EventEsfCancelled,
	}
	expectedValues := map[string]string{
		"EventPeriodSoftClosed":    "period.soft_closed",
		"EventPeriodClosed":        "period.closed",
		"EventPeriodReopened":      "period.reopened",
		"EventApprovalRequested":   "approval.requested",
		"EventApprovalDecided":     "approval.decided",
		"EventInventoryReceived":   "inventory.received",
		"EventInventoryIssued":     "inventory.issued",
		"EventInventoryExpiryRisk": "inventory.expiry.risk",
		"EventEsfSubmitted":        "esf.submitted",
		"EventEsfAccepted":         "esf.accepted",
		"EventEsfRejected":         "esf.rejected",
		"EventEsfCancelled":        "esf.cancelled",
	}
	for name, got := range wantEvents {
		if got != expectedValues[name] {
			t.Fatalf("%s = %q, want %q", name, got, expectedValues[name])
		}
	}

	if AggregatePeriod != "Period" {
		t.Fatalf("AggregatePeriod = %q, want Period", AggregatePeriod)
	}
	if AggregateApprovalRequest != "ApprovalRequest" {
		t.Fatalf("AggregateApprovalRequest = %q, want ApprovalRequest", AggregateApprovalRequest)
	}
	if AggregateInventoryCostLayer != "InventoryCostLayer" {
		t.Fatalf("AggregateInventoryCostLayer = %q, want InventoryCostLayer", AggregateInventoryCostLayer)
	}
	if AggregateInventoryMovement != "InventoryMovement" {
		t.Fatalf("AggregateInventoryMovement = %q, want InventoryMovement", AggregateInventoryMovement)
	}
	if AggregateEsfDocument != "EsfDocument" {
		t.Fatalf("AggregateEsfDocument = %q, want EsfDocument", AggregateEsfDocument)
	}

	periodEv := PeriodEvent{
		BaseEvent: BaseEvent{
			Type:      EventPeriodClosed,
			Version:   1,
			Timestamp: "2026-11-17T03:00:00Z",
		},
		EntityID:     "sup-1",
		PeriodID:     "2026-10",
		Status:       "CLOSED",
		Timezone:     "Asia/Tashkent",
		ActorID:      "usr-fin",
		SnapshotURI:  "gs://pegasusx-worm/snapshots/sup-1/2026-10.json",
		SnapshotHash: strings.Repeat("a", 64),
		OccurredAt:   "2026-11-17T03:00:00Z",
	}
	assertJSONIntegerFields(t, "PeriodEvent", periodEv, map[string]any{
		"type":          EventPeriodClosed,
		"entity_id":     "sup-1",
		"period_id":     "2026-10",
		"status":        "CLOSED",
		"timezone":      "Asia/Tashkent",
		"actor_id":      "usr-fin",
		"snapshot_hash": strings.Repeat("a", 64),
	})

	reqEv := ApprovalRequestedEvent{
		BaseEvent: BaseEvent{
			Type:      EventApprovalRequested,
			Version:   1,
			Timestamp: "2026-11-17T03:01:00Z",
		},
		TenantID:          "sup-1",
		RequestID:         "req-1",
		PolicyID:          "pol-1",
		PolicyVersion:     2,
		ObjectType:        "Period",
		ObjectID:          "2026-10",
		Action:            "period.reopen",
		AmountMinor:       50000000,
		RequestedBy:       "usr-1",
		ActorType:         "human",
		PayloadHash:       strings.Repeat("b", 64),
		Status:            "PENDING",
		RequiredApprovers: 2,
		BreakGlass:        false,
	}
	assertJSONIntegerFields(t, "ApprovalRequestedEvent", reqEv, map[string]any{
		"type":               EventApprovalRequested,
		"tenant_id":          "sup-1",
		"request_id":         "req-1",
		"policy_id":          "pol-1",
		"policy_version":     json.Number("2"),
		"amount_minor":       json.Number("50000000"),
		"required_approvers": json.Number("2"),
		"break_glass":        false,
	})

	decEv := ApprovalDecidedEvent{
		BaseEvent: BaseEvent{
			Type:      EventApprovalDecided,
			Version:   1,
			Timestamp: "2026-11-17T03:02:00Z",
		},
		TenantID:    "sup-1",
		RequestID:   "req-1",
		DecisionID:  "dec-1",
		StepIndex:   0,
		ApproverID:  "usr-2",
		Decision:    "APPROVE",
		Status:      "APPROVED",
		MfaVerified: true,
		BreakGlass:  false,
		Comment:     "verified",
	}
	assertJSONIntegerFields(t, "ApprovalDecidedEvent", decEv, map[string]any{
		"type":         EventApprovalDecided,
		"decision_id":  "dec-1",
		"step_index":   json.Number("0"),
		"approver_id":  "usr-2",
		"decision":     "APPROVE",
		"status":       "APPROVED",
		"mfa_verified": true,
	})

	recvEv := InventoryReceivedEvent{
		BaseEvent: BaseEvent{
			Type:      EventInventoryReceived,
			Version:   1,
			Timestamp: "2026-11-17T03:03:00Z",
		},
		EntityID:       "sup-1",
		WarehouseID:    "wh-1",
		ProductID:      "sku-1",
		LotID:          "lot-1",
		LayerID:        "lay-1",
		MovementID:     "mov-1",
		ReferenceType:  "PO_RECEIPT",
		ReferenceID:    "po-1",
		Quantity:       100,
		UnitCostMinor:  12500,
		TotalCostMinor: 1250000,
		Currency:       "UZS",
		GlEntryID:      "je-1",
		ExpiryDate:     "2027-01-15",
		OccurredAt:     "2026-11-17T03:03:00Z",
	}
	assertJSONIntegerFields(t, "InventoryReceivedEvent", recvEv, map[string]any{
		"type":             EventInventoryReceived,
		"quantity":         json.Number("100"),
		"unit_cost_minor":  json.Number("12500"),
		"total_cost_minor": json.Number("1250000"),
		"currency":         "UZS",
		"occurred_at":      "2026-11-17T03:03:00Z",
	})

	issueEv := InventoryIssuedEvent{
		BaseEvent: BaseEvent{
			Type:      EventInventoryIssued,
			Version:   1,
			Timestamp: "2026-11-17T03:04:00Z",
		},
		EntityID:       "sup-1",
		WarehouseID:    "wh-1",
		ProductID:      "sku-1",
		LotID:          "lot-1",
		LayerID:        "lay-1",
		MovementID:     "mov-2",
		ReferenceType:  "ORDER_PICK",
		ReferenceID:    "ord-1",
		OrderID:        "ord-1",
		Quantity:       10,
		UnitCostMinor:  12500,
		TotalCostMinor: 125000,
		Currency:       "UZS",
		GlEntryID:      "je-2",
		OccurredAt:     "2026-11-17T03:04:00Z",
	}
	assertJSONIntegerFields(t, "InventoryIssuedEvent", issueEv, map[string]any{
		"type":             EventInventoryIssued,
		"order_id":         "ord-1",
		"quantity":         json.Number("10"),
		"unit_cost_minor":  json.Number("12500"),
		"total_cost_minor": json.Number("125000"),
		"occurred_at":      "2026-11-17T03:04:00Z",
	})

	riskEv := InventoryExpiryRiskEvent{
		BaseEvent: BaseEvent{
			Type:      EventInventoryExpiryRisk,
			Version:   1,
			Timestamp: "2026-11-17T03:05:00Z",
		},
		EntityID:         "sup-1",
		WarehouseID:      "wh-1",
		ProductID:        "sku-1",
		LayerID:          "lay-1",
		QtyRemaining:     90,
		UnitCostMinor:    12500,
		ValueAtRiskMinor: 1125000,
		Currency:         "UZS",
		ExpiryDate:       "2026-11-25",
		DaysRemaining:    8,
		DaysToExpiry:     8,
		OccurredAt:       "2026-11-17T03:05:00Z",
	}
	assertJSONIntegerFields(t, "InventoryExpiryRiskEvent", riskEv, map[string]any{
		"type":                EventInventoryExpiryRisk,
		"qty_remaining":       json.Number("90"),
		"unit_cost_minor":     json.Number("12500"),
		"value_at_risk_minor": json.Number("1125000"),
		"days_remaining":      json.Number("8"),
		"days_to_expiry":      json.Number("8"),
		"occurred_at":         "2026-11-17T03:05:00Z",
	})

	esfEv := EsfDocumentEvent{
		BaseEvent: BaseEvent{
			Type:      EventEsfSubmitted,
			Version:   1,
			Timestamp: "2026-11-17T03:06:00Z",
		},
		EntityID:        "sup-1",
		EsfID:           "esf-1",
		Direction:       "OUTBOUND",
		OrderID:         "ord-1",
		CounterpartyTIN: "123456789",
		TotalMinor:      1120000,
		VatMinor:        120000,
		VatRateBps:      1200,
		Status:          "SUBMITTED",
		Operator:        "didox",
		OperatorRef:     "didox-ref-1",
		PeriodID:        "2026-11",
		PayloadHash:     strings.Repeat("c", 64),
		ActorID:         "usr-1",
		OccurredAt:      "2026-11-17T03:06:00Z",
	}
	assertJSONIntegerFields(t, "EsfDocumentEvent", esfEv, map[string]any{
		"type":         EventEsfSubmitted,
		"esf_id":       "esf-1",
		"total_minor":  json.Number("1120000"),
		"vat_minor":    json.Number("120000"),
		"vat_rate_bps": json.Number("1200"),
		"period_id":    "2026-11",
		"occurred_at":  "2026-11-17T03:06:00Z",
	})

	envelopePath := filepath.Clean(filepath.Join("..", "..", "..", "packages", "types", "src", "envelope.ts"))
	envelopeSrc, err := os.ReadFile(envelopePath)
	if err != nil {
		t.Fatalf("read %s: %v", envelopePath, err)
	}
	for _, evVal := range expectedValues {
		if !strings.Contains(string(envelopeSrc), `"`+evVal+`"`) {
			t.Fatalf("%s missing %q in EventType union", envelopePath, evVal)
		}
	}
}

func assertJSONIntegerFields(t *testing.T, label string, payload any, want map[string]any) {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal %s: %v", label, err)
	}
	var decoded map[string]any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&decoded); err != nil {
		t.Fatalf("decode %s: %v", label, err)
	}
	for k, expected := range want {
		if decoded[k] != expected {
			t.Fatalf("%s[%q] = %#v, want %#v", label, k, decoded[k], expected)
		}
	}
}
