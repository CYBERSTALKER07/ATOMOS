#!/usr/bin/env bash
# ADR-009 fiscal hard-gate SSMR markers.
# Starts the isolated SSMR stack (same as lifecycle), runs fiscal e2e, gates markers.
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
MARKER_MANIFEST="$REPO_ROOT/contracts/ssmr_fiscal_markers.json"

if [[ ! -f "$COMPOSE_FILE" ]]; then
  echo "Missing compose file: $COMPOSE_FILE" >&2
  exit 1
fi
if [[ ! -f "$ENV_FILE" ]]; then
  echo "Missing env file: $ENV_FILE" >&2
  exit 1
fi
if ! command -v docker >/dev/null 2>&1; then
  echo "docker is required for fiscal SSMR" >&2
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
export FISCAL_PROVIDER="${FISCAL_PROVIDER:-FAKE}"
export PUBLIC_BASE_URL="${PUBLIC_BASE_URL:-http://localhost:8180}"

COMPOSE=(docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE")
HEALTH_URL="${PUBLIC_BASE_URL%/}/v1/health"

CURRENT_STEP="init"
cleanup() {
  local exit_code=$?
  trap - EXIT INT TERM
  if (( exit_code != 0 )); then
    echo "Fiscal smoke failed at step: ${CURRENT_STEP} (exit ${exit_code})" >&2
    "${COMPOSE[@]}" ps >&2 || true
    "${COMPOSE[@]}" logs --tail 80 spanner-emulator redis kafka backend-setup backend-go >&2 || true
  fi
  if [[ "${SSMR_FISCAL_KEEP_UP:-0}" != "1" ]]; then
    "${COMPOSE[@]}" down -v --remove-orphans >/dev/null 2>&1 || true
  fi
  exit "$exit_code"
}
trap cleanup EXIT INT TERM

CURRENT_STEP="compose-up"
echo "==> ssmr fiscal: starting stack"
"${COMPOSE[@]}" up -d --build

CURRENT_STEP="wait-health"
echo "==> waiting for $HEALTH_URL"
for i in $(seq 1 90); do
  if curl -sf "$HEALTH_URL" >/dev/null 2>&1; then
    echo "backend healthy"
    break
  fi
  if (( i == 90 )); then
    echo "backend health timeout" >&2
    exit 1
  fi
  sleep 2
done

CURRENT_STEP="fiscal-e2e"
LOG="$(mktemp -t ssmr-fiscal.XXXXXX.log)"
echo "==> running fiscal e2e (FISCAL_PROVIDER=$FISCAL_PROVIDER)"
(
  cd "$REPO_ROOT/apps/backend-go"
  go run ./cmd/ssmr-smokecheck fiscal
) 2>&1 | tee "$LOG"

CURRENT_STEP="marker-gate"
if [[ -f "$MARKER_MANIFEST" ]]; then
  python3 - "$MARKER_MANIFEST" "$LOG" <<'PY'
import json, sys
markers = json.load(open(sys.argv[1]))["required"]
text = open(sys.argv[2]).read()
missing = [m for m in markers if m not in text]
if missing:
    print("MISSING FISCAL MARKERS:", ", ".join(missing), file=sys.stderr)
    sys.exit(1)
print("__SSMR_FISCAL_OK__")
for m in markers:
    print(m)
PY
fi

echo "Fiscal hard-gate SSMR green."
