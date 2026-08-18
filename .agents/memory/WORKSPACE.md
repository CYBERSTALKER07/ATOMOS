# Shared workspace memory (all agents / IDEs)

<!-- VOID-GRAPH-MEMORY-SEED -->

**Read first every session:** `.agents/memory/GOAL.md` — the final goal does not change when the chat resets.

This file is working memory. It is not status. Re-verify in code before acting.

## Project Context

- Living product: `pegasusX/`. `pegasus/` is legacy port source.
- **Goal:** `GOAL.md` = `GLOBAL_SCALE_PROGRAM.md` + `GLOBAL_SCALE_LOCAL_ECOSYSTEM.md` + `GLOBAL_SCALE_CLIENT_UI.md` (set 2026-08-16). Isolation key stays `SupplierId`. GS-U0–U9 shipped. Next: leftovers (GS-M flag, cells apply, live PSP) — Layer B, do not execute. Named + continuous empty-currency invent train closed. Not terraform apply, not Stripe keys, not flipping the flag, not a U-motion slice. `checkout_reads_this` still false.
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
- Warehouse is supplier-scoped; checkout picks closest covering on-shift warehouse via `proximity.ResolveServingWarehouse` — `order/warehouse_resolver_spanner.go`. L0: empty country fail-closed; cells still require same country — `proximity/coverage_engine.go` `CoversRetailer`.
- L1: `proximity.StampNodeGeography` stamps pack country + H3 res 7 on warehouse/factory/retailer/topology writers. Topology US on UZ → 422. Setup writes `CountryCode`/`H3Cell`. Topology H3 is res 7 (`supplier/repository_spanner.go` `topologyH3CellString`).
- L2: catalog/order/preview share `CoverageStore` + engine. Active store = JWT `active_location_id` → `RetailerLocations` else `Retailers.Lat/Lng`. Factory resolve = primary (same country) else `SupplyLanes` priority else closest — `warehouse/transfers.go` / `supply_topology.go`. INTERNAL colocated factory is a transfer-mode override.
- K1: warehouse/supplier PSP allowlists deleted; pack catalog at `payment/catalog.go`. Empty UZ config = CASH+GLOBAL_PAY. POST STRIPE → 422 `pack_gateway_forbidden`.
- K2: default STRIPE/ADYEN executors are `catalogHonestyExecutor` (no redirect URL). PAYME/CLICK → `no_live_keys`. Empty gateway uses `LivePackGateways`, not a hardcoded GLOBAL_PAY string.
- K3: planned CA/AU/GB/PK packs on `GET /v1/platform/market-packs`. countrycfg lists no payment gateways.
- L3: `ServicePins` + `SupplierRegions` in `schema/spanner.ddl`. Engine pin steps in `proximity/coverage_engine.go`. Live resolver loads pins — `order/warehouse_resolver_spanner.go` + `catalog/stock.go`. Supplier ADMIN GET/PUT `/v1/supplier/warehouses/{id}/pins|coverage` and `/v1/supplier/regions`. W14: PATCH/create `region_id` must be this supplier's `SupplierRegions` (`unknown_supplier_region`). No `/v1/region*`.
- L4: Register uses partner pack + `assertAttachMarket`. Invite carries pack; CA invite on UZ stamp → 422. Planned KZ partner → 404. Parent cart `assertChildSuppliersSameMarket` before `insertParentOrder`. `auth.AssertSameMarket` on create/unified.
- GS-R supplier portal: `GET /v1/supplier/payment-catalog` → `payment.AvailablePSPs`. Pin editor at `/warehouses/{id}/coverage`. Billing/earnings/chargebacks no longer hardcode Stripe/Adyen. iOS/Android warehouse lists hand off pin edit to desktop by 2026-09-16.
- GS-R warehouse portal: `GET /v1/warehouse/ops/coverage` view-only (PUT 405). `GET /v1/warehouse/ops/supply-factory` engine-only. Payment-config GET returns pack catalog + `currency_code`. Location GET stamps pack country. Fee currency PATCH rejects USD on UZ. Portal `/coverage` + iOS/Android coverage/factory view.
- GS-R retailer clients: `GET /v1/retailer/payment-catalog` → `payment.AvailablePSPs` (`retailer/payment_catalog.go`). UZ omits Stripe/Adyen; planned CA → 404. Desktop checkout/PaymentModal + iOS/Android checkout and delivery-payment sheets filter through that catalog. Currency picker UI removed; unified checkout does not send client currency. Empty `available_card_gateways` no longer defaults to Adyen. Cart/catalog price labels use `packCurrency` (empty pack does not invent UZS).
- GS-R POS·orders·insights: `stampPackCurrency` on POS session/sale + shift open (`retailer/pack_currency.go`). Empty currency → shipped pack; USD on UZ → 422 `pack_currency_mismatch`; planned CA → 404. HQ insights coalesce pack. Desktop/iOS/Android POS+shifts send pack currency; orders/dock/insights labels use pack (empty pack does not invent UZS).
- GS-R claims·tracking·local-SKU: local-SKU create/list/PATCH stamp pack (`retailer/local_skus.go`); list SQL `COALESCE(Currency, '')` then `coalescePackCurrency`. Empty create → UZS on UZ; USD on UZ → 422; planned CA → 404. Desktop/iOS/Android claims + tracking labels + local-SKU create/display use pack (empty pack does not invent UZS).
- GS-R role-portal UZS: supplier analytics empty currency → pack (`supplier/analytics.go`); import new products stamp pack (`import_sessions_apply.go` `importRowCurrency`); USD on UZ import → mismatch; planned CA → fail-closed. Warehouse analytics/treasury emit `currency`. Supplier + warehouse portal/iOS/Android leftover labels use pack (empty pack does not invent UZS).
- GS-R maps camera: UZ pack advertises `map_center_lat/lng` (`auth/market_pack.go`); `PackMapCenter` empty/planned → false (`auth/maps_pack.go`). Clients use `packMapCenter` / `mapInitialViewState` / `sessionMapCenter` (empty → 0,0 zoom 1 or empty fields). Display stays MapLibre/MapKit/existing Android maps. `MapsAdapter: GOOGLE_ROUTES` is routing, not a display SDK swap. `checkout_reads_this` still false.
- GS-U0 visualization contract: warehouse dashboard omits invented `sparkline_*` / `completed_today` / `today_revenue` and sets `*_available: false` (`warehouse/ops_portal.go`). Supplier `orders_by_status` is the full funnel with zeros; aliases map in (`supplier/portal_ops.go`). `fleet_vu_used` is open-manifest VU sum, not `len(orders)*10`. Types export `ORDER_STATUS_FUNNEL` / `HistorySeries` (`packages/types/index.ts`). Portal + warehouse iOS/Android hide fake today money. U-T4 factory memory dashboard stays.
- GS-U1 viz kit: `guardHistorySeries` + `statusStackModel` (`packages/types/index.ts`). Web `StatusStack` / `SourceChip` / `KpiStat` (`packages/ui-kit/src/portal`). Supplier portal Live orders swapped to StatusStack. Warehouse KPI spark only if guard passes. iOS/Android twins on supplier dashboards parse `orders_by_status`. Preview: empty / zero / unavailable / 17-key. Remaining U1 table (TimeSeries, EntityBoard, …) not this slice.
- GS-UN nav: first group ≤5. Supplier Home/Orders/Dispatch/Plan — `apps/supplier-portal/lib/nav.ts`. Warehouse Home/Dispatch/Floor/Plan. Factory Home/Bay/Payload/Transfers. Retailer Home/Buy/Incoming/Store. Overflow in `<details>` + ⌘K. `/settings/planning` uses `portal.nav.planning_settings`. Native compact tabs ≤5 (supplier dispatch+plan; warehouse floor+plan; factory payload; retailer store+more). U3 brain tabs live on `/planning`.
- GS-L/K plan (rev 2, W1–W26) lives at `pegasusX/docs/GLOBAL_SCALE_LOCAL_ECOSYSTEM.md`; linked from `GLOBAL_SCALE_PROGRAM.md`. Catalog/order Retailers columns are `Lat`/`Lng`. CoverageRadiusKm is not catchment.
- Graph query `fiscal retry` returns **no hits**. Live: `HandleFiscalRetry` — `pegasusX/apps/backend-go/order/service.go:3072`. Mount `POST /v1/order/{orderID}/fiscal/retry` (driver/ADMIN/warehouse admin) — `pegasusX/apps/backend-go/orderroutes/routes.go:54`.
- Graph query `payment webhook` seeds `paymentroutes` / `payment/service.go`, not PSP ingress. Live PSP hooks are `webhookroutes` — `pegasusX/apps/backend-go/webhookroutes/routes.go:23`. `paymentroutes` says so — `pegasusX/apps/backend-go/paymentroutes/routes.go:2`.
- Graph query `warehouse dispatch` 0-hop includes checkout `ResolveNearestWarehouseID` — `pegasusX/apps/backend-go/order/warehouse_resolver_spanner.go:23` — and warehouse *apps*; warehouse-role package starts at `pegasusX/apps/backend-go/warehouse/service.go:1` (`errDispatchLockNotFound` at `:34`). Hits are paths, not a dispatch-live verdict.
- Cursor wired this session: `~/.cursor/skills/graph-retrieval-memory` → V.O.I.D skill; `~/.cursor/rules/graph-retrieval-memory.mdc` has `alwaysApply: true`. Walker cwd is `Desktop/V.O.I.D` (home-relative `.agents/...` does not exist).
- Cursor CLI (this session): walker from any cwd is `python3 $HOME/.cursor/skills/graph-retrieval-memory/scripts/graph_retrieve.py` (finds graph via `__file__`). `/graph-retrieve` → `~/.cursor/commands/graph-retrieve.md`. `sessionStart` hook `~/.cursor/hooks/graph-retrieval-session.sh` sets `VOID_REPO` / `GRAPH_RETRIEVE` / `VOID_MEMORY` (telemetry `preToolUse` kept). Prefer `agent --workspace $HOME/Desktop/V.O.I.D`. Project allowlist: `Desktop/V.O.I.D/.cursor/cli.json`.
- Cursor CLI first-turn memory is a bounded GOAL + latest WORKSPACE excerpt, not a product Memories toggle. Payload builder `build_payload` — `.agents/skills/graph-retrieval-memory/scripts/cursor_cli_memory.py:121`. Hook execs that helper — `scripts/cursor_cli_session_hook.sh:17`. User `sessionStart` — `~/.cursor/hooks.json:9`. Project `sessionStart` — `.cursor/hooks.json:4`. User rule `alwaysApply: true` — `~/.cursor/rules/graph-retrieval-memory.mdc:3`. Live hook JSON this session: env `VOID_REPO`/`VOID_GOAL`/`VOID_MEMORY`/`GRAPH_RETRIEVE` plus `additional_context` (ctx_len 7794). Tests: `scripts/cursor_cli_memory_test.py` and `scripts/graph_retrieve_test.py` printed `ok`. This chat's sessionStart already fired before the hook change — new `agent` session required for injection.
- **Final goal retarget (this session):** `.agents/memory/GOAL.md` now names `pegasusX/docs/GLOBAL_SCALE_PROGRAM.md` + `pegasusX/docs/GLOBAL_SCALE_LOCAL_ECOSYSTEM.md` as the destination. `PROD_ECOSYSTEM_GOAL.md` stays Class A coverage only — banners on both GS docs + `DOCS_SOURCE_OF_TRUTH.md`. This is a pointer change, not a code-path claim.

## Verified 2026-08-16 (GS-U client visualization plan)

- GS-U plan written at `pegasusX/docs/GLOBAL_SCALE_CLIENT_UI.md` (U0–U9). Destination for dashboards + Plan & Brain tabs. U0+U1+UN+UF+U2+U3+U4+U5+U6+U7+U8 shipped; U9 open. Pointers: `GLOBAL_SCALE_PROGRAM.md`, `GLOBAL_SCALE_LOCAL_ECOSYSTEM.md`, `DOCS_SOURCE_OF_TRUTH.md`, `.agents/memory/GOAL.md`.
- GS-U5 factory command: `GET /v1/factory/dashboard` emits `source` (`spanner|memory|empty`), `plane: factory_trucks`, live `transfers_by_state` + `manifests_by_state` (`factory/dashboard_rollups.go` + `HandleDashboard`). Memory only when `FACTORY_PORTAL_SEED`. Portal `/` StatusStacks + “Factory trucks”. iOS/Android same stacks. Tests: Go empty/memory/spanner; portal no last-mile boards; Android + iOS decode. U-T4 closed.
- GS-U4 warehouse command: `GET /v1/warehouse/ops/dashboard` emits 17-key `orders_by_status`, 8-key `truck_duty`, `hold_reasons`, `demand_source` (`warehouse/ops_portal.go` + `dashboard_rollups.go`). Idle is AVAILABLE only, not everything-except-IN_TRANSIT. Portal `/` StatusStacks; no date-range theatre; demand empty chip. iOS/Android same stacks. Tests: Go full keys + empty demand; portal no sparkline; Android decode.
- GS-U3 Plan & Brain: `/planning` has Planning | Digital Brain tabs + `?tab=` (`apps/supplier-portal/app/(portal)/planning/page.tsx`). Analytics no longer mounts `PlanningBrainPanel` — “Open in Brain” → `/planning?tab=brain`. `brainForecastLine` / `factoryPlanningDisabledCode` in `packages/types/forecast-confidence.ts`. Brain shows `blocked_reason`; no invented forecast line. Flag-off push 409 `factory_planning_disabled`. iOS/Android same two tabs. `/settings/planning` kept. Place stays off.
- GS-U2 supplier command: `GET /v1/supplier/dashboard` emits `deliveries_*_today` + `manifests_by_state` + pack `currency` (`supplier/portal_ops.go`). Portal `/dashboard` binds 17-key StatusStack, HealthStrip, pack `formatPackMoney`, history via analytics or unavailable. Chip → `/orders?status=`. iOS/Android parse revenue + increment `FISCAL_FAILED`. No invented yesterday Δ / UZS. Truck duty unavailable. iOS poll 60s.
- GS-UF freshness: `cache.DashboardKey` + `WriteJSONWithETag` (`apps/backend-go/cache/dashboard.go`). Supplier/warehouse/factory dashboard GETs 304. Invalidate dash keys on order/manifest commit. Client dirty-slice in `@pegasusx/ws-refresh-contract`. Location → `applyDriverLocationPatch` only. Factory fleet `usePolling` 15s/hidden 60s. Tests: 50 location events → 0 dashboard GETs; ETag 304 units.
- GS-U0 closed warehouse sparkline theatre and supplier 6-key / `len(orders)*10` VU.
- Factory dashboard prefers Spanner counts and always labels `source` — `pegasusX/apps/backend-go/factory/service.go:875` + `factory/dashboard_rollups.go`.
- `/planning` is Plan & Brain (two tabs, `?tab=`). Twin HTTP still last-mile only; factory plane stays unavailable and unmerged — `pegasusX/apps/backend-go/twin/http_handler.go`.
- Full order dictionary is 18 statuses — `pegasusX/packages/types/index.ts:268-289`. Truck duty keys seen in `warehouse/fleet_availability.go` + `bootstrap/bootstrap.go:2471-2495`.
- GS-U extended with **UN** (nav ≤5, shipped) and **UF** (event+cache freshness, shipped). Dashboard poll 60s + ETag + dirty-slice WS — `use-dashboard-data.ts`. WS debounce 500ms — `use-supplier-ws-refresh.ts`. `usePolling` pauses when hidden — `packages/api-client/usePolling.ts`. Factory fleet `usePolling` 15s/hidden 60s — `factory-portal/lib/use-factory-fleet-live-map.ts`. Server cache invalidate-after-commit — `apps/backend-go/cache/cache.go`.

## Verified 2026-08-16 (ecosystem backend + infra readiness)

