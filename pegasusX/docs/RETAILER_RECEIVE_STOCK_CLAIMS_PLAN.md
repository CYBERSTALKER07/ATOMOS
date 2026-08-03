# Rich Receiver ↔ Store Stock ↔ Claims Merge Plan

**Status:** Implementation plan (code-grounded)  
**Repo:** `/Users/shakhzod/Desktop/V.O.I.D/pegasusX`  
**Date:** 2026-08-02  
**Maps to:** Next-Layer **L7** (expanded) · Retail OS **STORE_STOCK** · existing claims / reverse logistics spine  
**Clients:** Retailer Android · iOS · Desktop (receiver + stock roles); supplier + warehouse notified via existing queues  

---

## 0. Executive summary

### Problem

Driver-side visible damage / OS&D and warehouse reverse logistics already work. After goods enter the **retailer store ledger**, there is no complete sink:

1. **Visible damage at dock** — driver withdraws/excepts units correctly, but store receive / stock managers are not always driven by those same line facts (risk: putaway of units that never should be sellable).  
2. **Concealed damage after close** — order COMPLETED, receive done, stock manager opens a case later and finds damage → needs photo + form + chargeback/return against an order from history.  
3. **Return window** — today a **global** `CLAIM_WINDOW_HOURS` (default 48h). Product needs **supplier/warehouse-configurable** windows (8h / 24h / custom) after which retailer stock **cannot** file on that `order_id`.

### Decision (best way)

> **Do not build a second “chargeback request” product.**  
> Extend the **one claims spine** already in `claims/` + reverse logistics + store stock bins.  
> Split only by **when** damage is discovered (dock vs post-putaway), not by parallel APIs.

| Discovery time | Existing mechanism | Store stock effect (to add) |
|----------------|--------------------|-----------------------------|
| At delivery (visible) | Driver exception → `CreateFromDriverException` / refuse / partial | **Never RECEIVE** those qty into FLOOR/BACKROOM; optional draft QUARANTINE or exclude lines |
| After COMPLETED (concealed) | `FileRetailerClaim` + `CONCEALED_DAMAGE` / `DAMAGED` + photo | Move OnHand → **QUARANTINE** (`CLAIM_HOLD`); on approve → reverse WH ticket (exists) |

**UX merge:** Receiver dock + Stock screens share the same order-line truth and the same “Request return / chargeback” sheet (claims API under the hood).

**Gap audit (2026-08-02):** Expanded catalog **§16** (G1–G25) covers product, backend, GCP, DevOps, and cross-role. Highest P0: TEAM claim auth (G6), quarantine bridge (G8), receive vs driver exceptions (G9).

---

## 1. Current baseline (code truth)

| Area | Status | Anchor |
|------|--------|--------|
| Driver OS&D → claim | Wired | `claims.CreateFromDriverException` |
| Retailer file claim post-COMPLETED | Wired | `FileRetailerClaim`; Android `FileClaimHost` / orders |
| Claim types | Wired | `DAMAGED`, `MISSING`, `CONCEALED_DAMAGE`, `TEMPERATURE`, `TAMPER`, `OTHER` |
| Photo required for damage types | Wired | `ErrEvidenceRequired` |
| Chargeback pricing from order lines | Wired | `PriceClaimLinesWithPrior` |
| Cumulative qty caps | Wired | prior claims reserve qty |
| Global claim window (API enforce) | Wired | `CLAIM_WINDOW_HOURS` / default **48h** from COMPLETED |
| Supplier UI/API to set claim/return window | **Missing** | No portal/Android/iOS settings; not in supplier role-row |
| Warehouse UI/API to set/view claim/return window | **Missing** | No portal/Android/iOS settings; exception-map `window_hours` is unrelated lookback |
| Per-supplier / per-WH return window (product) | **Missing — required RS1** | See §4 |
| WH reverse ticket on claim | Wired | `openReverseLogistics` |
| Store receive sessions | Wired | `POST …/stock/receive-sessions` |
| Store QUARANTINE bin | Wired | `BinQuarantine` |
| Claim → QUARANTINE hold | **Shipped (B1)** | File physical claim → FLOOR/BACKROOM→QUARANTINE (`CLAIM_HOLD`); approve → leave quarantine (`RETURN`); reject → restore FLOOR |
| Claim → store QUARANTINE / exclude from receive | **Missing bridge** | Next-Layer L7 stub |
| TEAM-scoped claim auth (`retailer_org_id`) | **Likely gap** | `FileRetailerClaim` still compares `o.RetailerID` to JWT `Subject` (breaks staff users) |
| Stock-first UX (pick order → SKU → reason → photo) | Partial | Claims from Orders; not from Stock quarantine workflow |

---

## 2. Product principles

1. **One liability SoT** — `claims.Claim` (+ chargeback settlement). No `StockChargebackRequest` table.  
2. **Stock is mirror, not finance** — ledger movements follow claim lifecycle; money follows approve/settle.  
3. **Visible ≠ Concealed** — different default claim types and evidence rules; same approve/reject path.  
4. **Window is policy** — owned by supplier (commercial) with optional warehouse override for reverse dock SLA; snapshot onto the order at COMPLETED.  
5. **Fail closed after window** — UI hides CTA; API returns `claim_window_expired`.  
6. **Role split** — RECEIVER confirms dock/putaway; STOCK_CLERK/MANAGER files concealed claims; OWNER sees money; cashiers never file returns.

---

## 3. Two timelines (merge rules)

