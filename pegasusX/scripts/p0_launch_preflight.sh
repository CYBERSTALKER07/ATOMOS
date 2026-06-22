#!/usr/bin/env bash
# P0 launch preflight — automated gates before real-client pilot (local/CI).
# GCP staging load-cert and human hypercare are documented in docs/P0_LAUNCH_CHECKLIST.md.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
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
		echo "SKIP: $name" >&2
		SKIP=$((SKIP + 1))
	fi
	return 0
}

echo "P0 launch preflight — pegasusX root: $ROOT"
echo "Date: $(date -u +%Y-%m-%dT%H:%M:%SZ)"

run_gate "backend unit tests (short)" bash -c 'cd apps/backend-go && go test ./... -short -count=1'
run_gate "production profile manifests" bash scripts/validate_production_profile.sh
run_gate "validate-backend-k8s" make validate-backend-k8s
run_gate "validate-ai-worker-k8s" make validate-ai-worker-k8s
run_gate "gen-contracts-gate" make gen-contracts-gate
run_gate "gap-hunter-gate" make gap-hunter-gate

if command -v kubectl >/dev/null 2>&1; then
	run_gate "kustomize prod overlay" kubectl kustomize infra/k8s/overlays/prod --load-restrictor LoadRestrictionsNone >/dev/null
else
	optional_gate "kustomize prod overlay" false
fi

if [[ "${P0_SKIP_SSMR:-}" == "1" ]]; then
	optional_gate "test-ssmr-infra" make test-ssmr-infra
else
	run_gate "test-ssmr-infra" make test-ssmr-infra
fi

if [[ -n "${PUBLIC_BASE_URL:-}" ]]; then
	run_gate "cloud-smoke" bash scripts/cloud_smoke_ssmr.sh
else
	optional_gate "cloud-smoke (set PUBLIC_BASE_URL)" false
fi

printf '\n--- P0 preflight summary ---\n'
printf 'PASS: %s  FAIL: %s  SKIP: %s\n' "$PASS" "$FAIL" "$SKIP"

if (( FAIL > 0 )); then
	echo "p0-preflight-FAIL" >&2
	exit 1
fi

echo "p0-preflight-ok — see docs/P0_LAUNCH_CHECKLIST.md for human/GCP steps"
exit 0
