# Project: PegasusX Foundry Triad, Role 7 Cashier, and Role 9 Fleet UI Integration

## Architecture
PegasusX is an enterprise FMCG distribution platform supporting the Single-Supplier Multi-Retailer (SSMR) doctrine across 9 operational roles.
The Foundry architecture integrates:
- **Semantic Layer**: First-class Ontology Graph objects, links, multi-hop BFS `SearchAround`, propagating security markings, and contaminated lot recall blast-radius analysis.
- **Kinetic Layer**: Deterministic pre-execution action engine validating pre-conditions before committing state mutations.
- **Dynamic Layer**: Copy-on-Write Scenario Sandbox overlays with optimistic `BaseVersion` watermark concurrency verification.
- **Role 7 (Depot Cashier & Vault)**: Tamper-evident bag scanning, dual-blind UZS denomination counting in integer Tiyin (strictly hiding `ExpectedCashMinor` during count and recount), recount ratchet alert (`BLIND_RECOUNT_REQUIRED`), RRN cash-to-card reclassification, CIT armored vault handover manifests, and uncapped multi-tender settlement tracking.
- **Role 9 (Fleet, Yard & Maintenance)**: Mandatory 8-point Pre-Trip DVIR walkaround, automated Red-Tag Grounding Gate (`OUT_OF_SERVICE`), ALPR Yard Gate check-in/out with outbound bolt seal verification, predictive reefer health compressor cooling rate diagnostics, and mid-route breakdown rescue with thermal excursion quarantine.

### Code Layout
- `apps/backend-go/foundry/`: Core Foundry backend engine (ontology, kinetic, scenarios, cashier_vault, fleet_yard, edge_guards).
- `apps/backend-go/foundryroutes/`: 29 registered `/v1/foundry/*` HTTP route handlers.
- `packages/types/`: Shared TypeScript domain models and interfaces.
- `packages/api-core/`: Shared TypeScript typed API clients.
- `packages/api-react/`: React hooks for API client communication.
- `apps/admin-portal/`: Next.js 15 App Router platform administration console.
- `apps/warehouse-portal/`: Next.js 15 / Tauri desktop portal for warehouse ops, cashier, and fleet.
- `apps/supplier-portal/`: Next.js 15 / Tauri portal for wholesale suppliers.
- `apps/factory-portal/`: Next.js 15 / Tauri portal for manufacturing plants.
- `apps/driver-app-android/` & `apps/driver-app-ios/`: Native mobile apps for drivers (DVIR, rescue, gate).
- `apps/warehouse-app-android/` & `apps/warehouse-app-ios/`: Native mobile apps for warehouse staff (cashier blind count, yard gate).
- `apps/supplier-app-android/` & `apps/supplier-app-ios/`: Native mobile apps for suppliers (scenario sandbox, lot recall).

## Feature Inventory
| # | Feature | Description | Milestone | Source |
|---|---------|-------------|-----------|--------|
| 1 | Foundry Domain Types & API Client (TS) | Shared TypeScript models and client functions for all 29 `/v1/foundry/*` routes in `packages/types` and `packages/api-core` | M1 | Backend Contracts Survey |
| 2 | Foundry Domain Models & API Client (Kotlin) | Kotlin data classes and Retrofit interfaces in shared packages / Android apps | M1 | Backend & Mobile Surveys |
| 3 | Foundry Domain Models & API Client (Swift) | Swift structs and URLSession service classes in shared packages / iOS apps | M1 | Backend & Mobile Surveys |
| 4 | Warehouse Portal Role 7 Cashier UI | Blind count denomination inputs (UZS in Tiyin), strictly masked `ExpectedCashMinor`, dual-blind recount ratchet, RRN reclassification, CIT handover, AML check | M2 | ORIGINAL_REQUEST §R2 |
| 5 | Warehouse Portal Role 9 Fleet UI | 8-point DVIR walkaround, Red-Tag Grounding Gate status, predictive reefer health card, yard gate ALPR & bolt-seal verification, mid-route rescue | M2 | ORIGINAL_REQUEST §R3 |
| 6 | Warehouse Portal Foundry Sandbox UI | Scenario Sandbox workbench, Copy-on-Write patch staging, BaseVersion commit, Ontology blast-radius viewer | M2 | ORIGINAL_REQUEST §R1 |
| 7 | Admin Portal Foundry & Roles Command Panel | Control Tower tabs for Foundry Sandbox & Action Guard, Cashier Vault & AML Monitor, Fleet DVIR & Yard Gate Monitor | M2 | ORIGINAL_REQUEST §R1, §R2, §R3 |
| 8 | Supplier Portal Scenario Sandbox & Lot Recall | Scenario patch editor and contaminated lot recall blast-radius graph view | M2 | ORIGINAL_REQUEST §R1 |
| 9 | Factory Portal Loading Bay & Recall Banner | Yard gate check-in/out bolt seal verification and raw material lot recall quarantine alerts | M2 | ORIGINAL_REQUEST §R1, §R3 |
| 10 | Driver Android Role 9 DVIR & Rescue UI | Jetpack Compose 8-point DVIR walkaround, Red-Tag Grounding Gate, mid-route rescue handoff, gate event logging | M3 | ORIGINAL_REQUEST §R3 |
| 11 | Warehouse Android Role 7 Cashier & Gate UI | Jetpack Compose blind denomination count (masked expected cash), UZS Tiyin math, dual-blind recount alert, RRN reclass, CIT handover, yard gate | M3 | ORIGINAL_REQUEST §R2, §R3 |
| 12 | Supplier Android Foundry Sandbox UI | Jetpack Compose scenario sandbox and lot recall inspection | M3 | ORIGINAL_REQUEST §R1 |
| 13 | Driver iOS Role 9 DVIR & Rescue UI | SwiftUI 8-point DVIR walkaround, Red-Tag Grounding Gate, mid-route rescue handoff, gate event logging | M3 | ORIGINAL_REQUEST §R3 |
| 14 | Warehouse iOS Role 7 Cashier & Gate UI | SwiftUI blind denomination count (masked expected cash), UZS Tiyin math, dual-blind recount alert, RRN reclass, CIT handover, yard gate | M3 | ORIGINAL_REQUEST §R2, §R3 |
| 15 | Supplier iOS Foundry Sandbox UI | SwiftUI scenario sandbox and lot recall inspection | M3 | ORIGINAL_REQUEST §R1 |
| 16 | Contract Parity & Verification Gates | 100% path parity (`make parity-contract-full`), CI hygiene gates, unit tests passing across backend, web, Android, iOS | M4 | ORIGINAL_REQUEST §R4 |
| 17 | Dual-Root Mirroring | 100% identical file synchronization to `/Users/shakhzod/Desktop/V.O.I.D/pegasusX` with 0 diff | M4 | ORIGINAL_REQUEST §R4 |

