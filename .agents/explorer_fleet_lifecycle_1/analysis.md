# FLEET & DRIVER LIFECYCLE MANAGEMENT: CROSS-ECOSYSTEM DOMAIN AUDIT & PRODUCTION PORTING SPECIFICATION

**Author:** explorer_fleet_lifecycle  
**Date:** 2026-09-14T14:26:00+05:00  
**Target Subsystems:** `pegasusX` (Global Enterprise Multi-Tenant) & `pegasus.x` (Sovereign Lean Single-Tenant)  
**Security & Boundary Mode:** STRICT TWO-SYSTEM ARCHITECTURAL SEPARATION (Zero Cross-Contamination)

---

## 1. Executive Summary

Within the `V.O.I.D` ecosystem, commercial transport and fleet operations serve as the physical backbone connecting central manufacturing factories and regional warehouses to tens of thousands of urban and regional retail shops across Uzbekistan.

An exhaustive, line-by-line investigation of both `pegasusX` and `pegasus.x` reveals two starkly different architectural paradigms, each with unique strengths, historical implementations, and critical gaps:

1. **`pegasusX` (Global Enterprise Multi-Tenant Cloud)**:
   - Built on **Google Cloud Spanner** (`schema/spanner.ddl`, 3,750 lines) with multi-tenant partitioning (`SupplierId STRING(36)`), **Apache Kafka**, and Go backend services.
   - Treats `Vehicles` and `Drivers` as core assets with volumetric capacities (`MaxVolumeVU`, `VehicleClass`), home node depots, driver scoring (`DriverScores`), and labor availability (`DriverAvailability`).
   - Implements dynamic shift assignment guards in `apps/backend-go/warehouse/fleet_guards.go` and roadside breakdown rescue / order hot-swapping in `apps/backend-go/driver/rescue.go`.
   - **Architectural Limitation**: Does NOT have a native, first-class Digital Pre-Trip DVIR (Driver Vehicle Inspection Report) table in Spanner; vehicle maintenance relies on manual admin flag toggling (`UnavailableReason`). Furthermore, fuel types and refrigeration temperature envelope profiles are not represented in the primary Spanner DDL.

2. **`pegasus.x` (Sovereign Lean Single-Tenant Core)**:
   - Engineered for sovereign deployment on **Servercore Tashkent Tier III** (direct TAS-IX peering) running **PostgreSQL 16** (`pgx/v5`) + **Redis 7** + Go Chi + Tauri v2 (Next.js 15) + Native Android/iOS.
   - Evolved through comprehensive migrations (`025_fleet_and_driver_lifecycle_management.sql`, `013_fleet_integrity_coldchain_and_blindspots.sql`, and `037_delivery_exceptions_and_driver_rescues.sql`).
   - Implements Uzbekistan-specific fleet reality: Compressed Natural Gas (`METHANE_CNG`), LPG (`PROPANE_LPG`), national biometric PINFL (14 digits), statutory OSAGO insurance, MOT (`texosmotr`), CNG hydrostatic cylinder safety validation, and CBU AML cash limits (25,000,000 UZS = 2,500,000,000 tiyins).
   - Features rich Go models in `backend/internal/fleet/models.go` (1,189 lines), a complete service layer in `backend/internal/fleet/service.go` (1,118 lines), and repository in `backend/internal/fleet/repository.go` (2,551 lines).
   - **Critical Architectural Gaps Discovered**:
     1. **Severe Cross-Contamination Build Break**: `backend/internal/outbox/relay.go:13` imports `github.com/pegasus-x/core/internal/kafka`, directly violating the sovereign boundary rule ("NEVER import Kafka into pegasus.x") and breaking package compilation.
     2. **Ephemeral Pub/Sub vs. Persistent Streams**: Service layer comments specify Redis Streams (`XADD`, `stream:fleet:events`), but the code calls ephemeral Redis Pub/Sub (`s.redis.Publish`), creating message loss vulnerabilities during mobile disconnection in regional corridors (e.g., Tashkent-Samarkand M39 highway).
     3. **Dispatch Safety Gate Bypass**: In `backend/internal/dispatch/service.go:258`, the query permits dispatch when `vi.is_safe_to_operate IS NULL`, violating the strict "No Inspection = No Dispatch" invariant.

---

## 2. Exhaustive Analysis of Fleet Management in `pegasusX`

### 2.1 Vehicles/Trucks Management
In `pegasusX`, vehicles are modeled as enterprise assets bound to a supplier and a specific home node (depot/warehouse/factory):

- **Spanner DDL Specification** (`pegasusX/apps/backend-go/schema/spanner.ddl:418-436`):
  ```sql
  CREATE TABLE Vehicles (
    VehicleId         STRING(36)    NOT NULL,
    Label             STRING(100),
    LicensePlate      STRING(32)    NOT NULL,
    SupplierId        STRING(36)    NOT NULL,
    HomeNodeType      STRING(20)    NOT NULL,
    HomeNodeId        STRING(36)    NOT NULL,
    VehicleClass      STRING(10)    NOT NULL DEFAULT ('CLASS_B'),
    MaxVolumeVU       FLOAT64       NOT NULL DEFAULT (150.0),
    IsActive          BOOL          NOT NULL,
    UnavailableReason STRING(64),
    UnavailableNote   STRING(255),
    CreatedAt         TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
    UpdatedAt         TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  ) PRIMARY KEY (VehicleId);

  CREATE INDEX Idx_Vehicles_BySupplierPlate ON Vehicles(SupplierId, LicensePlate);
  CREATE INDEX Idx_Vehicles_ByHomeNode ON Vehicles(HomeNodeType, HomeNodeId, IsActive);
  ```
