# G6 Proof — Brain quality

## Tests

```text
go test ./demand/ ./planning/ ./replenishment/ ./dispatch/ ./warehouse/ -count=1
# ok

cd services/optimizer-core/server-rust && cargo test greedy_ -- --nocapture
# greedy_vrp_never_reports_optimal ok
# greedy_cpsat_never_reports_optimal ok
```

## Evidence

| ID | Evidence |
|----|----------|
| G6-A2 | `demand/scope_match.go` + `worker_sensing.go` REGION/CITY fail-closed; tests CITY not global |
| G6-A1 | `planning/accuracy.go` MAPE28 + `FORECAST_DEMOTE_*`; list API `mape28`/`demoted`; baseline `accuracy_demoted` |
| G6-C1 | Contract `GREEDY_ASSIGN` alias; Rust CPSAT doc + status test never OPTIMAL; cold-chain score −1 |
| G6-B1 | `replenishment/mei_engine.go` `selectTransfersCostAware` + `meio_solver: cost_aware_v2` |
| G6-D1 | `dispatch/score.go` RoadKm + `matrix_source` haversine\|osrm; `DISPATCH_SCORE_USE_OSRM` |

## Flags (default fail-safe)

| Flag | Default | Effect |
|------|---------|--------|
| `FORECAST_DEMOTE_ENABLED` | false | Auto-demote baseline when WAPE28>max |
| `FORECAST_DEMOTE_WAPE28_MAX` | 0.45 | Demote threshold |
| `DISPATCH_SCORE_USE_OSRM` | false | Prefer road matrix in ScoreCandidate |
| Migration | `20260813_g6_forecast_mape.ddl` | `Mape28` column |

## Honesty

- MEIO is **heuristic** multi-commodity (`cost_aware_v2`), not LP-optimal.
- Rust `CP_SAT` path remains greedy; status always `HEURISTIC`.
- Demote requires flag on + sample≥14; MAPE is additive UI metric (WAPE remains gate).
- OSRM score path is flag-gated; falls back to Haversine with source label.
