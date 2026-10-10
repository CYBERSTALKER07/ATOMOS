# Pegasus Ecosystem Codebase Survey & Architectural Reality Report

**Generated:** 2026-09-26  
**Investigated Directory:** `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus`  
**Report Artifact:** `file:///Users/shakhzod/Desktop/V.O.I.D/.agents/explorer_survey_1/survey_pegasus.md`  
**Working Mode:** Teamwork Read-Only Explorer (Zero Hallucination / 100% Code Grounding)

---

## 1. Executive Summary & Ecosystem Overview

The **Pegasus** platform is an enterprise-grade, hyperscale multi-tenant B2B supply chain, logistics execution, and predictive commerce operating system. Built to serve hundreds of thousands of retail shops, suppliers, regional warehouses, and manufacturing factories across Central Asia and international markets, the ecosystem coordinates the entire end-to-end commerce lifecycle:
1. **Predictive Commerce & Empathy Engine:** AI-driven SKU-level purchase pattern forecasting, scheduled pre-order generation, and automated replenishment loops.
2. **Multi-Role Desktop & Mobile Operations:** Native and web/desktop portals tailored for 5 core personas: Suppliers, Warehouse Operators, Factory Managers, Retailers, and Fleet Drivers.
3. **Hyperscale Execution Engine:** A Go 1.25 transactional core backed by Google Cloud Spanner (94 tables, multi-region `nam-eur-asia3` topology, H3 geospatial hexagonal indexing), Apache Kafka (KRaft mode, 8 event topics with 128 partitions), Redis Memorystore, and transactional outbox relays.
4. **Operations Research & Dispatch Optimization:** Vehicle Routing Problem (VRP) and Constraint Programming (CP-SAT) solvers running as high-throughput Rust sidecars (`optimizer-core`) and Python AI worker microservices.
5. **Multi-Gateway Payment & Double-Entry Treasury:** Immutable fee snapshots, degressive regional fee scheduling, multi-provider execution (Adyen, Global Pay, Airwallex, Payme, Click), and strict double-entry ledgering with anomaly detection.

---

## 2. Repository Topology & File Tree Structure

