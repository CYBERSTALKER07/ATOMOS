#!/usr/bin/env bash
# EH0.2 — apply shop-closed / proximity / partial DDL and verify schema drift gate.
#
# Defaults target live SSMR Spanner. Override SPANNER_* for staging/emulator.
#
# Usage:
#   bash scripts/apply_shop_closed_ddl.sh
#   SPANNER_INSTANCE=pegasusx-staging-spanner SPANNER_DATABASE=pegasusx-staging-db \
#     bash scripts/apply_shop_closed_ddl.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DDL_FILE="${DDL_FILE:-$ROOT/apps/backend-go/schema/migrations/20260729_shop_closed_proximity_partial.ddl}"

export SPANNER_PROJECT="${SPANNER_PROJECT:-pegasus-503013}"
export SPANNER_INSTANCE="${SPANNER_INSTANCE:-pegasusx-ssmr-spanner}"
export SPANNER_DATABASE="${SPANNER_DATABASE:-pegasusx-ssmr-db}"

if [[ ! -f "$DDL_FILE" ]]; then
	echo "missing DDL: $DDL_FILE" >&2
	exit 1
fi

# Live GCP: prefer gcloud (more reliable ADC than Go client from some sandboxes).
if [[ -z "${SPANNER_EMULATOR_HOST:-}" ]]; then
	export DDL_FILE
	bash "$ROOT/scripts/apply_shop_closed_ddl_gcloud.sh"
	exit 0
fi

echo "==> target projects/${SPANNER_PROJECT}/instances/${SPANNER_INSTANCE}/databases/${SPANNER_DATABASE}"
echo "==> ddl $DDL_FILE (emulator)"

cd "$ROOT/apps/backend-go"
go run ./cmd/apply-migration --ddl "$DDL_FILE" --verify-shop-closed
go run ./cmd/schema-drift

echo "==> shop-closed DDL applied + schema-drift-ok"
