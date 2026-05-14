package kafka

import (
	"backend-go/auth"
	"backend-go/ws"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func newPayloaderHubServer(hub *ws.PayloaderHub, supplierID string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := &auth.PegasusClaims{UserID: supplierID, SupplierID: supplierID, Role: "PAYLOADER"}
		ctx := context.WithValue(r.Context(), auth.ClaimsContextKey, claims)
		hub.HandleConnection(w, r.WithContext(ctx))
	}))
}

func newRetailerHubServer(hub *ws.RetailerHub, retailerID string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := &auth.PegasusClaims{UserID: retailerID, Role: "RETAILER"}
		ctx := context.WithValue(r.Context(), auth.ClaimsContextKey, claims)
		hub.HandleConnection(w, r.WithContext(ctx))
	}))
}

func websocketURL(srv *httptest.Server) string {
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

func dialWebSocket(t *testing.T, url string) *websocket.Conn {
	t.Helper()
	if strings.Contains(url, "?") {
		url += "&sv=2"
	} else {
		url += "?sv=2"
	}
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("ws dial failed: %v", err)
	}
	return conn
}

func TestHandlePayloadSync_PushesTypedFrameOverPayloaderWebSocket(t *testing.T) {
	hub := ws.NewPayloaderHub()
	srv := newPayloaderHubServer(hub, "supplier-1")
	defer srv.Close()

	conn := dialWebSocket(t, websocketURL(srv))
	defer conn.Close()

	// Allow the hub to register the connection before the dispatcher pushes.
	time.Sleep(50 * time.Millisecond)

	data, err := json.Marshal(PayloadSyncEvent{
		SupplierID:  "supplier-1",
		WarehouseID: "warehouse-9",
		ManifestID:  "manifest-7",
		Reason:      EventManifestSealed,
		Timestamp:   time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("marshal payload sync event: %v", err)
	}

	handlePayloadSync(NotificationDeps{PayloaderHub: hub}, data)

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var got map[string]any
	if err := conn.ReadJSON(&got); err != nil {
		t.Fatalf("read websocket frame: %v", err)
	}

	if got["type"] != EventPayloadSync {
		t.Fatalf("type = %#v, want %q", got["type"], EventPayloadSync)
	}
	if got["channel"] != "SYNC" {
		t.Fatalf("channel = %#v, want %q", got["channel"], "SYNC")
	}
	if got["manifest_id"] != "manifest-7" {
		t.Fatalf("manifest_id = %#v, want manifest-7", got["manifest_id"])
	}
	if got["reason"] != EventManifestSealed {
		t.Fatalf("reason = %#v, want %q", got["reason"], EventManifestSealed)
	}
	if _, ok := got["supplier_id"]; ok {
		t.Fatal("websocket frame unexpectedly contains supplier_id")
	}
}

func TestHandleDeliverySessionUpdated_PushesTypedFrameOverRetailerWebSocket(t *testing.T) {
	hub := ws.NewRetailerHub()
	srv := newRetailerHubServer(hub, "retailer-1")
	defer srv.Close()

	conn := dialWebSocket(t, websocketURL(srv))
	defer conn.Close()

	// Allow the hub to register the connection before the dispatcher pushes.
	time.Sleep(50 * time.Millisecond)

	data, err := json.Marshal(DeliverySessionUpdatedEvent{
		SessionID:      "session-42",
		OrderID:        "order-42",
		RetailerID:     "retailer-1",
		State:          "SETTLEMENT_AWAIT",
		OriginalAmount: 150000,
		AdjustedAmount: 147500,
		FeeBasisPoints: 300,
		FeeAmount:      4425,
		Currency:       "UZS",
		Timestamp:      time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("marshal delivery session updated event: %v", err)
	}

	handleDeliverySessionUpdated(NotificationDeps{RetailerHub: hub}, data)

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var got map[string]any
	if err := conn.ReadJSON(&got); err != nil {
		t.Fatalf("read websocket frame: %v", err)
	}

	if got["type"] != EventDeliverySessionUpdated {
		t.Fatalf("type = %#v, want %q", got["type"], EventDeliverySessionUpdated)
	}
	if got["channel"] != "PUSH" {
		t.Fatalf("channel = %#v, want %q", got["channel"], "PUSH")
	}
	if got["order_id"] != "order-42" {
		t.Fatalf("order_id = %#v, want order-42", got["order_id"])
	}
	if got["session_id"] != "session-42" {
		t.Fatalf("session_id = %#v, want session-42", got["session_id"])
	}
	if got["state"] != "SETTLEMENT_AWAIT" {
		t.Fatalf("state = %#v, want SETTLEMENT_AWAIT", got["state"])
	}
	if got["currency"] != "UZS" {
		t.Fatalf("currency = %#v, want UZS", got["currency"])
	}
	if got["original_amount"] != float64(150000) {
		t.Fatalf("original_amount = %#v, want 150000", got["original_amount"])
	}
	if got["adjusted_amount"] != float64(147500) {
		t.Fatalf("adjusted_amount = %#v, want 147500", got["adjusted_amount"])
	}
	if got["fee_basis_points"] != float64(300) {
		t.Fatalf("fee_basis_points = %#v, want 300", got["fee_basis_points"])
	}
	if got["fee_amount"] != float64(4425) {
		t.Fatalf("fee_amount = %#v, want 4425", got["fee_amount"])
	}
}
