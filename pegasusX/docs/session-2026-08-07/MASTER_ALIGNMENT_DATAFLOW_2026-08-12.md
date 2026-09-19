# PegasusX — Master Alignment: Docs ↔ Code ↔ Data Flow
> **POINT-IN-TIME SNAPSHOT (2026-08-07) — do not treat as current status.**
> Re-verify any claim against live code before acting. Multiple ecosystem hardening phases have shipped since this audit.


> **Prod goal SoT:** [`../PROD_ECOSYSTEM_GOAL.md`](../PROD_ECOSYSTEM_GOAL.md) — what “prod ready” means. This file is docs↔code↔data-flow truth + remaining Class A loops.

**Date:** 2026-08-12  
**Trees:** `pegasusX` = source of truth · `pegasus` = legacy/secondary (do not plan from it alone)  
**Companions:** [`../DOCS_SOURCE_OF_TRUTH.md`](../DOCS_SOURCE_OF_TRUTH.md) · [`../PROD_ECOSYSTEM_GOAL.md`](../PROD_ECOSYSTEM_GOAL.md) · [`ECOSYSTEM_GAP_REGISTER_2026-08-12.md`](./ECOSYSTEM_GAP_REGISTER_2026-08-12.md) · [`../DATA_FLOW_AS_IMPLEMENTED.md`](../DATA_FLOW_AS_IMPLEMENTED.md) · [`../FEATURES_BY_APP_ROLE.md`](../FEATURES_BY_APP_ROLE.md) · [`../PROD_READINESS_SEQUENCE.md`](../PROD_READINESS_SEQUENCE.md)

This document answers: *What is actually true in the codebase, where docs lie, what “perfect data flow” means here, and what remains so every role/platform feature is enterprise-wired end-to-end.*

---

## 1. Thesis (agreed with architecture + live re-verify)

You do **not** need a new data-flow paradigm.

The as-built kernel already is:

```
Client (role app)
  → JWT-gated HTTP (backend-go)
  → Spanner domain write + OutboxEvents (same transaction)
  → Relay (worker tier) → Kafka
  → Consumers (NotificationDispatcher, order mutator, warehouse mutator, returns, billing, …)
  → WS hubs (8) + inbox + FCM + partner webhooks
  → other roles’ clients react
```

**Why it felt unmaintainable:** not “wrong architecture,” but **uneven coverage** (now largely closed in Part 2 of the gap register):

1. Some domains emit outbox events; a few still dual-write or best-effort post-commit (AR/payout now emit).
2. Behavior changes by run mode (`api` vs `worker` vs `all`) — mitigated by worker heartbeat + api notif consumer (P1-9).
3. Cross-role product loops that stopped at “API exists” — W2 Class A loops closed; residuals are flag/cert/scale.

**Coverage rule (definition of done for any feature):**

> Every Spanner state mutation emits an in-transaction outbox event; every event has a declared consumer (WS and/or push and/or webhook and/or domain mutator); no cross-role loop ends at an API with no client on the platforms that role actually uses.

---

## 2. Docs ↔ code alignment (stale claims fixed)

