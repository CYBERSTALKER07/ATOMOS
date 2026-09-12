# G7 Proof — Ecosystem polish & re-score

## Tests

```text
go test ./factory/ ./platformadmin/ -count=1
# ok (SLA EvaluateSLA + existing factory/platformadmin suites)
```

## Evidence

| ID | Evidence |
|----|----------|
| G7-1 | `factory/sla.go`, `HandleSLABoard`, `GET /v1/factory/sla-board`, worker `RunFactorySLABreachWorker`, `EventFactorySLABreach`, portal badges |
| G7-2 | `HandleOutboxDeadLetters`, OpsPanel dead-letters table, ROLE_ROW + parity-ledger G7 section |
| G7-3 | `FEATURES_BY_APP_ROLE.md` header **G7 regen 2026-08-13** + factory SLA + platformadmin/planning/partner deltas |
| G7-4 | SCORECARD + GAP G7 DONE + `RESIDUAL_REGISTER.md` |

## Honesty

- SLA is due-date / default-hours heuristic, not MES promise engine.
- Kafka topic DLQ still CLI; Spanner `OutboxDeadLetters` is what ops can browse.
- Scorecard 10s do **not** claim live Soliq keys or OR-Tools prod pods.
