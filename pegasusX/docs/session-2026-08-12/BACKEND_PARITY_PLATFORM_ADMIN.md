# Backend Parity — PLATFORM_ADMIN (A7)
> **POINT-IN-TIME SNAPSHOT (2026-08-12) — do not treat as current status.**
> Re-verify any claim against live code before acting. Multiple ecosystem hardening phases have shipped since this audit.


**Date:** 2026-08-12  
**Agent:** A7-PLATFORM_ADMIN  
**Phase:** AUDIT ONLY (no code changes)  
**Client SoT:** `admin-portal` (web only)  
**Protocol:** [`BACKEND_PARITY_PROTOCOL.md`](./BACKEND_PARITY_PROTOCOL.md)

---

## 0. Scope & packages

| Package | Role |
|---------|------|
| `apps/backend-go/platformadmin` | Tenant lifecycle, audit store, WS publish, flag auditor |
| `apps/backend-go/featureflags` | Dual-control money flags |
| `apps/backend-go/mfa` | TOTP enroll/verify + `RequireStepUp` |
| `apps/backend-go/platformroutes` | Client policy, device token, **auth refresh/logout** |
| `apps/backend-go/partner` | Admin partner keys / AS2 / SFTP / COA |
| `apps/backend-go/globalproducts` (+ routes) | Product match queue |
| `apps/backend-go/ar` + `creditroutes` | Dunning run-once |
| `apps/backend-go/auth` | `RolePlatformAdmin`, `RequireTenant` exemption, JWT revoke |
| `apps/backend-go/ws` | `platform-admin` room |

Mount wiring: `main.go:175-177` (platformadmin + featureflags + MFA step-up), `main.go:384` (partner admin keys), `globalproductsroutes`, `creditroutes` dunning.

---

## 1. Feature inventory (route → service → Class A)

Class A definition (protocol): JWT-scoped mutation → Spanner RW + in-txn outbox → relay → Kafka → consumers → WS/FCM/webhook; cache invalidate; idempotency; no silent writes.

**Legend:** ✅ pass · ⚠️ partial · ❌ fail / missing · n/a not applicable to this surface

### 1.1 Core platform-admin (`/v1/platform-admin/*`)

| Route | Method | Handler | Auth / MFA | Spanner | Outbox | Realtime | Idempotency | Audit | Class A |
|-------|--------|---------|------------|---------|--------|----------|-------------|-------|---------|
| `/v1/platform-admin/tenants` | GET | `HandleListTenants` | `RequireRole(PLATFORM_ADMIN)` + step-up | RO | — | — | — | — | n/a read |
| `/v1/platform-admin/tenants/{tenantType}/{tenantID}` | GET | `HandleGetTenant` | same | RO | — | — | — | — | n/a read |
| `/v1/platform-admin/tenants/{tenantType}/{tenantID}/transition` | POST | `HandleTransitionTenant` → `Service.Transition` | same | **Separate** `Apply` upserts (`spanner.go:40`, `:91`) — not single RW txn with audit | **None** | Hub `PLATFORM_ADMIN_AUDIT` (`service.go:212`) | **None** | Insert audit **error ignored** (`service.go:203`) | ❌ silent audit path |
| `/v1/platform-admin/audit` | GET | `HandleListAudit` | same | RO | — | — | — | — | n/a read |
| `/v1/platform-admin/ws-session` | GET | `HandleWebSocketSession` | same | — | — | Issues WS JWT (`handlers.go:103-127`) | — | — | n/a ticket |

Evidence mounts:

```130:148:apps/backend-go/platformadmin/handlers.go
// RegisterRoutes mounts PLATFORM_ADMIN-gated routes.
// Optional stepUp middleware enforces TOTP when MFA is enrolled/required.
func RegisterRoutes(r chi.Router, h *Handlers, stepUp ...func(http.Handler) http.Handler) {
	// ...
	r.Route("/v1/platform-admin", func(pr chi.Router) {
		pr.Use(auth.RequireRole(auth.RolePlatformAdmin))
		// stepUp...
		pr.Get("/tenants", h.HandleListTenants)
		// ...
		pr.Post("/tenants/{tenantType}/{tenantID}/transition", h.HandleTransitionTenant)
		pr.Get("/audit", h.HandleListAudit)
		pr.Get("/ws-session", h.HandleWebSocketSession)
	})
}
```

