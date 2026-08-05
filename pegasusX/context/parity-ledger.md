# Parity ledger (intentional divergences)

## Client parity closure (2026-08-05)

| Item | Status | Notes |
|------|--------|--------|
| P0-4 iOS offline geofence silent success | **Wired** | `deliverOrder` uses `isNetworkEnqueueable`; GPS fail-closed; lat/lng on queue payload |
| AUTHORIZE_BYPASS photo capture on retailer | **Wired** | Desktop + Android + iOS upload + `photo_url` |
| Supplier/WH return-policy settings | **Wired** | Portal + Android + iOS; types + `@pegasusx/api-client` |
| Driver PoD photo for credit leave | **Wired** | Android + iOS require evidence photo before credit |
| Empty analytics chart theatre | **Closed** | Forecast / heatmap / velocity unmounted; SpendAnalytics thin-wired |
| Portal i18n bootstrap | Partial | LocaleBootstrap on supplier+warehouse; shell `sign_out` via `createTranslator`; mobile catalogs still unwired |
| Interactive Substance Gate UI walks | READY_FOR_WALK | [`artifacts/SUBSTANCE_GATE_CLIENT_SIGNOFF_2026-08-05.md`](../artifacts/SUBSTANCE_GATE_CLIENT_SIGNOFF_2026-08-05.md); human PASS/FAIL pending |

## Claims / chargebacks (2026-07-29)

| Item | Status | Notes |
|------|--------|--------|
| Supplier portal claims queue + settlement modes | Wired | `@pegasusx/types` + `@pegasusx/api-client` |
| Supplier Android / iOS claims queue + settlement modes | Wired | Same backend endpoints as portal |
| Supplier claim-chargebacks ledger (all 3 clients) | Wired | `GET /v1/supplier/claim-chargebacks` |
| Claim chargebacks live WS push | **Wired** | `CLAIM_FILED` / `CLAIM_RESOLVED` fan via `handleDriverEdgeEvent` (supplier + retailer + driver/warehouse rooms) + inbox formatters |
| Retailer file-claim media | Prior work | Camera both platforms; not part of this close |
| Manual PSP chargebacks (payment/chargeback) | Unchanged | Separate from logistics claim chargebacks |

See also: `docs/CLAIM_ROLE_ROW.md`.

## Shop-closed / partial offload / proximity (Feature 1, 2026-07-29)

| Item | Status | Notes |
|------|--------|--------|
| Backend: proximity unlock, partial offload, timeout matrix, credit/cash gates | Wired | `ARRIVED_SHOP_CLOSED` wire status; LineItemsJson for line qty |
| Events + notification dispatcher + formatters | Wired | TIMEOUT, PROXIMITY_UNLOCKED, PARTIAL_OFFLOAD, CREDIT_LEAVE |
| Types + api-client | Wired | `@pegasusx/types` / `@pegasusx/api-client` |
| Driver Android API + VM (unlock, partial, credit gate, cash unlock) | Wired | Compose badge UI polish deferred |
| Driver iOS API methods | Wired | Full stop-screen UX polish deferred |
| Retailer desktop/Android/iOS enhanced response codes | Wired | RESCHEDULE / CREDIT_LEAVE / CANCEL / AUTHORIZE_BYPASS labels |
| Supplier portal + Android + iOS queue fields | Wired | grace_ends_at, reason, shop_closed_resolution |
| Spanner DDL applied on live SSMR | **Wired** | `20260729_shop_closed_proximity_partial.ddl` applied; CI `schema-drift-gate` + `ssmr-smokecheck spanner` assert Orders grace/proximity cols + `OrderShopClosedLog` |
| Offline action queue (full driver offline flush orchestrator) | **Wired** | Android Room/WorkManager + iOS QueuedDriverAction/BGTask; Sync Queue UI; 4.1 order; dead-letter |
| AUTHORIZE_BYPASS photo capture on retailer | **Wired** | 2026-08-05 Client Parity Closure |

See: `docs/big-platform-baseline/last-mile/`.

## Product-deferred (exit criteria only — no UI work until flags flip)

| Item | Status | Exit criteria |
|------|--------|----------------|
| Quantity negotiations | **Product-deferred** | `QUANTITY_NEGOTIATION_ENABLED=true` + client un-stub; e2e drops `PX_E2E_NEGOTIATION_SKIPPED` |
| Soliq OFD UI | **Product-deferred** | `FISCAL_PROVIDER=SOLIQ` + [`docs/SOLIQ_SANDBOX_READINESS.md`](../docs/SOLIQ_SANDBOX_READINESS.md); Pegasus branded receipts remain Wired |
| Offline POS | **Product-deferred** | Product flag + Retail OS offline phase; online-required v1 stays |

## Warehouse reverse-logistics / exceptions mobile (2026-07-31)

| Item | Status | Notes |
|------|--------|--------|
| Reverse-logistics panel + exceptions hub on warehouse mobile | **Wired** | Android + iOS: credit-note receive, exceptions triage, claims read-only, rescues UI; inbound OPEN + claim-ticket fields |

## Offline driver action queue (2026-08-01)

| Item | Status | Notes |
|------|--------|--------|
| Full offline flush orchestrator | **Wired** | Android Room + WorkManager + Sync Queue UI; iOS QueuedDriverAction + BGTask; 4.1 flush order; client_timestamp; dead-letter |
| iOS deliverOrder enqueue classifier | **Wired** | P0-4 closed 2026-08-05 |

## Retail OS (2026-08-02)

| Item | Status | Notes |
|------|--------|--------|
| Packs 0–6 backend + 3 clients | **Wired** | TEAM → ASSIST; see `docs/RETAILER_OS_E2E_MATRIX.md` |
| Control Tower retailer pulse | **Wired** | `GET /v1/retailer/control-tower/pulse`; empty or live; no `sup-demo-1` |
| Spanner migrations P0–P6 | **Pending ops** | DDL in `schema/migrations/20260802_retail_os_phase*.ddl` + `spanner.ddl` |
| Family members vs Team | Partial | Legacy family list may remain; Team is RBAC path |
| Reports inventory valuation | Partial | Qty/movements only — no COGS SoT |
| Auto-order execution worker | Deferred | Settings durable; worker not Phase 0–7 |
| Offline POS | **Product-deferred** | See exit criteria above |
| Supplier-style CT playbooks on retailer | N/A | Retailer CT is ops digest, not fleet playbooks |
