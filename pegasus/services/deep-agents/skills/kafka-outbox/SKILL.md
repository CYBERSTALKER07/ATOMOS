---
name: kafka-outbox
description: Outbox relay, Kafka topics, consumers, DLQ, run-mode parity, twin/TopicWebhooks hygiene, driver location bus.
---

# Kafka + outbox

## Topology

- Emit: domain → `OutboxEvents` (Spanner), ideally same RW txn (`outbox.SpannerTxnBuffer` / `EmitJSON`)
- Relay: `outbox.Relay` on **worker/all** tier → Kafka
- Consume: NotificationDispatcher, order mutator, warehouse mutator, returns, billing, partner webhooks, ai-worker groups

## Topics (defaults)

| Topic | Role |
|-------|------|
| TopicMain | default domain + payments + AR/payout + buyer acceptance |
| TopicOrders / TopicDispatch | domain dual-write |
| TopicRealtime | telemetry / **throttled** `DRIVER_LOCATION_UPDATED` (P1-10) |
| TopicWebhooks | **often unused** — verify before documenting as live (P2-11) |
| TopicExceptions | claims / reverse |
| TopicFreezeLocks | AI freeze |
| inventory.import.events | import worker |

## Run-mode parity (P1-9 — closed)

| Tier | `PEGASUSX_RUN_MODE` | Owns |
|------|---------------------|------|
| API Deployment | `api` | HTTP + WS relay subscribers; notification consumer **only if** no worker heartbeat |
| Worker Deployment | `worker` | Outbox relay + Kafka consumers + `StartWorkerHeartbeat` |
| Local default | `all` | Both |

Heartbeat: Redis key `pegasusx:runtime:worker:heartbeat` (`bootstrap/worker_heartbeat.go`). Fail-open when Redis down (start consumer).

## Driver location bus (P1-10 — closed)

- Full fidelity: WS hub + Redis last-location
- Bus copy: throttled (~5s/driver) via `telemetryroutes.SpannerLocationBusEmitter` → `TopicRealtime`
- Twin consumer still may be unwired at bootstrap (P2-11) — bus emit alone does not prove twin is live

## Still open hygiene

- Twin `EventConsumer` must be **started or deleted** (P2-11)
- `TopicWebhooks` const vs real emit path
- Manual commit + DLQ + Redis dedup patterns must not be bypassed

## Audit commands (mental)

Grep `EmitJSON`, `Start(ctx)`, `StartWorkerHeartbeat`, `LocationBusEmitter`, consumer registration in `bootstrap` / `runtime_workers` / `main.go`.


# Universal Agent & Engineering Guidelines
When developing, designing, or planning, always ensure to account for:
- Gaps, edge cases, and comprehensive feature validation.
- Best practices and optimized integration for Kafka, Redis, Backend, Optimizers, AI, and UI.
- Real-time concepts including WebSockets, webhooks, and their native app equivalents.
- Thorough business logic for features, understanding how the role, app, and ecosystem work together, and engagements with other roles and features.
- Best practices for backend, frontend, and infrastructure libraries/packages. Always prefer existing, high-quality open-source libraries and packages that best suit our features before creating our own.
- Optimal UI infrastructure and UX patterns (e.g., optimal screen positioning for drivers during an active route), applying the same high standards to backend and cloud architecture.
- ALWAYS search the web to find open-source code, libraries, packages, math, algorithms, approaches, and best practices for anything we are doing. If none exist, then create our own.
- Always search the web to get the correct logic, and incorporate edge cases, business logic for features, operations (ops), workflow, data consistency, finance, and AI into everything we do.
