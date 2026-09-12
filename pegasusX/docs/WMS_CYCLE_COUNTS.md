# WMS cycle counts (§8.7 Wave 1C + Gate 4 PR-4)

Warehouse cycle counts and inventory adjustments with **apply-on-approve**.

**Flag:** `WMS_CYCLE_COUNTS_ENABLED` (default `false`)

**Package:** [`apps/backend-go/stocklots`](../apps/backend-go/stocklots) (`counting.go`)

**DDL:** [`schema/migrations/20260806_wms_cycle_counts.ddl`](../apps/backend-go/schema/migrations/20260806_wms_cycle_counts.ddl)

## Behavior

1. Create OPEN count (expected from lot QoH or override)
2. Submit → variance; PENDING `InventoryAdjustments` when ≠ 0
3. **Approve** (admin) → mutate `StockLots` QoH by delta + V2 roll-up
4. Reject → `REJECTED` without lot change
5. `POST .../cycle-counts/enqueue-abc` — top-N product/location OPEN counts
6. `GET .../inventory-accuracy` — `1 − Σ|variance|/Σexpected`

## Clients

- Portal `/cycle-counts` (approve button)
- Android / iOS Transfer actions → cycle counts

## Residual

- Richer ABC cadence job (value-based A/B/C calendar)
- Nightly off-cycle triggers
