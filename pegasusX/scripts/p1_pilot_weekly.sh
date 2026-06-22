#!/usr/bin/env bash
# P1 weekly pilot gate — weeks 1–8 after go-live.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

echo "P1 pilot weekly — $(date -u +%Y-%m-%dT%H:%M:%SZ)"

make p0-preflight
make parity-contract-full

if [[ -n "${PUBLIC_BASE_URL:-}" ]]; then
	make cloud-smoke-ssmr
	echo "TIP: run make load-cert-cloud for full SLO profile on staging/prod"
fi

echo ""
echo "p1-pilot-weekly-ok"
echo "Complete human checklist: docs/P1_PILOT_CHECKLIST.md"
echo "  - PX12_ROLE_ROW_QA.md on real devices"
echo "  - Spanner: docs/SPANNER_HOT_PATH_REVIEW.md"
echo "  - Dashboard: pegasusX — Pilot Launch (P1)"
echo "  - Support roster staffed"
