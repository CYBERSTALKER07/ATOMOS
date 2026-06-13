# Migration Runbook — Manifest Route Geometry + Fleet Live Map

**Change ID:** `20250613-supplier-manifest-route-geometry`  
**Owner:** Platform / Fleet ops  
**Blast radius:** Spanner `SupplierTruckManifests`, driver map overlays, supplier/warehouse fleet live maps, dispatch preview geometry  
**Downtime:** None (additive DDL + backward-compatible API)

---

## 1. What this migration enables

| Capability | Depends on |
|---|---|
| Persisted planned-route polyline at manifest seal / reorder | `EncodedRoutePolyline`, `RouteGeometrySource` on `SupplierTruckManifests` |
| Driver planned-route overlay + turn-by-turn | `GET /v1/fleet/route/{routeID}/geometry` + optional `ROUTING_OSRM_URL` |
| Driver off-route reroute | Same geometry endpoint with `reroute=true&from_lat=&from_lng=` |
| Supplier operator fleet live map | `GET /v1/supplier/fleet/live-map` |
| Warehouse operator fleet live map | `GET /v1/warehouse/ops/fleet/live-map` |
| Dispatch preview route overlays | `routing.AttachRouteGeometryToProposedRoutes` on warehouse dispatch preview |

**Canonical DDL (live instances that predate the columns):**

`apps/backend-go/schema/migrations/20250613_supplier_manifest_route_geometry.ddl`

```sql
ALTER TABLE SupplierTruckManifests ADD COLUMN EncodedRoutePolyline STRING(MAX);
ALTER TABLE SupplierTruckManifests ADD COLUMN RouteGeometrySource STRING(32);
```

Fresh installs that apply full `schema/spanner.ddl` already include the columns — skip §3 if `INFORMATION_SCHEMA` shows them.

---

## 2. Preconditions

- [ ] Backend build containing `manifest/geometry.go`, `routing/{geometry.go,builder.go,osrm.go}`, `driver/route_geometry.go`, `supplier/fleet_live_map.go`, and `warehouse/fleet_live_map.go` is staged.
- [ ] Spanner admin IAM: `spanner.databases.updateDdl` on target database.
- [ ] Maintenance window **not required** — additive `ALTER TABLE` only.
- [ ] For **street-snapped polylines** (recommended): OSRM reachable from backend pods; set `ROUTING_OSRM_URL` (see `.env.example`). Without OSRM, geometry falls back to haversine-densified straight segments (`computed_dense` source).
- [ ] For **operator live maps**: manifests must reach `SEALED` or `DISPATCHED` with non-empty `EncodedRoutePolyline` (seal-time persistence or backfill — §5).

---

## 3. Spanner DDL — production / staging

```bash
cd pegasusX/apps/backend-go
# Apply via your standard Spanner migration path, e.g.:
gcloud spanner databases ddl update DATABASE_ID \
  --instance=INSTANCE_ID \
  --ddl-file=schema/migrations/20250613_supplier_manifest_route_geometry.ddl
```

Verify:

```sql
SELECT COLUMN_NAME, SPANNER_TYPE
FROM INFORMATION_SCHEMA.COLUMNS
WHERE TABLE_NAME = 'SupplierTruckManifests'
  AND COLUMN_NAME IN ('EncodedRoutePolyline', 'RouteGeometrySource');
```

---

## 4. Runtime configuration

| Variable | Required | Effect |
|---|---|---|
| `ROUTING_OSRM_URL` | No (recommended prod) | OSRM `/route/v1/driving` base URL; enables street-snapped polylines and turn-by-turn `steps` |
| `REQUIRE_INFRA_ADAPTERS` | No | Unrelated to geometry; keep existing prod policy |

Local dev example (`.env.example`):

```bash
# ROUTING_OSRM_URL=http://localhost:5000
```

---

## 5. Backfill existing manifests

One-off binary: `apps/backend-go/cmd/backfill-route-geometry/main.go`

