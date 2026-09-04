#!/usr/bin/env bash
# Shared helpers for SSMR stack bring-up (no teardown). Sourced by fire-drill and planning export scripts.
set -euo pipefail

ssmr_lib_init() {
	SSMR_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
	SSMR_REPO_ROOT="$(cd "$SSMR_LIB_DIR/../.." && pwd)"
	if [[ -f "$SSMR_REPO_ROOT/infra/docker-compose.sandbox.yml" ]]; then
		SSMR_COMPOSE_FILE="${SANDBOX_COMPOSE_FILE:-${SSMR_COMPOSE_FILE:-$SSMR_REPO_ROOT/infra/docker-compose.sandbox.yml}}"
	else
		SSMR_COMPOSE_FILE="${SSMR_COMPOSE_FILE:-$SSMR_REPO_ROOT/infra/docker-compose.ssmr.yml}"
	fi
	if [[ -n "${SANDBOX_ENV_FILE:-}" ]]; then
		SSMR_ENV_FILE="$SANDBOX_ENV_FILE"
	elif [[ -n "${SSMR_ENV_FILE:-}" ]]; then
		:
	elif [[ -f "$SSMR_REPO_ROOT/.env.sandbox.example" ]]; then
		SSMR_ENV_FILE="$SSMR_REPO_ROOT/.env.sandbox.example"
	else
		SSMR_ENV_FILE="$SSMR_REPO_ROOT/.env.ssmr.example"
	fi
	SSMR_GO_TMP_ROOT="${TMPDIR:-$SSMR_REPO_ROOT/.tmp}/ssmr-stack"
	SSMR_ARTIFACTS_DIR="${SSMR_ARTIFACTS_DIR:-$SSMR_REPO_ROOT/artifacts}"

	if [[ ! -f "$SSMR_COMPOSE_FILE" ]]; then
		echo "Missing compose file: $SSMR_COMPOSE_FILE" >&2
		return 1
	fi
	if [[ ! -f "$SSMR_ENV_FILE" ]]; then
		echo "Missing env file: $SSMR_ENV_FILE" >&2
		return 1
	fi
	if ! command -v docker >/dev/null 2>&1; then
		echo "docker is required" >&2
		return 1
	fi

	ssmr_load_env_file "$SSMR_ENV_FILE"

	SSMR_COMPOSE=(docker compose --env-file "$SSMR_ENV_FILE" -f "$SSMR_COMPOSE_FILE")
	SSMR_SPANNER_HOST="${SPANNER_EMULATOR_HOST%%:*}"
	SSMR_SPANNER_PORT="${SPANNER_EMULATOR_HOST##*:}"
	SSMR_REDIS_HOST="${REDIS_ADDR%%:*}"
	SSMR_REDIS_PORT="${REDIS_ADDR##*:}"
	SSMR_KAFKA_HOST="${KAFKA_BROKERS%%,*}"
	SSMR_KAFKA_HOST="${SSMR_KAFKA_HOST%%:*}"
	SSMR_KAFKA_PORT="${KAFKA_BROKERS%%,*}"
	SSMR_KAFKA_PORT="${SSMR_KAFKA_PORT##*:}"
	SSMR_HEALTH_URL="${PUBLIC_BASE_URL%/}/v1/health"
	SSMR_KAFKA_TOPIC_MAIN_DLQ="${KAFKA_TOPIC_MAIN:-ssmr.events.orders}-dlq"
}

