# Prod API / DB / server / DevOps wiring via CLI

**Date:** 2026-07-28  
**Target:** `pegasus-503013` · `pegasusx-ssmr-gke` · ns `pegasusx-ssmr`  
**Env label:** `PEGASUSX_ENV=ssmr` (prod-hardened staging; full `production` profile blocked until real Global Pay password + DNS TLS)

---

## Verified health

| Check | Result |
|-------|--------|
| `GET /healthz` | **200** ok |
| `GET /ready` | **200** redis+spanner ok |
| API / worker / ai-worker | Deployed; API+worker Ready 1/1 |
| Ingress | `136.69.43.141` · `api-ssmr.pegasusx.app` |
| HPA | min 1 / max 2 · CPU 75% |
| PDB | maxUnavailable 1 |
| ESO | SecretSynced |
| Spanner daily backups | **Active** schedule `default_daily_full_backup_schedule` + extra backup CREATING/READY |
| Budget | `pegasusX monthly cap (staging)` $1500 |

---

## Wired / hardened this session

### API & config
- `FISCAL_PROVIDER=PEGASUS` (platform receipts, no Soliq)
- `PUBLIC_BASE_URL=https://api-ssmr.pegasusx.app`
- `FIREBASE_AUTH_ENABLED=true` · `FIREBASE_PROJECT_ID=pegasus-503013`
- `GLOBAL_PAY_ENV=staging`
- `REQUIRE_INFRA_ADAPTERS=true` · `ALLOW_MEMORY_FALLBACK=false` · `ALLOW_AUTH_BYPASS=false`
- Kafka consume-domain **false** / dual-write **false**
- `WS_ALLOWED_ORIGINS` set for API/admin/retailer + local dev
- `GCS_BUCKET_NAME=pegasus-503013-ssmr-assets`
- `UPDATES_BASE_URL=https://storage.googleapis.com/pegasus-503013-ssmr-assets/updates`

### Database
- Spanner instance/db live
- **Daily full backup schedule** already present (7d retention)
- Additional on-demand backup triggered (14d retention)

### Object storage
- Bucket `pegasus-503013-ssmr-assets` created
- Prefixes `updates/` + `catalog/` seeded
- IAM `roles/storage.objectAdmin` on assets buckets for `ssmr-backend@…`

### Secrets (GSM + K8s)
- Strong non-placeholder: JWT, internal-api-key, adyen/stripe/payme/click webhooks, GP webhook secret
- **Still placeholder:** `global-pay-password=REPLACE_WITH_GP_STAGING_PASSWORD` (must come from Global Pay)

### DevOps
- HPA + PDB applied (cluster-CPU-safe sizes)
- Deploy annotations: `pegasusx.io/prod-wiring-date=2026-07-28`
- Billing budget present
- Cloud Logging / Monitoring APIs enabled

### Images
- API/worker: `backend-go:ssmr-s15-pegasus-receipts`
- AI: `ai-worker:ssmr-4a0796fd-glibc`

---

## Explicitly NOT flipped to pure production

| Item | Reason |
|------|--------|
| `PEGASUSX_ENV=production` | Requires real GP password + clean production secret profile; would not add value until GP + DNS |
| ManagedCertificate Active | Needs DNS A → `136.69.43.141` |
| SUCCESS Global Pay verify | Needs real merchant password |
| Soliq OFD | Deferred by product decision |
| optimizer-core / OSRM | Not deployed (optional). Geometry primary is Google Routes when Maps key has Routes API — see [`docs/OPTIMIZER_AND_ROUTING_RUNTIME.md`](../docs/OPTIMIZER_AND_ROUTING_RUNTIME.md) |
| HPA minReplicas=3 | Insufficient CPU on current 3-node pool |

---

## How to hit the API

```bash
IP=136.69.43.141
HOST=api-ssmr.pegasusx.app
curl -fsS --resolve "${HOST}:80:${IP}" "http://${HOST}/healthz"
curl -fsS --resolve "${HOST}:80:${IP}" "http://${HOST}/ready"
```

Or DNS: `api-ssmr.pegasusx.app. A 136.69.43.141`

---

## Remaining human inputs for true prod cutover

1. **DNS** A record for managed TLS  
2. **Global Pay** real service_id / username / password (+ webhook secret if they set it)  
3. Point apps at `https://api-ssmr.pegasusx.app` after cert Active  
4. Optional: Soliq, Payme/Click, Datadog, larger node pool  

See also: `artifacts/PROD_WIRING_AND_THIRD_PARTIES.md`, `artifacts/receipts-multi-provider.md`
