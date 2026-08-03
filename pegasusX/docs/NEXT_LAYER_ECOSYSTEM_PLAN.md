# Next-Layer Ecosystem Plan (Post Retail OS Phases 0–5)

**Status:** Implementation plan (code-grounded)  
**Repo:** `/Users/shakhzod/Desktop/V.O.I.D/pegasusX`  
**Date:** 2026-08-02  
**Companion to:** Flexible Retailer Operating System plan (Retail OS Phases 0–5)  
**Constraint:** **Ship Retail OS first.** Everything in this document is the next layer after that — not instead of it.  
**Clients:** All role-rows (retailer / supplier / warehouse / driver / factory / payload) where noted; retailer-facing items require role-row parity on Android · iOS · Desktop  

---

## 0. Executive summary

### Product principle (non-negotiable)

> **Retail OS first.** Do not start a competing “second big feature” until Retail OS Phases 0–1 are scheduled and Phase 0 foundations are green.  
> This plan sequences **ops unlocks**, **Retail OS CORE hardening**, the **sell-through → reorder flywheel**, then **strong follow-ons** and **later scale**.  
> Cross-role law: warehouse/factory/driver apps stay ignorant of POS internals; they consume Kafka/events only where needed.

### Relationship to Retail OS

| Retail OS phase | This plan may consume |
| ----------------- | ------------------------ |
| P0 packs + JWT v2 + durable settings | GP card UX, notif fanout, Control Tower pulse |
| P1 TEAM | Per-user FCM, multi-org staff (later), CUSTOMER_ASSIST |
| P2 LOCATIONS | Franchise / HQ rollups |
| P3 STORE_STOCK | Sell-through stock, quarantine bridge, local SKUs |
| P4 POS | Sell-through sales, local SKUs, POS fiscal, parked carts, offline count |
| P5 SHIFTS | Soft dependency for large-store assist / labor reports only |

**Code note (2026-08-02):** Parts of Retail OS (packs, TEAM, locations, store stock, POS) may already exist in-tree (`retailer/capability_packs.go`, `store_stock.go`, `pos.go`, `docs/RETAILER_*.md`). Treat this plan’s prerequisites as **product-complete + parity-tested**, not “folder exists.”

### What this plan covers

| Tier | IDs | Intent |
| ------ | ----- | -------- |
| **Do next** | L1–L3 | Highest ecosystem leverage; unblock field reality + flywheel |
| **Strong follow-ons** | L4–L7 | After Retail OS P0–P1 (and P3/P4 where noted) |
| **Later scale** | L8–L11 | Mode L / network effects |
| **Skip** | — | Explicit non-goals |

---

## 1. Goals and non-goals

### Goals

1. **Real money + real auth in the field** — Global Pay SUCCESS and Firebase phone OTP (not cash-fallback / emulator-only).  
2. **CORE hardening** that Retail OS and every retailer client depend on — durable settings leftovers, honest Control Tower, complete notif fanout.  
3. **Pegasus flywheel** — store on-hand + POS sell-through → retailer reorder suggestions → supplier demand signals.  
4. **Close logistics exception loops** — quantity negotiations (re-enable), claims → store quarantine → reverse logistics.  
5. **Legal fiscal readiness** — Soliq OFD for delivery + optional POS walk-in when pilots require it.  
6. **POS usefulness between wholesales** — local/manual SKUs.  
7. **Scale surfaces** — multi-org phones, HQ analytics, offline count, parked carts, floor assist — only after online POS is stable.

### Non-goals (deliberately skip)

- Full ERP / HR payroll / union rules / biometric attendance as legal SoT  
- Franchise multi-legal-entity tax consolidation  
- Forcing POS on pure B2B replenishment-only retailers  
- A second big feature competing with Retail OS until P0–P1 scheduled  
- Replacing warehouse ATP with store ledger (store stock remains retailer-owned)  
- Planogram computer vision (v2+ under L11)

---

## 2. Current baseline (code truth)

### Ops / money / auth

| Area | Status | Anchors |
| ------ | -------- | --------- |
| Global Pay executor + webhook | Wired; credentials placeholder | `payment/global_pay_executor.go`, `payment/global_pay_webhook.go`, `POST /v1/webhooks/global-pay` |
| GP env defaults | `doc-*` stubs; empty password → stub refs | `bootstrap/bootstrap.go` `GLOBAL_PAY_*` |
| Card checkout at delivery | Wired; e2e cash fallback today | `payment/retailer_checkout.go`; `PX_E2E_PAYMENT_CASH_FALLBACK_OK` |
| Pre-delivery B2B checkout | **410** removed | `POST /v1/checkout/b2b` |
| Card vault | Stub OK responses | `retailer/mobile_compat.go` cards endpoints |
| Firebase client configs | In-tree (6 Android + 6 iOS) | `google-services.json`, `GoogleService-Info.plist` |
| Firebase Auth backend | Env-gated; SMS/SHA owner | `FIREBASE_AUTH_ENABLED` (default false) |
| DNS/TLS SSMR | Active | `https://api-ssmr.pegasusx.app` |

### Retailer CORE leftovers

| Area | Status | Anchors |
| ------ | -------- | --------- |
| Auto-order settings | Durable Spanner (+ process cache) | `repository_settings_durable.go`, `auto_order.go` |
| Favorites | Durable | `RetailerFavoriteSuppliers` |
| Family members | **Team SoT + durable gone flag** | List RAM legacy; migrate → Team; `family_writes_gone` in `RetailerOrgFlags` |
| Auto-order **execution** | Settings + event only — **no place-order worker** | `RETAILER_AUTO_ORDER_UPDATED` |
| Device tokens | Platform path durable; actor = JWT subject | `POST /v1/user/device-token`, `DeviceTokens` |
| Retailer mobile device-token | Compat no-op | `mobile_compat.go` — clients must use platform route |
| Fanout | `broadcastRetailer` → active user IDs + legacy org | `kafka/notification_dispatcher.go`, `ListActiveUserIDs` |
| Control Tower backend | Real playbooks (flag off by default) | `controltower/*`, `CONTROL_TOWER_PLAYBOOKS_ENABLED` |
| Control Tower retailer UI | **Wired live pulse** | `GET /v1/retailer/control-tower/pulse` on desktop/Android/iOS; simulator env-gated |
| CT simulator | Fake H3 pulse | `simulator/control_tower.go` |

