# pegasusX Cloud Budget Model (Full-GCP-Minimal)

Canonical cost reference for the **full-GCP-minimal** deployment choice: real managed Spanner, Memorystore Redis, Kafka, and GKE with minimal replicas, trial/free-tier quotas where applicable, and strict monitoring. Target steady-state **&lt;$1,500/mo** initially.

See also: [`FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md`](./FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md) § infra.

## Monthly cost envelope (USD, asia-south1 baseline)

| Service | Minimal footprint | Est. monthly | Notes |
|---|---|---:|---|
| Cloud Spanner | 1× regional instance, 100 PU processing | $650–900 | Scale PU only after SLO breach; stale reads on dashboards |
| GKE Autopilot | 2 backend pods + 1 ai-worker | $180–280 | HPA max 4; no sticky sessions |
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
# infra/terraform/variables.tf
monthly_budget_usd   = 1500
billing_account_id   = "XXXXXX-XXXXXX-XXXXXX"  # optional; enables budget resource
budget_alert_emails  = ["ops@example.com"]
```

Apply with `terraform apply` in `pegasusX/infra/terraform/` after `billing_account_id` is set.
