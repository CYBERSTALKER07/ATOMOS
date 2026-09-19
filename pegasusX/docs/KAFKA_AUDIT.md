# Kafka + outbox + consumers — code audit

**Date:** 2026-08-18  
**Tree:** `pegasusX/`  
**Kind:** point-in-time audit. **Not** a go-live certificate. **Not** Schema Registry. **Not** terraform apply for Managed Kafka.

**Honesty:** Current source is SoT. Kafka is the **bus**. Spanner + same-txn `OutboxEvents` is the **ledger**.

**Related:** [`SURFACE_AUDITS.md`](./SURFACE_AUDITS.md) · [`DATA_FLOW_AS_IMPLEMENTED.md`](./DATA_FLOW_AS_IMPLEMENTED.md)

---

## 0. Verdict

```
VERDICT: PARTIAL
NOT READY FOR LAYER B (bus program)
EVIDENCE: paths below, opened 2026-08-18
DOCS vs CODE: kafka-event-contracts skill still mentions factory WriteMessages + 2s relay poll — DRIFT vs live EmitJSON + 250ms relay
NEXT: keep outbox-in-txn. Do not add Schema Registry. Do not flip consume-domain without dual-write + provisioned topics.
```

| Plane | Verdict | Meaning |
|-------|---------|---------|
| Same-txn outbox | **REAL** | Domain row + `OutboxEvents` in one RW txn |
| Relay → Kafka | **PARTIAL** | Worker-only; prod ConfigMap still in-cluster DNS; Managed Kafka is comments |
| Consumers | **PARTIAL** | Notification/order/warehouse/returns/billing/partner/twin exist; several handlers ACK parse errors |
| Direct `WriteMessages` for Class A | **GONE** on handlers | Remaining: relay, DLQ, planning ingest, tools |

---

## 1. Architecture (keep)

```text
handler ReadWriteTransaction
  → domain mutation + outbox.EmitJSON (same BufferWrite)
  → commit
worker: outbox.Relay Fetch+lease
  → Kafka Publish (RequiredAcks=all, Hash key, no auto-topic)
  → MarkPublished  |  poison → OutboxDeadLetters
consumers (PEGASUSX_RUN_MODE=worker)
  → notification: WS + optional FCM + Spanner inbox
  → order / warehouse / returns / billing / partner / twin
```

Canonical emit: `outbox/outbox.go` package comment + `EmitJSON`. Order create: `order/service.go` `EventOrderCreated` + `order/repository_spanner.go` same `BufferWrite`.

Relay start: `runtime_workers.go:22-24` only when workers run. API-only does **not** run the relay (`main.go` capabilities). K8s: ConfigMap `PEGASUSX_RUN_MODE: "api"`; worker Deployment overrides `worker`.

Publisher: `RequiredAcks=RequireAll`, `Async=false`, `AllowAutoTopicCreation=false`, Hash balancer — `outbox/kafka_publisher.go:81-91`. kafka-go has **no** `enable.idempotence` (`:61-64`); dedupe is Redis SETNX + consumer middleware.

---

## 2. Topics (three inventories — not one)

| Inventory | What it contains | Path |
|-----------|------------------|------|
| App env | `KAFKA_TOPIC_MAIN` default `pegasusx-main`; orders/dispatch/realtime/webhooks named | `events/events.go:11-28`, `events/topic_routing.go:10-23`; ConfigMap `infra/k8s/backend-go/configmap.yaml:14-22` |
| Strimzi CRDs | RF=3 min.isr=2; canonical `infra/k8s/kafka/kafka-topics.yaml`; legacy `infra/k8s/kafka-topics.yaml` still lists exceptions/telemetry | Cluster file says **DEV/LOCAL ONLY** (`infra/k8s/kafka.yaml:1-6`) — **not** in overlay `resources` |
| GCP Managed Kafka | `enable_managed_kafka` default **false** (`infra/terraform/kafka.tf:10-14`). Topics: main, main-dlq, spatial, realtime, webhooks, freeze-locks, inventory-import — **not** orders/dispatch/exceptions/demand | Staging overlay **does** set brokers + `GCP_MANAGED_OAUTH`. Prod overlay is **comments** (`overlays/prod/kustomization.yaml:61-71`) |

`KAFKA_TOPIC_DUAL_WRITE` and `KAFKA_TOPIC_CONSUME_DOMAIN` default **off**. Staging comment: consume-domain **without** dual-write leaves fiscal PENDING.

