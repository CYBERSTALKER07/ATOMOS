# SSMR Next-Layer Remaining Rollout — 2026-08-04

## Image

`asia-south1-docker.pkg.dev/pegasus-503013/pegasusx-ssmr-images/backend-go:ssmr-next-layer-db8f75fb`

Rolled to `backend-go` + `backend-go-worker` in `pegasusx-ssmr`. Healthz OK.

## Engineering landed

| Area | Change |
|------|--------|
| L1 | `PX_E2E_PAYMENT_CARD_SUCCESS_OK` / `_SKIPPED`; release checklist doc |
| L2 | CT sim never on ssmr/prod; `RunAutoOrderWorker` + `AUTO_ORDER_WORKER_ENABLED`; CI no-mock script |
| L3/L6/L7 | Docs + mobile local SKUs; claim quarantine e2e marker; local: guard already in batch |
| L4 | `QUANTITY_NEGOTIATION_ENABLED` env gate (default off); driver Android API restored |
| L5 | `docs/SOLIQ_SANDBOX_READINESS.md` + `PX_E2E_SOLIQ_SANDBOX_SKIPPED` |
| Hygiene | Removed 59MB `ssmr-smokecheck` binary; `.gitignore` |

## Pilot flags (SSMR)

| Flag | Value |
|------|-------|
| `OFFLINE_COUNT_ENABLED` | `true` |
| `ASSIST_SLA_ENABLED` | `true` |
| `HQ_ANALYTICS_ENABLED` | `true` (OWNER reads) |
| `AUTO_ORDER_WORKER_ENABLED` | `true` |
| `AUTO_ORDER_PLACE_ENABLED` | `false` (draft default) |
| `QUANTITY_NEGOTIATION_ENABLED` | `false` |
| `MULTI_ORG_LOGIN_ENABLED` | unset / off |

## Smoke

- Full `e2e`: Spanner `DeadlineExceeded` on early retailer cancel (pre-existing flake) — markers after that path not reached.
- Focused `payment`: Global Pay still **2030 unauthorized** (owner password) → cash fallback also timed out on force-complete.

## Owner blockers (unchanged)

1. Real Global Pay staging password → GSM → expect `PX_E2E_PAYMENT_CARD_SUCCESS_OK`
2. Firebase Phone/SHA for OTP on device
3. Enable `QUANTITY_NEGOTIATION_ENABLED=true` after supplier resolve UX QA

## Artifacts

- Baseline: `artifacts/next-layer-remaining-baseline-2026-08-04.md`
- Checklist: `docs/L1_FIELD_UNLOCK_RELEASE_CHECKLIST.md`
