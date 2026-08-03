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
make terraform-init
cp infra/terraform/staging.tfvars.example infra/terraform/staging.tfvars
# Edit staging.tfvars + copy .env.staging.secrets.example → .env.staging.secrets

make phase0-preflight
make phase0-plan
make phase0-apply
make phase0-sync-secrets
make phase0-migrate
```

See [`docs/PHASE_0_CLOUD_FOUNDATION_RUNBOOK.md`](../../docs/PHASE_0_CLOUD_FOUNDATION_RUNBOOK.md).

### Sandbox / custom tenant

```bash
cd infra/terraform
terraform init
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
  Create topics matching defaults (`ssmr.events.orders`, `ssmr.events.spatial`,
  `ssmr.events.realtime`, `ssmr.events.webhooks`, `pegasusx-freeze-locks`) and an
  API key with produce/consume ACLs. Pass the bootstrap string to
  `kafka_bootstrap_servers`; Terraform stores it in
  `pegasusx-<tenant>-kafka-bootstrap-servers` for External Secrets. Local SSMR
  keeps Docker Kafka in `infra/docker-compose.ssmr.yml` (no Confluent required).
- **Google Maps:** enable Geocoding API + Places API; restrict the server key to
  backend egress IPs. Android Maps SDK keys are separate app-restricted keys per
  `CLOUD_CREDENTIALS_CHECKLIST.md`. Pass `google_maps_api_key` on apply to seed GSM.
- Kafka is provider-agnostic in this baseline. The module stores bootstrap
  coordinates plus the isolated topic names in Secret Manager so backend-go and
  ai-worker can consume them at deploy time.
- When `enable_observability_resources=true`, the module provisions an ai-worker launch dashboard, alert policies for worker-up / readiness / Kafka lag, and optional uptime checks when `ai_worker_monitoring_host` is set.
- Spanner schema DDL is managed by backend migrations, not Terraform.
- Use a dedicated terraform backend key or workspace per tenant slug. The
  sandbox is meant to be hermetic; do not point multiple SSMR tenants at the
  same terraform state.
- Keep state out of git (`.terraform/`, `*.tfstate*`).

For local bootstrap, use `docker compose -f infra/docker-compose.ssmr.yml up -d`.

Kubernetes launch packaging for the worker now lives under `infra/k8s/ai-worker/` as a minimal configmap, deployment, and ClusterIP service with readiness/liveness probes on the worker monitoring surface.
