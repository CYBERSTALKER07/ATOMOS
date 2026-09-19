# GCP Managed Service for Apache Kafka (HA event backbone)

Staging already runs on Managed Kafka (`KAFKA_AUTH_MODE=GCP_MANAGED_OAUTH`).
Production follows the same path. Local Docker Kafka stays single-broker RF=1
by design; cluster HA is Strimzi (dev fallback) or Managed Kafka (staging/prod).

## Topology

| Environment | Broker | RF / min.ISR | Auth |
|-------------|--------|--------------|------|
| Local compose / SSMR Docker | 1× Confluent cp-kafka | 1 / 1 | plaintext |
| Dev/local k8s fallback | Strimzi 3 dual-role ([infra/k8s/kafka.yaml](../infra/k8s/kafka.yaml)) | 3 / 2 | plaintext or TLS |
| Staging / Prod | GCP Managed Kafka ([infra/terraform/kafka.tf](../infra/terraform/kafka.tf)) | 3 / 2 | `GCP_MANAGED_OAUTH` via [kafkautil](../apps/backend-go/kafkautil/auth.go) |

Outbox poison path: Spanner `OutboxDeadLetters` (relay `RecordPublishFailures`) plus Kafka topic `*-dlq`.

## Provision (owner)

```bash
cd infra/terraform
terraform apply -var="enable_managed_kafka=true" -var-file=staging.tfvars
terraform output managed_kafka_bootstrap
terraform output managed_kafka_topics
```

Topics created at RF=3 / min.isr=2: main, main_dlq, spatial, realtime, webhooks, freeze_locks, inventory_import.

## Wire secrets + IAM

```bash
bash scripts/phase0_wire_kafka_gcp_managed.sh
# Prod: PREFIX=pegasusx-prod CLUSTER=pegasusx-prod-kafka BACKEND_SA=prod-backend@...
```

## Activate prod overlay

1. Confirm `terraform output managed_kafka_bootstrap`.
2. Uncomment the `configMapGenerator` block in [infra/k8s/overlays/prod/kustomization.yaml](../infra/k8s/overlays/prod/kustomization.yaml) and replace `CLUSTER_ID`.
3. `kubectl apply -k infra/k8s/overlays/prod`

## Verify (CI)

```bash
make kafka-ha-gate   # → kafka-ha-gate-ok
```