| Source | Claim | Reality (2026-08-12 re-verify) |
|--------|-------|--------------------------------|
| Reality report §5 | Admin portal is redirect stub | **False now** — real Next console: Tenants / Flags / Audit (`apps/admin-portal`) |
| Reality report §5 | supplier-app-desktop redirect stub | **Path gone** — supplier desktop **is** `supplier-portal` (Next + Tauri 2) |
| Reality report §5 | DataMatrix “placeholder” only | **False now** — real ECC200 + **FNC1 GS1 element string + ZPL `^BX`** (P1-13 ✅ W5); square in-process encoder caps at 44×44 |
| Reality report §5 | SDK README-only | **False now** — partner Go SDK in `go.work` (P1-15 ✅ W5) |
| Reality report §2 | Optimizer prod replicas 0 | Prod still **replicas 0** until AR (P1-1 ✅ remap); **SSMR overlay patches replicas 1** |
| Reality report §4/§6 | AR/payout silent / bank-file only | **AR + payout now emit outbox** (P0-4/5 resolved). **Bank-file is permanent settlement** ([`PAYOUT_RAIL_DECISION.md`](../PAYOUT_RAIL_DECISION.md)); live rail deferred |
| Gap register P0-1…6 | Money/tenancy/routing P0s | **Resolved** in tree (re-verify AR/payout emits, admin portal real) |
| `FEATURES_BY_APP_ROLE.md` | admin-portal / HQ·Credit / warehouse WMS | **Synced 2026-08-12** — live admin console (+ match/partner); retailer HQ/Credit mobile; warehouse bins/pick-waves/cycle/cold-chain/labor |
| `OPTIMIZER_AND_ROUTING_RUNTIME.md` / `workers-kafka.md` | SSMR replicas 0 / not in overlay | **Synced 2026-08-12** — SSMR replicas 1 + in-overlay |
| `PARTNER_EDI.md` | datamatrix placeholder modules | **Synced W5** — FNC1 DataMatrix + EDI-lite breadth |
| Dual trees | “pegasus and pegasusX equal” | **False** — plan and ship from **pegasusX** |

---

## 3. Data-plane truth (re-verified on `pegasusX/apps/backend-go`)

| Piece | Status |
|-------|--------|
| Spanner SoT + outbox same-txn | Wired |
| Outbox relay → Kafka | Wired on **worker/all** tier |
| NotificationDispatcher → WS + inbox + FCM | Wired; api-only gets notif consumer when no worker heartbeat |
| Worker heartbeat | Wired |
| WS hubs | **8:** retailer, supplier, driver, payload, warehouse, factory, telemetry, platform-admin — browsers mint role `/ws-session` (`token_use=ws`) then `?token=`; natives use `Authorization: Bearer` |
| AR / payout outbox | **Emitting** (not silent) |
| Driver location bus | Throttled `DRIVER_LOCATION_UPDATED` → TopicRealtime outbox |
| Twin Kafka consumer | **Started** (W1) — group `void-digital-twin` |
| TopicWebhooks | **Retired** — do not emit; payment/partner use TopicMain/Orders |
| Search | Spanner LIKE — deferred engine ([`../SEARCH_DECISION.md`](../SEARCH_DECISION.md)) |
| Fiscal legal OFD | **Blocked on EDS / E-IMZO** (dev-HMAC not legal) |
| Live payout rail | **Decided:** bank-file permanent for prod bar; fail-closed if `live=true` without a real rail |

Order lifecycle core (create → reserve → dispatch → seal → depart → arrive → pay/fiscal → complete → claim) is the **strongest** end-to-end path. Finance, planning autonomy, partner cert, and client parity are where loops break.

---

## 4. Role × platform matrix (code sizes, pegasusX)

| Role | Android | iOS | Desktop | Web / portal | Notes |
|------|---------|-----|---------|--------------|-------|
| Retailer | ● ~34 screens | ● ~151 swift | ● Next+Tauri 2 (~32 pages) | same app | Best overall product depth |
| Supplier | ● ~62 screens | ● ~131 swift | ● via **supplier-portal** Tauri | ● ~73 pages | No separate `supplier-app-desktop` dir |
| Driver | ● ~15 screens | ● ~129 swift | ○ | ○ | Field-only by design; no ops portal |
| Warehouse | ● ~39 screens | ● ~96 swift | ● warehouse-portal Tauri | ● ~42 pages | WMS depth strong on portal |
| Factory | ● ~20 screens | ● ~67 swift | ● factory-portal Tauri | ● ~19 pages | Loading bay ↔ payload Class A (P1-18) |
| Payload | ● Android loading-bay | ● iOS loading-bay | ○ | ● Expo `payload-terminal` | Expo SoT; natives peer for factory+payloader (P2-25) |
| Platform admin | ○ | ○ | ○ | ● Next console | Tenants / flags / audit / match / partner |

**Desktop stack (recommendation at bottom):** all real desktops are **Next.js 15 + Tauri 2**. No Electron in tree.