- **Volumetric Capacity Tiers** (`apps/backend-go/dispatch/vehicle.go:5-35`):
  - `CLASS_A`: Damas / Chevrolet Labo = **50.0 VU** (Micro-drop, hyper-dense mahallas).
  - `CLASS_B`: Gazelle Next / Transit Van = **150.0 VU** (Default city delivery standard).
  - `CLASS_C`: Isuzu NPR 75L / Box Truck = **400.0 VU** (Heavy distributor backbone).
  - `CLASS_D`: MAN TGL / Isuzu FVR = **1000.0 VU** (Inter-regional linehaul).
- **CRUD & State Mutation Engine**:
  - `CreateVehicle`: `apps/backend-go/driver/repository_crud.go:169-188` inserts into Spanner and emits `VEHICLE_CREATED` to outbox.
  - Warehouse admin creation: `apps/backend-go/warehouse/fleet_ops.go:80-122` via `POST /v1/warehouse/ops/vehicles`.
  - Vehicle Status Patching: `apps/backend-go/warehouse/ops_fleet_handlers.go:477-543` via `PATCH /v1/warehouse/ops/vehicles/{vehicleID}`. Emits `VEHICLE_AVAILABILITY_CHANGED` (`fleet_ops.go:158-207`), updates Spanner atomically, broadcasts via WebSocket `s.broadcastWarehouseEvent`, and invalidates dispatch plan cache (`s.InvalidateDispatchPlanCache(ctx, warehouseID)`).
  - Reason codes normalized in `apps/backend-go/warehouse/fleet_availability.go:11-25`: `MAINTENANCE`, `TRUCK_DAMAGED`, `REGULATORY_HOLD`, `MANUAL_HOLD`, `OTHER`.

### 2.2 Driver Onboarding & Management
- **Spanner DDL Specification** (`pegasusX/apps/backend-go/schema/spanner.ddl:394-413`):
  ```sql
  CREATE TABLE Drivers (
    DriverId          STRING(36)    NOT NULL,
    Name              STRING(255)   NOT NULL,
    Phone             STRING(32)    NOT NULL,
    PinHash           STRING(MAX),
    SupplierId        STRING(36)    NOT NULL,
    HomeNodeType      STRING(20)    NOT NULL,
    HomeNodeId        STRING(36)    NOT NULL,
    VehicleId         STRING(36),
    IsActive          BOOL          NOT NULL,
    OnShift           BOOL          NOT NULL DEFAULT (true),
    UnavailableReason STRING(64),
    UnavailableNote   STRING(255),
    CreatedAt         TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
    UpdatedAt         TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  ) PRIMARY KEY (DriverId);

  CREATE INDEX Idx_Drivers_BySupplierPhone ON Drivers(SupplierId, Phone);
  CREATE INDEX Idx_Drivers_ByHomeNode ON Drivers(HomeNodeType, HomeNodeId, IsActive);
  ```
- **Driver Performance Scoring & Labor Capacity**:
  - `DriverScores` table (`schema/spanner.ddl:3048-3060`): Tracks `Score`, `OnTimeRate`, `CompletionRate`, `DamageRate`, `ShopClosedRate`, `FeedbackScore`, and `StopsPerHour`.
  - `DriverAvailability` table (`schema/spanner.ddl:3062-3070`): Composite PK `(DriverId, Date)` with `AvailableHours`, `ZoneH3`, `Status`.
- **Onboarding & Security**:
  - Handled in `apps/backend-go/warehouse/ops_fleet_handlers.go:49-132`.
  - Cryptographically secure 4-digit PIN generation: `generateOpsDriverPIN` (`ops_fleet_handlers.go:569-583`) using `crypto/rand`.
  - Hash stored via `bcrypt.GenerateFromPassword` (cost = `bcrypt.DefaultCost`).
  - Atomically writes to `Drivers` and emits `DRIVER_CREATED` event.

### 2.3 Dynamic Driver-Vehicle Daily Shift Assignments
In `pegasusX`, vehicle assignment is recorded by setting `VehicleId` on the `Drivers` table:
- **Assignment Handler**: `handleAssignVehicle` (`apps/backend-go/warehouse/ops_fleet_handlers.go:134-203`) via `PATCH /v1/warehouse/ops/drivers/{driverID}/assign-vehicle`.
- **Strict Guard Invariants** (`apps/backend-go/warehouse/fleet_guards.go:68-120` & `ops_fleet_handlers.go:205-256`):
  1. Driver cannot be reassigned if they have active orders: `countActiveOrdersForDriver` (`fleet_guards.go:79-97`) checks statuses `LOADED`, `IN_TRANSIT`, `ARRIVED`, `DISPATCHED`, `ARRIVING`, `EN_ROUTE`, `AWAITING_PAYMENT`, `PENDING_CASH_COLLECTION`.
  2. Vehicle cannot be reassigned if it has active in-transit orders: `countActiveOrdersForVehicle` (`fleet_guards.go:99-120`).
  3. If vehicle is assigned to another driver, the conflict driver cannot be active with open orders.
  4. Previous driver's `VehicleId` is cleared in Spanner atomically.
  5. Emits `DRIVER_VEHICLE_ASSIGNED` event to outbox (`ops_fleet_handlers.go:297-305`).

