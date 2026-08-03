# SSMR cloud apply — Retail OS + infra readiness (2026-08-02)

**Environment:** `pegasus-503013` / GKE `pegasusx-ssmr-gke` / ns `pegasusx-ssmr`  
**Public API:** `https://api-ssmr.pegasusx.app`  
**Profile:** `PEGASUSX_ENV=ssmr` (not production)

## Applied this session

### Spanner (`pegasusx-ssmr-spanner` / `pegasusx-ssmr-db`)

Retail OS tables applied via `cmd/apply-migration`:

| Migration | Status |
|-----------|--------|
| `20260802_retail_os_phase0.ddl` | Applied |
| `20260802_retail_os_phase2_locations.ddl` | Applied |
| `20260802_retail_os_phase3_store_stock.ddl` | Applied |
| `20260802_retail_os_phase4_pos.ddl` | Tables + indexes; receipt unique index may need re-apply if timeout |
| `20260802_retail_os_phase5_shifts.ddl` | Tables + most indexes; PosSession index may need re-apply if timeout |
| `20260802_retail_os_phase6_sections_assist.ddl` | Applied |

**Verify:**

```bash
gcloud spanner databases ddl describe pegasusx-ssmr-db \
  --instance=pegasusx-ssmr-spanner --project=pegasus-503013 \
  | grep -E 'CREATE TABLE Retailer(Users|Capability|Locations|Stock|Registers|Pos|Time|Shifts|Sections|Assistance)'
```

Logs: `artifacts/retail-os-spanner-apply.log`, `artifacts/retail-os-spanner-apply-retry.log`  
DDL snapshot before: `artifacts/spanner-ddl-before-retail-os.txt`

### Kafka (Strimzi, ns `kafka`)

Topics **READY=True** for:

- `pegasusx-main`, `pegasusx-main-dlq`, `pegasusx-orders`, `pegasusx-dispatch`, `pegasusx-realtime`, `pegasusx-webhooks`
- logistics-exceptions-v1 / logistics-telemetry-v1

Brokers: `pegasusx-kafka-bootstrap.kafka.svc.cluster.local:9092` (ConfigMap)

### Redis

Memorystore TLS — `/ready` reports `redis: ok` (no config change).

### WebSocket

- Service `backend-go-ws` → pods `app=backend-go`
- Ingress path `/v1/ws` → `backend-go-ws:80`
- `WS_ALLOWED_ORIGINS` includes api/admin/retailer hosts + local/capacitor

### Auth / webhooks (config present)

| Item | State |
|------|--------|
| Firebase auth | `FIREBASE_AUTH_ENABLED=true` |
| JWT | GSM secret `jwt-secret` |
| Global Pay webhook URL | `https://api-ssmr.pegasusx.app/v1/webhooks/global-pay` |
| Webhook secrets in ESO | GP, Stripe, Adyen, Click, Payme keys present |
| Auth bypass | `ALLOW_AUTH_BYPASS=false` |
| Memory fallback | `ALLOW_MEMORY_FALLBACK=false` |

## Still required for full Retail OS **runtime**

### 1. Backend image with Retail OS code

Live image still: `backend-go:ssmr-gap-closure-nomock4`  
→ `GET /v1/retailer/control-tower/pulse` returns **404** (routes not in that build).

**Owner/agent next (confirm before push):**

```bash
# Cloud Build + push (example tag)
# then:
kubectl -n pegasusx-ssmr set image deploy/backend-go \
  backend-go=asia-south1-docker.pkg.dev/pegasus-503013/pegasusx-ssmr-images/backend-go:ssmr-retail-os-p7
kubectl -n pegasusx-ssmr set image deploy/backend-go-worker \
  backend-go-worker=asia-south1-docker.pkg.dev/pegasus-503013/pegasusx-ssmr-images/backend-go:ssmr-retail-os-p7
kubectl -n pegasusx-ssmr rollout status deploy/backend-go
kubectl -n pegasusx-ssmr rollout status deploy/backend-go-worker
```

After rollout expect pulse → **401** without JWT (not 404).

### 2. Owner secrets (not inventable)

1. **Global Pay** real staging password → GSM → ESO refresh → register webhook  
2. **Firebase** Phone SMS + Android SHA + iOS APNs  
3. Do **not** set `PEGASUSX_ENV=production` until GP SUCCESS path proven  

### 3. Smoke after image deploy

```bash
PUBLIC_BASE_URL=https://api-ssmr.pegasusx.app bash scripts/cloud_smoke_ssmr.sh
curl -fsS https://api-ssmr.pegasusx.app/ready
# with retailer JWT:
# curl -H "Authorization: Bearer $JWT" https://api-ssmr.pegasusx.app/v1/retailer/control-tower/pulse
```

## Infra matrix (SSMR)

| Layer | Status after this apply |
|-------|-------------------------|
| Spanner Retail OS schema | **Applied** (core tables live) |
| Kafka critical topics | **Ready** |
| Redis | **Healthy** |
| WS path | **Wired** |
| Auth/Firebase config | **On** |
| Webhook secrets | **Present** |
| Retail OS HTTP handlers | **Need image deploy** |
| Prod flip | **Blocked** (by design) |

## Rollback

- Image only: set image back to `ssmr-gap-closure-nomock4`  
- Spanner: additive DDL — no drop; leave tables in place  
