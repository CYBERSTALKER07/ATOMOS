# pegasusX Release Train

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



## Environments

`dev` (local SSMR) → `staging` (GCP + seeded data) → `prod` (load cert + hypercare)

## Staging gate (required)

```bash
cd pegasusX
make test-ssmr-infra
PUBLIC_BASE_URL=https://api.staging.<domain> bash scripts/cloud_smoke_ssmr.sh
PUBLIC_BASE_URL=https://api.staging.<domain> bash scripts/load/load_cert_cloud.sh
bash scripts/parity/role_row_contract_check.sh
make validate-launch-readiness
```

## Production gate

1. All staging gates green within 24h of release candidate.
2. `docs/CLOUD_CREDENTIALS_CHECKLIST.md` secrets verified.
3. Coordinated client releases per `docs/DEPLOYMENT_AND_DISTRIBUTION_PLAN.md` (monorepo tag triggers CI).
4. Hypercare: 72h elevated monitoring (Kafka lag, WS failures, payment webhooks).

## Per-artifact deploy order

1. Spanner DDL migration (`go run ./apps/backend-go/cmd/setup`)
2. `backend-go` + `ai-worker` (GKE rollout)
3. Web portals (supplier / factory / warehouse)
4. Native apps via store pipelines (or sideload CDN for desktop)

## Rollback

Kubernetes rollout undo for API; client rollback via store previous version or Tauri updater channel pin.
