# DEPLOYMENT_READINESS_GAP_LEDGER

**Last updated:** 2026-08-02 (SSMR Retail OS cloud apply)

## SSMR stack

| Layer | Status | Notes |
|-------|--------|-------|
| GKE workloads | Running | API, worker, ai-worker |
| Spanner | OK | Retail OS tables applied 2026-08-02 |
| Redis Memorystore TLS | OK | `/ready` redis=ok |
| Kafka Strimzi | OK | Critical topics READY |
| Ingress TLS | Active | api-ssmr.pegasusx.app |
| Firebase config | On | Owner: SMS + SHA + APNs |
| Global Pay | Partial | Webhook URL ready; real password owner-only |
| Retail OS API image | **Gap** | Need image with Retail OS code (pulse 404 on nomock4) |
| PEGASUSX_ENV=production | Blocked | Until GP SUCCESS + profile validation |

## Owner blockers

1. Global Pay real credentials + webhook registration  
2. Firebase Phone SMS / device trust  
3. Approve Cloud Build + kubectl rollout of Retail OS backend tag  

## See also

- `docs/SSMR_RETAIL_OS_CLOUD_APPLY.md`
- `docs/RETAILER_OS_PRODUCTION_GATE.md`
- `artifacts/OWNER_SECRETS_HANDOFF_2026-08-01.md`
