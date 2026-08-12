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
