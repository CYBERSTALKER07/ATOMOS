# GS-L / GS-K: Local Multi-Supplier Ecosystem (Backend, Data Plane & Infra)

**Final Destination (2026-08-20):** This file + [`GLOBAL_SCALE_PROGRAM.md`](./GLOBAL_SCALE_PROGRAM.md) + [`FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md`](./FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md) define the canonical enterprise destination. Loaded on every session: [`.agents/memory/GOAL.md`](../../.agents/memory/GOAL.md).

**Date:** 2026-08-20 (Synchronized Master State)  
**Living Tree:** `pegasusX/`  
**Master Blueprint:** [`FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md`](./FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md)  
**Objective:** A multi-supplier, multi-country, local-first logistics ecosystem that is **geospatially local-first** (H3 Res 7), **pack-smart** (currency, PSP catalog, fiscal adapters), and **reproducible across regions** without cross-border complexity.

This specification **extends** [`GLOBAL_SCALE_PROGRAM.md`](./GLOBAL_SCALE_PROGRAM.md) and [`FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md`](./FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md). All 26 warehouse gap items (**W1–W26**) are resolved in code and verified with live-path tests.

---

## 0. Current Architectural Status (Verified in Code)

```
STATUS: SHIPPED & VERIFIED
- Local Matching (GS-L0–L4): CoverageEngine, StampNodeGeography, ServicePins,
  SupplierRegions, and same-market ParentOrders hard-gate are active in code.
- Pack PSP Catalog (GS-K1–K3): payment/catalog.go, catalogHonestyExecutor (HTTP 501
  no_live_keys), and planned country packs (CA, AU, GB, PK, EU, US, KZ) are active.
- Client Parity (GS-R & GS-U): All 6 role rows (Supplier, Retailer, Driver,
  Warehouse, Factory, Payload) + Platform Admin bind pack currency, local-first
  catchments, and honest status stacks with zero fake mock data.
- Gated Boundaries: checkout_reads_this remains false in local/SSMR until production
  live Soliq/cloud credentials; 410 boundaries explicitly guarded.
```

**Non-Negotiable Tenancy Law:** The isolation key is **`SupplierId`**. Market pack, home cell, country, city, and region are attributes. No second tenant key exists.

---

## 1. Product Laws & Invariants

| # | Product Law | Code Enforcement Mechanism |
|---|---|---|
| **L1** | **One Tenant Key** | Every durable supplier row is scoped by `SupplierId STRING(36) NOT NULL`. Handlers enforce `auth.RequireTenant()`. |
| **L2** | **Market Owns Money** | Currency, decimal precision, PSP list, fiscal adapter, and payout rail derive strictly from shipped `MarketPack`. Order currency picker remains disabled. |
| **L3** | **Same-Market Orders Only** | Orders require retailer country == warehouse country == factory country == supplier `MarketCode` country. Mismatch returns `422 cross_market_deferred`. |
| **L4** | **Local-First Default** | Retailer store resolves to closest covering warehouse of that supplier. Warehouse replenishment resolves to closest factory on an active `SupplyLane`. |
| **L5** | **Supplier Override Wins** | Supplier-configured `ServicePins` (location/retailer/region) and `WarehouseCoverageCells` override distance-based closest matching. |
| **L6** | **Empty Geography Fails Closed** | Missing `CountryCode` on warehouse/factory/store returns `422 geography_incomplete`. `proximity.StampNodeGeography` stamps country + H3 Res 7 on all write paths. |
| **L7** | **Pack Filters PSP UI** | GET `/v1/*/payment-catalog` returns `LivePackGateways(pack)`. UZ clients never see Stripe/Adyen as selectable. |
| **L8** | **Unkeyed Rails Fail Honestly** | Missing keys return HTTP 501 `no_live_keys` via `catalogHonestyExecutor`. Zero fake 200 redirects. |
| **L9** | **One Country = One Pack + Adapters**| New market onboardings introduce a versioned `MarketPack` + 1–3 adapters (fiscal, PSP, SMS). Zero codebase forks. |
| **L10**| **Class A Integrity** | Integer minor units, fiscal hard-gate, pay-at-delivery, dual manifests, H3 Res 7 indexing, and transactional outbox emissions. |

