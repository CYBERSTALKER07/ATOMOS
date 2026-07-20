---
name: gap-hunter
description: >
  Deep multi-pass audit of the pegasusX / V.O.I.D. monorepo for technical AND non-technical
  gaps before cloud cutover. Finds silent failures, unwired spines, contract drift, missing
  edge cases, SOP/ops holes, credential-gate confusion, and role-row incomplete surfaces.
  Goal: when Boss connects real third-party APIs (PSP, OFD, Firebase, Maps), the team only
  wires secrets and endpoints — no new product logic, no half-features, no invent-on-cloud.
  Use for: audit, gap hunt, pre-cloud readiness, wire-ready, sanity check, silent failures,
  contract drift, "are we ready for staging/prod keys", post-feature E2E verification.
version: 2.0.0
---

# Gap Hunter v2 — Pre-Cloud Silent-Failure & Ecosystem Completeness Detective

**North star:** Cloud day is a **wiring day**, not a feature day.

When Spanner / Kafka / Redis / Global Pay / Payme / Click / OFD (my.soliq) / Firebase / Maps
go live, engineers should only:

1. Put secrets in GSM / External Secrets  
2. Flip env (`FISCAL_PROVIDER=MY_SOLIQ`, `GLOBAL_PAY_ENV=production`, …)  
3. Point base URLs at sandbox then prod  
4. Run validation runbooks  

They must **not** discover missing state machines, unfinished clients, silent event drops,
or invent new business logic under production credentials.

This skill finds anything that would force “edit logic / create features” after cloud connect.

---

## 0. Ecosystem purpose (always load before hunting)

Read at least these (pegasusX tree is canonical):

| Doc | Why |
|-----|-----|
| `pegasusX/context/purpose.md` | Mission + invariants (SupplierId, outbox, claims-not-body, Tiyin) |
| `pegasusX/context/PEGASUSX_CURRENT.md` | What pegasusX is (single-supplier logistics OS) |
| `pegasusX/context/architecture.md` | Topology: roles → backend-go → Spanner/Kafka/Redis |
| `pegasusX/docs/PRE_CLOUD_THIRD_PARTY_GATE.md` | Layer A (fakes) vs Layer B (live keys) |
| `pegasusX/docs/FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md` | Role rows + E2E proof model |
| `pegasusX/docs/DEPLOYMENT_READINESS_GAP_LEDGER.md` | Known open LC / PC items |
| `pegasusX/docs/WIRE_READY_STAGING_RUNBOOK.md` | Automated gates |
| Relevant ADRs under `pegasusX/docs/adr/` (esp. 001 pay-at-delivery, 009 fiscal hard-gate) |

### What we are

**pegasusX** = single-supplier **execution + planning** logistics OS for thousands of retailers:

```
Retailer order → Supplier/Warehouse plan & dispatch → Payload seal → Driver depart
  → Arrive → Handoff (QR) → Pay-at-delivery (cash/card) → FISCALIZING → OFD
  → COMPLETED (or audited FORCE) → shift/cash-bag close
```

Roles (JWT → apps must stay in sync):

| Role | Surfaces |
|------|----------|
| SUPPLIER (`ADMIN`) | supplier-portal + desktop + Android + iOS |
| RETAILER | desktop + Android + iOS |
| DRIVER | Android + iOS |
| WAREHOUSE_ADMIN | portal + Android + iOS |
| FACTORY_ADMIN | portal + Android + iOS |
| PAYLOADER | terminal + Android + iOS |

**Invariants (non-negotiable):**

1. Integer money (Tiyin) only — no float fiscal/payment amounts  
2. Transactional outbox with domain mutation (no direct Kafka write for money/state)  
3. Scope from JWT claims — never body `supplier_id` / `warehouse_id` for auth  
4. Pay-at-delivery only (ADR-001) — no silent pre-pay complete  
5. Fiscal hard-gate (ADR-009) — no `COMPLETED` without SUCCESS or audited FORCE  
6. Deploy **API + worker** — fiscal/async consumers are not optional  

