package ws

import (
	"backend-go/auth"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// ═══════════════════════════════════════════════════════════════════════════════
// FleetHub Integration Tests
// ═══════════════════════════════════════════════════════════════════════════════

func newFleetHubServer(hub *FleetHub) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(hub.HandleConnection))
}

func wsURL(srv *httptest.Server) string {
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

func dial(t *testing.T, url string) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("WS dial failed: %v", err)
	}
	return conn
}

func TestFleetHub_ConnectAndRegister(t *testing.T) {
	hub := NewFleetHub()
	srv := newFleetHubServer(hub)
	defer srv.Close()

	c := dial(t, wsURL(srv))
	defer c.Close()

	// Give HandleConnection time to register
	time.Sleep(50 * time.Millisecond)

	hub.mu.Lock()
	count := len(hub.clients)
	hub.mu.Unlock()

	if count != 1 {
		t.Fatalf("clients = %d, want 1", count)
	}
}

func TestFleetHub_BroadcastDelivers(t *testing.T) {
	hub := NewFleetHub()
	srv := newFleetHubServer(hub)
	defer srv.Close()

	// Connect two clients
	c1 := dial(t, wsURL(srv))
	defer c1.Close()
	c2 := dial(t, wsURL(srv))
	defer c2.Close()

	time.Sleep(50 * time.Millisecond)

	update := LocationUpdate{DriverID: "DRV-001", Latitude: 41.311, Longitude: 69.279}
	hub.Broadcast(update)

	// Both clients should receive
	for _, c := range []*websocket.Conn{c1, c2} {
		c.SetReadDeadline(time.Now().Add(2 * time.Second))
		var got LocationUpdate
		if err := c.ReadJSON(&got); err != nil {
			t.Fatalf("read failed: %v", err)
		}
		if got.DriverID != "DRV-001" {
			t.Errorf("DriverID = %q, want DRV-001", got.DriverID)
		}
		if got.Latitude != 41.311 || got.Longitude != 69.279 {
			t.Errorf("coords = (%.3f, %.3f), want (41.311, 69.279)", got.Latitude, got.Longitude)
		}
	}
}

func TestFleetHub_DriverSendTriggersAdminReceive(t *testing.T) {
	hub := NewFleetHub()
	srv := newFleetHubServer(hub)
	defer srv.Close()

	admin := dial(t, wsURL(srv))
	defer admin.Close()
	driver := dial(t, wsURL(srv))
	defer driver.Close()

	time.Sleep(50 * time.Millisecond)

	// Driver sends a GPS ping
	gps := LocationUpdate{DriverID: "DRV-002", Latitude: 41.300, Longitude: 69.250}
	if err := driver.WriteJSON(gps); err != nil {
		t.Fatalf("driver write: %v", err)
	}

	// Admin should receive the broadcast
	admin.SetReadDeadline(time.Now().Add(2 * time.Second))
	var got LocationUpdate
	if err := admin.ReadJSON(&got); err != nil {
		t.Fatalf("admin read: %v", err)
	}
	if got.DriverID != "DRV-002" {
		t.Errorf("DriverID = %q, want DRV-002", got.DriverID)
	}
}

func TestFleetHub_DisconnectEvicts(t *testing.T) {
	hub := NewFleetHub()
	srv := newFleetHubServer(hub)
	defer srv.Close()

	c := dial(t, wsURL(srv))
	time.Sleep(50 * time.Millisecond)

	hub.mu.Lock()
	before := len(hub.clients)
	hub.mu.Unlock()
	if before != 1 {
		t.Fatalf("before close: clients = %d, want 1", before)
	}

	c.Close()
	time.Sleep(100 * time.Millisecond)

	hub.mu.Lock()
	after := len(hub.clients)
	hub.mu.Unlock()
	if after != 0 {
		t.Fatalf("after close: clients = %d, want 0", after)
	}
}

