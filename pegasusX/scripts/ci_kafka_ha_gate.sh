#!/usr/bin/env bash
# Fail closed if Kafka HA contracts regress: Strimzi RF≥3 / min.isr≥2,
# Managed Kafka terraform RF=3, staging on GCP_MANAGED_OAUTH, outbox DLQ present.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"
fail=0

check() {
  local desc=$1
  shift
  if "$@"; then
    echo "OK  $desc"
  else
    echo "FAIL $desc" >&2
    fail=1
  fi
}

check "Strimzi cluster replicas=3" \
  rg -q 'replicas:\s*3' infra/k8s/kafka.yaml
check "Strimzi default.replication.factor=3" \
  rg -q 'default\.replication\.factor:\s*3' infra/k8s/kafka.yaml
check "Strimzi min.insync.replicas=2" \
  rg -q 'min\.insync\.replicas:\s*2' infra/k8s/kafka.yaml

# Topic manifests must not advertise RF=1 against the HA cluster.
if rg -n 'replicas:\s*1' infra/k8s/kafka/kafka-topics.yaml infra/k8s/kafka/logistics-topics.yaml infra/k8s/kafka-topics.yaml >/tmp/kafka-rf1.txt 2>/dev/null; then
  echo "FAIL Strimzi KafkaTopic still has replicas: 1:" >&2
  cat /tmp/kafka-rf1.txt >&2
  fail=1
else
  echo "OK  Strimzi KafkaTopics RF!=1"
fi

check "Managed Kafka terraform replication_factor = 3" \
  rg -q 'replication_factor\s*=\s*3' infra/terraform/kafka.tf
check "Managed Kafka terraform min.insync.replicas = 2" \
  rg -q 'min\.insync\.replicas"\s*=\s*"2"' infra/terraform/kafka.tf
check "Managed Kafka provisions DLQ topic" \
  rg -q 'kafka_topic_main_dlq|main_dlq' infra/terraform/kafka.tf
check "Staging overlay uses GCP_MANAGED_OAUTH" \
  rg -q 'KAFKA_AUTH_MODE=GCP_MANAGED_OAUTH' infra/k8s/overlays/staging/kustomization.yaml
check "OutboxDeadLetters DDL present" \
  rg -q 'CREATE TABLE OutboxDeadLetters' apps/backend-go/schema/spanner.ddl apps/backend-go/schema/migrations/*.ddl
check "Outbox relay records dead-letters" \
  rg -q 'RecordPublishFailures' apps/backend-go/outbox/relay.go
check "kafkautil supports GCP Managed OAuth" \
  rg -q 'AuthModeGCPManaged' apps/backend-go/kafkautil/auth.go

if [[ "$fail" -ne 0 ]]; then
  echo "kafka-ha-gate FAILED" >&2
  exit 1
fi
echo "kafka-ha-gate-ok"