- Ecosystem / Layer B verdict this session: **NOT READY**. Not terraform apply. Not Stripe/Soliq keys. Not flip `checkout_reads_this`.
- UZ pack `CheckoutReadsThis: false` — `pegasusX/apps/backend-go/auth/market_pack.go:135`. Session returns the flag — `auth/session.go:56`.
- Seed fallback fail-closed when `TenantContextEnforced` (`ssmr`/`production` unless `ALLOW_SEED_FALLBACK`) — `auth/tenant.go:55-71`. `PreferTenantSupplierID` never seeds under enforced+authenticated — `:105-111`.
- `RequireTenant` middleware uses `cfg.TenantContextEnforced`, default **ssmr-only** — `bootstrap/bootstrap.go:356`; mount — `main.go:137`. Production profile checks `auth.TenantContextEnforced()` (ssmr|production) — `bootstrap/config_validate.go:17-19`. Split is live.
- Tenant register constructed + mounted — `bootstrap/bootstrap.go:718` / `:1854`; `platformroutes/routes.go:47-48`; `main.go:176`. Persist + same-txn outbox; `IsRegistered: false`, `NextStep: /setup/business` — `tenantreg/service.go:184-221`.
- Order create: retailer from claims — `order/service.go:2483-2489`. Same-txn `outbox.EmitJSON` EventOrderCreated — `:1410-1411`. Mount `POST /v1/order/create` — `orderroutes/routes.go:40`. Fiscal retry mount — `:54`.
- Outbox relay started when workers run — `runtime_workers.go:22-24`. Kafka publisher required when `REQUIRE_INFRA_ADAPTERS` — `bootstrap/bootstrap.go:538-544`.
- STRIPE/ADYEN/PAYME/CLICK execute via `catalogHonestyExecutor` (`no_live_keys` / `adapter_planned`) — `payment/execution.go:149-152`, `:363-387`. `no_live_keys` → 501 — `payment/service.go:1417-1418`.
- Live payout rail always unimplemented — `auth/payout_pack.go:45-48`.
- Only `cell-uz` is `live: true`; EU/US/KZ planned — `auth/cell_directory.go:10-44`. Session note says only cell-uz is the live GCP cell — `auth/session.go:68`.
- `make cell-plan` never applies — `scripts/cell_plan.sh:2-5`; `Makefile:326-328`. EU tfvars `enable_observability_resources = false`, `jwt_secret = ""` — `infra/terraform/cells/eu/cell.tfvars:36-38`. UZ cell tfvars same observability off — `cells/uz/cell.tfvars:31`.
- Prod overlay: `FISCAL_PROVIDER=MY_SOLIQ`, optimizer replicas **0**, managed Kafka is comments — `infra/k8s/overlays/prod/kustomization.yaml:50`, `:61-72`, `:77-84`. Base Redis TF is `BASIC` not HA — `infra/terraform/main.tf:96-98`. GKE still `enable_autopilot = true` — `gke.tf:44`.
- Staging overlay: `PEGASUSX_ENV=staging`, `FISCAL_PROVIDER=PEGASUS` — `infra/k8s/overlays/staging/kustomization.yaml:81-86`.
- Warehouse-portal payment-config is a read-only `gateways[]` view (not pack catalog / pins) — `apps/warehouse-portal/app/payment-config/page.tsx:25-28`.
- Focused tests this session passed: `go test ./auth/ ./payment/ ./bootstrap/ -run TestSeedFallbackAllowed|…|Honesty`.
- Architecture graph still `generatedAt: null` — `pegasusX/context/architecture-graph.json:4`.

## Verified 2026-08-16 (GS-U6 retailer command)

- GS-U6 pulse rollup: `GET /v1/retailer/control-tower/pulse` emits `source`, 17-key `orders_by_status`, `orders_by_supplier` child facets, `loyalty.enrolled=false` — `retailer/control_tower_pulse.go:27-30` + `:108-130`. Mount `retailerroutes/routes.go:60`. Spanner `GROUP BY SupplierId, Status` on `Orders` (not ParentOrders) — `retailer/dashboard_rollups.go`. Empty pulse stays empty; FISCAL_FAILED at supplier A is on the stack without `/orders`.
- Desktop `/dashboard` binds pulse at 60s + StatusStack + supplier facet + LoyaltyCard — `retailer-app-desktop/app/(dashboard)/dashboard/page.tsx` + `components/dashboard/CommandBoard.tsx`. Chip → `/orders?status=` / `&supplier=`.
- iOS `DashboardView` + Android home parse the same pulse. Tests: Go `TestControlTowerPulseEmptyAndLive` + `TestControlTowerPulseFiscalFailedSupplierFacet`; desktop vitest command-dashboard; Android `DashboardTests.testControlTowerPulseFiscalFailedFacet`; iOS `DashboardCommandTests`. Place stays off. Not terraform apply. Not `checkout_reads_this`.

## Verified 2026-08-16 (GS-U7 field)

- Payload board: `GET /v1/payloader/trucks` attaches `truck_status` + VU from the vehicle's current DRAFT/LOADING/SEALED/DISPATCHED manifest — `payload/service.go:651-667` + `payload/board.go:77-90`. COMPLETED/empty is not invented as DRAFT. `GET /v1/payload/capacity/{id}` stays 410 — `payload/vehicle_capacity.go:11-23`.
- Payload clients group four columns: Android `ManifestBoard.kt:7-8`, iOS `ManifestBoard.swift:4` + `TruckSidebar.swift:95-96`, terminal `manifestBoard.ts` + `TruckSidebar.tsx:57-62`. Empty columns say empty.
- Driver remaining-stops stepper first-class `ARRIVED_SHOP_CLOSED` + `FISCAL_FAILED` — iOS `RemainingStops.swift:12-13` + `HomeView.swift:124-132`; Android `RemainingStops.kt` + `HomeScreen.kt`. `OrderState.isActive` includes both — `Order.swift:59-60`.
- Driver earnings: pack currency or empty/unavailable, no invented UZS — `HomeView.swift:198-217`. History peek from `GET /v1/driver/history` — `APIClient.swift:674-675`. Used VU meter is "VU unavailable", not 0 — `RemainingStopsStepper.swift:105`.
- Tests this session: `go test ./payload/ -run TestCanonicalBoardState|TestGroupBoardColumns|TestHandleTrucks_TruckStatus|TestHandleVehicleCapacity`; driver iOS `driverappiosTests` (iPhone 17 / 26.5); payload iOS `ManifestBoardTests` (iPad Air 11-inch M4 / 26.5).

## Verified 2026-08-16 (GS-U8 platform admin)

- Dead-letter KPI is `SELECT COUNT(*) FROM OutboxDeadLetters` — `platformadmin/ops.go:247-263`. Summary/list omit `dead_letter_count` when unavailable (not invented 0) — `:266-304`. Old `count: len(page)` removed.
- Admin Command board + Ops panel bind `deadLetterHealth` — `admin-portal/lib/deadLetterHealth.ts:17-22` + `components/CommandBoard.tsx` + `OpsPanel.tsx`.
- Pending dual-control: `GET /v1/platform-admin/flags` — `featureflags/handlers.go:176-201` + `ListPending` on memory/Spanner.
- Accuracy GET allows `PLATFORM_ADMIN` — `planning/accuracy_handlers.go`. Table of `mape28`/`demoted`; no invented line. Tenant `IsRegistered` stays unavailable on the platform list.
- Tests: `go test ./platformadmin/ ./featureflags/ ./planning/` (U8 run) + `pnpm --filter @pegasusx/admin-portal test`.
- NEXT after U8 was GS-U9 (closed same day). Not terraform apply. Not `checkout_reads_this`. Not Stripe keys.

## Verified 2026-08-16 (GS-U9 role-row lock)

- iOS StatusStack `onSelect` — `packages/mobile-ios-design/StatusStack.swift:137` + `:181`. Android `onSelect` + `heightIn(min = 48.dp)` + `gs-u-chip-*` — `packages/mobile-android-design/.../StatusStack.kt:145` + `:189-191`.
- Supplier funnel vs coarse: `resolveSupplierOrdersQuery` — `apps/supplier-app-ios/.../OrdersViewModel.swift:138`; Android same name in supplier OrdersViewModel.
- Lock test: `apps/supplier-portal/lib/__tests__/gs-u9-role-row-lock.test.ts` (9 tests passed this session). Supplier/retailer Android `DashboardTests` passed. Supplier iOS `xcodebuild` **unverified** (pre-existing `TreasuryHubView.swift:88` async in sync refresh). Retailer iOS unit tests **unverified** (no `.xcodeproj` found).
- NEXT: leftovers (GS-M flag, cells apply, live PSP). Not Layer B. Not terraform apply. Not `checkout_reads_this`. Not Stripe keys.

## Verified 2026-08-16 (leftovers retrieve — GS-M flag, cells, live PSP)

- Graph walker: leftover query with “not terraform apply” seeds **terraform** nodes (`infra/terraform`, `observability.tf`). Queries `MarketPack checkout_reads_this` and `cell directory home_cell cell-plan` return **no hits**. Payment query seeds `paymentroutes` / `payment/service.go`, not `payment/execution.go` or `payment/catalog.go`. Graph `generatedAt: null` — `pegasusX/context/architecture-graph.json:4`. Hits are paths, not status.
- **GS-M flag PARTIAL (honesty leftover, readers REAL).** UZ `CheckoutReadsThis: false` — `auth/market_pack.go:138-139`. Session advertises it — `auth/session.go:56`. Mount `GET /v1/auth/session` — `platformroutes/routes.go:45`. Checkout still reads shipped currency/PSP — `auth/checkout_pack.go:19-23`. Fiscal pack is law; PEGASUS/FAKE on MY_SOLIQ pack allowed outside production — `auth/fiscal_pack.go:81-83`; `order/fiscal.go:167-170`. SSMR example is `FISCAL_PROVIDER=PEGASUS` — `.env.ssmr.example:117`. Tests assert flag stays false — `auth/market_pack_test.go:16-17`. Do **not** flip the flag.
- **Cells PARTIAL (directory REAL; apply GONE from this slice).** Only `cell-uz` `live: true`; EU/US/KZ planned — `auth/cell_directory.go:40-44`. List note: only cell-uz is the live GCP cell — `auth/session.go:66-68`. Mount `GET /v1/platform/cells` — `platformroutes/routes.go:50`. `make cell-plan` never applies — `scripts/cell_plan.sh:1-5`; `Makefile:326-328`. EU tfvars `jwt_secret=""` + observability off — `infra/terraform/cells/eu/cell.tfvars:36-38`. UZ cell same observability off — `cells/uz/cell.tfvars:31-32`. `terraform-apply` Makefile target exists (`Makefile:320-321`) — do **not** run it for leftovers. `Suppliers.MarketCode` / `HomeCell` nullable — `schema/spanner.ddl:18-19`.
- **Live PSP PARTIAL (catalog + honesty REAL; charges not live).** Registry: CASH+GLOBAL_PAY `live`; PAYME/CLICK `unkeyed`; STRIPE/ADYEN `planned` — `payment/catalog.go:35-42`. Checkout-init STRIPE/ADYEN/PAYME/CLICK → `catalogHonestyExecutor` (`adapter_planned` / `no_live_keys`) — `payment/execution.go:149-152`, `:363-387`. `no_live_keys` → 501 — `payment/service.go:1417-1418`. GLOBAL_PAY has a real executor; missing creds → `errGlobalPayCredentialsMissing` (not a fake redirect) — `payment/global_pay_executor.go:60-62`, `:251-279`. Webhooks mount Stripe/Adyen/Payme/Click/GlobalPay — `webhookroutes/routes.go:23-27` (ingress ≠ live checkout-init). Live payout rail always unimplemented — `auth/payout_pack.go:45-48`.
- Warehouse payment-config is pack `catalog` + `gateways` + `currency_code` (not gateways-only theatre) — `warehouse/payment_config.go:27-35`; portal binds `catalog` — `apps/warehouse-portal/app/payment-config/page.tsx:25-26`. GET/POST mounted — `warehouseroutes/routes.go:156-157`.
- Focused tests this session: `go test ./auth/ ./payment/ -run TestResolveMarketPack_UZShipped|TestHandleSession_UZPack|TestHandleListCells|TestHandleListMarketPacks|TestProviderExecutionRouter_|TestAPIURLForHomeCell_Catalog` passed.
- **VERDICT: PARTIAL / NOT READY for Layer B.** Closing these leftovers *is* Layer B (Soliq keys so SSMR can match pack, terraform apply of a second cell, Stripe/Payme keys). No next product-code slice. No U-motion. No flag flip. No `terraform apply`. No Stripe keys.

## Verified 2026-08-16 (leftover slice — GLOBAL_PAY unkeyed honesty)

- Unkeyed GLOBAL_PAY checkout-init/capture/status/refund is `501 no_live_keys`, not a fake redirect and not generic 502 — `payment/global_pay_executor.go:64-73` + `:150` / `:221` / `:291` / `:356` / `:411`. HTTP map already 501 for that code — `payment/service.go:1417-1418`.
- SSMR cash fallback treats `no_live_keys` like merchant-auth miss — `cmd/ssmr-smokecheck/e2e_payment.go:139-143`.
- Test: `TestProviderExecutionRouter_GlobalPayUnkeyed` — `payment/honesty_test.go:45-68`. `go test ./payment/` passed this session. No Stripe keys. Flag still false. No terraform apply.

## Verified 2026-08-16 (plan leftovers inventory + M5 NewService)

- CLIENT_UI U1 leftover kit: TimeSeries / ComboCapacity / BulletMeter / EntityBoard / CommandGrid **not** in `packages/ui-kit/src/portal/index.ts` (HealthStrip + RangeToggle exist). UN leftover: driver/payload/platform nav. UF leftover: other factory page polls. Motion deferred. Not a U-motion slice.
- LOCAL_ECOSYSTEM: W1–W26 claimed shipped. Maps SDK leftover continuous. C2 plan-only. JazzCash named unkeyed later. W20 delivery-fee empty now pack (order path was inventing UZS).
- INFRA: C1–C5 plan-only. Redis `BASIC` — `infra/terraform/main.tf:98`. Deep UZS leftover. **DOC DRIFT:** INFRA still says Stripe/Adyen are theatre redirects; code is `catalogHonestyExecutor` — `payment/execution.go:149-152`.
- Closed this session: factory/payload/driver empty NewService currency + order fee JSON use `auth.CurrencyFromContext`, planned pack stays empty. `go test ./factory/ ./payload/ ./driver/ ./order/` passed. Flag still false. No apply. No Stripe keys.

## Verified 2026-08-16 (origin/main cartograph — fbfd134e)

Surveyed `origin/main` `fbfd134ebb75a6de4904a5ce775320f544c091f3` only. Dirty-tree MarketPack / cells / `catalogHonestyExecutor` are **not** on this SHA.

- Lifecycle: `main.go` mounts chi after `bootstrap.NewApp`. `RequireInfraAdapters` default true — `bootstrap/bootstrap.go:287` + `:326`.
- Order create REAL: `POST /v1/order/create` — `orderroutes/routes.go:40`. Desktop cart is `POST /v1/checkout/unified` — `paymentroutes/routes.go:29` → same `Create`. `HandleCreate` uses `claims.Subject` — `order/service.go:2311` + `:2342`. Same-txn `OutboxEvents` `ORDER_CREATED` — `:1298`. Create does **not** Broadcast WS (stale comment). Fan-out only after `void-notification-dispatcher` — `kafka/notification_dispatcher.go:110`. Order mutator ignores `ORDER_CREATED` (`order/consumer.go:32-70`). Relay — `outbox/relay.go:48`; start — `runtime_workers.go:16`.
- Warehouse pick: H3 **res 9** + coverage radius — `order/warehouse_resolver_spanner.go:29` + `:47`. `ResolveServingWarehouse` / `ParentOrders` **GONE** on this SHA.
- Fiscal retry REAL: `POST /v1/order/{orderID}/fiscal/retry` — `orderroutes/routes.go:54`.
- WS: one upgrade `GET /v1/ws` — `ws/handler.go:84`. Seven hubs; local fan-out + Redis `ws:<hub>:fanout` fail-open — `ws/hub.go:3`. JWT supplier role is `ADMIN` — `auth/claims.go:20`.
- Payment on this SHA: STRIPE/ADYEN are `staticProviderExecutor` fake redirects — `payment/execution.go:145-156` + `:345`. No `/v1/payment/redirect/*` mount. GLOBAL_PAY has a real executor. PAYME/CLICK execute **absent**; webhooks mounted — `webhookroutes/routes.go:23-27`.
- Platform routes have no session / market packs / cells / tenant register — `platformroutes/routes.go:20-38`. `auth/` on this commit is JWT + scopes only.
- Terraform declares Redis `BASIC` — `infra/terraform/main.tf:57`. GKE Autopilot `enable_gke` default false — `gke.tf:1` + `:40`. Kafka is Secret Manager strings, not a cluster — `main.tf:84`.
- Dual truck tables: `SupplierTruckManifests` `:798` and `FactoryTruckManifests` `:884` in `schema/spanner.ddl`.
- Cartograph app (visualization, not a role row): `pegasusX/artifacts/cartograph/`. Built this session.

## Verified 2026-08-16 (Layer B sandbox rename + CI — NOT READY)

- **NOT READY FOR LAYER B.** No terraform apply. No Stripe/Soliq keys. Flag still false. Live GCS prefix unchanged — `infra/terraform/backend-ssmr.hcl:3` (`pegasusx/ssmr`).
- Env class: `sandbox|ssmr` → sandbox — `auth/env.go:40-42`. `TenantContextEnforced` defaults via `IsEnforcedEnv` — `auth/tenant.go:42-51`. Bootstrap middleware default `auth.IsSandbox()` — `bootstrap/bootstrap.go:356`.
- Memory warehouse ID fail-closed — `warehouse/service.go:855-856`. Multi-supplier checkout default follows `IsSandbox()` — `order/multi_supplier_checkout.go:28-31`.
- Two `register()` in parent-order smoke — `cmd/ssmr-smokecheck/e2e_parent_order.go:29-42`. Planned-pack 404 + cell-eu not live — `cmd/ssmr-smokecheck/e2e_sandbox_proof.go`.
- Compose/env: `infra/docker-compose.sandbox.yml`, `.env.sandbox.example` (`PEGASUSX_ENV=sandbox`). `make test-ssmr-infra` aliases `test-sandbox-infra`.
- CI: `.github/workflows/sandbox-infra.yml`; ssmr shim `.github/workflows/ssmr-infra.yml`; reusable unit `.github/workflows/reusable-go-unit.yml`. Cell isolation job in `pegasusx-ci.yml`. Apply-guard `scripts/ci_no_unattended_terraform_apply.sh`.
- Tests this session: `go test ./auth/ ./order/ ./warehouse/ ./fiscal/ ./storage/ ./factory/ ./payload/ ./outbox/ ./bootstrap/ ./retailer/ ./staffinvite/ ./simulator/ -short` passed. `go build ./cmd/ssmr-smokecheck` passed. `kubectl kustomize overlays/sandbox` passed. `make test-sandbox-infra` (compose) **unverified**.

