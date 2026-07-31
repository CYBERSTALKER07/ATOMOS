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
| Offline action queue (full driver offline flush orchestrator) | **Deferred** | Idempotency keys + client_timestamp fields present; full queue UX next |
| AUTHORIZE_BYPASS photo capture on retailer | Partial | Code requires photo_url; camera UX not expanded on all clients |

See: `docs/big-platform-baseline/last-mile/`.

## Quantity negotiations (2026-07-31)

| Item | Status | Notes |
|------|--------|--------|
| Driver quantity negotiation ecosystem | **Product-deferred** | Backend list empty; portal page disabled; native nav stubs hidden / empty-state only |

## Warehouse reverse-logistics / exceptions mobile (2026-07-31)

| Item | Status | Notes |
|------|--------|--------|
| Reverse-logistics panel + exceptions hub on warehouse mobile | **Deferred** | Portal-first; inbound returns wired on Android/iOS |

## Offline driver action queue (unchanged)

| Item | Status | Notes |
|------|--------|--------|
| Full offline flush orchestrator | **Deferred** | Idempotency keys present; UX after P0–P4 closure |