```
                    DELIVERY STOP
                          │
          ┌───────────────┴───────────────┐
          ▼                               ▼
   VISIBLE damage                    Clean / sealed
   (driver can see)                  (retailer signs)
          │                               │
   Driver exception /                Order → COMPLETED
   refuse / partial                  Receive session
   qty NOT in accepted               accepted → BACKROOM/FLOOR
          │                               │
          ▼                               ▼
   Reverse logistics (WH)        Later: open case / inspect
   Retailer stock: skip          Find concealed damage
   those lines (or never         │
   putaway)                      ▼
                          File claim (CONCEALED_DAMAGE…)
                          photo + reasons + order_id
                          → QUARANTINE hold
                          → notify supplier + warehouse
                          → approve → reverse + chargeback
```

### Rule A — Visible at dock (driver path)

```
on driver exception lines L for order O:
  mark O.line accepted_qty excludes L  (or exception qty field — existing)
  if STORE_STOCK receive draft exists:
    prefill receive lines with accepted only
    never auto-RECEIVE exception qty
  notify retailer users with dock.receive + stock.view:
    "Driver reported damage/missing on {skus} — excluded from putaway"
  open/keep WH reverse as today
```

**Do not** also create a retailer `CONCEALED_DAMAGE` claim for the same qty (double liability). Link UI: “Already handled at delivery — see claim/exception {id}.”

### Rule B — Concealed after receive (stock path)

```
on FileRetailerClaim (post COMPLETED, within window):
  validate order belongs to retailer org
  validate lines ⊆ order lines and ≤ remaining claimable
  require photo for DAMAGED | CONCEALED_DAMAGE | TEMPERATURE | TAMPER
  reason: enum (+ OTHER requires custom text)
  persist claim OPEN
  if store receive confirmed for O:
    for each line: movement CLAIM_HOLD → QUARANTINE
  notify supplier claims queue + warehouse reverse inbox
  (existing openReverseLogistics — keep; may open ticket at FILE or APPROVE — product pick below)
```

**Recommendation:** keep reverse ticket open at **FILE** (today) so WH can prepare dock; settlement money only on **APPROVE**.

---

## 4. Return / claim window policy (supplier + warehouse feature)

### 4.0 Gap statement (explicit deliverable)

**Today:** only ops/env `CLAIM_WINDOW_HOURS` (default 48h). Retailer file-claim is enforced server-side; **neither supplier nor warehouse can configure the window** in product UI/API.

**Required in this plan:** first-class **Claim / return window** settings on:

| Side | Role-row | Can set? | Can view? |
|------|----------|----------|-----------|
| **Supplier** | Portal + Android + iOS | **Yes** (commercial SoT) | Yes |
| **Warehouse** | Portal + Android + iOS | **Yes** for reverse-dock SLA; optional extend of retailer file window only if policy allows | Yes (effective window on orders they fulfill) |
| **Retailer** | All three | No | Yes — eligibility + countdown on order / stock claim CTA |
| **Platform env** | SSMR ConfigMap | Fallback default only when supplier has no row | — |

Do **not** confuse with supplier exception-map query `window_hours` (map lookback filter only).

### 4.1 Model

```
SupplierReturnPolicy  (commercial SoT — who may file how long)
  SupplierId
  DefaultWindowHours           # presets 8 | 24 | 48 | 72 | custom 1..168 (cap)
  ConcealedDamageWindowHours   # optional; default = DefaultWindowHours
  VisibleExceptionWindowHours  # optional; usually same (driver path often same-day)
  RequirePhoto                 # default true for damage types
  AllowExpiredClaims           # default false
  UpdatedAt, UpdatedByUserId

WarehouseReturnPolicy  (ops — reverse dock + optional commercial extend)
  WarehouseId
  SupplierId                   # null = warehouse default for all suppliers it serves
  ReverseDockSLAHours          # how fast WH expects to accept physical returns
  RetailerFileWindowHours      # optional override applied only if CanOverrideRetailerWindow
  CanOverrideRetailerWindow    # bool, default false — when true, resolve() may use max/min rules below
  UpdatedAt, UpdatedByUserId

OrderClaimWindowSnapshot  (immutable after COMPLETED — store on Orders row or JSON)
  ClaimWindowHours
  ClaimWindowEndsAt
  PolicySource                 # SUPPLIER | WAREHOUSE_OVERRIDE | ENV_DEFAULT
  SupplierPolicyVersion / hash (audit)
```

**Resolve algorithm (at COMPLETED):**

```
resolve_window(order):
  base = SupplierReturnPolicy[order.SupplierId].DefaultWindowHours
       ?? ENV CLAIM_WINDOW_HOURS
       ?? 48
  # optional stricter concealed window if claim type known later — snapshot uses Default at COMPLETED;
  # FileClaim may use min(snapshot, ConcealedDamageWindowHours) if product wants stricter concealed-only
  wh = WarehouseReturnPolicy[order.WarehouseId, order.SupplierId]
  if wh != nil && wh.CanOverrideRetailerWindow && wh.RetailerFileWindowHours > 0:
    # v1 rule: warehouse may only LENGTHEN (goodwill / cold-chain), never shorten below supplier base
    hours = max(base, wh.RetailerFileWindowHours)
    source = WAREHOUSE_OVERRIDE
  else:
    hours = base
    source = SUPPLIER or ENV_DEFAULT
  return hours
```

### 4.2 Snapshot at COMPLETED

```
on order → COMPLETED:
  hours, source = resolve_window(order)
  order.ClaimWindowHours = hours
  order.ClaimWindowEndsAt = CompletedAt + hours
  order.ClaimWindowPolicySource = source
```

File claim uses **snapshot**, not live policy (supplier can’t shorten an in-flight window mid-dispute).  
Live policy edits apply to **future** COMPLETED orders only.  
Lengthening an existing order: `POST /v1/orders/{id}/claim-window/extend` (supplier ADMIN or platform admin) + audit log.

