#!/usr/bin/env bash
# PX-PROD-0: preflight before terraform apply / staging wire.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

FAIL=0

check() {
	local label=$1
	shift
	if "$@" >/dev/null 2>&1; then
		echo "OK  $label"
	else
		echo "FAIL $label" >&2
		FAIL=1
	fi
}

check "gcloud" command -v gcloud
check "terraform" command -v terraform
check "kubectl" command -v kubectl
check "jq" command -v jq
check "gcloud-adc" gcloud auth application-default print-access-token

if [[ -f "$ROOT/infra/terraform/staging.tfvars" ]]; then
	echo "OK  staging.tfvars present"
else
	echo "WARN staging.tfvars missing — copy infra/terraform/staging.tfvars.example" >&2
fi

if [[ "${PHASE0_SKIP_WIRE:-0}" == "1" ]]; then
	echo "SKIP wire-ready (PHASE0_SKIP_WIRE=1)"
elif docker info >/dev/null 2>&1; then
	if make wire-ready >/tmp/phase0-wire-ready.log 2>&1; then
		echo "OK  wire-ready"
	else
		echo "FAIL wire-ready — see /tmp/phase0-wire-ready.log" >&2
		FAIL=1
	fi
else
	echo "WARN Docker unavailable — skipping wire-ready (set PHASE0_SKIP_WIRE=1 to silence)" >&2
fi

if ((FAIL != 0)); then
	echo "phase0-preflight-FAIL" >&2
	exit 1
fi

echo "phase0-preflight-ok"
exit 0
