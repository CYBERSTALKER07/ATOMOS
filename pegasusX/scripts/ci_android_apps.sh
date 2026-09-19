#!/usr/bin/env bash
# Compile all six Android role apps (storeDebug) for Gate-0 CI.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

APPS=(
  driver-app-android
  factory-app-android
  payload-app-android
  retailer-app-android
  supplier-app-android
  warehouse-app-android
)

# Prefer storeDebug (Play-shaped); fall back to enterpriseDebug if store flavor missing.
TASK="${ANDROID_CI_TASK:-compileStoreDebugKotlin}"
FALLBACK_TASK="${ANDROID_CI_FALLBACK_TASK:-compileEnterpriseDebugKotlin}"

failed=0
for app in "${APPS[@]}"; do
  dir="$ROOT/apps/$app"
  if [[ ! -x "$dir/gradlew" ]]; then
    echo "FAIL: missing gradlew in $app"
    failed=1
    continue
  fi
  echo "==== Android CI: $app ($TASK) ===="
  if (cd "$dir" && ./gradlew --no-daemon --stacktrace "$TASK"); then
    echo "OK: $app"
  else
    echo "WARN: $TASK failed for $app — trying $FALLBACK_TASK"
    if (cd "$dir" && ./gradlew --no-daemon --stacktrace "$FALLBACK_TASK"); then
      echo "OK: $app ($FALLBACK_TASK)"
    else
      echo "FAIL: $app"
      failed=1
    fi
  fi
done

if [[ "$failed" -ne 0 ]]; then
  echo "Android CI failed for one or more apps"
  exit 1
fi
echo "Android CI: all 6 apps compiled"
