package ws

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

const testWSJWTSecret = "ws-test-secret"

func TestRegisterRoutesRejectsWhenHubAtCapacity(t *testing.T) {
	t.Parallel()

	driverHub := NewHubWithLimits("driver", nil, nil, HubLimits{MaxTotal: 1})
	telemetryHub := NewHubWithLimits("telemetry", nil, nil, HubLimits{MaxTotal: 1})
	driverHub.Subscribe("driver:drv-1", newTestConnection("ghost"))

	router := chi.NewRouter()
	router.Use(testClaimsMiddleware(auth.Claims{
		Role:       auth.RoleDriver,
		Subject:    "drv-1",
		SupplierID: "sup-1",
	}))
	RegisterRoutes(router, slog.Default(), testWSJWTSecret, false, nil, nil, nil, nil, driverHub, nil, nil, nil, telemetryHub, RegisterConfig{})

	server := httptest.NewServer(router)
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/v1/ws"
	_, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if err == nil {
		t.Fatal("expected dial failure when hub is at capacity")
	}
	if resp == nil {
		t.Fatal("expected HTTP response on failed upgrade")
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", resp.StatusCode)
	}
	if resp.Header.Get("Retry-After") == "" {
		t.Fatal("Retry-After header missing on capacity rejection")
	}
}

func TestRegisterRoutesSubscribesSupplierToTelemetryRoom(t *testing.T) {
	t.Parallel()

	telemetryHub := NewHub("telemetry", nil, nil)
	router := chi.NewRouter()
	router.Use(testClaimsMiddleware(auth.Claims{
		Role:       auth.RoleAdmin,
		Subject:    "admin-1",
		SupplierID: "sup-1",
	}))
	RegisterRoutes(router, slog.Default(), testWSJWTSecret, false, nil, nil, nil, nil, nil, nil, nil, nil, telemetryHub, RegisterConfig{})

	server := httptest.NewServer(router)
	defer server.Close()

	conn := dialTestWebSocket(t, server.URL)
	defer conn.Close()

	telemetryHub.Broadcast(context.Background(), "telemetry:supplier:sup-1", []byte("supplier-location"))
	assertWebSocketMessage(t, conn, "supplier-location")
}

func TestRegisterRoutesSubscribesDriverToTelemetryRoom(t *testing.T) {
	t.Parallel()

	telemetryHub := NewHub("telemetry", nil, nil)
	router := chi.NewRouter()
	router.Use(testClaimsMiddleware(auth.Claims{
		Role:       auth.RoleDriver,
		Subject:    "drv-1",
		SupplierID: "sup-1",
	}))
	RegisterRoutes(router, slog.Default(), testWSJWTSecret, false, nil, nil, nil, nil, nil, nil, nil, nil, telemetryHub, RegisterConfig{})

	server := httptest.NewServer(router)
	defer server.Close()

	conn := dialTestWebSocket(t, server.URL)
	defer conn.Close()

	telemetryHub.Broadcast(context.Background(), "telemetry:driver:drv-1", []byte("driver-location"))
	assertWebSocketMessage(t, conn, "driver-location")
}

func TestRegisterRoutesTelemetryDriverReconnectChurn(t *testing.T) {
	t.Parallel()

	telemetryHub := NewHub("telemetry", nil, nil)
	router := chi.NewRouter()
	router.Use(testClaimsMiddleware(auth.Claims{
		Role:       auth.RoleDriver,
		Subject:    "drv-1",
		SupplierID: "sup-1",
	}))
	RegisterRoutes(router, slog.Default(), testWSJWTSecret, false, nil, nil, nil, nil, nil, nil, nil, nil, telemetryHub, RegisterConfig{})

	server := httptest.NewServer(router)
	defer server.Close()

	for i := 0; i < 12; i++ {
		conn := dialTestWebSocket(t, server.URL)
		payload := "driver-location-" + strconv.Itoa(i)
		telemetryHub.Broadcast(context.Background(), "telemetry:driver:drv-1", []byte(payload))
		assertWebSocketMessage(t, conn, payload)
		if err := conn.Close(); err != nil {
			t.Fatalf("close websocket: %v", err)
		}
		waitForHubConnections(t, telemetryHub, 0)
	}
}

func TestRegisterRoutesTelemetrySupplierReconnectChurn(t *testing.T) {
	t.Parallel()

	telemetryHub := NewHub("telemetry", nil, nil)
	router := chi.NewRouter()
	router.Use(testClaimsMiddleware(auth.Claims{
		Role:       auth.RoleAdmin,
		Subject:    "admin-1",
		SupplierID: "sup-1",
	}))
	RegisterRoutes(router, slog.Default(), testWSJWTSecret, false, nil, nil, nil, nil, nil, nil, nil, nil, telemetryHub, RegisterConfig{})

	server := httptest.NewServer(router)
	defer server.Close()

	for i := 0; i < 12; i++ {
		conn := dialTestWebSocket(t, server.URL)
		payload := "supplier-location-" + strconv.Itoa(i)
		telemetryHub.Broadcast(context.Background(), "telemetry:supplier:sup-1", []byte(payload))
		assertWebSocketMessage(t, conn, payload)
		if err := conn.Close(); err != nil {
			t.Fatalf("close websocket: %v", err)
		}
		waitForHubConnections(t, telemetryHub, 0)
	}
}

func TestRegisterRoutesAcceptsSignedQueryToken(t *testing.T) {
	t.Parallel()

	telemetryHub := NewHub("telemetry", nil, nil)
	router := chi.NewRouter()
	RegisterRoutes(router, slog.Default(), testWSJWTSecret, false, nil, nil, nil, nil, nil, nil, nil, nil, telemetryHub, RegisterConfig{})

	server := httptest.NewServer(router)
	defer server.Close()

	token, err := auth.Issue(auth.Claims{
		Role:       auth.RoleAdmin,
		Subject:    "admin-1",
		SupplierID: "sup-1",
	}, auth.IssueOptions{
		Secret: testWSJWTSecret,
		TTL:    5 * time.Minute,
		Now:    func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	conn := dialTestWebSocketWithQuery(t, server.URL, "token="+token)
	defer conn.Close()

	telemetryHub.Broadcast(context.Background(), "telemetry:supplier:sup-1", []byte("supplier-location"))
	assertWebSocketMessage(t, conn, "supplier-location")
}

func testClaimsMiddleware(claims auth.Claims) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(auth.WithClaims(r.Context(), claims)))
		})
	}
}

func dialTestWebSocket(t *testing.T, serverURL string) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(serverURL, "http") + "/v1/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	return conn
}

func dialTestWebSocketWithQuery(t *testing.T, serverURL string, rawQuery string) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(serverURL, "http") + "/v1/ws"
	if rawQuery != "" {
		url += "?" + rawQuery
	}
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	return conn
}

func assertWebSocketMessage(t *testing.T, conn *websocket.Conn, want string) {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	_, raw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read websocket message: %v", err)
	}
	if string(raw) != want {
		t.Fatalf("message = %q, want %q", string(raw), want)
	}
}

func waitForHubConnections(t *testing.T, hub *Hub, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if hub.Stats().Connections == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("connections = %d, want %d", hub.Stats().Connections, want)
}