---

## When to Use

- User asks to **audit / gap-hunt / sanity-check / pre-cloud / wire-ready**  
- Before connecting **real** third-party APIs  
- After a cross-cutting feature (order, payment, fiscal, dispatch, WS)  
- When symptoms are flaky with no stack trace  
- Phase exit: “can we stop building and only wire cloud?”  

**Pair with:** `pegasus-doctrine` (feature shape), `financial-integrity` (money),  
`kafka-event-contracts`, `spanner-discipline`, `native-mobile-safety`.

---

## Success criterion of a hunt

Answer explicitly:

> **Are we API-wiring-only for cloud?**  
> YES / NO — list every technical and non-technical item that still forces code or product work.

If NO, rank what blocks wiring-only cutover.

---

## Gap taxonomy (two tracks)

### Track T — Technical (software / contracts)

| ID | Class | Failure mode |
|----|--------|--------------|
| T1 | **Contract drift** | Same name, different JSON shape across Go / TS / Swift / Kotlin |
| T2 | **Dead / orphan code** | Defined never called; half-deleted |
| T3 | **Unwired feature** | Producer without consumer; route without client; WS without subscriber; event without dispatcher case |
| T4 | **Schema drift** | DDL ≠ Go scan/write ≠ DTO ≠ mobile model ≠ migration applied |
| T5 | **Enforcement gap** | Mutation without outbox / cache.Invalidate / RequireRole / idempotency |
| T6 | **Role / scope violation** | Body-trusted IDs; missing warehouse/factory scope |
| T7 | **Naming / role confusion** | ADMIN≠supplier; Customer≠retailer |
| T8 | **Silent fanout drop** | Event emitted → Kafka OK → dispatcher `default: return nil` → no WS/inbox |
| T9 | **Edge-case hole** | Happy path only; missing retry, late webhook, offline, concurrency, force path |
| T10 | **Fake/prod provider seam** | Real provider missing adapter **or** fake success invents fiscal/payment truth |
| T11 | **Worker / deploy topology** | API-only deploy; consumer never runs (orders stuck FISCALIZING) |
| T12 | **SSMR / proof gap** | Feature claims done without `PX_E2E_*` or unit proof |
| T13 | **Idempotency / replay** | Webhook or client key missing; double charge / double fiscal possible |
| T14 | **Money integrity** | Float; wrong amount (expected vs received); COMPLETED without fiscal |

### Track N — Non-technical (ops / product / cloud)

| ID | Class | Failure mode |
|----|--------|--------------|
| N1 | **Credential gate confusion** | Team thinks SSMR green = prod keys OK (Layer A vs B) |
| N2 | **SOP / runbook hole** | Support cannot resolve FISCAL_FAILED, shortfall, chargeback without inventing process |
| N3 | **Role training gap** | Driver/warehouse/finance don’t know force-complete authority or shift freeze |
| N4 | **Boss action backlog** | LC-01… Terraform, GSM, PSP, OFD TIN, Firebase plists — blocks cutover but not code |
| N5 | **Budget / capacity** | Features assume multi-region, multi-supplier, 24/7 on-call not staffed |
| N6 | **Legal / tax readiness** | OFD TIN, merchant agreements, force-complete audit retention not defined |
| N7 | **Comms / customer promise** | UI promises fiscal QR / tracking before path is E2E |
| N8 | **Pilot scope blur** | Product scope exceeds single-warehouse pilot without gates |
| N9 | **Hypercare / ownership** | No owner for worker-down, OFD-down, cash-bag freeze incidents |
| N10 | **Doc drift** | Checklist still open after code closed; wrong fail hooks; missing markers |

**Report both tracks.** Do not “fix” N-items with code unless they are really T-items mislabeled.

---

## Feature inventory — deep walk (loop every hunt)