### 2.4 Mid-Shift Hot-Swapping & Roadside Rescue Protocol
When a truck breaks down on the road, `pegasusX` handles the emergency handover in `apps/backend-go/driver/rescue.go`:
- **Driver SOS Request** (`HandleRescueRequest`, lines 14-109):
  - Driver calls `POST /v1/driver/ops/rescue/request`.
  - Transaction marks broken driver `UnavailableReason = 'NEEDS_RESCUE'`, `IsActive = false`.
  - Emits `RESCUE_REQUESTED` to `OutboxEvents`.
  - Broadcasts `RESCUE_BROADCAST` to all warehouse drivers via WebSocket `driverHub`.
- **Peer Acceptance & Hot-Swap Reassignment** (`HandleRescueRespond`, lines 118-288):
  - Peer driver calls `POST /v1/driver/ops/rescue/respond` with `{"accept": true, "broken_driver_id": "...", "rescue_id": "..."}`.
  - Queries all active orders for the broken driver (`rescue.go:169-191`).
  - Atomically reassigns all orders to the rescue driver: `UPDATE Orders SET DriverId = @newDriverId WHERE DriverId = @oldDriverId AND Status IN (...)` (`rescue.go:194-205`).
  - Transitions broken driver to `UnavailableReason = 'BROKEN_DOWN'`.
  - Emits `EventOrderReassigned` for each order with the rescue driver's license plate.
  - Emits `RESCUE_ACCEPTED` event.

### 2.5 Pre-Trip Vehicle Inspections (DVIR) Status in `pegasusX`
- **Critical Finding**: There is NO dedicated `VehicleInspections` or `DVIR` table in `pegasusX/apps/backend-go/schema/spanner.ddl`.
- Vehicle safety management in `pegasusX` is performed purely reactively via admin flags (`UnavailableReason = 'MAINTENANCE'` or `'TRUCK_DAMAGED'`).
- There is no driver-facing digital checklist, no tire/brake/CNG inspection gating, and no digital signature verification prior to departure.

### 2.6 Exact Client Surfaces in `pegasusX`
- **Warehouse Portal (Next.js)**:
  - `apps/warehouse-portal/app/vehicles/page.tsx:1-120`: List vehicles, Add Truck modal (`form.license_plate`, `form.vehicle_class`).
  - `apps/warehouse-portal/app/drivers/page.tsx:1-193`: Driver list, Add Driver with PIN generation, inline truck assignment dropdown.
  - `apps/warehouse-portal/app/fleet-live-map/page.tsx`: Real-time map displaying fleet telemetry.
- **Supplier Portal**:
  - `apps/supplier-portal/app/(portal)/fleet/page.tsx`: Live fleet overview and links to `/org-fleet`.
  - `apps/supplier-portal/app/org-fleet/page.tsx`: Multi-facility vehicle and personnel roster.
- **Android Warehouse & Driver**:
  - `apps/warehouse-app-android/.../ui/screens/vehicles/VehiclesScreen.kt` & `VehicleDetailScreen.kt`.
  - `apps/warehouse-app-android/.../ui/screens/drivers/DriversScreen.kt`.
  - `apps/driver-app-android/.../ui/`: Driver mobile execution screens.

---

## 3. Exhaustive Analysis of Fleet Management in `pegasus.x`

### 3.1 Database Schemas & Models in `pegasus.x`
`pegasus.x` contains a highly evolved, localized fleet data layer in PostgreSQL 16:

1. **Migration 025** (`pegasus.x/database/migrations/025_fleet_and_driver_lifecycle_management.sql`):
   - `vehicles` table (lines 9-38): Includes `payload_capacity_kg`, `has_refrigeration`, `refrig_min_celsius`, `refrig_max_celsius`, `fuel_type` (default `METHANE_CNG`), `fuel_tank_capacity`, `current_odometer_km`, `current_fuel_pct`, `operational_status` (`YARD_STANDBY`, `LOADING_AT_DOCK`, `ACTIVE_ON_ROAD`, `MAINTENANCE`, `OUT_OF_SERVICE`), `texosmotr_expiry`, `osago_insurance_expiry`.
   - `drivers` enhanced (lines 39-53): `warehouse_id`, `pinfl` (14-digit national biometric ID), `driver_license_number`, `license_categories` (`TEXT[]`), `medical_cert_expiry`, `pin_hash`, `cash_bag_limit_tiyins` (default `2500000000` = 25M UZS).
   - `driver_vehicle_assignments` (lines 54-74): Supports `REGULAR_SHIFT`, `HOT_SWAP_RESCUE`, `RELIEF_DRIVER`.
     - **Bijective Constraints**:
       ```sql
       CREATE UNIQUE INDEX idx_active_driver_assignment ON driver_vehicle_assignments (driver_id) WHERE released_at IS NULL;
       CREATE UNIQUE INDEX idx_active_vehicle_assignment ON driver_vehicle_assignments (vehicle_id) WHERE released_at IS NULL;
       ```
   - `vehicle_inspections` (lines 75-101): Complete digital DVIR. Fields include `odometer_km`, `fuel_level_pct`, `tires_pressure_status`, `brakes_status`, `lights_signals_status`, `cleanliness_sanitation`, `refrigeration_temp_celsius`, `cng_cylinder_seal_valid`, `fire_extinguisher_valid`, `walkaround_passed`, `is_safe_to_operate`, `defect_notes`, `driver_signature_hash`.
