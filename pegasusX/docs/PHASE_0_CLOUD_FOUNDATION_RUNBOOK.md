# Phase 0 — Cloud Foundation (PX-PROD-0)

**Goal:** Production-shaped staging with real Spanner, Memorystore Redis, managed Kafka, and GKE — no emulator fallbacks (`REQUIRE_INFRA_ADAPTERS=true`).

**Authority:** [`context/plan_production_scale.md`](../context/plan_production_scale.md) Phase 0. Complements [`WIRE_READY_STAGING_RUNBOOK.md`](./WIRE_READY_STAGING_RUNBOOK.md).

---

## Prerequisites (local)

| Check | Command | Pass |
|-------|---------|------|
| Code gates green | `make wire-ready` | `wire-ready-ok` |
| gcloud auth | `gcloud auth application-default login` | ADC configured |
| Billing linked | GCP Console → Billing | Project has active billing |
| Boss secrets file | Copy `.env.staging.secrets.example` → `.env.staging.secrets` | Filled (never commit) |

```bash
cd pegasusX
make phase0-preflight          # gcloud + terraform + optional wire-ready
```

---

## Step 1 — Terraform (VPC, Spanner, Redis, GSM, GKE)

```bash
cd pegasusX
cp infra/terraform/staging.tfvars.example infra/terraform/staging.tfvars
# Edit: project_id, billing_account_id, budget_alert_emails, kafka_bootstrap_servers, secrets

make terraform-init
make phase0-plan                 # terraform plan -var-file=staging.tfvars
make phase0-apply                # interactive apply
```

**Confluent Kafka (managed):** create cluster in `asia-south1`, topics per [`infra/terraform/README.md`](../infra/terraform/README.md), pass bootstrap servers on apply.

**Exit:** `terraform output spanner_database_uri` returns a real GCP URI (not emulator).

---

## Step 2 — Secret Manager sync (LC-01 partial)

Populate GSM from boss handoff file (values never printed):

```bash
make phase0-sync-secrets
```

Validates secret **versions** exist for JWT, webhooks, Global Pay, Maps, Kafka bootstrap. Full LC-01–LC-06 sign-off: [`PRODUCTION_CREDENTIAL_VALIDATION_RUNBOOK.md`](./PRODUCTION_CREDENTIAL_VALIDATION_RUNBOOK.md).

---

## Step 3 — Spanner migrations (through `20260702`)

Base schema + all incremental migrations including PX90/PX91:

```bash
# Uses SPANNER_* from terraform-generated .env.k8s.generated
make phase0-migrate
```

**Exit:** `DemandForecastBaseline`, `PlanningSignalProjections`, `SupplierPromotions.MaxRedemptions` exist in staging Spanner.

---

## Step 4 — Container images → Artifact Registry

```bash
export IMAGE_TAG="staging-$(git rev-parse --short HEAD)"
make docker-build-backend docker-build-ai-worker

# After terraform apply with enable_gke=true:
GAR="$(cd infra/terraform && terraform output -raw artifact_registry_url)"
docker tag pegasusx-backend:local "${GAR}/backend-go:${IMAGE_TAG}"
docker tag pegasusx-ai-worker:local "${GAR}/ai-worker:${IMAGE_TAG}"
gcloud auth configure-docker "$(echo "$GAR" | cut -d/ -f1)" --quiet
docker push "${GAR}/backend-go:${IMAGE_TAG}"
docker push "${GAR}/ai-worker:${IMAGE_TAG}"
```

---

## Step 5 — Render + deploy K8s (staging)

```bash
make render-k8s-from-terraform IMAGE_TAG="${IMAGE_TAG}"
kubectl apply -f artifacts/k8s-rendered/

# GKE context
gcloud container clusters get-credentials "$(cd infra/terraform && terraform output -raw gke_cluster_name)" \
  --region asia-south1 --project "$(cd infra/terraform && terraform output -raw project_id)"

kubectl apply -k infra/k8s/overlays/staging --load-restrictor=LoadRestrictionsNone
kubectl apply -f artifacts/k8s-rendered/backend-go-migrate-job.yaml -n pegasusx-staging
kubectl wait --for=condition=complete job/backend-go-migrate -n pegasusx-staging --timeout=600s
```

Deploys: `backend-go`, `backend-go-worker`, `ai-worker`, CronJobs (predictive-push + planning export), External Secrets.

**Config patches:** staging overlay sets `PEGASUSX_ENV=staging`, `GLOBAL_PAY_ENV=staging`, Kafka dual-write ON for consumer migration.

---

## Step 6 — TLS + `PUBLIC_BASE_URL`

1. Point DNS `api.staging.<domain>` → GCE Ingress IP (`kubectl get ingress -n pegasusx-staging`).
2. Attach managed certificate (or cert-manager) per [`WS_INGRESS_AFFINITY.md`](./WS_INGRESS_AFFINITY.md).
3. Set `WS_ALLOWED_ORIGINS` in staging ConfigMap to portal staging origins.

```bash
export PUBLIC_BASE_URL=https://api.staging.example.com
bash scripts/validate_staging_credentials.sh   # → staging-credentials-ok
PUBLIC_BASE_URL=$PUBLIC_BASE_URL make cloud-smoke-ssmr
```

---

## Step 7 — PX-PROD-0 sign-off

| Task | Evidence |
|------|----------|
| Terraform applied | `terraform output` non-empty |
| GSM secrets synced | `make phase0-sync-secrets` → `phase0-secrets-ok` |
| Migrations through `20260702` | `make phase0-migrate` → `phase0-migrate-ok` |
| K8s pods healthy | `kubectl get pods -n pegasusx-staging` all Running |
| Adapters required | Backend boot log: no emulator fallback |
| Health | `curl $PUBLIC_BASE_URL/v1/health` → 200 |
| Automated cred pre-check | `staging-credentials-ok` |

Update anchor in [`context/plan_production_scale.md`](../context/plan_production_scale.md): `PX-PROD-0` → **shipped** when all rows complete.

---

## Rollback

```bash
kubectl rollout undo deployment/backend-go -n pegasusx-staging
kubectl rollout undo deployment/backend-go-worker -n pegasusx-staging
```

Spanner DDL is forward-only; rollback is app revision, not schema drop.

---

## Related

- [`CLOUD_CREDENTIALS_CHECKLIST.md`](./CLOUD_CREDENTIALS_CHECKLIST.md)
- [`CLOUD_CUTOVER_RUNBOOK.md`](./CLOUD_CUTOVER_RUNBOOK.md)
- [`COST_GOVERNANCE_RUNBOOK.md`](./COST_GOVERNANCE_RUNBOOK.md)
