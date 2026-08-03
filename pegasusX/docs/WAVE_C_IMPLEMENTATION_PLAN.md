# Wave C — Agent-executable implementation plan

**Status:** Product answers locked · C1.1 done · ready for C1.2  
**Companion design:** `docs/WAVE_C_ENTERPRISE_SCALE_PLAN.md`  
**Repo root:** `pegasusX/`  
**Env for prove:** SSMR (`pegasus-503013`, `api-ssmr.pegasusx.app`, ns `pegasusx-ssmr`)

This document is the runbook for agents and humans. Each PR has: goal, files, DDL/API/JWT contracts, tests, markers, flag, rollback.

---

## 0. Locked product decisions (do not re-litigate)

| # | Decision |
|---|----------|
| 1 | Multi-org **frozen** for general pilot; flag off; allowlist 2–3 orgs only when enabling |
| 2 | HQ **Spanner first** |
| 3 | Hold TTL **24h**; resume **same LocationId** only |
| 4 | Count force **MANAGER or OWNER**, audited |
| 5 | Assist SLA **15 min**; push + WS first |
| 6 | PG1 optional; **PG3 out** |
| 7 | **Global UserId + memberships** |

---

## 1. Flags & config

| Env / flag | Default | Used by |
|------------|---------|---------|
| `MULTI_ORG_LOGIN_ENABLED` | `false` | C1.2 login branch |
| `MULTI_ORG_RETAILER_ALLOWLIST` | empty | Optional comma `RetailerId`s |
| `PENDING_ORG_SELECT_TTL_SEC` | `420` (7m) | Intermediate JWT |
| `HQ_ANALYTICS_ENABLED` | `false` | C2 |
| `POS_HOLDS_ENABLED` | `false` | C3 holds |
| `OFFLINE_COUNT_ENABLED` | `false` | C3 count |
| `ASSIST_SLA_ENABLED` | `false` | C4 |
| `ASSIST_SLA_MINUTES` | `15` | C4 |

---

## 2. PR sequence (titles + order)

| Order | PR title | Depends | Risk |
|------:|----------|---------|------|
| ✅ | **C1.1** Membership DDL + dual-write/read + backfill | — | Low (done) |
| ✅ | **C1.2** Multi-org auth: PendingOrgSelect → select-org → switch-org | C1.1 | **Highest** (done; flag off) |
| ✅ | **C1.3** Clients: org picker + switcher + clear-on-switch contract | C1.2 | Med (done) |
| ✅ | **C1.4** Multi-org e2e markers + SSMR gate (notif fanout deferred) | C1.2–1.3 | Med (done) |
| ✅ | **C3.1** RetailerPosHolds DDL + park/list/resume/void APIs | — | Med (done) |
| ✅ | **C3.2** Hold UI + 24h sweeper | C3.1 | Low (done) |
| ✅ | **C2.1** HQ daily tables + same-txn writers on sale/void | — | High (done) |
| ✅ | **C2.2** HQ REST + desktop `/hq` | C2.1 | Med (done) |
| 8 | **C3.3** Offline count `base_version` + 409 diff + force | C3.1 optional | High (stock) |
| 9 | **C4.1** Assist SLA worker 15m + push/WS | assist CRUD exists | Med |
| 10 | **C0.gate** Marker gate + CORE single-org regression | all above flags off path | Gate |

**Atomic RC:** Ship **C1.1 + C1.2** together. Do not enable `MULTI_ORG_LOGIN_ENABLED` until C1.3+C1.4 markers green.

---

## 3. C1.1 — DONE (reference)

| Artifact | Path |
|----------|------|
| DDL | `apps/backend-go/schema/migrations/20260810_retailer_user_memberships.ddl` |
| Repo | `apps/backend-go/retailer/memberships.go` |
| Dual-write | `org_users.go`, `org_members.go` |
| Tests | `memberships_test.go` |
| SSMR | `RetailerUserMemberships` applied |

**Semantics:** `LocationIdsJson` null/empty/`[]` ⇒ **all locations** of that org.

**Ops remaining (optional):**
```bash
# After backend image with BackfillMembershipsFromUsers is live:
# invoke admin/ops one-shot or temporary CLI
```

---

## 4. PR C1.2 — Auth (next)

### Goal
Flag-off path = **identical** single-org login. Flag-on + multi membership → intermediate token → select-org → full JWT. switch-org re-issues full JWT.

### JWT claim changes

**Existing full retailer JWT (v2) — keep:**
- `sub` = UserId  
- `role` = RETAILER  
- `retailer_org_id`  
- `retailer_role` (OWNER|ADMIN|MANAGER|CASHIER|…)  
- `location_ids` (optional scope)

