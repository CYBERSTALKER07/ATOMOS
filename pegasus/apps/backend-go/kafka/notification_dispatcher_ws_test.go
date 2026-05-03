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

func websocketURL(srv *httptest.Server) string {
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

func dialWebSocket(t *testing.T, url string) *websocket.Conn {
	t.Helper()
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
