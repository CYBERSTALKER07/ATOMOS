# D2 Terraform apply summary — 2026-07-20

**Project:** `void-494000` · **Region:** `asia-south1` · **Tenant:** `staging`  
**Status:** **Applied** (core infra green; observability deferred)

## Applied successfully

| Resource | ID / name |
|----------|-----------|
| Spanner instance | `pegasusx-staging-spanner` (100 PU, regional-asia-south1, READY) |
| Spanner database | `pegasusx-staging-db` |
| Memorystore Redis | `pegasusx-staging-redis` (STANDARD_HA, 1 GB, READY, port 6378) |
| GKE Autopilot | `pegasusx-staging-gke` (RUNNING, WI pool `void-494000.svc.id.goog`) |
| VPC | `pegasusx-staging-vpc` |
| Artifact Registry | `asia-south1-docker.pkg.dev/void-494000/pegasusx-staging-images` |
| GCS app updates | `pegasusx-staging-app-updates` (+ public objectViewer) |
| GSM secrets | Kafka topics, JWT shell, Maps shell, pay shells, Firebase shells, etc. |
| Runtime SA | `staging-backend@void-494000.iam.gserviceaccount.com` |
| Workload Identity | `roles/iam.workloadIdentityUser` → `pegasusx/backend-go` |
| IAM | Spanner databaseUser, Redis editor, Secret Accessor |
| Billing budget | $1500 / mo @ 80% + 100% → `cyberstalkerx7@gmail.com` |

## First-pass failures (fixed on retry)

1. **Workload Identity binding** — ran before WI pool existed; re-applied after cluster READY → **fixed**.  
2. **Prometheus alert policies** — custom metrics do not exist until apps run. Set `enable_observability_resources = false` → **deferred to D12**.  
3. **Pilot dashboard** — invalid `sparkChartType`; deferred with observability.

## Logs

- `artifacts/terraform-d2-apply-2026-07-20.txt` (first apply; partial errors)  
- `artifacts/terraform-d2-apply-retry-2026-07-20.txt` (**Apply complete**)

## Cost note

Spanner 100 PU + Redis HA + GKE Autopilot control plane are **billing now**. Tear down non-prod when idle for weeks (`terraform destroy` or delete Spanner instance) per cost governance.

## Not done by D2 apply

- [ ] Spanner schema migrations / fiscal tables (D3 migrate)  
- [ ] Real Confluent bootstrap (still placeholder in GSM)  
- [ ] Real Firebase project id  
- [ ] Container images pushed / k8s workloads (D8–D9)  
- [ ] Observability alerts re-enabled after first metrics scrape (D12)  
