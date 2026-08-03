# SSMR full e2e + supplier portal flywheel deploy — 2026-08-04

## 1. Full e2e

| Item | Result |
|------|--------|
| Target | `https://api-ssmr.pegasusx.app` (`backend-go:ssmr-wave-ab-a1fafaa0`) |
| Log | `artifacts/ssmr-e2e-2026-08-04-final.log` |
| Process | **PASS** (`ssmr smokecheck passed`) |
| Marker gate | **`ssmr-ecosystem-marker-gate-ok`** |
| Negotiations | `PX_E2E_NEGOTIATION_SKIPPED` (product-disabled; allowed) |
| Payment | cash fallback OK where GP merchant auth fails |
| Optimizer | `PX_E2E_OPTIMIZER_SOURCE_FALLBACK_OK` (no OR-Tools sidecar on SSMR) |

### Env required for full e2e (local smokecheck)

```bash
export PUBLIC_BASE_URL=https://api-ssmr.pegasusx.app
export JWT_SECRET=…          # k8s secret backend-go-secrets jwt-secret
export GLOBAL_PAY_WEBHOOK_SECRET=…
export SPANNER_PROJECT=pegasus-503013
export SPANNER_INSTANCE=pegasusx-ssmr-spanner
export SPANNER_DATABASE=pegasusx-ssmr-db
unset SPANNER_EMULATOR_HOST
export SSMR_E2E_TIMEOUT_SEC=1800
```

Without Spanner env, cancel/inventory release probes fail with `DeadlineExceeded`.

### Smokecheck hardening applied

- Fresh order + `ensureOrderDispatchable` before optimizer/dispatch spine so `undispatched_orders` is non-empty after long setup.

## 2. Supplier portal flywheel deploy

| Item | Result |
|------|--------|
| Static build | `NEXT_PUBLIC_API_URL=https://api-ssmr.pegasusx.app` → `apps/supplier-portal/out` |
| Flywheel page | `out/analytics/demand/flywheel.html` present |
| GCS archive | `gs://pegasus-503013-ssmr-assets/portals/supplier/latest/` (+ dated prefix) |
| Image | `…/supplier-portal:ssmr-supplier-portal-flywheel-a1fafaa0` |
| Cloud Build | `473544ad-0ffe-4c5a-b8b2-c9dfa7531bd1` SUCCESS |
| GKE Deploy | `supplier-portal` **1/1 Ready** in `pegasusx-ssmr` |
| Service | `supplier-portal` ClusterIP :80 |

### In-cluster smoke

| Path | Code |
|------|------|
| `/healthz` | **200** |
| `/analytics/demand/flywheel.html` | **200** |

### How to open UI

```bash
kubectl -n pegasusx-ssmr port-forward svc/supplier-portal 18080:80
# browser:
# http://127.0.0.1:18080/analytics/demand/flywheel.html
# login uses API https://api-ssmr.pegasusx.app
```

Public internet hostname not added (org public-access prevention on GCS; no DNS for `supplier-ssmr` yet). Cluster Service + port-forward is the SSMR access path.

### Flywheel API (backend)

```json
{"count":0,"days":7,"items":[],"source":"STORE_POS",
 "description":"Retailer POS sell-through flywheel (DEMAND_SIGNAL). …"}
```

Honest empty until POS sales land with supplier-scoped SKUs.

## Artifacts / code

- `cloudbuild.supplier-portal.yaml`
- `apps/supplier-portal/deploy/{Dockerfile,nginx.conf}`
- `infra/k8s/overlays/ssmr/supplier-portal.yaml`
- `cmd/ssmr-smokecheck` dispatchable-order fix
