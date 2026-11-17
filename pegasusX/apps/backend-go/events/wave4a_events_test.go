package events

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWave4AEntitiesP2PConnectorEventsContract(t *testing.T) {
	wantEvents := map[string]string{
		"EventEntityIntercompanyPosted":  EventEntityIntercompanyPosted,
		"EventFxRevaluationPosted":       EventFxRevaluationPosted,
		"EventP2PPOCreated":              EventP2PPOCreated,
		"EventP2PReceiptPosted":          EventP2PReceiptPosted,
		"EventP2PThreeWayMatchCompleted": EventP2PThreeWayMatchCompleted,
		"EventConnectorPushCompleted":    EventConnectorPushCompleted,
	}
	expectedValues := map[string]string{
		"EventEntityIntercompanyPosted":  "entity.intercompany.posted",
		"EventFxRevaluationPosted":       "fx.revaluation.posted",
		"EventP2PPOCreated":              "p2p.po.created",
		"EventP2PReceiptPosted":          "p2p.receipt.posted",
		"EventP2PThreeWayMatchCompleted": "p2p.three_way_match.completed",
		"EventConnectorPushCompleted":    "connector.push.completed",
	}
	for name, got := range wantEvents {
		if got != expectedValues[name] {
			t.Fatalf("%s = %q, want %q", name, got, expectedValues[name])
		}
	}

	if AggregateLegalEntity != "LegalEntity" {
		t.Fatalf("AggregateLegalEntity = %q, want LegalEntity", AggregateLegalEntity)
	}
	if AggregateIntercompanyTransfer != "IntercompanyTransfer" {
		t.Fatalf("AggregateIntercompanyTransfer = %q, want IntercompanyTransfer", AggregateIntercompanyTransfer)
	}
	if AggregateFxRevaluationRun != "FxRevaluationRun" {
		t.Fatalf("AggregateFxRevaluationRun = %q, want FxRevaluationRun", AggregateFxRevaluationRun)
	}
	if AggregateProcessTemplate != "ProcessTemplate" {
		t.Fatalf("AggregateProcessTemplate = %q, want ProcessTemplate", AggregateProcessTemplate)
	}
	if AggregateProcessInstance != "ProcessInstance" {
		t.Fatalf("AggregateProcessInstance = %q, want ProcessInstance", AggregateProcessInstance)
	}
	if AggregatePurchaseOrder != "PurchaseOrder" {
		t.Fatalf("AggregatePurchaseOrder = %q, want PurchaseOrder", AggregatePurchaseOrder)
	}
	if AggregateGoodsReceipt != "GoodsReceipt" {
		t.Fatalf("AggregateGoodsReceipt = %q, want GoodsReceipt", AggregateGoodsReceipt)
	}
	if AggregateVendorInvoice != "VendorInvoice" {
		t.Fatalf("AggregateVendorInvoice = %q, want VendorInvoice", AggregateVendorInvoice)
	}

	icEv := EntityIntercompanyPostedEvent{
		BaseEvent: BaseEvent{
			Type:      EventEntityIntercompanyPosted,
			Version:   1,
			Timestamp: "2026-11-17T06:45:00Z",
		},
		TenantID:        "sup-1",
		TransferID:      "ic-1",
		FromEntityID:    "ent-hq",
		ToEntityID:      "ent-sub-1",
		TransferType:    "CASH_ADVANCE",
		AmountMinor:     25000000,
		Currency:        "UZS",
		PeriodID:        "2026-11",
		ReferenceType:   "LOAN",
		ReferenceID:     "loan-1",
		DebitJournalID:  "gle_debit_1",
		CreditJournalID: "gle_credit_1",
		Status:          "POSTED",
		CreatedBy:       "usr-cfo",
		OccurredAt:      "2026-11-17T06:45:00Z",
	}
	assertJSONIntegerFields(t, "EntityIntercompanyPostedEvent", icEv, map[string]any{
		"type":         EventEntityIntercompanyPosted,
		"transfer_id":  "ic-1",
		"amount_minor": json.Number("25000000"),
		"currency":     "UZS",
	})

	fxEv := FxRevaluationPostedEvent{
		BaseEvent: BaseEvent{
			Type:      EventFxRevaluationPosted,
			Version:   1,
			Timestamp: "2026-11-17T06:45:05Z",
		},
		TenantID:                "sup-1",
		RunID:                   "fxrun-1",
		EntityID:                "ent-hq",
		PeriodID:                "2026-11",
		BaseCurrency:            "UZS",
		UnrealizedGainLossMinor: 1245000,
		RoundingDeltaMinor:      -250,
		JournalEntryID:          "gle_fx_1",
		RatesJSON:               `{"USD/UZS":"12850.00"}`,
		CreatedBy:               "usr-acct",
		OccurredAt:              "2026-11-17T06:45:05Z",
	}
	assertJSONIntegerFields(t, "FxRevaluationPostedEvent", fxEv, map[string]any{
		"type":                       EventFxRevaluationPosted,
		"run_id":                     "fxrun-1",
		"unrealized_gain_loss_minor": json.Number("1245000"),
		"rounding_delta_minor":       json.Number("-250"),
	})

	poEv := P2PEvent{
		BaseEvent: BaseEvent{
			Type:      EventP2PPOCreated,
			Version:   1,
			Timestamp: "2026-11-17T06:45:10Z",
		},
		TenantID:          "sup-1",
		SupplierID:        "sup-1",
		EntityID:          "ent-hq",
		PoID:              "po-1",
		VendorID:          "vend-1",
		Status:            "APPROVED",
		SubtotalMinor:     1000000,
		VatMinor:          120000,
		TotalMinor:        1120000,
		Currency:          "UZS",
		LineCount:         2,
		ApprovalRequestID: "apr-1",
		ActorID:           "usr-buyer",
		OccurredAt:        "2026-11-17T06:45:10Z",
	}
	assertJSONIntegerFields(t, "P2PEvent(po.created)", poEv, map[string]any{
		"type":           EventP2PPOCreated,
		"po_id":          "po-1",
		"subtotal_minor": json.Number("1000000"),
		"vat_minor":      json.Number("120000"),
		"total_minor":    json.Number("1120000"),
		"line_count":     json.Number("2"),
	})

	matchEv := P2PEvent{
		BaseEvent: BaseEvent{
			Type:      EventP2PThreeWayMatchCompleted,
			Version:   1,
			Timestamp: "2026-11-17T06:45:15Z",
		},
		TenantID:        "sup-1",
		EntityID:        "ent-hq",
		PoID:            "po-1",
		ReceiptID:       "rcpt-1",
		VendorInvoiceID: "vinv-1",
		EsfDocID:        "esf-1",
		VendorID:        "vend-1",
		Status:          "MATCHED_3WAY",
		MatchStatus:     "MATCHED_3WAY",
		SubtotalMinor:   1000000,
		VatMinor:        120000,
		TotalMinor:      1120000,
		VarianceMinor:   1500,
		Currency:        "UZS",
		JournalEntryID:  "gle_3way_1",
		ActorID:         "usr-acct",
		OccurredAt:      "2026-11-17T06:45:15Z",
	}
	assertJSONIntegerFields(t, "P2PEvent(three_way_match)", matchEv, map[string]any{
		"type":              EventP2PThreeWayMatchCompleted,
		"vendor_invoice_id": "vinv-1",
		"match_status":      "MATCHED_3WAY",
		"variance_minor":    json.Number("1500"),
	})

	pushEv := ConnectorPushCompletedEvent{
		BaseEvent: BaseEvent{
			Type:      EventConnectorPushCompleted,
			Version:   1,
			Timestamp: "2026-11-17T06:45:20Z",
		},
		TenantID:         "sup-1",
		RunID:            "run-push-1",
		Provider:         "moysklad",
		AccountLabel:     "default",
		Direction:        "PUSH",
		StreamName:       "goods_receipt",
		Status:           "COMPLETED",
		RecordsProcessed: 12,
		ApprovalID:       "apr-push-1",
		ExternalRef:      "ms-supply-42",
		TriggeredBy:      "usr-admin",
		CompletedAt:      "2026-11-17T06:45:20Z",
		OccurredAt:       "2026-11-17T06:45:20Z",
	}
	assertJSONIntegerFields(t, "ConnectorPushCompletedEvent", pushEv, map[string]any{
		"type":              EventConnectorPushCompleted,
		"run_id":            "run-push-1",
		"provider":          "moysklad",
		"records_processed": json.Number("12"),
		"approval_id":       "apr-push-1",
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
