# Optimizer & routing — runtime SoT (aligned to codebase)

**Last verified:** 2026-08-06 against `pegasusX` tree + §8.5 constraint/multi-depot/OSRM-matrix wiring.  
**Supersedes** conflicting wording in older artifacts that imply OR-Tools/OSRM are live on SSMR/prod GKE.

---

## 1. Dispatch optimizer (OR-Tools)

### What exists in code

| Piece | Path / fact |
|-------|-------------|
| Solver | `services/optimizer-core/` — **Python 3.12 + OR-Tools** (`server/contract_solver.py`) |
| HTTP API | `POST /v1/optimizer/solve`, `/healthz`, `/ready` — port **8082** |
| Contract | `packages/optimizer-contract/` — Stop/Vehicle constraint fields + optional `distance_matrix_m` |
| Backend client | `apps/backend-go/dispatch/optimizerclient/` via `OPTIMIZER_BASE_URL` + `INTERNAL_API_KEY` |
| Timeouts | Solver default `time_limit_ms=5000` (clamp 60s); Go HTTP soft timeout **8s**; sidecar `OPTIMIZER_SOFT_TIMEOUT_SEC=8` |
| Callers | **Supplier** + **warehouse** dispatch preview/execute only (`plan.OptimizeAndValidate`) |
| Clients | **Never** call the solver; they only see `optimizer_source` on API responses |
| Factory | **Does not** use optimizer-core |
| Certification (P2-2) | `testdata/cert/*.json` + `server/test_certification_harness.py`; Rust greedy reports `HEURISTIC` not `OPTIMAL` |

### Constraint fidelity (§8.5)

| Constraint | Contract field | OR-Tools mechanism |
|------------|----------------|--------------------|
| Cold chain | `Stop.requires_cold_chain` → `Vehicle.has_refrigeration` | `VehicleVar.SetValues` (allowed vehicles) |
| Hazmat | `Stop.is_hazardous` → `Vehicle.hazmat_certified` | same (intersection when both) |
| Shift / HOS | `shift_start`/`shift_end` or `max_route_minutes` | Time dimension `SetSpanUpperBoundForVehicle` |
| Max stops | `tunables.max_stops_per_route` | **StopCount** capacity dimension (no post-hoc tail chop) |
| Multi-depot | per-vehicle `start_lat`/`start_lng` (+ optional end) | `RoutingIndexManager(starts, ends)` |

Order handling flags are aggregated (OR) from line-item snapshots / `Products` during dispatch hydrate (`dispatch/volume.go`) and projected onto `GeoOrder` → contract `Stop`.

### Distance matrix ownership

**Go builds the matrix; Python stays pure** (no OSRM HTTP from the sidecar).

```
dispatch OptimizeAndValidate
  → optimizerclient builds nodes: vehicle starts (+ distinct ends) then stops
  → OSRM /table/v1/driving/?annotations=distance  (circuit breaker)
  → haversine fill for unreachable / OSRM miss
  → SolveRequest.distance_matrix_m
  → optimizer-core
```

| Piece | Path |
|-------|------|
| OSRM table client | `apps/backend-go/routing/osrm.go` → `DistanceMatrix` |
| Haversine fallback | `routing.HaversineDistanceMatrixM` + `MergeDistanceMatrix` |
| Wire attach | `optimizerclient.buildDistanceMatrix` |

Residual: GPS-telemetry calibration of matrix vs actual legs (not in this wave).

### Fallback (always on)

When optimizer is missing, slow, empty, or rejected:

| Source label | Meaning |
|--------------|---------|
| `optimizer` | OR-Tools sidecar returned usable routes |
| `fallback_phase1` | H3 + BinPack heuristic |
| `fallback_validation_rejected` | Solver response failed capacity checks |
| `pure_small_batch` | Below `DISPATCH_AI_MIN_STOPS` (default 12) — skip sidecar |

SSMR e2e accepts `fallback_phase1` when the sidecar is absent (`PX_E2E_OPTIMIZER_SOURCE_FALLBACK_OK`). Cold-chain fixture: soft-skip only when sidecar unreachable (`PX_E2E_OPTIMIZER_CONSTRAINT_SKIPPED`); mis-assignment is hard `PX_E2E_OPTIMIZER_CONSTRAINT_FAIL` (P2-1).

### Runtime by environment (truth)

| Environment | OR-Tools live? | Notes |
|-------------|----------------|-------|
| **Local** `infra/docker-compose.ssmr.yml` | **Yes** (when compose up) | Builds `optimizer-core`; backend `OPTIMIZER_BASE_URL=http://optimizer-core:8082` |
| **SSMR GKE** (`overlays/ssmr`) | **Yes (manifest)** | Overlay includes optimizer-core; kustomize patch sets **`replicas: 1`**; image remapped to `…/optimizer-core:ssmr`. Heuristic fallback remains if the pod/image is unhealthy. |
| **Prod GKE** (`overlays/prod`) | **No** | Manifests exist but **`replicas: 0`** until AR publish of a real optimizer-core digest (not backend-go). |
| **Staging overlay** | Intended | Remaps image to `…/pegasusx-staging-images/optimizer-core:staging` — only live if that image exists and deploy is applied |

### How to turn OR-Tools on in cloud

1. **Build / push** (from repo root):

```bash
docker build -t asia-south1-docker.pkg.dev/pegasus-503013/pegasusx-ssmr-images/optimizer-core:ssmr \
  -f pegasusX/services/optimizer-core/Dockerfile \
  pegasusX/services/optimizer-core

docker push asia-south1-docker.pkg.dev/pegasus-503013/pegasusx-ssmr-images/optimizer-core:ssmr
```

   Staging tag: `…/pegasusx-staging-images/optimizer-core:staging` (same Dockerfile).

2. Deploy Deployment+Service (SSMR/prod overlays already include them); set **replicas ≥ 1**.
3. Ensure backend ConfigMap:  
   `OPTIMIZER_BASE_URL=http://optimizer-core:8082`  
   (or namespace-qualified DNS). Base ConfigMap already carries the cluster-local URL.
4. Prove: dispatch response `"optimizer_source":"optimizer"` (not `fallback_phase1`).
5. Optional: smokecheck prints `PX_E2E_OPTIMIZER_CONSTRAINT_OK` when `OPTIMIZER_BASE_URL` reaches the sidecar cold-chain fixture.

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
| OSRM | Optional fallback + **optimizer matrix** when `ROUTING_OSRM_URL` set; PVC extract often **empty** — fail-loud if URL set but no `/data/region.osrm` |
| Dense / haversine | Last resort (geometry dense; matrix haversine) |

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
