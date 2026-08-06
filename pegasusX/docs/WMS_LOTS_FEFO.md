# WMS lots + FEFO (§8.7 Wave 1A)

Warehouse bin locations, stock lots with expiry, and FEFO/FIFO allocation with shelf-life gating.

**Flag:** `WMS_LOTS_ENABLED` (default `false`) — when off, bag-of-SKU `SupplierInventoryV2` reserve/credit is unchanged.

**Package:** [`apps/backend-go/stocklots`](../apps/backend-go/stocklots) (kept separate from the `warehouseops` facade alias to avoid import cycles).

**DDL:** [`schema/migrations/20260806_wms_locations_lots.ddl`](../apps/backend-go/schema/migrations/20260806_wms_locations_lots.ddl)

## Tables

| Table | Role |
|-------|------|
| `WarehouseLocations` | Bin/slot: zone, aisle, rack, level, bin, `LocationType` (`PICK`\|`BULK`\|`STAGE`\|`QUARANTINE`), `PickSequence` |
| `StockLots` | Lot QoH/reserved, expiry, location, status |
| `OrderLotReservations` | Per-order lot reservation lines (release on cancel) |

Shelf-life knobs: `Products.MinShelfLifeDays`, `Retailers.MinShelfLifeDays` (retailer wins).

## Behavior (flag on)

1. **Putaway** `POST /v1/warehouse/ops/lots/putaway` — credits a lot into a bin; perishables require `expiry_date`; quarantine bins → lot `QUARANTINE` (excluded from available).
2. **Roll-up** — `SupplierInventoryV2` QoH/Reserved recomputed from `AVAILABLE` lots in the same RW txn.
3. **Reserve** — FEFO by `ExpiryDate` for perishables (with min remaining shelf life vs expected delivery); FIFO by `ReceivedAt` otherwise.
4. **Transfer receive** — when lots enabled, receive lines putaway (default location `recv-default` STAGE bin if omitted).

## HTTP

| Method | Path |
|--------|------|
| GET/POST | `/v1/warehouse/ops/bins` |
| GET/PATCH | `/v1/warehouse/ops/bins/{locationID}` |
| GET | `/v1/warehouse/ops/lots` |
| GET | `/v1/warehouse/ops/lots/{lotID}` |
| POST | `/v1/warehouse/ops/lots/putaway` |

Inventory list includes `lots_enabled`.

## Clients

- Warehouse portal: `/bins` (Bins & lots)
- Android / iOS: Transfer actions → WMS putaway fields
- `@pegasusx/types` + `@pegasusx/api-client` methods

## SSMR

`PX_E2E_WMS_PUTAWAY_OK|_SKIPPED`, `PX_E2E_WMS_FEFO_OK|_SKIPPED` (skip when API returns lots disabled / putaway fails).

## Residual (later §8.7)

- Pick waves → **Wave 1B** — [`WMS_PICK_WAVES.md`](./WMS_PICK_WAVES.md)
- Cycle counts → **Wave 1C stub** — [`WMS_CYCLE_COUNTS.md`](./WMS_CYCLE_COUNTS.md); residual ABC + apply-on-approve
- S-shape / LIFO load sequencing
- Cold-chain temperature ingestion
- Forbid direct `SupplierInventoryV2` writes outside roll-up when flag on
