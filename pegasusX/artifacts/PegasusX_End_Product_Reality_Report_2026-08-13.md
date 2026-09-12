**PegasusX / ATOMOS**

**End-Product Reality Report**

*What the system actually is today — verified against the live source tree,*

*what it can honestly replace, and what must still be built.*

Evidence date: 13 August 2026

Codebase SoT: /Users/shakhzod/Desktop/V.O.I.D/pegasusX (HEAD 29108a18)

Classification: internal — direct, non-promotional assessment

# Contents

0. Executive summary and scorecard

0.4 Method and evidence base

1. Human / field-agent displacement

2. Problem coverage vs. O9, Kinaxis, Blue Yonder, ERP+WMS stacks, B2B marketplaces

3. Alignment with systems big retailers and suppliers already run (1C, SAP, WMS, POS, accounting)

4. Does a true unified platform already exist?

5. Per-role, per-app, per-feature detail

5.1 Retailer (Android / iOS / Desktop)

5.2 Supplier (Portal / Android / iOS)

5.3 Warehouse (Portal / Android / iOS)

5.4 Factory (Portal / Android / iOS)

5.5 Driver (Android / iOS)

5.6 Payload / Loading (Terminal / Android / iOS)

5.7 Platform administration

5.8 Cross-cutting platform truth (money, orders, events, fiscal, deploy)

6. Recommendations and prioritized gap list (P0–P3)

Appendix A. Key evidence register (file:line)

# 0. Executive summary and scorecard

**What PegasusX is, verified in code:** a vertical B2B distribution / logistics OS with a deep physical spine: Go backend (~198k non-test LOC, ~748 route registrations, 183 Spanner tables) with same-txn outbox → Kafka, an enforced order state machine with ADR-009 fiscal hard-gate, COD/credit/card doorstep settlement, AR+dunning (flag-gated), partner M2M (OpenAPI + API keys + webhooks + EDI-lite + AS2 + SFTP + GS1 + 1C-dialect journals), mid-depth WMS (bins/lots/FEFO/pick-waves/cycle-counts, flag-gated), statistical replenishment + gated auto-order, and native apps/portals for retailer, supplier, warehouse, factory, driver, and payload that are overwhelmingly wired to live APIs.

**What PegasusX is not:** a field-sales CRM that replaces van sales agents; an enterprise APS (o9/Kinaxis/BY-class concurrent planning); a full ERP/MES; a certified EDIFACT/SAP drop-in; or a tax-legal fiscal system by default (FISCAL_PROVIDER defaults to PEGASUS commercial receipts with tax_ofd:false — MY_SOLIQ + EDS exist but are not the production default). Auto-order place mode and several WMS/cold-chain paths remain fail-closed behind flags. Admin is a break-glass console, not a full governance product.

## 0.1 Headline answers

| **Question** | **Answer (one line)** | **Detail** |
| --- | --- | --- |
| Can it replace field sales agents? | No wipe-out. ~45–55% of the commercial + collections loop is automatable in code today; physical last-mile + relationship creation + doorstep credit bargaining remain human. Trajectory is hybrid reduction, not elimination. | §1 |
| Does it cover O9/Kinaxis/Blue Yonder-class planning? | No. It has real statistical forecast models (Holt–Winters/Croston/SES), safety-stock math, greedy MEIO, scenarios, and OR-Tools/heuristic VRP — not concurrent enterprise APS. | §2 |
| Can a big retailer/supplier integrate without re-keying? | Partial. Partner API/EDI-lite/AS2/webhooks/1C journals/GS1/POS feed exist. No SAP IDoc/OData layer; EDI not certified; master-data + bidirectional WMS sync incomplete for chain adoption. | §3 |
| Does a true unified platform already exist elsewhere? | Pieces exist (RELEX replenishment, SFA/DMS, B2B marketplaces, ERP+WMS stitches). Few public systems hold factory→WH→payload→driver→retailer POS→AR/fiscal in one transactional fabric. PegasusX’s vertical depth is real; autonomy is gated, not touchless. | §4 |
| Is the codebase honest? | Transactional core and role apps are largely honest. Residual theatre: negotiation UI vs 410 API; credit score stub; default commercial fiscal; some admin/settings dead rows; flag-off capabilities that UIs still expose. | §5–§6 |

## 0.2 Scorecard (evidence-based, 2026-08-13)

| **Layer** | **Score** | **One-line justification** |
| --- | --- | --- |
| Go backend transactional core | 8.5/10 | Same-txn outbox, Version CAS, money idempotency indexes, ADR-009 gate; loses points for post-commit AR fail-open and unused TransitionOpts fields. |
| Domain model depth | 8.5/10 | Full delivery lifecycle, COD/credit/card, claims/returns, cash recon, multi-supplier parent orders, control-tower playbooks. |
| AI / forecasting / optimization | 5/10 | Real Holt–Winters/Croston/SES + safety-stock v2 + OR-Tools path; MEIO greedy; Rust “CP_SAT” is heuristic; auto-order place fail-closed. |
| Integration surface (API/EDI/export) | 6/10 | Partner OpenAPI (~1270 lines), keys/OAuth, webhooks+DLQ, EDI-lite, AS2, SFTP, GS1, 1C journals — not SAP-certified, not full GL sync. |
| Multi-tenancy (runtime) | 6/10 | PreferTenantSupplierID + RequireTenant exist; seed fallbacks remain when enforcement off; multi-supplier checkout present. |
| Retailer clients | 8/10 | Android/iOS/Desktop deep and wired (POS, auto-order, credit/AR, HQ); residual dead settings prefs + tracking GPS gaps. |
| Supplier / factory / warehouse clients | 7.5/10 | Portals + natives cover ops; WMS/cold-chain/labor gated; factory is logistics node not MES. |
| Driver / payload clients | 8/10 | Production-grade money edges + telemetry; payload seal real; mid-delivery update still not_implemented; state PATCH always 501. |
| Infra / operability | 5.5/10 | Spanner/Kafka/WS/FCM architecture real; prod optimizer replicas historically gated; FCM can no-op; admin thin. |
| Fiscal / legal readiness | 4/10 | Hard-gate machine works; default PEGASUS commercial; MY_SOLIQ+PKCS#12 EDS code exists; production cutover + credentials are the blocker. |

## 0.3 The five findings that matter most

- **1. Physical + money spine is real.**  Order SM (`order/state_machine.go`), cash/card/credit leave with payment legs + unique idempotency keys, fiscal consumer, cash recon, and role apps are wired — not a mock.

- **2. Default fiscal path is commercial, not tax OFD.**  `FISCAL_PROVIDER` defaults to PEGASUS (`order/fiscal_provider.go`); staging/prod overlays still set PEGASUS. MY_SOLIQ + EDS signer code exists but is not the live default.

