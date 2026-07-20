# D5 — Managed Kafka (Confluent Cloud) runbook

**Project:** `void-494000` · **Region align:** `asia-south1` (same as GKE/Spanner)  
**Status:** **Prepared** — GSM topic names live; bootstrap still **placeholder** until you create a Confluent cluster.

Kafka is **outside GCP Console** (Confluent Cloud UI or CLI). App is broker-agnostic: bootstrap + SASL/TLS.

---

## Why not automatic apply?

- No Confluent account/CLI login on this machine  
- Creating a paid/trial cluster needs **your** Confluent identity  
- Current GSM bootstrap: `pkc-xxxxx.asia-south1.gcp.confluent.cloud:9092` (fake)

---

## A. Create cluster (Confluent Cloud Console) — ~10 min

1. Open https://confluent.cloud/ and sign up / log in  
2. **Add cloud environment** → create **Basic** cluster (pilot)  
3. **Cloud provider:** Google Cloud · **Region:** `asia-south1` (Mumbai) if available; else nearest GCP region  
4. Copy **Bootstrap server** (looks like `pkc-xxxxx.asia-south1.gcp.confluent.cloud:9092`)  
5. **API keys → Create** (scope: cluster) → save **Key** + **Secret** once  

### CLI alternative (after `brew install confluent-cli`)

```bash
confluent login
confluent environment create pegasusx-staging
confluent kafka cluster create pegasusx-staging \
  --cloud gcp --region asia-south1 --type basic
confluent kafka cluster describe
confluent api-key create --resource <cluster-id>
```

---

## B. Create topics (3 partitions pilot)

| Topic | Purpose |
|-------|---------|
| `staging.events.orders` | Main outbox / order + state events (**TopicMain**) |
| `staging.events.orders-dlq` | DLQ for main |
| `staging.events.spatial` | Spatial / H3 fanout |
| `staging.events.realtime` | Realtime / fleet |
| `staging.events.webhooks` | Outbound webhooks |
| `staging.events.freeze-locks` | Shift / freeze locks |
| `staging.events.inventory-import` | Inventory import |
| `planning.signal.ingest.v1` | Planning brain (optional pilot) |
| `planning.forecast.request.v1` | optional |
| `planning.forecast.result.v1` | optional |

**Partitions:** 3 · **Cleanup:** delete or compact per Confluent defaults (delete is fine for pilot).

Console: Cluster → **Topics → Add topic**  
CLI:

```bash
for t in \
  staging.events.orders \
  staging.events.orders-dlq \
  staging.events.spatial \
  staging.events.realtime \
  staging.events.webhooks \
  staging.events.freeze-locks \
  staging.events.inventory-import
do
  confluent kafka topic create "$t" --partitions 3
done
```

---

## C. Wire into Google Secret Manager

```bash
export KAFKA_BOOTSTRAP="pkc-REAL....asia-south1.gcp.confluent.cloud:9092"
export KAFKA_SASL_USERNAME="<api-key>"
export KAFKA_SASL_PASSWORD="<api-secret>"
cd pegasusX
bash scripts/phase0_wire_kafka_confluent.sh
```

Also update `infra/terraform/staging.tfvars`:

```hcl
kafka_bootstrap_servers = "pkc-REAL....:9092"
```

(Topic names already match staging defaults.)

---

## D. Consumer groups (app-owned, no Confluent pre-create)

| Group ID | Role |
|----------|------|
| `void-order-mutator` | Order state machine consumer |
| `void-notification-dispatcher` | Fanout / notifications |
| `void-warehouse-mutator` | Warehouse events |
| `pegasusx-ai-worker` | AI worker (if used) |

---

## E. App env (D9)

```
KAFKA_BROKERS=<bootstrap>
KAFKA_SASL_MECHANISM=PLAIN
KAFKA_SECURITY_PROTOCOL=SASL_SSL
KAFKA_SASL_USERNAME=<from GSM>
KAFKA_SASL_PASSWORD=<from GSM>
KAFKA_TOPIC_MAIN=staging.events.orders
REQUIRE_INFRA_ADAPTERS=true
```

---

## F. Prove checklist

- [ ] Bootstrap is **not** `pkc-xxxxx`  
- [ ] Topics listed in Confluent UI  
- [ ] GSM secrets updated via `phase0_wire_kafka_confluent.sh`  
- [ ] Produce test: Confluent UI **Messages** or `kafkacat`/`confluent kafka topic produce`  
- [ ] After D9: outbox publish → message on `staging.events.orders`  
- [ ] Consumer group lag metrics after first traffic  

---

## Cost note

Confluent Basic ~$120–200/mo pilot (see `CLOUD_BUDGET_MODEL.md`). Use free credits if offered. Tear down cluster when staging idle for weeks.

---

## Already in GCP (D2)

| GSM secret | Current value |
|------------|---------------|
| `pegasusx-staging-kafka-bootstrap-servers` | placeholder until wire script |
| `pegasusx-staging-kafka-topic-orders` | `staging.events.orders` |
| `pegasusx-staging-kafka-topic-spatial` | `staging.events.spatial` |
| `pegasusx-staging-kafka-topic-realtime` | `staging.events.realtime` |
| `pegasusx-staging-kafka-topic-webhooks` | `staging.events.webhooks` |

SASL secrets created empty/on wire: `pegasusx-staging-kafka-sasl-username`, `pegasusx-staging-kafka-sasl-password`.
