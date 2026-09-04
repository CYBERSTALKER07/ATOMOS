---
name: redis-cache
description: Redis as cache and coordination — invalidation, WS Pub/Sub, heartbeats; never source of truth.
---

# Redis

## Allowed roles

- Cache for hot reads (catalog, inventory projections) with **post-mutation invalidation**
- Cross-pod WS fanout via Pub/Sub
- Worker liveness heartbeat for api-only notification consumer
- Rate limits / short-lived locks where coded

## Forbidden

- Treating Redis as system of record (Spanner is SoT)
- Caching money/ledger without invalidation proof
- Silent drop of invalidation on cancel paths

## Checks

1. Mutation path: is cache key deleted/updated?
2. WS: multi-pod path uses Redis when scaled?
3. Heartbeat key `pegasusx:runtime:worker:heartbeat` + TTL (45s) / interval (15s) in `bootstrap/worker_heartbeat.go` (P1-9)?
4. Driver last-location is Redis-backed cache, not SoT — bus copy is Spanner outbox throttled (P1-10).


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
