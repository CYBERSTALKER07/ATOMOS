# Forecast algorithm (§8.1)

Croston (Syntetos–Boylan), SES, and Holt–Winters baselines for `DemandForecastBaseline`.

**Next planning slice (plan, not status):** persist/show SBC class already computed at fit time — [`DEMAND_CLASS_IBP_SLICE.md`](./DEMAND_CLASS_IBP_SLICE.md). Parent o9 catalog: [`PegasusX_o9_Digital_Brain_Feature_Extraction_Integration_Blueprint.md`](./PegasusX_o9_Digital_Brain_Feature_Extraction_Integration_Blueprint.md). Do not treat those files as wired.

## Flags

| Env | Default | Meaning |
|-----|---------|---------|
| `FORECAST_ALGO_ENABLED` | off (SSMR example on) | Nightly/ops writer materializes baselines |
| `FORECAST_ALGO_REQUIRE_GATE` | off | Backtest exits non-zero unless algo beats 7-day mean by >15% WAPE on ≥80% of series |
| `FORECAST_ACCURACY_ENABLED` | separate | §8.4 accuracy table / confidence |
| `FORECAST_DEMOTE_ENABLED` | off | G6: auto-demote baselines when WAPE28 fails gate |
| `FORECAST_DEMOTE_WAPE28_MAX` | 0.45 | Demote if WAPE28 > max and sample≥14 |
| `FORECAST_SEASONAL_ESTIMATE_ENABLED` | off | YoY calendar multiplier draft suggestions (inactive overrides) |

## Accuracy publish (G6.A1)

- Nightly pass writes **WAPE** (primary) and **MAPE28** (`mean(|f−a|/a)` for a>0) to `ForecastAccuracyDaily`.
- `GET /v1/admin/planning/accuracy` returns `mape28`, `demoted` (threshold view), and demote config.
- When `FORECAST_DEMOTE_ENABLED=true` and WAPE28>max with sample≥14: set `BlockedReason=accuracy_demoted`, low confidence; forecast algo pass respects demote.

## Jobs

- `planning-forecast -mode=forecast` — CronJob `02:00 UTC` → write baselines for tomorrow
- `planning-forecast -mode=backtest` — rolling-origin WAPE vs trailing 7-day mean
- `POST /v1/admin/planning/forecast/run-once` — ADMIN ops / SSMR

When `FORECAST_ALGO_ENABLED=true`, predictive-push **does not** overwrite baselines with CartItems AVG.

## Seasonality (Theatre #8)

| Piece | Behavior |
|-------|----------|
| Builtins | `holiday_peak` ×1.35, `summer_surge` ×1.15 in `seasonalcore` (shared by planning + replenishment) |
| Overrides | `SeasonalTemplateOverrides.Multiplier` persisted on create (clamp [0.5, 2.5]; inherit builtin by `template_id`; else 1.2). `ActiveSeasonalTemplate` never hardcodes 1.2. |
| Replenishment | `Ceil(suggestedQty * seasonMul)` uses Spanner override reader when active, else builtins |
| Estimate | `POST /v1/supplier/planning/seasonal-estimate` + optional `planning-forecast` hook behind `FORECAST_SEASONAL_ESTIMATE_ENABLED` (default off) writes **inactive** draft overrides from YoY/month ratios — never auto-activates |

Holt–Winters m=7 day-of-week indices remain inside the series forecast and are **not** the annual calendar library.

Marker: `PX_E2E_SEASONAL_OVERRIDE_OK` / `_SKIPPED`.

## Classification

| Class | Rule | Model |
|-------|------|-------|
| Smooth | ADI < 1.32, CV² < 0.49 | Holt–Winters (m=7) |
| Erratic | ADI < 1.32, CV² ≥ 0.49 | SES |
| Intermittent | ADI ≥ 1.32 | Croston SBA |

Short series (<60 calendar days or <14 non-zero days): SES defaults + `BlockedReason=insufficient_history`.

## Cutover gate

```bash
FORECAST_ALGO_REQUIRE_GATE=true /usr/local/bin/planning-forecast -mode=backtest -days 90
```

Promote only when the gate passes on a supplier with sufficient COMPLETED history. Bias matters as much as WAPE — inspect backtest logs.
