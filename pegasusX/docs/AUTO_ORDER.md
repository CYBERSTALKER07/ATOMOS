# Auto-order — E2E wiring SoT

Inventory-grounded auto-order proposes replenishment from stock, sell-through / baseline demand, and §8.2 reorder points — not `last_invoice / 2`.

**Companions:** [`AUTO_ORDER_PLACE_FLIP.md`](./AUTO_ORDER_PLACE_FLIP.md) (place-flip evidence criteria) · [`session-2026-08-07/DOMAIN2_AUTONOMY_PROGRESS.md`](./session-2026-08-07/DOMAIN2_AUTONOMY_PROGRESS.md)

> Place is **not** one env flip. Flag says “may”; soak gate says “ready”; dual-control + human approve enable place. Do not auto-flip.

---

## Ladder (done correctly)

```
settings (mode + scopes)
  → worker 15m OR POST …/run
  → candidates (inventory_rs → reorder → AI)
  → mode:
       shadow → RetailerAutoOrderShadowProposals
       draft  → cart upsert
       place  → placeAllowedForRetailer?
                  yes → order.Create Source=AUTO_ORDER
                  no  → draft fallback
  → RETAILER_AUTO_ORDER_UPDATED (outbox + inbox)
```

Shadow soak (≥30d) → soak-gate pass → PLATFORM_ADMIN dual-control `AUTO_ORDER_PLACE_ENABLED` → retailer Place.

---

## Execution modes

Global `execution_mode` on `RetailerAutoOrderSettings`:

| Mode | Behavior |
|------|----------|
| `off` | No worker action (`GlobalEnabled` cleared on PATCH) |
| `shadow` | Persist proposals to `RetailerAutoOrderShadowProposals`; **no** cart / place |
| `draft` | Upsert cart draft lines (default when mode empty — backward compat) |
| `place` | Create `AUTO_ORDER` via `order.Service.Create` only when `placeAllowedForRetailer` + manager/`order.place`; else **draft fallback** |

Worker: if settings say `place` but gate fails → **downgrade to draft** (fail-closed).

## Scope policy

Bool overrides: variant → product → category → supplier → global.  
Most-specific wins. `Enabled=false` at any matching scope **blocks** even when global is on.  
When global is off, only scoped enables allow.

## Candidate chain

1. Test seed (unit tests)
2. Inventory `(R,s,S)` proposals (`candidate_source=inventory_rs`)
3. OPEN `ReorderSuggestions`
4. AI prediction lines — **skipped** when `AUTO_ORDER_INVENTORY_GROUNDED=true`

Proposal math: `IP = on_hand + in_flight − reserved`; if `IP ≤ s` (ROP), order up to `S = s + d̄·R` (default `R=1` day), round by `UnitsPerPack`. Confidence decays with days since stock `UpdatedAt`.

Prefer `AUTO_ORDER_INVENTORY_GROUNDED=true` on soak/prod so evidence is stock-based (honest soak).

---

## Flags and overlays

| Env | Default | Role |
|-----|---------|------|
| `AUTO_ORDER_WORKER_ENABLED` | off | 15m background sweep |
| `AUTO_ORDER_SHADOW_ENABLED` | on if unset | Allow shadow persist |
| `AUTO_ORDER_PLACE_ENABLED` | **off** | Money flag; dual-control; process env + tenant override |
| `AUTO_ORDER_SOAK_GATE_DISABLED` | off | Break-glass soak bypass (money flag + audit) |
| `AUTO_ORDER_INVENTORY_GROUNDED` | off | Prefer inventory proposals; divert AI `/2` preorder |
| `AUTO_ORDER_SOAK_MIN_PROPOSALS` | 20 | Soak gate |
| `AUTO_ORDER_SOAK_MAX_WAPE` | 0.30 | Soak gate |
| `AUTO_ORDER_SOAK_MIN_UNMODIFIED` | 0.80 | Soak gate (matches place-flip policy) |

**Money flags** (`AUTO_ORDER_PLACE_ENABLED`, `AUTO_ORDER_SHADOW_ENABLED`, `AUTO_ORDER_SOAK_GATE_DISABLED`): set → PENDING with non-empty reason → **different** `PLATFORM_ADMIN` Approve. Audits: `FLAG_AUTO_ORDER_PLACE` / `FLAG_AUTO_ORDER_SOAK_GATE`. Runtime soak bypass also audits `AUTO_ORDER_SOAK_GATE_BYPASS`.

**Overlay defaults:**

