# Claim → Store Stock Quarantine Bridge (L7)

## Purpose

When a retailer has already received goods into **store ledger** (`STORE_STOCK`), filing a claim must move claimed qty into **QUARANTINE** so sellable on-hand is honest. Warehouse reverse logistics remains the supplier/WH SoT for inbound dock.

## Flow

1. **Claim FILEd** + receive session exists for order → `HoldForClaim` → FLOOR/BACKROOM → QUARANTINE (`CLAIM_HOLD` movement).
2. **Claim APPROVED** → reverse ticket / RETURN_TO_SUPPLIER (−QUARANTINE) or await WH ack.
3. **Claim REJECTED** → restore QUARANTINE → sellable or WASTE per resolution.

## Anchors

- Port: `claims.StoreStockClaimPort` in `apps/backend-go/claims/service.go`
- Movements: `MoveClaimHold` in `apps/backend-go/retailer/store_stock.go`
- Event: `STORE_STOCK_CLAIM_HOLD`
- E2E marker: `PX_E2E_CLAIM_STORE_QUARANTINE_OK` (soft skip if APIs unavailable)

## Rules

- Claim **before** store receive → no store movement (WH reverse only).
- Partial claim → hold only claimed lines.
- Idempotent approve → one movement set per claim line.
