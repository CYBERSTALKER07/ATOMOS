# Wave C — Enterprise-scale design (L8–L11) for prod readiness

> **HISTORICAL / FROZEN — do not plan from this file alone.**
> Current SoT: [`PROD_READINESS_SEQUENCE.md`](./PROD_READINESS_SEQUENCE.md) · [`session-2026-08-07/ECOSYSTEM_GAP_REGISTER_2026-08-12.md`](./session-2026-08-07/ECOSYSTEM_GAP_REGISTER_2026-08-12.md) · [`FEATURES_BY_APP_ROLE.md`](./FEATURES_BY_APP_ROLE.md).
> Body is a point-in-time snapshot; re-verify any claim against code before acting.


**Date:** 2026-08-04  
**Status:** Product answers **LOCKED** (design review 2026-08-04)  
**Repo:** pegasusX  
**Naming:** Wave C = Next-Layer Phase D / L8–L11 (multi-org, HQ analytics, offline count + parked carts, CUSTOMER_ASSIST + planogram path).  
**Executable plan:** `docs/WAVE_C_IMPLEMENTATION_PLAN.md`

**Out of scope:** Wave A/B rework · B2 negotiations (disabled) · Soliq OFD · PG3 vision GPU.

---

## 0. Executive summary

| Epic | Name | Outcome |
|------|------|---------|
| **C1 / L8** | Multi-org staff phones | One person → many `RetailerOrgId`s; org picker + switch |
| **C2 / L9** | Franchise HQ analytics | Multi-location rollups; WH/factory never see POS internals |
| **C3 / L10** | Offline count + parked carts | Park/resume; offline count conflicts; **no** offline card |
| **C4 / L11** | Assist + planogram path | Assist SLA hardened; structure before vision |

**Prod-readiness bar:** DDL · JWT/RBAC · outbox+notif · three-client parity · emulator tests · SSMR markers · pack gates · honest empty · flags · rollback.

**Pilot posture:** Multi-org **frozen for general pilot roster**. Ship C1 behind `MULTI_ORG_LOGIN_ENABLED` (default off); enable only for 2–3 controlled retailers after picker + switch e2e markers are green on SSMR.

---

## 1. Preconditions

Retail OS packs 0–6 wired · JWT v2 single-org · POS online + offline cash · locations · assist CRUD · flywheel/local SKUs/claims · negotiations off · SSMR e2e green · **C1.1 membership DDL live on SSMR**.

---

## 2. Principles

1. Tenant = `RetailerOrgId` (no merged ledgers).  
2. Person ≠ org (global `RetailerUserId` + `RetailerUserMemberships`).  
3. Honest empty > fake HQ.  
4. POS retailer-owned (BI aggregates only externally).  
5. Offline: no card; session open online-only.  
6. Pack progressive disclosure.  
7. Money int64 minor units.  
8. Mutations: Idempotency-Key + Spanner RW + outbox.  
9. Additive DDL + dual-read + flags default off.  
10. Parked holds **never** touch `OnHand`.  
11. HQ daily writers **same Spanner RW txn** as sale/void (no eventual consistency).

---

## 3. Baseline gaps

| Area | Today |
|------|--------|
| Multi-org | Memberships table + dual-write (C1.1); login still single-org first match |
| HQ | Org reports only; no `hq/*` |
| Parked cart | No server hold entity |
| Offline count | No version conflict protocol |
| Assist | CRUD yes; no durable SLA worker |
| Vision | Design-only (`PLANOGRAM_VISION_PLAN.md`) |

---

## 4. Architecture

```
Login → memberships → intermediate token OR full JWT (one active org)
         │
         ├─ POS / Stock / Shifts (org + location)
         ├─ HQ APIs (OWNER/ADMIN multi-loc rollups)
         └─ Assist / Sections
                │
         Spanner ledgers + RetailerHqDaily* (+ optional BI Kafka aggregates later)
```

---

## 5. C1 / L8 — Multi-org

