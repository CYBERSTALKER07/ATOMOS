# Shared workspace memory (all agents / IDEs)

<!-- VOID-GRAPH-MEMORY-SEED -->

**Read first every session:** `.agents/memory/GOAL.md` — the final goal does not change when the chat resets.

This file is working memory. It is not status. Re-verify in code before acting.

## Project Context

- Living product: `pegasusX/`. `pegasus/` is legacy port source.
- **Goal:** `GOAL.md` = `GLOBAL_SCALE_PROGRAM.md` + `GLOBAL_SCALE_LOCAL_ECOSYSTEM.md` (set 2026-08-16). Isolation key stays `SupplierId`. Next claimed slices: L0 + K1.
- Tenant key: `SupplierId` only. Market pack + home cell are attributes, not a second RLS key.
- Dual planes: factory truck manifests vs supplier truck manifests — do not merge.
- Factory planning / auto-order place: flags default off.
- Money: integer minor units. Fiscal hard-gate. Pay-at-delivery.

## Verified 2026-08-15

- Unique ecosystem product features: **250** `BF-*` IDs in `pegasusX/docs/GLOBAL_SCALE_BACKEND_FEATURES.md` (recount if the file changes).
- MarketPack advertised (`GET /v1/auth/session`, `GET /v1/platform/market-packs`); `checkout_reads_this: false`. Checkout/fiscal/proximity do not read the pack until GS-M.
- `POST /v1/platform/tenants/register` is mounted when `TenantRegister != nil` (`platformroutes/routes.go`). Re-trace before claiming self-serve is complete (KYB / seed freeze / pack-as-law still GS-T/M).
- Architecture graph: `pegasusX/context/architecture-graph.json` — 88 nodes, 160 edges, `generatedAt: null`. Routing index only.
- Grok `[memory] enabled = true` in `~/.grok/config.toml` (2026-08-15). First-turn injection requires a new Grok session.
- Retrieval skill: `.agents/skills/graph-retrieval-memory/`. Walker: `scripts/graph_retrieve.py`.

## Verified 2026-08-16

- Re-read this session: MarketPack UZ `CheckoutReadsThis: false` — `pegasusX/apps/backend-go/auth/market_pack.go:121`. Session advertises pack; checkout is not pack law until GS-M — `pegasusX/apps/backend-go/auth/session.go:50`.
- `POST /v1/platform/tenants/register` mounts only if `TenantRegister != nil` — `pegasusX/apps/backend-go/platformroutes/routes.go:46`. Bootstrap constructs `tenantRegSvc` and passes it — `pegasusX/apps/backend-go/bootstrap/bootstrap.go:707` and `:1834`.
- Architecture graph still 88 nodes / 160 edges / `generatedAt: null` (routing index only).
- Feature inventory still **250** unique `BF-*` table rows in `pegasusX/docs/GLOBAL_SCALE_BACKEND_FEATURES.md` (ids 001–359, not contiguous).
- Warehouse is supplier-scoped; checkout picks closest covering on-shift warehouse — `order/warehouse_resolver_spanner.go:23`. Empty country fail-open — `order/coverage.go:107`. Coverage cells skip country entirely.
- Three warehouse writers persist different columns: CRUD (`warehouse/repository_crud.go:50`), topology (`supplier/repository_spanner.go:1043`), setup omits country/H3 (`warehouse/setup.go:87`).
- Factory for a warehouse is `PrimaryFactoryId` only — `warehouse/transfers.go:266`, `warehouse/supply_topology.go:18`.
- Warehouse payment POST allowlist hardcoded — `warehouse/payment_config.go:36`.
- GS-L/K plan (rev 2, W1–W26) lives at `pegasusX/docs/GLOBAL_SCALE_LOCAL_ECOSYSTEM.md`; linked from `GLOBAL_SCALE_PROGRAM.md`. Setup omit country — `warehouse/setup.go:87`. Cells skip country — `order/coverage.go:102`. Catalog `Latitude` — `catalog/stock.go:77`.
- Graph query `fiscal retry` returns **no hits**. Live: `HandleFiscalRetry` — `pegasusX/apps/backend-go/order/service.go:3072`. Mount `POST /v1/order/{orderID}/fiscal/retry` (driver/ADMIN/warehouse admin) — `pegasusX/apps/backend-go/orderroutes/routes.go:54`.
- Graph query `payment webhook` seeds `paymentroutes` / `payment/service.go`, not PSP ingress. Live PSP hooks are `webhookroutes` — `pegasusX/apps/backend-go/webhookroutes/routes.go:23`. `paymentroutes` says so — `pegasusX/apps/backend-go/paymentroutes/routes.go:2`.
- Graph query `warehouse dispatch` 0-hop includes checkout `ResolveNearestWarehouseID` — `pegasusX/apps/backend-go/order/warehouse_resolver_spanner.go:23` — and warehouse *apps*; warehouse-role package starts at `pegasusX/apps/backend-go/warehouse/service.go:1` (`errDispatchLockNotFound` at `:34`). Hits are paths, not a dispatch-live verdict.
- Cursor wired this session: `~/.cursor/skills/graph-retrieval-memory` → V.O.I.D skill; `~/.cursor/rules/graph-retrieval-memory.mdc` has `alwaysApply: true`. Walker cwd is `Desktop/V.O.I.D` (home-relative `.agents/...` does not exist).
- Cursor CLI (this session): walker from any cwd is `python3 $HOME/.cursor/skills/graph-retrieval-memory/scripts/graph_retrieve.py` (finds graph via `__file__`). `/graph-retrieve` → `~/.cursor/commands/graph-retrieve.md`. `sessionStart` hook `~/.cursor/hooks/graph-retrieval-session.sh` sets `VOID_REPO` / `GRAPH_RETRIEVE` / `VOID_MEMORY` (telemetry `preToolUse` kept). Prefer `agent --workspace $HOME/Desktop/V.O.I.D`. Project allowlist: `Desktop/V.O.I.D/.cursor/cli.json`.
- **Final goal retarget (this session):** `.agents/memory/GOAL.md` now names `pegasusX/docs/GLOBAL_SCALE_PROGRAM.md` + `pegasusX/docs/GLOBAL_SCALE_LOCAL_ECOSYSTEM.md` as the destination. `PROD_ECOSYSTEM_GOAL.md` stays Class A coverage only — banners on both GS docs + `DOCS_SOURCE_OF_TRUTH.md`. This is a pointer change, not a code-path claim.
