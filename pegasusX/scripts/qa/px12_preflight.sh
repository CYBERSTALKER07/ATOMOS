#!/usr/bin/env bash
# PX-12 automated QA preflight — run before manual role-row sign-off.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

PASS=0
FAIL=0
SKIP=0

run_gate() {
	local name=$1
	shift
	printf '\n==> %s\n' "$name"
	if "$@"; then
		echo "PASS: $name"
		PASS=$((PASS + 1))
		return 0
	fi
	echo "FAIL: $name" >&2
	FAIL=$((FAIL + 1))
	return 1
}

optional_gate() {
	local name=$1
	shift
	printf '\n==> %s (optional)\n' "$name"
	if "$@"; then
		echo "PASS: $name"
		PASS=$((PASS + 1))
	else
		echo "SKIP: $name (non-blocking)" >&2
		SKIP=$((SKIP + 1))
	fi
	return 0
}

echo "PX12 preflight — pegasusX root: $ROOT"
echo "Date: $(date -u +%Y-%m-%dT%H:%M:%SZ)"

run_gate "parity-contract-full" make parity-contract-full
run_gate "gap-hunter-gate" make gap-hunter-gate
run_gate "validate-launch-readiness" make validate-launch-readiness

if [[ "${PX12_SKIP_SSMR:-}" == "1" ]]; then
	optional_gate "test-ssmr-infra" make test-ssmr-infra
else
	run_gate "test-ssmr-infra" make test-ssmr-infra
fi

printf '\n--- PX12 preflight summary ---\n'
printf 'PASS: %s  FAIL: %s  SKIP: %s\n' "$PASS" "$FAIL" "$SKIP"

if (( FAIL > 0 )); then
	echo "px12-preflight-FAIL — fix failing gates before role-row sign-off" >&2
	exit 1
fi

echo "px12-preflight-ok — proceed to docs/qa/PX12_MANUAL_QA_RUNBOOK.md"
exit 0
