# pegasusX Cloud Cutover Runbook

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



## Preconditions

- `make test-ssmr-infra` green locally (`PX_E2E_ORDER_OK`, `PX_E2E_CLIENT_POLICY_OK`, `__SSMR_OK__`).
- `docs/CLOUD_CREDENTIALS_CHECKLIST.md` items provisioned.
- `docs/DEPLOYMENT_AND_DISTRIBUTION_PLAN.md` reviewed.
- Container images built and pushed to Artifact Registry (`enable_gke=true` in Terraform).

## Sequence

1. `terraform apply` in `pegasusX/infra/terraform` with `project_id` and `tenant_slug`.
2. Run Spanner DDL + seed: `go run ./apps/backend-go/cmd/setup` with cloud env.
3. Deploy `infra/k8s/backend` and `infra/k8s/ai-worker` manifests.
4. Set secrets: JWT, webhook HMAC, Kafka topic names, Redis addr, Spanner DSN.
5. Enable Firebase on mobile clients; supplier portal keeps cookie JWT.
6. Smoke staging: health, SSMR-equivalent e2e script against `PUBLIC_BASE_URL`.
7. `make validate-launch-readiness`.

## Rollback

- Scale backend deployment to zero; keep Spanner/Kafka data intact.
- Re-point DNS to previous revision via load balancer backend service swap.

## Firebase enablement

- Backend: `FIREBASE_AUTH_ENABLED=true`, `FIREBASE_PROJECT_ID=<id>`.
- Retailer/driver routes use bearer verification; supplier portal unchanged.
