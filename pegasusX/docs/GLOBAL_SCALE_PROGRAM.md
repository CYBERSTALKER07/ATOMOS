# PegasusX — Global-Scale Enterprise Program

**Final Goal (2026-08-20):** This file + [`GLOBAL_SCALE_LOCAL_ECOSYSTEM.md`](./GLOBAL_SCALE_LOCAL_ECOSYSTEM.md) + [`FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md`](./FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md) define the canonical enterprise destination. Loaded on every session: [`.agents/memory/GOAL.md`](../../.agents/memory/GOAL.md).

**Date:** 2026-08-20 (Synchronized Master State)  
**Authoritative Tree:** `pegasusX/`  
**Master Blueprint:** [`FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md`](./FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md)  
**Route & Client Inventory:** [`FEATURES_BY_APP_ROLE.md`](./FEATURES_BY_APP_ROLE.md) + 29 mounted `*routes` packages  
**Role Parity Matrix:** [`ROLE_ROW_PARITY_MATRIX.md`](./ROLE_ROW_PARITY_MATRIX.md)  
**Tenancy Architecture:** Multi-Tenancy `SupplierId STRING(36)` across Cloud Spanner (`schema/spanner.ddl`)  
**Ops Residuals Register:** [`session-2026-08-13/RESIDUAL_REGISTER.md`](./session-2026-08-13/RESIDUAL_REGISTER.md)

**North Star:** A **global multi-supplier** logistics operating system: companies anywhere **register** (new `SupplierId`, zero seed fallback in production), land in a **home cell**, receive a versioned **market pack** that checkout and fiscal engines read, invite roles, and execute Class A operations (`order → stock → truck → cash/credit → fiscal → payout`). Retailers trade with **multiple** suppliers; mixed checkout carts split per supplier into isolated child orders via `ParentOrders`. Same codebase, cloned cells — not a country-specific fork.

**Enterprise Backend & Infra Plan:** [`GLOBAL_SCALE_BACKEND_INFRA.md`](./GLOBAL_SCALE_BACKEND_INFRA.md)  
**Classified Backend Feature Inventory:** [`GLOBAL_SCALE_BACKEND_FEATURES.md`](./GLOBAL_SCALE_BACKEND_FEATURES.md)  
**Local-First Ecosystem & Pack PSPs (GS-L + GS-K):** [`GLOBAL_SCALE_LOCAL_ECOSYSTEM.md`](./GLOBAL_SCALE_LOCAL_ECOSYSTEM.md) — W1–W26 items closed, `proximity.ResolveServingWarehouse` + `CoverageEngine` active, `ServicePins` and `SupplierRegions` in Spanner, pack PSP catalog (`payment/catalog.go`), honest placeholder executors (`catalogHonestyExecutor`).  
**Client Visualization Program (GS-U):** [`GLOBAL_SCALE_CLIENT_UI.md`](./GLOBAL_SCALE_CLIENT_UI.md) — U0–U9 shipped (command dashboards, `StatusStack` kit, primary nav ≤5, ETag cache freshness, Plan & Brain tabs, dead-letter count `COUNT(*)`).

**Non-Negotiable Architecture Invariants:**
1. Do not merge factory and last-mile manifest planes (`FactoryTruckManifests` vs `SupplierTruckManifests` stay distinct).
2. Do not invent a second tenant key (all multi-tenancy anchors on `SupplierId`).
3. Maintain honest RFC 7807 HTTP 410 boundaries for gated/unwired endpoints (`audit_unwired`, `feature_disabled`).
4. Keep integer minor currency units, fiscal hard-gate, and transactional outbox state atomicity.

---

## 0. What “Global Enterprise” Means in Code