The Pegasus repository (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus`) is organized as a high-density polyglot monorepo containing Go services, TypeScript/React web portals, Rust sidecars, Python AI engines, Kotlin/Compose Android apps, and Swift/SwiftUI iOS apps:

```
pegasus/
├── Makefile                               # Central developer & emulator orchestrator
├── docker-compose.yml                     # Local emulator fleet (Kafka, Redis, Spanner, Firebase, WireMock)
├── package.json                           # Root workspace governance, guard scripts, Playwright test harness
├── go.work                                # Go multi-module workspace
├── go.work.sum                            # Checksums for Go workspace
├── playwright.config.ts                   # E2E multi-role browser & API test runner
├── v.o.i.d._features.yaml                 # Master Hyperscale Feature Registry (569 lines)
├── cors.json                              # Cloud Storage CORS specification
├── firebase.json                          # Firebase emulator port mapping
├── replace_gateways.sh                    # Gateway migration helper
│
├── apps/                                  # 18 Application Subsystems
│   ├── backend-go/                        # Core backend API service (Go 1.25, Chi, Spanner, Kafka)
│   ├── ai-worker/                         # Predictive forecasting & import worker (Go, gRPC, Kafka)
│   ├── admin-portal/                      # Supplier & Global Admin portal (Next.js 15, Tauri 2, HeroUI)
│   ├── factory-portal/                    # Factory replenishment & dispatch desktop portal (Next.js, Tauri)
│   ├── warehouse-portal/                  # Warehouse inventory & payload receipt desktop portal (Next.js, Tauri)
│   ├── retailer-app-desktop/              # Retailer POS & wholesale desktop portal (Next.js 15, Tauri 2)
│   ├── payload-terminal/                  # Cross-platform QR scanner terminal (Expo 55, React Native 0.83)
│   ├── driver-app-android/                # Driver mobile app (Kotlin, Jetpack Compose, Hilt, Room/Offline)
│   ├── driverappios/                      # Driver mobile app (Swift, SwiftUI, SwiftData, CoreLocation)
│   ├── retailer-app-android/              # Retailer mobile app (Kotlin, Jetpack Compose, Material 3)
│   ├── retailer-app-ios/                  # Retailer mobile app (Swift, SwiftUI, HIG)
│   ├── factory-app-android/               # Factory loading bay mobile app (Kotlin, Jetpack Compose)
│   ├── factory-app-ios/                   # Factory loading bay iOS app (Swift, SwiftUI)
│   ├── warehouse-app-android/             # Warehouse operator mobile app (Kotlin, Jetpack Compose)
│   ├── warehouse-app-ios/                 # Warehouse operator iOS app (Swift, SwiftUI)
│   ├── payload-app-android/               # Payloader scanner mobile app (Kotlin, Jetpack Compose)
│   ├── payload-app-ios/                   # Payloader scanner iOS app (Swift, SwiftUI)
│   ├── synthetic-tester/                  # High-throughput load generation tool (Go)
│   └── parse_features.py                  # Script to parse v.o.i.d._features.yaml
│
├── services/                              # Autonomous Microservices & Sidecars
│   ├── optimizer-core/                    # High-performance VRP & CP-SAT solver
│   │   ├── proto/                         # Protocol buffer specifications (`optimizer_core.proto`)
│   │   ├── server-rust/                   # Production Rust gRPC solver sidecar (Tonic, Prost, Tokio)
│   │   ├── adapters/go/                   # Go Kafka-to-gRPC adapter worker (`cmd/optimizer-worker`)
│   │   └── server/                        # Deprecated reference Python solver
│   └── deep-agents/                       # LangChain / LangGraph workspace (`void-deep-agents`)
│
├── packages/                              # 8 Shared Packages
│   ├── ai-bridge/                         # Gemini AI provider & schema mapper (Go)
│   ├── api-client/                        # Shared HTTP client for web portals (TypeScript)
│   ├── config/                            # Environment variable parser & validator (Go)
│   ├── i18n/                              # Internationalization engine (EN, RU, UZ-Latn, UZ-Cyrl, TR, AR)
│   ├── optimizer-contract/                # Solver request/response contract & job envelope (Go)
│   ├── types/                             # Universal TypeScript definitions & WS events (44KB ws-events.ts)
│   ├── ui-kit/                            # Material 3 Tailwind CSS design system tokens
│   └── validation/                        # Shared runtime validation contracts
│
├── infra/                                 # Cloud & Infrastructure Provisioning
│   ├── terraform/                         # GCP Terraform IaC (Spanner, Memorystore, Cloud Run, GKE, Multi-region)
│   ├── k8s/                               # Kubernetes manifests (deployments, HPAs, PDBs, KEDA, Prometheus)
│   ├── spanner/                           # Spanner DDL schema definitions
│   └── chaos/                             # Chaos engineering fault injection experiments
│
├── contracts/                             # Interface Contracts
│   └── events.schema.json                 # Auto-generated JSON Schema for all Kafka & WS events (142KB)
│
├── context/                               # Machine-Readable Inventories
│   ├── architecture-graph.json            # Canonical dependency and topology graph (125KB)
│   └── technology-inventory.json          # Complete technology and contract inventory (117KB)
│
├── mocks/                                 # WireMock Simulation Fixtures
│   └── globalpay/                         # WireMock stubs for Global Pay payment gateway simulator
│
├── scripts/                               # Architecture, Security & Contract Guardrails
│   ├── sprint1_execution_gate.py          # Master enterprise gate runner
│   ├── versionscan.py                     # Version scanning & enforcement engine (34KB)
│   ├── contract_drift_guard.py            # Contract drift detection
│   ├── architecture_boundary_guard.py     # Architectural boundary enforcer
│   ├── design_token_enforcement_guard.py  # Material 3 design token enforcer
│   ├── production_safety_guard.py         # Production safety validator
│   ├── security_guard.py                  # Static security auditing
│   └── parity/gen_contracts_gate.sh       # Gen-contracts schema synchronization gate
│
└── tests/                                 # Playwright E2E Test Suites
    ├── cross-role/                        # Multi-persona end-to-end integration journeys
    ├── supplier/                          # Admin & Supplier web portal tests
    ├── retailer/                          # Retailer desktop web tests
    ├── factory/                           # Factory replenishment & dispatch tests
    ├── warehouse/                         # Warehouse stock receipt & dispatch tests
    ├── driver-api/                        # Driver API contract tests
    ├── payloader-api/                     # Payloader API contract tests
    └── fixtures/                          # Reusable test auth tokens and mocks
```

---

## 3. Package & Dependency Manifests

### 3.1 Go Workspaces & Manifests
- **Go Workspace Manifest:** `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/go.work`
  - Go Version: `1.25.0`
  - Member Modules:
    - `./adyen-go-api-library-main`
    - `./apps/ai-worker`
    - `./apps/backend-go`
    - `./packages/ai-bridge`
    - `./packages/config`
    - `./packages/optimizer-contract`
    - `./services/optimizer-core/adapters/go`
- **Core Backend Manifest:** `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/go.mod`
  - Module: `backend-go` (Go 1.25.0)
  - Key Dependencies:
    - HTTP Router: `github.com/go-chi/chi/v5` v5.2.5
    - Database: `cloud.google.com/go/spanner` v1.88.0, `cloud.google.com/go/storage` v1.56.0, `cloud.google.com/go/secretmanager` v1.18.0
    - Messaging: `github.com/segmentio/kafka-go` v0.4.50, `github.com/gorilla/websocket` v1.5.3
    - Caching: `github.com/redis/go-redis/v9` v9.18.0, `github.com/alicebob/miniredis/v2` v2.37.0
    - Authentication: `firebase.google.com/go/v4` v4.19.0, `github.com/golang-jwt/jwt/v5` v5.3.1, `golang.org/x/crypto` v0.49.0
    - Geospatial: `github.com/uber/h3-go/v4` v4.4.1 (H3 hexagonal hierarchical spatial index)
    - Payments: `github.com/adyen/adyen-go-api-library/v21` v21.2.0
    - Observability: `go.opentelemetry.io/otel` v1.43.0, `github.com/prometheus/client_golang` v1.23.2
    - Concurrency & Async: `golang.org/x/sync` v0.20.0 (singleflight, errgroup), `github.com/robfig/cron/v3` v3.0.1
- **AI Worker Manifest:** `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/ai-worker/go.mod`
  - Module: `pegasus-ai-worker` (Go 1.25.0)
  - Dependencies: `cloud.google.com/go/spanner`, `cloud.google.com/go/storage`, `github.com/segmentio/kafka-go`, `github.com/xuri/excelize/v2` v2.10.1 (Excel import parser), `google.golang.org/grpc` v1.80.0
- **Optimizer Adapter Go Worker:** `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/services/optimizer-core/adapters/go/go.mod`
  - Module: `optimizercoreadapter` (Go 1.25.0)
  - Dependencies: `cloud.google.com/go/spanner`, `github.com/redis/go-redis/v9`, `google.golang.org/grpc`, `github.com/segmentio/kafka-go`
- **Packages Go Manifests:**
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/packages/config/go.mod`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/packages/optimizer-contract/go.mod`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/packages/ai-bridge/go.mod`

### 3.2 Node / TypeScript Manifests
- **Root Package:** `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/package.json`
  - DevDependencies: `@playwright/test` ^1.52.0
  - Scripts: Localization generation, Version scanning, MCP architectural guards, Playwright E2E suites, Tauri desktop dev/build targets
- **Admin Portal:** `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/admin-portal/package.json`
  - Framework: Next.js 15.5.12 (Turbopack), React 19.1.0, Tauri 2.10.1
  - Dependencies: `@heroui/react` ^3.0.2, `@heroui/styles` ^3.0.2, `tailwindcss` ^4, `h3-js` ^4.4.0, `maplibre-gl` ^5.19.0, `mapbox-gl` ^3.20.0, `recharts` ^3.8.0, `firebase` ^12.12.0
- **Factory Portal:** `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/factory-portal/package.json`
  - Next.js 15.5.12, React 19.1.0, Tauri 2.10.1, HeroUI 3.0.2, Tailwind 4, Lucide React
- **Warehouse Portal:** `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/warehouse-portal/package.json`
  - Next.js 15.5.12, React 19.1.0, Tauri 2.10.1, HeroUI 3.0.2, Framer Motion, Recharts
- **Retailer App Desktop:** `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/retailer-app-desktop/package.json`
  - Next.js 15.5.12, React 19.1.0, Tauri 2.10.1, `qrcode.react`, MapLibre GL, Cypress
- **Payload Terminal:** `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/payload-terminal/package.json`
  - Expo 55.0.4, React Native 0.83.2, React 19.2.0, NativeWind 4.2.2, Expo Secure Store, Expo Haptics
- **Shared Packages:**
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/packages/types/package.json`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/packages/i18n/package.json`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/packages/api-client/package.json`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/packages/ui-kit/package.json`

### 3.3 Rust & Python Manifests
- **Optimizer Core Rust Sidecar:** `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/services/optimizer-core/server-rust/Cargo.toml`
  - Edition: 2021
  - Dependencies: `tokio` 1.43, `tonic` 0.12 (gRPC server), `prost` 0.13, `tracing` 0.1, `tonic-build` 0.12
- **Tauri Shells Cargo Manifests:**
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/admin-portal/src-tauri/Cargo.toml`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/factory-portal/src-tauri/Cargo.toml`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/warehouse-portal/src-tauri/Cargo.toml`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/retailer-app-desktop/src-tauri/Cargo.toml`
- **Deep Agents Workspace:** `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/services/deep-agents/pyproject.toml`
  - Python: `>=3.11`
  - Dependencies: `deepagents>=0.7.5`, `langchain>=1.3.0`, `langgraph>=1.2.0`, `langchain-openai>=1.4.0`, `langchain-xai>=1.3.0`

### 3.4 Mobile Gradle & Xcode Project Manifests
- **Android Gradle Manifests (Kotlin 2.x, Compose, SDK 35, Hilt, Room):**
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/driver-app-android/app/build.gradle.kts`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/retailer-app-android/app/build.gradle.kts`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/factory-app-android/app/build.gradle.kts`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/warehouse-app-android/app/build.gradle.kts`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/payload-app-android/app/build.gradle.kts`
- **iOS Xcode Projects (Swift 6, SwiftUI, SwiftData, Apple HIG):**
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/driverappios/driverappios.xcodeproj`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/retailer-app-ios/retailerapp`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/factory-app-ios/factory-app-ios.xcodeproj`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/warehouse-app-ios/warehouse-app-ios.xcodeproj`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/payload-app-ios/payload-app-ios.xcodeproj`

---

## 4. Architecture & Design Principles

### 4.1 Composition Root & Lifecycle Management
All long-lived clients, Redis connections, Spanner pools, WebSocket hubs, and domain services are constructed inside a centralized composition root at `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/bootstrap/app.go` (`NewApp()`).
- Handlers receive strictly typed dependencies via dependency injection (`Deps` structs) passed to domain router registrars.
- `apps/backend-go/main.go` acts solely as the orchestrator: parsing configuration, invoking `bootstrap.NewApp`, mounting Chi domain routers, launching background crons and Kafka consumers, and executing graceful SIGTERM shutdowns with a 30-second context and OpenTelemetry trace buffer flushing (`TracerShutdown`).

### 4.2 Transactional Outbox Pattern (V.O.I.D. Phase VII)
To guarantee consistency between Spanner database mutations and Kafka event streams without two-phase commit overhead:
- **Table Definition:** `OutboxEvents` in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/schema/spanner.ddl` (lines 2151–2170).
- **Outbox Engine:** `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/outbox/relay.go`.
- **Mechanism:** Any domain transaction writes state mutations and an `OutboxEvents` row atomically within the same Spanner `ReadWriteTransaction`. The `outbox.Relay` background loop polls `OutboxEvents` where `Status = 'PENDING'`, publishes the message to the corresponding Kafka topic via synchronous writer, and marks the event `PUBLISHED` (or routes to `OutboxDLQ` on persistent failure).

### 4.3 Multi-Region Read Routing (Maglev Spanner Router)
- **Source:** `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/bootstrap/spannerrouter/router.go`.
- **Mechanism:** When `enable_multiregion=true` is enabled, reads are routed to the nearest regional read replica based on the entity's H3 spatial index, minimizing latency across continents (`asia-south1`, `europe-west1`, `us-central1`), while all writes route strictly through `SpannerRouter.Primary()`.

### 4.4 Geospatial Sovereignty via Uber H3
- **Source:** `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/proximity/h3.go` and `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/proximity/proximity.go`.
- **Mechanism:** Avoids expensive O(N) Cartesian/Haversine math. Physical entities (Retailers, Warehouses, Factories) have an indexed H3 resolution-7 cell (`H3Index` / `H3Cell`). Warehouse service territories are represented as compact H3 polygon arrays (`Warehouses.H3Indexes`). Real-time proximity, territory validation, and clustering use H3 hex operations (`PolygonToCells`, `GridDisk`, `GridDistance`) with panic-safe fallback to Haversine approximations.

### 4.5 Dual Authentication & Single-Device Enforcement
- **Source:** `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/auth/auth.go` and `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/auth/device.go`.
- **Mechanism:** Dual-mode authentication supports Google Firebase Auth tokens (verified against Firebase Auth / emulator via `InitFirebaseAuth`) and HMAC-SHA256 signed JWTs.
- **Edge 24 Single-Device Enforcement:** On every login, the client's `X-Device-Id` is registered in `DeviceFingerprints` (`schema/spanner.ddl:1522`). If a newer device logs in, previous devices are invalidated and forcefully evicted via a WebSocket `FORCE_LOGOUT` frame.

### 4.6 Single-Flight Cache Coalescing & Backpressure
- **Source:** `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/cache/cache.go` and `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/cache/backpressure.go`.
- **Mechanism:** High-traffic read keys (catalog, retailer profiles, pricing overrides) use Redis with `golang.org/x/sync/singleflight`. Concurrent cache misses for the same key coalesce into a single Spanner read query, preventing database stampedes. The `BackpressureEngine` monitors system load and sheds non-critical background traffic when Spanner or CPU thresholds breach safety limits.

---

## 5. Backend Services & Core Domains (`apps/backend-go`)

The backend codebase (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go`) contains 75 domain packages and routes. Below is the complete architecture of its operational domains:

