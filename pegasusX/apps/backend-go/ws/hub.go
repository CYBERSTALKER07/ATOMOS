// Package ws is the canonical WebSocket relay seam. Every role hub (driver,
// retailer, supplier, payload, warehouse, factory, telemetry) is constructed
// as a *Hub with a role-specific name. Hub.Broadcast(room, payload) does:
//
//  1. local fan-out to every Connection subscribed to room,
//  2. best-effort Publish to the cross-pod Pub/Sub channel "ws:<hub>:fanout"
//     using a typed envelope {source, room, payload}, so peer pods can relay
//     the same payload to their local subscribers.
//
// Fail-Open: a Publish failure MUST NOT panic, MUST NOT block local delivery,
// and MUST NOT return an error to the HTTP handler that triggered the
// broadcast. A degraded cross-pod relay is always preferred over a crashed
// pod.
//
// Authentication is the caller's responsibility: Subscribe takes the resolved
// (role, supplier_id, home_node_id) tuple and the room key MUST be scoped to
// the caller's identity. Unauthenticated upgrades are not permitted.
package ws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
)

// Connection is the per-socket write seam. Production wraps a real WebSocket
// connection (nhooyr.io/websocket or gorilla/websocket) and implements Send
// with a bounded write deadline. Tests bind to InMemoryConnection.
type Connection interface {
	// ID is the connection's stable identifier (used for unsubscribe).
	ID() string
	// Identity returns the resolved auth context bound at upgrade time.
	Identity() auth.Claims
	// Send delivers payload to the client. Returns an error on write failure;
	// the Hub treats any error as a dead connection and reaps it synchronously.
	Send(ctx context.Context, payload []byte) error
}

// Hub is the cross-pod broadcast relay for one role.
type Hub struct {
	name     string
	relay    cache.Backend // Pub/Sub seam; nil disables cross-pod fan-out.
	log      *slog.Logger
	instance string

	mu    sync.RWMutex
	rooms map[string]map[string]Connection // room -> connectionID -> conn

	// failureCount is bumped on every Publish failure. Exposed for metrics.
	failureCount uint64
}

// NewHub constructs a Hub. relay is the Pub/Sub backend (typically the same
// Redis Backend used by cache.Cache); pass nil for single-process scaffold.
func NewHub(name string, relay cache.Backend, log *slog.Logger) *Hub {
	if log == nil {
		log = slog.Default()
	}
	return &Hub{
		name:     name,
		relay:    relay,
		log:      log,
		instance: fmt.Sprintf("%s-%d", name, time.Now().UnixNano()),
		rooms:    make(map[string]map[string]Connection),
	}
}