| Layer | Implementation State (Codebase Reality) | Global Target Standard |
|---|---|---|
| **Multi-Tenancy** | `SupplierId STRING(36) NOT NULL` on all core tables; request-scoped JWT extraction; `auth.RequireTenant()` middleware; fail-closed in staging/production. | Many companies register; each completely isolated by `SupplierId` + home `CellId` + `MarketPack`. |
| **Multi-Supplier Carts** | `ParentOrders` table (`spanner.ddl:221-239`); atomic split into child `Orders` within same market pack. | Retailer shops multiple suppliers; child orders remain strictly supplier-scoped. |
| **Market Packs** | `MarketPack` catalog in code (`auth/market_pack.go`); M1–M7 read pack for currency, fiscal, radius, timezone, payout rail, and tax country. | Shipped versioned `MarketPack` governing checkout, fiscalization, SMS, and maps. |
| **Identity & Access** | Per-role password/Firebase login; Platform Admin TOTP MFA with step-up enforcement; GS-I per-supplier OIDC attach & exchange (`orgoidc` package). | Optional per-tenant OIDC/SAML integration; staff JWT unchanged. |
| **Financial Integrity** | Integer minor units; double-entry ledger compatibility; Global Pay + cash + credit; MY_SOLIQ fiscal contract; bank-file payout settlement. | Market-specific fiscal adapter per pack (MY_SOLIQ / PEPPOL / Commercial) and pack-filtered PSPs. |
| **Cloud & Cells** | GKE + Cloud Spanner single-cell configuration (`cell-uz`); regional cell scaffolds (`cell-eu`, `cell-us`) defined in Terraform. | Regional cloned cells sharing identical DDL and container images. |
| **Client Parity** | 6 complete role rows (Supplier, Retailer, Driver, Warehouse, Factory, Payload) + Admin across Web, Desktop (Tauri), Android (Compose), iOS (SwiftUI). | Identical route contracts, pack-driven currency formatting, and offline queueing. |

---

## 1. Role Feature Inventory & Global Classification

### 1.1 Shared Platform & Security
- **JWT HS256 & Session**: Stamps `market_code` and `home_cell` on token issue (`auth/jwt.go`). `GET /v1/auth/session` returns user identity and resolved `MarketPack`.
- **Tenant Context**: Enforces `SupplierId` scope on all domain handlers. Seed fallback strictly disabled in sandbox, SSMR, and production environments (`auth/tenant.go`).
- **MFA Step-Up**: TOTP enrollment, verification, and step-up middleware protecting administrative endpoints (`mfa/handlers.go`).
- **Feature Flags Dual-Control**: Proposal and approval lifecycle with audit logging (`featureflags/handlers.go`).

### 1.2 Retailer Role (Desktop + Android + iOS)
- **Multi-Org Authentication**: Dynamic supplier attachment and invite token validation.
- **Catalog, Cart & Checkout**: Minor unit integer pricing derived from pack currency; unified checkout creates order with pay-at-delivery cash/credit.
- **Retail OS & Store Stock**: Live store inventory tracking, barcode receiving sessions, cycle counts, POS sales, shifts, sections, and assist tickets (`retailerroutes/routes.go`).
- **AI Demand Predictions**: Real `/v1/retailer/ai/predictions` integration.

### 1.3 Supplier Role (`ADMIN` + Portal + Desktop + Android + iOS)
- **Control Tower & Command**: Live operational metrics, order status funnel (`StatusStack`), and scored exceptions.
- **Catalog & Pricing**: Multi-tier pricing rules, volume discounts, and customer-specific price overrides.
- **Dispatch & Routing**: H3 geospatial clustering (Res 7), fleet capacity fitting, and optional OR-Tools sidecar integration.
- **S&OP & Planning**: Forecast accuracy tracking, safety stock replay, and Digital Brain tabs (`/planning?tab=brain`).
- **Entity Resolution & CRM**: Master data deduplication and customer relationship history.

### 1.4 Warehouse Role (Portal + Desktop + Android + iOS)
- **WMS Core**: Bin locations, lot tracking with expiry dates, pick wave generation, and cycle counts.
- **Cold Chain & Telemetry**: Temperature sensor logging and excursion alerts.
- **Dispatch Execution**: Order assignment, freeze-lock manual override, and manifest staging.
- **Supply Requests & QC**: Inbound factory transfer receiving and quality inspection.

### 1.5 Factory Role (Portal + Desktop + Android + iOS)
- **Loading Bay & Seals**: Live Spanner `FactoryTruckManifests` management with start-loading and seal transitions.
- **Supply Request Fulfillment**: Stock allocation and transfer creation for connected warehouses.
- **Payload Overrides**: Loading bay exceptions and variance approvals.