`TopicWebhooks` is **retired** in code (`topic_routing.go:16-19`) but still in ConfigMap + Terraform.

Returns consumer reads `logistics.exceptions.v1` (`bootstrap/bootstrap.go` ~1667). That topic is Strimzi logistics YAML, **missing** from Managed Kafka TF list → consumer idle / relay fail-closed if dual-write later.

---

## 3. Consumers (worker)

Started in `runtime_workers.go`: notification, order, warehouse, returns, billing, partner, twin.

| Group | Scope | Honesty |
|-------|-------|---------|
| `void-notification-dispatcher` | WS rooms + FCM + inbox | Inbox persist failure still **commits** offset |
| `void-order-mutator` | Payment/fiscal/dispute — **not** `ORDER_CREATED` | Parse error `return nil` (ACK, no DLQ) |
| `void-warehouse-mutator` | `SUPPLY_REQUEST_ACCEPTED` only | Same ACK-on-parse-fail |
| `void-returns-reverse` | exceptions topic | Topic may not exist on Managed Kafka |
| `void-digital-twin` | Needs Spanner | Location sampled (telemetry throttle) |

Manual commit: `CommitInterval: 0` — `kafka/consumer.go:83-100`. **ai-worker** uses `CommitInterval: time.Second` and can commit **failed** imports (`apps/ai-worker/import_worker.go`).

`REQUIRE_INFRA_ADAPTERS` default true (`bootstrap/bootstrap.go:351`). Kafka/Spanner missing → boot fail. Production validate requires the flag (`bootstrap/config_validate.go:20-21`). ConfigMap `"true"`.

---

## 4. Theatre / leftovers

| Item | Why |
|------|-----|
| `loggingOutboxPublisher` | Always `return nil` (`bootstrap/outbox_runtime.go:20-28`). Used only if Kafka init fails **and** adapters not required. Then relay `MarkPublished` with no broker — **THEATRE**. |
| In-memory outbox | `bootstrap/memory/outbox_store.go` — blocked when `RequireInfraAdapters`. |
| `AnalyticsStreamProcessor` | In tree, **never** started from `runtime_workers.go`. |
| Planning ingest | Direct `WriteMessages`, no `RequireAll` (`planning/ingest.go`) — not Class A money, but not outbox. |

---

## 5. Best practices vs skills

| Practice | Live |
|----------|------|
| `acks=all` | **REAL** |
| Idempotent producer | **PARTIAL** — kafka-go cannot set it; Redis/event-id dedup instead |
| `auto.commit=false` | **REAL** on backend-go; **PARTIAL** on ai-worker |
| Schema Registry | **GONE** — JSON `events/` + `contracts/events.schema.json`. **Do not add.** |
| Partition key = aggregate id | **REAL** Hash; relay may suffix `order_id` (can split related streams) |
| Dual-write DB+Kafka | **Avoided** on Class A. Optional **topic** dual-write is a different flag |
| Per-topic DLQ | **PARTIAL** — one `main-dlq` writer for all groups |

**INTEGRATE:** Keep transactional outbox. Direct `WriteMessages` only for relay, DLQ, replay tool, smoke. Harden or outbox planning ingest if signals must survive broker outage.

---

## 6. Ranked blockers

**Code**

1. Do not add post-commit Kafka for money/stock/order.
2. Do not set `KAFKA_TOPIC_CONSUME_DOMAIN=true` without dual-write **and** provisioned domain topics.
3. Align exceptions topic vs Managed Kafka topic list if returns-on-Kafka stays in scope.
4. Stop ACKing parse failures (`order/consumer.go`, `warehouse/consumer.go`).
5. ai-worker: `CommitInterval: 0`; do not commit failed imports.
6. Planning ingest: `RequireAll` + timeout.

**Ops (after spine, not instead)**

1. Prod overlay: copy staging’s Managed Kafka env **or** run a real in-cluster broker. Today ConfigMap is `kafka.pegasusx.svc.cluster.local:9092` while Strimzi is not in overlays.
2. Deploy `backend-go-worker`. API-only has no relay.
3. Do not `terraform apply -var=enable_managed_kafka=true` as a substitute for (1) or for missing PSP/fiscal code.

---

## 7. Next slice (when asked)

Keep outbox. One code phase: ACK-on-parse-fail → DLQ for order/warehouse consumers. Not Schema Registry. Not Layer B keys.
