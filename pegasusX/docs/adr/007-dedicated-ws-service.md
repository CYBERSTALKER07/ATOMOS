# ADR-007: Dedicated WebSocket Service

## Status

Proposed — implement when pilot WS load exceeds single-cluster comfort zone.

## Context

WebSocket hubs (`ws.Hub`) run inside `backend-go` API pods today. Cross-pod fan-out uses Redis Pub/Sub (`ws:<hub>:fanout`). Metrics: `void_ws_connections{hub}`, `void_ws_pub_failures_total`.

Pilot architecture is correct for &lt; ~2,000 concurrent connections. Beyond that, long-lived connections compete with HTTP request threads and HPA scale-out multiplies Redis relay traffic.

## Decision

**Defer** a dedicated WS deployment until trigger conditions are met. When triggered:

1. New deployment `backend-go-ws` with `PEGASUSX_RUN_MODE=ws` (or separate binary).
2. Ingress routes `/ws/*` to WS service; REST stays on API.
3. Shared Redis relay channel naming unchanged — API pods publish, WS pods subscribe.
4. Auth: same JWT at upgrade; no session stickiness required.

## Trigger conditions (any)

- Sum of `void_ws_connections` &gt; 2,000 for 24h, or
- API p99 latency correlates with WS connection count (HPA adds API pods but WS load per pod stays high), or
- `void_ws_pub_failures_total` rate &gt; 10/min sustained.

## Consequences

**Positive:** API HPA scales on HTTP CPU only; WS pods scale on connection count.

**Negative:** Extra deployment, ingress rules, and on-call surface. Must verify multi-pod WS still works after split (existing hub tests + manual PX12 check).

## Alternatives considered

| Option | Rejected because |
|--------|------------------|
| Sticky sessions only | Does not reduce per-pod connection memory |
| Kafka for WS fan-out | Wrong latency profile for realtime |
| Single mega API pod | Violates HA and blast-radius goals |

## References

- [`WS_INGRESS_AFFINITY.md`](../WS_INGRESS_AFFINITY.md)
- [`P2_SCALE_ROADMAP.md`](../P2_SCALE_ROADMAP.md)
- Metrics: `apps/backend-go/ws/metrics.go`
