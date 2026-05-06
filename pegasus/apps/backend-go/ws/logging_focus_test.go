package ws

import (
	"backend-go/auth"
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func captureWSJSONLogs(t *testing.T) (*bytes.Buffer, func()) {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)
	return &buf, func() {
		slog.SetDefault(prev)
	}
}

func waitWSLogContains(t *testing.T, buf *bytes.Buffer, want string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(buf.String(), want) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected ws logs to contain %q, got logs: %s", want, buf.String())
}

func dialWSWithTrace(t *testing.T, url, traceID string) *websocket.Conn {
	t.Helper()
	headers := http.Header{"X-Trace-Id": []string{traceID}}
	conn, _, err := websocket.DefaultDialer.Dial(url, headers)
	if err != nil {
		t.Fatalf("WS dial with trace failed: %v", err)
	}
	return conn
}

func assertHubLifecycleLogs(
	t *testing.T,
	handler http.HandlerFunc,
	expectedConnectMsg string,
	expectedDisconnectMsg string,
	expectedHub string,
) {
	t.Helper()

	logs, restore := captureWSJSONLogs(t)
	defer restore()

	srv := httptest.NewServer(handler)
	defer srv.Close()

	traceID := "trace-ws-log-1"
	conn := dialWSWithTrace(t, wsURL(srv), traceID)
	time.Sleep(50 * time.Millisecond)
	_ = conn.Close()

	waitWSLogContains(t, logs, expectedConnectMsg)
	waitWSLogContains(t, logs, expectedDisconnectMsg)
	waitWSLogContains(t, logs, "\"trace_id\":\""+traceID+"\"")
	waitWSLogContains(t, logs, "\"hub\":\""+expectedHub+"\"")
}

func TestWSHubLifecycle_ConnectionLogsIncludeTraceID(t *testing.T) {
	tests := []struct {
		name                  string
		handler               http.HandlerFunc
		expectedConnectMsg    string
		expectedDisconnectMsg string
		expectedHub           string
	}{
		{
			name: "fleet",
			handler: func(w http.ResponseWriter, r *http.Request) {
				NewFleetHub().HandleConnection(w, r)
			},
			expectedConnectMsg:    "fleet hub client connected",
			expectedDisconnectMsg: "fleet hub client disconnected",
			expectedHub:           "fleet",
		},
		{
			name: "retailer",
			handler: func(w http.ResponseWriter, r *http.Request) {
				hub := NewRetailerHub()
				claims := &auth.PegasusClaims{UserID: "RET-LOG", Role: "RETAILER"}
				ctx := context.WithValue(r.Context(), auth.ClaimsContextKey, claims)
				hub.HandleConnection(w, r.WithContext(ctx))
			},
			expectedConnectMsg:    "retailer hub client connected",
			expectedDisconnectMsg: "retailer hub client disconnected",
			expectedHub:           "retailer",
		},
		{
			name: "driver",
			handler: func(w http.ResponseWriter, r *http.Request) {
				hub := NewDriverHub()
				claims := &auth.PegasusClaims{UserID: "DRV-LOG", Role: "DRIVER"}
				ctx := context.WithValue(r.Context(), auth.ClaimsContextKey, claims)
				hub.HandleConnection(w, r.WithContext(ctx))
			},
			expectedConnectMsg:    "driver hub client connected",
			expectedDisconnectMsg: "driver hub client disconnected",
			expectedHub:           "driver",
		},
		{
			name: "supplier",
			handler: func(w http.ResponseWriter, r *http.Request) {
				hub := NewSupplierHub()
				claims := &auth.PegasusClaims{UserID: "SUP-ADMIN", Role: "ADMIN", SupplierID: "SUP-LOG"}
				ctx := context.WithValue(r.Context(), auth.ClaimsContextKey, claims)
				hub.HandleConnection(w, r.WithContext(ctx))
			},
			expectedConnectMsg:    "supplier hub client connected",
			expectedDisconnectMsg: "supplier hub client disconnected",
			expectedHub:           "supplier",
		},
		{
			name: "payloader",
			handler: func(w http.ResponseWriter, r *http.Request) {
				hub := NewPayloaderHub()
				claims := &auth.PegasusClaims{UserID: "PAY-1", Role: "PAYLOADER", SupplierID: "SUP-LOG"}
				ctx := context.WithValue(r.Context(), auth.ClaimsContextKey, claims)
				hub.HandleConnection(w, r.WithContext(ctx))
			},
			expectedConnectMsg:    "payloader hub client connected",
			expectedDisconnectMsg: "payloader hub client disconnected",
			expectedHub:           "payloader",
		},
		{
			name: "warehouse",
			handler: func(w http.ResponseWriter, r *http.Request) {
				hub := NewWarehouseHub()
				claims := &auth.PegasusClaims{UserID: "WH-OPS", Role: "WAREHOUSE_ADMIN", WarehouseID: "WH-LOG"}
				ctx := context.WithValue(r.Context(), auth.ClaimsContextKey, claims)
				hub.HandleConnection(w, r.WithContext(ctx))
			},
			expectedConnectMsg:    "warehouse hub client connected",
			expectedDisconnectMsg: "warehouse hub client disconnected",
			expectedHub:           "warehouse",
		},
		{
			name: "factory",
			handler: func(w http.ResponseWriter, r *http.Request) {
				hub := NewFactoryHub()
				claims := &auth.PegasusClaims{UserID: "FAC-OPS", Role: "FACTORY", FactoryID: "FAC-LOG", SupplierID: "SUP-LOG"}
				ctx := context.WithValue(r.Context(), auth.ClaimsContextKey, claims)
				hub.HandleConnection(w, r.WithContext(ctx))
			},
			expectedConnectMsg:    "factory hub client connected",
			expectedDisconnectMsg: "factory hub client disconnected",
			expectedHub:           "factory",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertHubLifecycleLogs(t, tc.handler, tc.expectedConnectMsg, tc.expectedDisconnectMsg, tc.expectedHub)
		})
	}
}

func TestCheckWSOrigin_UnknownBlocked_LogsStructured(t *testing.T) {
	logs, restore := captureWSJSONLogs(t)
	defer restore()

	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://admin.void.pegasus.uz")

	r, _ := http.NewRequest("GET", "/ws", nil)
	r.Header.Set("Origin", "https://evil.example")

	if CheckWSOrigin(r) {
		t.Fatal("expected unknown origin to be blocked")
	}

	waitWSLogContains(t, logs, "websocket origin rejected")
	waitWSLogContains(t, logs, "\"hub\":\"fleet\"")
	waitWSLogContains(t, logs, "\"origin\":\"https://evil.example\"")
}

func TestConfigureKeepalive_ReturnsDoneChannel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade failed: %v", err)
			return
		}
		defer conn.Close()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	conn := dialWSWithTrace(t, wsURL(srv), "trace-keepalive-1")
	done := ConfigureKeepalive(conn)
	if done == nil {
		t.Fatal("ConfigureKeepalive returned nil done channel")
	}
	close(done)
	_ = conn.Close()
}
