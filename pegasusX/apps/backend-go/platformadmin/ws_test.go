package platformadmin

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
)

func TestTransitionBroadcastsPlatformAdminRoom(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo)
	hub := ws.NewHub("platform-admin", nil, nil)
	svc.SetHub(hub)

	conn := &captureConn{sent: make(chan []byte, 4)}
	unsub := hub.Subscribe(ws.PlatformAdminRoom(), conn)
	defer unsub()

	if err := svc.EnsurePending(context.Background(), TenantSupplier, "sup-ws-1", "WS Co"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-conn.sent:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for pending broadcast")
	}

	if _, err := svc.Transition(context.Background(), TransitionInput{
		Actor: "admin-a", TenantType: TenantSupplier, TenantID: "sup-ws-1",
		Status: StatusApproved, KybNotes: "ok", MarketCode: "UZ", HomeCell: "cell-uz",
	}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-conn.sent:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for approve-requested broadcast")
	}
	if _, err := svc.Transition(context.Background(), TransitionInput{
		Actor: "admin-b", TenantType: TenantSupplier, TenantID: "sup-ws-1",
		Status: StatusApproved, MarketCode: "UZ", HomeCell: "cell-uz",
	}); err != nil {
		t.Fatal(err)
	}
	var got []byte
	select {
	case got = <-conn.sent:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for transition broadcast")
	}

	var msg map[string]any
	if err := json.Unmarshal(got, &msg); err != nil {
		t.Fatalf("json: %v body=%s", err, string(got))
	}
	if msg["type"] != "PLATFORM_ADMIN_AUDIT" {
		t.Fatalf("type=%v", msg["type"])
	}
	if msg["action"] != "TENANT_APPROVED" {
		t.Fatalf("action=%v", msg["action"])
	}
}

type captureConn struct {
	sent chan []byte
}

func (c *captureConn) ID() string { return "c1" }

func (c *captureConn) Identity() auth.Claims {
	return auth.Claims{Role: auth.RolePlatformAdmin, Subject: "admin-a"}
}

func (c *captureConn) Send(_ context.Context, payload []byte) error {
	c.sent <- append([]byte(nil), payload...)
	return nil
}
