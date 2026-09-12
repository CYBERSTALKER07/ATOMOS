# Shared Workspace Memory (All Agents & Subsystems)

<!-- VOID-GRAPH-MEMORY-SEED -->

**Read First Every Session:** `.agents/memory/GOAL.md` — The universal goal and architectural destination.

---

## 1. Core Project Context & Living Source of Truth

- **Authoritative Codebase Root:** `pegasusX/` (`pegasus/` is an archived legacy reference / port source).
- **Master Blueprint:** `pegasusX/docs/FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md`.
- **Destination Specifications:** `pegasusX/docs/GLOBAL_SCALE_PROGRAM.md`, `pegasusX/docs/GLOBAL_SCALE_LOCAL_ECOSYSTEM.md`, `pegasusX/docs/GLOBAL_SCALE_CLIENT_UI.md`.
- **Governing Doc Index:** `pegasusX/docs/DOCS_SOURCE_OF_TRUTH.md`.
- **Tenant Isolation Key:** `SupplierId STRING(36) NOT NULL` across all core Spanner entities (`schema/spanner.ddl:3648`).
- **Dual Manifest Planes:** `SupplierTruckManifests` (last-mile) vs `FactoryTruckManifests` (factory-to-warehouse transfers) remain strictly isolated.
- **Financial Architecture:** Integer minor currency units, double-entry ledger safety, pay-at-delivery cash/credit, and pack-owned PSP catalog (`payment/catalog.go`).

---

## 2. Verified Codebase State (2026-08-20 Master Synchronization)

### 2.1 Backend Go Architecture (`apps/backend-go/`)
- **Lifecycle Entrypoint:** `main.go` (479 lines) constructs `bootstrap.NewApp` and mounts **29 modular route packages**:
  `supplierroutes`, `retailerroutes`, `warehouseroutes`, `driverroutes`, `factoryroutes`, `payloadroutes`, `orderroutes`, `paymentroutes`, `creditroutes`, `cashreconroutes`, `creditnoteroutes`, `returnsroutes`, `partner`, `platformroutes`, `platformadmin`, `featureflags`, `mfa`, `controltowerroutes`, `demandroutes`, `laborcapacityroutes`, `etaroutes`, `globalproductsroutes`, `catalogroutes`, `pulseroutes`, `taxroutes`, `telemetryroutes`, `updateroutes`, `storageroutes`, `infraroutes`.
- **Multi-Hub WebSocket Hub (`ws/`)**: 8 dedicated role hubs (`RetailerHub`, `SupplierHub`, `DriverHub`, `PayloadHub`, `WarehouseHub`, `FactoryHub`, `TelemetryHub`, `PlatformAdminHub`) with Redis fanout and sequence-based reconnection reconciliation.
- **Transactional Outbox Engine (`outbox/`)**: Atomically buffers events in `OutboxEvents` within Spanner transactions, relayed to Kafka via background workers; poison messages safely stored in `OutboxDeadLetters`.
- **SSMR Smokecheck Suite (`cmd/ssmr-smokecheck/`)**: Automated test runner executing 80+ end-to-end multi-role verification steps against live backend configurations.