func TestFleetHub_MultipleClients(t *testing.T) {
	hub := NewFleetHub()
	srv := newFleetHubServer(hub)
	defer srv.Close()

	conns := make([]*websocket.Conn, 5)
	for i := range conns {
		conns[i] = dial(t, wsURL(srv))
		defer conns[i].Close()
	}
	time.Sleep(50 * time.Millisecond)

	hub.mu.Lock()
	count := len(hub.clients)
	hub.mu.Unlock()
	if count != 5 {
		t.Fatalf("clients = %d, want 5", count)
	}

	update := LocationUpdate{DriverID: "DRV-MULTI", Latitude: 1, Longitude: 2}
	hub.Broadcast(update)

	for i, c := range conns {
		c.SetReadDeadline(time.Now().Add(2 * time.Second))
		var got LocationUpdate
		if err := c.ReadJSON(&got); err != nil {
			t.Fatalf("client %d read: %v", i, err)
		}
		if got.DriverID != "DRV-MULTI" {
			t.Errorf("client %d: DriverID = %q, want DRV-MULTI", i, got.DriverID)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// RetailerHub Integration Tests
// ═══════════════════════════════════════════════════════════════════════════════

func newRetailerHubServer(hub *RetailerHub, retailerID string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := &auth.LabClaims{UserID: retailerID, Role: "RETAILER"}
		ctx := context.WithValue(r.Context(), auth.ClaimsContextKey, claims)
		hub.HandleConnection(w, r.WithContext(ctx))
	}))
}

