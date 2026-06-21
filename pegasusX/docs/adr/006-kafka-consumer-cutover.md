# ADR-006: Kafka consumer domain cutover

**Status:** Accepted  
**Date:** 2026-06-22  
**Context:** ADR-005 dual-writes domain topics from the outbox relay. Consumers still read `pegasusx-main`.

## Decision

Domain-specific consumers switch topics via `KAFKA_TOPIC_CONSUME_DOMAIN=true`:

| Consumer group | Topic when cutover enabled |
|----------------|----------------------------|
| `void-order-mutator` | `pegasusx-orders` |
| `void-warehouse-mutator` | `pegasusx-dispatch` |
| `void-notification-dispatcher` | `pegasusx-main` (unchanged) |

Cutover requires **dual-write enabled** so domain topics receive events. Notification dispatcher stays on main until multi-topic fan-in is implemented (avoids duplicate WS delivery).

Resolution: `events.OrderConsumerTopic()`, `events.DispatchConsumerTopic()`.

## Rollout

1. Staging: `KAFKA_TOPIC_DUAL_WRITE=true` + `KAFKA_TOPIC_CONSUME_DOMAIN=true`
2. Verify order mutator + warehouse mutator process events from domain topics
3. Production: enable dual-write first, then consume-domain per consumer group
4. Future: notification multi-topic consumer, retire `pegasusx-main`

## References

- `events/topic_routing.go`, `bootstrap/bootstrap.go`
- `infra/k8s/overlays/staging/kustomization.yaml`
