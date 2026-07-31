# Staging Wiring Matrix

Generated during gap-closure staging validation. Update after each deploy.

## GCP project

| Item | Value |
|------|--------|
| Config project | `pegasus-503013` ([infra/terraform/staging.tfvars](infra/terraform/staging.tfvars)) |
| gcloud active account | Run `gcloud auth list` |
| ADC | `gcloud auth application-default login` |

## Public API

| Item | Value |
|------|--------|
| SSMR / staging API | `https://api-ssmr.pegasusx.app` |
| Health | `curl -fsS https://api-ssmr.pegasusx.app/healthz` |
| Client policy smoke | `make cloud-smoke-ssmr` with `PUBLIC_BASE_URL=https://api-ssmr.pegasusx.app` |

## GKE (expected)

| Item | Value |
|------|--------|
| Cluster | `pegasusx-ssmr-gke` (asia-south1) |
| Namespace | `pegasusx-ssmr` (see [artifacts/prod-cli-wiring-2026-07-28.md](../artifacts/prod-cli-wiring-2026-07-28.md)) |
| kubectl plugin | Install `gke-gcloud-auth-plugin` for cluster access |

Inspect:

```bash
kubectl get deploy,sts,pods -n pegasusx-ssmr
kubectl get ingress -n pegasusx-ssmr
```

## GSM secrets (sample)

Secrets prefixed `pegasusx-staging-*` and `pegasusx-ssmr-*` in project `pegasus-503013`.

```bash
gcloud secrets list --project=pegasus-503013 --filter="name:pegasusx"
```

Sync from local: `make phase0-sync-secrets` (requires `.env.staging.secrets`).

## Spanner migrations (gap closure minimum)

- `20260803_phase_a_reconciliation.ddl`
- `20260804_phase_c_credit_risk.ddl`
- `20260804_phase_c_reorder.ddl`
- `20260804_phase_d_analytics.ddl`
- `20260804_phase_d_notifications.ddl`

Apply all via: `make phase0-migrate` (applies ordered `*.ddl` through latest).

## Gap-closure feature flags (staging OFF until validated)

| Flag | Staging value (baseline) |
|------|--------------------------|
| `CASH_RECONCILIATION_REQUIRED` | `false` |
| `CREDIT_NOTE_AUTO_FROM_BUYER_REJECT` | `false` |
| `CREDIT_NOTE_AUTO_FROM_CLAIM` | `false` |
| `CREDIT_SCORE_ENFORCEMENT_ENABLED` | `false` |

Enable sequence: [STAGING_FLAGS.md](STAGING_FLAGS.md). Rollout helper: `scripts/gap_closure_flag_rollout.sh`.

## Deploy order (gap closure)

1. `backend-go`
2. `supplier-portal`
3. `warehouse-portal`
4. Native apps (driver, supplier, warehouse) as needed

## Automated validation

```bash
export PUBLIC_BASE_URL=https://api-ssmr.pegasusx.app
bash scripts/validate_staging_credentials.sh
bash scripts/staging_smoke.sh
make ssmr-ecosystem-marker-gate LOG=.tmp/staging_e2e.log
```

## Manual critical paths

See [MANUAL_CRITICAL_WALKTHROUGHS.md](MANUAL_CRITICAL_WALKTHROUGHS.md).