### 1.6 Payload Role (Expo Terminal + Android Tablet + iPad)
- **Manifest Board & Scanning**: Realtime ship-unit scanning, camera barcode reader, and manifest checklist.
- **Seal-All Execution**: `POST /v1/payloader/manifests/seal-all` fully wired and tested across all 3 client apps.
- **Variance Management**: Volume utilization calculation and load variance approvals.

### 1.7 Driver Role (Android + iOS)
- **Mission Execution**: Doorstep arrival, QR verification, cash collection bag logging, and POD photo capture.
- **Telemetry Streaming**: Realtime background GPS streaming to `/v1/ws?sv=2` with dead-reckoning interpolation and spoken navigation cues.
- **Offline Delivery Store**: SwiftData / Room SQLite local queue for zero-connectivity delivery execution.

### 1.8 Platform Admin Role (Web Console)
- **Governance Panels**: Tenant onboarding, dual-control feature flags, outbox dead-letter replay (`/v1/admin/ops/dead-letters/replay`), billing invoices, and audit log inspection.

---

## 2. Technical Reshape Summary

| Architectural Area | Codebase Implementation State | Global Standard Pattern |
|---|---|---|
| **Currency Handling** | Integer minor units; pack `currency_code` and `decimal_places` derived via `auth.CoalesceCurrency`. | Shipped `MarketPack` owns currency; no hardcoded currency invention. |
| **Geospatial Indexing** | Uber H3 Resolution 7 indexing; `proximity.StampNodeGeography` stamps H3 and ISO country on writers. | Single `CoverageEngine` resolving warehouse catchment and factory supply lanes. |
| **Payment Routing** | `payment/catalog.go` filters gateways by pack; `catalogHonestyExecutor` returns HTTP 501 for unkeyed rails. | Pack-filtered gateway options with honest placeholder executors. |
| **Outbox & Messaging** | `OutboxEvents` table in Cloud Spanner with Kafka relay worker and `OutboxDeadLetters` recovery. | Transactional outbox pattern across all state-mutating domains. |
| **Service Probes** | `/healthz`, `/ready`, `/metrics`, and `/v1/health/capabilities` mounted in `infraroutes`. | Standard Kubernetes probes and Prometheus metrics export. |

---

## 3. Phased Modular Program (GS Progress)

```
GS-0  Honesty Stamp (Docs)           ← COMPLETED
GS-A  Auth & Session MarketPack      ← A0–A2 SHIPPED
GS-T  Self-Serve Tenant Register     ← T1–T5 SHIPPED
GS-M  MarketPack Reading (M1–M7)     ← M1–M7 SHIPPED (Flag stays false in SSMR without Soliq)
GS-C  Regional Cell Scaffold         ← C1–C5 SCAFFOLDED (Plan-only, apply deferred)
GS-I  Enterprise Identity (OIDC)     ← SHIPPED (SupplierOIDC + auth endpoints)
GS-R  Role UI Parity                 ← SHIPPED (Client pack currency and gateway bindings)
GS-U  Client Visualization (U0–U9)   ← U0–U9 SHIPPED (Command boards, StatusStack, ETag cache)
GS-P  Partner & B2B Dialects         ← SHIPPED (EDI INVOIC/PRICAT/REMADV, 1C adapters, AS2)
```

---

## 4. Verification Commands & Quality Evidence

```bash
# 1. Backend Go Test Suite
cd pegasusX/apps/backend-go
go test ./... -v

# 2. Automated SSMR Multi-Role Smoke Suite
cd pegasusX/apps/backend-go
go test -v ./cmd/ssmr-smokecheck/

# 3. Client Workspaces (Vitest)
pnpm --filter @pegasusx/supplier-portal test
pnpm --filter @pegasusx/retailer-app-desktop test
pnpm --filter @pegasusx/warehouse-portal test
pnpm --filter @pegasusx/factory-portal test
pnpm --filter @pegasusx/admin-portal test
pnpm --filter @pegasusx/desktop-bridge test
pnpm --filter @pegasusx/desktop-cache test
cd pegasusX/apps/payload-terminal && pnpm test
```