Do **not** only grep randomly. Walk the **business spine** feature-by-feature. For each feature:

1. **Purpose** (one sentence from docs/purpose)  
2. **Backend package + routes**  
3. **Spanner tables / migrations**  
4. **Events produced** → dispatcher case → consumers → WS rooms  
5. **Every client in the role row** (or explicit portal-only deferral in ledger)  
6. **Edge cases** (table below)  
7. **SSMR marker or unit proof**  
8. **Cloud dependency** (none / env-only / third-party API)  
9. **Verdict:** wiring-only? | needs code | needs ops | deferred intentional  

### Spine map (minimum full ecosystem pass)

| # | Feature spine | Key packages / apps | Cloud third-party? |
|---|---------------|---------------------|--------------------|
| F1 | Auth login / OTP / JWT | `auth/`, portals, native | Firebase (live) |
| F2 | Retailer register + credit | `retailer/`, `credit/` | none |
| F3 | Catalog + checkout | `catalog/`, `order/`, retailer apps | none / maps geocode optional |
| F4 | Supplier order oversight | `supplier/`, supplier apps | none |
| F5 | Warehouse dispatch + fleet | `warehouse/`, `dispatch/`, `manifest/` | OSRM optional |
| F6 | Factory supply / transfer | `factory/` | none |
| F7 | Payload seal / gate | `payload/` | none |
| F8 | Driver depart / transit / arrive | `driver/`, `order/` | Maps SDK optional |
| F9 | Handoff QR / offload | `order/`, handoff | none |
| F10 | Pay-at-delivery cash | `order/CollectCash`, driver UI | none (FAKE fiscal) |
| F11 | Pay-at-delivery card | `payment/` webhooks → `SettleExternalPayment` | **PSP** |
| F12 | Fiscal hard-gate | `order/fiscal*`, worker consumer | **OFD MY_SOLIQ** |
| F13 | Force-complete + audit | force API, admin/warehouse UI | none |
| F14 | Cash shortfall / overage | collect + events + fanout | none |
| F15 | Shift freeze / return-complete | `driver` open-fiscal | none |
| F16 | Tracking / receipts / fiscal QR | `retailer` tracking | none |
| F17 | Ledger / settlement / disputes | `payment/`, supplier finance | PSP webhooks |
| F18 | Telemetry live map | `telemetry/`, fleet maps | Maps basemap optional |
| F19 | Planning / AI preorder | `planning/`, `ai-worker` | none |
| F20 | Offline sync batch | `order/sync_batch` | none |
| F21 | Notifications inbox / FCM | `notifications/`, device-token | **FCM/APNs** |
| F22 | Idempotency + reliability | `idempotency/`, middleware | Redis |

For a **scoped** hunt (user named a domain), still run **adjacent spines** (e.g. fiscal hunt must touch F10–F15, F16, F20, F22).

---

## Edge-case matrix (must search each spine)

For money/delivery spines especially, explicitly prove or flag:

| Edge | Expected behavior | Where to verify |
|------|-------------------|-----------------|
| Double event delivery | Idempotent; one fiscal SUCCESS | worker GetFiscalAttempt |
| Worker crash after OFD success | Re-check SUCCESS; no second OFD doc | ApplyFiscalWorkerResult |
| OFD timeout | FISCAL_FAILED / timeout code; retry | FiscalOFDTimeout |
| OFD down | Force path with reason + actor | ForceCompleteOrder |
| Late payment webhook | No-op if terminal / FISCAL_* | SettleExternalPayment |
| Offline client invents COMPLETED | Rejected | sync_batch |
| Cash received ≠ total | Fiscal = received; CASH_SHORTFALL event + fanout | CollectCash + dispatcher |
| Credit leave-behind | No fiscal until money received | SM + service |
| Force without reason / wrong role | 4xx | NormalizeForceReasonCode |
| Force when already SUCCESS | Rejected | ErrFiscalAlreadySucceeded |
| Shift end with open fiscal | 409 open_fiscal_block | HandleDriverReturnComplete |
| Optimistic concurrency | Retry or conflict, no silent overwrite | UpdateOrder version |
| Idempotency key + different body | 409 mismatch | middleware |
| API up, worker down | Stuck FISCALIZING; cash bag frozen | ops + deploy topology |
| Misconfigured real OFD | Hard-fail attempts, never invent SUCCESS | ProviderFromEnv hardFail |
| Soft COMPLETED from any path | Forbidden | state_machine + service tests |

