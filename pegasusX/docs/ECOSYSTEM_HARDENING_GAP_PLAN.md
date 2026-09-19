# Ecosystem Hardening Gap Plan (Beyond Retail OS / Next-Layer / Claims)

> **HISTORICAL / FROZEN — do not plan from this file alone.**
> Current SoT: [`PROD_READINESS_SEQUENCE.md`](./PROD_READINESS_SEQUENCE.md) · [`session-2026-08-07/ECOSYSTEM_GAP_REGISTER_2026-08-12.md`](./session-2026-08-07/ECOSYSTEM_GAP_REGISTER_2026-08-12.md) · [`FEATURES_BY_APP_ROLE.md`](./FEATURES_BY_APP_ROLE.md).
> Body is a point-in-time snapshot; re-verify any claim against code before acting.
>
> **Implementation progress (2026-08-20):**
> - **EH0 (Stop Lying Desks & Schema Truth):** SHIPPED — `supplier/session_scope.go` removes `sup-demo-1`; `compliance/handler.go` uses `resolveSessionSupplier()` via `auth.ResolveSupplierID`; rejects legacy query params.
> - **EH1 (Multi-Tenant Delivery Perimeters):** SHIPPED — `warehouse/perimeter.go` implements `PublishSupplierPerimeter`/`CheckSupplierPerimeter` using Redis sets `perimeter:supplier:{id}`; `order/warehouse_resolver_spanner.go` checks `SIsMember` during checkout; route: `POST /v1/warehouses/publish-perimeter`.
> - **Phase 6.1 (Platform Admin Feature Flags):** SHIPPED — `platformadmin/feature_flags.go` + `feature_flags_spanner.go`; routes: `GET/PUT /v1/platform-admin/tenants/{tenantType}/{tenantID}/flags`.
> - **EH2–EH5:** NOT STARTED — code does not exist for these phases.


**Status:** Implementation plan (code-grounded, sequential audit)  
**Repo:** `/Users/shakhzod/Desktop/V.O.I.D/pegasusX`  
**Date:** 2026-08-02  
**Method:** Iterative problem→evidence→solution→integrations (same shape as `RETAILER_RECEIVE_STOCK_CLAIMS_PLAN.md` §16)  
**Does not replace:** Retail OS · `NEXT_LAYER_ECOSYSTEM_PLAN.md` (L1–L11) · `RETAILER_RECEIVE_STOCK_CLAIMS_PLAN.md` (G1–G25) · `PLANOGRAM_VISION_PLAN.md`

---

## 0. Sequential reasoning trail (compressed)

| Step | Thought | Outcome |
|------|---------|---------|
| 1 | Spine is WIRED_E2E — where does field still break? | Look for silent omissions / half-wires, not wishlist |
| 2 | Exclude known plans (Retail OS, L1–L11, claims G*, planogram) | Avoid duplicate epics |
| 3 | Scan role desks + last-mile + money + geo + ops | Supplier CT demo, global perimeter, credit flag-off… |
| 4 | Verify with paths | `sup-demo-1`, `ssmr:delivery_perimeter`, rescue PROPOSED-only, float billing… |
| 5 | Revise: shop-closed DDL still “Pending ops” in parity-ledger despite other migrations | Keep as **P0 ops** |
| 6 | Branch: build pick-wave vs seal-only? | Pick wave after Spanner-only payload (E6→E7) |
| 7 | Prioritize by blast radius | Multi-tenant wrongness + money + load integrity first |

---

## 1. Executive verdict

Commerce/logistics spine works in happy path. Remaining ecosystem risk is **wrong-tenant UI**, **shared geo SoT**, **compute-without-enforce money engines**, **load-path dual memory**, and **ops toggles defaulted off** (observability/DR).

| Tier | IDs | Intent |
|------|-----|--------|
| **P0** | E1, E2, E3 | Stop demo-tenant desks; per-supplier zones; apply shop-closed DDL |
| **P1** | E4–E14, E16 | Credit/billing, WH load, driver edges, observability, DR, IAM |
| **P2** | E15 | Staging cost governance |

---

## 2. Gap catalog

Each: **Real-world problem** · **Code truth** · **Solution** · **Features** · **Integrations** (code / GCP / DevOps).

---

### EH-SUP — Supplier Desk Truth

#### E1 — Control Tower & Compliance bind `sup-demo-1` **(P0)**