- **3. Integration is no longer “absent” — certification and SAP are.**  Partner OpenAPI, EDI-lite, AS2, webhooks, GS1, 1C-dialect journal export are in tree. Big-chain adoption still needs certified EDI/SAP adapters and deeper master-data sync.

- **4. Autonomy is governed and mostly fail-closed.**  Auto-order place + soak gate (`retailer/auto_order_soak_gate.go`); WMS pick waves / cold chain env-gated; touchless replenishment policy-gated. Correct for prod honesty; means “touchless” is not the default product truth.

- **5. Field-agent wipe-out is not the product.**  No sales-rep role; credit terms set in supplier portal, not doorstep bargaining; pitch/relationship creation absent. Product is retailer self-serve + driver last-mile money + warehouse/factory execution.

# 0.4 Method and evidence base

Every claim was verified against the live source tree at /Users/shakhzod/Desktop/V.O.I.D/pegasusX (HEAD 29108a18, 2026-08-13). Three parallel code audits covered money/fiscal/orders/outbox, planning/integration, and per-role client apps. Prior markdown/docx exports and the gap register were used only as pointers to modules; every load-bearing claim was re-checked in code. Deleted documentation was not used as evidence.

Verdict vocabulary:

- **WIRED —** live path: real API calls / real DB writes on production request or worker paths.

- **HALF —** present but narrow, flag-gated off by default, or partially stubbed.

- **DECORATIVE —** UI/knobs that nothing meaningful acts on (or clients calling 410/501/stub backends).

- **ABSENT —** no code exists.

| **Surface** | **Measured size (this engagement)** |
| --- | --- |
| Go backend (apps/backend-go) | ~1203 .go files; ~198k non-test LOC; ~748 route registrations; ~1274 Test* funcs |
| Database | 183 CREATE TABLE in apps/backend-go/schema/spanner.ddl + dated migrations under schema/migrations/ |
| Native / web clients | retailer×3, supplier×2+portal, warehouse×2+portal, factory×2+portal, driver×2, payload×2+Expo terminal, admin-portal |
| Partner contract | contracts/partner.openapi.yaml (~1270 lines) + sdk/partner/go |
| Events / bus | Typed events + Spanner outbox relay → Kafka; role WS hubs; partner webhooks with DLQ |

# 1. Human / field-agent displacement

**The field agent's job, decomposed:** walk into stores; check the shelf; pitch products and promotions; negotiate quantity, price and credit; write the order; promise a delivery window; on delivery day negotiate exceptions; collect cash or get a signature for credit; and chase late payers. The honest question is which of those steps the code can perform today.

| **RISK —** PegasusX does not implement a field-sales / van-sales agent role. retailer/assist is floor help tickets, not sales CRM. Displacement claims must be framed as retailer self-serve + driver collections + planning automation — not agent wipe-out. |
| --- |

## 1.1 Step-by-step automatable share (verified against code)

*“Automatable today” means a flag, policy, sweeper or worker already exists in the tree. Flag-gated paths count as automatable only if the code path is complete when enabled.*

