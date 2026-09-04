# Redis — cache / hot location / perimeter — code audit

**Date:** 2026-08-18  
**Tree:** `pegasusX/`  
**Kind:** point-in-time audit. Redis is **not** the ledger. **Not** a go-live certificate.

**Related:** [`SURFACE_AUDITS.md`](./SURFACE_AUDITS.md) · [`E2_PER_SUPPLIER_PERIMETER_DESIGN.md`](./E2_PER_SUPPLIER_PERIMETER_DESIGN.md) · [`MAPS_AUDIT.md`](./MAPS_AUDIT.md)

---

## 0. Verdict

```
VERDICT: PARTIAL
NOT a durable SoT
EVIDENCE: cache/, telemetry/location_store.go, retailer/proximity_service.go, terraform main.tf Redis
DOCS vs CODE: infra/terraform/README.md claims Memorystore STANDARD_HA — live TF is BASIC (main.tf:105)
NEXT: keep Redis for dashboards, last-loc, debounce, geo cache. Move money-adjacent tokens off Redis. Do not call perimeter “wired” on checkout.
```

---

## 1. Role (keep)

| Use | Type | Key / fact | Verdict |
|-----|------|------------|---------|
| Dashboard JSON | String, 20s | `dash:{role}:{scope}:today` — `cache/dashboard.go` | **REAL** |
| Driver last location | String JSON, **2m** | `telemetry:driver:last_location:{id}` — `telemetry/location_store.go:14-19` | **REAL** (hot, not SoT) |
| Dispatch plan preview | String, 60s | `warehouse:dispatch_plan:{wh}` | **REAL** |
| Geo provider cache | String | `geo:autocomplete:` / forward / reverse / place | **REAL** |
| Kafka event dedup | String SETNX, 7d | `dedup:event:{key}` | **REAL** sidecar |
| JWT revoke list | String | `jwt:revoked:{jti}` — Redis error **fail-closed** (treat revoked) | **PARTIAL** (Redis-only denylist) |
| Idempotency | String SETNX | `idem:{key}` | **PARTIAL** (money path depends on Redis survival) |
| Worker heartbeat | String, 45s | `pegasusx:runtime:worker:heartbeat` — `bootstrap/worker_heartbeat.go` | **REAL** |
| Cache invalidate | Pub/Sub `cache:invalidate` | After Spanner **commit** — `cache/cache.go:5-7`, `:114-130` | **REAL** pattern; many Invalidate keys have **no Get** (no-op) |
| Delivery perimeter | Set `SISMEMBER` | `ssmr:delivery_perimeter` — **global** | **THEATRE** on checkout (see §3) |
| Payment bypass / early-complete | String | `payment_bypass:` / `early_complete:` — Redis only | **Wrong SoT** |

No Hash / List / Stream / Cluster client. `redis.NewClient` — `cache/redis_backend.go:28-63`.

Circuit: wrap Redis + memory, fail-closed when `REQUIRE_INFRA_ADAPTERS` or production — `bootstrap/bootstrap.go:446-449`. Boot ping fail → error.

---

## 2. How it should be modeled (Redis Inc skills)

| Skill | vs code |
|-------|---------|
| Colon keys | Mostly OK. Mixed `payment_bypass:` underscore prefixes. |
| Hash vs JSON blob | Every object is `SET` of JSON. Fine for 20s dashboards / 2m location. Wrong if Redis stays SoT for bypass tokens. |
| Cluster hash tags | `SupplierScopedKey` → `{sup:id}:suffix` (`cache/keys.go:8-10`) — **tests only**. Production keys untagged. Multi-key `DEL` will CROSSSLOT if Cluster is ever used. Memorystore BASIC is standalone today. |
| `allkeys-lru` | Terraform `maxmemory-policy=allkeys-lru` (`main.tf:112-114`) **evicts TTL=0** perimeter `Persist` sets. |

**INTEGRATE:** Keep Redis as cache + hot loc + optional perimeter **index** re-seeded from Spanner/H3. Not money. Not session SoT.

---

## 3. Perimeter (gap)

- Production key: `ssmr:delivery_perimeter` — `retailer/proximity_service.go:18-24`.
- `PerimeterKeyForSupplier` exists (`:33-41`) — **not** wired to order-create.
- Boot seeds one global circle — `bootstrap/bootstrap.go:579-602`.
- Comment says order creation checks SISMEMBER. **Live order create does not call `IsRetailerInZone`.** Callers: proximity service, tests, `cmd/ssmr-smokecheck`. Retailer service holds `proximity` with **zero** uses.
- Checkout `delivery_perimeter_unavailable` is warehouse resolver / geography, not this Redis set.

Until SISMEMBER is on create **per `SupplierId`**, the Redis set is a seeded artifact, not zone law. Design: [`E2_PER_SUPPLIER_PERIMETER_DESIGN.md`](./E2_PER_SUPPLIER_PERIMETER_DESIGN.md).

---

## 4. Infra

| Item | Live | Path |
|------|------|------|
| Tier | **BASIC** (no replica) | `infra/terraform/main.tf:103-105` |
| AUTH | default on | `variables.tf` redis_auth_enabled |
| TLS | `SERVER_AUTHENTICATION` | same |
| K8s addr | `redis.pegasusx.svc.cluster.local:6379` + `REDIS_TLS_ENABLED=true` | `configmap.yaml:12-13` |
| Staging | Memorystore IP `:6378` TLS | `overlays/staging/kustomization.yaml` |
| Prod overlay | Does **not** override addr (inherits in-cluster hostname) | |
| Password | Secret `REDIS_PASSWORD`; ExternalSecret `redis-addr` **not** injected as `REDIS_ADDR` | `deployment.yaml` vs ESO |

---

## 5. Wiring gaps

1. Rate limiter Lua only attaches if backend is `*cache.RedisBackend`. Production wraps **circuit breaker** first → limiter stays **in-process** (`bootstrap.go:1363-1368` vs `:448-449`).
2. Worker heartbeat nil Redis → `WorkerLive` false → API-only may start a **second** notification consumer (`runtime_workers.go`).
3. Catalog/inventory/notification Invalidate without matching Get — dead keys, not stale-cache risk.

---

## 6. Ranked blockers

1. **P0** — `payment_bypass` / `early_complete` only in Redis + LRU. Move to Spanner or delete.
2. **P0/P1** — Perimeter global + unused on checkout. Wire per-supplier **or** drop the “order checks SISMEMBER” comment.
3. **P1** — BASIC + LRU vs TTL=0 sets and denylist/idempotency.
4. **P1** — Prod `REDIS_ADDR` vs Memorystore TLS port; unused `redis-addr` secret.
5. **P2** — Use `SupplierScopedKey` before any Cluster.

**Ops HA Redis is not Layer B permission.** Cache loss is acceptable; money-token loss is not.

---

## 7. Next slice (when asked)

Attach Redis rate limiter via `redisAdapter.Client()`. Or: stop documenting perimeter as checkout gate until `AssertInPerimeter(supplierID, cell)` is on create. Not terraform apply.
