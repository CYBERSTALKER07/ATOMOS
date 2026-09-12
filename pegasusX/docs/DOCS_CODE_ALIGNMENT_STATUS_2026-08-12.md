# PegasusX — Docs↔Code Alignment Status

*Date: 2026-08-12 · Living Word export. Markdown under docs/ remains the planning SoT.*

## Planning SoT

1. docs/DOCS_SOURCE_OF_TRUTH.md — living vs frozen map

2. docs/PROD_ECOSYSTEM_GOAL.md — north-star prod goal

3. docs/PROD_READINESS_SEQUENCE.md — residuals R0–R6

4. docs/session-2026-08-07/ECOSYSTEM_GAP_REGISTER_2026-08-12.md

5. docs/session-2026-08-07/MASTER_ALIGNMENT_DATAFLOW_2026-08-12.md

## Code-verified truths

• Kernel: Spanner + same-txn outbox → Kafka → WS/FCM/webhooks

• AR + payout emit outbox; twin consumer started; JWT jti denylist

• Multi-tenancy Phase 1–3 backends wired; seed = bootstrap fallback

• Partner /partner/v1 + OpenAPI + SDK in go.work; AS2 MDN/MIC; GS1 FNC1 DataMatrix

• admin-portal is a live PLATFORM_ADMIN console (not a stub)

• Supplier desktop = supplier-portal Tauri (no supplier-app-desktop)

• Optimizer: SSMR replicas 1; prod replicas 0 until AR image

• Bank-file is permanent payout for this prod bar

## Still open (ops)

• Global Pay live SUCCESS (merchant password)

• Firebase real SMS / device trust

• Soliq legal OFD needs E-IMZO PKCS#12

• Prod optimizer replicas ≥1 after AR publish

• enable_observability_resources for live SLO paging

• Auto-order place stays off until soak dual-control flip

## Frozen Reality Reports

END_PRODUCT_REALITY_REPORT*.docx and parent PegasusX_Reality_Report.docx are historical only. See README_DOCX sidecars and PegasusX_Reality_Report.README.md.

## Alignment method

Six parallel explore agents scanned backend, clients, money, infra, docx, and stale phrases. Living SoTs updated; historical banners applied across stubs, baseline, and session archives.
