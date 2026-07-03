#!/usr/bin/env bash
# PX-LC-2: verify N consecutive OK days in artifacts/planning-export-audit.log
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
AUDIT_LOG="${PLANNING_EXPORT_AUDIT_LOG:-$ROOT/artifacts/planning-export-audit.log}"
REQUIRED_DAYS="${1:-${PLANNING_EXPORT_AUDIT_DAYS:-7}}"

if [[ ! -f "$AUDIT_LOG" ]]; then
	echo "planning-export-audit-gate-FAIL: missing audit log $AUDIT_LOG" >&2
	echo "Run: bash scripts/planning_export_local_cron.sh" >&2
	exit 1
fi

TAIL_LINES=()
while IFS= read -r line; do
	TAIL_LINES+=("$line")
done < <(tail -n "$REQUIRED_DAYS" "$AUDIT_LOG")

if ((${#TAIL_LINES[@]} < REQUIRED_DAYS)); then
	echo "planning-export-audit-gate-FAIL: audit log has fewer than ${REQUIRED_DAYS} lines" >&2
	exit 1
fi

for line in "${TAIL_LINES[@]}"; do
	if [[ "$line" != *" OK "* ]]; then
		echo "planning-export-audit-gate-FAIL: non-OK line in last ${REQUIRED_DAYS} entries: $line" >&2
		exit 1
	fi
done

UNIQUE_DAYS="$(
	printf '%s\n' "${TAIL_LINES[@]}" | awk -F'T' '{print $1}' | sort -u | wc -l | tr -d '[:space:]'
)"
if (( UNIQUE_DAYS < REQUIRED_DAYS )); then
	echo "planning-export-audit-gate-FAIL: last ${REQUIRED_DAYS} OK lines span only ${UNIQUE_DAYS} distinct UTC days (need ${REQUIRED_DAYS})" >&2
	exit 1
fi

LAST_LINE="${TAIL_LINES[$((${#TAIL_LINES[@]} - 1))]}"
echo "planning-export-audit-gate-ok: ${REQUIRED_DAYS} consecutive OK entries (${UNIQUE_DAYS} distinct days)"
echo "latest: $LAST_LINE"
