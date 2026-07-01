# Wire-Ready Staging Runbook

Automated loop before GCP wiring. See [COST_GOVERNANCE_RUNBOOK.md](./COST_GOVERNANCE_RUNBOOK.md) for year-1 pilot caps ($1,700/mo).

## Local gates (run until green)

```bash
cd pegasusX

# Loop A — every commit / before push
make wire-ready                    # exit: wire-ready-ok (requires Docker)

# Loop C — release candidate
make px12-preflight                # exit: px12-preflight-ok

# Loop weekly after go-live
make p1-pilot-weekly
PUBLIC_BASE_URL=https://api.staging.example.com make p1-pilot-weekly
```

## Exit criteria to start staging wire

- [ ] `make wire-ready` → `wire-ready-ok`
- [ ] `make px12-preflight` → `px12-preflight-ok`
- [ ] Changes committed to `main` (or release branch)

## Staging GCP wire (Boss / ops)

**Full Phase 0 playbook:** [`PHASE_0_CLOUD_FOUNDATION_RUNBOOK.md`](./PHASE_0_CLOUD_FOUNDATION_RUNBOOK.md)

Prerequisites: GCP project, `billing_account_id`, WIF for GitHub Actions ([`pegasusx-deploy-gke.yml`](../../.github/workflows/pegasusx-deploy-gke.yml)).

```bash
cd pegasusX
make phase0-preflight
cp infra/terraform/staging.tfvars.example infra/terraform/staging.tfvars
make phase0-plan
make phase0-apply
make phase0-sync-secrets
make phase0-migrate
make render-k8s-from-terraform IMAGE_TAG=staging-$(git rev-parse --short HEAD)
```

Populate Secret Manager per [`CLOUD_CREDENTIALS_CHECKLIST.md`](./CLOUD_CREDENTIALS_CHECKLIST.md).

### Spanner migrations

Apply base DDL + migrations including:

- `apps/backend-go/schema/spanner.ddl`
- `apps/backend-go/schema/migrations/20250622_pilot_hot_path_indexes.ddl`

Use `infra/k8s/backend-go/migrate-job.yaml` or `go run ./apps/backend-go/cmd/apply-migration`.

### K8s deploy

```bash
cd pegasusX
# Staging (dual-write allowed for consumer migration):
kubectl apply -k infra/k8s/overlays/staging
# Year-1 pilot / production (2–4 API pods, dual-write OFF):
kubectl apply -k infra/k8s/overlays/pilot
```

Deploy **both** `backend-go` (api) and `backend-go-worker` (worker).

Pilot overlay sets: `KAFKA_TOPIC_DUAL_WRITE=false`, `KAFKA_TOPIC_CONSUME_DOMAIN=false`, `WAREHOUSE_DISPATCH_PLAN_TTL_SEC=60`, HPA max **4**.

### Staging proof

```bash
PUBLIC_BASE_URL=https://api.staging.example.com bash scripts/cloud_smoke_ssmr.sh
PUBLIC_BASE_URL=https://api.staging.example.com make load-cert-cloud
PUBLIC_BASE_URL=https://api.staging.example.com make p1-pilot-weekly
```

## Human gates before prod traffic

- [`qa/PX12_ROLE_ROW_QA.md`](./qa/PX12_ROLE_ROW_QA.md) on real devices
- Support roster in [`P1_PILOT_CHECKLIST.md`](./P1_PILOT_CHECKLIST.md) §4
- Global Pay production credentials + Firebase prod project
- 72h hypercare per [`INCIDENT_RESPONSE_RUNBOOK.md`](./INCIDENT_RESPONSE_RUNBOOK.md)

## Rollback

```bash
kubectl rollout undo deployment/backend-go -n pegasusx
kubectl rollout undo deployment/backend-go-worker -n pegasusx
make validate-launch-readiness
```
