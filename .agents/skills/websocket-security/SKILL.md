# WebSocket Security — Authentication, Relay, and Connection Safety

## Description
Prevents WebSocket security vulnerabilities and reliability bugs: unauthenticated connections, role spoofing via query params, missing cross-pod relay, concurrent write panics, and keepalive misconfigurations. Activated when writing or reviewing any code that touches WebSocket hubs, real-time features, push notifications, or live updates.

## Trigger Keywords
websocket, hub, Upgrader, WriteMessage, PushTo, Broadcast, real-time, live, push, notification, ws, connection, subscribe, room

## Anti-Pattern Catalog

### 1. Zero Authentication on WebSocket Upgrade (P0)
```go
// WRONG — anyone can connect and receive all telemetry
func (h *FleetHub) HandleConnection(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    // ... immediately starts sending GPS data to conn
}

// RIGHT — authenticate BEFORE upgrade
func (h *FleetHub) HandleConnection(w http.ResponseWriter, r *http.Request) {
    claims, ok := auth.ExtractClaims(r)
    if !ok || claims == nil {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil { return }
    room := fmt.Sprintf("supplier:%s", claims.SupplierID)
    h.Subscribe(conn, room)
}
```
**Real finding**: `ws/hub.go` `FleetHub.HandleConnection` — zero JWT verification. Any network client gets all GPS telemetry.

**Rule**: EVERY WebSocket hub MUST authenticate before `Upgrader.Upgrade`. No exceptions. Extract JWT from `Authorization` header or signed query-string token (HMAC/JWT, NOT a bare ID). Reject with 401 BEFORE upgrade.

### 2. Query Param as Auth Fallback (P0 — ROLE SPOOFING)
```go
// WRONG — query param fallback when JWT fails = impersonation
if ok && claims != nil {
    driverID = claims.UserID
} else {
    driverID = r.URL.Query().Get("driver_id") // ANYONE CAN SET THIS
}

// RIGHT — reject if no claims
claims, ok := auth.ExtractClaims(r)
if !ok || claims == nil {
    http.Error(w, "unauthorized", http.StatusUnauthorized)
    return
}
driverID = claims.UserID
```
**Real findings**: `ws/driver_hub.go`, `ws/retailer_hub.go`, `ws/warehouse_hub.go`, `ws/payloader_hub.go` — all four hub types use this pattern. Anyone passing `?driver_id=X` receives that driver's payment notifications.

**Rule**: Query parameters MUST NEVER be used as auth fallback. If JWT extraction fails → reject connection. A query param may carry a signed token, but NEVER a bare entity ID.

### 3. No Cross-Pod Redis Pub/Sub Relay (SILENT DELIVERY FAILURE)
```go
// WRONG — local-only delivery (single-pod)
func (h *DriverHub) PushToDriver(driverID string, payload []byte) {
    h.mu.RLock()
    conns := h.connections[driverID]
    h.mu.RUnlock()
    for _, conn := range conns {
        conn.WriteMessage(websocket.TextMessage, payload)
    }
}

// RIGHT — local delivery + Redis Pub/Sub for cross-pod
func (h *DriverHub) PushToDriver(driverID string, payload []byte) {
    // Local delivery
    h.mu.RLock()
    conns := h.connections[driverID]
    h.mu.RUnlock()
    for _, conn := range conns {
        sc.WriteJSON(payload) // use SafeConn
    }
    // Cross-pod relay (fail-open)
    if err := cache.Publish(ctx, "ws:driver:"+driverID, payload); err != nil {
        slog.Error("redis pub/sub relay failed", "hub", "driver", "driver_id", driverID, "err", err)
        // DO NOT return error — local delivery already succeeded
    }
}
```
**Real findings**: `DriverHub`, `WarehouseHub`, `PayloaderHub` — local-only delivery. `RetailerHub` is the only hub with Redis relay.

**Rule**: Every hub broadcast MUST publish to a Redis Pub/Sub channel. Every pod subscribes and delivers to local connections. Pub/Sub failures are fail-open (log + metric, never block local delivery).

