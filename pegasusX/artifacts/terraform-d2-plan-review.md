# Terraform D2 plan review — 2026-07-20

**Status:** plan only — **NOT applied**

**Project:** `void-494000` · **Region:** `asia-south1` · **Tenant:** `staging`

**Result:** Plan: 59 to add, 0 to change, 0 to destroy.

## Resource counts by type

| Type | Count |
|------|------:|
| `google_secret_manager_secret` | 19 |
| `google_monitoring_alert_policy` | 8 |
| `google_secret_manager_secret_version` | 7 |
| `google_project_iam_member` | 3 |
| `google_monitoring_dashboard` | 2 |
| `google_artifact_registry_repository` | 1 |
| `google_billing_budget` | 1 |
| `google_compute_network` | 1 |
| `google_container_cluster` | 1 |
| `google_redis_instance` | 1 |
| `google_service_account` | 1 |
| `google_service_account_iam_member` | 1 |
| `google_spanner_database` | 1 |
| `google_spanner_instance` | 1 |
| `google_storage_bucket` | 1 |
| `google_storage_bucket_iam_member` | 1 |

**Total creates:** 50

## Cost-driving resources (if applied)

| Resource | Est. monthly (CLOUD_BUDGET_MODEL) |
|----------|----------------------------------:|
| Spanner `pegasusx-staging-spanner` 100 PU regional-asia-south1 | $650–900 |
| Memorystore Redis STANDARD_HA 1 GB | $35–55+ (HA higher than Basic) |
| GKE Autopilot `pegasusx-staging-gke` | $180–280 (with pods) + ~$73 cluster fee |
| Artifact Registry + GCS + secrets + monitoring | low tens |
| Billing budget $1500 @ 80%/100% → email | $0 |
| **Steady-state if full stack applied** | **~$1,050–1,665** |

## D2 fixes applied before final plan

1. Wired `budget_alert_emails` → Monitoring email channels + budget `all_updates_rule`
2. GKE Autopilot: removed non-existent secondary range names `pods`/`services` on auto-mode VPC

## Pre-apply blockers / decisions

- [ ] Review this plan (human sign-off)
- [ ] Confirm Spanner 100 PU cost is acceptable before first paid apply
- [ ] Replace Kafka placeholder `pkc-xxxxx…` before relying on secrets
- [ ] Set real `firebase_project_id` before D11
- [ ] Optional: remote state GCS (`backend.gcs.tf.example`) before multi-operator use
- [ ] GCS `app-updates` grants `allUsers` objectViewer — intentional for desktop updater?

## Artifacts

- `artifacts/terraform-d2-plan-2026-07-20.txt`
- `artifacts/terraform-d2-plan-2026-07-20.tfplan`
- This file: `artifacts/terraform-d2-plan-review.md`

## Command used

```bash
cd pegasusX/infra/terraform
terraform init
terraform plan -var-file=staging.tfvars -out=../../artifacts/terraform-d2-plan-2026-07-20.tfplan
# DO NOT apply until explicit go: terraform apply ../../artifacts/terraform-d2-plan-2026-07-20.tfplan
```

