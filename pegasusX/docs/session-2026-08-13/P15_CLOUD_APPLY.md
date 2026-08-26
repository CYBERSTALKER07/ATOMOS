# P15 — Cloud apply artifacts (not applied)
> **POINT-IN-TIME SNAPSHOT (2026-08-13) — do not treat as current status.**
> Re-verify any claim against live code before acting. Multiple ecosystem hardening phases have shipped since this audit.


**Date:** 2026-08-13  
**Stamp:** **not cloud-ready.** This file inventories in-tree apply targets. No `gcloud` / `terraform apply` / `kubectl apply` was run from this program. Owner GSM secrets were **not** invented.

## Already in-tree (apply is ops)

| Artifact | Path |
|----------|------|
| Terraform | `infra/terraform/` |
| Backend ConfigMap | `infra/k8s/backend-go/configmap.yaml` |
| Billing CronJob (P12-C YAML) | `infra/k8s/billing_monthly_cronjob.yaml` |
| Planning CronJobs | `infra/k8s/planning_*.yaml` |

## Order (matches PROD_READINESS_SEQUENCE.md)

1. **R2** — GSM secrets, ManagedCert, `enable_observability_resources`, launch runbook vs staging URL.
2. Overlay `FACTORY_PLANNING_ENABLED` / `FACTORY_BATCHER_ENABLED` only after P9–P10 proof. ConfigMap **keys exist**; prod values stay `"false"`.
3. **Optimizer** — publish AR image; prod replicas 0→≥1. Residual until image exists.
4. **R1 owner keys** — Soliq/EDS, Global Pay, Twilio/FCM. Fail-closed without them.
5. Billing CronJob + `AR_INVOICES_ENABLED` on SSMR after P12 APIs exist. Prod `AR_INVOICES_ENABLED` stays `"false"` until soak.
6. Auto-order place: shadow soak then dual-control (R3). Do **not** flip `AUTO_ORDER_PLACE_ENABLED` in prod YAML from this program.

## Not done here

- Staging smoke evidence
- Overlay apply evidence
- Optimizer replica bump
- Secret values in GSM
