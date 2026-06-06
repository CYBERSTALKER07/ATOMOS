# pegasusX Deployment and Distribution Plan

Last updated: 2026-06-04. Canonical tree: `pegasusX/`.

## Principle

One monorepo, many ship units. Backend, ai-worker, and each client surface deploy independently with path-filtered CI and shared contract packages (`contracts/`, `packages/types`, `packages/api-client`).

## Recommended GCP topology

| Layer | Service | Notes |
|-------|---------|-------|
| API + workers | GKE Autopilot | `backend-go` (3+ replicas), `ai-worker` (2+); manifests in `infra/k8s/` |
| Data | Cloud Spanner + Memorystore Redis | Provisioned via `infra/terraform/main.tf` |
| Kafka | Confluent Cloud or managed Kafka | Bootstrap in Secret Manager (`kafka_bootstrap_servers`) |
| Container images | Artifact Registry | `REGION-docker.pkg.dev/PROJECT/pegasusx/{backend-go,ai-worker}` |
| Web portals | Cloud Run or Firebase App Hosting | Next.js apps; env `NEXT_PUBLIC_API_URL` |
| Desktop installers | GCS + Cloud CDN | Signed URLs for Tauri `.msi`/`.dmg`; optional MS Store MSIX |
| Secrets | Secret Manager + Workload Identity | JWT, webhooks, Firebase, Kafka topics |

See [`infra/terraform/gke.tf`](infra/terraform/gke.tf) for cluster, registry, and Workload Identity wiring.

## Environment separation

| Env | Spanner DB | API host | Proof gate |
|-----|------------|----------|------------|
| dev | Emulator / Docker SSMR | `localhost:8180` | `make test-ssmr-infra` |
| staging | Dedicated instance | `api.staging.<domain>` | `PUBLIC_BASE_URL=... make load-cert-cloud` + `scripts/cloud_smoke_ssmr.sh` |
| prod | Dedicated instance | `api.<domain>` | Load cert (production SLOs) + hypercare |

## Monorepo CI model

```text
.github/workflows/
  backend-go.yml       # paths: apps/backend-go/**
  ai-worker.yml
  supplier-portal.yml
  driver-app-android.yml
  …
```

Release tag `vYYYY.MM.DD` triggers all workflows; each artifact publishes independently.

## Client distribution channels

| Surface | Channel | Update mechanism |
|---------|---------|------------------|
| iOS native | App Store / TestFlight | Store review; `GET /v1/platform/client-policy` for force/min version |
| Android native | Google Play | In-app update (flexible) or immediate when below `minimum_version` |
| Windows desktop (Tauri) | MS Store + website CDN | Tauri updater plugin + signed manifest on GCS |
| Web portals | CI deploy | Cache-bust static assets; feature flags in backend |
| payload-terminal (Expo) | EAS Build + EAS Update | Runtime version + channel `production` |

Binaries are **not** embedded in URLs. Clients poll or receive WS `SYSTEM_APP_OUTDATED` from:

`GET /v1/platform/client-policy?role=DRIVER&platform=ios&version=1.2.3&channel=production`

Response includes `minimum_version`, `recommended_version`, `force_update`, `update_url`, `update_deferred`, `defer_reason`.

## Smart update (non-disruptive)

Server defers hard block when actor has active critical session:

- Order `IN_TRANSIT` or `ARRIVED` (driver / retailer)
- Manifest `LOADING` or `SEALED` (payload / factory)
- Payment `AUTHORIZED` pending capture

Clients queue restart until safe checkpoint (driver: after `COMPLETED`; retailer: after payment settled).

## Cutover sequence

1. `terraform apply` in `infra/terraform` with `project_id`, `tenant_slug`, `enable_gke=true`.
2. Push images to Artifact Registry; deploy `infra/k8s/backend-go` and `infra/k8s/ai-worker`.
3. Run Spanner DDL + seed: `go run ./apps/backend-go/cmd/setup`.
4. Populate Secret Manager (Kafka, JWT, webhooks, Firebase).
5. Smoke: `PUBLIC_BASE_URL=https://api.staging.<domain> bash scripts/cloud_smoke_ssmr.sh`.
6. `make validate-launch-readiness`.

Rollback: scale backend deployment to zero; swap load balancer backend service; Spanner/Kafka data retained.

## Boss handoff

See [`CLOUD_CREDENTIALS_CHECKLIST.md`](CLOUD_CREDENTIALS_CHECKLIST.md) for GCP project, store accounts, and signing keys.
