#!/usr/bin/env bash
# Build all six iOS role apps (simulator, unsigned) for Gate-0 CI.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if ! command -v xcodebuild >/dev/null 2>&1; then
  echo "xcodebuild not available — skip iOS CI on this runner"
  exit 0
fi

SDK="${IOS_CI_SDK:-iphonesimulator}"
DEST="${IOS_CI_DESTINATION:-generic/platform=iOS Simulator}"

build_one() {
  local name="$1"
  local project="$2"
  local scheme="$3"
  local gen_dir="${4:-}"

  echo "==== iOS CI: $name ===="
  if [[ -n "$gen_dir" && -f "$gen_dir/project.yml" ]] && command -v xcodegen >/dev/null 2>&1; then
    (cd "$gen_dir" && xcodegen generate)
  fi
  if [[ ! -d "$project" ]]; then
    echo "FAIL: missing project $project"
    return 1
  fi
  xcodebuild \
    -project "$project" \
    -scheme "$scheme" \
    -sdk "$SDK" \
    -destination "$DEST" \
    -configuration Debug \
    CODE_SIGNING_ALLOWED=NO \
    CODE_SIGNING_REQUIRED=NO \
    build
  echo "OK: $name"
}

failed=0

build_one supplier \
  "$ROOT/apps/supplier-app-ios/SupplierAppIOS.xcodeproj" \
  SupplierAppIOS \
  "$ROOT/apps/supplier-app-ios" || failed=1

build_one warehouse \
  "$ROOT/apps/warehouse-app-ios/WarehouseAppIOS.xcodeproj" \
  WarehouseAppIOS \
  "$ROOT/apps/warehouse-app-ios" || failed=1

build_one factory \
  "$ROOT/apps/factory-app-ios/FactoryAppIOS.xcodeproj" \
  FactoryAppIOS \
  "$ROOT/apps/factory-app-ios" || failed=1

build_one payload \
  "$ROOT/apps/payload-app-ios/payload-app-ios.xcodeproj" \
  payload-app-ios \
  "$ROOT/apps/payload-app-ios" || failed=1

# Nested Xcode projects (no root project.yml)
build_one driver \
  "$ROOT/apps/driver-app-ios/driverappios/driverappios.xcodeproj" \
  driverappios \
  "" || failed=1

build_one retailer \
  "$ROOT/apps/retailer-app-ios/retailerapp/retailerapp.xcodeproj" \
  retailerapp \
  "" || failed=1

if [[ "$failed" -ne 0 ]]; then
  echo "iOS CI failed for one or more apps"
  exit 1
fi
echo "iOS CI: all 6 apps built"
