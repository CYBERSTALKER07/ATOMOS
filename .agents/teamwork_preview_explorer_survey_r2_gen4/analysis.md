# Requirement 2 (R2: Geography, Maps, and Security) Investigation Report

## Executive Summary
This investigation analyzes the architecture, implementation status, and gaps across three core functional areas in `pegasusX/apps/backend-go/`:
1. **H3 Spatial Indexing & Resolution Partitioning**: Contrasting matching resolution (Res 7) vs. settlement/doorstep proximity & coverage resolution (Res 9), identifying all writer locations and establishing field separation rules.
2. **Geocoding & Location APIs**: Analyzing `/v1/platform/geocode/*` routing, authentication middleware coverage, and country-bias integration gaps in upstream Google Maps / OpenStreetMap Nominatim calls.
3. **Factory Fleet List & Manifest Projections**: Investigating `GET /v1/factory/fleet` and related endpoints currently backed by in-memory mock state, and outlining Spanner queries against `FactoryTruckManifests`, `Vehicles`, and `Drivers`.

---

## 1. H3 Index Resolution Investigation

### 1.1 Resolution Architecture & Policy
The system distinguishes between two fundamental H3 spatial scales:
- **Matching & Node Geography (`MatchingResolution = 7`)**: Hexagon edge length ≈ 1.22 km, area ≈ 5.16 km². Used for retailer store points, warehouse coverage envelopes, factory candidate locations, dispatch candidate indexing, and order checkout geo-stamping.
- **Settlement & Doorstep Proximity (`SettlementH3Resolution = 9` / `CoverageResolution = 9`)**: Hexagon edge length ≈ 174 m, area ≈ 0.1 km². Used for driver arrival / delivery proximity unlocking (`EvaluateSettlementProximity`), doorstep verification, and fine-grained polygon coverage cells.

### 1.2 Key Package Findings

#### A. Package `proximity`
- **`proximity/node_geography.go`**:
  - `MatchingResolution = 7` (Line 8).
  - `MatchingH3Cell(lat, lng float64) string` (Lines 17–23): Computes Res 7 cell using `PanicSafeLatLngToCell(lat, lng, MatchingResolution)`.
  - `StampNodeGeography(pack auth.MarketPack, lat, lng float64, requestedCountry string) (NodeGeography, error)` (Lines 44–57): Enforces pack market compliance and stamps Res 7 cell.
- **`proximity/h3.go`**:
  - `CoverageResolution = 9` (Line 11): Target resolution for detailed coverage expansions.
  - Safe CGO wrappers: `PanicSafeLatLngToCell`, `PanicSafePolygonToCells`, `CompactCells`, `UncompactCells`, `PanicSafeGridDistance`, `HaversineDistance`.
- **`proximity/h3_cell.go`**:
  - `H3CellFromLatLng(lat, lng float64) string` (Lines 4–13): Computes **Resolution 9** cell via `CellsInRadius(lat, lng, 9, 0)`.
  - *Risk/Ambiguity Note*: Having `MatchingH3Cell` generate Res 7 while `H3CellFromLatLng` generates Res 9 without resolution in the function name created legacy parity tests expecting Res 9.
- **`proximity/coverage_engine.go`**:
  - `MergeStorePin` (Line 166) & `ResolveServingWarehouse` (Line 186): Call `MatchingH3Cell` (Res 7).
  - `CellInCoverage` (Lines 80–119): Evaluates whether a retailer's Res 7 cell matches or is a child/parent of stored warehouse coverage cells via `h3.Cell.Parent(res)`.
  - `PerimeterCells` (Lines 371–391): Aggregates explicit coverage cells for Redis caching.

#### B. Package `order`
- **`order/proximity_settlement.go`**:
  - `SettlementH3Resolution = 9` (Line 25): Tight doorstep unlock threshold (equivalent to ~100m geofence).
  - `EvaluateSettlementProximity(driverLat, driverLng, orderLat, orderLng float64, orderH3Cell string)` (Lines 72–102): Takes driver GPS coords, converts to Res 9 cell (`PanicSafeLatLngToCell(driverLat, driverLng, SettlementH3Resolution)`), and checks compatibility with `orderH3Cell` (which was stamped at Res 7 during checkout) via `h3CellsCompatible` (Lines 104–129).
  - `HandleProximityUnlock` (Lines 156–355): Unlocks payment modes on orders (`ProximityUnlockedAt`, `ProximityMethod = "H3"` or `"GEOFENCE_100M"`).
