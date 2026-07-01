#!/usr/bin/env bash
# PX-PROD-0: apply all Spanner migrations through 20260702 on cloud Spanner.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MIG_DIR="$ROOT/apps/backend-go/schema/migrations"
LAST_MIGRATION="20260702_supplier_promotions_redemption_caps.ddl"

if [[ -f "$ROOT/.env.k8s.generated" ]]; then
	set -a
	# shellcheck disable=SC1091
	source "$ROOT/.env.k8s.generated"
	set +a
elif [[ -f "$ROOT/.env.k8s" ]]; then
	set -a
	# shellcheck disable=SC1091
	source "$ROOT/.env.k8s"
	set +a
fi

if [[ -n "${SPANNER_EMULATOR_HOST:-}" ]]; then
	echo "FAIL: SPANNER_EMULATOR_HOST is set — Phase 0 targets cloud Spanner" >&2
	exit 1
fi

require_env() {
	if [[ -z "${!1:-}" ]]; then
		echo "FAIL: $1 not set (run make render-k8s-from-terraform after terraform apply)" >&2
		exit 1
	fi
}

require_env SPANNER_PROJECT
require_env SPANNER_INSTANCE
require_env SPANNER_DATABASE

echo "==> Spanner: projects/${SPANNER_PROJECT}/instances/${SPANNER_INSTANCE}/databases/${SPANNER_DATABASE}"

cd "$ROOT/apps/backend-go"
echo "==> Base schema convergence (cmd/setup)"
go run ./cmd/setup

applied=0
last_seen=0
for ddl in $(find "$MIG_DIR" -name '*.ddl' | sort); do
	base="$(basename "$ddl")"
	echo "==> Applying $base"
	go run ./cmd/apply-migration --ddl "$ddl" --verify
	applied=$((applied + 1))
	if [[ "$base" == "$LAST_MIGRATION" ]]; then
		last_seen=1
	fi
done

if [[ "$last_seen" -ne 1 ]]; then
	echo "FAIL: migration $LAST_MIGRATION was not applied" >&2
	exit 1
fi

echo "phase0-migrate-ok migrations_applied=${applied} through=${LAST_MIGRATION}"
exit 0
