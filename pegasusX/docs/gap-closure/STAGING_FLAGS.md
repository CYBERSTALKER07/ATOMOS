# Gap Closure — Phase 10 Staging Flag Enablement

Enable **one flag at a time** on staging after Phase 9 mobile/desktop parity passes. Roll back immediately if smoke fails.

## Sequence

1. `CREDIT_NOTE_AUTO_FROM_BUYER_REJECT=true` — verify draft credit notes from buyer reject poller.
2. `CREDIT_NOTE_AUTO_FROM_CLAIM=true` — verify claim approval hook.
3. `CASH_RECONCILIATION_REQUIRED=true` — **only after Phase 3 driver cash recon UI verified**.

## Removed (do not enable)

- `CREDIT_SCORE_ENFORCEMENT_ENABLED` — **dead flag**. Credit risk scoring / RiskTier was removed in Phase A; zero Go references. Do not set in staging or prod.

## Production readiness (after staging parity)

- Re-run all gap-closure Spanner migrations on production.
- Redeploy `backend-go`, supplier surfaces (web + mobile), driver apps, warehouse apps.
- Sync Secret Manager env vars with staging-validated values.
- Verify outbox relay + notification consumer healthy (no new Kafka topics).
- Enable flags in production using the same order as staging.
