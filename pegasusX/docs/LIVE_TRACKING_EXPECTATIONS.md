# Live tracking expectations

Contract for what operators, retailers, and supplier finance teams should expect to see during an active delivery. Pairs with `DRIVER_SUPPORT_PLAYBOOK.md` (driver-side triage) and `DELIVERY_ESCALATION_POLICY.md` (escalation policy).

## Visibility contract

| Surface | Expected refresh cadence | Frame shape |
| --- | --- | --- |
| Supplier fleet live map (`GET /v1/supplier/fleet/live-map`) | WebSocket-accelerated refresh on supplier realtime events; 15 s polling fallback | Per active `SEALED`/`DISPATCHED` manifest: `route_id`, `manifest_state`, planned `route_geometry` polyline (when persisted), `driver_location`, `live_location_available`, `location_stale` |
| Warehouse fleet live map (`GET /v1/warehouse/ops/fleet/live-map`) | Same as supplier portal pattern for warehouse-scoped sockets | Warehouse-filtered manifest routes with the same geometry + driver location fields |
| Driver execution map | Near-real-time via authenticated `/v1/ws` telemetry | Live pin + traveled breadcrumb polyline; planned route from `GET /v1/fleet/route/{routeID}/geometry` (not inlined on manifest detail) |
| Retailer tracking screen | Near-real-time during active delivery; eventual consistency for pre and post states | Order status, ETA hint, driver-arrival flag, payment-required hint when applicable |
| Supplier finance dashboard | Eventually consistent within reconciliation window | Order final amount, payout split snapshot, settlement state, fee amount |

## Required states

Every live tracking surface must explicitly handle:

1. Loading: data is being fetched for the first time.
2. Empty: there are no active deliveries to display.
3. Offline or reconnecting: the WebSocket has dropped and is being re-established.
4. Stale data: the connection is open but no frames have arrived within the cadence threshold.
5. Permission-restricted: the viewer does not have authority to see the live frame.

Silent failures, such as a frozen map without an offline indicator, are not acceptable.

## Frame integrity

- Every telemetry frame must carry a trace ID that stitches it back to its originating driver event.
- Operator map hover or focus state should expose at minimum: driver identity, truck identity, route identity, assigned order count, current or next stop, and last update timestamp.
- Clicking a route, marker, driver, or truck should open the corresponding detail surface when the product already has or clearly needs one.

## Deviation surfacing

- If the planned route and the actual execution diverge, the deviation should be visible to operators rather than hidden.
- Operators should be able to tell whether the active sequencing is the optimized default or a driver-selected override.
- Driver clients detect sustained off-route deviation locally and request `GET /v1/fleet/route/{routeID}/geometry?reroute=true&from_lat=&from_lng=` to refresh the planned overlay; operator maps should show both the persisted planned polyline and the live driver pin so drift is visually obvious.

## Route geometry prerequisites

- Planned polylines require `SupplierTruckManifests.EncodedRoutePolyline` populated at seal/reorder (migration `20250613_supplier_manifest_route_geometry.ddl`).
- Street-snapped geometry requires `ROUTING_OSRM_URL` on backend pods; without OSRM, `RouteGeometrySource` falls back to `computed_dense`.
- Pre-migration manifests: run `go run ./apps/backend-go/cmd/backfill-route-geometry` (see `docs/MIGRATION_RUNBOOK_MANIFEST_ROUTE_GEOMETRY.md`).

## Out of band events

- Force-complete, force-reassign, and manual override actions must appear in the live tracking audit trail with the actor identity.
- These actions never replace the underlying telemetry; they only annotate it.

## Cross-references

- `DRIVER_SUPPORT_PLAYBOOK.md` for driver-side triage.
- `DELIVERY_ESCALATION_POLICY.md` for when to escalate.
- `REASSIGNMENT_SUPPORT_PLAYBOOK.md` for reassignment context that affects live tracking.
- `MIGRATION_RUNBOOK_MANIFEST_ROUTE_GEOMETRY.md` for Spanner DDL, OSRM config, backfill, and fleet live-map API parity.