### 1.2 Feature flags (dual-control)

| Route | Method | Handler | Auth / MFA | Dual-control | Fail-closed audit | Class A |
|-------|--------|---------|------------|--------------|-------------------|---------|
| `/v1/platform-admin/flags/{flagKey}` | GET | `HandleEvaluate` | PLATFORM_ADMIN + step-up | PENDING overrides ignored at eval (`service.go:123-127`) | — | n/a read |
| `/v1/platform-admin/flags/{flagKey}` | PUT | `HandleSetOverride` | same | Money → `PENDING` + reason required (`service.go:146-150`) | Audit **after** Upsert; 500 on audit fail but row already written (`handlers.go:81-110`) | ⚠️ |
| `/v1/platform-admin/flags/{flagKey}/approve` | POST | `HandleApproveOverride` | same | Approver ≠ setter (`service.go:184-186`) | Approve then audit; on audit fail override already **ACTIVE** (`handlers.go:128-156`) | ❌ money activate without durable audit |

Money-affecting set:

```13:21:apps/backend-go/featureflags/service.go
var MoneyAffectingFlags = map[string]bool{
	"AR_INVOICES_ENABLED":              true,
	"AR_DUNNING_ENABLED":               true,
	"AUTO_ORDER_PLACE_ENABLED":         true,
	"AUTO_ORDER_SHADOW_ENABLED":        true,
	"AUTO_ORDER_SOAK_GATE_DISABLED":    true,
	"FISCAL_PROVIDER":                  true,
}
```

Special approve audit actions: `FLAG_AUTO_ORDER_PLACE`, `FLAG_AUTO_ORDER_SOAK_GATE` (`handlers.go:13-19`, `:141-145`).

No outbox / Kafka / idempotency on flag mutators.

### 1.3 MFA

| Route | Method | Handler | Notes | Class A |
|-------|--------|---------|-------|---------|
| `/v1/platform-admin/mfa/status` | GET | `HandleStatus` | No step-up (enroll surface) | n/a |
| `/v1/platform-admin/mfa/enroll` | POST | `HandleEnroll` | Writes pending secret `Enabled=false` | ⚠️ Spanner Apply only |
| `/v1/platform-admin/mfa/confirm` | POST | `HandleConfirm` | Enables MFA; issues `mfa_verified` JWT; audit error ignored (`mfa/service.go:122-124`) | ⚠️ |
| `/v1/platform-admin/mfa/verify` | POST | `HandleVerify` | Step-up verify; audit error ignored (`:144-146`) | ⚠️ |

`RequireStepUp` (`mfa/handlers.go:152-191`): required when enrolled **or** `PLATFORM_ADMIN_MFA_REQUIRED`; production config forces required (`bootstrap/config_validate.go:30-31`).

MFA routes intentionally **exclude** step-up self-gate (`handlers.go:161-163`).

### 1.4 Partner admin (admin-portal Partner panel)

