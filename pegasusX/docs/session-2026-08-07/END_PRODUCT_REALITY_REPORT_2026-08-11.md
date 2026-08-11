# PegasusX — End-Product Reality Report

**Evidence base:** live source tree `/Users/shakhzod/Desktop/V.O.I.D/pegasusX` as of **2026-08-11**. Every claim below traces to code, schema, or configuration. File:line references are given so findings can be re-checked. Deleted documentation is not used.

Enterprise program status (proved by gates this session): Phase 0 money-path, Phase 1 money/law, Phase 2 integration, Phase 3 ops truth, Phase 4 autonomy foundations, Phase 5 runtime tenancy — all `*-gate-ok` on Spanner emulator. Analytics column tenancy wired (`analytics-tenancy-gate-ok`). This report states what is **wired and live** vs **present but decorative/blocked** and what is still **absent**.

---

## 1. Human / Field-Agent Displacement

### What the field agent does vs what the code does

| Field-agent job | PegasusX coverage today | Evidence |
|-----------------|--------------------------|----------|
| Walk in, present catalog / price | **Automated** — retailer self-serve browse/catalog/cart | `apps/retailer-app-android/.../ui/screens/`, `apps/retailer-app-desktop/app/(dashboard)/` (catalog, cart) |
| Take order | **Automated** — self-serve checkout, incl. multi-supplier split | `apps/backend-go/order/unified_checkout.go`, `order/parent_orders.go` |
| Propose/replenish automatically | **Automated (shadow), gated flip** — auto-order shadow proposals; `place` requires evidence + manager + two-person rule | `retailer/auto_order.go:15-24,146-178`, `auto_order_shadow.go`, `scripts/auto_order_place_flip_check.sh` |
| Negotiate price | **Partial** — delivery-date negotiation exists; price/RFQ does not | `NegotiationProposals` (date, not price); RFQ absent |
| Negotiate credit terms | **Not automated for retailer** — terms are supplier/admin-set; retailer view is read-only | `credit/service.go`, `creditroutes/routes.go:29,46` |
| Extend credit decision | **Automated by policy/status, not a score** — terms + aging + `DelinquencyCount` + CREDIT_HOLD auto-freeze; credit scoring desk was deliberately removed | `docs/CREDIT_ECOSYSTEM_BEHAVIOR.md`, `ar/dunning*.go` |
| Collect payment (cash) | **Automated at delivery, human driver still hands cash** — PENDING_CASH_COLLECTION → driver collect/confirm | `order/state_machine.go:24-71`, `apps/driver-app-android/.../CashCollectionScreen.kt` |
| Dunning / collections outreach | **Automated in-app + off-app SMS/email; WhatsApp absent** | `ar/dunning_channels.go:19-31`, `ar/dunning_worker.go` |
| Physical relationship / trust / store visit | **Hard human requirement** — no code substitutes the visit | — |

### Automatable today (live code)

- Catalog presentation, order capture, order split across suppliers, delivery + cash capture at the door, AR invoicing, aging, dunning step machine with SMS/email, credit hold enforcement, refunds, payout batching, billing fee invoices.
- Auto-order runs in **shadow** and is only promotable to `place` behind evidence + human signoff (place stays off by default).

### Hard human requirements (no code path)

- New-account acquisition and pitch (no telesales/agent console; `apps/admin-portal` and `apps/supplier-app-desktop` are redirect stubs).
- Credit **negotiation** dialogue (terms are set, not negotiated).
- Physical cash hand-off (driver still collects; no cash-in-app rail).
- On-shelf merchandising/planogram enforcement.

### Trajectory

- **Near-term (0–12 mo):** hybrid reduction is realistic — routine replenishment (auto-order after evidence) + self-serve ordering + automated dunning remove the *order-taking and follow-up* portion of the agent job. Acquisition and exception handling stay human.
- **3–5 yr:** complete field-agent wipe-out is **not** credible from this codebase. The durable residue is acquisition, credit negotiation, and physical execution. Best case is a small human layer on top of an automated transaction plane; the code is structured exactly that way (shadow → gated place; policy credit, not negotiated credit).

---

## 2. Problem Coverage vs Existing Logistics / Planning Software

