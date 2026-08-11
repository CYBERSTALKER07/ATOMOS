# Live Migration Runbook — Domain 1.3 (Tenancy NOT NULL + Payout Rail)

Apply these to the **live** Spanner database, in order, after preconditions are
met. Each uses the existing `scripts/apply_spanner_migration.sh` harness
(`DDL_FILE=... ./scripts/apply_spanner_migration.sh`). For live GCP, unset
`SPANNER_EMULATOR_HOST` and authenticate with `spanner.databases.updateDdl`.

## Precondition: zero-NULL backfill
Run the outbox backfill until it reports zero remaining rows, then verify:

```sql
SELECT COUNT(*) FROM OutboxEvents WHERE SupplierId IS NULL OR SupplierId = "";
-- must be 0 before the NOT NULL migration
```

## Migrations (in order)

1. `20260811_outbox_supplier_id.ddl` — OutboxEvents.SupplierId column (if not already present).
2. `20260819_outbox_supplier_id_not_null.ddl` — `ALTER COLUMN SupplierId NOT NULL`. **Blocked** until backfill = 0.
3. `20260819_route_performance_supplier_id.ddl` — RoutePerformanceAnalytics.SupplierId + index.
4. `20260819_demand_analytics_supplier_id.ddl` — DemandSignals/DemandAdjustments.SupplierId + shadow index.
5. `20260811_payout_rail_reference.ddl` — PayoutBatches.RailReference (additive, no precondition).

## Verify after each
The harness runs `--verify`. For the NOT NULL step, additionally confirm the
outbox writer is stamping `ResolveSupplierID` (incl. the `_platform` sentinel)
before flipping NOT NULL, or live writes will fail closed.

## Rollback
NOT NULL drops and additive columns are low-risk; to back out the rail column:
`ALTER TABLE PayoutBatches DROP COLUMN RailReference;` (only if no live rail
has recorded references).
