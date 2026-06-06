# pegasusX Load Test Profile (10k Retailer Concurrency)

## Target envelope

- 10,000 concurrent retailer sessions (read-heavy).
- Mix: 60% `GET /v1/retailer/tracking`, 25% `GET /v1/retailer/cart/sync`, 15% `POST /v1/order/create` bursts.
- p99 read &lt; 300 ms (stale Spanner reads + Redis cache hits).
- p99 mutation &lt; 800 ms under burst (production / smoke gate).
- **Local SSMR `cert` profile** (emulator): split scenarios — 200 VU reads (p99 &lt; 3s), 5 VU orders for 50s hard-stop (p99 &lt; 8s, fail &lt; 15%), supplier read p99 &lt; 2.5s. Production/staging targets remain 300ms / 800ms reads+writes.
- WebSocket fan-out: Redis Pub/Sub fail-open; no pod-local-only delivery.

## Local run (PX9-F load cert)

```bash
cd pegasusX
make load-cert          # smoke profile: bootstrap tokens + k6 + report

k6 is resolved in order: native binary (`brew install k6`), then **Docker** (`grafana/k6` against `host.docker.internal:8180`). Health-only fallback if neither is available.
make load-cert-ssmr     # brings up SSMR stack, then cert gate

# Manual profiles
LOAD_PROFILE=cert bash scripts/load/load_cert.sh --with-ssmr
LOAD_PROFILE=stress VUS=1000 DURATION=120s bash scripts/load/load_cert.sh
```

Legacy health burst (no k6):

```bash
BASE_URL=http://localhost:8180 ./scripts/load/retailer_burst.sh
```

Artifacts land under `artifacts/load/<timestamp>/` and update `docs/LOAD_TEST_REPORT.md`.
Successful gate prints `__LOAD_CERT_OK__`.

**Cert profile rate limits:** `loadtokens` registers `LOAD_RETAILER_POOL_SIZE` retailers (default **64** for `cert`) so k6 VUs do not share one JWT — per-actor limits apply per retailer, not one bucket for all VUs. Supplier cert p99 read SLO is **700 ms** on local SSMR (smoke remains 400 ms).

## Production gate

- Memorystore Redis hit ratio &gt; 80% on catalog/pricing keys.
- Kafka consumer lag &lt; 10 s sustained (`void_kafka_consumer_lag_seconds`).
- Spanner CPU &lt; 70% at peak; no full-scan alerts on `Orders` or `Retailers`.
