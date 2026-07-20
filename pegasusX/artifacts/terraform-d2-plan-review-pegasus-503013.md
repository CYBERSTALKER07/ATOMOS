# Terraform D2 plan — pegasus-503013 — 2026-07-20
**Status:** plan only — **NOT applied**
**Account:** `blackfoxenterprise3697@gmail.com`
**Project:** `pegasus-503013` · **Region:** `asia-south1` · **Tenant:** `staging`
**Result:** Plan: 49 to add, 0 to change, 0 to destroy.
## Resource counts by type

| Type | Count |
|------|------:|
| `google_secret_manager_secret` | 19 |
| `google_project_service` | 8 |
| `google_secret_manager_secret_version` | 7 |
| `google_project_iam_member` | 3 |
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

**Total create lines matched:** 48

## Cost drivers if applied

| Resource | Est. monthly |
|----------|-------------:|
| Spanner 100 PU regional-asia-south1 | $650–900 |
| Memorystore Redis STANDARD_HA 1 GB | $35–55+ |
| GKE Autopilot | $180–280 + cluster fee |
| Budget $1500 alerts | $0 |
| **Envelope** | **~$1,050–1,665** |

## Artifacts
- `artifacts/terraform-d2-plan-pegasus-503013-2026-07-20.tfplan`
- `artifacts/terraform-d2-plan-pegasus-503013-2026-07-20.txt`

## Next

Say **`apply d2`** to create resources (starts Spanner bill).