## Verified 2026-08-16 (M5 leftover — fiscal/claims/AR/payout empty currency)

- Empty receipt/claim/invoice/payout currency reads shipped pack via `auth.CoalesceCurrency` / `CurrencyFromContext`; planned pack fail-closed — never invent `UZS`. Helpers: `order/currency_picker.go:142-150` (`fiscalCurrency`), `claims/service.go:980-988` (`claimCurrency`), `payout/payout.go:300-308` (`coalescePayoutCurrency`). Callers: PEGASUS create+corrective `order/fiscal_provider_pegasus.go:59` + `:147`; MY_SOLIQ `order/fiscal_provider.go:293`; GLOBAL_PAY receipt `order/fiscal_provider_globalpay.go:78`; retailer+driver claims `claims/service.go:452` + `:618`; AR open `ar/service.go:129-134`; payout generate `payout/payout.go:145-148`. Compliance mismatch empty uses pack (no invent) — `order/compliance_audit.go:550-553`; credit-freeze CSV uses pack currency — `:242` + `:283`.
- Remaining runtime invents: EDI (`partner/edi/invoic.go`, `inbound_response.go`, `breadth.go`), billing (`internal/services/billing/fees.go`, `kafka/billing_tier_worker.go`), `globalproducts/service.go`, auto-order worker (`retailer/auto_order_worker.go:493`, place stays off).
- Tests this session: `go test ./order/ ./claims/ ./ar/ ./payout/ -count=1` passed. Flag still false. Not terraform apply. Not Stripe/Soliq keys.

## Verified 2026-08-16 (M5 leftover — EDI-lite empty currency)

- Outbound INVOIC/PRICAT/REMADV empty currency uses shipped pack via `ediCurrency` — `partner/edi/currency.go:12-17`; INVOIC `invoic.go:36-37`; PRICAT `breadth.go:199`; REMADV `breadth.go:612`. Planned pack stays empty (no `UZS` invent).
- Inbound ParseINVOIC no longer stamps `UZS`; reads MOA component 3 when present — `inbound_response.go:158-161`. Empty inbound stays empty.
- Remaining invents: billing fees/tier, globalproducts, auto-order worker (place off), partner export-journal SQL `COALESCE(..., 'UZS')` — `partner/export_journals.go:215`.
- Tests this session: `go test ./partner/edi/ ./partner/ -count=1` passed. Flag still false. Not terraform apply. Not Stripe keys.

## Verified 2026-08-16 (M5 leftover — billing empty currency)

- `ZeroSchedule` no longer invents `UZS`; missing schedule uses pack via `packCurrencyOrEmpty` — `internal/services/billing/fees.go:32-34` + `:80` + `:130-135`.
- Billing meter operating currency from shipped pack; planned/unknown pack skips metering (no UZS invent) — `kafka/billing_tier_worker.go:27-37` + `:94-102`.
- Remaining invents: globalproducts, auto-order worker (place off), partner export-journal SQL `COALESCE(..., 'UZS')` — `partner/export_journals.go:215`, fxrates seed empty operating — `fxrates/seed.go:26`.
- Tests this session: `go test ./internal/services/billing/ ./kafka/ -count=1` passed. Flag still false. Not terraform apply. Not Stripe keys.

## Verified 2026-08-16 (M5 leftover — globalproducts empty currency)

- Catalog offer currency from shipped pack via `offerCurrency` — `globalproducts/service.go:310-315`. `linkOffer` — `:211`. Match-queue ACCEPT — `:269`. Planned pack stays empty (no `UZS` invent).
- Remaining invents: auto-order worker (place off) — `retailer/auto_order_worker.go:368` + `:493`; partner export-journal SQL `COALESCE(..., 'UZS')` — `partner/export_journals.go:215`; fxrates seed empty operating — `fxrates/seed.go:26`.
- Tests this session: `go test ./globalproducts/ -count=1` passed. Flag still false. Not terraform apply. Not Stripe keys.

## Verified 2026-08-16 (M5 leftover — auto-order cart empty currency)

- Draft and place-fallback cart currency from shipped pack via `autoOrderCartCurrency` — `retailer/auto_order_worker.go:1020-1028`. Draft upsert — `:368`. Place-unavailable fallback — `:493`. Planned pack stays empty (no `UZS` invent). Place stays off (`AutoOrderPlaceEnabled` not flipped).
- Remaining invents: partner export-journal SQL `COALESCE(..., 'UZS')` — `partner/export_journals.go:215` + `:222`; fxrates seed empty operating — `fxrates/seed.go:24-26`.
- Tests this session: `go test ./retailer/ -count=1` passed (`TestAutoOrderCartCurrency_*`, `TestAutoOrderDraftCart_*`, `TestAutoOrderPlaceFallbackCart_*`). Flag still false. Not terraform apply. Not Stripe/Soliq keys.

## Verified 2026-08-16 (M5 leftover — partner export-journal empty currency)

- Credit-note journal SQL no longer invents `UZS`; empty order currency is `COALESCE(o.Currency, '')` — `partner/export_journals.go:207-209`. Empty journal currency from shipped pack via `journalCurrency` — `:349-356`. Credit-note rows — `:262`. AR — `:193`. Payment ledger — `:335`. Planned pack stays empty.
- Remaining invent: fxrates seed empty operating — `fxrates/seed.go:24-26`.
- Tests this session: `go test ./partner/ -count=1` passed (`TestJournalCurrency_*`, `TestCreditNotesJournalQuery_EmptyCurrencyNoInvent`). Flag still false. Not terraform apply. Not Stripe/Soliq keys.

## Verified 2026-08-16 (M5 leftover — fxrates seed empty operating)

- Empty operating currency reads shipped pack via `seedOperatingCurrency` — `fxrates/seed.go:69-76`. Identity seed — `:26` + `:36-42`. Planned/unknown pack skips identity (no `UZS` invent) — `:43-45`. Optional USD/UZS pair only when `USDToUZSScaled > 0` — `:47-54`.
- Named empty-currency invent train closed (fiscal/claims/AR/payout/EDI/billing/globalproducts/auto-order/journals/fxrates). Continuous UZS (not this train): smokecheck empty-op fallbacks, `auth/seed_scope.go` catalog, `notifications/inbox_format.go:221`, simulator fixtures, UZ pack catalog.
- Tests this session: `go test ./fxrates/ -count=1` passed. Flag still false. Not terraform apply. Not Stripe/Soliq keys. NEXT leftovers: GS-M flag, cells apply, live PSP — not Layer B.

## Verified 2026-08-16 (continuous leftover — price-override inbox currency)

- Retailer price-override inbox copy uses shipped pack via `priceOverrideCurrency` — `notifications/inbox_format.go:386-393`. Call site — `:223`. Planned pack stays empty (no `UZS` invent). Event payload still has no `currency` field; formatter reads pack from `supplier_id`.
- Remaining continuous UZS: smokecheck empty-op fallbacks (`cmd/ssmr-smokecheck/e2e_fx_rates.go:52-53`), sandbox catalog seed (`auth/seed_scope.go:226`), simulator fixtures. UZ pack catalog is product law.
- Tests this session: `go test ./notifications/ -count=1` passed. Flag still false. Not terraform apply. Not Stripe/Soliq keys. Not Layer B.

## Verified 2026-08-16 (continuous leftover — smokecheck empty operating currency)

- Empty smokecheck operating currency reads shipped pack via `smokeOperatingCurrency` — `cmd/ssmr-smokecheck/smoke_currency.go:10-18`. FX identity assert — `e2e_fx_rates.go:51-53`. Settlement convert — `e2e_fx_settlement.go:38-42`. Order currency picker — `e2e_order_currency_picker.go:65-71`. Planned pack skips the marker (no `UZS` invent).
- Remaining continuous UZS: sandbox catalog seed (`auth/seed_scope.go:226`), simulator fixtures, explicit UZ smoke payloads (retailer/warehouse/soliq). UZ pack catalog is product law.
- Tests this session: `go test ./cmd/ssmr-smokecheck/ -count=1` passed. Flag still false. Not terraform apply. Not Stripe/Soliq keys. Not Layer B.

## Verified 2026-08-16 (continuous leftover — sandbox catalog seed currency)

- Demo Products currency uses `seedSupplierCurrency` (env or shipped pack; planned pack empty) — `auth/seed_scope.go:226` + `:400-408`. No `"UZS"` invent in seed_scope. Country seed still defaults UZ (`seedSupplierCountry`) — not this slice.
- Remaining continuous UZS: simulator fixtures (`cmd/ecosystem-simulator/simulator.go:309`), explicit UZ smoke payloads. UZ pack catalog is product law.
- Tests this session: `go test ./auth/ -count=1` passed. Flag still false. Not terraform apply. Not Stripe/Soliq keys. Not Layer B.

## Verified 2026-08-16 (continuous leftover — simulator fixture currency)

- Simulator order/webhook currency from shipped pack via `simOperatingCurrency` — `cmd/ecosystem-simulator/sim_currency.go:10-27`. Place order — `simulator.go:307-313`. Prepay — `:346`. Global Pay success/fail webhooks — `:476` / `:505`. Empty/planned pack fails closed (`empty_operating_currency`), never invents `UZS`.
- Remaining continuous UZS: smokecheck explicit payloads (`e2e_retailer.go:271`, warehouse/soliq/parent-order/global-products). `countrycfg.UZDefault` and UZ pack catalog are product law.
- Tests this session: `go test ./cmd/ecosystem-simulator/ -count=1` passed. Flag still false. Not terraform apply. Not Stripe/Soliq keys. Not Layer B.

## Verified 2026-08-16 (continuous leftover — smokecheck explicit payloads)

- Smokecheck request/seed payloads use `smokeOperatingCurrency` — `cmd/ssmr-smokecheck/smoke_currency.go:10-18`. Quote — `e2e_retailer.go:269`. Delivery-fee rules — `e2e_warehouse.go:77`. Soliq live/contract — `e2e_soliq.go:29` (skip if empty). Parent-order product seed — `e2e_parent_order.go:103` + `:228` + `:248`. Global products offers — `e2e_global_products.go:43`. Planning/AR fixtures — `e2e_forecast_accuracy.go:38`, `e2e_safety_stock.go:40`, `e2e_safety_stock_replay.go:36`, `e2e_forecast_algo.go:38`, `e2e_collections.go:35`. Planned pack skips the insert/marker (no `UZS` invent). Package `"UZS"` remains only in `smoke_currency_test.go` (UZ pack assert).
- Named + continuous empty-currency invent train closed. Remaining non-test `= "UZS"`: UZ pack catalog `auth/market_pack.go:120`, `countrycfg.UZDefault` `:61` (product law), optional FX `USD/UZS` when scaled > 0 (`fxrates/seed.go:47-53`), Global Pay `case "UZS":` (`simulator/global_pay.go:178`). Out-of-train leftover: `apps/ai-worker/synthesis/engine.go:357` empty fallback invents `UZS` — not this slice, not re-armed.
- Tests this session: `go test ./cmd/ssmr-smokecheck/ -count=1` passed. Flag still false (`auth/market_pack.go:139`). Not terraform apply. Not Stripe/Soliq keys. Not Layer B. NEXT: GS-M flag / cells apply / live PSP (do not execute). Do not re-arm leftover loop.

## Verified 2026-08-16 (Layer B readiness plan file)

- Phased plan written at `pegasusX/docs/LAYER_B_ECOSYSTEM_READINESS_PLAN.md` (LB-0…LB-G + gated LB-B). Pointers: `LAYER_B_SANDBOX_READINESS.md`, `DOCS_SOURCE_OF_TRUTH.md`. **VERDICT still NOT READY FOR LAYER B.** Flag still false — `auth/market_pack.go:139`. RequireTenant cfg default still `IsSandbox()` — `bootstrap/bootstrap.go:356`. No terraform apply. No Stripe/Soliq keys. Do not execute LB-B from the plan file.

## Verified 2026-08-16 (LB-1 Tenant RequireTenant production default)

- `LoadConfig` sets `cfg.TenantContextEnforced` from `auth.TenantContextEnforced()` — `bootstrap/bootstrap.go:357-358`. Mount unchanged `RequireTenant(cfg.TenantContextEnforced)` — `main.go:137`. Production + unset flag: helper true (`auth/tenant.go:42-51`) and cfg true (`TestConfigTenantContextEnforced_ProductionDefault`). Staging/local stay off. `go test ./auth/ ./bootstrap/ -count=1` passed. Flag still false. Not terraform apply. Not Stripe/Soliq keys. Not READY FOR LAYER B. NEXT: LB-2 doc drift.

## Verified 2026-08-16 (LB-2 honesty doc drift)

- Living GS docs no longer call Stripe/Adyen checkout-init a theatre redirect. INFRA §0 + FEATURES §17/BF-269 + LOCAL §2.5/K2 name `catalogHonestyExecutor` (`adapter_planned` / `no_live_keys`). Frozen `session-2026-08-07/` left. Flag still false. Not terraform apply. Not Stripe keys. Not READY FOR LAYER B. NEXT: LB-3 secret shells.

## Verified 2026-08-16 (LB-3 secret shells)

- TF GSM shells for Soliq + PlayMobile — `infra/terraform/fiscal_sms_secrets.tf`. Live ESO `spec.data` unchanged (atomic). Names commented on sandbox/ssmr ExternalSecrets. Deployment env refs `optional: true`. `kubectl kustomize` sandbox+prod + apply-guard green. No terraform apply. No secret values. Flag still false. NEXT: LB-4.

## Verified 2026-08-16 (LB-4 sandbox fiscal fail-closed)

- Sandbox default still `FISCAL_PROVIDER=PEGASUS`. `SignerFromEnv` forbids `dev-hmac` in sandbox/ssmr/production — `fiscal/signer_env.go:62-64`. `TestSignerFromEnv_SandboxMYSoliqMissingCreds` covers missing signer + missing pkcs12. `go test ./fiscal/ ./order/ ./bootstrap/ ./cmd/ssmr-smokecheck/ -count=1` passed. Flag still false. Not READY FOR LAYER B. NEXT: LB-5 ai-worker UZS.

## Verified 2026-08-17 (LB-5 ai-worker empty currency)

- AI_PREORDER currency from `auth.CoalesceCurrency` — `apps/ai-worker/synthesis/currency.go`. Empty/planned skips insert — `engine.go:356-362` (no `"UZS"` invent). Seed default from shipped pack — `config.go:31` + `seedCurrencyFromPack`. `go test ./apps/ai-worker/synthesis/ ./apps/ai-worker/ -count=1` passed. Flag still false. Not terraform apply. Not Stripe/Soliq keys. Not READY FOR LAYER B. NEXT: LB-6 sandbox compose proof.

## Verified 2026-08-17 (LB-6 sandbox proof PARTIAL)

- `kubectl kustomize` sandbox + prod OK. Apply-guard OK. Focused Go OK. `make test-sandbox-infra` **PARTIAL**: Docker daemon up. Smoke fixes: no dual `backend-setup`; seed lookup by `seed.DefaultSupplierID` (`cmd/ssmr-smokecheck/main.go`) because `EnsureDemoScopeLinks` rewrites Name to `SSMR Smoke Supplier` (`auth/seed_scope.go:170-172`). Passed `spanner`/`spatial`/`kafka`. Failed `e2e` `PUT /v1/supplier/topology` 500 `persist_supplier_topology_failed`. Persist error now logged. Flag still false. Not terraform apply. Not READY FOR LAYER B.

## Verified 2026-08-17 (LB-6 continue — topology + retailer + parent-order)

- `ReplaceTopology` outbox now uses `outbox.EventRowMap` via `portalOutboxMutation` — `supplier/portal_admin_ops.go:424-427`. `OutboxEvents.SupplierId` is NOT NULL — `schema/spanner.ddl:691`. Live smoke printed `PX_E2E_TOPOLOGY_EDIT_OK`.
- Same EventRowMap stamp on retailer/order/credit/catalog/factory/returns/warehouse/payment/claims writers this session. Retailer register + order create logged live (`retailer registered` / `order created`).
- Sandbox `envOr("SSMR_SMOKE_SUPPLIER_ID")` is empty when unset (`cmd/ssmr-smokecheck/main.go:465-476`). `smokeSupplierID()` falls back to `seed.DefaultSupplierID` — `e2e_auth_helpers.go`. Env examples list the seed id.
- Parent-order second tenant had no covering warehouse (`zone_miss`). Register now logs in and `PUT /v1/supplier/topology`; seed warehouse also stamps `CountryCode`. Live smoke printed `PX_E2E_PARENT_ORDER_SPLIT_OK`.
- Import async host path aligned to compose mount `.ssmr/import-uploads` — `scripts/smoke_sandbox.sh`. Apply no longer 500s on replayed staged `row_index` (`import_sessions_apply.go`). Live `PX_E2E_SUPPLIER_IMPORT_ASYNC_OK`.
- Latest `make test-sandbox-infra` still fails e2e at factory lifecycle dispatch: `no_available_drivers` / missing `manifest_id` for `factory-demo-1`. Dispatch execute earlier printed `PX_E2E_WAREHOUSE_DISPATCH_EXECUTE_OK`. Flag still false. Not terraform apply. Not READY FOR LAYER B.

