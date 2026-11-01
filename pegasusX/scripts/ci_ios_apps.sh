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
  if ! xcodebuild \
    -project "$project" \
    -scheme "$scheme" \
    -sdk "$SDK" \
    -destination "$DEST" \
    -configuration Debug \
    CODE_SIGNING_ALLOWED=NO \
    CODE_SIGNING_REQUIRED=NO \
    build; then
    echo "FAIL: $name"
    return 1
  fi
  echo "OK: $name"
}

# unit_test is the XCTest bundle. Empty skips (payload has no test target).
# UI test bundles are not included.
test_one() {
  local name="$1"
  local project="$2"
  local scheme="$3"
  local unit_test="$4"
  if [[ -z "$unit_test" ]]; then
    echo "SKIP unit test: $name"
    return 0
  fi
  local sim_id
  sim_id="$(xcrun simctl list devices available | awk -F '[()]' '/iPhone/ && $0 !~ /unavailable/ { print $2; exit }')"
  if [[ -z "$sim_id" ]]; then
    echo "FAIL: no available iPhone simulator for $name"
    return 1
  fi
  echo "==== iOS unit test: $name ($unit_test) ===="
  if ! xcodebuild test \
    -project "$project" \
    -scheme "$scheme" \
    -destination "platform=iOS Simulator,id=${sim_id}" \
    -only-testing:"$unit_test" \
    -configuration Debug \
    CODE_SIGNING_ALLOWED=NO \
    CODE_SIGNING_REQUIRED=NO; then
    echo "FAIL test: $name"
    return 1
  fi
  echo "OK test: $name"
}

run_one() {
  local name="$1"
  local project="$2"
  local scheme="$3"
  local gen_dir="$4"
  local unit_test="$5"
  if ! build_one "$name" "$project" "$scheme" "$gen_dir"; then
    failed=1
    return
  fi
  if ! test_one "$name" "$project" "$scheme" "$unit_test"; then
    failed=1
  fi
}

failed=0

run_one supplier \
  "$ROOT/apps/supplier-app-ios/SupplierAppIOS.xcodeproj" \
  SupplierAppIOS \
  "$ROOT/apps/supplier-app-ios" \
  SupplierAppTests

run_one warehouse \
  "$ROOT/apps/warehouse-app-ios/WarehouseAppIOS.xcodeproj" \
  WarehouseAppIOS \
  "$ROOT/apps/warehouse-app-ios" \
  WarehouseAppTests

run_one factory \
  "$ROOT/apps/factory-app-ios/FactoryAppIOS.xcodeproj" \
  FactoryAppIOS \
  "$ROOT/apps/factory-app-ios" \
  FactoryAppTests

run_one payload \
  "$ROOT/apps/payload-app-ios/payload-app-ios.xcodeproj" \
  payload-app-ios \
  "$ROOT/apps/payload-app-ios" \
  ""

# Nested Xcode projects (no root project.yml)
run_one driver \
  "$ROOT/apps/driver-app-ios/driverappios/driverappios.xcodeproj" \
  driverappios \
  "" \
  driverappiosTests

run_one retailer \
  "$ROOT/apps/retailer-app-ios/retailerapp/retailerapp.xcodeproj" \
  retailerapp \
  "" \
  retailerappTests

if [[ "$failed" -ne 0 ]]; then
  echo "iOS CI failed for one or more apps"
  exit 1
fi
echo "iOS CI: all 6 apps built; unit tests ran where a test target exists"
