# WMS cold chain (§8.7 Gate 4 PR-6 + theatre #12 breach)

Temperature ingest with excursion quarantine of AVAILABLE lots for manifest products, product band hydrate, and system `TEMPERATURE_BREACH` condition reports.

**Flag:** `WMS_COLD_CHAIN_ENABLED` (default `false`)

**DDL:** [`schema/migrations/20260806_wms_cold_chain.ddl`](../apps/backend-go/schema/migrations/20260806_wms_cold_chain.ddl)

## HTTP

| Method | Path |
|--------|------|
| GET | `/v1/warehouse/ops/temperature-readings?manifest_id=` |
| POST | `/v1/warehouse/ops/temperature-readings` `{ manifest_id, temp_c, sensor_id?, min_c?, max_c?, lat?, lng? }` |

### Band resolution

1. If both `min_c` and `max_c` are present in the body → use as override.
2. Else hydrate from `Products.StorageTempMinC/MaxC` for cold SKUs on the manifest (intersection / tightest band).
3. Else fallback `[0, 8]°C` chilled default.

### Excursion

`temp_c` outside band →:

1. Persist `TemperatureReadings`
2. Quarantine AVAILABLE lots for manifest products + V2 roll-up
3. Auto-raise `TEMPERATURE_BREACH` (`ReportedBy=wms-cold-chain`, role `SYSTEM`) per reportable manifest order in the **same** Spanner RW txn, with `ORDER_CONDITION_REPORTED` outbox including `supplier_id` / `retailer_id` for inbox fanout
4. Idempotent: skips if an open SYSTEM cold-chain breach already exists for that order

Markers: `PX_E2E_WMS_COLD_CHAIN_OK|_SKIPPED`, `PX_E2E_WMS_COLD_CHAIN_BREACH_OK|_SKIPPED`

## Residual (Phase-B)

- Driver Bluetooth logger ingest
- Cumulative minutes outside band per lot
