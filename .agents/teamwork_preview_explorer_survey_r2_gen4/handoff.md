# Handoff Report: Requirement 2 (R2: Geography, Maps, and Security)

## 1. Observation

### Area 1: H3 Index Resolution
- **`proximity/node_geography.go:8`**:
  `const MatchingResolution = 7`
- **`proximity/node_geography.go:17–23`**:
  `func MatchingH3Cell(lat, lng float64) string` calls `PanicSafeLatLngToCell(lat, lng, MatchingResolution)` (Res 7).
- **`proximity/h3.go:11`**:
  `const CoverageResolution = 9`
- **`proximity/h3_cell.go:4–13`**:
  `func H3CellFromLatLng(lat, lng float64) string` calls `CellsInRadius(lat, lng, 9, 0)` (Res 9).
- **`order/proximity_settlement.go:25`**:
  `const SettlementH3Resolution = 9`
- **`order/proximity_settlement.go:82–86`**:
  `cell, err := proximity.PanicSafeLatLngToCell(driverLat, driverLng, SettlementH3Resolution)` (converts driver GPS to Res 9, then matches against `orderH3Cell` via `h3CellsCompatible`).
- **`order/unified_checkout.go:17` & `450`**:
  `const orderH3Resolution = 7` used by `h3CellFromLatLng` to write order cells.
- **Writers enforcing Resolution 7**:
  - `warehouse/geo.go:21`: calls `proximity.StampNodeGeography`
  - `factory/geo.go:17`: calls `proximity.StampNodeGeography`
  - `retailer/geo.go:17`: calls `proximity.StampNodeGeography`
  - `retailer/service.go:682`: calls `proximity.StampNodeGeography`
  - `supplier/portal_handlers.go:570, 629`: calls `proximity.StampNodeGeography`
  - `order/unified_checkout.go:243`: calls `h3CellFromLatLng` (Res 7)
  - `order/warehouse_resolver_spanner.go:56`: calls `proximity.MatchingH3Cell` (Res 7)

### Area 2: Geocode API Endpoints
- **`platformroutes/routes.go:38–40`**:
  ```go
  if d.GeocodeHandler != nil {
      geolocation.RegisterRoutes(r, d.GeocodeHandler)
  }
  ```
  Mounted directly on `r` with no `auth.RequireRole(...)` or `auth.RequireAnyAuthenticated()` wrapper.
- **`geolocation/handlers.go:28–31`**:
  Registers `/v1/platform/geocode/autocomplete`, `/v1/platform/geocode/place`, `/v1/platform/geocode/reverse`, `/v1/platform/geocode/forward`.
- **`bootstrap/reliability_middleware.go:292–293`**:
  Applies IP rate limiting (`RateLimitGeocodeMax = 60` req/min) for `/v1/platform/geocode/` paths under `reliabilityClassGeocode`.
- **`geolocation/service.go:166–175` & `286–292`**:
  Requests to Google Places Autocomplete (`endpoint := "https://maps.googleapis.com/maps/api/place/autocomplete/json"`) and Google Geocoding (`endpoint := "https://maps.googleapis.com/maps/api/geocode/json"`) contain no `components=country:...` parameter.
- **`geolocation/service.go:358–368`**:
  Requests to Nominatim (`nominatimURL+"/search?"+q.Encode()`) contain no `countrycodes=...` parameter.
- **`geolocation/cache_keys.go:28–34`**:
  Cache keys `geo:autocomplete:input` and `geo:forward:address` do not include country/market prefix.

### Area 3: Factory Fleet List
- **`factoryroutes/routes.go:55`**:
  `rr.Get("/v1/factory/fleet", d.Service.HandleFleet)` inside `mountOps` group (`auth.RequireRole(factoryRoles...)` + `auth.RequireFactoryScope`).
- **`factory/ios_compat.go:168–179`**:
  `HandleFleet` calls `s.ensureDemoDataLocked()` and `s.iosFleetVehiclesLocked()`, returning mock data from `s.fleetVehicles`.
- **`factory/service.go:408–415`**:
  Initializes `s.fleetVehicles` with dummy entries `"fv-01"`, `"fv-02"`, `"fv-03"`.
- **`schema/spanner.ddl:991–1009`**:
  Table `FactoryTruckManifests` exists with columns: `ManifestId`, `FactoryId`, `SupplierId`, `DriverId`, `VehicleId`, `State`, `TotalVolumeVU`, `MaxVolumeVU`, `StopCount`, `TransferCount`, `LoadingStartedAt`, `SealedAt`, `DispatchedAt`, `CompletedAt`, `CancelledAt`, `CreatedAt`, `UpdatedAt`.
- **`factory/fleet_live_map.go:98–107`**:
  `listFactoryFleetLiveRoutes` queries `FactoryTruckManifests` joined with `Drivers` filtered by `m.FactoryId = @fid` and `m.State IN ('SEALED', 'DISPATCHED', 'LOADING')`.

---

## 2. Logic Chain