### 4.3 Supplier UX (role-row parity)

**Surface:** Settings → **Returns & claim window** (portal `/settings/return-policy`; Android/iOS Settings).

| Control | Behavior |
|---------|----------|
| Preset chips | **8h · 24h · 48h · 72h · Custom** |
| Custom hours | Integer 1–168; validate client + server |
| Concealed vs default | Optional second field; helper text |
| Require photo | Toggle (default on) |
| Preview | “Retailers may file until {hours}h after delivery complete.” |
| Audit | Last updated by / at |

Permissions: supplier `ADMIN` / org role that can manage commercial policy (mirror pricing-settings perm pattern).

### 4.4 Warehouse UX (role-row parity)

**Surface:** Settings → **Returns & reverse SLA** (portal; Android/iOS warehouse settings).

| Control | Behavior |
|---------|----------|
| Reverse dock SLA | Presets **8h · 24h · 48h · Custom** (ops KPI; does not by itself change retailer CTA) |
| Override retailer file window | Off by default; when on, set hours with warning: “May only extend supplier window (v1)” |
| Per-supplier override | Optional table: supplier → hours (advanced) |
| Read-only effective view | On reverse ticket / order: show `ClaimWindowEndsAt` + policy source |

Warehouse **cannot** silently cut retailer window below supplier base in v1.

### 4.5 Retailer visibility

- Order detail + stock “Request return”: show **Eligible until {local time}** or **Window closed**.  
- `GET /v1/orders/{id}/claim-eligibility` → `{ eligible, ends_at, window_hours, policy_source }`.  
- No retailer edit of window.

### 4.6 API

```
# Supplier (new)
GET/PUT  /v1/supplier/return-policy
GET      /v1/supplier/return-policy/audit          # optional

# Warehouse (new)
GET/PUT  /v1/warehouse/return-policy
GET/PUT  /v1/warehouse/return-policy/suppliers/{supplierId}   # optional per-supplier

# Order (extend existing)
GET      /v1/orders/{id}/claim-eligibility
POST     /v1/orders/{id}/claim-window/extend      # body: { hours_to_add, reason }
POST     /v1/orders/{id}/claims                   # enforce order.ClaimWindowEndsAt
```

Replace global-only `s.window` in `FileRetailerClaim` with `order.ClaimWindowEndsAt` when set; fallback to env default for legacy orders without snapshot.

### 4.7 Clients checklist (RS1 exit criteria)

| Client | Supplier set window | Warehouse set/view | Retailer see countdown |
|--------|---------------------|--------------------|------------------------|
| Portal / desktop | Required | Required | Required (retailer desktop) |
| Android | Required | Required | Required |
| iOS | Required | Required | Required |

Parity ledger rows: **Claim/return window policy** for supplier + warehouse; **Claim eligibility countdown** for retailer.

---

## 5. Retailer UX (merged receiver + stock)

### 5.1 Receiver (dock / receive session)

- Line list shows: ordered · driver-accepted · exception · already-claimed.  
- Confirm putaway **only** for accepted clean qty.  
- Banner when driver exceptions exist.  
- Deep link to exception/claim detail (read-only).

### 5.2 Stock manager — “Request return / chargeback”

Entry points:

1. Stock → Quarantine / Movements → **New return request**  
2. Order history → order detail → **File claim** (exists)  
3. Receive session detail → **Report concealed damage** (after confirm)

Form (single component, three clients):

| Field | Rule |
|-------|------|
| Order | Picker from COMPLETED history (search id) — must be in-org + in window |
| SKU / lines | From order lines; qty ≤ remaining claimable − on-hand awareness warn |
| Reason (hardcoded) | `DAMAGED`, `CONCEALED_DAMAGE`, `MISSING`, `TEMPERATURE`, `TAMPER`, `WRONG_ITEM`, `EXPIRED`, `OTHER` |
| Custom note | Required if OTHER; optional otherwise (max N chars) |
| Photos | ≥1 for damage/temp/tamper/concealed |
| Location / bin | Default current stock bin; system moves to QUARANTINE on submit |

Map new UI reasons → existing `ClaimType` (+ line `Reason` string). Add `WRONG_ITEM` / `EXPIRED` as types or as `OTHER`+reason code in v1 — prefer **new ClaimTypes** if finance reports need them.

### 5.3 Permissions

| Action | Perm |
|--------|------|
| Confirm receive | `dock.receive` |
| File concealed claim | `stock.adjust` or new `claim.file` (OWNER/ADMIN/MANAGER/STOCK_CLERK) |
| View claim status | `stock.view` / `order.place` |
| Approve claim | Supplier (existing) |

---

## 6. Notifications

| Event | Retailer | Supplier | Warehouse |
|-------|----------|----------|-----------|
| Driver visible exception | Receiver + stock.view | Claims/exceptions | Reverse inbox |
| Concealed claim filed | Filer + owner digest | **Claims queue** (existing) | **Reverse ticket** (existing) |
| Claim approved/rejected | Stock + buyer | Ledger | Close/adjust ticket |
| Window expired | CTA hidden | — | — |

Use existing outbox `CLAIM_*` + reverse events; add `STORE_STOCK_CLAIM_HOLD` for retailer WS rooms.

---

## 7. Backend algorithms (canonical)

### 7.1 Receive confirm (merge)

```
confirm_receive(session):
  for line in session.lines:
    excl = driver_exception_qty(order, sku) + prior_approved_or_open_claim_hold(order, sku)
    max_accept = ordered - excl
    if line.qty > max_accept: reject
    RECEIVE +qty → BACKROOM|FLOOR
```

### 7.2 File claim (extend)