### Flywheel / replenishment

| Area | Status | Anchors |
| ------ | -------- | --------- |
| `syncReorderCurrentStock` | Updates `ReorderSuggestions.CurrentStock` if row exists | `retailer/store_stock.go`, called from POS/stock |
| `ReorderSuggestionWorker` | Batch from `DemandAdjustments`; prefers store OnHand | `replenishment/reorder_suggestion_batch.go` |
| Demand sensing | Order-history / factors — **not POS lines** | `demand/worker_sensing.go` |
| POS → DemandAdjustments | **Partial** | `SELL_THROUGH` factor + `RetailerSellThroughDaily` on POS sale/void (`retailer/sell_through.go`) |
| Reorder worker sell-through merge | **Shipped (L3.3)** | `replenishment/velocity_merge.go` + batch max(base, ST 7d vel) + `sources[]` on suggestions |
| UI source chips + auto-order from suggestions | **Shipped (L3.4–L3.5)** | Supplier portal chips; retailer reorder list; AutoOrderWorker prefers OPEN ReorderSuggestions |
| Auto-order settings → worker | **Draft shipped** | `auto_order_worker.go` draft/runs; candidates from suggestions then AI |

### Deferred / gated product

| Area | Status | Anchors |
| ------ | -------- | --------- |
| Quantity negotiations | **Disabled** (`410` / empty list) | `quantityNegotiationDisabled = true`; code retained, product-deferred |
| Negotiation schema | Present | `NegotiationProposals` in Spanner DDL |
| Fiscal | `FISCAL_PROVIDER=PEGASUS` live | `order/fiscal_*.go` |
| Soliq | Adapter path exists; deferred secrets | `FISCAL_PROVIDER=MY_SOLIQ`, `FISCAL_MY_SOLIQ_*` |
| POS fiscal | Local receipt number only | `retailer/pos.go` — no `FiscalProvider` |
| Local SKU catalog | **Missing** (free-form sku string on POS) | sale DTO accepts any sku |
| Claims → reverse WH | Wired | `claims/service.go` → `returns`, warehouse reverse APIs |
| Claims → store QUARANTINE | **Missing bridge** | bin exists; claims never touch `RetailerStockBalances` |
| CUSTOMER_ASSIST / parked carts / offline count / HQ analytics | Catalog or absent | packs docs / capability catalog |

---

## 3. Dependency graph (hard + soft)

```
Retail OS P0–P1  ──hard──► L2 CORE hardening (TEAM fanout assumptions)
Retail OS P3     ──hard──► L3 sell-through stock, L7 quarantine bridge, L6 local SKUs
Retail OS P4     ──hard──► L3 sell-through sales, L6 local SKUs, L5 POS OFD, L10 parked/offline
L1 GP SUCCESS    ──soft──► card vault polish, POS card tender reuse
L1 Firebase OTP  ──soft──► production phone login (all roles)
L3 flywheel      ──soft──► auto-order execution worker (L2.4)
L4 negotiations  ──independent──► logistics (driver↔supplier); no Retail OS hard dep
L5 Soliq         ──soft──► L1 (money path proven) + pilot legal need
L7 returns bridge──hard──► claims spine (exists) + Retail OS P3
L8 multi-org     ──hard──► Retail OS P1 schema change
L9 HQ analytics  ──hard──► P2 LOCATIONS + REPORTS_PRO data
L10 offline/park ──hard──► P4 POS stable online
L11 assist/vision──hard──► SECTIONS + TEAM (+ SHIFTS soft)
```

**Algorithm — start a next-layer epic**

```
start(epic):
  missing = hard_deps(epic) - done
  if missing:
    return BLOCKED { missing, message: "finish Retail OS / prior layer first" }
  soft = soft_deps(epic) - done
  if soft:
    return WARN { soft, continue_anyway | finish_soft_first }
  schedule PRs
```

---

## 4. Tier catalog

| ID | Name | Unlocks | Default priority |
| ---- | ------ | --------- | ------------------ |
| **L1** | Field unlock (GP + Firebase) | Real card SUCCESS; real SMS OTP | **P0 ops** (eng ready 2026-08-04; owner GP password still blocks SUCCESS) |
| **L2** | Retail OS CORE hardening | Family durability/migrate; CT de-demo; notif honesty; auto-order execution | **Shipped** (worker flag + CT sim gate) |
| **L3** | Sell-through → reorder bridge | POS/stock → DemandAdjustments → suggestions → supplier signals | **P0 flywheel** |
| **L4** | Quantity negotiations | Driver propose ↔ supplier resolve (re-enable) | **Env-gated** `QUANTITY_NEGOTIATION_ENABLED` (default off) |
| **L5** | Soliq OFD | Legal fiscal delivery + POS path | P1 when legally required |
| **L6** | Local / manual POS SKUs | Non-Pegasus goods between wholesales | P1 after P3/P4 |
| **L7** | Store reverse loop | Claim → QUARANTINE → RETURN/WASTE + WH awareness | P1 after P3 |
| **L8** | Multi-org staff phones | One person, many retailer orgs | P2 scale |
| **L9** | Franchise / HQ analytics | Multi-location rollups + Kafka BI | P2 scale |
| **L10** | Offline count + parked carts | Count queue; hold tickets | P2 after online POS |
| **L11** | CUSTOMER_ASSIST / planogram | Floor help; vision later | P3 lowest urgency |

---

## 5. Feature deep-dives

Each feature: purpose · actors · algorithm · APIs · UI · edges · cross-role · platforms · success.

---

### L1 — Global Pay SUCCESS + Firebase real OTP

#### Purpose

