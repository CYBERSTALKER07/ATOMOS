package telemetry

import (
	"backend-go/auth"
	wsEvents "backend-go/ws"
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

func captureTelemetryJSONLogs(t *testing.T) (*bytes.Buffer, func()) {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)
	return &buf, func() {
		slog.SetDefault(prev)
	}
}

func waitTelemetryLogContains(t *testing.T, buf *bytes.Buffer, want string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(buf.String(), want) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected telemetry logs to contain %q, got logs: %s", want, buf.String())
}

func dialTelemetryWithTrace(t *testing.T, srv *httptest.Server, traceID string) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	headers := http.Header{"X-Trace-Id": []string{traceID}}
	conn, _, err := websocket.DefaultDialer.Dial(url, headers)
	if err != nil {
		t.Fatalf("telemetry ws dial with trace failed: %v", err)
	}
	return conn
}

func TestTelemetryHub_ConnectionLifecycleLogsIncludeTraceID(t *testing.T) {
	logs, restore := captureTelemetryJSONLogs(t)
	defer restore()

	hub := newTestHub()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := &auth.PegasusClaims{UserID: "SUP-LOG", Role: "ADMIN"}
		ctx := context.WithValue(r.Context(), auth.ClaimsContextKey, claims)
		hub.HandleConnection(w, r.WithContext(ctx))
	}))
	defer srv.Close()

	traceID := "trace-telemetry-1"
	conn := dialTelemetryWithTrace(t, srv, traceID)
	waitClients(t, hub, 1)
	_ = conn.Close()
	waitNoClients(t, hub)

	waitTelemetryLogContains(t, logs, "telemetry client connected")
	waitTelemetryLogContains(t, logs, "telemetry client disconnected")
	waitTelemetryLogContains(t, logs, "\"trace_id\":\""+traceID+"\"")
	waitTelemetryLogContains(t, logs, "\"hub\":\"telemetry\"")
	waitTelemetryLogContains(t, logs, "\"supplier_id\":\"SUP-LOG\"")
}

func TestGPSBuffer_LogsStartFlushStop(t *testing.T) {
	logs, restore := captureTelemetryJSONLogs(t)
	defer restore()

	hub := newTestHub()
	buf := NewGPSBuffer(hub)

	buf.Ingest(GPSEntry{DriverID: "DRV-LOG", Latitude: 41.311, Longitude: 69.279, Timestamp: time.Now().Unix(), SupplierID: "SUP-LOG"})
	buf.flush()
	buf.Stop()

	waitTelemetryLogContains(t, logs, "gps buffer started")
	waitTelemetryLogContains(t, logs, "gps buffer flush completed")
	waitTelemetryLogContains(t, logs, "gps buffer stopped")
	waitTelemetryLogContains(t, logs, "\"component\":\"gps_buffer\"")
}

func TestTelemetryHub_BroadcastDeltaMarshalFailure_LogsStructured(t *testing.T) {
	logs, restore := captureTelemetryJSONLogs(t)
	defer restore()

	hub := newTestHub()
	hub.BroadcastDelta("SUP-DELTA", wsEvents.DeltaEvent{
		T:  wsEvents.DeltaOrderUpdate,
		I:  "ORD-LOG",
		TS: time.Now().Unix(),
		D: map[string]interface{}{
			"bad": func() {},
		},
	})

	waitTelemetryLogContains(t, logs, "telemetry delta marshal failed")
	waitTelemetryLogContains(t, logs, "\"hub\":\"telemetry\"")
	waitTelemetryLogContains(t, logs, "\"topic\":\"SUP-DELTA\"")
	waitTelemetryLogContains(t, logs, "\"aggregate_id\":\"ORD-LOG\"")
}
