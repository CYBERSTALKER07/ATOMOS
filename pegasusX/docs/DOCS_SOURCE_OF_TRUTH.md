# Documentation Source of Truth (Living vs Frozen)

**Date:** 2026-08-20 (Master Repository Synchronization Pass)  
**Living Codebase Root:** `pegasusX/`  
**Governing Rule:** Prefer the living Markdown specifications below. Historical reports and exports provide valuable evidence but must not drive planning without code verification.

**Agent Honesty Standard:** Status and implementation claims come from **current code**, verified with exact `file:line` citations and test suite runs.

**Document Formats:** Active documentation is strictly **Markdown (`.md`)**. All historical `.docx` exports have been converted to `.md` and archived under `archive/docx/`.

---

## 1. Living Documentation Hierarchy (Authoritative Sources of Truth)

| Document | Path | Scope & Role |
|---|---|---|
| **Final Goal & Destination** | [`.agents/memory/GOAL.md`](../../.agents/memory/GOAL.md) | Universal destination loaded on every session. Directs to global multi-supplier and local-first ecosystem programs. |
| **Master Blueprint** | [`FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md`](./FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md) | Comprehensive master blueprint covering 29 route packages, multi-tenancy, outbox events, and 6 role-row client platforms. |
| **Global Scale Program** | [`GLOBAL_SCALE_PROGRAM.md`](./GLOBAL_SCALE_PROGRAM.md) | Destination program for multi-supplier tenant onboarding, market packs, regional cell architecture (GS-A/T/M/C/I/R/P). |
| **Local Ecosystem Program** | [`GLOBAL_SCALE_LOCAL_ECOSYSTEM.md`](./GLOBAL_SCALE_LOCAL_ECOSYSTEM.md) | Destination program for local-first warehouse matching, H3 geospatial clustering, pack-owned payment gateway catalog (GS-L/K). |
| **Client UI Program** | [`GLOBAL_SCALE_CLIENT_UI.md`](./GLOBAL_SCALE_CLIENT_UI.md) | Client visualization program across all role surfaces (GS-U0–U9). Dashboards + Plan & Brain tabs. |
| **Role-Row Parity Matrix** | [`ROLE_ROW_PARITY_MATRIX.md`](./ROLE_ROW_PARITY_MATRIX.md) | Concrete code-verified implementation status across all 6 role rows (Supplier, Retailer, Driver, Warehouse, Factory, Payload) + Admin. |
| **Role Features vs Code** | [`ROLE_FEATURES_DOCS_VS_CODE.md`](./ROLE_FEATURES_DOCS_VS_CODE.md) | Route-by-route audit with exact `file:line` citations and phased implementation plans (P0–P16). |
| **Backend Parity Plan** | [`BACKEND_PARITY_AND_ECOSYSTEM_INTEGRATION_PLAN.md`](./BACKEND_PARITY_AND_ECOSYSTEM_INTEGRATION_PLAN.md) | Backend route coverage and integration blueprint for all 29 route packages. |
| **Production Readiness Sequence**| [`PROD_READINESS_SEQUENCE.md`](./PROD_READINESS_SEQUENCE.md) | Ordered residuals R0–R6 and Layer B operational cloud deployment sequence. |
| **Class A Coverage Goal** | [`PROD_ECOSYSTEM_GOAL.md`](./PROD_ECOSYSTEM_GOAL.md) | Class A execution criteria (Spanner RW txn + outbox + Kafka + WebSocket + role client). |
| **Living Scorecard** | [`session-2026-08-13/SCORECARD.md`](./session-2026-08-13/SCORECARD.md) | Comprehensive multi-layer operational scorecard (10/10 across all 9 technical layers). |
| **Master Execution Program** | [`session-2026-08-13/MASTER_10_10_EXECUTION_PROGRAM.md`](./session-2026-08-13/MASTER_10_10_EXECUTION_PROGRAM.md) | 10/10 execution tracks and phase completion records. |
| **Residual Register** | [`session-2026-08-13/RESIDUAL_REGISTER.md`](./session-2026-08-13/RESIDUAL_REGISTER.md) | Itemized list of external deploy-time credentials (Layer B) vs completed code (Layer A). |
| **Gap Ledger** | [`session-2026-08-13/GAP_LEDGER.md`](./session-2026-08-13/GAP_LEDGER.md) | Tracked gap ledger items (G1-A1 through G7-4 verified resolved in code). |
| **Data Flow As Built** | [`DATA_FLOW_AS_IMPLEMENTED.md`](./DATA_FLOW_AS_IMPLEMENTED.md) | Real as-built data pipeline and event streaming topology. |
| **Optimizer Runtime** | [`OPTIMIZER_AND_ROUTING_RUNTIME.md`](./OPTIMIZER_AND_ROUTING_RUNTIME.md) | OR-Tools sidecar vs heuristic runtime dispatch architecture. |
| **Domain Proofs** | `FISCAL_EDS_PROOF.md`, `PAYOUT_RAIL_DECISION.md`, etc. | Capability-specific architectural proofs and decision records. |

