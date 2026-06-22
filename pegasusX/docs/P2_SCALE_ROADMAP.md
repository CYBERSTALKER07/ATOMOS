# P2 Scale Roadmap — After Pilot Proves Unit Economics

Do **not** start P2 until P1 gates hold for 30+ days and unit economics are validated.

## Trigger matrix

| Initiative | Start when | Defer if |
|------------|------------|----------|
| Kafka domain topics | `pegasusx-main` lag p95 &gt; 30s for 7 days | Pilot still on single topic |
| `order/service.go` splits | Bug fix velocity blocked OR file &gt; 3.5k lines | Team can still navigate |
| Dedicated WS service | Sum `void_ws_connections` &gt; 2,000 sustained | &lt; 500 concurrent WS |
| Rust/GPU optimizer | Dispatch solve p95 &gt; 8s at peak | OR-Tools sidecar meets SLO |

## 1. Kafka domain topics

**Current state:** Dual-write and consume-domain flags exist (ADR-005, ADR-006). **Off in prod** by default.

**Cutover sequence:**

1. Staging: `KAFKA_TOPIC_DUAL_WRITE=true` → validate relay publishes to `pegasusx-orders`, `pegasusx-dispatch`.
2. Staging: `KAFKA_TOPIC_CONSUME_DOMAIN=true` → workers read domain topics.
3. Monitor lag on **both** main and domain topics for 14 days.
4. Prod: dual-write only (consumers still on main).
5. Prod: consume-domain after 7 clean days.

**Rollback:** set both flags `false`; consumers fall back to `pegasusx-main`.

## 2. `order/service.go` splits (planned)

Current ~3,100 lines. Extract in this order:

| File | Methods | ~Lines |
|------|---------|--------|
| `driver_delivery_service.go` | `MarkArrived` … `CollectCash`, QR handlers, `transitionDriverOrder` | ~950 |
| `order_http_handlers.go` | `HandleCreate`, `HandleUpdateStatus`, mutation error helpers | ~400 |
| `order_payment_emit.go` | `emitPaymentRequired`, `emitSettlementRequired`, settlement data | ~200 |

Already extracted: `state_machine.go`, `preorder_service.go`, `cancel_service.go`, `delivery_handshake.go`.

## 3. Dedicated WebSocket service

See [`adr/007-dedicated-ws-service.md`](./adr/007-dedicated-ws-service.md).

**Summary:** Extract `ws.Hub` relay + upgrade handlers to a standalone deployment when connection count or cross-pod Redis fan-out becomes the API bottleneck. API pods keep HTTP; WS pods own long-lived connections.

## 4. Rust/GPU optimizer

Not required for launch or early pilot. Revisit when:

- Fleet &gt; 50 vehicles per dispatch window, or
- ai-worker OR-Tools timeouts &gt; 5% of preview requests.

Reference: `or-tools-sidecar-optimization` skill / Pegasus Rust optimizer gap in parity matrix.

## Budget impact (order of magnitude)

| Item | Monthly delta |
|------|----------------|
| Kafka domain topics | +$0–50 (same cluster) |
| WS service (2 pods) | +$60–120 |
| Spanner PU +100 | +$150–250 |
| Rust optimizer GPU | +$200–400 |

Reconcile with [`CLOUD_BUDGET_MODEL.md`](./CLOUD_BUDGET_MODEL.md) before each P2 item ships.