| | |
|--|--|
| **Problem** | Ops open CT/Compliance on a real login and see empty/wrong-tenant charts — desk is decorative. |
| **Code truth** | `apps/supplier-portal/app/(portal)/control-tower/page.tsx` (`supplierId = "sup-demo-1"`); `…/compliance/page.tsx` same; empty mock/H3 stubs. |
| **Solution** | Session-scoped `scopedSupplierID`; fail closed; live pulse/WS; ban demo constants in prod builds. |
| **Features** | CT bootstrap from JWT; compliance export; honest empty states. |
| **Integrations** | `controltowerroutes`, supplier portal, WS, Spanner; ConfigMap `CONTROL_TOWER_PLAYBOOKS_*`. |

#### E12 — Playbooks shipped off; CT decorative **(P1)**

| | |
|--|--|
| **Problem** | Disruption playbooks never score/execute; ops stay on spreadsheets. |
| **Code truth** | `controltower/config.go` defaults off; worker only when enabled. |
| **Solution** | Staging: `ENABLED=true`, `AUTO_EXECUTE=false`; wire panel to E1 supplier id; allowlist auto-execute later. |
| **Features** | Run history; manual execute; zone-overrides (exists). |
| **Integrations** | `controltower/*`, Kafka, supplier portal, ConfigMap. |
| **Depends** | E1 |

---

### EH-GEO — Multi-tenant delivery perimeters

#### E2 — Single global Redis perimeter **(P0)**

| | |
|--|--|
| **Problem** | Two suppliers share one market: zone check uses one set `ssmr:delivery_perimeter` — wrong allow/deny. |
| **Code truth** | `retailer/proximity_service.go` `DeliveryPerimeterKey = "ssmr:delivery_perimeter"`; bootstrap one `DELIVERY_ZONE_CENTER_*`. |
| **Solution** | Per-supplier (later per-WH) keys; checkout passes `supplier_id`; typed `zone_miss`. |
| **Features** | Zone publish API; topology UI; retailer register/checkout gate. |
| **Integrations** | Redis Memorystore, Spanner topology, Places geocode, smoke perimeter assert. |

---

### EH-OPS — Ops reality

#### E3 — Shop-closed Spanner DDL pending on SSMR **(P0)**

| | |
|--|--|
| **Problem** | Driver shop-closed UX “wired” but grace/proximity columns may be missing → silent degrade. |
| **Code truth** | `context/parity-ledger.md`: `20260729_shop_closed_proximity_partial.ddl` **Pending ops**. |
| **Solution** | Apply DDL; schema-drift gate in CI/smoke; block release if columns absent. |
| **Features** | Migrate job; e2e shop-closed grace fields. |
| **Integrations** | Spanner, GKE migrate, ssmr-smokecheck. |

#### E13 — Observability Terraform default off **(P1)**

| | |
|--|--|
| **Problem** | FISCALIZING stick / outbox backlog grow with no page. |
| **Code truth** | `infra/terraform/variables.tf` `enable_observability_resources` default **false**. |
| **Solution** | Enable on SSMR+staging; alert channels; fiscal-aging SLO. |
| **Features** | Dashboards; runbook links from alerts. |
| **Integrations** | Cloud Monitoring, Prometheus metrics, Pager/email. |

#### E14 — DR restore unproven **(P1)**

| | |
|--|--|
| **Problem** | Backups Active; never restored to scratch + e2e — bad migration = market down. |
| **Code truth** | Backup schedule documented; no restore automation under `infra/terraform`. |
| **Solution** | Quarterly restore-to-scratch drill; RPO/RTO doc; Kafka/Redis rebuild checklist. |
| **Features** | Scripted restore; post-restore marker gate. |
| **Integrations** | Spanner backups, GKE, GSM, Kafka, Redis (E2 recompute). |

#### E15 — Staging/SSMR cost ungoverned **(P2)**

| | |
|--|--|
| **Problem** | Spanner/Redis/GKE burn between pilots; teardown is tribal. |
| **Code truth** | Budget emails exist; no schedule scale-to-zero. |
| **Solution** | Cost runbook; night/weekend scale; PU right-size; Slack budget. |
| **Features** | Scheduler jobs optional; destroy allowlist. |
| **Integrations** | Terraform, Billing, GKE HPA, Spanner. |

---

