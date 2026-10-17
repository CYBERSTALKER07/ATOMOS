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
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
)

const defaultRingBufferSize = 256

type bufferedEvent struct {
	Seq       int64
	EventID   string
	Payload   []byte
	Timestamp time.Time
}

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
	limits   HubLimits

	mu          sync.RWMutex
	rooms       map[string]map[string]Connection // room -> connectionID -> conn
	joinedAt    map[string]time.Time             // connectionID -> first subscribe time
	socketRooms map[string]int                   // connectionID -> rooms still joined

	historyMu sync.RWMutex
	history   map[string][]bufferedEvent // room -> ring buffer of recent events
	globalSeq int64

	// failureCount is bumped on every Publish failure. Exposed for metrics.
	failureCount uint64
	shedCount    uint64
}

// NewHub constructs a Hub with role-default connection limits. relay is the
// Pub/Sub backend (typically the same Redis Backend used by cache.Cache); pass
// nil for single-process scaffold.
func NewHub(name string, relay cache.Backend, log *slog.Logger) *Hub {
	return NewHubWithLimits(name, relay, log, DefaultHubLimits(name))
}

var hubInstanceCounter atomic.Uint64

// NewHubWithLimits constructs a Hub with explicit shedding/capacity limits.
func NewHubWithLimits(name string, relay cache.Backend, log *slog.Logger, limits HubLimits) *Hub {
	if log == nil {
		log = slog.Default()
	}
	return &Hub{
		name:        name,
		relay:       relay,
		log:         log,
		instance:    fmt.Sprintf("%s-%d-%d", name, time.Now().UnixNano(), hubInstanceCounter.Add(1)),
		limits:      limits,
		rooms:       make(map[string]map[string]Connection),
		joinedAt:    make(map[string]time.Time),
		socketRooms: make(map[string]int),
		history:     make(map[string][]bufferedEvent),
	}
}

