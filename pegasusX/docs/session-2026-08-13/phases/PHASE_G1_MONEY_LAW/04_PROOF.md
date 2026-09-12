# 04 — Proof — G1.A

## Commands

```bash
cd apps/backend-go
go test ./ar/ ./order/ -count=1
# ok ar, ok order
go build -o /tmp/pegasusx-backend .
```

## Grep

```bash
rg -n 'RecordPaymentForOrder' order/service.go
# Should only appear via RecordPaymentForOrderInTxn inside InTxn (no post-commit fail-open log)
```

## Tests added

- `ar/wave_g1a_test.go` — no invoice, pay-down, idempotent, bad input

## G1-A2 proof

```bash
go test ./credit/ ./order/ ./ar/ -count=1  # ok
```

- Post-commit ClearBalance on CompleteOrder **removed**
- ClearBalanceInTxn in `finalizeCardSettlement` (after CAPTURED) + CollectCash InTxn
- CLEARED reservation status for idempotency

## G1.B proof

```bash
go test ./order/ ./bootstrap/ -count=1  # ok
go build -o /tmp/pegasusx-backend .
```

- `ResolveFiscalProviderName`: production/staging/FISCAL_TAX_MARKET unset → MY_SOLIQ
- Production boot: MY_SOLIQ requires OFD+EDS; PEGASUS requires `FISCAL_ALLOW_COMMERCIAL_RECEIPTS`
- prod overlay: `FISCAL_PROVIDER=MY_SOLIQ`; staging keeps explicit PEGASUS + allow flag until secrets

## G1.C proof

Driver clients no longer call PATCH state or mid-delivery for success UX:
- Android: `CorrectionViewModel` amend-only; `ManifestViewModel.transitionOrder` → arrive only
- iOS: `legacyStartTransit` fails honest; correction skips mid-delivery
- Backend: `use_delivery_edges` / `use_amend_or_partial_offload` messages
- Portal: removed `credit.score.updated` default pref

```bash
go test ./driver/ ./order/ -count=1  # ok
```

## G1.D proof

```bash
go test ./payout/ ./notifications/ ./bootstrap/ -count=1  # ok
go build -o /tmp/pegasusx-backend .
```

- Payout: `GET .../rail`, dispatch live → `no_live_rail` 409, MarkPaid/Generate include rail honesty
- FCM: `IsNoOp`, Error-level push_degraded logs; production refuses silent no-op unless `FCM_ALLOW_NOOP`
- Runbook: `docs/PAYOUT_BANK_FILE_RUNBOOK.md`

## Verdict

- [x] PASS — **G1 complete (A–D)**  
- Next program phase: **G2** physical + autonomy (not auto-started)