### 5.1 Authentication & RBAC (`authroutes`, `auth`)
- **Route File:** `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/authroutes/routes.go`
- **Endpoints:**
  - `POST /v1/auth/login` — Universal login for all actor roles
  - `POST /v1/auth/refresh` — Refresh expired JWT tokens
  - `POST /v1/auth/supplier/register` — 4-step supplier registration wizard
  - `POST /v1/auth/supplier/login` — Supplier admin credentials verification
  - `POST /v1/auth/driver/login` — Driver credential check & device assignment
  - `POST /v1/auth/retailer/login`, `POST /v1/auth/retailer/register` — Retailer phone & SMS OTP auth
  - `POST /v1/auth/factory/login`, `POST /v1/auth/factory/register` — Factory manager credentials
  - `POST /v1/auth/warehouse/login`, `POST /v1/auth/warehouse/register` — Warehouse manager credentials
  - `POST /v1/auth/payloader/login` — Loading bay scanner credentials
  - `POST /debug/mint-token` — Development token generator (strictly mounted when `ENVIRONMENT=development`)

### 5.2 Order Core & Lifecycle Machine (`order`, `orderroutes`, `retailerroutes`)
- **Source Files:**
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/order/service.go`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/orderroutes/routes.go`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/retailerroutes/routes.go`
- **State Machine States:**
  `PENDING` → `PENDING_REVIEW` → `LOADED` → `DISPATCHED` → `IN_TRANSIT` → `ARRIVING` → `ARRIVED` → `AWAITING_PAYMENT` / `PENDING_CASH_COLLECTION` → `COMPLETED`  
  *Exceptions:* `CANCEL_REQUESTED`, `CANCELLED`, `ARRIVED_SHOP_CLOSED`, `NO_CAPACITY`, `DELIVERED_ON_CREDIT`, `SCHEDULED`, `QUARANTINE`, `STALE_AUDIT`.
- **Endpoints:**
  - `POST /v1/retailer/orders` / `POST /v1/order/create` — Retailer checkout & preorder placement
  - `GET /v1/orders/{id}` — Order detail with line items and real-time state
  - `POST /v1/orders/request-cancel` — Retailer cancellation request (prior to dispatch)
  - `POST /v1/order/deliver` — Driver arrival & geofence handshake
  - `POST /v1/order/validate-qr` — QR code manifest validation between driver and retailer
  - `POST /v1/order/confirm-offload` — Delivery item verification and offload
  - `POST /v1/order/collect-cash` — Driver cash collection confirmation
  - `POST /v1/order/amend` — Post-negotiation quantity and price amendments

### 5.3 Delivery Execution & Edge Cases (`deliveryroutes`, `order`)
- **Route File:** `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/deliveryroutes/routes.go`
- **Edge Protocols:**
  - `POST /v1/delivery/shop-closed` — Driver flags shop closed; triggers Shop-Closed Protocol: sends push notification to retailer, starts grace period timer, and escalates to admin.
  - `POST /v1/retailer/shop-closed-response` — Retailer responds (`OPEN_NOW`, `5_MIN`, `CALL_ME`, `CLOSED_TODAY`).
  - `POST /v1/delivery/bypass-offload` — Admin issues 6-digit bypass token when retailer cannot scan QR.
  - `POST /v1/delivery/negotiate` — Driver proposes downward item edits at the doorstep.
  - `POST /v1/delivery/credit-delivery` — Retailer approved for credit payment offloads without cash.
  - `POST /v1/delivery/split-payment` — Combination of partial cash and payment gateway.
  - `POST /v1/delivery/missing-items` — Discrepancy report recorded in `ManifestExceptions`.

### 5.4 Supplier Operations, Planning, Catalog & Logistics
- **Source Files:**
  - Catalog: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/suppliercatalogroutes/routes.go`
  - Core Operations: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/suppliercoreroutes/routes.go`
  - Logistics: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/supplierlogisticsroutes/routes.go`
  - Planning: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/supplierplanningroutes/routes.go`
  - Insights: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/supplierinsightsroutes/routes.go`
  - Entity Resolution: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/entityresolutionroutes/routes.go`
- **Core Functionality:**
  - Product Catalog CRUD with tier pricing and quantity step-sizes.
  - Dynamic B2B Retailer Pricing Engine with overrides (`RetailerPricingOverrides`).
  - Fleet management, driver vehicle assignments, truck capacity tracking.
  - Picking Manifest generation (`SupplierTruckManifests`, `ManifestOrders`).
  - Delivery zones polygon management (`DeliveryZones`) with H3 resolution-7 coverage.
  - Graph query analytics (`POST /v1/supplier/analytics/graph/query`) and forecast tournaments (`POST /v1/supplier/analytics/forecast/tournament`).
  - AI-assisted Entity Resolution (`POST /v1/supplier/entity-resolution/resolve` & `/explain`).

### 5.5 Factory & Warehouse Internal Replenishment Chain
- **Source Files:**
  - Factory Routes: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/factoryroutes/routes.go`
  - Warehouse Routes: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/warehouseroutes/routes.go`
  - Replenishment Services: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/factory/`
