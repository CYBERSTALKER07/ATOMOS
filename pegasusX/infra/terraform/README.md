# pegasusX Terraform

This module provisions baseline cloud infrastructure for an isolated pegasusX SSMR sandbox:

- VPC network
- Cloud Spanner instance + database
- Memorystore Redis (STANDARD_HA)
- Secret Manager entries for Kafka + Firebase runtime wiring
- Tenant-scoped secret names for orders, spatial, realtime, and webhook Kafka topics

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
  -var="firebase_auth_enabled=true"
terraform apply
```

## Notes

- Kafka is provider-agnostic in this baseline. The module stores bootstrap
  coordinates plus the isolated topic names in Secret Manager so backend-go and
  ai-worker can consume them at deploy time.
- Spanner schema DDL is managed by backend migrations, not Terraform.
- Use a dedicated terraform backend key or workspace per tenant slug. The
  sandbox is meant to be hermetic; do not point multiple SSMR tenants at the
  same terraform state.
- Keep state out of git (`.terraform/`, `*.tfstate*`).

For local bootstrap, use `docker compose -f infra/docker-compose.ssmr.yml up -d`.
