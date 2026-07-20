#!/usr/bin/env bash
# Build all native Android apps as Play Store (production channel) release AABs.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
APPS=(
  supplier-app-android
  driver-app-android
  retailer-app-android
  warehouse-app-android
  factory-app-android
  payload-app-android
)
for app in "${APPS[@]}"; do
  echo "=== $app storeRelease ==="
  (cd "$ROOT/apps/$app" && ./gradlew :app:bundleStoreRelease --stacktrace)
done
echo "native-store-android-ok"
