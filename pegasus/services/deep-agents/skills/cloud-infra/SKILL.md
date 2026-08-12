---
name: cloud-infra
description: K8s overlays, Terraform, Cloud Build, Spanner/Redis/Kafka env parity, secrets, prod image truth.
---

# Cloud / infra

## Anchors

- `infra/k8s/` overlays: ssmr, staging, prod
- `infra/terraform/`
- `cloudbuild*.yaml` (build/push — not always CD apply)
- ConfigMaps / ExternalSecrets / GSM

## Checks

1. Prod images real digests (optimizer not remapped to backend-go).
2. Kafka/Redis/Spanner env vars match runtime expectations.
3. Secrets never hardcoded; ESO/GSM versions non-empty for prod.
4. CronJobs referenced by overlays (forecast/accuracy not orphaned).
5. Run-mode: API vs worker deployments both considered for relay/consumers.

## Local

- `make infra-up` / docker-compose: Spanner emulator, Redis, Kafka
- Backend needs `SPANNER_EMULATOR_HOST`

## Docs

`docs/CLOUD_CREDENTIALS_CHECKLIST.md`, `docs/DEPLOYMENT_READINESS_GAP_LEDGER.md`
