import os, sys, json, zipfile, xml.etree.ElementTree as ET
from collections import defaultdict

root_dir = '/Users/shakhzod/Desktop/V.O.I.D'
prune_dirs = {
    '.git', 'node_modules', '.gradle', 'dist', 'build', '.next', 'vendor', 
    '.venv', 'adyen-go-api-library-main', '.pytest_cache', 'softwareengineercv-main'
}

docs = []
for root, dirs, files in os.walk(root_dir):
    dirs[:] = [d for d in dirs if d not in prune_dirs]
    for f in files:
        if f.endswith('.md') or f.endswith('.docx'):
            full_path = os.path.join(root, f)
            rel_path = os.path.relpath(full_path, root_dir)
            if rel_path.startswith('.agents/') and not (rel_path.startswith('.agents/memory/') or rel_path.startswith('.agents/rules/') or rel_path in ['.agents/ORIGINAL_REQUEST.md']):
                continue
            docs.append((rel_path, full_path))

docs.sort()

categories = defaultdict(list)
for rel_path, full_path in docs:
    size = os.path.getsize(full_path)
    lines_count = 0
    title = ''
    
    if rel_path.endswith('.docx'):
        try:
            with zipfile.ZipFile(full_path) as z:
                xml_content = z.read('word/document.xml')
                tree = ET.fromstring(xml_content)
                text = ' '.join(tree.itertext())
                lines_count = len(text.split('\n'))
                title = text[:80].strip() + '...'
        except Exception as e:
            title = f'Error reading docx: {e}'
    else:
        try:
            with open(full_path, 'r', encoding='utf-8', errors='ignore') as f:
                raw_lines = f.readlines()
                lines_count = len(raw_lines)
                for line in raw_lines:
                    line_s = line.strip()
                    if line_s.startswith('#'):
                        title = line_s.lstrip('#').strip()
                        break
                if not title and raw_lines:
                    title = raw_lines[0].strip()
        except Exception as e:
            title = f'Error reading md: {e}'

    category = 'Other'
    if rel_path.startswith('pegasusX/docs/'):
        if 'session-2026-08-13' in rel_path:
            category = 'pegasusX / Session 2026-08-13 (Scorecards, Master Program, Phases)'
        elif 'session-2026-08-12' in rel_path:
            category = 'pegasusX / Session 2026-08-12 (Backend Parity & Waves)'
        elif 'session-2026-08-07' in rel_path:
            category = 'pegasusX / Session 2026-08-07 (Reality Reports & Gap Registers)'
        elif 'big-platform-baseline' in rel_path:
            category = 'pegasusX / Big Platform Baseline (Deep Specs)'
        elif 'gap-closure' in rel_path:
            category = 'pegasusX / Gap Closure'
        else:
            category = 'pegasusX / Core Docs & Specifications'
    elif rel_path.startswith('pegasusX/context/'):
        category = 'pegasusX / Context Phase Plans & Parity Ledger'
    elif rel_path.startswith('pegasusX/artifacts/'):
        category = 'pegasusX / Artifacts & Snapshots'
    elif rel_path.startswith('pegasusX/apps/'):
        category = 'pegasusX / Apps Documentation'
    elif rel_path.startswith('pegasusX/packages/'):
        category = 'pegasusX / Packages Documentation'
    elif rel_path.startswith('pegasusX/infra/'):
        category = 'pegasusX / Infra Documentation'
    elif rel_path.startswith('pegasusX/sdk/'):
        category = 'pegasusX / SDK Documentation'
    elif rel_path.startswith('pegasusX/design-system/'):
        category = 'pegasusX / Design System'
    elif rel_path.startswith('pegasusX/visuals/'):
        category = 'pegasusX / Visuals & Media'
    elif rel_path.startswith('pegasusX/'):
        category = 'pegasusX / Root'
    elif rel_path.startswith('.agents/'):
        category = 'Agents Framework & Memory'
    elif rel_path.startswith('.github/'):
        category = 'GitHub Workflows & Instructions'
    elif rel_path.startswith('pegasus/'):
        category = 'Pegasus Legacy / Reference'
    elif rel_path.startswith('docs/'):
        category = 'Root Docs / Archive'
    elif '/' not in rel_path:
        category = 'Repository Root'

    categories[category].append({
        'path': rel_path,
        'size': size,
        'lines': lines_count,
        'title': title
    })

report_path = '/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_1/doc_inventory_report.md'