**New intermediate claims:**
```json
{
  "sub": "<user_id>",
  "role": "RETAILER",
  "token_use": "PendingOrgSelect",
  "phone": "<e164 or redacted>",
  "membership_count": 2,
  "exp": "<now+7m>"
}
```

Rules:
- `token_use == PendingOrgSelect` → **only** allow: `POST …/select-org`, `GET …/memberships` (optional list), logout/refresh-none.  
- All other `/v1/retailer/*` and business routes → **401/403** with `code=ORG_SELECT_REQUIRED`.  
- TTL: `PENDING_ORG_SELECT_TTL_SEC` default **420**. Max 600.

### API sketch

```
POST /v1/retailer/auth/login          # existing body; response branches
GET  /v1/retailer/auth/memberships    # requires PendingOrgSelect OR full JWT
POST /v1/retailer/auth/select-org     # { "retailer_id": "…" } + intermediate token
POST /v1/retailer/auth/switch-org     # { "retailer_id": "…" } + full JWT
```

**Login response shapes:**

A) Single membership OR flag off OR org not allowlisted multi:
```json
{ "token": "<full JWT>", "token_type": "full", "retailer_id": "…", "retailer_role": "…" }
```

B) Multi (flag on):
```json
{
  "token": "<intermediate JWT>",
  "token_type": "pending_org_select",
  "memberships": [
    { "retailer_id": "…", "retailer_role": "…", "name": "…", "is_active": true }
  ],
  "expires_in_sec": 420
}
```

**select-org / switch-org success:**
```json
{ "token": "<full JWT>", "token_type": "full", "retailer_id": "…", "retailer_role": "…" }
```

Errors:
- unknown membership → 403 `NOT_A_MEMBER`  
- inactive membership → 403 `MEMBERSHIP_INACTIVE`  
- intermediate expired → 401 `PENDING_ORG_EXPIRED`  
- intermediate on business route → 403 `ORG_SELECT_REQUIRED`

### Implementation files (expected)

| Area | Paths |
|------|--------|
| Login branch | `apps/backend-go/retailer/auth_login*.go` (or equivalent) |
| Mint helpers | `apps/backend-go/auth/` claims + issuer |
| Middleware | JWT middleware: reject `PendingOrgSelect` except allowlist paths |
| Memberships list | thin handler over `ListMembershipsByPhone` / ByUser |
| Config | bootstrap / feature flags |
| Tests | unit: single-org unchanged; multi → intermediate; TTL; middleware reject; select/switch |
| Markers (later C1.4) | picker + switch |

### Login algorithm (pseudocode)

```
u_list = ListMembershipsByPhone(phone)  // dual-read
if !MULTI_ORG_LOGIN_ENABLED || len(active) <= 1 || !allowlistHit(active):
    mint full JWT for sole (or first) membership  // LEGACY PATH — never remove
    return A
mint intermediate JWT (TTL 7m)
return B with memberships[]
```

### Rollback
1. Set `MULTI_ORG_LOGIN_ENABLED=false` (no deploy required if env).  
2. Intermediate mint code may remain dead.  
3. Do **not** remove legacy single-org mint in the same PR as multi.

### Acceptance
- [ ] Flag off: existing login tests + CORE single-org e2e green  
- [ ] Flag on: 2 memberships → intermediate only; cannot call POS until select-org  
- [ ] Intermediate expired → cannot select-org  
- [ ] switch-org only with full JWT + active membership  

---

## 5. PR C1.3 — Clients

### Goal
Picker UI + switcher + **clear-on-switch hard contract**.

### Clear-on-switch contract (all clients)

On successful `select-org` **or** `switch-org`:

| Must clear | Notes |
|------------|--------|
| Cart / hold draft | In-memory + local storage |
| Open POS session | Local session id; do not leave server session orphan without close policy |
| Offline count drafts | Local drafts for previous org/location |
| Assist context | Active assist ticket UI state |

Document in:
- `docs/WAVE_C_ENTERPRISE_SCALE_PLAN.md` (done)  
- Client README or `RETAILER_OS_*` note  
- Shared TS/native helper: `clearOrgScopedState()`

### Surfaces
- Retailer desktop / web  
- Retailer mobile (iOS/Android) if login exists  
- api-client: `selectOrg`, `switchOrg`, `listMemberships` types

### Acceptance
- [ ] Manual: switch org mid-cart → empty cart, no POS session, no stale assist  
- [ ] Unit/hook tests for clear helper  

---

## 6. PR C1.4 — Markers + SSMR

### Markers
```
PX_E2E_MULTI_ORG_PICKER_OK
PX_E2E_ORG_SWITCH_OK
PX_E2E_SINGLE_ORG_LOGIN_UNCHANGED_OK   # flag off regression
```

Wire into `ssmr-smokecheck` / marker gate JSON.

