#!/usr/bin/env bash
# GS-C3: apply the SAME Spanner DDL (migrations) to a cell. Never restore a UZ backup.
# Catalog default is dry-run. Set CELL_MIGRATE_APPLY=1 only when the EU project exists.
set -euo pipefail

CELL="${1:-}"
if [[ "$CELL" != "eu" ]]; then
  echo "usage: make cell-migrate CELL=eu" >&2
  exit 1
fi

if [[ "${*}" == *restore* || "${CELL_MIGRATE_RESTORE:-}" == "1" ]]; then
  echo "FAIL: GS-C3 forbids Spanner backup restore into a non-uz cell. Use migrations." >&2
  exit 1
fi

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PROJECT="pegasusx-cell-eu"
INSTANCE="pegasusx-eu-spanner"
DATABASE="pegasusx-eu-db"

if [[ "$PROJECT" == "pegasus-503013" ]]; then
  echo "FAIL: EU migrate must not target pegasus-503013" >&2
  exit 1
fi

echo "GS-C3 cell-migrate CELL=eu (DDL only, no UZ restore)"
echo "  SPANNER_PROJECT=$PROJECT"
echo "  SPANNER_INSTANCE=$INSTANCE"
echo "  SPANNER_DATABASE=$DATABASE"

if [[ "${CELL_MIGRATE_APPLY:-}" != "1" ]]; then
  echo "dry-run: set CELL_MIGRATE_APPLY=1 to run scripts/phase0_apply_spanner_migrations.sh against the EU project (C3 catalog does not)."
  exit 0
fi

export SPANNER_PROJECT="$PROJECT"
export SPANNER_INSTANCE="$INSTANCE"
export SPANNER_DATABASE="$DATABASE"
export PHASE0_SPANNER_PROJECT="$PROJECT"
export PHASE0_SPANNER_INSTANCE="$INSTANCE"
export PHASE0_SPANNER_DATABASE="$DATABASE"
exec bash "$ROOT/scripts/phase0_apply_spanner_migrations.sh"
