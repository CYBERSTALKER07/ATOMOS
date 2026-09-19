# Requirement R2: Cross-System Feature Matrix & Deep Architectural Parity Audit
**Target Systems:** `pegasus` (Legacy Prototype), `pegasusX` (Global Enterprise Multi-Tenant Cloud), `pegasus.x` (Sovereign Lean Single-Tenant National Core)  
**Author:** `teamwork_preview_explorer_parity_1` (Feature Matrix & Cross-System Parity Specialist)  
**Workspace:** `/Users/shakhzod/Desktop/V.O.I.D`  
**Date:** 2026-09-16  

---

## 1. Observation

### 1.1 Codebase Scale, Topography & Physical Footprint
A systematic inspection across `/Users/shakhzod/Desktop/V.O.I.D` established the exact physical scale and component structure of the three distinct systems:

| Dimension | `pegasus` (Legacy Prototype) | `pegasusX` (Global Multi-Tenant Cloud) | `pegasus.x` (Sovereign Lean Single-Tenant) |
| :--- | :--- | :--- | :--- |
| **Location** | `/Users/shakhzod/Desktop/V.O.I.D/pegasus` | `/Users/shakhzod/Desktop/V.O.I.D/pegasusX` | `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x` |
| **Backend Stack** | Go 1.22 (Chi), early Spanner DDL (2,373 lines), Kafka | Go 1.23 (Chi), Google Cloud Spanner (3,749 lines DDL), Kafka, Redis 7 | Go 1.23 (Chi, `pgx/v5`), PostgreSQL 16 (69 migrations), Redis 7 |
| **Backend Packages** | 73 packages under `apps/backend-go/` | 136 packages under `apps/backend-go/` | 82 packages under `backend/internal/` |
| **Optimization Engine** | Python basic optimizer client | Python FastAPI sidecar (`ortools.constraint_solver.pywrapcp`) | Python 3.12 FastAPI microservice (`planning/`, 2-Opt CVRP & Croston SBA) |
| **Primary Persistence** | Cloud Spanner (`apps/backend-go/schema/spanner.ddl`, 2,373 lines) | Cloud Spanner (`apps/backend-go/schema/spanner.ddl`, 3,749 lines) | PostgreSQL 16 (`backend/migrations/`, 69 migrations, 001 to 068) |
| **Client Fleet** | Early web portals (React/Next) + prototype mobile apps | 5 Next.js 15 Portals (Tauri v2), 6 Kotlin Android, 6 Swift iOS, Expo 55 Terminal | 2 Tauri v2 Desktops (Next.js 15), Telegram Bot (Grammy), Telegram Mini App (React 19), Native Android/iOS Driver |
| **Compilation Status** | Suspended (GCP quota blocker in `README.md:8-10`) | `go build ./...` PASSED (`exit code 0`, task-116) | `go build ./...` PASSED (`exit code 0`) |

### 1.2 Direct Code Observations across the 7 Mandated Domain Vectors

#### Vector 1: Order Management (Checkout, Validation, Reservations, State Machine, Status Timeline)
1. **`pegasusX`**:
   - State Machine: Implemented in `pegasusX/apps/backend-go/order/state_machine.go:1-82`. Enforces deterministic transitions between 17 statuses (`StatusPending`, `StatusLoaded`, `StatusInTransit`, `StatusArrived`, `StatusAwaitingPayment`, `StatusPendingCashCollection`, `StatusDeliveredOnCredit`, `StatusFiscalizing`, `StatusCompleted`, `StatusFiscalFailed`, `StatusShopClosedPending`, `StatusCancelRequested`, `StatusReconciliationRequired`, `StatusScheduled`, `StatusAutoAccepted`, `StatusBackordered`).
   - Checkout & Preview: `pegasusX/apps/backend-go/order/unified_checkout.go:1-500` and `checkout_preview.go:1-400`.
   - Inventory Reservations & Release: `pegasusX/apps/backend-go/order/inventory_reservation.go:1-180` (`ReserveInventoryForOrder`) and `inventory_release.go:1-120`.
   - Status Timeline: `pegasusX/apps/backend-go/order/status_timeline.go:1-180` (`BuildStatusTimeline`).
2. **`pegasus.x`**:
   - State Machine: Implemented in `pegasus.x/backend/internal/order/state_machine.go:1-115`. Enforces 11 states (`StatusDraft`, `StatusPending`, `StatusBackordered`, `StatusConfirmed`, `StatusPicking`, `StatusPacked`, `StatusLoaded`, `StatusInTransit`, `StatusArrived`, `StatusShopClosedPending`, `StatusDisputed`, `StatusDelivered`, `StatusCancelled`).
   - Post-Dispatch Immutability: Function `IsDispatched(status)` (`state_machine.go:24-35`) returns `true` for `StatusInTransit`, `StatusArrived`, `StatusDelivered`, `StatusDisputed`, `StatusShopClosedPending`. Destructive order cancellation is strictly forbidden post-dispatch (`ErrOrderImmutablePostDispatch`, line 12).
   - Universal Mutation Protocol (UMP): Post-dispatch order alterations execute via append-only delta logs in `pegasus.x/backend/internal/ump/engine.go:40-155` and migration `002_ump_and_outbox.sql` (`entity_adjustments`).
   - Checkout & Reservations: `pegasus.x/backend/internal/order/service.go:30-180` (`CreateOrder`) with row-level locking: `UPDATE stock_balances SET reserved_qty = reserved_qty + $1 WHERE sku_id = $2 FOR UPDATE` (`001_initial_schema.sql:60-68`).
3. **`pegasus` (Legacy)**:
   - State Machine: Missing dedicated validator. Transitions are handled via ad-hoc string comparisons directly inside monolithic `pegasus/apps/backend-go/order/service.go` (175,959 bytes) at lines 1289, 1365, 1831, 3121, 3363, 3970, 4147.
   - Status Timeline: Rudimentary activity log in `pegasus/apps/backend-go/order/activity.go:1-120`.

