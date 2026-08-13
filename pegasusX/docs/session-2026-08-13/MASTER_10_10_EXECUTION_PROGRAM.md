# PegasusX Master Plan — Enterprise 10/10 (Phased Deep Execution)

**Status:** PLAN (approve before any implementation wave)  
**SoT tree:** `pegasusX` only  
**Evidence base:** End-Product Reality Report 2026-08-13, `BACKEND_PARITY_*`, Waves B1–B7, live code HEAD  
**Doctrine:** Class A mutation path + holistic cross-role sync + flags as product truth  

---

## 0. North star (what “10/10” means)

### 0.1 End-state product truth

When the program is done, **all domain logic, algorithms, state machines, event contracts, and role clients that belong in the ecosystem are implemented in-repo and fail-closed**. Deployment work is limited to:

| Deploy-time work (allowed) | Code-time work (must be done in phases below) |
|----------------------------|-----------------------------------------------|
| Connect Spanner / Kafka / Redis / GCS | Order/money/WMS/planning algorithms |
| Inject secrets / EDS / Soliq / PSP keys | Same-txn outbox + dispatcher contracts |
| Turn tenant flags for known-good cohorts | Client ↔ API contract alignment |
| Scale pods / Kafka partitions | Auth/tenant/home-node enforcement |
| Wire FCM credentials / SMS providers | Dead UI theatre removal or real wiring |

**Not 10/10:** “flag off forever with UI still advertising the feature.”  
**Is 10/10:** feature complete **or** UI/docs remove it; if gated, gate is dual-control + evidence path.

### 0.2 Scorecard targets (from Reality Report → target 10)

| Layer | Today ~ | Target | What 10 looks like |
|-------|---------|--------|--------------------|
| Go backend transactional core | 8.5 | **10** | No post-commit money fail-open; every Class A mutator = JWT → RW+outbox → relay → consumer; unused TransitionOpts either enforced or deleted |
| Domain model depth | 8.5 | **10** | Full vertical loops closed; dual tables reconciled or documented single SoT with adapters |
| AI / forecast / optimization | 5 | **10** | Published accuracy; honest HEURISTIC vs OPTIMAL; auto-order place for soak-passed cohorts; MEIO beyond pure greedy where needed |
| Integration (API/EDI) | 6 | **10** | Profile-certified EDI or SAP map for ≥1 anchor partner; master-data sync; journals production-grade |
| Multi-tenancy runtime | 6 | **10** | PreferTenant fail-closed in enforced envs; seed fallbacks dead in prod/ssmr |
| Retailer clients | 8 | **10** | No dead settings; honest tracking; POS scan path if POS is product |
| Supplier / factory / WH clients | 7.5 | **10** | Flags match UI; WMS/pick/seal ledger on for seal tenants; factory SLA board if claimed |
| Driver / payload clients | 8 | **10** | No 501/410 theatre; mid-delivery real or gone; line-level load ledger |
| Infra / operability | 5.5 | **10** | FCM not silent no-op; admin operable; outbox/DLQ visibility; optimizer prod truth |
| Fiscal / legal readiness | 4 | **10** | Tax markets default MY_SOLIQ+EDS; PEGASUS labeled commercial-only path |

### 0.3 Explicit non-goals (do not dilute phases)

- Field-agent / van-sales CRM wipe-out product  
- Full MES/MRP/BOM factory manufacturing  
- o9/Kinaxis concurrent APS graph rewrite  
- Marketplace discovery ahead of enterprise I/O  
- Redesigning Spanner → outbox → Kafka bus (architecture is correct)

---

## 1. Operating system — how we execute (every phase)

### 1.1 One phase at a time (hard rule)

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

Never start N+1 with open P0 regressions from N.  
Never “implement logic later at deploy time.”

### 1.2 Per-phase template (mandatory artifacts)

Each phase folder: `docs/session-YYYY-MM-DD/phases/PHASE_<ID>/`