**Model:** Global `RetailerUserId` + `RetailerUserMemberships`.  
**DDL:** `(UserId, RetailerId, Role, IsActive, LocationIdsJson)` — **empty `LocationIdsJson` / null = all locations of that org.**  
**Backfill** from `RetailerUsers`; dual-read login; dual-write on create/update.

**Auth (C1.2):**
- Multi-membership → intermediate JWT `token_use=PendingOrgSelect`, **TTL 5–10 minutes** (default **7m**).  
- `POST /v1/retailer/auth/select-org` → full JWT (one active org).  
- `POST /v1/retailer/auth/switch-org` → re-issue full JWT.  
- Middleware rejects intermediate tokens on all business routes.  
- Single-membership (or flag off) = today’s full JWT path unchanged.

**Client hard contract on switch-org / select-org:**
1. Clear cart / hold draft  
2. Clear open POS session  
3. Clear local offline count drafts  
4. Clear in-memory assist context  

Not “clear cart only.”

**Markers:** `PX_E2E_MULTI_ORG_PICKER_OK`, `PX_E2E_ORG_SWITCH_OK`, plus **CORE single-org login regression**.

**Atomic release:** C1.1 + C1.2 as one release candidate; flag + previous JWT mint path live for rollback.

---

## 6. C2 / L9 — HQ analytics

**Source of truth: Spanner first** (`RetailerHqSalesDaily`, `RetailerHqStockSnapshot`). Kafka BI is a later consumer of the same writers — not required for Wave C exit.

**Writers (non-negotiable):** `RetailerHqSalesDaily` upsert lives **inside the same Spanner RW transaction** that records the POS sale/void. Same for stock snapshot movements where applicable. Include `local:` SKUs in HQ sales; exclude from supplier demand exports.

**APIs:** `/v1/retailer/hq/summary|sales-by-location|sales-by-sku|shrinkage|export`.  
**Auth:** OWNER/ADMIN + reports.view + REPORTS_PRO.

**UI:** Desktop `/hq`; mobile OWNER digest (later PR).

**Marker:** `PX_E2E_HQ_SALES_BY_LOCATION_OK` (sum locations = org total).

---

## 7. C3 / L10 — Parked carts + offline count

**Law:** no offline card; no offline session open.

**Holds:**
- DDL: `RetailerPosHolds` HELD|RESUMED|EXPIRED|VOIDED  
- TTL = **24 h**  
- Resume = **same LocationId only** (cross-register within location yes; cross-store no)  
- Never decrement `OnHand`  
- APIs: park / list / resume / void; 24h sweeper

**Count:**
- Local draft + `POST …/stock/counts/commit` with `base_version`  
- **409** returns **server version + server counted quantities** (draft-vs-current diff), not a bare conflict string  
- Force write = **MANAGER or OWNER only**, audited

**Markers:** `PX_E2E_POS_HOLD_RESUME_OK`, `PX_E2E_OFFLINE_COUNT_CONFLICT_OK`.

---

## 8. C4 / L11 — Assist + planogram

**Assist (required for Wave C exit):**
- Default SLA = **15 minutes**  
- Channel order: **in-app push + WS room first**; SMS only if configured  
- Worker, on-duty routing, metrics, idempotency  
- Marker: `PX_E2E_ASSIST_SLA_OK`

**Planogram:** PG1 structure **optional pilot-only**, not required for Wave C exit. PG2 later. **PG3 vision out.**

---

## 9. Flags (prod / pilot default off)

| Flag | Default | Purpose |
|------|---------|---------|
| `MULTI_ORG_LOGIN_ENABLED` | **off** | Multi-membership picker path |
| `HQ_ANALYTICS_ENABLED` | off | HQ APIs + writers gate |
| `POS_HOLDS_ENABLED` | pilot default on | Parked carts |
| `OFFLINE_COUNT_ENABLED` | off | Count commit + 409 protocol |
| `ASSIST_SLA_ENABLED` | off | SLA worker |

Allow-list for multi-org: optional `MULTI_ORG_RETAILER_ALLOWLIST` (comma org ids) for 2–3 controlled retailers.

---

## 10. Sequence (refined — design review)

