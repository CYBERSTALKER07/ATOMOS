# Step 10 — Cloud smoke (SSMR) — PASS

**Date:** 2026-07-27  
**Cluster:** `pegasusx-ssmr-gke` (asia-south1) / project `pegasus-503013`  
**Namespace:** `pegasusx-ssmr`  
**Fiscal:** `FISCAL_PROVIDER=FAKE`

## Verdict

| Gate | Result |
|------|--------|
| Deployments Ready | **PASS** — `backend-go`, `backend-go-worker`, `ai-worker` 1/1 |
| ESO | **PASS** — `backend-go-secrets` SecretSynced |
| In-cluster healthz/ready | **PASS** — all HTTP 200; API/worker ready `{redis:ok, spanner:ok}` |
| Port-forward `cloud_smoke_ssmr.sh` | **PASS** — `PX11_CLOUD_SMOKE_OK` |
| Client policy | **PASS** — `/v1/platform/client-policy` returns `minimum_version` |
| Kafka topics | **PASS** — main/orders/dispatch/realtime/webhooks/freeze-locks/inventory-import |
| Fiscal e2e (`ssmr-smokecheck fiscal`) | **PASS** — `PX_E2E_FISCAL_ALL_OK` |

## Fiscal markers (order → FISCALIZING → COMPLETED)

```
PX_E2E_FISCAL_CASH_OK
PX_E2E_FISCAL_FAIL_RETRY_OK
PX_E2E_FISCAL_FORCE_OK
PX_E2E_FISCAL_SHORTFALL_OK
PX_E2E_FISCAL_SHIFT_FREEZE_OK
PX_E2E_FISCAL_ALL_OK
```

Run:

```bash
export PATH="/opt/homebrew/share/google-cloud-sdk/bin:$PATH"
kubectl -n pegasusx-ssmr port-forward svc/backend-go 18080:80
export PUBLIC_BASE_URL=http://127.0.0.1:18080
export JWT_SECRET="$(kubectl -n pegasusx-ssmr exec deploy/backend-go -- printenv JWT_SECRET)"
export FISCAL_PROVIDER=FAKE PEGASUSX_ENV=ssmr
cd apps/backend-go && go run ./cmd/ssmr-smokecheck fiscal
```

## Fixes applied during Step 10

### 1. Missing Spanner preorder columns

`Orders` lacked `DeliverBefore`, `CancelLockedAt`, and related columns from
`schema/migrations/20250621_manual_preorder.ddl`.

- **Symptom:** order create 422 `Column not found … CancelLockedAt`; sweeper WARN on `DeliverBefore`.
- **Fix:** applied migration DDL to `pegasusx-ssmr-db` via `gcloud spanner databases ddl update`.

### 2. Kafka consume-domain without dual-write

Config had `KAFKA_TOPIC_CONSUME_DOMAIN=true` and `KAFKA_TOPIC_DUAL_WRITE=false`.

- Order mutator listened on `pegasusx-orders`.
- Outbox only published to `pegasusx-main`.
- **Symptom:** cash collect entered `FISCALIZING` / attempt `PENDING`; worker never ran `ApplyFiscalWorkerResult` (notification dispatcher saw the event; order consumer did not).
- **Fix (live):** ConfigMap `KAFKA_TOPIC_CONSUME_DOMAIN=false` + rollout restart.
- **Fix (repo):** `infra/k8s/overlays/staging/kustomization.yaml` aligned to `false` with comment.
- Confirmed log: `"order_topic":"pegasusx-main","consume_domain":false,"dual_write":false`.

### 3. Rollout note (CPU)

Rolling restart needed surge capacity; 3-node cluster hit `Insufficient cpu` until old pods were deleted so new RS could schedule. Prefer `maxUnavailable:1` / recreate-style restarts on this size cluster.

## Non-blocking remaining

| Item | Notes |
|------|--------|
| Datadog agent | Profiler errors expected (no agent in-cluster) |
| FCM / Firebase OTP | Stub credentials; Steps 12+ |
| Ingress / DNS / TLS | Step 11 |
| Global Pay webhooks | Step 14 |
| OFD sandbox | Step 15 (after 10+14) |
| Domain-topic cutover | Only enable `CONSUME_DOMAIN=true` **with** `DUAL_WRITE=true` |

## Next

**Step 11** — Ingress + DNS + TLS (public `PUBLIC_BASE_URL` without port-forward).
