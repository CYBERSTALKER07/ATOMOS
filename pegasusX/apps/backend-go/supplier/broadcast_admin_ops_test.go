package supplier

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
)

type broadcastTestConn struct {
	delivered int
}

func (c *broadcastTestConn) ID() string { return "broadcast-test" }

func (c *broadcastTestConn) Identity() auth.Claims { return auth.Claims{} }

func (c *broadcastTestConn) Send(_ context.Context, _ []byte) error {
	c.delivered++
	return nil
}

func TestBroadcastAdminMessageFansAllRoles(t *testing.T) {
	log := slog.Default()
	supplierHub := ws.NewHub("supplier", nil, log)
	retailerHub := ws.NewHub("retailer", nil, log)
	driverHub := ws.NewHub("driver", nil, log)
	warehouseHub := ws.NewHub("warehouse", nil, log)
	payloadHub := ws.NewHub("payload", nil, log)
	factoryHub := ws.NewHub("factory", nil, log)

	supplierConn := &broadcastTestConn{}
	retailerConn := &broadcastTestConn{}
	driverConn := &broadcastTestConn{}
	warehouseConn := &broadcastTestConn{}
	payloadConn := &broadcastTestConn{}
	factoryConn := &broadcastTestConn{}

	supplierHub.Subscribe("supplier:sup-1", supplierConn)
	retailerHub.Subscribe(ws.SupplierPromoRoom("sup-1"), retailerConn)
	driverHub.Subscribe("driver:drv-1", driverConn)
	warehouseHub.Subscribe("warehouse:wh-1", warehouseConn)
	payloadHub.Subscribe("payload:sup-1", payloadConn)
	factoryHub.Subscribe("factory:fc-1", factoryConn)

	repo := &onboardingTestRepo{
		drivers: []SupplierFleetDriver{{DriverID: "drv-1", IsActive: true}},
		topology: SupplierTopology{
			Warehouses: []WarehouseNode{{WarehouseID: "wh-1"}},
			Factories:  []FactoryNode{{FactoryID: "fc-1"}},
		},
	}

	svc := &Service{
		repo:               repo,
		portalSupplierHub:  supplierHub,
		portalRetailerHub:  retailerHub,
		portalDriverHub:    driverHub,
		portalWarehouseHub: warehouseHub,
		portalPayloadHub:   payloadHub,
		portalFactoryHub:   factoryHub,
	}

	payload := []byte(`{"type":"SUPPLIER_BROADCAST"}`)
	svc.broadcastAdminMessage(context.Background(), "sup-1", "ALL", payload)

	if supplierConn.delivered != 1 {
		t.Fatalf("supplier deliveries = %d want 1", supplierConn.delivered)
	}
	if retailerConn.delivered != 1 {
		t.Fatalf("retailer deliveries = %d want 1", retailerConn.delivered)
	}
	if driverConn.delivered != 1 {
		t.Fatalf("driver deliveries = %d want 1", driverConn.delivered)
	}
	if warehouseConn.delivered != 1 {
		t.Fatalf("warehouse deliveries = %d want 1", warehouseConn.delivered)
	}
	if payloadConn.delivered != 1 {
		t.Fatalf("payload deliveries = %d want 1", payloadConn.delivered)
	}
	if factoryConn.delivered != 1 {
		t.Fatalf("factory deliveries = %d want 1", factoryConn.delivered)
	}
}

func TestHandleBroadcast_NoSpanner503(t *testing.T) {
	svc := &Service{
		supplierID: "sup-1",
		now:        func() time.Time { return time.Now().UTC() },
	}
	body := []byte(`{"title":"Hold","body":"Yard closed","role":"DRIVER"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/broadcast", bytes.NewReader(body))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{SupplierID: "sup-1", Role: auth.RoleAdmin, Subject: "admin-1"}))
	rr := httptest.NewRecorder()
	svc.HandleBroadcast(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d want 503 body=%s", rr.Code, rr.Body.String())
	}
}

func TestSupplierBroadcastOutboxEventType(t *testing.T) {
	if events.EventSupplierBroadcast != "SUPPLIER_BROADCAST" {
		t.Fatalf("event type %q", events.EventSupplierBroadcast)
	}
}
