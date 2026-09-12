# Google Routes world-scale maps — wiring closeout

**Date:** 2026-08-05  
**Project:** `pegasus-503013`  
**Also see:** [`docs/OPTIMIZER_AND_ROUTING_RUNTIME.md`](../docs/OPTIMIZER_AND_ROUTING_RUNTIME.md) (geometry + optimizer cloud status in one place)

## Decision

- **Primary geometry:** Google Routes API (`routes.googleapis.com`) via `GOOGLE_MAPS_API_KEY`
- **Fallback:** OSRM (`ROUTING_OSRM_URL`) → dense haversine
- **Mode:** `ROUTING_PROVIDER=auto|google|osrm` (default `auto`)
- **Clients:** display-only (Google Maps / MapLibre / MapKit); no Mapbox Directions; no dual routers

## Provider order & cost

1. On seal / reorder / preview miss / reroute → `GeometryBuilder` calls Google (then OSRM).
2. `GET /v1/fleet/route/{id}/geometry` and live-maps prefer **persisted** `EncodedRoutePolyline` (no Routes call when stored).
3. Source labels: `google_routes_driving`, `osrm_driving`, `computed_dense` (+ `reroute_*` prefix).

## Cloud / Terraform

- [`infra/terraform/maps_platform.tf`](../infra/terraform/maps_platform.tf) — enable Routes, Geocoding, Places, Maps backend; optional Android/iOS SDK GSM shells
- Live enablement applied on `pegasus-503013` (2026-08-05)
- Key model: [`docs/CLOUD_CREDENTIALS_CHECKLIST.md`](../docs/CLOUD_CREDENTIALS_CHECKLIST.md)
- Budget: existing `budget.tf` covers project monthly spend (include Maps SKUs)

## Role-row surfaces

| Role | Change |
|------|--------|
| Driver / supplier / warehouse | Auto-benefit from Google geometry on seal |
| Retailer | `route_geometry` on `GET /v1/retailer/tracking` + desktop/Android/iOS overlays |
| Factory | `GET /v1/factory/fleet/live-map` + portal MapLibre + Android/iOS live driver lists |
| Payload | Manifest wire `driver_lat` / `driver_lng` / `live_location_available` (terminal + Android) |

## SSMR deploy (2026-08-06, gcloud)

| Item | Value |
|------|--------|
| Image | `asia-south1-docker.pkg.dev/pegasus-503013/pegasusx-ssmr-images/backend-go:ssmr-google-routes-20260806-005506` |
| Deploys | `backend-go` + `backend-go-worker` (`ROUTING_PROVIDER=auto`, Maps key from `backend-go-secrets`) |
| Spanner | `OutboxEvents.ClaimedBy` / `ClaimedUntil` + `SchemaMigrations` (Gate-0 DDL) applied on `pegasusx-ssmr-db` |
| Proof | Pod log `Google Routes geometry enabled`; in-cluster Routes `computeRoutes` → polyline; `PUBLIC_BASE_URL=https://api-ssmr.pegasusx.app` → `PX11_CLOUD_SMOKE_OK`; `/ready` 200 |
| Prod | **Deferred** until SSMR seal→geometry walk (`source=google_routes_driving`) is signed |

## Residual

- Factory sealed-route polylines need DDL on `FactoryTruckManifests` (pins-first shipped)
- OSRM PVC extract remains optional regional cost shield
- Raise Routes QPM after first production traffic; restrict server key to GKE NAT
- Optional: full seal → `GET /v1/fleet/route/{id}/geometry` e2e on SSMR seed data
