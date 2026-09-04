# Auto-order `place` flip criteria (Phase 4)

Do **not** set `AUTO_ORDER_PLACE_ENABLED=true` until all of the following are true.

**Wiring SoT (modes, overlays, ladder):** [`AUTO_ORDER.md`](./AUTO_ORDER.md).

## Evidence gate

1. Shadow soak ≥ **30 days** for the pilot retailer cohort with `AUTO_ORDER_SHADOW_ENABLED=true` (+ `AUTO_ORDER_WORKER_ENABLED=true`).
2. Unmodified acceptance rate on `RetailerAutoOrderShadowProposals` ≥ **80%** over the trailing 30 days (runtime default `AUTO_ORDER_SOAK_MIN_UNMODIFIED=0.80`).
3. Weekly WAPE/bias/TS report archived under `artifacts/forecast-shadow/` (from `FORECAST_ACCURACY_ENABLED` + shadow worker).
4. Human sign-off recorded in platform admin audit (`PLATFORM_ADMIN` action `FLAG_AUTO_ORDER_PLACE` on dual-control approve).
5. Two-person rule: money-affecting flag override must include a non-empty reason (`featureflags` rejects empty reasons); a second PLATFORM_ADMIN must approve.

## Runtime dual-control

`retailer.placeAllowedForRetailer` calls `featureflags.Evaluate("AUTO_ORDER_PLACE_ENABLED", "RETAILER", orgID)` so tenant ACTIVE overrides (post-approve) apply without process restart. Env remains the default when no override exists.

## Artifact

```bash
# From authenticated retailer session:
RETAILER_BEARER=… API_BASE=https://api… bash scripts/generate_auto_order_soak_artifact.sh
# Writes artifacts/forecast-shadow/acceptance-30d.json (both rate field names).
```

## Rollback

Set `AUTO_ORDER_PLACE_ENABLED=false` (env or tenant override) — place path fails closed to draft/shadow.

## Gate helper

```bash
# Prints place-flip-blocked until artifacts/acceptance report exists and passes thresholds.
bash scripts/auto_order_place_flip_check.sh
```