---

## 5. Feature wiring classes (how to think about “every feature”)

Classify every feature into one of four classes. Only **Class A** is shippable enterprise product.

| Class | Meaning | Example |
|-------|---------|---------|
| **A — Wired E2E** | Spanner write + outbox + consumer + client(s) for involved roles | Order create → supplier WS → warehouse dispatch → driver cash → fiscal path |
| **B — Backend island** | API/schema exist; missing client or missing consumer | (was twin/cold-chain/labor — closed P2-23) |
| **C — UI island** | Screen exists; mock/local or incomplete backend hop | Thin admin billing analytics; warehouse Control Tower portal-primary (mobile absent) |
| **D — Flag / cert blocked** | Logic present; off in prod or needs external proof | Auto-order place, Soliq EDS live, optimizer prod replicas, live dunning provider keys |

**Major cross-role loops still Class B/C (not A):**

| Loop | Gap |
|------|-----|
| Auto-order place | Dual-control Evaluate + soak gate + `FLAG_AUTO_ORDER_PLACE` audit; soak bypass is money-flag + `FLAG_AUTO_ORDER_SOAK_GATE` / runtime `AUTO_ORDER_SOAK_GATE_BYPASS` (P2-7 ✅); place env remains false until flip evidence (P1-2…5 ✅) |

**W2 Class A loops closed 2026-08-12:** factory↔payload (P1-18), retailer AR/HQ mobile (P1-17), supplier planning web (P2-26), warehouse WMS screens (P2-24), admin match/partner/dunning (P1-16).

---

## 6. Remaining gap backlog (deduped; use gap register for evidence)

### P0 — already closed in code (do not re-open unless regression)

AR pay-down on cash collect · payout fail-closed without live rail · platform-admin tenant exempt · AR/payout outbox · supplier-portal API route depth · several tenancy/money gates.

### Still open — prioritize for “perfect data flow”

**P1 structural**

1. ~~Prod optimizer image remap~~ ✅ W4 (replicas 0 until AR publish)  
2. ~~Auto-order place dual-control + soak artifact + 80% alignment~~ ✅ W4  
3. ~~Forecast/accuracy CronJobs + prod soak flags (place off)~~ ✅ W4  
4. ~~Fiscal sign→submit→poll contract~~ ✅ W3 (live EDS still needs E-IMZO procurement)  
5. ~~JWT revocation/denylist~~ ✅ W0  
6. ~~Spanner.ddl vs migrations drift~~ ✅ 2026-08-12 (P1-12)  
7. ~~GS1 DataMatrix FNC1 conformance + label path~~ ✅ W5  
8. ~~AS2 MDN/MIC verify~~ ✅ W5  
9. ~~Partner Go SDK module path / go.work~~ ✅ W5  
10. ~~Admin UI depth for match queue + partner rails~~ ✅ W2  
11. ~~Retailer AR/HQ mobile parity~~ ✅ W2  
12. ~~Factory → payload client loop~~ ✅ W2 

**P2 enterprise**

~~Demand-sensing producers~~ ✅ P2-6 · ~~optimizer constraint e2e hard-fail~~ ✅ P2-1 · ~~optimizer cert harness + heuristic status~~ ✅ P2-2 · ~~MEIO capital-cap~~ ✅ P2-3 · ~~scenario workbench clone/compare/publish + real RaR~~ ✅ P2-4 · ~~S&OP capacity/demand split + production-lines~~ ✅ P2-5 · ~~multi-currency AR~~ ✅ P2-8 · ~~dev webhook secret defaults in NewService~~ ✅ P2-9 · ~~MFA~~ ✅ P2-17 · ~~CI gates for multi-tenancy~~ ✅ P2-18 · ~~SLO on relay/DLQ~~ ✅ · ~~EDIFACT breadth~~ ✅ W5 · ~~partner sandbox~~ ✅ W5 · ~~warehouse dedicated WMS screens~~ ✅ W2 · ~~payload native factory loading-bay~~ ✅ P2-25 · ~~supplier planning on web~~ ✅ W2 · ~~cold-chain/labor/twin clients~~ ✅ P2-23.

