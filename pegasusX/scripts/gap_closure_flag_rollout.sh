#!/usr/bin/env bash
# Enable gap-closure flags on staging one at a time. Roll back on smoke failure.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

usage() {
	cat <<'EOF'
Usage: gap_closure_flag_rollout.sh <flag_name> <true|false>

Flags (enable in order per docs/gap-closure/STAGING_FLAGS.md):
  CREDIT_NOTE_AUTO_FROM_BUYER_REJECT
  CREDIT_NOTE_AUTO_FROM_CLAIM
  CASH_RECONCILIATION_REQUIRED   # last — only after manual driver cash path

Do not use CREDIT_SCORE_ENFORCEMENT_ENABLED — dead flag (Phase A scoring removal).

After each enable:
  export PUBLIC_BASE_URL=<staging>
  bash scripts/staging_smoke.sh
EOF
}

FLAG="${1:-}"
VALUE="${2:-}"
if [[ -z "$FLAG" || -z "$VALUE" ]]; then
	usage
	exit 1
fi

case "$FLAG" in
	CREDIT_NOTE_AUTO_FROM_BUYER_REJECT|CREDIT_NOTE_AUTO_FROM_CLAIM|CASH_RECONCILIATION_REQUIRED) ;;
	CREDIT_SCORE_ENFORCEMENT_ENABLED)
		echo "dead flag: CREDIT_SCORE_ENFORCEMENT_ENABLED (credit risk scoring removed Phase A)" >&2
		exit 1
		;;
	*)
		echo "unknown flag: $FLAG" >&2
		usage
		exit 1
		;;
esac

if [[ "$VALUE" != "true" && "$VALUE" != "false" ]]; then
	echo "value must be true or false" >&2
	exit 1
fi

echo "==> Set $FLAG=$VALUE on staging backend (GSM / k8s configmap / Secret Manager)"
echo "    Update pegasusx-staging backend env or overlay configmap, then rollout backend-go:"
echo "    kubectl rollout restart deployment/backend-go -n pegasusx-ssmr"
echo ""
echo "Post-change smoke:"
echo "  export PUBLIC_BASE_URL=\${PUBLIC_BASE_URL:-https://api-ssmr.pegasusx.app}"
echo "  bash scripts/staging_smoke.sh"
echo ""
echo "Rollback: re-run this script with value false for the same flag."
