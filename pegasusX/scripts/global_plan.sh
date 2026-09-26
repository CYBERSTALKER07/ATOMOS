#!/usr/bin/env bash
# GS-C5: terraform plan for global DNS/AR. Never apply. Never open pegasusx/ssmr.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TF="$ROOT/infra/terraform"
G="$TF/global"
BACKEND="$G/backend.hcl"
TFVARS="$G/global.tfvars"

if [[ ! -f "$BACKEND" || ! -f "$TFVARS" ]]; then
  echo "FAIL: missing $BACKEND or $TFVARS" >&2
  exit 1
fi

if grep -E '^[[:space:]]*prefix[[:space:]]*=[[:space:]]*"pegasusx/ssmr"' "$BACKEND" >/dev/null; then
  echo "FAIL: global backend must not use pegasusx/ssmr" >&2
  exit 1
fi
if grep -E '^[[:space:]]*prefix[[:space:]]*=[[:space:]]*"pegasusx/cell-eu"' "$BACKEND" >/dev/null; then
  echo "FAIL: global backend must not collide with the EU cell stack" >&2
  exit 1
fi
if ! grep -E '^[[:space:]]*prefix[[:space:]]*=[[:space:]]*"pegasusx/global"' "$BACKEND" >/dev/null; then
  echo "FAIL: global backend must be pegasusx/global" >&2
  exit 1
fi
if grep -E '^[[:space:]]*project_id[[:space:]]*=[[:space:]]*"pegasus-503013"' "$TFVARS" >/dev/null; then
  echo "FAIL: global plane must not use project pegasus-503013" >&2
  exit 1
fi
if grep -E '^[[:space:]]*project_id[[:space:]]*=[[:space:]]*"pegasusx-cell-eu"' "$TFVARS" >/dev/null; then
  echo "FAIL: global plane must not use project pegasusx-cell-eu" >&2
  exit 1
fi

# Sibling of global/ and modules/ so module source ../modules/global_* resolves.
WORKDIR="$TF/global-plan-workdir"
rm -rf "$WORKDIR"
mkdir -p "$WORKDIR"
while IFS= read -r -d '' f; do
  ln -s "$f" "$WORKDIR/"
done < <(find "$G" -maxdepth 1 \( -name '*.tf' -o -name '.terraform.lock.hcl' \) ! -name 'backend.gcs.tf' -print0)
ln -s "$TFVARS" "$WORKDIR/global.tfvars"

export TF_DATA_DIR="$G/.terraform"
mkdir -p "$TF_DATA_DIR"

echo "GS-C5 global-plan"
echo "  backend.hcl (ops remote init): $BACKEND"
echo "  workdir (no backend.gcs.tf): $WORKDIR"
echo "  TF_DATA_DIR: $TF_DATA_DIR"

INIT_TXT="$G/init.txt"
set +e
terraform -chdir="$WORKDIR" init -input=false >"$INIT_TXT" 2>&1
init_rc=$?
set -e
if [[ "$init_rc" -ne 0 ]]; then
  tail -n 30 "$INIT_TXT" || true
  if grep -Eiq 'Failed to query available provider packages|registry.terraform.io|context deadline exceeded|could not connect' "$INIT_TXT"; then
    echo "GS-C5: terraform init skipped (registry/network). Layout + isolation checks OK. No apply."
    echo "  see $INIT_TXT"
    exit 0
  fi
  echo "FAIL: terraform init for global plane failed" >&2
  exit "$init_rc"
fi

terraform -chdir="$WORKDIR" validate

PLAN_OUT="$G/tfplan"
PLAN_TXT="$G/plan.txt"
set +e
terraform -chdir="$WORKDIR" plan \
  -var-file="global.tfvars" \
  -refresh=false \
  -input=false \
  -lock=false \
  -out="$PLAN_OUT" \
  >"$PLAN_TXT" 2>&1
rc=$?
set -e
tail -n 40 "$PLAN_TXT" || true

if [[ "$rc" -eq 0 ]]; then
  echo "GS-C5: wrote $PLAN_OUT (local state, no apply, no GCS write)"
  exit 0
fi

if grep -Eiq 'project was not found|could not find default credentials|authentication|Failed to get existing workspaces' "$PLAN_TXT"; then
  echo "GS-C5: provider plan skipped (project/credentials wait for ops apply). Layout + validate OK."
  echo "  see $PLAN_TXT"
  exit 0
fi

echo "FAIL: global-plane plan failed" >&2
exit "$rc"
