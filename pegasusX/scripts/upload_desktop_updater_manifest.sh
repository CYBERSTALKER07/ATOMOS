#!/usr/bin/env bash
# Build and upload a Tauri 2 desktop updater.json (+ optional bundle) to GCS.
#
# Usage:
#   ./scripts/upload_desktop_updater_manifest.sh <app> <version> <bundle-path> [os] [arch]
#
# Example (Windows NSIS):
#   ./scripts/upload_desktop_updater_manifest.sh retailer-app-desktop 0.1.1 \
#     apps/retailer-app-desktop/src-tauri/target/x86_64-pc-windows-msvc/release/bundle/nsis/*.exe \
#     windows x86_64
#
# Example (macOS aarch64):
#   ./scripts/upload_desktop_updater_manifest.sh supplier-portal 0.1.1 \
#     path/to/app.app.tar.gz darwin aarch64
#
# Requires: TAURI_SIGNING_PRIVATE_KEY (or TAURI_SIGNING_PRIVATE_KEY_PATH), gsutil, pnpm
#
# CDN layout (Tauri 2 {{target}}/{{arch}}):
#   gs://BUCKET/{slug}/{os}/{arch}/updater.json
#   e.g. supplier-desktop/windows/x86_64/updater.json
set -euo pipefail

APP="${1:-}"
VERSION="${2:-}"
BUNDLE_GLOB="${3:-}"
# Tauri 2 endpoint vars: target ∈ {windows,darwin,linux}, arch ∈ {x86_64,aarch64,...}
OS_TARGET="${4:-windows}"
ARCH="${5:-x86_64}"

if [[ -z "$APP" || -z "$VERSION" || -z "$BUNDLE_GLOB" ]]; then
  echo "Usage: $0 <app> <version> <bundle-glob> [os=windows] [arch=x86_64]" >&2
  exit 1
fi

case "$OS_TARGET" in
  windows|darwin|linux) ;;
  win) OS_TARGET="windows" ;;
  mac|macos|osx) OS_TARGET="darwin" ;;
  *)
    echo "Unknown os target: $OS_TARGET (use windows|darwin|linux)" >&2
    exit 1
    ;;
esac

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

case "$APP" in
  retailer-app-desktop) SLUG="retailer-desktop" ; FILTER="@pegasusx/retailer-app-desktop" ;;
  supplier-portal) SLUG="supplier-desktop" ; FILTER="@pegasusx/supplier-portal" ;;
  warehouse-portal) SLUG="warehouse-desktop" ; FILTER="@pegasusx/warehouse-portal" ;;
  factory-portal) SLUG="factory-desktop" ; FILTER="@pegasusx/factory-portal" ;;
  *)
    echo "Unknown app: $APP" >&2
    exit 1
    ;;
esac

BUCKET="${DESKTOP_UPDATES_GCS_BUCKET:-pegasusx-ssmr-app-updates}"
GCS_PREFIX="gs://${BUCKET}/${SLUG}/${OS_TARGET}/${ARCH}"
PLATFORM_KEY="${OS_TARGET}-${ARCH}"

shopt -s nullglob
BUNDLE_FILES=($BUNDLE_GLOB)
shopt -u nullglob
if [[ ${#BUNDLE_FILES[@]} -eq 0 ]]; then
  echo "No bundle matched: $BUNDLE_GLOB" >&2
  exit 1
fi
BUNDLE="${BUNDLE_FILES[0]}"
BUNDLE_NAME="$(basename "$BUNDLE")"

echo "Signing bundle: $BUNDLE"
pnpm --filter "$FILTER" exec tauri signer sign "$BUNDLE" -f "${TAURI_SIGNING_PRIVATE_KEY_PATH:-$ROOT/contracts/desktop-updater/dev.key}"

SIG_FILE="${BUNDLE}.sig"
if [[ ! -f "$SIG_FILE" ]]; then
  echo "Missing signature file: $SIG_FILE" >&2
  exit 1
fi
SIGNATURE="$(tr -d '\n' <"$SIG_FILE")"

echo "Uploading to $GCS_PREFIX/"
gsutil cp "$BUNDLE" "${GCS_PREFIX}/${BUNDLE_NAME}"
PUB_DATE="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
MANIFEST="$(mktemp)"
cat >"$MANIFEST" <<EOF
{
  "version": "${VERSION}",
  "notes": "Desktop enterprise release ${VERSION}",
  "pub_date": "${PUB_DATE}",
  "platforms": {
    "${PLATFORM_KEY}": {
      "signature": "${SIGNATURE}",
      "url": "https://storage.googleapis.com/${BUCKET}/${SLUG}/${OS_TARGET}/${ARCH}/${BUNDLE_NAME}"
    }
  }
}
EOF

gsutil cp "$MANIFEST" "${GCS_PREFIX}/updater.json"
rm -f "$MANIFEST"
echo "upload-desktop-updater-ok: ${GCS_PREFIX}/updater.json"
echo "  platform key: ${PLATFORM_KEY}"
echo "  endpoint: https://storage.googleapis.com/${BUCKET}/${SLUG}/{{target}}/{{arch}}/updater.json"
