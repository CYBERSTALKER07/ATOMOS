#!/usr/bin/env bash
# SSMR-equivalent smoke against a deployed staging API (set PUBLIC_BASE_URL).
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
export PUBLIC_BASE_URL="${PUBLIC_BASE_URL:?set PUBLIC_BASE_URL to staging origin}"
export SSMR_ENV_FILE="${SSMR_ENV_FILE:-$REPO_ROOT/.env.ssmr.example}"
cd "$REPO_ROOT/apps/backend-go"
go run ./cmd/ssmr-smokecheck e2e
echo "__STAGING_SMOKE_OK__"
