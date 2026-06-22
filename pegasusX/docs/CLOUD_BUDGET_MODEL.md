# pegasusX Cloud Budget Model (Full-GCP-Minimal)

Canonical cost reference for the **full-GCP-minimal** deployment choice: real managed Spanner, Memorystore Redis, Kafka, and GKE with minimal replicas, trial/free-tier quotas where applicable, and strict monitoring. Target steady-state **$1,500–1,700/mo** for year-1 pilot (1 warehouse, thousands of retailers).

Ops runbook: [COST_GOVERNANCE_RUNBOOK.md](./COST_GOVERNANCE_RUNBOOK.md).

See also: [`FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md`](./FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md) § infra.

## Monthly cost envelope (USD, asia-south1 baseline)

| Service | Minimal footprint | Est. monthly | Notes |
|---|---|---:|---|
| Cloud Spanner | 1× regional instance, 100 PU processing | $650–900 | Scale PU only after SLO breach; stale reads on dashboards |
| GKE Autopilot | 2 backend pods + 1 ai-worker | $180–280 | HPA max **4** (pilot overlay); no sticky sessions |
| Memorystore Redis | Basic 1 GB | $35–55 | Pub/Sub invalidation + rate limits |
| Confluent / managed Kafka | 1 cluster, 3 partitions TopicMain | $120–200 | Trial credits may cover first month |
| Cloud Run (portals) | min-instances=0, max=2 | $20–60 | Supplier + warehouse + factory portals |
| Artifact Registry + GCS CDN | desktop/static | $15–40 | Retailer desktop artifacts |
| Cloud Monitoring + Logging | budgets + 5 alert policies | $30–80 | See `infra/terraform/budget.tf` |
| Firebase (Auth + FCM) | Spark → Blaze minimal | $0–50 | FCM within free tier at launch scale |
| **Total** | | **$1,050–1,665** | Alert at 80% / 100% of `monthly_budget_usd` |

## Cost controls

1. **Spanner**: default stale reads (15 s) on list/dashboard paths; index-backed queries only.
2. **GKE**: Autopilot; scale-to-zero Cloud Run for portals; no dedicated optimizer GPU until PX-ECO-010.
3. **Kafka**: single sync writer for state transitions; telemetry async only.
4. **Redis**: 5 min TTL safety net; invalidation via Pub/Sub (not long TTL caches).
5. **Budget alerts**: Terraform `google_billing_budget` when `billing_account_id` is set.

## Monitoring gates

| Alert | Threshold | Action |
|---|---|---|
| Monthly spend | 80% of budget | Slack `#ops-alerts`; freeze non-prod scale-ups |
| Monthly spend | 100% of budget | Page on-call; shed load via priority guard |
| Spanner CPU | &gt;70% 10 min | Review hot queries; add index before PU bump |
| GKE pod restarts | &gt;3 in 15 min | Rollback last deploy |

## Terraform wiring

```hcl
# infra/terraform/budget.tf
monthly_budget_usd   = 1700
billing_account_id   = "XXXXXX-XXXXXX-XXXXXX"  # optional; enables budget resource
budget_alert_emails  = ["ops@example.com"]
spanner_processing_units_cap = 100  # enforce in console until TF supports hard cap
```

Apply with `terraform apply` in `pegasusX/infra/terraform/` after `billing_account_id` is set.

## Growth scale (year 1: 1 warehouse, thousands of retailers)

Pilot envelope (`monthly_budget_usd = 1700`) assumes **1 warehouse**, thousands of retailers (inactive retailers are cheap), and moderate order volume. When scaling order volume:

| Driver | Pilot | Growth (1 WH, 5k retailers) |
|---|---|---|
| Spanner PU | 100 PU cap | **120–200 PU** after SLO breach — largest cost swing |
| GKE backend | 2 pods (pilot overlay) | **2–4 pods** (HPA on dispatch preview p95) |
| optimizer-core | 1 pod, 1 vCPU | **1–2 pods** — OR-Tools CPU, no GPU |
| Confluent Basic | ~$120–200/mo | unchanged at this throughput |
| Maps Geocoding | within $200 credit | cache in Redis; rate-limit geocode endpoints |

**Revised steady-state:** **$1,400–1,700/mo** at year-1 pilot load with Redis geocode cache + stale reads. Raise `monthly_budget_usd` only after 2 weeks of staging metrics breach 80% consistently.

```hcl
# growth staging example (after metrics review)
monthly_budget_usd = 2000
# spanner: bump processing_units in console only when CPU > 70% for 10m
```
