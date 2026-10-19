#!/usr/bin/env bash
# Repo hygiene gate for Phase 0 legal/safety leftovers.
# Fails if secrets, terraform state, compiled binaries, or junk patches are tracked.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"
fail=0

tracked="$(git ls-files)"

deny_pat() {
  local label=$1
  local pat=$2
  local hits
  hits=$(printf '%s\n' "$tracked" | rg -n "$pat" || true)
  if [[ -n "$hits" ]]; then
    echo "FAIL tracked $label:" >&2
    echo "$hits" >&2
    fail=1
  else
    echo "OK  no tracked $label"
  fi
}

deny_pat "terraform state" '\.tfstate($|\.)'
deny_pat "local env secrets" '(^|/)\.env\.local$|(^|/)\.env\.staging\.secrets$|(^|/)\.env\.k8s\.generated'
deny_pat "compiled backend binary" '(^|/)apps/backend-go/backend-go$'
deny_pat "bak/orig junk" '\.(bak|orig)$'
deny_pat "root patch scripts" '^patch_[^/]+\.sh$'

archive_hits=$(printf '%s\n' "$tracked" | rg -n '^artifacts/tfstate-archive/' | rg -v 'README\.md$' || true)
if [[ -n "$archive_hits" ]]; then
  echo "FAIL tracked tfstate archive secrets (only README.md allowed):" >&2
  echo "$archive_hits" >&2
  fail=1
else
  echo "OK  no tracked tfstate archive secrets"
fi

# Working tree: reject an unignored local tfstate sitting under infra/terraform.
if compgen -G 'infra/terraform/*.tfstate*' >/dev/null 2>&1; then
  echo "FAIL local terraform state present under infra/terraform (use GCS backend)" >&2
  ls -la infra/terraform/*.tfstate* >&2 || true
  fail=1
else
  echo "OK  no local infra/terraform/*.tfstate*"
fi

if [[ ! -f infra/terraform/backend.gcs.tf ]]; then
  echo "FAIL infra/terraform/backend.gcs.tf missing (GCS remote state required)" >&2
  fail=1
else
  echo "OK  GCS terraform backend configured"
fi

if [[ "$fail" -ne 0 ]]; then
  echo "repo-hygiene-gate FAILED" >&2
  exit 1
fi
echo "repo-hygiene-gate-ok"
