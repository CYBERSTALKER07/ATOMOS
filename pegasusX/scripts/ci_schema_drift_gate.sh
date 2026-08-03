#!/usr/bin/env bash
# EH0.3 — CI schema-drift gate (Spanner emulator).
# Applies base schema + shop-closed migration, then fails if columns/tables absent.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

export SPANNER_EMULATOR_HOST="${SPANNER_EMULATOR_HOST:-localhost:9010}"
export SPANNER_PROJECT="${SPANNER_PROJECT:-pegasusx-local}"
export SPANNER_INSTANCE="${SPANNER_INSTANCE:-pegasusx-instance}"
export SPANNER_DATABASE="${SPANNER_DATABASE:-pegasusx-db}"
export JWT_SECRET="${JWT_SECRET:-ci-schema-drift-secret}"

DDL="$ROOT_DIR/apps/backend-go/schema/migrations/20260729_shop_closed_proximity_partial.ddl"

echo "Waiting for Spanner emulator at ${SPANNER_EMULATOR_HOST} ..."
for i in $(seq 1 60); do
	if (echo >/dev/tcp/${SPANNER_EMULATOR_HOST%:*}/${SPANNER_EMULATOR_HOST#*:}) 2>/dev/null; then
		break
	fi
	if [[ "$i" -eq 60 ]]; then
		echo "Spanner emulator not reachable" >&2
		exit 1
	fi
	sleep 1
done

echo "Applying Spanner schema via cmd/setup ..."
(cd apps/backend-go && go run ./cmd/setup)

echo "Applying shop-closed migration (idempotent) ..."
(cd apps/backend-go && go run ./cmd/apply-migration --ddl "$DDL")

echo "Running schema-drift gate ..."
(cd apps/backend-go && go run ./cmd/schema-drift)

echo "schema-drift-gate-ok"