// HasCapacity reports whether this hub can accept another connection without
// exceeding the pod-wide MaxTotal limit.
func (h *Hub) HasCapacity() bool {
	if h == nil || h.limits.MaxTotal <= 0 {
		return true
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.connectionCountLocked() < h.limits.MaxTotal
}

// Subscribe adds conn to room. When MaxPerRoom is exceeded the oldest
// connections in the room are reaped before the new one is retained.
func (h *Hub) Subscribe(room string, conn Connection) func() {
	if h == nil || conn == nil || room == "" {
		return func() {}
	}
	now := time.Now()
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.rooms[room]; !ok {
		h.rooms[room] = make(map[string]Connection)
	}
	if _, already := h.rooms[room][conn.ID()]; !already {
		h.rooms[room][conn.ID()] = conn
		h.noteJoinLocked(conn.ID(), now)
	}
	h.shedRoomLocked(room)
	h.recordMetricsLocked()
	roomName := room
	connID := conn.ID()
	return func() { h.unsubscribe(roomName, connID) }
}

func (h *Hub) unsubscribe(room, connID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.forgetRoomLocked(room, connID)
	h.recordMetricsLocked()
}

// noteJoinLocked records one more room for a socket. Caller holds h.mu.
func (h *Hub) noteJoinLocked(connID string, now time.Time) {
	if h.socketRooms[connID] == 0 {
		h.joinedAt[connID] = now
	}
	h.socketRooms[connID]++
}

// forgetRoomLocked removes connID from one room and frees the socket when no
// rooms remain. Caller holds h.mu.
func (h *Hub) forgetRoomLocked(room, connID string) {
	conns, ok := h.rooms[room]
	if !ok {
		return
	}
	if _, present := conns[connID]; !present {
		return
	}
	delete(conns, connID)
	if len(conns) == 0 {
		delete(h.rooms, room)
	}
	h.noteLeaveLocked(connID)
}

// noteLeaveLocked drops one room membership. Caller holds h.mu.
func (h *Hub) noteLeaveLocked(connID string) {
	n := h.socketRooms[connID]
	if n <= 1 {
		delete(h.socketRooms, connID)
		delete(h.joinedAt, connID)
		return
	}
	h.socketRooms[connID] = n - 1
}

// dropSocketLocked removes a dead socket from every room on this hub.
// Caller holds h.mu.
func (h *Hub) dropSocketLocked(connID string) {
	for room, conns := range h.rooms {
		if _, ok := conns[connID]; !ok {
			continue
		}
		delete(conns, connID)
		if len(conns) == 0 {
			delete(h.rooms, room)
		}
	}
	delete(h.socketRooms, connID)
	delete(h.joinedAt, connID)
}

func (h *Hub) connectionCountLocked() int {
	return len(h.socketRooms)
}

// HasSubscribers reports whether a room has active local subscribers.
func (h *Hub) HasSubscribers(room string) bool {
	if h == nil || room == "" {
		return false
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms[room]) > 0
}

// shedRoomLocked reaps the oldest connections until the room is within MaxPerRoom.
// Caller must hold h.mu.
func (h *Hub) shedRoomLocked(room string) {
	limit := h.limits.MaxPerRoom
	if limit <= 0 {
		return
	}
	conns := h.rooms[room]
	for len(conns) > limit {
		oldestID := oldestConnectionID(conns, h.joinedAt)
		if oldestID == "" {
			break
		}
		if c, ok := conns[oldestID]; ok {
			if reapable, ok := c.(Reapable); ok {
				reapable.Reap()
			}
			h.forgetRoomLocked(room, oldestID)
			h.shedCount++
			h.recordShed(1)
			h.log.Debug("ws shed stale connection",
				"hub", h.name, "room", room, "conn_id", oldestID, "limit", limit)
		}
	}
	if len(conns) == 0 {
		delete(h.rooms, room)
	}
}

func oldestConnectionID(conns map[string]Connection, joinedAt map[string]time.Time) string {
	var oldestID string
	var oldest time.Time
	for id := range conns {
		at, ok := joinedAt[id]
		if !ok {
			return id
		}
		if oldestID == "" || at.Before(oldest) {
			oldestID = id
			oldest = at
		}
	}
	return oldestID
}

// AttachRoomsForRetailer subscribes every live connection in retailer:{retailerID}
// to additional rooms (for example supplier-promo:{supplier_id}).
func (h *Hub) AttachRoomsForRetailer(retailerID string, rooms []string) {
	if h == nil || retailerID == "" || len(rooms) == 0 {
		return
	}
	baseRoom := "retailer:" + retailerID
	h.mu.Lock()
	defer h.mu.Unlock()
	baseConns := h.rooms[baseRoom]
	if len(baseConns) == 0 {
		return
	}
	for _, room := range rooms {
		room = strings.TrimSpace(room)
		if room == "" {
			continue
		}
		if _, ok := h.rooms[room]; !ok {
			h.rooms[room] = make(map[string]Connection)
		}
		for connID, conn := range baseConns {
			if _, already := h.rooms[room][connID]; already {
				continue
			}
			h.rooms[room][connID] = conn
			h.noteJoinLocked(connID, h.joinedAt[connID])
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
	deliveredPayload := h.recordHistory(room, payload)

	h.mu.RLock()
	conns := make([]Connection, 0, len(h.rooms[room]))
	for _, c := range h.rooms[room] {
		conns = append(conns, c)
	}
	h.mu.RUnlock()

	var dead []string
	for _, c := range conns {
		writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := c.Send(writeCtx, deliveredPayload)
		cancel()
		if err != nil {
			dead = append(dead, c.ID())
			h.log.Debug("ws send failed; reaping connection",
				"hub", h.name, "room", room, "conn_id", c.ID(), "err", err)
		}
	}
	if len(dead) > 0 {
		h.mu.Lock()
		for _, id := range dead {
			h.dropSocketLocked(id)
		}
		h.recordMetricsLocked()
		h.mu.Unlock()
	}
}

func enrichWSEventPayload(payload []byte, seq int64, eventID string) []byte {
	var object map[string]any
	if err := json.Unmarshal(payload, &object); err != nil {
		return payload
	}
	modified := false
	if _, ok := object["event_id"]; !ok && eventID != "" {
		object["event_id"] = eventID
		modified = true
	}
	if _, ok := object["seq"]; !ok && seq > 0 {
		object["seq"] = seq
		modified = true
	}
	if !modified {
		return payload
	}
	patched, err := json.Marshal(object)
	if err != nil {
		return payload
	}
	return patched
}

func (h *Hub) recordHistory(room string, payload []byte) []byte {
	if h == nil || room == "" || len(payload) == 0 {
		return payload
	}
	h.historyMu.Lock()
	defer h.historyMu.Unlock()
	h.globalSeq++
	_, eventID := parseSSEEventMetadata(payload)
	if eventID == "" {
		eventID = strconv.FormatInt(h.globalSeq, 10)
	}
	enriched := enrichWSEventPayload(payload, h.globalSeq, eventID)
	buf := h.history[room]
	evt := bufferedEvent{
		Seq:       h.globalSeq,
		EventID:   eventID,
		Payload:   append([]byte(nil), enriched...),
		Timestamp: time.Now().UTC(),
	}
	buf = append(buf, evt)
	if len(buf) > defaultRingBufferSize {
		buf = buf[len(buf)-defaultRingBufferSize:]
	}
	h.history[room] = buf
	return enriched
}

// ReplaySince delivers buffered events for room to conn, starting after sinceSeq or lastEventID.
func (h *Hub) ReplaySince(ctx context.Context, room string, sinceSeq int64, lastEventID string, conn Connection) int {
	if h == nil || conn == nil || room == "" {
		return 0
	}
	h.historyMu.RLock()
	events := append([]bufferedEvent(nil), h.history[room]...)
	h.historyMu.RUnlock()

	lastEventID = strings.TrimSpace(lastEventID)
	replayed := 0
	foundMarker := false
	for _, evt := range events {
		if lastEventID != "" && !foundMarker {
			if evt.EventID == lastEventID {
				foundMarker = true
			}
			continue
		}
		if sinceSeq > 0 && evt.Seq <= sinceSeq {
			continue
		}
		writeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		err := conn.Send(writeCtx, evt.Payload)
		cancel()
		if err != nil {
			break
		}
		replayed++
	}
	return replayed
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
		h.recordPubFailure()
		h.log.Warn("ws cross-pod publish failed; degraded relay",
			"hub", h.name, "room", room, "channel", channel, "err", err)
	}
}

// StartRelaySubscriber consumes one hub-scoped fanout channel and relays
// decoded room payloads to local subscribers. Messages from the same instance
// are ignored to avoid self-echo duplication. It features an exponential backoff
// reconnection loop so temporary Redis disconnects do not terminate relay fanout permanently.
func (h *Hub) StartRelaySubscriber(ctx context.Context) {
	if h == nil || h.relay == nil {
		return
	}
	channel := h.relayChannel()
	backoff := 50 * time.Millisecond
	const maxBackoff = 5 * time.Second

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		msgs, cancel, err := h.relay.Subscribe(ctx, channel)
		if err != nil {
			h.log.Warn("ws relay subscribe failed, retrying", "hub", h.name, "channel", channel, "err", err, "backoff", backoff)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				continue
			}
		}

		// Reset backoff on successful subscription
		backoff = 50 * time.Millisecond

		active := true
		for active {
			select {
			case <-ctx.Done():
				cancel()
				return
			case raw, ok := <-msgs:
				if !ok {
					active = false
					break
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
		cancel()
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
	return HubStats{
		Hub:         h.name,
		Rooms:       len(h.rooms),
		Connections: h.connectionCountLocked(),
		PubFailures: h.failureCount,
		ShedCount:   h.shedCount,
		MaxPerRoom:  h.limits.MaxPerRoom,
		MaxTotal:    h.limits.MaxTotal,
	}
}

// HubStats is the metrics shape.
type HubStats struct {
	Hub         string `json:"hub"`
	Rooms       int    `json:"rooms"`
	Connections int    `json:"connections"`
	PubFailures uint64 `json:"pub_failures"`
	ShedCount   uint64 `json:"shed_count"`
	MaxPerRoom  int    `json:"max_per_room"`
	MaxTotal    int    `json:"max_total"`
}

// ErrUnauthorized is returned by helper functions that reject a subscription
// because the caller's identity is not allowed on the requested room.
var ErrUnauthorized = errors.New("ws: connection not authorized for room")

func (h *Hub) TouchPresence(ctx context.Context, conn Connection) {
	if h.relay == nil {
		return
	}
	identity := conn.Identity()
	if identity.Subject == "" {
		return
	}
	key := fmt.Sprintf("{presence:%s}:%s", identity.Role, identity.Subject)
	// 3x ping interval for TTL
	_ = h.relay.Set(ctx, key, []byte("ONLINE"), 45*time.Second)
}

func (h *Hub) ClearPresence(ctx context.Context, conn Connection) {
	if h.relay == nil {
		return
	}
	identity := conn.Identity()
	if identity.Subject == "" {
		return
	}
	key := fmt.Sprintf("{presence:%s}:%s", identity.Role, identity.Subject)
	_ = h.relay.Delete(ctx, key)
}
