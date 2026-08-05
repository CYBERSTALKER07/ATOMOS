# Optimizer & routing — runtime SoT (aligned to codebase)

**Last verified:** 2026-08-05 against `pegasusX` tree + cloud residuals.  
**Supersedes** conflicting wording in older artifacts that imply OR-Tools/OSRM are live on SSMR/prod GKE.

---

## 1. Dispatch optimizer (OR-Tools)

### What exists in code

| Piece | Path / fact |
|-------|-------------|
| Solver | `services/optimizer-core/` — **Python 3.12 + OR-Tools** (`server/contract_solver.py`) |
| HTTP API | `POST /v1/optimizer/solve`, `/healthz`, `/ready` — port **8082** |
| Contract | `packages/optimizer-contract/` |
| Backend client | `apps/backend-go/dispatch/optimizerclient/` via `OPTIMIZER_BASE_URL` + `INTERNAL_API_KEY` |
| Callers | **Supplier** + **warehouse** dispatch preview/execute only (`plan.OptimizeAndValidate`) |
| Clients | **Never** call the solver; they only see `optimizer_source` on API responses |
| Factory | **Does not** use optimizer-core |

### Fallback (always on)

When optimizer is missing, slow, empty, or rejected:

| Source label | Meaning |
|--------------|---------|
| `optimizer` | OR-Tools sidecar returned usable routes |
| `fallback_phase1` | H3 + BinPack heuristic |
| `fallback_validation_rejected` | Solver response failed capacity checks |
| `pure_small_batch` | Below `DISPATCH_AI_MIN_STOPS` (default 12) — skip sidecar |

SSMR e2e intentionally accepts `fallback_phase1` when the sidecar is absent.

### Runtime by environment (truth)

| Environment | OR-Tools live? | Notes |
|-------------|----------------|-------|
| **Local** `infra/docker-compose.ssmr.yml` | **Yes** (when compose up) | Builds `optimizer-core`; backend `OPTIMIZER_BASE_URL=http://optimizer-core:8082` |
| **SSMR GKE** (`overlays/ssmr`) | **No** | Overlay does **not** include optimizer-core; dispatch uses heuristic |
| **Prod GKE** (`overlays/prod`) | **No** | Manifests exist but **`replicas: 0`**; no real AR image (placeholder pin) |
| **Staging overlay** | Intended | Remaps image to `…/pegasusx-staging-images/optimizer-core:staging` — only live if that image exists and deploy is applied |

### How to turn OR-Tools on in cloud

1. Build/push:  
   `asia-south1-docker.pkg.dev/pegasus-503013/pegasusx-ssmr-images/optimizer-core:<tag>`  
   (Dockerfile: `services/optimizer-core/Dockerfile`)
2. Deploy Deployment+Service (or include in SSMR overlay); set replicas ≥ 1.
3. Ensure backend ConfigMap:  
   `OPTIMIZER_BASE_URL=http://optimizer-core.pegasusx.svc.cluster.local:8082`  
   (or namespace-correct DNS).
4. Prove: dispatch response `"optimizer_source":"optimizer"` (not `fallback_phase1`).

Until then, **dispatch is correctly wired but heuristic-only in cloud**.

---

## 2. Route geometry (maps)

### What exists in code

| Piece | Fact |
|-------|------|
| Provider chain | **Google Routes → OSRM → dense** (`apps/backend-go/routing/`) |
| Env | `ROUTING_PROVIDER=auto\|google\|osrm` (default `auto`); `GOOGLE_MAPS_API_KEY`; optional `ROUTING_OSRM_URL` |
| Persist | `SupplierTruckManifests.EncodedRoutePolyline` on seal/reorder |
| Consumers | Driver geometry GET, supplier/WH live-map + preview, retailer tracking `route_geometry`, factory live-map **pins**, payload inbound lat/lng |
| Display | Clients render tiles only (Google / MapLibre / MapKit) — **no** Mapbox Directions |

### Runtime by environment

| Path | Cloud SSMR/prod |
|------|-----------------|
| Google Routes | **Primary** when GSM key has Routes API enabled |
| OSRM | Optional fallback; PVC extract often **empty** — fail-loud if URL set but no `/data/region.osrm` |
| Dense | Last resort |

Detail: [`GOOGLE_ROUTES_WORLD_SCALE_2026-08-05.md`](../artifacts/GOOGLE_ROUTES_WORLD_SCALE_2026-08-05.md), [`CLOUD_CREDENTIALS_CHECKLIST.md`](./CLOUD_CREDENTIALS_CHECKLIST.md).

---

## 3. Related code that is *not* the cloud VRP path

| Piece | Role |
|-------|------|
| `apps/ai-worker/optimizer/` | Go Clarke-Wright / other jobs — **not** the OR-Tools sidecar used for supplier/WH dispatch |
| `services/optimizer-core/server-rust/` | Alternate gRPC experiment — **not** deployed |

Do not document “OR-Tools via ai-worker gRPC” as the current dispatch path.

---

## 4. Doc ownership

| Living doc | Must match this file |
|------------|----------------------|
| [`ROLE_ROW_PARITY_MATRIX.md`](./ROLE_ROW_PARITY_MATRIX.md) | Maps + optimizer rows |
| [`ECOSYSTEM_FEATURES_BY_ROLE.md`](./ECOSYSTEM_FEATURES_BY_ROLE.md) | Spatial / dispatch bullets |
| [`../context/current_status.md`](../context/current_status.md) | Residuals |
| [`../infra/k8s/overlays/README.md`](../infra/k8s/overlays/README.md) | Overlay capabilities |
| [`../.env.example`](../.env.example) | Ports and provider comments |

Historical `artifacts/*` snapshots may still say “undeployed”; treat **this file** as current truth.
