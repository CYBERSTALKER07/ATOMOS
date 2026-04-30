# Kafka Event Contracts — Event Production & Consumption Discipline

## Description
Prevents ghost entities, lost trace correlation, contract drift between producers and consumers, and silent event delivery failures. Activates when writing or reviewing code that produces or consumes Kafka events, adds event types, modifies event structs, or touches the outbox relay.

## Trigger Keywords
kafka, event, producer, consumer, WriteMessages, outbox, EmitJSON, topic, partition, offset, consumer group, DLQ, dead letter, trace_id, notification_dispatcher, treasurer, approach

## Anti-Pattern Catalog

### 1. Direct WriteMessages for State Changes (GHOST ENTITIES)
```go
// ❌ WRONG — DB commits, Kafka write fails → entity exists but no one knows
_, err := spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
    return txn.BufferWrite(mutations)
})
if err != nil { return err }
// If this fails, entity is in Spanner but event is lost
_ = producer.WriteMessages(ctx, kafka.Message{Value: eventPayload}) // SWALLOWED!

// ✅ RIGHT — outbox makes event atomic with mutation
_, err := spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
    if err := txn.BufferWrite(mutations); err != nil { return err }
    return outbox.EmitJSON(txn, "Factory", factoryID, kafka.TopicMain,
        kafka.FactoryCreatedEvent{...}, telemetry.TraceIDFromContext(ctx))
})
```
**Real findings**: ~30 `writer.WriteMessages` in handler code — `factory/` (11 instances, all with `_ =` swallowed errors), `payment/webhooks.go`, `order/unified_checkout.go`, `supplier/warehouses.go`, `supplier/vetting.go`, `supplier/returns.go`.

**Rule**: State-change events MUST use `outbox.EmitJSON` inside `ReadWriteTransaction`. Direct `writer.WriteMessages` is acceptable ONLY for loss-tolerant telemetry.

### 2. Swallowed Kafka Write Errors (SILENT LOSS)
```go
// ❌ WRONG — if Kafka is unreachable, event silently vanishes
_ = s.Producer.WriteMessages(ctx, message)

// ✅ BETTER (if not using outbox) — handle the error
if err := s.Producer.WriteMessages(ctx, message); err != nil {
    slog.ErrorContext(ctx, "kafka write failed",
        "topic", topic, "aggregate_id", id, "err", err,
        "trace_id", telemetry.TraceIDFromContext(ctx))
    // DLQ, retry, or return error to caller
}

// ✅ BEST — use outbox (no inline write at all)
return outbox.EmitJSON(txn, aggregateType, id, topic, event, traceID)
```
**Real findings**: 11 `_ = s.Producer.WriteMessages(...)` in `factory/` package alone.

### 3. Missing trace_id on EmitJSON (LOST CORRELATION)
```go
// ❌ WRONG — traceID parameter omitted (variadic allows it)
outbox.EmitJSON(txn, "Order", orderID, kafka.TopicMain, event)

// ✅ RIGHT — always pass trace_id
traceID := telemetry.TraceIDFromContext(ctx)
outbox.EmitJSON(txn, "Order", orderID, kafka.TopicMain, event, traceID)
```
**Finding**: `outbox/emit.go` — `EmitJSON(txn ..., traceID ...string)` uses variadic. Compiler won't warn on omission. The `trace_id` is the ONLY way to correlate a request → Spanner commit → Kafka event → WS broadcast → mobile ACK.

**Rule**: Every `EmitJSON` call MUST include `telemetry.TraceIDFromContext(ctx)` as the last argument.

### 4. Event Struct Without trace_id Body Field
**Finding**: ZERO event structs in `kafka/events.go` have a `trace_id` JSON field. The trace_id propagates ONLY via Kafka message headers (set by outbox relay).

**Consequence for consumers**: Must extract `trace_id` from Kafka headers, not from the JSON body.
```go
// Consumer must do this:
func handleMessage(msg kafka.Message) {
    var traceID string
    for _, h := range msg.Headers {
        if h.Key == "trace_id" {
            traceID = string(h.Value)
            break
        }
    }
    ctx := telemetry.WithTraceID(context.Background(), traceID)
    // ... process with ctx
}
```

### 5. omitempty Inconsistency Across Event Structs
```go
// ❌ INCONSISTENT — same field has different omitempty behavior
type OrderDispatchedEvent struct {
    SupplierID string `json:"supplier_id"` // always present
}
type OrderCompletedEvent struct {
    SupplierID string `json:"supplier_id,omitempty"` // absent when empty!
}
```
**Finding**: `SupplierId` uses `json:"supplier_id"` in some structs and `json:"supplier_id,omitempty"` in others.

**Rule**: For the SAME field name across event structs:
- If always present → no `omitempty` on any struct
- If genuinely optional → `omitempty` on ALL structs
- Never mix. Grep existing structs before adding a new one.