Field pilots cannot depend on cash-only e2e or auth emulators. Card capture + phone OTP are the last **ops** gates before production profile.

#### Actors

Owner/ops (secrets), retailer (checkout), driver (delivery pay), all apps (login).

#### Algorithms

**Global Pay SUCCESS**

```
ops:
  put real service_id, username, password, webhook_secret → GSM
  ESO sync → restart API/worker
  register webhook https://api-ssmr.pegasusx.app/v1/webhooks/global-pay

runtime (existing):
  card-checkout → GP initiate
  webhook SUCCESS/SETTLED → re-verify GP API → capture path
  reject stub refs in production profile validation
```

**Firebase OTP**

```
ops:
  Blaze + Phone Auth enabled
  register Android SHA-1/SHA-256 (debug + release) per applicationId
  APNs key for iOS push if FCM required
  FIREBASE_AUTH_ENABLED=true on API with SA credentials

client:
  no firebase.auth.emulator / FIREBASE_AUTH_EMULATOR_HOST
  API base https://api-ssmr.pegasusx.app
  verifyIdToken → existing retailer/supplier/... login handlers
```

#### APIs (existing — verify, don’t reinvent)

```
POST /v1/order/card-checkout
POST /v1/webhooks/global-pay
POST /v1/auth/retailer/login   # (+ peer role logins) Firebase ID token path
POST /v1/user/device-token
```

#### UI

- No new IA; fix empty/error states when GP auth fails (no silent cash-only in prod builds).  
- Settings: “Payment ready” / “SMS login ready” owner diagnostics optional (desktop).

#### Edge cases

| Case | Expected |
| ------ | ---------- |
| Wrong GP password | Clear 401/merchant error; no stub SUCCESS in SSMR/prod |
| Webhook replay | Idempotent settle (existing) |
| SHA mismatch | Phone OTP fails on device; document per-app applicationId |
| Emulator flag left on | Refuse release checklist |

#### Cross-role

- Delivery card path is order/driver/retailer spine — unchanged.  
- POS card tender may later reuse GP patterns (soft); not required for L1.

#### Phased PRs

| PR | Scope |
| ---- | ------- |
| **L1.1** | Owner: GSM secrets + webhook portal + API restart |
| **L1.2** | Smokecheck: assert GP SUCCESS marker (not only cash fallback) |
| **L1.3** | Firebase Phone + SHA/APNs; enable `FIREBASE_AUTH_ENABLED` on SSMR |
| **L1.4** | Client release checklist: no emulator; public API host |
| **L1.5** | Optional: wire card vault beyond stub once GP SUCCESS green |

#### Success metrics

- `PX_E2E_PAYMENT_CARD_SUCCESS_OK` (or equivalent) on SSMR  
- OTP login on one Android + one iOS app without emulator  
- `ValidateProductionProfile` does not trip on GP stubs  

---

### L2 — Retail OS CORE hardening

#### Purpose

Foundations Retail OS and field ops depend on: no silent data loss, no demo Control Tower, notifications that reach the right person.

#### Sub-features

##### L2.1 Family → durable or TEAM migrate

```
migrate_family:
  for each in-memory FamilyMember (export window) OR one-shot API dump:
    either insert contact-only durable row OR invite as RetailerUser (TEAM)
  deprecate POST family-members that only touch RAM
  document: Family is not RBAC
```

**Tables (choose one product path):**

- A) `RetailerFamilyContacts` (durable contacts, no login) — minimal  
- B) Convert to `RetailerUsers` invites — preferred if TEAM shipped  

##### L2.2 Notification honesty

```
register_token:
  clients MUST call POST /v1/user/device-token (platform)
  remove or redirect retailer mobile_compat no-op

fanout:
  broadcastRetailer → ListActiveUserIDs ∩ permission/topic
  keep legacy org-id tokens during dual-read window
  inbox: prefer per-user rows when TEAM on; org fallback for CORE-only
```

##### L2.3 Control Tower de-demo

```
retailer CT clients:
  replace mock lists/charts with live APIs or empty-state
  never show hardcoded BarMark / “Mock Data”

backend:
  optional: expose retailer-safe pulse (orders in flight, dock waiting)
    from real Spanner queries — not simulator/control_tower.go
  supplier CT: ScopedSupplierID only; enable playbooks via flag when ready
  kill StartControlTowerSimulation in non-dev env
```

##### L2.4 Auto-order execution worker (beyond settings)

Settings are durable; **execution is missing**.

```
AutoOrderWorker tick:
  for retailers with auto-order enabled:
    read suggestions / predictions + overrides (supplier/category/product/variant)
    if policy allows (credit, ATP, schedule window):
      create draft cart or place order (product choice: draft vs auto-submit)
    emit RETAILER_AUTO_ORDER_PLACED | SKIPPED { reason }
    idempotency: retailer_id + sku + schedule_bucket
```

**Soft dep:** L3 improves suggestion quality; worker can ship on order-history suggestions first.

#### APIs

```
# Family migrate
GET/POST /v1/retailer/family-members     # durable or 410 gone after migrate
POST     /v1/retailer/org/members        # TEAM invite path

# Already exist — harden
GET/PATCH /v1/retailer/settings/auto-order*
POST      /v1/user/device-token

# Control Tower retailer pulse (new or extend)
GET /v1/retailer/control-tower/pulse
```

#### UI (parity)

- Settings: Family banner “Move to Team” or durable contacts list  
- Control Tower: empty or live — never mock numbers  
- Auto-order: show last run / last skip reason (desktop + mobile)

#### Edge cases

- Pod restart must not wipe family (today it does)  
- Staff without `order.place` must not get auto-order confirm pushes that deep-link to checkout  
- CT playbooks auto-execute default **off**

#### Phased PRs

| PR | Scope |
| ---- | ------- |
| **L2.1** | Family durable **or** migrate → TEAM; kill RAM SoT |
| **L2.2** | Client device-token → platform; remove false OK |
| **L2.3** | Retailer CT de-demo + stop simulator outside dev |
| **L2.4** | AutoOrderWorker draft/place + audit events |
| **L2.5** | Parity + SSMR markers for auto-order skip/place |