with open(report_path, 'w', encoding='utf-8') as out:
    out.write('# PegasusX Repository Documentation Inventory & Claims Report\n\n')
    out.write('**Generated:** 2026-08-20T17:25:00+05:00  \n')
    out.write('**Auditor:** Explorer 1 (`teamwork_preview_explorer_survey_1`)  \n')
    out.write('**Scope:** Full repository scan of all `.md` and `.docx` files in `/Users/shakhzod/Desktop/V.O.I.D`  \n')
    out.write(f'**Total Documents Cataloged:** {len(docs)} files across {len(categories)} categories  \n\n')
    out.write('---\n\n')
    
    out.write('## 1. Executive Summary\n\n')
    out.write(f'This report establishes a comprehensive inventory and claims audit of all documentation across the repository. The investigation cataloged **{len(docs)} project documents**, organized them into {len(categories)} distinct functional domains, and extracted all explicit status claims regarding implementation completeness, parity status ("Wired"), phase completion ("Done"), production/cloud readiness, scorecards, stubs, and residuals.\n\n')
    
    out.write('### Key Findings & Architectural Taxonomy\n\n')
    out.write('1. **Living Source of Truth Hierarchy**:\n')
    out.write('   - **Primary North Star**: `.agents/memory/GOAL.md` directing to destination programs `pegasusX/docs/GLOBAL_SCALE_PROGRAM.md` and `pegasusX/docs/GLOBAL_SCALE_LOCAL_ECOSYSTEM.md`.\n')
    out.write('   - **Living Monorepo**: `pegasusX/` is the active codebase and living specification root. `pegasus/` is a legacy reference/port source only.\n')
    out.write('   - **Documentation Map**: `pegasusX/docs/DOCS_SOURCE_OF_TRUTH.md` governs living vs frozen documentation.\n')
    out.write('   - **Core Status Matrix**: `pegasusX/docs/ROLE_ROW_PARITY_MATRIX.md` defines role-row client and backend parity across 6 core roles + Platform Admin.\n')
    out.write('   - **Living Scorecard & Program**: `pegasusX/docs/session-2026-08-13/SCORECARD.md` and `MASTER_10_10_EXECUTION_PROGRAM.md`.\n\n')

    out.write('2. **Frozen Word Exports (.docx) vs Living Markdown**:\n')
    out.write('   - **8 `.docx` files** exist across the workspace. All represent historical exports (notably the *End-Product Reality Report* v2 dated 2026-08-11 and 2026-08-13, and *Docs<->Code Alignment Status* dated 2026-08-12).\n')
    out.write('   - As mandated by `PegasusX_Reality_Report.README.md` and `DOCS_SOURCE_OF_TRUTH.md`, these `.docx` files are **frozen historical artifacts** and are strictly not to be used for active planning without verifying against current code.\n\n')

    out.write('3. **Status Claims Summary**:\n')
    out.write('   - **"Wired" Claims**: All 6 roles (Supplier, Retailer, Driver, Warehouse, Factory, Payload) plus Platform Admin are labeled as "Wired" for happy-path Class A execution in `ROLE_ROW_PARITY_MATRIX.md`.\n')
    out.write('   - **"Done" Phase Claims**: Phases 0 through G7 in `MASTER_10_10_EXECUTION_PROGRAM.md` and Gap Ledger items G1-A1 through G7-4 are marked "DONE" in `session-2026-08-13/GAP_LEDGER.md`.\n')
    out.write('   - **Scorecard Claims**: Living scorecard claims **10/10** across 9 layers and **9.5/10** on Fiscal/Legal readiness.\n')
    out.write('   - **Explicit Stubs & Residuals**: Documented in `RESIDUAL_REGISTER.md`, `PROD_READINESS_SEQUENCE.md`, and `ROLE_FEATURES_DOCS_VS_CODE.md`: Adyen/Stripe are stubs; Click/Payme unwired in execution routes; Soliq/EDS requires live PKCS#12 secrets; FCM requires owner credentials; Auto-Order place soak flip is OFF; Payout rail is Bank-File only; OR-Tools prod replicas = 0.\n\n')

    out.write('---\n\n')
    out.write('## 2. Category Breakdown & Document Counts\n\n')
    out.write('| Category | Document Count | Key Focus / Purpose |\n')
    out.write('|---|---|---|\n')
    for cat, items in sorted(categories.items(), key=lambda x: -len(x[1])):
        out.write(f'| {cat} | {len(items)} | Comprehensive documentation and specs for {cat} |\n')
    out.write('\n---\n\n')

    out.write('## 3. Detailed Document Inventory by Category\n\n')
    for cat, items in sorted(categories.items(), key=lambda x: -len(x[1])):
        out.write(f'### {cat} ({len(items)} files)\n\n')
        out.write('| File Path | Lines | Size (Bytes) | Header / Title |\n')
        out.write('|---|---|---|---|\n')
        for item in items:
            t = item['title'].replace('|', '-').replace('\n', ' ')
            out.write(f'| `{item["path"]}` | {item["lines"]} | {item["size"]} | {t} |\n')
        out.write('\n')

    out.write('---\n\n')
    out.write('## 4. Deep Claims Extraction & Parity Matrix Analysis\n\n')
    out.write('### 4.1. Role-Row Parity Claims (`ROLE_ROW_PARITY_MATRIX.md`)\n\n')
    out.write('Source: `pegasusX/docs/ROLE_ROW_PARITY_MATRIX.md` (Updated 2026-08-14)\n\n')
    out.write('| Role | Clients Specified | Backend Routes | UI Parity Claim | Implementation & Caveat Notes |\n')
    out.write('|---|---|---|---|---|\n')
    out.write('| **SUPPLIER** | portal (Tauri desktop), Android, iOS | `supplierroutes` + finance/claims/pulse + return-policy + planning | **Wired** | Desktop = `supplier-portal` Tauri; `/planning` web; CT scored+playbooks typed native; payout-policy thin UI, rail `no_live_rail`; negotiations product-deferred |\n')
    out.write('| **RETAILER** | desktop, Android, iOS | `retailerroutes`, order, payment, credit + Retail OS packs 0–6 | **Wired** | HQ / Credit-AR / CT on all 3; CT tiles navigate (P13-E); AUTHORIZE_BYPASS photo wired |\n')
    out.write('| **DRIVER** | Android, iOS | `driverroutes`, delivery, telemetry | **Wired** | P0-4 offline classifier fixed; PoD required for credit leave; §8.8 kit |\n')
    out.write('| **WAREHOUSE** | portal, Android, iOS | `warehouseroutes` + WMS + return-policy | **Wired** | Portal: bins/pick-waves/cycle/cold/labor; mobile pick/cycle under Transfer Actions; Control Tower typed scored list (P13-C) + portal |\n')
    out.write('| **FACTORY** | portal, Android, iOS | `factoryroutes` | **Wired** | Loading-bay start/seal **REAL** <-> payload Class A; factory **Payload/Load** factory-plane only. G7 SLA board + badges on portal. `POST /v1/factory/dispatch` live Spanner = warehouse solver class -> `FactoryTruckManifests` only. Staff POST + exception resolve are Class A persist + outbox. |\n')
    out.write('| **PAYLOAD** | Expo terminal + Android + iOS | `payloaderoutes` + factory manifests bridge | **Wired** | Seal/inject/reassign/returns; **seal-all** on terminal+Android+iOS (P13-A); factory loading-bay APIs on all three. Capacity 410. |\n')
    out.write('| **PLATFORM_ADMIN** | `admin-portal` (web only) | `platformadmin` + `featureflags` + partner admin | **Wired** | Login+MFA; tenants/flags dual-control/audit/match/partner; ops outbox + Spanner dead-letters; no mobile by design |\n\n')

    out.write('### 4.2. Cross-Role Spine & Realtime Status Claims\n\n')
    out.write('| Interaction Hop / Surface | Claimed Status | Evidence / Notes |\n')
    out.write('|---|---|---|\n')
    out.write('| Checkout -> Reserve -> Create | **Wired** | Atomic reservation, outbox event emit, ParentOrders split |\n')
    out.write('| Dispatch -> Manifest LOADED | **Wired** | Warehouse & Factory dispatch routes |\n')
    out.write('| Seal -> Depart IN_TRANSIT | **Wired** | Payload & Driver seal gates |\n')
    out.write('| Scan-QR -> Collect-Cash -> Fiscal -> COMPLETED | **Wired** | Real signature check, OFD fiscal stamp, AR pay-down in txn |\n')
    out.write('| Claim File -> Approve -> Chargeback + WS | **Wired** | Claims outbox event fanout, WS hub broadcast |\n')
    out.write('| Shop-Closed Cancel Inventory Release | **Wired** | Closed cancel releases reserved inventory (2026-07-31) |\n')
    out.write('| Factory Loading-Bay <-> Payload | **Wired (W2)** | Factory-plane load ledger bridge |\n')
    out.write('| Outbox -> Kafka -> Notification Dispatcher | **Wired** | Transactional outbox polling & Kafka publisher |\n')
    out.write('| Twin Consumer (`void-digital-twin`) | **Wired (W1)** | State sync to digital twin engine |\n')
    out.write('| FCM / Device-Token | **Env-dependent** | Push degraded / silent without owner credentials |\n')
    out.write('| Partner Webhooks / AS2 / EDI-lite | **Wired** | 1C + EDI profile packs; Drummond/SAP cert residual |\n\n')

    out.write('### 4.3. Living Scorecard Analysis (`session-2026-08-13/SCORECARD.md`)\n\n')
    out.write('| Architecture Layer | Baseline Score | Current Score Claim | Target Score | Phase Evidence |\n')
    out.write('|---|---|---|---|---|\n')
    out.write('| Go backend transactional core | 8.5 | **10** | 10 | G1+G7 Class A mutators + outbox |\n')
    out.write('| Domain model depth | 8.5 | **10** | 10 | G2+G7 Dual plane + load ledger + factory SLA |\n')
    out.write('| AI / forecast / optimization | 5.0 | **10** | 10 | G6 MAPE+demote, MEIO, CP_SAT honesty |\n')
    out.write('| Integration (API/EDI/export) | 6.0 | **10** | 10 | G5 1C+EDI+ASN; SAP/Drummond partner residual |\n')
    out.write('| Multi-tenancy (runtime) | 6.0 | **10** | 10 | G4 Seed fail-closed; OIDC optional residual |\n')
    out.write('| Retailer clients | 8.0 | **10** | 10 | G3+G7 Drift matrix Wired |\n')
    out.write('| Supplier / factory / WH clients | 7.5 | **10** | 10 | G2+G3+G7 Factory SLA board + badges |\n')
    out.write('| Driver / payload clients | 8.0 | **10** | 10 | G1.C+G2 Load ledger gate |\n')
    out.write('| Infra / operability | 5.5 | **10** | 10 | G4+G7 Outbox dead-letters browsable |\n')
    out.write('| Fiscal / legal readiness | 4.0 | **9.5** | 10 | G1.B Code default MY_SOLIQ+EDS; secrets cutover residual |\n\n')

    out.write('### 4.4. Program Phases & Gap Ledger Summary (`session-2026-08-13/`)\n\n')
    out.write('| Program Phase | Focus Area | Status Claim | Scorecard Delta / Notes |\n')
    out.write('|---|---|---|---|\n')
    out.write('| **Phase 0** | Control Plane | **DONE** | Baseline enable, proof harness |\n')
    out.write('| **Phase G1** | Money & Law | **DONE (A–D)** | Fiscal MY_SOLIQ default, AR in-txn pay-down, payout bank-file |\n')
    out.write('| **Phase G2** | Physical + Autonomy | **DONE (A–D; E partial)** | Stocklots outbox, load ledger, cold chain, auto-order place partial |\n')
    out.write('| **Phase G3** | Collections + Honesty | **DONE (A–D)** | Dunning status, risk scoring v1, POS scan-to-cart, GPS honesty |\n')
    out.write('| **Phase G4** | Tenancy + Ops | **DONE (A–C)** | Seed fail-closed in prod/ssmr, admin login, optimizer HEURISTIC label |\n')
    out.write('| **Phase G5** | Enterprise I/O | **DONE (A–D)** | EDI profiles, 1C import adapter + journals, ASN bidirectional |\n')
    out.write('| **Phase G6** | Brain | **DONE (A–D)** | Forecast MAPE demotion, MEIO cost-aware v2, CP_SAT alias honesty |\n')
    out.write('| **Phase G7** | Polish Re-Score | **DONE (1–4)** | Factory SLA board, dead-letters UI, features regen, scorecard 10s |\n\n')

    out.write('### 4.5. Explicit Residuals & Deploy-Time Prerequisites (`RESIDUAL_REGISTER.md`)\n\n')
    out.write('| Item | Classification | Reason / Status |\n')
    out.write('|---|---|---|\n')
    out.write('| **Soliq / EDS Secrets** | Ops / Legal Residual | Hard-fail provider wired; live PKCS#12 key & OFD contract pending procurement |\n')
    out.write('| **OR-Tools Optimizer Pods** | Ops Residual | Code reports HEURISTIC / OPTIMAL accurately; prod k8s replicas set to 0 |\n')
    out.write('| **Auto-Order Place Soak Flip** | Product / Ops Residual | Place flag stays false until live operational validation completes |\n')
    out.write('| **OIDC / External IdP** | Security Residual | `PreferTenant` fail-closed wired; external OIDC is optional adapter |\n')
    out.write('| **Drummond AS2 / SAP Cert** | Partner Residual | 1C + EDI-lite wired; third-party enterprise certifications pending partner |\n')
    out.write('| **FCM / APNs / SMS Credentials** | Ops Residual | Dispatcher and token paths wired; runs in degraded/silent mode without keys |\n')
    out.write('| **Substance Gate UI Walk** | QA Residual | Code `READY_FOR_WALK`; human visual sign-off required |\n')
    out.write('| **Draft i18n Review** | Localization Residual | Translation keys generated across 12 mobile + web apps; linguistic review pending |\n\n')

    out.write('---\n\n')
    out.write('## 5. Frozen Word Exports (.docx) Inventory & Analysis\n\n')
    out.write('| Document Path | Size | Text Extracted | Historical Context & Rule |\n')
    out.write('|---|---|---|---|\n')
    out.write('| `PegasusX_Reality_Report.docx` | 58,064 B | 51,326 chars | Root export v2 (11 Aug 2026). Frozen historical audit. Sidecar README forbids planning from it. |\n')
    out.write('| `pegasusX/artifacts/PegasusX_End_Product_Reality_Report_2026-08-13.docx` | 62,948 B | 46,667 chars | Artifact snapshot (13 Aug 2026 HEAD 29108a18). Comprehensive audit against source tree. |\n')
    out.write('| `pegasusX/artifacts/PegasusX_End_Product_Reality_Report.docx` | 21,018 B | 21,467 chars | Artifact export. Marked HISTORICAL / FROZEN. |\n')
    out.write('| `pegasusX/artifacts/DOCS_CODE_ALIGNMENT_STATUS_2026-08-12.docx` | 8,850 B | 1,650 chars | Alignment summary (12 Aug 2026). References living markdown SoTs. |\n')
    out.write('| `pegasusX/docs/DOCS_CODE_ALIGNMENT_STATUS_2026-08-12.docx` | 8,850 B | 1,650 chars | Duplicate living Word export in docs/. Markdown remains planning SoT. |\n')
    out.write('| `pegasusX/docs/session-2026-08-07/END_PRODUCT_REALITY_REPORT_2026-08-11.docx` | 13,068 B | 15,195 chars | Session export (11 Aug 2026). Marked historical/frozen. |\n')
    out.write('| `pegasusX/docs/session-2026-08-07/END_PRODUCT_REALITY_REPORT.docx` | 86,198 B | 83,829 chars | Complete session reality report (83k chars). Marked historical/frozen. |\n')
    out.write('| `pegasusX/docs/session-2026-08-07/END_PRODUCT_REALITY_REPORT_2026-08-13.docx` | 62,948 B | 46,667 chars | Session snapshot (13 Aug 2026). Sibling of artifact version. |\n\n')

    out.write('---\n\n')
    out.write('## 6. Synthesis & Downstream Recommendations for Survey Specialists\n\n')
    out.write('1. **Verification Priorities for Specialist 2 (Backend & Infra Explorer)**:\n')
    out.write('   - Verify Go backend routes against the 612 endpoints in `FEATURES_BY_APP_ROLE.md`.\n')
    out.write('   - Confirm Spanner schema mutations (`schema/spanner.ddl`) match claimed tables (`FactoryTruckManifests`, `Stocklots`, `EdiProfiles`, `LocalSKUs`, `SectionSKUs`).\n')
    out.write('   - Check outbox transactional atomicity in order creation, dispatch, payment collection, and claim approval.\n')
    out.write('   - Re-verify fail-closed behavior for `listLocalSKUs`, `HandleSectionByID`, and `me/sections` documented in `.agents/memory/WORKSPACE.md`.\n\n')

    out.write('2. **Verification Priorities for Specialist 3 (Client Apps & UI Explorer)**:\n')
    out.write('   - Inspect all 12 mobile apps (Android Compose & iOS SwiftUI across 6 roles) and 4 web/desktop portals (`admin-portal`, `supplier-portal`, `warehouse-portal`, `factory-portal`, `retailer-app-desktop`).\n')
    out.write('   - Confirm error state rendering on failed API calls (verifying that 500s do not render fake empty data or theatre).\n')
    out.write('   - Audit Control Tower typed scored lists, Retail OS packs 0–6 navigation, and seal-all button implementation.\n\n')

    out.write('3. **Documentation Updates & Alignment Rule**:\n')
    out.write('   - Never treat doc claims of "Wired" as production readiness without code proof.\n')
    out.write('   - Ensure all references to frozen `.docx` files point back to the living markdown files (`GLOBAL_SCALE_PROGRAM.md`, `GLOBAL_SCALE_LOCAL_ECOSYSTEM.md`, `ROLE_ROW_PARITY_MATRIX.md`).\n')

print(f'Report successfully generated at {report_path}!')
