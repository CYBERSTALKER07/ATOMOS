package kafka

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
	"github.com/segmentio/kafka-go"
)

type dispatcherConnSpy struct {
	id       string
	messages [][]byte
}

func (c *dispatcherConnSpy) ID() string { return c.id }

func (c *dispatcherConnSpy) Identity() auth.Claims { return auth.Claims{} }

func (c *dispatcherConnSpy) Send(_ context.Context, payload []byte) error {
	c.messages = append(c.messages, append([]byte(nil), payload...))
	return nil
}

func TestNotificationDispatcher_OrderAssignmentFansDriverAndRetailer(t *testing.T) {
	t.Parallel()

	supplierConn := &dispatcherConnSpy{id: "supplier"}
	retailerConn := &dispatcherConnSpy{id: "retailer"}
	driverConn := &dispatcherConnSpy{id: "driver"}

	supplierHub := ws.NewHub("supplier", nil, nil)
	retailerHub := ws.NewHub("retailer", nil, nil)
	driverHub := ws.NewHub("driver", nil, nil)
	supplierHub.Subscribe("supplier:sup-1", supplierConn)
	retailerHub.Subscribe("retailer:ret-1", retailerConn)
	driverHub.Subscribe("driver:drv-1", driverConn)

	dispatcher := NewNotificationDispatcher(DispatcherDeps{
		SupplierHub: supplierHub,
		RetailerHub: retailerHub,
		DriverHub:   driverHub,
	})

	payload, err := json.Marshal(map[string]any{
		"type":        events.EventOrderAssigned,
		"trace_id":    "trace-1",
		"order_id":    "ord-1",
		"supplier_id": "sup-1",
		"retailer_id": "ret-1",
		"driver_id":   "drv-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := dispatcher.HandleEvent(context.Background(), kafka.Message{Value: payload}); err != nil {
		t.Fatalf("handle event: %v", err)
	}
	if len(supplierConn.messages) != 1 || len(retailerConn.messages) != 1 || len(driverConn.messages) != 1 {
		t.Fatalf("expected one message per hub, got supplier=%d retailer=%d driver=%d",
			len(supplierConn.messages), len(retailerConn.messages), len(driverConn.messages))
	}
}

func TestNotificationDispatcher_FinanceEventFansRetailer(t *testing.T) {
	t.Parallel()

	supplierConn := &dispatcherConnSpy{id: "supplier"}
	retailerConn := &dispatcherConnSpy{id: "retailer"}
	supplierHub := ws.NewHub("supplier", nil, nil)
	retailerHub := ws.NewHub("retailer", nil, nil)
	supplierHub.Subscribe("supplier:sup-1", supplierConn)
	retailerHub.Subscribe("retailer:ret-1", retailerConn)

	dispatcher := NewNotificationDispatcher(DispatcherDeps{
		SupplierHub: supplierHub,
		RetailerHub: retailerHub,
	})

	payload, _ := json.Marshal(map[string]any{
		"type":        events.EventPaymentCleared,
		"trace_id":    "trace-2",
		"supplier_id": "sup-1",
		"retailer_id": "ret-1",
		"order_id":    "ord-1",
	})
	if err := dispatcher.HandleEvent(context.Background(), kafka.Message{Value: payload}); err != nil {
		t.Fatalf("handle event: %v", err)
	}
	if len(supplierConn.messages) != 1 || len(retailerConn.messages) != 1 {
		t.Fatalf("expected finance fanout to supplier and retailer")
	}
}

func TestNotificationDispatcher_ManifestEventUsesFactoryRoom(t *testing.T) {
	t.Parallel()

	factoryConn := &dispatcherConnSpy{id: "factory"}
	payloadConn := &dispatcherConnSpy{id: "payload"}
	factoryHub := ws.NewHub("factory", nil, nil)
	payloadHub := ws.NewHub("payload", nil, nil)
	factoryHub.Subscribe("factory:fc-1", factoryConn)
	payloadHub.Subscribe("payload:sup-1", payloadConn)

	dispatcher := NewNotificationDispatcher(DispatcherDeps{
		FactoryHub: factoryHub,
		PayloadHub: payloadHub,
	})

	payload, _ := json.Marshal(map[string]any{
		"type":        events.EventManifestSealed,
		"trace_id":    "trace-3",
		"supplier_id": "sup-1",
		"factory_id":  "fc-1",
	})
	if err := dispatcher.HandleEvent(context.Background(), kafka.Message{Value: payload}); err != nil {
		t.Fatalf("handle event: %v", err)
	}
	if len(factoryConn.messages) != 1 || len(payloadConn.messages) != 1 {
		t.Fatalf("expected manifest fanout to factory and payload hubs")
	}
}

func TestNotificationDispatcher_VehicleCreatedDoesNotFanDriverHub(t *testing.T) {
	t.Parallel()

	supplierConn := &dispatcherConnSpy{id: "supplier"}
	driverConn := &dispatcherConnSpy{id: "driver"}
	supplierHub := ws.NewHub("supplier", nil, nil)
	driverHub := ws.NewHub("driver", nil, nil)
	supplierHub.Subscribe("supplier:sup-1", supplierConn)
	driverHub.Subscribe("driver:drv-1", driverConn)

	dispatcher := NewNotificationDispatcher(DispatcherDeps{
		SupplierHub: supplierHub,
		DriverHub:   driverHub,
	})

	payload, _ := json.Marshal(map[string]any{
		"type":         events.EventVehicleCreated,
		"trace_id":     "trace-veh",
		"supplier_id":  "sup-1",
		"vehicle_id":   "veh-1",
		"home_node_id": "wh-1",
	})
	if err := dispatcher.HandleEvent(context.Background(), kafka.Message{Value: payload}); err != nil {
		t.Fatalf("handle event: %v", err)
	}
	if len(supplierConn.messages) != 1 {
		t.Fatalf("supplier messages = %d, want 1", len(supplierConn.messages))
	}
	if len(driverConn.messages) != 0 {
		t.Fatalf("driver hub should not receive vehicle created")
	}
}

func TestNotificationDispatcher_NegotiationEventFansOrderParties(t *testing.T) {
	t.Parallel()

	supplierConn := &dispatcherConnSpy{id: "supplier"}
	retailerConn := &dispatcherConnSpy{id: "retailer"}
	driverConn := &dispatcherConnSpy{id: "driver"}

	supplierHub := ws.NewHub("supplier", nil, nil)
	retailerHub := ws.NewHub("retailer", nil, nil)
	driverHub := ws.NewHub("driver", nil, nil)
	supplierHub.Subscribe("supplier:sup-1", supplierConn)
	retailerHub.Subscribe("retailer:ret-1", retailerConn)
	driverHub.Subscribe("driver:drv-1", driverConn)

	dispatcher := NewNotificationDispatcher(DispatcherDeps{
		SupplierHub: supplierHub,
		RetailerHub: retailerHub,
		DriverHub:   driverHub,
	})

	payload, err := json.Marshal(map[string]any{
		"type":        events.EventNegotiationProposed,
		"trace_id":    "trace-neg",
		"order_id":    "ord-1",
		"supplier_id": "sup-1",
		"retailer_id": "ret-1",
		"driver_id":   "drv-1",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := dispatcher.HandleEvent(context.Background(), kafka.Message{Value: payload}); err != nil {
		t.Fatalf("handle event: %v", err)
	}
	if len(supplierConn.messages) != 1 || len(retailerConn.messages) != 1 || len(driverConn.messages) != 1 {
		t.Fatalf("fanout counts supplier=%d retailer=%d driver=%d, want 1 each",
			len(supplierConn.messages), len(retailerConn.messages), len(driverConn.messages))
	}
}

func TestNotificationDispatcher_DedupesReplayWithinWindow(t *testing.T) {
	t.Parallel()

	supplierConn := &dispatcherConnSpy{id: "supplier"}
	supplierHub := ws.NewHub("supplier", nil, nil)
	supplierHub.Subscribe("supplier:sup-1", supplierConn)
	dispatcher := NewNotificationDispatcher(DispatcherDeps{SupplierHub: supplierHub})

	payload, _ := json.Marshal(map[string]any{
		"type":        events.EventSupplierUpdated,
		"trace_id":    "trace-dedup",
		"supplier_id": "sup-1",
	})
	msg := kafka.Message{Value: payload}
	if err := dispatcher.HandleEvent(context.Background(), msg); err != nil {
		t.Fatalf("first handle: %v", err)
	}
	if err := dispatcher.HandleEvent(context.Background(), msg); err != nil {
		t.Fatalf("second handle: %v", err)
	}
	if len(supplierConn.messages) != 1 {
		t.Fatalf("dedup messages = %d, want 1", len(supplierConn.messages))
	}
}

func TestNotificationDispatcher_RouteReorderedFansDriver(t *testing.T) {
	t.Parallel()

	driverConn := &dispatcherConnSpy{id: "driver"}
	driverHub := ws.NewHub("driver", nil, nil)
	driverHub.Subscribe("driver:drv-1", driverConn)

	dispatcher := NewNotificationDispatcher(DispatcherDeps{DriverHub: driverHub})
	payload, _ := json.Marshal(map[string]any{
		"type":      events.EventRouteReordered,
		"trace_id":  "trace-route",
		"route_id":  "route-1",
		"driver_id": "drv-1",
	})
	if err := dispatcher.HandleEvent(context.Background(), kafka.Message{Value: payload}); err != nil {
		t.Fatalf("handle event: %v", err)
	}
	if len(driverConn.messages) != 1 {
		t.Fatalf("driver messages = %d, want 1", len(driverConn.messages))
	}
}

func TestNotificationDispatcher_MissingItemsFansSupplierAndDriver(t *testing.T) {
	t.Parallel()

	supplierConn := &dispatcherConnSpy{id: "supplier"}
	driverConn := &dispatcherConnSpy{id: "driver"}
	supplierHub := ws.NewHub("supplier", nil, nil)
	driverHub := ws.NewHub("driver", nil, nil)
	supplierHub.Subscribe("supplier:sup-1", supplierConn)
	driverHub.Subscribe("driver:drv-1", driverConn)

	dispatcher := NewNotificationDispatcher(DispatcherDeps{
		SupplierHub: supplierHub,
		DriverHub:   driverHub,
	})
	payload, _ := json.Marshal(map[string]any{
		"type":        events.EventMissingItemsReported,
		"trace_id":    "trace-missing",
		"order_id":    "ord-1",
		"supplier_id": "sup-1",
		"driver_id":   "drv-1",
	})
	if err := dispatcher.HandleEvent(context.Background(), kafka.Message{Value: payload}); err != nil {
		t.Fatalf("handle event: %v", err)
	}
	if len(supplierConn.messages) != 1 || len(driverConn.messages) != 1 {
		t.Fatalf("supplier=%d driver=%d, want 1 each", len(supplierConn.messages), len(driverConn.messages))
	}
}

func TestNotificationDispatcher_DriverAvailabilityFansWarehouseHomeNode(t *testing.T) {
	t.Parallel()

	supplierConn := &dispatcherConnSpy{id: "supplier"}
	driverConn := &dispatcherConnSpy{id: "driver"}
	warehouseConn := &dispatcherConnSpy{id: "warehouse"}
	supplierHub := ws.NewHub("supplier", nil, nil)
	driverHub := ws.NewHub("driver", nil, nil)
	warehouseHub := ws.NewHub("warehouse", nil, nil)
	supplierHub.Subscribe("supplier:sup-1", supplierConn)
	driverHub.Subscribe("driver:drv-1", driverConn)
	warehouseHub.Subscribe("warehouse:wh-1", warehouseConn)

	dispatcher := NewNotificationDispatcher(DispatcherDeps{
		SupplierHub:  supplierHub,
		DriverHub:    driverHub,
		WarehouseHub: warehouseHub,
	})
	payload, _ := json.Marshal(map[string]any{
		"type":           events.EventDriverAvailabilityChanged,
		"trace_id":       "trace-avail",
		"supplier_id":    "sup-1",
		"driver_id":      "drv-1",
		"home_node_type": "WAREHOUSE",
		"home_node_id":   "wh-1",
		"on_shift":       true,
	})
	if err := dispatcher.HandleEvent(context.Background(), kafka.Message{Value: payload}); err != nil {
		t.Fatalf("handle event: %v", err)
	}
	if len(supplierConn.messages) != 1 || len(driverConn.messages) != 1 || len(warehouseConn.messages) != 1 {
		t.Fatalf("supplier=%d driver=%d warehouse=%d, want 1 each",
			len(supplierConn.messages), len(driverConn.messages), len(warehouseConn.messages))
	}
}

func TestNotificationDispatcher_AIRecommendationFansSupplierOnly(t *testing.T) {
	t.Parallel()

	supplierConn := &dispatcherConnSpy{id: "supplier"}
	driverConn := &dispatcherConnSpy{id: "driver"}
	supplierHub := ws.NewHub("supplier", nil, nil)
	driverHub := ws.NewHub("driver", nil, nil)
	supplierHub.Subscribe("supplier:sup-1", supplierConn)
	driverHub.Subscribe("driver:drv-1", driverConn)

	dispatcher := NewNotificationDispatcher(DispatcherDeps{
		SupplierHub: supplierHub,
		DriverHub:   driverHub,
	})
	payload, _ := json.Marshal(map[string]any{
		"type":              events.EventAIRecommendationDecided,
		"trace_id":          "trace-ai",
		"supplier_id":       "sup-1",
		"recommendation_id": "rec-1",
	})
	if err := dispatcher.HandleEvent(context.Background(), kafka.Message{Value: payload}); err != nil {
		t.Fatalf("handle event: %v", err)
	}
	if len(supplierConn.messages) != 1 {
		t.Fatalf("supplier messages = %d, want 1", len(supplierConn.messages))
	}
	if len(driverConn.messages) != 0 {
		t.Fatalf("driver hub should not receive ai recommendation event")
	}
}

func TestNotificationDispatcher_TelemetryLocationFansSupplierAndDriver(t *testing.T) {
	t.Parallel()

	supplierConn := &dispatcherConnSpy{id: "supplier"}
	driverConn := &dispatcherConnSpy{id: "driver"}
	supplierHub := ws.NewHub("supplier", nil, nil)
	driverHub := ws.NewHub("driver", nil, nil)
	supplierHub.Subscribe("supplier:sup-1", supplierConn)
	driverHub.Subscribe("driver:drv-1", driverConn)

	dispatcher := NewNotificationDispatcher(DispatcherDeps{
		SupplierHub: supplierHub,
		DriverHub:   driverHub,
	})
	payload, _ := json.Marshal(map[string]any{
		"type":     events.EventDriverLocationUpdated,
		"trace_id": "trace-telemetry",
		"data": map[string]any{
			"driver_id":   "drv-1",
			"supplier_id": "sup-1",
			"lat":         41.3,
			"lng":         69.2,
		},
	})
	if err := dispatcher.HandleEvent(context.Background(), kafka.Message{Value: payload}); err != nil {
		t.Fatalf("handle event: %v", err)
	}
	if len(supplierConn.messages) != 1 || len(driverConn.messages) != 1 {
		t.Fatalf("supplier=%d driver=%d, want 1 each", len(supplierConn.messages), len(driverConn.messages))
	}
}

func TestNotificationDispatcher_TransferStateFansFactory(t *testing.T) {
	t.Parallel()

	supplierConn := &dispatcherConnSpy{id: "supplier"}
	factoryConn := &dispatcherConnSpy{id: "factory"}
	supplierHub := ws.NewHub("supplier", nil, nil)
	factoryHub := ws.NewHub("factory", nil, nil)
	supplierHub.Subscribe("supplier:sup-1", supplierConn)
	factoryHub.Subscribe("factory:fc-1", factoryConn)

	dispatcher := NewNotificationDispatcher(DispatcherDeps{
		SupplierHub: supplierHub,
		FactoryHub:  factoryHub,
	})
	payload, _ := json.Marshal(map[string]any{
		"type":        "TRANSFER_APPROVED",
		"trace_id":    "trace-transfer",
		"supplier_id": "sup-1",
		"factory_id":  "fc-1",
		"transfer_id": "tr-1",
		"driver_id":   "drv-1",
	})
	if err := dispatcher.HandleEvent(context.Background(), kafka.Message{Value: payload}); err != nil {
		t.Fatalf("handle event: %v", err)
	}
	if len(supplierConn.messages) != 1 || len(factoryConn.messages) != 1 {
		t.Fatalf("supplier=%d factory=%d, want 1 each", len(supplierConn.messages), len(factoryConn.messages))
	}
}

func TestNotificationDispatcher_OrderCompletedFansParties(t *testing.T) {
	t.Parallel()

	supplierConn := &dispatcherConnSpy{id: "supplier"}
	retailerConn := &dispatcherConnSpy{id: "retailer"}
	driverConn := &dispatcherConnSpy{id: "driver"}
	supplierHub := ws.NewHub("supplier", nil, nil)
	retailerHub := ws.NewHub("retailer", nil, nil)
	driverHub := ws.NewHub("driver", nil, nil)
	supplierHub.Subscribe("supplier:sup-1", supplierConn)
	retailerHub.Subscribe("retailer:ret-1", retailerConn)
	driverHub.Subscribe("driver:drv-1", driverConn)

	dispatcher := NewNotificationDispatcher(DispatcherDeps{
		SupplierHub: supplierHub,
		RetailerHub: retailerHub,
		DriverHub:   driverHub,
	})
	payload, _ := json.Marshal(map[string]any{
		"type":        "ORDER_COMPLETED",
		"trace_id":    "trace-complete",
		"order_id":    "ord-1",
		"supplier_id": "sup-1",
		"retailer_id": "ret-1",
		"driver_id":   "drv-1",
	})
	if err := dispatcher.HandleEvent(context.Background(), kafka.Message{Value: payload}); err != nil {
		t.Fatalf("handle event: %v", err)
	}
	if len(supplierConn.messages) != 1 || len(retailerConn.messages) != 1 || len(driverConn.messages) != 1 {
		t.Fatalf("supplier=%d retailer=%d driver=%d, want 1 each",
			len(supplierConn.messages), len(retailerConn.messages), len(driverConn.messages))
	}
}

func TestNotificationDispatcher_CommandEventFansSupplier(t *testing.T) {
	t.Parallel()

	supplierConn := &dispatcherConnSpy{id: "supplier"}
	supplierHub := ws.NewHub("supplier", nil, nil)
	supplierHub.Subscribe("supplier:sup-1", supplierConn)

	dispatcher := NewNotificationDispatcher(DispatcherDeps{SupplierHub: supplierHub})
	payload, _ := json.Marshal(map[string]any{
		"type":        events.EventCommandDispatched,
		"trace_id":    "trace-cmd",
		"supplier_id": "sup-1",
		"order_id":    "ord-1",
	})
	if err := dispatcher.HandleEvent(context.Background(), kafka.Message{Value: payload}); err != nil {
		t.Fatalf("handle event: %v", err)
	}
	if len(supplierConn.messages) != 1 {
		t.Fatalf("supplier messages = %d, want 1", len(supplierConn.messages))
	}
}

type stubPromotionAudience struct {
	ids []string
	err error
}

func (s stubPromotionAudience) EngagedRetailerIDs(_ context.Context, _ string) ([]string, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.ids, nil
}

func TestNotificationDispatcher_PromotionChangedAllScopeFansSupplierPromoAndEngagedRetailers(t *testing.T) {
	t.Parallel()

	supplierConn := &dispatcherConnSpy{id: "supplier"}
	promoWatcherConn := &dispatcherConnSpy{id: "promo-watcher"}
	engagedConn := &dispatcherConnSpy{id: "engaged"}
	otherConn := &dispatcherConnSpy{id: "other"}

	supplierHub := ws.NewHub("supplier", nil, nil)
	retailerHub := ws.NewHub("retailer", nil, nil)
	supplierHub.Subscribe("supplier:sup-1", supplierConn)
	retailerHub.Subscribe(ws.SupplierPromoRoom("sup-1"), promoWatcherConn)
	retailerHub.Subscribe("retailer:ret-1", engagedConn)
	retailerHub.Subscribe("retailer:ret-9", otherConn)

	dispatcher := NewNotificationDispatcher(DispatcherDeps{
		SupplierHub:       supplierHub,
		RetailerHub:       retailerHub,
		PromotionAudience: stubPromotionAudience{ids: []string{"ret-1", "ret-1", "ret-2"}},
	})

	payload, err := json.Marshal(map[string]any{
		"type":           events.EventPromotionChanged,
		"trace_id":       "trace-promo-all",
		"supplier_id":    "sup-1",
		"promotion_id":   "promo-1",
		"retailer_scope": "ALL",
		"action":         "updated",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := dispatcher.HandleEvent(context.Background(), kafka.Message{Value: payload}); err != nil {
		t.Fatalf("handle event: %v", err)
	}
	if len(supplierConn.messages) != 1 {
		t.Fatalf("supplier messages = %d, want 1", len(supplierConn.messages))
	}
	if len(promoWatcherConn.messages) != 1 {
		t.Fatalf("supplier-promo room messages = %d, want 1", len(promoWatcherConn.messages))
	}
	if len(engagedConn.messages) != 1 {
		t.Fatalf("engaged retailer messages = %d, want 1", len(engagedConn.messages))
	}
	if len(otherConn.messages) != 0 {
		t.Fatalf("unengaged retailer should not receive ALL-scope personal fanout")
	}
}

func TestNotificationDispatcher_PromotionChangedAllowlistFansOnlyListedRetailers(t *testing.T) {
	t.Parallel()

	supplierConn := &dispatcherConnSpy{id: "supplier"}
	allowConn := &dispatcherConnSpy{id: "allow"}
	otherConn := &dispatcherConnSpy{id: "other"}
	promoWatcherConn := &dispatcherConnSpy{id: "promo-watcher"}

	supplierHub := ws.NewHub("supplier", nil, nil)
	retailerHub := ws.NewHub("retailer", nil, nil)
	supplierHub.Subscribe("supplier:sup-1", supplierConn)
	retailerHub.Subscribe("retailer:ret-1", allowConn)
	retailerHub.Subscribe("retailer:ret-9", otherConn)
	retailerHub.Subscribe(ws.SupplierPromoRoom("sup-1"), promoWatcherConn)

	dispatcher := NewNotificationDispatcher(DispatcherDeps{
		SupplierHub:       supplierHub,
		RetailerHub:       retailerHub,
		PromotionAudience: stubPromotionAudience{ids: []string{"ret-9"}},
	})

	payload, _ := json.Marshal(map[string]any{
		"type":           events.EventPromotionChanged,
		"trace_id":       "trace-promo-allow",
		"supplier_id":    "sup-1",
		"promotion_id":   "promo-2",
		"retailer_scope": "ALLOWLIST",
		"retailer_ids":   []string{"ret-1"},
		"action":         "created",
	})
	if err := dispatcher.HandleEvent(context.Background(), kafka.Message{Value: payload}); err != nil {
		t.Fatalf("handle event: %v", err)
	}
	if len(supplierConn.messages) != 1 || len(allowConn.messages) != 1 {
		t.Fatalf("supplier=%d allow=%d, want 1 each", len(supplierConn.messages), len(allowConn.messages))
	}
	if len(otherConn.messages) != 0 || len(promoWatcherConn.messages) != 0 {
		t.Fatalf("ALLOWLIST should not fan supplier-promo room or unlisted retailers")
	}
}

func TestParseEnvelope_RejectsMalformedJSON(t *testing.T) {
	t.Parallel()
	if _, err := ParseEnvelope([]byte("not-json")); err == nil {
		t.Fatal("expected malformed envelope error")
	}
}
