# Gate-0 — Spanner backup / PITR / restore rehearsal

**Project:** `pegasus-503013` (pegasus)  
**Account:** `blackfoxenterprise3697@gmail.com`  
**Date:** 2026-08-05

## Live SSMR database

| Setting | Value |
|---------|--------|
| Instance | `pegasusx-ssmr-spanner` |
| Database | `pegasusx-ssmr-db` |
| `versionRetentionPeriod` | **7d** (was 1h; updated via DDL) |
| `enableDropProtection` | **true** |
| Backup schedule (cloud) | `default_daily_full_backup_schedule` — FULL, cron `14 11 * * *`, retention 7d |
| Latest backup used | `pegasusx-ssmr-db_9b9b15a6-2728-4bb3-b5c6-b190acbb95ac` |

## IaC

- [`infra/terraform/spanner_backup.tf`](../infra/terraform/spanner_backup.tf) — `google_spanner_backup_schedule.daily_full`
- [`infra/terraform/main.tf`](../infra/terraform/main.tf) — `version_retention_period` + drop protection on `google_spanner_database.main`
- [`infra/terraform/backend.gcs.tf`](../infra/terraform/backend.gcs.tf) — remote state bucket `gs://pegasus-503013-terraform-state` (versioning on)

**Note:** Existing console schedule `default_daily_full_backup_schedule` remains authoritative until `terraform import` of the schedule resource; TF defines the intended end state for new environments.

## Restore rehearsal

| Step | Detail |
|------|--------|
| Source backup | `pegasusx-ssmr-db_9b9b15a6-2728-4bb3-b5c6-b190acbb95ac` |
| Destination | `ssmr-restore-rehearse` (same instance) |
| `createTime` | `2026-08-05T17:54:21.655464Z` |
| `READY_OPTIMIZING` | `2026-08-05T18:19:42Z` (~25 min; DB usable) |
| `READY` | `2026-08-05T18:24:35Z` |
| Cleanup | Scratch DB deleted after READY; only `pegasusx-ssmr-db` remains |

## RPO / RTO (SSMR)

| Metric | Value |
|--------|--------|
| RPO (scheduled full) | ≤ 24h (daily schedule) |
| RPO (PITR) | ≤ 7d window after retention change |
| RTO (restore rehearsal) | **~30.2 min** wall clock (`createTime` → `READY`); usable at ~25 min (`READY_OPTIMIZING`) |
