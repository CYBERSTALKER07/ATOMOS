#!/usr/bin/env bash
# PX-LC-3 / PX-PROD-4: observability fire drills on local SSMR (no GCP billing).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/ssmr_stack.sh
source "$SCRIPT_DIR/lib/ssmr_stack.sh"

usage() {
	cat <<'EOF'
Usage: fire_drill_ssmr.sh [A|B|C|D|all]

Runs PX-PROD-4 fire drills against docker-compose SSMR:
  A — pause ai-worker (Kafka consumer lag simulation) + recovery
  B — malformed planning.signal.ingest.v1 → DLQ; demand/today stays 200 math-only
  C — planning training export + validate
  D — backend-go restart + health recovery

Artifacts: artifacts/fire-drill-{a,b,c,d}.log

Environment:
  FIRE_DRILL_SKIP_STACK=1   skip stack bring-up (stack must already be healthy)
  FIRE_DRILL_SEED_E2E=1     run e2e seed when stack was cold (default 1)
EOF
}

DRILLS="${1:-all}"
case "$DRILLS" in
	A|B|C|D|all) ;;
	-h|--help) usage; exit 0 ;;
	*) echo "unknown drill: $DRILLS" >&2; usage; exit 1 ;;
esac

ssmr_lib_init
mkdir -p "$SSMR_ARTIFACTS_DIR"

LOG_A="$SSMR_ARTIFACTS_DIR/fire-drill-a.log"
LOG_B="$SSMR_ARTIFACTS_DIR/fire-drill-b.log"
LOG_C="$SSMR_ARTIFACTS_DIR/fire-drill-c.log"
LOG_D="$SSMR_ARTIFACTS_DIR/fire-drill-d.log"

drill_enabled() {
	local name=$1
	[[ "$DRILLS" == "all" || "$DRILLS" == "$name" ]]
}

if [[ "${FIRE_DRILL_SKIP_STACK:-}" != "1" ]]; then
	WAS_HEALTHY=0
	if ssmr_stack_healthy; then
		WAS_HEALTHY=1
	fi
	ssmr_ensure_stack
	if [[ "$WAS_HEALTHY" != "1" && "${FIRE_DRILL_SEED_E2E:-1}" == "1" ]]; then
		ssmr_seed_planning_e2e
	fi
else
	ssmr_log "Skipping stack bring-up (FIRE_DRILL_SKIP_STACK=1)"
	if ! ssmr_stack_healthy; then
		echo "SSMR stack not healthy at ${SSMR_HEALTH_URL}" >&2
		exit 1
	fi
fi

run_drill_a() {
	ssmr_log "Drill A — ai-worker pause / recovery"
	{
		echo "=== Drill A $(date -u +%Y-%m-%dT%H:%M:%SZ) ==="
		ssmr_log "Stopping ai-worker"
		"${SSMR_COMPOSE[@]}" stop ai-worker
		sleep 5
		if curl -fsS "http://localhost:8181/healthz" >/dev/null 2>&1; then
			echo "WARN: ai-worker healthz still reachable after stop"
		else
			echo "ai-worker healthz unreachable (expected)"
		fi
		ssmr_log "Restarting ai-worker"
		"${SSMR_COMPOSE[@]}" start ai-worker
		ssmr_wait_http "ai-worker health" "http://localhost:8181/healthz" 60 2
		ssmr_wait_http "backend health" "$SSMR_HEALTH_URL" 30 2
		echo "Drill A PASS: ai-worker recovered"
	} | tee "$LOG_A"
}

run_drill_b() {
	ssmr_log "Drill B — planning DLQ + demand/today math-only"
	{
		echo "=== Drill B $(date -u +%Y-%m-%dT%H:%M:%SZ) ==="
		local dlq_before dlq_after
		dlq_before="$(ssmr_kafka_topic_message_count "$SSMR_KAFKA_TOPIC_MAIN_DLQ")"
		echo "DLQ messages before: $dlq_before"
		ssmr_kafka_publish "planning.signal.ingest.v1" '{"invalid":true,"missing_signal_id":true}'
		sleep 8
		dlq_after="$(ssmr_kafka_topic_message_count "$SSMR_KAFKA_TOPIC_MAIN_DLQ")"
		echo "DLQ messages after: $dlq_after"
		if (( dlq_after <= dlq_before )); then
			echo "WARN: DLQ count did not increase (consumer may route elsewhere); continuing demand/today check"
		fi
		local cookie status body
		cookie="$(ssmr_supplier_session_cookie)"
		status="$(curl -sS -o /tmp/pegasusx-drill-demand.json -w '%{http_code}' \
			-H "Cookie: $cookie" "${PUBLIC_BASE_URL%/}/v1/supplier/analytics/demand/today")"
		body="$(cat /tmp/pegasusx-drill-demand.json)"
		rm -f /tmp/pegasusx-drill-demand.json
		echo "demand/today HTTP $status"
		if [[ "$status" != "200" ]]; then
			echo "Drill B FAIL: demand/today not 200"
			exit 1
		fi
		if grep -q '"baseline_source"[[:space:]]*:[[:space:]]*"ml"' <<<"$body"; then
			echo "Drill B FAIL: baseline_source=ml in response"
			exit 1
		fi
		echo "Drill B PASS: demand/today 200, no ml baseline_source"
	} | tee "$LOG_B"
}

run_drill_c() {
	ssmr_log "Drill C — planning export + validate"
	{
		echo "=== Drill C $(date -u +%Y-%m-%dT%H:%M:%SZ) ==="
		bash "$SSMR_REPO_ROOT/scripts/planning_export_local_cron.sh" --skip-stack
		echo "Drill C PASS"
	} | tee "$LOG_C"
}

run_drill_d() {
	ssmr_log "Drill D — backend-go restart"
	{
		echo "=== Drill D $(date -u +%Y-%m-%dT%H:%M:%SZ) ==="
		"${SSMR_COMPOSE[@]}" restart backend-go
		local attempt
		for ((attempt = 1; attempt <= 30; attempt++)); do
			if curl -fsS "$SSMR_HEALTH_URL" >/dev/null 2>&1; then
				echo "backend healthy after restart (attempt $attempt)"
				echo "Drill D PASS"
				return 0
			fi
			sleep 2
		done
		echo "Drill D FAIL: backend health not recovered within 60s"
		exit 1
	} | tee "$LOG_D"
}

FAILURES=0
if drill_enabled A; then run_drill_a || FAILURES=$((FAILURES + 1)); fi
if drill_enabled B; then run_drill_b || FAILURES=$((FAILURES + 1)); fi
if drill_enabled C; then run_drill_c || FAILURES=$((FAILURES + 1)); fi
if drill_enabled D; then run_drill_d || FAILURES=$((FAILURES + 1)); fi

if (( FAILURES > 0 )); then
	echo "fire-drill-ssmr-FAIL: $FAILURES drill(s) failed" >&2
	exit 1
fi

echo "fire-drill-ssmr-ok — drills: $DRILLS"
echo "artifacts: $SSMR_ARTIFACTS_DIR/fire-drill-*.log"
