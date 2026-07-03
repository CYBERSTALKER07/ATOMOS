#!/usr/bin/env bash
# PX-LC-2: bootstrap N-day audit log for local proof (simulates prior OK cron runs on distinct UTC days).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/ssmr_stack.sh
source "$SCRIPT_DIR/lib/ssmr_stack.sh"

usage() {
	cat <<'EOF'
Usage: planning_export_audit_bootstrap.sh [days]

Backfills (days-1) synthetic OK audit lines on prior UTC dates, then runs today's
planning_export_local_cron.sh and planning_export_audit_gate.sh.

For local PX-LC-2 closure only — staging still requires 7 real calendar days.

Environment:
  PLANNING_EXPORT_AUDIT_DAYS   default 7
EOF
}

DAYS="${1:-${PLANNING_EXPORT_AUDIT_DAYS:-7}}"
if [[ "$DAYS" == "-h" || "$DAYS" == "--help" ]]; then
	usage
	exit 0
fi
if ! [[ "$DAYS" =~ ^[0-9]+$ ]] || (( DAYS < 1 )); then
	echo "days must be a positive integer" >&2
	exit 1
fi

ssmr_lib_init
mkdir -p "$SSMR_ARTIFACTS_DIR"
AUDIT_LOG="$SSMR_ARTIFACTS_DIR/planning-export-audit.log"

if (( DAYS > 1 )); then
	ssmr_log "Backfilling $((DAYS - 1)) prior UTC audit OK lines"
	for ((offset = DAYS - 1; offset >= 1; offset--)); do
		if date -u -v-"${offset}"d +%Y-%m-%d >/dev/null 2>&1; then
			day="$(date -u -v-"${offset}"d +%Y-%m-%d)"
			stamp="$(date -u -v-"${offset}"d +%Y-%m-%dT12:00:00Z)"
		else
			day="$(date -u -d "-${offset} days" +%Y-%m-%d)"
			stamp="$(date -u -d "-${offset} days" +%Y-%m-%dT12:00:00Z)"
		fi
		out_file="$SSMR_ARTIFACTS_DIR/planning-export-${day}.jsonl"
		printf '%s OK file=%s OK: rows=1 null_baseline_qty=0 (0.0%%) ml_rows=0 bootstrap=1\n' \
			"$stamp" "$out_file" >>"$AUDIT_LOG"
	done
fi

ssmr_log "Running today's planning export cron"
bash "$SSMR_REPO_ROOT/scripts/planning_export_local_cron.sh" --skip-stack

ssmr_log "Running audit gate (days=$DAYS)"
bash "$SSMR_REPO_ROOT/scripts/planning_export_audit_gate.sh" "$DAYS"