| Overlay | Shadow | Worker | Place |
|---------|--------|--------|-------|
| Base configmap | false | false | false |
| Staging | true | **true** (soak path) | false |
| Prod | true | true | false |

Code: `apps/backend-go/retailer/auto_order_soak_gate.go` (`placeAllowedForRetailer`), `featureflags/service.go`, bootstrap `SetPlaceFlagEvaluator` / `SetSoakBypassAuditor`.

---

## Place gate (`placeAllowedForRetailer`)

1. Evaluate `AUTO_ORDER_PLACE_ENABLED` for `RETAILER` + org (ACTIVE override or env). Off → deny.
2. Soak bypass? (`AUTO_ORDER_SOAK_GATE_DISABLED` dual-control/env) → allow + audit.
3. Else `EvaluateSoakGate` on 30d shadow stats (fail-closed):
   - proposals ≥ min
   - if matched orders: WAPE ≤ max and unmodified ≥ min
   - enough proposals but **zero** matched orders → deny

Flip criteria detail: [`AUTO_ORDER_PLACE_FLIP.md`](./AUTO_ORDER_PLACE_FLIP.md).

---

## APIs

| Method | Path |
|--------|------|
| GET/PATCH | `/v1/retailer/settings/auto-order…` (global + scoped) |
| POST | `/v1/retailer/settings/auto-order/run?mode=shadow\|draft\|place` |
| GET | `…/runs`, `…/shadow-proposals`, `…/shadow-stats` |
| GET | `…/soak-gate` — live decision + thresholds |
| GET | `…/soak-artifact` — downloadable 30d evidence pack |

## Synthesis diversion

When `AUTO_ORDER_INVENTORY_GROUNDED=true`, `ai-worker` `maybePreorderMutation` does **not** insert `AI_PREORDER` with `qty/2`. `AIPredictions` advisory writes remain.

## Clients

| Client | Modes / run / shadow | Soak readiness |
|--------|----------------------|----------------|
| Desktop `/auto-order` | Full | soak-gate card + artifact download |
| Android / iOS AutoOrder | Full | soak-gate readiness strip (artifact via desktop/script) |
| Admin FlagsPanel | — | Dual-control PLACE / SHADOW / SOAK_GATE_DISABLED |

Events: `RETAILER_AUTO_ORDER_UPDATED` → outbox → notification dispatcher → WS/inbox (`mode` in payload).

---

## Operator ladder (wire E2E)

1. **Infra soak path** — `SHADOW=true`, `WORKER=true`, `PLACE=false`; prefer `INVENTORY_GROUNDED=true`; forecast accuracy on.
2. **Retailer** — mode `shadow`, scopes on, primary location with valid geo.
3. **Evidence (≥30 days)** — readiness pass; download artifact or `scripts/generate_auto_order_soak_artifact.sh` → `artifacts/forecast-shadow/acceptance-30d.json`; run `scripts/auto_order_place_flip_check.sh`.
4. **Dual-control** — Admin A sets `AUTO_ORDER_PLACE_ENABLED` for RETAILER+org + reason; Admin B Approves.
5. **Place** — OWNER/ADMIN/MANAGER + `order.place`: mode `place` or Place now.
6. **Rollback** — disable place flag/env → draft/shadow. Soak-bypass only as audited break-glass.

```bash
RETAILER_BEARER=… API_BASE=https://api… bash scripts/generate_auto_order_soak_artifact.sh
bash scripts/auto_order_place_flip_check.sh
```

---

## Verification checklist

- [ ] Shadow run → rows in `RetailerAutoOrderShadowProposals` + notification
- [ ] `GET …/soak-gate` reasons match thresholds
- [ ] Place with flag off → draft fallback, no `AUTO_ORDER` order
- [ ] Place with flag on + soak fail → draft fallback
- [ ] Place with flag on + soak pass + manager + geo → `order.Create` Source=`AUTO_ORDER`
- [ ] Dual-control same-admin approve rejected
- [ ] `auto_order_place_flip_check.sh` blocks env place without artifact
- [ ] SSMR: `PX_E2E_AUTO_ORDER_SHADOW_OK` / `PX_E2E_AUTO_ORDER_DRAFT_OK` when enabled

## SSMR markers

- `PX_E2E_AUTO_ORDER_DRAFT_OK` / `_SKIPPED`
- `PX_E2E_AUTO_ORDER_SHADOW_OK` / `_SKIPPED`

## Out of scope

- Auto-flip `place` at ≥80% without human + dual-control
- Per-scope execution mode
- POS shelf-count UX rebuild / marketplace multi-supplier cart
