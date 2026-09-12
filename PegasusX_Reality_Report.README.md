# PegasusX Reality Report & Document Source-of-Truth (SoT) Index

**Status:** Authoritative Markdown SoT Map & Historical Export Status Guide  
**Date:** 2026-08-20  
**Living Codebase Root:** `pegasusX/`

---

## 1. Important Notice: Frozen Word Exports (.docx) vs Living Markdown

All `.docx` files in the repository (e.g. historical *End-Product Reality Reports* and *Docs↔Code Alignment Status* exports) are **frozen historical artifacts**. They have been converted to Markdown (`.md`) format and archived under `archive/docx/`.

**Do NOT plan or execute from `.docx` files.** Always reference the living Markdown specifications listed below.

---

## 2. Authoritative Living Documentation Hierarchy

| Level | Document | Path | Purpose / Description |
|---|---|---|---|
| **North Star** | **Final Goal & Destination** | [`.agents/memory/GOAL.md`](.agents/memory/GOAL.md) | Universal destination loaded on every session. Directs to global multi-supplier and local-first ecosystem programs. |
| **Index** | **Documentation Map** | [`pegasusX/docs/DOCS_SOURCE_OF_TRUTH.md`](pegasusX/docs/DOCS_SOURCE_OF_TRUTH.md) | Governing index classifying living vs frozen documentation across the monorepo. |
| **Master Architecture** | **Full System Parity & Ecosystem Master Plan** | [`pegasusX/docs/FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md`](pegasusX/docs/FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md) | Comprehensive master blueprint covering 29 route packages, multi-tenancy, outbox events, and 6 role-row client platforms. |
| **Enterprise Program** | **Global Scale Program** | [`pegasusX/docs/GLOBAL_SCALE_PROGRAM.md`](pegasusX/docs/GLOBAL_SCALE_PROGRAM.md) | Multi-supplier tenant onboarding, market packs, regional cell architecture (GS-A/T/M/C/I/R/P). |
| **Local Ecosystem** | **Global Scale Local Ecosystem** | [`pegasusX/docs/GLOBAL_SCALE_LOCAL_ECOSYSTEM.md`](pegasusX/docs/GLOBAL_SCALE_LOCAL_ECOSYSTEM.md) | Local-first warehouse matching, H3 geospatial clustering, pack-owned payment gateway catalog (GS-L/K). |
| **Client UI Program** | **Global Scale Client UI** | [`pegasusX/docs/GLOBAL_SCALE_CLIENT_UI.md`](pegasusX/docs/GLOBAL_SCALE_CLIENT_UI.md) | Client visualization program across all role surfaces (GS-U0–U9). |
| **Role Parity** | **Role-Row Parity Matrix** | [`pegasusX/docs/ROLE_ROW_PARITY_MATRIX.md`](pegasusX/docs/ROLE_ROW_PARITY_MATRIX.md) | Concrete code-verified implementation status across all 6 role rows (Supplier, Retailer, Driver, Warehouse, Factory, Payload) + Admin. |
| **Features vs Code** | **Role Features Docs vs Code** | [`pegasusX/docs/ROLE_FEATURES_DOCS_VS_CODE.md`](pegasusX/docs/ROLE_FEATURES_DOCS_VS_CODE.md) | Route-by-route audit with exact `file:line` citations and phased implementation plans. |
| **Living Scorecard** | **Master Scorecard & Execution Program** | [`pegasusX/docs/session-2026-08-13/SCORECARD.md`](pegasusX/docs/session-2026-08-13/SCORECARD.md)<br/>[`pegasusX/docs/session-2026-08-13/MASTER_10_10_EXECUTION_PROGRAM.md`](pegasusX/docs/session-2026-08-13/MASTER_10_10_EXECUTION_PROGRAM.md) | Operational layer scores and 10/10 execution tracks. |
| **Residuals** | **Residual Register & Sequence** | [`pegasusX/docs/session-2026-08-13/RESIDUAL_REGISTER.md`](pegasusX/docs/session-2026-08-13/RESIDUAL_REGISTER.md)<br/>[`pegasusX/docs/PROD_READINESS_SEQUENCE.md`](pegasusX/docs/PROD_READINESS_SEQUENCE.md) | Concrete list of external deploy-time credentials (Layer B) vs completed code (Layer A). |

---

## 3. Codebase Reality & Verified Facts

The following findings represent the genuine, verified state of the `pegasusX` codebase:

1. **Backend Go Core & Routes**:
   - Organized in `pegasusX/apps/backend-go/` with **29 mounted route packages** (`main.go`).
   - Clean separation of concerns between HTTP routes, domain business logic, and lifecycle bootstrap.
   - Comprehensive test suite with 80+ packages passing unit and integration tests; `cmd/ssmr-smokecheck` executes 80+ automated multi-role checks.

2. **Cloud Spanner Multi-Tenancy & Outbox**:
   - Authoritative DDL at `pegasusX/apps/backend-go/schema/spanner.ddl` (3,648 lines).
   - Enforces `SupplierId STRING(36) NOT NULL` on all core entities with per-tenant isolation.
   - `ParentOrders` table (`spanner.ddl:221-239`) enables multi-supplier cart checkouts.
   - Transactional Outbox pattern backed by `OutboxEvents` and `OutboxDeadLetters` with Kafka relay.

3. **Multi-Platform Client Applications (6 Role Rows + Admin)**:
   - Complete, compilable, and tested implementations across all 6 roles (Supplier, Retailer, Driver, Warehouse, Factory, Payload) plus Platform Admin.
   - Full test coverage passing in Vitest (`@pegasusx/supplier-portal`, `@pegasusx/retailer-app-desktop`, `@pegasusx/warehouse-portal`, `@pegasusx/factory-portal`, `@pegasusx/admin-portal`, `payload-terminal`).
   - Local persistence and offline mutation queues wired via SQLite (`desktop-cache` for Tauri), Room (`AppDatabase` / `PegasusDriverDatabase` / `PayloadDatabase` for Android), and SwiftData (`OfflineDeliveryStore` / `PendingPosStore` for iOS).

4. **Honest Product Boundaries (RFC 7807 HTTP 410s)**:
   - `GET /v1/supplier/inventory/audit` returns HTTP 410 `audit_unwired` (no fake audit ledger).
   - `POST /v1/delivery/negotiate` returns HTTP 410 `feature_disabled` unless `QUANTITY_NEGOTIATION_ENABLED=true`.
   - `POST /v1/retailer/card*` returns HTTP 410 `saved_cards_not_product`.
   - `GET /v1/payloader/capacity` returns HTTP 410 `capacity_unwired` (VU computed directly from manifest items).

5. **Layer A (Code Complete) vs Layer B (Deploy-Time Cloud Secrets)**:
   - **Layer A**: All business logic, DDL, route handlers, WebSocket event brokers, and UI screens are 100% complete and tested.
   - **Layer B**: External third-party credentials (live Soliq OFD PKCS#12 keys, APNs/FCM production keys, live Global Pay credentials) are deferred to operational deployment environments.