| **#** | **Agent task** | **Status in code today** | **Evidence** |
| --- | --- | --- | --- |
| 1 | Detect reorder need | AUTOMATED (statistical) — SBC classify + Holt–Winters/Croston/SES; demand sensing; reorder suggestion workers | planning/forecast/*; replenishment/reorder_suggestion_*; demand/worker_sensing.go |
| 2 | Retailer confirms / auto-places | HALF — shadow/draft/place modes; place fail-closed behind soak gate + AUTO_ORDER_PLACE_ENABLED=false in overlays | retailer/auto_order_worker.go; auto_order_soak_gate.go; .env.ssmr.example |
| 3 | Supplier accepts order | AUTOMATABLE — preorder sweeper SCHEDULED→AUTO_ACCEPTED; vet paths on supplier clients | order/state_machine.go:65-71; preorder_sweeper.go |
| 4 | Credit decision at placement | AUTOMATED but shallow — limit + status + reserved headroom; risk scoring stubbed empty | order/credit_guard.go; credit/repository.go (GetScoresForRetailers stub) |
| 5 | Price determination | AUTOMATED — override → promotion → dated price list; promo evaluator | order/service.go; promotion/evaluator.go |
| 6 | Stock allocation / backorder | AUTOMATED — reservation in create path; FEFO lots when WMS on | order/*reservation*; stocklots/fefo.go |
| 7 | Qty negotiation at door | DECORATIVE by default — clients have UI; API returns 410 unless QUANTITY_NEGOTIATION_ENABLED | order/negotiation_disabled.go; negotiation.go |
| 8 | Credit terms negotiation at door | ABSENT — terms/limits set in supplier credit policy portal, not stop-level bargain | credit/policy_handlers.go |
| 9 | Dispatch + driver assignment | AUTOMATABLE — auto-dispatch worker + optimizer-core (OR-Tools / Rust heuristic) | warehouse/auto_dispatch.go; services/optimizer-core/ |
| 10 | Physical picking | HUMAN instrumented — pick waves/tasks exist behind WMS_PICK_WAVES_ENABLED (default false in .env.example) | stocklots/handlers.go HandlePickWaves; bootstrap.go SetPickWavesEnabled |
| 11 | Truck loading | HUMAN by design — payload apps + seal gate (pick-wave ready assert when enabled) | payload/*; payload assertPickWaveReadyForSeal |
| 12 | Driving | HUMAN | driver apps + telemetry |
| 13 | Delivery handshake | HUMAN by design — QR + proximity/geofence unlock | order/proximity.go; driver QR flows |
| 14 | Offload & claims | HUMAN adjudicates; code wires partial offload + claims approve/reject/chargeback | order/partial_offload.go; claims/service.go |
| 15 | Cash collection (COD) | HUMAN structural — deepest money workflow (geofenced CollectCash, variance, cashrecon) | order/service.go CollectCash; cashrecon/* |
| 16 | Card capture | AUTOMATED for Global Pay path (prod refuses stub); Adyen/Stripe executors; Click/Payme webhooks without full execution router capture | payment/global_pay_executor.go; payment/execution.go |
| 17 | Fiscal receipt | SHAPE automated; legality default commercial — FISCALIZING→COMPLETED works; PEGASUS tax_ofd:false; MY_SOLIQ+EDS code exists | order/fiscal.go; fiscal_provider*.go; fiscal/signer_pkcs12.go |
| 18 | Credit leave + AR | AUTOMATED when AR_INVOICES_ENABLED — but AR open is post-commit fail-open after credit leave | order/driver_edges.go; ar/service.go |
| 19 | Dunning / collections | HALF — DunningWorker constructed + transports (Twilio SMS/WhatsApp, SendGrid); flag-gated; admin run-once + worker when enabled | ar/dunning_worker.go; ar/dunning_channels.go; bootstrap wires ARDunningWorker |
| 20 | Pitch / relationship creation | ABSENT as product surface | no sales-rep role; assist ≠ CRM |

## 1.2 The honest percentage

**~45–55% of the classical agent job is automatable with code as it stands (when relevant flags are on).** Commercial detection + pricing + credit limit check + dispatch + card/fiscal machine + AR/dunning plumbing are real. Physical steps 10–15 and relationship/credit bargaining are hard caps. Auto-order place and WMS pick are complete-enough code but not default-on product truth.

**~60–70% is a realistic near-term ceiling if P1 gaps close** (place flip after soak evidence, WMS pick default-on for chilled ops, dunning transports provisioned, AR co-atomic with credit leave, Soliq cutover). That still leaves hybrid reduction — not wipe-out.

## 1.3 What remains a hard human requirement

- **Physical execution.**  Picking, loading, driving, offload. Payload and driver are first-class apps by design.

- **Doorstep handshake + COD.**  QR/geofence/cash are intentional two-party controls; in COD markets the driver is the collections function.

- **First-mile trust.**  KYB, first credit grant, dispute de-escalation, and skeptical shopkeeper onboarding are not coded as agent CRM.

- **Credit judgment beyond limits.**  Scoring stubbed; non-mechanical exceptions stay human at the credit desk.

- **Fiscal / payout ops.**  Soliq credentialing, force-complete queues, bank-file payout MarkPaid remain human/ops.

## 1.4 Realistic trajectory

| **Horizon** | **What changes** | **What does not** |
| --- | --- | --- |
| Near-term (0–18 months) | Touchless routine replenishment for opted-in retailers after soak; dunning reduces chase headcount; WMS pick/scan reduces warehouse variance; partner EDI reduces re-key for friendly partners | Van sales for new SKU pitch / new door acquisition; cash handling; doorstep exception judgment |
| 3–5 years (if gaps closed) | Hybrid: inside sales + algorithms take most reorder pad work; field force shrinks to hunters + exception handlers; credit desk becomes exception desk | Complete wipe-out of field humans in COD/credit emerging markets is not credible — physical last mile and trust creation persist |

| **NOTE —** Honest product claim: reduce agent dependency for routine replenishment and digitize last-mile money — not “replace every field agent.” |
| --- |

# 2. Problem coverage vs. existing logistics / planning software

## 2.1 What the incumbent categories solve

| **Category** | **Core problems solved** | **Typical buyer** |
| --- | --- | --- |
| APS — o9, Kinaxis, Blue Yonder Luminate, SAP IBP, RELEX | ML/statistical demand, concurrent S&OP/IBP, MEIO solvers, scenario planning; RELEX-class automated replenishment at grocery scale | Large retail/CPG enterprises |
| ERP + WMS — 1C, SAP S/4 + EWM, Manhattan, Odoo | System of record: procurement, stock, GL, tax, statutory reporting; engineered warehouse ops | Mid-size+ distributors/retailers |
| SFA / DMS — FieldAssist, Botree class | Rep order pad, route plans, distributor management, retailer apps | Brands with field forces |
| B2B marketplaces — Udaan / MaxAB / Faire class | Many-to-many discovery, ordering, often credit/fintech rails | Independent retail |

## 2.2 What PegasusX actually solves today (code)

| **Problem class** | **PegasusX today** | **Verdict** |
| --- | --- | --- |
| Demand forecasting | SBC ADI/CV² classify; Holt–Winters (m=7), Croston-SBA, SES; holdout grid; residual bands; blocks short history | WIRED — statistical, not ML platform |
| Demand sensing | 14-day adjustments × velocity; promo multipliers; UZ calendar / payday signals; density worker | WIRED — Phase-1 shortcuts remain (REGION/CITY → all retailers) |
| Safety stock / ROP | SS = z·√(L·σd² + d̄²·σL²); ROP = d̄·L + SS; legacy burn·lead·1.15; SAFETY_STOCK_V2_ENABLED | WIRED |
| MEIO / network | Greedy surplus→deficit transfers + capital cap; not network LP | HALF vs BY/Kinaxis MEIO |
| S&OP / scenarios | Env-calibrated capacity vs supply-request demand; shock scenarios DRAFT/PUBLISH CAS | HALF — dashboards + heuristics, not concurrent APS graph |
| Routing / VRP | OR-Tools Python path + Rust NN/2-opt (HEURISTIC); auto-dispatch worker | WIRED for domain; “CP_SAT” name is greedy+swap |
| WMS execution | Bins, lots, FEFO, putaway, pick waves, cycle counts, adjustments, cold-chain readings — flag-gated | WIRED mid-depth; not Manhattan/EWM breadth |
| Factory / manufacturing | Transfers, supply requests, loading bay seal, fleet/staff — logistics node | ABSENT as MRP/MES/BOM |
| OMS / B2B ordering | Catalog, cart, unified/multi-supplier checkout, EDI ORDERS, auto-order | WIRED vertical network (not general marketplace) |
| Last-mile money + fiscal shape | COD/credit/card + cash recon + AR + fiscal hard-gate | WIRED — rare depth vs pure planning suites |
| Partner I/O | OpenAPI, keys, webhooks, EDI-lite, AS2, SFTP, GS1, 1C journals | WIRED baseline; not certified SAP/EDI |

## 2.3 Where it falls short of incumbents

- **vs APS:**  no concurrent planning graph, no enterprise solver MEIO, S&OP capacities env-scaled, auto-order place not default-on.

- **vs ERP:**  AR + journal export ≠ full GL/AP/FA/payroll/tax suite; factory ≠ manufacturing ERP.

- **vs enterprise WMS:**  no slotting optimization, yard/dock appointment system, labor standards engineering, 3PL billing complexity.

- **vs SFA:**  no field-rep pitch/route-call CRM; quantity negotiation disabled by default.

- **vs marketplaces:**  supplier–retailer network with native apps, not broad discovery/bidding marketplace economics.

## 2.4 Vertical depth advantages that are real in code

- **One transactional fabric factory → WH → payload → driver → retailer stock/POS → demand → reorder.**  Shared Spanner entities + outbox events — not a BI stitch.

- **Multi-role native execution apps.**  Operational roles have real screens wired to `/v1/...`, not portal-only demos.

- **Money + physical evidence in one model.**  Payment legs, fiscal attempts, proximity, cash recon, claims/chargebacks.

- **Governed autonomy.**  Soak gates, dual-control flags, playbook auto-safe allowlists — honest for COD/credit markets.

| **NOTE —** PegasusX competes as a vertical execution + transaction OS with planning assist — not as a Kinaxis/o9 replacement. Advantage is closed-loop physical+financial depth; shortfall is APS math and certified ERP connectors. |
| --- |

# 3. Alignment with systems already used by big retailers and suppliers

**Bottom line:** PegasusX has a real machine-to-machine partner surface (API keys/OAuth, OpenAPI, webhooks, EDI-lite, AS2, SFTP, GS1, POS demand feed, 1C-dialect journal export). It does not yet offer certified SAP/EDIFACT drop-ins or bidirectional master-data sync sufficient for large chains to stop re-keying.

## 3.1 Current M2M integration (wired)

| **Surface** | **What exists in code** | **Evidence** |
| --- | --- | --- |
| Partner API | Scoped routes: orders, catalog, inventory, demand, webhooks, exports, EDI, COA, AS2; sandbox keys pxs_* | partner/routes.go; partner/keys.go; contracts/partner.openapi.yaml |
| Auth | OAuth client_credentials + API keys + scope checks | partner/types.go scopes; RequirePartner |
| Outbound webhooks | Subscribe/ping/rotate/DLQ/replay; URL SSRF protections; signed delivery | partner/handlers.go; partner/webhook_url.go |
| EDI-lite | ORDERS/ORDRSP/DESADV/INVOIC + CONTRL/APERAK + PRICAT/INVRPT/SLSRPT/RECADV/ORDCHG/DELFOR/REMADV — explicitly not certified EDIFACT | partner/edi/*; partner/edi/breadth.go |
| AS2 | RFC 4130 receive + outbound Send with MDN MIC checks | partner/routes.go /as2; partner/as2/* |
| SFTP | Partner SFTP config + export upload status | partner/routes.go admin SFTP |
| Exports | orders/invoices/inventory/ledger/journals as CSV/JSON/XML | partner/types.go export kinds |
| 1C-style accounting | Default COA 62.01/90.01/51.01; journals XML dialect="1c"; AR/payment/CN mapping | partner/coa.go; export_journals.go; export_worker.go |
| POS sell-through | POST /demand/pos-feed → DEMAND_SIGNAL; native retailer POS also writes sell-through | partner/pos_demand_feed.go; retailer sell_through |
| GS1 | GTIN/GLN/SSCC, GS1-128 ZPL, ECC200 DataMatrix with FNC1 AI string | gs1/* |
| SDK | Generated Go partner client | sdk/partner/go |

## 3.2 Where it falls short for big players

| **System** | **Gap** |
| --- | --- |
| SAP | No IDoc/OData/RFC/BAPI connector packages; no SAP-specific mapping layer found in backend-go. |
| Certified EDI | Code states EDI-lite is not certified EDIFACT/Drummond. |
| External WMS | Partner inventory upsert/availability only — not Manhattan/SAP EWM adapters. |
| External POS chains | Generic POS feed API; no Toast/Square/1C-Retail vendor connectors. |
| Full GL sync | Journals export (AR/payments/CN) — not bidirectional ERP master/GL. |
| Tax e-factura | MY_SOLIQ OFD adapter + VAT snapshots; not a full ЭФактура lifecycle product. |

## 3.3 What must exist for adoption without re-keying

#### Certified document map (EDI or SAP IDoc)

**Purpose:** Round-trip ORDERS↔ORDRSP↔DESADV↔INVOIC with partner’s certified dialect.

**Why it is needed:** Chains will not abandon SAP/1C order entry for a proprietary app alone.

**Math / algorithm / logic:** Idempotent upsert by partner document ID; ACK/NACK; semantic mapping tables per tenant; MIC/MDN for AS2.

**End-to-end behavior:** Partner ERP emits ORDERS → Pegasus creates order → ORDRSP → DESADV on seal → INVOIC/journals on fiscal/AR → partner GL posts without human re-key.

`Anchors: partner/edi/* + new SAP adapter package`

#### Master-data sync (customers, GTIN, GLN, price lists, plants)

**Purpose:** Keep Pegasus and ERP item/party truth aligned.

**Why it is needed:** Without it, every SKU/price/customer change becomes dual maintenance.

**Math / algorithm / logic:** Conflict rules: source-of-truth per field; version/CAS; dead-letter on unmapped GTIN.

**End-to-end behavior:** Nightly + event-driven PRICAT/INVRPT (already partial) expanded to parties/plants.

`Anchors: partner/edi inbound PRICAT/INVRPT handlers`

#### Inventory & ASN bidirectional with external WMS

**Purpose:** Avoid double warehouse truth.

**Why it is needed:** Large suppliers already run WMS; Pegasus must either be the WMS or sync.

**Math / algorithm / logic:** Availability upsert + DESADV packing ↔ putaway/FEFO; variance codes.

**End-to-end behavior:** External WMS ASN → Pegasus receive → or Pegasus DESADV → partner WMS.

`Anchors: partner inventory + DESADV paths`

#### Accounting: certified 1C/SAP FI profiles + Soliq cutover

**Purpose:** Money and tax land in the books of record.

**Why it is needed:** Journal XML + COA mapping is a start; production tax OFD must be MY_SOLIQ default where required.

**Math / algorithm / logic:** Journal entry generation from AR/payment events (exists); FISCAL_PROVIDER=MY_SOLIQ with EDS.

**End-to-end behavior:** Credit leave → AR → journals export → 1C import; card/cash → fiscal → ledger.

`Anchors: partner/export_journals.go; order/fiscal_provider.go`

| **INCOMPLETE / HALF —** M2M baseline is real and shippable for friendly partners. Enterprise procurement still fails on SAP/certified EDI and default commercial fiscal. |
| --- |

# 4. Existence of a true unified platform

**Ideal stated in the brief:** quality suppliers and retailers on one transactional platform; near-zero human interaction for routine replenishment; still supporting physical execution roles (pick, load, drive, receive, collect).

## 4.1 Do public systems already do this?

| **Class** | **What they typically deliver** | **Gap vs the ideal** |
| --- | --- | --- |
| RELEX / APS replenishment | High-automation store ordering for large grocers | Usually not factory→payload→COD/credit/fiscal in one fabric |
| SFA/DMS + ERP stitch | Rep digitization + 1C/SAP SoR | Humans still re-key across systems; physical apps fragmented |
| B2B marketplaces | Discovery + order + sometimes credit | Weak last-mile physical + warehouse execution ownership |
| ERP+WMS+TMS suites | Deep modules per layer | Integration project, not one transactional OS with native role apps |

Honest market read: the category of “touchless replenishment” exists in pockets (especially grocery APS). A single public system that also owns COD/credit doorstep money, fiscal hard-gate, payload seal, and factory loading bay as first-class native apps is rare. PegasusX’s claim to uniqueness is the vertical transactional chain — not the idea of auto-reorder.

## 4.2 PegasusX position vs the ideal (from code)

| **Ideal element** | **Code reality** | **Verdict** |
| --- | --- | --- |
| Suppliers + retailers on one network | Catalog, trading, multi-role apps, multi-supplier checkout/parent orders | WIRED |
| Near-zero human for routine replenishment | Touchless factory transfers (policy); retailer auto-order shadow/draft/place with soak fail-closed; place off in overlays | HALF |
| Physical execution supported | WMS pick/putaway/counts; factory loading bay; payload seal; driver money edges; retailer dock | WIRED |
| Closed-loop demand | POS/sell-through → DEMAND_SIGNAL → sensing → reorder suggestions → auto-order | WIRED (quality depends on POS feed fidelity) |
| Money + fiscal closed loop | Cash/card/credit → legs → FISCALIZING → COMPLETED; AR/dunning flag-gated; default fiscal commercial | HALF for legal/tax markets |
| Enterprise no-rekey | Partner M2M baseline; not SAP-certified | HALF |

## 4.3 Clear comparison verdict

- **Closer than a thin marketplace or SFA app**  because physical roles and money state machines are first-class.

- **Farther than RELEX-class grocery automation**  on measured touchless place rates — place mode is intentionally locked.

- **Farther than SAP+EWM certified stacks**  on ERP/WMS interchange and statutory depth.

- **Correct strategic posture**  governed autonomy + vertical OS, not “AI replaces everyone.”

| **WIRED / WORKING —** A unified operational platform exists in this codebase as a vertical fabric. A true touchless, enterprise-integrated, tax-legal default product does not — those are residual gates, not missing architecture. |
| --- |

# 5. Per-role, per-app, per-feature detail

*Verdict vocabulary: WIRED / HALF / DECORATIVE / ABSENT. Clients audited under apps/*; backend under apps/backend-go. Evidence date 2026-08-13, HEAD 29108a18.*

## 5.1 Retailer — Android / iOS / Desktop

### What exists and works today

- **Clients.**  retailer-app-android (Compose), retailer-app-ios (SwiftUI), retailer-app-desktop (Next.js 15 + Tauri 2).

- **Ordering.**  Catalog, cart sync, checkout (cash/card/unified), orders, claims, dock/QR handoff, tracking/WS.

- **Store ops.**  POS (open/close/sales/holds — holds pilot-gated), shifts, sections, stock, offline cash queue.

- **Autonomy / credit.**  Auto-order settings/runs/shadow/soak-gate APIs; Credit & AR + HQ multi-store screens on mobile and desktop.

- **Ops surfaces.**  Control tower, reports/export, pulse, assist tickets, FCM device token, notifications.

`Anchors: retailer-app-android/.../PegasusApi.kt; retailer-app-ios APIClient; retailer-app-desktop RetailerShell; retailerroutes/routes.go`

### Incomplete / decorative / weak

| **Issue** | **Status** | **Evidence** |
| --- | --- | --- |
| Settings General / Notifications prefs rows | DECORATIVE | SettingsSection.kt onClick = { } |
| Analytics week-nav buttons | DECORATIVE | WeeklySpendCard.kt no-ops |
| Live tracking without GPS trail | HALF | Desktop tracking copy admits driver location missing |
| Auto-order place | HALF (fail-closed) | AUTO_ORDER_PLACE_ENABLED=false; soak gate |
| POS camera barcode scan-to-cart | ABSENT / weak | Warehouse/payload have scanners; retailer POS is search/price entry |

### Missing / weak features to implement

#### R1. Notification preferences UI

**Purpose:** Persist push/email/SMS prefs already modeled server-side.

**Why it is needed:** Dead settings rows erode trust.

**Math / algorithm / logic:** GET/PATCH /v1/user/notification-preferences; honor quiet hours in dispatcher.

**End-to-end behavior:** Toggle → persist → FCM/inbox respects prefs.

`Anchors: notifications + retailer settings`

#### R2. Honest live tracking fallback

**Purpose:** ETA / last-known / awaiting telemetry states.

**Why it is needed:** Blank map while delivery active is operationally false.

**Math / algorithm / logic:** Prefer WS telemetry; else last Spanner ping; else route geometry remaining distance.

**End-to-end behavior:** Driver depart → map updates; GPS off → explicit state.

`Anchors: telemetryroutes + retailer tracking pages`

#### R3. Auto-order place flip (ops, not greenfield)

**Purpose:** Touchless replenishment for retailers that pass soak.

**Why it is needed:** Code path exists; product truth is shadow/draft until evidence.

**Math / algorithm / logic:** 30-day WAPE/unmodified-rate gate; dual-control AUTO_ORDER_PLACE_ENABLED.

**End-to-end behavior:** Shadow ≥ threshold → approve flag → place mode writes real orders via reservation/pricing/credit.

`Anchors: retailer/auto_order_soak_gate.go`

## 5.2 Supplier — Portal / Android / iOS

### What exists and works today

- **Ops.**  Dashboard/pulse, order vet/reassign/bypass, manifests start-loading/inject/seal, dispatch preview/execute, fleet live map, exceptions/claims.

- **Commercial.**  Catalog/pricing/promotions, inventory import, credit desk/collections, treasury/ledger/chargebacks/credit notes.

- **Planning.**  Planning brain / S&OP / scenarios, replenishment policies, analytics, AI recommendations.

- **Network.**  Warehouses, factories, zones, topology, supply lanes, partner webhook/key surfaces via admin/supplier routes.

### Incomplete / decorative / weak

| **Issue** | **Status** | **Evidence** |
| --- | --- | --- |
| Quantity Negotiations screens | DECORATIVE by default | API 410 unless QUANTITY_NEGOTIATION_ENABLED |
| Credit risk scores | STUB | credit/repository.go GetScoresForRetailers empty |
| Control tower playbooks UI | HALF | CONTROL_TOWER_PLAYBOOKS_ENABLED gated |
| Earnings/payments settlement | HALF | Portal falls back when settlement endpoint missing |
| Payout to suppliers | HALF | Bank-file rail only; no live bank API (payout/rail.go) |

#### S1. Quantity negotiation — enable or delete UI

**Purpose:** Driver proposes qty change; supplier resolves.

**Why it is needed:** Clients still navigate to Negotiations while API returns 410.

**Math / algorithm / logic:** Propose → pending → accept/reject → adjust reservation + total → outbox; timeout sweeper.

**End-to-end behavior:** Driver propose → supplier list → resolve → retailer totals update — or remove UI.

`Anchors: order/negotiation*.go`

#### S2. Credit risk scoring (or stop implying scores)

**Purpose:** Rank/auto-limit retailers for collections desk.

**Why it is needed:** Stub returns empty; desk uses limit/balance/delinquency only.

**Math / algorithm / logic:** risk = w1·util + w2·delinquency + w3·DPD + w4·(1−pay_velocity); freeze if risk≥T_high.

**End-to-end behavior:** Nightly scores → collections sort → optional auto-hold → notify.

`Anchors: credit/repository.go; ar/dunning_worker.go`

#### S3. Live payout rail or honest bank-file UX

**Purpose:** Move supplier funds or clearly operate as CSV→bank→MarkPaid.

**Why it is needed:** railByName always BankFileRail; live dispatch ErrNoLiveRail.

**Math / algorithm / logic:** Batch net = Σ settlements − fees − chargebacks; export file; webhook/MarkPaid.

**End-to-end behavior:** Generate batch → download → bank processes → settlement webhook/MarkPaid → PAYOUT_BATCH_PAID.

`Anchors: payout/rail.go; payout/store.go`

## 5.3 Warehouse — Portal / Android / iOS

### What exists and works today

- **Clients.**  warehouse-portal (Next+Tauri), warehouse-app-android, warehouse-app-ios.

- **WMS core.**  Inventory/bins/putaway, pick-waves (confirm/waive shorts), cycle counts (+ABC), adjustments, FEFO lots — backend stocklots handlers.

- **Dispatch.**  Preview/execute/rescue, fleet live map, auto-dispatch worker when enabled, labor-capacity surfaces.

- **Planning adjacent.**  Supply requests, replenishment insights, demand forecast views, preorders/tomorrow board.

- **Exceptions.**  Returns/reverse logistics, claims, cold-chain screens, control tower (portal).

`Anchors: warehouseroutes/routes.go; stocklots/handlers.go; warehouse-portal app/*; WarehouseApi.kt`

### Incomplete / flag-gated

| **Issue** | **Status** | **Evidence** |
| --- | --- | --- |
| Pick waves / cycle counts | HALF — code complete, env default false in .env.example | WMS_PICK_WAVES_ENABLED / WMS_CYCLE_COUNTS_ENABLED |
| Cold chain | HALF — UI warns flag must be on | WMS_COLD_CHAIN_ENABLED; ColdChainScreen |
| Labor → dispatch hard coupling | HALF | labor-capacity APIs exist; execute must refuse overload |
| Enterprise WMS breadth | ABSENT | No slotting/yard/labor-standards engine |

#### W1. Production default for pick waves on chilled/high-SKU tenants

**Purpose:** Seal only after pick confirmation ledger.

**Why it is needed:** Without waves, payload seal truth is weaker; variance hides until doorstep.

**Math / algorithm / logic:** Create wave from manifest → confirm tasks (scan) → waive shorts with reason → seal gate assertPickWaveReadyForSeal.

**End-to-end behavior:** Dispatch → wave → pick confirm → payload seal → driver depart.

`Anchors: stocklots/*; payload assertPickWaveReadyForSeal`

#### W2. Cold-chain always-on for temp-sensitive SKUs

**Purpose:** Auto-quarantine on excursion.

**Why it is needed:** Flag-off means chilled product can flow without sensor enforcement.

**Math / algorithm / logic:** If temp∉[min,max] → quarantine lots + breach exception; release via resolve.

**End-to-end behavior:** Ingest reading → breach → block pick/dispatch → resolve → re-enter inventory.

`Anchors: stocklots cold-chain; warehouse cold-chain UI`

## 5.4 Factory — Portal / Android / iOS

### What exists and works today

- **Logistics node features.**  Factories CRUD, internal transfers create/transition, supply-request accept/fulfill, loading bay start-loading/seal, manifest dispatch/complete/rebalance, fleet/staff, replenishment insights.

- **Payload merge.**  RolePayload allowed on factory loading-bay routes; terminal/natives can list/start/seal factory manifests.

`Anchors: factoryroutes/routes.go; factory-portal; FactoryApi.kt; factory-app-ios Views/*`

### Incomplete / absent

- **Not MES/MRP.**  No BOM, work orders, finite capacity production scheduling, yield/scrap manufacturing core.

- **Optimizer “factory slot”.**  Named CP_SAT path is greedy+swap heuristic in Rust sidecar.

- **Cold-chain desk.**  Weaker than warehouse; no factory temp ingest UI parity.

#### F1. Cross-dock handoff SLA board

**Purpose:** Factory→warehouse→payload timing visibility.

**Why it is needed:** Loading bay references handoff; needs durable SLA metrics.

**Math / algorithm / logic:** SLA clock from transfer create → seal → WH receive; late = now > ETA + slack.

**End-to-end behavior:** Transfer created → ETA → late flag → ops alert.

`Anchors: factory transfers + payload seal timestamps`

## 5.5 Driver — Android / iOS

### What exists and works today

- **Delivery OS.**  Manifest, map, QR scan, arrive, partial offload, cash collect, payment waiting, shop-closed, credit leave/delivery, early-complete, rescue/reorder, split-payment clients.

- **Offline + GPS.**  Offline queue + Room telemetry; FusedLocation ~10s adaptive; WS + FCM token.

- **Backend edges.**  Real OrderService-mounted edges (arrive/offload/cash/credit/…) — production panics if OrderService nil.

`Anchors: DriverApi.kt; driverroutes/routes.go; order/service.go CollectCash/CompleteOrder; order/driver_edges.go`

### Incomplete / intentionally fail-closed

| **Issue** | **Status** | **Evidence** |
| --- | --- | --- |
| PATCH /v1/orders/{id}/state | Always 501 | driver/mobile_compat.go |
| Mid-delivery order update | not_implemented | delivery_handshake.go |
| Quantity negotiation | 410 default | negotiation_disabled.go |

#### D1. Remove client calls to PATCH …/state

**Purpose:** Stop calling a hard-501 path.

**Why it is needed:** Confuses clients and support.

**Math / algorithm / logic:** Use edge routes only (arrive/depart/collect/offload).

**End-to-end behavior:** Delete state-patch from DriverApi clients.

`Anchors: mobile_compat.go:374-393`

#### D2. Durable mid-delivery line update

**Purpose:** Adjust lines at stop without full amend theatre.

**Why it is needed:** UpdateOrderDuringDelivery fail-closed today.

**Math / algorithm / logic:** Validate assignment + proximity → apply deltas in Spanner txn → outbox → new adjusted_total.

**End-to-end behavior:** ARRIVED → update lines → cash/card/fiscal use new total.

`Anchors: delivery_handshake.go`

## 5.6 Payload / Loading — Terminal / Android / iOS

### What exists and works today

- **Three clients.**  payload-app-android, payload-app-ios, payload-terminal (Expo) — trucks, pulse, manifests (supplier + factory paths), start loading, checklist/barcode check, seal, inject, exceptions, returns scan, WS, offline queue, FCM.

- **Seal gates.**  Backend assertPickWaveReadyForSeal when pick waves enabled; factory loading-bay merge for RolePayload.

`Anchors: PayloadApi.kt; payloaderoutes; factoryroutes loading-bay; payload-terminal ManifestWorkspaceScreen`

### Incomplete / weak

- **Barcode check ≠ full pick confirmation ledger.**  Lookup + “SKU on order?” rather than scanned_qty ≥ required_qty for every line.

- **Triple-client drift risk.**  Android / iOS / Expo implement the same flows separately.

#### P1. Line-level load confirmation ledger

**Purpose:** Prove each unit/case scanned before seal.

**Why it is needed:** Current scan only validates catalog membership on order.

**Math / algorithm / logic:** required_qty[sku], scanned_qty[sku]; seal iff ∀sku scanned≥required (or variance approved).

**End-to-end behavior:** Start loading → scan lines → seal blocked until complete → sealed event → driver depart.

`Anchors: payload HomeViewModel barcode check; stocklots pick confirm`

## 5.7 Platform administration — admin-portal

### What exists and works today

- **Break-glass console.**  Paste PLATFORM_ADMIN bearer; MFA enroll/confirm/verify; tabs for Tenants (KYB), Flags (set/approve dual-control), Audit, Product match queue, Partner keys/AS2/SFTP/COA, AR dunning run-once; admin WS refresh.

`Anchors: apps/admin-portal/app/page.tsx; components/*Panel.tsx; lib/api.ts; mfa/; featureflags/handlers.go`

### Incomplete

- **Not an end-user admin product.**  No normal login UX, no user management, thin billing analytics, no outbox/DLQ ops UI.

- **Dunning run-once is manual in UI.**  Worker exists when AR_DUNNING_ENABLED; console is break-glass trigger.

#### A1. Real admin identity (SSO/IdP + MFA)

**Purpose:** Replace token paste with durable break-glass identity.

**Why it is needed:** MFA exists; login does not — auditability gap.

**Math / algorithm / logic:** IdP OIDC → MFA step-up → short-lived PLATFORM_ADMIN session; audit every mutation.

**End-to-end behavior:** Login → MFA → scoped session → flag/tenant mutations audited.

`Anchors: admin-portal; packages/enterprise Auth0 config`

## 5.8 Cross-cutting platform truth (money, orders, events, fiscal)

### Order state machine — WIRED

Canonical graph enforced in ValidateStatusTransition (order/state_machine.go). ADR-009: ARRIVED cannot soft-complete; money paths enter FISCALIZING; fiscal consumer completes or fails. Actors: admin/retailer UpdateStatus (graph-only), driver money edges, warehouse transitions, fiscal worker, force-complete (role-gated).

| **INCOMPLETE / HALF —** TransitionOpts fields (PhotoURL/SupervisorToken) unused inside ValidateStatusTransition; OrderStatusTransitions timeline is best-effort separate Apply — not same-txn. |
| --- |

### Money path — WIRED with residual fail-open

| **Capability** | **Verdict** | **Notes** |
| --- | --- | --- |
| Cash CollectCash + cashrecon | WIRED | Geofence/proximity; payment legs cash-<orderId>; variance math |
| Card CompleteOrder / Global Pay | WIRED | Prod refuses GLOBAL_PAY_STUB_MODE |
| Credit leave + reserve | WIRED | DELIVERED_ON_CREDIT; credit-leave-<orderId> leg |
| AR open / pay-down | HALF | Post-commit after credit leave / cash — log-only on failure |
| Dunning | HALF | Worker+transports exist; flags off in base configmap; on in staging/SSMR examples |
| Payout | HALF | Bank-file only; ErrNoLiveRail for live dispatch |
| Escrow / wallet / AP | ABSENT | No modules |

### Fiscal — HARD-GATE WIRED; LEGAL DEFAULT HALF

- **Machine.**  FISCALIZING ↔ COMPLETED/FISCAL_FAILED; MY_SOLIQ adapter + PKCS#12 EDS signer code; FAKE for SSMR.

- **Default.**  FISCAL_PROVIDER=PEGASUS → platform commercial receipt, tax_ofd:false (fiscal_provider_pegasus.go). Staging/prod overlays still PEGASUS.

- **Buyer acceptance.**  MySoliq SUCCESS can stamp BuyerAcceptanceStatus + poller; parallel to ADR-009 complete.

### Outbox / concurrency — WIRED with known gaps

- **Same-txn outbox doctrine.**  outbox.TxnBuffer / SpannerTxnBuffer; relay → Kafka; DLQ; Redis event dedup on consumers.

- **Idempotency.**  Order Version CAS; OrderPaymentLegs + PaymentLedger unique indexes (20260816_*); HTTP Idempotency-Key on money edges.

- **Gaps.**  AR open/pay-down and some credit ClearBalance post-commit fail-open; some EmitJSON ignored errors on timeline/shop-closed paths.

### Cross-cutting client infra

| **Capability** | **Reality** |
| --- | --- |
| WebSockets | Role hubs + /v1/ws session mint; used across portals and natives |
| FCM | Device-token registration; InitFCM can fall back to no-op; LogTransport logs when unwired |
| Offline | Strongest on driver/payload/retailer POS; money edges must map 1:1 to OrderService handlers |
| Auth tenancy | PreferTenantSupplierID + RequireTenant; seed fallback when enforcement off |

# 6. Recommendations and prioritized gap list

**Strategy in one paragraph:** Architecture (Spanner → same-txn outbox → Kafka → WS/FCM/webhooks) is already right — do not redesign the bus. Remaining work is legality/default fiscal cutover, money bookkeeping atomicity, autonomy evidence flips, WMS defaults for chilled ops, certified enterprise connectors, and deleting client theatre that points at 410/501/stub backends. Stop adding surfaces until P0/P1 close.

## 6.1 Scope / architecture / prioritization changes

- **Keep vertical OS scope; drop field-agent wipe-out messaging.**  Product is self-serve + driver money + execution apps.

- **Treat flags as product truth.**  If pick waves / cold chain / auto-order place / MY_SOLIQ are required for a tenant class, ship them default-on for that class — or hide UI.

- **Partner layer before marketplace claims.**  Certify EDI/1C profiles and add SAP mapping before multi-supplier discovery marketing.

- **Bank-file payout is an accepted prod bar only if UX/ops runbook is first-class.**  Do not imply live bank rails.

- **Co-atomic AR with credit leave.**  Post-commit fail-open is a structural money bug class.

## 6.2 P0 — correctness, legality, money integrity

*Ordered by severity. Evidence verified 2026-08-13.*

| **#** | **Item** | **Why P0** | **Effort** |
| --- | --- | --- | --- |
| P0-1 | Production fiscal cutover: provision EDS + MY_SOLIQ creds; set FISCAL_PROVIDER=MY_SOLIQ for tax markets; keep PEGASUS only as explicitly labeled commercial path | Default tax_ofd:false completions are legally incomplete in UZ tax markets | M (ops+config) / S (code already) |
| P0-2 | Make AR OpenInvoice same-txn with credit leave; AR pay-down / credit ClearBalance same-txn with cash/card settle (or compensating saga with alert+block) | Delivery can succeed with missing AR/credit books — silent financial drift | M |
| P0-3 | Ship or delete negotiation UI; stop client calls to PATCH …/state (501); stop implying credit scores while stub empty | Product theatre / support lies | S |
| P0-4 | Ensure FCM/transports not silently LogTransport/no-op in prod profiles; alert when push degraded | Collections + ops notifications fail closed invisibly | S |
| P0-5 | Payout honesty: bank-file MarkPaid runbook + UI; or implement one live rail with IsLive fail-closed (already patterned) | Supplier money-out incomplete / misrepresented | M |
| P0-6 | Implement mid-delivery update or remove API surface advertised to drivers | Fail-closed not_implemented on a money-adjacent path | M |

## 6.3 P1 — structural product truth

| **#** | **Item** | **Closes** | **Effort** |
| --- | --- | --- | --- |
| P1-1 | Auto-order soak evidence → place flip for opted-in retailers (dual-control) | Touchless replenishment claim | M ops |
| P1-2 | WMS pick waves + cycle counts default-on for warehouse tenants that seal trucks; line-level payload scan ledger | Physical truth before seal | M |
| P1-3 | Cold-chain default-on for chilled SKUs; couple labor-capacity to dispatch execute | Chilled/ops integrity | M |
| P1-4 | Dunning: provision Twilio/SendGrid/WhatsApp in prod; AR_DUNNING_ENABLED on with dual-control; retailer mobile collections UX completeness | Agent collections displacement | M |
| P1-5 | Credit risk scoring v1 (or remove score UI) + delinquency already bumped by dunning | Credit desk automation | M |
| P1-6 | Admin identity beyond token paste (SSO/IdP); outbox/DLQ visibility | Operability | M |
| P1-7 | Retailer tracking fallback + kill dead settings rows; POS scan-to-cart if POS is end-product | Retailer honesty | S–M |

## 6.4 P2 — planning quality

| **#** | **Item** | **Math / logic core** | **Effort** |
| --- | --- | --- | --- |
| P2-1 | Forecast ops: publish per-SKU MAPE/bias; auto-demote losers; fix REGION/CITY sensing shortcut | Keep SBC quadrants; confidence = 1 − normalized MAPE decaying with data age | M |
| P2-2 | MEIO beyond greedy: capacity + transport cost constrained transfers | min cost flow / LP with capital cap already present | L |
| P2-3 | Rename/replace Rust “CP_SAT” heuristic; certify OR-Tools path in prod replicas with cold-chain constraints | Status HEURISTIC vs OPTIMAL honesty | M |
| P2-4 | Road-network ETA (OSRM/matrix) into dispatch scoring from telemetry | ETA = Σ leg durations + service; congestion = median(observed/free-flow) | M |

## 6.5 P3 — scale / enterprise

| **#** | **Item** | **Notes** | **Effort** |
| --- | --- | --- | --- |
| P3-1 | Certified EDI profiles + SAP IDoc/OData adapter pack | Required for chain no-rekey | L |
| P3-2 | Bidirectional master-data + external WMS ASN sync | Beyond PRICAT/INVRPT upsert | L |
| P3-3 | Full admin + support tooling (impersonation-safe, tenant health) | admin-portal expansion | M |
| P3-4 | Enterprise SSO/SCIM, audit export, BI sink | packages/enterprise currently config-heavy | M |
| P3-5 | Tenancy: fail-closed PreferTenant everywhere; retire seed fallbacks in enforced envs | auth/tenant.go patterns | M |
| P3-6 | External POS vendor connectors + hardware (drawer/print) if replacing till islands | Native POS exists; coexistence needed | M |

## 6.6 What not to build yet

- No field-agent CRM until (if ever) a real sales-rep role and doorstep credit negotiation are scoped — current code is not that product.

- No marketplace discovery UX ahead of P3-1/P3-5.

- No LLM assistant over weak demand data — finish soak + MAPE publishing first.

- No new client surfaces — twelve apps + portals already exceed residual gate capacity.

## 6.7 Suggested sequencing

| **Phase** | **Contents** | **Exit criteria** |
| --- | --- | --- |
| Gate 1 — Money/law | P0-1…P0-6 | MY_SOLIQ default in tax envs; AR co-atomic; no 410/501 theatre; push not silent-no-op; payout story honest |
| Gate 2 — Autonomy + WMS truth | P1-1…P1-5 | Place flip for ≥1 cohort with soak artifact; pick/seal ledger on; dunning live with SMS/WhatsApp |
| Gate 3 — Enterprise I/O | P3-1…P3-2 + P1-6 | One anchor chain orders via EDI/SAP map without re-key; admin operable |
| Gate 4 — Brain quality | P2-1…P2-4 | Published forecast accuracy; optimizer prod truth; ETA quality measured |

# Appendix A. Key evidence register

*Representative anchors (repo-relative under pegasusX). Line numbers drift; module path is durable.*

| **Claim area** | **Primary modules** |
| --- | --- |
| Order SM | apps/backend-go/order/state_machine.go; order/service.go |
| Money edges | order/service.go (CollectCash/CompleteOrder); order/driver_edges.go; order/settlement_hardening.go |
| Payment / PSP | payment/execution.go; global_pay_executor.go; *_webhook.go; reconciliation.go |
| Credit / AR / dunning | credit/*; order/credit_guard.go; ar/service.go; ar/dunning_worker.go; ar/dunning_channels.go |
| Fiscal | order/fiscal.go; order/fiscal_provider*.go; fiscal/signer_pkcs12.go; order/consumer.go |
| Outbox / Kafka | outbox/*; events/*; kafka/notification_dispatcher.go; runtime_workers.go |
| Payout | payout/rail.go; payout/store.go |
| WMS | stocklots/*; warehouseroutes/routes.go |
| Planning / auto-order | planning/forecast/*; replenishment/*; retailer/auto_order_*.go; demand/* |
| Optimizer | services/optimizer-core/; warehouse/auto_dispatch.go |
| Partner / EDI / GS1 / 1C | partner/*; contracts/partner.openapi.yaml; gs1/*; sdk/partner/go |
| Clients | apps/retailer-app-*; supplier-*; warehouse-*; factory-*; driver-*; payload-*; admin-portal |
| Flags / defaults | .env.example; .env.ssmr.example; infra/k8s/backend-go/configmap.yaml; overlays/*/kustomization.yaml |

End of report. Evidence date 2026-08-13 · SoT Desktop/V.O.I.D/pegasusX @ 29108a18.
