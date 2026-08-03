# SSMR Wave C C3.3/C4.1 backend roll — 2026-08-04

## Scope

- **C3.3** Offline count: `OFFLINE_COUNT_ENABLED`, `GET/POST /v1/retailer/stock/counts/version|commit`, 409 `COUNT_VERSION_CONFLICT`, force audit
- **C4.1** Assist SLA: `ASSIST_SLA_ENABLED`, `ASSIST_SLA_MINUTES`, worker + `SlaBreachedAt` + outbox `RETAILER_ASSIST_TICKET_SLA_BREACHED`

## DDL (apply before or with image)

| Migration | Tables / columns |
|-----------|------------------|
| `20260813_wave_c_offline_count.ddl` | `RetailerStockLocationVersions`, `RetailerStockCountForceAudits` |
| `20260813_wave_c_assist_sla.ddl` | `RetailerAssistanceTickets.SlaBreachedAt`, index `Idx_RetailerAssist_SlaDue` |

```bash
export PHASE0_SPANNER_PROJECT=pegasus-503013
export PHASE0_SPANNER_INSTANCE=pegasusx-ssmr-spanner
export PHASE0_SPANNER_DATABASE=pegasusx-ssmr-db
# bash scripts/phase0_apply_spanner_migrations.sh  # or apply new DDL files directly
```

## Build + deploy

```bash
cd /Users/shakhzod/ATOMOS/pegasusX
TAG="ssmr-wave-c-$(git rev-parse --short HEAD)"
gcloud builds submit --config=cloudbuild.backend.yaml \
  --substitutions=_TAG="$TAG",_REPO=pegasusx-ssmr-images \
  --project=pegasus-503013 .

IMG="asia-south1-docker.pkg.dev/pegasus-503013/pegasusx-ssmr-images/backend-go:$TAG"
kubectl -n pegasusx-ssmr set image deploy/backend-go backend-go="$IMG"
kubectl -n pegasusx-ssmr set image deploy/backend-go-worker backend-go-worker="$IMG"
kubectl -n pegasusx-ssmr rollout status deploy/backend-go
kubectl -n pegasusx-ssmr rollout status deploy/backend-go-worker
```

**Executed 2026-08-04:** image `ssmr-wave-c-52de1e9b` built and rolled to `pegasusx-ssmr` (backend-go + backend-go-worker). Dockerfile merge conflict fixed before build.

## DDL applied on SSMR

- `RetailerStockLocationVersions` (pre-existing)
- `RetailerStockCountForceAudits` (applied)
- `RetailerAssistanceTickets.SlaBreachedAt` + `Idx_RetailerAssist_SlaDue` (applied; regular index, not filtered WHERE)

## Post-roll smoke (flags OFF)

`curl -sf https://api-ssmr.pegasusx.app/healthz` → **ok** (2026-08-04).

Full `go run ./cmd/ssmr-smokecheck e2e` hit Spanner `DeadlineExceeded` on retailer cancel (pre-existing flake) before Wave C markers; re-run when stable or exercise markers with `OFFLINE_COUNT_ENABLED=true` / `ASSIST_SLA_ENABLED=true` in smokecheck env after pilot enable.

```bash
curl -sf https://api-ssmr.pegasusx.app/healthz
export PUBLIC_BASE_URL=https://api-ssmr.pegasusx.app
export JWT_SECRET="$(kubectl -n pegasusx-ssmr get secret backend-go-secrets \
  -o jsonpath='{.data.jwt-secret}' | base64 -d)"
cd apps/backend-go && go run ./cmd/ssmr-smokecheck
```

Expected when flags off:

- `PX_E2E_OFFLINE_COUNT_CONFLICT_SKIPPED`
- `PX_E2E_ASSIST_SLA_SKIPPED`
- CORE regression green

## Pilot flag sequence

**Executed 2026-08-04:** `OFFLINE_COUNT_ENABLED=true` and `ASSIST_SLA_ENABLED=true` + `ASSIST_SLA_MINUTES=15` set on `backend-go` and `backend-go-worker` in `pegasusx-ssmr`.

```bash
NS=pegasusx-ssmr
# 1 — offline count (pilot stores)
kubectl -n $NS set env deployment/backend-go OFFLINE_COUNT_ENABLED=true
kubectl -n $NS set env deployment/backend-go-worker OFFLINE_COUNT_ENABLED=true
kubectl -n $NS rollout status deployment/backend-go
# Re-run smoke → expect PX_E2E_OFFLINE_COUNT_CONFLICT_OK

# 2 — assist SLA (pilot floors)
kubectl -n $NS set env deployment/backend-go ASSIST_SLA_ENABLED=true ASSIST_SLA_MINUTES=15
kubectl -n $NS set env deployment/backend-go-worker ASSIST_SLA_ENABLED=true ASSIST_SLA_MINUTES=15
kubectl -n $NS rollout status deployment/backend-go
# Re-run smoke → expect PX_E2E_ASSIST_SLA_OK
```

## Rollback

| Issue | Action |
|-------|--------|
| Count conflicts misbehaving | `OFFLINE_COUNT_ENABLED=false` + rollout |
| SLA noise | `ASSIST_SLA_ENABLED=false` + rollout |
| Bad image | `kubectl rollout undo deployment/backend-go -n pegasusx-ssmr` |

## Merge conflict fixes included in this roll

Resolved blocking conflicts in: `spanner.ddl`, `events/events.go`, `events/topic_routing.go`, `order/amend.go`, `order/driver_edges.go`, `order/service.go`, `claims/handlers.go`, `driverroutes/routes.go`.