---

### L3 — Sell-through → reorder bridge (Pegasus flywheel)

#### Purpose

Unique ecosystem loop: what sold on the floor and what sits in the backroom should drive **retailer reorder** and **supplier demand**, not only historical wholesale orders.

#### Actors

Retailer (stock/POS), replenishment workers, supplier (suggestions / planning).

#### Algorithms

**POS / stock → demand signal**

```
on POS_SALE_COMPLETED | STORE_STOCK_RECEIVED | COUNT_VARIANCE | SALE_VOID:
  upsert SellThroughRollup { retailer_id, location_id, sku, window, qty_sold, qty_on_hand }
  write or adjust DemandAdjustments factor SELL_THROUGH (signed; voids reverse)
  syncReorderCurrentStock (already exists — keep)
  outbox RETAILER_SELL_THROUGH_UPDATED
```

**Suggestion batch (extend worker)**

```
ReorderSuggestionWorker:
  CurrentStock = store OnHand (exists)
  velocity = max(order_history_velocity, sell_through_velocity)
  respect auto-order overrides (enabled flags)
  emit suggestions; supplier Kafka DEMAND_SIGNAL (B4 shipped 2026-08-02)
```

**Supplier consumption**

```
supplier portal / apps:
  show sell-through influenced suggestions (label source: STORE_POS | WHOLESALE_HISTORY)
  do NOT require warehouse/factory UI changes
```

#### APIs

```
GET /v1/retailer/insights/sell-through?location_id=&from=&to=
GET /v1/supplier/reorder-suggestions   # extend payload with sources[]
# Internal: Kafka RETAILER_SELL_THROUGH_UPDATED / DEMAND_SIGNAL
```

#### Persistence

```
RetailerSellThroughDaily
  RetailerId, LocationId, SkuId, Day, QtySold, QtyVoided, QtyOnHandEod
  PK (RetailerId, LocationId, SkuId, Day)
```

#### UI

- Retailer Insights: “Sold vs on hand” cards (REPORTS_PRO soft; CORE digest OK)  
- Supplier reorder list: source chip  

#### Edge cases

| Case | Expected |
| ------ | ---------- |
| POS without STORE_STOCK | Should be impossible (Retail OS hard dep); if legacy, skip sell-through |
| Local SKU (L6) not in supplier catalog | Retailer reorder only; **no** supplier suggestion row |
| Void after suggestion emitted | Reverse adjustment; recompute |
| Multi-location | Rollup per location; org suggestion may sum |

#### Cross-role

- Warehouse ATP unchanged  
- Supplier demand improves; factory MEIO later may read same Kafka topic (out of scope)

#### Phased PRs

| PR | Scope | Depends |
| ---- | ------- | --------- |
| **L3.1** | SellThroughDaily + writers from POS/stock events | Retail OS P3/P4 |
| **L3.2** | DemandAdjustments SELL_THROUGH factor | L3.1 |
| **L3.3** | ReorderSuggestionWorker velocity merge + source labels | L3.2 |
| **L3.4** | Supplier + retailer UI chips / insights | L3.3 |
| **L3.5** | Wire AutoOrderWorker (L2.4) to sell-through-aware suggestions | L2.4 + L3.3 |

#### Success metrics

- After N POS sales, `CurrentStock` and suggestion qty move within one worker tick  
- Supplier list shows `STORE_POS` source on pilot SKU  
- No warehouse schema changes  

---

### L4 — Supplier quantity negotiations (re-enable)

#### Purpose

**Delivery-time qty exception with human approve** — driver proposes, supplier resolves.  
Not B2B price/RFQ negotiation. Peers often use POD exceptions; Pegasus already built the propose/resolve loop — it is **feature-gated off**.

#### Actors

Driver, supplier (resolver), retailer (notif / final qty), order state machine.

#### Algorithm (existing — re-enable)

```
driver POST /v1/delivery/negotiate { order_id, lines[] }:
  assert quantityNegotiationDisabled == false
  insert NegotiationProposals PENDING
  outbox NEGOTIATION_PROPOSED
  block finalize until resolve or timeout sweeper

supplier POST /v1/supplier/negotiate/resolve { proposal_id, APPROVE|REJECT|COUNTER }:
  apply qty to order lines / payment basis
  outbox NEGOTIATION_RESOLVED
```

#### Re-enable checklist

1. `order/negotiation_disabled.go` → `quantityNegotiationDisabled = false`  
2. Restore driver client `POST /v1/delivery/negotiate` (Android commented; iOS gated)  
3. Supplier portal + Android + iOS: pending queue (today empty-state “disabled”)  
4. Sweeper + Kafka fanout verification  
5. Re-enable `runNegotiationE2E` in `ssmr-smokecheck`  
6. Update `parity-ledger.md` + marker gate (remove skip-by-design)

#### UI

- Driver: propose sheet at stop (partial / short / refuse-as-negotiate policy — product pick)  
- Supplier: pending negotiations inbox (parity with claims queue quality)  
- Retailer: notification of approved qty change  

#### Edge cases

- Double propose → idempotency key  
- Approve after retailer already paid → money adjust policy (credit note / recapture)  
- Interaction with shop-closed / partial offload — single exception winner rules  
- STORE_STOCK receive must use **resolved** qty (bridge Soft → L7)

#### Cross-role

- Warehouse pick already done — negotiation is post-pick delivery variance  
- Do not reopen WMS tasks unless REJECT + return policy says so  

#### Phased PRs

| PR | Scope |
| ---- | ------- |
| **L4.1** | Flip gate + backend tests green |
| **L4.2** | Driver propose UX (Android + iOS) |
| **L4.3** | Supplier resolve UX (portal + Android + iOS) |
| **L4.4** | Retailer notif + e2e marker + parity ledger |

#### Success metrics

- E2E negotiation path PASS on SSMR  
- Pending list non-empty when proposed  
- Zero 410 in production config  

