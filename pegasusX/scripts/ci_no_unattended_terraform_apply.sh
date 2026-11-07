#!/usr/bin/env bash
# Fail if a Layer A / sandbox CI workflow contains `terraform apply` without
# workflow_dispatch AND environment: layer-b-ops.
# Deploy-production / pegasusx-deploy-gke are Layer B ops and are out of this program.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO="$(cd "$ROOT/.." && pwd)"

scan_files=()
for f in \
  "$REPO/.github/workflows/sandbox-infra.yml" \
  "$REPO/.github/workflows/ssmr-infra.yml" \
  "$REPO/.github/workflows/pegasusx-ci.yml" \
  "$REPO/.github/workflows/reusable-go-unit.yml" \
  "$ROOT/.github/workflows/"*.yml
do
  [[ -f "$f" ]] || continue
  scan_files+=("$f")
done

fail=0
for f in "${scan_files[@]}"; do
  if ! grep -E 'terraform[[:space:]]+apply' "$f" >/dev/null 2>&1; then
    continue
  fi
  # Ignore comments and GitHub `name:` labels; require a command-like apply.
  if ! grep -E '^[[:space:]]*(run:[[:space:]]*)?terraform[[:space:]]+apply' "$f" >/dev/null 2>&1; then
    continue
  fi
  has_dispatch=0
  has_env=0
  if grep -E 'workflow_dispatch' "$f" >/dev/null; then
    has_dispatch=1
  fi
  if grep -E 'environment:[[:space:]]*layer-b-ops' "$f" >/dev/null; then
    has_env=1
  fi
  if [[ "$has_dispatch" -ne 1 || "$has_env" -ne 1 ]]; then
    echo "FAIL: $f contains terraform apply without workflow_dispatch + environment: layer-b-ops" >&2
    fail=1
  fi
done

if [[ "$fail" -ne 0 ]]; then
  exit 1
fi
echo "ci-no-unattended-terraform-apply-ok"