- **Internal Supply Chain Architecture:**
  - **InternalTransferOrders:** Tracks stock movement from Factories to Warehouses across states: `DRAFT`, `APPROVED`, `LOADING`, `DISPATCHED`, `IN_TRANSIT`, `ARRIVED`, `RECEIVED`, `CANCELLED`.
  - **FactoryTruckManifests:** Groups multi-transfer freight into truck routes with volume unit (VU) limits.
  - **SupplyLanes:** Configurable production lanes between factories and destination warehouses with SLA thresholds.
  - **PullMatrix & PredictivePush:** 4-hour background aggregators evaluating current inventory burn rates, preorder pipelines, and lead times to generate predictive stock replenishment orders.
  - **DispatchLocks:** Mutual exclusion locks preventing fleet dispatch while inventory restock or territory adjustments are active.
  - **ForceReceiveService:** Warehouse operator override for incoming shipments with missing manifest tags.

### 5.6 Treasury, Double-Entry Ledger & Payments
- **Source Files:**
  - Treasury: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/treasury/routes.go` & `treasury.go`
  - Payments: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/paymentroutes/routes.go` & `unified_checkout.go`
  - Gateways: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/payment/`
- **Ledger Architecture:**
  - **Double-Entry Ledger:** All funds are accounted in `LedgerEntries` with debit/credit balance invariance. Background reconciliation cron (`StartReconciliationCron`) sweeps for balance discrepancies and flags anomalies in `LedgerAnomalies`.
  - **Immutable Fee Snapshots:** On order checkout, exact fee calculations (platform fee basis points, net supplier payout, warehouse slice) are snapshotted in `InvoiceSettlementSlices` and `MasterInvoices`.
  - **Regional Degressive Fee Engine:** Configured per currency in `RegionalConfigs` (`order/fee_policy.go`) with progressive rate drops at base, growth, and scale turnover thresholds.
  - **Payment Gateways Supported:**
    - **Adyen:** Hosted checkout Payment Links & direct webhooks (`payment/adyen_client.go`, `adyen_webhook.go`).
    - **Global Pay:** Local card tokenization, direct charge execution, and WireMock-backed emulation (`payment/global_pay_cards.go`, `global_pay_direct.go`).
    - **Airwallex:** Direct execution & checkout link generation (`payment/airwallex_client.go`).
    - **Payme & Click:** Uzbek national payment gateways with HMAC-SHA256 signature verification and idempotency locks (`payment/webhooks.go`).
  - **Chargeback Service:** Handles chargeback lifecycle (`NOTIFICATION_OF_CHARGEBACK`, `CHARGEBACK`, `SECOND_CHARGEBACK`, `CHARGEBACK_REVERSED`).

### 5.7 Real-time WebSocket Hubs (`ws`, `telemetry`)
- **Source Files:** `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/ws/` & `telemetry/hub.go`
- **Hub Ecosystem:**
  - `ws.FleetHub` (`/ws/fleet`, `/ws/telemetry`) — Real-time GPS coordinate ingestion from driver devices; broadcasts live fleet positions to admin dashboards.
  - `ws.DriverHub` (`/v1/ws/driver`) — Pushes new route assignments, stop cancellations, and urgent reroutes to drivers.
  - `ws.RetailerHub` (`/ws/retailer`) — Pushes `DRIVER_APPROACHING`, cart sync updates, and AI preorder prompts to retail shops.
  - `ws.WarehouseHub` (`/ws/warehouse`) — Live dispatch lock states, outbox failure alerts, and transfer inbound arrivals.
  - `ws.PayloaderHub` (`/v1/ws/payloader`) — Loading dock manifest seal updates and QR scanner acknowledgments.
  - `ws.SupplierHub` (`/ws/supplier`) — Supplier live KPI telemetry and async optimization solved notifications.
  - `ws.FactoryHub` — Factory loading bay status and outbound manifest progress.
  - **CommandRegistry:** Redis-backed command handshake protocol guaranteeing verified execution of desktop-to-mobile dispatch orders with native client ACK receipts.

### 5.8 Background Cron Engines (`cron.go`)
- **Source File:** `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/cron.go`
- **17 Background Engines:**
  1. `StartAwakener` (lines 32–145) — Hourly temporal heartbeat. Sweeps `AIPredictions` past `TriggerDate`, validates auto-order settings, and converts predictions to pending preorders with FCM/Telegram notifications.
  2. `StartScheduledOrderPromoter` (lines 146–229) — Promotes `SCHEDULED` preorders to `PENDING` within 24 hours of fulfillment delivery date.
  3. `StartGlobalPaySweeper` (lines 230–274) — Sweeps expired/stale payment sessions and synchronizes status with gateways.
  4. `StartPaymentSessionExpirer` (lines 275–349) — Expires abandoned checkout sessions and notifies retailers.
  5. `StartStaleOrderAuditor` (lines 350–563) — 15-minute sweep detecting orders stuck in transit >12 hours; flags `STALE_AUDIT` and initiates automated refunds.
  6. `StartOrphanedPredictionCleaner` (lines 564–701) — Daily purge of orphaned AI prediction line items.
  7. `StartPreOrderConfirmationSweeper` (lines 702–1086) — Enforces T-4 confirmation lock policy.
  8. `StartAutoConfirmSweeper` (lines 1087–1168) — Auto-confirms unacknowledged preorders past grace period.
  9. `StartNotificationExpirer` (lines 1169–1228) — Soft-deletes expired notification records.
  10. `StartPullMatrixAggregator` (lines 1229–1259) — 4-hour replenishment burn-rate aggregator.
  11. `StartFactorySLAMonitor` (lines 1260–1280) — 30-minute SLA monitor flagging delayed transfers.
  12. `StartCurrentLoadReset` (lines 1281–1324) — 24-hour reset of vehicle/driver load counters.
  13. `StartCoverageAuditor` (lines 1325–1337) — Scans for unassigned retailer H3 cells outside active warehouse polygons.
  14. `admin.StartReconciliationCron` — Daily double-entry ledger audit.
  15. `routing.StartCron` — Field General AI routing optimizer.
  16. `replenishEngine.StartReplenishmentCron` — 4-hour stock deficit replenishment scanner.
  17. `StartWaitlistedOrderPromoter` — Promotes waitlisted orders when inventory is replenished.

---

## 6. Data Models & Spanner Schema

The primary datastore is Google Cloud Spanner (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/schema/spanner.ddl`). The schema contains **94 enterprise tables**:

### Complete Table Catalog (Grouped by Subsystem):

| Subsystem | Tables | Primary Keys / Relationships |
|-----------|--------|------------------------------|
| **Core Logistics & Fleet** | `Retailers`, `Drivers`, `Vehicles`, `Orders`, `OrderLineItems`, `DeliverySessions`, `DeliverySessionAdjustments`, `ScheduledJobs`, `DriverTelemetry`, `GlobalPins` | Orders interleaved or indexed by Retailer/Warehouse; Driver locations mapped in `DriverTelemetry` |
| **Financial & Ledger** | `MasterInvoices`, `SupplierPayoutPolicies`, `InvoiceSettlementSlices`, `LedgerEntries`, `LedgerAnomalies`, `Refunds`, `SupplierGlobalPayntConfigs`, `GlobalPayntSessions`, `GlobalPayntAttempts`, `RetailerCardTokens` | Immutable `InvoiceSettlementSlices`; strict debit/credit entries in `LedgerEntries` |
| **Catalog & Products** | `Products`, `SupplierProducts`, `Categories`, `PlatformCategories`, `RetailerPricingOverrides`, `PricingAuditLog` | Product catalog with category hierarchy and per-retailer pricing overrides |
| **Inventory & Auditing** | `SupplierInventory`, `SupplierInventoryV2`, `InventoryAuditLog`, `InventoryImportSessions`, `InventoryImportRows`, `SupplierImportSessions`, `SupplierImportStagedRows`, `SupplierImportMapping`, `SupplierImportAnalyticsFacts`, `SupplierReturns` | `SupplierInventoryV2` keyed by `(SupplierId, WarehouseId, SkuId)` with audit logs |
| **Retailer Settings & AI** | `RetailerGlobalSettings`, `RetailerSupplierSettings`, `RetailerProductSettings`, `RetailerVariantSettings`, `AIPredictions`, `AIPredictionItems`, `RetailerCarts`, `CorrectionWeights`, `RetailerFamilyMembers`, `RetailerLoyaltyTiers`, `RetailerRatings` | `AIPredictionItems` interleaved in `AIPredictions`; RLHF correction feedback in `CorrectionWeights` |
| **Multi-Facility Supply Chain** | `Suppliers`, `Admins`, `WarehouseStaff`, `Warehouses`, `SupplierUsers`, `Factories`, `FactoryStaff`, `InternalTransferOrders`, `InternalTransferItems`, `FactoryTruckManifests`, `ReplenishmentInsights`, `SupplyRequests`, `FactorySupplyRequestQC`, `SupplyRequestItems`, `DispatchLocks` | Internal transfers from `Factories` to `Warehouses`; manifest aggregation and QA checks |
| **Replenishment Network** | `SupplyLanes`, `ReplenishmentLocks`, `FactorySLAEvents`, `NetworkOptimizationMode`, `PullMatrixRuns`, `SupplierOverrides`, `SupplierRetailerClients`, `DeliveryZones` | Network topology connecting production nodes to fulfillment warehouses |
| **Global Enterprise & Reg.** | `CountryConfigs`, `SupplierCountryOverrides`, `Regions`, `RegionalConfigs`, `SystemConfig`, `BillingMeterEvents`, `BillingSupplierMeters`, `BillingGlobalMeters` | ISO-country and regional operational configs; dynamic tenant billing meters |
| **Audit, Events & Protocols** | `OrderEvents`, `ShopClosedAttempts`, `DeviceFingerprints`, `NegotiationProposals`, `AuditLog`, `OrderActivityEvents`, `DispatchAudit` | Immutable chronological event logging with GPS coordinates and actor roles |
| **Transactional Outbox & Jobs**| `OutboxEvents`, `OutboxDLQ`, `OptimizationJobs`, `SupplierTruckManifests`, `ManifestOrders`, `ManifestExceptions`, `DeviceTokens`, `Notifications` | Outbox pattern persistence, optimization solver queue ledger, and notification queue |