2. **Migration 013** (`pegasus.x/database/migrations/013_fleet_integrity_coldchain_and_blindspots.sql`):
   - `vehicle_axle_profiles`: `wheelbase_mm`, `curb_front_kg`, `curb_rear_kg`, `gawr_front_kg`, `gawr_rear_kg`, `min_steer_ratio_pct` (3L-CVRP axle load balance).
   - `cold_chain_telemetry`: Real-time compartment temperature and humidity tracking.
   - `dock_dwell_events`: Tracks loading dock turnaround and demurrage penalties.
3. **Migration 037** (`pegasus.x/database/migrations/037_delivery_exceptions_and_driver_rescues.sql`):
   - `fleet_rescue_incidents`: Tracks roadside breakdowns, stranded vehicles, rescue drivers, and ETAs.

### 3.2 Implemented Services & Endpoints in `pegasus.x`
- **Domain Package** (`pegasus.x/backend/internal/fleet/`):
  - `models.go` (1,189 lines): Full struct representations with bidirectional normalization (`Normalize()`) for desktop and mobile clients.
  - `service.go` (1,118 lines): `CreateVehicle`, `CreateDriver`, `AssignDriverToVehicle`, `ReleaseAssignment`, `SwapVehicle`, `SwapDriver`, `RecordInspection`, `ListInspections`, `AuthenticateDriver`, `GetDriverActiveRoute`.
  - `repository.go` (2,551 lines): Uses `*db.Pool` (`pgxpool.Pool`) with atomic transactions (`r.pool.RunInTx`).
- **HTTP Routing & API Handlers**:
  - `handlers_fleet.go` (554 lines): Exposes `/v1/vehicles`, `/v1/drivers`, `/v1/fleet/assignments`, `/v1/fleet/assignments/swap`, `/v1/fleet/assignments/swap-driver`, `/v1/fleet/inspections`, `/v1/fleet/dvir`.
  - `handlers_fleet_driver.go` (1,258 lines): Exposes complete mobile driver cockpit APIs including auth, route geometry, SOS rescue, cash bag reconciliation, and QR validation.
  - `router.go:606-676`: All endpoints wired under Chi router with `auth.RequireAuthWithKeyManager`.

---

## 4. Architectural Gaps & Discrepancies in `pegasus.x`

During our line-by-line inspection and test verification of `pegasus.x`, four specific architectural defects were uncovered:

```
+--------------------------------------------------------------------------------------------------+
|                                    DISCOVERED ARCHITECTURAL GAPS                                  |
+----+----------------------------+-----------------------------------+----------------------------+
| #  | Gap Classification         | Location                          | Impact                     |
+----+----------------------------+-----------------------------------+----------------------------+
| 1  | Cross-Contamination Import | `internal/outbox/relay.go:13`     | Compilation failure;       |
|    | (Forbidden Kafka in pgx)   | `cmd/server/main.go:20`           | violates boundary rule     |
+----+----------------------------+-----------------------------------+----------------------------+
| 2  | Ephemeral Pub/Sub instead   | `internal/fleet/service.go:353`   | Driver/portal drops events |
|    | of Redis 7 Streams (XADD)  | `internal/fleet/service.go:453`   | during cellular disconnect |
+----+----------------------------+-----------------------------------+----------------------------+
| 3  | Pre-Trip Gating Loophole   | `internal/dispatch/service.go:258`| Uninspected vehicles can   |
|    | (Allows NULL inspections)  |                                   | be dispatched on routes    |
+----+----------------------------+-----------------------------------+----------------------------+
| 4  | Shift Board UI Wiring      | `apps/warehouse-desktop`          | Missing visual 2-column    |
|    | (Manual form vs Drag-Drop) | `app/vehicles/page.tsx`           | pairing board interaction  |
+----+----------------------------+-----------------------------------+----------------------------+
```

### Detailed Breakdown of Gaps:

#### Gap 1: Forbidden Kafka Import in `pegasus.x`
- In `pegasus.x/backend/internal/outbox/relay.go:13`:
  ```go
  import (
      ...
      "github.com/pegasus-x/core/internal/kafka" // <-- DOES NOT EXIST & FORBIDDEN
  )
  ```
- Running `go test ./internal/fleet/...` fails with:
  `internal/outbox/relay.go:13:2: no required module provides package github.com/pegasus-x/core/internal/kafka`