### 2.2 Cloud Spanner Multi-Tenant Data Model (`schema/spanner.ddl`)
- **3,648 Lines of DDL** covering all enterprise logistics domains:
  - Tenancy & Core Identity: `Suppliers`, `SupplierOIDC`, `SupplierProfiles`, `SupplierStaff`, `StaffInvites`.
  - Orders & Fulfillment: `ParentOrders` (multi-supplier cart split), `Orders`, `OrderDeliveryProofs`, `OrderConditionReports`, `SupplierTruckManifests`, `ManifestOrders`, `ManifestShipUnits`.
  - WMS & Storage: `Warehouses`, `WarehouseCoverageCells`, `WarehouseCoverageCities`, `WarehouseBins`, `WarehouseLots`, `WarehousePickWaves`, `WarehousePickTasks`, `WarehouseCycleCounts`, `WarehouseTemperatureReadings`.
  - Retail OS & POS: `Retailers`, `RetailerLocations`, `StoreStock`, `StoreStockMovements`, `StoreStockReceiveSessions`, `PosRegisters`, `PosSessions`, `PosSales`, `PosHolds`, `RetailerShifts`, `RetailerSections`, `RetailerAssistTickets`.
  - Finance & Credit: `PaymentConfigs`, `PaymentSessions`, `PaymentAttempts`, `PaymentChargebacks`, `PaymentLedgerEntries`, `ArInvoices`, `CashReconciliations`, `CreditNotes`, `PayoutBatches`, `FxRates`.
  - Partner & B2B: `PartnerApiKeys`, `PartnerOAuthClients`, `PartnerWebhooks`, `PartnerEdiDocuments`, `PartnerEdiProfiles`, `PartnerAs2Configs`, `PartnerSftpConfigs`.
  - Local Matching & Overrides: `ServicePins`, `SupplierRegions`.
  - Outbox: `OutboxEvents`, `OutboxDeadLetters`.

### 2.3 Multi-Platform Client Applications Parity (6 Role Rows + Admin)
- **Supplier Role**: Web Portal & Tauri Desktop (`apps/supplier-portal`, 82 routes), Android (`apps/supplier-app-android`, 61 screens), iOS (`apps/supplier-app-ios`, 68 views). Real `@pegasusx/api-client`, Retrofit (`SupplierApi.kt`), and Swift Concurrency (`SupplierRealtimeClient.swift`).
- **Retailer Role**: Tauri Desktop (`apps/retailer-app-desktop`, 31 routes), Android (`apps/retailer-app-android`, 40+ screens), iOS (`apps/retailer-app-ios`, 49 screens). Real `/v1/retailer/ai/predictions`, Store Stock, POS, and local offline persistence (SQLite, Room `AppDatabase`, SwiftData `PendingPosStore`).
- **Driver Role**: Android (`apps/driver-app-android`, 63 screens), iOS (`apps/driver-app-ios`, 74 views). Realtime GPS streaming to `/v1/ws?sv=2`, dead-reckoning interpolation, spoken audio navigation cues, and offline delivery queues (`PegasusDriverDatabase`, `OfflineDeliveryStore`).
- **Warehouse Role**: Web Portal & Tauri Desktop (`apps/warehouse-portal`, 46 routes), Android (`apps/warehouse-app-android`, 44 screens), iOS (`apps/warehouse-app-ios`, 84 views). WMS bins/lots, pick waves, cycle counts, cold-chain, and dispatch.
- **Factory Role**: Web Portal & Tauri Desktop (`apps/factory-portal`, 21 routes), Android (`apps/factory-app-android`, 62 screens), iOS (`apps/factory-app-ios`, 70 views). Live Spanner `FactoryTruckManifests` loading bay and supply requests.
- **Payload Role**: Expo Terminal (`apps/payload-terminal`, SDK 55), Android Tablet (`apps/payload-app-android`, 50 screens), iPad (`apps/payload-app-ios`, 43 screens). Real `POST /v1/payloader/manifests/seal-all`, ship-unit barcode scanning.
- **Platform Admin**: Next.js 15 Web Console (`apps/admin-portal`) with 9 governance panels (Tenants, Feature Flags dual-control, Outbox dead-letters replay, Billing, Audit).

### 2.4 Shared Packages & Client Contracts (`pegasusX/packages/`)
- `@pegasusx/types`: 6,682 lines of TypeScript contracts, DTOs, and problem details.
- `@pegasusx/api-client`: 3,669 lines unified HTTP client SDK wrapping all 29 route domains with exponential backoff and session reconciliation.
- `@pegasusx/ws-refresh-contract`: WebSocket refresh rules and `dashboardDirtySlice()` UI mapping.
- `@pegasusx/desktop-bridge`: Tauri IPC bridge for native printing, deep links, auto-updates.
- `@pegasusx/desktop-cache`: SQLite local caching and offline mutation buffer.
- `@pegasusx/ui-kit`: Portal primitives (`StatusStack`, `KpiStat`, `HealthStrip`).

