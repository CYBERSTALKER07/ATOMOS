# Code Changes: Milestone 2 (M2: Geography, Maps, and Security)

## Summary of Changes

### 1. H3 Spatial Indexing Disambiguation (Resolution 7 vs Resolution 9)
- **`proximity/h3_cell.go`**:
  - Defined `SettlementH3Resolution = 9` (~174m edge) for fine-grained doorstep settlement unlock and perimeter fence evaluations.
  - Implemented `H3CellRes9(lat, lng float64) string` returning explicit resolution 9 H3 cell.
  - Implemented `SettlementH3Cell(lat, lng float64) string` as the named resolution 9 helper to eliminate any ambiguity with coarse matching cell `MatchingH3Cell` (which uses `MatchingResolution = 7`).
  - Preserved `H3CellFromLatLng(lat, lng float64) string` as a compatibility wrapper delegating to `H3CellRes9`.
- **`order/proximity_settlement.go`**:
  - Added `SettlementH3Cell(lat, lng float64) string` helper.
  - Updated `EvaluateSettlementProximity` to call `proximity.SettlementH3Cell(driverLat, driverLng)` when evaluating driver arrival against order H3 cell.
- **`order/proximity.go`**:
  - Updated `CheckProximity` to default to `SettlementH3Resolution` (res 9) when `cfg.H3Resolution <= 0`.
- **`proximity/h3_cell_test.go` & `order/proximity_settlement_test.go`**:
  - Added `TestSettlementH3Cell_Res9` asserting resolution 9 property and differentiation from res 7 matching cell.
  - Added `TestEvaluateSettlementProximity_H3Match` verifying settlement doorstep unlocking with resolution 9 cell.

### 2. Geocode API Security & Country Bias
- **`geolocation/cache_keys.go`**:
  - Added `normalizeCountryCode(cc string) string` helper (defaulting to `"uz"`).
  - Updated all cache key formatters to namespace keys with country code:
    - `autocompleteCacheKey(cc, input)` -> `geo:autocomplete:<cc>:<normalized_input>`
    - `forwardCacheKey(cc, address)` -> `geo:forward:<cc>:<normalized_address>`
    - `reverseCacheKey(cc, lat, lng)` -> `geo:reverse:<cc>:<lat,lng>`
    - `placeCacheKey(cc, placeID)` -> `geo:place:<cc>:<place_id>`
- **`geolocation/service.go`**:
  - Added `resolveCountry(ctx context.Context, countryCode ...string) string` extracting country code from explicit parameter, request context claims, market pack, or env default (`DEFAULT_MARKET_CODE` / `UZ`).
  - Updated `Autocomplete`, `ForwardGeocode`, `ReverseGeocode`, and `ResolvePlaceID` to accept variadic `countryCode ...string` and pass country-namespaced cache keys.
  - Added `components=country:<cc>` query parameter for Google Places Autocomplete, Google Geocoding, and Google Reverse Geocoding.
  - Added `countrycodes=<cc>` query parameter for OpenStreetMap Nominatim search and reverse endpoints.
- **`geolocation/handlers.go`**:
  - Added `checkAuth(w, r)` authentication check to all geocode endpoints (`/v1/platform/geocode/*`), rejecting unauthenticated calls with 401 Unauthorized and rejecting WebSocket upgrade tickets with 403 Forbidden.
  - Added `queryCountry(r *http.Request) string` parsing `country` / `country_code` from query parameters and forward geocode request body.
- **`platformroutes/routes.go`**:
  - Wrapped `geolocation.RegisterRoutes(gr, d.GeocodeHandler)` inside `r.With(auth.RequireAnyAuthenticated()).Group(...)` to enforce authentication middleware across all geocode platform routes.
- **`geolocation/cache_test.go` & `geolocation/handlers_test.go`**:
  - Updated cache tests to verify country-namespaced key isolation (`TestCountryNamespacedCacheIsolation`).
  - Created `handlers_test.go` with `TestGeocodeRoutes_UnauthenticatedRejected` and `TestGeocodeRoutes_AuthenticatedAccepted` to verify auth enforcement and 401 rejection on unauthenticated requests.

### 3. Factory Fleet Spanner Data
- **`factory/service.go`**:
  - Implemented `loadFactoryFleetFromSpanner(ctx context.Context, factoryID string) (*factoryFleetSpannerResult, error)` querying Spanner:
    - `Vehicles` where `HomeNodeType = 'FACTORY' AND HomeNodeId = @factoryId`
    - LEFT JOIN active `FactoryTruckManifests` (`State IN ('LOADING', 'SEALED', 'DISPATCHED') AND FactoryId = @factoryId`)
    - LEFT JOIN `Drivers` on `DriverId = Manifest.DriverId`
  - Updated `HandleFleetVehicles(w http.ResponseWriter, r *http.Request)` to load from Spanner when `s.spannerClient != nil` and fallback to in-memory demo data otherwise.
- **`factory/ios_compat.go`**:
  - Updated `HandleFleet(w http.ResponseWriter, r *http.Request)` to load live vehicle snapshots (with capacity, active manifest route, driver name, and state) from Spanner when `s.spannerClient != nil` and fallback to in-memory demo data otherwise.
- **`factory/service_test.go`**:
  - Added `TestHandleFleet_DemoFallbackAndStructure` and `TestHandleFleetVehicles_DemoFallbackAndStructure` verifying fleet endpoint response shapes and fallback behavior.


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
