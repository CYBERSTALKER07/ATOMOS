#!/usr/bin/env bash
# Upload website-only enterprise packages for a native role app.
#
# Usage:
#   bash scripts/upload_enterprise_mobile_app.sh driver android \
#     --apk dist/driver.apk --version-code 12 --version-name 1.2.0
#
#   bash scripts/upload_enterprise_mobile_app.sh driver ios \
#     --ipa dist/driver.ipa --plist dist/manifest.plist \
#     --version-code 12 --version-name 1.2.0
#
# Roles: supplier | driver | retailer | warehouse | factory | payload
set -euo pipefail

ROLE="${1:-}"
PLATFORM="${2:-}"
shift 2 || true

BUCKET="${UPDATES_GCS_BUCKET:-pegasusx-ssmr-app-updates}"
BASE_URL="${UPDATES_BASE_URL:-https://storage.googleapis.com/${BUCKET}}"

APK=""
IPA=""
PLIST=""
VERSION_CODE=""
VERSION_NAME=""
NOTES="Enterprise ${ROLE} website build"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --apk) APK="$2"; shift 2 ;;
    --ipa) IPA="$2"; shift 2 ;;
    --plist) PLIST="$2"; shift 2 ;;
    --version-code) VERSION_CODE="$2"; shift 2 ;;
    --version-name) VERSION_NAME="$2"; shift 2 ;;
    --notes) NOTES="$2"; shift 2 ;;
    --bucket) BUCKET="$2"; BASE_URL="https://storage.googleapis.com/${BUCKET}"; shift 2 ;;
    *) echo "unknown arg: $1" >&2; exit 1 ;;
  esac
done

case "$ROLE" in
  supplier|driver|retailer|warehouse|factory|payload) ;;
  *) echo "role must be supplier|driver|retailer|warehouse|factory|payload" >&2; exit 1 ;;
esac

if [[ -z "$PLATFORM" || -z "$VERSION_CODE" || -z "$VERSION_NAME" ]]; then
  echo "usage: $0 <role> android|ios --version-code N --version-name X.Y.Z [--apk path | --ipa path --plist path]" >&2
  exit 1
fi

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

json_escape() {
  python3 -c 'import json,sys; print(json.dumps(sys.argv[1]))' "$1"
}

case "$PLATFORM" in
  android)
    [[ -n "$APK" && -f "$APK" ]] || { echo "--apk required" >&2; exit 1; }
    HASH="$(sha256_file "$APK")"
    DEST="android/${ROLE}/${ROLE}-${VERSION_NAME}.apk"
    gsutil cp "$APK" "gs://${BUCKET}/${DEST}"
    gsutil acl ch -u AllUsers:R "gs://${BUCKET}/${DEST}" 2>/dev/null || true
    APK_URL="${BASE_URL}/${DEST}"
    MANIFEST_JSON="$(mktemp)"
    cat >"$MANIFEST_JSON" <<EOF
{
  "version_code": ${VERSION_CODE},
  "version_name": "${VERSION_NAME}",
  "apk_url": "${APK_URL}",
  "sha256": "${HASH}",
  "notes": $(json_escape "$NOTES")
}
EOF
    gsutil cp "$MANIFEST_JSON" "gs://${BUCKET}/android/${ROLE}/updater.json"
    gsutil acl ch -u AllUsers:R "gs://${BUCKET}/android/${ROLE}/updater.json" 2>/dev/null || true
    rm -f "$MANIFEST_JSON"
    echo "Uploaded Android enterprise update for ${ROLE}:"
    echo "  ${BASE_URL}/android/${ROLE}/updater.json"
    echo "  apk: ${APK_URL}"
    echo "  sha256: ${HASH}"
    POLICY_ROLE="DRIVER"
    [[ "$ROLE" == "supplier" ]] && POLICY_ROLE="ADMIN"
    [[ "$ROLE" == "retailer" ]] && POLICY_ROLE="RETAILER"
    [[ "$ROLE" == "warehouse" ]] && POLICY_ROLE="WAREHOUSE"
    [[ "$ROLE" == "factory" ]] && POLICY_ROLE="FACTORY"
    [[ "$ROLE" == "payload" ]] && POLICY_ROLE="PAYLOAD"
    echo "  policy role: ${POLICY_ROLE} / android / enterprise"
    ;;
  ios)
    [[ -n "$IPA" && -f "$IPA" ]] || { echo "--ipa required" >&2; exit 1; }
    [[ -n "$PLIST" && -f "$PLIST" ]] || { echo "--plist required" >&2; exit 1; }
    IPA_DEST="ios/${ROLE}/${ROLE}-${VERSION_NAME}.ipa"
    PLIST_DEST="ios/${ROLE}/manifest.plist"
    gsutil cp "$IPA" "gs://${BUCKET}/${IPA_DEST}"
    gsutil cp "$PLIST" "gs://${BUCKET}/${PLIST_DEST}"
    gsutil acl ch -u AllUsers:R "gs://${BUCKET}/${IPA_DEST}" "gs://${BUCKET}/${PLIST_DEST}" 2>/dev/null || true
    PLIST_URL="${BASE_URL}/${PLIST_DEST}"
    MANIFEST_JSON="$(mktemp)"
    cat >"$MANIFEST_JSON" <<EOF
{
  "version_code": ${VERSION_CODE},
  "version_name": "${VERSION_NAME}",
  "manifest_url": "${PLIST_URL}",
  "notes": $(json_escape "$NOTES")
}
EOF
    gsutil cp "$MANIFEST_JSON" "gs://${BUCKET}/ios/${ROLE}/updater.json"
    gsutil acl ch -u AllUsers:R "gs://${BUCKET}/ios/${ROLE}/updater.json" 2>/dev/null || true
    rm -f "$MANIFEST_JSON"
    echo "Uploaded iOS enterprise update for ${ROLE}:"
    echo "  ${BASE_URL}/ios/${ROLE}/updater.json"
    echo "  plist: ${PLIST_URL}"
    ;;
  *)
    echo "platform must be android or ios" >&2
    exit 1
    ;;
esac
