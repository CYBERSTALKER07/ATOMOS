---
name: business-logic
description: Order/credit/payment/fiscal state machines, ADR-009, invalid transitions, money invariants.
---

# Business logic

## Order machine (ADR-009)

- Payment capture → `FISCALIZING` → OFD SUCCESS → `COMPLETED` (+ `ORDER_FINALIZED`)
- OFD fail → `FISCAL_FAILED` → retry or admin force-complete
- No soft `ARRIVED` → `COMPLETED`
- Cancel/reject/vet-reject must release inventory + payment + notifications

## Money invariants

- Amounts: int64 minor units only
- Cash collect shortfall/overage → variance events
- Credit leave opens AR; cash collect must pay down AR (P0-1)
- Live payout without live rail must fail closed (P0-2)

## Buyer EHF (parallel to COMPLETED)

- MySoliq SUCCESS stamps `BuyerAcceptanceStatus=PENDING` + deadline
- REJECT → exception ticket + credit note (default ON)
- Do not delay COMPLETED on OFD submit success; track buyer acceptance separately

## Evidence

`order/state_machine.go`, `order/fiscal.go`, `order/buyer_acceptance_poller.go`,
`ar/`, `payout/`, `payment/`, `docs/ORDER_FLOW_AND_EDGE_CASES.md`
