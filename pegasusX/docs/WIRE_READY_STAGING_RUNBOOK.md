# Wire-Ready Staging Runbook

Automated loop before GCP wiring. See the Wire-Ready Loop Plan in repo docs.

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

Prerequisites: GCP project, `billing_account_id`, WIF for GitHub Actions ([`pegasusx-deploy-gke.yml`](../../.github/workflows/pegasusx-deploy-gke.yml)).

```bash
cd pegasusX/infra/terraform
terraform init
terraform apply \
  -var="project_id=YOUR_PROJECT" \
  -var="tenant_slug=staging" \
  -var="enable_gke=true" \
  -var="billing_account_id=XXXXXX-XXXXXX-XXXXXX" \
  -var='budget_alert_emails=["ops@example.com"]'
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
# Set image tags + secrets in overlay or render script
kubectl apply -k infra/k8s/overlays/staging
# Or prod overlay for pilot:
kubectl apply -k infra/k8s/overlays/prod
```

Deploy **both** `backend-go` (api) and `backend-go-worker` (worker).

Pilot Kafka flags: **OFF** (`KAFKA_TOPIC_DUAL_WRITE`, `KAFKA_TOPIC_CONSUME_DOMAIN`).

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
