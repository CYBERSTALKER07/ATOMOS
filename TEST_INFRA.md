# Testing Infrastructure: Supplier Onboarding & Fleet Hub in pegasus.x

## 1. Overview & Architectural Boundary

This document defines the testing infrastructure, test execution framework, and test tier hierarchy for the **Supplier Onboarding and Warehouse/Fleet Management** subsystem in `pegasus.x`.

### Architectural Invariants:
- **System**: `pegasus.x` Sovereign Lean Single-Tenant Operating Core.
- **Persistence**: Pure PostgreSQL 16 via `pgx/v5` connection pool (`*db.Pool`). Zero mock or in-memory fallback repositories.
- **Cache & Real-time**: Redis 7 (Streams `stream:supplier:events`, Pub/Sub, Geo proximity `warehouses:locations`).
- **Architectural Boundary**: Zero Cloud Spanner, Zero Apache Kafka. All messaging uses PostgreSQL transactional outbox and Redis 7 Streams.
- **Financial Minor Units**: Strict 64-bit integer tiyins (`BIGINT`). Zero floating-point arithmetic for currency amounts.

---

## 2. Test Harness Architecture

The E2E test harness operates directly against the Go backend HTTP layer using Chi v5 routing and the Go standard library's `net/http/httptest`:

```
+---------------------------------------------------------------------------------+
|                                Test Harness                                     |
|  (supplier_onboarding_e2e_test.go in package api_test)                          |
+---------------------------------------------------------------------------------+
                                      |
                                      | HTTP JSON Requests (JWT Authorization)
                                      v
+---------------------------------------------------------------------------------+
|                           httptest.Server (in-process)                          |
|  Mounted with full Chi v5 router from api.NewServer(cfg, pool, rdb, ...)        |
+---------------------------------------------------------------------------------+
                                      |
         +----------------------------+----------------------------+
         |                                                         |
         v                                                         v
+------------------------------------+   +------------------------------------+
| Onboarding Gate Middleware         |   | Operational Endpoints              |
| RequireSupplierOnboardingCompleted |   | /v1/supplier/warehouses            |
| (HTTP 428 Precondition Required)   |   | /v1/orders                         |
+------------------------------------+   +------------------------------------+
         | Whitelist:                                              |
         | /v1/auth/*                                              |
         | /v1/supplier/onboarding/*                               |
         v                                                         v
+------------------------------------+   +------------------------------------+
| Auth & Wizard Handlers             |   | Fleet & Warehouse Handlers         |
| /v1/auth/supplier/register         |   | POST /v1/supplier/warehouses       |
| /v1/auth/supplier/login            |   | POST /.../warehouses/{id}/trucks   |
| POST /v1/supplier/onboarding/...   |   | POST /.../warehouses/{id}/payloaders
+------------------------------------+   +------------------------------------+
```

### Key Components:
- **Server Instance**: Created using `setupTestServer(t)` or `api.NewServer` initializing all domain dependencies.
- **Client Protocol**: Standard `http.Client` with a 5-second timeout and helper functions `sendJSON`, `parseJSON`, `parseJSONArray`.
- **Authentication**: JWT token generation using HMAC-SHA256 (`auth.GenerateToken` with `models.RoleSupplier`).

---

## 3. Test Tiers & Methodology

The test suite is derived strictly from specifications and interface contracts, divided into four rigorous tiers:

### Tier 1: Feature Coverage (>=5 Test Cases per Feature across 9 Features = 45 Tests)
Covers happy-path operational flows, standard responses, and field structures:
1. **F1: Supplier Registration**: Valid registration, standard 201 response, `supplier_id` generation, `onboarding_status: PENDING`, `next_step: /onboarding/products`, password security (no leaks).
2. **F2: Supplier Login**: Valid credentials authentication, JWT token issuance, initial PENDING status, next step routing, supplier ID retrieval.
3. **F3: Onboarding Gate Middleware**: HTTP 428 Precondition Required interception on operational endpoints (`/v1/supplier/warehouses`, `/v1/orders`), response body contract verification, whitelist verification for `/v1/auth/*` and `/v1/supplier/onboarding/*`.
4. **F4: Step 1 Products Catalog**: Add product with EAN-13, 17-digit MXIK, package code, units_per_case, int64 tiyins, 12% VAT; list products; edit product; delete product; gate rule requiring >=1 active product.
5. **F5: Step 2 Payment Gateway Configuration**: Cash payment rail enabled by default; Global Pay (`GLOBAL_PAY`) corporate card gateway setup with service ID, secret key, corporate BIN list; GET configuration; corporate BIN acceptance; dual rail persistence.
6. **F6: Step 3 Complete Onboarding**: Transition status to `COMPLETED`; success response payload; operational gate unblocking; subsequent login status reflection; event emission.
7. **F7: Warehouse Management**: Add warehouse with `latitude` and `longitude` (`DOUBLE PRECISION`); list warehouses; get warehouse by ID; update coordinates; delete empty warehouse with zero stock.
8. **F8: Fleet Management (Trucks)**: Add truck with license plate, capacity kg/m³, fuel type; list warehouse trucks; fuel type support (DIESEL, CNG, PETROL, ELECTRIC); capacity bounds; warehouse scoping.
9. **F9: Dock Payloaders**: Add payloader with name and Uzbekistan phone (`+998...`); list payloaders; phone validation; multiple payloaders support; warehouse scoping.