## Verified 2026-08-17 (LB-6 factory leftover + import async)

- Factory AUTO needs FACTORY-home on-shift driver with VehicleId. Seed now writes `veh_factory_1` and assigns `drv_factory_1` — `auth/seed_scope.go`. Live `factory.dispatch.committed` + `PX_E2E_FACTORY_MANIFEST_LIFECYCLE_OK`.
- Transfer seed `ssmr-factory-tr-{unixnano}-{i}` overflowed `FactoryInternalTransfers.OrderId STRING(36)` — 500 `transfer_create_failed`. Handler now 400 `order_id_too_long` — `factory/service.go`. Smoke IDs are `ord_ftr_%d_%d`; ssmr-a/b pin `transfer_ids` so AUTO does not pack both. Live `PX_E2E_FACTORY_PAYLOAD_OVERRIDE_OK`.
- Cancel-transfer wrote exceptions only in memory; GET `/v1/factory/manifest-exceptions` lists Spanner JOIN `FactoryTruckManifests` — empty. `FactoryTx.SaveException` + `apply()` persist — `factory/repository_spanner.go` / `apply.go`. Unit: `TestHandleManifestCancelTransfer_IdempotentAlreadyCancelled`. **Not live-proven** (later smoke died earlier).
- Import async: backend had no `IMPORT_LOCAL_FILE_ROOT`; worker GCS-first could miss the file. Local-first opener + sandbox mount + same-process discover after uploaded. Live `PX_E2E_SUPPLIER_IMPORT_ASYNC_OK` once; latest fail `apply 409 session not ready for apply` (after discovery OK). Flag still false. Not terraform apply. Not READY FOR LAYER B.

## Verified 2026-08-17 (LB-6 factory closed + import async + payload start-loading)

- Import async apply 409: missing optional `reorder_threshold` was `fmt.Sprint(nil)` → `"<nil>"` → `invalid_reorder_threshold`. Fixed `importCleanedString` — `supplier/import_discovery.go`. Discover uses wizard topology warehouses + 30s ctx. Apply now returns `no_applicable_rows` not a generic state 409. Live `PX_E2E_SUPPLIER_IMPORT_WIZARD_OK` + `PX_E2E_SUPPLIER_IMPORT_ASYNC_OK`.
- Factory exceptions persist live — `PX_E2E_FACTORY_MANIFEST_EXCEPTIONS_OK`. Loading-bay AUTO now includes unassigned `LOADING` — `factory/dispatch_execute.go` matches `dispatchableTransferState`. Live `PX_E2E_FACTORY_LOADING_BAY_OK` + inbox.
- Payload start-loading 500: `payload/repository_spanner.go` outbox flush omitted `OutboxEvents.SupplierId` (NOT NULL). Now `outbox.EventRowMap`. Live past start-loading (`PX_E2E_PAYLOAD_DISPATCH_JOURNEY_OK`). Latest fail: `payload seal order status 400`. Flag still false. Not terraform apply. Not READY FOR LAYER B.

## Verified 2026-08-17 (LB-7 cell/apply guards)

- `scripts/cell_plan.sh:1-5` never apply. `Makefile:344-346` calls that script. Only `cell-uz` live — `auth/cell_directory.go:40-44`. EU `jwt_secret=""` + observability off — `cells/eu/cell.tfvars:36-38`. UZ same — `cells/uz/cell.tfvars:31-32`. Apply-guard still green. No apply. NEXT: LB-G after compose smoke or explicit leftover.

## Verified 2026-08-17 (code-review-pro — pegasusX)

- `/v1/etas/*` mounts with no `RequireRole`; POST recalculate persists `RouteETAs` — `etaroutes/routes.go:16-21`, `eta/persist.go:11-29`. `RequireTenant` passes unauthenticated — `auth/tenant.go:126-129`. Mount — `main.go:413-415`.
- Card checkout uses client `amount_minor` when `> 0`; snapshot only if `<= 0` — `payment/retailer_checkout.go:125-140`. Session stores that amount — `:394-401`. Success webhook emits `PAYMENT_CLEARED` — `payment/service.go:1366-1370`. Consumer fiscalizes full `Orders.TotalMinor` with no paid==due check — `order/external_payment.go:15-43`, `:91`.
- `token_use=ws` tickets attach via `SessionAuth`; `RequireRole` does not reject them; refresh re-issues 24h — `auth/jwt.go:235-254`, `auth/claims.go:165-188`, `auth/refresh.go:41-47`. WS upgrade path does reject non-tickets — `ws/handler.go:111-113`.
- Payout GET scopes by claims; export/dispatch/mark-paid load by `batchID` only — `payout/handlers.go:101-111` vs `:172-174`, `:192-194`, `:218-224`.
- `PUT /v1/admin/fx-rates` is `RequireRole(ADMIN)` (supplier), not `PLATFORM_ADMIN` — `fxrates/handlers.go:36-37`.
- Forecast accuracy `supplier_id` is query, not claims — `planning/accuracy_handlers.go:39-40`, `:60-63`.
- JWT revoke + auth rate-limit fail open on Redis error; Memorystore `tier = "BASIC"` — `auth/jwt.go:265-268`, `bootstrap/redis_rate_limiter.go:70-71`, `infra/terraform/main.tf:103-105`.
- Public Ingress `/` serves `/metrics` and `/debug/infra/redis` — `infraroutes/routes.go:38-41`, `infra/k8s/ingress/ingress.yaml:30-36`.
- Cookie `Secure` hardcoded false — `bootstrap/bootstrap.go:705`, `:725`. SSMR HTTPS redirect off — `infra/k8s/overlays/ssmr/frontendconfig.yaml:13-15`.
- Layer B still **NOT READY**. Flag still false — `auth/market_pack.go:139`. No terraform apply. No Stripe/Soliq keys.

## Verified 2026-08-17 (Redis skills audit — pegasusX)

- Memorystore `tier = "BASIC"` + `maxmemory-policy = allkeys-lru` — `infra/terraform/main.tf:103-113`. JWT denylist / rate-limit / idempotency keys can be evicted.
- Cache circuit fail-closed in production — `bootstrap/bootstrap.go:445-448`. JWT revoke and auth rate-limit **do not** use that circuit — `auth/jwt.go:265-268`, `bootstrap/redis_rate_limiter.go:70-71`.
- Rate-limit Lua uses Sorted Set `rl:{actor}` (correct type) but fail-open on error — `bootstrap/redis_rate_limiter.go:13-42`, `:69-71`.
- `SupplierScopedKey` hash-tag helper exists and is **unused** outside tests — `cache/keys.go:8-10`. Live keys are untagged (`dash:`, `idem:`, `jwt:revoked:`, `orders:supplier:`).
- Delivery perimeter is a Redis **Set** (correct) on a global key `ssmr:delivery_perimeter`; per-supplier keys exist but are not wired — `retailer/proximity_service.go:18-40`.
- Driver last-location is a JSON **String** with TTL 2m — `telemetry/location_store.go:13-19`. GET-then-SET, not a Hash.
- Sandbox compose Redis has **no AUTH** — `infra/docker-compose.sandbox.yml:11-20`. TF AUTH default on + TLS `SERVER_AUTHENTICATION` — `infra/terraform/variables.tf:72-81`.
- No Redis Query Engine / vector / LangCache / Iris in this tree. Do not add (GOAL.md: no side vector DB). Not terraform apply. Flag still false.

## Verified 2026-08-17 (Kafka skills audit — pegasusX)

- State-change path is outbox `EmitJSON` in RW txn, not handler `WriteMessages`. Publisher is `RequiredAcks=all`, Hash key, sync — `outbox/kafka_publisher.go:81-92`. Relay tick 250ms + stuck watchdog — `outbox/relay.go:35-36`, `:88-105`.
- `EmitJSON` injects `trace_id` into JSON body from ctx — `outbox/outbox.go:114-131`. Relay also sets Kafka header `trace_id` + `event_id` — `outbox/kafka_publisher.go:117-118`, `relay.go:272-275`.
- Consumers: manual commit (`CommitInterval: 0`) — `kafka/consumer.go:83-101`. Handler retries then Kafka DLQ; DLQ fail → `ErrSkipCommit` — `:148-157`. Order mutator wrapped `WithEventDedup` — `bootstrap/bootstrap.go:1639`.
- **DOC DRIFT** in `kafka-event-contracts` skill: factory `_ = WriteMessages` is **GONE**. Remaining direct writes: outbox publisher, Kafka DLQ, planning ingest (`planning/ingest.go:39-50`, no `RequireAll`), smokecheck.
- Order consumer parse errors `return nil` (commit, no DLQ) — `order/consumer.go:26-29`. Notification empty `type` same — `kafka/notification_dispatcher.go:83-86`.
- Managed Kafka TF `enable_managed_kafka` default **false**; prod overlay Kafka is comments — `infra/terraform/kafka.tf:10-13`, `overlays/prod/kustomization.yaml:61-72`. Compose RF=1 — `docker-compose.sandbox.yml:72`. Not terraform apply. Flag still false.

## Verified 2026-08-17 (Maglev — do not integrate in-app)

- **No Maglev lookup table in product code.** Edge is GCE Ingress (`ingress.class: gce`) — External HTTP(S) **proxy** LB, not a passthrough Maglev NLB — `infra/k8s/ingress/ingress.yaml:8-36`.
- WS path is a separate Service + cookie affinity + 3600s timeout — `ingress/backendconfig.yaml:13-24`, `overlays/ssmr/service-ws.yaml:14-16`. Cross-pod delivery is Redis Pub/Sub hub — `ws/hub.go:3-13`. Cookie stickiness is optional given the hub.
- Kafka ordering is Hash on aggregate key — `outbox/kafka_publisher.go:81-88`. Do not Maglev-hash consumer groups.
- GKE Autopilot TF default **off** — `infra/terraform/gke.tf:1-5`, `:44`. No Cloud Armor in tree. Docs already say Maglev is LB notes, not the order SM — `docs/ECOSYSTEM_FEATURES_BY_ROLE.md:634`. Not terraform apply. Flag still false.

## Verified 2026-08-17 (push-notifications — pegasusX)

- Device token POST is JWT-scoped persist (claims, not body retailer id) — `platform/handlers.go:180-216`, mount `platformroutes/routes.go:37`. Spanner upsert when client present — `platform/repository.go:168-176`.
- Send path is **FCM Admin** `Send` with registration token — `notifications/fcm.go:97-124`. iOS apps upload **APNs hex**, no Firebase Messaging SDK — `payload-app-ios/.../PushNotificationManager.swift:7-9`, `:50-55`. Those tokens are not FCM tokens.
- Dispatcher FCM + inbox after Kafka — `kafka/notification_dispatcher.go:970-991`. `pushFCM` formats with **actor role** (`FormatFromEvent("DRIVER", …)`), not event type — `:974-990` vs `inbox_format.go:14-20`.
- Retailer iOS manager is **unwired**: no `UIApplicationDelegateAdaptor`, `requestAuthorization` never called — `retailerappApp.swift:12-18`. Warehouse/factory/supplier Android+iOS: no FCM service / no token POST caller (warehouse API exists unused — `WarehouseApi.kt:555`).
- Staging/EU `FCM_ALLOW_NOOP=true` — `overlays/staging/kustomization.yaml:89`. Prod overlay comments say push must be live; keys are ExternalSecret. Not terraform apply. Flag still false.

## Verified 2026-08-17 (LB-6 sandbox compose e2e GREEN)

- `make test-sandbox-infra` live-green this session: `ssmr smokecheck passed` (`spanner`/`spatial`/`kafka`/`e2e`) + `sandbox-ecosystem-marker-gate-ok` + `__SANDBOX_OK__`.
- Seal leftover: pick waves on, lots off. `CreatePickWaveInTxn` now builds bag-of-SKU tasks when `!EffectiveLots` — `stocklots/picking.go`. Confirm skips lot depletion when `LotId` empty. `ReadyAt` uses wall clock (column has no `allow_commit_timestamp`). Warehouse dispatch smoke creates + confirms wave to `READY_TO_SEAL`. Live `PX_E2E_WAREHOUSE_PICK_WAVE_READY_OK` + `PX_E2E_PAYLOAD_SEAL_FLOWS_OK`.
- `OutboxEvents.EventId` STRING(36): `outbox.ClampEventID` in `EventRowMap` — `outbox/outbox.go`. Return-scan + location IDs clamped. Live `PX_E2E_RETURN_GATE_RECEIVE_OK`.
- Reverse-logistics list 403: gap-closure warehouse JWT missing `HomeNodeType` — `cmd/ssmr-smokecheck/e2e_gap_closure.go`. Live `PX_E2E_REVERSE_LOGISTICS_OK` + `PX_E2E_GAP_CLOSURE_OK`.
- Partner retailer key stamped `TenantContext.SupplierID=retailerID` → 422 `supplier_id mismatch`. `withPartnerTenant` stamps isolation key only for `SUPPLIER` keys — `partner/auth.go`. Live `PX_E2E_PARTNER_ORDER_CREATE_OK`.
- Webhook create 422: `PARTNER_WEBHOOK_PING` added to `PartnerWebhookableEvents` — `partner/webhook_events.go`. Live `PX_E2E_WEBHOOK_DELIVERED_OK`.
- UZ `CheckoutReadsThis: false` — `auth/market_pack.go:139`. Only `cell-uz` live — `auth/cell_directory.go:40-44`. Flag still false. Not terraform apply. Not Stripe/Soliq keys. **NOT READY FOR LAYER B.** NEXT: LB-G readiness gate (no LB-B).

## Verified 2026-08-17 (LB-G readiness gate)

- LB-6 compose e2e still the last live green (`__SANDBOX_OK__`). Flag still false — `auth/market_pack.go:139`. Session advertises it — `auth/session.go:56`. Only `cell-uz` live — `auth/cell_directory.go:40-44`. `cell_plan.sh` never apply — `scripts/cell_plan.sh:1-5`. EU `jwt_secret=""` — `cells/eu/cell.tfvars:36-38`.
- Honesty executors still `no_live_keys` / `adapter_planned` — `payment/execution.go:149-152`, `:363-387`. Live payout rail always unimplemented — `auth/payout_pack.go:45-48`. Sandbox forbids `dev-hmac` — `fiscal/signer_env.go:62-64`. Pack fiscal vs PEGASUS runtime keeps the flag false — `order/fiscal.go:167-183`.
- **NOT READY FOR LAYER B.** Remaining is not secrets-only. Ranked code leftovers (opened this session): (1) card checkout uses client `amount_minor` when `> 0` — `payment/retailer_checkout.go:125-140`, session `:394-401`; settle fiscalizes `Orders.TotalMinor` with no paid==due — `order/external_payment.go:15-43`, `:91`. (2) payout export/dispatch/mark-paid load by `batchID` only — `payout/handlers.go:172-174`, `:192-194`, `:218-224`; `ExportBankFile` / `SubmitForDispatch` same — `payout/payout.go:180-204`, `payout/rail.go:145-155`. (3) `token_use=ws` attaches via `SessionAuth`; `RequireRole` does not reject tickets — `auth/jwt.go:250-253`, `auth/claims.go:165-188`; refresh re-issues 24h keeping `TokenUse` — `auth/refresh.go:41-47`. WS upgrade still requires ticket — `ws/handler.go:111-113`. (4) `/v1/etas/*` no `RequireRole`; POST persists — `etaroutes/routes.go:16-21`, mount `main.go:413-415`. (5) `PUT /v1/admin/fx-rates` is supplier `ADMIN` — `fxrates/handlers.go:36-37`. (6) forecast accuracy `supplier_id` query — `planning/accuracy_handlers.go:39-40`, `:60-63`. (7) JWT revoke fail-open — `auth/jwt.go:265-268`. (8) public `/metrics` + `/debug/infra/redis` — `infraroutes/routes.go:38-41`.
- Docs vs code: `GLOBAL_SCALE_PROGRAM.md` leftover is flag + apply + live PSP — match. Code still has money/auth leftovers above. Not terraform apply. Not Stripe/Soliq keys. NEXT: leftover P0 card checkout amount from order snapshot (not LB-B).

## Verified 2026-08-17 (P0 card checkout amount)

- Card checkout no longer uses client amount as authority. Snapshot required; client amount only accepted when it equals order total — `payment/retailer_checkout.go:131-148`, `resolveCardCheckoutAmount`. Session still stores snapshot amount — `:162`.
- `SettleExternalPayment` requires `paidMinor == Orders.TotalMinor` before FISCALIZING — `order/external_payment.go:23-56`, `assertPaidEqualsDue`. Consumer passes `amount_minor` — `order/consumer.go:33-43`.
- Units: `go test ./payment/ ./order/ ./cmd/ssmr-smokecheck/ -count=1` passed. Compose re-run died earlier on import async `409 no_applicable_rows` (not this path). Flag still false. Not terraform apply. Not READY FOR LAYER B. NEXT: leftover P0 payout mutate-by-batchID.

