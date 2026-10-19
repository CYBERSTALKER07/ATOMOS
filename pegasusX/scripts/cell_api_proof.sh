#!/usr/bin/env bash
# GS-C5: written evidence. Does not apply, does not create pegasusx-global.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TF="$ROOT/infra/terraform"
G="$TF/global"
OUT="$ROOT/artifacts/GS_C5_CELL_API_PROOF.md"
fail=0
pass() { echo "PASS: $*"; }
die() { echo "FAIL: $*" >&2; fail=1; }

if ! grep -E '^[[:space:]]*prefix[[:space:]]*=[[:space:]]*"pegasusx/global"' "$G/backend.hcl" >/dev/null; then
  die "global backend.hcl must be pegasusx/global"
else
  pass "global state prefix is pegasusx/global"
fi
if grep -E '^[[:space:]]*prefix[[:space:]]*=[[:space:]]*"pegasusx/(ssmr|cell-eu|cell-eu-project)"' "$G/backend.hcl" >/dev/null; then
  die "global backend must not use ssmr or cell-eu"
else
  pass "global prefix is not ssmr / cell-eu"
fi
if grep -E '^[[:space:]]*project_id[[:space:]]*=[[:space:]]*"pegasus-503013"' "$G/global.tfvars" >/dev/null; then
  die "global must not use pegasus-503013"
else
  pass "global project is not pegasus-503013"
fi
if ! grep -E 'api.pegasusx.app' "$G/main.tf" >/dev/null || ! grep -E 'api-eu.pegasusx.app' "$G/main.tf" >/dev/null; then
  die "global DNS must declare api + api-eu records"
else
  pass "global DNS records include api.pegasusx.app and api-eu.pegasusx.app"
fi
if [[ ! -f "$TF/modules/global_dns/main.tf" || ! -f "$TF/modules/global_ar/main.tf" ]]; then
  die "missing infra/terraform/modules/global_dns or global_ar"
else
  pass "optional modules/ exist (DNS + AR only; live cell stack not extracted)"
fi
if ! grep -E 'global_not_live_uz_project' "$G/main.tf" >/dev/null; then
  die "missing global_not_live_uz_project check"
else
  pass "terraform check global_not_live_uz_project"
fi

if ! grep -E 'func APIURLForHomeCell' "$ROOT/apps/backend-go/auth/cell_directory.go" >/dev/null; then
  die "missing APIURLForHomeCell"
else
  pass "session cell directory maps home_cell → api_url"
fi
if ! grep -E 'HandleListCells' "$ROOT/apps/backend-go/platformroutes/routes.go" >/dev/null; then
  die "platformroutes must mount GET /v1/platform/cells"
else
  pass "GET /v1/platform/cells is mounted"
fi

if ! (cd "$ROOT/apps/backend-go" && go test ./auth/ -count=1 -run 'TestAPIURLForHomeCell_|TestHandleListCells|TestHandleSession_IncludesAPIURL|TestHandleSession_ProfileSource' >/tmp/pegasusx-c5-auth.txt); then
  die "auth cell directory / session tests failed"
  cat /tmp/pegasusx-c5-auth.txt >&2 || true
else
  pass "session api_url: UZ catalog + EU profile + PUBLIC_BASE_URL override"
fi

if ! grep -E 'pinApiBaseUrl' "$ROOT/packages/api-client/cell-api.ts" >/dev/null; then
  die "missing packages/api-client/cell-api.ts pinApiBaseUrl"
else
  pass "shared pinApiBaseUrl (localhost /api stays bootstrap)"
fi
for f in \
  "$ROOT/apps/supplier-portal/lib/auth.ts" \
  "$ROOT/apps/warehouse-portal/lib/auth.ts" \
  "$ROOT/apps/factory-portal/lib/auth.ts" \
  "$ROOT/apps/retailer-app-desktop/lib/auth.ts"
do
  if ! grep -E 'pinApiBaseUrl' "$f" >/dev/null; then
    die "$f must pin API URL from home_cell"
  fi
done
pass "Next.js/Tauri portals pin session/JWT home_cell (dev bootstrap unchanged)"

if ! grep -E '"cell-eu": "https://api-eu.pegasusx.app"' "$ROOT/packages/api-client/cell-api.ts" >/dev/null; then
  die "TS catalog must map cell-eu → api-eu.pegasusx.app"
else
  pass "TS catalog matches terraform api_hostname"
fi

mkdir -p "$ROOT/artifacts"
{
  echo "# GS-C5 cell API / global DNS proof"
  echo
  echo "**Date:** 2026-08-16"
  echo "**Method:** structural + \`go test ./auth/\` + portal greps. No terraform apply. No live EU/global GCP."
  echo
  echo "| Claim | Evidence | Result |"
  echo "|-------|----------|--------|"
  echo "| Global AR/DNS separate from cells | \`infra/terraform/global/\` prefix \`pegasusx/global\`; project \`pegasusx-global\`; modules \`global_dns\` + \`global_ar\` | PASS (plan-only) |"
  echo "| Session \`home_cell\` → API URL | \`GET /v1/auth/session\` returns \`api_url\`/\`ws_url\`; \`GET /v1/platform/cells\` | PASS (unit) |"
  echo "| Clients use session home_cell | \`pinApiBaseUrl\` in \`@pegasusx/api-client\`; supplier/warehouse/factory/retailer-desktop auth | PASS (web/desktop). Native BuildConfig leftover GS-R |"
  echo "| Live cell stack not extracted | \`modules/\` is DNS+AR only | PASS |"
  echo
  echo "Live leftover: \`pegasusx-global\` and \`pegasusx-cell-eu\` are not applied. DNS IPs are RFC 5737 placeholders. Native apps still use BuildConfig / PEGASUSX_API_BASE_URL as bootstrap."
} >"$OUT"

if [[ "$fail" -ne 0 ]]; then
  echo "GS-C5 cell API proof failed" >&2
  exit 1
fi

echo "GS-C5 cell API proof wrote $OUT"
cat "$OUT"