- **`order/unified_checkout.go` & `order/coverage.go`**:
  - `orderH3Resolution = 7` (`unified_checkout.go:17`).
  - `h3CellFromLatLng(lat, lng float64)` (`unified_checkout.go:446`): Generates 15-character hex at Res 7 for the `Orders.H3Cell` column.
- **`order/warehouse_resolver_spanner.go`**:
  - Line 56: Falls back to `proximity.MatchingH3Cell(store.Lat, store.Lng)` (Res 7) when evaluating warehouse assignment and Redis perimeter checks (`r.redis.SIsMember(ctx, key, store.H3Cell)`).

#### C. Matching Writers Requiring Resolution 7
Every writer creating or mutating geography-stamped entities uses Res 7:
1. `warehouse/geo.go:stampWarehouseCoords` / `stampWarehouseEntity`: Calls `proximity.StampNodeGeography`.
2. `factory/geo.go:stampFactoryCoords` / `stampFactoryEntity`: Calls `proximity.StampNodeGeography`.
3. `retailer/geo.go:stampRetailerCoords` / `stampRetailerOptionalCoords`: Calls `proximity.StampNodeGeography`.
4. `retailer/service.go:CreateLocation` (Line 682): Calls `proximity.StampNodeGeography`.
5. `supplier/portal_handlers.go` (Lines 570, 629): Warehouses and Factories creation/patching call `proximity.StampNodeGeography`.
6. `order/unified_checkout.go` (Line 243, 450): Authoritative checkout order writing stamps `orderH3Resolution = 7`.

#### D. Resolution 9 vs. Resolution 7 Naming Separation
- Currently, Spanner schema columns across `Retailers`, `RetailerLocations`, `Warehouses`, `Factories`, and `Orders` are named simply `H3Cell STRING(15)`.
- Because `Orders.H3Cell` stores Res 7 while `EvaluateSettlementProximity` evaluates driver telemetry at Res 9, hierarchical matching (`h3CellsCompatible`) is required.
- **Recommendation**:
  - Retain `H3Cell` / `MatchingH3Cell` exclusively for Resolution 7 matching cells.
  - Where Resolution 9 is explicitly persisted or transferred (e.g. telemetry samples, doorstep markers, detailed coverage polygons), use explicitly named identifiers: `SettlementH3Cell`, `DoorstepH3Cell`, or `H3CellRes9`.
  - In `proximity/h3_cell.go`, alias or rename `H3CellFromLatLng` to `SettlementH3CellFromLatLng` or document that it produces Resolution 9 to avoid accidental usage in matching writers.

---

## 2. Geocode API Endpoints Investigation

### 2.1 Handlers & Routing
The Geocoding module is implemented in `geolocation/` and mounted in `platformroutes/`:
- **Mount Point** (`platformroutes/routes.go:38–40`):
  ```go
  if d.GeocodeHandler != nil {
      geolocation.RegisterRoutes(r, d.GeocodeHandler)
  }
  ```
- **Registered Endpoints** (`geolocation/handlers.go:28–31`):
  - `GET /v1/platform/geocode/autocomplete` (`h.handleAutocomplete`): Address autocomplete suggestion query.
  - `GET /v1/platform/geocode/place` (`h.handlePlace`): Fetches coordinates and formatted address for a `place_id`.
  - `GET /v1/platform/geocode/reverse` (`h.handleReverse`): Resolves `lat` & `lng` query params to an address.
  - `POST /v1/platform/geocode/forward` (`h.handleForward`): Resolves a JSON body `{"address": "..."}` to coordinates.

### 2.2 Upstream Integration & Caching
- **Provider Mechanism** (`geolocation/service.go`):
  - Primary: Google Maps API (`places/autocomplete`, `place/details`, `geocode/json`) when `googleAPIKey` is provided.
  - Fallback: OpenStreetMap Nominatim (`https://nominatim.openstreetmap.org/search`, `/reverse`) when no Google API key is configured.