// Subscribe adds conn to room. The caller MUST have validated that conn's
// identity is allowed to read room (e.g. supplier:<own-id>, driver:<self>,
// warehouse:<scoped-id>). Returns a function the caller invokes on disconnect.
func (h *Hub) Subscribe(room string, conn Connection) func() {
	if h == nil || conn == nil || room == "" {
		return func() {}
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.rooms[room]; !ok {
		h.rooms[room] = make(map[string]Connection)
	}
	h.rooms[room][conn.ID()] = conn
	roomName := room
	connID := conn.ID()
	return func() { h.unsubscribe(roomName, connID) }
}

func (h *Hub) unsubscribe(room, connID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if conns, ok := h.rooms[room]; ok {
		delete(conns, connID)
		if len(conns) == 0 {
			delete(h.rooms, room)
		}
	}
}

// Broadcast delivers payload to every local subscriber of room AND publishes
// it to the cross-pod channel. Fail-Open semantics: Pub/Sub failures are
// logged + counted, never returned. Dead connections are reaped synchronously
// after a failed Send.
func (h *Hub) Broadcast(ctx context.Context, room string, payload []byte) {
	if h == nil || room == "" || len(payload) == 0 {
		return
	}
	h.fanoutLocal(ctx, room, payload)
	h.publishCrossPod(ctx, room, payload)
}

func (h *Hub) fanoutLocal(ctx context.Context, room string, payload []byte) {
	h.mu.RLock()
	conns := make([]Connection, 0, len(h.rooms[room]))
	for _, c := range h.rooms[room] {
		conns = append(conns, c)
	}
	h.mu.RUnlock()

	var dead []string
	for _, c := range conns {
		writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := c.Send(writeCtx, payload)
		cancel()
		if err != nil {
			dead = append(dead, c.ID())
			h.log.Debug("ws send failed; reaping connection",
				"hub", h.name, "room", room, "conn_id", c.ID(), "err", err)
		}
	}
	if len(dead) > 0 {
		h.mu.Lock()
		if conns, ok := h.rooms[room]; ok {
			for _, id := range dead {
				delete(conns, id)
			}
			if len(conns) == 0 {
				delete(h.rooms, room)
			}
		}
		h.mu.Unlock()
	}
}

func (h *Hub) publishCrossPod(ctx context.Context, room string, payload []byte) {
	if h.relay == nil {
		return
	}
	envelope := relayEnvelope{
		Source:  h.instance,
		Room:    room,
		Payload: append([]byte(nil), payload...),
	}
	raw, err := json.Marshal(envelope)
	if err != nil {
		h.log.Warn("ws relay envelope marshal failed", "hub", h.name, "room", room, "err", err)
		return
	}
	channel := h.relayChannel()
	if err := h.relay.Publish(ctx, channel, raw); err != nil {
		h.mu.Lock()
		h.failureCount++
		h.mu.Unlock()
		h.log.Warn("ws cross-pod publish failed; degraded relay",
			"hub", h.name, "room", room, "channel", channel, "err", err)
	}
}

// StartRelaySubscriber consumes one hub-scoped fanout channel and relays
// decoded room payloads to local subscribers. Messages from the same instance
// are ignored to avoid self-echo duplication.
func (h *Hub) StartRelaySubscriber(ctx context.Context) {
	if h == nil || h.relay == nil {
		return
	}
	channel := h.relayChannel()
	msgs, cancel, err := h.relay.Subscribe(ctx, channel)
	if err != nil {
		h.log.Error("ws relay subscribe failed", "hub", h.name, "channel", channel, "err", err)
		return
	}
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return
		case raw, ok := <-msgs:
			if !ok {
				return
			}
			var envelope relayEnvelope
			if err := json.Unmarshal(raw, &envelope); err != nil {
				h.log.Warn("ws relay envelope decode failed", "hub", h.name, "err", err)
				continue
			}
			if envelope.Source == h.instance || envelope.Room == "" || len(envelope.Payload) == 0 {
				continue
			}
			h.fanoutLocal(ctx, envelope.Room, envelope.Payload)
		}
	}
}

func (h *Hub) relayChannel() string {
	return fmt.Sprintf("ws:%s:fanout", h.name)
}

type relayEnvelope struct {
	Source  string `json:"source"`
	Room    string `json:"room"`
	Payload []byte `json:"payload"`
}

// Stats returns instantaneous hub metrics for /debug/ws.
func (h *Hub) Stats() HubStats {
	h.mu.RLock()
	defer h.mu.RUnlock()
	conns := 0
	for _, room := range h.rooms {
		conns += len(room)
	}
	return HubStats{
		Hub:         h.name,
		Rooms:       len(h.rooms),
		Connections: conns,
		PubFailures: h.failureCount,
	}
}

// HubStats is the metrics shape.
type HubStats struct {
	Hub         string `json:"hub"`
	Rooms       int    `json:"rooms"`
	Connections int    `json:"connections"`
	PubFailures uint64 `json:"pub_failures"`
}

// ErrUnauthorized is returned by helper functions that reject a subscription
// because the caller's identity is not allowed on the requested room.
var ErrUnauthorized = errors.New("ws: connection not authorized for room")