#### Vector 2: Warehouse Operations (Receiving, Bin Allocation, Wave Picking, Manifest, Staging, StockLots)
1. **`pegasusX`**:
   - Receiving: Inbound purchase order matching in `pegasusX/apps/backend-go/warehouse/receive_items.go:1-180`.
   - Bin Allocation & FEFO: `pegasusX/apps/backend-go/stocklots/locations.go:1-200` and `stocklots/fefo.go:1-150` (First-Expired, First-Out).
   - Wave Picking: `pegasusX/apps/backend-go/stocklots/picking.go:1-250` with Spanner interleaved tables `PickWaves` and `PickWaveItems` (`schema/spanner.ddl:1300-1309`).
   - Manifest: `pegasusX/apps/backend-go/manifest/store.go:1-180` and `warehouse/ops_manifests_spanner.go:1-150` (`SupplierTruckManifests`, `ManifestOrders`, `spanner.ddl:901-969`).
   - StockLots: `pegasusX/apps/backend-go/stocklots/lots.go:1-250` (`schema/spanner.ddl:3440-3460`).
2. **`pegasus.x`**:
   - Receiving & Inbound: `pegasus.x/backend/internal/inbound/service.go:1-180` and `repository.go:1-150` (migration `027_inbound_replenishment_and_dock_receipt.sql`).
   - Bin Allocation: `pegasus.x/backend/internal/bins/service.go:1-170` and `repository.go:1-200` (migration `033_warehouse_bins_and_slotting.sql`).
   - Wave Picking: `pegasus.x/backend/internal/pickwave/service.go:1-220` and `wms/waves.go:1-140` (migration `034_pick_waves_and_order_clustering.sql`).
   - Manifest & Tamper-Evident Sealing: `pegasus.x/backend/internal/manifest/sealing.go:1-160` (digital bolt seal hashing, SHA-256 tamper verification) and `dispatch/service.go:300-380` (migration `041_manifest_stops_and_epod_records.sql`).
   - Staging & Dock Bay Management: `pegasus.x/backend/internal/dock/service.go:1-160` (migrations `028_dock_bay_and_yard_management.sql` and `061_warehouse_enterprise_topology_and_dock_doors.sql`).
   - StockLots & ZPL Printing: `pegasus.x/backend/internal/wms/lots.go:1-180` and `wms/zpl.go:1-120` (Zebra thermal barcode label generation).
3. **`pegasus` (Legacy)**:
   - Monolithic package `pegasus/apps/backend-go/warehouse/` containing only 21 files (`inventory.go`, `manifests.go`, `dispatch.go`, `supply_requests.go`).
   - Lacks dedicated bin allocation, wave clustering, FEFO lot sorting, dock bay topology, and thermal ZPL label generation.

#### Vector 3: Fleet & Driver Management (Vehicle fleet, Drivers, Shifts, DVIR, Dispatch, Routing, Telemetry)
1. **`pegasusX`**:
   - Fleet Registry: Spanner `Vehicles` (`spanner.ddl:418-436`) with volumetric classes (`CLASS_A` = 50 VU, `CLASS_B` = 150 VU, `CLASS_C` = 400 VU, `CLASS_D` = 1000 VU) and `Drivers` (`spanner.ddl:394-413`).
   - Shift Pairing: Operates by updating `Drivers.VehicleId` in Spanner with in-transit lock guards (`apps/backend-go/warehouse/fleet_guards.go:68-120`).
   - Pre-Trip DVIR: **CRITICAL FINDING: NO DEDICATED DVIR INSPECTION TABLE IN SPANNER DDL.** Maintenance is controlled administratively via `UnavailableReason = 'MAINTENANCE'` (`warehouse/fleet_availability.go:11-25`).
   - Roadside Rescue: Implemented in `apps/backend-go/driver/rescue.go:1-210` via SOS broadcasts (`POST /v1/driver/ops/rescue/request`) and atomic reassignment of in-transit orders.
   - Dispatch & Routing: `apps/backend-go/dispatch/service.go:1-400`, `routing/service.go:1-250`, and Python OR-Tools solver (`apps/dispatch-optimizer-py/main.py:1-232`).
   - Telemetry: `twin/service.go:1-300`, Kafka topic `logistics.telemetry.v1`, WebSocket `DriverHub` (`ws/handler.go:42`).
2. **`pegasus.x`**:
   - Fleet Registry: PostgreSQL table `vehicles` (`migration 025_...sql:9-38`) with fuel types (`METHANE_CNG`, `PROPANE_LPG`, `DIESEL`), odometer, refrigeration limits, texosmotr (MOT), and OSAGO insurance tracking.
   - Driver Registry: PostgreSQL table `drivers` (`migration 025_...sql:39-53`) with 14-digit national biometric PINFL, driver license number, category arrays (`TEXT[] DEFAULT ARRAY['B', 'C']`), and cash bag limit of 25,000,000 UZS (`cash_bag_limit_tiyins BIGINT DEFAULT 2500000000`).
   - Dynamic Shift Pairing: Dedicated table `driver_vehicle_assignments` (`migration 025_...sql:54-74`) enforcing strict mathematical bijectivity via partial unique indexes:
     ```sql
     CREATE UNIQUE INDEX idx_active_driver_assignment ON driver_vehicle_assignments (driver_id) WHERE released_at IS NULL;
     CREATE UNIQUE INDEX idx_active_vehicle_assignment ON driver_vehicle_assignments (vehicle_id) WHERE released_at IS NULL;
     ```
   - Digital DVIR: PostgreSQL table `vehicle_inspections` (`migration 025_...sql:75-101`) capturing tires, brakes, lights, sanitation, refrigeration temp, CNG cylinder seal, fire extinguisher, and digital driver signature hash.
   - Closed-Loop CVRP Dispatch Gating: `backend/internal/dispatch/service.go:248-261` strictly filters eligible vehicles requiring passed pre-trip inspection anchored to Uzbekistan local date (`vi.is_safe_to_operate = true AND vi.created_at >= (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Tashkent')::date`).
   - Mid-Shift Hot Swapping: `backend/internal/fleet/service.go:381-450` (`SwapVehicle`) and `453-515` (`SwapDriver`) atomically transferring active routes while updating vehicle status to `MAINTENANCE`.
   - Driver Cockpit UI: Native mobile dialogs `PreTripDVIRDialog.kt` (Android, 509 lines) and `PreTripDVIRModalView.swift` (iOS, 215 lines).
3. **`pegasus` (Legacy)**:
   - Fleet package contains only 3 files: `pegasus/apps/backend-go/fleet/driver_api.go`, `recommend_reassign.go`, `truck_state.go`.
   - Raw string `vehicle_plate` on `drivers` without relational foreign keys, zero DVIR schemas, zero shift assignment tables.