## Milestones
| # | Name | Scope | Dependencies | Status |
|---|------|-------|-------------|--------|
| M1 | Shared Contracts & Central API Clients | Define typed domain models and API clients across TypeScript (`packages/types`, `packages/api-core`), Kotlin, and Swift matching canonical routes | none | PLANNED |
| M2 | Web Portals UI Wiring | Implement UI views, screens, and components for Foundry Sandbox, Role 7 Cashier, and Role 9 Fleet in `warehouse-portal`, `admin-portal`, `supplier-portal`, `factory-portal` | M1 | PLANNED |
| M3 | Mobile Apps UI Wiring | Implement UI screens and ViewModels for Role 7 Cashier, Role 9 Fleet DVIR, and Foundry Sandbox in Android and iOS apps | M1 | PLANNED |
| M4 | Verification, Gating & Dual-Root Mirroring | Run full suite of CI gates (`parity-contract-full`, `anti_theatre`, `repo_hygiene`, `fail_todo_inject`, tests), forensic audit, and mirror to V.O.I.D | M2, M3 | PLANNED |

## Interface Contracts

### Canonical Backend Route Crosswalk
All clients must use the exact registered routes:
- `POST /v1/foundry/ontology/objects`
- `GET /v1/foundry/ontology/objects/{cellID}/{objectType}/{objectID}`
- `POST /v1/foundry/ontology/links`
- `POST /v1/foundry/ontology/search-around`
- `POST /v1/foundry/ontology/lot-recall` (NOT shorthand `/recall`)
- `POST /v1/foundry/kinetic/actions/execute` (NOT shorthand `/execute`)
- `POST /v1/foundry/scenarios/{scenarioID}/patches`
- `GET /v1/foundry/scenarios/{scenarioID}/resolve/{cellID}/{objectType}/{objectID}`
- `POST /v1/foundry/scenarios/{scenarioID}/commit`
- `POST /v1/foundry/cashier/sessions`
- `POST /v1/foundry/cashier/sessions/{sessionID}/blind-count` (NOT shorthand `/count`)
- `POST /v1/foundry/cashier/reclassify-tender` (NOT shorthand `/reclassify-rrn`)
- `POST /v1/foundry/cashier/vault/cit-handover` (NOT shorthand `/cit-handover`)
- `POST /v1/foundry/fleet/dvir`
- `POST /v1/foundry/fleet/gate-events`
- `POST /v1/foundry/fleet/mid-route-rescue` (NOT `/guards/mid-route-rescue`)
- `POST /v1/foundry/fleet/predictive-reefer` (NOT `/fleet/reefer-health`)
- `GET /v1/foundry/fleet/vehicles/{vehicleID}/dispatch-check`
- `POST /v1/foundry/guards/aml-check` (NOT `/guards/aml-exposure`)

### Currency Contract
- All monetary amounts are represented as 64-bit signed integers in minor units (`Tiyin`: `1 UZS = 100 Tiyin`).
- Zero floating-point arithmetic for currency.
- Valid Uzbek Som banknote denominations: `200000`, `100000`, `50000`, `20000`, `10000`, `5000`, `2000`, `1000` UZS.
- Physical count formula: `CountedMinor = sum(noteUZS * count * 100)`.

### Role 7 Blind Count Ritual Contract
- Initial session creation returns `CashierSessionView` with `expected_cash_minor: null`.
- Attempt 1 discrepancy returns `409 Conflict` with `status: "BLIND_RECOUNT_REQUIRED"` and `expected_cash_minor: null`. Cashier must perform recount #2 before supervisor clearance is allowed.

### Role 9 DVIR Contract
- 8 checklist items: `steering_and_horn_ok`, `service_and_parking_brakes_ok`, `tires_tread_and_pressure_ok`, `lights_and_hazards_ok`, `wipers_and_fluids_ok`, `emergency_kit_and_extinguisher_ok`, `cargo_box_sanitation_ok`, `reefer_temp_calibrated_ok`.
- Any false value triggers `409 Conflict` with `red_tag_applied: true`, setting status to `RED_TAG_QUARANTINE` (`OUT_OF_SERVICE`) and uncoupling the driver-vehicle assignment.
