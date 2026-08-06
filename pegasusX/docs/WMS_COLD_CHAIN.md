# WMS cold chain (§8.7 Gate 4 PR-6)

Temperature ingest with excursion quarantine of AVAILABLE lots for manifest products.

**Flag:** `WMS_COLD_CHAIN_ENABLED` (default `false`)

**DDL:** [`schema/migrations/20260806_wms_cold_chain.ddl`](../apps/backend-go/schema/migrations/20260806_wms_cold_chain.ddl)

## HTTP

| Method | Path |
|--------|------|
| GET | `/v1/warehouse/ops/temperature-readings?manifest_id=` |
| POST | `/v1/warehouse/ops/temperature-readings` `{ manifest_id, temp_c, sensor_id?, min_c?, max_c?, lat?, lng? }` |

Default band `[0, 8]°C` when min/max omitted. Excursion → lot `QUARANTINE` + V2 roll-up.

## Residual

- Driver Bluetooth logger ingest (Phase-B)
- Condition-report auto-raise `TEMPERATURE_BREACH` inbox event
