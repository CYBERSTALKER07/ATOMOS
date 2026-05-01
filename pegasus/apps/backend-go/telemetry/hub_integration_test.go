package telemetry

import (
	"backend-go/auth"
	wsEvents "backend-go/ws"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

// newTestHub returns a fresh Hub with no Spanner/ProximityEngine dependencies.
func newTestHub() *Hub {
	return &Hub{
		Clients:    make(map[*websocket.Conn]*clientMeta),
		subscribed: make(map[string]bool),
	}
}

func telemetryServer(hub *Hub, userID, role string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := &auth.PegasusClaims{UserID: userID, Role: role}
		ctx := context.WithValue(r.Context(), auth.ClaimsContextKey, claims)
		hub.HandleConnection(w, r.WithContext(ctx))
	}))
}

func telemetryDial(t *testing.T, srv *httptest.Server) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("WS dial: %v", err)
	}
	return conn
}

func waitClients(t *testing.T, hub *Hub, want int) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		hub.RLock()
		n := len(hub.Clients)
		hub.RUnlock()
		if n >= want {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for %d clients, got %d", want, n)
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func waitNoClients(t *testing.T, hub *Hub) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		hub.RLock()
		n := len(hub.Clients)
		hub.RUnlock()
		if n == 0 {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for clients to drain, still have %d", n)
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func TestTelemetryHub_DriverSendsGPS_AdminReceives(t *testing.T) {
	hub := newTestHub()
	// Pre-populate driver→supplier cache so Spanner is not needed
	hub.driverSupplierCache.Store("DRV-001", "SUP-001")

	// Admin for SUP-001
	adminSrv := telemetryServer(hub, "SUP-001", "ADMIN")
	defer adminSrv.Close()
	adminConn := telemetryDial(t, adminSrv)
	defer adminConn.Close()
	waitClients(t, hub, 1)

	// Driver for DRV-001 → resolves to SUP-001 via cache
	driverSrv := telemetryServer(hub, "DRV-001", "DRIVER")
	defer driverSrv.Close()
	driverConn := telemetryDial(t, driverSrv)
	defer driverConn.Close()
	waitClients(t, hub, 2)

	// Driver sends GPS payload
	gps := GPSPayload{DriverID: "DRV-001", Latitude: 41.311, Longitude: 69.280, Timestamp: time.Now().Unix()}
	data, _ := json.Marshal(gps)
	if err := driverConn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("driver write: %v", err)
	}

	// Admin should receive the broadcast
	adminConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := adminConn.ReadMessage()
	if err != nil {
		t.Fatalf("admin read: %v", err)
	}

	var got GPSPayload
	json.Unmarshal(msg, &got)
	if got.DriverID != "DRV-001" {
		t.Errorf("DriverID = %q, want DRV-001", got.DriverID)
	}
	if got.Latitude != 41.311 {
		t.Errorf("Lat = %f, want 41.311", got.Latitude)
	}
}

func TestTelemetryHub_DifferentSupplierAdmin_NoReceive(t *testing.T) {
	hub := newTestHub()
	hub.driverSupplierCache.Store("DRV-A", "SUP-A")

	// Admin for SUP-B (different supplier)
	adminSrv := telemetryServer(hub, "SUP-B", "ADMIN")
	defer adminSrv.Close()
	adminConn := telemetryDial(t, adminSrv)
	defer adminConn.Close()
	waitClients(t, hub, 1)

	// Driver for SUP-A
	driverSrv := telemetryServer(hub, "DRV-A", "DRIVER")
	defer driverSrv.Close()
	driverConn := telemetryDial(t, driverSrv)
	defer driverConn.Close()
	waitClients(t, hub, 2)

	// Driver sends GPS
	gps := GPSPayload{DriverID: "DRV-A", Latitude: 41.0, Longitude: 69.0, Timestamp: 1}
	data, _ := json.Marshal(gps)
	driverConn.WriteMessage(websocket.TextMessage, data)

	// Admin for SUP-B should NOT receive — read should timeout
	adminConn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	_, _, err := adminConn.ReadMessage()
	if err == nil {
		t.Error("admin for different supplier should NOT receive GPS from DRV-A")
	}
}

func TestTelemetryHub_UnauthorizedRole_Rejected(t *testing.T) {
	hub := newTestHub()

	// Try connecting as RETAILER — should be rejected
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := &auth.PegasusClaims{UserID: "RET-001", Role: "RETAILER"}
		ctx := context.WithValue(r.Context(), auth.ClaimsContextKey, claims)
		hub.HandleConnection(w, r.WithContext(ctx))
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestTelemetryHub_NoClaims_Rejected(t *testing.T) {
	hub := newTestHub()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.HandleConnection(w, r) // No claims in context
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestTelemetryHub_DriverDisconnect_Removed(t *testing.T) {
	hub := newTestHub()
	hub.driverSupplierCache.Store("DRV-DC", "SUP-DC")

	srv := telemetryServer(hub, "DRV-DC", "DRIVER")
	defer srv.Close()

	conn := telemetryDial(t, srv)
	waitClients(t, hub, 1)

	conn.Close()
	time.Sleep(100 * time.Millisecond)

	hub.RLock()
	count := len(hub.Clients)
	hub.RUnlock()
	if count != 0 {
		t.Fatalf("after disconnect: clients = %d, want 0", count)
	}
}

func TestTelemetryHub_BroadcastOrderStateChange(t *testing.T) {
	hub := newTestHub()

	// Admin for SUP-X
	adminSrv := telemetryServer(hub, "SUP-X", "ADMIN")
	defer adminSrv.Close()
	adminConn := telemetryDial(t, adminSrv)
	defer adminConn.Close()
	waitClients(t, hub, 1)

	// Admin for SUP-Y (should NOT receive)
	adminSrv2 := telemetryServer(hub, "SUP-Y", "ADMIN")
	defer adminSrv2.Close()
	adminConn2 := telemetryDial(t, adminSrv2)
	defer adminConn2.Close()
	waitClients(t, hub, 2)

	// Broadcast state change for SUP-X
	hub.BroadcastOrderStateChange("SUP-X", "ORD-999", "DISPATCHED", "DRV-777")

	// SUP-X admin receives
	adminConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := adminConn.ReadMessage()
	if err != nil {
		t.Fatalf("admin SUP-X read: %v", err)
	}
	var event map[string]interface{}
	json.Unmarshal(msg, &event)
	if event["type"] != "ORDER_STATE_CHANGED" {
		t.Errorf("type = %v, want ORDER_STATE_CHANGED", event["type"])
	}
	if event["order_id"] != "ORD-999" {
		t.Errorf("order_id = %v, want ORD-999", event["order_id"])
	}
	if event["state"] != "DISPATCHED" {
		t.Errorf("state = %v, want DISPATCHED", event["state"])
	}
	if event["driver_id"] != "DRV-777" {
		t.Errorf("driver_id = %v, want DRV-777", event["driver_id"])
	}

	// SUP-Y admin should NOT receive
	adminConn2.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	_, _, err2 := adminConn2.ReadMessage()
	if err2 == nil {
		t.Error("admin SUP-Y should NOT receive ORDER_STATE_CHANGED for SUP-X")
	}
}

func TestTelemetryHub_MultipleAdminsSameSupplier(t *testing.T) {
	hub := newTestHub()

	// Two admins for same supplier
	srv1 := telemetryServer(hub, "SUP-SAME", "ADMIN")
	defer srv1.Close()
	c1 := telemetryDial(t, srv1)
	defer c1.Close()
	waitClients(t, hub, 1)

	srv2 := telemetryServer(hub, "SUP-SAME", "ADMIN")
	defer srv2.Close()
	c2 := telemetryDial(t, srv2)
	defer c2.Close()
	waitClients(t, hub, 2)

	hub.BroadcastOrderStateChange("SUP-SAME", "ORD-DUAL", "ARRIVED", "DRV-DUAL")

	for i, c := range []*websocket.Conn{c1, c2} {
		c.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, msg, err := c.ReadMessage()
		if err != nil {
			t.Fatalf("admin %d read: %v", i, err)
		}
		var event map[string]interface{}
		json.Unmarshal(msg, &event)
		if event["order_id"] != "ORD-DUAL" {
			t.Errorf("admin %d: order_id = %v, want ORD-DUAL", i, event["order_id"])
		}
	}
}

func TestGraceReconnectDelay_UsesGraceDeadlineBudget(t *testing.T) {
	oldCloseAfter := graceReconnectCloseAfter
	graceReconnectCloseAfter = 30 * time.Second
	t.Cleanup(func() {
		graceReconnectCloseAfter = oldCloseAfter
	})

	if got := graceReconnectDelay(nil); got != 30*time.Second {
		t.Fatalf("graceReconnectDelay(nil) = %s, want %s", got, 30*time.Second)
	}

	claims := &auth.PegasusClaims{GraceDeadline: time.Now().Add(5 * time.Second)}
	got := graceReconnectDelay(claims)
	if got > 5*time.Second {
		t.Fatalf("graceReconnectDelay(claims) = %s, want <= 5s", got)
	}
	if got <= 0 {
		t.Fatalf("graceReconnectDelay(claims) = %s, want > 0", got)
	}
}

func TestTelemetryHub_GraceDriver_RefreshNeededThenForcedClose(t *testing.T) {
	oldCloseAfter := graceReconnectCloseAfter
	graceReconnectCloseAfter = 120 * time.Millisecond
	t.Cleanup(func() {
		graceReconnectCloseAfter = oldCloseAfter
	})

	hub := newTestHub()
	hub.driverSupplierCache.Store("DRV-GRACE", "SUP-GRACE")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := &auth.PegasusClaims{
			UserID:        "DRV-GRACE",
			Role:          "DRIVER",
			GracePeriod:   true,
			GraceDeadline: time.Now().Add(1 * time.Hour),
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-30 * time.Minute)),
			},
		}
		ctx := context.WithValue(r.Context(), auth.ClaimsContextKey, claims)
		hub.HandleConnection(w, r.WithContext(ctx))
	}))
	defer srv.Close()

	conn := telemetryDial(t, srv)
	defer conn.Close()
	waitClients(t, hub, 1)

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read TOKEN_REFRESH_NEEDED: %v", err)
	}
	var frame map[string]interface{}
	if err := json.Unmarshal(msg, &frame); err != nil {
		t.Fatalf("unmarshal refresh frame: %v", err)
	}
	if frame["type"] != wsEvents.EventTokenRefreshNeeded {
		t.Fatalf("type = %v, want %s", frame["type"], wsEvents.EventTokenRefreshNeeded)
	}
	if frame["required_action"] != "REFRESH_TOKEN_AND_RECONNECT" {
		t.Fatalf("required_action = %v, want REFRESH_TOKEN_AND_RECONNECT", frame["required_action"])
	}

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err = conn.ReadMessage()
	if err == nil {
		t.Fatal("expected forced close after grace reconnect window")
	}
	if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
		t.Fatalf("expected forced close, got read timeout: %v", err)
	}

	waitNoClients(t, hub)
}