```
0. Product freeze on the 7 answers (§12)     ← DONE
1. C1.1  Membership DDL + backfill + dual-read (flag off)  ← DONE (DDL on SSMR)
2. C1.2  Auth: intermediate token → select-org → full JWT + switch-org
3. C1.3  Clients: picker + switcher + clear-on-switch contract
4. C3.1  RetailerPosHolds DDL + park/resume/void APIs (no stock touch)
5. C3.2  Hold UI + 24 h sweeper
6. C2.1  HQ writers (same txn as sale/void) + daily tables
7. C2.2  HQ REST + desktop /hq
8. C3.3  Offline count with base_version + 409 protocol
9. C4    Assist SLA worker (15 min default)
10. Marker gate + CORE single-org regression
```

C3 parked carts before full HQ UI: simpler, immediate operational value.

---

## 11. Exit checklist

- [ ] Unit + emulator tests multi-org / holds / HQ / count 409  
- [ ] SSMR markers + gate green  
- [ ] CORE single-org login + POS regression green  
- [ ] DDL windows SSMR → staging → prod  
- [ ] Flags safe; runbooks; export audit  
- [ ] Intermediate token TTL enforced; business routes reject `PendingOrgSelect`  
- [ ] Switch-org client contract documented + tested  

---

## 12. Product answers — LOCKED

| # | Question | Decision | Rationale |
|---|----------|----------|-----------|
| 1 | Multi-org in pilot now or freeze? | **Freeze for pilot.** Ship C1 behind flag; enable only for 2–3 controlled retailers. | Login rewrite is highest risk; wait for picker/switch markers green on SSMR. |
| 2 | HQ: Spanner first vs external BI? | **Spanner first** (`RetailerHqSalesDaily` + `RetailerHqStockSnapshot`). | Matches projection patterns; honest empty easier; Kafka BI later. |
| 3 | Hold TTL + cross-register resume? | **TTL 24 h. Cross-register = yes, same LocationId only.** | Prevents Store A resuming Store B cart. |
| 4 | Count force = MANAGER only? | **MANAGER or OWNER only**, audited. | Staff force breaks version-etag purpose. |
| 5 | Assist SLA + channel? | **15 min.** Push + WS first; SMS if configured. | Useful and achievable inside outbox fabric. |
| 6 | PG1 in Wave C? | **Optional pilot-only; not Wave C exit.** | Do not block C1–C3; PG3 out. |
| 7 | Global UserId vs per-org rows? | **Global `RetailerUserId` + memberships.** | Only model that scales across franchisees. |

---

## 13. Risks

| Risk | Weight | Mitigation |
|------|--------|------------|
| **Login path rewrite** | **Highest** | Dual-read; flag off by default; single-membership = legacy path; C1.1+C1.2 atomic RC; rollback = flag off + previous mint path live; CORE single-org e2e gate |
| Intermediate token leak | High | **TTL 5–10 min (default 7m)**; reject on business routes |
| HQ double-count | High | Same txn as sale/void; property tests |
| Count split-brain | Med | Version etag; force MANAGER/OWNER audited; 409 returns server quantities |
| Hold ≠ stock | Med | Holds never decrement OnHand |
| Vision creep | Low | PG3 out; PG1 optional |

---

## 14. Success metrics

| Epic | Metric |
|------|--------|
| C1 | Two-org picker < 30s; switch clears cart + POS session + offline drafts + assist context; single-org login unchanged with flag off |
| C2 | HQ by location = sum of stores |
| C3 | Park/resume same-location + 24h expire; count 409 shows draft-vs-server |
| C4 | Assist SLA notif within 15 min + complete path |

---

## 15. Tightening checklist (before first auth PR)

| # | Rule | Status |
|---|------|--------|
| T1 | Intermediate token TTL 5–10 min (default 7m) | Spec locked; implement in C1.2 |
| T2 | Switch-org clears cart + POS session + offline count drafts + assist context | Spec locked; C1.3 contract |
| T3 | HQ writers same Spanner RW txn as sale/void | Spec locked; C2.1 code rule |
| T4 | Count 409 returns server version + server quantities | Spec locked; C3.3 |
| T5 | Empty membership `LocationIds` = all org locations | Spec locked; C1.2+ RBAC |

