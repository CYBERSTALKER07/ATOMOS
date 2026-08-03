# ADR-005: Kafka domain topic dual-write

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



**Status:** Accepted  
**Date:** 2026-06-22  
**Context:** All business events fan into `pegasusx-main`, making consumer scaling and replay isolation difficult.

## Decision

Introduce domain topics with **opt-in dual-write** from the outbox relay:

| Topic | Events |
|-------|--------|
| `pegasusx-orders` | Order lifecycle, pre-orders, shop-closed, payment-at-delivery signals |
| `pegasusx-dispatch` | Manifest, dispatch lock, fleet availability, freeze locks |
| `pegasusx-realtime` | Driver location, command bus, transfer approaching |
| `pegasusx-webhooks` | (reserved) payment gateway settlement |

When `KAFKA_TOPIC_DUAL_WRITE=true`, relay publishes to `TopicMain` **and** the domain topic. Consumers migrate per domain; main topic remains until cutover completes.

Routing lives in `events/topic_routing.go`; relay calls `events.RelayPublishTopics`.

## Consequences

- No emit call-site changes required (topic stays `TopicMain` in outbox rows).
- Enable dual-write in staging first; create domain topics in kafka-init.
- Future: consumer groups subscribe to domain topics only; retire main fan-in.

## References

- `events/topic_routing.go`, `outbox/relay.go`
- `infra/docker-compose.ssmr.yml` kafka-init
