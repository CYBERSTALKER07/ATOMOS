#!/usr/bin/env bash
# PX-LC-2 / PX-PROD-3: daily planning export + validate (local SSMR; no GCP billing).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/ssmr_stack.sh
source "$SCRIPT_DIR/lib/ssmr_stack.sh"

usage() {
	cat <<'EOF'
Usage: planning_export_local_cron.sh [--skip-seed] [--skip-stack]

Runs planning-training-export against the local SSMR Spanner emulator, validates
the output, and appends a line to artifacts/planning-export-audit.log.

Environment:
  PLANNING_EXPORT_DAYS          lookback window (default 30)
  PLANNING_EXPORT_MIN_ROWS      min rows for export cmd (default 1)
  PLANNING_EXPORT_SEED_E2E      1 = run ssmr-smokecheck e2e when stack was cold (default 1)
  SSMR_ENV_FILE                 env file path (default .env.ssmr.example)

Gate only:
  bash scripts/planning_export_audit_gate.sh [days]
EOF
}

SKIP_SEED=0
SKIP_STACK=0
for arg in "$@"; do
	case "$arg" in
		--skip-seed) SKIP_SEED=1 ;;
		--skip-stack) SKIP_STACK=1 ;;
		-h|--help) usage; exit 0 ;;
		*) echo "unknown arg: $arg" >&2; usage; exit 1 ;;
	esac
done

ssmr_lib_init
mkdir -p "$SSMR_ARTIFACTS_DIR"

AUDIT_LOG="$SSMR_ARTIFACTS_DIR/planning-export-audit.log"
STAMP="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
DAY_STAMP="$(date -u +%Y-%m-%d)"
OUT_FILE="$SSMR_ARTIFACTS_DIR/planning-export-${DAY_STAMP}.jsonl"
MIN_ROWS="${PLANNING_EXPORT_MIN_ROWS:-1}"

record_audit() {
	local status=$1
	local detail=$2
	printf '%s %s file=%s %s\n' "$STAMP" "$status" "$OUT_FILE" "$detail" >>"$AUDIT_LOG"
}

fail_audit() {
	record_audit "FAIL" "$1"
	echo "planning-export-local-cron-FAIL: $1" >&2
	exit 1
}

if [[ "$SKIP_STACK" != "1" ]]; then
	WAS_HEALTHY=0
	if ssmr_stack_healthy; then
		WAS_HEALTHY=1
	fi
	ssmr_ensure_stack
	if [[ "$SKIP_SEED" != "1" && "$WAS_HEALTHY" != "1" && "${PLANNING_EXPORT_SEED_E2E:-1}" == "1" ]]; then
		ssmr_seed_planning_e2e || fail_audit "e2e seed failed"
	fi
fi

ssmr_load_env_file "$SSMR_ENV_FILE"

if [[ "$SKIP_SEED" != "1" ]]; then
	ssmr_seed_planning_baseline_min || fail_audit "planning baseline seed failed"
fi

ssmr_log "Planning training export → $OUT_FILE"
(
	cd "$SSMR_REPO_ROOT/apps/backend-go"
	TMPDIR="$SSMR_GO_TMP_ROOT" \
	GOCACHE="$SSMR_GO_TMP_ROOT/go-build" \
	GOFLAGS="${GOFLAGS:-} -buildvcs=false" \
	go run ./cmd/planning-training-export \
		-days "${PLANNING_EXPORT_DAYS:-30}" \
		-format "${PLANNING_EXPORT_FORMAT:-jsonl}" \
		-min-rows "$MIN_ROWS" \
		-out "$OUT_FILE"
) || fail_audit "export command failed"

ssmr_log "Validating export"
if ! VALIDATOR_OUT="$(bash "$SSMR_REPO_ROOT/scripts/planning_export_validate.sh" "$OUT_FILE" "$MIN_ROWS")"; then
	fail_audit "validator failed"
fi
echo "$VALIDATOR_OUT"
record_audit "OK" "$VALIDATOR_OUT"
echo "planning-export-local-cron-ok — $VALIDATOR_OUT"
echo "audit log: $AUDIT_LOG"