### EH-CREDIT-BILL — Credit & billing integrity

#### E4 — Credit scores suggest but don’t enforce **(P1)**

| | |
|--|--|
| **Problem** | HIGH risk / suggested limit 0 overnight; CREDIT_LEAVE still works — collections think they’re protected. |
| **Code truth** | `CREDIT_SCORE_ENFORCEMENT_ENABLED` default false; worker writes `SuggestedLimitMinor` only. |
| **Solution** | Desk “apply suggested”; freeze HIGH; staging kill-switch then enable. |
| **Features** | Apply flow; factor UI; audit. |
| **Integrations** | Spanner credit, outbox, supplier credit portal, ConfigMap. |

#### E5 — Billing meter `float64` + empty milestones **(P1)**

| | |
|--|--|
| **Problem** | Platform fees drift; volume tier never adjusts (milestone stub). |
| **Code truth** | `kafka/billing_tier_worker.go` float convert; `billing/meter_worker.go` empty milestone block. |
| **Solution** | Meter in **minor units only**; idempotent milestones + `FEE_RATE_ADJUSTED`. |
| **Features** | Tier table; supplier billing statement. |
| **Integrations** | Kafka ORDER_FINALIZED, Spanner billing, finance portal. |

---

### EH-WH-LOAD — Load integrity

#### E6 — Factory/Payload in-memory overlays **(P1)**

| | |
|--|--|
| **Problem** | Pod restart / multi-replica → sealed on one screen, open on another. |
| **Code truth** | `payload/service.go` overlay dual-write; `factory/service.go` in-memory maps + seeds. |
| **Solution** | Seal/reassign only via `manifest.Store`+Spanner; overlay non-prod only; fail closed with `REQUIRE_INFRA_ADAPTERS`. |
| **Features** | Spanner-only truck board; hydrate e2e. |
| **Integrations** | Spanner, payload/factory apps, Redis invalidate. |

#### E7 — No pick/pack wave before seal **(P1)**

| | |
|--|--|
| **Problem** | Short picks discovered at truck or doorstep — no SKU confirm on dock. |
| **Code truth** | Dispatch+seal exist; no pick-session/wave handlers. |
| **Solution** | Minimal pick wave → confirm → ready-to-seal gate (flag). |
| **Features** | Pick API + WH mobile/portal UI; barcode. |
| **Integrations** | Inventory Spanner, outbox, payload seal gate. |
| **Depends** | E6 |

#### E8 — Rescue propose event-only; capacity half-wired **(P1)**

| | |
|--|--|
| **Problem** | Broken truck; “propose rescue” doesn’t durable-rank capacity — accept can steal orders without VU check. |
| **Code truth** | `warehouse/dispatch_rescue.go` emits PROPOSED only; driver accept bulk-updates `DriverId` without residual. |
| **Solution** | `RescueRequests` state machine; rank by free VU+ETA; atomic order+manifest transfer. |
| **Features** | Preview/propose/accept/reject; capacity warnings. |
| **Integrations** | Spanner, outbox RESCUE_*, FCM, residual helpers. |

---

### EH-DRIVER-EDGE — Driver edge hardening

#### E9 — Offline nonce not single-use crypto **(P1)**

| | |
|--|--|
| **Problem** | Offline complete can be replayed — nonce derivable, no used-nonce ledger. |
| **Code truth** | `driver/service.go` `GenerateOfflineNonce` (hash pattern); baseline 8.3 target unmet. |
| **Solution** | Signed manifest + random single-use nonces in Spanner; CAS sync. |
| **Features** | Nonce table; reject codes; Sync Queue (exists). |
| **Integrations** | Spanner, driver Room/BGTask, signing key in GSM/KMS. |

#### E10 — Cash shortfall doesn’t seed bag recon **(P1)**

| | |
|--|--|
| **Problem** | Doorstep short cash → event only; EOD bag recon misses variance — finance chases noise. |
| **Code truth** | `order/fiscal.go` `emitCashVariance` outbox-only; `cashrecon` separate. |
| **Solution** | Upsert day’s recon line in same txn as collect-cash. |
| **Features** | Auto-seed lines; bag UI shows doorstep variances. |
| **Integrations** | cashrecon, CollectCash, supplier treasury, escalation worker. |

#### E11 — Temperature/condition ≠ claims/reverse **(P1)**

