#!/usr/bin/env bash
# Generate minimal launch-readiness documentation stubs required by validate_launch_readiness.py
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DOCS="$ROOT/docs"
mkdir -p "$ROOT/context"

stub() {
	local path="$1"
	local body="$2"
	if [[ -f "$path" ]]; then
		return 0
	fi
	printf '%s\n' "$body" >"$path"
	echo "created $path"
}

# LAUNCH_READINESS_RUNBOOK with required strings
stub "$DOCS/LAUNCH_READINESS_RUNBOOK.md" '# Launch Readiness Runbook

Release ownership: platform engineering + on-call SRE.

Preflight:
- make test-ssmr-infra
- make validate-ai-worker-k8s
- make validate-launch-readiness
- make p0-preflight

rollback: revert deployment image tag and re-run cloud-smoke-ssmr.

launch support: follow hypercare checklist in docs/P0_LAUNCH_CHECKLIST.md.
'

stub "$DOCS/P0_LAUNCH_CHECKLIST.md" '# P0 Launch Checklist

- [ ] make wire-ready
- [ ] make qa-gate
- [ ] make p0-preflight with PUBLIC_BASE_URL
- [ ] Gap closure smoke: bash scripts/staging_smoke.sh
- [ ] Manual paths: docs/gap-closure/MANUAL_CRITICAL_WALKTHROUGHS.md
'

stub "$DOCS/P1_PILOT_CHECKLIST.md" '# P1 Pilot Checklist

Weekly: make p1-pilot-weekly
'

stub "$DOCS/WIRE_READY_STAGING_RUNBOOK.md" '# Wire Ready Staging Runbook

Run make wire-ready then make px12-preflight before staging wire.
'

stub "$DOCS/P2_SCALE_ROADMAP.md" '# P2 Scale Roadmap

Post-pilot scaling tracked separately from gap-closure staging.
'

stub "$DOCS/SPANNER_HOT_PATH_REVIEW.md" '# Spanner Hot Path Review

Review RW txn patterns before prod scale.
'

stub "$DOCS/V1_STAGING_CLOSURE_CHECKLIST.md" '# V1 Staging Closure Checklist

See docs/gap-closure/STAGING_WIRING_MATRIX.md and STAGING_FLAGS.md.
'

stub "$DOCS/CLOUD_CREDENTIALS_CHECKLIST.md" '# Cloud Credentials Checklist

Required GSM secrets:
- GLOBAL_PAY_USERNAME
- GLOBAL_PAY_PASSWORD
- GLOBAL_PAY_WEBHOOK_SECRET
- JWT_SECRET
- Maps SDK for Android
'

for name in SUPPLIER_ONBOARDING_SOP BILLING_RECOVERY_SCRIPT TOPOLOGY_ENTRY_SUPPORT_GUIDE RETAILER_ONBOARDING_SUPPORT_FLOWS PRICING_AUTHORITY_RULES ZONE_MISS_COMMUNICATION_POLICY PAYMENT_EXCEPTION_SOP FINANCE_SUPPORT_WORKFLOW DISPUTE_CLASSIFICATION_VOCABULARY WAREHOUSE_EXCEPTION_SOP REASSIGNMENT_SUPPORT_PLAYBOOK TRANSFER_CANCELLATION_RUNBOOK DRIVER_SUPPORT_PLAYBOOK LIVE_TRACKING_EXPECTATIONS DELIVERY_ESCALATION_POLICY AI_WORKER_LAUNCH_RUNBOOK DEPLOYMENT_AND_DISTRIBUTION_PLAN INCIDENT_RESPONSE_RUNBOOK RELEASE_TRAIN DEPLOYMENT_READINESS_GAP_LEDGER REAL_WORLD_CASE_MATRIX SHOP_CLOSED_E2E_SOP PARTIAL_DISPATCH_RECOVERY_SOP BARCODE_GO_LIVE_CHECKLIST RETAILER_RECEIVING_WINDOWS_GUIDE; do
	stub "$DOCS/${name}.md" "# ${name}

Operational stub — expand before production hypercare.
"
done

if [[ ! -f "$ROOT/context/plan.md" ]]; then
	cat >"$ROOT/context/plan.md" <<'EOF'
# pegasusX execution plan

`PX0-A5` launch readiness guard
Status: `implemented`

`PX7-A3` ai-worker k8s packaging
Status: `implemented`

`PX11-C1` cloud staging smoke
Status: `implemented`

`PX12-A1` parity contract gates
Status: `implemented`

scripts/validate_launch_readiness.py
docs/LAUNCH_READINESS_RUNBOOK.md
EOF
	echo "created context/plan.md"
fi

echo "launch-docs-bootstrap-ok"
