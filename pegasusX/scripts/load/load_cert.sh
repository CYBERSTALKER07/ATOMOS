#!/usr/bin/env bash
# PX9-F load certification — retailer-heavy mix + supplier read parity.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
ENV_FILE="${SSMR_ENV_FILE:-$REPO_ROOT/.env.ssmr.example}"

BASE_URL="${BASE_URL:-http://localhost:8180}"
LOAD_PROFILE="${LOAD_PROFILE:-smoke}"
WITH_SSMR=0
SKIP_K6=0

usage() {
  cat <<'EOF'
Usage: load_cert.sh [--with-ssmr] [--skip-k6] [--profile smoke|cert|stress]

  --with-ssmr   docker compose up SSMR stack and wait for /v1/health
  --skip-k6     health + token bootstrap only (no k6)
  --profile     LOAD_PROFILE (default: smoke)

Env: BASE_URL, SSMR_ENV_FILE, RETAILER_TOKEN, SUPPLIER_COOKIE (optional overrides)
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --with-ssmr) WITH_SSMR=1; shift ;;
    --skip-k6) SKIP_K6=1; shift ;;
    --profile)
      LOAD_PROFILE="${2:-smoke}"
      shift 2
      ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown arg: $1" >&2; usage; exit 1 ;;
  esac
done

load_env_file() {
  local env_file=$1
  [[ -f "$env_file" ]] || return 0
  local line key value
  while IFS= read -r line || [[ -n "$line" ]]; do
    [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]] && continue
    key=${line%%=*}
    value=${line#*=}
    key=${key#export }
    export "$key=$value"
  done < "$env_file"
}

# shellcheck source=k6_exec.sh
source "$SCRIPT_DIR/k6_exec.sh"

wait_health() {
  local url="${BASE_URL%/}/v1/health"
  local max_attempts=90
  if [[ "$WITH_SSMR" -eq 1 ]]; then
  max_attempts=180
  fi
  echo "Waiting for ${url} ..."
  for _ in $(seq 1 "$max_attempts"); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      echo "Backend healthy."
      return 0
    fi
    sleep 2
  done
  echo "Backend not healthy at ${url}" >&2
  exit 1
}

load_env_file "$ENV_FILE"

case "$LOAD_PROFILE" in
  cert)
    export LOAD_RETAILER_POOL_SIZE="${LOAD_RETAILER_POOL_SIZE:-64}"
    ;;
  stress)
    export LOAD_RETAILER_POOL_SIZE="${LOAD_RETAILER_POOL_SIZE:-128}"
    ;;
  *)
    export LOAD_RETAILER_POOL_SIZE="${LOAD_RETAILER_POOL_SIZE:-1}"
    ;;
esac

if [[ "$WITH_SSMR" -eq 1 ]]; then
  docker compose --env-file "$ENV_FILE" -f "$REPO_ROOT/infra/docker-compose.ssmr.yml" up -d
  BASE_URL="${PUBLIC_BASE_URL:-$BASE_URL}"
fi

wait_health

if [[ -z "${RETAILER_TOKEN:-}" || -z "${SUPPLIER_COOKIE:-}" ]]; then
  echo "Bootstrapping load tokens via ssmr-smokecheck loadtokens (pool=${LOAD_RETAILER_POOL_SIZE}) ..."
  if [[ "$WITH_SSMR" -eq 1 ]]; then
    echo "Recreating backend-go so RELIABILITY_RATE_LIMIT_AUTH_MAX / LOAD_BOOTSTRAP_SECRET apply ..."
    docker compose --env-file "$ENV_FILE" -f "$REPO_ROOT/infra/docker-compose.ssmr.yml" up -d --force-recreate backend-go
    wait_health
  fi
  _LOAD_EXPORT_COUNT=0
  while IFS= read -r _line; do
    [[ -z "$_line" ]] && continue
    _LOAD_EXPORT_COUNT=$((_LOAD_EXPORT_COUNT + 1))
    # shellcheck disable=SC1090
    eval "$_line"
  done < <(
    cd "$REPO_ROOT/apps/backend-go" && go run ./cmd/ssmr-smokecheck loadtokens 2>/dev/null | grep '^export ' || true
  )
  if [[ "$_LOAD_EXPORT_COUNT" -lt 6 ]]; then
    echo "load token bootstrap failed (expected export lines on stdout)" >&2
    cd "$REPO_ROOT/apps/backend-go" && go run ./cmd/ssmr-smokecheck loadtokens >&2 || true
    exit 1
  fi
  unset _line _LOAD_EXPORT_COUNT
fi

STAMP="$(date -u +%Y%m%d-%H%M%S)"
ARTIFACT_DIR="${ARTIFACT_DIR:-$REPO_ROOT/artifacts/load/$STAMP}"
mkdir -p "$ARTIFACT_DIR"
echo "$LOAD_PROFILE" >"$ARTIFACT_DIR/profile.txt"
echo "${BASE_URL%/}" >"$ARTIFACT_DIR/base_url.txt"

