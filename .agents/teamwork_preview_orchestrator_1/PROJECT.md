# Project: Repository Documentation Alignment & Codebase Synchronization

## Architecture
- Repository Root: `/Users/shakhzod/Desktop/V.O.I.D`
- Codebase SoT: `/Users/shakhzod/Desktop/V.O.I.D/pegasusX` (Go backend, Spanner DDL, TypeScript contracts & API client, 6 role-row client apps across Web/Android/iOS)
- Documentation Targets: 803 total files across 21 categories (including 8 `.docx` files to be converted to `.md` and archived).

## Feature Inventory & Codebase Reality
| # | Feature / Area | Codebase Reality (SoT) | Documented Claims & Alignment Actions | Milestone |
|---|---|---|---|---|
| 1 | **Docx Conversion & Archival** | 8 `.docx` files across root, `pegasusX/artifacts/`, `pegasusX/docs/`, and `session-2026-08-07/`. | Convert all 8 `.docx` files to Markdown (`.md`) format. Move original `.docx` files to `archive/docx/` so no `.docx` remains in active doc directories. | M1 |
| 2 | **Multi-Tenancy & Data Model** | `SupplierId STRING(36)` on all core Spanner tables (`schema/spanner.ddl:3648`), `ParentOrders` table (`spanner.ddl:221-239`), Outbox tables (`spanner.ddl:679-709`). | Sync all architecture and database docs to reflect true multi-tenancy and outbox patterns. | M2 |
| 3 | **Core Ecosystem & Master Plans** | Living monorepo in `pegasusX/`. Destination programs defined in `GLOBAL_SCALE_PROGRAM.md` and `GLOBAL_SCALE_LOCAL_ECOSYSTEM.md`. | Synchronize `DOCS_SOURCE_OF_TRUTH.md`, `GLOBAL_SCALE_PROGRAM.md`, `GLOBAL_SCALE_LOCAL_ECOSYSTEM.md`, `.agents/memory/WORKSPACE.md`. | M2 |
| 4 | **Client Applications Parity** | All 6 role rows (Supplier, Retailer, Driver, Warehouse, Factory, Payload) + Admin have complete screens, real API integration, zero mock data, and passing vitest suites. Local persistence via SQLite (Tauri), Room (Android), SwiftData (iOS). | Update `ROLE_ROW_PARITY_MATRIX.md` and role feature docs to accurately cite file:lines and confirm live client test passes. | M3 |
| 5 | **Scorecards & Residual Registers** | Gap ledger items G1-A1 through G7-4 are resolved. Stubs and deploy-time residuals (live cloud IdP, Soliq PKCS#12, APNs/FCM credentials) are explicit. | Synchronize `SCORECARD.md`, `RESIDUAL_REGISTER.md`, `GAP_LEDGER.md`, `MASTER_10_10_EXECUTION_PROGRAM.md`, `PROD_READINESS_SEQUENCE.md`. | M3 |
| 6 | **Backend Route Coverage** | 29 mounted `*routes` packages in `main.go:1-479`. RFC 7807 problem details, SessionAuth, MFA step-up. `cmd/ssmr-smokecheck` passes 80+ checks. | Align backend integration plans and route tables with all 29 route packages. | M4 |
| 7 | **Gated & Unwired Boundaries** | `inventory/audit` returns HTTP 410 `audit_unwired`. Quantity negotiation returns HTTP 410 `feature_disabled` unless env var enabled. Payme/Click webhooks commented out. | Accurately document these intentional 410 and deferred boundaries in backend and context docs. | M4 |
| 8 | **Context & Phase Documents** | `pegasusX/context/plan.md`, `parity-ledger.md`, `*_PHASE.md` need exact alignment with live codebase paths and statuses. | Synchronize context phase docs and parity ledgers. | M4 |
| 9 | **Multi-Agent Quality & Review** | Independent verification of updated documentation against live codebase implementation. | Verify sampled documents, ensure zero references to deprecated features and zero unverified claims. | M5 |

## Milestones
| # | Name | Scope | Dependencies | Status |
|---|---|---|---|---|
| 0 | Survey & SoT Audit | Full inventory of docs & codebase status (803 files cataloged, 3 explorers) | None | DONE |
| 1 | Docx Conversion & Active Archival | Convert 8 `.docx` files to `.md` and move `.docx` to `archive/docx/` | M0 | DONE |
| 2 | Core Ecosystem & Architecture Docs | `GLOBAL_SCALE_PROGRAM.md`, `GLOBAL_SCALE_LOCAL_ECOSYSTEM.md`, `DOCS_SOURCE_OF_TRUTH.md`, `FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md`, `.agents/memory/WORKSPACE.md`, root docs | M0 | DONE |
| 3 | Role Row Parity, Feature Matrix & Scorecards | `ROLE_ROW_PARITY_MATRIX.md`, `ROLE_FEATURES_DOCS_VS_CODE.md`, `session-2026-08-13/SCORECARD.md`, `RESIDUAL_REGISTER.md`, `GAP_LEDGER.md`, `MASTER_10_10_EXECUTION_PROGRAM.md`, `PROD_READINESS_SEQUENCE.md` | M0 | DONE |
| 4 | Backend, Data Plane & Context Docs | `BACKEND_PARITY_AND_ECOSYSTEM_INTEGRATION_PLAN.md`, `CLOUD_CREDENTIALS_CHECKLIST.md`, `pegasusX/context/plan.md`, `parity-ledger.md`, `*_PHASE.md` | M0 | DONE |
| 5 | Multi-Agent Review & Final Verification | Independent multi-agent review of all updated docs against codebase reality | M1, M2, M3, M4 | DONE |

## Exclusive Write Ownership
- **Worker 1 (Docx)**: Convert `.docx` to `.md` and archive `.docx` files to `archive/docx/`.
- **Worker 2 (Core Architecture)**: `pegasusX/docs/GLOBAL_SCALE_PROGRAM.md`, `pegasusX/docs/GLOBAL_SCALE_LOCAL_ECOSYSTEM.md`, `pegasusX/docs/DOCS_SOURCE_OF_TRUTH.md`, `pegasusX/docs/FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md`, `.agents/memory/WORKSPACE.md`, `.agents/memory/GOAL.md`, `README.md`, `PegasusX_Reality_Report.README.md`.
- **Worker 3 (Parity & Scorecards)**: `pegasusX/docs/ROLE_ROW_PARITY_MATRIX.md`, `pegasusX/docs/ROLE_FEATURES_DOCS_VS_CODE.md`, `pegasusX/docs/session-2026-08-13/SCORECARD.md`, `pegasusX/docs/session-2026-08-13/RESIDUAL_REGISTER.md`, `pegasusX/docs/session-2026-08-13/GAP_LEDGER.md`, `pegasusX/docs/session-2026-08-13/MASTER_10_10_EXECUTION_PROGRAM.md`, `pegasusX/docs/session-2026-08-13/PROD_READINESS_SEQUENCE.md`.
- **Worker 4 (Backend & Context)**: `pegasusX/docs/BACKEND_PARITY_AND_ECOSYSTEM_INTEGRATION_PLAN.md`, `pegasusX/docs/CLOUD_CREDENTIALS_CHECKLIST.md`, `pegasusX/context/plan.md`, `pegasusX/context/parity-ledger.md`, `pegasusX/context/*_PHASE.md`.

## Interface Contracts & Guidelines
- Every document update must be based on verified code state in `pegasusX/`.
- No unsubstantiated claims of "wired" or "cloud-ready" without explicit file:line citations and differentiation between Layer A (code complete) and Layer B (cloud secrets/infra).
- In-place edits must preserve the structure, depth, and clarity of the documents while correcting all obsolete references and aligning with the latest codebase.