## Verified 2026-08-18 (P0 payout supplier scope)

- Export / dispatch / mark-paid require claims supplier via `requirePayoutBatchScope` — `payout/handlers.go:56-68`, `:187-191`, `:209-214`, `:234-247`. Service methods take `supplierID` and reuse `GetBatch` (foreign → `ErrBatchNotFound`) — `payout/payout.go:180-184`, `:204-207`, `payout/rail.go:145-150`. Rail settlement webhook stays secret-auth M2M — `handlers.go:271-296`.
- `go test ./payout/ -count=1` passed. Flag still false. Not terraform apply. Not READY FOR LAYER B. NEXT: leftover P0 WS tickets (`token_use=ws` via SessionAuth/RequireRole).

## Verified 2026-08-18 (P0 WS ticket not a session)

- `SessionAuth` / `ParseBearerClaims` do not attach `token_use=ws` — `auth/jwt.go:146-148`, `:245-253`. `RequireRole` / `RequireAnyAuthenticated` reject tickets with `ws_ticket_not_allowed` — `auth/claims.go:183-189`, `:202-207`. Refresh rejects tickets (`ws_ticket_not_refreshable`) and clears `TokenUse` on full refresh — `auth/refresh.go:41-50`. Ticket-from-ticket mint refused — `auth/jwt.go:130-132`.
- Query-ticket WS upgrade still works — `ws/handler.go:102-113`; `go test ./auth/ -count=1` + `./ws/ -run 'AcceptsSignedQueryToken|RejectsSessionJWTInQuery|ClaimsFromRequest|RegisterRoutes'` passed. Flag still false. Not READY FOR LAYER B. NEXT: leftover P1 unauthenticated `/v1/etas/*`.

## Verified 2026-08-18 (P1 ETA routes require role)

- GET `/v1/etas/route|stop` requires delivery roles; POST `/recalculate` is warehouse/admin/platform only — `etaroutes/routes.go:16-41`. Nil service 503. Unauthenticated 401 — `etaroutes/routes_test.go`. Residual: no supplier/route ownership check on `routeId`. `go test ./etaroutes/ ./eta/ -count=1` passed. Flag still false. Not READY FOR LAYER B. NEXT: leftover P1 `PUT /v1/admin/fx-rates` is supplier ADMIN not PLATFORM_ADMIN.

## Verified 2026-08-18 (P1 FX PUT platform-admin)

- `PUT /v1/admin/fx-rates` is `PLATFORM_ADMIN` only — `fxrates/handlers.go:32-39`. GET admin still ADMIN or PLATFORM_ADMIN. Supplier portal is read-only via `GET /v1/supplier/fx-rates` — `supplier-portal/.../fx-rates/page.tsx`. Settlement e2e PUT uses platform JWT — `e2e_fx_settlement.go`. `go test ./fxrates/ ./cmd/ssmr-smokecheck/ -count=1` passed. Flag still false. Not READY FOR LAYER B. NEXT: leftover P1 forecast accuracy `supplier_id` query.

## Verified 2026-08-18 (P1 forecast accuracy supplier scope)

- List + run-once resolve supplier from TenantContext/claims for ADMIN; query `supplier_id` only for PLATFORM_ADMIN — `planning/accuracy_handlers.go` `resolveAccuracySupplierID`. Run-once mount allows platform admin — `main.go` ProtectMutations. `go test ./planning/ -count=1` passed. Flag still false. Not READY FOR LAYER B. NEXT: leftover P1 JWT revoke fail-open.

## Verified 2026-08-18 (P1 JWT revoke fail-closed)

- `tokenRevoked` treats store errors as revoked — `auth/jwt.go` `checkTokenRevoked`. SessionAuth does not attach. Refresh returns `503 revocation_store_unavailable` — `auth/refresh.go`. WS query ticket fails closed on store error — `ws/handler.go`. `go test ./auth/ -count=1` passed. Flag still false. Not READY FOR LAYER B. NEXT: leftover P1 public `/metrics` + `/debug/infra/redis`.

## Verified 2026-08-18 (P1 metrics/debug not public)

- `/debug/infra/redis` is `PLATFORM_ADMIN` — `infraroutes/routes.go`. `/metrics` stays scrape-open on the pod for GMP PodMonitoring. Ingress no longer uses path `/`; public paths are `/v1`, `/partner`, `/healthz`, `/ready` — `infra/k8s/ingress/ingress.yaml` + SSMR/sandbox overlays. `go test ./infraroutes/ -count=1` passed. Flag still false. Not terraform apply. Not READY FOR LAYER B. NEXT: LB-G re-verdict (flag + cells apply + live PSP remain Layer B).

## Verified 2026-08-18 (LB-G re-verdict)

- **NOT READY FOR LAYER B.** Last live compose green is still 2026-08-17 `__SANDBOX_OK__`. Leftover batch after that was unit-tested only.
- Flag still false — `auth/market_pack.go:139`; session returns it — `auth/session.go:56`. Only `cell-uz` live — `auth/cell_directory.go:40-44`. `cell_plan.sh` never apply — `scripts/cell_plan.sh:1-5`. Honesty STRIPE/ADYEN/PAYME/CLICK — `payment/execution.go:149-152`. Live payout unimplemented — `auth/payout_pack.go:45-48`. Pack vs PEGASUS keeps flag false — `order/fiscal.go:167-183`.
- Closed leftovers re-opened this session: checkout snapshot — `payment/retailer_checkout.go:131-148`; paid==due — `order/external_payment.go:46-56`; payout scope — `payout/handlers.go:56-68`; WS tickets skipped — `auth/jwt.go:150`; revoke fail-closed — `:261-264`; ETA roles — `etaroutes/routes.go:16-41`; FX PUT platform — `fxrates/handlers.go:37-38`; accuracy claims — `planning/accuracy_handlers.go:96`; redis debug — `infraroutes/routes.go:43-44`; Ingress no `/` — `infra/k8s/ingress/ingress.yaml:30-58`.
- Remaining **code** (not secrets): ETA GET/POST still no route/supplier ownership — `etaroutes/routes.go:52-57`. Retailer `NewService` does not pass `FirebaseVerifier` — `bootstrap/bootstrap.go:656-672` vs `retailer/service.go:511`.
- Docs vs code: `GLOBAL_SCALE_PROGRAM.md` leftover is flag + apply + live PSP — match on those. Code still has residuals above. Not terraform apply. Not Stripe/Soliq keys. NEXT: leftover ETA `routeId` ownership (not LB-B).

## Verified 2026-08-18 (ETA route ownership)

- GET/POST `/v1/etas/*` call `AuthorizeRoute` — `etaroutes/routes.go`. Scope from RouteTwins+Drivers, `SupplierTruckManifests`, `Orders` — `eta/access.go`. Platform admin bypass. Other roles fail-closed (404 `route_not_found`). `AllowRouteAccess` unit-tested. `go test ./eta/ ./etaroutes/ -count=1` passed. Flag still false. Not READY FOR LAYER B. NEXT: leftover retailer FirebaseVerifier omitted at construct (`bootstrap/bootstrap.go:656-672`).

## Verified 2026-08-18 (FirebaseVerifier on role login)

- `newLoginFirebaseVerifier` — `bootstrap/firebase.go`. Passed into retailer/driver/factory/warehouse/payload `NewService` and `App.FirebaseVerifier`. `main.go` reuses that instance for route middleware. Default flag still off. `go test ./bootstrap/ ./retailer/ -count=1` passed. Flag still false. Not READY FOR LAYER B. NEXT: Android TokenManager prefers Firebase ID token as HTTP Bearer.

## Verified 2026-08-18 (Android Bearer is session JWT)

- `TokenManager.getPreferredToken` / `httpAuthorizationToken` use session JWT only — `retailer-app-android/.../TokenManager.kt`. Driver interceptor uses `TokenHolder.token` not `firebaseIdToken` — `driver-app-android/.../NetworkModule.kt`. Firebase ID token stays for login/SDK. `./gradlew :app:testStoreDebugUnitTest --tests com.pegasus.retailer.data.local.TokenAuthTest` passed. Flag still false. Not READY FOR LAYER B. NEXT: web portals still may send Firebase ID as Bearer (`factory-portal/lib/auth.ts`); compose e2e not re-green since leftover batch.

## Verified 2026-08-18 (Firebase Data Connect / SQL Connect)

- **GONE.** No `dataconnect.yaml`, `schema.gql`, `connector/`, `.dataconnect/`, or `*.gql` in `pegasusX/`. Zero source hits for `dataconnect`, `schema.gql`, `SQL Connect`, `getDataConnect`. Did not run `firebase init dataconnect`.
- `firebase.json` emulators are Auth + UI only (`:2-11`). Web Firebase is `firebase/app` + `firebase/auth` — `apps/supplier-portal/lib/firebase.ts:6-15`. Backend Firebase is ID-token verify — `apps/backend-go/auth/firebase.go:1-4`, `:36-38`.
- `@firebase/data-connect` exists only as a transitive dep of the `firebase` umbrella in `pnpm-lock.yaml` (`firebase@12.14.0` → `@firebase/data-connect@0.7.1`). No package.json or source import.
- Product SoT is Spanner + same-txn outbox — `apps/backend-go/schema/spanner.ddl:1-8`, `apps/backend-go/outbox/outbox.go:1-4`. **INTEGRATE: NO** (competing Cloud SQL Postgres + GraphQL plane; GOAL one tenant key).

## Verified 2026-08-18 (Firebase Hosting + App Hosting)

- **GONE** as a hosting plane. `pegasusX/firebase.json` is Auth emulator + UI only (`:1-12`); no `"hosting"` key. No `.firebaserc` in V.O.I.D. No `apphosting.yaml`. No `vercel.json`. No `firebase deploy` / `firebase-tools` in Makefile or `.github/workflows`.
- Class A API is GCE Ingress → `backend-go` / `backend-go-ws` — `infra/k8s/ingress/ingress.yaml:1-36` (`api.pegasusx.app`). SSMR twin `api-ssmr.pegasusx.app` — `infra/k8s/overlays/ssmr/ingress.yaml:8-40`. Prod overlay patches managed cert only — `overlays/prod/ingress-managed-cert-patch.yaml:1-11`. Base kustomize is API+worker, no portals — `infra/k8s/base/kustomization.yaml:6-20`.
- Portals: Tauri static export when `TAURI_BUILD=1` — `apps/supplier-portal/next.config.mjs:17-24` (same pattern warehouse/factory/retailer). Browser supplier uses `/api` Next proxy — `apps/supplier-portal/lib/auth.ts:24-38` + `app/api/[...path]/route.ts`. Only supplier has k8s nginx image — `apps/supplier-portal/deploy/Dockerfile:1-4`, `cloudbuild.supplier-portal.yaml:1-19`. `overlays/ssmr/supplier-portal.yaml` is ClusterIP, **not** listed in `overlays/ssmr/kustomization.yaml`. Warehouse/factory/retailer: no Dockerfile, no k8s. `WS_ALLOWED_ORIGINS` names `*.pegasusx.app` portal hosts with no Ingress rules — `infra/k8s/backend-go/configmap.yaml:33`.
- `apps/marketing-site` exists (`package.json` `next start --port 3004`); `next.config.ts` has no `output: "export"`; no k8s; no Hosting public dir. **INTEGRATE Hosting/App Hosting: NO.** Keep GKE Ingress for API. Do not move Class A API to Firebase Hosting. Marketing Hosting later only if static-exported — not this slice, not App Hosting for portals.

## Verified 2026-08-18 (Firebase skills audit — pegasusX)

- Suite verdict **PARTIAL**. Not firebase init/deploy. Not Firestore/SQL Connect create. Not Layer B keys. MCP catalog listed `plugin-firebase-firebase`; `CallMcpTool` failed (`MCP server does not exist`). CLI `npx firebase-tools@latest` = 15.27.0. No `.firebaserc`. `firebase.json` Auth emulator 9099 + UI 4000 only.
- Client configs `project_id` / `PROJECT_ID` = `pegasus-503013` (Android google-services.json ×6, iOS plists ×6). Base ConfigMap `FIREBASE_PROJECT_ID: pegasusx-prod` — `infra/k8s/backend-go/configmap.yaml:35`. Prod overlay comment says ExternalSecret; `external-secrets/backend-go-externalsecret.yaml` has **no** `FIREBASE_PROJECT_ID` key. Deployment `envFrom` that ConfigMap — `infra/k8s/backend-go/deployment.yaml:36-38`. GSM project in SecretStore is `pegasus-503013`.
- **Auth PARTIAL.** Default flag false in code — `bootstrap/bootstrap.go:326`. K8s base `"true"` — `configmap.yaml:40`. Staging/EU overlay `FIREBASE_AUTH_ENABLED=false`. Phone OTP → `id_token` login then pegasus HS256 JWT — `retailer/auth_login.go:43-50`. Verifier SecureToken x509 — `auth/firebase.go:65-91`, `:137-139`. `FirebaseAuth` permissive pass-through — `:189-206`. Mint custom token Admin SDK — `auth/firebase_admin.go:78-92`; login responses `retailer/auth_login.go:463-472`. Session SoT remains `SessionAuth` — `auth/jwt.go:236-258`, mount `main.go:129`.
- Login OTP is **unwired at composition**: `retailer.NewService` accepts `FirebaseVerifier` (`retailer/service.go:511`) but bootstrap construct omits it (`bootstrap/bootstrap.go:656-672`). Same pattern factory/driver/warehouse/payload NewService. `main.go` passes the verifier to **route middleware only** (`:219-220`), not into those services. Live `id_token` login sees nil verifier. Mint still independent.
- Retailer/driver **Android prefer Firebase ID token as HTTP Bearer** after custom-token exchange — `TokenManager.getPreferredToken` `retailer-app-android/.../TokenManager.kt:40-41`, `NetworkModule.kt:50-53`; driver `NetworkModule.kt:52-59`. WS still uses pegasus JWT — `RetailerWebSocket.kt:112`. iOS retailer API uses pegasus JWT — `AuthManager.swift:233-248`. Payload Android HTTP uses pegasus JWT — `AuthInterceptor.kt:21`.
- **FCM PARTIAL.** Admin Messaging `Send` — `notifications/fcm.go:97-124`. Boot `InitFCM` — `bootstrap/bootstrap.go:1459-1477`. Device-token JWT persist — `platform/handlers.go:180-216`, mount `platformroutes/routes.go:37` (**no** FirebaseAuth on that route). Android messaging: retailer/driver/payload. Factory/warehouse Android auth-only gradle. iOS: no `FirebaseMessaging`. `pushFCM` formats with actor role not event type — `kafka/notification_dispatcher.go:974-990` vs `inbox_format.go:14-20`. Staging/EU `FCM_ALLOW_NOOP=true`.
- **Firestore GONE.** Zero SDK/rules. **SQL Connect GONE** (see prior bullet). **Hosting/App Hosting GONE** (see prior). **Crashlytics GONE** (no `Crashlytics` / `firebase-crashlytics` in pegasusX). **Firebase Storage GONE** — media is GCS signed PUT `storage/gcs.go` + `GET /v1/media/upload-ticket`. google-services `storage_bucket` unused by product code.
- **RTDB / Functions / Remote Config / App Check / Analytics / Perf / In-App / A-B / Dynamic Links / Genkit GONE** in app source. Admin SDK used for Auth custom tokens + FCM only (`firebase.google.com/go/v4`).
- **INTEGRATE:** Auth keep OTP bootstrap only, do not make Firebase session SoT (fix Android preferred-token). FCM keep as push rail after iOS FCM tokens. Crashlytics optional later. Firestore / SQL Connect / Hosting / Storage / RTDB / Functions / Remote Config / App Check: **NO**. Flag still false. Not terraform apply. Not Stripe/Soliq keys. **NOT READY FOR LAYER B.**

## Verified 2026-08-18 (maps / geography / display — pegasusX)