- **Caching Layer** (`geolocation/cache_keys.go` & `service.go`):
  - Redis cache with TTLs: Autocomplete (24h), Forward/Reverse/Place (7 days).

### 2.3 Authentication Middleware Gaps
- **Observation**:
  - In `platformroutes/routes.go`, `geolocation.RegisterRoutes(r, d.GeocodeHandler)` is mounted directly onto the top-level router without any `auth.RequireRole(...)` or `auth.RequireAnyAuthenticated()` middleware.
  - The docstring in `handlers.go:20` states: `// RegisterRoutes mounts public geocode helpers used during onboarding.`
  - However, because these endpoints trigger external API calls to Google Maps (incurring billing) or OpenStreetMap Nominatim (subject to strict upstream rate limits and Terms of Service), unauthenticated open access poses a financial exhaustion and abuse risk.
- **Reliability Layer Check**:
  - In `bootstrap/reliability_middleware.go:292–293`, paths matching `/v1/platform/geocode/` are assigned `reliabilityClassGeocode`, which enforces an IP-based rate limit (`RateLimitGeocodeMax = 60` req/min, key `"ip:" + ipKey + ":geocode"`).
- **Missing Controls**:
  - There is no option for session-gated geocoding (e.g. `auth.RequireAnyAuthenticated()` for logged-in users or specific onboarding token guards for registration).
  - No client API key / signature verification is enforced on unauthenticated onboarding calls.

### 2.4 Country-Bias Parameter Handling (Missing)
- **Observation**:
  - The codebase currently has **zero country parameter support** in the `geolocation` package.
  - `geolocation/handlers.go` does not parse any `country`, `country_bias`, or `market` query parameter / body field.
  - `geolocation/service.go` does not inject country filters into upstream requests:
    - Google Places Autocomplete: Omit `components=country:<cc>`.
    - Google Geocoding (`forwardGoogle`): Omit `components=country:<cc>`.
    - Nominatim Search (`forwardNominatim`): Omit `countrycodes=<cc>`.
- **Consequences**:
  - Search queries for streets or cities (e.g. "Amir Temur", "Chilanzar") can return matches in foreign countries instead of the active operational market (e.g. Uzbekistan `UZ`).
  - Cache keys in `geolocation/cache_keys.go` (`geo:autocomplete:input`, `geo:forward:address`) do not namespace by country, leading to cross-market cache collisions.
- **Remediation Requirements**:
  1. Add optional `country` query/body parameter in `handlers.go`, defaulting to `auth.CheckoutPackFromContext(ctx)` or `auth.DefaultMarketCodeFromEnv()`.
  2. Pass `components=country:<cc>` to Google Places/Geocoding and `countrycodes=<cc>` to Nominatim.
  3. Include the normalized country code in the Redis cache key (e.g. `geo:forward:<cc>:<normalized_address>`).

---

## 3. Factory Fleet List & Spanner Schema

### 3.1 Current Factory Fleet Endpoints & Implementation
The factory routes are defined in `factoryroutes/routes.go:40–77` under the `mountOps` group (requiring `RoleFactory`, `RoleFactoryAdmin`, or `RoleAdmin` with `auth.RequireFactoryScope`):
- `GET /v1/factory/fleet` (`d.Service.HandleFleet` in `factory/ios_compat.go:168–179`):
  - **Current Implementation**:
    ```go
    s.mu.Lock()
    s.ensureDemoDataLocked()
    vehicles := s.iosFleetVehiclesLocked()
    s.mu.Unlock()
    writeJSON(w, http.StatusOK, map[string]any{"vehicles": vehicles})
    ```
  - **Issue**: Backed entirely by in-memory mock slice `s.fleetVehicles` (initialized with dummy IDs `"fv-01"`, `"fv-02"`, `"fv-03"` in `factory/service.go:408–415`). Hardcodes dummy metrics (`capacity_m3: 12.0`, `capacity_kg: 3200.0`, `capacity_l: 12000.0`, `driver_name: ""`, `current_route_id: ""`).
