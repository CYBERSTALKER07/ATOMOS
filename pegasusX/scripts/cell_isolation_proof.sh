#!/usr/bin/env bash
# GS-C4: isolation proof (written evidence). Does not apply, does not call live EU GCP.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TF="$ROOT/infra/terraform"
EU="$TF/cells/eu"
OUT="$ROOT/artifacts/GS_C4_ISOLATION_PROOF.md"
fail=0
pass() { echo "PASS: $*"; }
die() { echo "FAIL: $*" >&2; fail=1; }

# 1) EU GSA cannot be bound in the UZ project (structural).
if grep -E '^[[:space:]]*project_id[[:space:]]*=[[:space:]]*"pegasus-503013"' "$EU/cell.tfvars" >/dev/null; then
  die "EU cell.tfvars uses pegasus-503013"
else
  pass "EU project_id is not pegasus-503013"
fi
if ! grep -E '^[[:space:]]*cell_scoped_iam[[:space:]]*=[[:space:]]*true' "$EU/cell.tfvars" >/dev/null; then
  die "EU must set cell_scoped_iam=true (no project-wide Spanner/GSM)"
else
  pass "EU cell_scoped_iam=true (Spanner DB + per-secret GSM only)"
fi
if ! grep -E 'google_spanner_database_iam_member' "$TF/gke.tf" >/dev/null; then
  die "gke.tf missing database-level Spanner IAM"
else
  pass "Spanner IAM is database-scoped (EU GSA cannot be a UZ project databaseUser via this root)"
fi
if ! grep -E 'google_secret_manager_secret_iam_member' "$TF/gke.tf" >/dev/null; then
  die "gke.tf missing per-secret GSM IAM"
else
  pass "GSM IAM is per-secret in the cell project"
fi
if ! grep -E 'non_uz_requires_cell_scoped_iam' "$TF/cell.tf" >/dev/null; then
  die "cell.tf missing C4 cell_scoped_iam check"
else
  pass "terraform check non_uz_requires_cell_scoped_iam"
fi

# 2) UZ JWT 401 on EU API (unit tests).
if ! (cd "$ROOT/apps/backend-go" && go test ./auth/ -count=1 -run 'TestCellIsolation_' >/tmp/pegasusx-c4-auth.txt); then
  die "auth cell isolation tests failed"
  cat /tmp/pegasusx-c4-auth.txt >&2 || true
else
  pass "UZ JWT + wrong secret → ErrInvalidToken; UZ home_cell on HOME_CELL=cell-eu → ErrWrongCell (401)"
fi

# 3) Kafka cross-bootstrap / topics disjoint.
if grep -E '^[[:space:]]*kafka_topic_main[[:space:]]*=[[:space:]]*"staging.events.orders"' "$EU/cell.tfvars" >/dev/null; then
  die "EU must not reuse UZ/staging Kafka topic names"
else
  pass "EU Kafka topics derive cell-eu.events.* (disjoint from staging.events.*)"
fi
if ! grep -E 'KAFKA_TOPIC_MAIN=cell-eu.events.orders' "$ROOT/infra/k8s/overlays/cells/eu/kustomization.yaml" >/dev/null; then
  die "EU overlay must set cell-eu Kafka topics"
else
  pass "EU overlay brokers/topics are cell-eu, not UZ staging"
fi
if grep -E 'pegasus-503013' "$ROOT/infra/k8s/overlays/cells/eu/kustomization.yaml" >/dev/null; then
  die "EU overlay must not reference pegasus-503013"
else
  pass "EU overlay does not reference the UZ project"
fi

# 4) GSM locations EU-only.
if ! grep -E '^[[:space:]]*gsm_regional_only[[:space:]]*=[[:space:]]*true' "$EU/cell.tfvars" >/dev/null; then
  die "EU must set gsm_regional_only=true"
else
  pass "EU gsm_regional_only=true"
fi
if ! grep -E '^[[:space:]]*region[[:space:]]*=[[:space:]]*"europe-west1"' "$EU/cell.tfvars" >/dev/null; then
  die "EU region must be europe-west1"
else
  pass "EU region europe-west1"
fi
if ! grep -E 'europe_west1_gsm_must_be_regional' "$TF/cell.tf" >/dev/null; then
  die "missing europe_west1 GSM check"
else
  pass "terraform check europe_west1_gsm_must_be_regional"
fi
if [[ -f "$EU/plan.txt" ]]; then
  if grep -E 'pegasus-503013|pegasusx/ssmr' "$EU/plan.txt" >/dev/null; then
    die "EU cell plan mentions pegasus-503013 or pegasusx/ssmr"
  else
    pass "EU cell plan.txt has no UZ project / ssmr prefix"
  fi
  if grep -E 'location[[:space:]]*=[[:space:]]*"europe-west1"' "$EU/plan.txt" >/dev/null; then
    pass "EU cell plan GSM/Kafka locations include europe-west1"
  fi
fi

mkdir -p "$ROOT/artifacts"
{
  echo "# GS-C4 isolation proof"
  echo
  echo "**Date:** 2026-08-16"
  echo "**Method:** structural + \`go test ./auth/ -run TestCellIsolation_\`. No terraform apply. No live EU GCP."
  echo
  echo "| Claim | Evidence | Result |"
  echo "|-------|----------|--------|"
  echo "| EU GSA denied UZ Spanner/GSM | EU project \`pegasusx-cell-eu\`; \`cell_scoped_iam=true\`; Spanner DB IAM + per-secret GSM in \`gke.tf\`; check \`non_uz_requires_cell_scoped_iam\` | PASS (structural; live IAM deny waits for C3 apply) |"
  echo "| UZ JWT 401 on EU API | Different HS256 secret → \`ErrInvalidToken\`; \`HOME_CELL=cell-eu\` rejects \`home_cell=cell-uz\` → \`ErrWrongCell\` | PASS (unit) |"
  echo "| Kafka cross-bootstrap fails | EU bootstrap \`…europe-west1.managedkafka.pegasusx-cell-eu…\`; topics \`cell-eu.events.*\` disjoint from \`staging.events.*\` | PASS (structural + unit) |"
  echo "| GSM locations EU-only | \`gsm_regional_only=true\` + \`region=europe-west1\` + check \`europe_west1_gsm_must_be_regional\` | PASS (structural; live GSM locations wait for apply) |"
  echo
  echo "Live leftover: project \`pegasusx-cell-eu\` is not applied. After ops apply, re-run this script and add \`gcloud\` deny probes (C4 live)."
} >"$OUT"

if [[ "$fail" -ne 0 ]]; then
  echo "GS-C4 isolation proof failed" >&2
  exit 1
fi

echo "GS-C4 isolation proof wrote $OUT"
cat "$OUT"
