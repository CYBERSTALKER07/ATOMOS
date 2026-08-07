# Forecast algorithm (§8.1)

Croston (Syntetos–Boylan), SES, and Holt–Winters baselines for `DemandForecastBaseline`.

## Flags

| Env | Default | Meaning |
|-----|---------|---------|
| `FORECAST_ALGO_ENABLED` | off (SSMR example on) | Nightly/ops writer materializes baselines |
| `FORECAST_ALGO_REQUIRE_GATE` | off | Backtest exits non-zero unless algo beats 7-day mean by >15% WAPE on ≥80% of series |
| `FORECAST_ACCURACY_ENABLED` | separate | §8.4 accuracy table / confidence |
| `FORECAST_SEASONAL_ESTIMATE_ENABLED` | off | YoY calendar multiplier draft suggestions (inactive overrides) |

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
