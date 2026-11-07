package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// ClientEnvelope represents an inbound JSON control/protocol message sent by a
// connected WebSocket client over /v1/ws.
type ClientEnvelope struct {
	Type        string `json:"type"`
	ID          string `json:"id,omitempty"`
	EventID     string `json:"event_id,omitempty"`
	MessageID   string `json:"message_id,omitempty"`
	Room        string `json:"room,omitempty"`
	RoomID      string `json:"roomId,omitempty"`
	Status      string `json:"status,omitempty"`
	LastEventID string `json:"last_event_id,omitempty"`
	SinceSeq    int64  `json:"since_seq,omitempty"`
}

func (e ClientEnvelope) resolvedRoom() string {
	if r := strings.TrimSpace(e.Room); r != "" {
		return r
	}
	return strings.TrimSpace(e.RoomID)
}

func (e ClientEnvelope) resolvedAckID() string {
	if id := strings.TrimSpace(e.EventID); id != "" {
		return id
	}
	if id := strings.TrimSpace(e.MessageID); id != "" {
		return id
	}
	return strings.TrimSpace(e.ID)
}

// AuthorizeRoomForIdentity validates that the authenticated caller is allowed
// to subscribe to, unsubscribe from, or replay events from the target room, and
// returns the owning Hub. Arbitrary room creation or cross-tenant room access
// is strictly rejected with ErrUnauthorized.
func AuthorizeRoomForIdentity(ident auth.Claims, room string, hubs roleHubs) (*Hub, error) {
	room = strings.TrimSpace(room)
	if room == "" {
		return nil, ErrUnauthorized
	}

	switch ident.Role {
	case auth.RoleRetailer:
		orgID := strings.TrimSpace(auth.ResolveRetailerOrgID(ident))
		if orgID != "" && (room == "retailer:"+orgID || room == RetailerTrackingRoom(orgID)) {
			if hubs.retailer != nil {
				return hubs.retailer, nil
			}
		}
		if strings.HasPrefix(room, "supplier-promo:") {
			suppID := strings.TrimSpace(strings.TrimPrefix(room, "supplier-promo:"))
			if suppID != "" && !strings.ContainsAny(suppID, " \t\r\n*?") && hubs.retailer != nil {
				return hubs.retailer, nil
			}
		}

	case auth.RoleDriver:
		sub := strings.TrimSpace(ident.Subject)
		if sub != "" {
			if (room == "driver:"+sub || room == DriverCommandRoom(sub)) && hubs.driver != nil {
				return hubs.driver, nil
			}
			if room == "telemetry:driver:"+sub && hubs.telemetry != nil {
				return hubs.telemetry, nil
			}
		}

	case auth.RoleAdmin:
		suppID := strings.TrimSpace(ident.SupplierID)
		homeID := strings.TrimSpace(ident.HomeNodeID)
		if suppID != "" && room == "supplier:"+suppID && hubs.supplier != nil {
			return hubs.supplier, nil
		}
		if suppID != "" && room == SupplierTelemetryRoom(suppID) && hubs.telemetry != nil {
			return hubs.telemetry, nil
		}
		if homeID != "" && room == "warehouse:"+homeID && hubs.warehouse != nil {
			return hubs.warehouse, nil
		}
		if homeID != "" && room == "factory:"+homeID && hubs.factory != nil {
			return hubs.factory, nil
		}

	case auth.RoleWarehouseAdmin, auth.RoleWarehouse:
		homeID := strings.TrimSpace(ident.HomeNodeID)
		suppID := strings.TrimSpace(ident.SupplierID)
		if homeID != "" && room == "warehouse:"+homeID && hubs.warehouse != nil {
			return hubs.warehouse, nil
		}
		if suppID != "" && room == "supplier:"+suppID && hubs.supplier != nil {
			return hubs.supplier, nil
		}
		if suppID != "" && room == SupplierTelemetryRoom(suppID) && hubs.telemetry != nil {
			return hubs.telemetry, nil
		}

	case auth.RoleFactoryAdmin, auth.RoleFactory:
		homeID := strings.TrimSpace(ident.HomeNodeID)
		suppID := strings.TrimSpace(ident.SupplierID)
		if ((homeID != "" && room == "factory:"+homeID) || (suppID != "" && room == "factory:"+suppID)) && hubs.factory != nil {
			return hubs.factory, nil
		}
		if suppID != "" && room == "supplier:"+suppID && hubs.supplier != nil {
			return hubs.supplier, nil
		}
		if suppID != "" && room == SupplierTelemetryRoom(suppID) && hubs.telemetry != nil {
			return hubs.telemetry, nil
		}

	case auth.RolePayload:
		sub := strings.TrimSpace(ident.Subject)
		suppID := strings.TrimSpace(ident.SupplierID)
		if ((sub != "" && room == "payload:"+sub) || (suppID != "" && room == "payload:"+suppID)) && hubs.payload != nil {
			return hubs.payload, nil
		}

	case auth.RolePlatformAdmin:
		if room == PlatformAdminRoom() && hubs.platformAdmin != nil {
			return hubs.platformAdmin, nil
		}
	}

	return nil, ErrUnauthorized
}