ssmr_load_env_file() {
	local env_file=$1
	local line key value
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

ssmr_log() {
	printf '==> %s\n' "$1"
}

ssmr_tcp_ready() {
	local host=$1
	local port=$2
	if command -v nc >/dev/null 2>&1; then
		nc -z "$host" "$port" >/dev/null 2>&1
		return $?
	fi
	(: >"/dev/tcp/$host/$port") >/dev/null 2>&1
}

ssmr_wait_tcp() {
	local label=$1
	local host=$2
	local port=$3
	local attempts=${4:-30}
	local delay=${5:-2}
	local attempt
	for ((attempt = 1; attempt <= attempts; attempt++)); do
		if ssmr_tcp_ready "$host" "$port"; then
			return 0
		fi
		sleep "$delay"
	done
	echo "Timed out waiting for ${label} at ${host}:${port}" >&2
	return 1
}

ssmr_wait_http() {
	local label=$1
	local url=$2
	local attempts=${3:-60}
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

ssmr_compose_up_retry() {
	local attempts=${1:-5}
	local delay=${2:-3}
	local attempt
	shift 2
	for ((attempt = 1; attempt <= attempts; attempt++)); do
		if "${SSMR_COMPOSE[@]}" up -d "$@"; then
			return 0
		fi
		echo "compose up retry ${attempt} for: $*" >&2
		sleep "$delay"
	done
	return 1
}

ssmr_stack_healthy() {
	curl -fsS "$SSMR_HEALTH_URL" >/dev/null 2>&1
}

ssmr_ensure_stack() {
	ssmr_lib_init

	if ssmr_stack_healthy; then
		ssmr_log "SSMR stack already healthy at ${SSMR_HEALTH_URL}"
		return 0
	fi

	ssmr_log "Bringing up SSMR stack (no teardown on exit)"
	"${SSMR_COMPOSE[@]}" down -v --remove-orphans >/dev/null 2>&1 || true

	ssmr_compose_up_retry 5 3 spanner-emulator redis optimizer-core
	ssmr_compose_up_retry 5 3 zookeeper
	ssmr_wait_tcp "Zookeeper" "localhost" "22181" 60 2
	sleep 3
	ssmr_compose_up_retry 8 4 kafka
	"${SSMR_COMPOSE[@]}" up -d backend-setup kafka-ui

	ssmr_wait_tcp "Spanner emulator" "$SSMR_SPANNER_HOST" "$SSMR_SPANNER_PORT" 60 2
	ssmr_wait_tcp "Redis" "$SSMR_REDIS_HOST" "$SSMR_REDIS_PORT" 30 2

	local kafka_attempt=0
	while (( kafka_attempt < 40 )); do
		kafka_attempt=$((kafka_attempt + 1))
		if "${SSMR_COMPOSE[@]}" ps --status running --format json kafka 2>/dev/null | grep -q '"Health":"healthy"'; then
			break
		fi
		if ssmr_tcp_ready "$SSMR_KAFKA_HOST" "$SSMR_KAFKA_PORT" && \
			"${SSMR_COMPOSE[@]}" exec -T kafka kafka-topics --bootstrap-server localhost:9094 --list >/dev/null 2>&1; then
			break
		fi
		if (( kafka_attempt >= 40 )); then
			echo "Timed out waiting for Kafka" >&2
			return 1
		fi
		sleep 3
	done

	ssmr_log "Kafka topic bootstrap"
	"${SSMR_COMPOSE[@]}" run --rm kafka-init

	ssmr_log "Spanner schema + seed"
	"${SSMR_COMPOSE[@]}" run --rm backend-setup

	ssmr_log "Starting backend-go and ai-worker"
	"${SSMR_COMPOSE[@]}" up -d backend-go ai-worker

	ssmr_wait_http "backend health" "$SSMR_HEALTH_URL" 90 2
	ssmr_log "SSMR stack ready"
}

ssmr_supplier_session_cookie() {
	local phone="${SSMR_SMOKE_SUPPLIER_PHONE:-+998901000001}"
	local password="${SSMR_SMOKE_SUPPLIER_PASSWORD:-SmokeTest!234}"
	local base="${PUBLIC_BASE_URL%/}"
	local hdr
	hdr="$(mktemp)"
	local status
	status="$(curl -sS -D "$hdr" -o /dev/null -w '%{http_code}' -X POST "$base/v1/auth/supplier/login" \
		-H 'Content-Type: application/json' \
		-d "{\"phone\":\"$phone\",\"password\":\"$password\"}")"
	if [[ "$status" != "200" ]]; then
		rm -f "$hdr"
		echo "supplier login failed: HTTP $status" >&2
		return 1
	fi
	local cookie
	cookie="$(grep -i '^set-cookie:' "$hdr" | head -1 | sed -E 's/^[Ss]et-[Cc]ookie:[[:space:]]*([^;]+).*/\1/')"
	rm -f "$hdr"
	if [[ -z "$cookie" ]]; then
		echo "supplier login missing session cookie" >&2
		return 1
	fi
	printf '%s' "$cookie"
}

ssmr_seed_planning_e2e() {
	ssmr_log "Seeding planning baseline via ssmr-smokecheck e2e (PX90/PX91 markers)"
	mkdir -p "$SSMR_GO_TMP_ROOT/go-build" "$SSMR_REPO_ROOT/.ssmr/import-uploads"
	(
		cd "$SSMR_REPO_ROOT/apps/backend-go"
		export SSMR_IMPORT_LOCAL_ROOT="$SSMR_REPO_ROOT/.ssmr/import-uploads"
		TMPDIR="$SSMR_GO_TMP_ROOT" \
		GOCACHE="$SSMR_GO_TMP_ROOT/go-build" \
		GOFLAGS="${GOFLAGS:-} -buildvcs=false" \
		go run ./cmd/ssmr-smokecheck e2e
	)
}

ssmr_seed_planning_baseline_min() {
	ssmr_log "Ensuring DemandForecastBaseline seed row (PX-PROD-3 local export)"
	mkdir -p "$SSMR_GO_TMP_ROOT/go-build"
	(
		cd "$SSMR_REPO_ROOT/apps/backend-go"
		TMPDIR="$SSMR_GO_TMP_ROOT" \
		GOCACHE="$SSMR_GO_TMP_ROOT/go-build" \
		GOFLAGS="${GOFLAGS:-} -buildvcs=false" \
		go run ./cmd/ssmr-smokecheck planning-baseline-seed
	)
}

ssmr_kafka_publish() {
	local topic=$1
	local payload=$2
	printf '%s\n' "$payload" | "${SSMR_COMPOSE[@]}" exec -T kafka \
		kafka-console-producer --bootstrap-server localhost:9094 --topic "$topic"
}

ssmr_kafka_topic_message_count() {
	local topic=$1
	local count
	count="$("${SSMR_COMPOSE[@]}" exec -T kafka kafka-run-class kafka.tools.GetOffsetShell \
		--broker-list localhost:9094 --topic "$topic" --time -1 2>/dev/null | awk -F: '{sum+=$3} END {print sum+0}')"
	printf '%s' "${count:-0}"
}