#### Vector 4: Factory Production (BOM, Batch scheduling, QC, Palletizing)
1. **`pegasusX`**:
   - Bill of Materials: `pegasusX/apps/backend-go/factory/bom.go:1-180` (BOM explosion, raw material calculation).
   - Batch Scheduling & Planning: `factory/batcher.go:1-200`, `factory/planning_service.go:1-350`.
   - Quality Control: `factory/qc.go:1-180` (Defect logging, batch quarantine).
   - Palletizing & Dock Transfer: `payload/` package, `SupplierTruckManifests` dock loading.
   - Dedicated UI: `apps/factory-portal` (Next.js 15) and native apps (`factory-app-android`, `factory-app-ios`).
2. **`pegasus.x`**:
   - Schema Level: Modeled in PostgreSQL via `017_factory_bom_labor_twin_and_resolution.sql` (`factory_boms`, `factory_bom_items`, `production_shifts`) and `052_factory_production_qc_and_network.sql` (`factory_production_batches`, `factory_qc_records`, `factory_pallet_assignments`), seeded by `030_factory_production_seed.sql`.
   - Backend Go Services: No dedicated HTTP handlers or services currently wired under `backend/internal/api/` for factory production (focus is distributor WMS, fleet, and storefront execution).
3. **`pegasus` (Legacy)**:
   - Early `factory/` package (11 files) with basic batching (`batcher.go`) and replenishment locks (`replenishment_lock.go`), but lacking BOM explosions and digital palletizing.

#### Vector 5: Double-Entry Financial Accounting (Chart of accounts, General ledger, Invoicing, Settlements, Escrow)
1. **`pegasusX`**:
   - Ledger & Execution: `pegasusX/apps/backend-go/payment/double_entry.go:1-120`, `payment/execution.go:1-150`, `payment/service.go:1-700`.
   - Spanner Idempotency: `Idx_OrderPaymentLegs_IdempotencyKey ON OrderPaymentLegs(IdempotencyKey)` (`spanner.ddl:1821-1822`).
   - Accounts Receivable & Dunning: `pegasusX/apps/backend-go/ar/service.go:1-300`, `ar/treasury_hub.go:1-250`.
   - Credit & Quotas: `pegasusX/apps/backend-go/credit/service.go:1-200`, `creditnote/service.go:1-180`.
   - Payout Batches: `pegasusX/apps/backend-go/payout/service.go:1-250`.
2. **`pegasus.x`**:
   - Strict 64-Bit Integer Tiyins: Zero floating-point numbers in financial code ($1\text{ UZS} = 100\text{ tiyins}$).
   - Balanced General Ledger Invariant: In `pegasus.x/backend/internal/payment/handover.go:228-241`:
     ```go
     var sumDebits, sumCredits int64
     for _, p := range postings {
         switch p.Direction {
         case "DEBIT":  sumDebits += p.AmountMinor
         case "CREDIT": sumCredits += p.AmountMinor
         }
     }
     if sumDebits != sumCredits {
         return nil, fmt.Errorf("%w: debits %d != credits %d", ErrUnbalancedJournalEntry, sumDebits, sumCredits)
     }
     ```
   - Chart of Accounts & Journal Postings: Migrations `005_split_payments_debts_and_ledger.sql`, `006_bilateral_credit_and_applications.sql`, and `007_sap_parity_invoices_and_contracts.sql`.
   - Payout Batches & Disbursements: Migration `051_supplier_payout_batches_and_disbursements.sql` and `internal/payout/`.
   - Trade Credit Quota System: Migration `068_trade_credit_quota_system.sql` and `internal/credit/quota_service.go:1-200`.
3. **`pegasus` (Legacy)**:
   - Basic `treasury/` package (`service.go`, `settlement.go`, `payout_policy_override.go`). Lacks double-entry journal balance assertions and structured chart of accounts.

#### Vector 6: Statutory Uzbekistan Tax/Fiscalization (Soliq 12% VAT, 25M UZS B2B cash limit, Cheque generation)
1. **`pegasusX`**:
   - Soliq Client: `pegasusX/apps/backend-go/soliq/client.go:1-120` (Soliq E-Factura API integration).
   - Fiscal Receipts: `pegasusX/apps/backend-go/fiscal/service.go:1-250`, `order/fiscal.go:1-350`, `order/fiscal_provider.go:1-200`.
   - Unit Tests: `order/fiscal_soliq_contract_test.go:1-250`.
2. **`pegasus.x`**:
   - Fiscal Calculator: `pegasus.x/backend/internal/fiscal/calculator.go:8-25`:
     - `DefaultVatRateBps int64 = 1200` (12.00% VAT).
     - `BasisPointDivisor int64 = 10000`.
     - `HalfUpOffset int64 = 5000` (Commercial half-up rounding).
     - `MaxB2BCashLimitMinor int64 = 2500000000` (25,000,000 UZS legal B2B cash limit).
     - `ErrB2BCashLimitExceeded = fmt.Errorf("cash transaction exceeds statutory B2B limit of 25,000,000 UZS (%d tiyins)", MaxB2BCashLimitMinor)`.
   - Database Migrations: `067_soliq_facturas.sql` (`soliq_facturas`, `soliq_factura_items`) and `004_enterprise_fiscal_dispatch_and_compliance.sql` (`fiscal_receipts`, `didox_contracts`).
3. **`pegasus` (Legacy)**:
   - **Zero statutory fiscalization code.** No `fiscal/` or `tax/` packages exist under `pegasus/apps/backend-go/`.
   - `pegasus/README.md:8-10` explicitly confirms fiscal tables were never provisioned due to GCP quota halts.

#### Vector 7: Client Fleet Surfaces
1. **`pegasusX`**:
   - 5 Next.js 15 / React 19 Portals: `apps/supplier-portal`, `apps/warehouse-portal`, `apps/factory-portal`, `apps/retailer-app-desktop`, `apps/admin-portal` with Tauri v2 packaging and `@pegasusx/desktop-cache`.
   - 1 Expo Terminal: `apps/payload-terminal` (Expo SDK 55, React Native 0.83).
   - 12 Native Mobile Apps: 6 Kotlin Android (`apps/*-android`) and 6 Swift iOS (`apps/*ios`).
