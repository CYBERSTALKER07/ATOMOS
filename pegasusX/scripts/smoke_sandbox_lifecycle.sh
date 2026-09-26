#!/usr/bin/env bash
# Phase 6 — SSMR lifecycle vertical only (create → dispatch → seal → complete).
# Faster iteration than full smoke_ssmr.sh; same Docker SSMR stack.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
if [[ -f "$REPO_ROOT/infra/docker-compose.sandbox.yml" ]]; then
	COMPOSE_FILE="$REPO_ROOT/infra/docker-compose.sandbox.yml"
else
	COMPOSE_FILE="$REPO_ROOT/infra/docker-compose.ssmr.yml"
fi
if [[ -n "${SANDBOX_ENV_FILE:-}" ]]; then
	ENV_FILE="$SANDBOX_ENV_FILE"
elif [[ -n "${SSMR_ENV_FILE:-}" ]]; then
	ENV_FILE="$SSMR_ENV_FILE"
elif [[ -f "$REPO_ROOT/.env.sandbox.example" ]]; then
	ENV_FILE="$REPO_ROOT/.env.sandbox.example"
else
	ENV_FILE="$REPO_ROOT/.env.ssmr.example"
fi
MARKER_MANIFEST="$REPO_ROOT/contracts/ssmr_lifecycle_vertical_markers.json"

if [[ ! -f "$COMPOSE_FILE" ]]; then
  echo "Missing compose file: $COMPOSE_FILE" >&2
  exit 1
fi
if [[ ! -f "$ENV_FILE" ]]; then
  echo "Missing env file: $ENV_FILE" >&2
  exit 1
fi
if ! command -v docker >/dev/null 2>&1; then
  echo "docker is required for Phase 6 lifecycle vertical" >&2
  echo "Install/start Docker Desktop, then re-run: make test-ssmr-lifecycle" >&2
  exit 1
fi
if ! command -v go >/dev/null 2>&1; then
  echo "go is required" >&2
  exit 1
fi