---

## 2. Warehouse Gap Register Resolution (W1–W26 Audit)

Every item in the warehouse and proximity gap ledger has been resolved and verified in code:

| Item | Architectural Area | Implementation & Resolution in Code | Status |
|---|---|---|---|
| **W1** | Warehouse Setup Geography | `POST /v1/warehouse/setup` calls `proximity.StampNodeGeography`, persisting `CountryCode` and H3 Res 7. | **RESOLVED (L1)** |
| **W2** | Warehouse Location Patch | `PATCH /v1/warehouse/ops/location` updates lat/lng while preserving pack country and recalculating H3 Res 7. | **RESOLVED (L1)** |
| **W3** | Topology Resolution Alignment | Supplier topology writes H3 at **Resolution 7** matching checkout resolver (`repository_spanner.go`). | **RESOLVED (L1)** |
| **W4** | Warehouse CRUD Derived H3 | Warehouse repository automatically derives H3 Res 7 from lat/lng if omitted. | **RESOLVED (L1)** |
| **W5** | Unified Geography Helper | Single helper `proximity.StampNodeGeography` governs all warehouse/factory writers. | **RESOLVED (L1)** |
| **W6** | Fail-Closed Country Match | `WarehouseCoversRetailer` returns false if warehouse or retailer country is empty (`order/coverage.go`). | **RESOLVED (L0)** |
| **W7** | Cross-Border Cell Guard | Country equality check enforced even when coverage cells are populated (`coverage.go:102-104`). | **RESOLVED (L0)** |
| **W8** | Unified Stock & Order Picker | Catalog stock path and checkout order path both invoke `proximity.ResolveServingWarehouse`. | **RESOLVED (L2)** |
| **W9** | Retailer Lat/Lng Column Fix | Catalog queries read canonical `Retailers.Lat` and `Lng` columns (`catalog/stock.go`). | **RESOLVED (L2)** |
| **W10** | Checkout Fallback Columns | Unified checkout fallback reads canonical `Lat` and `Lng` columns (`unified_checkout.go`). | **RESOLVED (L2)** |
| **W11** | Active Store Resolution | Checkout resolves pin from JWT `active_location_id` → `RetailerLocations` before body fallback. | **RESOLVED (L2)** |
| **W12** | Catchment Model Alignment | Catchment is governed by country, coverage cells, and pins; `CoverageRadiusKm` treated as decorative hint. | **RESOLVED (L2)** |
| **W13** | H3 Index Persistence | Warehouse `H3Cell` persisted at Res 7 for `Idx_Warehouses_ByH3Cell`. | **RESOLVED (L2)** |
| **W14** | Supplier Regions Architecture | Dead global `Regions` table replaced with tenant-scoped `SupplierRegions` (`spanner.ddl:3600-3619`). | **RESOLVED (L3)** |
| **W15** | Store-to-Warehouse Service Pins | `ServicePins` table (`spanner.ddl:3620-3640`) allows supplier ADMIN to explicitly pin stores/regions to warehouses. | **RESOLVED (L3)** |
| **W16** | Factory Supply Resolution | `ResolveSupplyFactory` evaluates `PrimaryFactoryId`, `SupplyLanes` priority, and closest fallback. | **RESOLVED (L2)** |
| **W17** | SupplyLanes Priority Reader | `SupplyLanes` table read to prioritize multi-factory replenishment. | **RESOLVED (L2)** |
| **W18** | Supply Accept Seed Cleanup | Missing warehouse id returns `warehouse_id_required` error; fake `"wh-1"` fallback removed. | **RESOLVED (L1)** |
| **W19** | Warehouse Payment Filtering | Warehouse payment config GET/POST filtered through `payment.AvailablePSPs(pack)`. | **RESOLVED (K1)** |
| **W20** | Delivery Fee Currency Default | Delivery fee rules derive currency from `auth.PackCurrency` rather than hardcoding `"UZS"`. | **RESOLVED (L1/K1)** |
| **W21** | Delivery Perimeter Alignment | Delivery perimeter derived directly from `WarehouseCoverageCells`. | **RESOLVED (L2)** |
| **W22** | Preview Multi-Supplier Split | `checkout_preview` resolves serving warehouse per supplier group identically to create. | **RESOLVED (L2)** |
| **W23** | Supplier Billing Allowlist | Supplier billing uses pack-filtered gateway catalog instead of hardcoded Adyen/Stripe. | **RESOLVED (K1)** |
| **W24** | Payment-Required Event Filter | `available_card_gateways` on payment-required events filtered through pack catalog. | **RESOLVED (K1)** |
| **W25** | Executor Empty Gateway Default | Executor resolves empty gateway via `LivePackGateways(pack)`. | **RESOLVED (K2)** |
| **W26** | Factory Geography Stamping | Factory setup and location patch call `proximity.StampNodeGeography`. | **RESOLVED (L1)** |

