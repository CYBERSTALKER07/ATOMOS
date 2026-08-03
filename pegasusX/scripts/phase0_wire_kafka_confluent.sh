#!/usr/bin/env bash
# D5: wire Confluent Cloud (or any SASL/TLS Kafka) into GSM for pegasusx-staging.
#
# Usage:
#   export KAFKA_BOOTSTRAP="pkc-xxxxx.asia-south1.gcp.confluent.cloud:9092"
#   export KAFKA_SASL_USERNAME="..."   # Confluent API key
#   export KAFKA_SASL_PASSWORD="..."   # Confluent API secret
#   bash scripts/phase0_wire_kafka_confluent.sh
#
# Optional overrides (defaults = staging names from D2/D5 plan):
#   KAFKA_TOPIC_MAIN, KAFKA_TOPIC_SPATIAL, KAFKA_TOPIC_REALTIME, KAFKA_TOPIC_WEBHOOKS
#   PROJECT_ID (default void-494000)
set -euo pipefail

PROJECT_ID="${PROJECT_ID:-void-494000}"
PREFIX="${SECRET_PREFIX:-pegasusx-staging}"

require() {
	if [[ -z "${!1:-}" ]]; then
		echo "FAIL: set $1" >&2
		exit 1
	fi
}

require KAFKA_BOOTSTRAP
require KAFKA_SASL_USERNAME
require KAFKA_SASL_PASSWORD

# Reject leftover plan placeholder
if [[ "$KAFKA_BOOTSTRAP" == *"pkc-xxxxx"* ]]; then
	echo "FAIL: KAFKA_BOOTSTRAP still looks like a placeholder (pkc-xxxxx)" >&2
	exit 1
fi

TOPIC_MAIN="${KAFKA_TOPIC_MAIN:-staging.events.orders}"
TOPIC_SPATIAL="${KAFKA_TOPIC_SPATIAL:-staging.events.spatial}"
TOPIC_REALTIME="${KAFKA_TOPIC_REALTIME:-staging.events.realtime}"
TOPIC_WEBHOOKS="${KAFKA_TOPIC_WEBHOOKS:-staging.events.webhooks}"
TOPIC_FREEZE="${KAFKA_TOPIC_FREEZE_LOCKS:-staging.events.freeze-locks}"
TOPIC_INVENTORY="${KAFKA_TOPIC_INVENTORY_IMPORT:-staging.events.inventory-import}"
TOPIC_DLQ="${KAFKA_TOPIC_MAIN_DLQ:-${TOPIC_MAIN}-dlq}"

put() {
	local name=$1
	local value=$2
	if gcloud secrets describe "$name" --project="$PROJECT_ID" >/dev/null 2>&1; then
		printf '%s' "$value" | gcloud secrets versions add "$name" --project="$PROJECT_ID" --data-file=- >/dev/null
		echo "OK  version+ $name"
	else
		printf '%s' "$value" | gcloud secrets create "$name" \
			--project="$PROJECT_ID" \
			--replication-policy=automatic \
			--data-file=- >/dev/null
		echo "OK  created $name"
	fi
}

echo "==> Project $PROJECT_ID prefix $PREFIX"
put "${PREFIX}-kafka-bootstrap-servers" "$KAFKA_BOOTSTRAP"
put "${PREFIX}-kafka-sasl-username" "$KAFKA_SASL_USERNAME"
put "${PREFIX}-kafka-sasl-password" "$KAFKA_SASL_PASSWORD"
put "${PREFIX}-kafka-topic-orders" "$TOPIC_MAIN"
put "${PREFIX}-kafka-topic-spatial" "$TOPIC_SPATIAL"
put "${PREFIX}-kafka-topic-realtime" "$TOPIC_REALTIME"
put "${PREFIX}-kafka-topic-webhooks" "$TOPIC_WEBHOOKS"
put "${PREFIX}-kafka-topic-freeze-locks" "$TOPIC_FREEZE"
put "${PREFIX}-kafka-topic-inventory-import" "$TOPIC_INVENTORY"
put "${PREFIX}-kafka-topic-main-dlq" "$TOPIC_DLQ"

# Operator env fragment (no passwords)
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
{
	echo ""
	echo "# D5 Kafka (wired $(date -u +%Y-%m-%dT%H:%MZ))"
	echo "KAFKA_BROKERS=${KAFKA_BOOTSTRAP}"
	echo "KAFKA_TOPIC_MAIN=${TOPIC_MAIN}"
	echo "KAFKA_TOPIC_SPATIAL=${TOPIC_SPATIAL}"
	echo "KAFKA_TOPIC_REALTIME=${TOPIC_REALTIME}"
	echo "KAFKA_TOPIC_WEBHOOKS=${TOPIC_WEBHOOKS}"
	echo "KAFKA_TOPIC_FREEZE_LOCKS=${TOPIC_FREEZE}"
	echo "KAFKA_TOPIC_INVENTORY_IMPORT=${TOPIC_INVENTORY}"
	echo "KAFKA_TOPIC_MAIN_DLQ=${TOPIC_DLQ}"
	echo "KAFKA_SASL_MECHANISM=PLAIN"
	echo "KAFKA_SECURITY_PROTOCOL=SASL_SSL"
	echo "# SASL: gcloud secrets versions access latest --secret=${PREFIX}-kafka-sasl-username --project=${PROJECT_ID}"
} >>"$ROOT/.env.k8s.generated"

echo "phase0-wire-kafka-ok bootstrap=${KAFKA_BOOTSTRAP%%.*}…"
echo "Next: create topics in Confluent (see docs / artifacts/d5-kafka-confluent-runbook.md)"
echo "Prove: produce a test message, then D9 app with REQUIRE_INFRA_ADAPTERS=true"