### Acceptance
- [ ] SSMR: flag off green  
- [ ] SSMR: allowlisted two-org user picker + switch green  
- [ ] Gate blocks if single-org login breaks  

---

## 7. PR C3.1 — Pos holds API

### DDL sketch (`YYYYMMDD_retailer_pos_holds.ddl`)

```sql
CREATE TABLE RetailerPosHolds (
  HoldId        STRING(36)  NOT NULL,
  RetailerId    STRING(36)  NOT NULL,
  LocationId    STRING(36)  NOT NULL,
  RegisterId    STRING(36),
  UserId        STRING(36)  NOT NULL,
  Status        STRING(16)  NOT NULL,  -- HELD|RESUMED|EXPIRED|VOIDED
  CartJson      STRING(MAX) NOT NULL,
  Note          STRING(512),
  ExpiresAt     TIMESTAMP   NOT NULL,
  CreatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  ResumedAt     TIMESTAMP,
  VoidedAt      TIMESTAMP
) PRIMARY KEY (RetailerId, HoldId);

CREATE INDEX Idx_RetailerPosHolds_ByLocationStatus
  ON RetailerPosHolds (RetailerId, LocationId, Status, ExpiresAt);
CREATE INDEX Idx_RetailerPosHolds_ByExpires
  ON RetailerPosHolds (Status, ExpiresAt);
```

### Invariants
- **Never** write stock / OnHand from hold park/resume/void.  
- Resume only if `LocationId` matches request location.  
- Default `ExpiresAt = now + 24h`.  
- Flag: `POS_HOLDS_ENABLED`.

### APIs
```
POST   /v1/retailer/pos/holds
GET    /v1/retailer/pos/holds?location_id=
POST   /v1/retailer/pos/holds/{id}/resume
POST   /v1/retailer/pos/holds/{id}/void
```

### Marker (with C3.2)
`PX_E2E_POS_HOLD_RESUME_OK`

---

## 8. PR C3.2 — Hold UI + sweeper

- Worker/cron: `Status=HELD AND ExpiresAt < now` → `EXPIRED`  
- Desktop/mobile park/resume UI  
- Sweeper idempotent  

---

## 9. PR C2.1 — HQ writers

### DDL sketch

```sql
CREATE TABLE RetailerHqSalesDaily (
  RetailerId   STRING(36) NOT NULL,
  LocationId   STRING(36) NOT NULL,
  Day          DATE       NOT NULL,
  SkuId        STRING(128) NOT NULL,
  QtySold      INT64      NOT NULL,
  QtyVoided    INT64      NOT NULL,
  GrossMinor   INT64      NOT NULL,
  NetMinor     INT64      NOT NULL,
  Currency     STRING(8)  NOT NULL,
  UpdatedAt    TIMESTAMP  NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (RetailerId, Day, LocationId, SkuId);

CREATE TABLE RetailerHqStockSnapshot (
  RetailerId   STRING(36) NOT NULL,
  LocationId   STRING(36) NOT NULL,
  SkuId        STRING(128) NOT NULL,
  OnHand       INT64      NOT NULL,
  Reserved     INT64      NOT NULL,
  AsOf         TIMESTAMP  NOT NULL,
  UpdatedAt    TIMESTAMP  NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (RetailerId, LocationId, SkuId);
```

### Code rule (non-negotiable)
In POS complete sale / void Spanner `ReadWriteTransaction`:
1. Write sale/void ledger  
2. Upsert `RetailerHqSalesDaily` deltas  
3. Outbox events  
**Same txn.** No post-commit goroutine for HQ sales.

Include `local:` SKUs in HQ sales. Exclude from supplier demand.

Flag: `HQ_ANALYTICS_ENABLED` may gate **reads**; writers can still run if cheap, or gate both — prefer **writers always on after C2.1 deploy**, reads behind flag (honest empty when flag off = no API).

### Tests
- Property: sum over locations = org total for day  
- Void reverses daily deltas  

---

## 10. PR C2.2 — HQ REST + desktop

```
GET /v1/retailer/hq/summary
GET /v1/retailer/hq/sales-by-location
GET /v1/retailer/hq/sales-by-sku
GET /v1/retailer/hq/shrinkage
GET /v1/retailer/hq/export
```

Auth: OWNER/ADMIN + reports.view + REPORTS_PRO pack.

UI: desktop `/hq`. Marker: `PX_E2E_HQ_SALES_BY_LOCATION_OK`.

---

## 11. PR C3.3 — Offline count conflict

### Commit API
```
POST /v1/retailer/stock/counts/commit
{
  "location_id": "…",
  "base_version": 12,
  "lines": [{ "sku_id": "…", "counted_qty": 5 }],
  "force": false
}
```

