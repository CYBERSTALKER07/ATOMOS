#!/usr/bin/env bash
# Apply an incremental Spanner migration before backend deploy.
# Uses SPANNER_* env vars (see pegasusX/.env.example). For GCP, ensure
# SPANNER_EMULATOR_HOST is unset and gcloud is authenticated with
# spanner.databases.updateDdl on the target database.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DDL_FILE="${DDL_FILE:-$ROOT/apps/backend-go/schema/migrations/20250616_warehouse_stock_policy_supply_items.ddl}"

if [[ -f "$ROOT/apps/backend-go/.env.local" ]]; then
	set -a
	# shellcheck disable=SC1091
	source "$ROOT/apps/backend-go/.env.local"
	set +a
elif [[ -f "$ROOT/.env.ssmr" ]]; then
	set -a
	# shellcheck disable=SC1091
	source "$ROOT/.env.ssmr"
	set +a
fi

export SPANNER_PROJECT="${SPANNER_PROJECT:-pegasusx-local}"
export SPANNER_INSTANCE="${SPANNER_INSTANCE:-pegasusx-instance}"
export SPANNER_DATABASE="${SPANNER_DATABASE:-pegasusx-db}"

echo "==> Spanner target: projects/${SPANNER_PROJECT}/instances/${SPANNER_INSTANCE}/databases/${SPANNER_DATABASE}"
if [[ -n "${SPANNER_EMULATOR_HOST:-}" ]]; then
	echo "==> Emulator: ${SPANNER_EMULATOR_HOST}"
else
	echo "==> GCP (no SPANNER_EMULATOR_HOST)"
fi
echo "==> DDL: ${DDL_FILE}"

cd "$ROOT/apps/backend-go"
go run ./cmd/setup
go run ./cmd/apply-migration --ddl "$DDL_FILE" --verify

echo "==> Migration complete"