| | |
|--|--|
| **Problem** | Cold-chain breach reported; no claim/reverse — perishable loss never hits dock OPEN. |
| **Code truth** | `order/condition.go` event only; missing-items uses claims bridge. |
| **Solution** | TEMPERATURE/DAMAGED → claim draft + reverse when qty known; photo required. |
| **Features** | Condition→claim mapper; WH source `CONDITION`. |
| **Integrations** | claims, returns, GCS media, exceptions topic. |
| **Note** | Distinct from retailer store QUARANTINE (claims plan G8). |

---

### EH-IAM — Platform break-glass

#### E16 — No `PLATFORM_ADMIN` **(P1)**

| | |
|--|--|
| **Problem** | Support can’t safely cross-tenant inspect with audit; privacy doc expects platform admin. |
| **Code truth** | `privacy-multi-tenant.md` specifies it; `auth` has no `PLATFORM_ADMIN`. |
| **Solution** | Break-glass role + audit log on every cross-tenant read; IDOR tests. |
| **Features** | Break-glass JWT; audit export; support tooling. |
| **Integrations** | auth JWT, Spanner audit, compliance (after E1). |

---

## 3. Light notes (owned elsewhere)

| Item | Home |
|------|------|
| Quantity negotiations | Next-Layer L4 |
| Auto-order execution | Next-Layer L2.4 |
| Soliq OFD | L5 |
| GCS placeholder evidence | Claims plan G16/G21 |
| Receive↔stock↔claims | `RETAILER_RECEIVE_STOCK_CLAIMS_PLAN.md` |
| Planogram vision | `PLANOGRAM_VISION_PLAN.md` |
| GP SUCCESS / Firebase OTP | L1 |
| Supplier iOS `inventoryAudit()` stub | Fold into catalog epic |

---

## 4. Epics

| Epic | Gaps | Name |
|------|------|------|
| **EH-SUP** | E1, E12 | Supplier Desk Truth |
| **EH-GEO** | E2 | Multi-Tenant Delivery Perimeters |
| **EH-OPS** | E3, E13, E14, E15 | Ops Reality (DDL, observability, DR, cost) |
| **EH-CREDIT-BILL** | E4, E5 | Credit Enforcement & Integer Billing |
| **EH-WH-LOAD** | E6, E7, E8 | Load Integrity (seal, pick, rescue) |
| **EH-DRIVER-EDGE** | E9, E10, E11 | Driver Edge (offline crypto, cash bag, cold-chain) |
| **EH-IAM** | E16 | Platform Break-Glass IAM |

---

## 5. Phased PRs (suggested)

### Phase EH0 — Stop lying desks + schema truth

| PR | Gap |
|----|-----|
| EH0.1 Remove `sup-demo-1`; session-scope CT/Compliance | E1 |
| EH0.2 Apply `20260729_shop_closed_proximity_partial.ddl` + smoke | E3 |
| EH0.3 Schema-drift gate in CI | E3 |

### Phase EH1 — Multi-tenant geo

| PR | Gap |
|----|-----|
| EH1.1 Per-supplier perimeter Redis keys + checkout gate | E2 |
| EH1.2 Zone publish API + supplier topology UI | E2 |
| EH1.3 Smoke multi-perimeter | E2 |

### Phase EH2 — Load path

| PR | Gap |
|----|-----|
| EH2.1 Payload/factory Spanner-only seal path | E6 |
| EH2.2 Pick wave MVP + seal gate flag | E7 |
| EH2.3 RescueRequests + capacity accept | E8 |

### Phase EH3 — Money & risk

| PR | Gap |
|----|-----|
| EH3.1 Credit apply suggested + staging enforce flag | E4 |
| EH3.2 Billing meter int64 + milestones | E5 |
| EH3.3 Cash variance → recon seed | E10 |

### Phase EH4 — Driver crypto & cold chain

| PR | Gap |
|----|-----|
| EH4.1 Single-use offline nonces + signed manifest | E9 |
| EH4.2 Condition → claim/reverse bridge | E11 |

### Phase EH5 — Platform ops & IAM

| PR | Gap |
|----|-----|
| EH5.1 Enable observability resources + fiscal aging alert | E13 |
| EH5.2 DR restore drill runbook + first restore | E14 |
| EH5.3 PLATFORM_ADMIN + audit | E16 |
| EH5.4 CT playbooks staging (manual execute) | E12 |
| EH5.5 Cost schedule / right-size (optional) | E15 |

