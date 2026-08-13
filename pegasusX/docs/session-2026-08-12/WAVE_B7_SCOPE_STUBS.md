# Wave B7 — Scope & stubs (fail-closed)

**Date:** 2026-08-13  
**Master:** [`BACKEND_PARITY_MASTER.md`](./BACKEND_PARITY_MASTER.md)

## Goal

Eliminate **fake success** and **scope holes** left after B1–B6 so mis-bootstrap or missing home-node never looks like a durable mutation.

## Changes

### R-P0-3 Retailer stubs
- `HandleCancelOrder` / `HandleRequestCancel` → **503** `order_service_unwired` when OrderService path not mounted
- `HandleCreateOrder` / `HandleUnifiedCheckout` (dead fallbacks) → **503** (never fabricate order ids)

### D-P0-6 / D-P0-7 Driver stubs
- `HandleDriverDepart` when `depart` nil → **503** `depart_unwired` (+ release idempotency claim)
- `HandleDriverReturnComplete` when `returnComplete` nil → **503** `return_complete_unwired` (no in-memory availability flip)

### WH-P0-3 Reverse logistics receive
- Role gate: `WAREHOUSE` + `WAREHOUSE_ADMIN`
- Routes: `RequireWarehouseOpsScope`
- Body `warehouse_id` may only restate JWT home node; mismatch → **403** `warehouse_scope_violation`

### WH-P0-4 Stocklots by-id membership
- `assertResourceWarehouse` on lot/wave/cycle-count/adjustment by-id read + mutators
- Cross-warehouse resource → **403** `warehouse_scope_forbidden`

### WH-P0-5 Returns inbound scan outbox
- Event `RETURN_SCAN_RECEIVED` in same RW txn as `ReceivedQty` bump
- Dispatcher fans via `handleReturnGateEvent` (supplier + warehouse + payload rooms)

### FAC-P0-3 Factory setup outbox
- Setup insert/update buffers `FACTORY_CREATED` / `FACTORY_LOCATION_UPDATED` on same Spanner RW txn

### PL-P0-6 Payload warehouse scope
- Spanner `ListManifests` filters by repo warehouse (NULL/empty WH still visible)
- In-memory list filters by JWT home-node warehouse
- Detail / seal / start-loading: both JWT and row WH non-empty and differ → **403**

## Residual (out of B7)

- Full dual factory/payload table merge (M-P1-8)
- POS single-txn (R-P1-4)
- Platform governance audit atomicity
- Stocklots package-wide remaining silent sites beyond B2 WMS events (if any)

## Verification

```bash
cd apps/backend-go
go test ./driver/ ./retailer/ ./stocklots/ ./returns/ ./creditnote/ ./payload/ ./factory/ ./kafka/ -count=1
go build -o /tmp/pegasusx-backend .
```
