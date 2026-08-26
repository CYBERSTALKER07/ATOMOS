# PegasusX Repository Documentation Inventory & Claims Report

**Generated:** 2026-08-20T17:25:00+05:00  
**Auditor:** Explorer 1 (`teamwork_preview_explorer_survey_1`)  
**Scope:** Full repository scan of all `.md` and `.docx` files in `/Users/shakhzod/Desktop/V.O.I.D`  
**Total Documents Cataloged:** 803 files across 21 categories  

---

## 1. Executive Summary

This report establishes a comprehensive inventory and claims audit of all documentation across the repository. The investigation cataloged **803 project documents**, organized them into 21 distinct functional domains, and extracted all explicit status claims regarding implementation completeness, parity status ("Wired"), phase completion ("Done"), production/cloud readiness, scorecards, stubs, and residuals.

### Key Findings & Architectural Taxonomy

1. **Living Source of Truth Hierarchy**:
   - **Primary North Star**: `.agents/memory/GOAL.md` directing to destination programs `pegasusX/docs/GLOBAL_SCALE_PROGRAM.md` and `pegasusX/docs/GLOBAL_SCALE_LOCAL_ECOSYSTEM.md`.
   - **Living Monorepo**: `pegasusX/` is the active codebase and living specification root. `pegasus/` is a legacy reference/port source only.
   - **Documentation Map**: `pegasusX/docs/DOCS_SOURCE_OF_TRUTH.md` governs living vs frozen documentation.
   - **Core Status Matrix**: `pegasusX/docs/ROLE_ROW_PARITY_MATRIX.md` defines role-row client and backend parity across 6 core roles + Platform Admin.
   - **Living Scorecard & Program**: `pegasusX/docs/session-2026-08-13/SCORECARD.md` and `MASTER_10_10_EXECUTION_PROGRAM.md`.

2. **Frozen Word Exports (.docx) vs Living Markdown**:
   - **8 `.docx` files** exist across the workspace. All represent historical exports (notably the *End-Product Reality Report* v2 dated 2026-08-11 and 2026-08-13, and *Docs<->Code Alignment Status* dated 2026-08-12).
   - As mandated by `PegasusX_Reality_Report.README.md` and `DOCS_SOURCE_OF_TRUTH.md`, these `.docx` files are **frozen historical artifacts** and are strictly not to be used for active planning without verifying against current code.

3. **Status Claims Summary**:
   - **"Wired" Claims**: All 6 roles (Supplier, Retailer, Driver, Warehouse, Factory, Payload) plus Platform Admin are labeled as "Wired" for happy-path Class A execution in `ROLE_ROW_PARITY_MATRIX.md`.
   - **"Done" Phase Claims**: Phases 0 through G7 in `MASTER_10_10_EXECUTION_PROGRAM.md` and Gap Ledger items G1-A1 through G7-4 are marked "DONE" in `session-2026-08-13/GAP_LEDGER.md`.
   - **Scorecard Claims**: Living scorecard claims **10/10** across 9 layers and **9.5/10** on Fiscal/Legal readiness.
   - **Explicit Stubs & Residuals**: Documented in `RESIDUAL_REGISTER.md`, `PROD_READINESS_SEQUENCE.md`, and `ROLE_FEATURES_DOCS_VS_CODE.md`: Adyen/Stripe are stubs; Click/Payme unwired in execution routes; Soliq/EDS requires live PKCS#12 secrets; FCM requires owner credentials; Auto-Order place soak flip is OFF; Payout rail is Bank-File only; OR-Tools prod replicas = 0.

---

## 2. Category Breakdown & Document Counts

| Category | Document Count | Key Focus / Purpose |
|---|---|---|
| Pegasus Legacy / Reference | 226 | Comprehensive documentation and specs for Pegasus Legacy / Reference |
| pegasusX / Core Docs & Specifications | 137 | Comprehensive documentation and specs for pegasusX / Core Docs & Specifications |
| pegasusX / Artifacts & Snapshots | 70 | Comprehensive documentation and specs for pegasusX / Artifacts & Snapshots |
| pegasusX / Big Platform Baseline (Deep Specs) | 57 | Comprehensive documentation and specs for pegasusX / Big Platform Baseline (Deep Specs) |
| pegasusX / Session 2026-08-07 (Reality Reports & Gap Registers) | 42 | Comprehensive documentation and specs for pegasusX / Session 2026-08-07 (Reality Reports & Gap Registers) |
| pegasusX / Visuals & Media | 39 | Comprehensive documentation and specs for pegasusX / Visuals & Media |
| pegasusX / Apps Documentation | 36 | Comprehensive documentation and specs for pegasusX / Apps Documentation |
| pegasusX / Session 2026-08-13 (Scorecards, Master Program, Phases) | 34 | Comprehensive documentation and specs for pegasusX / Session 2026-08-13 (Scorecards, Master Program, Phases) |
| pegasusX / SDK Documentation | 32 | Comprehensive documentation and specs for pegasusX / SDK Documentation |
| pegasusX / Root | 28 | Comprehensive documentation and specs for pegasusX / Root |
| GitHub Workflows & Instructions | 23 | Comprehensive documentation and specs for GitHub Workflows & Instructions |
| pegasusX / Design System | 19 | Comprehensive documentation and specs for pegasusX / Design System |
| pegasusX / Session 2026-08-12 (Backend Parity & Waves) | 17 | Comprehensive documentation and specs for pegasusX / Session 2026-08-12 (Backend Parity & Waves) |
| Other | 13 | Comprehensive documentation and specs for Other |
| Repository Root | 8 | Comprehensive documentation and specs for Repository Root |
| Agents Framework & Memory | 5 | Comprehensive documentation and specs for Agents Framework & Memory |
| pegasusX / Context Phase Plans & Parity Ledger | 5 | Comprehensive documentation and specs for pegasusX / Context Phase Plans & Parity Ledger |
| pegasusX / Gap Closure | 5 | Comprehensive documentation and specs for pegasusX / Gap Closure |
| pegasusX / Packages Documentation | 3 | Comprehensive documentation and specs for pegasusX / Packages Documentation |
| Root Docs / Archive | 2 | Comprehensive documentation and specs for Root Docs / Archive |
| pegasusX / Infra Documentation | 2 | Comprehensive documentation and specs for pegasusX / Infra Documentation |

---

## 3. Detailed Document Inventory by Category

### Pegasus Legacy / Reference (226 files)