| Problem class | O9 / Kinaxis / Blue Yonder / ERP+WMS | B2B marketplace | PegasusX today | Evidence |
|---|---|---|---|---|
| Demand forecast / baseline | Strong | None | **Wired** — baseline per supplier/day/wh/SKU + WAPE/bias/tracking | `DemandForecastBaseline`, `ForecastAccuracyDaily` (`spanner.ddl:1462,1482`) |
| Safety stock / reorder | Strong | None | **Wired (flag)** — SS v2 + reorder suggestion worker | `replenishment/`, `SAFETY_STOCK_V2_ENABLED` |
| Demand sensing (weather/promo/payday) | Strong | None | **Wired** — `DemandSignals` + `DemandAdjustments`, tenant-stamped | `demand/`, `20260819_demand_analytics_supplier_id.ddl` |
| S&OP capacity plan | Strong | None | **Wired (foundations)** — real CapacityModel; no projection literals | `planning/service.go:246,253,297`; gate asserts no `sku-projection` |
| Route optimization | Strong | None | **Partial** — OR-Tools service; replicas 1 in SSMR/staging, **0 in prod** | `services/optimizer-core/`, `overlays/prod/kustomization.yaml:62-69` |
| WMS (lots/FEFO/pick/cycle) | Strong | None | **Wired (flags)** | `20260806_wms_*.ddl`, portal + Android/iOS |
| Order-to-cash / AR / dunning | ERP-side | Weak | **Wired + off-app** | `ar/`, `billing/`, `payout/` |
| Fiscal legal receipt | ERP/local | None | **Blocked on EDS** — MY_SOLIQ wired; signer is dev-HMAC only | `order/fiscal_provider.go`, `fiscal/signer_env.go:69-70` |
| Product master / match | MDM | Partial | **Wired (backend)** — GlobalProducts + match queue | `globalproducts/` |
| Marketplace RFQ / scorecards / escrow | — | Strong | **Absent** (Phase 6, decision-gated) | — |

**Where it falls short:** certified optimizer at prod scale, certified GS1 DataMatrix (placeholder ECC200 — `gs1/datamatrix.go:48-53`), certified EDI/AS2 (Drummond), BI sink, RFQ/scorecards/escrow.

**Vertical advantage:** the code models the whole chain transactionally — factory supply request → warehouse lot/FEFO pick → dispatch → driver cash → fiscal → AR → payout — with native apps per role. Generic suites integrate these; PegasusX already couples them in one schema and outbox.

---

## 3. Alignment with Systems Used by Big Retailers / Suppliers

| Integration surface | State | Evidence |
|---|---|---|
| Partner auth/keys/scopes | Wired | `partner/keys.go`, `partner/auth.go` |
| Master-data upsert API | Wired, idempotent | `partner/masterdata*.go`, routes `partner/routes.go` |
| Webhooks | Wired — allowlist (13 types), rotate-secret | `partner/webhook_events.go:9-23`, `service.go:316` |
| EDI | Wired — CONTRL/APERAK ACKs, inbound ORDRSP/INVOIC, outbound worker | `partner/edi/`, `partner/edi_inbound.go`, `edi_outbound.go` |
| Transport | SFTP strict host-key pin (fail-closed); AS2 present | `partner/sftp.go:53-77` |
| GS1 | GTIN/GLN/SSCC/ZPL wired; **DataMatrix placeholder** | `gs1/` |
| 1C / CommerceML | Design + reference converter only (not certified) | `docs/COMMERCEML_EXCHANGE.md`, `scripts/commerceml_import_ref.py` |
| Journals (1C CSV/XML) | Wired; VAT breakout + credit-note legs | `partner/export_journals.go` |
| SDK | OpenAPI contract + generator; **generated SDK not committed** | `contracts/partner.openapi.yaml`, `scripts/gen_partner_sdk.sh`, `sdk/partner/README.md` only |

**To adopt without re-keying:** a big supplier running 1C needs (a) certified ECC200 DataMatrix for UZ/CIS marking, (b) certified AS2/EDIFACT (Drummond), (c) the generated SDK committed/published, (d) bidirectional CommerceML beyond the reference script, and (e) a scripted 5k-SKU reference chain proven on SSMR. Items (a–c) are cert/packaging, not architecture gaps.

---

## 4. Existence of a True Unified Platform

No public system combines, in one transactional plane: multi-supplier catalog with a shared product master, cross-supplier checkout that splits into per-supplier child orders, per-role native execution apps (warehouse/driver/payload/factory), policy credit + AR dunning, fiscal receipts, and tenant isolation.

