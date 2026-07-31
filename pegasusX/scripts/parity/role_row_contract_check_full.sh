#!/usr/bin/env bash
# PX12-A: verifies shipped client /v1 paths have backend route registrations.
set -euo pipefail
ulimit -n 10240 || true

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

fail() { echo "parity-contract-full-error: $1" >&2; exit 1; }

ROUTES_TMP="$(mktemp)"
PATHS_TMP="$(mktemp)"
trap 'rm -f "$ROUTES_TMP" "$PATHS_TMP"' EXIT

# Collect registered HTTP paths from route mounts, main.go, and handler doc comments.
grep -RhE '"/v1/[^"]+"' \
  apps/backend-go/*routes/*.go \
  apps/backend-go/main.go \
  apps/backend-go/geolocation/handlers.go \
  apps/backend-go/supplier/import_sessions.go \
  apps/backend-go/order/compliance_audit.go \
  apps/backend-go/order/shop_closed.go \
  apps/backend-go/analytics/*.go \
  2>/dev/null \
  | grep -v '_test.go' \
  | sed -E 's/.*"(\/v1\/[^"]+)".*/\1/' \
  | sort -u >"$ROUTES_TMP"

# Canonicalize chi path params and legacy client placeholders for comparison.
normalize_path() {
  local p="$1"
  # Strip trailing punctuation from doc-comment false positives.
  p="${p%%[.,;:!?)]}"
  # Unify all chi params to {id}.
  echo "$p" | sed -E 's/\{[^}]+\}/{id}/g'
}

segment_match() {
  local client_norm="$1"
  local route_norm="$2"
  local c="${client_norm#/}"
  local r="${route_norm#/}"
  if [[ -z "$c" && -z "$r" ]]; then
    return 0
  fi
  if [[ -z "$c" || -z "$r" ]]; then
    return 1
  fi
  local chead="${c%%/*}"
  local rhead="${r%%/*}"
  local ctail="${c#*/}"
  local rtail="${r#*/}"
  if [[ "$ctail" == "$c" ]]; then ctail=""; fi
  if [[ "$rtail" == "$r" ]]; then rtail=""; fi
  if [[ "$rhead" == "{id}" ]]; then
    :
  elif [[ "$chead" != "$rhead" ]]; then
    return 1
  fi
  if [[ -z "$ctail" && -z "$rtail" ]]; then
    return 0
  fi
  if [[ -z "$ctail" || -z "$rtail" ]]; then
    return 1
  fi
  segment_match "/$ctail" "/$rtail"
}

route_matches() {
  local client_path="$1"
  local norm route rnorm

  norm="$(normalize_path "$client_path")"
  while IFS= read -r route; do
    rnorm="$(normalize_path "$route")"
    if [[ "$norm" == "$rnorm" ]]; then
      return 0
    fi
    if segment_match "$norm" "$rnorm"; then
      return 0
    fi
  done <"$ROUTES_TMP"
  return 1
}

is_test_file() {
  local f="$1"
  case "$f" in
    *Test.swift|*Tests.swift|*_test.go|*.test.ts|*.test.tsx|*.spec.ts|*.spec.tsx) return 0 ;;
  esac
  return 1
}

is_doc_only_path() {
  local path="$1"
  case "$path" in
    /v1/fleet/active|/v1/order-items/*) return 0 ;;
  esac
  return 1
}

should_skip_path() {
  local path="$1"
  [[ -z "$path" ]] && return 0
  [[ "$path" == *"?"* ]] && return 0
  [[ "$path" == */ ]] && return 0
  [[ "$path" == *"Binary file"* ]] && return 0
  # Prefix-only assertions in tests (e.g. hasPrefix("/v1/catalog")).
  case "$path" in
    /v1/auth/retailer|/v1/catalog) return 0 ;;
  esac
  case "$path" in
    /v1/ws|/v1/ws/*|/v1/token|/v1/sync/batch) return 0 ;;
  esac
  if is_doc_only_path "$path"; then
    return 0
  fi
  return 1
}

# Client trees that ship production API calls.
CLIENT_DIRS=(
  apps/supplier-portal
  apps/supplier-app-android
  apps/supplier-app-ios
  apps/retailer-app-desktop
  apps/retailer-app-android
  apps/retailer-app-ios
  apps/driver-app-android
  apps/driverappios
  apps/warehouse-portal
  apps/warehouse-app-android
  apps/warehouse-app-ios
  apps/factory-portal
  apps/factory-app-android
  apps/factory-app-ios
  apps/payload-terminal
  apps/payload-app-android
  apps/payload-app-ios
  packages/api-client
)

for dir in "${CLIENT_DIRS[@]}"; do
  [[ -d "$ROOT/$dir" ]] || continue
  while IFS= read -r -d '' file; do
    is_test_file "$file" && continue
    grep -hoE '/v1/[a-zA-Z0-9_./{}-]+' "$file" 2>/dev/null \
      | sed -E 's/[.,;:!?)]+$//' \
      | sed 's/"//g' \
      | sed "s/'//g" \
      | grep -v '\.\.' || true
  done < <(find "$ROOT/$dir" -type f \( \
    -name '*.ts' -o -name '*.tsx' -o -name '*.js' -o -name '*.jsx' \
    -o -name '*.kt' -o -name '*.swift' -o -name '*.go' -o -name '*.dart' \
  \) ! -path '*/node_modules/*' ! -path '*/.next/*' ! -path '*/build/*' \
    ! -path '*/dist/*' ! -path '*/target/*' ! -path '*/.derivedData/*' \
    -print0 2>/dev/null)
done | sort -u >"$PATHS_TMP"

MISSING=0
while IFS= read -r path; do
  should_skip_path "$path" && continue
  if ! route_matches "$path"; then
    echo "missing-route: $path" >&2
    MISSING=$((MISSING + 1))
  fi
done <"$PATHS_TMP"

# Required P0 paths (PX12-B) must be registered.
REQUIRED=(
  "/v1/auth/refresh"
  "/v1/catalog/categories/{categoryID}/suppliers"
  "/v1/catalog/suppliers/search"
  "/v1/products"
  "/v1/fleet/route/reorder"
  "/v1/delivery/bypass-offload"
  "/v1/delivery/credit-delivery"
  "/v1/delivery/missing-items"
  "/v1/delivery/split-payment"
  "/v1/user/device-token"
  "/v1/retailer/suppliers/{supplierID}/add"
  "/v1/retailer/suppliers/{supplierID}/remove"
  "/v1/platform/geocode/autocomplete"
  "/v1/platform/geocode/place"
  "/v1/platform/geocode/reverse"
  "/v1/platform/geocode/forward"
)
for path in "${REQUIRED[@]}"; do
  if ! route_matches "$path"; then
    echo "missing-required-route: $path" >&2
    MISSING=$((MISSING + 1))
  fi
done

# Legacy narrow checks (PX11-E1).
bash "$ROOT/scripts/parity/role_row_contract_check.sh"

if [[ "$MISSING" -gt 0 ]]; then
  fail "$MISSING client path(s) lack backend route registration"
fi

echo "role-row-contract-full-ok"
