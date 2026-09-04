# Gate 4 WMS — scanning throughput notes (§8.7 / §8.8)

## Backend (closed 2026-08-06)

- `GET /v1/warehouse/ops/inventory-reconcile` — V2 vs AVAILABLE lot sum drift
- Lots path remains FEFO via `stocklots` when `WMS_LOTS_ENABLED`
- **Forbid non-rollup V2 quantity writers:** `inventory.CreditSupplierInventoryV2InTxn` returns `ErrLotsEnabledDirectV2` when lots on; credits go through `stocklots.CreditViaDefaultPutawayInTxn` / `PutawayInTxn` → `RollupInventoryV2InTxn`
- Absolute `UpdateInventoryQuantity`, supplier import QoH apply, and lot release without `order_id` fail closed when lots on
- **Still deferred:** serial / SerializedUnits tracking

Ops checklist: [`WMS_GATE4_OPS.md`](./WMS_GATE4_OPS.md)

## Native scan UX (P2 #9)

Shared barcode kits (`packages/mobile-android-barcode-scanner`, `packages/mobile-ios-barcode`):
- EAN-8/13 only (Android `BarcodeScannerOptions`; iOS already restricted)
- 300 ms same-code debounce (was 1.5 s)
- Torch toggle + keyboard-wedge input for Zebra / hardware scanners
- Warehouse pick-wave confirm wired to camera/wedge (increment qty, no toggle)

Residual follow-ups: DataWedge intent profile docs per device fleet; portal remains text/ops UI.