PegasusX’s actual position vs that ideal:

- **Multi-supplier cart/split** — Wired (`ParentOrders`, `MULTI_SUPPLIER_CHECKOUT_ENABLED`).
- **Shared product master** — Wired backend (`GlobalProducts` + match queue); retailer multi-partner UI absent.
- **Physical execution roles** — Wired (native apps per role; WMS waves/lots/cycle).
- **Routine replenishment without humans** — present but **shadow-gated**; place requires 30-day ≥80% acceptance artifact + human flip.
- **Tenant isolation** — Wired (TenantContext, freeze, outbox `SupplierId NOT NULL`, rate limits per tenant); two-supplier SSMR IDOR markers previously green; registration reopen is flag-gated.
- **Fiscal legality** — PEGASUS commercial receipts live; legal Soliq OFD blocked on EDS key.

Net: closest to the ideal in code, but autonomy is deliberately evidence-gated and fiscal/optimizer are not yet production-certified.

---

## 5. Per-Role / Per-App / Per-Feature Detail

Legend: **Live** (wired, gate/test evidence) · **Partial** (present, incomplete/flag-gated) · **Decorative/Broken** · **Absent**.

### Retailer (Android / iOS / Desktop)
- Live: catalog, cart, multi-supplier checkout, orders, delivery payment, credit profile (read), POS + sell-through feed, shifts. (`retailer-app-android/…/ui/screens`, `retailer-app-desktop/app/(dashboard)/` 32 pages)
- Partial: auto-order (shadow default; place gated); iOS scheme/dir typo `reatilerapp` (buildable, sloppy).
- Absent: pitch/telesales surface; credit negotiation.
- Hygiene: `retailer-app-desktop` has `clean.tsx`, `fix_settings.py` debris at app root.

### Supplier (portal + apps)
- Live: portal ~70 pages (orders, catalog, pricing + retailer overrides, credit policy/collections, treasury, dispatch, fleet, replenishment, exceptions, integrations); Android + iOS substantial.
- Absent: RFQ / supplier scorecards (Phase 6); negotiation is date-only.
- Decorative: `apps/supplier-app-desktop`, `apps/admin-portal` are redirect stubs.

### Driver
- Live: manifest, offload review, **cash collection**, fiscalizing/fiscal-fail handling, offline queue + sync. (`driver-app-android/…/CashCollectionScreen.kt`, `OfflineVerifierScreen.kt`; iOS mirror)
- Absent: none structural; cash remains physical.

### Payload / Loading
- Live: order checklist, truck sidebar, offline queue, idempotency keys; RN terminal real.
- Maintainability outlier: `payload-terminal/App.tsx` ~2.5k-line single file.

### Factory
- Live: loading bay, transfers, supply requests, payload override, staff/fleet; portal 20 pages.

### Warehouse
- Live: dispatch, pick waves, bins/lots FEFO, cycle counts, claims/returns, stock commitments, demand forecast, CRM; Android 109 screens + scanner, iOS 96 files, portal 43 pages.
- Partial: mobile floor execution for some surfaces still routed via transfer actions (no dedicated pick-wave nav).

### Admin (platform)
- Live: backend `platformadmin` tenant lifecycle + audit + `PLATFORM_ADMIN` role; feature flags with money-flag reason. (`platformadmin/`, `featureflags/`, `main.go:160`, `auth/claims.go:20`)
- Absent: **no dedicated admin console UI** (API-only); two-person dual-control for money flags is reason-only today.

### Cross-cutting backend (money / integrity / concurrency)
- Order state machine: 18 states, centralized transitions — `order/service.go:48-71`, `state_machine.go:24-71`.
- Capture: query-before-capture, idempotent — `payment/service.go:658-702`; unique idempotency index `spanner.ddl:1724`.
- Refunds: capped at captured−refunded, reversal legs, fiscal corrective — `order/refunds.go:17-45`; `UQ_Refunds_IdempotencyKey`.
- Payouts: batch = captured−refunds−commission, **bank-file export only (no live payout rail)** — `payout/payout.go:14-28`.
- Fiscal: env provider PEGASUS/FAKE/GLOBAL_PAY/MY_SOLIQ; misconfig fails closed; Soliq HTTP adapter real; **EDS signer dev-only** — `fiscal/signer_env.go:69-70`.
- Outbox: `SupplierId NOT NULL` + `_platform` sentinel + fair interleave + backfill — `spanner.ddl:640-656`, `outbox/fair.go:8-23`, `backfill.go`.
- Partial allocation: flag-gated, default off — `allocation/service.go:82-91`.

