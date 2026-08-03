# Gap Closure — Cloud / SSMR Checklist (2026-07-31)

## Done in code (deploy required)

- Shop-closed inventory release (P0)
- CLAIM fanout + LogisticsException contracts
- Role-row client parity closes (credit, finance, driver, warehouse rescue, factory mutations)
- BillingTierWorker consumer; AnalyticsStreamProcessor removed

## Live on cluster (kubectl verified)

```
namespace: pegasusx-ssmr
pods: backend-go, backend-go-worker, ai-worker — Running
```

## Operator steps (order)

| # | Action | Command / pointer |
|---|--------|-------------------|
| 1 | Fix gcloud quota project | `gcloud config set project pegasus-503013 && gcloud auth application-default set-quota-project pegasus-503013` |
| 2 | Apply Spanner migrations | `bash scripts/apply_ssmr_spanner_migrations.sh` |
| 3 | Build/push/redeploy | `scripts/deploy_gap_closure_staging.sh` or Cloud Build |
| 4 | Smoke Order→COMPLETED | `make cloud-smoke-ssmr` / `scripts/staging_smoke.sh` |
| 5 | DNS + ManagedCert | `artifacts/step11-ingress-ssmr.md` |
| 6 | Firebase app configs | `artifacts/step12-firebase-ssmr.md` |
| 7 | GP SUCCESS password | `artifacts/step14-globalpay-ssmr.md` |
| 8 | SSMR marker gate | produce `ssmr-e2e.log` → `bash scripts/parity/ssmr_ecosystem_marker_gate.sh ssmr-e2e.log` |

## Blocked on external owners

- Domain DNS A record  
- Global Pay merchant password  
- Soliq/OFD (optional legal)  
- Apple/Play distribution configs  
