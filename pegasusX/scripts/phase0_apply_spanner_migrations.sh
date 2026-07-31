#!/usr/bin/env bash
# PX-PROD-0: apply all Spanner migrations through 20260702 on cloud Spanner.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MIG_DIR="$ROOT/apps/backend-go/schema/migrations"
# Through fiscal hard-gate receipts (D3 / ADR-009). Keep newest required migration here.
LAST_MIGRATION="20260804_phase_d_notifications.ddl"

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

# Allow operators to target SSMR Spanner without editing generated env files.
if [[ -n "${PHASE0_SPANNER_PROJECT:-}" ]]; then
	export SPANNER_PROJECT="$PHASE0_SPANNER_PROJECT"
fi
if [[ -n "${PHASE0_SPANNER_INSTANCE:-}" ]]; then
	export SPANNER_INSTANCE="$PHASE0_SPANNER_INSTANCE"
fi
if [[ -n "${PHASE0_SPANNER_DATABASE:-}" ]]; then
	export SPANNER_DATABASE="$PHASE0_SPANNER_DATABASE"
fi

# Force cloud Spanner client (LoadConfig would otherwise default emulator for local projects).
# Explicit empty host + non-local SPANNER_PROJECT both select real GCP ADC.
if [[ -n "${SPANNER_EMULATOR_HOST:-}" ]]; then
	echo "WARN: unsetting SPANNER_EMULATOR_HOST=${SPANNER_EMULATOR_HOST} for cloud migrate" >&2
fi
unset SPANNER_EMULATOR_HOST
export SPANNER_EMULATOR_HOST=""

require_env() {
	if [[ -z "${!1:-}" ]]; then
		echo "FAIL: $1 not set (run make render-k8s-from-terraform after terraform apply)" >&2
		exit 1
	fi
}

require_env SPANNER_PROJECT
require_env SPANNER_INSTANCE
require_env SPANNER_DATABASE

if [[ "${SPANNER_PROJECT}" == "pegasusx-local" ]]; then
	echo "FAIL: SPANNER_PROJECT=pegasusx-local looks like emulator — set cloud project id" >&2
	exit 1
fi

echo "==> Spanner (cloud): projects/${SPANNER_PROJECT}/instances/${SPANNER_INSTANCE}/databases/${SPANNER_DATABASE}"
echo "==> SPANNER_EMULATOR_HOST empty (real GCP)"

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