```
FileRetailerClaim(...):
  org = retailer_org_id from JWT (fix Subject==RetailerId for TEAM)
  assert now <= order.ClaimWindowEndsAt
  assert photo rules
  insert claim
  if receive_confirmed(order):
    quarantine_hold(lines)
  openReverseLogistics(...)  # existing
  notify
```

### 7.3 Approve / reject (extend L7)

```
APPROVE:
  settle chargeback (existing modes)
  QUARANTINE −qty → RETURN_TO_SUPPLIER (or wait WH ack then −qty)
  syncReorderCurrentStock

REJECT:
  QUARANTINE → restore previous bin (BACKROOM/FLOOR snapshot on hold)
  or WASTE if retailer marks unsellable despite reject (policy)
```

---

## 8. Edge cases the earlier plans under-counted

| # | Case | Expected behavior |
|---|------|-------------------|
| 1 | Driver already claimed qty; stock files again | Block / reduce to remaining claimable only |
| 2 | Claim before store receive | No store movement; WH reverse only; receive later must exclude claimed qty |
| 3 | Partial putaway then concealed on rest | Hold only claimed qty; other units stay sellable |
| 4 | POS sold unit before concealed claim | Cap claim by `min(remaining_order_qty, on_hand + documented)`; prefer require on_hand ≥ qty or MANAGER override + audit |
| 5 | Window expires mid-form | API 409 `claim_window_expired`; keep draft photo locally discarded |
| 6 | Supplier shortens policy after COMPLETED | Snapshot wins; no retroactive cut |
| 7 | Supplier lengthens policy | Does not auto-extend old orders; optional admin extend |
| 8 | Multi-location receive | Claim scoped to order’s delivery location stock |
| 9 | TEAM staff JWT | Authorize via `retailer_org_id`, not `sub == retailer_id` |
| 10 | Mixed visible + concealed on same SKU | Separate claims/exception ids; qty math additive under cap |
| 11 | Temperature / seal (visible to driver) | Prefer driver path; retailer concealed only if not reported at stop |
| 12 | Wrong item (label vs content) | New reason; photo of label + contents |
| 13 | Expired / near-expiry inside case | Reason EXPIRED; may be commercial not logistics — supplier policy flag `AllowExpiredClaims` |
| 14 | Duplicate photo upload / double submit | Idempotency-Key on file claim |
| 15 | Credit vs gateway refund | Existing `settlement_mode`; stock movement independent |
| 16 | Auto-approve small claims | Existing `CLAIM_AUTO_APPROVE_MAX_MINOR`; still quarantine |
| 17 | Warehouse can’t accept return physically | Claim stays APPROVED financially; stock WASTE + note; ticket CANCELLED_WITH_CREDIT |
| 18 | Shop-closed / partial offload interaction | Exceptions at stop reduce receivable qty; concealed window still from final COMPLETED |
| 19 | Negotiation (L4) changed qty | Claim against **post-negotiation** order lines |
| 20 | Local SKU (L6) on same shelf | Not claimable via Pegasus order claim; separate path |
| 21 | Fiscal already COMPLETED | Chargeback/credit note path only; don’t reopen fiscal blindly |
| 22 | Receiver confirms all good, then same user files concealed | Allowed within window (human error / sealed goods) |
| 23 | Quarantine qty > on_hand (data drift) | Clamp + force count task |
| 24 | Multi-supplier order (if ever) | Claim lines split per supplier_id |

---

## 9. What not to do

- Don’t invent `POST /v1/retailer/stock/chargebacks` as a second ledger.  
- Don’t let stock adjust “damage” without a claim if money/return is expected.  
- Don’t use live global env window alone once policies ship.  
- Don’t notify stock to putaway units the driver already withdrew.  
- Don’t put CV/planogram in this epic.

---

## 10. Phased PRs

### Phase RS0 — Auth + eligibility (unblocks TEAM)

| PR | Scope |
|----|-------|
| **RS0.1** | Fix claim auth to `retailer_org_id` + `claim.file` / `stock.adjust` perm |
| **RS0.2** | `GET …/claim-eligibility` + clients hide CTA when expired |

### Phase RS1 — Claim/return window on supplier + warehouse (was missing)

| PR | Scope |
|----|-------|
| **RS1.1** | DDL: `SupplierReturnPolicy`, `WarehouseReturnPolicy`; Order snapshot columns |
| **RS1.2** | `GET/PUT /v1/supplier/return-policy` + enforce auth |
| **RS1.3** | Supplier portal + Android + iOS Settings UI (8/24/48/72/custom chips) |
| **RS1.4** | `GET/PUT /v1/warehouse/return-policy` (+ optional per-supplier) |
| **RS1.5** | Warehouse portal + Android + iOS Settings UI (SLA + extend-only override) |
| **RS1.6** | Snapshot `ClaimWindowEndsAt` + `PolicySource` on COMPLETED |
| **RS1.7** | Enforce snapshot in `FileRetailerClaim`; env = fallback only |
| **RS1.8** | Retailer eligibility API + countdown on order/stock claim CTAs (3 clients) |
| **RS1.9** | `POST …/claim-window/extend` + audit; parity ledger rows |

### Phase RS2 — Dock ↔ receive merge (visible damage)

| PR | Scope |
|----|-------|
| **RS2.1** | Receive confirm respects driver exception qty |
| **RS2.2** | Retailer notif + receive UI banners |
| **RS2.3** | Dedup: no double claim for same exception lines |

### Phase RS3 — Concealed path + store sink (core ask)

