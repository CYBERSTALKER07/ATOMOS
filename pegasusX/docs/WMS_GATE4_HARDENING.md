# Gate 4 WMS — scanning throughput notes (§8.7 / §8.8)

PR-7 documents warehouse scan hardening still open on native pickers (prefetch EAN→SKU map, increment vs toggle, torch/wedge). Backend Gate 4 closeout does **not** rewrite scanner UX in this PR; track as follow-up under warehouse Android/iOS scan screens.

Backend hardening shipped in PR-7:
- `GET /v1/warehouse/ops/inventory-reconcile` — V2 vs AVAILABLE lot sum drift
- Lots path remains FEFO via `stocklots` when `WMS_LOTS_ENABLED` (direct V2 reserve bypasses documented as residual forbid)

Ops checklist: [`WMS_GATE4_OPS.md`](./WMS_GATE4_OPS.md)