- Matching SoT is H3 **res 7** + pack country — `proximity/node_geography.go:7-22`, `StampNodeGeography` `:42-56`. Checkout same — `order/unified_checkout.go:17`. Settlement / perimeter / leftover `H3CellFromLatLng` are **res 9** — `order/proximity_settlement.go:24-25`, `retailer/proximity_service.go:29`, `proximity/h3_cell.go:3-8`. Exception map buckets res 9 — `supplier/exception_map.go:169`.
- Serving warehouse: pins then closest covering same-country — `proximity/coverage_engine.go:171-192`, `CoversRetailer` `:121-135`. Empty country fail-closed. Resolver live — `order/warehouse_resolver_spanner.go`.
- Route overlay: Google Routes → OSRM → haversine densify 25 m — `routing/builder.go:5-44`, `geometry.go:24-45`. `ROUTING_PROVIDER=auto` — `infra/k8s/backend-go/configmap.yaml:27-30`. OSRM pod requires `/data/region.osrm` or init fails — `infra/k8s/osrm/deployment.yaml:24-27`. Dispatch resequence NN+2-opt — `routing/localsearch.go:12-19`.
- Geocode: Google Geocoding/Places Autocomplete JSON when key set, else Nominatim — `geolocation/service.go:27-40`, `:155-201`. Mounts **no RequireRole** — `geolocation/handlers.go:20-32`. Pack `MapsAdapter: GOOGLE_ROUTES` is routing not display — `auth/market_pack.go:126-128`. Camera helper `PackMapCenter` fail-closed — `auth/maps_pack.go:5-12`; clients `mapInitialViewState` empty → 0,0 zoom 1 — `packages/api-client/market-pack.ts:88-94`.
- Live driver: `POST /v1/telemetry/location` RequireRole DRIVER → Redis last-loc + WS rooms + throttled outbox — `telemetryroutes/routes.go:123-166`. Fleet REST: supplier/warehouse polylines + 15s stale read — `supplier/fleet_live_map.go:34-77`; factory **pins only, geometry deferred** — `factory/fleet_live_map.go:63`. Dual plane: supplier vs factory manifests not merged.
- Display split: web MapLibre + Carto positron (portals); iOS MapKit; last-mile Android Google Maps; supplier/warehouse Android MapLibre Native. Control-tower web uses **Mapbox GL dark-v11**, hardcoded SF camera `-122.4, 37.74` and fallback token — `packages/ui-kit/src/control-tower/HexagonalControlTowerMap.tsx:10-12`, `:43-67`. Used supplier + warehouse portals. Payload apps have **no** map SDK. Factory mobile fetches live-map as **list**, no canvas — `factory-app-android/.../FleetScreen.kt:63-66`. Perimeter Redis global `ssmr:delivery_perimeter` — `retailer/proximity_service.go:18-24`. Flag still false. Not terraform apply. **NOT READY FOR LAYER B** (OSRM extract + Maps key are ops; control-tower camera/token and SDK split are **code**).
- Full write-up: `pegasusX/docs/MAPS_AUDIT.md`. Indexed from `docs/SURFACE_AUDITS.md` + `docs/DOCS_SOURCE_OF_TRUTH.md`. Audit snapshot, not a go-live certificate.

## Verified 2026-08-18 (surface audit suite — pegasusX)

- Agent index: `pegasusX/docs/SURFACE_AUDITS.md`. Audits: `KAFKA_AUDIT.md`, `REDIS_AUDIT.md`, `INFRA_AUDIT.md`, `DEVOPS_CICD_AUDIT.md`, `FIREBASE_AUDIT.md`, `BACKEND_SYSTEM_DESIGN_AUDIT.md`, `UI_SURFACE_AUDIT.md`, plus existing `MAPS_AUDIT.md`. Not terraform apply. Not `checkout_reads_this`. **NOT READY FOR LAYER B.**
- Kafka **PARTIAL.** Same-txn outbox REAL (`outbox/outbox.go`). Publisher `RequiredAcks=all` — `outbox/kafka_publisher.go:81-91`. Prod ConfigMap still `kafka.pegasusx.svc.cluster.local:9092` — `infra/k8s/backend-go/configmap.yaml:14`; Managed Kafka prod overlay comments — `overlays/prod/kustomization.yaml:61-71`. `loggingOutboxPublisher` always nil-success — `bootstrap/outbox_runtime.go:20-28` (blocked when `REQUIRE_INFRA_ADAPTERS`).
- Redis **PARTIAL.** Last-loc key `telemetry:driver:last_location:` TTL 2m — `telemetry/location_store.go:14-19`. TF tier **BASIC** — `infra/terraform/main.tf:103-105` (README HA is drift). Perimeter `ssmr:delivery_perimeter` not called from order create. Retailer Android HTTP is **JWT only** — `TokenManager.kt:33-54` (older Firebase-Bearer WORKSPACE bullet is stale).
- CI **PARTIAL.** Canonical is root `.github/workflows/pegasusx-ci.yml` (Go **1.26.0** vs `go.mod` **1.25**). Nested `pegasusX/.github/workflows/ci.yml` is not executed. Live iOS retailer matrix `reatilerapp.xcodeproj` — `.github/workflows/pegasusx-native-mobile-build.yml:94-96`; on-disk `retailerapp.xcodeproj`. Docker-push has no `needs` on Gate-0.
- Payout rail live flag always false — `auth/payout_pack.go:45-48`. UZ `CheckoutReadsThis: false` — `auth/market_pack.go:138-139`.

## Verified 2026-08-18 (FirebaseVerifier leftover closed)

- Construct **does** pass `newLoginFirebaseVerifier` into retailer/factory/payload/driver/warehouse `NewService` and `App.FirebaseVerifier` — `bootstrap/firebase.go:12-27`, `bootstrap/bootstrap.go:657`, `:673`, `:1033`, `:1051`, `:1205`, `:1343`, `:1883`. Default flag still **false** in code — `bootstrap/bootstrap.go:327`. K8s base `FIREBASE_AUTH_ENABLED=true` — `infra/k8s/backend-go/configmap.yaml:40`.
- `id_token` with nil verifier is **503** `firebase_login_unavailable` (login + factory/warehouse register) — `auth/firebase.go:35-37`, `retailer/auth_login.go:38-40`, `driver/auth_login.go:60-62`, `payload/auth_login.go:58-60`, `factory/auth_login.go:56-58`, `warehouse/auth_login.go:53-55`, `factory/auth_register.go:61-64`, `warehouse/auth_register.go:61-64`. Password/PIN still works when flag is off.
- Older WORKSPACE rows that say bootstrap omits the verifier (`bootstrap.go:656-672`) are **stale**. `FIREBASE_AUDIT.md` §2 rewritten. `go test ./auth/ ./retailer/ ./driver/ ./payload/ ./factory/ ./warehouse/ ./bootstrap/ -count=1` passed. Flag still false. Not READY FOR LAYER B. NEXT: iOS FCM device tokens (not Layer B). ConfigMap `FIREBASE_PROJECT_ID: pegasusx-prod` vs client `pegasus-503013` still split.

## Verified 2026-08-18 (iOS FCM tokens + Firebase project id)

- iOS retailer/driver/warehouse/payload POST **FCM registration tokens** (not APNs hex) to `POST /v1/user/device-token` — AppDelegate `Messaging.apnsToken` + `MessagingDelegate`; upload after JWT. Retailer had no AppDelegate before — `retailerapp/Services/AppDelegate.swift`. Backend persists via claims — `platform/handlers.go:179-216`, mount `platformroutes/routes.go:37`.
- `FIREBASE_PROJECT_ID` ConfigMap + rendered artifact = `pegasus-503013` — `infra/k8s/backend-go/configmap.yaml:36`. Matches GoogleService-Info.plist. Compose SSMR still `demo-pegasus`. ExternalSecret still has no project-id key.
- Proof: `go test ./platform/ -count=1` passed. Retailer `DeviceTokenPayloadTests` **TEST SUCCEEDED** (xcodebuild, Firebase iOS SDK 11.15.0 resolved). Driver/warehouse/payload compile not run this session. Factory iOS still has no PushNotificationManager. Flag still false. Not READY FOR LAYER B. NEXT: factory iOS FCM (not Layer B).

## Verified 2026-08-18 (factory iOS FCM)

- Factory iOS POSTs **FCM registration tokens** (not APNs hex) to `POST /v1/user/device-token` — `FactoryApp/Services/AppDelegate.swift:11-20`, `PushNotificationManager.swift:44-57`, `APIClient.swift:47-51`. Adaptor + login upload — `PegasusFactoryApp.swift:6`, `LoginView.swift:134-136`, `RootView.swift:19-21`. SPM `FirebaseMessaging` — `project.yml:32-33`.
- Persist keys from JWT: `ClaimsActorID` uses `HomeNodeID` for factory — `platform/service.go:196-207`; login sets `HomeNodeID` = factoryID — `factory/auth_login.go:148-154`. Fanout `pushFCM(ctx, factoryID, "FACTORY")` — `kafka/notification_dispatcher.go:1011`. Handler persist — `platform/handlers.go:179-216`.
- Proof: `go test ./platform/ -count=1` (includes `TestHandleDeviceToken_FactoryActorIsHomeNodeID`). FactoryAppTests **TEST SUCCEEDED** on iPad (A16) OS 26.5 (`deviceTokenEncodesTokenAndPlatformOnly`). Firebase iOS SDK 11.15.0. Flag still false. Not READY FOR LAYER B. NEXT: factory/warehouse Android Messaging (auth-only gradle; not Layer B).

## Verified 2026-08-18 (factory/warehouse Android FCM)

- Factory Android POSTs FCM tokens after JWT — `data/push/DeviceTokenRegistrar.kt:19-36`, `FactoryFirebaseMessagingService.kt`, `FactoryApi.kt:233-236`, login `LoginScreen.kt:88-93`, cold start `PegasusXFactoryApp.kt:11-19`. Gradle `firebase-messaging-ktx`. Manifest `MESSAGING_EVENT`. Skip upload when JWT blank.
- Warehouse Android: unused `registerDeviceToken` Map is now `DeviceTokenRequest` and called — `WarehouseApi.kt:555-558`, `data/push/DeviceTokenRegistrar.kt`, `WarehouseFirebaseMessagingService.kt`, `LoginScreen.kt:79-84`, `PegasusWarehouseApp.kt:12-19`. Fanout `pushFCM(warehouseID, "WAREHOUSE")` — `kafka/notification_dispatcher.go:1000`. JWT `HomeNodeID` = warehouseID — `warehouse/auth_login.go:126-132`.
- Proof: `go test ./platform/` (factory + warehouse HomeNodeID keying). Factory `FactoryModelsTest` + warehouse `DeviceTokenRequestTest` **BUILD SUCCESSFUL** (`testEnterpriseDebugUnitTest`). Flag still false. Not READY FOR LAYER B. NEXT: supplier Android Messaging (no firebase-messaging; not Layer B).

## Verified 2026-08-18 (supplier Android FCM)

- Supplier Android POSTs FCM tokens after JWT — `data/push/DeviceTokenRegistrar.kt:19-36`, `SupplierFirebaseMessagingService.kt`, `SupplierApi.kt:19-20`, login `LoginScreen.kt:122-127`, register `OnboardingViewModel.kt`, cold start `PegasusXSupplierApp.kt:11-17`. Gradle `firebase-messaging-ktx`. Manifest `MESSAGING_EVENT`. Skip upload when JWT blank.
- Persist keys from JWT: `ClaimsActorID` for ADMIN uses `SupplierID` — `platform/service.go:196-201`; login JWT `RoleAdmin` + `SupplierID` — `supplier/service.go:972-976`. Fanout `pushFCM(ctx, supplierID, "ADMIN")` — `kafka/notification_dispatcher.go:906`.
- Proof: `go test ./platform/` (`TestHandleDeviceToken_AdminActorIsSupplierID`). Supplier `DeviceTokenRequestTest` **BUILD SUCCESSFUL** (`testEnterpriseDebugUnitTest`). Flag still false. Not READY FOR LAYER B. NEXT: leftover JWT Firebase-ID comments / staging `FCM_ALLOW_NOOP` (not Layer B). Native FCM leftover row closed 2026-08-18 (supplier iOS).

## Verified 2026-08-18 (supplier iOS FCM)

- Supplier iOS POSTs FCM registration tokens after JWT — `AppDelegate.swift`, `PushNotificationManager.swift:52-57`, `APIClient.swift:51-56`, login `LoginView.swift:93-94`, register `OnboardingViewModel.swift:107-108`, RootView `.task` `RootView.swift:51-53`. SPM FirebaseCore + FirebaseMessaging 11.15.0. Skip upload when JWT blank / under XCTest. Body `{token, platform}` only.
- Persist keys from JWT: `ClaimsActorID` ADMIN → `SupplierID` — `platform/service.go:196-201`; mount `POST /v1/user/device-token` — `platformroutes/routes.go:37`; fanout `pushFCM(ctx, supplierID, "ADMIN")` — `kafka/notification_dispatcher.go:906`.
- Proof: SupplierAppTests `testEncodesTokenAndPlatformOnly` **TEST SUCCEEDED** on iPad (A16) OS 26.5. Flag still false. Not READY FOR LAYER B. NEXT: leftover JWT “preferred Firebase ID” comments/stores closed 2026-08-18. Residual: backend `FirebaseAuth` Bearer attach; staging `FCM_ALLOW_NOOP` (do not flip without credentials).

## Verified 2026-08-18 (HTTP Bearer is pegasus JWT)

- Factory portal `apiFetch` uses JWT only — `factory-portal/lib/auth.ts:150-151`; `httpAuthorizationToken` never falls back to Firebase ID — `:119-127`. OTP still posts `id_token` on login — `app/auth/login/page.tsx:55-58`.
- Retailer Android `getPreferredToken` = session JWT; unused Firebase ID store removed — `TokenManager.kt:33-47`. Driver interceptor `TokenHolder.httpAuthorizationToken(TokenHolder.token, null)` — `NetworkModule.kt:56`; leftover `firebaseIdToken` store + “preferred over legacy JWT” comment removed.
- Proof: factory-portal vitest `auth.test.ts` 6 passed. Retailer + driver `TokenAuthTest` **BUILD SUCCESSFUL** (`testEnterpriseDebugUnitTest`). Flag still false. Not READY FOR LAYER B. NEXT: `FirebaseAuth` Bearer attach closed 2026-08-18. Staging `FCM_ALLOW_NOOP` (do not flip without credentials).

## Verified 2026-08-18 (FirebaseAuth Bearer is not a session)

- `FirebaseAuth` pass-through: does not call `VerifyIDToken` and does not `WithClaims` from `Authorization` — `auth/firebase.go:192-202`. Session remains `SessionAuth` JWT — `auth/jwt.go:236-256`. OTP `id_token` body still verified at login — `retailer/auth_login.go:37-40`.
- Proof: `go test ./auth/ -count=1` (`TestFirebaseAuth_DoesNotAttachFromAuthorization`, `TestFirebaseAuth_DoesNotOverwriteJWTClaims`). `./retailer/` passed. Flag still false. Not READY FOR LAYER B. NEXT: unused `FirebaseAuth` wraps **closed** 2026-08-18. Do not flip staging `FCM_ALLOW_NOOP` without credentials. Not terraform apply. Not Stripe/Soliq keys. Not flip `checkout_reads_this`.

## Verified 2026-08-18 (unused FirebaseAuth route wraps dropped)

- `ProtectMutations` no longer mounts `FirebaseAuth` — `auth/route_guard.go:11-22`. Factory/warehouse always `r.Group(register)` — `factoryroutes/routes.go:102`, `warehouseroutes/routes.go:191`. Payload RequireRole-only — `payloaderoutes/routes.go:73-76`. Telemetry `RequireRole(DRIVER)` — `telemetryroutes/routes.go:122`. WS `GET /v1/ws` unwrapped — `ws/handler.go:81`. Returns/delivery same. `main.go` no longer constructs a route-level Firebase verifier; login verifier stays `bootstrap/firebase.go:12-13`.
- `FirebaseAuth()` kept as pass-through + tests — `auth/firebase.go:192-203`. HTTP SoT remains `SessionAuth` — `main.go:129`, `auth/jwt.go:236-256`. Unused portal `getFirebaseIdToken` removed (factory/warehouse/supplier/retailer-desktop `lib/firebase.ts`). OTP still `verifyPhoneOtp`.
- Proof: `go test ./auth/ ./telemetryroutes/ ./countrycfg/ ./bootstrap/ -count=1` passed (`TestNewLoginFirebaseVerifier|TestNewApp_WiresLoginFirebaseVerifier`). `go test ./ws/ -run TestRegisterRoutes` passed. `go build` backend-go. Flag still false. Not READY FOR LAYER B. NEXT: unused portal `exchangeCustomToken` **closed** 2026-08-18. Payload-terminal Expo tokens still not FCM. Do not flip staging/EU `FCM_ALLOW_NOOP=true` without credentials (`infra/k8s/overlays/staging/kustomization.yaml:89`). Not terraform apply. Not Stripe/Soliq keys. Not flip `checkout_reads_this`.

## Verified 2026-08-18 (portal exchangeCustomToken dropped)

- Factory/warehouse/supplier/retailer-desktop `lib/firebase.ts` no longer export `exchangeCustomToken`. OTP is `verifyPhoneOtp`; HTTP session is JWT — factory login `persistSession(data.token, data.refresh_token)` — `factory-portal/app/auth/login/page.tsx:71`. Native SDK exchange **stays** — `payload-app-android/.../AuthRepository.kt:72`, retailer/driver iOS `FirebaseAuthHelper`.
- Payload-terminal no longer persists login `firebase_token` or POSTs it as a device token — `payload-terminal/authSession.ts:19-28`, `pushRegistration.ts:29-47`. Mint for native payload still `payload/auth_login.go:215`. FCM Send still uses stored tokens as FCM registration tokens — `notifications/fcm.go:113-124`.
- Proof: factory-portal vitest `auth.test.ts` 6 passed. payload-terminal vitest 15 passed (manifest tests; push path re-read, no unit covering `registerPayloadPushTokens`). Flag still false. Not READY FOR LAYER B. NEXT: Expo tokens **rejected** 2026-08-18. Do not flip staging/EU `FCM_ALLOW_NOOP`. Not terraform apply. Not Stripe/Soliq keys. Not flip `checkout_reads_this`.

## Verified 2026-08-18 (Expo tokens are not FCM)

