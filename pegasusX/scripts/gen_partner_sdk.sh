#!/usr/bin/env bash
# Generate partner API SDKs from contracts/partner.openapi.yaml.
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
SPEC="$ROOT_DIR/contracts/partner.openapi.yaml"
LANG="${1:-ts}"

if [[ ! -f "$SPEC" ]]; then
  echo "missing $SPEC" >&2
  exit 1
fi

case "$LANG" in
  ts|typescript)
    OUT="$ROOT_DIR/sdk/partner/typescript"
    GEN=typescript-fetch
    ;;
  go|golang)
    OUT="$ROOT_DIR/sdk/partner/go"
    GEN=go
    ;;
  *)
    echo "usage: $0 [ts|go]" >&2
    exit 1
    ;;
esac

mkdir -p "$OUT"
docker run --rm -v "$ROOT_DIR:/local" openapitools/openapi-generator-cli:v7.10.0 generate \
  -i /local/contracts/partner.openapi.yaml \
  -g "$GEN" \
  -o "/local/sdk/partner/${LANG}" \
  --additional-properties=supportsES6=true,withInterfaces=true,packageName=partnerclient

echo "generated $OUT"
echo "partner-sdk-gen-ok"