1. **H3 Spatial Indexing Disambiguation**:
   - Matching algorithms (`proximity.ResolveServingWarehouse`, `order.warehouse_resolver_spanner`, `proximity.StampNodeGeography`) operate at coarse hexagonal cells (Res 7, ~1.22 km edge) to aggregate regional supply-demand and avoid excessive polygon sharding.
   - Settlement proximity (`order.EvaluateSettlementProximity`) and detailed doorstep geofencing operate at fine hexagonal cells (Res 9, ~174 m edge) to prevent fraudulent off-site payment unlocks.
   - Because `proximity/h3_cell.go:H3CellFromLatLng` returns Res 9 while `proximity/node_geography.go:MatchingH3Cell` returns Res 7, implicit conversions create fragility. Establishing strict naming conventions (`MatchingH3Cell` / `H3Cell` for Res 7 vs. `SettlementH3Cell` / `H3CellRes9` for Res 9) ensures type-level and semantic clarity across all writers and readers.

2. **Geocode Endpoint Security & Market Isolation**:
   - Geocode endpoints invoke external paid APIs (Google Maps) and rate-limited public services (OSM Nominatim).
   - Although IP rate-limiting is configured in `reliability_middleware.go`, the lack of session authentication or signed token validation allows external anonymous scraping of geocoding endpoints.
   - Furthermore, omitting the country parameter causes Google/Nominatim to return global results and pollutes the un-namespaced Redis cache with cross-market collisions. Passing the market country (e.g. `UZ`) via `components=country:uz` / `countrycodes=uz` and namespacing cache keys resolves both ambiguity and cache poisoning.

3. **Factory Fleet Spanner Integration**:
   - `HandleFleet` (`GET /v1/factory/fleet`) and `HandleFleetVehicles` (`GET /v1/factory/fleet/vehicles`) rely on in-memory mock data (`s.fleetVehicles`), causing discrepancy with the actual factory state.
   - The Spanner schema already defines `FactoryTruckManifests`, `Vehicles` (with `HomeNodeType = 'FACTORY'`), and `Drivers`.
   - `factory/fleet_live_map.go` already demonstrates the correct pattern by querying `FactoryTruckManifests` in Spanner. Replicating this query in `HandleFleet` (joining `Vehicles`, `FactoryTruckManifests`, and `Drivers`) will make the endpoint production-ready and consistent across web and mobile clients.

---

## 3. Caveats
- No changes to database schemas are strictly required since `FactoryTruckManifests`, `Vehicles`, `Drivers`, `WarehouseCoverageCells`, and `Orders` already contain the necessary columns (`H3Cell`, `VehicleId`, `DriverId`, `State`, etc.).
- When adding authentication to `/v1/platform/geocode/*`, client onboarding workflows that resolve addresses before registration must either use an onboarding token or public geocoding must be strictly scoped to `DefaultMarketCodeFromEnv()`.
- No source code modifications were performed during this exploration turn (read-only mode respected).

---

## 4. Conclusion
1. **H3 Partitioning**: Writers across `warehouse`, `factory`, `retailer`, `supplier`, and `order` checkout consistently require **Resolution 7**. Doorstep settlement verification correctly evaluates at **Resolution 9**. Res 9 fields/helpers should be explicitly named (`SettlementH3Cell`, `H3CellRes9`) to avoid confusion with Res 7 matching cells.
2. **Geocode Endpoints**: `/v1/platform/geocode/*` requires country-bias integration (`components=country:<cc>` / `countrycodes=<cc>`), country-namespaced cache keys (`geo:<endpoint>:<cc>:<query>`), and authentication middleware options (`RequireAnyAuthenticated` for logged-in users / rate-limited onboarding guards).
3. **Factory Fleet List**: `GET /v1/factory/fleet` can be migrated from in-memory `s.fleetVehicles` to Spanner by querying `Vehicles` where `HomeNodeType = 'FACTORY' AND HomeNodeId = @factoryId` left-joined with active `FactoryTruckManifests` (`State IN ('LOADING', 'SEALED', 'DISPATCHED')`) and `Drivers`.

---

## 5. Verification Method
1. **H3 Verification**:
   - Check test assertions in `proximity/node_geography_test.go`:
     ```bash
     cd /Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go && go test ./proximity -run "TestStampNodeGeography|TestMatchingH3Cell"
     ```
   - Verify `order/proximity_settlement_test.go`:
     ```bash
     cd /Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go && go test ./order -run "TestEvaluateSettlementProximity|TestProximityUnlock"
     ```
2. **Geocode Verification**:
   - Run geolocation cache and handler tests:
     ```bash
     cd /Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go && go test ./geolocation/...
     ```
3. **Factory Fleet Verification**:
   - Run factory live map and manifest tests:
     ```bash
     cd /Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go && go test ./factory -run "TestFactory|TestLiveMap"
     ```


# Universal Agent & Engineering Guidelines
When developing, designing, or planning, always ensure to account for:
- Gaps, edge cases, and comprehensive feature validation.
- Best practices and optimized integration for Kafka, Redis, Backend, Optimizers, AI, and UI.
- Real-time concepts including WebSockets, webhooks, and their native app equivalents.
- Thorough business logic for features, understanding how the role, app, and ecosystem work together, and engagements with other roles and features.
- Best practices for backend, frontend, and infrastructure libraries/packages. Always prefer existing, high-quality open-source libraries and packages that best suit our features before creating our own.
- Optimal UI infrastructure and UX patterns (e.g., optimal screen positioning for drivers during an active route), applying the same high standards to backend and cloud architecture.
- ALWAYS search the web to find open-source code, libraries, packages, math, algorithms, approaches, and best practices for anything we are doing. If none exist, then create our own.
- Always search the web to get the correct logic, and incorporate edge cases, business logic for features, operations (ops), workflow, data consistency, finance, and AI into everything we do.
