#!/usr/bin/env bash
# Validates that staging credential *names* are present and backend health responds.
# Does not print secret values.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

MISSING=()

require_env() {
	local name=$1
	if [[ -z "${!name:-}" ]]; then
		MISSING+=("$name")
	fi
}

# Core production contract (names only)
require_env JWT_SECRET
require_env INTERNAL_API_KEY
require_env KAFKA_BROKERS

# Payment (at least Global Pay for primary rail)
require_env GLOBAL_PAY_WEBHOOK_SECRET

# Optional but recommended for full staging
OPTIONAL=(
	GLOBAL_PAY_SERVICE_ID
	GLOBAL_PAY_USERNAME
	GLOBAL_PAY_PASSWORD
	PAYME_WEBHOOK_SECRET
	CLICK_WEBHOOK_SECRET
	GOOGLE_MAPS_API_KEY
	FIREBASE_CREDENTIALS_PATH
)

OPTIONAL_MISSING=()
for name in "${OPTIONAL[@]}"; do
	if [[ -z "${!name:-}" ]]; then
		OPTIONAL_MISSING+=("$name")
	fi
done

BASE="${PUBLIC_BASE_URL:-}"
if [[ -z "$BASE" ]]; then
	MISSING+=("PUBLIC_BASE_URL")
else
	if ! curl -fsS --max-time 15 "${BASE%/}/v1/health" >/dev/null 2>&1; then
		echo "FAIL: health check failed for ${BASE}/v1/health" >&2
		exit 1
	fi
fi

if ((${#MISSING[@]} > 0)); then
	echo "staging-credentials-FAIL — missing required env:" >&2
	printf '  - %s\n' "${MISSING[@]}" >&2
	exit 1
fi

if ((${#OPTIONAL_MISSING[@]} > 0)); then
	echo "WARN: optional staging env not set (non-blocking):" >&2
	printf '  - %s\n' "${OPTIONAL_MISSING[@]}" >&2
fi

echo "staging-credentials-ok"
exit 0
