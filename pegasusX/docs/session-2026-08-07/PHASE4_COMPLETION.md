# Phase 4 completion — Autonomy on evidence foundations (2026-08-11)

**Gate:** `make phase4-gate` → **`phase4-gate-ok`** (re-proved 2026-08-11, Spanner emulator `localhost:9010`; full regression, no skip)  
**Status:** Wired (foundations); 30-day soak / prod optimizer / human place flip remain owner residuals

## Proof (2026-08-11)

| Step | Result |
|------|--------|
| [1/6] Phase-3 regression | `phase3-gate-ok` |
| [2/6] Partial allocation unit path | PASS |
| [3/6] S&OP no `sku-projection-%d` + CapacityModel | OK |
| [4/6] Optimizer replicas ≥ 1 SSMR + staging | OK |
| [5/6] Place flip check (place not enabled) | `place-flip-check-skipped` |
| [6/6] Planning + allocation packages | PASS |

## Shipped

| Item | Detail |
|------|--------|
| Optimizer SSMR/staging | `replicas: 1` (prod stays 0 until AR image) |
| Shadow soak flags | SSMR + staging: forecast algo/accuracy, safety stock v2, auto-order shadow ON; place OFF |
| Partial allocation | `PARTIAL_ALLOCATION_ENABLED` — best-effort fill + PARTIAL/NO_STOCK decisions |
| S&OP | Live supply-request projections preferred; calibrated daily rates via env; no `sku-projection-%d` |
| Place flip | `docs/AUTO_ORDER_PLACE_FLIP.md` + `scripts/auto_order_place_flip_check.sh` (≥80% / 30d artifact) |

## Fix landed this run

- `AllocateOrder` used `client.Single()` across warehouse + inventory queries (Single is one-shot) → switched to `ReadOnlyTransaction`.
- `TestAllocateOrder` now forces `PARTIAL_ALLOCATION_ENABLED=false` so hard-fail assertions are not softened by shell/SSMR env.

## Explicit soak / owner work remaining

- 30-day shadow acceptance report artifact (`artifacts/forecast-shadow/acceptance-30d.json`)
- Prod optimizer AR image + replicas ≥ 1
- Human two-person place flip
- Live `PX_E2E_OPTIMIZER_CONSTRAINT_OK` on cloud

## Next deep-dive

**Phase 6** (decision-gated marketplace/cert) or analytics column tenancy / client residuals.
