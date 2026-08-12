# SSMR smoke: Auto-order place + L3 DDL (2026-08-02)


> **HISTORICAL / FROZEN — do not plan from this file alone.**
> Current SoT: [`../docs/DOCS_SOURCE_OF_TRUTH.md`](../docs/DOCS_SOURCE_OF_TRUTH.md) · [`../docs/PROD_READINESS_SEQUENCE.md`](../docs/PROD_READINESS_SEQUENCE.md) · [`../context/current_status.md`](../context/current_status.md).
> Body is a point-in-time snapshot; re-verify any claim against code before acting.

## Environment
- Project: `pegasus-503013`
- Cluster: `pegasusx-ssmr-gke` / ns `pegasusx-ssmr`
- Spanner: `pegasusx-ssmr-spanner` / `pegasusx-ssmr-db`
- API: `https://api-ssmr.pegasusx.app`
- Image: `asia-south1-docker.pkg.dev/pegasus-503013/pegasusx-ssmr-images/backend-go:ssmr-ao-place-a1fafaa0-sup`
- Flag: `AUTO_ORDER_PLACE_ENABLED=true`

## Phase A — DDL (applied)
| Migration | Result |
|-----------|--------|
| `20260802_retail_os_sell_through.ddl` | Applied (`RetailerSellThroughDaily`) |
| `20260802_retail_os_sell_through_sources.ddl` | Applied (`SourcesJson`, velocities) |
| `20260802_retail_os_auto_order_bucket.ddl` | Applied |
| `20260809_retailer_auto_order_runs.ddl` | Applied |

## Hotfix found during smoke
1. **Retailer `Service` missing Spanner client in bootstrap** — suggestions/bucket/runs/locations Spanner paths were memory-only. Fixed: `Spanner: spannerClient` on `retailer.NewService`.
2. **Supplier resolution** for first-time SKUs skipped with `missing_supplier`. Fixed: Products.SupplierId lookup + seed `supplierID` fallback.

## Phase G — Place smoke (PASS)
| Check | Result |
|-------|--------|
| Draft / place with flag off | `place_disabled_set_AUTO_ORDER_PLACE_ENABLED` (no create) |
| Place with flag on | `status=OK`, `message=orders_placed`, `placed_lines=1` |
| Order | `ord_1785643051460043917` supplier `sup_61d822c6ab9714ca11f20db9` total 100000 |
| Second place same day | `already_processed_bucket` for SSMR-SKU-1 |
| Durable runs API | Returns place audit with `placed_orders` |
| Retailer | `ret_1785640864428777396` geo 41.2995/69.2401 H3 `8920a52d203ffff` |

## Not fully re-verified in this transcript
- Direct Spanner `OrderSource` column read (confirm with SQL if needed)
- Client UI smoke (desktop/mobile) — API path proven

## Rollback
```bash
kubectl -n pegasusx-ssmr set env deployment/backend-go AUTO_ORDER_PLACE_ENABLED=false
kubectl -n pegasusx-ssmr set env deployment/backend-go-worker AUTO_ORDER_PLACE_ENABLED=false
```

## Spanner verification (post-smoke)

```
OrderId                  SupplierId                    OrderSource  Status   TotalMinor
ord_1785643051460043917  sup_61d822c6ab9714ca11f20db9  AUTO_ORDER   PENDING  100000
```

`RetailerAutoOrderRuns` has `place` run `status=OK` `PlacedLines=1` `message=orders_placed`.

Subsequent place runs: `already_processed_bucket`.

## Live E2E verdict: **PASS** (API/backend path)

