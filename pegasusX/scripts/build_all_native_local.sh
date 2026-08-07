#!/usr/bin/env bash
# Local build orchestrator for PegasusX native clients (macOS host).
# Desktop = Tauri macOS; Android = enterpriseDebug; iOS = simulator build.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

DESKTOP_APPS=(
  retailer-app-desktop
  supplier-portal
  warehouse-portal
  factory-portal
)
ANDROID_APPS=(
  retailer-app-android
  supplier-app-android
  driver-app-android
  warehouse-app-android
  factory-app-android
  payload-app-android
)

SKIP_DESKTOP="${SKIP_DESKTOP:-0}"
SKIP_ANDROID="${SKIP_ANDROID:-0}"
SKIP_IOS="${SKIP_IOS:-0}"
IOS_DEST="${IOS_DEST:-platform=iOS Simulator,name=iPhone 17,OS=26.5}"

fail=0

if [[ "$SKIP_DESKTOP" != "1" ]]; then
  echo "=== Desktop (static + Tauri host target) ==="
  if [[ -z "${TAURI_UPDATER_PUBKEY:-}" && -z "${TAURI_UPDATER_PUBKEY_PATH:-}" ]]; then
    export ALLOW_DEV_UPDATER_PUBKEY="${ALLOW_DEV_UPDATER_PUBKEY:-1}"
  fi
  bash scripts/apply_desktop_updater_pubkey.sh
  pnpm install --frozen-lockfile 2>/dev/null || pnpm install
  for app in "${DESKTOP_APPS[@]}"; do
    echo "--- $app ---"
    if ! pnpm --filter "@pegasusx/$app" run build:static; then
      echo "FAIL desktop static: $app" >&2
      fail=1
      continue
    fi
    if ! pnpm --filter "@pegasusx/$app" exec tauri build; then
      echo "FAIL desktop tauri: $app" >&2
      fail=1
    fi
  done
fi

if [[ "$SKIP_ANDROID" != "1" ]]; then
  echo "=== Android (assembleEnterpriseDebug) ==="
  for app in "${ANDROID_APPS[@]}"; do
    echo "--- $app ---"
    if ! (cd "apps/$app" && ./gradlew :app:assembleEnterpriseDebug --stacktrace); then
      echo "FAIL android: $app" >&2
      fail=1
    fi
  done
fi

if [[ "$SKIP_IOS" != "1" ]]; then
  echo "=== iOS (simulator) ==="
  declare -a IOS_BUILDS=(
    "apps/retailer-app-ios/retailerapp/reatilerapp.xcodeproj|reatilerapp"
    "apps/supplier-app-ios/SupplierAppIOS.xcodeproj|SupplierAppIOS"
    "apps/driver-app-ios/driverappios/driverappios.xcodeproj|driverappios"
    "apps/warehouse-app-ios/WarehouseAppIOS.xcodeproj|WarehouseAppIOS"
    "apps/factory-app-ios/FactoryAppIOS.xcodeproj|FactoryAppIOS"
    "apps/payload-app-ios/payload-app-ios.xcodeproj|payload-app-ios"
  )
  for entry in "${IOS_BUILDS[@]}"; do
    proj="${entry%%|*}"
    scheme="${entry##*|}"
    echo "--- $scheme ---"
    if ! xcodebuild -project "$ROOT/$proj" -scheme "$scheme" -destination "$IOS_DEST" -quiet build; then
      echo "FAIL ios: $scheme" >&2
      fail=1
    fi
  done
fi

if [[ "$fail" -ne 0 ]]; then
  echo "native-local-builds: FAILED" >&2
  exit 1
fi
echo "native-local-builds-ok"
