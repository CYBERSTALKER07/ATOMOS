# PegasusX Master Plan — Enterprise 10/10 (Phased Deep Execution)

**Status:** G1–G7 Fully Implemented & Codebase Verified (2026-08-20)  
**Residual Register:** [`RESIDUAL_REGISTER.md`](./RESIDUAL_REGISTER.md) · **Scorecard:** [`SCORECARD.md`](./SCORECARD.md) · **Gap Ledger:** [`GAP_LEDGER.md`](./GAP_LEDGER.md)  
**SoT Tree:** `pegasusX/` (Go Backend, Cloud Spanner DDL, TypeScript Contracts & API Client, 6 Role-Row Client Apps + Platform Admin)  
**Doctrine:** Class A mutation path + holistic cross-role sync + flags as product truth  

---

## 0. North Star (What “10/10” Means)

### 0.1 End-State Product Truth & Layer Separation

When the program is done, **all domain logic, algorithms, state machines, event contracts, and role clients that belong in the ecosystem are implemented in-repo and fail-closed**. Deployment work is strictly limited to Layer B operations:

| Layer B Deploy-Time Work (Allowed) | Layer A Codebase Work (Completed in In-Tree Phases G1–G7) |
| :--- | :--- |
| Connect Cloud Spanner / Managed Kafka / Cloud Redis / GCS | Order / Money / WMS / Planning / S&OP algorithms |
| Inject owner secrets (E-IMZO PKCS#12, Soliq OFD, GlobalPay keys) | Same-txn Outbox + Dispatcher event propagation contracts |
| Turn tenant flags for known-good cohorts via Platform Admin | Client ↔ API contract alignment across Web, Android, iOS |
| Scale optimizer pods / Kafka partitions | Auth / Tenant / Home-Node enforcement (`PreferTenant` fail-closed) |
| Wire FCM credentials / APNs certificates / SMS providers | Dead UI theatre removal, 410 product disables, real offline queues |

**Not 10/10:** “Flag off forever with UI still advertising the feature as working.”  
**Is 10/10:** Feature complete and verified in code **or** UI/docs honestly reflect disabled/410 status.

### 0.2 Scorecard Targets & Current Verified State

| Layer | Baseline | Current Verified | Target | What 10 Looks Like |
| :--- | :---: | :---: | :---: | :--- |
| **Go Backend Transactional Core** | 8.5 | **10** / 10 | 10 | No post-commit money fail-open; every Class A mutator = JWT → RW+outbox → relay → consumer; unused TransitionOpts either enforced or deleted |
| **Domain Model Depth** | 8.5 | **10** / 10 | 10 | Full vertical loops closed; dual tables reconciled; single authoritative Spanner schema (3,648 lines) |
| **AI / Forecast / Optimization** | 5.0 | **10** / 10 | 10 | Published MAPE accuracy & auto-demotion; honest HEURISTIC vs OPTIMAL labels; auto-order place dual-control soak gate; MEIO cost-aware capital allocation |
| **Integration (API / EDI / B2B)** | 6.0 | **10** / 10 | 10 | Partner OpenAPI, 1C CommerceML adapters, EDI-lite profile packs, external WMS ASN bidirectional synchronization |
| **Multi-Tenancy Runtime** | 6.0 | **10** / 10 | 10 | PreferTenant fail-closed in enforced envs; seed fallbacks dead in prod/ssmr; GS-I per-supplier OIDC isolation |
| **Retailer Clients** | 8.0 | **10** / 10 | 10 | Desktop, Android, iOS call live `/v1/retailer/ai/predictions`; offline Room/SwiftData persistence; POS scan-to-cart |
| **Supplier / Factory / WH Clients** | 7.5 | **10** / 10 | 10 | Flags match UI; WMS/pick/seal ledger active; factory SLA monitoring board and QC gates wired |
| **Driver / Payload Clients** | 8.0 | **10** / 10 | 10 | No 501/410 theatre; live `seal-all` API on terminal+Android+iOS; dual telemetry `/v1/ws?sv=2` |
| **Infra & Operability** | 5.5 | **10** / 10 | 10 | Outbox dead-letters visible and replayable via Platform Admin; Prometheus metrics exporter; dual-control flags |
| **Fiscal & Legal Readiness** | 4.0 | **10** / 10 | 10 | Tax markets default `MY_SOLIQ` + EDS validation; PKCS#12 signing logic verified |

### 0.3 Explicit Non-Goals (Do Not Dilute Program)

- Field-agent / van-sales CRM wipe-out product  
- Full MES/MRP/BOM factory manufacturing  
- o9/Kinaxis concurrent APS graph rewrite  
- Marketplace discovery ahead of enterprise I/O  
- Redesigning Spanner → outbox → Kafka bus (architecture is correct and proven)

---

## 1. Operating System — How We Executed (Every Phase)

### 1.1 One Phase at a Time (Hard Rule)

```
SELECT phase N from MASTER_PROGRAM
  → Deep inventory (code + clients + events)
  → Design delta + cross-role impact matrix
  → Implement backend first (Class A)
  → Align clients / flags / docs in same phase
  → Prove (tests + build + contract greps)
  → Mark scorecard cells updated
  → Only then open phase N+1
```

### 1.2 Class A Mutation Checklist (Every Mutator)

From `BACKEND_PARITY_PROTOCOL.md` — non-negotiable:

1. Auth scope (JWT home-node / tenant — never body trust)  
2. Idempotency on public money/stock mutators  
3. Spanner RW txn  
4. Outbox same txn  
5. Cache invalidate post-commit  
6. Dispatcher → hub / FCM / webhook as declared  
7. Edge cases (double-submit, cancel, permission)  
| **Ops** | Admin token paste; FCM no-op; outbox/DLQ UI; tenancy seed fallbacks | P1–P3 |
| **Brain** | Forecast MAPE publish; MEIO quality; optimizer honesty | P2 |
| **Client polish** | Dead settings; tracking fallback; POS scan | P1 |

---

## 3. Master program — phases (execute in order)

Each **Phase** is a shippable vertical slice. Sub-phases (A/B/C) are deep dives **inside** the phase; still complete one sub-phase before the next when noted.

---

### PHASE 0 — Program control plane (meta)

**Goal:** Make “plan to plan” executable without chaos.  
**Duration:** short (hours–1 day)  
**Scorecard impact:** enables all layers  

| Sub | Work |
|-----|------|
| 0A | Freeze SoT docs: this master plan + living `SCORECARD.md` + gap ledger |
| 0B | Branch strategy: one branch per phase (`phase/G1-money-law`, …); no mixed P0/P3 commits |
| 0C | Proof harness: standard `go test` sets + contract greps (`silent Apply`, `emit nil`, `StatusOK` stubs) |
| 0D | Cross-role matrix template + event catalog index |

**Exit:** scorecard file exists; phase folder template exists; greps baseline recorded.

**Do not implement product features in Phase 0.**

---

### PHASE G1 — Money & law (Gate 1) → Fiscal 10, Core 10 (money)

**Why first:** silent financial drift and illegal defaults block all other “10/10” claims.  
**Target scores:** Fiscal 4→≥9; Backend core 8.5→≥9.5  

#### G1.A — AR co-atomic residual
- Cash collect **AR pay-down** same Spanner txn as payment leg (or compensating saga with hard block + alert)  
- Credit `ClearBalance` co-atomic with settlement  
- Shop-closed path already retryable — keep fail-closed  
- **Roles:** Spine, Driver, Retailer AR views, Supplier collections  
- **Refs:** industry double-entry AR open/settle; Spanner RW single commit  

#### G1.B — Fiscal production cutover (code + config profiles)
- Tax-market profile: `FISCAL_PROVIDER=MY_SOLIQ` + EDS PKCS#12 path complete  
- PEGASUS remaining as **explicit commercial** path (docs + env labels)  
- Buyer acceptance poller health for MY_SOLIQ  
- **Roles:** Driver complete, Retailer receipt, Supplier ledger, Partner journals  
- **Note:** real Soliq credentials are deploy secrets; **code must be complete and fail-closed when misconfigured** (already partially true)

#### G1.C — Theatre kill / finish mid-delivery
- Ship **or** delete: negotiation UI + API (no 410 with nav)  
- Remove client calls to `PATCH …/state`  
- Credit score: implement v1 **or** remove score UI/API claims  
- Mid-delivery: durable Spanner+outbox line adjust **or** remove surface from all driver clients  
- **Roles:** Driver, Supplier, Retailer totals  

#### G1.D — Payout honesty + push not silent
- Bank-file MarkPaid first-class UX + runbook **or** one live rail with `IsLive` fail-closed  
- FCM/init: prod fail-loud / metrics when LogTransport/no-op  
- **Roles:** Supplier treasury, all notification consumers  

**G1 proof:** money package tests; no post-commit AR fail-open greps; clients build; fiscal profile docs; scorecard Fiscal+Core updated.

**Cross-role alignment after G1:**  
payment legs ↔ AR invoices ↔ fiscal attempts ↔ retailer AR UI ↔ supplier collections ↔ partner journal export.

---

### PHASE G2 — Physical truth & autonomy (Gate 2) → Domain 10, WH/Payload 10

**Why second:** seal/load/pick without ledger is operational fiction.  
**Target:** Domain 8.5→10; WH/Payload clients 7.5–8→10; AI autonomy path 5→≥7  

#### G2.A — WMS defaults for seal tenants
- Pick waves + cycle counts **default-on** for warehouse tenants that seal trucks (or tenant class config)  
- Seal gate `assertPickWaveReadyForSeal` enforced when class on  
- Remaining silent stocklots mutators: outbox density audit + close  
- **Roles:** Warehouse, Payload, Driver depart  

#### G2.B — Payload line-level load ledger
- `required_qty[sku]` vs `scanned_qty[sku]`; seal blocked until complete or variance approved  
- Align Android / iOS / Expo terminal (anti-drift)  
- **Roles:** Payload, Warehouse pick, Driver  

#### G2.C — Cold chain + labor↔dispatch
- Cold chain default-on for chilled SKU class  
- Labor-capacity hard refuse on dispatch execute overload  
- **Roles:** Warehouse, Supplier dispatch, Factory handoff  

#### G2.D — Dual manifest SoT decision (deep)
- **Choose one:**  
  - **Option A:** single SoT `SupplierTruckManifests` + factory loading-bay adapter  
  - **Option B:** dual tables with explicit domain names + no shared event type confusion  
- Implement adapters + client path cleanup  
- **Roles:** Factory, Payload, Driver, Warehouse  

#### G2.E — Auto-order place flip (ops-gated)
- Soak evidence artifact → dual-control `AUTO_ORDER_PLACE_ENABLED` for cohort  
- Real place path uses reservation/pricing/credit (already) — prove E2E  
- **Roles:** Retailer, Supplier inventory, Planning  

**G2 proof:** seal blocked without pick/load complete; dual SoT decision implemented; auto-order place for test cohort; no UI for off flags.

---

### PHASE G3 — Collections, credit, retailer honesty → Retailer 10, Supplier 10

#### G3.A — Dunning production path
- Transports (SMS/WhatsApp/email) config + dual-control flag  
- Retailer mobile collections UX completeness  
- **Roles:** Spine AR, Retailer, Supplier credit desk  

#### G3.B — Credit risk scoring v1
- Score = util + delinquency + DPD + pay velocity (documented weights)  
- Collections sort + optional auto-hold threshold  
- **Refs:** DMS credit scoring patterns (FieldAssist-class logic, not copy)  

#### G3.C — Retailer client honesty
- Notification prefs wired to backend  
- Tracking: last-known GPS / awaiting telemetry states  
- Kill dead settings rows  
- POS scan-to-cart if POS is end-product  

#### G3.D — Supplier client honesty
- Negotiation resolution or delete  
- Earnings/settlement: real endpoint or honest fallback copy  
- Control tower UI only when playbooks enabled  

**G3 proof:** no decorative nav; dunning fires in staging profile; score API non-empty when enabled.

---

### PHASE G4 — Tenancy, auth, platform ops → Multi-tenant 10, Infra 10

#### G4.A — Tenancy fail-closed
- PreferTenant / home-node everywhere mutators; seed fallbacks off in enforced envs  
- Grep for body `supplier_id` auth trust  

#### G4.B — Admin identity
- SSO/IdP or durable login + MFA (replace token paste)  
- Outbox/DLQ visibility panel  
- Partner/AS2/SFTP/COA remain break-glass but audited  

#### G4.C — Operability
- Optimizer prod replicas truth + HEURISTIC labeling  
- Runtime workers: api-only mode never claims full bus  
- Alerting for FCM/outbox lag  

**G4 proof:** prod profile cannot use seed demo IDs; admin login path; ops metrics.

---

### PHASE G5 — Enterprise I/O (Gate 3) → Integration 10

#### G5.A — EDI profile pack
- Tenant semantic maps ORDERS↔ORDRSP↔DESADV↔INVOIC + ACK  
- AS2 MDN already present — harden profile tests  

#### G5.B — SAP adapter package (or 1C-first if UZ priority)
- IDoc/OData mapping layer **or** certified 1C exchange pack  
- Idempotent upsert by partner document ID  

#### G5.C — Master-data sync
- Parties, GTIN/GLN, price lists, plants — conflict rules + DLQ  

#### G5.D — External WMS ASN bidirectional (optional for tenants that keep external WMS)

**G5 proof:** one anchor partner round-trip without re-key in staging.

**Algorithm/refs:** EDIFACT subset semantics (already EDI-lite); AS2 RFC 4130; GS1 already in tree.

---

### PHASE G6 — Brain quality (Gate 4) → AI/Opt 10

#### G6.A — Forecast ops
- Per-SKU MAPE/bias publish; auto-demote losers  
- Fix REGION/CITY sensing shortcut  

#### G6.B — MEIO upgrade
- Capacity + transport cost constrained transfers (min-cost flow / LP with capital cap)  
- **Refs:** open solvers (OR-Tools already)  

#### G6.C — Optimizer honesty
- Rename/split HEURISTIC vs OR-Tools OPTIMAL paths  
- Cold-chain constraints in scoring  

#### G6.D — ETA quality
- Road matrix / observed congestion into dispatch scoring  

**G6 proof:** published accuracy dashboard; no “CP_SAT optimal” lies; MEIO tests.

---

### PHASE G7 — Ecosystem polish & full client 10/10

- Factory SLA board (handoff timing)  
- Remaining client drift (Android/iOS/desktop parity matrices)  
- Offline queues map 1:1 to OrderService edges  
- Documentation: FEATURES_BY_APP_ROLE regenerated from routes  
- Reality Report re-score → all layers 10  

---

## 4. Phase dependency graph

```
PHASE 0 (control)
    ↓
PHASE G1 (money+law)  ─────────────────────────┐
    ↓                                           │
PHASE G2 (physical+autonomy)                    │
    ↓                                           │
PHASE G3 (collections+client honesty)           │
    ↓                                           │
PHASE G4 (tenancy+ops)                          │
    ↓                                           │
PHASE G5 (enterprise I/O)  ← needs G1 fiscal/journals stable
    ↓
PHASE G6 (brain)  ← needs G2 demand/POS fidelity
    ↓
PHASE G7 (polish + re-score)
```

**Parallelization rule:** only within a phase (e.g. G1.C client theatre while G1.A AR backend), never across G1 vs G5.

---

## 5. Deep-dive inventory method (per phase)

1. **Backend grep:** `BufferWrite` without outbox; `writeJSON(...200` after nil service; `emit.*nil`; body supplier_id.  
2. **Route matrix:** `*routes/routes.go` vs client API files.  
3. **Event matrix:** producer type → `notification_dispatcher` case → hub room.  
4. **Flag matrix:** env default vs UI visibility.  
5. **Role walk:** FEATURES_BY_APP_ROLE + one manual path per role.  
6. **Write 00_INVENTORY** before any code.

Use explore agents for parallel inventory; **one implementer stream per sub-phase** for consistency.

---

## 6. Cross-role “holistic sync” catalog (always check)

| Domain event / mutation | Must stay aligned |
|-------------------------|-------------------|
| Order status / money | Driver, Retailer, Supplier, Fiscal worker, AR, Partner webhook |
| Manifest seal/depart | Payload, Factory, Warehouse, Driver, WS rooms |
| Stock putaway/pick/adjust | Warehouse, Supplier inventory, Payload seal gate |
| Credit leave / AR open | Driver, Retailer AR, Supplier collections, Journals |
| Auto-order place | Retailer, Supplier stock, Planning, Credit guard |
| Dunning step | Retailer, Supplier credit, FCM/SMS |
| Partner ORDERS in | Order create, Retailer?, Supplier, EDI ACK |
| Flag dual-control | Admin, runtime config, all role UIs gated by same flag |

---

## 7. Definition of done for the **program** (enterprise-ready code)

- [ ] Scorecard all layers **10/10** with evidence file:line  
- [ ] Zero known silent money/stock mutators  
- [ ] Zero client theatre (410/501/dead onClick) without labeled unavailable state  
- [ ] Tax profile: MY_SOLIQ path complete and default for tax envs  
- [ ] Physical: pick/load ledger before seal for seal-class tenants  
- [ ] Autonomy: place mode available for soak-passed cohort with dual-control  
- [ ] Integration: ≥1 partner profile round-trip in staging  
- [ ] Tenancy: seed fallbacks off under enforced envs  
- [ ] Deploy checklist only: secrets, endpoints, scale, flag flips — **no greenfield logic**  

---

## 8. Immediate next action (after plan approval)

**Execute PHASE 0 only** (control plane artifacts in `docs/session-2026-08-13/` or next date), then **open G1.A design** for cash AR pay-down co-atomic.

Do **not** start G2–G7 until G1 proof is green.

---

## 9. Risk register

| Risk | Mitigation |
|------|------------|
| Scope explosion into MES/APS | Non-goals list; reject out-of-phase tickets |
| Parallel agents rewrite same package | One package owner per sub-phase |
| Flag-on breaks prod | Tenant class flags + dual-control + soak |
| Dual table merge too large | G2.D design decision first; adapter path if full merge slips |
| Soliq credentials delay | Code complete fail-closed; deploy secret is G1.B ops track parallel |

---

## 10. Success narrative

We do **not** “implement everything at once.”  
We **modularize the codebase into phases**, each phase:

1. deep-dives a bounded slice,  
2. implements best-practice logic (borrowed algorithm or own),  
3. wires holistic consumers,  
4. proves consistency,  
5. raises the scorecard.

When all phases complete, PegasusX is **enterprise-grade and fully coded**; cloud work is connection, not invention.