| PR | Scope |
|----|-------|
| **RS3.1** | On claim file → QUARANTINE `CLAIM_HOLD` if received |
| **RS3.2** | Approve/reject → RETURN / restore / WASTE |
| **RS3.3** | Stock UI “Request return” sheet (order picker, reasons, photo) — Android/iOS/Desktop |
| **RS3.4** | Wire reasons enum + OTHER custom; optional WRONG_ITEM/EXPIRED |
| **RS3.5** | E2E: complete → receive → concealed claim → quarantine → supplier approve → WH reverse |

### Phase RS4 — Harden

| PR | Scope |
|----|-------|
| **RS4.1** | Sold-before-claim policy + manager override (G14 money policy docs + G15 location) |
| **RS4.2** | Admin window extend audit |
| **RS4.3** | Parity ledger + REAL_WORLD_CASE_MATRIX rows from §8 + §16 |
| **RS4.4** | Credit-note from claim lines + ConfigMap flag (G13) |
| **RS4.5** | WH inbound first-class `ClaimId` + unified returns hub (G4, G5) |

### Phase RS5 — Cloud / DevOps / evidence integrity

| PR | Scope | Gaps |
|----|-------|------|
| **RS5.1** | Idempotency on `POST …/claims` + client keys | G11, G25 |
| **RS5.2** | Reverse open via outbox/retry + metrics/alerts | G12, G18 |
| **RS5.3** | GCS evidence fail-closed; reject placeholder URIs in prod | G16, G21 |
| **RS5.4** | Evidence max bytes + MIME sniff; ConfigMap keys | G17, G19 |
| **RS5.5** | CLAIM_FILED payload includes `warehouse_id`; role-aware FCM | G22 |
| **RS5.6** | SSMR e2e: receive → concealed+photo → quarantine → WH inbound | G20 |
| **RS5.7** | ConfigMap: `CLAIM_WINDOW_HOURS`, `CLAIM_AUTO_APPROVE_MAX_MINOR`, `CREDIT_NOTE_AUTO_FROM_CLAIM`, `CLAIM_EVIDENCE_MAX_BYTES` | G19 |
| **RS5.8** | Terraform/GCM alerts for reverse-open fail + window-expired rate | G18 |

---

## 11. Recommended implementation order

1. **RS0** (TEAM-safe claims + `claim.file` perm) — G6, G7.  
2. **RS3.1–RS3.2** (stock quarantine sink) — G8, G24.  
3. **RS2** dock/receive merge — G9, G23.  
4. **RS5.1 + RS5.3** (idempotency + real GCS evidence) — G11, G16.  
5. **RS3.3** stock UX form — G1.  
6. **RS1** supplier + warehouse windows — G3, G10, G2.  
7. **RS5.2 / RS5.5 / RS5.6** notifications + e2e — G12, G22, G20.  
8. **RS4 + RS5.4/7/8** harden money, media, ConfigMap, alerts.

---

## 12. Open questions

1. Reverse ticket at **FILE** (current) vs only at **APPROVE**? → recommend FILE.  
2. ~~Who owns window SoT?~~ **Decided for plan:** supplier commercial SoT; warehouse may **lengthen only** when `CanOverrideRetailerWindow`; WH SLA is separate. Confirm before RS1.5.  
3. If unit already sold via POS — allow claim with override, or block? → recommend block by default + MANAGER override.  
4. Add `WRONG_ITEM` / `EXPIRED` as first-class `ClaimType` or fold into OTHER?  
5. Quarantine auto on driver visible exception without retailer receive? → recommend **no putaway**, not auto quarantine of ghost qty.  
6. Should concealed claims use a **stricter** `ConcealedDamageWindowHours` than default, or one window for all post-delivery types?  

---

## 13. Success metrics

- Zero sellable OnHand increase for driver-excepted qty (RS2 / G9).  
- Concealed claim → QUARANTINE within same Spanner txn as claim insert (RS3 / G8).  
- TEAM staff can file claims (G6); cashiers cannot (G7).  
- After `ClaimWindowEndsAt`, file claim API **409** and CTA gone.  
- Supplier/warehouse set windows in product UI (RS1 / G3); retailer countdown matches snapshot (G2).  
- Evidence URIs are real GCS in SSMR/prod — zero placeholder accepts (G16/G21).  
- Double-submit file claim returns same `claim_id` (G11/G25).  
- WH inbound row exists for every filed claim with reverse (G12/G20).  
- CLAIM_FILED reaches warehouse room + stock roles (G22).  
- No double chargeback for same SKU qty.

---

## 14. Documentation / cross-links

- Expands Next-Layer **L7** — link this file from `NEXT_LAYER_ECOSYSTEM_PLAN.md`.  
- Update `docs/RETAILER_STORE_STOCK.md` receive rules.  
- Update `docs/CLAIM_ROLE_ROW.md` with stock entry + eligibility + gap closures.  
- Fill `docs/REAL_WORLD_CASE_MATRIX.md` with §8 + §16 rows.  
- `docs/gap-closure/PRODUCTION_CUTOVER.md` — ConfigMap claim/CN/evidence keys (G19).  
- GCS/WI signBlob runbook for evidence (G16).  

---

## 15. Immediate next steps after approval

1. Confirm open questions §12 (sold-before-claim; concealed vs default window hours).  
2. **P0 gaps:** RS0 (G6/G7 auth) → RS3 quarantine (G8) → RS2 receive exclude exceptions (G9).  
3. **P1 cloud:** RS5.1 idempotency + RS5.3 GCS fail-closed evidence (G11/G16).  
4. Stock “Request return” UX (G1) + **RS1** supplier/warehouse windows (G3/G10).  
5. Close RS5 e2e + notif + ConfigMap/alerts (G20/G22/G19/G18).  

---

## 16. Gap catalog (expanded) — problem → solution → integrations