---

### L5 — Soliq OFD (legal fiscal)

#### Purpose

Pegasus branded HTML/PDF receipts are commercial. Legal OFD/EHF for Uzbekistan when pilots or POS cash/card require it.

#### Actors

Fiscal worker, order lifecycle, optional POS, ops (TIN/EDS secrets).

#### Algorithm (ADR-009 — extend)

```
Payment capture → FISCALIZING
  → CreateReceipt(provider=MY_SOLIQ)
  → SUCCESS → COMPLETED
  → FAILED → FISCAL_FAILED → retry / admin force

POS path (new):
  on POS sale tender finalize:
    if fiscal_pack_or_config.enabled:
      enqueue FISCAL_RECEIPT_REQUESTED { source: POS_SALE, sale_id }
      sale status FISCALIZING|COMPLETED per policy
    else:
      keep local ReceiptNumber only
```

#### Env / secrets

```
FISCAL_PROVIDER=MY_SOLIQ
FISCAL_MY_SOLIQ_BASE_URL=
FISCAL_MY_SOLIQ_API_KEY=
FISCAL_MY_SOLIQ_TIN=
# EDS / client certs in GSM — see docs/big-platform-baseline/regulatory/soliq-ehf-integration.md
```

#### Soft gates

- Prefer L1 money path proven before switching provider on SSMR  
- POS OFD requires Retail OS P4 + legal decision (open question)

#### Phased PRs

| PR | Scope |
| ---- | ------- |
| **L5.1** | Sandbox MY_SOLIQ delivery path on staging/SSMR flag |
| **L5.2** | Clearance tracking + buyer reject handling |
| **L5.3** | POS → FiscalProvider bridge (optional pack/config) |
| **L5.4** | Compliance audit export |

#### Success metrics

- Delivery order COMPLETED only after Soliq SUCCESS when provider enabled  
- POS pilot: fiscal id on receipt when enabled  
- PEGASUS provider remains fallback for non-legal environments  

---

### L6 — Local / manual POS SKUs

#### Purpose

POS must work between Pegasus wholesales. Retailers sell bread, bags, locally sourced goods that never appear on supplier ATP.

#### Actors

Owner/admin/manager (catalog), cashier (sell), stock clerk (receive/count local).

#### Model

```
RetailerLocalCatalog
  RetailerId, LocalSkuId, Barcode, Name, Unit, DefaultPriceMinor, TaxCode?, IsActive
  optional: SectionId, LinkedPegasusSkuId (null if pure local)

RetailerStockBalances already keys by SkuId string — use LocalSkuId namespace prefix `local:` or separate SkuSource enum
```

#### Algorithms

```
create_local_sku(...):
  assert STORE_STOCK or POS pack
  assert actor catalog.manage (OWNER/ADMIN/MANAGER)
  insert; optional opening OnHand via ADJUST

pos_sale line:
  if sku in Pegasus catalog OR local catalog: allow
  else reject unknown (stricter than today free-form)

reorder bridge (L3):
  local skus → retailer insights only; never supplier ReorderSuggestions
```

#### APIs

```
GET/POST   /v1/retailer/local-skus
PATCH      /v1/retailer/local-skus/{id}
POST       /v1/retailer/local-skus/{id}/barcode
```

#### UI

- Desktop: bulk CSV import  
- Mobile: quick-add + barcode  
- POS search: union Pegasus received SKUs + local catalog  

#### Edge cases

- Barcode collision with Pegasus SKU → prefer explicit location policy  
- Price override still manager PIN  
- Disable local sku with OnHand > 0 → block or force count  

#### Phased PRs

| PR | Scope | Depends |
| ---- | ------- | --------- |
| **L6.1** | Schema + CRUD API | P3 |
| **L6.2** | POS validation + search union | P4 |
| **L6.3** | CSV import desktop + scan mobile | L6.1 |
| **L6.4** | Exclude from supplier demand (L3 guard) | L3 |

---

### L7 — Returns / reverse logistics → store stock

Detailed merge plan (rich receiver ↔ store stock ↔ claims, visible vs concealed damage, configurable return windows): [`docs/RETAILER_RECEIVE_STOCK_CLAIMS_PLAN.md`](./RETAILER_RECEIVE_STOCK_CLAIMS_PLAN.md).

#### Purpose

Close the loop when goods already entered the **store ledger**: claim → QUARANTINE → RETURN_TO_SUPPLIER / WASTE, with warehouse reverse ticket remaining SoT for inbound dock.

#### Actors

Retailer receiver/manager, supplier claims, warehouse reverse dock.

#### Algorithm

```
on Claim FILED (and store had received order):
  if STORE_STOCK enabled and receive session exists for order:
    for claimed lines:
      transfer/adjust FLOOR|BACKROOM → QUARANTINE qty
      movement CLAIM_HOLD
      outbox STORE_STOCK_CLAIM_HOLD

on Claim APPROVED:
  open/keep reverse logistics ticket (exists)
  movement RETURN_TO_SUPPLIER (−QUARANTINE) OR await WH receive ack
  syncReorderCurrentStock

on Claim REJECTED:
  QUARANTINE → FLOOR|BACKROOM restore OR WASTE per resolution code
```

**Never** double-subtract warehouse ATP and store ledger incorrectly: warehouse reverse remains supplier/WH document; store movements are retailer-owned mirror when putaway already happened.

#### APIs

```
# extend existing claims — no parallel system
POST /v1/claims/{id}/approve   # side effect: store movements when applicable
GET  /v1/retailer/stock/movements?ref_type=CLAIM&ref_id=
```

#### UI

- Retailer stock: Quarantine bin filter + “from claim” deep link  
- Warehouse reverse UI unchanged; show claim id (already)  
- Supplier claims: badge “store hold applied” |

#### Edge cases

| Case | Expected |
| ------ | ---------- |
| Claim before store receive | No store movement; WH reverse only |
| Partial claim | Hold only claimed qty |
| Negotiation (L4) then claim | Use post-negotiation qty as baseline |
| Idempotent approve | One movement set per claim line |

