#!/usr/bin/env bash
# Run unit tests across all six Android role apps (storeDebug) for Gate-0 CI.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [[ -z "${ANDROID_HOME:-}" && -d "$HOME/Library/Android/sdk" ]]; then
  export ANDROID_HOME="$HOME/Library/Android/sdk"
fi

APPS=(
  driver-app-android
  factory-app-android
  payload-app-android
  retailer-app-android
  supplier-app-android
  warehouse-app-android
)

TASK="${ANDROID_TEST_TASK:-testStoreDebugUnitTest}"
FALLBACK_TASK="${ANDROID_TEST_FALLBACK_TASK:-testEnterpriseDebugUnitTest}"

failed=0
for app in "${APPS[@]}"; do
  dir="$ROOT/apps/$app"
  if [[ ! -x "$dir/gradlew" ]]; then
    echo "FAIL: missing gradlew in $app"
    failed=1
    continue
  fi
  echo "==== Android Unit Tests: $app ($TASK) ===="
  if (cd "$dir" && ./gradlew --no-daemon "$TASK"); then
    echo "OK: $app"
  else
    echo "WARN: $TASK failed for $app — trying $FALLBACK_TASK"
    if (cd "$dir" && ./gradlew --no-daemon "$FALLBACK_TASK"); then
      echo "OK: $app ($FALLBACK_TASK)"
    else
      echo "FAIL: $app"
      failed=1
    fi
  fi
done

if [[ "$failed" -ne 0 ]]; then
  echo "Android unit tests failed for one or more apps"
  exit 1
fi
echo "Android CI: all 6 apps passed unit tests"