- `POST /v1/user/device-token` 422 `not_fcm_registration_token` for Expo platform/token — `platform/handlers.go:201-203`, `IsFCMRegistrationToken` `platform/service.go:248-261`. `ListTokens` skips `ExponentPushToken[` leftovers — `platform/repository.go:207-209`. FCM Send still uses stored strings as FCM registration tokens — `notifications/fcm.go:113-124`.
- Payload-terminal no longer POSTs Expo push tokens. `getDevicePushTokenAsync` → Android FCM only — `fcmDeviceToken.ts:12-25`, `pushRegistration.ts:41-45`. iOS Expo APNs skipped. `app.json` has no `googleServicesFile` — Android Expo FCM is best-effort, not claimed live. Native payload Android/iOS still POST FCM.
- Proof: `go test ./platform/ -count=1` (RejectsExpoPlatform, RejectsExponentPushToken, ListTokens_SkipsExpoShapedToken). payload-terminal vitest 19 passed (`fcmDeviceToken.test.ts`). Flag still false. Not READY FOR LAYER B. NEXT: do not flip staging/EU `FCM_ALLOW_NOOP=true` (`infra/k8s/overlays/staging/kustomization.yaml:89`). Residual: unmounted `HandleDeviceTokenNoop` `payload/inbox.go:75-82`. Not terraform apply. Not Stripe/Soliq keys. Not flip `checkout_reads_this`.

## Verified 2026-08-18 (Payme + Click implemented, unwired)