func waitConnected(t *testing.T, hub *RetailerHub, id string) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		if hub.IsConnected(id) {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for %s to connect", id)
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func TestRetailerHub_ConnectAndPush(t *testing.T) {
	hub := NewRetailerHub()
	srv := newRetailerHubServer(hub, "RET-001")
	defer srv.Close()

	c := dial(t, wsURL(srv))
	defer c.Close()

	waitConnected(t, hub, "RET-001")

	payload := ApproachPayload{
		Type:    "DRIVER_APPROACHING",
		OrderID: "ORD-100",
	}
	ok := hub.PushToRetailer("RET-001", payload)
	if !ok {
		t.Fatal("PushToRetailer returned false")
	}

	c.SetReadDeadline(time.Now().Add(2 * time.Second))
	var got ApproachPayload
	if err := c.ReadJSON(&got); err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Type != "DRIVER_APPROACHING" || got.OrderID != "ORD-100" {
		t.Errorf("unexpected payload: %+v", got)
	}
}

func TestRetailerHub_PushNoConnections(t *testing.T) {
	hub := NewRetailerHub()
	ok := hub.PushToRetailer("GHOST-001", ApproachPayload{Type: "TEST"})
	if ok {
		t.Error("PushToRetailer should return false for unconnected retailer")
	}
}

func TestRetailerHub_MultipleConnectionsSameRetailer(t *testing.T) {
	hub := NewRetailerHub()
	srv := newRetailerHubServer(hub, "RET-002")
	defer srv.Close()

	c1 := dial(t, wsURL(srv))
	defer c1.Close()
	c2 := dial(t, wsURL(srv))
	defer c2.Close()

	// Wait for both connections
	deadline := time.After(2 * time.Second)
	for {
		hub.mu.RLock()
		count := len(hub.clients["RET-002"])
		hub.mu.RUnlock()
		if count >= 2 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for 2 connections")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	payload := ApproachPayload{Type: "DRIVER_APPROACHING", OrderID: "ORD-MULTI"}
	hub.PushToRetailer("RET-002", payload)

	for i, c := range []*websocket.Conn{c1, c2} {
		c.SetReadDeadline(time.Now().Add(2 * time.Second))
		var got ApproachPayload
		if err := c.ReadJSON(&got); err != nil {
			t.Fatalf("conn %d read: %v", i, err)
		}
		if got.OrderID != "ORD-MULTI" {
			t.Errorf("conn %d: OrderID = %q, want ORD-MULTI", i, got.OrderID)
		}
	}
}

func TestRetailerHub_DisconnectOneKeepsOther(t *testing.T) {
	hub := NewRetailerHub()
	srv := newRetailerHubServer(hub, "RET-003")
	defer srv.Close()

	c1 := dial(t, wsURL(srv))
	c2 := dial(t, wsURL(srv))
	defer c2.Close()

	// Wait for both
	deadline := time.After(2 * time.Second)
	for {
		hub.mu.RLock()
		count := len(hub.clients["RET-003"])
		hub.mu.RUnlock()
		if count >= 2 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Close first connection
	c1.Close()
	time.Sleep(100 * time.Millisecond)

	// Push should still work for second connection
	ok := hub.PushToRetailer("RET-003", ApproachPayload{Type: "STILL_ALIVE", OrderID: "ORD-X"})
	if !ok {
		t.Fatal("PushToRetailer returned false after one disconnect")
	}

	c2.SetReadDeadline(time.Now().Add(2 * time.Second))
	var got ApproachPayload
	if err := c2.ReadJSON(&got); err != nil {
		t.Fatalf("c2 read: %v", err)
	}
	if got.OrderID != "ORD-X" {
		t.Errorf("OrderID = %q, want ORD-X", got.OrderID)
	}
}

func TestRetailerHub_NoClaims_QueryParamRejected(t *testing.T) {
	hub := NewRetailerHub()
	// Server WITHOUT injecting claims — query param must not grant identity.
	srv := httptest.NewServer(http.HandlerFunc(hub.HandleConnection))
	defer srv.Close()

	_, _, err := websocket.DefaultDialer.Dial(wsURL(srv)+"?retailer_id=RET-QP", nil)
	if err == nil {
		t.Fatal("expected websocket dial to fail without authenticated claims")
	}
}

func TestRetailerHub_NoClaims_NoQueryParam_401(t *testing.T) {
	hub := NewRetailerHub()
	srv := httptest.NewServer(http.HandlerFunc(hub.HandleConnection))
	defer srv.Close()

	// Direct HTTP request without WS upgrade to check 401
	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// DriverHub Integration Tests
// ═══════════════════════════════════════════════════════════════════════════════

func newDriverHubServer(hub *DriverHub, driverID string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := &auth.LabClaims{UserID: driverID, Role: "DRIVER"}
		ctx := context.WithValue(r.Context(), auth.ClaimsContextKey, claims)
		hub.HandleConnection(w, r.WithContext(ctx))
	}))
}

func waitDriverConnected(t *testing.T, hub *DriverHub, id string) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		hub.mu.RLock()
		conns := hub.clients[id]
		hub.mu.RUnlock()
		if len(conns) > 0 {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for driver %s to connect", id)
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func TestDriverHub_ConnectAndPush(t *testing.T) {
	hub := NewDriverHub()
	srv := newDriverHubServer(hub, "DRV-001")
	defer srv.Close()

	c := dial(t, wsURL(srv))
	defer c.Close()

	waitDriverConnected(t, hub, "DRV-001")

	payload := DriverPayload{
		Type:    "PAYMENT_SETTLED",
		OrderID: "ORD-200",
		Amount:  500000,
		Message: "Payment received",
	}
	ok := hub.PushToDriver("DRV-001", payload)
	if !ok {
		t.Fatal("PushToDriver returned false")
	}

	c.SetReadDeadline(time.Now().Add(2 * time.Second))
	var got DriverPayload
	if err := c.ReadJSON(&got); err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Type != "PAYMENT_SETTLED" || got.OrderID != "ORD-200" || got.Amount != 500000 {
		t.Errorf("unexpected payload: %+v", got)
	}
}

func TestDriverHub_PushNoConnections(t *testing.T) {
	hub := NewDriverHub()
	ok := hub.PushToDriver("GHOST-DRV", DriverPayload{Type: "TEST"})
	if ok {
		t.Error("PushToDriver should return false for unconnected driver")
	}
}

func TestDriverHub_MultipleConnections(t *testing.T) {
	hub := NewDriverHub()
	srv := newDriverHubServer(hub, "DRV-002")
	defer srv.Close()

	c1 := dial(t, wsURL(srv))
	defer c1.Close()
	c2 := dial(t, wsURL(srv))
	defer c2.Close()

	deadline := time.After(2 * time.Second)
	for {
		hub.mu.RLock()
		count := len(hub.clients["DRV-002"])
		hub.mu.RUnlock()
		if count >= 2 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	payload := DriverPayload{Type: "PAYMENT_FAILED", OrderID: "ORD-FAIL"}
	hub.PushToDriver("DRV-002", payload)

	for i, c := range []*websocket.Conn{c1, c2} {
		c.SetReadDeadline(time.Now().Add(2 * time.Second))
		var got DriverPayload
		if err := c.ReadJSON(&got); err != nil {
			t.Fatalf("conn %d read: %v", i, err)
		}
		if got.OrderID != "ORD-FAIL" {
			t.Errorf("conn %d: OrderID = %q, want ORD-FAIL", i, got.OrderID)
		}
	}
}

func TestDriverHub_DisconnectCleanup(t *testing.T) {
	hub := NewDriverHub()
	srv := newDriverHubServer(hub, "DRV-003")
	defer srv.Close()

	c := dial(t, wsURL(srv))
	waitDriverConnected(t, hub, "DRV-003")

	c.Close()
	time.Sleep(100 * time.Millisecond)

	hub.mu.RLock()
	count := len(hub.clients["DRV-003"])
	hub.mu.RUnlock()
	if count != 0 {
		t.Fatalf("after close: connections = %d, want 0", count)
	}

	ok := hub.PushToDriver("DRV-003", DriverPayload{})
	if ok {
		t.Error("PushToDriver should return false after disconnect")
	}
}

func TestDriverHub_NoClaims_401(t *testing.T) {
	hub := NewDriverHub()
	srv := httptest.NewServer(http.HandlerFunc(hub.HandleConnection))
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

func TestDriverHub_QueryParamRejected(t *testing.T) {
	hub := NewDriverHub()
	srv := httptest.NewServer(http.HandlerFunc(hub.HandleConnection))
	defer srv.Close()

	_, _, err := websocket.DefaultDialer.Dial(wsURL(srv)+"?driver_id=DRV-QP", nil)
	if err == nil {
		t.Fatal("expected websocket dial to fail without authenticated claims")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Wire Format Contract Tests
// ═══════════════════════════════════════════════════════════════════════════════

func TestApproachPayload_JSON(t *testing.T) {
	p := ApproachPayload{
		Type:            "DRIVER_APPROACHING",
		OrderID:         "ORD-1",
		SupplierID:      "SUP-1",
		SupplierName:    "Nestle Uzb",
		RetailerID:      "RET-1",
		DeliveryToken:   "tok-abc-def",
		DriverLatitude:  41.311158,
		DriverLongitude: 69.279737,
	}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}

	var decoded map[string]interface{}
	json.Unmarshal(data, &decoded)

	checks := map[string]interface{}{
		"type":             "DRIVER_APPROACHING",
		"order_id":         "ORD-1",
		"delivery_token":   "tok-abc-def",
		"supplier_id":      "SUP-1",
		"retailer_id":      "RET-1",
		"driver_latitude":  41.311158,
		"driver_longitude": 69.279737,
	}
	for key, want := range checks {
		got := decoded[key]
		if got != want {
			t.Errorf("%s = %v, want %v", key, got, want)
		}
	}
}

func TestDriverPayload_JSON(t *testing.T) {
	p := DriverPayload{
		Type:    "PAYMENT_SETTLED",
		OrderID: "ORD-42",
		Amount:  1500000,
		Message: "Payment received. Tap Complete.",
	}
	data, _ := json.Marshal(p)
	var decoded map[string]interface{}
	json.Unmarshal(data, &decoded)

	if decoded["type"] != "PAYMENT_SETTLED" {
		t.Errorf("type = %v", decoded["type"])
	}
	if decoded["order_id"] != "ORD-42" {
		t.Errorf("order_id = %v", decoded["order_id"])
	}
	if decoded["amount"] != float64(1500000) {
		t.Errorf("amount = %v", decoded["amount"])
	}
}