### 4. Concurrent WriteMessage Without Lock (PANIC)
```go
// WRONG — gorilla/websocket panics on concurrent writes
for _, conn := range connections {
    conn.WriteMessage(websocket.TextMessage, msg) // two callers = panic
}

// RIGHT — per-connection write lock
type SafeConn struct {
    conn *websocket.Conn
    mu   sync.Mutex
}
func (sc *SafeConn) Write(msg []byte) error {
    sc.mu.Lock()
    defer sc.mu.Unlock()
    return sc.conn.WriteMessage(websocket.TextMessage, msg)
}
```

**Alternative**: Dedicated write goroutine per connection:
```go
type Conn struct {
    ws   *websocket.Conn
    send chan []byte // buffered channel
}
func (c *Conn) writePump() {
    for msg := range c.send {
        if err := c.ws.WriteMessage(websocket.TextMessage, msg); err != nil {
            return // connection dead
        }
    }
}
// Push is always safe — just send to channel
func (c *Conn) Push(msg []byte) { c.send <- msg }
```

### 5. Origin Allowlist Must Include Production
```go
// WRONG — hardcoded localhost only
var allowedOrigins = map[string]bool{
    "http://localhost:3000": true,
    "http://localhost:3001": true,
}

// RIGHT — include production + dynamic dev patterns
func checkOrigin(r *http.Request) bool {
    origin := r.Header.Get("Origin")
    if productionOrigins[origin] { return true }
    if isLocalDev(origin) { return true }
    return false
}
```
**Real finding**: `ws/hub.go` `CheckWSOrigin` — only localhost origins. Production WebSocket connections from `admin.thelab.uz` are rejected.

**Rule**: Use the same CORS allowlist from `bootstrap.NewApp` for WebSocket origins. Production domains are non-negotiable.

### 6. Keepalive Timing
```go
// Current (suboptimal): 30s ping, 65s pong wait
// Doctrine target: 15s ping, 30s read deadline

const (
    PingInterval = 15 * time.Second
    PongWait     = 30 * time.Second
    WriteWait    = 10 * time.Second
)
```
**Rule**: `PingInterval` < `PondWait / 2`. Dead connections must be reaped within 30s. Current 65s pong wait means a dead driver connection holds a slot for over a minute.

### 7. Room Scoping by Resolved Identity
```go
// WRONG — room key from user-controlled input
room := r.URL.Query().Get("room")

// RIGHT — room key from JWT-resolved identity
room := fmt.Sprintf("supplier:%s", claims.SupplierID)
// or
room := fmt.Sprintf("warehouse:%s", claims.HomeNodeID)
```
**Rule**: Room keys are ALWAYS derived from authenticated claims. A client must NEVER subscribe to a room they don't own. Room spoofing is equivalent to data access bypass.

### 8. Payload Shape for WebSocket Messages
```json
{
    "type": "ORDER_STATUS_CHANGED",
    "trace_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-04-20T12:00:00Z",
    "data": { ... }
}
```
**Rule**: All WebSocket messages MUST include `type` (event discriminator), `trace_id` (for end-to-end tracing), and `timestamp` (server time). Clients switch on `type` to route messages. Missing `trace_id` breaks observability.

## Canonical Verification
After writing WebSocket code, verify:
1. Authentication happens BEFORE `Upgrader.Upgrade`
2. No query-param auth fallback for bare entity IDs
3. `WriteMessage` is protected by per-connection lock or write-pump channel
4. Broadcast includes Redis Pub/Sub relay for cross-pod delivery
5. Origin check includes production domains
6. Ping interval < pong wait / 2
7. Room keys derived from JWT claims, not user input
8. Payload includes `type`, `trace_id`, `timestamp`

## Cross-References
- `intrusions.md` §6 — WebSocket & Real-Time Security
- `gemini-instructions.md` Comms-Hardening §1 — WebSocket Hubs
- `gemini-instructions.md` Agent Protocol — WebSocket Hub Playbook
- `.agents/skills/concurrency-shield/SKILL.md` §6 — gorilla concurrent write protection