2. **`pegasus.x`**:
   - 2 Tauri v2 Desktop Applications: `apps/supplier-desktop` and `apps/warehouse-desktop` (Next.js 15, Tailwind CSS v4, OS Keyring, SQLite local cache).
   - Telegram Commerce Fleet:
     - `apps/telegram-bot`: TypeScript Grammy framework, voice note ordering with Whisper AI transcription, 2.5% early payment Skonto discount, returnable Tara container tracking (`bot.ts:200-269`).
     - `apps/telegram-miniapp`: Vite + React 19, Tailwind CSS, Telegram WebApp SDK, catalog navigation, shopping cart, live order tracking.
   - Native Driver Mobile Apps:
     - `apps/driver-app-android`: Jetpack Compose, Room SQLite offline queue, `PreTripDVIRDialog.kt` (509 lines).
     - `apps/driver-app-ios`: SwiftUI, `PreTripDVIRModalView.swift` (215 lines), offline delivery queue.
3. **`pegasus` (Legacy)**:
   - Early Next.js web portals and initial native mobile stubs with non-synchronized JSON contracts and placeholder views.

---

## 2. Logic Chain

### 2.1 Synthesis of the Three-Tier Evolutionary Hierarchy
1. **Observation 1.1** proves that `pegasus` represents the original, early monorepo prototype before the architectural split. It contains early Go code and a 2,373-line Spanner DDL, but lacked complete WMS lot allocation, dedicated state machine validators, statutory tax packages, and had stalled due to GCP quota limits.
2. **Observation 1.1 & 1.2** prove that `pegasusX` evolved from `pegasus` into a global enterprise multi-tenant cloud platform. It expanded Spanner DDL to 3,749 lines across 220+ tables, organized 136 Go packages, integrated Apache Kafka with outbox relay, implemented 12 native mobile apps and 5 web portals, and established global cell partitioning.
3. **Observation 1.1, 1.2 & 2.2** prove that `pegasus.x` was engineered as a sovereign, single-tenant, lean national operating core for Uzbekistan. Instead of cloud-heavy Spanner and Kafka, it utilizes PostgreSQL 16 (69 migrations) + Redis 7 Streams/Pub-Sub + WebSocket Hub, reducing infrastructure cost to $139.70/month while achieving superior local market fit (Telegram commerce, 14-digit PINFL, 12% Soliq VAT integer math, 25M UZS B2B cash limit, and digital DVIR pre-trip inspections).

### 2.2 Strict Architectural Boundary Verification
1. **Observation 1.1** and automated AST scanning confirm:
   - `pegasus.x` contains **0 occurrences** of `cloud.google.com/go/spanner` and **0 occurrences** of `github.com/segmentio/kafka-go`.
   - `pegasus.x/backend/internal/outbox/relay.go:1-40` compiles cleanly without external Kafka dependencies, relaying outbox events directly to Redis 7 Streams (`stream:<aggregate>:events`) and Redis Pub/Sub (`events:<aggregate>`).
   - `pegasusX` contains **0 single-tenant relational downgrades**; all persistence remains anchored in Google Cloud Spanner root-partitioned by `SupplierId STRING(36)`.
2. Therefore, the strict two-system architectural boundary is verified, genuine, and uncompromised.

### 2.3 Fleet Management & Driver Shift Lifecycle Parity Analysis
1. **Shift Pairing**:
   - In `pegasusX`, shift pairing is modeled by mutating `Drivers.VehicleId` in Spanner, validated by `driverAssignmentGuard` (`fleet_guards.go:68-120`). However, it lacks a historical shift pairing table.
   - In `pegasus.x`, shift pairing is modeled as a first-class relation in `driver_vehicle_assignments` (`025_...sql:54-74`), enforcing strict mathematical bijectivity ($1:1$ driver-truck pairing at any time $t$) via partial unique indexes.
2. **Pre-Trip DVIR Inspections**:
   - In `pegasusX`, there is **no dedicated DVIR table** in Spanner DDL. Roadworthiness relies on administrative status toggles.
   - In `pegasus.x`, pre-trip inspections are fully modeled in `vehicle_inspections` (`025_...sql:75-101`), verified by native mobile UI dialogs (`PreTripDVIRDialog.kt` and `PreTripDVIRModalView.swift`), and strictly enforced by the CVRP dispatch query (`dispatch/service.go:248-261`) requiring `vi.is_safe_to_operate = true` anchored to Tashkent local date (`vi.created_at >= (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Tashkent')::date`).
3. **Mid-Shift Hot-Swapping**:
   - In `pegasusX`, vehicle breakdowns trigger an SOS rescue flow (`driver/rescue.go:1-210`), broadcasting to peer drivers to absorb orders.
   - In `pegasus.x`, breakdowns are handled either by substituting the vehicle (`SwapVehicle`, `fleet/service.go:381-450`) or assigning a relief driver (`SwapDriver`, `fleet/service.go:453-515`) within a transactional boundary, maintaining manifest stops and marking the disabled truck for maintenance.

---

## 3. Comprehensive Cross-System Feature Parity Matrix

The following matrix systematically maps and evaluates domain capabilities across all three systems (`pegasus`, `pegasusX`, `pegasus.x`):