| File Path | Lines | Size (Bytes) | Header / Title |
|---|---|---|---|
| `pegasus/E2E_TEST_PROTOCOL.md` | 86 | 3827 | LEVIATHAN RC1: E2E INTEGRATION TEST PROTOCOL |
| `pegasus/README.md` | 18 | 1078 | PegasusX Platform |
| `pegasus/apps/admin-portal/.expo/README.md` | 15 | 891 | > Why do I have a folder named ".expo" in my project? |
| `pegasus/apps/admin-portal/README.md` | 36 | 1450 | Getting Started |
| `pegasus/apps/factory-app-ios/README.md` | 11 | 381 | // !$*UTF8*$! |
| `pegasus/apps/payload-app-android/README.md` | 52 | 2022 | Pegasus Payload — Android Tablet App |
| `pegasus/apps/payload-app-ios/README.md` | 53 | 2561 | Pegasus Payload — iPad App |
| `pegasus/apps/payload-terminal/.expo/README.md` | 13 | 756 | > Why do I have a folder named ".expo" in my project? |
| `pegasus/context/FRONTEND_STATUS.md` | 74 | 1184 | Frontend Apps - Codebase Status |
| `pegasus/context/architecture.md` | 53 | 2506 | Real Codebase Infrastructure & Architecture |
| `pegasus/context/current_status.md` | 40 | 2025 | Detailed Migration Status & Execution Plan |
| `pegasus/context/design-system.md` | 51 | 2695 | V.O.I.D. Design System |
| `pegasus/context/plan.md` | 229 | 16117 | Pegasus Enterprise Execution Plan (90 Days) |
| `pegasus/context/plan_2.0.md` | 230 | 7798 | Pegasus OR-Tools Deployment Plan 2.0 |
| `pegasus/context/purpose.md` | 17 | 1488 | V.O.I.D. Ecosystem Purpose & Protocol |
| `pegasus/context/technology-inventory.md` | 454 | 72805 | Technology Inventory |
| `pegasus/context/ui-design.md` | 49 | 3095 | V.O.I.D. UI / UX Doctrine |
| `pegasus/design.md` | 34 | 1188 | Desktop Design Contract (Pegasus Mirror) |
| `pegasus/docs/PAYMENT_SYSTEM_TARGET_ARCHITECTURE.md` | 254 | 5849 | Pegasus Payment System Target Architecture |
| `pegasus/docs/adr/009-fiscal-hard-gate.md` | 8 | 382 | Moved: ADR-009 lives in pegasusX |
| `pegasus/docs/adr/README.md` | 3 | 84 | ADRs |
| `pegasus/patent-dossier/counsel/driver-delivery-correction/patent_application.md` | 249 | 18337 | Driver Delivery Correction Patent Package |
| `pegasus/patent-dossier/i18n/README.md` | 49 | 5310 | Pegasus Patent Dossier I18N Index |
| `pegasus/patent-dossier/i18n/figure-caption-templates.multilingual.md` | 83 | 4065 | Templates |
| `pegasus/patent-dossier/i18n/filing-review-pack-high-priority.manifest.md` | 149 | 4863 | Surfaces |
| `pegasus/patent-dossier/i18n/filing-review-pack-high-priority.ru.md` | 113 | 25787 | Pegasus: Высокоприоритетный Пакет Для Патентной Подачи |
| `pegasus/patent-dossier/i18n/filing-review-pack-high-priority.uz.md` | 113 | 13762 | Pegasus: Yuqori Ustuvor Patent Taqdimoti Uchun Review Paket |
| `pegasus/patent-dossier/i18n/future-autonomous-vision.ru.md` | 98 | 8824 | Будущее Автономной Логистики Pegasus (No Human Loop) |
| `pegasus/patent-dossier/i18n/multilingual-abstracts.md` | 107 | 9207 | Abstracts |
| `pegasus/patent-dossier/i18n/page-dossiers.ru.md` | 9660 | 353983 | Notes |
| `pegasus/patent-dossier/i18n/page-dossiers.uz.md` | 9660 | 302329 | Notes |
| `pegasus/patent-dossier/i18n/patent-algorithm-atlas.ru.md` | 402 | 20594 | Атлас Алгоритмов Pegasus (RU) |
| `pegasus/patent-dossier/i18n/patent-claim-skeleton.ru.md` | 95 | 10682 | Скелет Патентных Притязаний Pegasus (RU) |
| `pegasus/patent-dossier/i18n/patent-claims-draft.ru.md` | 212 | 23129 | Черновик Формулы Изобретения Pegasus (RU) |
| `pegasus/patent-dossier/i18n/patent-line-art-prompts.ru.md` | 694 | 44082 | Pegasus Patent Line-Art Prompt Library For Nano Banana |
| `pegasus/patent-dossier/i18n/patent-packet.ru.md` | 140 | 17231 | Патентный Пакет Pegasus |
| `pegasus/patent-dossier/i18n/patent-packet.uz.md` | 128 | 9518 | Pegasus Patent Paketi |
| `pegasus/patent-dossier/i18n/terminology-glossary.multilingual.md` | 265 | 3677 | Terms |
| `pegasus/patent-dossier/page-dossiers/driver-android-cash-collection.md` | 153 | 2373 | Sourcefiles |
| `pegasus/patent-dossier/page-dossiers/driver-android-delivery-correction.md` | 320 | 5216 | Sourcefiles |
| `pegasus/patent-dossier/page-dossiers/driver-android-offload-review.md` | 217 | 3283 | Sourcefiles |
| `pegasus/patent-dossier/page-dossiers/driver-android-payment-waiting.md` | 145 | 2341 | Sourcefiles |
| `pegasus/patent-dossier/page-dossiers/driver-android-root-shell.md` | 194 | 3063 | Sourcefiles |
| `pegasus/patent-dossier/page-dossiers/driver-android-scanner.md` | 218 | 3174 | Sourcefiles |
| `pegasus/patent-dossier/page-dossiers/driver-android-secondary-surfaces.md` | 404 | 6205 | Surfaces |
| `pegasus/patent-dossier/page-dossiers/driver-ios-cash-collection.md` | 144 | 1953 | Sourcefiles |
| `pegasus/patent-dossier/page-dossiers/driver-ios-delivery-correction.md` | 225 | 3468 | Sourcefiles |
| `pegasus/patent-dossier/page-dossiers/driver-ios-offload-review.md` | 199 | 2978 | Sourcefiles |
| `pegasus/patent-dossier/page-dossiers/driver-ios-payment-waiting.md` | 151 | 2365 | Sourcefiles |
| `pegasus/patent-dossier/page-dossiers/driver-ios-qr-scanner.md` | 187 | 2741 | Sourcefiles |
| `pegasus/patent-dossier/page-dossiers/driver-ios-root-shell.md` | 201 | 2872 | Sourcefiles |
| `pegasus/patent-dossier/page-dossiers/driver-ios-secondary-surfaces.md` | 640 | 11823 | Surfaces |
| `pegasus/patent-dossier/page-dossiers/payload-auth-loading.md` | 60 | 888 | Sourcefiles |
| `pegasus/patent-dossier/page-dossiers/payload-dispatch-success.md` | 97 | 1534 | Sourcefiles |
| `pegasus/patent-dossier/page-dossiers/payload-login.md` | 86 | 1272 | Sourcefiles |
| `pegasus/patent-dossier/page-dossiers/payload-manifest-workspace.md` | 186 | 3352 | Layoutzones |
| `pegasus/patent-dossier/page-dossiers/payload-truck-selection.md` | 104 | 1549 | Sourcefiles |
| `pegasus/patent-dossier/page-dossiers/retailer-android-active-deliveries-sheet.md` | 120 | 1825 | Sourcefiles |
| `pegasus/patent-dossier/page-dossiers/retailer-android-auth.md` | 218 | 3244 | Sourcefiles |
| `pegasus/patent-dossier/page-dossiers/retailer-android-cart.md` | 220 | 3164 | Sourcefiles |
| `pegasus/patent-dossier/page-dossiers/retailer-android-catalog.md` | 125 | 1961 | Sourcefiles |
| `pegasus/patent-dossier/page-dossiers/retailer-android-checkout-sheet.md` | 166 | 2458 | Sourcefiles |
| `pegasus/patent-dossier/page-dossiers/retailer-android-orders.md` | 172 | 2358 | Sourcefiles |
| `pegasus/patent-dossier/page-dossiers/retailer-android-payment-sheet.md` | 217 | 3083 | Sourcefiles |
| `pegasus/patent-dossier/page-dossiers/retailer-android-root-shell.md` | 237 | 3729 | Layoutzones |
| `pegasus/patent-dossier/page-dossiers/retailer-android-secondary-surfaces.md` | 780 | 12345 | Surfaces |
| `pegasus/patent-dossier/page-dossiers/retailer-ios-active-deliveries.md` | 138 | 2112 | Sourcefiles |
| `pegasus/patent-dossier/page-dossiers/retailer-ios-cart.md` | 206 | 2965 | Sourcefiles |
| `pegasus/patent-dossier/page-dossiers/retailer-ios-catalog.md` | 168 | 2563 | Sourcefiles |
| `pegasus/patent-dossier/page-dossiers/retailer-ios-checkout.md` | 212 | 3180 | Sourcefiles |
| `pegasus/patent-dossier/page-dossiers/retailer-ios-delivery-payment-sheet.md` | 232 | 3452 | Sourcefiles |
| `pegasus/patent-dossier/page-dossiers/retailer-ios-login.md` | 252 | 3718 | Sourcefiles |
| `pegasus/patent-dossier/page-dossiers/retailer-ios-orders.md` | 190 | 2745 | Sourcefiles |
| `pegasus/patent-dossier/page-dossiers/retailer-ios-root-shell.md` | 230 | 3736 | Layoutzones |
| `pegasus/patent-dossier/page-dossiers/retailer-ios-secondary-surfaces.md` | 1072 | 15825 | Surfaces |
| `pegasus/patent-dossier/page-dossiers/supplier-orders.md` | 334 | 5031 | Layoutzones |
| `pegasus/patent-dossier/page-dossiers/web-supplier-analytics-demand.md` | 148 | 2770 | Layoutzones |
| `pegasus/patent-dossier/page-dossiers/web-supplier-analytics.md` | 197 | 3509 | Layoutzones |
| `pegasus/patent-dossier/page-dossiers/web-supplier-catalog.md` | 268 | 4656 | Layoutzones |
| `pegasus/patent-dossier/page-dossiers/web-supplier-crm.md` | 162 | 2893 | Layoutzones |
| `pegasus/patent-dossier/page-dossiers/web-supplier-dashboard.md` | 203 | 3641 | Layoutzones |
| `pegasus/patent-dossier/page-dossiers/web-supplier-depot-reconciliation.md` | 204 | 3545 | Layoutzones |
| `pegasus/patent-dossier/page-dossiers/web-supplier-dispatch.md` | 68 | 1199 | Layoutzones |
| `pegasus/patent-dossier/page-dossiers/web-supplier-fleet.md` | 355 | 6132 | Layoutzones |
| `pegasus/patent-dossier/page-dossiers/web-supplier-inventory.md` | 204 | 3184 | Layoutzones |
| `pegasus/patent-dossier/page-dossiers/web-supplier-login.md` | 149 | 2126 | Layoutzones |
| `pegasus/patent-dossier/page-dossiers/web-supplier-manifests.md` | 68 | 1199 | Layoutzones |
| `pegasus/patent-dossier/page-dossiers/web-supplier-onboarding.md` | 78 | 1482 | Layoutzones |
| `pegasus/patent-dossier/page-dossiers/web-supplier-payment-config.md` | 291 | 4872 | Layoutzones |
| `pegasus/patent-dossier/page-dossiers/web-supplier-pricing.md` | 172 | 3237 | Layoutzones |
| `pegasus/patent-dossier/page-dossiers/web-supplier-product-detail.md` | 215 | 3934 | Layoutzones |
| `pegasus/patent-dossier/page-dossiers/web-supplier-products.md` | 252 | 4210 | Layoutzones |
| `pegasus/patent-dossier/page-dossiers/web-supplier-profile.md` | 248 | 4138 | Layoutzones |
| `pegasus/patent-dossier/page-dossiers/web-supplier-register.md` | 219 | 3443 | Layoutzones |
| `pegasus/patent-dossier/page-dossiers/web-supplier-returns.md` | 196 | 3287 | Layoutzones |
| `pegasus/patent-dossier/page-dossiers/web-supplier-settings.md` | 170 | 3114 | Layoutzones |
| `pegasus/patent-dossier/page-dossiers/web-supplier-staff.md` | 198 | 3185 | Layoutzones |
| `pegasus/patent-dossier/text-architecture-pdf/INDEX.md` | 100 | 13138 | Text Architecture PDF Export |
| `pegasus/patent-dossier/text-architecture/INDEX.md` | 106 | 11246 | Text-Only Patent Architecture Corpus |
| `pegasus/patent-dossier/text-architecture/counsel/driver-delivery-correction/patent_application.md` | 91 | 7173 | Technical Patent Architecture: Driver Delivery Correction Patent Package |
| `pegasus/patent-dossier/text-architecture/i18n/README.md` | 31 | 1189 | Technical Patent Architecture: Pegasus Patent Dossier I18N Index |
| `pegasus/patent-dossier/text-architecture/i18n/figure-caption-templates.multilingual.md` | 29 | 868 | Technical Patent Architecture: Templates |
| `pegasus/patent-dossier/text-architecture/i18n/filing-review-pack-high-priority.manifest.md` | 29 | 977 | Technical Patent Architecture: Surfaces |
| `pegasus/patent-dossier/text-architecture/i18n/filing-review-pack-high-priority.ru.md` | 31 | 2206 | Technical Patent Architecture: Pegasus: Высокоприоритетный Пакет Для Патентной Подачи |
| `pegasus/patent-dossier/text-architecture/i18n/filing-review-pack-high-priority.uz.md` | 31 | 1552 | Technical Patent Architecture: Pegasus: Yuqori Ustuvor Patent Taqdimoti Uchun Review Paket |
| `pegasus/patent-dossier/text-architecture/i18n/future-autonomous-vision.ru.md` | 42 | 2616 | Technical Patent Architecture: Будущее Автономной Логистики Pegasus (No Human Loop) |
| `pegasus/patent-dossier/text-architecture/i18n/multilingual-abstracts.md` | 29 | 959 | Technical Patent Architecture: Abstracts |
| `pegasus/patent-dossier/text-architecture/i18n/page-dossiers.ru.md` | 137 | 7569 | Technical Patent Architecture: Notes |
| `pegasus/patent-dossier/text-architecture/i18n/page-dossiers.uz.md` | 137 | 7201 | Technical Patent Architecture: Notes |
| `pegasus/patent-dossier/text-architecture/i18n/patent-algorithm-atlas.ru.md` | 119 | 9238 | Technical Patent Architecture: Атлас Алгоритмов Pegasus (RU) |
| `pegasus/patent-dossier/text-architecture/i18n/patent-claim-skeleton.ru.md` | 58 | 3893 | Technical Patent Architecture: Скелет Патентных Притязаний Pegasus (RU) |
| `pegasus/patent-dossier/text-architecture/i18n/patent-claims-draft.ru.md` | 67 | 5426 | Technical Patent Architecture: Черновик Формулы Изобретения Pegasus (RU) |
| `pegasus/patent-dossier/text-architecture/i18n/patent-line-art-prompts.ru.md` | 49 | 3516 | Technical Patent Architecture: Библиотека Промптов Для Патентных Иллюстраций (RU, Black-White Line Art) |
| `pegasus/patent-dossier/text-architecture/i18n/patent-packet.ru.md` | 31 | 2539 | Technical Patent Architecture: Патентный Пакет Pegasus |
| `pegasus/patent-dossier/text-architecture/i18n/patent-packet.uz.md` | 31 | 1768 | Technical Patent Architecture: Pegasus Patent Paketi |
| `pegasus/patent-dossier/text-architecture/i18n/terminology-glossary.multilingual.md` | 29 | 868 | Technical Patent Architecture: Terms |
| `pegasus/patent-dossier/text-architecture/page-dossiers/driver-android-cash-collection.md` | 46 | 1882 | Technical Patent Architecture: driver-android-cash-collection |
| `pegasus/patent-dossier/text-architecture/page-dossiers/driver-android-delivery-correction.md` | 56 | 2222 | Technical Patent Architecture: driver-android-delivery-correction |
| `pegasus/patent-dossier/text-architecture/page-dossiers/driver-android-offload-review.md` | 50 | 2044 | Technical Patent Architecture: driver-android-offload-review |
| `pegasus/patent-dossier/text-architecture/page-dossiers/driver-android-payment-waiting.md` | 48 | 1874 | Technical Patent Architecture: driver-android-payment-waiting |
| `pegasus/patent-dossier/text-architecture/page-dossiers/driver-android-root-shell.md` | 61 | 2381 | Technical Patent Architecture: driver-android-root-shell |
| `pegasus/patent-dossier/text-architecture/page-dossiers/driver-android-scanner.md` | 46 | 1800 | Technical Patent Architecture: driver-android-scanner |
| `pegasus/patent-dossier/text-architecture/page-dossiers/driver-android-secondary-surfaces.md` | 97 | 2972 | Technical Patent Architecture: Surfaces |
| `pegasus/patent-dossier/text-architecture/page-dossiers/driver-ios-cash-collection.md` | 43 | 1558 | Technical Patent Architecture: driver-ios-cash-collection |
| `pegasus/patent-dossier/text-architecture/page-dossiers/driver-ios-delivery-correction.md` | 51 | 1920 | Technical Patent Architecture: driver-ios-delivery-correction |
| `pegasus/patent-dossier/text-architecture/page-dossiers/driver-ios-offload-review.md` | 52 | 1935 | Technical Patent Architecture: driver-ios-offload-review |
| `pegasus/patent-dossier/text-architecture/page-dossiers/driver-ios-payment-waiting.md` | 51 | 1867 | Technical Patent Architecture: driver-ios-payment-waiting |
| `pegasus/patent-dossier/text-architecture/page-dossiers/driver-ios-qr-scanner.md` | 47 | 1665 | Technical Patent Architecture: driver-ios-qr-scanner |
| `pegasus/patent-dossier/text-architecture/page-dossiers/driver-ios-root-shell.md` | 55 | 2128 | Technical Patent Architecture: driver-ios-root-shell |
| `pegasus/patent-dossier/text-architecture/page-dossiers/driver-ios-secondary-surfaces.md` | 112 | 4104 | Technical Patent Architecture: Surfaces |
| `pegasus/patent-dossier/text-architecture/page-dossiers/payload-auth-loading.md` | 39 | 1314 | Technical Patent Architecture: payload-auth-loading |
| `pegasus/patent-dossier/text-architecture/page-dossiers/payload-dispatch-success.md` | 42 | 1625 | Technical Patent Architecture: payload-dispatch-success |
| `pegasus/patent-dossier/text-architecture/page-dossiers/payload-login.md` | 45 | 1499 | Technical Patent Architecture: payload-login |
| `pegasus/patent-dossier/text-architecture/page-dossiers/payload-manifest-workspace.md` | 60 | 2196 | Technical Patent Architecture: Layoutzones |
| `pegasus/patent-dossier/text-architecture/page-dossiers/payload-truck-selection.md` | 47 | 1736 | Technical Patent Architecture: payload-truck-selection |
| `pegasus/patent-dossier/text-architecture/page-dossiers/retailer-android-active-deliveries-sheet.md` | 44 | 1871 | Technical Patent Architecture: retailer-android-active-deliveries-sheet |
| `pegasus/patent-dossier/text-architecture/page-dossiers/retailer-android-auth.md` | 51 | 1872 | Technical Patent Architecture: retailer-android-auth |
| `pegasus/patent-dossier/text-architecture/page-dossiers/retailer-android-cart.md` | 49 | 1912 | Technical Patent Architecture: retailer-android-cart |
| `pegasus/patent-dossier/text-architecture/page-dossiers/retailer-android-catalog.md` | 45 | 1876 | Technical Patent Architecture: retailer-android-catalog |
| `pegasus/patent-dossier/text-architecture/page-dossiers/retailer-android-checkout-sheet.md` | 45 | 1688 | Technical Patent Architecture: retailer-android-checkout-sheet |
| `pegasus/patent-dossier/text-architecture/page-dossiers/retailer-android-orders.md` | 52 | 1817 | Technical Patent Architecture: retailer-android-orders |
| `pegasus/patent-dossier/text-architecture/page-dossiers/retailer-android-payment-sheet.md` | 48 | 1819 | Technical Patent Architecture: retailer-android-payment-sheet |
| `pegasus/patent-dossier/text-architecture/page-dossiers/retailer-android-root-shell.md` | 59 | 2177 | Technical Patent Architecture: Layoutzones |
| `pegasus/patent-dossier/text-architecture/page-dossiers/retailer-android-secondary-surfaces.md` | 109 | 4028 | Technical Patent Architecture: Surfaces |
| `pegasus/patent-dossier/text-architecture/page-dossiers/retailer-ios-active-deliveries.md` | 49 | 1781 | Technical Patent Architecture: retailer-ios-active-deliveries |
| `pegasus/patent-dossier/text-architecture/page-dossiers/retailer-ios-cart.md` | 46 | 1768 | Technical Patent Architecture: retailer-ios-cart |
| `pegasus/patent-dossier/text-architecture/page-dossiers/retailer-ios-catalog.md` | 48 | 1698 | Technical Patent Architecture: retailer-ios-catalog |
| `pegasus/patent-dossier/text-architecture/page-dossiers/retailer-ios-checkout.md` | 52 | 1916 | Technical Patent Architecture: retailer-ios-checkout |
| `pegasus/patent-dossier/text-architecture/page-dossiers/retailer-ios-delivery-payment-sheet.md` | 49 | 1806 | Technical Patent Architecture: retailer-ios-delivery-payment-sheet |
| `pegasus/patent-dossier/text-architecture/page-dossiers/retailer-ios-login.md` | 51 | 1772 | Technical Patent Architecture: retailer-ios-login |
| `pegasus/patent-dossier/text-architecture/page-dossiers/retailer-ios-orders.md` | 53 | 1902 | Technical Patent Architecture: retailer-ios-orders |
| `pegasus/patent-dossier/text-architecture/page-dossiers/retailer-ios-root-shell.md` | 53 | 1792 | Technical Patent Architecture: Layoutzones |
| `pegasus/patent-dossier/text-architecture/page-dossiers/retailer-ios-secondary-surfaces.md` | 109 | 4120 | Technical Patent Architecture: Surfaces |
| `pegasus/patent-dossier/text-architecture/page-dossiers/supplier-orders.md` | 64 | 2328 | Technical Patent Architecture: Layoutzones |
| `pegasus/patent-dossier/text-architecture/page-dossiers/web-supplier-analytics-demand.md` | 54 | 2046 | Technical Patent Architecture: Layoutzones |
| `pegasus/patent-dossier/text-architecture/page-dossiers/web-supplier-analytics.md` | 54 | 2176 | Technical Patent Architecture: Layoutzones |
| `pegasus/patent-dossier/text-architecture/page-dossiers/web-supplier-catalog.md` | 64 | 2567 | Technical Patent Architecture: Layoutzones |
| `pegasus/patent-dossier/text-architecture/page-dossiers/web-supplier-crm.md` | 54 | 2073 | Technical Patent Architecture: Layoutzones |
| `pegasus/patent-dossier/text-architecture/page-dossiers/web-supplier-dashboard.md` | 54 | 2386 | Technical Patent Architecture: Layoutzones |
| `pegasus/patent-dossier/text-architecture/page-dossiers/web-supplier-depot-reconciliation.md` | 52 | 2145 | Technical Patent Architecture: Layoutzones |
| `pegasus/patent-dossier/text-architecture/page-dossiers/web-supplier-dispatch.md` | 39 | 1505 | Technical Patent Architecture: Layoutzones |
| `pegasus/patent-dossier/text-architecture/page-dossiers/web-supplier-fleet.md` | 74 | 3032 | Technical Patent Architecture: Layoutzones |
| `pegasus/patent-dossier/text-architecture/page-dossiers/web-supplier-inventory.md` | 56 | 2082 | Technical Patent Architecture: Layoutzones |
| `pegasus/patent-dossier/text-architecture/page-dossiers/web-supplier-login.md` | 40 | 1243 | Technical Patent Architecture: Layoutzones |
| `pegasus/patent-dossier/text-architecture/page-dossiers/web-supplier-manifests.md` | 39 | 1514 | Technical Patent Architecture: Layoutzones |
| `pegasus/patent-dossier/text-architecture/page-dossiers/web-supplier-onboarding.md` | 43 | 1854 | Technical Patent Architecture: Layoutzones |
| `pegasus/patent-dossier/text-architecture/page-dossiers/web-supplier-payment-config.md` | 56 | 2108 | Technical Patent Architecture: Layoutzones |
| `pegasus/patent-dossier/text-architecture/page-dossiers/web-supplier-pricing.md` | 56 | 2208 | Technical Patent Architecture: Layoutzones |
| `pegasus/patent-dossier/text-architecture/page-dossiers/web-supplier-product-detail.md` | 53 | 2030 | Technical Patent Architecture: Layoutzones |
| `pegasus/patent-dossier/text-architecture/page-dossiers/web-supplier-products.md` | 58 | 2389 | Technical Patent Architecture: Layoutzones |
| `pegasus/patent-dossier/text-architecture/page-dossiers/web-supplier-profile.md` | 57 | 2230 | Technical Patent Architecture: Layoutzones |
| `pegasus/patent-dossier/text-architecture/page-dossiers/web-supplier-register.md` | 45 | 1545 | Technical Patent Architecture: Layoutzones |
| `pegasus/patent-dossier/text-architecture/page-dossiers/web-supplier-returns.md` | 54 | 2056 | Technical Patent Architecture: Layoutzones |
| `pegasus/patent-dossier/text-architecture/page-dossiers/web-supplier-settings.md` | 56 | 2146 | Technical Patent Architecture: Layoutzones |
| `pegasus/patent-dossier/text-architecture/page-dossiers/web-supplier-staff.md` | 52 | 1898 | Technical Patent Architecture: Layoutzones |
| `pegasus/patent-dossier/text-architecture/pegasus-protected-technical-brief.md` | 313 | 45212 | Pegasus Protected Technical and Non-Technical Brief |
| `pegasus/patent-dossier/text-architecture/pegasus-protected-technical-brief.ru.md` | 313 | 91597 | Pegasus: защищенное техническое и нетехническое описание |
| `pegasus/patent-dossier/text-architecture/pegasus-protected-technical-brief.uz.md` | 313 | 45963 | Pegasus himoyalangan texnik va notexnik bayoni |
| `pegasus/patent-dossier/text-architecture/tools/patent-architect/batch-01-supplier-core/patent-alternative-embodiments.md` | 49 | 2090 | Technical Patent Architecture: Alternative Embodiment Permutation Matrix |
| `pegasus/patent-dossier/text-architecture/tools/patent-architect/batch-01-supplier-core/patent-core-algorithms.md` | 75 | 4146 | Technical Patent Architecture: Patent Core Algorithms |
| `pegasus/patent-dossier/text-architecture/tools/patent-architect/batch-01-supplier-core/patent-feature-catalog.md` | 69 | 2426 | Technical Patent Architecture: Comprehensive Feature Catalog |
| `pegasus/patent-dossier/text-architecture/tools/patent-architect/batch-02a-driver-mobile/patent-alternative-embodiments.md` | 37 | 1395 | Technical Patent Architecture: Batch 02A - Driver Mobile Alternative Embodiments |
| `pegasus/patent-dossier/text-architecture/tools/patent-architect/batch-02a-driver-mobile/patent-core-algorithms.md` | 64 | 3342 | Technical Patent Architecture: Batch 02A - Driver Mobile Core Algorithms |
| `pegasus/patent-dossier/text-architecture/tools/patent-architect/batch-02a-driver-mobile/patent-feature-catalog.md` | 43 | 1738 | Technical Patent Architecture: Batch 02A - Driver Mobile Feature Catalog |
| `pegasus/patent-dossier/text-architecture/tools/patent-architect/batch-02b-retailer-multi/patent-alternative-embodiments.md` | 37 | 1467 | Technical Patent Architecture: Batch 02B - Retailer Multi-Surface Alternative Embodiments |
| `pegasus/patent-dossier/text-architecture/tools/patent-architect/batch-02b-retailer-multi/patent-core-algorithms.md` | 63 | 3041 | Technical Patent Architecture: Batch 02B - Retailer Multi-Surface Core Algorithms |
| `pegasus/patent-dossier/text-architecture/tools/patent-architect/batch-02b-retailer-multi/patent-feature-catalog.md` | 38 | 1385 | Technical Patent Architecture: Batch 02B - Retailer Multi-Surface Feature Catalog |
| `pegasus/patent-dossier/text-architecture/tools/patent-architect/batch-02c-payload-surfaces/patent-alternative-embodiments.md` | 37 | 1373 | Technical Patent Architecture: Batch 02C - Payload Surfaces Alternative Embodiments |
| `pegasus/patent-dossier/text-architecture/tools/patent-architect/batch-02c-payload-surfaces/patent-core-algorithms.md` | 62 | 2916 | Technical Patent Architecture: Batch 02C - Payload Surface Core Algorithms |
| `pegasus/patent-dossier/text-architecture/tools/patent-architect/batch-02c-payload-surfaces/patent-feature-catalog.md` | 39 | 1590 | Technical Patent Architecture: Batch 02C - Payload Surfaces Feature Catalog |
| `pegasus/patent-dossier/text-architecture/tools/patent-architect/batch-02d-backend-cross-role/patent-alternative-embodiments.md` | 37 | 1524 | Technical Patent Architecture: Batch 02D - Cross-Role Alternative Embodiments |
| `pegasus/patent-dossier/text-architecture/tools/patent-architect/batch-02d-backend-cross-role/patent-core-algorithms.md` | 62 | 3077 | Technical Patent Architecture: Batch 02D - Cross-Role Backend Core Algorithms |
| `pegasus/patent-dossier/text-architecture/tools/patent-architect/batch-02d-backend-cross-role/patent-feature-catalog.md` | 34 | 1319 | Technical Patent Architecture: Batch 02D - Backend and Cross-Role Feature Catalog |
| `pegasus/patent-dossier/text-architecture/tools/patent-architect/patent-feature-official-explanations.md` | 150 | 13102 | Technical Patent Architecture: Patent Feature Official Explanations |
| `pegasus/patent-dossier/tools/patent-architect/batch-01-supplier-core/patent-alternative-embodiments.md` | 40 | 8588 | Alternative Embodiment Permutation Matrix |
| `pegasus/patent-dossier/tools/patent-architect/batch-01-supplier-core/patent-core-algorithms.md` | 112 | 4414 | Patent Core Algorithms |
| `pegasus/patent-dossier/tools/patent-architect/batch-01-supplier-core/patent-feature-catalog.md` | 68 | 11965 | Comprehensive Feature Catalog |
| `pegasus/patent-dossier/tools/patent-architect/batch-02a-driver-mobile/patent-alternative-embodiments.md` | 17 | 1999 | Batch 02A - Driver Mobile Alternative Embodiments |
| `pegasus/patent-dossier/tools/patent-architect/batch-02a-driver-mobile/patent-core-algorithms.md` | 77 | 2853 | Batch 02A - Driver Mobile Core Algorithms |
| `pegasus/patent-dossier/tools/patent-architect/batch-02a-driver-mobile/patent-feature-catalog.md` | 37 | 3590 | Batch 02A - Driver Mobile Feature Catalog |
| `pegasus/patent-dossier/tools/patent-architect/batch-02b-retailer-multi/patent-alternative-embodiments.md` | 17 | 1906 | Batch 02B - Retailer Multi-Surface Alternative Embodiments |
| `pegasus/patent-dossier/tools/patent-architect/batch-02b-retailer-multi/patent-core-algorithms.md` | 67 | 2324 | Batch 02B - Retailer Multi-Surface Core Algorithms |
| `pegasus/patent-dossier/tools/patent-architect/batch-02b-retailer-multi/patent-feature-catalog.md` | 40 | 3757 | Batch 02B - Retailer Multi-Surface Feature Catalog |
| `pegasus/patent-dossier/tools/patent-architect/batch-02c-payload-surfaces/patent-alternative-embodiments.md` | 17 | 1826 | Batch 02C - Payload Surfaces Alternative Embodiments |
| `pegasus/patent-dossier/tools/patent-architect/batch-02c-payload-surfaces/patent-core-algorithms.md` | 66 | 2224 | Batch 02C - Payload Surface Core Algorithms |
| `pegasus/patent-dossier/tools/patent-architect/batch-02c-payload-surfaces/patent-feature-catalog.md` | 33 | 3060 | Batch 02C - Payload Surfaces Feature Catalog |
| `pegasus/patent-dossier/tools/patent-architect/batch-02d-backend-cross-role/patent-alternative-embodiments.md` | 17 | 2089 | Batch 02D - Cross-Role Alternative Embodiments |
| `pegasus/patent-dossier/tools/patent-architect/batch-02d-backend-cross-role/patent-core-algorithms.md` | 69 | 2540 | Batch 02D - Cross-Role Backend Core Algorithms |
| `pegasus/patent-dossier/tools/patent-architect/batch-02d-backend-cross-role/patent-feature-catalog.md` | 41 | 4328 | Batch 02D - Backend and Cross-Role Feature Catalog |
| `pegasus/patent-dossier/tools/patent-architect/patent-feature-official-explanations.md` | 1898 | 128210 | Patent Feature Official Explanations |
| `pegasus/services/deep-agents/README.md` | 174 | 5559 | Deep Agents (LangChain) — PegasusX ecosystem quality |
| `pegasus/services/deep-agents/skills/architecture/SKILL.md` | 32 | 966 | Architecture |
| `pegasus/services/deep-agents/skills/backend-mutations/SKILL.md` | 30 | 1054 | Backend mutations |
| `pegasus/services/deep-agents/skills/business-logic/SKILL.md` | 77 | 4046 | Business logic |
| `pegasus/services/deep-agents/skills/cloud-infra/SKILL.md` | 30 | 886 | Cloud / infra |
| `pegasus/services/deep-agents/skills/code-quality/SKILL.md` | 29 | 941 | Code quality |
| `pegasus/services/deep-agents/skills/data-flow-coverage/SKILL.md` | 38 | 1449 | Data-flow coverage |
| `pegasus/services/deep-agents/skills/kafka-outbox/SKILL.md` | 50 | 2084 | Kafka + outbox |
| `pegasus/services/deep-agents/skills/money-fiscal/SKILL.md` | 53 | 2125 | Money & fiscal |
| `pegasus/services/deep-agents/skills/redis-cache/SKILL.md` | 26 | 899 | Redis |
| `pegasus/services/deep-agents/skills/regulatory-gov/SKILL.md` | 50 | 2543 | Regulatory & gov / trade APIs |
| `pegasus/services/deep-agents/skills/role-row-clients/SKILL.md` | 80 | 3698 | Role-row clients + feature parity |
| `pegasus/services/deep-agents/skills/security-tenancy/SKILL.md` | 30 | 937 | Security & tenancy |
| `pegasus/services/deep-agents/skills/void-overview/SKILL.md` | 20 | 795 | V.O.I.D monorepo overview |
| `pegasus/services/optimizer-core/README.md` | 154 | 4384 | OR-Tools Optimizer Core (Pattern A Sidecar) |
| `pegasus/tests/cross-role/sprint1_execution_gate_evidence.md` | 30 | 921 | Sprint-1 Execution Gate Evidence |

