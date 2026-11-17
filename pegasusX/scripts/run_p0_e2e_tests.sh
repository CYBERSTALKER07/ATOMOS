#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "================================================================="
echo "PegasusX P0 Remediation: Comprehensive E2E Verification Suite"
echo "Root directory: ${ROOT_DIR}"
echo "Timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "================================================================="

echo ""
echo "[Step 1/2] Running Backend Go P0 E2E Test Suite with -race..."
(
    cd "${ROOT_DIR}/apps/backend-go"
    go test -race -v -count=1 ./e2ep0/...
)
echo ">> Backend Go P0 E2E tests PASSED (100% success)."

echo ""
echo "[Step 2/2] Running Admin Portal Session Route Vitest Suite..."
(
    cd "${ROOT_DIR}/apps/admin-portal"
    npx vitest run app/api/auth/session/__tests__/session-route.test.ts
)
echo ">> Admin Portal Session tests PASSED (100% success)."

echo ""
echo "================================================================="
echo "ALL P0 E2E TESTS PASSED SUCCESSFULLY (Zero Defects, Zero Bypass)"
echo "================================================================="
