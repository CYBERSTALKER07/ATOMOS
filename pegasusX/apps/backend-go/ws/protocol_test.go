package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
)

func readJSONFrame(t *testing.T, conn *websocket.Conn) map[string]any {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, raw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read websocket frame: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal frame %q: %v", string(raw), err)
	}
	return out
}

func TestProtocol_PingAckAndPresenceLifecycle(t *testing.T) {
	t.Parallel()

	backend := cache.NewInMemoryBackend()
	driverHub := NewHub("driver", backend, nil)
	telemetryHub := NewHub("telemetry", backend, nil)

	router := chi.NewRouter()
	router.Use(testClaimsMiddleware(auth.Claims{
		Role:       auth.RoleDriver,
		Subject:    "drv-proto-1",
		SupplierID: "sup-1",
	}))
	RegisterRoutes(router, slog.Default(), testWSJWTSecret, nil, nil, nil, driverHub, nil, nil, nil, telemetryHub, nil, RegisterConfig{})

	server := httptest.NewServer(router)
	defer server.Close()

	conn := dialTestWebSocket(t, server.URL)
	defer conn.Close()

	// 1. Application-level ping -> pong
	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"ping","id":"hb-1"}`)); err != nil {
		t.Fatalf("write ping: %v", err)
	}
	pong := readJSONFrame(t, conn)
	if pong["type"] != "pong" || pong["id"] != "hb-1" {
		t.Fatalf("unexpected pong frame: %v", pong)
	}

	// Verify presence is ONLINE in Redis
	status, online := driverHub.GetPresence(context.Background(), auth.RoleDriver, "drv-proto-1")
	if !online || status != "ONLINE" {
		t.Fatalf("expected ONLINE presence, got status=%q online=%v", status, online)
	}

	// 2. Message acknowledgment -> ack:confirmed
	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"ack","event_id":"evt-900"}`)); err != nil {
		t.Fatalf("write ack: %v", err)
	}
	ackResp := readJSONFrame(t, conn)
	if ackResp["type"] != "ack:confirmed" || ackResp["event_id"] != "evt-900" {
		t.Fatalf("unexpected ack response: %v", ackResp)
	}

	// 3. Presence status transition -> AWAY
	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"presence","status":"away"}`)); err != nil {
		t.Fatalf("write presence away: %v", err)
	}
	presResp := readJSONFrame(t, conn)
	if presResp["type"] != "presence:updated" || presResp["status"] != "AWAY" {
		t.Fatalf("unexpected presence response: %v", presResp)
	}
	status, online = driverHub.GetPresence(context.Background(), auth.RoleDriver, "drv-proto-1")
	if !online || status != "AWAY" {
		t.Fatalf("expected AWAY presence in backend, got status=%q online=%v", status, online)
	}
}

func TestProtocol_DynamicRoomJoinLeaveAndAuthorization(t *testing.T) {
	t.Parallel()

	retailerHub := NewHub("retailer", nil, nil)
	platformAdminHub := NewHub("platform-admin", nil, nil)

	router := chi.NewRouter()
	router.Use(testClaimsMiddleware(auth.Claims{
		Role:    auth.RoleRetailer,
		Subject: "ret-org-1",
	}))
	RegisterRoutes(router, slog.Default(), testWSJWTSecret, nil, retailerHub, nil, nil, nil, nil, nil, nil, platformAdminHub, RegisterConfig{})

	server := httptest.NewServer(router)
	defer server.Close()

	conn := dialTestWebSocket(t, server.URL)
	defer conn.Close()

	// 1. Authorized dynamic join to supplier-promo:sup-77
	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"room:join","room":"supplier-promo:sup-77"}`)); err != nil {
		t.Fatalf("write room:join: %v", err)
	}
	joined := readJSONFrame(t, conn)
	if joined["type"] != "room:joined" || joined["room"] != "supplier-promo:sup-77" {
		t.Fatalf("expected room:joined for supplier-promo:sup-77, got %v", joined)
	}

	// Broadcast to supplier-promo:sup-77 and verify delivery
	retailerHub.Broadcast(context.Background(), "supplier-promo:sup-77", []byte(`{"type":"PROMOTION_CHANGED","supplier_id":"sup-77"}`))
	promoEvt := readJSONFrame(t, conn)
	if promoEvt["type"] != "PROMOTION_CHANGED" {
		t.Fatalf("expected PROMOTION_CHANGED broadcast, got %v", promoEvt)
	}

	// 2. Leave supplier-promo:sup-77
	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"room:leave","room":"supplier-promo:sup-77"}`)); err != nil {
		t.Fatalf("write room:leave: %v", err)
	}
	left := readJSONFrame(t, conn)
	if left["type"] != "room:left" || left["room"] != "supplier-promo:sup-77" {
		t.Fatalf("expected room:left, got %v", left)
	}

	// 3. Unauthorized room join attempt (cross-tenant retailer room and platform-admin room)
	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"room:join","room":"retailer:other-org-999"}`)); err != nil {
		t.Fatalf("write unauthorized room:join: %v", err)
	}
	errFrame := readJSONFrame(t, conn)
	if errFrame["type"] != "error" || errFrame["code"] != "unauthorized_room" {
		t.Fatalf("expected unauthorized_room error frame, got %v", errFrame)
	}

	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"room:join","room":"platform-admin"}`)); err != nil {
		t.Fatalf("write platform-admin room:join: %v", err)
	}
	errFrame2 := readJSONFrame(t, conn)
	if errFrame2["type"] != "error" || errFrame2["code"] != "unauthorized_room" {
		t.Fatalf("expected unauthorized_room error frame for platform-admin, got %v", errFrame2)
	}
}

func TestProtocol_InBandReplayRequest(t *testing.T) {
	t.Parallel()

	driverHub := NewHub("driver", nil, nil)
	ctx := context.Background()
	driverHub.Broadcast(ctx, "driver:drv-rep-1", []byte(`{"event_id":"evt-r1","type":"STOP_1"}`))
	driverHub.Broadcast(ctx, "driver:drv-rep-1", []byte(`{"event_id":"evt-r2","type":"STOP_2"}`))

	router := chi.NewRouter()
	router.Use(testClaimsMiddleware(auth.Claims{
		Role:    auth.RoleDriver,
		Subject: "drv-rep-1",
	}))
	RegisterRoutes(router, slog.Default(), testWSJWTSecret, nil, nil, nil, driverHub, nil, nil, nil, nil, nil, RegisterConfig{})

	server := httptest.NewServer(router)
	defer server.Close()

	conn := dialTestWebSocket(t, server.URL)
	defer conn.Close()

	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"replay","room":"driver:drv-rep-1","last_event_id":"evt-r1"}`)); err != nil {
		t.Fatalf("write replay request: %v", err)
	}

	replayedEvt := readJSONFrame(t, conn)
	if !strings.EqualFold(replayedEvt["event_id"].(string), "evt-r2") {
		t.Fatalf("expected replayed event evt-r2, got %v", replayedEvt)
	}

	doneFrame := readJSONFrame(t, conn)
	if doneFrame["type"] != "replay:completed" {
		t.Fatalf("expected replay:completed frame, got %v", doneFrame)
	}
}
