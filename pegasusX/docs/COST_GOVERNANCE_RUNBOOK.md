# pegasusX Cost Governance Runbook

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



Year-1 target: **$1,500–1,700/mo** with **1 warehouse** and thousands of retailers (inactive retailers are cheap; **orders, Spanner reads, geocoding, dispatch CPU, and client polling** drive cost).

See also: [CLOUD_BUDGET_MODEL.md](./CLOUD_BUDGET_MODEL.md), [SPANNER_HOT_PATH_REVIEW.md](./SPANNER_HOT_PATH_REVIEW.md), [CLOUD_CREDENTIALS_CHECKLIST.md](./CLOUD_CREDENTIALS_CHECKLIST.md).

## Fixed-cost floor (do not shrink below pilot without SLO breach)

| Service | Pilot footprint | Est. monthly |
|---------|-----------------|-------------:|
| Cloud Spanner | 100 PU cap (regional) | $650–900 |
| Confluent Kafka Basic | 1 cluster | $120–200 |
| Memorystore Redis | 1 GB BASIC, `allkeys-lru` | $35–55 |
| GKE (2–4 API + worker + optimizer + ai-worker) | see `overlays/pilot` | $180–320 |
| Cloud Run portals | min-instances=0 | $20–60 |
| Monitoring + logging | budget alerts | $30–80 |

## Scale order (never reverse)

1. **Index / query fix** — [SPANNER_HOT_PATH_REVIEW.md](./SPANNER_HOT_PATH_REVIEW.md)
2. **Redis cache** — geocode, dispatch plan, catalog (short TTL + invalidation)
3. **Client polling backoff** — WS-first, slower polls when tab hidden
4. **GKE HPA** — API pods 2→4 only when p95 or CPU SLO breached
5. **Spanner PU bump** — only when CPU > 70% for 10+ minutes after steps 1–4

## Freeze rule

At **80% of `monthly_budget_usd`** (default $1,700):

- No HPA `maxReplicas` increase
- No Spanner PU increase
- No new managed services (second Kafka cluster, GPU nodes, etc.)

At **100%**: page on-call; enable load shedding; defer non-prod scale-ups.

Terraform: [infra/terraform/budget.tf](../infra/terraform/budget.tf).

## Redis key conventions

| Prefix | Purpose | TTL |
|--------|---------|-----|
| `geo:autocomplete:` | Places autocomplete JSON | 24h |
| `geo:forward:` | Forward geocode JSON | 7d |
| `geo:reverse:` | Reverse geocode JSON | 7d |
| `geo:place:` | Place details JSON | 7d |
| `warehouse:dispatch_plan:` | Smart dispatch preview | 60s (+ invalidation) |
| `rl:` | Rate limit counters | window-based |
| `idem:` | Idempotency replay | request TTL |

Invalidation-backed caches: post-commit `cache.Invalidate` via Pub/Sub channel `cache:invalidate`. External API caches (geocode) rely on TTL only.

Memorystore: `maxmemory-policy=allkeys-lru` ([infra/terraform/main.tf](../infra/terraform/main.tf)).

## Kafka discipline

- **Pilot/prod:** `KAFKA_TOPIC_DUAL_WRITE=false`, `KAFKA_TOPIC_CONSUME_DOMAIN=false`
- **Staging:** dual-write allowed for consumer migration only
- Fix consumer lag before scaling cluster; partitions stay at pilot size until lag SLO breach
- State transitions: **outbox only** — no inline Kafka from API handlers (telemetry uses Redis)

## Maps & geocoding

- Display: MapLibre (web), MapKit (iOS), Android Maps SDK only where native tiles required
- Geocoding/Places: **backend-only** via `GOOGLE_MAPS_API_KEY`; Redis cache + rate limits on `/v1/platform/geocode/*`
- Route geometry: OSRM sidecar — not Google Directions

## WebSocket cost control

- Hub room shedding: [apps/backend-go/ws/hub.go](../apps/backend-go/ws/hub.go)
- Clients: WS-accelerated refresh; polling is fallback (fleet live map 15s visible / 60s hidden)

## Client polling budgets

| Surface | Visible interval | Hidden / background |
|---------|------------------|---------------------|
| Fleet live map | 15s | 60s |
| Dashboard analytics | 60s | pause |
| Catalog / lists | manual or 60s+ | pause |

Implemented via [packages/api-client/usePolling.ts](../packages/api-client/usePolling.ts) `pauseWhenHidden` and portal-specific intervals.

## Weekly ops checklist (15 min)

- [ ] GCP billing % of $1,700 cap
- [ ] Spanner CPU (console) — target < 60% steady
- [ ] Spanner Query Insights — top 5 by CPU
- [ ] `void_kafka_consumer_lag_seconds` max
- [ ] `void_optimizer_source_total{source="fallback_phase1"}` rate < 5%
- [ ] Google Maps Platform usage vs $200/mo credit
- [ ] `void_redis_cache_hit_total` / miss ratio for `geo:*` prefix

## Deploy overlays

| Overlay | Use | API replicas | Kafka dual-write |
|---------|-----|--------------|------------------|
| `overlays/pilot` | Year-1 prod | 2, HPA max 4 | off |
| `overlays/staging` | Pre-prod wire | inherits prod base | on (migration) |
| `overlays/dev` | Local cluster | 1 | on |

```bash
kubectl apply -k infra/k8s/overlays/pilot
```

## Verification gates

```bash
cd pegasusX
make wire-ready                    # includes cost governance gates
bash scripts/validate_production_profile.sh
kubectl kustomize infra/k8s/overlays/pilot --load-restrictor LoadRestrictionsNone
```

## Related alerts

- Billing 80% / 100% — Terraform budget
- Spanner CPU > 65% — [observability_pilot.tf](../infra/terraform/observability_pilot.tf)
- Optimizer `fallback_phase1` > 5% — observability_pilot.tf
- Kafka consumer lag — observability.tf / observability_pilot.tf