Code-grounded audit of `claims/`, `returns/`, `retailer/store_stock.go`, `creditnote/`, `platform/media_upload.go`, `storage/gcs.go`, `kafka/notification_dispatcher.go`, retailer/supplier/warehouse clients, `infra/k8s/backend-go/configmap.yaml`, `cmd/ssmr-smokecheck/e2e_claims.go`.

For each gap: **Problem** · **Solution** · **Features** · **Integrations** (codebase / GCP / DevOps).

---

### 16.1 Product / UX

#### G1 — Stock-first claim UX missing (orders-only CTA)
| | |
|--|--|
| **Problem** | Concealed damage filed from order detail only (`FileClaimHost` / desktop panel). Stock clerk inspecting OnHand has no native path. |
| **Solution** | Shared “Request return / chargeback” sheet on Stock → same `POST /v1/orders/{id}/claims`. |
| **Features** | Order picker, SKU/qty, hardcoded+custom reasons, photo, eligibility countdown. |
| **Integrations** | `retailer-app-{android,ios,desktop}`; `packages/api-client` / `types`; FCM deep-link to `/stock` + order. |

#### G2 — No claim-eligibility / countdown API
| | |
|--|--|
| **Problem** | Clients learn expiry only via `409 claim_window_expired`. |
| **Solution** | `GET /v1/orders/{id}/claim-eligibility` → ends_at, hours_remaining, policy_source, photo_required. |
| **Features** | Hide/disable CTA; supplier/WH read-only effective window on order. |
| **Integrations** | `claims/handlers.go`; order snapshot; marker `PX_E2E_CLAIM_ELIGIBILITY_OK`. |

#### G3 — Supplier/warehouse window settings UI absent
| | |
|--|--|
| **Problem** | Only env `CLAIM_WINDOW_HOURS`; no product control (see §4). |
| **Solution** | Role-row Settings for `SupplierReturnPolicy` + `WarehouseReturnPolicy`. |
| **Features** | 8/24/48/72/custom; WH SLA; lengthen-only override. |
| **Integrations** | Spanner DDL; supplier/warehouse portal+Android+iOS; ConfigMap env = fallback. |

#### G4 — Warehouse claim tickets are notes-parsed
| | |
|--|--|
| **Problem** | Mobile detects claim via `driverNotes.contains("claim_id=")` — fragile; no structured photos/SKUs. |
| **Solution** | First-class `ClaimId` (+ evidence refs) on `SupplierReturns` / inbound DTO. |
| **Features** | Badge “store return”; open claim detail; thumbnails. |
| **Integrations** | `returns/tickets.go`; Spanner column + index; warehouse Android/iOS/portal. |

#### G5 — Dual reverse-logistics UIs
| | |
|--|--|
| **Problem** | WH mixes `SupplierReturns` (claim/OS&D) vs `ReverseLogisticsTasks` (credit-note) — two mental models. |
| **Solution** | Unified inbound hub with `source: CLAIM \| DRIVER_EXCEPTION \| CREDIT_NOTE`. |
| **Features** | One list/filter/receive flow. |
| **Integrations** | `returns` + `creditnote/repository_spanner.go`; warehouse `ReturnsScreen` / iOS Returns. |

---

### 16.2 Backend domain

#### G6 — TEAM auth: `Subject` vs `retailer_org_id` **(P0)**
| | |
|--|--|
| **Problem** | `FileRetailerClaim` / `ListOrderClaims` compare `o.RetailerID` to `claims.Subject` (`claims/service.go`). Staff JWT `sub`=user → always forbidden. |
| **Solution** | `ResolveRetailerOrgID` + `HasRetailerPerm(claim.file \| stock.adjust)`. |
| **Features** | TEAM staff can file/list; tests with `RetailerOrgID`. |
| **Integrations** | `auth/claims.go`; `orderroutes`; Firebase/custom JWT already has org claim. |

#### G7 — No `claim.file` permission
| | |
|--|--|
| **Problem** | Route is `RequireRole(RETAILER)` only — cashiers not blocked at claims layer. |
| **Solution** | Add `claim.file` to OWNER/ADMIN/MANAGER/STOCK_CLERK/(optional RECEIVER); 403 others. |
| **Features** | Permission matrix; clients hide CTA. |
| **Integrations** | `auth` retailerRolePerms; three retailer clients. |

#### G8 — Claim → QUARANTINE bridge missing **(P0)**
| | |
|--|--|
| **Problem** | `BinQuarantine` exists; claim file never moves OnHand — sellable stock stays after concealed claim. |
| **Solution** | Same Spanner txn: `CLAIM_HOLD` → QUARANTINE if receive confirmed; approve/reject dispositions. |
| **Features** | Outbox `STORE_STOCK_CLAIM_HOLD`; clamp if on_hand &lt; qty. |
| **Integrations** | `claims` ↔ `retailer` bootstrap bridge; Spanner balances/movements; retailer WS. |

#### G9 — Receive ignores driver exception qty **(P0)**
| | |
|--|--|
| **Problem** | `loadOrderLinesForReceive` sets `AcceptedQty = ordered` — can putaway withdrawn units. |
| **Solution** | `accepted = ordered − exception − prior_claim`; block over-confirm. |
| **Features** | Receive banner + deep link to exception/claim. |
| **Integrations** | `store_stock.go` + `ClaimedQtyBySKU` + order exceptions; dock UX. |

#### G10 — Claim window global env only
| | |
|--|--|
| **Problem** | `claimWindowFromEnv()` only; no snapshot/policy. |
| **Solution** | §4 policies + `ClaimWindowEndsAt` on COMPLETED. |
| **Features** | Resolve + extend audit. |
| **Integrations** | Spanner; order COMPLETED path; ConfigMap fallback. |