### Missing / weak features — purpose, why, logic, end-to-end
1. **Legal fiscal EDS signing** — *Purpose:* close Soliq OFD sales legally. *Why:* current signer is dev-HMAC. *Logic:* pkcs12/E-IMZO sign EHF → submit → poll status → on SUCCESS stamp receipt; on failure FISCAL_FAILED retry. *E2E:* order arrive → awaiting payment → capture → fiscalize → OFD success → completed. *Ref:* `fiscal/signer_env.go`, `order/fiscal.go`.
2. **Live payout rail** — *Purpose:* actually move supplier money. *Why:* today only a bank-file CSV is produced. *Logic:* batch = Σcaptured − Σrefunds − commission; execute via rail with webhook confirm; ledger entries per leg. *Ref:* `payout/payout.go`.
3. **RFQ / supplier scorecards (Phase 6)** — *Purpose:* competitive sourcing. *Why:* no price-negotiation path. *Logic:* RFQ → quotes → award; score = fill-rate, OTIF, claim rate. *E2E:* retailer RFQ → supplier quotes → award → order.
4. **Certified DataMatrix + AS2/Drummond** — *Purpose:* legal marking + certified EDI. *Ref:* `gs1/datamatrix.go` placeholder.
5. **Prod optimizer deploy** — *Purpose:* real routing at scale. *Why:* prod replicas = 0, placeholder image. *Ref:* `overlays/prod/kustomization.yaml:62-69`.
6. **Admin console UI + dual-control money flags** — *Purpose:* governed operations. *Ref:* `platformadmin/` API-only.
7. **Committed partner SDK** — *Purpose:* no-re-key adoption. *Ref:* `sdk/partner/README.md` only.

### Clearly decorative / broken
- `apps/admin-portal/`, `apps/supplier-app-desktop/` redirect stubs.
- `gs1/datamatrix.go` self-declared non-certified placeholder.
- `sdk/partner/` empty (README only).
- Root/repo debris: `refactor*.py`, `patch_*.py`, `apps/backend-go/patch_spy*.py`, `rewrite_factory.patch`, `warnings.log`, `mocks/globalpay`, nested `softwareengineercv-main/` (unrelated CV site).

---

## 6. Recommendations

**P0 — correctness / legality (block revenue or legality)**
1. Procure EDS key; implement pkcs12/E-IMZO signing; prove Soliq sandbox SUCCESS behind `FISCAL_PROVIDER=MY_SOLIQ`.
2. Owner keys: Global Pay merchant, Twilio/SendGrid/PlayMobile, Firebase SMS/APNs.
3. Apply NOT-NULL/tenancy migrations on live (`20260819_outbox_*`, `20260819_route_performance_*`, `20260819_demand_analytics_*`) after backfill confirms zero NULLs.

**P1 — structural product truth**
4. Live payout rail (replace bank-file export) or document it as the permanent settlement mechanism.
5. Admin console UI for tenant lifecycle + flags with dual-control on money flags.
6. Commit/publish generated partner SDK; certify DataMatrix + AS2.

**P2 — planning quality**
7. Prod optimizer image + replicas ≥1; chaos-prove fallback (`optimizer_source` heuristic).
8. Run 30-day shadow soak → produce `artifacts/forecast-shadow/acceptance-30d.json` → two-person `place` flip for pilot cohort.

**P3 — scale / enterprise (decision-gated Phase 6)**
9. RFQ + supplier scorecards + escrow/second gateway only on Phase 1 billing + Phase 4 evidence.
10. BI sink (BigQuery) and certified EDIFACT/1C package.

**Scope/architecture**
- Keep autonomy evidence-gated (shadow → measured place); do not flip by assertion.
- Delete repo debris (`refactor*.py`, `patch_*.py`, `mocks/globalpay`, nested CV site) and fix the `reatilerapp` iOS path typo.
- Decide the fate of redirect-stub apps (delete or replace) to stop advertising non-existent surfaces.

---

*Report generated from the live tree; nothing here is sourced from deleted docs. Gates referenced: `money-path-gate`, `phase1..phase5-gate`, `analytics-tenancy-gate` (all green 2026-08-11 on Spanner emulator).*