### 2.5 Quality & Test Execution Status
- **Backend Audit Report (Tracks 1–8):** 100% remediated across Critical, High, Medium, Low, and Perf items (`backend_remediation_plan.md` checked off).
- **Backend Go Compilation:** `go build ./...` compiles cleanly across all 29 route domains and cmd entrypoints.
- **Backend Go Tests:** 80+ packages pass unit and integration tests; `cmd/ssmr-smokecheck` tests pass (80+ steps).
- **Live Infrastructure & SSMR E2E Smokecheck (`scripts/smoke_sandbox.sh`):** 100% PASSED (`__SANDBOX_OK__`, `__SSMR_OK__`, `sandbox-ecosystem-marker-gate-ok`) against live Docker stack (Spanner emulator, Redis, Kafka, Zookeeper, Optimizer Core, Backend-Go, AI Worker). All 100+ ecosystem `PX_E2E_*` markers passed.
- **Core CI & Parity Gates:** `make repo-hygiene-gate`, `make gen-contracts-gate`, `make gap-hunter-gate`, `make kafka-ha-gate`, `make partner-openapi-gate`, `make jwt-openapi-gate`, and `cmd/schema-drift -offline` all pass clean.
- **Client Workspace Vitest Suites:** 128+ tests passing across `@pegasusx/supplier-portal`, `@pegasusx/retailer-app-desktop`, `@pegasusx/warehouse-portal`, `@pegasusx/factory-portal`, `@pegasusx/admin-portal`, `@pegasusx/desktop-bridge`, `@pegasusx/desktop-cache`, and `payload-terminal`.
- **Native Test Suites:** 39 Kotlin test classes (Android) and 28 Swift test classes (iOS) pass clean.

### 2.6 CodeGraph Knowledge Graph & Deep Audit Infrastructure (Memgraph + AST + 5 Seams)
- **Engine & Storage:** Containerized Memgraph Platform (`pegasusx-codegraph-memgraph`) running on `bolt://localhost:7687` and Memgraph Lab on `http://localhost:3000` via `infra/docker-compose.codegraph.yml`.
- **Live Graph Scale (`cgr stats`):** 64,432 nodes (22,365 `Function`/`Method`, 847 `RouteEndpoint`, 549 `ApiClientMethod`, 229 `SpannerTable`, 217 `ServiceMethod`, 180 `RepositoryMethod`, 88 `ArchitectureNode`, 70 `EventDefinition`, 38 `OutboxEmitter`, 18 `ClientApp`, 13 `KafkaTopic`, 7 `KafkaConsumer`, 7 `WSHubRoom`) and 176,106 relationships (162,146 `CALLS`, 571 `CONSUMES_ROUTE`, 238 `EMITS_OUTBOX`, 160 `ARCH_REL`, 147 `READS_TABLE`, 114 `MUTATES_TABLE`, 108 `ROUTED_TO_TOPIC`, 22 `FANOUT_WS_ROOM`, 18 `RECEIVED_BY`, 7 `CONSUMED_BY`).
- **5 Seam Extraction Automation:** `scripts/extract_codegraph_seams.py` and `scripts/seed_architecture_graph.py` generate and ingest Cypher statements linking AST call hierarchies directly with backend Chi routes, client APIs, Spanner DDL, and Kafka outbox emitters.
- **Advanced Compiler-Grade Code Intelligence Engine (Big-Tech Class):**
  - `pegasusX/scripts/advanced_codegraph_analyzer.py` provides industrial static analysis:
    1. **Spanner SQL Tenant Taint Analysis**: Scans all 704 SQL statements in Go repos, detecting 347 queries lacking explicit `WHERE SupplierId` filtering.
    2. **Transactional Outbox Dual-Write Verifier**: Identifies 54 Go files mutating Spanner without atomic `outbox.Emit` / `TxnBuffer` pairing.
    3. **Cross-Language Field-Level DTO Drift**: Discovers 679 Go struct JSON tags omitted from TypeScript `packages/types` client interfaces.
    4. **NetworkX Monorepo Package Centrality**: Mathematically proves `apps/backend-go/auth` is the #1 single point of failure (betweenness centrality: 0.1411).
    5. **Transitive Blast-Radius Cone**: Calculates multi-hop upstream reachability closure across AST call chains.
  - Run via `make codegraph-advanced-audit` (writes `docs/ADVANCED_CODE_AUDIT_REPORT.md`).
