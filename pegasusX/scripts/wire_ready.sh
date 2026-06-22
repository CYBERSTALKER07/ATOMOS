#!/usr/bin/env bash
# Wire-ready gate — automated evidence before GCP staging wiring.
# Requires Docker for SSMR (make test-ssmr-infra).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

PASS=0
FAIL=0

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

echo "wire-ready gate — pegasusX root: $ROOT"
echo "Date: $(date -u +%Y-%m-%dT%H:%M:%SZ)"

run_gate "production profile manifests" bash scripts/validate_production_profile.sh
run_gate "backend unit tests (short)" bash -c 'cd apps/backend-go && go test ./... -short -count=1'
run_gate "parity-contract-full" make parity-contract-full
run_gate "gap-hunter-gate" make gap-hunter-gate
run_gate "gen-contracts-gate" make gen-contracts-gate
run_gate "validate-backend-k8s" make validate-backend-k8s
run_gate "validate-ai-worker-k8s" make validate-ai-worker-k8s

if command -v kubectl >/dev/null 2>&1; then
	run_gate "kustomize prod overlay" kubectl kustomize infra/k8s/overlays/prod --load-restrictor LoadRestrictionsNone >/dev/null
else
	echo "SKIP: kustomize prod overlay (kubectl not installed)" >&2
fi

run_gate "test-ssmr-infra" make test-ssmr-infra

printf '\n--- wire-ready summary ---\n'
printf 'PASS: %s  FAIL: %s\n' "$PASS" "$FAIL"

if (( FAIL > 0 )); then
	echo "wire-ready-FAIL — fix failing gates before staging wire" >&2
	exit 1
fi

echo "wire-ready-ok — proceed to: make px12-preflight"
exit 0
