# GS-C4 isolation proof

**Date:** 2026-08-16
**Method:** structural + `go test ./auth/ -run TestCellIsolation_`. No terraform apply. No live EU GCP.

| Claim | Evidence | Result |
|-------|----------|--------|
| EU GSA denied UZ Spanner/GSM | EU project `pegasusx-cell-eu`; `cell_scoped_iam=true`; Spanner DB IAM + per-secret GSM in `gke.tf`; check `non_uz_requires_cell_scoped_iam` | PASS (structural; live IAM deny waits for C3 apply) |
| UZ JWT 401 on EU API | Different HS256 secret → `ErrInvalidToken`; `HOME_CELL=cell-eu` rejects `home_cell=cell-uz` → `ErrWrongCell` | PASS (unit) |
| Kafka cross-bootstrap fails | EU bootstrap `…europe-west1.managedkafka.pegasusx-cell-eu…`; topics `cell-eu.events.*` disjoint from `staging.events.*` | PASS (structural + unit) |
| GSM locations EU-only | `gsm_regional_only=true` + `region=europe-west1` + check `europe_west1_gsm_must_be_regional` | PASS (structural; live GSM locations wait for apply) |

Live leftover: project `pegasusx-cell-eu` is not applied. After ops apply, re-run this script and add `gcloud` deny probes (C4 live).
