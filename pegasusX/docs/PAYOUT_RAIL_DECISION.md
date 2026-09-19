# Payout rail decision

**Date:** 2026-08-12  
**Gap:** P0-2 live-rail remainder  
**Decision:** **Bank-file CSV is the permanent settlement transport for this prod bar.**

## Locked choice

| Option | Choice |
|--------|--------|
| Settlement transport | **Bank-file** (`BankFileRail`) — CSV payment instruction exported to the bank |
| Live API rail (bank / Global Pay payout) | **Deferred** — interface + fail-closed remain; no silent live dispatch |
| Money truth | Spanner payout batch + ledger legs; file is transport only |

Rationale:

1. Fail-closed live dispatch already exists (`Rail.IsLive()`, `ErrNoLiveRail`) — a batch cannot strand in `SUBMITTED` without a real rail.
2. No bank/Global Pay payout credentials or webhook certification are in-repo yet; inventing a stub “live” rail would create false money movement.
3. Prod ecosystem goal allows either a live rail **or** bank-file documented as permanent settlement — this doc is that documentation.

## Operational model

1. Generate batch (`DRAFT`) for period — net = Σcaptured − Σrefunds − commission.
2. `ExportBankFile` / dry `SubmitForDispatch(live=false)` → `EXPORTED` + CSV.
3. Ops/bank processes the file.
4. Human (or bank recon import later) marks `PAID` via `MarkPaid` — not a fake rail webhook.

`SubmitForDispatch(live=true)` on the default rail **must fail** with `ErrNoLiveRail`.

## Future live rail (out of this bar)

When a real rail is ready:

1. Implement `Rail` with `IsLive() == true` and webhook → `ConfirmSettlement`.
2. Register by name in `railByName` (do not fall through to bank-file for unknown live names).
3. Certify with bank sandbox + webhook secret (`PAYOUT_RAIL_WEBHOOK_SECRET`).
4. Update this decision doc; do not remove bank-file as fallback.

## Code anchors

- `apps/backend-go/payout/rail.go` — `BankFileRail`, `ErrNoLiveRail`
- `apps/backend-go/payout/handlers.go` — export + settlement webhook surface
- Gap register P0-2 (fail-closed resolved; live integration deferred by this decision)
