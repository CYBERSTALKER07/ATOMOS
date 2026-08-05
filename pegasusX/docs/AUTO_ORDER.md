# Auto-order (§8.3 inventory-grounded + shadow)

Inventory-grounded auto-order proposes replenishment from stock, sell-through / baseline demand, and §8.2 reorder points — not `last_invoice / 2`.

## Execution modes

Global `execution_mode` on `RetailerAutoOrderSettings`:

| Mode | Behavior |
|------|----------|
| `off` | No worker action (`GlobalEnabled` cleared on PATCH) |
| `shadow` | Persist proposals to `RetailerAutoOrderShadowProposals`; **no** cart / place |
| `draft` | Upsert cart draft lines (default when mode empty — backward compat) |
| `place` | Create `AUTO_ORDER` via `order.Service.Create` when `AUTO_ORDER_PLACE_ENABLED` + manager perm |

## Scope policy

Bool overrides: variant (size/SKU) → product → category → supplier → global.  
Most-specific wins. `Enabled=false` at any matching scope **blocks** even when global is on.  
When global is off, only scoped enables allow.

## Candidate chain

1. Test seed (unit tests)
2. Inventory `(R,s,S)` proposals (`candidate_source=inventory_rs`)
3. OPEN `ReorderSuggestions`
4. AI prediction lines — **skipped** when `AUTO_ORDER_INVENTORY_GROUNDED=true`

Proposal math: `IP = on_hand + in_flight − reserved`; if `IP ≤ s` (ROP), order up to `S = s + d̄·R` (default `R=1` day), round by `UnitsPerPack`. Confidence decays with days since stock `UpdatedAt`.

## Flags

| Env | Default / notes |
|-----|-----------------|
| `AUTO_ORDER_WORKER_ENABLED` | Background sweep (default off) |
| `AUTO_ORDER_PLACE_ENABLED` | Place mode process gate (default off) |
| `AUTO_ORDER_SHADOW_ENABLED` | Shadow persist (default **on** unless `false`) |
| `AUTO_ORDER_INVENTORY_GROUNDED` | Prefer inventory proposals; divert synthesis `/2` AI_PREORDER insert |

## APIs

- `GET/PATCH /v1/retailer/settings/auto-order...` — settings + scoped overrides; GET may include `shadow_stats`
- `POST /v1/retailer/settings/auto-order/run?mode=shadow\|draft\|place`
- `GET /v1/retailer/settings/auto-order/runs`
- `GET /v1/retailer/settings/auto-order/shadow-proposals`
- `GET /v1/retailer/settings/auto-order/shadow-stats` — 30d WAPE + unmodified accept rate

## Synthesis diversion

When `AUTO_ORDER_INVENTORY_GROUNDED=true`, `ai-worker` `maybePreorderMutation` does **not** insert `AI_PREORDER` with `qty/2`. `AIPredictions` advisory writes remain.

## Clients

Desktop / Android / iOS Auto-Order surfaces: mode segmented control, scope toggles, shadow inbox + acceptance strip.

## SSMR

- `PX_E2E_AUTO_ORDER_DRAFT_OK` / `_SKIPPED`
- `PX_E2E_AUTO_ORDER_SHADOW_OK` / `_SKIPPED`

## Out of scope (this delivery)

- Auto-flip `place` at ≥80% acceptance without human + env
- Per-scope execution mode
- POS shelf-count UX rebuild / marketplace multi-supplier cart