#### Phased PRs

| PR | Scope | Depends |
| ---- | ------- | --------- |
| **L7.1** | Claim filed → QUARANTINE movements | P3 |
| **L7.2** | Approve/reject → RETURN/WASTE/restore | L7.1 |
| **L7.3** | UI quarantine + parity | L7.2 |
| **L7.4** | E2E: receive → claim → quarantine → WH reverse | L7.3 |

---

### L8 — Multi-org staff phones

#### Purpose

Cashiers/managers who work multiple shops (chains, relief staff) need one phone ↔ many `RetailerOrgId`s.

#### Current law (v1)

`RetailerUsers` uniqueness `(RetailerId, Phone)` — one active org per phone at login resolution.

#### Target

```
RetailerUserMemberships
  UserId, RetailerId, RetailerRole, IsActive, LocationIds[]
Login:
  if multiple memberships: return org picker intermediate token
  POST /v1/auth/retailer/select-org { retailer_id } → full JWT
```

#### Soft deps

- Retail OS P1 TEAM  
- Product answer to open question #1 in Retail OS plan  

#### Phased PRs

| PR | Scope |
| ---- | ------- |
| **L8.1** | Membership schema + migration from single-row users |
| **L8.2** | Login picker + select-org |
| **L8.3** | Client org switcher (all three retailer clients) |
| **L8.4** | Notif fanout across memberships (opt-in quiet hours) |

---

### L9 — Franchise / HQ analytics

#### Purpose

Mode L chains need multi-location rollups without teaching warehouse/factory about POS.

#### Approach

```
Kafka consumers (BI):
  POS_SALE_*, STORE_STOCK_*, SHIFT_* → Warehouse-neutral analytics store
    or Spanner rollup tables RetailerHqDaily*

APIs:
  GET /v1/retailer/hq/summary          # OWNER/ADMIN multi-location
  GET /v1/retailer/hq/sales-by-location
  GET /v1/retailer/hq/shrinkage

UI:
  Desktop primary dashboards; mobile owner digest cards
```

**Do not** add POS screens to warehouse/factory apps.

#### Depends

P2 LOCATIONS, P4 POS data, soft REPORTS_PRO pack.

---

### L10 — Offline-tolerant count + parked POS carts

#### Purpose

After online-required POS is stable: clerks count without signal; cashiers park tickets.

#### Algorithms

```
offline_count:
  mobile queues CountDraft locally (Room / SwiftData)
  on reconnect: POST counts/commit with base_version
  if balance changed: conflict UI (recount | force with manager)

parked_cart:
  PosHold { hold_id, register_id, lines, expires_at }
  resume → same register or manager transfer
```

#### Depends

Retail OS P4 stable; driver offline patterns are **reference only** (do not reuse driver queue blindly for cash).

#### Phased PRs

| PR | Scope |
| ---- | ------- |
| **L10.1** | Parked carts API + cashier UI |
| **L10.2** | Offline count queue + conflict |  
| **L10.3** | Explicitly **no** offline card capture |

---

### L11 — CUSTOMER_ASSIST / planogram vision

#### Purpose

Large-store floor help; vision planogram is v2+.

#### Pack deps (already in Retail OS catalog)

`CUSTOMER_ASSIST` requires `SECTIONS` + `TEAM`; soft `SHIFTS` for on-duty routing.

#### v1 assist (if ever prioritized)

```
ticket: section_id, note, created_by
route → SECTION_LEAD on shift else section staff
SLA timer; complete/cancel
```

#### Vision

Detailed plan: [`docs/PLANOGRAM_VISION_PLAN.md`](./PLANOGRAM_VISION_PLAN.md) — non-AI planogram structure (PG1) → human photo audits (PG2) → sidecar CV reusing OSS pipeline shape (PG3). Do **not** fork OSS apps or invent novel detectors. Deferred until L1–L3 and Mode L demand.

**Priority:** lowest vs TEAM → STOCK → POS → SHIFTS → L1–L3.

---

## 6. Cross-role integration map

| Next-layer feature | Other role impact |
| -------------------- | ------------------- |
| L1 GP SUCCESS | Driver/retailer delivery pay; finance ledger |
| L1 Firebase | All role logins |
| L2 CT de-demo | Supplier CT flags unchanged; retailer pulse new |
| L3 sell-through | Supplier suggestions; **no** WH/factory UI |
| L4 negotiations | Driver propose; supplier resolve; retailer notif |
| L5 Soliq | Delivery fiscal; optional POS; finance audit |
| L6 local SKUs | Retailer only |
| L7 quarantine bridge | Claims + WH reverse + retailer stock |
| L8–L11 | Retailer-scale; Kafka BI optional |

---

## 7. Real-world edge cases (must design tests)

| Case | Expected |
| ------ | ---------- |
| GP password wrong in GSM | No stub SUCCESS; ops alert |
| OTP works on emulator flag in release | Blocked by checklist / build config |
| Family list empty after API restart | Fixed by L2.1 |
| POS sells 10 units; suggestion still wholesale-only | Fail L3 until sell-through factor lands |
| Negotiate then store receive | Receive resolved qty only |
| Claim without prior store receive | WH reverse only; no store panic |
| Local SKU appears on supplier demand | Forbidden |
| Multi-org cashier wrong shop | Org picker + location scope fail closed |
| Offline count vs concurrent sale | Conflict path, not silent overwrite |
| Soliq timeout | FISCAL_FAILED + retry; no silent COMPLETED |

---

## 8. Security & compliance

- GP secrets only in GSM/ESO — never commit  
- Firebase SA least privilege  
- Soliq EDS keys in GSM; audit force-complete  
- Negotiations: supplier-scoped resolve; driver can only propose on assigned stop  
- Local SKUs: tenant-scoped; no leak into other retailers’ catalogs  
- HQ analytics: OWNER/ADMIN + location ACL  
- PCI: still no PAN storage; POS card via gateway  

