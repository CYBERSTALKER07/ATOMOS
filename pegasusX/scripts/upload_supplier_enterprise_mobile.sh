#!/usr/bin/env bash
# Compatibility wrapper — prefer upload_enterprise_mobile_app.sh
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLATFORM="${1:-}"
shift || true
exec bash "$ROOT/upload_enterprise_mobile_app.sh" supplier "$PLATFORM" "$@"