### pegasusX / Core Docs & Specifications (137 files)

| File Path | Lines | Size (Bytes) | Header / Title |
|---|---|---|---|
| `pegasusX/docs/ADDITIONAL_ECOSYSTEM_SOLUTIONS_ROADMAP_2026-08-19.md` | 498 | 14553 | Additional V.O.I.D. Ecosystem Solutions Roadmap |
| `pegasusX/docs/AI_WORKER_LAUNCH_RUNBOOK.md` | 6 | 404 | AI_WORKER_LAUNCH_RUNBOOK |
| `pegasusX/docs/AUTO_DISPATCH_IMPROVEMENT_PLAN.md` | 241 | 10409 | Auto-dispatch — assessment + ordered improvement plan |
| `pegasusX/docs/AUTO_ORDER.md` | 164 | 7182 | Auto-order — E2E wiring SoT |
| `pegasusX/docs/AUTO_ORDER_PLACE_FLIP.md` | 36 | 1725 | Auto-order `place` flip criteria (Phase 4) |
| `pegasusX/docs/BACKEND_SYSTEM_DESIGN_AUDIT.md` | 124 | 5191 | Backend programming + system design — audit |
| `pegasusX/docs/BARCODE_GO_LIVE_CHECKLIST.md` | 6 | 405 | BARCODE_GO_LIVE_CHECKLIST |
| `pegasusX/docs/BILLING_RECOVERY_SCRIPT.md` | 6 | 403 | BILLING_RECOVERY_SCRIPT |
| `pegasusX/docs/CLAIM_ROLE_ROW.md` | 45 | 2097 | Claims role-row (logistics OS&D + chargebacks) |
| `pegasusX/docs/CLAIM_STORE_STOCK_BRIDGE.md` | 24 | 1090 | Claim → Store Stock Quarantine Bridge (L7) |
| `pegasusX/docs/CLOUD_CREDENTIALS_CHECKLIST.md` | 76 | 5655 | Cloud Credentials Checklist |
| `pegasusX/docs/COMMERCEML_EXCHANGE.md` | 72 | 2456 | CommerceML 2.x exchange package (Phase 2 design) |
| `pegasusX/docs/CREDIT_ECOSYSTEM_BEHAVIOR.md` | 100 | 5628 | Credit Ecosystem Behavior |
| `pegasusX/docs/CREDIT_SCORING_V1.md` | 35 | 1329 | Credit risk scoring v1 (G3.B) |
| `pegasusX/docs/DATA_FLOW_AS_IMPLEMENTED.md` | 196 | 7413 | PegasusX Data Flow — As Implemented |
| `pegasusX/docs/DELIVERY_ESCALATION_POLICY.md` | 6 | 406 | DELIVERY_ESCALATION_POLICY |
| `pegasusX/docs/DEMAND_CLASS_IBP_SLICE.md` | 117 | 9089 | Demand-class honesty (o9 map, one slice) |
| `pegasusX/docs/DEPLOYMENT_AND_DISTRIBUTION_PLAN.md` | 6 | 412 | DEPLOYMENT_AND_DISTRIBUTION_PLAN |
| `pegasusX/docs/DEPLOYMENT_READINESS_GAP_LEDGER.md` | 34 | 1441 | DEPLOYMENT_READINESS_GAP_LEDGER |
| `pegasusX/docs/DEVOPS_CICD_AUDIT.md` | 129 | 5266 | DevOps + CI/CD — audit |
| `pegasusX/docs/DISPUTE_CLASSIFICATION_VOCABULARY.md` | 6 | 413 | DISPUTE_CLASSIFICATION_VOCABULARY |
| `pegasusX/docs/DOCS_CODE_ALIGNMENT_STATUS_2026-08-12.docx` | 1 | 8850 | PegasusX — Docs↔Code Alignment Status Date: 2026-08-12 · Living Word export. Mar... |
| `pegasusX/docs/DOCS_SOURCE_OF_TRUTH.md` | 84 | 7214 | Documentation source of truth (living vs frozen) |
| `pegasusX/docs/DRIVER_SUPPORT_PLAYBOOK.md` | 6 | 403 | DRIVER_SUPPORT_PLAYBOOK |
| `pegasusX/docs/E2_PER_SUPPLIER_PERIMETER_DESIGN.md` | 65 | 2713 | E2 — Per-supplier delivery perimeter (design) |
| `pegasusX/docs/ECOSYSTEM_FEATURES_BY_ROLE.md` | 669 | 28432 | PegasusX Ecosystem — Features by Role (Deep Reference) |
| `pegasusX/docs/ECOSYSTEM_FEATURE_IMPLEMENTATION_PLAN.md` | 364 | 22394 | V.O.I.D. Ecosystem Feature Implementation Plan |
| `pegasusX/docs/ECOSYSTEM_HARDENING_GAP_PLAN.md` | 422 | 17209 | Ecosystem Hardening Gap Plan (Beyond Retail OS / Next-Layer / Claims) |
| `pegasusX/docs/FEATURES_BY_APP_ROLE.md` | 354 | 33557 | PegasusX — Features by App / Role |
| `pegasusX/docs/FINANCE_SUPPORT_WORKFLOW.md` | 6 | 404 | FINANCE_SUPPORT_WORKFLOW |
| `pegasusX/docs/FIREBASE_AUDIT.md` | 116 | 8916 | Firebase + App Hosting — code audit |
| `pegasusX/docs/FISCAL_EDS_PROOF.md` | 49 | 2031 | Fiscal EDS proof (MY_SOLIQ / Soliq OFD) |
| `pegasusX/docs/FORECAST_ALGO.md` | 61 | 3381 | Forecast algorithm (§8.1) |
| `pegasusX/docs/FX_RATES.md` | 91 | 4229 | FX rates (theatre #13) |
| `pegasusX/docs/GATE0_CI.md` | 30 | 1119 | Gate-0 CI for pegasusX (lint / race / secrets / 12 native apps). |
| `pegasusX/docs/GCP_MANAGED_KAFKA.md` | 45 | 1749 | GCP Managed Service for Apache Kafka (HA event backbone) |
| `pegasusX/docs/GCP_MIGRATION_CHECKLIST.md` | 24 | 2047 | GCP Migration & Wire-Ready Checklist |
| `pegasusX/docs/GLOBAL_PAY_REFUND_PROOF.md` | 37 | 1526 | Global Pay refund (`RF`) proof |
| `pegasusX/docs/GLOBAL_SCALE_BACKEND_FEATURES.md` | 670 | 66300 | PegasusX — Every backend feature, classified for global scale |
| `pegasusX/docs/GLOBAL_SCALE_BACKEND_INFRA.md` | 270 | 18944 | PegasusX — Enterprise global plan (backend + infra) |
| `pegasusX/docs/GLOBAL_SCALE_CLIENT_UI.md` | 1362 | 76617 | GS-U — Client visualization program (all role apps) |
| `pegasusX/docs/GLOBAL_SCALE_LOCAL_ECOSYSTEM.md` | 691 | 45089 | GS-L / GS-K — Local multi-supplier ecosystem (backend + infra) |
| `pegasusX/docs/GLOBAL_SCALE_PROGRAM.md` | 285 | 17968 | PegasusX — Global-scale enterprise program |
| `pegasusX/docs/GS1_LABELS.md` | 55 | 2374 | GS1 labels (Gate-3 Wave 2C) |
| `pegasusX/docs/INCIDENT_RESPONSE_RUNBOOK.md` | 6 | 405 | INCIDENT_RESPONSE_RUNBOOK |
| `pegasusX/docs/INFRA_AUDIT.md` | 107 | 5162 | Infra (Terraform / GKE / K8s / cells) — audit |
| `pegasusX/docs/JWT_CORE_OPENAPI.md` | 31 | 1245 | JWT Core OpenAPI |
| `pegasusX/docs/KAFKA_AUDIT.md` | 136 | 7329 | Kafka + outbox + consumers — code audit |
| `pegasusX/docs/L1_FIELD_UNLOCK_RELEASE_CHECKLIST.md` | 24 | 1050 | L1 Field Unlock — Release Checklist |
| `pegasusX/docs/LAUNCH_READINESS_RUNBOOK.md` | 14 | 346 | Launch Readiness Runbook |
| `pegasusX/docs/LAYER_B_ECOSYSTEM_READINESS_PLAN.md` | 325 | 16777 | Layer B ecosystem readiness — phased modular plan |
| `pegasusX/docs/LAYER_B_SANDBOX_READINESS.md` | 62 | 2911 | Layer B readiness + sandbox (ops sequence) |
| `pegasusX/docs/LIVE_TRACKING_EXPECTATIONS.md` | 6 | 406 | LIVE_TRACKING_EXPECTATIONS |
| `pegasusX/docs/MANIFEST_DUAL_PLANE.md` | 36 | 1415 | Manifest dual plane (G2.D Option B) |
| `pegasusX/docs/MAPS_AUDIT.md` | 311 | 20990 | Maps / geography / display — code audit |
| `pegasusX/docs/MOBILE_SHARED_KIT.md` | 21 | 1416 | §8.8 Mobile shared kit + offline queue |
| `pegasusX/docs/MULTI_TENANCY_GATE5_PHASE1.md` | 293 | 15002 | ADR: Gate 5 / §8.10 Phase 1 — Request-scoped multi-tenancy |
| `pegasusX/docs/MULTI_TENANCY_GATE5_PHASE2.md` | 54 | 2958 | ADR: Gate 5 / §8.10 Phase 2 — Multi-supplier cart / ParentOrders |
| `pegasusX/docs/MULTI_TENANCY_GATE5_PHASE3.md` | 56 | 2930 | ADR: Gate 5 / §8.10 Phase 3 — GlobalProducts master |
| `pegasusX/docs/NEXT_LAYER_ECOSYSTEM_PLAN.md` | 1135 | 43100 | Next-Layer Ecosystem Plan (Post Retail OS Phases 0–5) |
| `pegasusX/docs/OPTIMIZER_AND_ROUTING_RUNTIME.md` | 153 | 7901 | Optimizer & routing — runtime SoT (aligned to codebase) |
| `pegasusX/docs/ORDER_FLOW_AND_EDGE_CASES.md` | 308 | 12232 | PegasusX — Order Flow & Edge Cases (CODE) |
| `pegasusX/docs/P0_LAUNCH_CHECKLIST.md` | 8 | 234 | P0 Launch Checklist |
| `pegasusX/docs/P1_PILOT_CHECKLIST.md` | 6 | 398 | P1 Pilot Checklist |
| `pegasusX/docs/P2_SCALE_ROADMAP.md` | 6 | 396 | P2 Scale Roadmap |
| `pegasusX/docs/PARTIAL_DISPATCH_RECOVERY_SOP.md` | 6 | 409 | PARTIAL_DISPATCH_RECOVERY_SOP |
| `pegasusX/docs/PARTNER_ADAPTER_1C.md` | 30 | 914 | 1C adapter pack (G5.B) |
| `pegasusX/docs/PARTNER_API.md` | 145 | 7565 | Partner Integration Layer (Gate 3 / §8.9) |
| `pegasusX/docs/PARTNER_AS2.md` | 64 | 2462 | Partner AS2 transport (§8.9) |
| `pegasusX/docs/PARTNER_EDI.md` | 106 | 4671 | Partner EDI-lite (Gate 3 / §8.9 Wave 2B) |
| `pegasusX/docs/PARTNER_JOURNALS_1C.md` | 64 | 2101 | Partner journals export (1C-friendly) |
| `pegasusX/docs/PARTNER_MASTERDATA.md` | 25 | 751 | Partner master-data sync (G5.C) |
| `pegasusX/docs/PARTNER_WMS_ASN.md` | 25 | 644 | External WMS ASN (G5.D) |
| `pegasusX/docs/PAYMENT_EXCEPTION_SOP.md` | 6 | 401 | PAYMENT_EXCEPTION_SOP |
| `pegasusX/docs/PAYMENT_SPLIT_AND_SETTLEMENT_IMPLEMENTATION_PLAN.md` | 401 | 18839 | Payment Split and Settlement Implementation Plan |
| `pegasusX/docs/PAYOUT_BANK_FILE_RUNBOOK.md` | 30 | 1355 | Payout bank-file runbook (G1.D) |
| `pegasusX/docs/PAYOUT_RAIL_DECISION.md` | 43 | 2028 | Payout rail decision |
| `pegasusX/docs/PEGASUSX_MASTER_ROADMAP.md` | 203 | 10059 | PegasusX Master Roadmap (Everything, Sequenced) |
| `pegasusX/docs/PLANOGRAM_VISION_PLAN.md` | 502 | 17469 | Planogram & Shelf Vision Plan |
| `pegasusX/docs/PLATFORM_SLOS.md` | 21 | 1123 | Platform SLOs (Phase 3) |
| `pegasusX/docs/PRICING_AUTHORITY_RULES.md` | 6 | 403 | PRICING_AUTHORITY_RULES |
| `pegasusX/docs/PROD_ECOSYSTEM_GOAL.md` | 101 | 6412 | PegasusX — Production Ecosystem Goal |
| `pegasusX/docs/PROD_READINESS_SEQUENCE.md` | 138 | 6485 | PegasusX — Enterprise Prod-Readiness Sequence (post W0–W5) |
| `pegasusX/docs/PegasusX_o9_Demand_Planning_Problems_Extraction.md` | 247 | 14693 | PegasusX — Demand Planning Problems & Logistics Challenges Extraction |
| `pegasusX/docs/PegasusX_o9_Digital_Brain_Feature_Extraction_Integration_Blueprint.md` | 415 | 23947 | PegasusX — o9 Digital Brain Feature Extraction & Integration Blueprint |
| `pegasusX/docs/REAL_WORLD_CASE_MATRIX.md` | 57 | 3119 | REAL_WORLD_CASE_MATRIX |
| `pegasusX/docs/REASSIGNMENT_SUPPORT_PLAYBOOK.md` | 6 | 409 | REASSIGNMENT_SUPPORT_PLAYBOOK |
| `pegasusX/docs/REDIS_AUDIT.md` | 106 | 5852 | Redis — cache / hot location / perimeter — code audit |
| `pegasusX/docs/RELEASE_TRAIN.md` | 6 | 393 | RELEASE_TRAIN |
| `pegasusX/docs/RETAILER_ASSIST.md` | 31 | 820 | Retailer Floor Assist (Retail OS Phase 6) |
| `pegasusX/docs/RETAILER_CAPABILITY_PACKS.md` | 136 | 4712 | Retailer capability packs (Retail OS Phase 0) |
| `pegasusX/docs/RETAILER_LOCAL_SKU.md` | 32 | 819 | Retailer Local / Manual POS SKUs (L6) |
| `pegasusX/docs/RETAILER_ONBOARDING_SUPPORT_FLOWS.md` | 6 | 413 | RETAILER_ONBOARDING_SUPPORT_FLOWS |
| `pegasusX/docs/RETAILER_OS_CLOSEOUT.md` | 83 | 3046 | Retail OS close-out (v1) |
| `pegasusX/docs/RETAILER_OS_E2E_MATRIX.md` | 52 | 3455 | Retail OS E2E / parity matrix (Phase 7) |
| `pegasusX/docs/RETAILER_OS_PRODUCTION_GATE.md` | 58 | 2397 | Retail OS production gate (Phase 7) |
| `pegasusX/docs/RETAILER_POS.md` | 82 | 2628 | Retailer POS (Retail OS Phase 4 + Offline) |
| `pegasusX/docs/RETAILER_RECEIVE_STOCK_CLAIMS_PLAN.md` | 846 | 41624 | Rich Receiver ↔ Store Stock ↔ Claims Merge Plan |
| `pegasusX/docs/RETAILER_RECEIVING_WINDOWS_GUIDE.md` | 6 | 412 | RETAILER_RECEIVING_WINDOWS_GUIDE |
| `pegasusX/docs/RETAILER_REPORTS_PRO.md` | 25 | 775 | Retailer Reports Pro (Retail OS Phase 6) |
| `pegasusX/docs/RETAILER_SECTIONS.md` | 31 | 807 | Retailer Sections (Retail OS Phase 6) |
| `pegasusX/docs/RETAILER_SELL_THROUGH.md` | 228 | 9069 | Retailer sell-through flywheel (Next Layer L3) |
| `pegasusX/docs/RETAILER_SHIFTS.md` | 74 | 2230 | Retailer Shifts & Time (Retail OS Phase 5) |
| `pegasusX/docs/RETAILER_STORE_STOCK.md` | 49 | 1595 | Retailer store stock (Retail OS Phase 3) |
| `pegasusX/docs/ROLE_CAPABILITIES_MATH_LOGIC.md` | 261 | 14150 | PegasusX — Capabilities: Math, Logic, Algorithms (CODE) |
| `pegasusX/docs/ROLE_FEATURES_DOCS_VS_CODE.md` | 630 | 45230 | PegasusX — Role features: docs vs code |
| `pegasusX/docs/ROLE_ROW_BUSINESS_LOGIC_AUDIT_2026-08-19.md` | 347 | 19337 | V.O.I.D. Role-Row Business Logic Audit |
| `pegasusX/docs/ROLE_ROW_PARITY_MATRIX.md` | 51 | 4776 | pegasusX Role-Row Parity Matrix |
| `pegasusX/docs/SAFETY_STOCK.md` | 108 | 4225 | Safety stock (§8.2) |
| `pegasusX/docs/SEARCH_DECISION.md` | 25 | 1362 | Search Decision (W1 — 2026-08-12) |
| `pegasusX/docs/SHOP_CLOSED_E2E_SOP.md` | 6 | 399 | SHOP_CLOSED_E2E_SOP |
| `pegasusX/docs/SMART_DISPATCH_ARCHITECTURE_2026-08-19.md` | 230 | 6230 | PegasusX Smart Dispatch Architecture |
| `pegasusX/docs/SOLIQ_SANDBOX_READINESS.md` | 42 | 1950 | L5 Soliq OFD — Sandbox Readiness (Delivery-only) |
| `pegasusX/docs/SPANNER_HOT_PATH_REVIEW.md` | 6 | 403 | Spanner Hot Path Review |
| `pegasusX/docs/SSMR_RETAIL_OS_CLOUD_APPLY.md` | 116 | 4244 | SSMR cloud apply — Retail OS + infra readiness (2026-08-02) |
| `pegasusX/docs/SUBSTANCE_GATE.md` | 389 | 20699 | Substance Gate — E2E verification for every role, app, and platform |
| `pegasusX/docs/SUPPLIER_ONBOARDING_SOP.md` | 6 | 403 | SUPPLIER_ONBOARDING_SOP |
| `pegasusX/docs/SUPPLY_CHAIN_TRANSFORMATION_INSIGHTS.md` | 104 | 8698 | PegasusX — Exhaustive Supply Chain Transformation Insights |
| `pegasusX/docs/SURFACE_AUDITS.md` | 55 | 4097 | Surface audits — agent index (2026-08-18) |
| `pegasusX/docs/TENANCY_ENFORCEMENT.md` | 35 | 1094 | Tenancy enforcement (G4.A) |
| `pegasusX/docs/TOPOLOGY_ENTRY_SUPPORT_GUIDE.md` | 6 | 408 | TOPOLOGY_ENTRY_SUPPORT_GUIDE |
| `pegasusX/docs/TRANSFER_CANCELLATION_RUNBOOK.md` | 6 | 409 | TRANSFER_CANCELLATION_RUNBOOK |
| `pegasusX/docs/UI_SURFACE_AUDIT.md` | 114 | 5607 | UI surfaces (web / desktop / native) — audit |
| `pegasusX/docs/V1_STAGING_CLOSURE_CHECKLIST.md` | 6 | 408 | V1 Staging Closure Checklist |
| `pegasusX/docs/WAREHOUSE_EXCEPTION_SOP.md` | 6 | 403 | WAREHOUSE_EXCEPTION_SOP |
| `pegasusX/docs/WAVE_C_ENTERPRISE_SCALE_PLAN.md` | 377 | 17038 | Wave C — Enterprise-scale design (L8–L11) for prod readiness |
| `pegasusX/docs/WAVE_C_IMPLEMENTATION_PLAN.md` | 517 | 16343 | Wave C — Agent-executable implementation plan |
| `pegasusX/docs/WIRE_READY_STAGING_RUNBOOK.md` | 6 | 406 | Wire Ready Staging Runbook |
| `pegasusX/docs/WMS_COLD_CHAIN.md` | 36 | 1540 | WMS cold chain (§8.7 Gate 4 PR-6 + theatre #12 breach) |
| `pegasusX/docs/WMS_CYCLE_COUNTS.md` | 28 | 1003 | WMS cycle counts (§8.7 Wave 1C + Gate 4 PR-4) |
| `pegasusX/docs/WMS_GATE4_HARDENING.md` | 21 | 1199 | Gate 4 WMS — scanning throughput notes (§8.7 / §8.8) |
| `pegasusX/docs/WMS_GATE4_OPS.md` | 63 | 2710 | Gate 4 — WMS DDL + flag ops checklist |
| `pegasusX/docs/WMS_LOTS_FEFO.md` | 56 | 2817 | WMS lots + FEFO (§8.7 Wave 1A) |
| `pegasusX/docs/WMS_PICK_WAVES.md` | 54 | 2851 | WMS pick waves + seal gate (§8.7 Wave 1B) |
| `pegasusX/docs/ZONE_MISS_COMMUNICATION_POLICY.md` | 6 | 410 | ZONE_MISS_COMMUNICATION_POLICY |
| `pegasusX/docs/agents/README.md` | 126 | 4993 | PegasusX — LangChain / Deep Agents ecosystem audit orchestra |
| `pegasusX/docs/stock_acceptance_policy.md` | 84 | 3855 | Stock acceptance policy (out-of-stock behaviour) |

### pegasusX / Artifacts & Snapshots (70 files)

| File Path | Lines | Size (Bytes) | Header / Title |
|---|---|---|---|
| `pegasusX/artifacts/AGENT_WIPEOUT_10_10_PROGRESS.md` | 48 | 2189 | Agent wipe-out 10/10 — implementation progress |
| `pegasusX/artifacts/CLAIM_CHARGEBACK_FLOW.md` | 127 | 5691 | Post-delivery claims & supplier chargebacks |
| `pegasusX/artifacts/D3_APPLY_FROM_IDE.md` | 40 | 1306 | D3 schema — apply from IDE (gcloud) |
| `pegasusX/artifacts/DOCS_CODE_ALIGNMENT_STATUS_2026-08-12.docx` | 1 | 8850 | PegasusX — Docs↔Code Alignment Status Date: 2026-08-12 · Living Word export. Mar... |
| `pegasusX/artifacts/GAP_CLOSURE_BUILD_RESULT_2026-08-01.md` | 109 | 4268 | Gap closure build result — 2026-08-01 |
| `pegasusX/artifacts/GAP_CLOSURE_CLOUD_CHECKLIST_2026-07-31.md` | 35 | 1434 | Gap Closure — Cloud / SSMR Checklist (2026-07-31) |
| `pegasusX/artifacts/GATE0_BATCH_B_GCP_2026-08-05.md` | 25 | 1537 | Gate-0 Batch B — GCP / Spanner / prod images / OSRM |
| `pegasusX/artifacts/GATE0_BATCH_C_CREDS_2026-08-05.md` | 53 | 2751 | Gate-0 Batch C — credentials / Firebase / Maps / Global Pay |
| `pegasusX/artifacts/GATE0_BLAST_RADIUS_2026-08-05.md` | 19 | 1609 | Gate-0 Track A — Blast radius (preflight) |
| `pegasusX/artifacts/GATE0_SPANNER_BACKUP_RESTORE_2026-08-05.md` | 43 | 1944 | Gate-0 — Spanner backup / PITR / restore rehearsal |
| `pegasusX/artifacts/GCP_SUPPORT_CASE_QUOTA.md` | 92 | 3943 | GCP Support case — SSD + IP quota (copy/paste) |
| `pegasusX/artifacts/GOOGLE_ROUTES_WORLD_SCALE_2026-08-05.md` | 51 | 2835 | Google Routes world-scale maps — wiring closeout |
| `pegasusX/artifacts/GS_C4_ISOLATION_PROOF.md` | 13 | 1090 | GS-C4 isolation proof |
| `pegasusX/artifacts/GS_C5_CELL_API_PROOF.md` | 13 | 955 | GS-C5 cell API / global DNS proof |
| `pegasusX/artifacts/GS_P_PARTNER_DIALECT_PROOF.md` | 16 | 939 | GS-P partner dialect proof |
| `pegasusX/artifacts/GS_R_PACK_CLIENT_PROOF.md` | 14 | 833 | GS-R pack client bind proof |
| `pegasusX/artifacts/OWNER_SECRETS_HANDOFF_2026-08-01.md` | 54 | 3413 | Owner secrets / ops handoff — 2026-08-01 (updated 2026-08-02) |
| `pegasusX/artifacts/PROD_WIRING_AND_THIRD_PARTIES.md` | 163 | 8456 | Production wiring status + third-party inventory |
| `pegasusX/artifacts/PegasusX_Ecosystem_Status_Report.md` | 107 | 5071 | PegasusX Ecosystem Status Report |
| `pegasusX/artifacts/PegasusX_End_Product_Reality_Report.docx` | 1 | 21018 | HISTORICAL / FROZEN EXPORT — Do not plan from this .docx alone. Current SoT: doc... |
| `pegasusX/artifacts/PegasusX_End_Product_Reality_Report_2026-08-13.docx` | 1 | 62948 | PegasusX / ATOMOS End-Product Reality Report What the system actually is today —... |
| `pegasusX/artifacts/PegasusX_O9-1_Segmentation_Constrained_Allocation_Plan.md` | 37 | 1487 | O9-1 — Segmentation + Constrained Allocation (extract) |
| `pegasusX/artifacts/PegasusX_O9_Gap_Closure_Implementation_Plan.md` | 404 | 20237 | PegasusX — O9-Style Planning Capabilities |
| `pegasusX/artifacts/README_DOCX.md` | 7 | 377 | Frozen Word export |
| `pegasusX/artifacts/ROLE_ROW_PARITY_MATRIX_SNAPSHOT_2026-07-07.md` | 347 | 39474 | pegasusX Role-Row Parity Matrix |
| `pegasusX/artifacts/SUBSTANCE_GATE_API_SIGNOFF_2026-08-04.md` | 58 | 2835 | Substance Gate — API sign-off (2026-08-04) |
| `pegasusX/artifacts/SUBSTANCE_GATE_CLIENT_SIGNOFF_2026-08-05.md` | 61 | 2793 | Substance Gate — Client sign-off (2026-08-05) |
| `pegasusX/artifacts/cartograph/README.md` | 15 | 556 | PegasusX Cartograph |
| `pegasusX/artifacts/d4-redis-prove-2026-07-20.md` | 66 | 1417 | D4 Memorystore Redis — 2026-07-20 |
| `pegasusX/artifacts/d5-kafka-confluent-runbook.md` | 147 | 4589 | D5 — Managed Kafka (Confluent Cloud) runbook |
| `pegasusX/artifacts/forecast-shadow/README.md` | 4 | 185 | Forecast-shadow soak evidence for AUTO_ORDER place flip. |
| `pegasusX/artifacts/load/20260604-154622/LOAD_TEST_REPORT.md` | 27 | 1001 | pegasusX load certification report |
| `pegasusX/artifacts/load/20260604-154852/LOAD_TEST_REPORT.md` | 27 | 1000 | pegasusX load certification report |
| `pegasusX/artifacts/load/20260604-155311/LOAD_TEST_REPORT.md` | 27 | 1000 | pegasusX load certification report |
| `pegasusX/artifacts/load/20260604-155357/LOAD_TEST_REPORT.md` | 27 | 1000 | pegasusX load certification report |
| `pegasusX/artifacts/load/20260604-160440/LOAD_TEST_REPORT.md` | 28 | 1139 | pegasusX load certification report |
| `pegasusX/artifacts/load/20260604-174408/LOAD_TEST_REPORT.md` | 32 | 1331 | pegasusX load certification report |
| `pegasusX/artifacts/load/20260604-175632/LOAD_TEST_REPORT.md` | 33 | 1380 | pegasusX load certification report |
| `pegasusX/artifacts/load/20260604-181132/LOAD_TEST_REPORT.md` | 32 | 1383 | pegasusX load certification report |
| `pegasusX/artifacts/load/20260604-181832/LOAD_TEST_REPORT.md` | 29 | 1184 | pegasusX load certification report |
| `pegasusX/artifacts/load/20260617-180124/LOAD_TEST_REPORT.md` | 31 | 1256 | pegasusX load certification report |
| `pegasusX/artifacts/load/20260617-180549/LOAD_TEST_REPORT.md` | 31 | 1256 | pegasusX load certification report |
| `pegasusX/artifacts/load/20260617-181207/LOAD_TEST_REPORT.md` | 31 | 1275 | pegasusX load certification report |
| `pegasusX/artifacts/load/20260617-183441/LOAD_TEST_REPORT.md` | 32 | 1324 | pegasusX load certification report |
| `pegasusX/artifacts/load/20260617-183643/LOAD_TEST_REPORT.md` | 32 | 1308 | pegasusX load certification report |
| `pegasusX/artifacts/load/20260617-184510/LOAD_TEST_REPORT.md` | 33 | 1377 | pegasusX load certification report |
| `pegasusX/artifacts/load/20260617-184512/LOAD_TEST_REPORT.md` | 33 | 1361 | pegasusX load certification report |
| `pegasusX/artifacts/next-layer-remaining-baseline-2026-08-04.md` | 35 | 1446 | Next-Layer Remaining — Baseline (2026-08-04) |
| `pegasusX/artifacts/prod-cli-wiring-2026-07-28.md` | 97 | 3604 | Prod API / DB / server / DevOps wiring via CLI |
| `pegasusX/artifacts/receipts-multi-provider.md` | 90 | 4281 | Multi-layer receipts (no Soliq required for now) |
| `pegasusX/artifacts/ssmr-auto-order-place-smoke-2026-08-02.md` | 60 | 2839 | SSMR smoke: Auto-order place + L3 DDL (2026-08-02) |
| `pegasusX/artifacts/ssmr-e2e-portal-flywheel-2026-08-04.md` | 83 | 3151 | SSMR full e2e + supplier portal flywheel deploy — 2026-08-04 |
| `pegasusX/artifacts/ssmr-negotiation-isolation-2026-08-02.md` | 57 | 2185 | SSMR negotiation isolation check — 2026-08-02 |
| `pegasusX/artifacts/ssmr-next-layer-rollout-2026-08-04.md` | 51 | 2207 | SSMR Next-Layer Remaining Rollout — 2026-08-04 |
| `pegasusX/artifacts/ssmr-wave-ab-rollout-2026-08-03.md` | 58 | 2181 | SSMR Wave A/B backend image roll — 2026-08-03 |
| `pegasusX/artifacts/ssmr-wave-c-rollout-2026-08-04.md` | 100 | 4486 | SSMR Wave C C3.3/C4.1 backend roll — 2026-08-04 |
| `pegasusX/artifacts/step10-smoke-ssmr.md` | 80 | 3237 | Step 10 — Cloud smoke (SSMR) — PASS |
| `pegasusX/artifacts/step11-ingress-ssmr.md` | 98 | 3239 | Step 11 — Ingress + DNS + TLS (SSMR) |
| `pegasusX/artifacts/step12-firebase-ssmr.md` | 101 | 3642 | Step 12 — Firebase phone OTP + FCM (SSMR) |
| `pegasusX/artifacts/step13-maps-ssmr.md` | 90 | 2782 | Step 13 — Maps API key + geocode (SSMR) |
| `pegasusX/artifacts/step14-globalpay-ssmr.md` | 118 | 4658 | Step 14 — Global Pay staging webhooks (SSMR) |
| `pegasusX/artifacts/step6-external-secrets-ssmr.md` | 31 | 1093 | Step 6 — External Secrets (SSMR) — DONE |
| `pegasusX/artifacts/step8-images-ssmr.md` | 38 | 1399 | Step 8 — amd64 images → Artifact Registry (SSMR) — DONE |
| `pegasusX/artifacts/step9-deploy-ssmr.md` | 53 | 2011 | Step 9 — K8s rollout (SSMR) — DONE |
| `pegasusX/artifacts/terraform-d2-apply-summary-2026-07-20.md` | 44 | 2228 | D2 Terraform apply summary — 2026-07-20 |
| `pegasusX/artifacts/terraform-d2-apply-summary-pegasus-503013.md` | 35 | 1379 | D2 apply summary — pegasus-503013 — 2026-07-20 |
| `pegasusX/artifacts/terraform-d2-plan-review-pegasus-503013.md` | 44 | 1451 | Terraform D2 plan — pegasus-503013 — 2026-07-20 |
| `pegasusX/artifacts/terraform-d2-plan-review.md` | 71 | 2576 | Terraform D2 plan review — 2026-07-20 |
| `pegasusX/artifacts/tfstate-archive/README.md` | 8 | 414 | Terraform state archives (historical void-494000 snapshots) are **not** kept in git. |
| `pegasusX/artifacts/wave-b-deploy-status-2026-07-21.md` | 22 | 1252 | Wave B deploy status (2026-07-21) |

### pegasusX / Big Platform Baseline (Deep Specs) (57 files)

| File Path | Lines | Size (Bytes) | Header / Title |
|---|---|---|---|
| `pegasusX/docs/big-platform-baseline/README.md` | 59 | 4034 | Big-Platform Baseline Plan (O9 / Blue Yonder / Manhattan / Kinaxis + PegasusX) |
| `pegasusX/docs/big-platform-baseline/collaboration/6.1-control-tower.md` | 23 | 528 | 6.1 Multi-Enterprise Control Tower |
| `pegasusX/docs/big-platform-baseline/collaboration/6.2-exception-rooms.md` | 15 | 552 | 6.2 Broadcast + Exception Rooms |
| `pegasusX/docs/big-platform-baseline/collaboration/6.3-supplier-scorecards.md` | 17 | 406 | 6.3 Supplier Scorecards |
| `pegasusX/docs/big-platform-baseline/collaboration/README.md` | 12 | 612 | 6. Collaboration, Network & Visibility |
| `pegasusX/docs/big-platform-baseline/differentiators/8.1-multi-supplier-cart-split-settlement.md` | 35 | 1279 | 8.1 Unified Multi-Supplier Cart + Delivery-Triggered Split Settlement |
| `pegasusX/docs/big-platform-baseline/differentiators/8.2-fiscal-aware-regime-versioning.md` | 25 | 843 | 8.2 Fiscal-Aware State Machine + Regime Versioning |
| `pegasusX/docs/big-platform-baseline/differentiators/8.3-offline-crypto-manifest.md` | 30 | 849 | 8.3 Offline-First Driver + Cryptographic Manifest Verification |
| `pegasusX/docs/big-platform-baseline/differentiators/8.4-shop-closed-economic-decision.md` | 21 | 565 | 8.4 Shop-Closed + Credit-Leave as First-Class Economic Decision |
| `pegasusX/docs/big-platform-baseline/differentiators/8.5-claim-order-line-pricing.md` | 27 | 835 | 8.5 Claim Pricing From Original Order Lines + Auto-Approve |
| `pegasusX/docs/big-platform-baseline/differentiators/8.6-payload-seal-role.md` | 20 | 590 | 8.6 Payload Seal as Explicit Role |
| `pegasusX/docs/big-platform-baseline/differentiators/8.7-durable-freezes.md` | 21 | 556 | 8.7 Dynamic Credit + Fiscal Freeze Across Partition |
| `pegasusX/docs/big-platform-baseline/differentiators/8.8-disruption-playbooks.md` | 33 | 787 | 8.8 Emerging-Market Disruption Playbooks as Code |
| `pegasusX/docs/big-platform-baseline/differentiators/README.md` | 17 | 1199 | 8. Differentiators — Solutions Incumbents Do Not Solve Cleanly |
| `pegasusX/docs/big-platform-baseline/execution/3.1-advanced-wms.md` | 32 | 1044 | 3.1 Advanced Warehouse Management |
| `pegasusX/docs/big-platform-baseline/execution/3.2-full-tms.md` | 33 | 1224 | 3.2 Full Transportation Management |
| `pegasusX/docs/big-platform-baseline/execution/3.3-labor-management.md` | 23 | 657 | 3.3 Labor Management |
| `pegasusX/docs/big-platform-baseline/execution/README.md` | 10 | 504 | 3. Execution Depth (Blue Yonder / Manhattan class) |
| `pegasusX/docs/big-platform-baseline/foundations/1.1-canonical-model-knowledge-graph.md` | 54 | 2506 | 1.1 Unified Canonical Model + Lightweight Knowledge Graph |
| `pegasusX/docs/big-platform-baseline/foundations/1.2-multi-horizon-planning.md` | 43 | 1831 | 1.2 Multi-Horizon Planning Engine |
| `pegasusX/docs/big-platform-baseline/foundations/1.3-continuous-intelligence.md` | 39 | 1278 | 1.3 Continuous Intelligence Layer |
| `pegasusX/docs/big-platform-baseline/foundations/1.4-labor-capacity.md` | 35 | 992 | 1.4 Labor & Capacity Model |
| `pegasusX/docs/big-platform-baseline/foundations/README.md` | 13 | 689 | 1. Architectural Foundations |
| `pegasusX/docs/big-platform-baseline/last-mile/4.1-offline-sync-protocol.md` | 35 | 1712 | Shop-Closed / Partial / Proximity — Offline Sync Protocol |
| `pegasusX/docs/big-platform-baseline/last-mile/4.1-shop-closed-protocol.md` | 60 | 2480 | 4.1 Shop-Closed / No-Answer Protocol (Enhanced) |
| `pegasusX/docs/big-platform-baseline/last-mile/4.2-partial-offload-resequence.md` | 40 | 1348 | 4.2 Partial Offload + Resequence |
| `pegasusX/docs/big-platform-baseline/last-mile/4.3-damage-missing-temperature.md` | 35 | 1033 | 4.3 Damage / Missing / Temperature Dual-Wire |
| `pegasusX/docs/big-platform-baseline/last-mile/4.4-rescue-capacity-sharing.md` | 30 | 860 | 4.4 Rescue & Dynamic Capacity Sharing |
| `pegasusX/docs/big-platform-baseline/last-mile/4.5-proximity-settlement.md` | 41 | 1331 | 4.5 Proximity Settlement |
| `pegasusX/docs/big-platform-baseline/last-mile/README.md` | 14 | 870 | 4. Last-Mile & Exception Mastery |
| `pegasusX/docs/big-platform-baseline/phases/README.md` | 20 | 677 | Implementation Phases |
| `pegasusX/docs/big-platform-baseline/phases/phase-1.md` | 38 | 1806 | Phase 1 (3–6 months) — Regulatory + Differentiation Must-Haves |
| `pegasusX/docs/big-platform-baseline/phases/phase-2.md` | 25 | 784 | Phase 2 — Planning Depth + WMS/TMS |
| `pegasusX/docs/big-platform-baseline/phases/phase-3.md` | 19 | 657 | Phase 3 — Network OS Scale |
| `pegasusX/docs/big-platform-baseline/planning/2.1-causal-demand-sensing.md` | 36 | 1125 | 2.1 Causal Demand Sensing + External Signals |
| `pegasusX/docs/big-platform-baseline/planning/2.2-meio.md` | 39 | 1234 | 2.2 Multi-Echelon Inventory Optimization (MEIO) |
| `pegasusX/docs/big-platform-baseline/planning/2.3-scenario-workbench.md` | 24 | 832 | 2.3 Scenario & What-If Workbench |
| `pegasusX/docs/big-platform-baseline/planning/2.4-supplier-collaboration.md` | 27 | 799 | 2.4 Supplier Collaboration Portal (multi-tier) |
| `pegasusX/docs/big-platform-baseline/planning/README.md` | 11 | 584 | 2. Planning & Intelligence (O9 / Kinaxis baseline) |
| `pegasusX/docs/big-platform-baseline/regulatory/README.md` | 29 | 1445 | Regulatory & Compliance Side |
| `pegasusX/docs/big-platform-baseline/regulatory/claim-settlement-insurance.md` | 20 | 644 | Claim Settlement + Insurance Trigger |
| `pegasusX/docs/big-platform-baseline/regulatory/compliance-audit-dashboard.md` | 23 | 707 | Compliance & Fiscal Audit Dashboard |
| `pegasusX/docs/big-platform-baseline/regulatory/credit-engine-compliance.md` | 26 | 821 | Credit Engine (Compliance-Facing) |
| `pegasusX/docs/big-platform-baseline/regulatory/integer-money-guarantees.md` | 28 | 906 | Integer Money Everywhere + Zero-Leak Guarantees |
| `pegasusX/docs/big-platform-baseline/regulatory/labor-hours-prep.md` | 19 | 534 | Labor Hours / Fatigue (Regulation Prep) |
| `pegasusX/docs/big-platform-baseline/regulatory/privacy-multi-tenant.md` | 30 | 1701 | Privacy & Multi-Tenant Isolation |
| `pegasusX/docs/big-platform-baseline/regulatory/soliq-ehf-integration.md` | 50 | 1622 | Soliq / EHF Integration (Regulatory) |
| `pegasusX/docs/big-platform-baseline/regulatory/tax-regime-versioning.md` | 34 | 952 | Versioned Tax & Regime Engine |
| `pegasusX/docs/big-platform-baseline/technical/README.md` | 17 | 929 | Technical Side |
| `pegasusX/docs/big-platform-baseline/technical/api-contracts-sketch.md` | 31 | 1508 | API Contracts Sketch |
| `pegasusX/docs/big-platform-baseline/technical/edge-case-matrix.md` | 23 | 1131 | Edge-Case Matrix Template |
| `pegasusX/docs/big-platform-baseline/technical/role-row-parity.md` | 21 | 832 | Role-Row Parity Obligations |
| `pegasusX/docs/big-platform-baseline/technical/schema-sketch.md` | 42 | 1805 | Schema Sketch (Spanner) |
| `pegasusX/docs/big-platform-baseline/technical/spine-laws.md` | 37 | 919 | Spine Laws (Non-Negotiable) |
| `pegasusX/docs/big-platform-baseline/technical/state-machines.md` | 55 | 1124 | State Machines |
| `pegasusX/docs/big-platform-baseline/technical/sustainability-risk-tech.md` | 24 | 744 | Sustainability & Risk (Technical Notes) |
| `pegasusX/docs/big-platform-baseline/technical/workers-kafka.md` | 43 | 1646 | Workers & Kafka Design |

### pegasusX / Session 2026-08-07 (Reality Reports & Gap Registers) (42 files)

| File Path | Lines | Size (Bytes) | Header / Title |
|---|---|---|---|
| `pegasusX/docs/session-2026-08-07/ANALYTICS_COLUMN_TENANCY_PROGRESS.md` | 42 | 2044 | Analytics column tenancy — progress (2026-08-11) |
| `pegasusX/docs/session-2026-08-07/DOMAIN2_AUTONOMY_PROGRESS.md` | 70 | 3853 | Domain 2 — Autonomy on evidence (P1) · Progress |
| `pegasusX/docs/session-2026-08-07/DOMAIN3_PLANNING_PROGRESS.md` | 64 | 3675 | Domain 3 — Planning: Forecast Accuracy Surface |
| `pegasusX/docs/session-2026-08-07/DOMAIN4_INTEGRATION_PROGRESS.md` | 67 | 3647 | Domain 4 — Integration (P1–P3) · Progress |
| `pegasusX/docs/session-2026-08-07/DOMAIN5_OPS_TRUTH_PROGRESS.md` | 63 | 3471 | Domain 5 — Tenancy & Ops Truth (P1) · Progress |
| `pegasusX/docs/session-2026-08-07/DOMAIN6_TENANT_AUDIT.md` | 54 | 3034 | Domain 6.2 — Per-Role Tenant Wiring Audit |
| `pegasusX/docs/session-2026-08-07/ECOSYSTEM_GAP_REGISTER_2026-08-12.md` | 197 | 28028 | PegasusX — Consolidated Ecosystem Gap Register & Data-Flow Blueprint |
| `pegasusX/docs/session-2026-08-07/END_PRODUCT_REALITY_REPORT.docx` | 1 | 86198 | HISTORICAL / FROZEN EXPORT — Do not plan from this .docx alone. Current SoT: doc... |
| `pegasusX/docs/session-2026-08-07/END_PRODUCT_REALITY_REPORT.md` | 603 | 88002 | 0. Executive Verdict |
| `pegasusX/docs/session-2026-08-07/END_PRODUCT_REALITY_REPORT_2026-08-11.docx` | 1 | 13068 | HISTORICAL / FROZEN EXPORT — Do not plan from this .docx alone. Current SoT: doc... |
| `pegasusX/docs/session-2026-08-07/END_PRODUCT_REALITY_REPORT_2026-08-11.md` | 191 | 16391 | PegasusX — End-Product Reality Report |
| `pegasusX/docs/session-2026-08-07/END_PRODUCT_REALITY_REPORT_2026-08-13.docx` | 1 | 62948 | PegasusX / ATOMOS End-Product Reality Report What the system actually is today —... |
| `pegasusX/docs/session-2026-08-07/ENTERPRISE_GRADE_EXECUTION_PLAN.md` | 142 | 15146 | PegasusX — Enterprise-Grade Execution Plan |
| `pegasusX/docs/session-2026-08-07/LIVE_MIGRATION_RUNBOOK_2026-08-11.md` | 36 | 1929 | Live Migration Runbook — Domain 1.3 (Tenancy NOT NULL + Payout Rail) |
| `pegasusX/docs/session-2026-08-07/MASTER_ALIGNMENT_DATAFLOW_2026-08-12.md` | 203 | 13791 | PegasusX — Master Alignment: Docs ↔ Code ↔ Data Flow |
| `pegasusX/docs/session-2026-08-07/NEXT_FORK_PICK.md` | 18 | 1053 | Fork pick — post Enterprise Phase 5 (2026-08-11) |
| `pegasusX/docs/session-2026-08-07/PHASE0_MONEY_PATH_PROGRESS.md` | 76 | 3802 | Enterprise Phase 0 — Money-path correctness |
| `pegasusX/docs/session-2026-08-07/PHASE1_MONEY_LAW_PROGRESS.md` | 61 | 4195 | Enterprise Phase 1 — Money and law |
| `pegasusX/docs/session-2026-08-07/PHASE2_COMPLETION.md` | 48 | 2330 | Phase 2 completion — Enterprise integration (2026-08-11) |
| `pegasusX/docs/session-2026-08-07/PHASE3_PROGRESS.md` | 45 | 2406 | Phase 3 progress — Operational truth (2026-08-11) |
| `pegasusX/docs/session-2026-08-07/PHASE4_COMPLETION.md` | 45 | 2254 | Phase 4 completion — Autonomy on evidence foundations (2026-08-11) |
| `pegasusX/docs/session-2026-08-07/PHASE5_PHASE2_PROGRESS.md` | 56 | 3758 | Gate 5 / §8.10 Phase 2 — Progress |
| `pegasusX/docs/session-2026-08-07/PHASE5_PHASE3_PROGRESS.md` | 44 | 2107 | Gate 5 / §8.10 Phase 3 — GlobalProducts Progress |
| `pegasusX/docs/session-2026-08-07/PHASE5_PROGRESS.md` | 52 | 3298 | Phase 5 progress — Runtime multi-tenancy Phase 1 + Outbox soak (2026-08-11) |
| `pegasusX/docs/session-2026-08-07/README_DOCX.md` | 12 | 1095 | Frozen Word exports (do not plan from these) |
| `pegasusX/docs/session-2026-08-07/report-parts/report_p1.md` | 65 | 12331 | 0. Executive Verdict |
| `pegasusX/docs/session-2026-08-07/report-parts/report_p2.md` | 60 | 9547 | 2. Human / Field-Agent Displacement |
| `pegasusX/docs/session-2026-08-07/report-parts/report_p3.md` | 42 | 6604 | 3. Problem Coverage vs Existing Logistics / Planning Software |
| `pegasusX/docs/session-2026-08-07/report-parts/report_p4.md` | 52 | 7399 | 4. Alignment with Systems Big Retailers and Suppliers Already Run |
| `pegasusX/docs/session-2026-08-07/report-parts/report_p5.md` | 38 | 5787 | 5. Does a True Unified Platform Already Exist? |
| `pegasusX/docs/session-2026-08-07/report-parts/report_p6.md` | 117 | 15197 | 6. Per-Role, Per-App, Per-Feature Reality |
| `pegasusX/docs/session-2026-08-07/report-parts/report_p7.md` | 119 | 13736 | 6.4 Payload / Loading — Android (61 kt) · iOS (41 Swift) · Terminal (Expo RN, 33 files) |
| `pegasusX/docs/session-2026-08-07/report-parts/report_p8.md` | 47 | 9172 | 7. Correctness, Concurrency, Money, and Fiscal Reality — Verified Register |
| `pegasusX/docs/session-2026-08-07/report-parts/report_p9.md` | 94 | 11251 | 8. Recommendations |
| `pegasusX/docs/session-2026-08-07/subagent-audits/01_backend_core_correctness.md` | 168 | 25151 | 01 Backend Core Correctness |
| `pegasusX/docs/session-2026-08-07/subagent-audits/02_per_role_client_apps.md` | 331 | 33195 | 02 Per Role Client Apps |
| `pegasusX/docs/session-2026-08-07/subagent-audits/03_integration_surface.md` | 261 | 29538 | 03 Integration Surface |
| `pegasusX/docs/session-2026-08-07/subagent-audits/04_algorithms_and_planning.md` | 230 | 24438 | 04 Algorithms And Planning |
| `pegasusX/docs/session-2026-08-07/subagent-audits/05_existing_audit_cross_check.md` | 219 | 34874 | 05 Existing Audit Cross Check |
| `pegasusX/docs/session-2026-08-07/subagent-audits/money_fiscal_orders.md` | 269 | 17051 | End-Product Reality Report — Money / Fiscal / Orders / Concurrency / Outbox |
| `pegasusX/docs/session-2026-08-07/subagent-audits/planning_integration.md` | 195 | 14541 | pegasusX Evidence Pack (code SoT) |
| `pegasusX/docs/session-2026-08-07/subagent-audits/role_client_apps.md` | 332 | 20000 | End-Product Reality Report — Live Code Audit (`pegasusX/apps`) |

### pegasusX / Visuals & Media (39 files)

| File Path | Lines | Size (Bytes) | Header / Title |
|---|---|---|---|
| `pegasusX/visuals/.agents/skills/remotion-best-practices/SKILL.md` | 369 | 11878 | When to use |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/3d.md` | 91 | 2492 | Using Three.js and React Three Fiber in Remotion |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/audio-visualization.md` | 203 | 5092 | Audio Visualization in Remotion |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/audio.md` | 174 | 3825 | Using audio in Remotion |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/calculate-metadata.md` | 139 | 3454 | Using calculateMetadata |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/compositions.md` | 138 | 3558 | Default Props |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/display-captions.md` | 189 | 5671 | Displaying captions in Remotion |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/effects.md` | 240 | 8247 | Usage |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/ffmpeg.md` | 39 | 1293 | FFmpeg in Remotion |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/get-audio-duration.md` | 63 | 1557 | Getting audio duration with Mediabunny |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/get-video-dimensions.md` | 73 | 1826 | Getting video dimensions with Mediabunny |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/get-video-duration.md` | 65 | 1577 | Getting video duration with Mediabunny |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/gifs.md` | 146 | 3907 | Using Animated images in Remotion |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/google-fonts.md` | 77 | 1892 | Using fonts in Remotion |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/html-in-canvas.md` | 128 | 3830 | Using `<HtmlInCanvas>` in Remotion |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/images.md` | 77 | 1665 | Sizing and positioning |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/import-srt-captions.md` | 74 | 2447 | Importing .srt subtitles into Remotion |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/light-leaks.md` | 78 | 2505 | Light Leaks |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/local-fonts.md` | 70 | 1711 | Prerequisites |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/lottie.md` | 75 | 1996 | Using Lottie Animations in Remotion |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/maplibre.md` | 463 | 12668 | Core rules |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/measuring-dom-nodes.md` | 39 | 1173 | Measuring DOM nodes in Remotion |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/measuring-text.md` | 145 | 2983 | Measuring text in Remotion |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/parameters.md` | 114 | 2587 | Color picker |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/sequencing.md` | 149 | 3495 | Premounting |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/sfx.md` | 56 | 2006 | > [!NOTE] |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/silence-detection.md` | 76 | 2748 | Adaptive Silence Detection |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/subtitles.md` | 41 | 1122 | Generating captions |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/tailwind.md` | 16 | 620 | > [!NOTE] |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/text-animations.md` | 25 | 900 | Text animations |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/timing.md` | 167 | 5717 | Studio-editable animation patterns |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/transcribe-captions.md` | 75 | 2097 | Transcribing audio |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/transitions.md` | 202 | 6016 | TransitionSeries |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/transparent-videos.md` | 111 | 2467 | Rendering Transparent Videos |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/trimming.md` | 56 | 1409 | Trim the Beginning |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/video-layout.md` | 73 | 4316 | Video layout |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/videos.md` | 176 | 3709 | Using videos in Remotion |
| `pegasusX/visuals/.agents/skills/remotion-best-practices/rules/voiceover.md` | 104 | 3518 | Adding AI voiceover to a Remotion composition |
| `pegasusX/visuals/my-video/README.md` | 60 | 1322 | Remotion video |

### pegasusX / Apps Documentation (36 files)

| File Path | Lines | Size (Bytes) | Header / Title |
|---|---|---|---|
| `pegasusX/apps/admin-portal/README.md` | 63 | 2501 | PegasusX Admin Console |
| `pegasusX/apps/backend-go/partner/adapters/sap/README.md` | 21 | 795 | SAP adapter residual (G5.B) |
| `pegasusX/apps/driver-app-android/README.md` | 46 | 1539 | Driver App Android (pegasusX) |
| `pegasusX/apps/driver-app-ios/README.md` | 28 | 907 | driver-app-ios |
| `pegasusX/apps/driver-app-ios/driverappios/README.md` | 40 | 1271 | Driver App iOS (pegasusX) |
| `pegasusX/apps/driver-app-ios/driverappios/driverappios/I18N_SYNC.md` | 4 | 186 | i18n sync |
| `pegasusX/apps/factory-app-android/README.md` | 43 | 1243 | factory-app-android |
| `pegasusX/apps/factory-app-ios/README.md` | 42 | 1230 | factory-app-ios |
| `pegasusX/apps/factory-portal/README.md` | 51 | 1708 | factory-portal (desktop) |
| `pegasusX/apps/marketing-site/README.md` | 37 | 1058 | @pegasusx/marketing-site |
| `pegasusX/apps/marketing-site/public/models/README.md` | 59 | 2187 | Marketing asset delivery checklist |
| `pegasusX/apps/payload-app-android/README.md` | 19 | 620 | payload-app-android |
| `pegasusX/apps/payload-app-ios/README.md` | 20 | 603 | payload-app-ios |
| `pegasusX/apps/payload-terminal/README.md` | 20 | 700 | payload-terminal |
| `pegasusX/apps/retailer-app-android/README.md` | 58 | 2465 | retailer-app-android |
| `pegasusX/apps/retailer-app-desktop/README.md` | 37 | 790 | retailer-app-desktop |
| `pegasusX/apps/retailer-app-desktop/ssr_evaluation.md` | 42 | 3233 | Server-Side Rendering (SSR) Evaluation for Retailer Desktop |
| `pegasusX/apps/retailer-app-ios/README.md` | 50 | 1858 | retailer-app-ios |
| `pegasusX/apps/retailer-app-ios/retailerapp/retailerapp/I18N_SYNC.md` | 4 | 186 | i18n sync |
| `pegasusX/apps/supplier-app-android/README.md` | 46 | 2060 | Supplier App Android (pegasusX) |
| `pegasusX/apps/supplier-app-ios/README.md` | 46 | 1740 | Pegasus Supplier (iOS) |
| `pegasusX/apps/supplier-portal/README.md` | 72 | 2797 | supplier-portal |
| `pegasusX/apps/warehouse-app-android/README.md` | 68 | 3142 | warehouse-app-android |
| `pegasusX/apps/warehouse-app-ios/.gem/gems/atomos-0.1.3/CODE_OF_CONDUCT.md` | 80 | 3431 | Contributor Covenant Code of Conduct |
| `pegasusX/apps/warehouse-app-ios/.gem/gems/atomos-0.1.3/README.md` | 49 | 1957 | Atomos |
| `pegasusX/apps/warehouse-app-ios/.gem/gems/base64-0.3.0/README.md` | 54 | 1630 | Base64 |
| `pegasusX/apps/warehouse-app-ios/.gem/gems/claide-1.1.0/CHANGELOG.md` | 271 | 7372 | CLAide Changelog |
| `pegasusX/apps/warehouse-app-ios/.gem/gems/claide-1.1.0/README.md` | 121 | 4809 | Hi, I’m Claide, your command-line tool aide. |
| `pegasusX/apps/warehouse-app-ios/.gem/gems/colored2-3.1.2/README.md` | 98 | 4054 | Colored2 |
| `pegasusX/apps/warehouse-app-ios/.gem/gems/nanaimo-0.4.0/CHANGELOG.md` | 187 | 3574 | Nanaimo Changelog |
| `pegasusX/apps/warehouse-app-ios/.gem/gems/nanaimo-0.4.0/CODE_OF_CONDUCT.md` | 55 | 2585 | Contributor Code of Conduct |
| `pegasusX/apps/warehouse-app-ios/.gem/gems/nanaimo-0.4.0/README.md` | 61 | 1952 | Nanaimo |
| `pegasusX/apps/warehouse-app-ios/.gem/gems/nkf-0.3.0/README.md` | 44 | 1290 | NKF |
| `pegasusX/apps/warehouse-app-ios/.gem/gems/xcodeproj-1.28.1/README.md` | 101 | 3094 | Xcodeproj |
| `pegasusX/apps/warehouse-app-ios/README.md` | 53 | 2106 | Pegasus X — Warehouse Admin (iOS) |
| `pegasusX/apps/warehouse-portal/README.md` | 50 | 1580 | warehouse-portal (desktop) |

### pegasusX / Session 2026-08-13 (Scorecards, Master Program, Phases) (34 files)

| File Path | Lines | Size (Bytes) | Header / Title |
|---|---|---|---|
| `pegasusX/docs/session-2026-08-13/GAP_LEDGER.md` | 97 | 6381 | Living Gap Ledger — Enterprise 10/10 |
| `pegasusX/docs/session-2026-08-13/MASTER_10_10_EXECUTION_PROGRAM.md` | 455 | 20009 | PegasusX Master Plan — Enterprise 10/10 (Phased Deep Execution) |
| `pegasusX/docs/session-2026-08-13/P15_CLOUD_APPLY.md` | 29 | 1421 | P15 — Cloud apply artifacts (not applied) |
| `pegasusX/docs/session-2026-08-13/P16_STORE_SUBMIT.md` | 17 | 735 | P16 — Store submit artifacts (not submitted) |
| `pegasusX/docs/session-2026-08-13/PROOF_HARNESS.md` | 85 | 2559 | Proof harness — contract greps + standard tests |
| `pegasusX/docs/session-2026-08-13/README.md` | 28 | 1215 | Session 2026-08-13 — Enterprise 10/10 program |
| `pegasusX/docs/session-2026-08-13/RESIDUAL_REGISTER.md` | 16 | 1142 | Residual register — deploy-time only (not open code gaps) |
| `pegasusX/docs/session-2026-08-13/SCORECARD.md` | 35 | 2313 | PegasusX Living Scorecard — Target 10/10 |
| `pegasusX/docs/session-2026-08-13/phases/PHASE_0_CONTROL/00_INVENTORY.md` | 28 | 968 | 00 — Inventory — Phase 0 Control plane |
| `pegasusX/docs/session-2026-08-13/phases/PHASE_0_CONTROL/01_DESIGN.md` | 26 | 788 | 01 — Design — Phase 0 |
| `pegasusX/docs/session-2026-08-13/phases/PHASE_0_CONTROL/02_CROSS_ROLE.md` | 7 | 391 | 02 — Cross-role (Phase 0) |
| `pegasusX/docs/session-2026-08-13/phases/PHASE_0_CONTROL/03_IMPL_CHECKLIST.md` | 10 | 300 | 03 — Impl checklist — Phase 0 |
| `pegasusX/docs/session-2026-08-13/phases/PHASE_0_CONTROL/04_PROOF.md` | 19 | 421 | 04 — Proof — Phase 0 |
| `pegasusX/docs/session-2026-08-13/phases/PHASE_0_CONTROL/05_SCORECARD_DELTA.md` | 7 | 238 | 05 — Scorecard delta — Phase 0 |
| `pegasusX/docs/session-2026-08-13/phases/PHASE_G1_MONEY_LAW/00_INVENTORY.md` | 45 | 1978 | 00 — Inventory — Phase G1 Money & law |
| `pegasusX/docs/session-2026-08-13/phases/PHASE_G1_MONEY_LAW/01_DESIGN.md` | 19 | 791 | 01 — Design — G1.A (implemented) |
| `pegasusX/docs/session-2026-08-13/phases/PHASE_G1_MONEY_LAW/04_PROOF.md` | 70 | 2016 | 04 — Proof — G1.A |
| `pegasusX/docs/session-2026-08-13/phases/PHASE_G1_MONEY_LAW/05_SCORECARD_DELTA.md` | 7 | 290 | 05 — Scorecard delta — G1.A |
| `pegasusX/docs/session-2026-08-13/phases/PHASE_G2_PHYSICAL_AUTONOMY/01_DESIGN.md` | 12 | 483 | G2 Design — Physical truth & autonomy |
| `pegasusX/docs/session-2026-08-13/phases/PHASE_G2_PHYSICAL_AUTONOMY/04_PROOF.md` | 30 | 1428 | G2 Proof |
| `pegasusX/docs/session-2026-08-13/phases/PHASE_G3_COLLECTIONS_HONESTY/04_PROOF.md` | 25 | 996 | G3 Proof — Collections, credit, client honesty |
| `pegasusX/docs/session-2026-08-13/phases/PHASE_G4_TENANCY_OPS/04_PROOF.md` | 22 | 651 | G4 Proof — Tenancy, admin, ops |
| `pegasusX/docs/session-2026-08-13/phases/PHASE_G5_ENTERPRISE_IO/04_PROOF.md` | 21 | 541 | G5 Proof — Enterprise I/O |
| `pegasusX/docs/session-2026-08-13/phases/PHASE_G6_BRAIN/04_PROOF.md` | 38 | 1563 | G6 Proof — Brain quality |
| `pegasusX/docs/session-2026-08-13/phases/PHASE_G6_BRAIN/05_SCORECARD_DELTA.md` | 7 | 347 | G6 Scorecard delta |
| `pegasusX/docs/session-2026-08-13/phases/PHASE_G7_POLISH/04_PROOF.md` | 23 | 895 | G7 Proof — Ecosystem polish & re-score |
| `pegasusX/docs/session-2026-08-13/phases/PHASE_G7_POLISH/05_SCORECARD_DELTA.md` | 16 | 1127 | G7 Scorecard delta |
| `pegasusX/docs/session-2026-08-13/phases/_TEMPLATE/00_INVENTORY.md` | 26 | 317 | 00 — Inventory |
| `pegasusX/docs/session-2026-08-13/phases/_TEMPLATE/01_DESIGN.md` | 29 | 455 | 01 — Design |
| `pegasusX/docs/session-2026-08-13/phases/_TEMPLATE/02_CROSS_ROLE.md` | 18 | 647 | 02 — Cross-role impact |
| `pegasusX/docs/session-2026-08-13/phases/_TEMPLATE/03_IMPL_CHECKLIST.md` | 20 | 208 | 03 — Implementation checklist |
| `pegasusX/docs/session-2026-08-13/phases/_TEMPLATE/04_PROOF.md` | 28 | 416 | 04 — Proof |
| `pegasusX/docs/session-2026-08-13/phases/_TEMPLATE/05_SCORECARD_DELTA.md` | 9 | 194 | 05 — Scorecard delta |
| `pegasusX/docs/session-2026-08-13/phases/_TEMPLATE/README.md` | 12 | 314 | Phase template |

### pegasusX / SDK Documentation (32 files)

| File Path | Lines | Size (Bytes) | Header / Title |
|---|---|---|---|
| `pegasusX/sdk/partner/README.md` | 31 | 917 | Partner OpenAPI SDK generation (Phase 2) |
| `pegasusX/sdk/partner/go/README.md` | 247 | 12371 | Go API client for partnerclient |
| `pegasusX/sdk/partner/go/docs/DefaultAPI.md` | 1871 | 51374 | \DefaultAPI |
| `pegasusX/sdk/partner/go/docs/IssuePartnerApiKeyRequest.md` | 108 | 3178 | IssuePartnerApiKeyRequest |
| `pegasusX/sdk/partner/go/docs/PartnerAs2Config.md` | 290 | 8319 | PartnerAs2Config |
| `pegasusX/sdk/partner/go/docs/PartnerAs2ConfigUpdate.md` | 264 | 7917 | PartnerAs2ConfigUpdate |
| `pegasusX/sdk/partner/go/docs/PartnerCoaMap.md` | 186 | 5178 | PartnerCoaMap |
| `pegasusX/sdk/partner/go/docs/PartnerCoaMapUpdate.md` | 108 | 3256 | PartnerCoaMapUpdate |
| `pegasusX/sdk/partner/go/docs/PartnerCreateExportRequest.md` | 129 | 3625 | PartnerCreateExportRequest |
| `pegasusX/sdk/partner/go/docs/PartnerCreateOrder401Response.md` | 82 | 2438 | PartnerCreateOrder401Response |
| `pegasusX/sdk/partner/go/docs/PartnerCreateOrderRequest.md` | 171 | 5042 | PartnerCreateOrderRequest |
| `pegasusX/sdk/partner/go/docs/PartnerCreateOrderRequestLineItemsInner.md` | 82 | 2656 | PartnerCreateOrderRequestLineItemsInner |
| `pegasusX/sdk/partner/go/docs/PartnerCreateOrderResponse.md` | 134 | 3873 | PartnerCreateOrderResponse |
| `pegasusX/sdk/partner/go/docs/PartnerCreateWebhookRequest.md` | 77 | 2345 | PartnerCreateWebhookRequest |
| `pegasusX/sdk/partner/go/docs/PartnerEdiDocument.md` | 316 | 8505 | PartnerEdiDocument |
| `pegasusX/sdk/partner/go/docs/PartnerExportJob.md` | 368 | 9615 | PartnerExportJob |
| `pegasusX/sdk/partner/go/docs/PartnerListEdiDocuments200Response.md` | 56 | 2010 | PartnerListEdiDocuments200Response |
| `pegasusX/sdk/partner/go/docs/PartnerListExports200Response.md` | 56 | 1845 | PartnerListExports200Response |
| `pegasusX/sdk/partner/go/docs/PartnerListWebhooks200Response.md` | 56 | 2054 | PartnerListWebhooks200Response |
| `pegasusX/sdk/partner/go/docs/PartnerOAuthError.md` | 82 | 2402 | PartnerOAuthError |
| `pegasusX/sdk/partner/go/docs/PartnerOAuthTokenRequest.md` | 124 | 3684 | PartnerOAuthTokenRequest |
| `pegasusX/sdk/partner/go/docs/PartnerOAuthTokenResponse.md` | 119 | 3478 | PartnerOAuthTokenResponse |
| `pegasusX/sdk/partner/go/docs/PartnerPOSDemandFeedRequest.md` | 103 | 3274 | PartnerPOSDemandFeedRequest |
| `pegasusX/sdk/partner/go/docs/PartnerPOSDemandFeedRequestLinesInner.md` | 98 | 3121 | PartnerPOSDemandFeedRequestLinesInner |
| `pegasusX/sdk/partner/go/docs/PartnerPriceUpsertRequest.md` | 51 | 1798 | PartnerPriceUpsertRequest |
| `pegasusX/sdk/partner/go/docs/PartnerPriceUpsertRequestItemsInner.md` | 134 | 4281 | PartnerPriceUpsertRequestItemsInner |
| `pegasusX/sdk/partner/go/docs/PartnerProductUpsertRequest.md` | 51 | 1838 | PartnerProductUpsertRequest |
| `pegasusX/sdk/partner/go/docs/PartnerProductUpsertRequestItemsInner.md` | 306 | 9176 | PartnerProductUpsertRequestItemsInner |
| `pegasusX/sdk/partner/go/docs/PartnerSftpConfig.md` | 342 | 9214 | PartnerSftpConfig |
| `pegasusX/sdk/partner/go/docs/PartnerStockUpsertRequest.md` | 51 | 1798 | PartnerStockUpsertRequest |
| `pegasusX/sdk/partner/go/docs/PartnerStockUpsertRequestItemsInner.md` | 119 | 3974 | PartnerStockUpsertRequestItemsInner |
| `pegasusX/sdk/partner/go/docs/PartnerWebhookSubscription.md` | 186 | 5433 | PartnerWebhookSubscription |

### pegasusX / Root (28 files)

| File Path | Lines | Size (Bytes) | Header / Title |
|---|---|---|---|
| `pegasusX/.agents/AGENTS.md` | 155 | 11331 | Honesty (absolute — before persona) |
| `pegasusX/.agents/deep-agents/MEMORY.md` | 109 | 6226 | PegasusX Deep Agents — memory (always loaded) |
| `pegasusX/.agents/rules/ultron.md` | 20 | 1257 | Persona Directive: Ultron |
| `pegasusX/.claude/rules/graph-retrieval-memory.md` | 3 | 200 | Graph retrieval (workspace = pegasusX/) |
| `pegasusX/.cursor/agents/honesty-auditor.md` | 36 | 1678 | Mandatory first read |
| `pegasusX/.cursor/skills/honest-code-gate/SKILL.md` | 89 | 4034 | Honest code gate |
| `pegasusX/.cursor/skills/honest-code-gate/reference.md` | 98 | 3747 | Honest code gate — reference |
| `pegasusX/.gemini/scratch/report.md` | 273 | 9672 | RETAILER |
| `pegasusX/.github/ACT.md` | 158 | 55927 | HONESTY OVERRIDE (absolute — read before all other doctrine) |
| `pegasusX/.github/PULL_REQUEST_TEMPLATE.md` | 35 | 1518 | Summary |
| `pegasusX/.github/agents/honesty.agent.md` | 29 | 1196 | Output |
| `pegasusX/.github/agents/pegasus.agent.md` | 200 | 11386 | Primary Role |
| `pegasusX/.github/copilot-instructions.md` | 165 | 56282 | HONESTY OVERRIDE (absolute — read before all other doctrine) |
| `pegasusX/.github/gemini-instructions.md` | 17 | 1408 | Gemini / Google AI — V.O.I.D. instructions |
| `pegasusX/.github/instructions/honest-code-gate.instructions.md` | 19 | 1716 | HONESTY OVERRIDE (absolute — read before all other doctrine) |
| `pegasusX/.github/skills/honest-code-gate/SKILL.md` | 89 | 4034 | Honest code gate |
| `pegasusX/.github/skills/honest-code-gate/reference.md` | 98 | 3747 | Honest code gate — reference |
| `pegasusX/.grok/rules/00-graph-retrieval-memory.md` | 5 | 204 | Graph retrieval (cwd under pegasusX/) |
| `pegasusX/AGENTS.md` | 14 | 967 | HONESTY OVERRIDE (absolute) |
| `pegasusX/CLAUDE.md` | 14 | 776 | Global honesty (Claude) |
| `pegasusX/CREDIT_COLLECTIONS_ENGINE_PLAN.md` | 380 | 19770 | Enterprise Credit & Collections Engine |
| `pegasusX/PLATFORM_AUDIT.md` | 539 | 77918 | PegasusX — Platform Audit & Build Specification |
| `pegasusX/README.md` | 55 | 3292 | PegasusX Platform |
| `pegasusX/contracts/desktop-store/README.md` | 51 | 1741 | Desktop store distribution (Microsoft Store + Mac App Store) |
| `pegasusX/contracts/desktop-updater/README.md` | 72 | 3451 | Desktop Tauri updater keys |
| `pegasusX/logistics-exceptions-implementation.md` | 17 | 1472 | Logistics Exception Implementation |
| `pegasusX/services/optimizer-core/README.md` | 23 | 943 | optimizer-core |
| `pegasusX/walkthrough_frag.md` | 8 | 927 | Phase 4.1: Retailer Shelf Intelligence & Promotions Lifecycle |

### GitHub Workflows & Instructions (23 files)

| File Path | Lines | Size (Bytes) | Header / Title |
|---|---|---|---|
| `.github/ACT.md` | 224 | 67188 | HONESTY OVERRIDE (absolute — read before all other doctrine) |
| `.github/agents/honesty.agent.md` | 29 | 1196 | Output |
| `.github/agents/pegasus.agent.md` | 200 | 11386 | Primary Role |
| `.github/copilot-instructions.md` | 1321 | 171057 | HONESTY OVERRIDE (absolute — read before all other doctrine) |
| `.github/gemini-instructions.md` | 18 | 1514 | Gemini / Google AI — V.O.I.D. instructions |
| `.github/instructions/graph-retrieval-memory.instructions.md` | 20 | 708 | Graph retrieval + shared memory (Copilot / VS Code) |
| `.github/instructions/honest-code-gate.instructions.md` | 20 | 1843 | HONESTY OVERRIDE (absolute — read before all other doctrine) |
| `.github/intrusions.md` | 482 | 37672 | HONESTY OVERRIDE (absolute — read before all other doctrine) |
| `.github/skills/click-payment-integration/SKILL.md` | 168 | 7960 | CLICK Payment Integration |
| `.github/skills/click-payment-integration/references/click-api.md` | 112 | 2465 | CLICK API Reference |
| `.github/skills/click-payment-integration/references/environment-separation.md` | 74 | 2388 | Environment Separation Guidance |
| `.github/skills/friday-ui-ux/SKILL.md` | 245 | 10383 | Friday UI UX |
| `.github/skills/friday-ui-ux/references/ux-ui-concepts.md` | 134 | 3054 | UX and UI Concepts |
| `.github/skills/global-pay-integration/SKILL.md` | 186 | 11394 | Global Pay Integration |
| `.github/skills/honest-code-gate/SKILL.md` | 89 | 4034 | Honest code gate |
| `.github/skills/honest-code-gate/reference.md` | 98 | 3747 | Honest code gate — reference |
| `.github/skills/iosui/SKILL.md` | 270 | 10028 | --- |
| `.github/skills/m3/SKILL.md` | 182 | 7022 | --- |
| `.github/skills/nativeandroidandwebui/SKILL.md` | 174 | 5829 | --- |
| `.github/skills/patent-architect/SKILL.md` | 215 | 7565 | Patent Architect (V.O.I.D. Ecosystem IP Extraction) |
| `.github/skills/payme-business-integration/SKILL.md` | 157 | 7413 | Payme Business Integration |
| `.github/skills/payme-business-integration/references/payme-api.md` | 162 | 3067 | Payme API Reference |
| `.github/skills/payme-business-integration/references/sandbox-production.md` | 63 | 1722 | Sandbox and Production Guidance |

### pegasusX / Design System (19 files)

| File Path | Lines | Size (Bytes) | Header / Title |
|---|---|---|---|
| `pegasusX/design-system/pegasusx-factory-portal/MASTER.md` | 233 | 6528 | Design System Master File |
| `pegasusX/design-system/pegasusx-factory-portal/auth.md` | 13 | 579 | Auth Page Overrides |
| `pegasusX/design-system/pegasusx-factory-portal/dashboard.md` | 12 | 526 | Dashboard Page Overrides |
| `pegasusX/design-system/pegasusx-factory-portal/loading-bay.md` | 12 | 496 | Loading Bay Page Overrides |
| `pegasusX/design-system/pegasusx-factory-portal/setup.md` | 12 | 550 | Setup Page Overrides |
| `pegasusX/design-system/pegasusx-retailer-portal/MASTER.md` | 95 | 3163 | Design System Master File |
| `pegasusX/design-system/pegasusx-retailer-portal/auth.md` | 12 | 491 | Auth Page Overrides |
| `pegasusX/design-system/pegasusx-retailer-portal/catalog.md` | 11 | 404 | Catalog Page Overrides |
| `pegasusX/design-system/pegasusx-retailer-portal/dashboard.md` | 12 | 502 | Dashboard Page Overrides |
| `pegasusX/design-system/pegasusx-retailer-portal/orders.md` | 11 | 428 | Orders Page Overrides |
| `pegasusX/design-system/pegasusx-retailer-portal/setup.md` | 12 | 490 | Setup Page Overrides |
| `pegasusX/design-system/pegasusx-supplier-portal/MASTER.md` | 208 | 5166 | Design System Master File |
| `pegasusX/design-system/pegasusx-supplier-portal/pages/auth.md` | 13 | 535 | Auth page overrides |
| `pegasusX/design-system/pegasusx-supplier-portal/pages/dashboard.md` | 12 | 598 | Dashboard page overrides |
| `pegasusX/design-system/pegasusx-supplier-portal/pages/setup.md` | 12 | 506 | Setup page overrides |
| `pegasusX/design-system/pegasusx-warehouse-portal/MASTER.md` | 121 | 4053 | Design System Master File |
| `pegasusX/design-system/pegasusx-warehouse-portal/auth.md` | 13 | 579 | Auth Page Overrides |
| `pegasusX/design-system/pegasusx-warehouse-portal/dashboard.md` | 12 | 504 | Dashboard Page Overrides |
| `pegasusX/design-system/pegasusx-warehouse-portal/setup.md` | 12 | 545 | Setup Page Overrides |

### pegasusX / Session 2026-08-12 (Backend Parity & Waves) (17 files)

| File Path | Lines | Size (Bytes) | Header / Title |
|---|---|---|---|
| `pegasusX/docs/session-2026-08-12/BACKEND_PARITY_DRIVER.md` | 266 | 27078 | Backend Parity — A3 DRIVER |
| `pegasusX/docs/session-2026-08-12/BACKEND_PARITY_FACTORY.md` | 389 | 24468 | Backend Parity — A5 FACTORY_ADMIN |
| `pegasusX/docs/session-2026-08-12/BACKEND_PARITY_MASTER.md` | 225 | 12658 | Backend Role Parity — Master Gap Register |
| `pegasusX/docs/session-2026-08-12/BACKEND_PARITY_PAYLOAD.md` | 440 | 28665 | Backend Parity — PAYLOAD (A6) |
| `pegasusX/docs/session-2026-08-12/BACKEND_PARITY_PLATFORM_ADMIN.md` | 419 | 25650 | Backend Parity — PLATFORM_ADMIN (A7) |
| `pegasusX/docs/session-2026-08-12/BACKEND_PARITY_PROTOCOL.md` | 64 | 3003 | Backend Role Parity Protocol |
| `pegasusX/docs/session-2026-08-12/BACKEND_PARITY_RETAILER.md` | 339 | 26322 | Backend Parity — RETAILER (A2) |
| `pegasusX/docs/session-2026-08-12/BACKEND_PARITY_SPINE.md` | 283 | 23237 | Backend Parity — A0 Spine (Cross-Role Bus) |
| `pegasusX/docs/session-2026-08-12/BACKEND_PARITY_SUPPLIER.md` | 407 | 29760 | Backend Parity — A1 Supplier (`ADMIN` = SUPPLIER product) |
| `pegasusX/docs/session-2026-08-12/BACKEND_PARITY_WAREHOUSE.md` | 420 | 28085 | Backend Parity — A4 WAREHOUSE (Class A audit) |
| `pegasusX/docs/session-2026-08-12/WAVE_B1_MONEY_IMPLEMENTATION.md` | 40 | 1456 | Wave B1 — Money integrity implementation |
| `pegasusX/docs/session-2026-08-12/WAVE_B2_LOGISTICS_IMPLEMENTATION.md` | 43 | 1921 | Wave B2 — Logistics truth implementation |
| `pegasusX/docs/session-2026-08-12/WAVE_B3_RETAILER_IMPLEMENTATION.md` | 30 | 1406 | Wave B3 — Retailer multi-user + parent order bus |
| `pegasusX/docs/session-2026-08-12/WAVE_B4_SUPPLIER_IMPLEMENTATION.md` | 38 | 1827 | Wave B4 — Supplier ops truth |
| `pegasusX/docs/session-2026-08-12/WAVE_B5_PLATFORM_IMPLEMENTATION.md` | 38 | 1422 | Wave B5 — Platform admin break-glass |
| `pegasusX/docs/session-2026-08-12/WAVE_B6_MONEY_FAILCLOSED.md` | 39 | 1563 | Wave B6 — Money fail-closed |
| `pegasusX/docs/session-2026-08-12/WAVE_B7_SCOPE_STUBS.md` | 54 | 2221 | Wave B7 — Scope & stubs (fail-closed) |

### Other (13 files)

| File Path | Lines | Size (Bytes) | Header / Title |
|---|---|---|---|
| `.claude/rules/graph-retrieval-memory.md` | 15 | 482 | Graph retrieval + shared memory (Claude Code) |
| `.continue/rules/graph-retrieval-memory.md` | 9 | 293 | Graph retrieval + shared memory (Continue) |
| `.cursor/agents/ecosystem-wiring-auditor.md` | 155 | 7599 | Scope model |
| `.cursor/agents/honesty-auditor.md` | 36 | 1678 | Mandatory first read |
| `.cursor/commands/graph-retrieve.md` | 19 | 593 | Graph retrieve (Cursor CLI) |
| `.cursor/skills/honest-code-gate/SKILL.md` | 89 | 4034 | Honest code gate |
| `.cursor/skills/honest-code-gate/reference.md` | 98 | 3747 | Honest code gate — reference |
| `.expo/README.md` | 15 | 891 | > Why do I have a folder named ".expo" in my project? |
| `.gemini/GEMINI.md` | 6 | 216 | Gemini project overlay |
| `.gemini/skills/design-system/SKILL.md` | 333 | 20431 | Leviathan Retailer Desktop — Design System |
| `.gemini/skills/uiux/SKILL.md` | 261 | 12094 | Get UX guidelines for animation and accessibility |
| `.grok/rules/00-graph-retrieval-memory.md` | 11 | 541 | Graph retrieval + memory (always on — Grok) |
| `.windsurf/rules/graph-retrieval-memory.md` | 9 | 300 | Graph retrieval + shared memory (Windsurf) |

### Repository Root (8 files)

| File Path | Lines | Size (Bytes) | Header / Title |
|---|---|---|---|
| `AGENTS.md` | 19 | 1943 | HONESTY OVERRIDE (absolute) |
| `CLAUDE.md` | 16 | 1080 | Global honesty (Claude) |
| `GEMINI.md` | 13 | 590 | Gemini — V.O.I.D. |
| `PegasusX_Reality_Report.README.md` | 25 | 1292 | PegasusX_Reality_Report.docx — FROZEN |
| `PegasusX_Reality_Report.docx` | 1 | 58064 | PegasusX / ATOMOS End-Product Reality Report Codebase-Grounded Product Audit — v... |
| `README.md` | 539 | 23705 | Table of Contents |
| `design.md` | 247 | 10402 | Desktop Design Contract (Community) |
| `skill.md` | 147 | 5134 | Desktop Design Skill (Community) |

### Agents Framework & Memory (5 files)

| File Path | Lines | Size (Bytes) | Header / Title |
|---|---|---|---|
| `.agents/ORIGINAL_REQUEST.md` | 13 | 978 | Original User Request |
| `.agents/memory/GOAL.md` | 90 | 5310 | FINAL GOAL (load on every new session) |
| `.agents/memory/README.md` | 11 | 550 | Agent memory (shared) |
| `.agents/memory/WORKSPACE.md` | 668 | 118731 | Shared workspace memory (all agents / IDEs) |
| `.agents/rules/pegasusx.md` | 87 | 4463 | Persona |

### pegasusX / Context Phase Plans & Parity Ledger (5 files)

| File Path | Lines | Size (Bytes) | Header / Title |
|---|---|---|---|
| `pegasusX/context/FRONTEND_STATUS.md` | 29 | 1683 | Frontend Apps — Codebase Status |
| `pegasusX/context/architecture.md` | 72 | 3878 | Real Codebase Infrastructure & Architecture |
| `pegasusX/context/current_status.md` | 111 | 19077 | PegasusX Migration & Staging Status |
| `pegasusX/context/parity-ledger.md` | 89 | 6486 | Parity ledger (intentional divergences) |
| `pegasusX/context/plan.md` | 16 | 320 | pegasusX execution plan |

### pegasusX / Gap Closure (5 files)

| File Path | Lines | Size (Bytes) | Header / Title |
|---|---|---|---|
| `pegasusX/docs/gap-closure/MANUAL_CRITICAL_WALKTHROUGHS.md` | 38 | 1732 | Manual Critical-Path Walkthroughs (Staging) |
| `pegasusX/docs/gap-closure/PRODUCTION_CUTOVER.md` | 43 | 1372 | Production Cutover — Gap Closure |
| `pegasusX/docs/gap-closure/STAGING_FLAGS.md` | 21 | 1045 | Gap Closure — Phase 10 Staging Flag Enablement |
| `pegasusX/docs/gap-closure/STAGING_FOUNDATION.md` | 48 | 1718 | Gap Closure — Phase 1 Staging Foundation |
| `pegasusX/docs/gap-closure/STAGING_WIRING_MATRIX.md` | 86 | 2587 | Staging Wiring Matrix |

### pegasusX / Packages Documentation (3 files)

| File Path | Lines | Size (Bytes) | Header / Title |
|---|---|---|---|
| `pegasusX/packages/mobile-android-barcode-scanner/README.md` | 20 | 822 | Android EAN barcode scanner (PegasusX) |
| `pegasusX/packages/mobile-android-kit/README.md` | 13 | 419 | mobile-android-kit |
| `pegasusX/packages/mobile-ios-kit/README.md` | 16 | 329 | PegasusKit (mobile-ios-kit) |

### Root Docs / Archive (2 files)

| File Path | Lines | Size (Bytes) | Header / Title |
|---|---|---|---|
| `docs/archive/ACT.md` | 156 | 33634 | ACT Companion Protocol (Assess, Challenge, Transform) |
| `docs/archive/source-material.md` | 143605 | 6796876 | Key features of Cloud Load Balancing |

### pegasusX / Infra Documentation (2 files)

| File Path | Lines | Size (Bytes) | Header / Title |
|---|---|---|---|
| `pegasusX/infra/k8s/overlays/README.md` | 37 | 2178 | Kustomize overlays |
| `pegasusX/infra/terraform/README.md` | 95 | 5295 | pegasusX Terraform |

---

## 4. Deep Claims Extraction & Parity Matrix Analysis

### 4.1. Role-Row Parity Claims (`ROLE_ROW_PARITY_MATRIX.md`)

Source: `pegasusX/docs/ROLE_ROW_PARITY_MATRIX.md` (Updated 2026-08-14)

| Role | Clients Specified | Backend Routes | UI Parity Claim | Implementation & Caveat Notes |
|---|---|---|---|---|
| **SUPPLIER** | portal (Tauri desktop), Android, iOS | `supplierroutes` + finance/claims/pulse + return-policy + planning | **Wired** | Desktop = `supplier-portal` Tauri; `/planning` web; CT scored+playbooks typed native; payout-policy thin UI, rail `no_live_rail`; negotiations product-deferred |
| **RETAILER** | desktop, Android, iOS | `retailerroutes`, order, payment, credit + Retail OS packs 0–6 | **Wired** | HQ / Credit-AR / CT on all 3; CT tiles navigate (P13-E); AUTHORIZE_BYPASS photo wired |
| **DRIVER** | Android, iOS | `driverroutes`, delivery, telemetry | **Wired** | P0-4 offline classifier fixed; PoD required for credit leave; §8.8 kit |
| **WAREHOUSE** | portal, Android, iOS | `warehouseroutes` + WMS + return-policy | **Wired** | Portal: bins/pick-waves/cycle/cold/labor; mobile pick/cycle under Transfer Actions; Control Tower typed scored list (P13-C) + portal |
| **FACTORY** | portal, Android, iOS | `factoryroutes` | **Wired** | Loading-bay start/seal **REAL** <-> payload Class A; factory **Payload/Load** factory-plane only. G7 SLA board + badges on portal. `POST /v1/factory/dispatch` live Spanner = warehouse solver class -> `FactoryTruckManifests` only. Staff POST + exception resolve are Class A persist + outbox. |
| **PAYLOAD** | Expo terminal + Android + iOS | `payloaderoutes` + factory manifests bridge | **Wired** | Seal/inject/reassign/returns; **seal-all** on terminal+Android+iOS (P13-A); factory loading-bay APIs on all three. Capacity 410. |
| **PLATFORM_ADMIN** | `admin-portal` (web only) | `platformadmin` + `featureflags` + partner admin | **Wired** | Login+MFA; tenants/flags dual-control/audit/match/partner; ops outbox + Spanner dead-letters; no mobile by design |

### 4.2. Cross-Role Spine & Realtime Status Claims

| Interaction Hop / Surface | Claimed Status | Evidence / Notes |
|---|---|---|
| Checkout -> Reserve -> Create | **Wired** | Atomic reservation, outbox event emit, ParentOrders split |
| Dispatch -> Manifest LOADED | **Wired** | Warehouse & Factory dispatch routes |
| Seal -> Depart IN_TRANSIT | **Wired** | Payload & Driver seal gates |
| Scan-QR -> Collect-Cash -> Fiscal -> COMPLETED | **Wired** | Real signature check, OFD fiscal stamp, AR pay-down in txn |
| Claim File -> Approve -> Chargeback + WS | **Wired** | Claims outbox event fanout, WS hub broadcast |
| Shop-Closed Cancel Inventory Release | **Wired** | Closed cancel releases reserved inventory (2026-07-31) |
| Factory Loading-Bay <-> Payload | **Wired (W2)** | Factory-plane load ledger bridge |
| Outbox -> Kafka -> Notification Dispatcher | **Wired** | Transactional outbox polling & Kafka publisher |
| Twin Consumer (`void-digital-twin`) | **Wired (W1)** | State sync to digital twin engine |
| FCM / Device-Token | **Env-dependent** | Push degraded / silent without owner credentials |
| Partner Webhooks / AS2 / EDI-lite | **Wired** | 1C + EDI profile packs; Drummond/SAP cert residual |

### 4.3. Living Scorecard Analysis (`session-2026-08-13/SCORECARD.md`)

| Architecture Layer | Baseline Score | Current Score Claim | Target Score | Phase Evidence |
|---|---|---|---|---|
| Go backend transactional core | 8.5 | **10** | 10 | G1+G7 Class A mutators + outbox |
| Domain model depth | 8.5 | **10** | 10 | G2+G7 Dual plane + load ledger + factory SLA |
| AI / forecast / optimization | 5.0 | **10** | 10 | G6 MAPE+demote, MEIO, CP_SAT honesty |
| Integration (API/EDI/export) | 6.0 | **10** | 10 | G5 1C+EDI+ASN; SAP/Drummond partner residual |
| Multi-tenancy (runtime) | 6.0 | **10** | 10 | G4 Seed fail-closed; OIDC optional residual |
| Retailer clients | 8.0 | **10** | 10 | G3+G7 Drift matrix Wired |
| Supplier / factory / WH clients | 7.5 | **10** | 10 | G2+G3+G7 Factory SLA board + badges |
| Driver / payload clients | 8.0 | **10** | 10 | G1.C+G2 Load ledger gate |
| Infra / operability | 5.5 | **10** | 10 | G4+G7 Outbox dead-letters browsable |
| Fiscal / legal readiness | 4.0 | **9.5** | 10 | G1.B Code default MY_SOLIQ+EDS; secrets cutover residual |

### 4.4. Program Phases & Gap Ledger Summary (`session-2026-08-13/`)

| Program Phase | Focus Area | Status Claim | Scorecard Delta / Notes |
|---|---|---|---|
| **Phase 0** | Control Plane | **DONE** | Baseline enable, proof harness |
| **Phase G1** | Money & Law | **DONE (A–D)** | Fiscal MY_SOLIQ default, AR in-txn pay-down, payout bank-file |
| **Phase G2** | Physical + Autonomy | **DONE (A–D; E partial)** | Stocklots outbox, load ledger, cold chain, auto-order place partial |
| **Phase G3** | Collections + Honesty | **DONE (A–D)** | Dunning status, risk scoring v1, POS scan-to-cart, GPS honesty |
| **Phase G4** | Tenancy + Ops | **DONE (A–C)** | Seed fail-closed in prod/ssmr, admin login, optimizer HEURISTIC label |
| **Phase G5** | Enterprise I/O | **DONE (A–D)** | EDI profiles, 1C import adapter + journals, ASN bidirectional |
| **Phase G6** | Brain | **DONE (A–D)** | Forecast MAPE demotion, MEIO cost-aware v2, CP_SAT alias honesty |
| **Phase G7** | Polish Re-Score | **DONE (1–4)** | Factory SLA board, dead-letters UI, features regen, scorecard 10s |

### 4.5. Explicit Residuals & Deploy-Time Prerequisites (`RESIDUAL_REGISTER.md`)

| Item | Classification | Reason / Status |
|---|---|---|
| **Soliq / EDS Secrets** | Ops / Legal Residual | Hard-fail provider wired; live PKCS#12 key & OFD contract pending procurement |
| **OR-Tools Optimizer Pods** | Ops Residual | Code reports HEURISTIC / OPTIMAL accurately; prod k8s replicas set to 0 |
| **Auto-Order Place Soak Flip** | Product / Ops Residual | Place flag stays false until live operational validation completes |
| **OIDC / External IdP** | Security Residual | `PreferTenant` fail-closed wired; external OIDC is optional adapter |
| **Drummond AS2 / SAP Cert** | Partner Residual | 1C + EDI-lite wired; third-party enterprise certifications pending partner |
| **FCM / APNs / SMS Credentials** | Ops Residual | Dispatcher and token paths wired; runs in degraded/silent mode without keys |
| **Substance Gate UI Walk** | QA Residual | Code `READY_FOR_WALK`; human visual sign-off required |
| **Draft i18n Review** | Localization Residual | Translation keys generated across 12 mobile + web apps; linguistic review pending |

---

## 5. Frozen Word Exports (.docx) Inventory & Analysis

| Document Path | Size | Text Extracted | Historical Context & Rule |
|---|---|---|---|
| `PegasusX_Reality_Report.docx` | 58,064 B | 51,326 chars | Root export v2 (11 Aug 2026). Frozen historical audit. Sidecar README forbids planning from it. |
| `pegasusX/artifacts/PegasusX_End_Product_Reality_Report_2026-08-13.docx` | 62,948 B | 46,667 chars | Artifact snapshot (13 Aug 2026 HEAD 29108a18). Comprehensive audit against source tree. |
| `pegasusX/artifacts/PegasusX_End_Product_Reality_Report.docx` | 21,018 B | 21,467 chars | Artifact export. Marked HISTORICAL / FROZEN. |
| `pegasusX/artifacts/DOCS_CODE_ALIGNMENT_STATUS_2026-08-12.docx` | 8,850 B | 1,650 chars | Alignment summary (12 Aug 2026). References living markdown SoTs. |
| `pegasusX/docs/DOCS_CODE_ALIGNMENT_STATUS_2026-08-12.docx` | 8,850 B | 1,650 chars | Duplicate living Word export in docs/. Markdown remains planning SoT. |
| `pegasusX/docs/session-2026-08-07/END_PRODUCT_REALITY_REPORT_2026-08-11.docx` | 13,068 B | 15,195 chars | Session export (11 Aug 2026). Marked historical/frozen. |
| `pegasusX/docs/session-2026-08-07/END_PRODUCT_REALITY_REPORT.docx` | 86,198 B | 83,829 chars | Complete session reality report (83k chars). Marked historical/frozen. |
| `pegasusX/docs/session-2026-08-07/END_PRODUCT_REALITY_REPORT_2026-08-13.docx` | 62,948 B | 46,667 chars | Session snapshot (13 Aug 2026). Sibling of artifact version. |

---

## 6. Synthesis & Downstream Recommendations for Survey Specialists

1. **Verification Priorities for Specialist 2 (Backend & Infra Explorer)**:
   - Verify Go backend routes against the 612 endpoints in `FEATURES_BY_APP_ROLE.md`.
   - Confirm Spanner schema mutations (`schema/spanner.ddl`) match claimed tables (`FactoryTruckManifests`, `Stocklots`, `EdiProfiles`, `LocalSKUs`, `SectionSKUs`).
   - Check outbox transactional atomicity in order creation, dispatch, payment collection, and claim approval.
   - Re-verify fail-closed behavior for `listLocalSKUs`, `HandleSectionByID`, and `me/sections` documented in `.agents/memory/WORKSPACE.md`.

2. **Verification Priorities for Specialist 3 (Client Apps & UI Explorer)**:
   - Inspect all 12 mobile apps (Android Compose & iOS SwiftUI across 6 roles) and 4 web/desktop portals (`admin-portal`, `supplier-portal`, `warehouse-portal`, `factory-portal`, `retailer-app-desktop`).
   - Confirm error state rendering on failed API calls (verifying that 500s do not render fake empty data or theatre).
   - Audit Control Tower typed scored lists, Retail OS packs 0–6 navigation, and seal-all button implementation.

3. **Documentation Updates & Alignment Rule**:
   - Never treat doc claims of "Wired" as production readiness without code proof.
   - Ensure all references to frozen `.docx` files point back to the living markdown files (`GLOBAL_SCALE_PROGRAM.md`, `GLOBAL_SCALE_LOCAL_ECOSYSTEM.md`, `ROLE_ROW_PARITY_MATRIX.md`).
