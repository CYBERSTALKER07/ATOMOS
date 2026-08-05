# Gate-0 Batch B — GCP / Spanner / prod images / OSRM

**Project:** `pegasus-503013`  
**Date:** 2026-08-05

## Done

| Item | Evidence |
|------|----------|
| Spanner PITR 7d + drop protection | Live on `pegasusx-ssmr-db`; TF `version_retention_period` |
| Daily full backups | Cloud schedule + TF `spanner_backup.tf` |
| Restore rehearsal | Scratch → READY (~30.2 min RTO), deleted — see `GATE0_SPANNER_BACKUP_RESTORE_2026-08-05.md` |
| GCS remote Terraform state | Bucket `gs://pegasus-503013-terraform-state` (versioning); `backend.gcs.tf`; `terraform init -migrate-state` |
| Prod image digests | asia-south1 AR pins + cronjob remaps; `scripts/ci_fail_placeholder_images.sh` OK |
| ManagedCertificate | `overlays/prod/managed-certificate.yaml` + ingress patch |
| OSRM PVC + fail-loud | `infra/k8s/osrm/{pvc,deployment,service}.yaml` — pod refuses start without `/data/region.osrm` |

## Residual (owner / cluster ops)

| Item | Why deferred |
|------|----------------|
| Populate OSRM PVC with Uzbekistan (or target) extract | Needs cluster write + Geofabrik download (~GB); manifests ready |
| `terraform import` of existing `default_daily_full_backup_schedule` | Avoid duplicate schedule until import |
| Publish `optimizer-core` image | Prod replicas scaled to 0 until AR image exists |
| Apply prod overlay to live cluster | Image/TLS manifests ready; apply is a release step |