---

## 9. Testing strategy

| Layer | What |
| ------- | ------ |
| Ops smoke | `cloud_smoke_ssmr.sh`; GP SUCCESS marker; Firebase OTP manual matrix |
| Unit | Sell-through rollup math; claim→quarantine qty; negotiation resolve |
| Integration | Spanner: sale → DemandAdjustments → suggestion; claim → QUARANTINE |
| E2E SSMR | Re-enable negotiation e2e; receive→claim→reverse; auto-order place/skip |
| Parity | Feature × Android × iOS × Desktop for retailer; supplier negotiation trio |
| Negatives | Local SKU never in supplier suggestions; CT has zero mock constants in release |

---

## 10. Phased delivery (epic plan)

### Phase A — Field unlock + CORE (parallel with Retail OS P0)

| Epic | Scope | Depends |
| ------ | ------- | --------- |
| **A1 / L1** | GP SUCCESS + Firebase OTP | Owner secrets |
| **A2 / L2.1–L2.3** | Family, tokens, CT de-demo | Retail OS P0–P1 preferred |
| **A3 / L2.4** | AutoOrderWorker (history-based OK) | Durable settings |

### Phase B — Flywheel (after Retail OS P3–P4)

| Epic | Scope | Depends |
|------|-------|---------|
| **B1 / L3** | Sell-through → reorder → supplier signals | P3/P4 |
| **B2** | AutoOrderWorker consumes sell-through | A3 + B1 |

### Phase C — Strong follow-ons

| Epic | Scope | Depends |
| ------ | ------- | --------- |
| **C1 / L4** | Re-enable quantity negotiations | Retail OS P0–P1 scheduled (process gate) |
| **C2 / L7** | Claim ↔ store quarantine bridge | P3 + claims |
| **C3 / L6** | Local POS SKUs | P3/P4 |
| **C4 / L5** | Soliq OFD (+ optional POS fiscal) | Legal need + L1 soft |

### Phase D — Later scale

| Epic | Scope | Depends |
| ------ | ------- | --------- |
| **D1 / L8** | Multi-org phones | P1 + product decision |
| **D2 / L9** | HQ analytics + Kafka BI | P2 + POS data |
| **D3 / L10** | Parked carts + offline count | Stable P4 |
| **D4 / L11** | CUSTOMER_ASSIST; vision later | SECTIONS + TEAM |

### Production policy recommendation

- May run **CORE-only** retailers with L1 ops complete.  
- **Do not** market multi-scale retail without Retail OS P0–P5 code-complete.  
- **Do not** flip `PEGASUSX_ENV=production` until L1 GP SUCCESS + production profile gates pass.  
- L3 flywheel is the first **differentiator** epic after Retail OS stock/POS.  
- L4–L7 strengthen logistics integrity; L8–L11 are scale polish.

---

## 11. Suggested implementation order (“minimum wage” path)

1. **L1** — secrets + OTP (unblocks every field test)  
2. Finish **Retail OS P0–P1** (process gate from companion plan)  
3. **L2.1–L2.3** — stop data loss / mock CT / fake token OK  
4. Retail OS **P3 → P4**  
5. **L3** sell-through bridge (+ **L2.4** auto-order execution)  
6. **L7** quarantine bridge (claims trust)  
7. **L4** negotiations if drivers need propose/approve more than claims  
8. **L6** local SKUs when POS pilots complain “catalog empty”  
9. **L5** Soliq when counsel/pilot requires  
10. **L8–L11** only under Mode L pressure  

---

## 12. Key decisions

| Decision | Choice | Rationale |
| ---------- | -------- | ----------- |
| Sequencing | Retail OS before big follow-ons | Avoid second competing epic |
| Negotiations meaning | Delivery qty propose/resolve | Matches code; not RFQ pricing |
| Sell-through sink | DemandAdjustments + SellThroughDaily | Reuse replenishment spine |
| Local SKUs | Retailer catalog namespace | Keep supplier ATP clean |
| Claim/store bridge | Side effects on claim lifecycle | One SoT for dispute |
| POS fiscal | Optional after delivery Soliq | Legal complexity |
| Offline POS sales | Still out of scope in L10 | Cash/stock split-brain |
| WH/factory vs POS | Events only | Ecosystem boundary |
| Family | Migrate to TEAM or durable contacts | RAM list is unacceptable |

---

## 13. Open questions (product confirmation)

1. Auto-order execution: **auto-submit** vs **draft cart + notify buyer**?  
2. Negotiation vs partial-offload: which wins when both could apply?  
3. Soliq required for **POS** in v1 pilots or delivery-only?  
4. Local SKU barcode collision policy with Pegasus SKUs?  
5. Multi-org staff: allow in v1.1 or freeze single-org until Mode L pilot?  
6. Family: contacts-only durability or force TEAM invites?  
7. HQ analytics: Spanner rollups vs external BI consumer first?  
8. GP card vault: needed for POS tenders or delivery-only tokenization?  

---

## 14. Success metrics (rollup)

| Epic | Metric |
| ------ | -------- |
| L1 | Card SUCCESS e2e; OTP on device; prod profile green |
| L2 | Zero family data loss on restart; CT mock strings = 0 in release; auto-order run audit visible |
| L3 | Sell-through moves suggestions within one tick; supplier source chip visible |
| L4 | Negotiation e2e PASS; 410 gone |
| L5 | MY_SOLIQ SUCCESS on sandbox order |
| L6 | POS sale of local SKU with stock decrement; absent from supplier suggestions |
| L7 | Claim → QUARANTINE on_hand proven in Spanner |
| L8 | Two-org login picker &lt; 30s |
| L9 | Desktop HQ sales-by-location matches sum of stores |
| L10 | Count conflict reproduced in test; park/resume sale |
| L11 | Assist ticket SLA complete path (if built) |

---

## 15. Documentation deliverables