- UZ launch rails unchanged: CASH + GLOBAL_PAY live catalog — `payment/catalog.go:36-39`; UZ `CheckoutReadsThis: false` — `auth/market_pack.go:139`; bank-file payout `IsLivePayoutRailImplemented` always false — `auth/payout_pack.go:47-48`.
- Payme Merchant JSON-RPC (CheckPerform/Create/Perform/Cancel/Check/GetStatement) + Subscribe receipts.* executor exist — `payment/payme_webhook.go:11-16`, `payment/payme_executor.go:16-18`. Click SHOP Prepare/Complete (complete sign includes `merchant_prepare_id`; so'm→minor int64) + Merchant invoice/status/reversal/card_token exist — `payment/click_webhook.go`, `payment/click_executor.go:17-18`.
- **Not wired:** live router still `catalogHonestyExecutorFor("PAYME"|"CLICK")` — `payment/execution.go:157-158`. `POST /v1/webhooks/payme` and `/click` commented out — `webhookroutes/routes.go:26-31`. Checkout PAYME/CLICK remains 501 `no_live_keys`.
- Proof: `go test ./payment/ ./webhookroutes/ -count=1` passed (`TestPaymeMerchant_CheckCreatePerformCancelCheckStatement`, `TestClickWebhook_PrepareDoesNotMarkPaid`, `TestRegisterRoutes_PaymeAndClickUnmounted`, honesty unkeyed). Not READY FOR LAYER B. Do not connect Payme/Click keys. Not terraform. Not flip `checkout_reads_this`.

## Verified 2026-08-18 (Payme/Click remaining official methods, still unwired)

- Payme Subscribe `cards.create/get_verify_code/verify/check/remove`, `receipts.pay`, `receipts.set_fiscal_data` (refuses empty fiscal) — `payment/payme_executor.go`. Merchant inbound `SetFiscalData` persists Payme-supplied OFD payload, does not invent QR/sign — `payment/payme_webhook.go:71-72`.
- Click `partial_reversal`, OFD get/submit_items/submit_qrcode (empty QR/items fail closed), Click Pass payment/confirm/confirmation — `payment/click_executor.go`. Still not registered.
- Live path unchanged: honesty `PAYME`/`CLICK` — `payment/execution.go:159-160`; webhook mounts commented — `webhookroutes/routes.go:30-31`; catalog `unkeyed` — `payment/catalog.go:38-39`.
- Proof: `go test ./payment/ ./webhookroutes/ -count=1`. Not READY FOR LAYER B. Do not connect Payme/Click keys. Not terraform. Not flip `checkout_reads_this`.

## Verified 2026-08-18 (device-token honesty leftover)

- Unmounted `HandleDeviceTokenNoop` deleted from `payload/inbox.go` (file ends at `HandleMarkNotificationsRead`). Live POST remains `POST /v1/user/device-token` — `platformroutes/routes.go:37`.
- Nil `DeviceTokens` now 503 `device_token_unavailable`, not 200 `{status:ok}` — `platform/handlers.go:205-207`. Expo still 422 — `:201-203`. Bootstrap still injects memory repo when Spanner is nil, Spanner repo when client is up — `bootstrap/bootstrap.go:1376-1387` (process rarely hits 503 unless Handler is constructed without a store).
- Payme/Click still unwired — `payment/execution.go:159-160`, `webhookroutes/routes.go:30-31`.
- Proof: `go test ./platform/ ./payload/ ./platformroutes/ -count=1` (`TestHandleDeviceToken_NilStoreUnavailable`). FIREBASE_AUDIT residual no longer names the noop. Not READY FOR LAYER B. Do not flip `FCM_ALLOW_NOOP`. Not terraform. Not Stripe/Soliq keys. Not flip `checkout_reads_this`. Not Payme/Click mounts.

## Verified 2026-08-18 (inbox list no longer fake-empty)

- Graph query `user notifications inbox` / `notifications` → **no hits** (`generatedAt` null). Live mount is `GET/POST /v1/user/notifications` — `main.go:418-419` (`InboxHandlers.HandleList` / `HandleMarkRead`).
- Nil service → 503 `inbox_unavailable`; list error → 500 `inbox_list_failed` (not 200 `notifications:[]`) — `notifications/handlers.go:34-43`. Mark-read nil store 503 — `:79-81`. `ApplyMarkRead` nil svc errors — `notifications/inbox_wire.go:77-79`.
- Bootstrap always constructs `InboxHandlers` (Service nil when Spanner is nil) — `bootstrap/bootstrap.go:655`. Overwritten role-row mounts removed from payload/retailer/driver/supplier routes. Handlers kept fail-closed (unmounted).
- Payme/Click still unwired — `payment/execution.go:159-160`, `webhookroutes/routes.go:30-31`.
- Proof: `go test ./notifications/ ./payload/ ./retailer/ ./driver/ ./payloaderoutes/ ./retailerroutes/ ./driverroutes/ ./supplierroutes/ ./bootstrap/ -count=1`. Not READY FOR LAYER B. NEXT leftover: `UnreadCount` error ignored on successful list (`handlers.go:47`). Do not flip `FCM_ALLOW_NOOP`. Not terraform. Not Stripe/Soliq keys. Not flip `checkout_reads_this`. Not Payme/Click mounts.

## Verified 2026-08-18 (inbox UnreadCount fail-closed)

- Graph query `UnreadCount inbox notifications` → **no hits**. Live GET still `main.go:418` `HandleList`. Unread query error now 500 `inbox_unread_failed` (not 200 with `unread_count: 0`) — `notifications/handlers.go:47-53`. Same on unmounted payload/retailer/driver copies. Pulse unread error fails the merge — `pulse/service.go:106-109` (HTTP `pulse_failed`).
- Payme/Click still unwired — `payment/execution.go:159-160`, `webhookroutes/routes.go:30-31`.
- Proof: `go test ./notifications/ ./payload/ ./retailer/ ./driver/ ./pulse/ -count=1` (`TestHandleList_UnreadErrorFailed`, `TestListForRecipient_UnreadError`). Not READY FOR LAYER B. NEXT leftover: pulse still warn-and-continue on transitions / supplier activity (`pulse/service.go:113-123`). Do not flip `FCM_ALLOW_NOOP`. Not terraform. Not Stripe/Soliq keys. Not flip `checkout_reads_this`. Not Payme/Click mounts.

## Verified 2026-08-18 (pulse transitions fail-closed)

- Graph query `pulse transitions` → **no hits**. Live mounts `GET /v1/*/pulse` — `pulseroutes/routes.go:26-46`. Transition / supplier-activity read errors now fail the merge (`pulse transitions` / `pulse supplier activity`) — `pulse/service.go:114-130`; HTTP 500 `pulse_failed` — `pulse/handlers.go:49-51`. Scan errors no longer skip rows — `:199-200`.
- Payme/Click still unwired — `payment/execution.go:159-160`, `webhookroutes/routes.go:30-31`.
- Proof: `go test ./pulse/ -count=1` (`TestListForRecipient_TransitionsError`, `TestListForRecipient_SupplierActivityError`, `TestHandleSupplierPulse_ActivityError`). Not READY FOR LAYER B. NEXT leftover: portal `NetworkPulsePanel` still maps fetch failure to empty `events[]` (PulseTimeline `error` unused) — `supplier-portal/components/NetworkPulsePanel.tsx:17-22`. Do not flip `FCM_ALLOW_NOOP`. Not terraform. Not Stripe/Soliq keys. Not flip `checkout_reads_this`. Not Payme/Click mounts.

## Verified 2026-08-18 (portal pulse panels fail-closed)

- Graph query `NetworkPulsePanel pulse timeline` → **no hits**. Live GET still `pulseroutes/routes.go:26-46`. Supplier/factory/warehouse/retailer-desktop `NetworkPulsePanel` + factory/warehouse `HandoffTimelinePanel` now pass `error={error}` / `pulse_failed` and no longer `setEvents([])` on fetch failure — `supplier-portal/components/NetworkPulsePanel.tsx:19-24`. PulseTimeline `error` prop already existed — `packages/pulse-ui/src/PulseTimeline.tsx:8,31-32`.
- Payme/Click still unwired — `payment/execution.go:159-160`, `webhookroutes/routes.go:30-31`.
- Proof: vitest `network-pulse-honesty.test.ts` passed in supplier-portal, factory-portal, warehouse-portal, retailer-app-desktop. Not READY FOR LAYER B. NEXT leftover: native pulse still swallows errors as empty (`try?` / `body()?.let`) — `supplier-app-ios/.../DashboardView.swift:210`, `supplier-app-android/.../DashboardScreen.kt:123`. Do not flip `FCM_ALLOW_NOOP`. Not terraform. Not Stripe/Soliq keys. Not flip `checkout_reads_this`. Not Payme/Click mounts.

## Verified 2026-08-18 (native pulse fail-closed)

- Graph query `native pulse dashboard NetworkPulseStrip` is stale (tracking/dashboard nodes, misses supplier pulse paths). Live GET still `pulseroutes/routes.go:27-46`. Native pulse GET failure now keeps previous events and surfaces `pulse_failed` instead of `[]` — Android `PulseHonesty.applyHttp` `packages/mobile-android-design/src/main/java/com/pegasus/design/PulseHonesty.kt:4-12`; supplier load `supplier-app-android/.../DashboardScreen.kt:123-136` + strip `error = pulseError` `:247-252`; iOS `PulseHonesty.apply` `packages/mobile-ios-design/RealtimeRefresh.swift:52-53`; supplier load `supplier-app-ios/.../DashboardView.swift:210-221`. Same pattern on factory/warehouse handoff timelines and driver/payload/retailer pulse strips.
- Payme/Click still unwired — `payment/execution.go:159-160`, `webhookroutes/routes.go:30-31`.
- Proof: Android `testEnterpriseDebugUnitTest` (`PulseHonestyTest`, supplier/factory/warehouse/driver/payload/retailer source honesty) + iOS XCTest `testPulseHonestyKeepsPreviousOnFailure` (supplier + factory). Warehouse iOS `WarehouseAppTests/DashboardTests.swift` is not in the xcodeproj. Not READY FOR LAYER B. NEXT leftover: retailer command-tower pulse still swallows (`commandPulse = nil` on catch) — `retailer-app-ios/.../DashboardView.swift:305-312`, `retailer-app-android/.../DashboardViewModel.kt:141-146`. Do not flip `FCM_ALLOW_NOOP`. Not terraform. Not Stripe/Soliq keys. Not flip `checkout_reads_this`. Not Payme/Click mounts.

## Verified 2026-08-18 (retailer command-tower pulse fail-closed)

- Graph query `retailer control tower pulse commandPulse DashboardView` is stale (tracking screens, misses control-tower pulse). Live GET `GET /v1/retailer/control-tower/pulse` — `retailerroutes/routes.go:58` (`HandleControlTowerPulse`). Handler still always 200 after identity — `retailer/control_tower_pulse.go:57-58`.
- Client HTTP/throw is no longer treated as `source: "empty"`. Keep last pulse + `control_tower_pulse_failed` — Android `PulseHonesty.applyObject` `packages/mobile-android-design/src/main/java/com/pegasus/design/PulseHonesty.kt:16-18`; dashboard `retailer-app-android/.../DashboardViewModel.kt:141-160` + screen error (not SourceChip empty) `DashboardScreen.kt:174-197`; iOS `PulseHonesty.applyObject` `retailer-app-ios/.../PulseStripView.swift:24-28`; `DashboardView.swift:315-325` + `gs-u-retailer-command-error`. Desktop CommandBoard returns error, not zero tiles — `CommandBoard.tsx:34-46`; dashboard passes `control_tower_pulse_failed` — `dashboard/page.tsx:210-214`. Dedicated control-tower page keeps last pulse, does not `setPulse(null)` — `control-tower/page.tsx:51-63`.
- Payme/Click still unwired — `payment/execution.go:159-160`, `webhookroutes/routes.go:30-31`.
- Proof: vitest `command-dashboard.test.ts` (4 passed); Android `PulseHonestyTest` + `DashboardTests.commandPulseFailureDoesNotTreatAsEmpty`; iOS `DashboardCommandTests/commandPulseHonestyKeepsPreviousOnFailure` (iPhone 17). Not READY FOR LAYER B. NEXT leftover: backend `buildControlTowerPulse` still skips tracking/shift/assist read errors as zeros (`retailer/control_tower_pulse.go:68-97`) so HTTP 200 can look empty when Spanner failed. Do not flip `FCM_ALLOW_NOOP`. Not terraform. Not Stripe/Soliq keys. Not flip `checkout_reads_this`. Not Payme/Click mounts.

## Verified 2026-08-18 (control-tower pulse build fail-closed)

- Graph query `buildControlTowerPulse` is stale (tracking screens). Live GET still `retailerroutes/routes.go:58`. Tracking / POS / shifts / assist / stock / packs / dashboard-order read errors now 500 `control_tower_pulse_failed` (not 200 empty/zeros) — `retailer/control_tower_pulse.go:59-62`, `:72-138`. `loadRetailerDashboardOrders` no longer maps Spanner/hook errors to `source: empty` — `dashboard_rollups.go:107-122`. POS Spanner query error is not a memory fallback — `control_tower_pulse.go:176-197`. Honest empty (no store, queries succeed) still 200 `empty: true`.
- Payme/Click still unwired — `payment/execution.go:159-160`, `webhookroutes/routes.go:30-31`.
- Proof: `go test ./retailer/ ./webhookroutes/ ./payment/ -count=1` (`TestControlTowerPulseTrackingErrorFailed`, `TestControlTowerPulseDashboardOrdersErrorFailed`, empty/live/facet still 200). Not READY FOR LAYER B. NEXT leftover: desktop `KpiGrid` still uses `pulse?.open_orders ?? 0` on the command dashboard (`retailer-app-desktop/.../dashboard/page.tsx:217`) so a first-load 500 can show zero inbound next to the honest command error. `reports_pro` still swallows inventory/shift list errors as zeros — `retailer/reports_pro.go:36-37`. Do not flip `FCM_ALLOW_NOOP`. Not terraform. Not Stripe/Soliq keys. Not flip `checkout_reads_this`. Not Payme/Click mounts.

## Verified 2026-08-18 (command KPI zeros fail-closed)

- Graph query `retailer KpiGrid command pulse` is stale (tracking screens). Live GET still `retailerroutes/routes.go:58`. Pulse-derived KPI tiles are not rendered on command-tower GET failure (or first load before pulse): desktop `KpiGrid` gated — `retailer-app-desktop/app/(dashboard)/dashboard/page.tsx:216-226`; iOS `KpiGrid` only when `commandPulseError == nil` — `DashboardView.swift:44-52`; Android `DashboardOverviewCard` only when `commandPulseError` blank — `DashboardScreen.kt:228-236`. Honest empty 200 still shows zeros from a real pulse body.
- Payme/Click still unwired — `payment/execution.go:159-160`, `webhookroutes/routes.go:30-31`.
- Proof: vitest `command-dashboard.test.ts` (5 passed); Android `DashboardTests.commandPulseFailureDoesNotTreatAsEmpty`; iOS `retailerappTests` TEST SUCCEEDED (iPhone 17). Not READY FOR LAYER B. NEXT leftover: `GET /v1/retailer/reports/summary` still swallows inventory/shift list errors as zeros — `retailer/reports_pro.go:36-37`, `:386-401`. Residual: `blockedPredictionCount={0}` on KpiGrid when shown. Do not flip `FCM_ALLOW_NOOP`. Not terraform. Not Stripe/Soliq keys. Not flip `checkout_reads_this`. Not Payme/Click mounts.

## Verified 2026-08-18 (retailer reports inventory/shift fail-closed)

- Graph query `retailer reports summary LoadEnabledPacks REPORTS_PRO` is stale (tracking screens). Live mounts `GET /v1/retailer/reports/summary|inventory|shifts` — `retailerroutes/routes.go:136-140`. Inventory / shift / movement read errors now 500 `reports_failed` (not 200 zeroed tiles / empty items) — `retailer/reports_pro.go:37-45`, `:154-162`, `:216-219`. Honest empty memory store still 200 (`phase6_test.go`).
- Clients do not render zero digest on GET failure: desktop `summaryError` / `salesError` — `retailer-app-desktop/app/(dashboard)/reports/page.tsx:43-64`, `:118-137`; iOS `loadError = "reports_failed"` + hide Last 7 days — `ReportsProView.swift:25`, `:74`; Android same — `ReportsScreen.kt:88`, `:146`.
- Payme/Click still unwired — `payment/execution.go:159-160`, `webhookroutes/routes.go:30-31`.
- Proof: `go test ./retailer/ ./webhookroutes/ ./payment/ -count=1` (`TestHandleReportsSummaryInventoryErrorFailed`, `TestHandleReportsSummaryShiftsErrorFailed`, `TestHandleReportsInventoryErrorFailed`, `TestHandleReportsShiftsErrorFailed`); vitest `reports-honesty.test.ts`; Android `DashboardTests.reportsFailureDoesNotTreatAsZeroDigest` (`--rerun-tasks`). Not READY FOR LAYER B.

## Verified 2026-08-18 (reportsAuth pack load/enable fail-closed)

- `reportsAuth` no longer discards `LoadEnabledPacks` / `SetPackEnabled` errors and auto-enables on a failed read — `retailer/reports_pro.go:287-297`. Fail is 500 `reports_failed`. Auto-enable on first view remains only after a successful pack read. Test seams `enabledPacksQuery` / `setPackEnabledFn` — `retailer/service.go:403-404`, `repository_capabilities.go:29-30`, `:75-76`.
- Payme/Click still unwired — `payment/execution.go:159-160`, `webhookroutes/routes.go:30-31`.
- Proof: `go test ./retailer/ ./webhookroutes/ ./payment/ -count=1` (`TestHandleReportsSummaryPacksLoadErrorFailed`, `TestHandleReportsSummaryPackEnableErrorFailed`; phase6 summary still 200). Not READY FOR LAYER B. NEXT leftover: HQ clone still swallows pack load/enable — `retailer/hq_handlers.go:60-62`. Then `aggregateSales` is memory-only (`s.posSales`) while Spanner is the live POS ledger — `reports_pro.go:361-368`. Residual: `blockedPredictionCount={0}` on KpiGrid when shown. Do not flip `FCM_ALLOW_NOOP`. Not terraform. Not Stripe/Soliq keys. Not flip `checkout_reads_this`. Not Payme/Click mounts.

## Verified 2026-08-18 (HQ pack load/enable fail-closed)

- Graph query `retailer HQ reports LoadEnabledPacks hq_handlers` is stale (tracking screens). Live mounts `GET /v1/retailer/hq/summary|sales-by-location|sales-by-sku|shrinkage|export` — `retailerroutes/routes.go:143-147`. `hqAuth` pack load/enable errors are 500 `hq_failed` (not 200 zeroed net / auto-enable on a failed read) — `retailer/hq_handlers.go:60-69`. Auto-enable on first HQ use remains only after a successful pack read. Honest empty (flag on, queries succeed) still 200 `honest_empty`. Flag off still 404 `HQ_ANALYTICS_DISABLED`.
- Clients do not render zero HQ tiles / "No HQ rows" on GET failure: desktop `hqError` + tiles only when `summary` exists — `retailer-app-desktop/app/(dashboard)/hq/page.tsx:92-94`, `:186-190`; iOS `loadError = "hq_failed"` + hide sections — `HqView.swift:20`, `:61`; Android same — `HqScreen.kt:97`, `:126`.
- Payme/Click still unwired — `payment/execution.go:159-160`, `webhookroutes/routes.go:30-31`.
- Proof: `go test ./retailer/ ./webhookroutes/ ./payment/ -count=1` (`TestHqAPI_PacksLoadErrorFailed`, `TestHqAPI_PackEnableErrorFailed`; disabled 404 / cashier 403 / balanced 200 still pass); vitest `hq-honesty.test.ts`; Android `DashboardTests.hqFailureDoesNotTreatAsEmptyRows`. Not READY FOR LAYER B. NEXT leftover: `aggregateSales` is memory-only (`s.posSales`) while Spanner is the live POS ledger — `retailer/reports_pro.go:361-368`. Residual: `blockedPredictionCount={0}` on KpiGrid when shown. Do not flip `FCM_ALLOW_NOOP`. Not terraform. Not Stripe/Soliq keys. Not flip `checkout_reads_this`. Not Payme/Click mounts.

## Verified 2026-08-18 (reports POS sales from ledger, fail-closed)

- Graph query `retailer aggregateSales posSales POS reports summary` is stale (tracking screens). Live mounts `GET /v1/retailer/reports/summary|sales` — `retailerroutes/routes.go:136-137`; pulse `GET /v1/retailer/control-tower/pulse` — `:58`. POS complete writes `RetailerPosSales` — `retailer/pos.go:1028` (`Idx_RetailerPosSales_ByRetailer` `schema/spanner.ddl:2336`).
- Reports/pulse no longer scan in-memory `s.posSales` when Spanner is the live ledger. `listRetailerPosSales` reads Spanner (memory only if client unset); query/decode/truncation errors fail closed — `pos.go:1219-1280`. Summary/sales 500 `reports_failed` — `reports_pro.go:36-39`, `:84-87`. Pulse 500 `control_tower_pulse_failed` — `control_tower_pulse.go:128-131`. Honest empty (no rows) still 200 zeros. Window >2000 sales is 500, not a silent undercount.
- Clients already hide digest on those 500s (prior leftovers). No client edit this slice.
- Payme/Click still unwired — `payment/execution.go:159-160`, `webhookroutes/routes.go:30-31`.
- Proof: `go test ./retailer/ ./webhookroutes/ ./payment/ -count=1` (`TestHandleReportsSummarySalesErrorFailed`, `TestHandleReportsSalesErrorFailed`, `TestHandleReportsSummarySalesFromLedger`, `TestControlTowerPulseSalesErrorFailed`; phase6 sales 200 still pass). Not READY FOR LAYER B. NEXT leftover: `listStockBalances` errors discarded as empty union — `retailer/local_skus.go:341`, `sections.go:373`. Residual: `blockedPredictionCount={0}` on KpiGrid when shown. Do not flip `FCM_ALLOW_NOOP`. Not terraform. Not Stripe/Soliq keys. Not flip `checkout_reads_this`. Not Payme/Click mounts.

## Verified 2026-08-18 (POS catalog + unassigned SKUs stock union fail-closed)

- Graph query `retailer listStockBalances local_skus sections unassigned` is stale (tracking screens). Live mounts `GET /v1/retailer/pos/catalog` — `retailerroutes/routes.go:100`; `GET /v1/retailer/sections/unassigned-skus` — `:123`. Stock read errors no longer look like empty union: catalog 500 `pos_catalog_failed` — `local_skus.go:349-352`; unassigned 500 `unassigned_skus_failed` — `sections.go:380-388`. `allAssignedSkusAtLocation` section/SKU list errors also fail the unassigned GET — `:794-804`. `listStockBalances` honors `stockBalancesQuery` — `store_stock.go:800-802`. Honest empty (queries succeed, no rows) still 200 `items[]` / `skus[]`.
- Desktop sections no longer maps unassigned 500 to "None (or no stock yet)" — `retailer-app-desktop/app/(dashboard)/sections/page.tsx:44-46`, `:250-252`. Android/iOS sections screens do not call unassigned-skus or pos/catalog (no client on those GETs).
- Payme/Click still unwired — `payment/execution.go:159-160`, `webhookroutes/routes.go:30-31`.
- Proof: `go test ./retailer/ ./webhookroutes/ ./payment/ -count=1` (`TestHandlePOSCatalogSearchStockErrorFailed`, `TestHandleUnassignedSkusStockErrorFailed`; phase6 unassigned 200 still pass); vitest `sections-honesty.test.ts`. Not READY FOR LAYER B. NEXT leftover: `listLocalSKUs` Spanner scan error still falls back to memory — `local_skus.go:440-442`. Residual: `blockedPredictionCount={0}` on KpiGrid — `dashboard/page.tsx:223`. Do not flip `FCM_ALLOW_NOOP`. Not terraform. Not Stripe/Soliq keys. Not flip `checkout_reads_this`. Not Payme/Click mounts.

## Verified 2026-08-18 (listLocalSKUs Spanner fail-closed, no memory fallback)

- Graph query `retailer listLocalSKUs local-skus pos catalog` is stale (tracking screens). Live mounts `GET /v1/retailer/local-skus` — `retailerroutes/routes.go:109`; `GET /v1/retailer/pos/catalog` — `:100`; `POST /v1/retailer/pos/scan` — `:96`; `POST /v1/retailer/pos/sales` — `:97`.
- Spanner scan/column errors return `err` (no `listLocalSKUsMem` success). Memory list only when `spannerClient == nil`. Hook `localSKUsQuery` — `retailer/service.go:406`, `local_skus.go:430-456`. GET list 500 `local_skus_failed` (no `items[]`) — `:73-76`. Catalog 500 `pos_catalog_failed` — `:327-330`. Scan/sale barcode lookup 500 `local_skus_failed` (not 400 `invalid_line` / 404 `sku_not_found`) — `:282-283`, `:290-292`, `pos.go:517-520`. `getLocalSKU` error in `validatePosSaleSKU` is `local_skus_failed`, not free-form sell — `local_skus.go:393-397`. Honest empty (no Spanner, empty mem) still 200.
- Clients do not render "No local SKUs yet" / `no_local_skus` on GET failure: desktop `local_skus_failed` — `retailer-app-desktop/app/(dashboard)/stock/local-skus/page.tsx:35-43`, `:175`; iOS `loadError` + hide Catalog — `LocalSkusView.swift:16-17`, `:28`, `:69`; Android same — `LocalSkusScreen.kt:99`, `:159`.
- Payme/Click still unwired — `payment/execution.go:159-160`, `webhookroutes/routes.go:30-31`.
- Proof: `go test ./retailer/ ./webhookroutes/ ./payment/ -count=1` (`TestHandleLocalSKUsListErrorFailed`, `TestHandlePOSCatalogSearchLocalSKUsErrorFailed`, `TestHandlePOSScanLocalSKUsErrorFailed`, `TestValidatePosSaleSKUBarcodeLookupFailed`, `TestHandlePosSaleLocalSKUsErrorFailed`; `TestLocalSKUCRUDAndPOSCatalog` still 200); vitest `local-skus-honesty.test.ts`; Android `DashboardTests.localSkusFailureDoesNotTreatAsEmptyCatalog` (`--rerun-tasks`). Not READY FOR LAYER B. NEXT leftover: `GET /v1/retailer/sections/{sectionID}` still returns 200 empty `skus[]` / `staff_ids` when list errors — `retailer/sections.go:186-187` (`retailerroutes/routes.go:126`). Residual: `blockedPredictionCount={0}` on KpiGrid — `dashboard/page.tsx:223`. PATCH local-sku still discards `getLocalSKU` err after save — `local_skus.go:235`. Do not flip `FCM_ALLOW_NOOP`. Not terraform. Not Stripe/Soliq keys. Not flip `checkout_reads_this`. Not Payme/Click mounts.

## Verified 2026-08-18 (section GET SKU/staff lists fail-closed)

- Graph query `retailer sections HandleSectionByID listSectionSkus` is stale (tracking screens). Live mounts `GET /v1/retailer/sections/{sectionID}` — `retailerroutes/routes.go:126`; `GET .../skus` — `:130`; `GET .../staff` — `:132`.
- Section detail no longer returns 200 empty `skus[]` / `staff_ids` on list errors: 500 `section_detail_failed` (no those keys) — `retailer/sections.go:186-204`. GET SKUs 500 `section_skus_failed` — `:281-284`. GET staff 500 `section_staff_failed` — `:346-349`. Hooks `sectionSkusQuery` / `sectionStaffQuery` — `service.go:407-408`, `sections.go:592-594`, `:727-729`. Staff scan column errors return `err` — `:757-759`. Honest empty (queries succeed, no rows) still 200 `[]`.
- Desktop does not show "Current SKUs: none" on that 500 — `retailer-app-desktop/app/(dashboard)/sections/page.tsx:88-98`, `:222-226`. Android/iOS sections screens do not GET by ID (list + PUT only).
- Payme/Click still unwired — `payment/execution.go:159-160`, `webhookroutes/routes.go:30-31`.
- Proof: `go test ./retailer/ ./webhookroutes/ ./payment/ -count=1` (`TestHandleSectionByIDSkusErrorFailed`, `TestHandleSectionByIDStaffErrorFailed`, `TestHandleSectionByIDGetHonestEmpty`, `TestHandleSectionSkusGetErrorFailed`, `TestHandleSectionStaffGetErrorFailed`; phase6 sections 200 still pass); vitest `sections-honesty.test.ts`. Not READY FOR LAYER B. NEXT leftover: `PUT /v1/retailer/sections/{sectionID}/skus|staff` still discards replace errors and 200s — `sections.go:301-311`, `:364-365` (desktop/Android/iOS all PUT). Residual: `blockedPredictionCount={0}` — `dashboard/page.tsx:223`. Do not flip `FCM_ALLOW_NOOP`. Not terraform. Not Stripe/Soliq keys. Not flip `checkout_reads_this`. Not Payme/Click mounts.

## Verified 2026-08-18 (section SKU/staff PUT fail-closed)

- Graph query `retailer sections PUT replaceSectionSkus` is stale (tracking screens). Live mounts `PUT /v1/retailer/sections/{sectionID}/skus` — `retailerroutes/routes.go:131`; `PUT .../staff` — `:133`.
- Replace/add/remove and post-write list errors are 500 `section_skus_failed` / `section_staff_failed` (no `skus[]` / `user_ids`) — `retailer/sections.go:300-325`, `:382-392`. Success event is not emitted on fail. Hooks `replaceSectionSkusFn` / `addSectionSkusFn` / `removeSectionSkusFn` / `replaceSectionStaffFn` — `service.go:409-412`. Column decode in replace txns returns `err`. Invalid PUT JSON is 400 `invalid_json`. Honest memory PUT still 200 with the written list.
- Clients do not show "SKUs saved" / "Staff assigned" on that 500: desktop `saveError` — `retailer-app-desktop/app/(dashboard)/sections/page.tsx:116-125`, `:163-168`; Android `saveError = "section_skus_failed"` — `SectionsScreen.kt:166-169`; iOS same — `SectionsView.swift:16-17`, `:86`.
- Payme/Click still unwired — `payment/execution.go:159-160`, `webhookroutes/routes.go:30-31`.
- Proof: `go test ./retailer/ ./webhookroutes/ ./payment/ -count=1` (`TestHandleSectionSkusPutReplaceErrorFailed`, `TestHandleSectionSkusPutAddErrorFailed`, `TestHandleSectionSkusPutListAfterWriteFailed`, `TestHandleSectionStaffPutReplaceErrorFailed`, `TestHandleSectionSkusPutHonestOK`; phase6 PUT still 200); vitest `sections-honesty.test.ts`; Android `DashboardTests.sectionSkuPutFailureDoesNotTreatAsSaved` (`--rerun-tasks`). Not READY FOR LAYER B. NEXT leftover: `GET /v1/retailer/me/sections` still drops sections when `listSectionStaff` errors — `sections.go:867`, `HandleMySections` `:464-480` (`retailerroutes/routes.go:134`). Residual: `blockedPredictionCount={0}` — `dashboard/page.tsx:223`. Do not flip `FCM_ALLOW_NOOP`. Not terraform. Not Stripe/Soliq keys. Not flip `checkout_reads_this`. Not Payme/Click mounts.

## Verified 2026-08-18 (me/sections staff-list fail-closed)

- Graph query `retailer HandleMySections listSectionsForUser` is stale (tracking screens). Live mount `GET /v1/retailer/me/sections` — `retailerroutes/routes.go:134`. Staff list errors no longer look like "not assigned": 500 `my_sections_failed` (no `items[]`) — `retailer/sections.go:481-483`, `:870-872`. Honest empty (no staff match) still 200 `items[]`. Assigned membership still 200 with that section.
- Docs vs code: `RETAILER_SECTIONS.md` lists desktop/Android/iOS as clients; those screens call `GET /v1/retailer/sections`, not `me/sections`. No shipped role-row client on this GET (grep this session). Comment "owners see all" is also drift — filter is staff membership only — `:866-868`.
- Payme/Click still unwired — `payment/execution.go:159-160`, `webhookroutes/routes.go:30-31`.
- Proof: `go test ./retailer/ ./webhookroutes/ ./payment/ -count=1` (`TestHandleMySectionsStaffErrorFailed`, `TestHandleMySectionsHonestEmpty`, `TestHandleMySectionsAssignedOK`). Not READY FOR LAYER B. NEXT leftover: `getSection` errors are 404 `section_not_found` — `sections.go:175-178` (same for SKU/staff handlers). Residual: `blockedPredictionCount={0}` — `dashboard/page.tsx:223`. Do not flip `FCM_ALLOW_NOOP`. Not terraform. Not Stripe/Soliq keys. Not flip `checkout_reads_this`. Not Payme/Click mounts.