---

## 16. Implementation log

### C1.1 — Membership foundation (DONE)

| Item | Path / note |
|------|-------------|
| DDL migration | `apps/backend-go/schema/migrations/20260810_retailer_user_memberships.ddl` |
| Canonical schema | `apps/backend-go/schema/spanner.ddl` → `RetailerUserMemberships` + indexes |
| Repo | `apps/backend-go/retailer/memberships.go` |
| Dual-write | Owner ensure, staff create/update; memory + Spanner |
| Dual-read | Memberships first; synthesize from `RetailerUsers` if none |
| Tests | `memberships_test.go` (memory) PASS |
| SSMR | Table + indexes **applied** on `pegasusx-ssmr-db` |

### C1.2 — Multi-org auth (DONE 2026-08-04)

| Item | Path / note |
|------|-------------|
| JWT | `token_use=PendingOrgSelect`, `phone_number`; TTL 5–10m (default 7m via `PENDING_ORG_SELECT_TTL_SEC`) |
| Flag | `MULTI_ORG_LOGIN_ENABLED` default off; optional `MULTI_ORG_RETAILER_ALLOWLIST` |
| Login | Dual-read memberships by phone; multi → intermediate; single/flag off → full JWT (legacy) |
| APIs | `GET /v1/auth/retailer/memberships`, `POST …/select-org`, `POST …/switch-org` |
| Middleware | `RequireRole` rejects intermediate with `ORG_SELECT_REQUIRED` |
| Refresh | Rejects intermediate tokens |
| Tests | `auth_multi_org_test.go` + existing login tests PASS |
| Not in C1.2 | Client picker UI (**C1.3**); e2e markers (**C1.4**) |

### C1.3 — Clients: picker + switcher + clear-on-switch (DONE 2026-08-04)

| Surface | Paths |
|---------|--------|
| Shared types | `packages/types` — `RetailerLoginResponse`, `RetailerMembershipDTO` |
| Desktop contract | `lib/clear-org-scoped-state.ts` + event `pegasusx:org-switched` |
| Desktop multi-org | `lib/multi-org-auth.ts` — select/switch/list + pending cookie |
| Desktop picker | `app/auth/select-org/page.tsx` |
| Desktop switcher | `components/OrgSwitcher.tsx` in `RetailerShell` |
| Desktop login | branches on `pending_org_select` |
| Desktop cart | listens for org-switched and clears |
| iOS | `AuthResponse` multi-org, `AuthManager` select/switch/clear, `SelectOrgView` |
| Android | `AuthResponse` multi-org, API endpoints, `AuthViewModel` + `AuthScreen` picker |
| Tests | desktop `clear-org-scoped-state.test.ts` PASS |

**Clear-on-switch contract:** cart · POS parked/session · offline count drafts · assist context.

### C1.4 — Multi-org e2e markers + gate (DONE 2026-08-04)

| Item | Detail |
|------|--------|
| E2E | `cmd/ssmr-smokecheck/e2e_multi_org.go` wired into `runE2ECheck` |
| Markers always | `PX_E2E_SINGLE_ORG_LOGIN_UNCHANGED_OK`, `PX_E2E_PENDING_ORG_REJECT_OK`, `PX_E2E_MULTI_ORG_AUTH_OK` |
| Flag on | `PX_E2E_MULTI_ORG_PICKER_OK`, `PX_E2E_ORG_SWITCH_OK` |
| Flag off | `*_SKIPPED` alternatives |
| Gate | `contracts/ssmr_ecosystem_markers.json` required + alternatives |
| Notif fanout | **Deferred** — opt-in quiet hours across memberships is L8.4 / later; C1.4 does not change notif routing |

### C3.1 — Parked POS holds API (DONE 2026-08-11)

