# Documentation source of truth (living vs frozen)

**Date:** 2026-08-12  
**Rule:** Prefer the living SoTs below. Historical reports keep evidence value but must not drive planning without a code re-verify.

**Agent honesty:** Status and cloud-readiness answers come from **current code**, not these docs. Always-on rules: `honest-code-gate` skill + `.cursor/rules/{honesty-code-is-truth,production-cloud-gate,phased-verify-impact}.mdc`. Matrix **"Wired"** and this file are not permission to wire cloud.

**Formats:** Living specs are **Markdown**. Three historical **`.docx`** exports exist (End Product Reality Report); each is frozen and has a README sidecar — do not treat Word as SoT.

## Living SoTs (plan from these)

| Doc | Role |
|-----|------|
| [`PROD_READINESS_SEQUENCE.md`](./PROD_READINESS_SEQUENCE.md) | Ordered residuals R0–R6 after W0–W5 |
| [`PROD_ECOSYSTEM_GOAL.md`](./PROD_ECOSYSTEM_GOAL.md) | North-star product goal |
| [`session-2026-08-07/ECOSYSTEM_GAP_REGISTER_2026-08-12.md`](./session-2026-08-07/ECOSYSTEM_GAP_REGISTER_2026-08-12.md) | Evidence backlog + Part 5 re-verify notes |
| [`session-2026-08-07/MASTER_ALIGNMENT_DATAFLOW_2026-08-12.md`](./session-2026-08-07/MASTER_ALIGNMENT_DATAFLOW_2026-08-12.md) | Docs↔code↔role×platform alignment |
| [`FEATURES_BY_APP_ROLE.md`](./FEATURES_BY_APP_ROLE.md) | Routes + client nav inventory |
| [`ROLE_FEATURES_DOCS_VS_CODE.md`](./ROLE_FEATURES_DOCS_VS_CODE.md) | Docs vs code + **phased modular plan** (P0–P16). Evidence only — **re-verify in code**; not a cloud go-live certificate |
| [`GLOBAL_SCALE_PROGRAM.md`](./GLOBAL_SCALE_PROGRAM.md) | Global register + market packs + cells; **GS-A** session/JWT pack |
| [`GLOBAL_SCALE_BACKEND_INFRA.md`](./GLOBAL_SCALE_BACKEND_INFRA.md) | Enterprise backend + infra plan (4-agent audit) |
| [`ROLE_ROW_PARITY_MATRIX.md`](./ROLE_ROW_PARITY_MATRIX.md) | Role×platform parity matrix |
| [`../PLATFORM_AUDIT.md`](../PLATFORM_AUDIT.md) | Platform audit (R0 banner + §8 often current; frozen §0/§3/§5 need care) |
| [`AUTO_ORDER.md`](./AUTO_ORDER.md) | Auto-order wiring SoT |
| [`OPTIMIZER_AND_ROUTING_RUNTIME.md`](./OPTIMIZER_AND_ROUTING_RUNTIME.md) | OR-Tools vs heuristic runtime |
| [`DATA_FLOW_AS_IMPLEMENTED.md`](./DATA_FLOW_AS_IMPLEMENTED.md) | As-built event pipeline |
| Domain proofs (`FISCAL_EDS_PROOF`, `PAYOUT_RAIL_DECISION`, `WMS_*`, `PARTNER_*`, …) | Capability-specific SoTs |

## Secondary / overview (defer to living SoTs on conflict)

| Doc | Note |
|-----|------|
| [`ECOSYSTEM_FEATURES_BY_ROLE.md`](./ECOSYSTEM_FEATURES_BY_ROLE.md) | Deep narrative; prefer `FEATURES_BY_APP_ROLE` for nav/routes |
| App `README.md` files under `apps/*` | Local how-to; parity claims must match shells |

## Historical / frozen (banners applied 2026-08-12)

Point-in-time snapshots. Banners point to living SoTs above.

### Markdown
- `session-2026-08-07/END_PRODUCT_REALITY_REPORT*.md`
- `session-2026-08-07/report-parts/*`
- `session-2026-08-07/subagent-audits/*`
- `session-2026-08-07/ENTERPRISE_GRADE_EXECUTION_PLAN.md`
- `session-2026-08-07/PHASE*`, `DOMAIN*`, `NEXT_FORK*`, `ANALYTICS*`, `LIVE_MIGRATION*` progress notes
- `DEPLOYMENT_READINESS_GAP_LEDGER.md`
- `ECOSYSTEM_HARDENING_GAP_PLAN.md`
- `NEXT_LAYER_ECOSYSTEM_PLAN.md`
- `PEGASUSX_MASTER_ROADMAP.md`
- `WAVE_C_*.md`
- `big-platform-baseline/**` (planning baseline; many items later wired)

### Word (`.docx`) — frozen exports only
- [`session-2026-08-07/END_PRODUCT_REALITY_REPORT.docx`](./session-2026-08-07/END_PRODUCT_REALITY_REPORT.docx) (+ `_2026-08-11`) — see [`session-2026-08-07/README_DOCX.md`](./session-2026-08-07/README_DOCX.md)
- [`../artifacts/PegasusX_End_Product_Reality_Report.docx`](../artifacts/PegasusX_End_Product_Reality_Report.docx) — see [`../artifacts/README_DOCX.md`](../artifacts/README_DOCX.md)

## Alignment checklist (when code ships)

1. Update `FEATURES_BY_APP_ROLE` client nav rows.  
2. Mark the matching `PROD_READINESS_SEQUENCE` R-item ✅.  
3. Append a Part 5 note on the gap register.  
4. Strike orphan/portal-only wording in `MASTER_ALIGNMENT` / `PLATFORM_AUDIT` if still present.  
5. Do **not** rewrite frozen historical bodies — banner is enough unless the file claims to be current SoT.  
6. If exporting Word again, stamp FROZEN + link this file; never let `.docx` become the planning SoT.

## Alignment pass (2026-08-12 multi-agent)

Six parallel audits (backend/data-plane, clients/role-row, money/fiscal, infra/ops/partner, docx+parent README, stale-phrase grep) re-verified docs against code.

**Living docs fixed this pass:** `README.md` (root pegasusX), `context/current_status.md`, `context/FRONTEND_STATUS.md`, `docs/SUBSTANCE_GATE.md`, `docs/DATA_FLOW_AS_IMPLEMENTED.md`, `docs/MULTI_TENANCY_GATE5_PHASE1.md` (Context historical), `docs/ROLE_ROW_PARITY_MATRIX.md`, `docs/SOLIQ_SANDBOX_READINESS.md`, `CREDIT_COLLECTIONS_ENGINE_PLAN.md`, `docs/CREDIT_ECOSYSTEM_BEHAVIOR.md`, `docs/OPTIMIZER_AND_ROUTING_RUNTIME.md` header, parent `V.O.I.D/README.md` paths → `pegasusX/`, app README path prefixes, privacy-multi-tenant note.

**Batch banners:** ~28 empty SOP stubs → OPS STUB; historical session/artifacts banners; big-platform-baseline PLANNING BASELINE notes.

**Word:** frozen Reality Reports keep sidecars only. New living export: [`DOCS_CODE_ALIGNMENT_STATUS_2026-08-12.docx`](./DOCS_CODE_ALIGNMENT_STATUS_2026-08-12.docx) (also under `artifacts/`). Parent: [`../../PegasusX_Reality_Report.README.md`](../../PegasusX_Reality_Report.README.md).

