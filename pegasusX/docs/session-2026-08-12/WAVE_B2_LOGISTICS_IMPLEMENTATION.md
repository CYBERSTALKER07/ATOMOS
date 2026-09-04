# Wave B2 — Logistics truth implementation
> **POINT-IN-TIME SNAPSHOT (2026-08-12) — do not treat as current status.**
> Re-verify any claim against live code before acting. Multiple ecosystem hardening phases have shipped since this audit.


**Date:** 2026-08-13  
**Master:** [`BACKEND_PARITY_MASTER.md`](./BACKEND_PARITY_MASTER.md)

## Changes

### M-P0-3 WMS / inventory outbox
- New events: `INVENTORY_QUANTITY_UPDATED`, `INVENTORY_POLICY_UPDATED`, `WMS_PUTAWAY`, `WMS_PICK_CONFIRMED`, `WMS_CYCLE_APPROVED`, `WMS_TEMPERATURE_BREACH`
- `stocklots` putaway / pick confirm / cycle approve / temp excursion emit in same Spanner txn (`stocklots/outbox_emit.go`)
- Warehouse absolute inventory + policy PATCH no longer pass `emit=nil`
- Dispatcher `handleWMSStockEvent` → supplier + warehouse rooms

### M-P0-7 Payload routes + seal
- `main.go` mounts **`payloaderoutes`** (ws-session, ship-units, labels) instead of thin `payloaderroutes`
- `POST /v1/payload/seal` fixed: no mutex across `apply` (deadlock); **requires `manifest_id`**; order-only silent seal forbidden
- Fleet reassign emits `ORDER_REASSIGNED` outbox (was `emit=nil`)
- Seal events include `WarehouseID` from JWT home-node when present

### M-P0-8 Factory home-node
- `resolveFactoryNode(ctx)` prefers `FactoryScope` / JWT `HomeNodeID` over bootstrap `FACTORY_DEMO_ID`
- Used for WS room + manifest outbox `FactoryID`

### M-P0-9 Memory RunTx fail-closed
- Payload + factory in-memory repos **error** under `PEGASUSX_ENV=production|prod|ssmr` or `REQUIRE_INFRA_ADAPTERS=true`
- Local still runs mutate + discard emit (no silent skip of `fn`)

### M-P0-15 Supply-transfer arrive
- Driver arrive writes Spanner **and** `SUPPLY_TRANSFER_ARRIVED` outbox + warehouse hub broadcast

### M-P1-8/9
- Manifest sealed events carry `warehouse_id` when JWT warehouse home-node is set
- Factory/payload remain separate aggregates (documented; not merged this wave)

## Verification

```bash
cd apps/backend-go
go test ./payload/ ./factory/ ./stocklots/ ./warehouse/ ./kafka/ ./bootstrap/ -count=1
go build -o /tmp/pegasusx-backend .
```

Green as of 2026-08-13.
