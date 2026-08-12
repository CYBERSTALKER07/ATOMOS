# Domain 2 — Autonomy on evidence (P1) · Progress

Date: 2026-08-11 · Roadmap ref: `/Users/shakhzod/.cursor/plans/Ecosystem Capability Roadmap-7cbf327a.plan.md` (Domain 2).

## What was already wired (Phase 4)

- Shadow mode: `retailer/auto_order_shadow.go` persists `RetailerAutoOrderShadowProposals`
  and computes `AutoOrderShadowStats` (proposal count, matched orders, WAPE,
  unmodified accept rate) over a 30-day window by joining shadow proposals to
  completed orders in a ±3-day window.
- Execution modes `off | shadow | draft | place`; `place` was gated only by the
  process-wide `AUTO_ORDER_PLACE_ENABLED` env flag.

## What this change adds

### 1. Gated place flip (evidence-gated, fail-closed)

- New `retailer/auto_order_soak_gate.go`:
  - `SoakGateConfig` (env-tunable): `AUTO_ORDER_SOAK_MIN_PROPOSALS` (20),
    `AUTO_ORDER_SOAK_MAX_WAPE` (0.30), `AUTO_ORDER_SOAK_MIN_UNMODIFIED` (0.80 —
    aligned with place-flip policy), `AUTO_ORDER_SOAK_GATE_DISABLED` (break-glass
    bypass; money-flag dual-control as of P2-7).
  - `EvaluateSoakGate` — allows place only when the 30-day shadow stats pass all
    thresholds. **Fail-closed**: stats errors, too few proposals, or enough
    proposals with zero matched orders all deny the flip.
  - `placeAllowedForRetailer` — combines process env, soak gate, and runtime
    `featureflags.Evaluate("AUTO_ORDER_PLACE_ENABLED", …)` (W4 / P1-5); wired into
    place paths in `auto_order_worker.go`.
  - `Service.soakGateDisabled` test seam (avoids `t.Setenv` + `t.Parallel` panic).
- Handlers (registered in `retailerroutes/routes.go`):
  - `GET /v1/retailer/settings/auto-order/soak-gate` — live gate decision +
    thresholds + flag state.
  - `GET /v1/retailer/settings/auto-order/soak-artifact` — the **exportable 30-day
    soak evidence pack** (decision + thresholds + up to 1000 proposals) served as a
    `Content-Disposition: attachment` JSON download, for attaching to a place-flip
    approval.

### 2. UI — retailer desktop auto-order page

- `apps/retailer-app-desktop/app/(dashboard)/auto-order/page.tsx`:
  - New **"Place readiness (30-day shadow soak)"** card — live gate verdict
    (passed/blocked), proposal/match counts, WAPE + unmodified rates, the active
    thresholds, the deny reasons, and a **Download evidence (JSON)** button that
    pulls the soak artifact.
  - Reorder suggestions now surface `safety_stock` (SS) alongside qty/stock.

## Verification

- `go build ./...` — clean; `go vet ./retailer/` — clean.
- `retailer/auto_order_soak_gate_test.go` — disabled-allows, insufficient-proposals
  denies, no-matched-orders denies, flag+gate composition — all pass.
- Full `./retailer/` suite: only pre-existing failures remain
  (`TestAutoOrderWorkerFromAIPredictions`, `TestHandleProfileBackfillsSupplierID`),
  both confirmed failing with these changes stashed and unmodified by me.
- Retailer desktop `tsc --noEmit`: 0 errors in `auto-order/page.tsx` (the ~60
  remaining errors are a pre-existing `@types/react` version conflict across
  unrelated components).

## Notes / follow-ups

- Dual-control + runtime `featureflags.Evaluate` govern `AUTO_ORDER_PLACE_ENABLED`
  / soak-bypass money flags; the soak gate is the *runtime evidence* counterpart.
  Together: flag says "may", soak gate says "ready". Place env remains false until
  flip evidence (W4).
- Pre-existing flake `TestAutoOrderWorkerFromAIPredictions` (no_suggestions under
  parallel seeding) is worth a separate hardening pass — out of scope here.
