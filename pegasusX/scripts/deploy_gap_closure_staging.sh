#!/usr/bin/env bash
# Deploy gap-closure surfaces to staging (run from operator workstation with GCP access).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

echo "==> gap-closure staging deploy"
echo "Prereq: phase0-apply, phase0-sync-secrets, phase0-migrate completed"

echo "1. Confirm gap flags OFF in backend config:"
echo "   CASH_RECONCILIATION_REQUIRED=false"
echo "   CREDIT_NOTE_AUTO_FROM_BUYER_REJECT=false"
echo "   CREDIT_NOTE_AUTO_FROM_CLAIM=false"
echo "   (do not set CREDIT_SCORE_ENFORCEMENT_ENABLED — dead flag)"

echo "2. Build and push images (example):"
echo "   make docker-build-backend"
echo "   # push to Artifact Registry per your release train"

echo "3. Rollout order:"
echo "   - backend-go"
echo "   - supplier-portal"
echo "   - warehouse-portal"

echo "4. Post-deploy smoke:"
echo "   export PUBLIC_BASE_URL=https://api-ssmr.pegasusx.app"
echo "   bash scripts/discover_staging_wiring.sh"
echo "   bash scripts/staging_smoke.sh"

echo "gap-closure-deploy-instructions-ok"
