# pegasusX Terraform

This module provisions baseline cloud infrastructure for pegasusX:

- VPC network
- Cloud Spanner instance + database
- Memorystore Redis (STANDARD_HA)
- Secret Manager entries for Kafka + Firebase runtime wiring

## Usage

```bash
cd infra/terraform
terraform init
terraform plan \
  -var="project_id=<gcp-project>" \
  -var="region=asia-south1" \
  -var="kafka_bootstrap_servers=<host1:9092,host2:9092>" \
  -var="firebase_project_id=<firebase-project-id>" \
  -var="firebase_auth_enabled=true"
terraform apply
```

## Notes

- Kafka is provider-agnostic in this baseline. The module stores bootstrap
  coordinates in Secret Manager so backend-go and ai-worker can consume them
  at deploy time.
- Spanner schema DDL is managed by backend migrations, not Terraform.
- Keep state out of git (`.terraform/`, `*.tfstate*`).