func handleClientMessage(ctx context.Context, raw []byte, conn Connection, hubs roleHubs, registerDynamicUnsub func(func()), log *slog.Logger) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return
	}
	if strings.EqualFold(trimmed, "ping") {
		sendJSONFrame(ctx, conn, map[string]any{
			"type":      "pong",
			"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		})
		touchIdentityPresence(ctx, conn, hubs)
		return
	}

	var env ClientEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return
	}

	msgType := strings.ToLower(strings.TrimSpace(env.Type))
	ident := conn.Identity()

	switch msgType {
	case "ping", "heartbeat":
		touchIdentityPresence(ctx, conn, hubs)
		resp := map[string]any{
			"type":      "pong",
			"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		}
		if env.ID != "" {
			resp["id"] = env.ID
		}
		sendJSONFrame(ctx, conn, resp)

	case "ack", "message:ack", "chat:message:ack":
		ackID := env.resolvedAckID()
		recordClientAck(string(ident.Role))
		touchIdentityPresence(ctx, conn, hubs)
		if ackID != "" {
			sendJSONFrame(ctx, conn, map[string]any{
				"type":      "ack:confirmed",
				"event_id":  ackID,
				"status":    "acknowledged",
				"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
			})
		}

	case "presence":
		status := strings.ToUpper(strings.TrimSpace(env.Status))
		if status == "" {
			status = "ONLINE"
		}
		for _, h := range roleHubsForIdentity(ident, hubs) {
			if h != nil {
				h.SetPresenceStatus(ctx, conn, status)
			}
		}
		sendJSONFrame(ctx, conn, map[string]any{
			"type":      "presence:updated",
			"status":    status,
			"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		})

	case "room:join", "join_room":
		room := env.resolvedRoom()
		hub, err := AuthorizeRoomForIdentity(ident, room, hubs)
		if err != nil || hub == nil {
			recordUnauthorizedRoom(string(ident.Role))
			if log != nil {
				log.Warn("ws unauthorized room join rejected",
					"conn_id", conn.ID(), "role", ident.Role, "room", room)
			}
			sendJSONFrame(ctx, conn, map[string]any{
				"type":    "error",
				"code":    "unauthorized_room",
				"room":    room,
				"message": ErrUnauthorized.Error(),
			})
			return
		}
		unsub := hub.Subscribe(room, conn)
		if registerDynamicUnsub != nil {
			registerDynamicUnsub(unsub)
		}
		sendJSONFrame(ctx, conn, map[string]any{
			"type":      "room:joined",
			"room":      room,
			"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		})

	case "room:leave", "leave_room":
		room := env.resolvedRoom()
		hub, err := AuthorizeRoomForIdentity(ident, room, hubs)
		if err != nil || hub == nil {
			recordUnauthorizedRoom(string(ident.Role))
			sendJSONFrame(ctx, conn, map[string]any{
				"type":    "error",
				"code":    "unauthorized_room",
				"room":    room,
				"message": ErrUnauthorized.Error(),
			})
			return
		}
		hub.UnsubscribeRoom(room, conn)
		sendJSONFrame(ctx, conn, map[string]any{
			"type":      "room:left",
			"room":      room,
			"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		})

	case "replay":
		room := env.resolvedRoom()
		hub, err := AuthorizeRoomForIdentity(ident, room, hubs)
		if err != nil || hub == nil {
			recordUnauthorizedRoom(string(ident.Role))
			sendJSONFrame(ctx, conn, map[string]any{
				"type":    "error",
				"code":    "unauthorized_room",
				"room":    room,
				"message": ErrUnauthorized.Error(),
			})
			return
		}
		replayed := hub.ReplaySince(ctx, room, env.SinceSeq, env.LastEventID, conn)
		sendJSONFrame(ctx, conn, map[string]any{
			"type":      "replay:completed",
			"room":      room,
			"replayed":  replayed,
			"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		})
	}
}

func touchIdentityPresence(ctx context.Context, conn Connection, hubs roleHubs) {
	for _, h := range roleHubsForIdentity(conn.Identity(), hubs) {
		if h != nil {
			h.TouchPresence(ctx, conn)
		}
	}
}

func sendJSONFrame(ctx context.Context, conn Connection, body map[string]any) {
	raw, err := json.Marshal(body)
	if err != nil {
		return
	}
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2500*time.Millisecond)
	_ = conn.Send(writeCtx, raw)
	cancel()
}