#### G11 — Idempotency not enforced on file claim
| | |
|--|--|
| **Problem** | Smokecheck sends `Idempotency-Key`; handler ignores → double-tap races. |
| **Solution** | Redis idempotency guard (same as store_stock). |
| **Features** | Replay returns same `claim_id`. |
| **Integrations** | Redis on GKE; CI double-submit assert; clients send key (G25). |

#### G12 — Reverse logistics open best-effort outside txn
| | |
|--|--|
| **Problem** | `openReverseLogistics` warn-and-continue — claim OPEN with no dock ticket. |
| **Solution** | Outbox → worker retry; alert on persistent fail; smokecheck asserts inbound row. |
| **Features** | Metric `claim_reverse_open_fail_total`; reconcile job. |
| **Integrations** | `returns/tickets.go`; Kafka; Cloud Monitoring; e2e. |

#### G13 — Credit note from claim is draft stub / flag-off
| | |
|--|--|
| **Problem** | `CreateFromClaim` DRAFT, no lines; `CREDIT_NOTE_AUTO_FROM_CLAIM` off / absent ConfigMap. |
| **Solution** | Lines from claim SKUs; issue path; enable flag after smoke. |
| **Features** | Line-level CN linked to `claim_id`. |
| **Integrations** | Spanner CN + fiscal snapshots; ConfigMap; WH reverse on Issue. |

#### G14 — GP refund soft-fail; fiscal after COMPLETED
| | |
|--|--|
| **Problem** | Gateway refund failures fall to ledger-only without alert; Soliq reopen is non-goal. |
| **Solution** | Document money policy; alert GP soft-fail; CN for B2B credit; never reopen OFD. |
| **Features** | Settlement modes already exist; ops runbook. |
| **Integrations** | Global Pay + GSM secrets; supplier chargebacks UI; fiscal untouched. |

#### G15 — Multi-location claim scope undefined
| | |
|--|--|
| **Problem** | Claims lack `location_id`; quarantine may hit wrong store. |
| **Solution** | Bind to order delivery / receive location; hold stock only there. |
| **Features** | Location on eligibility + claim DTO. |
| **Integrations** | `RetailerLocations`; receive sessions; optional claims column. |

---

### 16.3 Cloud / GCP

#### G16 — Evidence upload placeholders when GCS/signBlob fails
| | |
|--|--|
| **Problem** | `storage/gcs.go` can return `placehold.co`; claims accept any URI → fake “photo”. |
| **Solution** | Fail closed when `REQUIRE_INFRA_ADAPTERS`; WI SA needs `roles/iam.serviceAccountTokenCreator` (signBlob); reject non-GCS hosts on file claim. |
| **Features** | Real signed PUT; prod rejects placeholders. |
| **Integrations** | GCS bucket (`GCS_BUCKET_NAME`); GKE Workload Identity; Terraform IAM; `platform/media_upload.go`. |

#### G17 — Media MIME / size / virus controls incomplete
| | |
|--|--|
| **Problem** | Ticket allowlists image ext; no max bytes; no magic-byte sniff; client mime unchecked. |
| **Solution** | `max_bytes` on ticket (e.g. 8MB); Content-Type match; optional Cloud Run scanner later. |
| **Features** | Client size errors; server reject. |
| **Integrations** | GCS prefix `evidence/claims/`; ConfigMap `CLAIM_EVIDENCE_MAX_BYTES`; IAM objectAdmin scoped. |

#### G18 — Observability gaps
| | |
|--|--|
| **Problem** | Window expiry = HTTP only; reverse fail = slog.Warn; no claim panels in Terraform alerts. |
| **Solution** | Counters + GCM alert policies + dashboard “Claims ops”. |
| **Features** | `void_claims_filed_total`, `window_expired_total`, `reverse_open_fail_total`. |
| **Integrations** | backend-go metrics; `infra/terraform/observability*.tf`; Cloud Logging. |

---

### 16.4 DevOps / CI

#### G19 — ConfigMap missing claim/credit flags
| | |
|--|--|
| **Problem** | `infra/k8s/backend-go/configmap.yaml` omits `CLAIM_WINDOW_HOURS`, `CLAIM_AUTO_APPROVE_MAX_MINOR`, `CREDIT_NOTE_AUTO_FROM_CLAIM` → silent code defaults. |
| **Solution** | Explicit ConfigMap keys + cutover docs. |
| **Features** | Ops-tunable without rebuild. |
| **Integrations** | Kustomize/overlays SSMR+staging; PRODUCTION_CUTOVER.md; flags not GSM. |

#### G20 — SSMR e2e skips concealed + store stock
| | |
|--|--|
| **Problem** | `e2e_claims.go` files `MISSING` (no photo), no receive, no QUARANTINE, no inbound assert; media warn-only. |
| **Solution** | Spine: COMPLETED → receive → CONCEALED+photo → quarantine → approve → WH `ClaimId` row. |
| **Features** | Markers `PX_E2E_CLAIMS_CONCEALED_OK`, `STORE_STOCK_CLAIM_HOLD_OK`, `CLAIMS_REVERSE_OK`. |
| **Integrations** | ssmr-smokecheck + marker gate; real GCS (fail if placeholder). |

#### G21 — CI does not gate GCS IAM / placeholder media
| | |
|--|--|
| **Problem** | `PX_E2E_CLAIMS_MEDIA_TICKET_WARN` allows ship without signBlob. |
| **Solution** | Production/SSMR smoke requires GCS signed URL host. |
| **Features** | Hard `PX_E2E_CLAIMS_MEDIA_TICKET_OK`. |
| **Integrations** | GKE WI + TokenCreator; CI env `PEGASUSX_ENV`. |

---

### 16.5 Cross-role

