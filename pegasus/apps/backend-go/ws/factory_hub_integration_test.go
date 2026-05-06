package ws

import (
	"backend-go/auth"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newFactoryHubServer(hub *FactoryHub, factoryID, supplierID string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := &auth.PegasusClaims{
			UserID:     "factory-user-" + factoryID,
			Role:       "FACTORY",
			FactoryID:  factoryID,
			SupplierID: supplierID,
		}
		ctx := context.WithValue(r.Context(), auth.ClaimsContextKey, claims)
		hub.HandleConnection(w, r.WithContext(ctx))
	}))
}

func waitFactoryConnected(t *testing.T, hub *FactoryHub, factoryID string) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		hub.mu.RLock()
		count := len(hub.clients[factoryID])
		hub.mu.RUnlock()
		if count > 0 {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for factory %s websocket connection", factoryID)
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func TestFactoryHub_BroadcastTransferUpdateIncludesTraceID(t *testing.T) {
	hub := NewFactoryHub()
	srv := newFactoryHubServer(hub, "FAC-001", "SUP-001")
	defer srv.Close()

	conn := dial(t, wsURL(srv))
	defer conn.Close()
	waitFactoryConnected(t, hub, "FAC-001")

	hub.BroadcastTransferUpdate(
		"FAC-001",
		"TR-001",
		"WH-001",
		"MAN-001",
		"APPROVED",
		"LOADING",
		"BATCH_ASSIGN",
		"SUP-001",
		"trace-factory-1",
	)

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var event map[string]interface{}
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatalf("read event: %v", err)
	}

	if event["type"] != EventFactoryTransferUpdate {
		t.Fatalf("type = %v, want %s", event["type"], EventFactoryTransferUpdate)
	}
	if event["factory_id"] != "FAC-001" {
		t.Fatalf("factory_id = %v, want FAC-001", event["factory_id"])
	}
	if event["transfer_id"] != "TR-001" {
		t.Fatalf("transfer_id = %v, want TR-001", event["transfer_id"])
	}
	if event["trace_id"] != "trace-factory-1" {
		t.Fatalf("trace_id = %v, want trace-factory-1", event["trace_id"])
	}
}

func TestFactoryHub_PushToFactoryRespectsScopeIsolation(t *testing.T) {
	hub := NewFactoryHub()
	srvA := newFactoryHubServer(hub, "FAC-A", "SUP-001")
	defer srvA.Close()
	srvB := newFactoryHubServer(hub, "FAC-B", "SUP-001")
	defer srvB.Close()

	connA := dial(t, wsURL(srvA))
	defer connA.Close()
	connB := dial(t, wsURL(srvB))
	defer connB.Close()
	waitFactoryConnected(t, hub, "FAC-A")
	waitFactoryConnected(t, hub, "FAC-B")

	hub.BroadcastManifestUpdate("FAC-A", "MAN-100", "LOADING", "REBALANCE", "", "SUP-001", []string{"TR-100"}, "trace-scope")

	connA.SetReadDeadline(time.Now().Add(2 * time.Second))
	var gotA map[string]interface{}
	if err := connA.ReadJSON(&gotA); err != nil {
		t.Fatalf("FAC-A read event: %v", err)
	}
	if gotA["factory_id"] != "FAC-A" {
		t.Fatalf("FAC-A event factory_id = %v, want FAC-A", gotA["factory_id"])
	}

	connB.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	var gotB map[string]interface{}
	err := connB.ReadJSON(&gotB)
	if err == nil {
		t.Fatalf("FAC-B received unexpected event: %+v", gotB)
	}
	if nerr, ok := err.(net.Error); !ok || !nerr.Timeout() {
		t.Fatalf("FAC-B expected timeout without event, got: %v", err)
	}
}

func TestFactoryHub_BroadcastOutboxFailureToAllFactories(t *testing.T) {
	hub := NewFactoryHub()
	srvA := newFactoryHubServer(hub, "FAC-OUT-1", "SUP-001")
	defer srvA.Close()
	srvB := newFactoryHubServer(hub, "FAC-OUT-2", "SUP-001")
	defer srvB.Close()

	connA := dial(t, wsURL(srvA))
	defer connA.Close()
	connB := dial(t, wsURL(srvB))
	defer connB.Close()
	waitFactoryConnected(t, hub, "FAC-OUT-1")
	waitFactoryConnected(t, hub, "FAC-OUT-2")

	hub.BroadcastOutboxFailure("evt-1", "agg-1", "topic-main", "publish failed", "trace-outbox-1")

	for _, tc := range []struct {
		name string
		conn interface {
			ReadJSON(v interface{}) error
			SetReadDeadline(t time.Time) error
		}
		factoryID string
	}{
		{name: "FAC-OUT-1", conn: connA, factoryID: "FAC-OUT-1"},
		{name: "FAC-OUT-2", conn: connB, factoryID: "FAC-OUT-2"},
	} {
		tc.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		var event map[string]interface{}
		if err := tc.conn.ReadJSON(&event); err != nil {
			t.Fatalf("%s read event: %v", tc.name, err)
		}
		if event["type"] != EventFactoryOutboxFailed {
			t.Fatalf("%s type = %v, want %s", tc.name, event["type"], EventFactoryOutboxFailed)
		}
		if event["factory_id"] != tc.factoryID {
			t.Fatalf("%s factory_id = %v, want %s", tc.name, event["factory_id"], tc.factoryID)
		}
		if event["event_id"] != "evt-1" {
			t.Fatalf("%s event_id = %v, want evt-1", tc.name, event["event_id"])
		}
		if event["trace_id"] != "trace-outbox-1" {
			t.Fatalf("%s trace_id = %v, want trace-outbox-1", tc.name, event["trace_id"])
		}
	}
}