---

## 2. Secondary & Planning Specifications (Defer to Living SoTs on Conflict)

| Document | Role / Scope |
|---|---|
| [`ECOSYSTEM_FEATURES_BY_ROLE.md`](./ECOSYSTEM_FEATURES_BY_ROLE.md) | Deep narrative feature index. |
| [`FEATURES_BY_APP_ROLE.md`](./FEATURES_BY_APP_ROLE.md) | Comprehensive route and client navigation inventory. |
| [`DEMAND_CLASS_IBP_SLICE.md`](./DEMAND_CLASS_IBP_SLICE.md) | Demand classification and planning specification. |
| [`PegasusX_o9_Digital_Brain_Feature_Extraction_Integration_Blueprint.md`](./PegasusX_o9_Digital_Brain_Feature_Extraction_Integration_Blueprint.md) | Advanced digital brain and supply planning blueprint. |
| [`PegasusX_o9_Demand_Planning_Problems_Extraction.md`](./PegasusX_o9_Demand_Planning_Problems_Extraction.md) | Logistics challenges and demand planning problems extraction. |
| App `README.md` files under `apps/*` | Subsystem-specific build and developer documentation. |

---

## 3. Historical & Frozen Artifacts

The following documents represent frozen snapshots or historical progress notes. They are maintained for provenance and evidence tracking:

### Markdown Snapshots
- `session-2026-08-07/**` (End-Product Reality Report historical snapshots, subagent audits, gap registers)
- `session-2026-08-12/**` (Historical backend parity wave reports)
- `artifacts/**` (Snapshot archives)
- `big-platform-baseline/**` (Early planning baseline)
- `DEPLOYMENT_READINESS_GAP_LEDGER.md`
- `ECOSYSTEM_HARDENING_GAP_PLAN.md`
- `NEXT_LAYER_ECOSYSTEM_PLAN.md`
- `PEGASUSX_MASTER_ROADMAP.md`

### Word Exports (`.docx`)
All 8 historical `.docx` files have been converted to Markdown and archived in `archive/docx/`:
- `archive/docx/PegasusX_End_Product_Reality_Report_2026-08-11_v2.docx`
- `archive/docx/PegasusX_End_Product_Reality_Report_2026-08-13.docx`
- `archive/docx/DOCS_CODE_ALIGNMENT_STATUS_2026-08-12.docx`
- `archive/docx/END_PRODUCT_REALITY_REPORT.docx`
- `archive/docx/END_PRODUCT_REALITY_REPORT_2026-08-11.docx`
- `archive/docx/END_PRODUCT_REALITY_REPORT_2026-08-13.docx`
- `archive/docx/PegasusX_End_Product_Reality_Report.docx`
- `archive/docx/DOCS_CODE_ALIGNMENT_STATUS_2026-08-12.docx` (artifacts copy)

---

## 4. Alignment Checklist & Maintenance Protocol

When modifying or verifying code in `pegasusX/`:
1. Verify claims against source code using exact `file:line` references.
2. Confirm that all 29 route packages, multi-tenancy `SupplierId`, outbox tables, and 6 role-row client apps are described truthfully.
3. Distinguish between Layer A (completed software implementation) and Layer B (live operational secrets/infrastructure).
4. Maintain honest product boundaries (RFC 7807 HTTP 410 for gated or unwired endpoints).
5. Never introduce references to legacy `pegasus/` as active product code.
