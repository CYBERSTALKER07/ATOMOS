# Concurrency Shield — Go Race Condition & Goroutine Safety

## Description
Prevents concurrency bugs in Go code: data races, goroutine leaks, deadlocks, and panic-on-concurrent-write. Activated when writing or reviewing any Go code that uses goroutines, channels, mutexes, sync primitives, or concurrent data structures.

## Trigger Keywords
goroutine, go func, channel, mutex, sync.Map, sync.RWMutex, sync.Once, errgroup, concurrent, parallel, worker pool, race condition, data race

## Anti-Pattern Catalog

### 1. Package-Level Mutable Global (DATA RACE)
```go
// WRONG — written by health monitor, read by all request goroutines
var Client *redis.Client

// RIGHT — mutex-protected struct field
type Cache struct {
    mu     sync.RWMutex
    client *redis.Client
}
```
**Real example**: `cache/redis.go` — `var Client *redis.Client` is set to `nil` by health monitor and reassigned on reconnect, while every request goroutine reads it. Classic data race → nil pointer dereference.

**Rule**: Mutable state shared across goroutines lives on a struct behind a `sync.RWMutex` or `atomic.Value`. Package-level `var` that is written after `init()` is forbidden. Singletons live on `*bootstrap.App`.

### 2. init() Spawning Goroutines (PANIC)
```go
// WRONG — goroutines start before dependencies are initialized
func init() {
    for i := 0; i < 8; i++ {
        go worker() // may reference nil Client
    }
}

// RIGHT — explicit Start method called after all deps are initialized
func (c *Cache) StartWorkers(ctx context.Context) {
    for i := 0; i < 8; i++ {
        go c.worker(ctx)
    }
}
```
**Real example**: `cache/middleware.go` — `init()` spawns 8 workers that call `Client.Set()`. If Redis isn't initialized yet, first enqueued job panics.

**Rule**: `init()` may register codecs, drivers, or flags. It MUST NOT dial connections, read files, or start goroutines. Start goroutines from explicit `Start(ctx)` methods called by `bootstrap.NewApp`.

### 3. Fire-and-Forget go func() Without Context (ORPHAN)
```go
// WRONG — goroutine outlives request, loses trace_id
go func() {
    sendNotification(userID, msg) // what context? what trace?
}()

// RIGHT — propagate context (or detach cancellation if intentional)
go func(ctx context.Context) {
    ctx = context.WithoutCancel(ctx) // outlives request, keeps trace values
    if err := sendNotification(ctx, userID, msg); err != nil {
        slog.ErrorContext(ctx, "notification failed", "err", err)
    }
}(ctx)
```
**Real examples**: `supplier/returns.go`, `factory/crud.go`, `supplier/registration.go`.

**Rule**: Every `go func()` MUST accept `ctx` and exit on `<-ctx.Done()`. If the goroutine intentionally outlives the request, use `context.WithoutCancel(ctx)` (Go 1.21+) and comment why.

### 4. Unbounded Goroutine Fan-Out (OOM)
```go
// WRONG — N goroutines for N items, unbounded
for _, order := range orders {
    go processOrder(ctx, order) // 10,000 orders = 10,000 goroutines
}

// RIGHT — bounded pool
g, ctx := errgroup.WithContext(ctx)
g.SetLimit(runtime.GOMAXPROCS(0))
for _, order := range orders {
    order := order
    g.Go(func() error { return processOrder(ctx, order) })
}
if err := g.Wait(); err != nil { ... }
```

**Rule**: `for _, x := range items { go process(x) }` on unbounded collections is forbidden. Use `errgroup.SetLimit(n)` or `workers.Pool`.

### 5. sync.Once for Reconnectable Resources (PERMANENT FAILURE)
```go
// WRONG — if Redis is down at boot, relay is permanently disabled
var relayOnce sync.Once
func startRelay() {
    relayOnce.Do(func() {
        if client == nil { return } // fires once, never retries
        // ... start relay
    })
}

// RIGHT — mutex-guarded initialization with retry
func (r *Relay) ensureStarted() {
    r.mu.Lock()
    defer r.mu.Unlock()
    if r.running { return }
    if r.client == nil { return } // will retry next call
    r.startLocked()
    r.running = true
}
```
**Real example**: `cache/pubsub.go` — `globalRelayOnce sync.Once` permanently disables Pub/Sub relay if Redis is down at boot.

### 6. gorilla/websocket Concurrent WriteMessage (PANIC)
```go
// WRONG — two goroutines calling WriteMessage on same conn
func (h *Hub) pushToAll(msg []byte) {
    for _, conn := range h.snapshot() {
        conn.WriteMessage(websocket.TextMessage, msg) // PANIC if concurrent
    }
}

// RIGHT — per-connection write lock
type SafeConn struct {
    conn *websocket.Conn
    mu   sync.Mutex
}
func (sc *SafeConn) WriteJSON(v interface{}) error {
    sc.mu.Lock()
    defer sc.mu.Unlock()
    return sc.conn.WriteJSON(v)
}
```
**Real example**: `ws/driver_hub.go` `PushToDriver` — iterates snapshots, writes without lock.

**Alternative pattern**: Dedicated write goroutine per connection with a buffered channel:
```go
type Conn struct {
    ws   *websocket.Conn
    send chan []byte
}
func (c *Conn) writePump() {
    for msg := range c.send {
        c.ws.WriteMessage(websocket.TextMessage, msg)
    }
}
```

### 7. Standard Map in Multi-Goroutine Path (DATA RACE)
```go
// WRONG — bare map read/written from multiple goroutines
var connections = map[string]*Conn{}

// RIGHT option A — sync.Map (read-heavy, write-rare)
var connections sync.Map

// RIGHT option B — mutex-guarded map (general case)
type ConnMap struct {
    mu    sync.RWMutex
    conns map[string]*Conn
}
```

**Decision tree**:
- Read-heavy, write-rare → `sync.Map`
- Mixed read/write → `sync.RWMutex` guarded struct
- Single-writer, many-readers → `sync.RWMutex`
- Profile before choosing `RWMutex` over plain `Mutex` — `RWMutex` is slower for <4 readers

### 8. Channel Buffer Sizing
```go
make(chan Event, 0)   // synchronous — sender blocks until receiver ready
make(chan Event, 1)   // single-item buffer — comment WHY
make(chan Event, 100) // backpressure buffer — comment the sizing rationale
```

**Rule**: Every non-zero buffer size MUST have a comment explaining the sizing. `make(chan Event)` (unbuffered) is synchronous by design — use when you need rendezvous semantics. Large buffers hide backpressure problems.

## Canonical Verification
After writing concurrent Go code, verify:
1. `go test -race ./...` passes on touched packages
2. Every `go func()` has a `ctx` parameter
3. Every custom `CoroutineScope` (goroutine lifecycle) exits on `<-ctx.Done()`
4. No package-level mutable `var` written after init
5. No `init()` with side effects (goroutines, network, file I/O)
6. Every WebSocket `WriteMessage` is protected by a per-connection lock
7. No unbounded `go func()` in a loop over a dynamic collection

## Cross-References
- `intrusions.md` §1 — Concurrency & Race Shield (full audit findings)
- `gemini-instructions.md` §7 Clean Code — Concurrency Discipline
- `gemini-instructions.md` §6 Clean Code — Dependency Injection (no package-level globals)