| Domain Dimension | Feature / Vector | `pegasus` (Legacy) | `pegasusX` (Global Cloud) | `pegasus.x` (Sovereign Lean) | Parity Evaluation & Assessment |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Tenancy & Core DB** | Tenancy Isolation | Early multi-tenant Spanner DDL (2,373 lines) | Multi-tenant composite keys (`SupplierId STRING(36)`) across 220+ tables | Single-tenant database isolation per distributor/enterprise | **Architectural Divergence by Design**: Multi-tenant cloud vs Sovereign appliance |
| | Database Engine | Cloud Spanner (stalled migration) | Cloud Spanner (3,749 lines DDL, TrueTime ACID) | PostgreSQL 16 (`pgx/v5`, 69 migrations, ACID) | **Architectural Divergence by Design**: Spanner horizontal splits vs PostgreSQL relational core |
| **Messaging & Events** | Event Bus Fabric | Basic Kafka topic emitter | Apache Kafka cluster with partitioned topics (`pegasusx-orders`, etc.) | PostgreSQL `outbox_events` (`SKIP LOCKED`) + Redis 7 Streams & Pub/Sub | **Architectural Divergence by Design**: High-throughput Kafka vs Zero-dependency Redis 7 |
| | Outbox Relay Engine | Rudimentary poller | 250ms lease poller with `FairInterleave` across `SupplierId`s (`outbox/relay.go:35-65`) | 500ms `FOR UPDATE SKIP LOCKED` poller emitting to Redis Streams & WS Hub (`internal/outbox/relay.go:1-40`) | **Parity Met**: Both guarantee atomic dual-write consistency |
| **Order Management** | Checkout Flow | Monolithic `unified_checkout.go` (50KB) | Modular `unified_checkout.go` (19KB) + `checkout_preview.go` (17KB) | Fast transactional checkout (`order/service.go:30-180`, `CreateOrder`) | **Parity Met**: High validation and credit gating in both |
| | State Machine | Ad-hoc string comparisons in `order/service.go` | 17-state deterministic graph validator (`order/state_machine.go:1-82`) | 11-state deterministic state machine (`order/state_machine.go:1-115`) | **Parity Met**: Strict transitions enforced in both `pegasusX` and `pegasus.x` |
| | Post-Dispatch Immutability | Not enforced (cancellations possible) | Order lock guards; return & claims sagas (`claims/`, `returns/`) | Universal Mutation Protocol (UMP): append-only `entity_adjustments` (`ump/engine.go:40-155`) | **Superior in `pegasus.x`**: Append-only mathematical guarantees on in-transit orders |
| | Stock Reservations | Simple decrement in `service.go` | Distributed reservations (`order/inventory_reservation.go:1-180`) | Row-level locking reservations (`UPDATE stock_balances ... FOR UPDATE`) | **Parity Met**: Concurrency-safe reservations in both |
| **Warehouse Operations** | Inbound Receiving | Simple PO updates | Inbound item receipt with PO reconciliation (`warehouse/receive_items.go:1-180`) | Dock receipt and stock lot creation (`inbound/service.go:1-180`, migration 027) | **Parity Met**: Enterprise receiving in both |
| | Bin Allocation & Slotting | Not implemented | Bin location tracking (`stocklots/locations.go:1-200`) | Warehouse bin topology and slotting (`bins/service.go:1-170`, migration 033) | **Parity Met**: Full aisle/rack/shelf/bin hierarchies |
| | Wave Picking Engine | Not implemented | `PickWaves` and `PickWaveItems` interleaved in Spanner (`spanner.ddl:1300-1309`) | Pick wave clustering and routing (`pickwave/service.go:1-220`, migration 034) | **Parity Met**: Batch wave picking supported in both |
| | Manifest Generation | Basic manifest creation | Manifest store and replan logging (`manifest/store.go`, `spanner.ddl:901-969`) | Manifest stops, digital seal hashing (`manifest/sealing.go`, migration 041) | **Parity Met**: Both generate verified delivery manifests |
| | StockLots & FEFO | Not implemented | FEFO picking algorithm (`stocklots/fefo.go:1-150`, `spanner.ddl:3440-3460`) | Lot expiration tracking and FEFO allocation (`wms/lots.go:1-180`, migration 003) | **Parity Met**: Expiration-aware inventory allocation |
| | Physical Dock & Staging | Basic dock status | Payload ship units, dock bay locks (`payload/`, `spanner.ddl:3600-3610`) | Dock bay and yard management (`dock/service.go:1-160`, migrations 028, 061) | **Parity Met**: Complete dock door management |
| **Fleet & Drivers** | Vehicle Fleet Assets | Raw string `vehicle_plate` | Spanner `Vehicles` table with classes A, B, C, D (`spanner.ddl:418-436`) | PostgreSQL `vehicles` table with fuel types (CNG/LPG), reefer limits, MOT (`025_...sql:9-38`) | **Superior in `pegasus.x`**: Fuel types, MOT, and insurance fields modeled |
| | Driver Onboarding | Phone and name | Spanner `Drivers` table (`spanner.ddl:394-413`) and driver scores | PostgreSQL `drivers` with 14-digit PINFL, license array, 25M UZS cash limit (`025_...sql:39-53`) | **Superior in `pegasus.x`**: Strict Uzbekistan biometric & AML compliance |
| | Dynamic Shift Pairing | Not implemented | Updates `Drivers.VehicleId` with active order checks (`fleet_guards.go:68-120`) | Dedicated `driver_vehicle_assignments` with bijective partial unique indexes (`025_...sql:54-74`) | **Superior in `pegasus.x`**: Formalized bijective shift history |
| | Pre-Trip DVIR Inspections | Not implemented | **None** (Administrative maintenance toggles only) | Full `vehicle_inspections` schema (`025_...sql:75-101`) + Native mobile dialogs | **Major Domain Gap**: Present only in `pegasus.x` |
| | Dispatch Safety Gating | Not implemented | Basic vehicle availability check | Closed-loop CVRP gate requiring passed same-day inspection in Tashkent time (`dispatch/service.go:248-261`) | **Superior in `pegasus.x`**: Uninspected vehicles blocked from dispatch |
| | Mid-Shift Hot-Swapping | Reassign recommendation | Peer-to-peer SOS rescue broadcast (`driver/rescue.go:1-210`) | Transactional vehicle (`SwapVehicle`) and driver (`SwapDriver`) hot swaps (`fleet/service.go:381-515`) | **Parity Met**: Both support active in-transit vehicle failures |
| | CVRP Routing Engine | Simple optimizer client | Google OR-Tools sidecar with Guided Local Search (`apps/dispatch-optimizer-py/main.py`) | Standalone Python FastAPI microservice with 2-Opt CVRP & Croston SBA (`planning/`) | **Parity Met**: Heuristic vehicle routing optimization |
| | Telemetry & Tracking | Basic GPS ping to Redis | Kafka `logistics.telemetry.v1` + `RouteTwins` digital twin (`twin/service.go`) | Redis `GEOADD drivers:active` + cold chain probe logs (`telemetry/service.go`) | **Parity Met**: Live driver tracking and breadcrumb logs |
| **Factory Production** | Bill of Materials (BOM) | Not implemented | BOM explosion and material requirements (`factory/bom.go:1-180`) | Schema modeled via `factory_boms` (`migration 017_...sql`), backend API pending | **Superior in `pegasusX`**: Shipped Go service and UI portal |
| | Batch Scheduling | Basic batcher | Production batch scheduling (`factory/batcher.go:1-200`, `planning_service.go`) | Schema modeled via `factory_production_batches` (`migration 052_...sql`) | **Superior in `pegasusX`**: Full scheduling engine in Go |
| | Quality Control (QC) | Not implemented | QC defect logging and pass/fail gates (`factory/qc.go:1-180`) | Schema modeled via `factory_qc_records` (`migration 052_...sql`) | **Superior in `pegasusX`**: Active QC service wired to outbox |
| **Finance & Accounting** | General Ledger & Precision | Basic treasury service | Double-entry payment ledger (`payment/double_entry.go:1-120`) | Strict 64-bit integer tiyins + verified $\sum\text{Debits}=\sum\text{Credits}$ (`payment/handover.go:228-241`) | **Superior in `pegasus.x`**: Strict balance invariant assertion |
| | Invoicing & Contracts | Basic contract record | Credit notes and AR treasury invoices (`ar/`, `creditnote/`) | Enterprise invoices, bilateral contracts (`migration 007_...sql`) | **Parity Met**: Full commercial invoicing |
| | Escrow Holds & Payouts | Payout overrides | Escrow holds on card auth, release upon proof of delivery; payout batches (`payout/`) | Cash bag turn-in, doorstep cash settlement, payout batches (`payout/`, migration 051) | **Parity Met**: Idempotent provider settlement in both |
| **Tax & Fiscalization** | Soliq 12% VAT | Not implemented | Soliq E-Factura client (`soliq/client.go:1-120`) | 1200 bps VAT integer math, half-up rounding (`fiscal/calculator.go:8-25`, migration 067) | **Parity Met & Localized in `pegasus.x`** |
| | Statutory B2B Cash Limit | Not implemented | Cash recon compliance checks | Enforced 25M UZS ceiling (`MaxB2BCashLimitMinor = 2500000000`) in `fiscal/calculator.go:17` | **Parity Met & Localized in `pegasus.x`** |
| | Fiscal Receipt / Cheque | Draft DDL only | `OrderLineFiscalSnapshots` interleaved in Spanner (`fiscal/service.go:1-250`) | `fiscal_receipts` table + Didox integration (`004_...sql`) | **Parity Met**: Statutory electronic receipts |
| **Client Surfaces** | Desktop Applications | Prototype web portals | 5 Next.js 15 Web/Desktop Portals (Tauri v2) with `@pegasusx/desktop-cache` | 2 Tauri v2 Next.js 15 Desktops (`supplier-desktop`, `warehouse-desktop`) with OS Keyring | **Parity Met**: High-density desktop control towers |
| | Merchant / Retailer UI | Early desktop/mobile stubs | Next.js 15 desktop app + Native Android & iOS apps | Telegram Bot (Grammy, voice AI, Skonto) + Telegram Mini App (React 19) | **Superior Market Fit in `pegasus.x`**: Telegram ubiquity for Uzbek retail |
| | Driver Mobile Cockpit | Prototype Android/iOS | Native Kotlin Android & Swift iOS apps with voice nav | Native Kotlin Android & Swift iOS apps with Room offline queue & Pre-Trip DVIR dialogs | **Superior in `pegasus.x`**: DVIR safety gate in driver cockpit |