load_env_file() {
  local env_file=$1 line key value
  while IFS= read -r line || [[ -n "$line" ]]; do
    if [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]]; then
      continue
    fi
    key=${line%%=*}
    value=${line#*=}
    key=${key#export }
    export "$key=$value"
  done < "$env_file"
}

load_env_file "$ENV_FILE"

COMPOSE=(docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE")
HEALTH_URL="${PUBLIC_BASE_URL%/}/v1/health"
GO_TMP_ROOT="${TMPDIR:-$REPO_ROOT/.tmp}/ssmr-lifecycle"
SPANNER_HOST="${SPANNER_EMULATOR_HOST%%:*}"
SPANNER_PORT="${SPANNER_EMULATOR_HOST##*:}"
REDIS_PORT="${REDIS_ADDR##*:}"
REDIS_HOST="${REDIS_ADDR%%:*}"
KAFKA_HOST="${KAFKA_BROKERS%%,*}"
KAFKA_HOST="${KAFKA_HOST%%:*}"
KAFKA_PORT="${KAFKA_BROKERS%%,*}"
KAFKA_PORT="${KAFKA_PORT##*:}"

CURRENT_STEP="init"
cleanup() {
  local exit_code=$?
  trap - EXIT INT TERM
  if (( exit_code != 0 )); then
    echo "Lifecycle smoke failed at step: ${CURRENT_STEP} (exit ${exit_code})" >&2
    "${COMPOSE[@]}" ps >&2 || true
    "${COMPOSE[@]}" logs --tail 50 spanner-emulator redis kafka backend-setup backend-go >&2 || true
  fi
  "${COMPOSE[@]}" down -v --remove-orphans >/dev/null 2>&1 || true
  exit "$exit_code"
}
trap cleanup EXIT INT TERM

tcp_ready() {
  local host=$1 port=$2
  if command -v nc >/dev/null 2>&1; then
    nc -z "$host" "$port" >/dev/null 2>&1
    return $?
  fi
  (: >"/dev/tcp/$host/$port") >/dev/null 2>&1
}

wait_for_tcp() {
  local label=$1 host=$2 port=$3 attempts=${4:-40} delay=${5:-2} attempt
  for ((attempt = 1; attempt <= attempts; attempt++)); do
    if tcp_ready "$host" "$port"; then
      return 0
    fi
    sleep "$delay"
  done
  echo "Timed out waiting for ${label} at ${host}:${port}" >&2
  return 1
}

wait_for_http() {
  local label=$1 url=$2 attempts=${3:-90} delay=${4:-2} attempt
  for ((attempt = 1; attempt <= attempts; attempt++)); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep "$delay"
  done
  echo "Timed out waiting for ${label} at ${url}" >&2
  return 1
}

run_go_smokecheck() {
  local check=$1
  local goflags="${GOFLAGS:-}"
  mkdir -p "$GO_TMP_ROOT/go-build"
  if [[ "$goflags" != *"-buildvcs=false"* ]]; then
    goflags="${goflags:+$goflags }-buildvcs=false"
  fi
  (
    cd "$REPO_ROOT/apps/backend-go"
    TMPDIR="$GO_TMP_ROOT" GOCACHE="$GO_TMP_ROOT/go-build" GOFLAGS="$goflags" \
      go run ./cmd/ssmr-smokecheck "$check"
  )
}

echo "==> Phase 6 lifecycle vertical (create → dispatch → seal → complete)"
CURRENT_STEP="compose-down"
"${COMPOSE[@]}" down -v --remove-orphans >/dev/null 2>&1 || true

CURRENT_STEP="compose-up"
"${COMPOSE[@]}" up -d spanner-emulator redis optimizer-core
"${COMPOSE[@]}" up -d zookeeper
wait_for_tcp "Zookeeper" "localhost" "22181" 60 2
sleep 2
"${COMPOSE[@]}" up -d kafka
wait_for_tcp "Spanner emulator" "$SPANNER_HOST" "$SPANNER_PORT" 60 2
wait_for_tcp "Redis" "$REDIS_HOST" "$REDIS_PORT" 30 2

# Kafka readiness (best-effort list topics)
kafka_attempt=0
while (( kafka_attempt < 40 )); do
  kafka_attempt=$((kafka_attempt + 1))
  if tcp_ready "$KAFKA_HOST" "$KAFKA_PORT" && \
    "${COMPOSE[@]}" exec -T kafka kafka-topics --bootstrap-server localhost:9094 --list >/dev/null 2>&1; then
    break
  fi
  if (( kafka_attempt >= 40 )); then
    echo "Timed out waiting for Kafka" >&2
    exit 1
  fi
  sleep 3
done

CURRENT_STEP="kafka-init"
"${COMPOSE[@]}" run --rm kafka-init

CURRENT_STEP="backend-setup"
"${COMPOSE[@]}" run --rm backend-setup

CURRENT_STEP="smokecheck-spanner"
run_go_smokecheck spanner

CURRENT_STEP="compose-backend"
"${COMPOSE[@]}" up -d backend-go ai-worker

CURRENT_STEP="backend-health"
wait_for_http "backend health" "$HEALTH_URL" 180 2

CURRENT_STEP="lifecycle-vertical"
E2E_LOG="$GO_TMP_ROOT/ssmr-lifecycle-e2e.log"
mkdir -p "$GO_TMP_ROOT"
run_go_smokecheck lifecycle-vertical 2>&1 | tee "$E2E_LOG"

CURRENT_STEP="marker-gate"
SSMR_MARKER_MANIFEST="$MARKER_MANIFEST" \
  bash "$REPO_ROOT/scripts/parity/ssmr_ecosystem_marker_gate.sh" "$E2E_LOG"

echo "__SSMR_LIFECYCLE_VERTICAL_OK__"
