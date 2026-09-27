# Pegasus.X Ecosystem Comprehensive Architectural Survey

> **Survey Date**: 2026-09-26  
> **Target Root**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x`  
> **Audience**: Core Architects, Infrastructure Engineers, AI Agents (`agents.md`)  
> **Integrity Mode**: 100% Code-Grounded Reality ("Zero Theatre")

---

## Executive Summary

**Pegasus.X** is a hyper-modular, enterprise-scale B2B FMCG (Fast-Moving Consumer Goods) supply chain and logistics operating system engineered specifically for the national commerce and distribution infrastructure of Uzbekistan. It orchestrates the full trade lifecycle across five core commercial roles: **Suppliers**, **Warehouses**, **Retailers**, **Commercial Fleet Drivers**, and **Warehouse Payloaders**.

The ecosystem is implemented as a polyglot monorepo governed by **pnpm 9.15.0** and **Turborepo 2.4.4**, integrating:
1. **Core Go Backend**: High-throughput distributed API server (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend`) built on Go 1.22+ and Chi v5, structured into 83 domain packages, 125 API modules/handlers, and 52 end-to-end living loop test verifications.
2. **Mathematical S&OP Planning Service**: Python FastAPI service (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/planning`) implementing Croston SBA intermittent forecasting, MEIO (Multi-Echelon Inventory Optimization) dynamic safety stock, and a multi-wave Capacitated Vehicle Routing Problem (CVRP) solver with 2-Opt local search and a 95% volumetric Tetris buffer.
3. **17 User-Facing Applications**:
   - 3 Next.js 15.1 + React 19 desktop portals powered by Tauri 2 desktop bridges (`supplier-desktop`, `warehouse-desktop`, `retailer-desktop`).
   - 5 Native Android applications in Kotlin with Jetpack Compose (Material 3) and Room offline databases (`apps/*-android`).
   - 5 Native iOS applications in Swift 6.0 with SwiftUI (`apps/*-ios`).
   - 2 Telegram applications: a production bot using grammY 1.34 (`apps/telegram-bot`) and a Telegram WebApp/MiniApp in Vite 6 + React 19 (`apps/telegram-miniapp`).
   - 2 Specialized cross-platform Expo React Native apps (`apps/field-sales-mobile`, `apps/payloader-tablet`).
4. **24 Shared Packages**: Reusable TypeScript contracts, UI design systems, charts, maps, mobile native kits, desktop caching, and WebSocket dirty-slice refresh contracts.
5. **Database & Storage**: PostgreSQL 16 with TimescaleDB (`timescale/timescaledb-ha:pg16`), 78 schema migrations, 31 seed datasets, and Redis 7 with AOF persistence.
6. **Enterprise Cloud & Production Infrastructure**: Production deployment runbook targeting Tashkent Tier III DC (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/scripts/deploy_prod.sh`), Docker Compose topologies (base, dev, prod with Caddy 2, Prometheus, and Grafana), full Google Cloud Platform Terraform modules (`infra/terraform`), and Kubernetes fleet manifests (`infra/k8s`).

---

## 1. Monorepo Architecture & Directory Topology

### 1.1 Root Topology & Workspace Manifests

| Path | Purpose & Technology |
|---|---|
| `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/package.json` | Root workspace manifest named `@pegasusx/monorepo` with pnpm package manager definition (`pnpm@9.15.0`), React 19 overrides, and Playwright 1.63 E2E test commands. |
| `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/pnpm-workspace.yaml` | Declares active workspace globs: `apps/*`, `packages/*`, excluding `packages/api-client`. |
| `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/turbo.json` | Turborepo pipeline configuration (`ui: "stream"`) orchestrating `build`, `lint`, `typecheck`, `check-types`, and `test` tasks with topological cache dependencies (`^build`). |
| `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/Makefile` | 132-line unified command orchestration interface covering Turborepo targets, backend Go builds, Python planning tests, iOS Swift tests, Android Kotlin tests, dual-engine E2E tests, Terraform validation, and Kubernetes Kustomize validation. |
| `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/.env.example` | Canonical environment configuration template specifying ports, database connection pools, Redis parameters, JWT secrets, geofence radii, and service endpoints. |
| `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/playwright.config.ts` | Playwright test configuration running Chromium, Firefox, and WebKit against the desktop portal instances. |

---

## 2. Backend Go Core Service Architecture

### 2.1 Entry Point & Runtime Bootstrap
- **Path**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/cmd/server/main.go`
- **Go Version & Module**: `github.com/pegasus-x/core` on Go 1.22+ (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/go.mod`).
- **Core Dependencies**: `chi/v5` router, `pgx/v5` connection pool, `go-redis/v9`, `golang-jwt/jwt/v5`, `gorilla/websocket`, `uber/h3-go/v3` (hexagonal spatial indexing), and `playwright-community/playwright-go`.
- **Boot Lifecycle**:
  1. **Secrets Ingestion**: Loads environment variables and conditionally ingests production secrets from HashiCorp Vault via AppRole authentication (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/secrets/vault.go`).
  2. **Fail-Closed Configuration Validation**: Enforces minimum secret lengths, database URL presence, and operational thresholds (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/config/config.go`).
  3. **Structured Observability**: Configures `slog` structured JSON/text logging (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/observability/logger.go`).
  4. **PostgreSQL Connection Pool**: Initializes `pgxpool` with a maximum of 25 connections and automatic schema migrations (`pool.Migrate()`).
  5. **Redis Connection**: Establishes Redis 7 client for pub/sub, live telemetry caching, and rate limiting.
  6. **Background Workers**:
     - **Transactional Outbox Relay**: `outbox.NewRelayWorker` polls the PostgreSQL `outbox_events` table every 500ms in batches of 50, delivering events to Redis pub/sub channels (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/outbox/relay.go`).
     - **WebSocket Hub**: `ws.NewHub` maintains real-time bidirectional client socket connections (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/ws/hub.go`).
  7. **HTTP Server**: Runs Chi router on configured port (default 8080) with graceful shutdown handling `SIGINT`/`SIGTERM`.

### 2.2 Global Middleware & Security Pipeline
- **Router Construction**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/api/router.go`
- **Global Pipeline**:
  - `middleware.RequestID`: Injects unique X-Request-ID.
  - `middleware.RealIP`: Resolves client IP across reverse proxies.
  - `observability.StructuredLoggingMiddleware`: Logs HTTP status, duration, method, and URI with slog.
  - `middleware.Recoverer`: Panic recovery preventing server termination.
  - `metricsReg.HTTPMiddleware`: Exposes Prometheus metrics (`http_requests_total`, `http_request_duration_seconds`).
  - `tracer.HTTPTracingMiddleware`: OpenTelemetry distributed trace context propagation.
  - `IdempotencyMiddleware`: Enforces `Idempotency-Key` headers on mutating requests (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/api/middleware_idempotency.go`).
  - `cors.Handler`: Cross-origin resource sharing configuration.

### 2.3 Strict Precondition Onboarding Gates (HTTP 428)
The server wraps operational endpoints in `mountProtected` with four mandatory verification gates:
1. **Supplier Onboarding Gate**: `requireSupplierOnboardingCompleted` checks PostgreSQL for `onboarding_status == 'COMPLETED'`. Incomplete suppliers receive `428 Precondition Required` with `next_step: "/onboarding/products"` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/api/router.go`).
2. **Warehouse Onboarding Gate**: `requireWarehouseOnboardingCompleted` verifies dock bays, bin slotting, and initial stock setup before granting operational access.
3. **Driver Shift Gate**: `requireDriverShiftReady` intercepts dispatch access if driver compliance, vehicle pairing, or DVIR inspections are outstanding.
4. **Payloader Onboarding Gate**: `requirePayloaderOnboardingCompleted` ensures tablet commissioning, bay binding, and axle calibration are completed.

### 2.4 The Five Bounded Domain Route Modules
The router delegates endpoints to five domain modules registered in `backend/internal/api/modules`:

```
backend/internal/api/
├── core.go          (System health, JWKS, public auth, onboarding wizards, DLQ replay)
├── logistics.go     (Fleet, driver lifecycle, dispatch solver, manifests, ePOD, telemetry, geocoding)
├── warehouse.go     (WMS, bin slotting, FEFO lots, pick waves, cross-docking, dock bays, empties)
├── commercial.go    (Catalog, orders, checkout, supplier portal, retailer POS, auto-order)
└── finance.go       (Soliq e-factura, credit notes, cash ledger, trade credit quotas, AR dunning, payroll)
```

#### A. Core Module (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/api/core.go`)
- **Probes**: `/health`, `/healthz`, `/ready` (verifies PostgreSQL and Redis connectivity), `/metrics` (Prometheus exposition).
- **Public Auth & JWKS**: `/.well-known/jwks.json` (RS256 public key set), `/v1/auth/token`, role-based logins (`/v1/auth/driver/login`, `/v1/auth/payloader/login`, `/v1/auth/retailer/login`, `/v1/auth/supplier/login`, `/v1/auth/warehouse/login`), token refreshes, and TOTP MFA enrollment/verification (`/v1/auth/mfa/enroll`, `/v1/auth/mfa/verify`).
- **Real-Time WebSockets & SSE**: `/v1/ws` (Gorilla WebSocket connection), `/v1/supplier/events` (Server-Sent Events stream), `/v1/supplier/sync` (catch-up mutation feed).
- **Universal Multi-Role Onboarding**: `/v1/onboarding/suppliers`, `/v1/onboarding/retailers`, `/v1/onboarding/warehouses`, `/v1/onboarding/drivers`, `/v1/onboarding/payloaders`, `/v1/onboarding/regions` (SOATO data), `/v1/onboarding/market-pack`.
- **Operational Resiliency**: `/v1/admin/ops/dead-letters` and `/v1/admin/ops/dead-letters/replay` for outbox dead-letter queue (DLQ) investigation and re-dispatch.

#### B. Logistics Module (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/api/logistics.go`)
- **Payloader Dock Tablet**: `/v1/payloader/pulse`, `/v1/payloader/manifests/{id}/axle-feasibility`, `/v1/payloader/manifests/{id}/seal`, `/v1/payloader/manifests/{id}/load-ledger/scan`, `/v1/payloader/variance`.
- **Regional Transport & Passes**: `/v1/regional/soato` (Uzbekistan administrative units), `/v1/regional/route-feasibility`, `/v1/regional/linehaul/shuttle`, `/v1/sms-gateway/pod`.
- **Driver Shift & DVIR**: `/v1/driver/onboarding/compliance`, `/v1/driver/onboarding/vehicle-pairing`, `/v1/driver/onboarding/dvir`, `/v1/driver/onboarding/complete`.
- **Fleet Asset Tracking**: `/v1/vehicles`, `/v1/drivers`, `/v1/drivers/{id}/active-route`, `/v1/fleet/assignments`, `/v1/fleet/assignments/swap` (hot-swap vehicle/driver).
- **Driver Telemetry & Geofence**: `/v1/telemetry/ping` (2500ms GPS stream), `/v1/telemetry/verify-arrival` (enforces `< 150m` delivery proximity hard gate).
- **Doorstep Handshake Execution**: `/v1/delivery/arrive`, `/v1/delivery/scan-qr`, `/v1/delivery/proximity-unlock`, `/v1/delivery/bypass-offload`, `/v1/delivery/negotiate`, `/v1/delivery/split-payment`, `/v1/delivery/shop-closed`, `/v1/delivery/partial-offload`.
- **Smart Dispatch & Breakdown Rescue**: `/v1/dispatch/preview`, `/v1/dispatch/commit`, `/v1/dispatch/lock`, `/v1/dispatch/freeze`, `/v1/warehouse/dispatch/rescues` (incident reporting, candidate ranking, transfer dispatch).
- **ePOD & Manifest Stops**: `/v1/warehouse/manifests/{id}/stops/{orderID}/arrive`, `/v1/warehouse/manifests/{id}/stops/{orderID}/confirm-epod`, `/v1/epod/offline-sync`.

#### C. Warehouse Module (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/api/warehouse.go`)
- **Warehouse Setup & Settings**: `/v1/warehouse/onboarding/bays`, `/v1/warehouse/onboarding/bins`, `/v1/warehouse/onboarding/initial-stock`, `/v1/warehouse/approval-settings`, `/v1/warehouse/supervisor-override-pin` (4-digit supervisor authorization code).
- **Facility-Locked Balances**: `/v1/inventory/{warehouseID}/balances` (enforced via facility isolation middleware `EnforceWarehouseFacility`).
- **Warehouse Management (WMS)**: `/v1/wms/locations`, `/v1/wms/putaway`, `/v1/wms/lots/{id}` (FEFO lot tracking), `/v1/wms/waves/generate`, `/v1/wms/tasks/{id}/pick`, `/v1/wms/labels/zpl` (Zebra thermal label generation).
- **Dock Bays & Yard**: `/v1/dock/bays`, `/v1/dock/bays/{bayNumber}/assign`, `/v1/dock/bays/{bayNumber}/progress`, `/v1/dock/bays/{bayNumber}/seal`, `/v1/dock/bays/{bayNumber}/depart`.
- **Inter-Warehouse & Bin Replenishment**: `/v1/transfers`, `/v1/replenishments`.
- **Physical Cycle Counts**: `/v1/cycle-counts`, `/v1/inventory-adjustments`.
- **Demand Forecasting & Replenishment**: `/v1/warehouse/demand-forecast/recalculate`, `/v1/warehouse/demand-forecast/generate-po`, `/v1/warehouse/demand-forecast/trigger-replenishments`.
- **Cross-Docking Engine**: `/v1/warehouse/crossdock/evaluate`, `/v1/warehouse/crossdock/orders`, `/v1/warehouse/crossdock/stage`, `/v1/warehouse/crossdock/load`.
- **Reverse Logistics**: `/v1/warehouse/returns/inbound`, `/v1/warehouse/returns/inspect`, `/v1/warehouse/returns/quick-restock`.
- **3D Bins & FEFO Lots**: `/v1/warehouse/bins`, `/v1/warehouse/lots`.
- **Batch Wave Picking & Tomorrow Board**: `/v1/warehouse/pick-waves`, `/v1/warehouse/stock-commitments`, `/v1/warehouse/preorders`, `/v1/warehouse/tomorrow-board`.
- **Live Operations Radar & Heatmap**: `/v1/warehouse/ops/board`, `/v1/warehouse/ops/broadcast/send`, `/v1/warehouse/ops/heatmap` (picker telemetry), `/v1/warehouse/ops/express/fast-track` (Plan-90).
- **SAP Empties (RTI) & QM**: `/v1/empties/movements`, `/v1/empties/demurrage/evaluate`, `/v1/empties/intakes`, `/v1/qm/quarantine` (quarantine lot ingestion & disposition).
- **Enterprise S/4HANA Parity**: `/v1/enterprise/ewm/slotting/rebalance`, `/v1/enterprise/consignment/agreements`.

#### D. Commercial Module (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/api/commercial.go`)
- **Product Catalog & Packaging Hierarchy**: `/v1/catalog/products`, `/v1/catalog/barcode/{ean}`, `/v1/catalog/categories/schemas`, `/v1/catalog/products/{id}/packaging-units` (multi-tier packaging conversion: Base Unit -> Inner Pack -> Case -> Pallet).
- **Voice-Note Ordering**: `/v1/speech/voice-order` (Whisper transcription + dialect parsing).
- **Order Lifecycle**: `/v1/orders`, `/v1/orders/{id}/confirm`, `/v1/orders/{id}/transition`, `/v1/orders/{id}/claim-eligibility`.
- **Universal Mutation Protocol (UMP)**: `/v1/ump/disputes`, `/v1/ump/concealed-damage` (auto-apply threshold <= 600k UZS / 60,000,000 tiyins).
- **Supplier Portal & CRM**: `/v1/supplier/profile`, `/v1/supplier/kyc`, `/v1/supplier/topology`, `/v1/supplier/org/members`, `/v1/supplier/pricing/rules`, `/v1/supplier/pricing/retailer-overrides`, `/v1/supplier/orders/vet` (order vetting & audit logs), `/v1/supplier/service-policy`, `/v1/supplier/ai/recommendations`, `/v1/supplier/crm/retailers`.
- **Promotions & Loyalty**: `/v1/supplier/promotions` (volume deals), `/v1/supplier/loyalty` (points accrual & tier progression).
- **Retailer Store Operations (Retail OS)**:
  - POS Registers & Shifts: `/v1/retailer/registers`, `/v1/retailer/shifts/open`, `/v1/retailer/shifts/close`, `/v1/retailer/shifts/cash-drop`, `/v1/retailer/pos/sales`, `/v1/retailer/pos/holds` (parked cart holds).
  - Store Stock & Sections: `/v1/retailer/stock`, `/v1/retailer/stock/receive-sessions`, `/v1/retailer/stock/counts`, `/v1/retailer/sections` (shelf intelligence & alerts), `/v1/retailer/assist/tickets` (floor assist task queue).
  - Autonomous Replenishment: `/v1/retailer/auto-order/settings`, `/v1/retailer/auto-order/evaluate`, `/v1/retailer/auto-order/proposals` (enforces draft-only invariant).
  - B2B Inbound & Doorstep Review: `/v1/retailer/orders/{id}/tracking`, `/v1/retailer/orders/{id}/handshake-token`, `/v1/retailer/orders/{id}/doorstep-review`.
- **Enterprise Commercial AI**:
  - `/v1/enterprise/ai/orders/parse-message` (conversational order parsing).
  - `/v1/enterprise/ai/vision/shelf-to-cart` (shelf deficit calculation from photos).
  - `/v1/enterprise/ai/depletion/predict` (predictive stockout alerting).
  - `/v1/enterprise/planning/allocation/fair-share` (constrained quota allocation).
  - `/v1/enterprise/multisupplier/checkout` (multi-supplier cart split into isolated child orders).

#### E. Finance Module (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/api/finance.go`)
- **MySoliq Fiscal Integration**: `/v1/soliq/invoices/{orderID}`, `/v1/soliq/invoices/{orderID}/sign` (E-Imzo RSA PKI signature), `/v1/soliq/facturas/generate`, `/v1/soliq/facturas/corrective` (Tax Code Art. 257 Tuzatuvchi Factura).
- **Cash Reconciliation & Cage Settlement**: `/v1/cash/payment-legs`, `/v1/cash/drivers/{driverID}/shift-summary`, `/v1/cash/reconcile` (discrepancy detection & driver shift locking), `/v1/cash/deposits`, `/v1/cash/expected`.
- **Corrective Invoices (Credit Notes)**: `/v1/creditnotes` (integer arithmetic with basis points tax offset).
- **Trade Credit Quota System**: `/v1/credit/config`, `/v1/credit/pins` (delegated 4-digit PIN authorization from warehouse to retailer), `/v1/credit/tranches`, `/v1/credit/restrictions` (product category credit locks), `/v1/credit/checkout`.
- **Accounts Receivable (AR) & Dunning Ladder**: `/v1/ar/summary`, `/v1/ar/invoices`, `/v1/ar/dunning/evaluate` (5-tier dunning ladder with automatic credit freeze and debt auto-charge).
- **Central Bank FX Indexation**: `/v1/fx/rates`, `/v1/fx/cbu-sync` (Central Bank of Uzbekistan rate sync), `/v1/fx/revaluations` (dual-currency quotation & dynamic order revaluation).
- **Regional Seasonality Profiles**: `/v1/seasonality/templates`, `/v1/seasonality/safety-stock/calculate` (Navruz, Ramadan, harvest season surges).
- **Supplier Payout Batches**: `/v1/payout/batches`, `/v1/payout/batches/{id}/disburse`, `/v1/payout/batches/{id}/export` (national bank payment file generation).
- **GlobalPay & Contactless SoftPOS**: `/v1/webhooks/global-pay`, `/v1/softpos/charge` (EMV contactless card kernel for Uzcard & Humo).
- **Autonomous Headcount Reduction**: `/v1/enterprise/adm/process-drop` (Automated Deposit Machine smart safe drops), `/v1/enterprise/opex/submit`, `/v1/enterprise/commission/accrue`.
- **Autonomous GlobalPay Payroll**: `/v1/enterprise/payroll/staff`, `/v1/enterprise/payroll/batches/execute`, `/v1/enterprise/payroll/batches/{id}/soliq-registry` (Soliq payroll registry export).

---

## 3. Data Models & Business Logic Modules

### 3.1 Strict Currency & Tax Arithmetic Invariant
- **Rule**: Floating-point currency calculations are strictly forbidden across the backend.
- **Implementation**: All prices, balances, discounts, and payments are stored and calculated as 64-bit integers in minor units (**tiyins**, where `1 UZS = 100 tiyins`).
- **Tax Math**: Uzbekistan 12% Value Added Tax is computed using basis points (`1200 bps`), half-up integer rounding, and divisor offsets (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/fiscal/tax.go`):
  ```go
  const (
      BasisPointDivisor = 10000
      HalfUpOffset      = 5000
      DefaultVatRateBps = 1200 // 12.00%
  )
  vatDelta := (subtotalDelta*vatRateBps + HalfUpOffset) / BasisPointDivisor
  ```

### 3.2 Key Domain Packages Breakdown
The 83 packages in `backend/internal/` encode precise business domain models:

| Package | Source Path | Core Domain Responsibility |
|---|---|---|
| `adm` | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/adm` | Automated Deposit Machine (smart safe) cash deposit hardware integration and optical gate reconciliation. |
| `aiorder` | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/aiorder` | Conversational NLP order intake, shelf deficit vision analysis, and predictive depletion ordering. |
| `allocation` | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/allocation` | Fair-share allocation algorithms and constrained supply quota rationing. |
| `ar` | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/ar` | Accounts Receivable aging ledger (0-30, 31-60, 61-90, 90+ days) and 5-tier dunning ladder. |
| `bins` | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/bins` | 3D warehouse coordinate slotting (`Zone-Aisle-Rack-Shelf-Bin`) and S-curve pick path traversal. |
| `cashrecon` | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/cashrecon` | Driver cash-on-delivery cashier cage settlement, CIT drawer tracking, and discrepancy shift locking. |
| `claims` | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/claims` | Cargo transit damage claims, photographic evidence inspection, and claim adjudication. |
| `compliance` | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/compliance` | Asl Belgisi (CRPT Turon) national track-and-trace 3-tier aggregation: Unit CIS -> Carton ATK -> Pallet SSCC. |
| `credit` | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/credit` | Trade credit quota system, 4-digit PIN delegation from warehouse to retailer, credit tranches, and category locks. |
| `dispatch` | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/dispatch` | Volumetric Kahan compensated summation, CVRP fleet vehicle packing, 2-Opt TSP routing, and fleet breakdown rescue. |
| `doorstep` | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/doorstep` | Cryptographic QR code generation, fraud-proof doorstep handshake verification, and proximity unlocking. |
| `empties` | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/empties` | Returnable Transport Items (RTI / SAP Empties) tracking, tara dock intake tallies, and monthly demurrage settlement. |
| `epod` | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/epod` | Electronic Proof of Delivery certificate generation, digital signatures, and offline store-and-forward sync. |
| `ewm` | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/ewm` | Extended Warehouse Management: dynamic slotting rebalancing, ABC velocity zoning, and cross-dock transshipment. |
| `fiscal` | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/fiscal` | Statutory B2B cash limit enforcement (25M UZS ceiling per transaction) and tax invoice calculations. |
| `fleet` | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/fleet` | Commercial vehicles and drivers roster, daily shift pairing, digital DVIR inspections, and emergency hotswaps. |
| `fxrates` | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/fxrates` | Central Bank of Uzbekistan (CBU) automated exchange rate synchronization, dual-currency pricing, and dynamic revaluation. |
| `gs1core` | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/gs1core` | GS1 Modulo-10 check digit verification, GTIN-13/14 parsing, SSCC-18 serials, and Application Identifier (AI) decoding. |
| `onec` | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/onec` | 1C:Enterprise dual-sync engine: CommerceML 2.09 XML parsing/generation and OData REST bidirectional synchronization. |
| `order` | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/order` | State machine governing order status transitions with strict role permissions and mutability boundaries. |
| `outbox` | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/outbox` | Transactional outbox relay worker preventing dual-write inconsistencies between PostgreSQL and Redis. |
| `payload` | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/payload` | Warehouse loading bay terminal operations, item load ledger scanning, axle weight checking, and tamper sealing. |
| `payroll` | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/payroll` | GlobalPay mass payroll disbursal, card binding, and Soliq government tax registry export. |
| `qm` | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/qm` | Quality Management quarantine lot ingestion, defect grading, return-to-vendor (RTV), and scrap dispositioning. |
| `regional` | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/regional` | Uzbekistan SOATO administrative region hierarchy, municipal entry passes for heavy trucks in Tashkent, and linehauls. |
| `softpos` | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/softpos` | EMV contactless card kernel parsing for on-delivery card payments via Uzcard and Humo payment rails. |
| `soliq` | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/soliq` | State Tax Committee (MySoliq) electronic facturas, E-Imzo cryptographic PKI signing, and Art. 257 corrective facturas. |
| `speech` | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/speech` | Whisper speech-to-text integration with phonetic normalization for Uzbek dialects (Tashkent, Fergana, Samarkand, Khorezm). |
| `ump` | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/ump` | Universal Mutation Protocol: auto-applies physical doorstep discrepancies `<= 600,000 UZS` and cascades ledger adjustments. |

### 3.3 The 52 Living Loop Smokecheck Steps
The file `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/cmd/smokecheck/main.go` (7,462 lines) validates the entire supply chain workflow in a deterministic sequence:
1. `Step 1`: o9 Demand Sensing & S&OP Planning Engine (Croston SBA, MEIO Dynamic Safety Stock, ROP).
2. `Step 2`: Provisioning Warehouse Facilities & S-Curve Bin Topology (`Zone-Aisle-Rack-Shelf-Bin`).
3. `Step 3`: Inbound Receiving with FEFO Expiration & 17-digit MXIK Classification.
4. `Step 4`: Order Ingestion & Atomic FEFO Stock Reservation.
5. `Step 5`: Staging Delivery Manifest & Generating Single-Pass Pick Wave.
6. `Step 6`: Executing Warehouse Barcode Pick & Packing Orders.
7. `Step 7`: Payloader Dock Terminal Axle Check & Digital Tamper Sealing (SHA-256 seal).
8. `Step 8`: Driver GPS Telemetry Stream & Doorstep Proximity Gate (`< 150m`).
9. `Step 9`: Physical Discrepancy & Universal Mutation Protocol (UMP) Auto-Apply (`<= 600k UZS`).
10. `Step 10`: Sealing MySoliq E-Factura & Pre-Computing Tuzatuvchi Faktura.
11. `Step 11`: Autonomous Smart Dispatch (CVRP, 2-Opt TSP, 95% Tetris Buffer) & Fleet Rescue.
12. `Step 12`: Retailer Receiving Dock Bolt Seal Verification & GS1 Barcode Ingestion (GTIN-14, Modulo-10).
13. `Step 13`: Driver Cash-on-Delivery Collection, Statutory B2B Limit & Cashier Cage Settlement.
14. `Step 14`: National GS1 & Fiscal Compliance Integration (ZPL Thermal Labels, Asl Belgisi 3-tier, E-Imzo PKI).
15. `Step 15`: On-Delivery Split Payments, Autonomous Debt Engine & GlobalPay Gateway.
16. `Step 16`: Bilateral Commercial Credit, Zero-Collection Handover & UMP Restitution.
17. `Step 17`: Autonomous 3-Way Matching (MM/FI Evaluated Receipt Settlement) & Condition Contracts Volume Rebates.
18. `Step 18`: CO-PA Multi-Dimensional Profitability, Dynamic Checkout Policy & EWM Slotting/Cross-Docking.
19. `Step 19`: Dynamic FSCM Credit Risk Scoring, 5-Tier Dunning Ladder & Consignment Stock (VMI).
20. `Step 20`: Returnable Packaging (SAP Empties) & Digital Quality Management (SAP QM Quarantine).
21. `Step 21`: GlobalPay Autonomous Payroll Engine, Mass Salary Disbursal & Soliq Tax Compliance.
22. `Step 22`: Enterprise Hardening (RS256/JWKS IAM, TOTP MFA, Prometheus/OTel, E-Imzo Signer).
23. `Step 23`: Physical Fleet Integrity, Multi-Zone Cold Chain & Enterprise Blindspots.
24. `Step 24`: Autonomous Zero-Touch AI Ordering, Shelf-to-Cart Vision & Predictive Depletion.
25. `Step 25`: o9 Digital Brain Causal Planning, True Demand Reconstruction & Constrained Allocation.
26. `Step 26`: Resilient Payment Integration, Idempotent Webhooks & Transactional Outbox/Inbox.
27. `Step 27`: Factory Production Twin, Bill of Materials (BOM), Labor Routing & Live Dispute Resolution.
28. `Step 28`: Autonomous Headcount Reduction: ADM Smart Safe & Optical Gate.
29. `Step 29`: Dual-Owner Workforce Concurrency, Shift Clock, RFM CRM & Doorstep Resilience.
30. `Step 30`: Big Tech Spatial Analytics, Demurrage Cascade, Batch Homogeneity & Fiscal FX Indexation.
31. `Step 31`: Multi-Supplier Parent Cart Checkout, Child Order Isolation & GlobalPay Split Settlement.
32. `Step 32`: Policy-Driven Constrained Allocation, Fair-Share Quota Rationing & Autonomous SKU Substitution.
33. `Step 33`: 1C:Enterprise Dual-Sync (CommerceML 2.09 XML & OData REST Reconciliation).
34. `Step 34`: Multilingual Voice-Note Ingestion, Uzbek Dialect Normalization & Telegram WebApp HMAC Auth.
35. `Step 35`: Multi-Warehouse Split Direct Delivery vs. Consolidated Feeder Shuttle Optimization.
36. `Step 36`: Fuel Telematics Theft Detection, Economic Court Dossier Assembly & Asl Belgisi Aggregation.
37. `Step 37`: Retailer Unified Data Plane (Catalog, Inbound Dock Confirm, 48h UMP & Credit Engine).
38. `Step 38`: Uzbekistan All-Roles Adaptability (Supplier, Warehouse, Factory, Driver, Payloader, Retailer).
39. `Step 39`: Nationwide Logistics Architecture (SOATO Hierarchy, Passes, Offline CRDT, SMS POD, Dialects & SoftPOS).
40. `Step 40`: Auto-Order Draft-Only Invariant & Explicit Confirmation Gate.
41. `Step 41`: Retailer Unified Operations (Till POS Shifts, Store Inventory, (R, s, S) Auto-Order & Sell-Through Engine).
42. `Step 42`: Warehouse Realtime Operations Radar, Broadcast Engine & Inbound QC.
43. `Step 43`: Advanced Warehouse Geofencing, Floor Heatmap, Express Ops, Replenishment Insights & Plan Warmer.
44. `Step 44`: Supplier Core, Multi-Echelon Topology, Dynamic Pricing & Order Vetting.
45. `Step 45`: Google Maps & Places Geolocation Platform.
46. `Step 46`: Location Racking (A-01-02), Fleet Asset Provisioning, Daily Shift Pairing, DVIR & Hot-Swap.
47. `Step 47`: Retailer Bilateral Credit, COD AML Ceiling, POS Split Tender & UMP Claims.
48. `Step 48`: Driver Mobile Lifecycle, Route Geometry, SOS Rescue & Shift Return Gates.
49. `Step 49`: 3L-CVRP Gross Axle Weight Rating (GAWR), Moment Equilibrium & Steer Traction Safety Gates.
50. `Step 50`: Returnable Packaging (SAP Empties), Dock Tara Intake Tally, Balance Decrement & Month-End Settlement.
51. `Step 51`: Sovereign Observability, Prometheus Metrics Exposition & Latency Pipeline.
52. `Step 52`: Soliq Electronic Invoicing, Tax Code Art. 257 Tuzatuvchi (Corrective) Factura & e-Imzo Digital Sealing.

---

## 4. Python S&OP Demand Sensing & CVRP Planning Service

### 4.1 Service Overview & Runtime
- **Path**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/planning`
- **Entry Point**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/planning/main.py`
- **Dependencies**: `fastapi>=0.110.0`, `uvicorn>=0.28.0`, `pydantic>=2.6.0`, `numpy>=1.26.0`, `scipy>=1.12.0`, `httpx>=0.27.0` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/planning/requirements.txt`).
- **Container**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/docker/Dockerfile.planning` (Python 3.11-slim, listening on port 8000).

### 4.2 Mathematical Algorithms & Endpoints
1. **Croston's Method with Syntetos-Boylan Approximation (SBA)**:
   - Endpoint: `POST /v1/planning/forecast`
   - File: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/planning/engine/croston.py`
   - Corrects positive bias in traditional Croston forecasting for slow-moving, intermittent FMCG demand:
     $$\hat{y}_{SBA} = \left(1 - \frac{\alpha}{2}\right) \frac{z_t}{p_t}$$
2. **Multi-Echelon Dynamic Safety Stock (MEIO)**:
   - Endpoint: `POST /v1/planning/safety-stock`
   - File: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/planning/engine/meio.py`
   - Uses Acklam inverse normal CDF approximation to calculate dynamic safety stock and Reorder Points (ROP) under demand variance and lead-time jitter.
3. **Demand Sensing & Censored Demand Reconstruction**:
   - Endpoint: `POST /v1/planning/demand-sensing/run`
   - File: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/planning/engine/demand_sensing.py`
   - Identifies stockout-censored historical days, imputes true unconstrained demand, classifies demand into Smooth, Intermittent, Erratic, or Lumpy via Syntetos-Boylan-Croston (SBC) criteria, and generates SQL queries to synchronize PostgreSQL tables (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/planning/engine/database.py`).
4. **S&OP Monte Carlo Scenario Simulation**:
   - Endpoint: `POST /v1/planning/simulate`
   - File: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/planning/engine/simulator.py`
   - Simulates inventory trajectories across 7-180 day horizons under promotional multipliers, supplier delivery delays, and varying target service levels.
5. **A/B Safety Stock Policy Evaluator**:
   - Endpoint: `POST /v1/planning/experiment/evaluate`
   - Compares the legacy heuristic policy (`STATIC_HEURISTIC_25PCT`) against the algorithmic policy (`DYNAMIC_MEIO_ACKLAM`).
6. **Capacitated Vehicle Routing Problem (CVRP) Solver**:
   - Endpoint: `POST /v1/dispatch/optimize-fleet`
   - File: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/planning/engine/cvrp.py`
   - Implements Haversine distance matrix computation, farthest-first greedy initial insertion with a 95% volumetric Tetris buffer (`tetris_buffer = 0.95`), multi-wave virtual vehicle generation, and 2-Opt local search refinement. Produces deterministic SHA-256 dispatch plan fingerprints.

---

## 5. Applications Ecosystem (17 Apps)

### 5.1 Desktop Portals (Next.js 15 + React 19 + Tauri 2)
The ecosystem provides three specialized desktop portals configured with Next.js 15.1, React 19, Tailwind CSS, MapLibre GL, Framer Motion, and Tauri 2 Rust desktop bridge:

| Application | Port | Location | Key Capabilities |
|---|---|---|---|
| **Supplier Desktop** | `3000` | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/apps/supplier-desktop` | Multi-facility topology management, wholesale catalog maintenance, flexible packaging conversion ratios, dynamic pricing rules with retailer overrides, trade promotion creation, loyalty programs, order vetting, and S&OP demand analytics. |
| **Warehouse Desktop** | `3001` | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/apps/warehouse-desktop` | Live supervisor operations radar, 3D bin slotting coordinates, FEFO lots, batch pick wave clustering, dock bay yard scheduling, inter-warehouse transfers, physical cycle counts, returns intake dispositioning, and emergency driver rescue dispatch. |
| **Retailer Desktop** | `3002` | `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/apps/retailer-desktop` | Retail OS: POS cash register shifts, till cash-drops, barcode checkout, parked cart holds, local store SKUs, team roster, store section shelf alerts, wholesale supplier ordering, auto-order proposal confirmation, and credit quota summary. |

### 5.2 Telegram Ecosystem
1. **Telegram Bot**:
   - **Path**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/apps/telegram-bot`
   - **Stack**: Node.js, TypeScript, `grammy v1.34.0`, Express, Axios.
   - **Functionality**: B2B FMCG retailer interaction via Telegram. Delivers order confirmations, out-for-delivery notifications, doorstep delivery handshake OTPs, and launches the Telegram MiniApp.
2. **Telegram MiniApp**:
   - **Path**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/apps/telegram-miniapp`
   - **Stack**: Vite 6, React 19, Tailwind CSS, Lucide React.
   - **Functionality**: Embedded WebApp for retail store owners in Uzbekistan. Enables fast mobile wholesale cart building, reordering from favorite suppliers, order tracking, and delivery confirmation without installing native mobile apps.

### 5.3 Native Android Applications (Kotlin + Jetpack Compose)
All 5 Android applications are built with Kotlin, Jetpack Compose (Material 3), Compose BOM `2024.12.01`, CompileSdk 35, MinSdk 26, Java 17, Retrofit 2, Coroutines, Room SQLite (for offline store-and-forward), and Android WorkManager:
1. `apps/driver-app-android` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/apps/driver-app-android`): GPS telemetry background service (`FusedLocationProvider`), active route turn-by-turn navigation, geofenced doorstep handshake, split-tender cash/card collection, and SOS breakdown assistance.
2. `apps/payload-app-android` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/apps/payload-app-android`): Loading bay barcode scanner, manifest load ledger tally, axle load balance feasibility checker, and bolt seal recording.
3. `apps/retailer-app-android` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/apps/retailer-app-android`): Mobile store ordering, receiving dock seal verification, UMP dispute filing with photos, and trade credit management.
4. `apps/supplier-app-android` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/apps/supplier-app-android`): Supplier executive dashboard, sales velocity, order approval, and real-time shipment monitoring.
5. `apps/warehouse-app-android` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/apps/warehouse-app-android`): Floor picker terminal, S-curve bin traversal, pick task barcode confirmation, and cycle counting.

### 5.4 Native iOS Applications (Swift 6.0 + SwiftUI)
All 5 iOS apps are defined as modular Swift packages targeting iOS 17+ and macOS 14+ (`swift-tools-version: 6.0`), tested via `swift test -q`:
1. `apps/driver-app-ios` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/apps/driver-app-ios`)
2. `apps/payload-app-ios` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/apps/payload-app-ios`)
3. `apps/retailer-app-ios` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/apps/retailer-app-ios`)
4. `apps/supplier-app-ios` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/apps/supplier-app-ios`)
5. `apps/warehouse-app-ios` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/apps/warehouse-app-ios`)

### 5.5 Specialized Mobile Clients (Expo React Native)
1. **Field Sales Mobile**:
   - **Path**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/apps/field-sales-mobile`
   - **Stack**: Expo SDK 51, React Native 0.74.2, React 18.2, Expo Location, Expo Camera.
   - **Screens**: `RouteAgendaScreen.tsx`, `StoreVisitScreen.tsx`, `ShelfAuditScreen.tsx`, `ProxyOrderScreen.tsx`, `CashCollectionScreen.tsx`.
   - **Security**: 50-meter GPS proximity gate ensuring sales agents are physically inside the retail store before placing proxy orders or performing shelf audits.
2. **Payloader Tablet**:
   - **Path**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/apps/payloader-tablet`
   - **Stack**: Expo SDK 51, React Native 0.74.2, Expo Barcode Scanner.
   - **Purpose**: Mounted ruggedized Android/iPad warehouse dock terminal for loading dock workers.

---

## 6. Shared Monorepo Packages (24 Packages)

The `packages/` directory provides modular building blocks across web, desktop, and mobile platforms:

### 6.1 TypeScript & Web Core Packages
- `packages/types` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/packages/types`): Canonical TypeScript domain contracts (`supplier`, `warehouse`, `retailer`, `driver`, `payloader`, `events`, `compliance`, `dispatch`, `claims`, `auto-order`, `regional`).
- `packages/api-core` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/packages/api-core`): HTTP client with automatic JWT token attachment, 401 refresh interceptors, and typed API client methods.
- `packages/api-react` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/packages/api-react`): React Query / custom hook bindings for data fetching and real-time state synchronization.
- `packages/desktop-bridge` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/packages/desktop-bridge`): Tauri 2 IPC wrapper abstracting file system, deep linking, native dialogs, shell processes, and auto-updaters.
- `packages/desktop-cache` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/packages/desktop-cache`): Local SQLite offline cache using `@tauri-apps/plugin-sql` for desktop portal resilience during network dropouts.
- `packages/ws-refresh-contract` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/packages/ws-refresh-contract`): Shared taxonomy of real-time WebSocket and SSE events. Defines `dashboardDirtySlice(eventType)` mapping events to dirty slices (`orders`, `manifests`, `money`, `shop_closed`, `pulse`, `map`, `plan`). Enforces zero-rerender in-memory patching for `DRIVER_LOCATION_UPDATED` events without refetching parent rollups.
- `packages/ui-kit` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/packages/ui-kit`): Shared design system components (buttons, tables, modals, badges, inputs).
- `packages/ui-maps` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/packages/ui-maps`): MapLibre GL and H3 spatial visualization components for vehicle routes and delivery zones.
- `packages/ui-charts` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/packages/ui-charts`): Recharts wrappers for velocity, revenue, and inventory time-series analytics.
- `packages/motion-tokens` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/packages/motion-tokens`): Framer Motion transition curves, physics tokens, and animations.
- `packages/pulse-ui` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/packages/pulse-ui`): Real-time pulse indicator widgets, status pills, and live telemetry badges.
- `packages/explain-ui` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/packages/explain-ui`): AI decision explanation widgets (explaining why a safety stock level or auto-order recommendation was made).
- `packages/i18n` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/packages/i18n`): Localization dictionaries for Uzbek (Latin & Cyrillic), Russian, and English.
- `packages/validation` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/packages/validation`): Zod schemas for form and API payload validation.

### 6.2 Mobile Native Packages
- **Android**:
  - `packages/mobile-android-kit`: Shared Android utility classes, coroutine scopes, and networking helpers.
  - `packages/mobile-android-design`: Jetpack Compose theme, typography, and color tokens.
  - `packages/mobile-android-barcode-scanner`: CameraX barcode scanning implementation.
- **iOS**:
  - `packages/mobile-ios-core`: Shared Swift network actors, error types, and storage.
  - `packages/mobile-ios-kit`: Reusable SwiftUI view components.
  - `packages/mobile-ios-design`: Apple Human Interface design tokens.
  - `packages/mobile-ios-barcode`: AVFoundation barcode capture session manager.

### 6.3 Shared Go Packages
- `packages/optimizer-contract` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/packages/optimizer-contract`): Go types and error models for external optimization solvers.
- `packages/handoff` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/packages/handoff`): Go package for structured handoff processing.

---

## 7. Database Architecture & Schema Management

### 7.1 Database Engine
- **Engine**: PostgreSQL 16 with TimescaleDB (`timescale/timescaledb-ha:pg16`).
- **Connection Configuration**: Tuned for high-concurrency production workloads in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/docker/postgres.conf`:
  - `shared_buffers = 512MB`
  - `work_mem = 16MB`
  - `maintenance_work_mem = 128MB`
  - `wal_buffers = 16MB`
  - `checkpoint_completion_target = 0.9`
  - `random_page_cost = 1.1`
  - `effective_io_concurrency = 200`

### 7.2 Migrations & Seeds
- **Migrations Directory**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/database/migrations`
  - 78 Sequential SQL migration files (numbered `001_initial_schema.sql` to `077_supervisor_override_pin.sql`, with enterprise hardening additions).
  - Establishes relational integrity, foreign key cascades, unique constraints, and PostgreSQL triggers for transactional outbox emission.
- **Seeds Directory**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/database/seeds`
  - 31 Structured seed datasets (e.g. `001_seed_dev.sql`, `002_production_reality_seed.sql`, `003_fleet_and_pairing_seed.sql`, up to `030_factory_production_seed.sql`).
  - Populates realistic enterprise FMCG supply chains in Uzbekistan (Pepsi, Korzinka, Makro, Sergeli DC, Tashkent routes).

---

## 8. Infrastructure, Deployment & Operations

### 8.1 Docker & Docker Compose Fleet
1. `docker-compose.base.yml` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/docker-compose.base.yml`): Base services (`postgres`, `redis`, `backend`, `planning`, `supplier-portal`, `retailer-portal`, `warehouse-portal`).
2. `docker-compose.dev.yml` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/docker-compose.dev.yml`): Development overlay exposing database ports and mounting seed datasets.
3. `docker-compose.prod.yml` (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/docker-compose.prod.yml`): Production deployment stack featuring:
   - **Caddy 2 Reverse Proxy**: Automatic TLS, HTTP/3, and virtual host routing (`docker/Caddyfile`).
   - **TimescaleDB pg16**: Mounts custom `postgres.conf`.
   - **Redis 7**: Password-protected with AOF persistence.
   - **Web Portals & MiniApp**: Multi-stage Docker builds.
   - **Telegram Bot**: Containerized Node.js service.
   - **Monitoring Stack**: Prometheus v2.51.0 and Grafana 10.4.0 with pre-configured supply chain dashboards.

### 8.2 Kubernetes Fleet (`infra/k8s/`)
The Kubernetes configuration uses **Kustomize** (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/infra/k8s/base/kustomization.yaml`):
- **Namespace**: `pegasusx`
- **Core Components**:
  - `backend-go`: Deployment, Service, HorizontalPodAutoscaler (HPA), PodDisruptionBudget (PDB), ConfigMap.
  - `planning-engine`: Deployment, Service, HPA, ConfigMap.
  - `osrm`: Deployment, Service, PersistentVolumeClaim for offline road network routing.
  - `web-portals`: Deployments and services for `supplier-portal`, `warehouse-portal`, and `retailer-portal`.
  - `kafka`: Strimzi Kafka cluster (`kafka.yaml`) and canonical topics (`kafka-topics.yaml`).
  - `external-secrets`: SecretStore and ExternalSecret syncing Vault/GCP secrets into Kubernetes secrets.
  - `ingress`: ManagedCertificate, BackendConfig, FrontendConfig, and Ingress with Cloud Armor integration.
  - `podmonitoring`: Google Managed Prometheus monitoring definitions.
- **Overlays**: Staging (`infra/k8s/overlays/staging`) and Production (`infra/k8s/overlays/production`).

### 8.3 Terraform Infrastructure on GCP (`infra/terraform/`)
The root configuration (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/infra/terraform/main.tf`) composes six enterprise modules:
1. `modules/networking`: VPC, private subnets, Cloud Router, Cloud NAT with 2 static egress IPs, Private Service Access (PSA) peering, and firewall rules.
2. `modules/database`: Cloud SQL PostgreSQL 16 HA, Cloud Spanner ledger, and Cloud Memorystore Redis 7.0 HA.
3. `modules/messaging`: Google Managed Service for Apache Kafka with canonical topics.
4. `modules/storage_security`: 4 Google Cloud Storage buckets (media, updates, imports, tf-state), Cloud Armor WAF security policies, Secret Manager, and Workload Identity bindings.
5. `modules/compute`: GKE Autopilot/Standard regional multi-zone cluster with dedicated node service accounts.
6. `modules/monitoring`: Cloud Monitoring alert policies, notification channels, dashboards, and synthetic uptime probes.

### 8.4 CI/CD Pipelines & Testing Automation
1. **GitHub Actions CI Workflow** (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/.github/workflows/ci.yml`):
   - Changed-subsystem path filtering (`dorny/paths-filter@v3`).
   - Web Suite: `turbo run check-types`, `lint`, `test`, `build`.
   - Backend Go Suite: `go test -v -race -cover ./...`.
   - Planning Python Suite: `python3 -m unittest discover -s tests -v`.
   - iOS Suite: `swift test -q` across all 5 iOS packages.
   - Android Suite: `./gradlew testDebugUnitTest` across all 5 Android apps.
   - Infrastructure Validation: `terraform validate` and `kubectl kustomize` validation across all overlays.
2. **Dual-Engine E2E Testing**:
   - **Playwright TypeScript**: Multi-browser execution across Chromium, Firefox, and WebKit (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/.github/workflows/e2e.yml`).
   - **Playwright Go**: Go-native browser automation suite testing auth flows, warehouse operations, and control towers (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/tests/e2e/e2e_test.go`).

---

## 9. Recommendations for `pegasus.x/agents.md`

Based on the verified codebase reality, `pegasus.x/agents.md` must be constructed according to the following tailored specifications:

### 9.1 Purpose & Final Ecosystem Goal
- **Mission**: Document and maintain the Pegasus.X sovereign supply chain operating system for Uzbekistan B2B FMCG distribution.
- **Scale**: Multi-tenant, multi-region (SOATO), multi-role ecosystem supporting high-frequency wholesale ordering, automated warehouse logistics, mobile driver execution, and national fiscal compliance.

### 9.2 Critical Architectural Conventions
1. **Financial & Currency Invariant**:
   - All monetary values MUST be stored and computed as 64-bit signed integers in **tiyins** (`1 UZS = 100 tiyins`).
   - Floating-point currency representation in backend logic is strictly prohibited.
   - Uzbekistan VAT (12%) must be computed using basis points (`1200 bps`) and integer half-up rounding via `fiscal.CalculateInvoiceTotals`.
2. **National Fiscal & Legal Compliance**:
   - Every invoice must support 17-digit national MXIK codes, packaging codes, and E-Imzo cryptographic signing.
   - Doorstep discrepancies must be processed through UMP and emit Art. 257 Tuzatuvchi (corrective) facturas.
   - Asl Belgisi Track & Trace aggregation hierarchy must be preserved (Unit CIS -> Carton ATK -> Pallet SSCC).
3. **Logistics & Dispatch Constraints**:
   - CVRP dispatch planning must strictly enforce the **95% volumetric Tetris buffer** (`tetris_buffer = 0.95`).
   - Doorstep delivery completion is strictly gated behind the **`< 150m` GPS proximity check**.
   - Driver cash variance must trigger `DISCREPANCY_FLAGGED` and lock the driver from new shift assignments.
4. **WebSocket & Real-Time Invalidation**:
   - Live location updates (`DRIVER_LOCATION_UPDATED`) must NEVER trigger full dashboard rollup invalidation; they must patch in-memory coordinates via `applyDriverLocationPatch`.
   - Real-time event consumption must adhere to the `dashboardDirtySlice` contract in `@pegasusx/ws-refresh-contract`.
5. **The 4 Onboarding Precondition Gates (HTTP 428)**:
   - Never bypass or mock out onboarding status checks on operational endpoints; database status in PostgreSQL is authoritative.

### 9.3 Honesty Rules ("Zero Theatre")
- Every claim made by an AI agent must cite exact source code paths in the format `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus.x/...`.
- Speculation about "missing" features or hypothetical architectures is strictly forbidden: if a feature exists, point to its Go handler, Python engine, SQL migration, or UI component.
- Do not introduce placeholder implementations, mock data, or fake passes. Always verify against the 52 living loop steps in `backend/cmd/smokecheck/main.go`.

### 9.4 Verification & Testing Procedures
- Any proposed backend change must pass `cd backend && go test -v -race ./...` and `go run cmd/smokecheck/main.go`.
- Any frontend or shared package change must pass `pnpm check-types` and `pnpm test`.
- Mobile modifications must be verified via `swift test -q` (iOS) and `./gradlew testDebugUnitTest` (Android).
- Infrastructure modifications must pass `terraform validate` and `kubectl kustomize`.
