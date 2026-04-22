#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# Domino Smoke Test — validates the WireMock Global Pay simulator is alive
# and all 14 mappings are loaded before starting the backend.
#
# Usage:  ./scripts/domino-smoke.sh
# Prereq: docker-compose up -d globalpay-mock
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail

MOCK_URL="${GLOBAL_PAY_MOCK_URL:-http://localhost:8082}"
PASS=0
FAIL=0

smoke() {
  local label="$1" method="$2" path="$3" body="${4:-}" expect_status="$5"
  local url="${MOCK_URL}${path}"
  local args=(-s -o /dev/null -w "%{http_code}" -X "$method" -H "Content-Type: application/json")
  if [[ -n "$body" ]]; then
    args+=(-d "$body")
  fi
  local status
  status=$(curl "${args[@]}" "$url" --max-time 5 2>/dev/null || echo "000")
  if [[ "$status" == "$expect_status" ]]; then
    printf "  ✓ %-40s %s\n" "$label" "$status"
    ((PASS++))
  else
    printf "  ✗ %-40s got %s, expected %s\n" "$label" "$status" "$expect_status"
    ((FAIL++))
  fi
}

echo "══════════════════════════════════════════════════════════════"
echo " Domino Smoke Test — Global Pay WireMock Simulator"
echo "══════════════════════════════════════════════════════════════"
echo ""

# Check WireMock is alive
echo "▸ Health Check"
smoke "WireMock admin" GET "/__admin" "" "200"

# Check mapping count
MAPPING_COUNT=$(curl -s "${MOCK_URL}/__admin/mappings" 2>/dev/null | grep -o '"total" *: *[0-9]*' | grep -o '[0-9]*' || echo "0")
if [[ "$MAPPING_COUNT" -ge 14 ]]; then
  printf "  ✓ %-40s %s mappings loaded\n" "Mapping count" "$MAPPING_COUNT"
  ((PASS++))
else
  printf "  ✗ %-40s %s mappings (expected ≥14)\n" "Mapping count" "$MAPPING_COUNT"
  ((FAIL++))
fi

echo ""
echo "▸ Happy Path"
smoke "Auth (merchant login)"        POST "/v1/merchant/auth"                    '{"username":"test","password":"test"}' "200"
smoke "Hosted checkout create"       POST "/v1/user-service-tokens"              '{"serviceId":"SVC-1","amount":50000}'  "200"
smoke "Status lookup (paid)"         POST "/v1/payment/status"                   '{"paymentId":"PAY-1"}'                "200"
smoke "Card save"                    POST "/cards/v1/card"                       '{"phone":"998901234567"}'             "200"
smoke "Card OTP confirm"             POST "/cards/v1/card/confirm/MOCK_TOKEN"    '{"otp":"123456"}'                     "200"
smoke "Direct payment init"          POST "/payments/v2/payment/init"            '{"cardToken":"CT-1","amount":50000}'  "200"
smoke "Direct payment perform"       POST "/payments/v2/payment/MOCK-ID/perform" ''                                    "200"
smoke "Direct payment revert"        POST "/payments/v2/payment/MOCK-ID/refund"  '{"amount":50000}'                    "200"

echo ""
echo "▸ Chaos Path"
smoke "Declined (402)"               POST "/payments/v2/payment/init"            '{"externalId":"ORDER-CHAOS_DECLINE"}' "402"
smoke "Bank error (500)"             POST "/payments/v2/payment/init"            '{"externalId":"ORDER-CHAOS_500"}'     "500"
smoke "Invalid split (400)"          POST "/payments/v2/payment/init"            '{"recipients":[{"amount":-100}]}'     "400"
smoke "3DS redirect (200+URL)"       POST "/payments/v2/payment/init"            '{"externalId":"ORDER-CHAOS_3DS"}'     "200"
smoke "Status pending"               POST "/v1/payment/status"                   '{"paymentId":"CHAOS_PENDING"}'        "200"

# Timeout test (skipped by default — takes 35s)
echo ""
echo "▸ Timeout test (skipped — use: curl -X POST ${MOCK_URL}/payments/v2/payment/init -d '{\"externalId\":\"CHAOS_TIMEOUT\"}' --max-time 5)"

echo ""
echo "══════════════════════════════════════════════════════════════"
echo " Results: ${PASS} passed, ${FAIL} failed"
echo "══════════════════════════════════════════════════════════════"

if [[ "$FAIL" -gt 0 ]]; then
  exit 1
fi
