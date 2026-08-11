# Auto-order `place` flip criteria (Phase 4)

Do **not** set `AUTO_ORDER_PLACE_ENABLED=true` until all of the following are true.

## Evidence gate

1. Shadow soak ≥ **30 days** for the pilot retailer cohort with `AUTO_ORDER_SHADOW_ENABLED=true`.
2. Unmodified acceptance rate on `RetailerAutoOrderShadowProposals` ≥ **80%** over the trailing 30 days.
3. Weekly WAPE/bias/TS report archived under `artifacts/forecast-shadow/` (from `FORECAST_ACCURACY_ENABLED` + shadow worker).
4. Human sign-off recorded in platform admin audit (`PLATFORM_ADMIN` action `FLAG_AUTO_ORDER_PLACE`).
5. Two-person rule: money-affecting flag override must include a non-empty reason (`featureflags` rejects empty reasons).

## Rollback

Set `AUTO_ORDER_PLACE_ENABLED=false` (env or tenant override) — place path fails closed to draft/shadow.

## Gate helper

```bash
# Prints place-flip-blocked until artifacts/acceptance report exists.
bash scripts/auto_order_place_flip_check.sh
```
