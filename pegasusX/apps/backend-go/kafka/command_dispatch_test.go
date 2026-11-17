package kafka

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
)

func TestHandleRouteEvent_StampsConsistentCommandIDAcrossRooms(t *testing.T) {
	driverHub := ws.NewHub("driver", nil, nil)
	cmdConn := &dispatcherConnSpy{id: "cmd-conn"}
	driverHub.Subscribe(ws.DriverCommandRoom("drv-cmd-1"), cmdConn)

	roomConn := &dispatcherConnSpy{id: "room-conn"}
	driverHub.Subscribe("driver:drv-cmd-1", roomConn)

	d := NewNotificationDispatcher(DispatcherDeps{
		DriverHub: driverHub,
	})
	raw := []byte(`{"type":"` + events.EventRouteReplanned + `","route_id":"rt-1","driver_id":"drv-cmd-1","supplier_id":"sup-1","warehouse_id":"wh-1","trace_id":"tr-9"}`)
	if err := d.handleRouteEvent(context.Background(), raw, "tr-9"); err != nil {
		t.Fatalf("handleRouteEvent: %v", err)
	}

	if len(roomConn.messages) != 1 || len(cmdConn.messages) != 1 {
		t.Fatalf("expected 1 message in each room, got room=%d cmd=%d", len(roomConn.messages), len(cmdConn.messages))
	}
	var roomMsg, cmdMsg map[string]any
	if err := json.Unmarshal(roomConn.messages[0], &roomMsg); err != nil {
		t.Fatalf("unmarshal driver room msg: %v", err)
	}
	if err := json.Unmarshal(cmdConn.messages[0], &cmdMsg); err != nil {
		t.Fatalf("unmarshal cmd room msg: %v", err)
	}
	cmdID1, _ := roomMsg["command_id"].(string)
	cmdID2, _ := cmdMsg["command_id"].(string)
	if cmdID1 == "" || cmdID1 != cmdID2 {
		t.Fatalf("expected matching non-empty command_id across rooms, got %q and %q", cmdID1, cmdID2)
	}
}