- **Rule Violation**: System boundary rule explicitly mandates: *"Zero Cross-Contamination: NEVER import Spanner libraries, Spanner DDL, or Kafka into pegasus.x"*.
- **Required Fix**: Strip Kafka from `pegasus.x` outbox relay. The outbox relay in `pegasus.x` must read PostgreSQL `outbox_events` and publish directly to Redis Streams / WebSocket Hub.

#### Gap 2: Ephemeral Pub/Sub vs. Persistent Redis 7 Streams
- In `internal/fleet/service.go:353, 372, 453, 518, 597`:
  Events are dispatched via `s.redis.Publish(ctx, "events:FLEET", eventData)`.
- Standard Redis `PUBLISH` does not store messages in memory. If the driver's phone is in a dead-zone or the warehouse desktop disconnects during page navigation, the event is permanently lost.
- **Required Specification**: Port to Redis 7 Streams (`XADD`) with consumer groups (`XREADGROUP`) on stream `stream:fleet:events` and `stream:fleet:telemetry`.

#### Gap 3: Smart Dispatch Pre-Trip Gating Loophole
- In `pegasus.x/backend/internal/dispatch/service.go:258`:
  ```sql
  WHERE d.supplier_id = $1 
    AND d.on_shift = true
    AND (v.operational_status IS NULL OR v.operational_status IN ('YARD_STANDBY', 'LOADING_AT_DOCK', 'ACTIVE_ON_ROAD'))
    AND (vi.is_safe_to_operate IS NULL OR vi.is_safe_to_operate = true) -- <-- BUG: NULL ALLOWED
  ```
- Allowing `vi.is_safe_to_operate IS NULL` permits vehicles with NO inspection on the calendar day to be loaded and dispatched.
- **Required Specification**: Strictly enforce `vi.is_safe_to_operate = true AND vi.created_at >= CURRENT_DATE`.

---

## 5. Concrete Porting & Parity Specification for `pegasus.x`

### 5.1 Production PostgreSQL 16 Schema Specification

The following DDL represents the complete, hardened relational schema for `pegasus.x`:

