#!/usr/bin/env bash
# D5: wire Google Cloud Managed Service for Apache Kafka into GSM for staging.
#
# Prerequisites:
#   gcloud managed-kafka clusters create ... (see docs/GCP_MANAGED_KAFKA.md)
#   topics created
#
# Usage:
#   bash scripts/phase0_wire_kafka_gcp_managed.sh
#   # or override:
#   CLUSTER=pegasusx-staging-kafka LOCATION=asia-south1 PROJECT=pegasus-503013 \
#     bash scripts/phase0_wire_kafka_gcp_managed.sh
set -euo pipefail

PROJECT="${PROJECT:-pegasus-503013}"
LOCATION="${LOCATION:-asia-south1}"
CLUSTER="${CLUSTER:-pegasusx-staging-kafka}"
PREFIX="${SECRET_PREFIX:-pegasusx-staging}"

TOPIC_MAIN="${KAFKA_TOPIC_MAIN:-staging.events.orders}"
TOPIC_SPATIAL="${KAFKA_TOPIC_SPATIAL:-staging.events.spatial}"
TOPIC_REALTIME="${KAFKA_TOPIC_REALTIME:-staging.events.realtime}"
TOPIC_WEBHOOKS="${KAFKA_TOPIC_WEBHOOKS:-staging.events.webhooks}"
TOPIC_FREEZE="${KAFKA_TOPIC_FREEZE_LOCKS:-staging.events.freeze-locks}"
TOPIC_INVENTORY="${KAFKA_TOPIC_INVENTORY_IMPORT:-staging.events.inventory-import}"
TOPIC_DLQ="${KAFKA_TOPIC_MAIN_DLQ:-${TOPIC_MAIN}-dlq}"

echo "==> describe cluster $CLUSTER @ $LOCATION"
BOOT=$(gcloud managed-kafka clusters describe "$CLUSTER" \
	--location="$LOCATION" --project="$PROJECT" \
	--format='value(bootstrapAddress)')
STATE=$(gcloud managed-kafka clusters describe "$CLUSTER" \
	--location="$LOCATION" --project="$PROJECT" \
	--format='value(state)')

if [[ -z "$BOOT" ]]; then
	echo "FAIL: empty bootstrapAddress" >&2
	exit 1
fi
echo "state=$STATE bootstrap=$BOOT"

put() {
	local name=$1
	local value=$2
	if gcloud secrets describe "$name" --project="$PROJECT" >/dev/null 2>&1; then
		printf '%s' "$value" | gcloud secrets versions add "$name" --project="$PROJECT" --data-file=- >/dev/null
		echo "OK  version+ $name"
	else
		printf '%s' "$value" | gcloud secrets create "$name" \
			--project="$PROJECT" --replication-policy=automatic --data-file=- >/dev/null
		echo "OK  created $name"
	fi
}

put "${PREFIX}-kafka-bootstrap-servers" "$BOOT"
put "${PREFIX}-kafka-auth-mode" "GCP_MANAGED_OAUTH"
put "${PREFIX}-kafka-security-protocol" "SASL_SSL"
put "${PREFIX}-kafka-sasl-mechanism" "OAUTHBEARER"
put "${PREFIX}-kafka-topic-orders" "$TOPIC_MAIN"
put "${PREFIX}-kafka-topic-spatial" "$TOPIC_SPATIAL"
put "${PREFIX}-kafka-topic-realtime" "$TOPIC_REALTIME"
put "${PREFIX}-kafka-topic-webhooks" "$TOPIC_WEBHOOKS"
put "${PREFIX}-kafka-topic-freeze-locks" "$TOPIC_FREEZE"
put "${PREFIX}-kafka-topic-inventory-import" "$TOPIC_INVENTORY"
put "${PREFIX}-kafka-topic-main-dlq" "$TOPIC_DLQ"

# IAM: backend runtime SA needs Managed Kafka Client (+ token creator for OIDC)
PROJECT_NUMBER=$(gcloud projects describe "$PROJECT" --format='value(projectNumber)')
BACKEND_SA="staging-backend@${PROJECT}.iam.gserviceaccount.com"
for role in roles/managedkafka.client roles/iam.serviceAccountTokenCreator roles/iam.serviceAccountOpenIdTokenCreator; do
	gcloud projects add-iam-policy-binding "$PROJECT" \
		--member="serviceAccount:${BACKEND_SA}" \
		--role="$role" \
		--condition=None \
		--quiet >/dev/null 2>&1 || \
	gcloud projects add-iam-policy-binding "$PROJECT" \
		--member="serviceAccount:${BACKEND_SA}" \
		--role="$role" \
		--quiet >/dev/null
	echo "OK  IAM $role → $BACKEND_SA"
done

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
{
	echo ""
	echo "# D5 GCP Managed Kafka ($(date -u +%Y-%m-%dT%H:%MZ))"
	echo "KAFKA_BROKERS=${BOOT}"
	echo "KAFKA_AUTH_MODE=GCP_MANAGED_OAUTH"
	echo "KAFKA_SECURITY_PROTOCOL=SASL_SSL"
	echo "KAFKA_SASL_MECHANISM=OAUTHBEARER"
	echo "KAFKA_TOPIC_MAIN=${TOPIC_MAIN}"
	echo "KAFKA_TOPIC_SPATIAL=${TOPIC_SPATIAL}"
	echo "KAFKA_TOPIC_REALTIME=${TOPIC_REALTIME}"
	echo "KAFKA_TOPIC_WEBHOOKS=${TOPIC_WEBHOOKS}"
	echo "KAFKA_TOPIC_FREEZE_LOCKS=${TOPIC_FREEZE}"
	echo "KAFKA_TOPIC_INVENTORY_IMPORT=${TOPIC_INVENTORY}"
	echo "KAFKA_TOPIC_MAIN_DLQ=${TOPIC_DLQ}"
	echo "# App must use Workload Identity SA with roles/managedkafka.client"
} >>"$ROOT/.env.k8s.generated"

# staging.tfvars bootstrap for docs/consistency (non-secret URL)
if [[ -f "$ROOT/infra/terraform/staging.tfvars" ]]; then
	if grep -q 'kafka_bootstrap_servers' "$ROOT/infra/terraform/staging.tfvars"; then
		python3 - "$ROOT/infra/terraform/staging.tfvars" "$BOOT" <<'PY'
from pathlib import Path
import re, sys
p = Path(sys.argv[1])
boot = sys.argv[2]
t = p.read_text()
t2 = re.sub(
    r'kafka_bootstrap_servers\s*=\s*"[^"]*"',
    f'kafka_bootstrap_servers = "{boot}"',
    t,
    count=1,
)
p.write_text(t2)
print("OK  staging.tfvars kafka_bootstrap_servers updated")
PY
	fi
fi

echo "phase0-wire-kafka-gcp-ok bootstrap=$BOOT state=$STATE"
echo "NOTE: Go clients currently use plain segmentio dialer — OAUTHBEARER support is required before REQUIRE_INFRA_ADAPTERS=true (see docs/GCP_MANAGED_KAFKA.md)"
