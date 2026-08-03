# D2 apply summary — pegasus-503013 — 2026-07-20

**Account:** `blackfoxenterprise3697@gmail.com`  
**Project:** `pegasus-503013`  
**Status:** **Apply complete**

## Live resources

| Resource | Name / URI | Status |
|----------|------------|--------|
| Spanner | `pegasusx-staging-spanner` / `pegasusx-staging-db` · 100 PU | READY |
| Spanner URI | `projects/pegasus-503013/instances/pegasusx-staging-spanner/databases/pegasusx-staging-db` | |
| Redis | `pegasusx-staging-redis` STANDARD_HA 1GB · port 6378 | READY |
| GKE Autopilot | `pegasusx-staging-gke` · asia-south1 | RUNNING |
| VPC | `pegasusx-staging-vpc` | live |
| Artifact Registry | `asia-south1-docker.pkg.dev/pegasus-503013/pegasusx-staging-images` | live |
| GCS updates | `pegasus-503013-pegasusx-staging-app-updates` | live |
| Budget | $1500 @ 80%/100% → blackfoxenterprise3697@gmail.com | live |
| Runtime SA | `staging-backend@pegasus-503013.iam.gserviceaccount.com` | WI bound |

## Fix during apply

GCS bucket name `pegasusx-staging-app-updates` was globally taken (old project).  
Now: `${project_id}-${resource_prefix}-app-updates`.

## Billing

Spanner + Redis HA + GKE are **on**. Tear down old `void-494000` when ready to avoid double spend.

## Next

1. D3 — Spanner migrations  
2. D4 — Redis AUTH prove in VPC  
3. D5 — real Confluent bootstrap  
4. D8–D9 — images + deploy  