| Route | Method | Role gate | PLATFORM_ADMIN tenant resolution | Audit / outbox | Class A |
|-------|--------|-----------|----------------------------------|----------------|---------|
| `POST /v1/admin/partner-keys` | POST | Admin/Retailer/**PlatformAdmin** (`partner/routes.go:56`) | Body `tenant_id` required (`handlers.go:784-791`) | **No** PlatformAdminAudit | ❌ silent issue |
| `GET /v1/admin/partner-keys` | GET | same | Query `tenant_id` required (`:841-849`) | — | n/a read |
| `POST /v1/admin/partner-keys/{keyID}/revoke` | POST | same | **Broken for PLATFORM_ADMIN** — no PlatformAdmin branch; uses empty `claims.SupplierID` (`handlers.go:867-885`) | **No** audit | ❌ **P0** |
| `GET/PUT /v1/admin/partner-sftp` | GET/PUT | same | Query via `jwtTenant` (`:731-741`) | No governance audit | ⚠️ |
| `GET/PUT /v1/admin/partner-as2` | GET/PUT | same | same | No governance audit | ⚠️ |
| `GET/PUT /v1/admin/partner-coa` | GET/PUT | same | same | No governance audit | ⚠️ |

**MFA step-up:** partner admin routes are **not** wrapped in `mfa.RequireStepUp` (only `platformadmin` + `featureflags` mounts in `main.go:175-176`).

Admin-portal revoke call omits tenant scope: `api.revokePartnerKey(token, keyId)` → `POST …/revoke` with `{}` (`apps/admin-portal/lib/api.ts:120-121`) — even if body/query were added, handler currently ignores them for platform admin.

### 1.5 Global products match queue

| Route | Method | Role | Notes | Class A |
|-------|--------|------|-------|---------|
| `GET /v1/admin/product-match-queue` | GET | ADMIN **or** PLATFORM_ADMIN (`globalproductsroutes/routes.go:31-32`) | List | n/a read |
| `POST /v1/admin/product-match-queue/{id}/resolve` | POST | same (`:33-34`) | ACCEPT: `UpsertOffer` then `UpdateMatchQueue` as **two** Spanner Applies (`service.go:267-281`); no ownership check when actorSupplier empty (PLATFORM_ADMIN path leaves it empty — `handlers.go:143-147`); **no** PlatformAdminAudit; **no** outbox; **no** MFA step-up | ❌ silent catalog mutation |

### 1.6 AR dunning run-once

| Route | Method | Role gate (router) | Handler gate | Class A |
|-------|--------|--------------------|--------------|---------|
| `POST /v1/admin/ar/dunning/run-once` | POST | ADMIN **or** PLATFORM_ADMIN (`creditroutes/routes.go:55`) | **Only `RoleAdmin`** (`ar/handlers.go:83-86`) | ❌ **route/handler mismatch** — PLATFORM_ADMIN always 403 |

Admin-portal calls this from Partner panel (`api.runDunningOnce`, `PartnerPanel.tsx:46-53`).

Money side-effects when run: aging + dunning step advance + possible credit holds (worker plane). No platform audit row on trigger.

### 1.7 Auth revoke / logout (shared; used by platform session)

| Route | Method | Handler | Notes | Class A |
|-------|--------|---------|-------|---------|
| `POST /v1/auth/refresh` | POST | `auth.HandleTokenRefresh` (`platformroutes/routes.go:37`) | New jti; preserves claims including `MFAVerified` if present in source JWT | n/a session |
| `POST /v1/auth/logout` | POST | `auth.HandleLogout` (`:38`) | Denylists jti until exp (`refresh.go:56-100`); Redis store in prod | ⚠️ memory fallback multi-pod |
| JWT parse path | — | `tokenRevoked` (`auth/jwt.go:246`) | Checks denylist | ✅ mechanism present |

### 1.8 Platform routes (non-governance)

| Route | Role | Notes |
|-------|------|-------|
| `GET /v1/platform/client-policy` | public-ish | Version policy |
| `PUT /v1/platform/client-policy` | **ADMIN** only (`platformroutes/routes.go:26`) | Not PLATFORM_ADMIN |
| `GET /v1/media/upload-ticket` | multi-role, no PLATFORM_ADMIN | Out of admin-portal |
| `POST /v1/user/device-token` | open post | Shared |

### 1.9 WebSocket platform-admin room

| Piece | Evidence | Status |
|-------|----------|--------|
| Hub name | bootstrap `ws.NewHub("platform-admin", …)` | ✅ |
| Room | `ws.PlatformAdminRoom()` → `"platform-admin"` (`ws/rooms.go:10-12`) | ✅ |
| Subscribe | `subscribePlatformAdminRooms` for `RolePlatformAdmin` (`ws/handler.go:149-160`) | ✅ |
| Publish | `Service.publish` → `hub.Broadcast(PlatformAdminRoom)` (`platformadmin/service.go:77-93`) | ✅ |
| Event type | `PLATFORM_ADMIN_AUDIT` (tenant + flag audits) | ✅ |
| Test | `platformadmin/ws_test.go` transition broadcast | ✅ |
| Client | `admin-portal/lib/use-admin-ws-refresh.ts` mints `/ws-session` then `/v1/ws` | ✅ |
| Kafka path | None — direct hub only | ⚠️ intentional fan-out, not Class A bus |

---

## 2. Class A checklist summary (platform mutators)

| Check | Tenants transition | Money flags set/approve | Partner keys | Match resolve | Dunning run-once | MFA enroll/verify |
|-------|--------------------|-------------------------|--------------|---------------|------------------|-------------------|
| 1 Auth scope | ✅ role + MFA | ✅ role + MFA | ⚠️ role, **no MFA**; revoke tenant scope broken | ⚠️ role, no MFA | ❌ handler blocks PLATFORM_ADMIN | ✅ |
| 2 Idempotency | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| 3 Spanner RW (atomic) | ⚠️ separate Applies | ⚠️ separate Apply + audit | Apply keys only | ❌ split offer/queue | worker path | Apply only |
| 4 Outbox in-txn | ❌ | ❌ | ❌ | ❌ | (worker-internal) | ❌ |
| 5 Cache invalidate | n/a | n/a (env/override read path) | n/a | n/a | n/a | n/a |
| 6 Realtime | ✅ WS hub | ✅ WS via `RecordFlagAudit` | ❌ | ❌ | ❌ | ⚠️ audit→WS if audit succeeds |
| 7 Edge cases | lifecycle FSM tested | dual-control tested | revoke broken | partial write risk | role mismatch | step-up tested |
| 8 Tests | unit + WS | dual-control + audit actor tests | limited | service tests | handler gate untested for PA | totp_test step-up |

**Overall Class A for PLATFORM_ADMIN plane: FAIL** — governance uses audit table + WS, not outbox/Kafka; several mutators are silent or broken for the admin-portal role.

---

## 3. Gaps (P0 / P1 / P2)

### P0 — money / auth IDOR / silent money state / broken break-glass ops

| ID | Gap | Evidence | Impact |
|----|-----|----------|--------|
| **P0-1** | ~~Partner key revoke broken~~ **FIXED Wave B5** | PA requires tenant_type/tenant_id; admin-portal passes tenant | was empty SupplierID scope |
| **P0-2** | ~~Money-flag approve not fail-closed~~ **FIXED Wave B5** | audit fail → `RevertApproveToPending` | was ACTIVE without audit |
| **P0-3** | ~~Dunning handler role mismatch~~ **FIXED Wave B5** | handler allows ADMIN \| PLATFORM_ADMIN | was PA always 403 |

### P1 — incomplete governance plane / MFA holes / silent mutations

| ID | Gap | Evidence | Impact |
|----|-----|----------|--------|
| **P1-1** | **Tenant transition audit is silent on failure** | `_ = s.repo.InsertAudit(...)` after successful Upsert (`platformadmin/service.go:197-211`). | Tenant status changes without audit trail; WS may still fire. |
| **P1-2** | **Tenant transition not single RW txn** (tenant + audit) | Two independent `client.Apply` (`spanner.go:40`, `:91`). | Partial state: approved without audit or opposite race. |
| **P1-3** | **Flag set audit after write** | `SetOverride` then audit (`featureflags/handlers.go:81-110`). | PENDING row without set-audit if audit fails (less severe than approve). |
| **P1-4** | ~~No MFA step-up on partner/match/dunning~~ **FIXED Wave B5** | step-up on partner keys, match queue, dunning run-once | was platformadmin+flags only |
| **P1-5** | **Match resolve silent + non-atomic** | `ResolveMatch` two Applies; no PlatformAdminAudit (`globalproducts/service.go:246-287`). | Catalog link changes without governance trail or outbox. |
| **P1-6** | **Partner key issue / AS2 / SFTP / COA mutations silent** | No `RecordFlagAudit` / PlatformAdminAudit on issue/revoke/put. | Break-glass partner surface not in console audit feed. |
| **P1-7** | **No Class A outbox for governance state** | No OutboxEvents emission in platformadmin/featureflags/partner admin/match. | Multi-pod / offline consumers cannot reconstruct admin actions from Kafka; only Spanner tables + live WS. |
| **P1-8** | **No idempotency on public mutators** | No Idempotency-Key on transition / flags / partner revoke / match resolve. | Double-submit: dual-control re-set rewrites PENDING; match double-accept risk. |
| **P1-9** | **MFA enroll/verify audit errors ignored** | `mfa/service.go:122-124`, `:144-146`. | Security-relevant events may drop from audit. |

### P2 — tests / polish / intentional deferrals

| ID | Gap | Evidence |
|----|-----|----------|
| **P2-1** | No unit test for `RequireTenant` PLATFORM_ADMIN exemption | Code `auth/tenant.go:109-114`; `tenant_test.go` only covers ADMIN without tenant. |
| **P2-2** | `HandleRevokeKey` asymmetric vs List/Issue for platform admin | List/Issue handle PlatformAdmin; revoke does not. |
| **P2-3** | Platform client-policy upsert is ADMIN-only | `platformroutes/routes.go:26` — OK if out of admin-portal scope. |
| **P2-4** | Direct hub WS without durable relay | Acceptable for console refresh if Spanner audit is SoT; document intentional. |
| **P2-5** | FISCAL_PROVIDER treated as bool money flag | `MoneyAffectingFlags` map is bool-enabled only; fiscal provider string flags may not fit Enabled model. |
| **P2-6** | Logout without jti reports OK | Legacy tokens (`refresh.go:84-88`) — intentional soft path. |

---

## 4. Event / consumer matrix

| Emitter | Event / record | Bus | Consumer | Room / channel |
|---------|----------------|-----|----------|----------------|
| `platformadmin.Service.publish` | JSON `{type: PLATFORM_ADMIN_AUDIT, action, tenant_*, actor, detail}` | **In-process WS hub only** | Connected PLATFORM_ADMIN consoles | `platform-admin` |
| `platformadmin.RecordFlagAudit` | Spanner `PlatformAdminAudit` + same WS | Spanner + hub | Audit GET + WS refresh | same |
| Tenant transition | `TENANT_{STATUS}` actions | Spanner + hub | same | same |
| Flag set | `FLAG_OVERRIDE_SET` | Spanner + hub | same | same |
| Flag approve | `FLAG_OVERRIDE_APPROVE` / `FLAG_AUTO_ORDER_PLACE` / `FLAG_AUTO_ORDER_SOAK_GATE` | Spanner + hub | same + place flip ops | same |
| MFA | `MFA_ENROLL_CONFIRM` / `MFA_VERIFY` | Spanner audit (if wired) + hub | Audit panel | same |
| Partner key / match / dunning | **None to governance bus** | — | — | — |
| Kafka / OutboxEvents | **Not used** for this role plane | — | — | — |

Flag runtime consumers (not admin-portal, but money effect):

| Consumer | Flag | Path |
|----------|------|------|
| `retailer.placeAllowedForRetailer` | `AUTO_ORDER_PLACE_ENABLED` | dual-control ACTIVE override |
| AR dunning worker | `AR_DUNNING_ENABLED` / `AR_INVOICES_ENABLED` | env (+ tenant override if evaluated) |

---

## 5. Edge-case matrix

| Case | Behavior | Verdict |
|------|----------|---------|
| Illegal tenant transition (e.g. SUSPENDED→PENDING) | `validateTransition` rejects (`service.go:216-234`); 409 | ✅ |
| OFFBOARDED terminal | No outbound transitions | ✅ |
| Legacy tenant missing row | `IsActive` true (`service.go:148-150`) | ✅ intentional |
| Money flag set without reason | rejected (`service.go:147-149`) | ✅ |
| Self-approve money flag | `approver_must_differ_from_setter` | ✅ |
| PENDING override runtime effect | Evaluate ignores non-ACTIVE | ✅ |
| Approve when not PENDING | `override_not_pending` | ✅ |
| MFA required, not enrolled | 403 `mfa_enrollment_required` | ✅ |
| MFA enrolled, not verified | 403 `mfa_verification_required` | ✅ |
| Production MFA required flag | config validate fails if false | ✅ |
| PLATFORM_ADMIN + RequireTenant enforced | exempt (`tenant.go:109-114`) | ✅ code; ⚠️ untested |
| PLATFORM_ADMIN entity cross-tenant read | `EntitySupplierAllowed` bypass (`entity_scope.go:24-26`) | ✅ |
| PLATFORM_ADMIN partner list without tenant_id | 400 `tenant_id_required` | ✅ |
| PLATFORM_ADMIN partner revoke | broken (P0-1) | ❌ |
| PLATFORM_ADMIN dunning run-once | router OK, handler 403 (P0-3) | ❌ |
| Match resolve as PLATFORM_ADMIN | ownership skipped (empty actorSupplier) | ✅ intended break-glass |
| Double flag approve | second fails not pending | ✅ |
| Audit Insert fails on Transition | ignored; HTTP 200 with new status | ❌ P1-1 |
| Audit fails on money Approve | HTTP 500 but flag ACTIVE | ❌ P0-2 |
| Logout + refresh revoked jti | 401 `token_revoked` (tested) | ✅ |
| WS ticket copies MFAVerified | `IssueWSTicket` copies session claims | ✅ |

---

## 6. Tenant exemption for platform admin (`RequireTenant`)

**Status: IMPLEMENTED**

```109:115:apps/backend-go/auth/tenant.go
			// Break-glass governance: PLATFORM_ADMIN is a cross-tenant role whose
			// JWT carries no SupplierID. Requiring a tenant would 401 every
			// platform-admin route in enforced envs, so exempt it here.
			if claims, ok := FromContext(r.Context()); ok && claims.Role == RolePlatformAdmin {
				next.ServeHTTP(w, r)
				return
			}
```

- Wired globally: `main.go:132` `r.Use(auth.RequireTenant(cfg.TenantContextEnforced))`.
- Without this exemption, all `/v1/platform-admin/*` and partner admin GETs would 401 under `TENANT_CONTEXT_ENFORCED` / production.
- **Gap:** no dedicated test asserting PLATFORM_ADMIN passes without TenantContext (P2-1).

---

## 7. Silent mutations (inventory)

| Mutation | Silent? | Why |
|----------|---------|-----|
| Tenant transition | **Yes (audit path)** | Audit insert error discarded; no outbox |
| Flag set (money PENDING) | **Partial** | HTTP fails if audit fails but PENDING row remains |
| Flag approve (money ACTIVE) | **Yes on audit fail** | ACTIVE without audit (P0-2) |
| MFA confirm/verify | **Partial** | Spanner MFA write OK; audit discarded on error |
| Partner issue key | **Yes** | No PlatformAdminAudit |
| Partner revoke key | **Yes + broken** | No audit; platform tenant resolution missing |
| Partner SFTP/AS2/COA PUT | **Yes** | No governance audit |
| Match queue resolve | **Yes** | Catalog write only |
| Dunning run-once | **Yes (if it ran)** | No admin audit; currently blocked for PLATFORM_ADMIN |

Protocol flag: *“silent mutation, orphan event, wrong room, api-only run-mode hole.”*  
Platform plane uses intentional WS-only fanout; silent = **Spanner state change without durable governance audit and/or without fail-closed coupling**.

---

## 8. MFA step-up coverage map

| Surface | Step-up enforced? |
|---------|-------------------|
| `/v1/platform-admin/tenants*`, `/audit`, `/ws-session` | ✅ via `RegisterRoutes(..., mfa.RequireStepUp)` |
| `/v1/platform-admin/flags/*` | ✅ |
| `/v1/platform-admin/mfa/*` | ❌ by design (bootstrap path) |
| `/v1/admin/partner-*` | ❌ |
| `/v1/admin/product-match-queue*` | ❌ |
| `/v1/admin/ar/dunning/run-once` | ❌ (+ handler role bug) |
| `/v1/auth/logout`, `/refresh` | n/a |

---

## 9. Dual-control money flags — fail-closed assessment

| Requirement | Status |
|-------------|--------|
| Reason required on money set | ✅ `reason_required_for_money_flag` |
| PENDING until second approver | ✅ |
| Approver ≠ setter | ✅ |
| PENDING not evaluated at runtime | ✅ |
| Place-flip audit action name | ✅ `FLAG_AUTO_ORDER_PLACE` |
| Soak-gate break-glass money flag | ✅ `AUTO_ORDER_SOAK_GATE_DISABLED` + `FLAG_AUTO_ORDER_SOAK_GATE` |
| Atomic override + audit | ❌ separate writes |
| Reject mutation if audit cannot persist | ⚠️ set returns 500 after write; **approve leaves ACTIVE** |
| Dual-control tests | ✅ `featureflags/service_test.go` + `handlers_test.go` |

---

## 10. Proposed fixes (do **not** implement in audit phase)

### P0

1. **`HandleRevokeKey`:** mirror List/Issue — for `RolePlatformAdmin` require `tenant_type` + `tenant_id` (query and/or body); scope revoke to that tenant. Align admin-portal `revokePartnerKey` to pass tenant fields.
2. **Money flag approve fail-closed:** single Spanner RW txn: read PENDING → set ACTIVE + insert `PlatformAdminAudit` (or reverse-order with compensating rollback). Return success only if both commit.
3. **`HandleRunDunningOnce`:** allow `auth.RolePlatformAdmin` (and optionally MFA step-up); write PlatformAdminAudit action `AR_DUNNING_RUN_ONCE`.

### P1

4. **Tenant `Transition`:** one `ReadWriteTransaction` UpsertTenant + InsertAudit; **return error** if audit fails (no `_ =`).
5. **Wrap partner + match + dunning admin routes** with `mfa.RequireStepUp` (or shared governance middleware group).
6. **Record PlatformAdminAudit** on partner key issue/revoke, match resolve, dunning run-once (actor + tenant + detail).
7. **Match resolve:** atomic offer + queue update; optional outbox if catalog consumers need it.
8. **Idempotency-Key** on transition / flag set / flag approve / partner revoke / match resolve.
9. **MFA audit:** surface error or fail closed on enroll confirm if audit required in production.

### P2

10. Unit test: `RequireTenant(true)` + `RolePlatformAdmin` without tenant → 200.
11. Document intentional “governance plane = Spanner audit + WS hub (no Kafka)” if product accepts non-Class-A bus for admin-only events.
12. Review `FISCAL_PROVIDER` as money flag (bool model fit).

---

## 11. Client inventory cross-check (`admin-portal`)

| UI surface | API used | Backend parity |
|------------|----------|----------------|
| Tenants panel | list/get/transition | Wired; transition Class A fail (audit silence) |
| Flags panel | eval/set/approve | Dual-control OK; approve audit fail-closed fail |
| Audit panel | list audit | Wired |
| Match queue | list/resolve | Wired; silent resolve |
| Partner panel | keys list/revoke, AS2/SFTP/COA GET, dunning run-once | List/GET OK; **revoke P0**; **dunning P0** |
| MFA gate | status/enroll/confirm/verify | Wired + prod required |
| Live refresh | ws-session + `/v1/ws` | Wired |

Client: **web only** (`ROLE_ROW_PARITY_MATRIX.md` PLATFORM_ADMIN row) — no mobile by design. ✅

---

## 12. Verdict

| Area | Verdict |
|------|---------|
| Role + MFA spine for core `/v1/platform-admin/*` | **Strong** (prod MFA required, step-up middleware, dual-control semantics, WS room) |
| RequireTenant exemption | **Present** (test gap) |
| Class A data plane (outbox/Kafka) | **Not met** (by design gap or incomplete — flag as P1-7) |
| Fail-closed money flag audit | **Not met on approve** (P0-2) |
| admin-portal partner revoke + dunning | **Broken for PLATFORM_ADMIN** (P0-1, P0-3) |
| Silent mutations | **Multiple** (P1-1, P1-5, P1-6) |

**Role readiness for production break-glass:** **CONDITIONAL** — core tenant/flag/MFA path is largely functional under MFA; partner revoke + dunning console actions and money-flag audit atomicity must be fixed before claiming full admin-portal Class A / ops parity.

---

## 13. File index (primary evidence)

| Path | Why |
|------|-----|
| `apps/backend-go/platformadmin/handlers.go` | Routes, transition, ws-session |
| `apps/backend-go/platformadmin/service.go` | Transition, publish, RecordFlagAudit, silent InsertAudit |
| `apps/backend-go/platformadmin/spanner.go` | Separate Apply paths |
| `apps/backend-go/featureflags/{handlers,service,spanner}.go` | Dual-control + audit ordering |
| `apps/backend-go/mfa/{handlers,service}.go` | Step-up + MFA audit ignore |
| `apps/backend-go/auth/tenant.go` | PLATFORM_ADMIN RequireTenant exemption |
| `apps/backend-go/auth/refresh.go` | logout / refresh revoke |
| `apps/backend-go/partner/{routes,handlers}.go` | Partner admin gates + revoke bug |
| `apps/backend-go/globalproducts/{handlers,service}.go` | Match queue |
| `apps/backend-go/globalproductsroutes/routes.go` | Role gates |
| `apps/backend-go/creditroutes/routes.go` | Dunning route roles |
| `apps/backend-go/ar/handlers.go` | Dunning handler RoleAdmin only |
| `apps/backend-go/ws/{rooms,handler}.go` | Platform admin room subscribe |
| `apps/backend-go/main.go` | Mount + step-up wiring |
| `apps/backend-go/bootstrap/{bootstrap,config_validate}.go` | Hub, MFA required, flag audit wiring |
| `apps/admin-portal/lib/api.ts` | Client contract |
| `apps/admin-portal/components/PartnerPanel.tsx` | Revoke + dunning UI |

---

*End of A7-PLATFORM_ADMIN audit.*
