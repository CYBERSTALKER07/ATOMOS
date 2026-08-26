# E2 — Per-supplier delivery perimeter (design)

**Status:** IMPLEMENTED (2026-08-20)  
**Implementation:** `warehouse/perimeter.go` (`PublishSupplierPerimeter`, `CheckSupplierPerimeter`) + `warehouse/perimeter_handlers.go` (`HandlePublishPerimeter`)  
**Route:** `POST /v1/warehouses/publish-perimeter`  
**Checkout enforcement:** `order/warehouse_resolver_spanner.go` checks `SIsMember` against `perimeter:supplier:{id}` before Spanner coverage query  
**Tests:** `warehouse/perimeter_test.go` (miniredis-backed)  
**Legacy reads:** `ssmr:delivery_perimeter` still present in `retailer/proximity_service.go` — migration not yet cutover

## Goal

Second supplier onboarding must not share a single Redis SISMEMBER set. Each supplier publishes its own H3 cell membership; checkout asserts against the order’s `supplier_id`.

## Key naming

| Purpose | Key |
|---------|-----|
| Membership set | `perimeter:supplier:{supplier_id}` |
| Compacted (debug/transport) | `perimeter:supplier:{supplier_id}:compacted` |
| Legacy (SSMR single-tenant) | `ssmr:delivery_perimeter` (+ `:compacted`) |

Helpers (pure; not wired to order-create yet):

```go
PerimeterKeyForSupplier(supplierID)           // perimeter:supplier:{id}
PerimeterCompactedKeyForSupplier(supplierID)  // …:compacted
```

## Migration window

1. **Dual-write (publish):** zone publish / topology jobs write both legacy global key and per-supplier key for the active SSMR supplier.
2. **Dual-read (checkout):** `AssertInPerimeter(supplierID, cell)` tries `perimeter:supplier:{id}` first; on missing key, fall back to `ssmr:delivery_perimeter` (fail-closed if neither exists).
3. **Cutover:** after smoke (two suppliers, two keys, A≠B), remove legacy write+read and delete global keys.

Do **not** switch production reads in this tranche — avoids breaking single-tenant SSMR mid-flight.

## Checkout / order-create contract

- Caller must pass order `supplier_id` into perimeter assert (today many paths use the global key only).
- Signature sketch:

```text
AssertInPerimeter(ctx, supplierID, h3Cell) error
  key := PerimeterKeyForSupplier(supplierID)
  if Exists(key) → SISMEMBER(key, cell)
  else if Exists(legacy) → SISMEMBER(legacy, cell)  // migration only
  else → ErrPerimeterUnavailable / zone_miss
```

## Publish API sketch

- Input: supplier topology polygon / center+radius / precomputed H3 cells.
- Action: `ReplaceSet(PerimeterKeyForSupplier(id), cells, ttl)` + compacted companion.
- Auth: supplier admin / warehouse topology role; never write another supplier’s key.
- Idempotent replace (full set swap), same as today’s global publish.

## Smoke exit (EH1.1)

1. Supplier A and B publish distinct perimeters.
2. Redis keys `perimeter:supplier:A` ≠ `perimeter:supplier:B` (cell membership differs).
3. Retailer in A-only cell: order for A succeeds; order for B → zone_miss.
4. Marker (future): `PX_E2E_PERIMETER_PER_SUPPLIER_OK`.

## Out of scope here

- Full EH1.1 UI / dual-supplier cutover
- Control Tower playbooks (E12)
- Changing SSMR default topology publish paths