- `GET /v1/factory/fleet/drivers` (`factory/service.go:1348`): Returns `s.fleetDrivers` in-memory mock data.
- `GET /v1/factory/fleet/vehicles` (`factory/service.go:1361`): Returns `s.fleetVehicles` in-memory mock data.
- `GET /v1/factory/fleet/live-map` (`factory/fleet_live_map.go:64–86`):
  - **Production-Ready Spanner Query**: This endpoint *already* queries Spanner table `FactoryTruckManifests` joined with `Drivers` and integrates with `telemetry.DriverLocation`!

### 3.2 Spanner Schema for `FactoryTruckManifests`
From `schema/spanner.ddl:991–1013`:
```sql
CREATE TABLE FactoryTruckManifests (
  ManifestId        STRING(36)  NOT NULL,
  FactoryId         STRING(36)  NOT NULL,
  SupplierId        STRING(36)  NOT NULL,
  DriverId          STRING(36),
  VehicleId         STRING(36),
  State             STRING(20)  NOT NULL, -- DRAFT, LOADING, SEALED, DISPATCHED, COMPLETED, CANCELLED
  TotalVolumeVU     FLOAT64     NOT NULL DEFAULT (0),
  MaxVolumeVU       FLOAT64     NOT NULL DEFAULT (0),
  StopCount         INT64       NOT NULL DEFAULT (0),
  TransferCount     INT64       NOT NULL DEFAULT (0),
  LoadingStartedAt  TIMESTAMP,
  SealedAt          TIMESTAMP,
  DispatchedAt      TIMESTAMP,
  CompletedAt       TIMESTAMP,
  CancelledAt       TIMESTAMP,
  CreatedAt         TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt         TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (ManifestId);

CREATE INDEX Idx_FactoryManifests_ByFactoryId ON FactoryTruckManifests(FactoryId);
CREATE INDEX Idx_FactoryManifests_BySupplierId ON FactoryTruckManifests(SupplierId);
CREATE INDEX Idx_FactoryManifests_ByState ON FactoryTruckManifests(State);
```

### 3.3 Related Spanner Schema: `Vehicles` and `Drivers`
- **`Vehicles` Table** (`schema/spanner.ddl:412–430`):
  ```sql
  CREATE TABLE Vehicles (
    VehicleId         STRING(36)    NOT NULL,
    Label             STRING(100),
    LicensePlate      STRING(32)    NOT NULL,
    SupplierId        STRING(36)    NOT NULL,
    HomeNodeType      STRING(20)    NOT NULL, -- 'FACTORY' or 'WAREHOUSE'
    HomeNodeId        STRING(36)    NOT NULL, -- FactoryId / WarehouseId
    VehicleClass      STRING(10)    NOT NULL DEFAULT ('CLASS_B'),
    MaxVolumeVU       FLOAT64       NOT NULL DEFAULT (150.0),
    IsActive          BOOL          NOT NULL,
    UnavailableReason STRING(64),
    UnavailableNote   STRING(255),
    CreatedAt         TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
    UpdatedAt         TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  ) PRIMARY KEY (VehicleId);
  CREATE INDEX Idx_Vehicles_ByHomeNode ON Vehicles(HomeNodeType, HomeNodeId, IsActive);
  ```
- **`Drivers` Table** (`schema/spanner.ddl:388–407`):
  ```sql
  CREATE TABLE Drivers (
    DriverId          STRING(36)    NOT NULL,
    Name              STRING(255)   NOT NULL,
    Phone             STRING(32)    NOT NULL,
    PinHash           STRING(MAX),
    SupplierId        STRING(36)    NOT NULL,
    HomeNodeType      STRING(20)    NOT NULL, -- 'FACTORY' or 'WAREHOUSE'
    HomeNodeId        STRING(36)    NOT NULL,
    VehicleId         STRING(36),
    IsActive          BOOL          NOT NULL,
    OnShift           BOOL          NOT NULL DEFAULT (true),
    ...
  ) PRIMARY KEY (DriverId);
  CREATE INDEX Idx_Drivers_ByHomeNode ON Drivers(HomeNodeType, HomeNodeId, IsActive);
  ```

