# Parity ledger (intentional divergences)

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
| Spanner DDL applied on live SSMR | **Pending ops** | Migration `20260729_shop_closed_proximity_partial.ddl` |
| Offline action queue (full driver offline flush orchestrator) | **Wired** | Android Room/WorkManager + iOS QueuedDriverAction/BGTask; Sync Queue UI; 4.1 order; dead-letter |
| AUTHORIZE_BYPASS photo capture on retailer | Partial | Code requires photo_url; camera UX not expanded on all clients |

See: `docs/big-platform-baseline/last-mile/`.

## Quantity negotiations (product-disabled)

| Item | Status | Notes |
|------|--------|--------|
| Driver quantity negotiation ecosystem | **Product-deferred** | `quantityNegotiationDisabled = true` → 410 propose/resolve, empty pending list, sweeper no-op; clients stubbed; e2e `PX_E2E_NEGOTIATION_SKIPPED`. **Not** a substitute for claims / shop-closed / missing-items / partial offload. |

## Warehouse reverse-logistics / exceptions mobile (2026-07-31)

| Item | Status | Notes |
|------|--------|--------|
| Reverse-logistics panel + exceptions hub on warehouse mobile | **Wired** | Android + iOS: credit-note receive, exceptions triage, claims read-only, rescues UI; inbound OPEN + claim-ticket fields |

## Offline driver action queue (2026-08-01)

| Item | Status | Notes |
|------|--------|--------|
| Full offline flush orchestrator | **Wired** | Android Room + WorkManager + Sync Queue UI; iOS QueuedDriverAction + BGTask; 4.1 flush order; client_timestamp; dead-letter |

## Retail OS (2026-08-02)

| Item | Status | Notes |
|------|--------|--------|
| Packs 0–6 backend + 3 clients | **Wired** | TEAM → ASSIST; see `docs/RETAILER_OS_E2E_MATRIX.md` |
| Control Tower retailer pulse | **Wired** | `GET /v1/retailer/control-tower/pulse`; empty or live; no `sup-demo-1` |
| Spanner migrations P0–P6 | **Pending ops** | DDL in `schema/migrations/20260802_retail_os_phase*.ddl` + `spanner.ddl` |
| Family members vs Team | Partial | Legacy family list may remain; Team is RBAC path |
| Reports inventory valuation | Partial | Qty/movements only — no COGS SoT |
| Auto-order execution worker | Deferred | Settings durable; worker not Phase 0–7 |
| Offline POS | Deferred | Online-required v1 |
| Supplier-style CT playbooks on retailer | N/A | Retailer CT is ops digest, not fleet playbooks |