---

## 4. Primary Domain Gap Deep Dive: Fleet Management & Driver Shift Lifecycle

### 4.1 Daily Driver-Vehicle Shift Pairing

```
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                 SHIFT PAIRING COMPARISON                                         │
├─────────────────────────────────────┬────────────────────────────────────────────────────────────┤
│ pegasusX Implementation             │ pegasus.x Implementation                                   │
├─────────────────────────────────────┼────────────────────────────────────────────────────────────┤
│ • In-place mutation of Drivers.VehicleId│ • Dedicated table: driver_vehicle_assignments            │
│ • Guarded by active order count query   │ • Bijective constraint via partial unique indexes        │
│   (warehouse/fleet_guards.go:68-120)│   (idx_active_driver, idx_active_vehicle)                  │
│ • No relational shift history table │ • Full shift history by date and warehouse               │
│ • Lacks national biometric PINFL    │ • Enforces 14-digit national biometric PINFL             │
└─────────────────────────────────────┴────────────────────────────────────────────────────────────┘
```

#### Detailed Code Analysis
- **In `pegasusX`**:
  - The `Drivers` table in `schema/spanner.ddl:394-413` stores `VehicleId STRING(64)`.
  - When an operator assigns a vehicle to a driver, the service executes `driverAssignmentGuard` (`pegasusX/apps/backend-go/warehouse/fleet_guards.go:68-120`):
    ```go
    func driverAssignmentGuard(state driverAssignmentState, activeOrders int64) error {
        if activeOrders > 0 {
            return &FleetMutationError{
                StatusCode: http.StatusConflict,
                Code:       "driver_active_orders",
                Message:    fmt.Sprintf("driver %s has %d active orders and cannot change vehicle assignment", state.DriverID, activeOrders),
            }
        }
        return nil
    }
    ```
  - Spanner executes queries `countActiveOrdersForDriver` and `countActiveOrdersForVehicle` (lines 79-120) checking for orders in `LOADED`, `IN_TRANSIT`, `ARRIVED`, `DISPATCHED`, `AWAITING_PAYMENT`, or `PENDING_CASH_COLLECTION`. If none exist, `Drivers.VehicleId` is updated.
  - **Limitation**: `pegasusX` does not persist a distinct shift record. Once unassigned or reassigned, previous pairing records are overwritten.

- **In `pegasus.x`**:
  - Shift pairings are modeled in PostgreSQL table `driver_vehicle_assignments` (`database/migrations/025_fleet_and_driver_lifecycle_management.sql:54-74`):
    ```sql
    CREATE TABLE IF NOT EXISTS driver_vehicle_assignments (
        assignment_id       VARCHAR(64) PRIMARY KEY,
        supplier_id         VARCHAR(64) NOT NULL REFERENCES suppliers(supplier_id),
        warehouse_id        VARCHAR(64) NOT NULL REFERENCES warehouses(warehouse_id),
        shift_date          DATE NOT NULL DEFAULT CURRENT_DATE,
        driver_id           VARCHAR(64) NOT NULL REFERENCES drivers(driver_id),
        vehicle_id          VARCHAR(64) NOT NULL REFERENCES vehicles(vehicle_id),
        paired_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        released_at         TIMESTAMPTZ,
        assignment_type     VARCHAR(32) NOT NULL DEFAULT 'REGULAR_SHIFT',
        swap_reason         VARCHAR(64),
        previous_vehicle_id VARCHAR(64) REFERENCES vehicles(vehicle_id),
        notes               TEXT,
        created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );
    CREATE UNIQUE INDEX idx_active_driver_assignment ON driver_vehicle_assignments (driver_id) WHERE released_at IS NULL;
    CREATE UNIQUE INDEX idx_active_vehicle_assignment ON driver_vehicle_assignments (vehicle_id) WHERE released_at IS NULL;
    ```
  - At the Go layer, `AssignDriverToVehicle` (`backend/internal/fleet/service.go:311-354` and `fleet/repository.go`) verifies that neither the driver nor vehicle has an open assignment (`released_at IS NULL`), ensures the vehicle is not in `MAINTENANCE`, creates the assignment record, updates `drivers.on_shift = true`, and emits `fleet.driver.assigned` to Redis 7 Streams (`s.redis.XAddFleetEvent`).

