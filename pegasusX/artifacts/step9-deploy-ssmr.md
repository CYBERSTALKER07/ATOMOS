# Step 9 — K8s rollout (SSMR) — DONE

**Cluster:** `pegasusx-ssmr-gke` (asia-south1)  
**Namespace:** `pegasusx-ssmr`  
**Date:** 2026-07-26

> **Still true for optimizer:** SSMR does not run `optimizer-core`; dispatch soft-fails to heuristic. Current SoT: [`docs/OPTIMIZER_AND_ROUTING_RUNTIME.md`](../docs/OPTIMIZER_AND_ROUTING_RUNTIME.md). Geometry is Google Routes → OSRM → dense (not OSRM-only).

## Workloads (all Ready 1/1)

| Deployment | Image | Notes |
|------------|-------|--------|
| `backend-go` | `…/backend-go:ssmr-4a0796fd` | API listening `:8080` |
| `backend-go-worker` | `…/backend-go:ssmr-4a0796fd` | Worker health `:8081` |
| `ai-worker` | `…/ai-worker:ssmr-4a0796fd-glibc` | glibc rebuild (Alpine CGO was broken) |

Services: `backend-go:80`, `backend-go-worker:8081`, `ai-worker:8081` (ClusterIP)

## Infra wiring

| Component | Value |
|-----------|--------|
| Spanner | `pegasus-503013` / `pegasusx-ssmr-spanner` / `pegasusx-ssmr-db` |
| Redis | `10.129.212.243:6378` + TLS + AUTH (GSM + Memorystore CA) |
| Kafka | `pegasusx-kafka-bootstrap.kafka.svc.cluster.local:9092` (Strimzi, no SASL) |
| Secrets | ESO `backend-go-secrets` SecretSynced |
| Fiscal | `FISCAL_PROVIDER=FAKE` |

## Fixes applied during rollout

1. **Stale Redis IP** in GSM (`10.251.238.236` → live `10.129.212.243`)
2. **Wrong Redis AUTH** in GSM — refreshed from `gcloud redis get-auth-string`
3. **Memorystore CA** in secret `redis-ca` + `REDIS_CA_CERT` env
4. **ai-worker** glibc runtime image tag `ssmr-4a0796fd-glibc`

## Non-blocking warnings (expected for staging)

- Datadog agent not in-cluster
- FCM no-op (no Firebase project ID)
- `UPDATES_BASE_URL` unset
- Optimizer base URL points to undeployed `optimizer-core`

## Quick health check

```bash
kubectl -n pegasusx-ssmr port-forward svc/backend-go 8080:80
curl -sS http://127.0.0.1:8080/healthz
curl -sS http://127.0.0.1:8080/ready
```

## Next (Steps 10–15)

Validation / smoke tests, DNS/ingress, third-party webhooks as needed.
