# pegasusX Terraform

This module provisions baseline cloud infrastructure for an isolated pegasusX SSMR sandbox:

- VPC network
- Cloud Spanner instance + database
- Memorystore Redis (STANDARD_HA)
- Secret Manager entries for Kafka + Firebase runtime wiring
- Tenant-scoped secret names for orders, spatial, realtime, and webhook Kafka topics
- Optional ai-worker launch-readiness observability resources (dashboard, alert policies, optional uptime checks)

## Usage

```bash
cd infra/terraform
terraform init
terraform plan \
  -var="project_id=<gcp-project>" \
  -var="tenant_slug=ssmr" \
  -var="region=asia-south1" \
  -var="kafka_bootstrap_servers=<host1:9092,host2:9092>" \
  -var="firebase_project_id=<firebase-project-id>" \
  -var="firebase_auth_enabled=true" \
  -var="enable_observability_resources=true" \
  -var="ai_worker_monitoring_host=<worker-host-or-ip>" \
  -var='alert_notification_channels=["projects/<project>/notificationChannels/<id>"]'
terraform apply
```

## Notes

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
