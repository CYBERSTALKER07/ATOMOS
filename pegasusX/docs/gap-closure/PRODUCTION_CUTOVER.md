# Production Cutover — Gap Closure

After staging validation passes (see `docs/gap-closure/STAGING_FLAGS.md`).

## Pre-cutover

1. `make validate-production-profile`
2. `make validate-backend-k8s` && `make validate-ai-worker-k8s`
3. `make validate-launch-readiness` (or `pnpm infra:launch:validate`)
4. `P0_SKIP_SSMR=1 make p0-preflight` + cloud smoke with `PUBLIC_BASE_URL`

## Production migrate

```bash
# Against prod Spanner credentials in .env.k8s
make phase0-migrate
```

## Deploy sequence

1. `backend-go` + workers (outbox relay, notification consumer, cash escalation, credit score, reorder)
2. `supplier-portal`, `warehouse-portal`
3. Driver + supplier + warehouse native apps (staging-validated builds)

## Flag enablement (same order as staging)

1. `CREDIT_NOTE_AUTO_FROM_BUYER_REJECT=true`
2. `CREDIT_NOTE_AUTO_FROM_CLAIM=true`
3. `CREDIT_SCORE_ENFORCEMENT_ENABLED=true`
4. `CASH_RECONCILIATION_REQUIRED=true` (last)

Use `scripts/gap_closure_flag_rollout.sh` for operator notes.

## Post-deploy

- `curl -fsS "$PUBLIC_BASE_URL/healthz"`
- `bash scripts/cloud_smoke_ssmr.sh`
- `bash scripts/staging_smoke.sh` (full SSMR e2e against prod URL if seeded)

## Rollback

Set failing flag to `false`, rollout backend, re-run smoke. Do not enable `CASH_RECONCILIATION_REQUIRED` until driver manual path signed off on staging.
