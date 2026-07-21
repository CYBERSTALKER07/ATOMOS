# GCP Managed Service for Apache Kafka (staging)

**Decision:** Use **Google Cloud Managed Service for Apache Kafka** (not Confluent) for `pegasus-503013`.

## Cluster

| Field | Value |
|-------|--------|
| Name | `pegasusx-staging-kafka` |
| Location | `asia-south1` |
| Capacity | 3 vCPU · 3 GiB (minimum) |
| Subnet | `projects/pegasus-503013/regions/asia-south1/subnetworks/pegasusx-staging-vpc` |
| Bootstrap | `bootstrap.pegasusx-staging-kafka.asia-south1.managedkafka.pegasus-503013.cloud.goog:9092` |

### Create (already issued)

```bash
gcloud managed-kafka clusters create pegasusx-staging-kafka \
  --project=pegasus-503013 \
  --location=asia-south1 \
  --cpu=3 \
  --memory=3GiB \
  --subnets=projects/pegasus-503013/regions/asia-south1/subnetworks/pegasusx-staging-vpc \
  --async
```

### Topics (3 partitions, RF=3)

```text
staging.events.orders
staging.events.orders-dlq
staging.events.spatial
staging.events.realtime
staging.events.webhooks
staging.events.freeze-locks
staging.events.inventory-import
```

```bash
for t in staging.events.orders staging.events.orders-dlq staging.events.spatial \
  staging.events.realtime staging.events.webhooks staging.events.freeze-locks \
  staging.events.inventory-import; do
  gcloud managed-kafka topics create "$t" \
    --cluster=pegasusx-staging-kafka --location=asia-south1 \
    --project=pegasus-503013 --partitions=3 --replication-factor=3
done
```

### Wire GSM + IAM

```bash
bash scripts/phase0_wire_kafka_gcp_managed.sh
```

## Auth model (important)

GCP Managed Kafka **does not** use Confluent API key/secret.

Supported client modes (Google docs):

1. **SASL/PLAIN + access token** (what our Go app uses via `kafkautil`)
   - `username` = service account email  
   - `password` = short-lived Google OAuth access token (ADC / Workload Identity)  
   - `security.protocol` = SASL_SSL  

2. **OAUTHBEARER** (Java library `GcpLoginCallbackHandler`) — not used by Go path yet.

Env for pods:

```text
KAFKA_AUTH_MODE=GCP_MANAGED_OAUTH
KAFKA_SASL_USERNAME=staging-backend@pegasus-503013.iam.gserviceaccount.com
KAFKA_BROKERS=bootstrap.pegasusx-staging-kafka.asia-south1.managedkafka.pegasus-503013.cloud.goog:9092
```

IAM on runtime SA (`staging-backend@pegasus-503013.iam.gserviceaccount.com`):

- `roles/managedkafka.client`
- `roles/iam.serviceAccountTokenCreator`
- `roles/iam.serviceAccountOpenIdTokenCreator`

## App wiring (done)

`apps/backend-go/kafkautil` implements SASL PLAIN + access token + TLS for:

- outbox publisher  
- consumers  
- DLQ writer  

Local SSMR Docker Kafka stays **PLAINTEXT** (`KAFKA_AUTH_MODE` empty).

## Cost note

Minimum cluster is always-on (vCPU+RAM). Watch Billing; delete when idle:

```bash
gcloud managed-kafka clusters delete pegasusx-staging-kafka \
  --location=asia-south1 --project=pegasus-503013 --async
```

## Console

[Managed Kafka clusters](https://console.cloud.google.com/managedkafka/clusters?project=pegasus-503013)