#### G22 — CLAIM_FILED fanout misses warehouse + role targeting
| | |
|--|--|
| **Problem** | File payload omits `warehouse_id`; WH room empty for CLAIM_FILED; retailer FCM is whole-org not stock roles. |
| **Solution** | LogisticsException-shaped payload with WH id; FCM filter `stock.view` / `dock.receive` / `claim.file`. |
| **Features** | WH reverse ping; stock “hold applied” push. |
| **Integrations** | Kafka; `notification_dispatcher.go`; Firebase FCM; inbox formatters. |

#### G23 — Visible vs concealed double-liability UX
| | |
|--|--|
| **Problem** | Qty caps exist but receive/claim UI don’t show “already handled at delivery”. |
| **Solution** | Eligibility + receive draft surface prior exception/claim ids (Rule A). |
| **Features** | Banner + deep link. |
| **Integrations** | claims list; receive UI; `bootstrap/claims_bridge.go`. |

#### G24 — Approve/reject stock lifecycle unwired
| | |
|--|--|
| **Problem** | Even with quarantine, approve/reject/WH-can’t-receive dispositions undefined in code. |
| **Solution** | Approve → RETURN/−QUARANTINE; Reject → restore; WH cancel → WASTE + credit. |
| **Features** | Disposition codes; movements `ref_type=CLAIM`. |
| **Integrations** | claims + store_stock + returns status; 3-role notifs. |

#### G25 — Clients omit Idempotency-Key on file claim
| | |
|--|--|
| **Problem** | Desktop/Android sheets often omit key while stock paths use it. |
| **Solution** | `Idempotency-Key: claim-file:{orderId}:{bodyHash}` + disable double submit. |
| **Features** | Safe retry on flaky network. |
| **Integrations** | FileClaimSheet/Panel/View + Redis guard G11. |

---

### 16.6 Priority cut

| Priority | IDs | Why |
|----------|-----|-----|
| **P0** | G6, G8, G9 | TEAM broken; sellable stock wrong; putaway of excepted qty |
| **P1** | G10, G11, G12, G16, G20, G22 | Window policy, dupes, silent reverse miss, fake photos, e2e blind, WH notifs |
| **P2** | G1–G5, G7, G13–G15, G17–G19, G21, G23–G25 | UX, CN/money polish, media harden, ConfigMap/CI, cross-role polish |

---

## 17. Ecosystem integration map

```
                    ┌─────────────────────────────────────────┐
                    │           Retailer clients               │
                    │  Dock receive · Stock · Orders · FCM     │
                    └───────────────┬─────────────────────────┘
                                    │ HTTPS / WS
                    ┌───────────────▼─────────────────────────┐
                    │  backend-go (GKE)                        │
                    │  order · claims · retailer/store_stock   │
                    │  returns · creditnote · payment · media  │
                    └─┬──────┬──────┬──────┬──────┬───────────┘
                      │      │      │      │      │
              Spanner │ Redis│ Kafka│ GCS  │ FCM  │
              (claims,│ idem │CLAIM_│evid. │push  │
               stock, │      │events│PUT   │      │
               returns│      │      │      │      │
               policy)│      │      │      │      │
                      │      │      │      │      │
         ┌────────────▼┐  ┌──▼──┐ ┌─▼──┐ ┌▼────┐ ┌▼──────────┐
         │ Supplier       │  │ Warehouse │  │ Driver │  │ Global Pay │  │ Firebase   │
         │ claims + policy│  │ inbound   │  │ OS&D   │  │ refund/GSM │  │ Auth / FCM │
         └────────────────┘  └───────────┘  └────────┘  └────────────┘  └────────────┘

DevOps: ConfigMap flags · ESO (GP/Firebase SA) · WI signBlob IAM ·
        Spanner migrations · ssmr-smokecheck markers · Terraform alerts
```

| Concern | Codebase | GCP | DevOps |
|---------|----------|-----|--------|
| Liability SoT | `claims/` | Spanner | migrations + e2e |
| Store mirror | `retailer/store_stock.go` | Spanner | RS3 txn tests |
| Reverse dock | `returns/`, `creditnote/` | Spanner | WH e2e inbound |
| Evidence | `platform/media_upload.go`, `storage/gcs.go` | GCS + WI TokenCreator | fail-closed smoke |
| Windows | new policy APIs | Spanner | ConfigMap fallback |
| Money | `payment/` chargeback, CN | GSM GP secrets | soft-fail alerts |
| Realtime | `kafka/notification_dispatcher.go` | Kafka + FCM | topic ACLs / consumer health |
| AuthZ | `auth` org + perms | Firebase tokens | JWT claim contract tests |

---

## Appendix C — Claim window feature matrix (target)

| Capability | Today | After RS1 |
|------------|-------|-----------|
| Enforce late claim blocked | Env 48h | Snapshot from policy |
| Supplier sets 8h/24h/custom | ❌ | ✅ portal + Android + iOS |
| Warehouse sets reverse SLA | ❌ | ✅ portal + Android + iOS |
| Warehouse lengthens retailer window | ❌ | ✅ opt-in flag |
| Warehouse shortens below supplier | ❌ | ❌ still forbidden v1 |
| Retailer sees deadline | Weak / server-only | ✅ eligibility + UI countdown |
| Exception-map `window_hours` | Unrelated lookback | Unchanged (not this feature) |

---

## Appendix D — Gap ID → phase mapping

| Gap | Phase |
|-----|-------|
| G6, G7 | RS0 |
| G3, G10, G2 | RS1 |
| G9, G23 | RS2 |
| G8, G1, G24 | RS3 |
| G13, G14, G15, G4, G5 | RS4 |
| G11, G12, G16–G22, G25 | RS5 |

---

*End of plan.*