---

### 4.2 Pre-Trip DVIR Inspections (Vehicle Inspection Checklist & Safety Interlock)

```
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                   DVIR SAFETY GATING FLOW                                        │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
   Driver Starts Shift
          │
          ▼
   ┌──────────────────────────────────────────────┐
   │ Native Mobile Cockpit (Android / iOS)        │
   │ PreTripDVIRDialog.kt / PreTripDVIRModalView  │
   │ • Brakes, Tires, Lights, Steering            │
   │ • Methane CNG Cylinder Seal Integrity        │
   │ • Reefer Temp Probe (-25°C to +15°C)         │
   │ • Digital Driver Signature Hash              │
   └──────────────────────────────────────────────┘
          │
          ▼ POST /v1/fleet/inspections
   ┌──────────────────────────────────────────────┐
   │ Go Fleet Service (internal/fleet/service.go) │
   │ Persists to vehicle_inspections              │
   └──────────────────────────────────────────────┘
          │
          ├───────────────────────────────┐
   [is_safe_to_operate = true]    [is_safe_to_operate = false]
          │                               │
          ▼                               ▼
   ┌─────────────────────────────┐ ┌───────────────────────────────────────┐
   │ Closed-Loop CVRP Dispatch   │ │ Ground Vehicle Immediately            │
   │ (dispatch/service.go:248)   │ │ • vehicles.operational_status =       │
   │ Vehicle Approved for Routes │ │   'MAINTENANCE'                       │
   │ on Active Shift Date        │ │ • Emit 'alerts.fleet.safety_failure'  │
   └─────────────────────────────┘ └───────────────────────────────────────┘
```

#### Detailed Code Analysis
- **In `pegasusX`**:
  - There is **no table** named `vehicle_inspections` or `VehicleInspections` in `pegasusX/apps/backend-go/schema/spanner.ddl`.
  - Vehicle roadworthiness is managed solely by setting `UnavailableReason` to `'MAINTENANCE'` or `'TRUCK_DAMAGED'` (`warehouse/fleet_availability.go:11-25`). Drivers do not submit structured pre-trip mechanical check reports.
- **In `pegasus.x`**:
  - Modeled in table `vehicle_inspections` (`025_...sql:75-101`):
    `inspection_id`, `assignment_id`, `vehicle_id`, `driver_id`, `inspection_type`, `odometer_km`, `fuel_level_pct`, `tires_pressure_status`, `brakes_status`, `lights_signals_status`, `cleanliness_sanitation`, `refrigeration_temp_celsius`, `cng_cylinder_seal_valid`, `fire_extinguisher_valid`, `walkaround_passed`, `is_safe_to_operate`, `defect_notes`, `driver_signature_hash`.
  - In `backend/internal/fleet/service.go:555-590`, `RecordInspection` generates a tamper-evident signature hash (`sha256.Sum256`) from `driver_id:vehicle_id:odometer:type:safe`. If `is_safe_to_operate == false`, the vehicle is immediately grounded to `MAINTENANCE` and an alert is emitted to Redis 7 Streams (`alerts.fleet.safety_failure`).
  - **Closed-Loop CVRP Dispatch Integration** (`backend/internal/dispatch/service.go:248-261`):
    ```sql
    JOIN LATERAL (
        SELECT is_safe_to_operate, created_at 
        FROM vehicle_inspections 
        WHERE (assignment_id = dva.assignment_id OR vehicle_id = v.vehicle_id)
          AND inspection_type = 'PRE_TRIP'
        ORDER BY created_at DESC 
        LIMIT 1
    ) vi ON true
    WHERE d.supplier_id = $1 
      AND d.on_shift = true
      AND v.operational_status IN ('YARD_STANDBY', 'LOADING_AT_DOCK', 'ACTIVE_ON_ROAD')
      AND vi.is_safe_to_operate = true
      AND vi.created_at >= (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Tashkent')::date
    ```
    This ensures that only vehicles that have passed a pre-trip inspection on the local calendar date can be assigned orders during automated CVRP dispatch planning.

---

### 4.3 Mid-Shift Hot-Swapping (Breakdown Recovery & Handover Protocols)

#### Detailed Code Analysis
- **In `pegasusX`**:
  - Implemented in `apps/backend-go/driver/rescue.go:1-210`:
    1. Driver suffering mechanical breakdown calls `POST /v1/driver/ops/rescue/request`.
    2. Backend creates a rescue record in Spanner and publishes `RESCUE_BROADCAST` to Kafka / WebSocket `DriverHub`.
    3. An eligible peer driver accepts via `POST /v1/driver/ops/rescue/respond`.
    4. Spanner transaction updates all in-transit `Orders` where `DriverId = brokenDriverID` to `DriverId = rescueDriverID`, linking the new vehicle.
- **In `pegasus.x`**:
  - Implemented in `backend/internal/fleet/service.go:381-450` (`SwapVehicle`) and `453-515` (`SwapDriver`), backed by migration `037_delivery_exceptions_and_driver_rescues.sql`:
    1. **Vehicle Substitution (`SwapVehicle`)**:
       - Closes current assignment: `released_at = NOW()`, `swap_reason = reason` (`MECHANICAL_BREAKDOWN`, `FLAT_TIRE`, `REFRIGERATION_FAILURE`, etc.).
       - Transitions disabled vehicle: `operational_status = 'MAINTENANCE'`, `unavailable_reason = reason`.
       - Atomically creates new assignment with `assignment_type = 'HOT_SWAP_RESCUE'`, `previous_vehicle_id = oldVehID`.
       - Updates open delivery manifests (`manifests.vehicle_id = newVehID`).
       - Emits `fleet.vehicle.swapped` to Redis 7 Streams.
    2. **Driver Substitution (`SwapDriver`)**:
       - When a driver is incapacitated or hits statutory overtime limits, `SwapDriver` transfers the loaded truck and manifest to a relief driver (`assignment_type = 'RELIEF_DRIVER'`).
       - Updates `manifests.driver_id = newDriverID` and `orders.driver_id = newDriverID` within an ACID PostgreSQL transaction (`pgx.Tx`).

