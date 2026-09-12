# Safety stock (§8.2)

Service-level safety stock for warehouse replenishment reorder points.

## Formula

When `SAFETY_STOCK_V2_ENABLED=true`:

\[
SS = z_\alpha \cdot \sqrt{L \cdot \sigma_d^2 + \bar{d}^2 \cdot \sigma_L^2}
\]

\[
\text{ReorderPoint} = \bar{d} \cdot L + SS
\]

When the flag is off, the engine keeps the legacy buffer `burn · L · 1.15`.

## Inputs

| Symbol | Source |
|--------|--------|
| \(\bar{d}\) | Avg last-7 `DemandForecastBaseline.BaselineQty` when present; else 7-day burn |
| \(\sigma_d\) | Stdev of `ForecastAccuracyDaily.SignedError` (28d, min 7 samples); else `max(d̄·0.25, 1)` labeled `sigma_d_assumed` |
| \(L\), \(\sigma_L\) | `ReplenishmentPolicies.LeadTimeDays` / `LeadTimeSigmaDays`; overwritten by observed transfer lead times when ≥10 samples with `ReceivedAt` |
| \(z_\alpha\) | From `TargetServiceLevel` (default 0.98 → 2.054) |

## Policy knobs

`GET` / `PATCH` `/v1/supplier/replenishment/policies`:

- `target_service_level`
- `lead_time_days`
- `lead_time_sigma_days` (assumed until observed history exists)

Supplier portal **Operations → Replenishment policies** edits these. Warehouse clients consume engine `DemandBreakdown` only (no second policy UI).

## Lead-time observation

`FactoryInternalTransfers.ReceivedAt` is set on warehouse receive and factory INTERNAL create-as-RECEIVED. Observed L/σ_L replace policy values when sample size ≥10 (INTERNAL / &lt;0.5d durations ignored).

## In-transit

`InTransitQty` sums open transfer quantities linked via `SourceInsightId` → `ReplenishmentInsights.SuggestedQuantity` before `computeSuggestedQty`.

## Retailer reorder suggestions

[`RunBatch`](../apps/backend-go/replenishment/reorder_suggestion_batch.go) writes `ReorderSuggestions.SafetyStock`:

| Flag | Safety stock | Lead days |
|------|--------------|-----------|
| off | `demand · 0.15` | hardcoded `2` |
| on | `SafetyStockUnits` (same formula; σ_d from residuals or assumed) | supplier `ReplenishmentPolicies` / observed L |

`SuggestedQty = demand·L + SafetyStock − stock − inFlight` (unchanged). Exposed as `safety_stock` on supplier `GET /v1/replenishment/suggestions` and retailer `GET /v1/retailer/reorder-suggestions`.

## Flags / ops

| Env | Default | Meaning |
|-----|---------|---------|
| `SAFETY_STOCK_V2_ENABLED` | off (SSMR example on; K8s ConfigMap `false`) | Use service-level SS |
| `SAFETY_STOCK_REPLAY_REQUIRE_GATE` | off | Fail CLI/admin when v2 cycle SL &lt; target−2pp or avg OH &gt; legacy×1.02 |

## 90-day fill-rate replay

Offline simulation of inventory under **legacy** vs **v2** ROP on COMPLETED order demand (no Spanner writes).

### Model

1. Load SKU-day demand from COMPLETED `LineItemsJson` (lookback default 90d).
2. Warmup 28 days (not scored): trailing \(\bar{d}\), residual \(\sigma_d\) (prefer `ForecastAccuracyDaily.SignedError` when present).
3. Two parallel continuous-review sims: order up to ROP when `IP ≤ ROP`; receipts arrive after `round(L)` days.
4. Opening stock: `ceil(mean(warmup)·L·1.5)` (same for both policies).

### Metrics

| Metric | Definition |
|--------|------------|
| `unit_fill_rate` | Σ fulfilled / Σ demand (scored window) |
| `cycle_service_level` | fraction of demand-days with unmet = 0 |
| `avg_on_hand` | mean end-of-day on-hand |
| `sku_count` | series with ≥45 calendar days and ≥14 nonzero days |

### How to run

```bash
# CLI
go run ./cmd/safety-stock-replay -supplier-id=... -days=90

# Optional hard gate
SAFETY_STOCK_REPLAY_REQUIRE_GATE=true go run ./cmd/safety-stock-replay -supplier-id=...

# ADMIN (JWT RoleAdmin)
POST /v1/admin/planning/safety-stock/replay?supplier_id=...&days=90
```

Series eligibility: ≥45 calendar days, ≥14 nonzero demand days. Promote v2 when gate passes on a representative supplier.

## Validation

- Unit tests: `replenishment/safety_stock_test.go`, `replenishment/fill_rate_replay_test.go`, `replenishment/reorder_suggestion_batch_test.go`
- SSMR: `PX_E2E_SAFETY_STOCK_OK` / `_SKIPPED`, `PX_E2E_SAFETY_STOCK_REPLAY_OK` / `_SKIPPED`

## Out of scope

- §8.3 inventory-grounded auto-order / shadow mode
- Nightly CronJob for replay (ad-hoc ops tool only)
- Per-retailer service-level knobs (supplier policy applies)
