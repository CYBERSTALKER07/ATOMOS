package kafka

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
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