---

### 4.4 What `pegasus.x` Needs to Achieve Complete Parity
While `pegasus.x` has surpassed `pegasusX` in DVIR inspections, fuel typing, and biometric PINFL validation, the following concrete actions remain to achieve 100% operational maturity:
1. **Interactive Shift Board Drag-and-Drop in `warehouse-desktop`**:
   - `apps/warehouse-desktop/app/vehicles/page.tsx` (772 lines) and `app/drivers/page.tsx` (727 lines) currently use modals (`handleSaveVehicleManagement` and `handleOpenShiftModal`).
   - A visual two-column drag-and-drop Smena Board (Available Drivers on left, Available Vehicles on right) should be added to streamline rapid morning dispatch in large Tashkent depots (50+ trucks).
2. **Supplier Desktop Fleet Surface Integration**:
   - `apps/supplier-desktop/app/(portal)/fleet/page.tsx` is currently a 59-line overview linking to `/org-fleet`. It should embed the comprehensive 4-tab control tower (Avtopark, Haydovchilar, Smena, DVIR Logs) aligned with `warehouse-desktop`.
3. **Factory Production Service Wiring**:
   - While migrations `017` and `052` provide the PostgreSQL schema for BOM, batch scheduling, and QC, dedicated Go services under `backend/internal/factory/` should be ported from `pegasusX/apps/backend-go/factory/` if factory production management is required on the sovereign appliance.

---

## 5. Caveats

1. **Production Deployment Differences**:
   - `pegasusX` is architected for globally distributed multi-region Kubernetes clusters with Spanner and Kafka. Performance and latency metrics cited assume multi-node cloud environments.
   - `pegasus.x` is designed for a single-node or dual-node sovereign appliance (Servercore Tashkent Tier III, direct TAS-IX domestic peering, $139.70/month budget). High-availability replication between disparate cloud providers is intentionally out of scope.
2. **Legacy `pegasus` Codebase**:
   - `pegasus/` was audited as a historical artifact to understand the origin of domain concepts. No further active development or bug fixing should be performed in `pegasus/`; all modern cloud development belongs in `pegasusX/` and sovereign development in `pegasus.x/`.
3. **No Code Written during Audit**:
   - In accordance with the Explorer archetype instructions, this audit was entirely read-only. No source files were modified, and all cited defects, structures, and lines represent the verified repository state.

---

## 6. Conclusion

1. **Evolutionary Trajectory & Clean Boundary**:
   - The workspace contains three distinct evolutionary tiers: `pegasus` (legacy prototype), `pegasusX` (global multi-tenant cloud enterprise), and `pegasus.x` (sovereign single-tenant national core).
   - The Strict Two-System Architectural Boundary is verified: `pegasus.x` contains 0 Spanner and 0 Kafka dependencies, while `pegasusX` preserves its multi-tenant Spanner and Kafka architecture.
2. **Compiler-Grade Compilation Health**:
   - Both production backends compile cleanly with zero errors:
     - `pegasusX/apps/backend-go`: `go build ./...` exited with code 0.
     - `pegasus.x/backend`: `go build ./...` exited with code 0.
3. **Fleet & Driver Management Parity Assessment**:
   - `pegasus.x` has successfully closed and surpassed the Fleet Management gap:
     - Bijective shift pairing (`driver_vehicle_assignments`).
     - Digital pre-trip DVIR inspections (`vehicle_inspections`, `PreTripDVIRDialog.kt`, `PreTripDVIRModalView.swift`).
     - Closed-loop CVRP dispatch gating enforcing safe same-day inspections in Tashkent local time.
     - Mid-shift hot-swapping for broken vehicles and relief drivers.
   - `pegasusX` retains superior Factory Production services and global cell routing, while `pegasus.x` delivers superior sovereign localization (Telegram commerce, 64-bit tiyin GL balance assertions, 12% Soliq VAT math, 25M UZS B2B cash limit, and DVIR).

---

## 7. Verification Method

To independently verify all observations and conclusions in this report, execute the following commands in terminal:

### 1. Verification of Clean Compilation
```bash
# Verify pegasusX Go backend compiles cleanly
cd /Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go && go build ./...

# Verify pegasus.x Go backend compiles cleanly
cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend && go build ./...
```

### 2. Verification of Architectural Boundary (Zero Spanner / Kafka in pegasus.x)
```bash
# Ensure zero Spanner imports in pegasus.x
grep -rn "cloud.google.com/go/spanner" /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/

# Ensure zero Kafka imports in pegasus.x
grep -rn "github.com/segmentio/kafka-go" /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/
```
*Expected Result:* Zero matches found.

### 3. Verification of Fleet Schema in `pegasus.x`
```bash
# Verify vehicles, drivers, assignments, and vehicle_inspections tables
grep -E "CREATE TABLE IF NOT EXISTS (vehicles|drivers|driver_vehicle_assignments|vehicle_inspections)" \
  /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/migrations/025_fleet_and_driver_lifecycle_management.sql
```

### 4. Verification of Pre-Trip DVIR Dispatch Gate in `pegasus.x`
```bash
# Inspect closed-loop dispatch query enforcing Tashkent date and safe inspection
sed -n '248,261p' /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/dispatch/service.go
```

### 5. Verification of Mobile DVIR Components
```bash
# Inspect Android Pre-Trip DVIR Dialog
head -n 40 /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/apps/driver-app-android/app/src/main/java/com/pegasusx/driver/ui/PreTripDVIRDialog.kt

# Inspect iOS Pre-Trip DVIR View
head -n 40 /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/apps/driver-app-ios/Sources/DriverApp/Views/Components/PreTripDVIRModalView.swift
```

### 6. Invalidation Conditions
This report's conclusions shall be deemed invalid if:
- Any Spanner or Kafka client library is added to `pegasus.x/backend/go.mod`.
- Single-tenant PostgreSQL code or migrations are imported into `pegasusX`.
- The CVRP dispatch query in `pegasus.x` is altered to allow uninspected or unsafe vehicles (`vi.is_safe_to_operate = false` or `NULL`) onto outbound delivery routes.
- The double-entry general ledger balance check (`sumDebits == sumCredits`) in `pegasus.x/backend/internal/payment/handover.go` is bypassed or removed.