### 409 body (required shape)
```json
{
  "error": "COUNT_VERSION_CONFLICT",
  "server_version": 14,
  "server_lines": [
    { "sku_id": "…", "counted_qty": 7, "on_hand": 7 }
  ],
  "message": "Draft base_version is stale"
}
```

Force: `force=true` only if role ∈ {MANAGER, OWNER}; write audit row.

Marker: `PX_E2E_OFFLINE_COUNT_CONFLICT_OK`.

---

## 12. PR C4.1 — Assist SLA

- Default 15 min (`ASSIST_SLA_MINUTES`)  
- Worker scans open assist tickets past SLA  
- Emit via outbox → notif dispatcher: push + WS room  
- SMS only if channel configured  
- Marker: `PX_E2E_ASSIST_SLA_OK`  
- Flag: `ASSIST_SLA_ENABLED`

---

## 13. Marker catalog (Wave C)

| Marker | Epic |
|--------|------|
| `PX_E2E_SINGLE_ORG_LOGIN_UNCHANGED_OK` | C1 safety |
| `PX_E2E_MULTI_ORG_PICKER_OK` | C1 |
| `PX_E2E_ORG_SWITCH_OK` | C1 |
| `PX_E2E_POS_HOLD_RESUME_OK` | C3 |
| `PX_E2E_OFFLINE_COUNT_CONFLICT_OK` | C3 |
| `PX_E2E_HQ_SALES_BY_LOCATION_OK` | C2 |
| `PX_E2E_ASSIST_SLA_OK` | C4 |

Gate also runs **existing CORE single-org / POS / stock** suite with all Wave C flags off.

---

## 14. Rollout windows

| Step | Where | Action |
|------|--------|--------|
| 1 | SSMR | DDL apply (memberships done; holds/HQ later) |
| 2 | SSMR | Backend image with flag **off** |
| 3 | SSMR | Enable allowlist multi-org for 2 test orgs only |
| 4 | Staging | Same; no pilot roster multi-org |
| 5 | Prod | Flags off until product enablement ticket |

---

## 15. Agent start commands

```bash
# Verify C1.1 table
gcloud spanner databases ddl describe pegasusx-ssmr-db \
  --instance=pegasusx-ssmr-spanner --project=pegasus-503013 \
  | grep RetailerUserMemberships

# C1.2 worktree
cd apps/backend-go
go test ./retailer/ ./auth/ -count=1

# After C1.2 code: flag off regression
MULTI_ORG_LOGIN_ENABLED=false go test ./retailer/ -count=1
```

### C1.2 status — DONE (flag default off)

Implemented in-tree:
- `auth/claims.go` + `auth/jwt.go` — `TokenUse`, `PhoneNumber`, `RequireRole` reject
- `retailer/auth_multi_org.go` — select/switch/list + flag helpers
- `retailer/auth_login.go` — multi-org branch; refresh rejects intermediate
- `retailerroutes` — public memberships + select-org; protected switch-org
- Tests: `retailer/auth_multi_org_test.go`

### C1.3 status — DONE

- Desktop: clear-on-switch, select-org page, OrgSwitcher, login branch
- iOS: SelectOrgView + AuthManager multi-org
- Android: AuthScreen picker + API + ViewModel
- Types: `RetailerLoginResponse` / memberships in `@pegasusx/types`

### C1.4 status — DONE

- `e2e_multi_org.go` + `runE2ECheck` hook
- Manifest required: single-org + pending reject + multi_org_auth umbrella
- Alternatives: picker/switch OK|SKIPPED
- Notif multi-membership fanout: **not in C1.4** (document-only deferral)

### C3.1 status — DONE

- DDL `20260811_retailer_pos_holds.ddl`
- `pos_holds.go` + routes; flag `POS_HOLDS_ENABLED`
- Never touches stock; same-location resume; 24h TTL
- Unit tests PASS

### C3.2 status — DONE

- Sweeper every 15m (no-op when flag off)
- Desktop POS park/list/resume/void UI
- Types for hold wire shapes

### C2.1 status — DONE

- DDL applied pattern + SSMR when available
- `savePosSaleWithHQ`: ledger + HQ sales daily **same Apply**
- Void reverse net; local: SKUs included
- Tests PASS

### C2.2 status — DONE

- REST under `/v1/retailer/hq/*`; flag `HQ_ANALYTICS_ENABLED`
- Desktop `/hq` page + shell nav
- Tests PASS

**Next agent task:** **C3.3** offline count `base_version` + 409 protocol, **or C4** Assist SLA.

---

## 16. Definition of done (Wave C)

- [ ] All markers green on SSMR  
- [ ] Flags default off; pilot multi-org allowlist only  
- [ ] CORE single-org regression green  
- [ ] Design tightening T1–T5 implemented  
- [ ] No PG3; negotiations remain disabled  
- [ ] Runbooks for select-org, hold expire, count force audit  
