# pegasusX Terraform

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



This module provisions baseline cloud infrastructure for an isolated pegasusX SSMR sandbox:

- VPC network
- Cloud Spanner instance + database
- Memorystore Redis (STANDARD_HA)
- Secret Manager entries for Kafka + Firebase runtime wiring
- Tenant-scoped secret names for orders, spatial, realtime, and webhook Kafka topics
- Optional ai-worker launch-readiness observability resources (dashboard, alert policies, optional uptime checks)

## Usage

### Staging (Phase 0 — PX-PROD-0)

```bash
cd pegasusX
make terraform-init   # -backend-config=backend-ssmr.hcl → prefix pegasusx/ssmr
cp infra/terraform/staging.tfvars.example infra/terraform/staging.tfvars
# Edit staging.tfvars + copy .env.staging.secrets.example → .env.staging.secrets

make phase0-preflight
make phase0-plan
make phase0-apply
make phase0-sync-secrets
make phase0-migrate
```

**GS-C1–C5 (plan only):** state prefix is **not** hardcoded in `backend.gcs.tf`. Live init uses `backend-ssmr.hcl` (`pegasusx/ssmr`). Per-cell files live in `cells/{uz,eu}/`. `make cell-plan CELL=eu` plans the EU stack (isolated local state). `make cell-project-plan CELL=eu` plans project `pegasusx-cell-eu` (`cells/eu/project/`, prefix `pegasusx/cell-eu-project`). Same DDL via `make cell-migrate CELL=eu` (dry-run; never a UZ backup restore). New JWT: `scripts/mint_cell_jwt.sh eu`. Isolation proof: `make cell-isolation-proof`. Global DNS/AR: `make global-plan` (`global/`, prefix `pegasusx/global`, project `pegasusx-global`, modules `global_dns` + `global_ar`). Do not apply from this catalog. Do not use Terraform workspaces. Do not put two cells in `pegasus-503013`.

See [`docs/PHASE_0_CLOUD_FOUNDATION_RUNBOOK.md`](../../docs/PHASE_0_CLOUD_FOUNDATION_RUNBOOK.md).

### Sandbox / custom tenant

```bash
cd infra/terraform
terraform init -backend-config=backend-ssmr.hcl
terraform plan \
  -var="project_id=<gcp-project>" \
  -var="tenant_slug=ssmr" \
  -var="region=asia-south1" \
  -var="kafka_bootstrap_servers=<pkc-xxx.asia-south1.gcp.confluent.cloud:9092>" \
  -var="google_maps_api_key=<server-side-geocoding-places-key>" \
  -var="firebase_project_id=<firebase-project-id>" \
  -var="firebase_auth_enabled=true" \
  -var="enable_observability_resources=true" \
  -var="ai_worker_monitoring_host=<worker-host-or-ip>" \
  -var='alert_notification_channels=["projects/<project>/notificationChannels/<id>"]'
terraform apply
```

## Notes

- **Kafka (Confluent Cloud Basic):** provision one cluster in `asia-south1` (or nearest).
  Create topics matching the cell (`cell-${cell_id}.events.*` when topic vars
  are empty; staging.tfvars still sets `staging.events.*`) and an
  API key with produce/consume ACLs. Pass the bootstrap string to
  `kafka_bootstrap_servers`; Terraform stores it in
  `pegasusx-<tenant>-kafka-bootstrap-servers` for External Secrets. Local SSMR
  keeps Docker Kafka in `infra/docker-compose.ssmr.yml` (no Confluent required).
- **Google Maps Platform:** `maps_platform.tf` enables Routes, Geocoding, Places,
  and Maps backend APIs. Pass `google_maps_api_key` (server: Geocode + Places +
  Routes; IP-restrict to GKE NAT). Optional `maps_android_api_key` /
  `maps_ios_api_key` seed client SDK GSM shells. See
  `CLOUD_CREDENTIALS_CHECKLIST.md`. Geometry order in backend:
  Google Routes → OSRM → dense (`ROUTING_PROVIDER=auto`).
- **Billing:** set `billing_account_id` + `monthly_budget_usd` +
  `budget_alert_emails` so Maps Routes spend is covered by the monthly budget.
- Kafka is provider-agnostic in this baseline. The module stores bootstrap
  coordinates plus the isolated topic names in Secret Manager so backend-go and
  ai-worker can consume them at deploy time.
- When `enable_observability_resources=true`, the module provisions an ai-worker launch dashboard, alert policies for worker-up / readiness / Kafka lag, and optional uptime checks when `ai_worker_monitoring_host` is set.
- Spanner schema DDL is managed by backend migrations, not Terraform.
- Use a dedicated GCS backend prefix per **cell** (`backend-ssmr.hcl` vs a
  copy of `backend-cell.example.hcl`). Do not use Terraform workspaces for
  cells. `tenant_slug` namespaces names; it is not a cell. The sandbox is
  hermetic — do not point a second cell at `pegasusx/ssmr`.
- New cells must set `gsm_regional_only=true` and `vpc_custom_mode=true`.
  Live ssmr/staging keeps both **false** so an accidental apply does not
  ForceNew JWT/PSP secrets or the auto-mode VPC.
- Kafka topic vars that are empty derive `cell-${cell_id}.events.*`.
- Workload Identity member is `${k8s_namespace}/backend-go`.
- Spanner + GSM IAM are instance/secret scoped when `cell_scoped_iam=true`
  (default). Memorystore BASIC has no instance IAM — Redis stays project-level.
- Keep state out of git (`.terraform/`, `*.tfstate*`).

For local bootstrap, use `docker compose -f infra/docker-compose.ssmr.yml up -d`.

Kubernetes launch packaging for the worker now lives under `infra/k8s/ai-worker/` as a minimal configmap, deployment, and ClusterIP service with readiness/liveness probes on the worker monitoring surface.