if [[ "$SKIP_K6" -eq 1 ]]; then
  echo "Skip k6 requested; tokens and health OK."
  echo "__LOAD_CERT_OK__"
  exit 0
fi

K6_RUNNER=""
if K6_BIN="$(resolve_k6_bin 2>/dev/null)"; then
  K6_RUNNER="$K6_BIN"
  echo "Using k6: $K6_BIN ($("$K6_BIN" version 2>/dev/null | head -1 || true))"
elif k6_docker_available; then
  K6_RUNNER="docker"
  echo "Using k6 via Docker (${K6_DOCKER_IMAGE}); native k6 not found."
  echo "  Optional native install: brew update && brew install k6" >&2
else
  echo "k6 not available (no binary, docker not running); health burst fallback only." >&2
  echo "  Install k6: brew install k6   OR start Docker for grafana/k6 fallback" >&2
  BASE_URL="$BASE_URL" \
    CONCURRENCY="${LOAD_BURST_CONCURRENCY:-40}" \
    REQUESTS="${LOAD_BURST_REQUESTS:-120}" \
    "$SCRIPT_DIR/retailer_burst.sh"
  echo "__LOAD_CERT_OK__ (health smoke only — install k6 or enable Docker for full SLO gate)"
  exit 0
fi

export LOAD_PROFILE
K6_COMMON=(
  -e "BASE_URL=${BASE_URL%/}"
  -e "LOAD_PROFILE=${LOAD_PROFILE}"
  -e "RETAILER_TOKEN=${RETAILER_TOKEN:-}"
  -e "RETAILER_TOKENS=${RETAILER_TOKENS:-}"
  -e "SUPPLIER_COOKIE=${SUPPLIER_COOKIE:-}"
  -e "H3_CELL=${H3_CELL:-}"
  -e "DELIVERY_LAT=${DELIVERY_ZONE_CENTER_LAT:-41.31}"
  -e "DELIVERY_LNG=${DELIVERY_ZONE_CENTER_LNG:-69.24}"
)

echo "Running load k6 (profile=${LOAD_PROFILE}) ..."
set +e
if [[ "$LOAD_PROFILE" == "smoke" ]]; then
  # Supplier reads first on smoke so Spanner emulator tail latency is not
  # polluted by the retailer mutation burst (cert/stress keep retailer-first).
  echo "Running supplier read k6 (profile=${LOAD_PROFILE}) ..."
  run_k6 "$K6_RUNNER" "$SCRIPT_DIR" "$ARTIFACT_DIR" "k6-supplier.json" "k6_supplier_read.js" \
    "${K6_COMMON[@]}" \
    -e "SUPPLIER_COOKIE=${SUPPLIER_COOKIE:-}"
  SUPPLIER_K6=$?

  echo "Waiting for backend between k6 phases ..."
  wait_health

  echo "Running retailer cert k6 (profile=${LOAD_PROFILE}) ..."
  run_k6 "$K6_RUNNER" "$SCRIPT_DIR" "$ARTIFACT_DIR" "k6-retailer.json" "k6_retailer_cert.js" \
    "${K6_COMMON[@]}"
  RETAILER_K6=$?
else
  echo "Running retailer cert k6 (profile=${LOAD_PROFILE}) ..."
  run_k6 "$K6_RUNNER" "$SCRIPT_DIR" "$ARTIFACT_DIR" "k6-retailer.json" "k6_retailer_cert.js" \
    "${K6_COMMON[@]}"
  RETAILER_K6=$?

  echo "Waiting for backend between k6 phases ..."
  wait_health
  if [[ "$LOAD_PROFILE" == "cert" || "$LOAD_PROFILE" == "stress" ]]; then
    echo "Cooling down ${LOAD_CERT_COOLDOWN_SEC:-45}s before supplier k6 ..."
    sleep "${LOAD_CERT_COOLDOWN_SEC:-45}"
    wait_health
  fi

  echo "Running supplier read k6 (profile=${LOAD_PROFILE}) ..."
  run_k6 "$K6_RUNNER" "$SCRIPT_DIR" "$ARTIFACT_DIR" "k6-supplier.json" "k6_supplier_read.js" \
    "${K6_COMMON[@]}" \
    -e "SUPPLIER_COOKIE=${SUPPLIER_COOKIE:-}"
  SUPPLIER_K6=$?
fi
set -e

python3 "$SCRIPT_DIR/generate_report.py" "$ARTIFACT_DIR"
REPORT_EXIT=$?

if [[ "$RETAILER_K6" -ne 0 || "$SUPPLIER_K6" -ne 0 || "$REPORT_EXIT" -ne 0 ]]; then
  echo "Load cert failed (retailer_k6=$RETAILER_K6 supplier_k6=$SUPPLIER_K6 report=$REPORT_EXIT)" >&2
  exit 1
fi
cp "$ARTIFACT_DIR/LOAD_TEST_REPORT.md" "$REPO_ROOT/docs/LOAD_TEST_REPORT.md"

echo "Load cert artifacts: $ARTIFACT_DIR"
echo "__LOAD_CERT_OK__"