---

## 7. Event-Driven Architecture & Kafka Topics

The messaging backbone (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/backend-go/kafka/events.go`) standardizes event contracts across 8 Kafka topics:

### 7.1 Kafka Topic Topology
1. `pegasus-logistics-events` (128 partitions) — Core order lifecycle, payment settlements, cancellations, and delivery updates.
2. `pegasus-demand-forecast` (128 partitions) — AI demand forecast updates, preorder confirmations, and sales velocity signals.
3. `pegasus-freeze-locks` (16 partitions) — Preorder cancel locks (T-4 hours), inventory freeze locks, and dispatch mutices.
4. `pegasus-driver-sync-events` (16 partitions) — Driver offline sync reconciliations, offline delivery settlement frames.
5. `pegasus-telemetry-raw` (16 partitions) — High-frequency GPS telemetry streams from mobile fleet devices.
6. `inventory.import.events` (16 partitions) — Supplier catalog and bulk stock CSV/Excel upload pipeline events.
7. `pegasus-optimizer-jobs` (16 partitions) — Asynchronous VRP routing and CP-SAT constraint solver job requests.
8. `pegasus-logistics-events-dlq` (1 partition) — Dead Letter Queue for permanently failed event processing.

### 7.2 Event Catalog (Defined in `kafka/events.go`)
- **Order Lifecycle:** `ORDER_DISPATCHED`, `DRIVER_APPROACHING`, `DRIVER_ARRIVED`, `ORDER_STATUS_CHANGED`, `ORDER_COMPLETED`, `ORDER_MODIFIED`, `ORDER_REASSIGNED`.
- **Payment & Settlement:** `PAYMENT_SETTLED`, `PAYMENT_FAILED`, `PAYMENT_GATEWAY_DEGRADED`, `PAYMENT_INTENT_CREATED`, `PAYMENT_CLEARED`, `DELIVERY_DELTA_REFUNDED`, `SETTLEMENT_REQUIRED`, `SETTLEMENT_REVISED`.
- **Edge Protocols:** `SHOP_CLOSED`, `SHOP_CLOSED_RESPONSE`, `SHOP_CLOSED_ESCALATED`, `SHOP_CLOSED_RESOLVED`, `CANCEL_REQUESTED`, `CANCEL_APPROVED`, `EARLY_COMPLETE_REQUESTED`, `EARLY_COMPLETE_APPROVED`, `NEGOTIATION_PROPOSED`, `NEGOTIATION_RESOLVED`, `CREDIT_DELIVERY_MARKED`, `MISSING_ITEMS_REPORTED`, `SPLIT_PAYMENT_CREATED`.
- **Pre-Order & AI:** `PRE_ORDER_NOTIFIED`, `PRE_ORDER_AUTO_ACCEPTED`, `PRE_ORDER_CONFIRMED`, `PRE_ORDER_EDITED`, `PRE_ORDER_CANCELLED`, `AI_ORDER_CONFIRMED`, `AI_ORDER_REJECTED`.
- **Supply Chain:** `SUPPLY_REQUEST_SUBMITTED`, `SUPPLY_REQUEST_ACKNOWLEDGED`, `SUPPLY_REQUEST_READY`, `SUPPLY_REQUEST_FULFILLED`, `MANIFEST_REBALANCED`, `TRANSFER_UNASSIGNED`, `MANIFEST_CANCELLED`.
- **Optimization:** `OPTIMIZATION_JOB_QUEUED`, `OPTIMIZATION_SOLVED`, `ROUTE_CREATED`, `ORDER_ASSIGNED`.

---

## 8. Optimizer & AI Subsystems

### 8.1 AI Worker (`apps/ai-worker`)
- **Source Files:**
  - Prediction Engine: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/ai-worker/main.go`
  - RLHF Feedback Store: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/ai-worker/correction_store.go`
  - gRPC Solver Service: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/ai-worker/grpc_server.go`
  - Bulk Import Engine: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/apps/ai-worker/import_worker.go`
- **Predictive Empathy Engine (v3):** Analyzes historical SKU-level purchase quantities and order frequency medians, applies retailer packaging constraints (Minimum Order Quantity `MOQ` and `StepSize`), applies RLHF retailer correction weights, and generates predictive preorder drafts.
- **Bulk Import Worker:** Consumes `INVENTORY_IMPORT_UPLOADED` from `inventory.import.events`, streams large Excel/CSV files from GCS, utilizes Gemini LLM (`packages/ai-bridge`) for zero-shot column mapping, validates inventory constraints, and inserts staged rows into Spanner.
- **Dispatch Solver:** Clarke-Wright savings heuristic with 2-opt trajectory local search optimization exposed via HTTP (`/v1/optimizer/solve`) and gRPC (`pegasus.optimizer.v1.OptimizerService` on `:8088`).

### 8.2 Production Optimizer Core (`services/optimizer-core`)
- **Protocol Buffer:** `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/services/optimizer-core/proto/optimizer_core.proto`
- **Rust Sidecar:** `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/services/optimizer-core/server-rust`
  - Built with Tonic/Prost gRPC listening on `:50055`.
  - Implements `CalculateRoute` (capacitated VRP with time windows) and `ResolveConstraint` (CP-SAT multi-facility factory assignment).
  - Uses `SCALE_FACTOR=10000` integer conversion to eliminate floating point imprecision across solver calculations.
- **Go Adapter Worker:** `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/services/optimizer-core/adapters/go/cmd/optimizer-worker/main.go`
  - Tunnels jobs from `pegasus-optimizer-jobs` Kafka topic to the Rust gRPC sidecar, applies bounded retry with exponential jitter, and writes results transactionally to `OptimizationJobs` and `OutboxEvents` in Spanner.

### 8.3 Deep Agents (`services/deep-agents`)
- **Source:** `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/services/deep-agents/src/void_deep_agents`
- **Tooling:** LangChain / LangGraph workspace with agent fleets (`fleet.py`, `factory.py`, `subagents.py`) for automated codebase auditing (`void-ecosystem-audit`) and autonomous task orchestration.

---

## 9. Client, Desktop & Native Mobile Ecosystem

Pegasus provides full cross-platform parity across Web, Desktop, and Mobile:

### 9.1 Desktop & Web Portals (Next.js 15 + React 19 + Tauri 2 + HeroUI 3)
1. **Admin / Supplier Portal (`apps/admin-portal`):**
   - Live Bento grid telemetry dashboard, MapLibre GL fleet tracking map.
   - 4-step supplier onboarding wizard, billing and webhook configuration.
   - Picking manifest manager, route dispatch queue, dynamic pricing overrides.
2. **Factory Portal (`apps/factory-portal`):**
   - Factory production lane settings, outbound truck loading bays.
   - Inter-warehouse transfer acceptance, replenishment matrix dashboard.
3. **Warehouse Portal (`apps/warehouse-portal`):**
   - Inventory restock receiving, digital seal verification.
   - Live `/ws/warehouse` dispatch lock observer with auto-reconnection.
   - Active driver fleet queue assignment.
4. **Retailer Desktop POS (`apps/retailer-app-desktop`):**
   - Point-of-Sale keyboard navigation & barcode scanner integration.
   - Wholesale cart management, offline local caching, invoice PDF generation.

### 9.2 Mobile Native Android Fleet (Kotlin / Jetpack Compose / Hilt / KSP)
- `driver-app-android` — Offline-first driver execution, dead reckoning GPS tracker, QR scanner manifest handshake, cash collection ledger sync, M3 Jetpack Compose.
- `retailer-app-android` — Retailer preorder management, auto-order toggles, shop-closed alert response, push notifications.
- `factory-app-android` — Factory dock loading, pallet barcode scanner, manifest dispatch.
- `warehouse-app-android` — Warehouse floor stock counting, bay intake, driver offload verification.
- `payload-app-android` — Loading dock payload inspection, digital seal validation.

### 9.3 Mobile Native iOS Fleet (Swift 6 / SwiftUI / SwiftData / Apple HIG)
- `driverappios` — Native SwiftUI MapKit integration, CoreLocation background telemetry streamer, SwiftData offline cache, QR scanner.
- `retailer-app-ios` — SwiftUI HIG-compliant shopkeeper app, preorder confirmation gate, dispute submission.
- `factory-app-ios` — iPad/iPhone factory loading terminal, pallet verification.
- `warehouse-app-ios` — Warehouse floor management and inventory intake.
- `payload-app-ios` — Tablet-optimized bay scanner with SF Symbols and camera handshake.

### 9.4 Cross-Platform Mobile Terminal
- `payload-terminal` (`apps/payload-terminal`) — Expo 55 / React Native 0.83 terminal for warehouse ruggedized hand-held scanners.

---

## 10. Shared Packages (`packages/`)

1. **`packages/ai-bridge`:** Google Gemini AI adapter for schema mapping and document inference.
2. **`packages/api-client`:** TypeScript client with interceptors, auth token injection, and failover queueing.
3. **`packages/config`:** Fail-closed Go environment configuration parser with panic-on-missing validation.
4. **`packages/i18n`:** Type-safe localization engine for English, Russian, Uzbek (Latin/Cyrillic), Turkish, and Arabic.
5. **`packages/optimizer-contract`:** Protocol definitions and canonical Go structs for VRP and CP-SAT solvers.
6. **`packages/types`:** Universal TypeScript models for entities, orders, fleet, treasury, and the 44KB `ws-events.ts` WebSocket protocol.
7. **`packages/ui-kit`:** Shared Material 3 design tokens and Tailwind CSS presets.
8. **`packages/validation`:** Runtime schema validators ensuring frontend-backend data compatibility.

---

## 11. Infrastructure, Deployment & Operations

### 11.1 Local Simulation Fleet (`docker-compose.yml`)
- **Apache Kafka 7.7.0 (KRaft mode):** Port 9092, no Zookeeper requirement.
- **Kafka UI:** Port 8081 for real-time topic monitoring.
- **Kafka Init:** Auto-provisions 8 core topics with production partition counts (128 for events/forecast).
- **Google Cloud Spanner Emulator:** Port 9010 (gRPC) & 9020 (REST).
- **Redis Alpine (Memorystore emulator):** Port 6379.
- **Firebase Auth Emulator:** Port 9099 & UI on 4000.
- **Global Pay WireMock Mock:** Port 8085.
- **Optimizer Core Simulation Profile (`--profile optimizer-sim`):** Runs Rust sidecar on port 50055 and Go adapter worker with Prometheus metrics on 8082.

### 11.2 Kubernetes Production Topology (`infra/k8s/`)
- Namespaces: Dedicated `pegasus` namespace (`infra/k8s/namespace.yaml`).
- Backend Deployment: 3-replica default with PodDisruptionBudget (`minAvailable: 2`) and HPA scaling from 3 to 50 pods based on 70% CPU and 80% Memory targets (`infra/k8s/backend/`).
- AI Worker Deployment: Co-located with Rust optimizer sidecar on localhost loopback (`127.0.0.1:50055`). Scaled via KEDA Kafka lag trigger (`infra/k8s/ai-worker/keda-scaledobject.yaml`).
- Monitoring: Prometheus ServiceMonitors and AlertingRules for solver latency, circuit breakers, and consumer lag (`infra/k8s/monitoring/`).

### 11.3 Google Cloud Terraform Topology (`infra/terraform/`)
- Single-Region Base (`main.tf`): Asia-South1 VPC, Cloud Memorystore Redis 7.0 Standard HA, Cloud Spanner Regional instance, Cloud Run Go gateway, GKE private cluster.
- Multi-Region Scale-Out (`multiregion.tf`): Spanner `nam-eur-asia3` (3 continents, 9 nodes total), regional GKE clusters across `asia-south1`, `europe-west1`, and `us-central1`.

### 11.4 Developer Tooling & Verification (`Makefile`, `scripts/`)
- `make env-up` / `make env-down` — Boots and stops local emulator fleet.
- `make spanner-init` — Provisions Spanner database and executes 94 DDL statements.
- `make seed` — Seeds deterministic test data across Spanner, Redis, and Kafka.
- `make sprint1-gate` — Executes the automated architecture, security, and contract guard suite.
- `npm run test:e2e` — Runs full Playwright integration suites across all 4 web portals and mobile APIs.

---

## 12. Recommendations for `pegasus/agents.md`

Based on the actual codebase investigation, `pegasus/agents.md` must be constructed with the following authoritative structure:

1. **System Mission & Architectural Boundaries:**
   - Define Pegasus as the core multi-tenant logistics and commerce execution engine.
   - Mandate adherence to the 5 core personas: Supplier, Warehouse, Factory, Retailer, and Driver.
2. **Mandatory Honesty Rules ("Zero Theatre"):**
   - No fabricated API endpoints or hallucinated database columns. Every proposed route must connect to an existing router in `apps/backend-go/` or be explicitly implemented.
   - Spanner schema changes must be accompanied by idempotent DDL statements matching `schema/spanner.ddl`.
   - Never bypass the Transactional Outbox pattern when emitting Kafka events.
3. **Geospatial & Calculation Conventions:**
   - Strict ban on raw Cartesian or unindexed Haversine loops in critical paths. All spatial indexing must use Uber H3 resolution-7 cells (`H3Cell` / `H3Index`).
   - Monetary values must always be represented as 64-bit integers in minor currency units (e.g. tiyin/cents). No floating-point math for money.
   - Solver coordinates and distances must use `SCALE_FACTOR=10000` integer conversion.
4. **Multi-Tenancy & Data Isolation:**
   - Every Spanner query touching business entities must filter strictly by `SupplierId` and, where applicable, `WarehouseId`.
   - RBAC roles (`GLOBAL_ADMIN`, `NODE_ADMIN`, `FACTORY_ADMIN`, `FACTORY_PAYLOADER`, `DRIVER`, `RETAILER`) must be enforced at the router middleware layer.
5. **Development & Verification Workflows:**
   - Document the mandatory local emulator boot command sequence (`make env-up`, `make spanner-init`, `make seed`, `make run-backend`).
   - Require running `make sprint1-gate` and `npm run guard:one-eye` prior to opening pull requests.
   - Document how to test mobile and desktop clients using LAN IP configuration via `make dev-ip` and `make dev-devices`.