- This file: `docs/NEXT_LAYER_ECOSYSTEM_PLAN.md`  
- Update `artifacts/OWNER_SECRETS_HANDOFF_*.md` after L1  
- Update `context/parity-ledger.md` for negotiations + CT de-demo  
- Expand `docs/ECOSYSTEM_FEATURES_BY_ROLE.md` with L3/L6/L7  
- `docs/RETAILER_SELL_THROUGH.md` (new)  
- `docs/RETAILER_LOCAL_SKU.md` (new)  
- `docs/CLAIM_STORE_STOCK_BRIDGE.md` (new)  
- Link from Retail OS plan §20 (“second major feature”) → this document  

---

## 16. Risk register

| Risk | Mitigation |
| ------ | ------------ |
| Scope steals focus from Retail OS | Hard process gate: no L4+ epic start before P0–P1 scheduled |
| GP stub false confidence | Prod profile rejects doc-* / stub refs |
| Sell-through double counts voids | Signed adjustments; tests |
| Claim + reverse + store triple movement | Idempotency keys per claim line + bin |
| Soliq latency sticks FISCALIZING | Existing retry/force audit; don’t enable on prod until sandbox green |
| Local SKU pollutes demand | Explicit source filter in L3.3 |
| Negotiation money mismatch | Resolve before fiscal COMPLETED; or credit-note path |
| CT “de-demo” half-done | Gate: grep Mock Data / hardcoded BarMark in CI |

---

## 17. Out of scope (reaffirmed)

- Full ERP/HR/payroll, biometric legal attendance  
- Franchise tax consolidation  
- Forcing POS on pure B2B buyers  
- Competing greenfield epic vs Retail OS P0–P1  
- Offline card / offline POS sale queue  
- Planogram vision (until L11 assist proven)  

**Sibling hardening (other roles / ops):** see [`docs/ECOSYSTEM_HARDENING_GAP_PLAN.md`](./ECOSYSTEM_HARDENING_GAP_PLAN.md) (E1–E16 — supplier CT demo, per-supplier zones, pick/rescue, offline crypto, observability/DR). Not a substitute for L1–L11.

**Program sequence (all plans):** see [`docs/PEGASUSX_MASTER_ROADMAP.md`](./PEGASUSX_MASTER_ROADMAP.md).

---

## 18. Immediate next steps after plan approval

1. Resolve Open Questions §13 (especially auto-submit vs draft, Soliq POS, family path).  
2. Execute **L1** owner secrets (GP password + Firebase Phone/SHA) — unblocks all field proof.  
3. Keep Retail OS P0–P5 as primary engineering track.  
4. Schedule **L2.1–L2.3** in parallel with Retail OS P0 (low conflict).  
5. Freeze L3 event names + `RetailerSellThroughDaily` in `packages/types` once P3/P4 date is known.  
6. Do **not** flip negotiation gate until Phase C is explicitly scheduled.  

---

## Appendix A — Epic × Retail OS dependency matrix

| Epic | P0 | P1 TEAM | P2 LOC | P3 STOCK | P4 POS | P5 SHIFTS |
| ------ | ---- | --------- | -------- | ---------- | -------- | ----------- |
| L1 GP/Firebase | — | — | — | — | — | — |
| L2 CORE harden | soft | soft | — | — | — | — |
| L3 sell-through | — | — | soft | **hard** | **hard** | — |
| L4 negotiations | process | process | — | soft | — | — |
| L5 Soliq | — | — | — | — | soft POS | — |
| L6 local SKUs | — | — | — | **hard** | **hard** | — |
| L7 quarantine | — | — | — | **hard** | — | — |
| L8 multi-org | — | **hard** | soft | — | — | — |
| L9 HQ analytics | — | soft | **hard** | soft | **hard** | soft |
| L10 offline/park | — | — | — | soft | **hard** | — |
| L11 assist | — | **hard** | — | — | — | soft |

---

## Appendix B — File touch map (implementation anchors)

**Ops / pay / auth:** `payment/global_pay_*.go`, `bootstrap/bootstrap.go`, GSM/ESO, Firebase console, client `FirebaseAuthHelper` / plists  

**CORE:** `retailer/core_handlers.go`, `repository_settings_durable.go`, `platform/handlers.go` (device-token), `kafka/notification_dispatcher.go`, `controltower/*`, `simulator/control_tower.go`, retailer CT screens  

**Flywheel:** `retailer/pos.go`, `retailer/store_stock.go`, `replenishment/reorder_suggestion_batch.go`, `demand/worker_sensing.go`, `packages/types`  

**Negotiations:** `order/negotiation_disabled.go`, `order/negotiation.go`, `negotiation_list.go`, driver/supplier clients, `cmd/ssmr-smokecheck/e2e_*.go`  

**Fiscal:** `order/fiscal_*.go`, ConfigMap `FISCAL_PROVIDER`, regulatory docs  

**Local SKU:** new `retailer/local_sku.go` + migrations; POS search clients  

**Returns bridge:** `claims/service.go`, `returns/*`, `retailer/store_stock.go`  

**Docs:** `context/parity-ledger.md`, `artifacts/OWNER_SECRETS_HANDOFF_*.md`, `docs/ECOSYSTEM_FEATURES_BY_ROLE.md`  

---

## Appendix C — Mapping from recommendation list

| Recommendation | Plan ID |
| ---------------- | --------- |
| 1. Global Pay SUCCESS + Firebase real OTP | **L1** |
| 2. Retail OS CORE hardening | **L2** |
| 3. Sell-through → reorder bridge | **L3** |
| 4. Supplier quantity negotiations | **L4** |
| 5. Soliq OFD | **L5** |
| 6. Local / manual POS SKUs | **L6** |
| 7. Returns / reverse logistics to store stock | **L7** |
| 8. Multi-org staff phones | **L8** |
| 9. Franchise / HQ analytics | **L9** |
| 10. Offline-tolerant count + parked POS carts | **L10** |
| 11. CUSTOMER_ASSIST / planogram vision | **L11** |
| Skip ERP/force POS/competing epic | §1 Non-goals, §17 |

---

*End of plan.*
