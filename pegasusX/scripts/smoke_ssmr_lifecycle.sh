#!/usr/bin/env bash
# Deprecated alias — SSMR renamed to sandbox. Prefer scripts/smoke_sandbox_lifecycle.sh.
set -euo pipefail
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/smoke_sandbox_lifecycle.sh" "$@"