### Tier 2: Boundary & Corner Cases (>=5 Test Cases per Category across 8 Categories = 40 Tests)
Covers adversarial boundary checks, data constraints, and statutory Uzbekistan compliance:
1. **Category 1: Duplicate STIR Registration**: Exact duplicate STIR -> 409 Conflict; duplicate STIR with different company name -> 409; duplicate STIR with different phone -> 409; conflict payload mentions `tax_id`; whitespace-padded STIR conflict.
2. **Category 2: Invalid STIR Formats**: Alphanumeric characters -> 400 Bad Request; 8 digits (too short) -> 400; 10 digits (too long) -> 400; empty STIR -> 400; special characters/dashes -> 400.
3. **Category 3: Non-Integer / Float / Negative Prices**: Float price rejected -> 400/422; negative price rejected -> 400/422; zero price rejected for commercial catalog items -> 400/422; string price rejected -> 400/422; valid int64 tiyins accepted.
4. **Category 4: Invalid MXIK Code**: 16 digits (too short) -> 400/422; 18 digits (too long) -> 400/422; letters in MXIK -> 400/422; empty MXIK -> 400/422; valid 17 numeric digits accepted.
5. **Category 5: Invalid EAN-13 Barcode**: 12 digits -> 400/422; 14 digits -> 400/422; checksum mismatch (modulo 10 failure) -> 400/422; duplicate barcode in catalog -> 409 Conflict; valid EAN-13 accepted.
6. **Category 6: Warehouse Deletion Guard**: Active stock on hand > 0 -> 409 Conflict (`cannot_delete_warehouse`); active non-terminal orders -> 409 Conflict; error body verification; non-existent warehouse -> 404; zero stock and zero orders permitted to delete -> 200 OK.
7. **Category 7: Corporate Card BIN Validation**: Retail consumer Uzcard BIN (`8600`) rejected -> 400/422; retail Visa BIN (`4000`) rejected -> 400/422; empty BIN list when enabled -> 400/422; invalid BIN length (<4 digits) -> 400/422; valid corporate BINs (`5614`, `9860`, `5440`, `4073`, `5168`) accepted.
8. **Category 8: Warehouse GPS Coordinates Validation**: Missing latitude -> 400/422; missing longitude -> 400/422; latitude > 90 -> 400/422; longitude > 180 -> 400/422; valid Uzbekistan coordinates accepted.

### Tier 3: Cross-Feature Combinations (Pairwise State Machine Scenarios = 2 Scenarios / 13 Verification Steps)
1. **Scenario 3.1: Full Wizard Sequence with Gate Interception**:
   Register -> Login -> Intercept operational route (428) -> Step 1 Products -> Step 2 Payment -> Step 3 Complete -> Retry operational route (200 OK unblocked).
2. **Scenario 3.2: Warehouse, Fleet, and Deletion Lifecycle**:
   Provision warehouse -> Assign truck -> Assign payloader -> Verify inventory deletion guard -> Clear inventory -> Delete warehouse.

### Tier 4: Real-World Application Scenario (Enterprise Simulation = 1 Scenario / 10 Verification Stages)
1. **Scenario 4.1: OOO Samarkand Logistics Enterprise Onboarding**:
   End-to-end multi-step onboarding simulation for real-world enterprise:
   - Legal STIR: `309876543`, Phone: `+998935001122`, Company: `OOO Samarkand Logistics`
   - Products: "Samarkand Mineral Water 1.5L" (12,000 UZS) & "Registan Premium Tea 250g" (35,000 UZS) with MXIK codes
   - Payment Rails: Cash on Delivery + Global Pay Corporate Acquirer
   - Regional Hub: "Samarkand Central Distribution Hub" (Lat: 39.6542, Lon: 66.9597)
   - Fleet: 2 Isuzu Trucks (DIESEL & CNG)
   - Loading Dock Staff: 2 Payloaders ("Alisher Navoiy" & "Bobur Mirzo")

---

## 4. Test Runner Commands

| Target | Command | Purpose |
|--------|---------|---------|
| **Compilation Check** | `go test -c ./internal/api/` | Verify syntax, types, and imports compile cleanly |
| **Targeted Onboarding Suite** | `go test -v ./internal/api/ -run "TestSupplierOnboarding"` | Run all 88 onboarding tests across Tiers 1-4 |
| **All Supplier Tests** | `go test -v ./internal/api/ -run "TestSupplier"` | Run onboarding suite plus existing supplier suites |
| **Race Detector** | `go test -v -race ./internal/api/ -run "TestSupplierOnboarding"` | Concurrency and race condition verification |
| **All Package Tests** | `go test -v ./internal/api/...` | Run entire API package test suite |

---

## 5. Test File Location
- **Path**: `pegasus.x/backend/internal/api/supplier_onboarding_e2e_test.go`
- **Package**: `api_test`
- **Total Test Cases/Scenarios**: 88
