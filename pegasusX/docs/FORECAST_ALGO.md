# Forecast algorithm (§8.1)

Croston (Syntetos–Boylan), SES, and Holt–Winters baselines for `DemandForecastBaseline`.

## Flags

| Env | Default | Meaning |
|-----|---------|---------|
| `FORECAST_ALGO_ENABLED` | off (SSMR example on) | Nightly/ops writer materializes baselines |
| `FORECAST_ALGO_REQUIRE_GATE` | off | Backtest exits non-zero unless algo beats 7-day mean by >15% WAPE on ≥80% of series |
| `FORECAST_ACCURACY_ENABLED` | separate | §8.4 accuracy table / confidence |

## Jobs

- `planning-forecast -mode=forecast` — CronJob `02:00 UTC` → write baselines for tomorrow
- `planning-forecast -mode=backtest` — rolling-origin WAPE vs trailing 7-day mean
- `POST /v1/admin/planning/forecast/run-once` — ADMIN ops / SSMR

When `FORECAST_ALGO_ENABLED=true`, predictive-push **does not** overwrite baselines with CartItems AVG.

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