```sql
-- ==============================================================================
-- PEGASUS.X SOVEREIGN FLEET & DRIVER SCHEMA (POSTGRESQL 16)
-- Engine: PostgreSQL 16 + TimescaleDB + PostGIS
-- Financials: Strict 64-bit integer minor units (tiyins/cents)
-- ==============================================================================

-- 1. FLEET VEHICLES
CREATE TABLE IF NOT EXISTS vehicles (
    vehicle_id              VARCHAR(64) PRIMARY KEY,
    supplier_id             VARCHAR(64) NOT NULL REFERENCES suppliers(supplier_id),
    warehouse_id            VARCHAR(64) NOT NULL REFERENCES warehouses(warehouse_id),
    license_plate           VARCHAR(32) NOT NULL,
    make_model              VARCHAR(128) NOT NULL,
    vehicle_class           VARCHAR(16) NOT NULL DEFAULT 'CLASS_B',
    max_volume_vu           NUMERIC(10, 2) NOT NULL DEFAULT 150.0,
    payload_capacity_kg     INT NOT NULL DEFAULT 3500,
    has_refrigeration       BOOLEAN NOT NULL DEFAULT FALSE,
    refrig_min_celsius      NUMERIC(5, 2),
    refrig_max_celsius      NUMERIC(5, 2),
    fuel_type               VARCHAR(20) NOT NULL DEFAULT 'METHANE_CNG',
    fuel_tank_capacity      NUMERIC(8, 2) NOT NULL DEFAULT 150.0,
    current_odometer_km     INT NOT NULL DEFAULT 0,
    current_fuel_pct        INT NOT NULL DEFAULT 100,
    operational_status      VARCHAR(32) NOT NULL DEFAULT 'YARD_STANDBY',
    unavailable_reason      VARCHAR(64),
    unavailable_note        TEXT,
    texosmotr_expiry        DATE,
    osago_insurance_expiry  DATE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_fuel_pct CHECK (current_fuel_pct BETWEEN 0 AND 100),
    CONSTRAINT chk_odometer_nonneg CHECK (current_odometer_km >= 0),
    CONSTRAINT chk_vehicle_class CHECK (vehicle_class IN ('CLASS_A', 'CLASS_B', 'CLASS_C', 'CLASS_D')),
    CONSTRAINT chk_fuel_type CHECK (fuel_type IN ('METHANE_CNG', 'PROPANE_LPG', 'DIESEL', 'PETROL', 'ELECTRIC')),
    CONSTRAINT chk_operational_status CHECK (operational_status IN ('YARD_STANDBY', 'LOADING_AT_DOCK', 'ACTIVE_ON_ROAD', 'MAINTENANCE', 'OUT_OF_SERVICE'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_vehicles_supplier_plate ON vehicles(supplier_id, license_plate);
CREATE INDEX IF NOT EXISTS idx_vehicles_wh_status ON vehicles(warehouse_id, operational_status);

-- 2. DRIVERS TABLE ENHANCEMENTS
CREATE TABLE IF NOT EXISTS drivers (
    driver_id               VARCHAR(64) PRIMARY KEY,
    supplier_id             VARCHAR(64) NOT NULL REFERENCES suppliers(supplier_id),
    warehouse_id            VARCHAR(64) REFERENCES warehouses(warehouse_id),
    name                    VARCHAR(128) NOT NULL,
    phone                   VARCHAR(32) NOT NULL,
    pinfl                   VARCHAR(14),
    driver_license_number   VARCHAR(32),
    license_categories      TEXT[] DEFAULT ARRAY['B', 'C'],
    medical_cert_expiry     DATE,
    pin_hash                VARCHAR(128),
    on_shift                BOOLEAN NOT NULL DEFAULT FALSE,
    max_volume_vu           NUMERIC(10, 2) DEFAULT 150.0,
    cash_bag_limit_tiyins   BIGINT NOT NULL DEFAULT 2500000000, -- 25M UZS
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_drivers_supplier_phone ON drivers(supplier_id, phone);
CREATE INDEX IF NOT EXISTS idx_drivers_wh_shift ON drivers(warehouse_id, on_shift);

-- 3. DYNAMIC SHIFT PAIRINGS (BIJECTIVE AT ANY TIME t)
CREATE TABLE IF NOT EXISTS driver_vehicle_assignments (
    assignment_id           VARCHAR(64) PRIMARY KEY,
    supplier_id             VARCHAR(64) NOT NULL REFERENCES suppliers(supplier_id),
    warehouse_id            VARCHAR(64) NOT NULL REFERENCES warehouses(warehouse_id),
    shift_date              DATE NOT NULL DEFAULT CURRENT_DATE,
    driver_id               VARCHAR(64) NOT NULL REFERENCES drivers(driver_id),
    vehicle_id              VARCHAR(64) NOT NULL REFERENCES vehicles(vehicle_id),
    paired_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    released_at             TIMESTAMPTZ,
    assignment_type         VARCHAR(32) NOT NULL DEFAULT 'REGULAR_SHIFT',
    swap_reason             VARCHAR(64),
    previous_vehicle_id     VARCHAR(64) REFERENCES vehicles(vehicle_id),
    notes                   TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_assignment_type CHECK (assignment_type IN ('REGULAR_SHIFT', 'HOT_SWAP_RESCUE', 'RELIEF_DRIVER'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_active_driver_assignment ON driver_vehicle_assignments(driver_id) WHERE released_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_active_vehicle_assignment ON driver_vehicle_assignments(vehicle_id) WHERE released_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_dva_wh_date ON driver_vehicle_assignments(warehouse_id, shift_date);

-- 4. DIGITAL DVIR (PRE-TRIP / POST-TRIP INSPECTIONS)
CREATE TABLE IF NOT EXISTS vehicle_inspections (
    inspection_id               VARCHAR(64) PRIMARY KEY,
    assignment_id               VARCHAR(64) REFERENCES driver_vehicle_assignments(assignment_id),
    vehicle_id                  VARCHAR(64) NOT NULL REFERENCES vehicles(vehicle_id),
    driver_id                   VARCHAR(64) NOT NULL REFERENCES drivers(driver_id),
    inspection_type             VARCHAR(16) NOT NULL DEFAULT 'PRE_TRIP',
    odometer_km                 INT NOT NULL,
    fuel_level_pct              INT NOT NULL,
    tires_pressure_status       VARCHAR(16) NOT NULL DEFAULT 'PASS',
    brakes_status               VARCHAR(16) NOT NULL DEFAULT 'PASS',
    lights_signals_status       VARCHAR(16) NOT NULL DEFAULT 'PASS',
    cleanliness_sanitation      VARCHAR(16) NOT NULL DEFAULT 'PASS',
    refrigeration_temp_celsius  NUMERIC(5, 2),
    cng_cylinder_seal_valid     BOOLEAN NOT NULL DEFAULT TRUE,
    fire_extinguisher_valid     BOOLEAN NOT NULL DEFAULT TRUE,
    walkaround_passed           BOOLEAN NOT NULL DEFAULT TRUE,
    is_safe_to_operate          BOOLEAN NOT NULL DEFAULT TRUE,
    defect_notes                TEXT,
    photo_evidence_urls         TEXT[],
    driver_signature_hash       VARCHAR(128) NOT NULL,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_inspection_type CHECK (inspection_type IN ('PRE_TRIP', 'POST_TRIP', 'SWAP_HANDOVER')),
    CONSTRAINT chk_insp_tires CHECK (tires_pressure_status IN ('PASS', 'WARN', 'FAIL')),
    CONSTRAINT chk_insp_brakes CHECK (brakes_status IN ('PASS', 'FAIL')),
    CONSTRAINT chk_insp_lights CHECK (lights_signals_status IN ('PASS', 'FAIL'))
);

CREATE INDEX IF NOT EXISTS idx_inspections_vehicle_created ON vehicle_inspections(vehicle_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_inspections_driver_created ON vehicle_inspections(driver_id, created_at DESC);
```

### 5.2 Redis 7 Streams & Keys Architecture

To ensure zero event loss and reliable reconnection over Uzbekistan 4G/LTE mobile networks, Redis 7 must be structured with explicit stream persistence and geohashing:

```
+----------------------------------------------------------------------------------------------------+
|                                    REDIS 7 DATA PLANE ARCHITECTURE                                 |
+------------------------------+------------+--------------------------------------------------------+
| Key / Stream Pattern         | Type       | Purpose & Invariants                                   |
+------------------------------+------------+--------------------------------------------------------+
| `stream:fleet:events`        | Stream     | Persistent append-only stream for all assignment, swap,|
|                              |            | and DVIR safety events (`XADD`). Retained for 7 days.  |
| `stream:fleet:telemetry`     | Stream     | Ingests high-frequency GPS pings from driver apps.     |
| `drivers:active`             | Geospatial | `GEOADD drivers:active <lng> <lat> <driver_id>`.       |
|                              |            | Polled by warehouse control tower radar map.           |
| `driver:presence:<driver_id>`| String     | Heartbeat key with TTL = 60s. Value: `ONLINE`/`ON_ROUTE`|
| `vehicle:state:<vehicle_id>` | Hash       | Cached status, current fuel %, odometer, driver ID.    |
| `fleet:lock:dispatch:<wh_id>`| String     | Distributed lock with NX PX = 15000ms for CVRP run.    |
+------------------------------+------------+--------------------------------------------------------+
```

#### Redis Stream Producer Implementation:
```go
func (c *Client) XAddFleetEvent(ctx context.Context, eventType string, payload map[string]any) (string, error) {
    values := map[string]any{
        "event": eventType,
        "ts":    time.Now().UTC().Format(time.RFC3339Nano),
    }
    for k, v := range payload {
        values[k] = v
    }
    return c.XAdd(ctx, &redis.XAddArgs{
        Stream: "stream:fleet:events",
        MaxLen: 100000,
        Approx: true,
        Values: values,
    }).Result()
}
```

### 5.3 Go Chi API Contracts and Handlers Specification

All endpoints are registered under `r.Route("/v1", ...)` protected by `auth.RequireAuthWithKeyManager`:

| Method | Route | Request Body | Response Body | HTTP Status |
|---|---|---|---|---|
| `GET` | `/v1/vehicles` | None (Query: `warehouse_id`, `status`) | `[]fleet.Vehicle` | `200 OK` |
| `POST` | `/v1/vehicles` | `fleet.CreateVehicleRequest` | `fleet.Vehicle` | `201 Created` |
| `PATCH` | `/v1/vehicles/{id}/status` | `{"status": "MAINTENANCE", "reason": "...", "note": "..."}` | `{"status": "updated"}` | `200 OK` |
| `GET` | `/v1/drivers` | None (Query: `warehouse_id`, `on_shift`) | `[]fleet.Driver` | `200 OK` |
| `POST` | `/v1/drivers` | `fleet.CreateDriverRequest` | `fleet.Driver` | `201 Created` |
| `POST` | `/v1/fleet/assignments` | `{"warehouse_id": "...", "driver_id": "...", "vehicle_id": "..."}` | `fleet.DriverVehicleAssignment` | `201 Created` |
| `POST` | `/v1/fleet/assignments/{id}/release` | None | `{"status": "released"}` | `200 OK` |
| `POST` | `/v1/fleet/assignments/swap` | `fleet.SwapVehicleRequest` | `fleet.DriverVehicleAssignment` | `200 OK` |
| `POST` | `/v1/fleet/assignments/swap-driver` | `fleet.SwapDriverRequest` | `fleet.DriverVehicleAssignment` | `200 OK` |
| `POST` | `/v1/fleet/inspections` | `fleet.VehicleInspection` | `fleet.VehicleInspection` | `201 Created` |
| `GET` | `/v1/fleet/inspections` | None (Query: `vehicle_id`, `limit`) | `[]fleet.VehicleInspection` | `200 OK` |
| `GET` | `/v1/driver/active-route` | None (Driver Token) | `fleet.DriverActiveRoute` | `200 OK` |

### 5.4 UI/UX Design System Compliance Specification

