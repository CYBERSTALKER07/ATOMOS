# Gate 4 — WMS DDL + flag ops checklist

Owner ops apply (Spanner + env). Code is already in-tree for Waves 1A–1C.

## Migrations (order)

```bash
# From pegasusX/apps/backend-go — uses cmd/apply-migration + SchemaMigrations
go run ./cmd/apply-migration --ddl schema/migrations/20260806_wms_locations_lots.ddl
go run ./cmd/apply-migration --ddl schema/migrations/20260806_wms_pick_waves.ddl
go run ./cmd/apply-migration --ddl schema/migrations/20260806_wms_cycle_counts.ddl
# After Gate 4 PR-6 lands:
# go run ./cmd/apply-migration --ddl schema/migrations/20260806_wms_cold_chain.ddl
```

Also keep [`schema/spanner.ddl`](../apps/backend-go/schema/spanner.ddl) as the greenfield source of truth.

## Flags (SSMR / staging)

| Flag | Wave | Default | Notes |
|------|------|---------|--------|
| `WMS_LOTS_ENABLED` | 1A | false | FEFO + putaway |
| `WMS_PICK_WAVES_ENABLED` | 1B | false | Pick + hard seal gate |
| `WMS_CYCLE_COUNTS_ENABLED` | 1C/PR-4 | false | Counts + adjustments |
| `WMS_PICK_SSHAPE_ENABLED` | PR-5 | false | Zone serpentine + LIFO seq |
| `WMS_SEAL_SOFT_WARN` | PR-5 | false | Soft-warn seal (hard block default) |
| `WMS_COLD_CHAIN_ENABLED` | PR-6 | false | Temperature ingest + quarantine |

See [`.env.example`](../.env.example) / [`.env.ssmr.example`](../.env.ssmr.example).

## Verify

- `GET /v1/warehouse/ops/bins` → `lots_enabled`
- `GET /v1/warehouse/ops/pick-waves` → not `wms_pick_waves_disabled` when flag on
- `GET /v1/warehouse/ops/cycle-counts` → not `wms_cycle_counts_disabled` when flag on
- SSMR markers: `PX_E2E_WMS_*`

**This checklist does not replace live apply** — run migrations against the target Spanner database with project credentials.
