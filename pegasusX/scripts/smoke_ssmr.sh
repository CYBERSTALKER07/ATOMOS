#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_FILE="$REPO_ROOT/infra/docker-compose.ssmr.yml"
ENV_FILE="${SSMR_ENV_FILE:-$REPO_ROOT/.env.ssmr.example}"

if [[ ! -f "$COMPOSE_FILE" ]]; then
	echo "Missing compose file: $COMPOSE_FILE" >&2
	exit 1
fi
if [[ ! -f "$ENV_FILE" ]]; then
	echo "Missing env file: $ENV_FILE" >&2
	exit 1
fi
if ! command -v docker >/dev/null 2>&1; then
	echo "docker is required" >&2
	exit 1
fi
if ! command -v go >/dev/null 2>&1; then
	echo "go is required" >&2
	exit 1
fi
if ! command -v curl >/dev/null 2>&1; then
	echo "curl is required" >&2
	exit 1
fi

load_env_file() {
	local env_file=$1
	local line
	local key
	local value
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
SPANNER_HOST="${SPANNER_EMULATOR_HOST%%:*}"
SPANNER_PORT="${SPANNER_EMULATOR_HOST##*:}"
REDIS_PORT="${REDIS_ADDR##*:}"
REDIS_HOST="${REDIS_ADDR%%:*}"
KAFKA_HOST="${KAFKA_BROKERS%%,*}"
KAFKA_HOST="${KAFKA_HOST%%:*}"
KAFKA_PORT="${KAFKA_BROKERS%%,*}"
KAFKA_PORT="${KAFKA_PORT##*:}"
HEALTH_URL="${PUBLIC_BASE_URL%/}/v1/health"
GO_TMP_ROOT="${TMPDIR:-$REPO_ROOT/.tmp}/ssmr-smoke"

log_step() {
	printf '==> %s\n' "$1"
}

cleanup() {
	local exit_code=$?
	trap - EXIT INT TERM
	if (( exit_code != 0 )); then
		echo "Smoke-check failed; dumping compose status and recent logs" >&2
		"${COMPOSE[@]}" ps >&2 || true
		"${COMPOSE[@]}" logs --tail 40 spanner-emulator redis zookeeper kafka kafka-init backend-setup backend-go ai-worker >&2 || true
	fi
	"${COMPOSE[@]}" down -v --remove-orphans >/dev/null 2>&1 || true
	exit "$exit_code"
}

trap cleanup EXIT INT TERM

tcp_ready() {
	local host=$1
	local port=$2
	if command -v nc >/dev/null 2>&1; then
		nc -z "$host" "$port" >/dev/null 2>&1
		return $?
	fi
	(: >"/dev/tcp/$host/$port") >/dev/null 2>&1
}

wait_for_tcp() {
	local label=$1
	local host=$2
	local port=$3
	local attempts=${4:-30}
	local delay=${5:-2}
	local attempt
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
	local label=$1
	local url=$2
	local attempts=${3:-30}
	local delay=${4:-2}
	local attempt
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
	mkdir -p "$GO_TMP_ROOT/go-build" "$GO_TMP_ROOT/go-mod"
	if [[ "$goflags" != *"-buildvcs=false"* ]]; then
		goflags="${goflags:+$goflags }-buildvcs=false"
	fi
	(
		cd "$REPO_ROOT"
		TMPDIR="$GO_TMP_ROOT" \
		GOCACHE="$GO_TMP_ROOT/go-build" \
		GOMODCACHE="$GO_TMP_ROOT/go-mod" \
		GOFLAGS="$goflags" \
		go run ./apps/backend-go/cmd/ssmr-smokecheck "$check"
	)
}

assert_redis() {
	local pong
	if command -v redis-cli >/dev/null 2>&1; then
		pong="$(redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" PING | tr -d '\r')"
	else
		pong="$("${COMPOSE[@]}" exec -T redis redis-cli -p 6379 PING | tr -d '\r')"
	fi
	if [[ "$pong" != "PONG" ]]; then
		echo "redis ping failed: got ${pong}" >&2
		return 1
	fi
	return 0
}

log_step "Tearing down any prior SSMR stack (including Kafka/Zookeeper volumes)"
"${COMPOSE[@]}" down -v --remove-orphans >/dev/null 2>&1 || true

log_step "Starting isolated SSMR compose stack"
"${COMPOSE[@]}" up -d

log_step "Waiting for Spanner emulator, Redis, Zookeeper, and Kafka"
wait_for_tcp "Spanner emulator" "$SPANNER_HOST" "$SPANNER_PORT" 60 2
wait_for_tcp "Redis" "$REDIS_HOST" "$REDIS_PORT" 30 2
wait_for_tcp "Zookeeper" "localhost" "22181" 30 2

log_step "Waiting for Kafka healthcheck (broker registration in Zookeeper)"
kafka_attempt=0
while (( kafka_attempt < 40 )); do
	kafka_attempt=$((kafka_attempt + 1))
	if "${COMPOSE[@]}" ps --status running --format json kafka 2>/dev/null | grep -q '"Health":"healthy"'; then
		break
	fi
	if tcp_ready "$KAFKA_HOST" "$KAFKA_PORT" && \
		"${COMPOSE[@]}" exec -T kafka kafka-topics --bootstrap-server localhost:9094 --list >/dev/null 2>&1; then
		break
	fi
	if (( kafka_attempt >= 40 )); then
		echo "Timed out waiting for Kafka to become healthy" >&2
		exit 1
	fi
	sleep 3
done

log_step "Creating isolated Kafka topics"
"${COMPOSE[@]}" run --rm kafka-init

log_step "Running idempotent bootstrap against isolated Spanner"
"${COMPOSE[@]}" run --rm backend-setup

log_step "Asserting seeded supplier row and Retailers schema via direct Spanner probe"
run_go_smokecheck spanner

log_step "Pinging isolated Redis"
assert_redis

log_step "Ensuring backend and ai-worker are running"
"${COMPOSE[@]}" up -d backend-go ai-worker

log_step "Waiting for backend health"
wait_for_http "backend health" "$HEALTH_URL" 90 2
# Warm up go run backend after volume-mount compile before e2e traffic.
wait_for_http "backend health (warmup)" "$HEALTH_URL" 10 3

log_step "Asserting Redis-backed delivery perimeter key and positive membership path"
run_go_smokecheck spatial

log_step "Asserting isolated Kafka topics and round-trip message flow"
run_go_smokecheck kafka

log_step "Running end-to-end supplier→retailer→order→tracking flow"
mkdir -p "$REPO_ROOT/.ssmr/import-uploads"
export SSMR_IMPORT_LOCAL_ROOT="$REPO_ROOT/.ssmr/import-uploads"
E2E_LOG="$GO_TMP_ROOT/ssmr-e2e.log"
run_go_smokecheck e2e 2>&1 | tee "$E2E_LOG"

log_step "Asserting full-ecosystem PX_E2E marker coverage (configuration + lifecycle + realtime)"
bash "$REPO_ROOT/scripts/parity/ssmr_ecosystem_marker_gate.sh" "$E2E_LOG"

log_step "SSMR smoke-check completed successfully"
echo "__SSMR_OK__"