```bash
cd pegasusX
go run ./apps/backend-go/cmd/backfill-route-geometry \
  -limit 500 \
  -states SEALED,DISPATCHED \
  -dry-run=false
```

Flags:

| Flag | Default | Purpose |
|---|---|---|
| `-limit` | `100` | Max manifests processed per run |
| `-dry-run` | `false` | List candidates without writing |
| `-states` | `SEALED,DISPATCHED,LOADING,DRAFT` | Manifest states to scan |
| `-timeout` | `10m` | Job timeout |

Re-run safely until `skipped` dominates (manifests already have geometry).

---

## 6. API contract summary

### Driver — planned route overlay

`GET /v1/fleet/route/{routeID}/geometry` (DRIVER auth)

| Query | Purpose |
|---|---|
| `include_steps=true` | OSRM turn-by-turn steps (when OSRM configured) |
| `reroute=true&from_lat=&from_lng=` | Recompute route from current driver position |

Response shape: `routing.RouteGeometry` — `route_id`, `encoded_polyline`, `coordinates[]`, `source`, `stop_count`, optional `steps[]`.

**Note:** `GET /v1/driver/manifest` intentionally does **not** inline geometry; clients must call the geometry endpoint using `route_id` from manifest detail.

### Supplier — fleet live map

`GET /v1/supplier/fleet/live-map` (ADMIN / supplier JWT)

Returns `routes[]` with `manifest_id`, `route_id`, `driver_id`, `manifest_state`, optional `route_geometry`, `driver_location`, `live_location_available`, `location_stale`.

### Warehouse — scoped fleet live map

`GET /v1/warehouse/ops/fleet/live-map?warehouse_id=` (WAREHOUSE_ADMIN)

Same route shape as supplier; filtered to manifests for the resolved warehouse scope.

---

## 7. Client surfaces (pegasusX)

| Role | Surface | Status |
|---|---|---|
| DRIVER | `driver-app-android`, `driver-app-ios` | Planned polyline + breadcrumb + turn-by-turn + off-route reroute |
| SUPPLIER | `supplier-portal` (`FleetLiveMap`, WS-accelerated refresh) | MapLibre live map + animated driver markers |
| SUPPLIER | `supplier-app-ios` (`FleetLiveMapView`) | MapKit live map |
| SUPPLIER | `supplier-app-android` (`FleetLiveMapScreen`, MapLibre) | Polyline map + route cards |
| WAREHOUSE | `warehouse-portal` (dashboard + dispatch) | MapLibre live map + animated markers |
| WAREHOUSE | `warehouse-app-android`, `warehouse-app-ios` | **Not yet wired** — portal-only for live fleet map |

Shared contracts: `packages/types` (`RouteGeometryWire`, `SupplierFleetLiveRoute`, `WarehouseFleetLiveMapResponse`), `packages/api-client` (`getSupplierFleetLiveMap`, `getWarehouseFleetLiveMap`).

---

## 8. Operational validation

1. Seal a manifest with ≥2 stops that have retailer coordinates.
2. Confirm `SupplierTruckManifests.EncodedRoutePolyline` is non-empty and `RouteGeometrySource` is set (e.g. `osrm`, `computed_dense`, `manifest_sealed`).
3. Driver app: open active route — planned polyline renders; maneuver banner advances with `include_steps=true`.
4. Supplier portal dashboard anchor: fleet live map shows route polylines + driver pins; WS event triggers refresh within ~1s (15s polling fallback).
5. Warehouse portal dispatch page: same live map authority scoped to warehouse.

---

## 9. Rollback

- DDL columns are additive — no rollback required for API compatibility.
- To disable OSRM snapping: unset `ROUTING_OSRM_URL` and redeploy; new geometry writes fall back to `computed_dense`. Existing stored polylines remain until recomputed on next seal/reorder/backfill.

---

## 10. Cross-references

- `docs/LIVE_TRACKING_EXPECTATIONS.md` — operator/retailer visibility contract
- `docs/ROLE_ROW_PARITY_MATRIX.md` — per-role client parity
- `docs/README.md` — docs index