Add domain-specific edges when hunting non-money features (seal before depart, cancel races, etc.).

---

## Hunt protocol — multi-pass loops

### Pass 0 — Context load (mandatory)

1. Read purpose + PRE_CLOUD_THIRD_PARTY_GATE + architecture summary.  
2. State the **wiring-only** question.  
3. Define scope: full ecosystem | money path | single role row | single package.  
4. Note Boss locks (e.g. force roles ADMIN+WAREHOUSE_ADMIN; FAKE OFD until SSMR green).

### Pass 1 — Cheap technical sweeps (Track T)

Run in parallel where possible:

```bash
# T1/T3 — events: constants vs dispatcher vs types
# events.go constants
# notification_dispatcher switch + parity default
# packages/types EventType union
# contracts/events.schema.json mapping

# T3 — HTTP mounts vs clients
# *routes/RegisterRoutes vs packages/api-client vs native API clients

# T5 — mutations without outbox (sample hot packages)
# UpdateOrder / BufferWrite / Apply without EmitJSON nearby

# T8 — events that only appear in EmitJSON, not in dispatcher cases
# diff producer types vs case lists

# T14 — soft COMPLETED
# NextStatus: StatusCompleted, ValidateStatusTransition, CollectCash
```

Prefer ripgrep / `grep` tools; keep paths under `pegasusX/` (not `pegasus/` reference unless parity).

### Pass 2 — Feature spine deep dive (loop)

For each feature in scope from the spine map:

```
LOOP feature F:
  1. Grep handlers + service methods
  2. Grep events produced (EmitJSON / Event*)
  3. Confirm dispatcher case (not silent default)
  4. Confirm consumer if async required (order mutator / worker)
  5. Confirm DDL + migration file exists and is applied in SSMR setup
  6. Confirm packages/types + schema
  7. Confirm EVERY client in role row OR ledger says intentional deferral
  8. Confirm edge-case tests or SSMR markers
  9. Confirm third-party seam is adapter-behind-env (fake default)
  10. Log gap or CLOSED
```

**Do not stop at first gap.** Complete the loop for the spine; then rank.

### Pass 3 — Non-technical (Track N)

Cross-read:

| Source | Look for |
|--------|----------|
| `PRE_CLOUD_THIRD_PARTY_GATE.md` | Layer A incomplete vs Layer B open items |
| `CLOUD_CREDENTIALS_CHECKLIST.md` | Secrets without validation steps |
| `PRODUCTION_CREDENTIAL_VALIDATION_RUNBOOK.md` | LC open rows |
| `PAYMENT_EXCEPTION_SOP.md`, `DRIVER_SUPPORT_PLAYBOOK.md` | Missing fiscal / shortfall / freeze language |
| `DEPLOYMENT_READINESS_GAP_LEDGER.md` | Stale “open” after code closed (N10) |
| Role QA docs under `docs/qa/` | Manual proof not scheduled |
| Budget docs | Features that blow $1500 pilot envelope |

Flag N-items as **Boss / ops / finance**, not as engineer code tickets, unless fixing is a doc update.

### Pass 4 — Triangulation (anti-false-positive)

For every candidate gap:

1. Confirm with file read (not grep alone).  
2. Check tests / SSMR markers that already cover it.  
3. Check intentional deferral in gap ledger / parity-ledger / ADR non-goals.  
4. If intentional: **N10 doc drift** only if docs claim “open” or wrong behavior.  
5. If real: classify T* or N*, severity, blast radius.

**Report only real gaps** when user asks for report-only. Still list CLOSED spines briefly if useful for wiring-only YES.

### Pass 5 — Rank & act

| Severity | Meaning |
|----------|---------|
| **P0** | Money/auth integrity; double charge/fiscal; body-trusted scope; COMPLETED without fiscal; invent SUCCESS on misconfigured OFD |
| **P1** | User-visible silent failure; missing fanout; client missing critical state; SOP hole that invents process under stress |
| **P2** | Dead code, naming, optional markers, polish |

Actions:

| Track | Default action |
|-------|----------------|
| T P0/P1 | Fix code + test + update docs in same change set |
| T P2 | Fix if cheap; else ledger |
| N* | Update docs / SOPs / checklists; do **not** invent product features |
| Cloud credentials | Never “implement” with fake prod keys; document LC steps |

**Cloud wiring readiness rule:**  
If any **T P0** remains → **NOT wiring-only**.  
If only **N / LC** remain → wiring-only for **code**; Boss must still complete credentials.

### Pass 6 — Proof

After fixes:

```bash
cd pegasusX/apps/backend-go && go test ./<packages>/ -count=1
# money path:
cd pegasusX && make test-ssmr-fiscal   # when Docker available
# contracts:
make gen-contracts-gate
```

Update `DEPLOYMENT_READINESS_GAP_LEDGER.md` / `PRE_CLOUD_THIRD_PARTY_GATE.md` § gap log when status changes.

---

## Deep search recipes (copy/adapt)

### A. Event produced → fanout complete?

```text
1. events/events.go  → constant
2. events/types.go   → payload struct + @Sync
3. outbox EmitJSON sites
4. order/consumer or other consumers
5. kafka/notification_dispatcher.go primary switch
6. notification_dispatcher_parity.go (must not be silent default for new money events)
7. packages/types EventType union
8. contracts/events.schema.json payload map
9. client WS handlers (optional but required for human-visible alerts)
```

### B. HTTP route → all clients?

```text
1. *routes/RegisterRoutes
2. packages/api-client
3. portal lib/*-api.ts
4. Android *Api.kt / Retrofit
5. iOS APIClient.swift
6. Idempotency key helpers both sides
```

### C. Money complete path?

```text
CollectCash / webhook → PAYMENT_CLEARED → FISCALIZING
→ FISCAL_RECEIPT_REQUESTED → worker → SUCCESS → COMPLETED + ORDER_FINALIZED
OR FORCE_SKIPPED + ORDER_FORCE_COMPLETED
Never: ARRIVED→COMPLETED, client-side fiscal, float amounts
```

### D. Provider seam safe for cloud?

```text
FISCAL_PROVIDER=FAKE default
MY_SOLIQ requires BASE_URL + API_KEY + TIN
Misconfig → hardFailProvider (no SUCCESS invent)
Payment: staging env before production; webhook HMAC required
```

---

## Pre-cloud “wiring-only” checklist (hunt must score)

### Layer A — Software (fakes) — eng owns

- [ ] `make wire-ready` (includes unit, parity, gap-hunter-gate, SSMR, **test-ssmr-fiscal**)  
- [ ] No soft COMPLETED; fiscal SUCCESS/FORCE only  
- [ ] FAKE OFD + fail/retry/force/shortfall/shift-freeze markers  
- [ ] Card clear → FISCALIZING path unit-proven  
- [ ] Dispatcher does not silently drop money/fiscal/cash-variance events  
- [ ] API **and** worker documented as required  
- [ ] Role-row clients show FISCALIZING / FISCAL_FAILED / force UI where needed  
- [ ] Offline cannot invent COMPLETED  
- [ ] Integer Tiyin throughout money path  

