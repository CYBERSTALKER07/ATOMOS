#!/usr/bin/env bash
# CI gate for macOS Native Pegasus Control Tower
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
APP_DIR="$ROOT/apps/control-tower-macos"

echo "=== Pegasus Control Tower: macOS Native CI Gate ==="
cd "$APP_DIR"

if ! command -v xcodegen >/dev/null 2>&1; then
  echo "Error: xcodegen is required" >&2
  exit 1
fi

echo "1. Generating Xcode project from project.yml..."
xcodegen generate

echo "2. Building Pegasus Control Tower native application..."
xcodebuild -project PegasusControlTower.xcodeproj \
  -target PegasusControlTower \
  build CODE_SIGNING_ALLOWED=NO

echo "3. Building & running unit tests..."
xcodebuild -project PegasusControlTower.xcodeproj \
  -target PegasusControlTowerTests \
  build CODE_SIGNING_ALLOWED=NO

xcrun xctest build/Debug/PegasusControlTowerTests.xctest

echo "macos-control-tower-ok"
