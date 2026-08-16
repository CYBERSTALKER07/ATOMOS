#!/usr/bin/env bash
# GS-C2: terraform plan for one cell. Never apply.
# Never reconfigure infra/terraform/.terraform (live metadata → pegasusx/ssmr).
# Catalog plan copies the root into cells/$CELL/workdir *without* backend.gcs.tf
# so Terraform uses local state — no GCS lock, no pegasusx/ssmr, no pegasusx/cell-eu write.
# C3 inits the real EU backend: terraform init -backend-config=cells/eu/backend.hcl
set -euo pipefail

CELL="${1:-}"
case "$CELL" in
  uz|eu) ;;
  *)
    echo "usage: make cell-plan CELL=uz|eu" >&2
    exit 1
    ;;
esac

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TF="$ROOT/infra/terraform"
CELL_DIR="$TF/cells/$CELL"
BACKEND="$CELL_DIR/backend.hcl"
TFVARS="$CELL_DIR/cell.tfvars"

if [[ ! -f "$BACKEND" || ! -f "$TFVARS" ]]; then
  echo "FAIL: missing $BACKEND or $TFVARS" >&2
  exit 1
fi

if [[ "$CELL" == "eu" ]]; then
  if grep -E '^[[:space:]]*prefix[[:space:]]*=[[:space:]]*"pegasusx/ssmr"' "$BACKEND" >/dev/null; then
    echo "FAIL: cells/eu/backend.hcl must not use prefix pegasusx/ssmr" >&2
    exit 1
  fi
  if ! grep -E '^[[:space:]]*prefix[[:space:]]*=[[:space:]]*"pegasusx/cell-eu"' "$BACKEND" >/dev/null; then
    echo "FAIL: cells/eu/backend.hcl must set prefix pegasusx/cell-eu" >&2
    exit 1
  fi
  if grep -E '^[[:space:]]*project_id[[:space:]]*=[[:space:]]*"pegasus-503013"' "$TFVARS" >/dev/null; then
    echo "FAIL: cells/eu must not use project pegasus-503013" >&2
    exit 1
  fi
  if ! grep -E '^[[:space:]]*gsm_regional_only[[:space:]]*=[[:space:]]*true' "$TFVARS" >/dev/null; then
    echo "FAIL: cells/eu must set gsm_regional_only=true" >&2
    exit 1
  fi
  if ! grep -E '^[[:space:]]*vpc_custom_mode[[:space:]]*=[[:space:]]*true' "$TFVARS" >/dev/null; then
    echo "FAIL: cells/eu must set vpc_custom_mode=true" >&2
    exit 1
  fi
  if ! grep -E '^[[:space:]]*region[[:space:]]*=[[:space:]]*"europe-west1"' "$TFVARS" >/dev/null; then
    echo "FAIL: cells/eu must set region europe-west1" >&2
    exit 1
  fi
fi

bash "$ROOT/scripts/assert_cell_backend.sh"

WORKDIR="$CELL_DIR/workdir"
rm -rf "$WORKDIR"
mkdir -p "$WORKDIR"
# Link the root modules but omit the GCS backend block.
while IFS= read -r -d '' f; do
  ln -s "$f" "$WORKDIR/"
done < <(find "$TF" -maxdepth 1 \( -name '*.tf' -o -name '.terraform.lock.hcl' \) ! -name 'backend.gcs.tf' -print0)
ln -s "$TFVARS" "$WORKDIR/cell.tfvars"

export TF_DATA_DIR="$CELL_DIR/.terraform"
mkdir -p "$TF_DATA_DIR"

echo "GS-C2 cell-plan CELL=$CELL"
echo "  backend.hcl (C3 remote init): $BACKEND"
echo "  tfvars: $TFVARS"
echo "  workdir (no backend.gcs.tf): $WORKDIR"
echo "  TF_DATA_DIR: $TF_DATA_DIR"

terraform -chdir="$WORKDIR" init -input=false
terraform -chdir="$WORKDIR" validate

PLAN_OUT="$CELL_DIR/tfplan"
PLAN_TXT="$CELL_DIR/plan.txt"
set +e
terraform -chdir="$WORKDIR" plan \
  -var-file="cell.tfvars" \
  -refresh=false \
  -input=false \
  -lock=false \
  -out="$PLAN_OUT" \
  >"$PLAN_TXT" 2>&1
rc=$?
set -e
tail -n 50 "$PLAN_TXT" || true

if [[ "$rc" -eq 0 ]]; then
  echo "GS-C2: wrote $PLAN_OUT (local state, no apply, no GCS write)"
  exit 0
fi

if grep -Eiq 'project was not found|could not find default credentials|unable to detect a resource location|Failed to get existing workspaces|authentication|google: could not find default credentials' "$PLAN_TXT"; then
  echo "GS-C2: provider plan skipped (GCP project/credentials wait for C3). Layout + isolation + validate OK."
  echo "  see $PLAN_TXT"
  exit 0
fi

echo "FAIL: terraform plan for CELL=$CELL failed (not an expected missing-project/auth skip)" >&2
exit "$rc"
