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


# Universal Agent & Engineering Guidelines
When developing, designing, or planning, always ensure to account for:
- Gaps, edge cases, and comprehensive feature validation.
- Best practices and optimized integration for Kafka, Redis, Backend, Optimizers, AI, and UI.
- Real-time concepts including WebSockets, webhooks, and their native app equivalents.
- Thorough business logic for features, understanding how the role, app, and ecosystem work together, and engagements with other roles and features.
- Best practices for backend, frontend, and infrastructure libraries/packages. Always prefer existing, high-quality open-source libraries and packages that best suit our features before creating our own.
- Optimal UI infrastructure and UX patterns (e.g., optimal screen positioning for drivers during an active route), applying the same high standards to backend and cloud architecture.
- ALWAYS search the web to find open-source code, libraries, packages, math, algorithms, approaches, and best practices for anything we are doing. If none exist, then create our own.
- Always search the web to get the correct logic, and incorporate edge cases, business logic for features, operations (ops), workflow, data consistency, finance, and AI into everything we do.
