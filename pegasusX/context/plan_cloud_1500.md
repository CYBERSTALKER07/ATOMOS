# PX-PROD-0 — $1,500/mo cloud foundation

Last updated: 2026-07-04

**Authority:** [`docs/CLOUD_BUDGET_MODEL.md`](../docs/CLOUD_BUDGET_MODEL.md), [`docs/PHASE_0_CLOUD_FOUNDATION_RUNBOOK.md`](../docs/PHASE_0_CLOUD_FOUNDATION_RUNBOOK.md), [`context/plan_production_scale.md`](plan_production_scale.md) Phase 0.

**Target:** Full-GCP-minimal staging at **$1,500/mo** (`monthly_budget_usd = 1500`) — same footprint as the $1,700 pilot envelope (100 PU Spanner, GKE Autopilot 2+1, Redis 1 GB, Confluent Basic, Cloud Run portals min=0).

---

## Execution status

| Step | Command / action | Status | Blocker |
|------|------------------|--------|---------|
| 0 | Fix Terraform GSM `replication` blocks | **done** | — |
| 1 | `staging.tfvars` — real `project_id`, `billing_account_id`, `monthly_budget_usd=1500` | **ready** | Confirm GCP project access |
| 2 | Boss secrets — `.env.staging.secrets` | **pending** | Human handoff |
| 3 | Confluent — cluster + bootstrap + topics | **pending** | Human / Confluent console |
| 4 | `make phase0-preflight` | **pass** | Docker optional (`PHASE0_SKIP_WIRE=1`) |
| 5 | `make terraform-init && make phase0-plan` | **pending** | Steps 1–3 |
| 6 | `make phase0-apply` | **pending** | Human approval after plan review |
| 7 | `make phase0-sync-secrets` | **pending** | Post-apply GSM |
| 8 | `make phase0-migrate` | **pending** | Post-apply Spanner URI |
| 9 | Build/push images + K8s deploy | **pending** | GAR + GKE from apply |
| 10 | `make cloud-smoke-ssmr` | **pending** | Live staging API |
| 11 | `PX-ECS-5` cloud staging proof | **pending** | Ecosystem sync gate |

---

## Human checklist (before `phase0-apply`)

1. GCP project `v-o-i-d` with billing linked to void account `01444D-F6DDEC-B7DC05`.
2. gcloud ADC + project IAM for operator running apply.
3. Fill `.env.staging.secrets` from example (JWT, Global Pay, Maps, webhooks).
4. Confluent cluster in `asia-south1` → `kafka_bootstrap_servers` in `staging.tfvars`.
5. `firebase_project_id` in `staging.tfvars`.
6. Staging API DNS after ingress.

```bash
cd pegasusX
export PHASE0_SKIP_WIRE=1
make phase0-preflight
make terraform-init
make phase0-plan
make phase0-apply
make phase0-sync-secrets
make phase0-migrate
```