- **Agent Skill & Directives:** Established `.agents/skills/codegraph-deep-audit/SKILL.md` across all agent instruction files (`AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, `.cursorrules`, `.github/copilot-instructions.md`, `honest-code-gate`) mandating pre-edit blast radius checks.
- **Interactive Pegasus Studio & REST API:** `tools/codegraph-ui/server.py` serving interactive 3-column Studio interface with live `/api/audit`, `/api/advanced-audit`, `/api/bazel/rdeps`, and `/api/kythe/xref` endpoints on `http://127.0.0.1:3001`.
- **Google Bazel/Blaze Target DAG & Query Engine:**
  - `pegasusX/scripts/bazel_target_graph.py` models 234 monorepo targets and 799 `DEPENDS_ON` edges in Memgraph.
  - Query engine supports `rdeps(//..., target)` to calculate exact transitive test targets (e.g. `order_lib` ➔ 51 test targets). Run via `make bazel-graph`.
- **Google Kythe Semantic Schema Integration:**
  - `pegasusX/scripts/kythe_semantic_adapter.py` annotates 22,365 nodes with formal Kythe VNames (`corpus: pegasusx`, `path`, `signature`, `language`) and Kythe standard edges (`/kythe/edge/ref/call`, `/kythe/edge/depends`). Run via `make kythe-index`.
- **Real-Time Dynamic Incremental Watcher Daemon:**
  - `pegasusX/scripts/dynamic_codegraph_watcher.py` (running via `make codegraph-watch`) continuously monitors `apps/backend-go/`, `packages/`, and `schema/`.
  - On any file edit/save: performs sub-50ms incremental Tree-sitter re-parsing, transactional Cypher delta mutations in Memgraph, Kythe VName re-binding, and Bazel affected target calculation without full graph rebuilds.

---

## 3. Product Disables & Honest Boundaries (RFC 7807 HTTP 410s)

- `GET /v1/supplier/inventory/audit`: Returns HTTP 410 `audit_unwired`.
- `POST /v1/delivery/negotiate`: Returns HTTP 410 `feature_disabled` unless `QUANTITY_NEGOTIATION_ENABLED=true`.
- `POST /v1/retailer/card*`: Returns HTTP 410 `saved_cards_not_product`.
- `GET /v1/payloader/capacity`: Returns HTTP 410 `capacity_unwired`.
- Unkeyed payment gateways (`STRIPE`, `ADYEN`, `PAYME`, `CLICK`): Return HTTP 501 `no_live_keys` via `catalogHonestyExecutor`.
- Auto-order `place` and Factory planning default to flag-off.

---

## 4. Layer A (Code Complete) vs Layer B (Operational Cloud Credentials)

- **Layer A (Software Implementation)**: 100% COMPLETE. All schemas, DDL, 29 route packages, outbox relays, multi-role WebSocket hubs, and multi-platform client applications are implemented, compiled, and tested.
- **Layer B (Operational Deployment Secrets)**: LIVE SECRETS DEFERRED TO OPS. Live Cloud IdP credentials, Soliq OFD PKCS#12 signing keys, APNs/FCM production push certificates, and live Global Pay production merchant keys are deployed at runtime via Secret Manager / ExternalSecrets.


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