| Item | Path / note |
|------|-------------|
| DDL | `schema/migrations/20260811_retailer_pos_holds.ddl` + `spanner.ddl` |
| Service | `retailer/pos_holds.go` — park/list/resume/void |
| Invariants | **No OnHand/stock writes**; resume **same LocationId only**; TTL **24h** |
| Flag | `POS_HOLDS_ENABLED` pilot default on; set `false` → 404 `POS_HOLDS_DISABLED` |
| Routes | `GET/POST /v1/retailer/pos/holds`, `…/{holdID}/resume`, `…/void` |
| Tests | `pos_holds_test.go` PASS |
| SSMR | DDL applied when gcloud available |

### C3.2 — Hold UI + 24h sweeper (DONE 2026-08-11)

| Item | Path / note |
|------|-------------|
| Sweeper | `SweepExpiredPosHolds` + `RunPosHoldsSweeper` (15m ticker); wired in `runtime_workers.go` |
| Desktop UI | POS page: park / list / resume / void; auto-hides on 404 when flag off |
| Types | `@pegasusx/types` PosHold DTOs |
| Tests | sweeper memory + disabled noop PASS |

### C2.1 — HQ writers (DONE 2026-08-12)

| Item | Path / note |
|------|-------------|
| DDL | `20260812_retailer_hq_analytics.ddl` — `RetailerHqSalesDaily` + `RetailerHqStockSnapshot` |
| Writers | `hq_analytics.go` + `savePosSaleWithHQ` — **sale ledger + HQ daily in one Apply** |
| Sale | QtySold/Gross/Net +; void QtyVoided + Net −; Gross retained |
| local: | Included in HQ sales |
| Stock snap | Best-effort after stock move (`refreshHqStockSnapshotsForSale`) |
| Tests | sale+void+local + multi-location sum property PASS |

### C2.2 — HQ REST + desktop `/hq` (DONE 2026-08-12)

| Item | Path / note |
|------|-------------|
| APIs | `GET /v1/retailer/hq/summary\|sales-by-location\|sales-by-sku\|shrinkage\|export` |
| Auth | OWNER/ADMIN + reports.view; REPORTS_PRO auto-enable |
| Flag | `HQ_ANALYTICS_ENABLED` (default off → 404 honest) |
| UI | Desktop `/hq` + nav “Franchise HQ” |
| Property | `balanced`: sum(location net) = org net |
| Tests | disabled 404, cashier forbidden, by-location balanced + local SKU |

### C3.3 — Offline count version protocol (DONE 2026-08-13)

| Item | Path / note |
|------|-------------|
| DDL | `20260813_retailer_stock_count_version.ddl` — location versions + force audit |
| Version | Bumps on every `applyDelta` stock mutation |
| APIs | `GET …/stock/counts/version`, `POST …/stock/counts/commit` |
| 409 | `COUNT_VERSION_CONFLICT` + `server_version` + `server_lines` (on_hand) |
| Force | MANAGER / OWNER / ADMIN only + audit row |
| Flag | `OFFLINE_COUNT_ENABLED` default off → commit 404 |
| Legacy | `POST /stock/counts` unchanged |
| Tests | conflict, force deny/allow, matching version PASS |

### C4.1 — Assist SLA worker (DONE 2026-08-14)

| Item | Path / note |
|------|-------------|
| Worker | `SweepAssistSLA` + `RunAssistSLAWorker` (1m ticker) in `assist_sla.go` |
| Wire | `runtime_workers.go` (no-op when flag off) |
| SLA | Default **15 min** (`ASSIST_SLA_MINUTES` or pack `sla_minutes`) |
| Channel | In-app notif (push+WS fabric); SMS stub if `ASSIST_SLA_SMS=1` |
| Idempotent | `SlaBreachNotifiedAt` column + memory fallback |
| Event | `RETAILER_ASSIST_SLA_BREACHED` |
| Flag | `ASSIST_SLA_ENABLED` default off |
| DDL | `20260814_assist_sla_notified.ddl` applied on SSMR |
| Tests | disabled noop + notify once PASS |

**Wave C implementation track complete for C1–C4.1.** Remaining: image roll, pilot flags, optional e2e markers gate.