---

## 6. Recommended sequence

1. **E3 + E1** — last-mile schema + stop demo desks  
2. **E2** — before second supplier onboarding  
3. **E6 → E7 → E8** — load integrity  
4. **E13 → E14** — see failures, prove restore  
5. **E4 + E10 + E9 + E11** — money/risk edges  
6. **E5 + E12 + E16 + E15** — commercial meter, playbooks, IAM, cost  

**Parallel OK with:** claims RS* / Retail OS / L1 secrets (different blast radius).  
**Hard conflict:** avoid large Spanner DDL windows overlapping E3 + claims RS1 without a freeze calendar.

---

## 7. Real-world case additions (for `REAL_WORLD_CASE_MATRIX.md`)

| Case | Expected |
|------|----------|
| Second supplier different zone | Perimeter A ≠ B; checkout uses order’s supplier |
| CT opened as supplier X | Never renders `sup-demo-1` data |
| Shop-closed after DDL | Grace columns persist; timeout matrix fires |
| Credit HIGH + enforce on | CREDIT_LEAVE blocked; audit row |
| Rescue accept over capacity | Reject or split with residual warning |
| Offline replay same nonce | Second sync rejected |
| Doorstep cash short | Bag recon line auto-present |
| Temp breach with qty | Claim + WH reverse OPEN |
| Fiscal stuck 15m | Alert fires (observability on) |
| Spanner restore drill | Scratch DB passes smoke markers |

---

## 8. Success metrics

- Zero `sup-demo-1` string in supplier portal production bundle  
- Two suppliers, two Redis perimeter keys, both smoke-green  
- Shop-closed migration `DONE` on SSMR + e2e  
- Billing meter: no `float64` amount path in hot workers  
- Rescue accept updates manifest membership in one txn  
- Offline nonce reuse → hard fail in e2e  
- Observability flag on; ≥1 alert policy for outbox or fiscal aging  
- DR drill completed once with dated artifact  

---

## 9. Open questions

1. Perimeter SoT: supplier-only or warehouse can publish too?  
2. Credit enforce: auto-apply SuggestedLimit or human accept only?  
3. Pick wave: block seal hard, or warn-only in v1?  
4. PLATFORM_ADMIN: human break-glass only, or service accounts too?  
5. Offline signing: JWT HMAC vs Cloud KMS asymmetric?  

---

## 10. Documentation deliverables

- This file: `docs/ECOSYSTEM_HARDENING_GAP_PLAN.md`  
- Link from `NEXT_LAYER_ECOSYSTEM_PLAN.md` §17 / status report  
- Update `context/parity-ledger.md` when E3 lands  
- Append §7 rows to `docs/REAL_WORLD_CASE_MATRIX.md`  
- DR runbook artifact under `artifacts/dr/` after first drill  

---

## 11. Immediate next steps

1. **EH0.1** kill `sup-demo-1` in supplier CT/Compliance (small PR, huge trust).  
2. **EH0.2** apply shop-closed DDL on SSMR; flip parity-ledger.  
3. Design **E2** key naming (`perimeter:supplier:{id}`) before second supplier pilot.  
4. Keep claims RS* and L1 secrets on parallel tracks.  

---

## Appendix A — Relationship to other plans

```
Retail OS (packs/POS/stock)     ── product scale for retailers
Next-Layer L1–L11               ── GP, flywheel, Soliq, local SKU…
Claims G1–G25                   ── receive ↔ stock ↔ return window
Planogram PG*                   ── deferred shelf vision
THIS PLAN (E1–E16)              ── rest-of-ecosystem hardening
```

---

## Appendix B — Gap → epic → phase

| Gap | Epic | Phase |
|-----|------|-------|
| E1 | EH-SUP | EH0 |
| E2 | EH-GEO | EH1 |
| E3 | EH-OPS | EH0 |
| E4, E5 | EH-CREDIT-BILL | EH3 |
| E6, E7, E8 | EH-WH-LOAD | EH2 |
| E9, E10, E11 | EH-DRIVER-EDGE | EH3/EH4 |
| E12 | EH-SUP | EH5 |
| E13, E14, E15 | EH-OPS | EH5 |
| E16 | EH-IAM | EH5 |

---

*End of plan.*
