#!/usr/bin/env bash
# PX-PROD-3: export DemandForecastBaseline rows for offline ML training (collect-only).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/ssmr_stack.sh
source "$SCRIPT_DIR/lib/ssmr_stack.sh"

ssmr_lib_init

OUT="${1:-${PLANNING_EXPORT_OUT:-$SSMR_ARTIFACTS_DIR/planning-export.jsonl}}"
mkdir -p "$(dirname "$OUT")"

if [[ "${PLANNING_EXPORT_SKIP_SEED:-}" != "1" ]]; then
	ssmr_seed_planning_baseline_min
fi

ARGS=(./cmd/planning-training-export -days "${PLANNING_EXPORT_DAYS:-30}" -format "${PLANNING_EXPORT_FORMAT:-jsonl}")
if [[ -n "${PLANNING_EXPORT_SUPPLIER_ID:-}" ]]; then
	ARGS+=(-supplier-id "$PLANNING_EXPORT_SUPPLIER_ID")
fi
if [[ -n "${PLANNING_EXPORT_MIN_ROWS:-}" ]]; then
	ARGS+=(-min-rows "$PLANNING_EXPORT_MIN_ROWS")
fi
ARGS+=(-out "$OUT")

(
	cd "$SSMR_REPO_ROOT/apps/backend-go"
	TMPDIR="$SSMR_GO_TMP_ROOT" \
	GOCACHE="$SSMR_GO_TMP_ROOT/go-build" \
	GOFLAGS="${GOFLAGS:-} -buildvcs=false" \
	go run "${ARGS[@]}"
)

echo "planning-training-export-ok — $OUT"