**Security initiative**

Detail IDORs + JWT revocation closed W0 2026-08-12 (gap register § Security).

---

## 7. Desktop technology decision

**Recommendation: keep Next.js + Tauri 2. Do not switch to Electron or “pure Go UI.”**

| Option | Verdict |
|--------|---------|
| **Next + Tauri 2 (current)** | **Stay.** Shared web+desktop UI, small native shell, updater/fs/sql/deep-link plugins already wired, matches retailer/supplier/warehouse/factory. |
| Electron | Larger binaries, higher memory, weaker sandbox story; only switch if Tauri blocks a hard OS integration you need (unlikely). |
| Pure Go (Wails/Fyne) | Would **duplicate** entire portal UI; kills shared React components. |
| Flutter desktop | Second UI stack; only if you abandon Next portals. |
| “Simple” web-only | Fine for admin break-glass; weak for offline POS / local SQL / auto-update — retailer already uses Tauri plugins for that. |

**Hardening Tauri (not rewriting):**

- One shared desktop kit (updater, auth storage, offline SQL, deep links) already partially in packages — finish standardization across portals.  
- Static export `out/` + Tauri hosting is correct for these apps.  
- Keep field roles (driver/payload) **native**, not Tauri Android (supplier-portal already deprecates that).

---

## 8. Operating model so complexity becomes manageable

1. **Single SoT tree:** `pegasusX` only for product.  
2. **Coverage checklist on every PR that mutates Spanner:** emit? consumer? which hubs? which clients? which roles?  
3. **Feature matrix flags** (void-guardian): `logic_verified` · `wire_connected` · platform cells.  
4. **No new Class C screens** without backend hop.  
5. **Run-mode parity tests:** api-only vs worker must not silently drop push/inbox.  
6. **Docs rule:** if code changes a P0/P1 gap, update gap register same PR; reality report is historical.  
7. **Close loops before adding roles/features:** factory↔payload, AR mobile, planning web, twin consumer decision.

---

## 9. Suggested execution order (business + technical)

| Wave | Goal | Why |
|------|------|-----|
| **W0** | Security IDOR sweep + JWT revocation design | Trust before scale |
| **W1** | Data-flow hygiene — **closed 2026-08-12** (twin started, TopicWebhooks retired, search decision, atomic DEMAND_SIGNAL) | Predictable bus |
| **W2** | Cross-role Class A — **closed 2026-08-12** (factory↔payload, retailer AR/HQ mobile, supplier planning web, warehouse WMS, admin match/partner/dunning) | Product loops close |
| **W3** | Money legality — **closed 2026-08-12** (EDS sign→submit→poll contract; bank-file payout decision; Global Pay RF simulator proof) | Revenue/law |
| **W4** | Autonomy — **closed 2026-08-12** (optimizer image remap fixed; soak artifact + flip-check aligned; CronJobs on overlays; shadow flags on; dual-control Evaluate + `FLAG_AUTO_ORDER_PLACE` audit) | Field-agent reduction |
| **W5** | Enterprise partner: GS1 FNC1+label, AS2 MDN verify, SDK go.work, EDI-lite breadth, sandbox keys — **closed 2026-08-12** | Big-supplier adoption (Drummond/cert EDIFACT still open) |

---

## 10. Bottom line

- Architecture for “Spanner → outbox → Kafka → WS/search/realtime” is **already correct**.  
- Gaps are **coverage, certification, client parity, and mode-dependent behavior** — not a missing bus.  
- Docs that still say “admin stub / silent AR / no desktop supplier” are **wrong** as of 2026-08-12; use this file + gap register.  
- Desktop: **keep Tauri 2 + Next**; invest in kit quality, not a framework migration.  
- Perfect maintainability comes from enforcing the **coverage rule** on every mutation, not from rewriting the stack.

*Generated from live tree re-verify 2026-08-12; supersedes contradictory gap narrative in END_PRODUCT_REALITY_REPORT_2026-08-11 for planning purposes.*
