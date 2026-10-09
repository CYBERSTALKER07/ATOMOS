#!/usr/bin/env bash
# ==============================================================================
# PegasusX Enterprise CI - Chaos Resilience & Failure Mode Validation Gate
#
# Validates distributed system resilience, fail-fast mechanics, circuit breakers,
# Redis degradation, and Kafka poison pill dead-letter isolation under -race.
# ==============================================================================
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BACKEND_DIR="${ROOT_DIR}/apps/backend-go"

BOLD="\033[1m"
GREEN="\033[0;32m"
RED="\033[0;31m"
YELLOW="\033[0;33m"
CYAN="\033[0;36m"
NC="\033[0m"

log_info() {
  echo -e "${CYAN}${BOLD}[CHAOS-GATE]${NC} $1"
}

log_pass() {
  echo -e "${GREEN}${BOLD}[PASS]${NC} $1"
}

log_fail() {
  echo -e "${RED}${BOLD}[FAIL]${NC} $1" >&2
}

log_info "Starting Chaos Resilience & Failure Mode Validation Suite..."
TOTAL_START=$(date +%s)
FAILED_CHECKS=0

run_check() {
  local name="$1"
  local cmd="$2"
  local start_time
  start_time=$(date +%s)

  log_info "Running: ${BOLD}${name}${NC}..."
  if eval "$cmd"; then
    local duration=$(( $(date +%s) - start_time ))
    log_pass "${name} (${duration}s)"
  else
    local duration=$(( $(date +%s) - start_time ))
    log_fail "${name} failed after ${duration}s"
    FAILED_CHECKS=$((FAILED_CHECKS + 1))
  fi
  echo ""
}

# 1. Circuit Breaker Core Primitive
run_check "Kernel: Circuit Breaker State Transitions & Fast-Fail" \
  "cd '${BACKEND_DIR}' && go test -race -v -run 'TestBreaker_' ./pkg/circuit/..."

# 2. Optimizer Client Circuit Breaker
run_check "Optimizer Client: 504 Timeout Fail-Fast & Auto-Recovery" \
  "cd '${BACKEND_DIR}' && go test -race -v -run 'TestClient_Solve_CircuitBreaker_' ./dispatch/optimizerclient/..."

# 3. Soliq OFD Tax Gateway Circuit Breaker
run_check "Soliq OFD Gateway: Tax Outage Tripping & Offline Queue Failover" \
  "cd '${BACKEND_DIR}' && go test -race -v -run 'TestClient_Submit_CircuitBreaker_' ./soliq/..."

# 4. Routing Engine Circuit Breakers
run_check "Routing Client: OSRM & Google Routes Outage Circuit Protection" \
  "cd '${BACKEND_DIR}' && go test -race -v ./routing/..."

# 5. Payment Provider Circuit Breakers
run_check "Payment Providers: Upstream Gateway Tripping & Isolation" \
  "cd '${BACKEND_DIR}' && go test -race -v ./payment/..."

# 6. Redis Cache Circuit Breaking & Degradation
run_check "Redis Cache: Degradation, In-Memory Fallback & Fail-Closed Invariants" \
  "cd '${BACKEND_DIR}' && go test -race -v -run 'TestCircuitBreakerBackend_' ./cache/..."

# 7. Transactional Outbox Poison Pill Dead-Letter Quarantine
run_check "Outbox Relay: Poison Pill Dead-Letter Quarantine & Non-Blocking Draining" \
  "cd '${BACKEND_DIR}' && go test -race -v -run 'TestRelayDrainOnce_PoisonPillIsolation_|TestRelayDrainOnce_RecordsPublishFailuresAndDeadLetters|TestRelayDrainOnceBoundsWedgedPublisher' ./outbox/..."

# 8. Kafka HA and Multi-Broker Infrastructure Contracts
run_check "Kafka HA: Strimzi & Managed Kafka RF=3 / min.isr=2 / DLQ Gate" \
  "bash '${ROOT_DIR}/scripts/ci_kafka_ha_gate.sh'"

TOTAL_DURATION=$(( $(date +%s) - TOTAL_START ))

echo "=============================================================================="
if [[ "${FAILED_CHECKS}" -eq 0 ]]; then
  echo -e "${GREEN}${BOLD}✓ ALL CHAOS RESILIENCE & FAILURE MODE CHECKS PASSED (${TOTAL_DURATION}s)${NC}"
  echo "=============================================================================="
  exit 0
else
  echo -e "${RED}${BOLD}✗ CHAOS RESILIENCE GATE FAILED: ${FAILED_CHECKS} check(s) failed (${TOTAL_DURATION}s)${NC}"
  echo "=============================================================================="
  exit 1
fi
