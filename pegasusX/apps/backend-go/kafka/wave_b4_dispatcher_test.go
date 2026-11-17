package kafka

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
	"github.com/segmentio/kafka-go"
)

func TestHandlePlanningScenarioPublished_NoError(t *testing.T) {
	d := NewNotificationDispatcher(DispatcherDeps{
		SupplierHub: ws.NewHub("supplier", nil, nil),
	})
	payload, _ := json.Marshal(events.PlanningEvent{
		BaseEvent:  events.BaseEvent{Type: events.EventPlanningScenarioPublished},
		SupplierID: "sup-1",
		ScenarioID: "sc-1",
		Action:     "PUBLISH",
	})
	if err := d.handlePlanningEvent(context.Background(), payload, "tr-b4-sc"); err != nil {
		t.Fatalf("handlePlanningEvent scenario published: %v", err)
	}
}

func TestHandleSupplierCreditProgramEvent_NoError(t *testing.T) {
	d := NewNotificationDispatcher(DispatcherDeps{
		SupplierHub: ws.NewHub("supplier", nil, nil),
	})
	payload, _ := json.Marshal(events.SupplierCreditProgramEvent{
		BaseEvent:      events.BaseEvent{Type: events.EventSupplierCreditProgramChanged},
		SupplierID:     "sup-1",
		ProgramEnabled: true,
		Action:         "ENABLE",
	})
	if err := d.handleSupplierCreditProgramEvent(context.Background(), payload, "tr-b4-prog"); err != nil {
		t.Fatalf("handleSupplierCreditProgramEvent: %v", err)
	}
}

func TestHandleSupplierCreditTermsEvent_NoError(t *testing.T) {
	d := NewNotificationDispatcher(DispatcherDeps{
		SupplierHub: ws.NewHub("supplier", nil, nil),
		RetailerHub: ws.NewHub("retailer", nil, nil),
	})
	payload, _ := json.Marshal(events.SupplierCreditTermsEvent{
		BaseEvent:     events.BaseEvent{Type: events.EventSupplierCreditTermsChanged},
		SupplierID:    "sup-1",
		RetailerID:    "ret-1",
		CreditEnabled: true,
		Action:        "ENABLE",
	})
	if err := d.handleSupplierCreditTermsEvent(context.Background(), payload, "tr-b4-terms"); err != nil {
		t.Fatalf("handleSupplierCreditTermsEvent: %v", err)
	}
}

func TestHandleControlTowerEvent_NoError(t *testing.T) {
	d := NewNotificationDispatcher(DispatcherDeps{
		SupplierHub: ws.NewHub("supplier", nil, nil),
	})
	payload, _ := json.Marshal(events.ControlTowerEvent{
		BaseEvent:  events.BaseEvent{Type: events.EventControlTowerRunCreated},
		SupplierID: "sup-1",
		PlaybookID: "pb-1",
		RunID:      "run-1",
		Status:     "SUGGESTED",
	})
	if err := d.handleControlTowerEvent(context.Background(), payload, "tr-b4-ct"); err != nil {
		t.Fatalf("handleControlTowerEvent: %v", err)
	}
}

func TestHandleReplenishmentPolicyUpdated_NoError(t *testing.T) {
	d := NewNotificationDispatcher(DispatcherDeps{
		SupplierHub: ws.NewHub("supplier", nil, nil),
	})
	payload, _ := json.Marshal(map[string]any{
		"type":        events.EventReplenishmentPolicyUpdated,
		"supplier_id": "sup-1",
	})
	if err := d.handleReplenishmentEvent(context.Background(), payload, "tr-s-p1-2"); err != nil {
		t.Fatalf("handleReplenishmentEvent policy: %v", err)
	}
}

func TestHandleRefundAndBuyerAcceptance_B6(t *testing.T) {
	d := NewNotificationDispatcher(DispatcherDeps{
		SupplierHub: ws.NewHub("supplier", nil, nil),
		RetailerHub: ws.NewHub("retailer", nil, nil),
	})
	refundPayload, _ := json.Marshal(events.FinanceEvent{
		BaseEvent:     events.BaseEvent{Type: events.EventRefundSucceeded},
		OrderID:       "ord-1",
		SupplierID:    "sup-1",
		RetailerID:    "ret-1",
		TransactionID: "ref-1",
	})
	if err := d.handleRefundEvent(context.Background(), refundPayload, "tr-b6-ref"); err != nil {
		t.Fatalf("handleRefundEvent: %v", err)
	}
	baPayload, _ := json.Marshal(events.BuyerAcceptanceEvent{
		BaseEvent:  events.BaseEvent{Type: events.EventBuyerAcceptanceAccepted},
		OrderID:    "ord-1",
		SupplierID: "sup-1",
		RetailerID: "ret-1",
		Status:     "ACCEPTED",
	})
	if err := d.handleBuyerAcceptanceEvent(context.Background(), baPayload, "tr-b6-ba"); err != nil {
		t.Fatalf("handleBuyerAcceptanceEvent: %v", err)
	}
}

func TestHandleFoundryOperationalEvents_AndTraceParent(t *testing.T) {
	d := NewNotificationDispatcher(DispatcherDeps{
		SupplierHub:  ws.NewHub("supplier", nil, nil),
		WarehouseHub: ws.NewHub("warehouse", nil, nil),
		DriverHub:    ws.NewHub("driver", nil, nil),
		RetailerHub:  ws.NewHub("retailer", nil, nil),
	})
	for _, evtType := range []string{
		events.EventCashierSessionStarted,
		events.EventCashierSessionTransition,
		events.EventFleetDVIRInspected,
		events.EventFleetDVIRRedTag,
		events.EventYardGateCheckRecorded,
		events.EventContainerLedgerUpdated,
		events.EventOfflineQuotaLeaseUpdated,
		events.EventAMLExposureUpdated,
		events.EventOntologyObjectUpdated,
	} {
		payload, _ := json.Marshal(map[string]any{
			"type":        evtType,
			"supplier_id": "sup-1",
			"depot_id":    "wh-1",
			"driver_id":   "drv-1",
			"retailer_id": "ret-1",
			"session_id":  "sess-" + evtType,
		})
		msg := kafka.Message{
			Topic:     events.TopicOrders,
			Partition: 0,
			Offset:    100,
			Value:     payload,
			Headers: []kafka.Header{
				{Key: "trace_id", Value: []byte("tr-foundry-1")},
				{Key: "traceparent", Value: []byte("00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")},
			},
		}
		ctx := WithTraceFromMessage(context.Background(), msg)
		if got := outbox.TraceParentFromContext(ctx); got != "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01" {
			t.Fatalf("expected traceparent in context, got %q", got)
		}
		if err := d.dispatchParityEvent(ctx, evtType, payload, "tr-foundry-1"); err != nil {
			t.Fatalf("dispatchParityEvent(%s): %v", evtType, err)
		}
	}
}