### 3.4 Query Strategy to Replace In-Memory Data Source
To eliminate mock data from `GET /v1/factory/fleet` and `GET /v1/factory/fleet/vehicles`, the service should execute a Spanner query that retrieves factory-assigned vehicles and joins their active manifest status:

```sql
SELECT 
    v.VehicleId,
    v.LicensePlate,
    v.VehicleClass,
    v.MaxVolumeVU,
    v.IsActive,
    v.UnavailableReason,
    COALESCE(m.State, 'AVAILABLE') AS VehicleStatus,
    m.ManifestId,
    m.DriverId,
    COALESCE(d.Name, '') AS DriverName,
    m.TotalVolumeVU,
    m.TransferCount
FROM Vehicles v
LEFT JOIN Drivers d 
    ON d.VehicleId = v.VehicleId AND d.IsActive = true
LEFT JOIN FactoryTruckManifests m 
    ON m.VehicleId = v.VehicleId 
    AND m.FactoryId = @fid 
    AND m.State IN ('LOADING', 'SEALED', 'DISPATCHED')
WHERE v.HomeNodeType = 'FACTORY' 
  AND v.HomeNodeId = @fid
ORDER BY v.LicensePlate ASC;
```

**Mapping to iOS / Web Fleet DTO**:
- `id`: `v.VehicleId`
- `plate_number`: `v.LicensePlate`
- `capacity_m3`: Derived from `v.MaxVolumeVU` or vehicle class standard volume
- `capacity_kg`: Derived from vehicle class standard weight payload
- `status`: `m.State` if active manifest exists (e.g. `"LOADING"`, `"DISPATCHED"`), else `"AVAILABLE"` / `"MAINTENANCE"`
- `driver_name`: `d.Name`
- `current_route_id`: `"route_" + m.ManifestId` (when manifest active)
- `current_route`: `m.ManifestId`

---

## 4. Summary Matrix of Findings & Gaps

| Area | Component | Current Code Location | Status & Core Gap | Actionable Resolution |
|---|---|---|---|---|
| **1. H3 Resolution** | Matching Writers | `proximity/node_geography.go`, `warehouse/geo.go`, `factory/geo.go`, `retailer/geo.go`, `order/unified_checkout.go` | Correctly enforcing Res 7 (`MatchingResolution = 7`). | Maintain Res 7 constraint across all node stamps and checkout orders. |
| **1. H3 Resolution** | Settlement / Proximity | `order/proximity_settlement.go`, `proximity/h3.go`, `proximity/h3_cell.go` | Settlement evaluates driver GPS at Res 9 against order Res 7 cell using hierarchy match. `H3CellFromLatLng` silently defaults to Res 9. | Distinctly name Res 9 helpers/fields (`SettlementH3Cell`, `H3CellRes9`) to avoid confusion with Res 7 matching cells. |
| **2. Geocode API** | Route Registration & Auth | `platformroutes/routes.go:38–40`, `geolocation/handlers.go:28–31` | Routes are completely unauthenticated (`GET /autocomplete`, `/place`, `/reverse`, `POST /forward`). Only protected by IP rate limiter. | Add authenticated route group options (`RequireAnyAuthenticated` or specific onboarding tokens) to guard Google Maps billing. |
| **2. Geocode API** | Country Bias | `geolocation/handlers.go`, `geolocation/service.go`, `geolocation/cache_keys.go` | Zero country filtering. Google/Nominatim queries unconstrained. Cache keys lack market namespace. | Add `country` query/body parameter, pass `components=country:<cc>` to Google and `countrycodes` to Nominatim, namespace cache keys. |
| **3. Factory Fleet** | Fleet List Endpoint | `factory/ios_compat.go:168` (`HandleFleet`), `factory/service.go:1360` (`HandleFleetVehicles`) | Uses in-memory mock slice `s.fleetVehicles` with dummy metrics. | Query Spanner `Vehicles` joined with `FactoryTruckManifests` and `Drivers` filtered by `HomeNodeId = @factoryId`. |
| **3. Factory Fleet** | Live Map Endpoint | `factory/fleet_live_map.go` (`HandleFactoryFleetLiveMap`) | Already queries Spanner `FactoryTruckManifests` and live driver telemetry. | Reference as model implementation for updating `HandleFleet`. |

---


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