### Layer B — Live credentials — Boss/ops owns (not code features)

- [ ] Terraform + GSM secrets (LC-01)  
- [ ] Global Pay staging (LC-02)  
- [ ] Payme/Click if in scope (LC-03)  
- [ ] Firebase per app (LC-04)  
- [ ] Maps + OSRM (LC-05)  
- [ ] OFD sandbox after Layer A green (LC-07 / PRE_CLOUD)  
- [ ] SOPs staffed: payment exception, driver support, fiscal force  

If Layer A incomplete → **fix T gaps**.  
If Layer A complete and Layer B open → **report N/LC only**; stop inventing features.

---

## Output format (operator-grade)

### 1. Verdict

```text
WIRING-ONLY FOR CLOUD CODEPATH: YES | NO
Layer A (software): GREEN | RED
Layer B (credentials): OPEN items …
```

### 2. Findings table

| Track | Class | Sev | Location | Description | Blocks wiring-only? |
|-------|-------|-----|----------|-------------|---------------------|
| T/N | T8… | P0–P2 | path | … | Y/N |

### 3. Feature spine scorecard (scoped)

| Feature | Status | Proof | Cloud dep |
|---------|--------|-------|-----------|
| F10 Cash | CLOSED / GAP | test/marker | none |

### 4. Fixed now / Flagged follow-up / Doc updates

### 5. Regression

Commands run + pass/fail.

### 6. Explicit non-goals of this hunt

Do not expand multi-supplier, provisional credit fiscal, or prod OFD credentials “to make the audit pass.”

---

## Fix patterns

| Class | Fix |
|-------|-----|
| T1 Contract | One shared shape; regen contracts (`gen-contracts -source events`); mirror types |
| T3 Unwired | Wire end-to-end **or** remove half-feature; never leave zombie |
| T8 Silent drop | Explicit dispatcher case + optional inbox formatter |
| T9 Edge | Service guard + unit test + SSMR marker when money |
| T10 Provider | Adapter + env; hard-fail misconfig; FAKE for SSMR |
| T11 Worker | Document + k8s both deployments; fail readiness if consumers required |
| T14 Money | SM + service hard-gate; integer; received amount fiscal |
| N1–N10 | Docs, SOPs, ledger, Boss checklist — **not** new product features |
| Doc drift | Align checklist to code; never leave “open” when CLOSED |

**Propose before large fixes** if user asked report-only or scope is huge.

---

## Anti-patterns for the agent

- Grep once and declare “clean”  
- Fix only docs when code silently drops events  
- Treat SSMR green as “prod credentials proven”  
- Invent multi-supplier / new OFD product mid-audit  
- Report style noise (“great question”)  
- Confuse `pegasus/` reference tree with `pegasusX/` canonical  
- Batch five unrelated fixes without tests  

---

## Related skills & gates

| Skill / command | Role |
|-----------------|------|
| `pegasus-doctrine` | How features must be shaped |
| `financial-integrity` | Money rules |
| `kafka-event-contracts` | Event seams |
| `make wire-ready` | Automated Layer A (includes fiscal SSMR) |
| `make test-ssmr-fiscal` | Money hard-gate markers |
| `make gap-hunter-gate` | Minimal CI symbol check (not a substitute for this skill) |

---

## Source material (when ambiguous)

- `pegasusX/context/purpose.md`  
- `pegasusX/docs/PRE_CLOUD_THIRD_PARTY_GATE.md`  
- `pegasusX/docs/adr/009-fiscal-hard-gate.md`  
- `pegasusX/docs/FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md`  
- `pegasusX/docs/DEPLOYMENT_READINESS_GAP_LEDGER.md`  
- `pegasusX/apps/backend-go/order/{state_machine,fiscal,fiscal_provider}.go`  
- `pegasusX/apps/backend-go/kafka/notification_dispatcher.go`  
- `pegasusX/contracts/ssmr_fiscal_markers.json`  