---

## 3. Geospatial Matching Architecture (`proximity/coverage_engine.go`)

### 3.1 Warehouse Resolution Pipeline
```
ResolveServingWarehouse(supplierID, retailerStore):
  1. Reject if store.CountryCode is empty                 → ErrGeographyIncomplete (422)
  2. Filter active, on-shift warehouses for supplierID
  3. Reject candidate if warehouse.CountryCode != store.CountryCode → Filter out
  4. Check ServicePins:
     - Exact RetailerLocations.LocationId match           → Return pinned warehouse
     - Exact Retailers.RetailerId match                   → Return pinned warehouse
     - SupplierRegions.RegionId match                     → Return closest within pinned region
  5. Check WarehouseCoverageCells:
     - If candidate has coverage cells: require H3 Res 7 disk membership
  6. Rank remaining candidate warehouses by Haversine distance
  7. Return closest candidate
  8. If candidates empty                                  → ErrZoneMiss (422)
```

### 3.2 Factory Supply Resolution Pipeline
```
ResolveSupplyFactory(warehouse):
  1. If PrimaryFactoryId is set, active, and same-country  → Return PrimaryFactoryId
  2. Query active SupplyLanes for warehouse, ordered by Priority DESC
     - If lane factory is active and same-country         → Return Lane.FactoryId
  3. Query all active same-country factories for supplier
     - Rank by Haversine distance to warehouse            → Return closest factory
  4. If no candidate found                                → ErrFactoryUnassigned (422)
```

---

## 4. Payment Gateway Catalog & Honest Executors (`payment/`)

### 4.1 Pack PSP Catalog Structure (`payment/catalog.go`)
```go
type PSPAdapter struct {
    Code          string   // GLOBAL_PAY, STRIPE, ADYEN, PAYME, CLICK, CASH, CREDIT
    Markets       []string // Allowed ISO country/pack codes
    Status        string   // live | unkeyed | planned
    NationalCards bool
}
```

### 4.2 Gateway Execution Router (`payment/execution.go`)
- **Live Executors**: `GLOBAL_PAY` (executes via `global_pay_executor.go`), `CASH` (manual cash turn-in), `CREDIT` (AR invoice ledger).
- **Honest Placeholder Executors (`catalogHonestyExecutor`)**: `STRIPE`, `ADYEN`, `PAYME`, `CLICK` return RFC 7807 HTTP 501 `no_live_keys` with descriptive diagnostic details when initialized without credentials.

---

## 5. Verification & Test Execution

```bash
# 1. Proximity & Coverage Engine Unit Tests
cd pegasusX/apps/backend-go
go test ./proximity/ ./order/ ./warehouse/ ./catalog/ -v

# 2. Payment Catalog & Honesty Router Tests
go test ./payment/ ./auth/ -v

# 3. Supplier Regions & Service Pins Tests
go test ./supplier/ -run "TestServicePins|TestSupplierRegions" -v

# 4. Multi-Role SSMR Smoke Suite
go test -v ./cmd/ssmr-smokecheck/
```
