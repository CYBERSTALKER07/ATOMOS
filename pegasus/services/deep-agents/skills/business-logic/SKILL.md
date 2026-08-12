---
name: business-logic
description: Order/credit/payment/fiscal state machines, ADR-009, edge cases, per-role business duties.
---

# Business logic

**SoT docs:** `docs/ORDER_FLOW_AND_EDGE_CASES.md`, `docs/ROLE_CAPABILITIES_MATH_LOGIC.md`,
`docs/FEATURES_BY_APP_ROLE.md`

## Order machine (ADR-009)

- Payment capture → `FISCALIZING` → OFD SUCCESS → `COMPLETED` (+ `ORDER_FINALIZED`)
- OFD fail → `FISCAL_FAILED` → retry or admin force-complete
- **No soft** `ARRIVED` → `COMPLETED`
- Credit leave only reaches COMPLETED via `FISCALIZING` after money
- Cancel/reject/vet-reject must release inventory + payment + notifications

## Happy paths auditors must recognize

| Path | Flow |
|------|------|
| Cash COD | PENDING→…→ARRIVED→PENDING_CASH_COLLECTION→collect-cash→FISCALIZING→COMPLETED |
| Card | …→AWAITING_PAYMENT→complete/capture→FISCALIZING→COMPLETED |
| Credit leave | …→DELIVERED_ON_CREDIT (CanLeaveOnCredit)→later capture→FISCALIZING→COMPLETED |
| Preorder | SCHEDULED→AUTO_ACCEPTED?→PENDING→same spine |
| Backorder | BACKORDERED→PENDING\|SCHEDULED\|CANCELLED |

## Edge-case checklist (must not silently break)

| Edge | Required behavior | Evidence anchors |
|------|-------------------|------------------|
| Shop closed | SHOP_CLOSED_PENDING + grace (~5m) → respond / credit / cancel / resume | `worker_shop_closed.go`, retailer/supplier shop-closed routes |
| Partial offload | DeliveredQty+RemainingQty=Quantity; reasons DAMAGED/MISSING/… | `order/partial_offload.go` |
| Cash ≠ expected | CASH_SHORTFALL / CASH_OVERAGE events | `order/fiscal.go` emitCashVariance |
| Credit leave | ACTIVE profile + Available≥Total; PoD; no RiskTier Phase A | `order/credit_guard.go` |
| Inventory short | inventory_exhausted; cancel releases reserve | `inventory_reservation.go`, `inventory_release.go` |
| OFD fail / timeout | FISCAL_FAILED; 8s OFD timeout; force-complete ADMIN/WAREHOUSE_ADMIN only | `order/fiscal.go`, orderroutes |
| Buyer EHF REJECT | Parallel to COMPLETED; exception + credit note default ON | `buyer_acceptance_poller.go` |
| Claim window | COMPLETED + default 48h; photo for damage family | `claims/eligibility.go` |
| CANCEL_REQUESTED | Must exit CANCELLED or resume LOADED/IN_TRANSIT/ARRIVED | `state_machine.go` |
| Live payout no rail | fail closed ErrNoLiveRail | `payout/rail.go` |
| Cash on credit order | CollectCash pays down AR | `ar.RecordPaymentForOrder` |
| Proximity / geofence | Approach 500m; settlement proximity unlock | `proximity/geofence.go` |

## Per-role business duties (what “correct” means)

| Role | Must correctly handle |
|------|----------------------|
| Retailer | Multi-supplier checkout, doorstep cash confirm, credit/AR, claims, shop-closed respond, POS/stock |
| Supplier | Vet, dispatch, credit policy/collections, claims adjudicate, shop-closed resolve, treasury |
| Warehouse | Dispatch/WMS, delay/reject, force-complete (admin), returns/claims |
| Driver | Arrive, QR, partial offload, shop-closed, cash collect, credit leave, fiscal retry, offline sync |
| Factory | Loading bay → seal/dispatch; transfers; supply requests (loop with payload) |
| Payload | Seal/inject/reassign; inbound returns; consume factory+supplier manifests |
| Platform admin | Tenant lifecycle; dual-control money flags; no silent tenant-less money |

## Money invariants

- Amounts: int64 minor units only
- Capture/refund idempotency keys required
- Credit leave opens AR; cash collect must pay down AR (P0-1)

## Buyer EHF (parallel to COMPLETED)

- MySoliq SUCCESS stamps `BuyerAcceptanceStatus=PENDING` + deadline
- REJECT → exception ticket + credit note (default ON)
- Do not delay COMPLETED on OFD submit success

## Regulatory cross-check

Load skill `regulatory-gov` for Soliq OFD/EHF, GS1, AS2/EDI. Do not claim legal fiscal without EDS proof (P1-7).

## Evidence

`order/state_machine.go`, `order/fiscal.go`, `order/buyer_acceptance_poller.go`,
`ar/`, `payout/`, `payment/`, `claims/`, gap register money + P1-6/7 rows
