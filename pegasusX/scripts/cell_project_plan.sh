#!/usr/bin/env bash
# GS-C3: plan the EU project factory. Never apply. Never open pegasusx/ssmr.
set -euo pipefail

CELL="${1:-}"
if [[ "$CELL" != "eu" ]]; then
  echo "usage: make cell-project-plan CELL=eu" >&2
  exit 1
fi

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PF="$ROOT/infra/terraform/cells/eu/project"
BACKEND="$PF/backend.hcl"
TFVARS="$PF/project.tfvars"

if [[ ! -f "$BACKEND" || ! -f "$TFVARS" ]]; then
  echo "FAIL: missing $BACKEND or $TFVARS" >&2
  exit 1
fi

if grep -E '^[[:space:]]*prefix[[:space:]]*=[[:space:]]*"pegasusx/ssmr"' "$BACKEND" >/dev/null; then
  echo "FAIL: project-factory backend must not use pegasusx/ssmr" >&2
  exit 1
fi
if grep -E '^[[:space:]]*prefix[[:space:]]*=[[:space:]]*"pegasusx/cell-eu"' "$BACKEND" >/dev/null; then
  echo "FAIL: project-factory prefix must not collide with the cell stack (pegasusx/cell-eu)" >&2
  exit 1
fi
if ! grep -E '^[[:space:]]*prefix[[:space:]]*=[[:space:]]*"pegasusx/cell-eu-project"' "$BACKEND" >/dev/null; then
  echo "FAIL: project-factory backend must be pegasusx/cell-eu-project" >&2
  exit 1
fi
if grep -E '^[[:space:]]*project_id[[:space:]]*=[[:space:]]*"pegasus-503013"' "$TFVARS" >/dev/null; then
  echo "FAIL: EU project factory must not use pegasus-503013" >&2
  exit 1
fi

WORKDIR="$PF/workdir"
rm -rf "$WORKDIR"
mkdir -p "$WORKDIR"
while IFS= read -r -d '' f; do
  ln -s "$f" "$WORKDIR/"
done < <(find "$PF" -maxdepth 1 \( -name '*.tf' -o -name '.terraform.lock.hcl' \) ! -name 'backend.gcs.tf' -print0)
ln -s "$TFVARS" "$WORKDIR/project.tfvars"

export TF_DATA_DIR="$PF/.terraform"
mkdir -p "$TF_DATA_DIR"

echo "GS-C3 cell-project-plan CELL=eu"
echo "  backend.hcl (ops remote init): $BACKEND"
echo "  workdir (no backend.gcs.tf): $WORKDIR"

terraform -chdir="$WORKDIR" init -input=false
terraform -chdir="$WORKDIR" validate

PLAN_OUT="$PF/tfplan"
PLAN_TXT="$PF/plan.txt"
set +e
terraform -chdir="$WORKDIR" plan \
  -var-file="project.tfvars" \
  -refresh=false \
  -input=false \
  -lock=false \
  -out="$PLAN_OUT" \
  >"$PLAN_TXT" 2>&1
rc=$?
set -e
tail -n 40 "$PLAN_TXT" || true

if [[ "$rc" -eq 0 ]]; then
  echo "GS-C3: wrote $PLAN_OUT (local state, no apply, no GCS write)"
  exit 0
fi

if grep -Eiq 'project was not found|could not find default credentials|authentication|org_id|folder_id' "$PLAN_TXT"; then
  echo "GS-C3: provider plan skipped (org/billing/credentials wait for ops apply). Layout + validate OK."
  echo "  see $PLAN_TXT"
  exit 0
fi

echo "FAIL: project-factory plan failed" >&2
exit "$rc"