### 6. Import Cycle: payment/proximity Can't Import kafka
```go
// ❌ WRONG — creates import cycle
import "pegasus/apps/backend-go/kafka"
const topic = kafka.EventOrderCompleted // CYCLE: payment → kafka → treasurer → payment

// ✅ RIGHT — local constant with comment
// Event constant — cannot import kafka package (cycle through treasurer)
// Keep aligned with kafka/events.go:EventOrderCompleted
const eventOrderCompleted = "ORDER_COMPLETED"
```
**Affected packages**: `payment`, `proximity`. Raw string constants are intentional. Do NOT "fix" by importing `kafka`.

### 7. Consumer Must Be Idempotent (Version Gating)
```go
// ❌ WRONG — blindly overwrites, stale replay corrupts data
func handleOrderEvent(ctx context.Context, event OrderEvent) {
    spannerClient.Apply(ctx, []*spanner.Mutation{
        spanner.Update("Orders", cols, event.Values()),
    })
}

// ✅ RIGHT — version check before applying
func handleOrderEvent(ctx context.Context, event OrderEvent) error {
    _, err := spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
        row, err := txn.ReadRow(ctx, "Orders", spanner.Key{event.OrderID}, []string{"Version"})
        if err != nil { return err }
        var currentVersion int64
        if err := row.Columns(&currentVersion); err != nil { return err }
        if event.Version <= currentVersion {
            slog.InfoContext(ctx, "stale replay, skipping",
                "order_id", event.OrderID,
                "event_version", event.Version,
                "current_version", currentVersion)
            return nil // ACK and skip
        }
        return txn.BufferWrite([]*spanner.Mutation{...})
    })
    return err
}
```

## Canonical Event Struct Shape
```go
// In kafka/events.go:
type FactoryCreatedEvent struct {
    FactoryID    string `json:"factory_id"`
    SupplierID   string `json:"supplier_id"`
    Name         string `json:"name"`
    HomeNodeType string `json:"home_node_type"`
    Lat          float64 `json:"lat"`
    Lng          float64 `json:"lng"`
    Version      int64  `json:"version"`
}
```
**Universal fields** (every event via headers, not body):
- `trace_id` — Kafka header, set by outbox relay
- `type` — event name string (e.g., `"FACTORY_CREATED"`)
- `timestamp` — RFC3339 UTC

**Body fields**:
- Aggregate root ID as first field
- `Version` for optimistic concurrency
- Supplier/scope IDs for routing
- Domain-specific payload

## Producer Key Rule
```go
// Producer key = aggregate root ID
// Orders → key by order_id
// Drivers → key by driver_id
// Routes → key by route_id
kafka.Message{
    Key:   []byte(orderID),        // ensures per-entity ordering within partition
    Value: eventBytes,
    Headers: []kafka.Header{
        {Key: "trace_id", Value: []byte(traceID)},
        {Key: "type", Value: []byte(kafka.EventOrderCompleted)},
    },
}
```

## Outbox Relay Awareness
The outbox relay (`outbox/relay.go`) has these characteristics:
- **Poll interval**: 2s (not 250ms per doctrine) — expect ~2s latency
- **Uses strong reads** (not stale) — adds Spanner contention under load
- **markPublished uses Apply** — no retry on abort → duplicate delivery possible
- **No stuck-event watchdog** — permanently stuck events go undetected

**Implications for consumers**: Always be idempotent. Expect duplicates. Don't assume sub-second delivery.

## Event Lifecycle Verification
Before declaring a new event complete:
1. **Constant**: defined in `kafka/events.go` with descriptive name
2. **Struct**: defined in `kafka/events.go` with consistent JSON tags
3. **Producer**: `outbox.EmitJSON` inside `ReadWriteTransaction`, with `traceID`
4. **Consumer**: registered in `notification_dispatcher.go`, `treasurer.go`, or domain consumer
5. **Version gating**: consumer checks version before applying
6. **DLQ**: consumer writes to `<topic>-dlq` after `MaxAttempts` failures
7. **Cross-platform**: if the event triggers UI updates, WebSocket broadcast is wired

## Verification Checklist
- [ ] State-change events use `outbox.EmitJSON`, not `writer.WriteMessages`
- [ ] No `_ = producer.WriteMessages(...)` — errors handled or outbox used
- [ ] Every `EmitJSON` includes `telemetry.TraceIDFromContext(ctx)` as last arg
- [ ] New event struct JSON tags match existing conventions (check omitempty)
- [ ] Consumer extracts trace_id from Kafka headers, not body
- [ ] Consumer version-gates before applying (stale replays are ACK+skip)
- [ ] Event constant matches `kafka/events.go` — no raw strings (except cycle-exempt packages)
- [ ] Producer key is the aggregate root ID

## Cross-References
- **intrusions.md** §4 (Kafka Event Contracts) — full finding details
- **intrusions.md** §12 (Outbox & Event Relay) — relay characteristics
- **gemini-instructions.md** Enterprise Algorithm §2 (Transactional Outbox)
- **gemini-instructions.md** Comms-Hardening §3 (Kafka Producers)
- **gemini-instructions.md** Kafka Producer & Consumer Playbook
- **spanner-discipline** skill — outbox integration pattern
- **gap-hunter** skill — detecting unwired events / contract drift