| Artifact | Purpose |
|----------|---------|
| `00_INVENTORY.md` | File:line gaps, current truth, flags |
| `01_DESIGN.md` | Algorithms, SoT tables, event types, API deltas |
| `02_CROSS_ROLE.md` | Who is affected; required alignment |
| `03_IMPL_CHECKLIST.md` | Tick-boxes by package |
| `04_PROOF.md` | `go test` packages, build, greps, manual paths |
| `05_SCORECARD_DELTA.md` | Before/after scores for touched layers |

### 1.3 Class A mutation checklist (every mutator)

From `BACKEND_PARITY_PROTOCOL.md` — non-negotiable:

1. Auth scope (JWT home-node / tenant — never body trust)  
2. Idempotency on public money/stock mutators  
3. Spanner RW txn  
4. Outbox same txn  
5. Cache invalidate post-commit  
6. Dispatcher → hub / FCM / webhook as declared  
7. Edge cases (double-submit, cancel, permission)  
8. Tests or explicit intentional deferral documented  

### 1.4 Cross-role impact matrix (required in every design)

For every phase, fill:

| Change | Spine | Retailer | Driver | WH | Factory | Payload | Supplier | Platform | Partner |
|--------|-------|----------|--------|----|---------|---------|----------|----------|---------|
| Event X | | | | | | | | | |
| API Y | | | | | | | | | |
| Flag Z | | | | | | | | | |
| Client call | | | | | | | | | |

**Alignment rule:** if backend emits/changes a contract, all consumer roles in the matrix get dispatcher + client updates in the **same phase** or phase is incomplete.

### 1.5 Algorithm / technology playbook (flexible)

When implementing a capability:

1. Prefer **existing pegasusX primitives** (outbox, SpannerTxnBuffer, PreferTenant, hubs).  
2. Prefer **proven open algorithms** that fit COD/credit B2B distribution:  
   - Demand: SBC ADI/CV² + Holt–Winters / Croston / SES (already in tree)  
   - Safety stock: classic `z·√(Lσd² + d̄²σL²)` (v2)  
   - VRP: OR-Tools path; keep Rust HEURISTIC honest  
   - Credit risk: logistic/util+DPD+velocity score (industry DMS)  
   - Pick waves: wave → confirm → seal gate (WMS standard)  
3. Prefer **open-source libraries** when license-clean and ops-simple.  
4. If proprietary-only: reimplement **logic**, not branding.  
5. If none fit: design own with documented math + property tests.

### 1.6 Client honesty rule

For every backend fail-closed or disabled feature:

- **Wire client** to real path, **or**  
- **Remove/hide UI**, **or**  
- Show explicit “unavailable / ops-gated” state  

Never leave navigation to 410/501/stub success.

---

## 2. Current baseline (do not re-litigate)

### 2.1 Already Class A (protect)

Order create→outbox, driver money edges+outbox, dispatch freeze, factory/payload seal under Spanner, payment webhooks+idempotency, claims file path, JWT revoke, dual-control money flags, Waves **B1–B7** fail-closed + scope pins.

### 2.2 Residual gap clusters (drive phases)

| Cluster | Examples | Severity |
|---------|----------|----------|
| **Money residual** | Cash AR pay-down post-commit; credit ClearBalance fail-open edges | P0 |
| **Fiscal legal** | Default PEGASUS commercial; MY_SOLIQ not production default | P0 |
| **Theatre** | Negotiation UI vs 410; credit score stub; driver state PATCH 501; mid-delivery not_implemented | P0 |
| **Physical truth** | Pick waves/cycle/cold default off; payload line ledger weak; dual factory/payload tables | P1 |
| **Autonomy** | Auto-order place off; soak gate unused in prod | P1 |
| **Collections** | Dunning transports/flags; credit scoring | P1 |
| **Integration** | SAP/certified EDI; master-data; external WMS ASN | P2/P3 |
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