In accordance with [.agents/rules/ui-design-system.md](file:///Users/shakhzod/Desktop/V.O.I.D/.agents/rules/ui-design-system.md) and [DESIGN.md](file:///Users/shakhzod/Desktop/V.O.I.D/DESIGN.md):

1. **Color Palette & Contrast Rules**:
   - Tactical Canvas: `#09090B` (Dark background) / `#F8FAFC` (Operational light canvas).
   - Deep Obsidian Surfaces: `#121216` with crisp 1px hairline border `#22222C`.
   - Accent Primary: Electric Cobalt Blue (`#2563EB` / `#3B82F6`).
   - Warning / Alert: Safety Orange (`#FF7A1A`).
   - Success / Active Radar: Tactical Lime (`#E2FD52`) / Emerald (`#10B981`).
   - Monospace Numeric Typography: All plate numbers, odometers, capacities, and currency tiyins MUST use `font-mono tabular-nums`.

2. **Tauri Desktop 3-Column Control Tower Architecture**:
   - **Column 1 (Left Nav Rail)**: 64px collapsed tactical navigation rail with fleet radar badge.
   - **Column 2 (Center Density Feed)**:
     - Header KPI strip with `MetricCard` components (Total Trucks, Standby, At Dock, Active on Road, Maintenance).
     - 4-segment tab bar: (1) Avtopark (Vehicles), (2) Haydovchilar (Drivers), (3) Smena Tarkibi (Shift Pairing), (4) Texnik Ko'rik (DVIR Logs).
     - Uzbekistan License Plate Blueprint: `[UZ | 01 | 772 AAA]` styled with 1px border `#22222C`, uppercase monospace font, and blue national flag bar.
   - **Column 3 (Right Inspector Drawer)**:
     - Slides out when clicking any vehicle or driver card.
     - Displays real-time fuel tank gauge (`GaugeChart`), volumetric utilization (`LedProgressBar`), 30-day DVIR history timeline (`ActivityTimeline`), and immediate "Ta'mirga Yuborish" / "Smenani Bo'shatish" action buttons.

3. **Driver Mobile App Cockpit (Android & iOS)**:
   - Mandatory modal interlock on shift start: `PreTripDVIRDialog.kt` (Android) / `PreTripDVIRModalView.swift` (iOS).
   - 5-point checklist with toggle switches (Tires, Brakes, Lights, Sanitation, CNG Cylinder Seal).
   - Odometer text field + 0-100% fuel level slider.
   - If inspection fails, app renders high-contrast crimson banner: *"Avtotransport nosoz. Marshrut boshlash taqiqlanadi"* and prevents departure gate unlock.

---

## 6. Comprehensive Parity & Delta Matrix

| Operational Vector | `pegasusX` (Global Spanner + Kafka) | `pegasus.x` (Sovereign PostgreSQL 16 + Redis 7) | Parity Status & Action Required |
|---|---|---|---|
| **Vehicles Entity** | Spanner `Vehicles` table; `MaxVolumeVU`, `VehicleClass`, `IsActive`, `UnavailableReason`. | PostgreSQL `vehicles` table; adds `has_refrigeration`, `fuel_type` (CNG/LPG), `payload_capacity_kg`, `texosmotr_expiry`. | **Parity Met & Exceeded** in `pegasus.x`. |
| **Driver Licensing** | Spanner `Drivers` table; `Phone`, `Name`, `PinHash`, `VehicleId`. | PostgreSQL `drivers` table; adds `pinfl` (14-digit), `driver_license_number`, `license_categories` (B/C/CE), `cash_bag_limit_tiyins`. | **Parity Met & Exceeded** in `pegasus.x`. |
| **Daily Shift Pairing** | Updated directly on `Drivers.VehicleId` in Spanner; guarded by `fleet_guards.go`. | Dedicated `driver_vehicle_assignments` table with partial unique indexes (`WHERE released_at IS NULL`). | **Superior in `pegasus.x`**; enforces mathematical bijectivity. |
| **Mid-Shift Hot Swap** | Handled via `apps/backend-go/driver/rescue.go` (`RESCUE_REQUESTED`, order reassignment). | Implemented via `SwapVehicle` & `SwapDriver` in `internal/fleet/service.go` and `fleet_rescue_service.go`. | **Full Logic Parity**; transfers open orders atomically. |
| **Digital DVIR Gate** | Missing from Spanner schema; manual admin flag toggles only. | Full `vehicle_inspections` table, Android `PreTripDVIRDialog.kt`, iOS `PreTripDVIRModalView.swift`. | **Built in `pegasus.x`**; missing strict dispatch query gate. |
| **Outbox Messaging** | Spanner `OutboxEvents` table + Kafka Outbox worker. | PostgreSQL `outbox_events` table + Redis 7 Streams. | **Fix Required**: Strip invalid Kafka import in `internal/outbox/relay.go`. |
| **Telemetry & Radar** | Spanner `RouteETAs` + In-memory driver hubs. | Redis Geospatial `GEOADD drivers:active` + PostGIS coords. | **Parity Met**; add `stream:fleet:telemetry`. |
| **Desktop UI View** | Next.js portal pages with basic forms. | Tauri v2 Desktop (Next.js 15) with rich tables & modals. | **Parity Met**; wire visual drag-and-drop shift canvas. |

---

## 7. Recommended Action Plan for Engineering Team

1. **Step 1: Eliminate Outbox Cross-Contamination**:
   - Remove `"github.com/pegasus-x/core/internal/kafka"` from `pegasus.x/backend/internal/outbox/relay.go` and `backend/cmd/server/main.go`.
   - Update `RelayWorker` to read `outbox_events` and publish directly to Redis Streams `stream:fleet:events` and Redis Pub/Sub for WebSockets.
2. **Step 2: Close Dispatch Pre-Trip Gating Loophole**:
   - In `pegasus.x/backend/internal/dispatch/service.go:258`, modify the query from `(vi.is_safe_to_operate IS NULL OR vi.is_safe_to_operate = true)` to:
     ```sql
     AND vi.is_safe_to_operate = true 
     AND vi.created_at >= CURRENT_DATE
     ```
3. **Step 3: Upgrade Real-Time Telemetry to Redis 7 Streams**:
   - Wire `XAddFleetEvent` in `internal/fleet/service.go` to emit to `stream:fleet:events`.
4. **Step 4: Wire Tauri Desktop Smena Board**:
   - Complete the interactive visual drag-and-drop shift pairing board in `apps/supplier-desktop` and `apps/warehouse-desktop` connecting to `POST /v1/fleet/assignments`.
