# GS-U — Client visualization program (all role apps)

**Final goal (2026-08-16):** this file is the **client UI / data-visualization destination**. It **extends** [`GLOBAL_SCALE_PROGRAM.md`](./GLOBAL_SCALE_PROGRAM.md) + [`GLOBAL_SCALE_LOCAL_ECOSYSTEM.md`](./GLOBAL_SCALE_LOCAL_ECOSYSTEM.md). It does not replace GS-A/T/M/C/I/R/P or GS-L/K. Agent load: `.agents/memory/GOAL.md`. **Not status.**

**Date:** 2026-08-16  
**Tree:** `pegasusX/` (not `pegasus/`)  
**Ask:** Make every role ritual **see** the data the backend already has — current, history, every status — on **web, native desktop, iOS (phone + iPad), Android (phone + tablet)**, with one visualization language, one status dictionary, **native screen/nav flows**, and **enterprise freshness** (event + cache, not 1s polling).

This program is **visualization-first**. Motion / animation is **explicitly deferred**. Do not spend a slice on choreography.

**Companion inventories (routes, not UI truth):** [`FEATURES_BY_APP_ROLE.md`](./FEATURES_BY_APP_ROLE.md) · [`ROLE_ROW_PARITY_MATRIX.md`](./ROLE_ROW_PARITY_MATRIX.md) · [`ROLE_FEATURES_DOCS_VS_CODE.md`](./ROLE_FEATURES_DOCS_VS_CODE.md)

---

## 0. Honest verdict (code this session)

```
VERDICT: PARTIAL (U0+U1+UN+UF+U2+U3+U4+U5+U6+U7+U8+U9 shipped)
DOCS vs CODE: GS-U0 dictionaries + U1 StatusStack kit + UN primary
  nav ≤5 + UF ETag/304 + dirty-slice WS + U2 supplier command
  + U3 Plan & Brain + U4 warehouse command stacks + U5 factory
  command (source + dual plane) + U6 retailer command (pulse
  source + 17-key per-supplier child stack + loyalty honesty)
  + U7 field (payload board + driver stepper) + U8 platform
  admin (dead_letter_count = COUNT(*), not page length)
  + U9 role-row lock (chip → same status key; dated skips).
  checkout_reads_this false.
BLOCKERS (ranked):
  1. checkout_reads_this still false (SSMR PEGASUS ≠ pack MY_SOLIQ)
CLOSED THIS SLICE:
  GS-U9 — StatusStack onSelect on iOS/Android kits. Supplier /
    retailer / warehouse / factory chips jump to the same
    status/state key as web. Lock test asserts markers or
    dated 2026-08-16 skips. Place stays off.
NEXT: leftovers (GS-M flag, cells apply, live PSP). Not a new
  U-motion slice. Do not flip checkout_reads_this. Do not
  terraform apply. Do not invent chart series. Do not add
  1s polling.
```

**Isolation key stays `SupplierId`.** Pack, cell, country are attributes. Dual factory/supplier truck planes stay separate.

---

## 1. Product laws (lock before pixels)

These are UI law, not taste.

| # | Law | Meaning |
|---|-----|---------|
| **U1** | **Live data only** | A chart, sparkline, gauge, or KPI is either a live-path field (`file:line`) or an honest empty / `*_available: false` / `source: empty`. Invented series (linear interpolations of today's count, hardcoded `[100,250,200…]`) are **theatre**. Ban them. |
| **U2** | **Full dictionary, including zero** | If the backend enum has N values, the status board shows **N chips**. Zero is a real number, not a hidden row. Collapsing 18 order states into 4 bars is a product bug. |
| **U3** | **Pack owns money + clock** | Every amount uses session MarketPack `currency_code` + `decimal_places` (integer minor). Every timestamp uses pack timezone. Never hardcode `UZS` or `Asia/Tashkent` in a client. |
| **U4** | **Color is never the only encoding** | Status = chip label + icon + position + optional color. Color-blind and print-safe. |
| **U5** | **Empty ≠ zero ≠ unavailable** | Empty query → empty state + next action. Missing adapter / nil planner → `unavailable` + why. Zero completed today → `0`, not a skeleton forever. |
| **U6** | **Dual plane** | Factory trucks (`FactoryTruckManifests`) and supplier last-mile trucks (`SupplierTruckManifests`) never share a board, map layer, or KPI. |
| **U7** | **Role-row parity** | A visualization that ships for a role ships on **every client in that row**, or is an explicit desktop-handoff with owner + date (same rule as GS-R pin editor). |
| **U8** | **Desktop = command; phone = ritual** | Desktop/iPad landscape show many boards at once. Phone shows the **one ritual** plus a compact status strip. Same data, different density — not a different product. |
| **U9** | **One Plan & Brain page** | Planning (what-if, S&OP, MEIO, seasonal, promo sim) and Digital Brain (twin, knowledge graph, forecast accuracy, demand signals, governed agents) share **one route** with **tabs**. Dashboard is the live ops home, not a third brain. |
| **U10** | **Flags stay off** | Factory planning **place** and retailer auto-order **place** stay default-off. UI may **preview** (`SYSTEM_PREDICTED`, scenario results). UI must not look like a placed order. |
| **U11** | **Same-market honesty** | Cross-market / planned pack / `no_live_keys` / `geography_incomplete` render as the machine code + human sentence. Never a fake success tile. |
| **U12** | **Motion deferred** | This program does not specify animation. State changes may snap. Do not add entrance choreography, EKG flourish, or chart draw-on. Existing `framer-motion` wrappers stay; new work does not grow them. |
| **U13** | **Event-driven freshness** | A screen updates because a **scoped WS event** arrived, the user mutated, the user pulled, or a **safety-net poll** fired (dashboard ≥ 60s, fleet map ≥ 15s visible / 60s hidden). Never `setInterval(1000)` on a command dashboard. GPS / `DRIVER_LOCATION_UPDATED` patches the **marker only** — it must not refetch the whole dashboard. |
| **U14** | **Primary nav ≤ 5** | Phone tab bar and the first desktop sidebar group are the **ritual destinations** only. Everything else is a section, search (`⌘K`), a dashboard jump, or More. A 60-item sidebar is not enterprise — it is a sitemap dump. |
| **U15** | **Native chrome** | iOS uses Tab Bar + `NavigationSplitView` on iPad. Android uses NavigationBar + NavigationRail on tablet. Desktop uses sidebar + inspector. Do not ship a phone web-layout inside a native shell. System back, swipe-back, and predictive-back stay intact. |
| **U16** | **Cache on both sides** | Backend: `cache.Get` / `GetOrLoad` + `Invalidate` **after** commit (`apps/backend-go/cache/cache.go`). Frontend: hydrate from `packages/desktop-cache` (15m) / native store, then revalidate. Coalesce in-flight GETs (`singleflight` already exists server-side). |

**Explicit defer (this program):**

- Animation / motion design language (later GS-U-motion, not scheduled).
- H3 revenue heatmap (`RevenueHeatmap.tsx` unmounted — no revenue-density SoT).
- Linguistic-complete i18n.
- New backend domains just to feed a pretty chart. If the field is missing, either add a **thin honest read** in the owning slice or hide the widget.
- Cross-country FX checkout visualization.
- Flipping `checkout_reads_this`, terraform apply, live Stripe/Adyen/Soliq keys.

---

## 2. What already exists (opened this session)

### 2.1 Clients on disk

| Role (JWT) | Web | Native desktop | iOS | Android |
|------------|-----|----------------|-----|---------|
| Supplier `ADMIN` | `supplier-portal` | same app, Tauri 2 | `supplier-app-ios` | `supplier-app-android` |
| Retailer | `retailer-app-desktop` (Next) | same, Tauri 2 | `retailer-app-ios` | `retailer-app-android` |
| Warehouse | `warehouse-portal` | Tauri 2 | `warehouse-app-ios` | `warehouse-app-android` |
| Factory | `factory-portal` | Tauri 2 | `factory-app-ios` | `factory-app-android` |
| Payload | `payload-terminal` (Expo) | — | `payload-app-ios` | `payload-app-android` |
| Driver | **none** | — | `driver-app-ios` | `driver-app-android` |
| Platform admin | `admin-portal` | — | — | — |

Shared kits: `packages/ui-kit` (portal primitives, control-tower map, EKG graph), `packages/pulse-ui`, `packages/explain-ui`, `packages/mobile-ios-design`, `packages/mobile-android-design`. Charts today: **Recharts** on web.

### 2.2 Dashboards (PARTIAL)

| Surface | Route / screen | What it actually paints | Gap |
|---------|----------------|-------------------------|-----|
| Supplier portal `/dashboard` | `GET /v1/supplier/dashboard` | Revenue today (pack money), completion completed/attempted, 17-key StatusStack, health strip, manifests, honest history or unavailable, map, pulse | Truck duty unavailable (no rollup). Plan & Brain lives on `/planning`. |
| Supplier `/analytics` | velocity + revenue + demand; **Open in Brain** → `/planning?tab=brain` | Line charts for created/completed and revenue | History workbook only. Does not mount `PlanningBrainPanel`. |
| Supplier `/planning` | Planning \| Digital Brain tabs (`?tab=`). S&OP + scenarios + factory ops + MEIO; Brain = belief, sparsity, twin list, KG, signals, accuracy, governed agent | Tabs + live-path panels | Horizon query is context only (S&OP GET has no horizon). Factory twin plane stays unavailable (not merged). Place stays off. |
| Warehouse `/` | `GET /v1/warehouse/ops/dashboard` | Pending-dispatch KPI, 17-key order StatusStack, 8-key truck-duty stack, hold reasons, demand `source` chip, map, pulse | Completed today / revenue stay `*_available: false`. WMS/labor/coverage boards stay on their routes. |
| Factory `/` | `GET /v1/factory/dashboard` | `source` chip, transfer StatusStack, **Factory trucks** manifest stack, factory fleet/SLA/QC | Analytics history stays on `/analytics`. Place stays off. |
| Retailer `/dashboard` | pulse + AI predictions + KPI grid + reorder | Retail OS pulse tiles exist | Not a full status board of every order state / every supplier child. |
| Driver | pulse + earnings + queue | Ritual-first (correct) | No “all my stops by status” strip that matches the 18-state dictionary. |

### 2.3 Backend reads that UI may bind (KEEP)

Do **not** invent new engines. Bind these.

| Domain | Evidence | Visual job |
|--------|----------|------------|
| Order status union (18 + aliases) | `packages/types/index.ts:268-289` | Status stack, funnel, board |
| Supplier dashboard counters | `GET /v1/supplier/dashboard` | Command KPI strip |
| Supplier velocity / revenue | `SupplierAnalyticsVelocityResponse`, `SupplierAnalyticsRevenueResponse` | History line / area |
| Demand today | `SupplierDemandSummaryResponse` + `baseline_source` + `ForecastConfidence` | Ranked SKU bars + honesty chip |
| Warehouse demand forecast | `GET /v1/warehouse/demand/forecast` `series[]` + `source` | Combo demand vs committed |
| Planning S&OP | `GET /v1/supplier/planning/s-and-op` | Capacity utilization gauge |
| Scenarios | run / list / compare / clone / publish | What-if compare |
| MEIO | `GET /v1/supplier/meio/network-summary` | Network inventory health |
| Seasonal + sparsity + promo sim | planning routes on `supplierroutes` | Planning tab |
| Knowledge graph | `GET /v1/supplier/knowledge-graph` | Brain graph |
| Twin routes | `twin/http_handler.go` active routes + inventory | Brain map / route twin |
| Control tower scored exceptions + playbooks | `controltower/types.go` | Exception board |
| Pulse | `PulseResponse` + retailer `RetailerControlTowerPulse` | Event rail |
| Route performance | `analytics/handlers.go` planned vs actual stops/duration + replan | Scatter / grouped bar |
| Forecast accuracy | `GET /v1/admin/planning/accuracy` (`mape28`) | Accuracy line (admin + supplier brain) |
| Coverage / pins | warehouse `/coverage`, supplier warehouse pins | Map (view); edit stays desktop handoff on mobile |
| Fleet live map | existing `FleetLiveMapPanel` | Spatial now |
| Pack session | `GET /v1/auth/session` | Currency, locale, receipts label, `checkout_reads_this` chip |

### 2.4 Known theatre to **delete from UI** until the read is honest

| ID | Code | UI must do until fixed |
|----|------|------------------------|
| **U-T1** | Warehouse dashboard sparklines invented — `warehouse/ops_portal.go:252-254` | Do not bind `sparkline_*`. Show `history_available: false`. |
| **U-T2** | Warehouse `completed_today` / `today_revenue` always `0` — `:239-245` | Hide those KPIs or label `unavailable`, do not show `0` as truth. |
| **U-T3** | Supplier `fleet_vu_used = len(orders)*10` — `supplier/portal_ops.go:607` | Hide VU used or replace with real manifest VU sum. |
| **U-T4** | Factory dashboard in-memory / demo lock — closed U5 | `source: spanner\|memory\|empty` is always on the JSON. Memory only when `FACTORY_PORTAL_SEED`. |
| **U-T5** | Supplier dashboard `formatCurrency(…, "UZS")` — portal dashboard (closed U2) | Pack `formatPackMoney`. Empty pack prints the number only. |

---

## 3. Target information architecture (one picture)

```
Role home = COMMAND DASHBOARD
  ├─ Now strip          (KPI + health + pack chip + clock)
  ├─ Status boards      (orders · manifests · trucks · drivers · money · exceptions)
  ├─ Spatial now        (map / H3 exceptions / coverage — role-appropriate)
  ├─ History ribbon     (7d / 30d / 90d honest series OR unavailable)
  └─ Ritual rail        (the 3–5 next actions for this role)

Plan & Brain  = ONE ROUTE, TWO TABS
  ├─ Tab Planning       (S&OP, scenarios, MEIO, seasonal, promo, pull-matrix preview)
  └─ Tab Digital Brain  (twin, knowledge graph, forecasts, signals, accuracy, agents)

Deep work stays on existing routes
  orders · dispatch · WMS · POS · credit · treasury · catalog · settings
  Dashboard tiles deep-link. They do not become a second app.
```

**Why dashboard ≠ Plan & Brain**

- Dashboard answers: *what is happening, what is stuck, what do I do in the next hour?*
- Planning answers: *what if factory downtime / demand +10% / seasonal template?*
- Digital Brain answers: *what does the network believe, how confident, which twin/route/SKU is the reason?*

They share **range, warehouse, factory, SKU filters** via a sticky context bar so a click on the dashboard “stockout SKUs” opens Brain with that SKU selected.

---

## 4. Visualization language (canonical)

One catalog for web (Recharts + tables), iOS (Swift Charts + lists), Android (Vico or Compose Canvas + lists). **Same encoding.** If a platform cannot draw a type, it falls back to the **table** in the same card.

### 4.1 Chart → data job

| Job | Chart | Max series / cats | Phone fallback |
|-----|-------|-------------------|----------------|
| Counts by **status** | **Status stack** (100% stacked bar) + chip rail with every key | All enum keys | Chip rail only (wrap) |
| Conversion through lifecycle | **Funnel** (ordered statuses) | One plane (supplier **or** factory) | Numbered list + % |
| Amount / units over time | **Line** (1–2 series) or **area** (composition) | 3 series | Sparkline + last value |
| Plan vs actual | **Grouped bar** or **scatter** (planned_x, actual_y) | 1 metric | Two-column table |
| Utilization vs cap | **Bullet / meter** (used / cap / alert tick) | 1 | Big % + cap number |
| Health (SLA, fiscal, cold, credit) | **Health strip** (ok / warn / fail counts) | 3 buckets | 3 tiles |
| Ranked SKUs / retailers | **Horizontal bar** | 12 rows, “+N more” | Same, top 5 |
| Mix of many SKUs | **Treemap** (desktop/iPad only) | 30 cells | Ranked bar |
| Spatial now | **Map** + H3 cells (existing control-tower / coverage) | One plane | List + “open map” |
| Network belief | **Graph** (knowledge-graph nodes/edges) | 80 nodes then cluster | Typeahead + neighbor list |
| Work queue | **Board** (columns = state) | 5 columns on phone via pager | Horizontal pager |
| Shift / wave / SLA window | **Timeline** (rows = people or waves) | Desktop/iPad | Agenda list |
| Money path | **Waterfall** (gross → fiscal → fee → net) | 6 steps | Stacked numbers |
| Capacity vs demand | **Combo** (bars demand, line capacity) | 2 | Two numbers + delta |

**Banned unless the SoT exists:** pie with >5 slices, 3D, gauges with decoration, dual-axis tricks, interpolated sparklines, “EKG” as a substitute for a missing series.

### 4.2 Card anatomy (every widget)

```
┌ header ─────────────────────────────────────────┐
│ Title                    range ▾   ↻   ↗ open   │
│ source: spanner|empty|unavailable   as-of 14:02 │
├ body ───────────────────────────────────────────┤
│ [chart OR table OR empty/unavailable]           │
├ footer ─────────────────────────────────────────┤
│ n entities · pack currency · deep-link label    │
└─────────────────────────────────────────────────┘
```

Required on every card:

- **Title** (user language, not handler name)
- **As-of** timestamp
- **Source** chip (`spanner` / `empty` / `unavailable` / `memory` / `env_default`)
- **Range** if the query is a window (`today` / `7d` / `30d` / custom)
- **Empty / error / loading** reserved height (no CLS)
- **Table alternative** (visually hidden on desktop if chart is showing; primary on VoiceOver / TalkBack)
- **Deep link** to the list that produced the numbers

### 4.3 Status dictionaries (UI must paint **every** key)

#### Orders (`OrderStatus` — `packages/types/index.ts:268-289`)

Canonical columns, left → right (funnel). Aliases map into the canonical column and show a small “alias” hint if the wire sent `DISPATCHED` / `EN_ROUTE` / `ARRIVING`.

| Key | Board column | Health |
|-----|--------------|--------|
| `PENDING` | Queue | info |
| `SCHEDULED` | Queue | info |
| `AUTO_ACCEPTED` | Queue | info |
| `BACKORDERED` | Queue | warn |
| `LOADED` | Yard | warn |
| `IN_TRANSIT` | Road | info |
| `DELAYED` | Road | warn |
| `ARRIVED` | Door | info |
| `ARRIVED_SHOP_CLOSED` | Door | warn |
| `AWAITING_PAYMENT` | Money | warn |
| `PENDING_CASH_COLLECTION` | Money | warn |
| `DELIVERED_ON_CREDIT` | Money | info |
| `FISCALIZING` | Fiscal | info |
| `FISCAL_FAILED` | Fiscal | fail |
| `RECONCILIATION_REQUIRED` | Fiscal | fail |
| `COMPLETED` | Done | ok |
| `CANCELLED` | Done | mute |

Aliases: `DISPATCHED` → `LOADED` (or yard), `EN_ROUTE` → `IN_TRANSIT`, `ARRIVING` → `ARRIVED`.

#### Manifests (supplier last-mile)

`DRAFT` · `LOADING` · `SEALED` · `DISPATCHED` · `COMPLETED` · `CANCELLED`  
Active for capacity: `DRAFT|LOADING|SEALED|DISPATCHED` (`manifest/store.go`).

Factory plane uses the **same labels** on a **separate** board titled “Factory trucks”. Never mix IDs.

#### Trucks / driver duty (`truck_status` seen in warehouse + bootstrap)

| Key | Meaning | Board |
|-----|---------|-------|
| `AVAILABLE` | On shift, no active in-transit manifest | Ready |
| `IN_TRANSIT` | On an active last-mile / factory route | Road |
| `RETURNING_TO_WAREHOUSE` | Off-shift reason mapped in `driverOffShiftTruckStatus` | Return |
| `OFF_SHIFT` | Declared off | Off |
| `UNASSIGNED` | No vehicle | Hold |
| `VEHICLE_INACTIVE` | Vehicle flag | Hold |
| `UNAVAILABLE` / `INACTIVE` | Driver inactive | Hold |

Vehicle hold reasons (show on the vehicle row, not as a fake truck status): `MAINTENANCE` · `TRUCK_DAMAGED` · `REGULATORY_HOLD` · `MANUAL_HOLD` · `OTHER` (`warehouse/fleet_availability.go`).

#### Drivers (orthogonal flags — **do not collapse into truck_status**)

`is_active` · `is_online` · `on_shift` · `unavailable_reason` · assigned `vehicle_id` · `max_volume_vu`.

Dashboard driver board = one row per driver with **all** of these, plus current manifest state if any.

#### Fiscal

`NONE` · `PENDING` · `SUCCESS` · `FAILED` · `FORCE_SKIPPED`.

#### Control-tower playbook runs

`SUGGESTED` · `APPROVED` · `EXECUTED` · `FAILED` · `SKIPPED`.

#### Planning scenarios

`DRAFT` · `PUBLISHED` · `SUPERSEDED` · `REJECTED`.

#### Retailer pulse (`RetailerControlTowerPulse`)

`open_orders` · `active_fulfillments` · `dock_pending` · `pos_open_sessions` · `open_shifts` · `open_assist_tickets` · `low_stock_sku_bins` · `shift_variances_7d` · `sales_minor_7d` · `capabilities[]` · `empty`.

### 4.4 Number, money, time

- Money: integer minor → `Intl` / `FormatStyle.Currency` / Android `NumberFormat` with **pack** currency. Tabular figures.
- Percents: one decimal max; show `n/d` in the tooltip.
- Time: pack TZ; range chips `Today` `7d` `30d` `90d`; custom on desktop.
- Distance: pack `distance_unit` for **display only**.
- Temperature: pack °C/°F on cold-chain.

### 4.5 Density + type

| Token | Phone | Tablet | Desktop |
|-------|-------|--------|---------|
| Page margin | 16 | 24 | 32 |
| Card gap | 12 | 16 | 16 |
| KPI value | 28/34 | 32/40 | 36/44 |
| Body | 16 | 16 | 14–16 |
| Chip | 28h, 44h hit | 28h | 24h |
| Table row | 52 | 48 | 40 |
| Chart height | 160 | 220 | 260–320 |

Type: existing portal MD typescale / iOS Dynamic Type / Material type roles. Tabular numbers on every KPI and table. Contrast ≥ 4.5:1. No indigo-as-brand (keep the existing monochrome + semantic tokens in `packages/ui-kit` / `PegasusMonochromeTheme`).

---

## 5. Responsive + platform shells

### 5.1 Breakpoints (all web/desktop shells)

| Name | Width | Columns | Nav |
|------|-------|---------|-----|
| `phone` | 320–767 | 1 | Bottom bar ≤5 + overflow “More” |
| `tablet` | 768–1023 | 2 | Rail (Android) / sidebar compact (iPad portrait) |
| `tablet-wide` | 1024–1279 | 12 (use 6+6 or 4+8) | Persistent rail + list/detail |
| `desktop` | 1280–1919 | 12 | Persistent sidebar (existing `*Shell.tsx`) |
| `wide` | 1920+ | 16 | Sidebar + optional third inspector |

**No horizontal page scroll.** Charts reflow: stacked bar → horizontal bar; 12-col bento → single column. Maps get a min height of 220 and a “expand” control on phone.

### 5.2 Platform mapping

| Platform | Shell | Dashboard layout | Plan & Brain |
|----------|-------|------------------|--------------|
| **Web** (browser) | Existing `*Shell` sidebar | 12-col bento | Tabs under `/planning` |
| **Desktop native** (Tauri) | Same web shell + window chrome + keyboard (`⌘K` search already a kit target) | Same as web; prefer 2–3 visible boards without scroll on 1440×900 | Same route; extra inspector pane on `wide` |
| **iOS phone** | Tab Bar ≤5 | Vertical: Now strip → status pager → ritual | Segmented control Planning \| Brain |
| **iPad** | `NavigationSplitView` sidebar + detail | 2-col boards + map | Tabs + trailing inspector |
| **Android phone** | NavigationBar ≤5 | Same as iOS phone, Material 3 | Secondary tabs |
| **Android tablet** | `NavigationRail` + list/detail | Same as iPad | Same |

**Handoff (allowed, dated):** dense editors (topology draw, pin polygons, EDI certs, playbook JSON) may stay desktop-primary with a native “Open on desktop” card — same pattern as warehouse pin edit (GS-R, deadline already 2026-09-16). **Read visualizations are not handoff.** If the phone can show a number, it shows the number.

### 5.3 Shared chrome (every role, every platform)

1. **Pack chip** — currency, market code, `checkout_reads_this` honesty (already GS-R).
2. **As-of + connection** — last successful fetch; WS live / polling / offline.
3. **Scope** — supplier (implicit), warehouse, factory, retailer location. Changing scope refetches every card.
4. **Range** — persisted per role in local store; default `today` on dashboard, `7d` on history ribbon, `horizon_days` on Planning.
5. **Search** — jump to order / driver / SKU / retailer (desktop/tablet). Phone uses the existing search screens.

### 5.4 Page zones (every command screen)

Enterprise layout is **zones**, not a pile of cards. Same zones on every role so muscle memory transfers.

```
┌ chrome ──────────────────────────────────────────────────────────┐
│ brand · pack · scope · range · as-of · search · bell · account   │
├ nav ─┬ work ──────────────────────────────────────┬ inspector ───┤
│  ≤5  │  A  Now / ritual (primary CTA)             │  selected    │
│  +   │  B  Status boards (orders/trucks/drivers)  │  entity      │
│  More│  C  Spatial (map) — desktop/tablet only    │  or empty    │
│      │  D  History ribbon                         │  hint        │
│      │  E  Queue / table                          │              │
└──────┴────────────────────────────────────────────┴──────────────┘
```

| Zone | What lives here | What does **not** |
|------|-----------------|-------------------|
| **Chrome** | Pack, scope, range, live/offline, search, notifications | Settings, topology editors |
| **Primary nav** | Home, the role ritual, Plan & Brain, one money/ops, More | Import CSV, FX rates, playbook JSON |
| **A Now** | 3–5 KPIs + the one next action | Full 18-chip encyclopedia (that is B) |
| **B Status** | Dictionaries + health strips | Raw JSON dumps |
| **C Spatial** | One plane’s map | The other plane’s trucks |
| **D History** | Honest series or `unavailable` | Fake sparks |
| **E Queue** | Virtualized table / board | Infinite unvirtualized lists |
| **Inspector** | Selected order/driver/SKU (tablet-wide+) | A second full page |

Phone **hides C and Inspector**. Tap a row → push a detail screen (native) or sheet (desktop <1280).

---



## 6. Role rituals → surfaces

A **ritual** is the job the human is hired to do in the next 15 minutes. Dashboard serves the ritual. Deep routes serve the case.

| Role | Primary ritual | Dashboard must answer | Deep routes stay |
|------|----------------|----------------------|------------------|
| Supplier | “Is cash + trucks + exceptions healthy?” | Money today, **all** order statuses, fleet duty, scored exceptions, fiscal fails | orders, dispatch, finance, topology |
| Warehouse | “What do I load, who is free, what is short?” | Pending dispatch, **full** truck/driver dictionary, low stock, pick/cycle/cold health, coverage | dispatch execute, WMS, supply requests |
| Factory | “What is in the bay, what is late, what is sealed?” | Transfer states, **factory** manifests, SLA board, QC gate, staff on shift | loading-bay, payload, transfers |
| Retailer | “What did I buy, what is arriving, what is in the store?” | Pulse, orders by status **per supplier child**, dock, POS/shift, stock, credit | catalog, POS, stock, credit, HQ |
| Driver | “What is my next legal doorstep action?” | Current stop, payment/fiscal state, remaining stops by status, cash to reconcile | scanner, offload, shop-closed |
| Payload | “Which truck can I seal / inject?” | Trucks by manifest state, open seals, inbound returns | seal, inject, exceptions |
| Platform admin | “Which tenant is stuck, which outbox is dead?” | Tenants by KYB, flags pending dual-control, dead-letters, billing | flags, partner, audit |

---

## 7. Phased modular program (GS-U)

```
GS-U0  Visualization contract     dictionaries + honesty + pack money
GS-U1  Shared viz kit             web + iOS + Android primitives
GS-UN  Navigation + placement     primary ≤5; sections; native chrome
GS-UF  Freshness + cache          WS + cache; no 1s poll; ETag
GS-U2  Supplier command           full status boards + honest history
GS-U3  Plan & Brain               one route, two tabs
GS-U4  Warehouse command          delete fake sparks; full fleet dictionary
GS-U5  Factory command            Spanner-sourced; dual-plane
GS-U6  Retailer command           pulse + per-supplier status + store OS
GS-U7  Field (driver + payload)   ritual boards with full stop/truck states
GS-U8  Platform admin             ops visualization (web only)
GS-U9  Role-row + responsive lock every U2–U8 surface on the role row
```

A slice is **not done** until: live-path bind (`file:line`) · no invented series · every dictionary key visible · phone + tablet + desktop (or explicit dated handoff) · tests for empty / unavailable / zero · **nav matches §24** · **refresh matches §26**.

Dependency: **U0 → U1 → (UN ∥ UF)** then **(U2 ∥ U4 ∥ U5 ∥ U6)** then **U3** then **U7 ∥ U8** then **U9**. UN/UF can land in parallel with the first command dashboard.

---

## 8. GS-U0 — Visualization contract

**Goal:** One typed dictionary and honesty helpers. No new screens.

**In scope**

- `packages/types`: `OrderStatus` already exists. Add `ManifestState`, `TruckDutyStatus`, `FiscalStatus` (exists), `PlaybookRunStatus` as **const arrays** used by UI (`ORDER_STATUS_FUNNEL`, `TRUCK_DUTY_STATUSES`, …).
- `packCurrency(session)` helper used by every portal KPI (kill hardcoded `UZS`).
- `HistorySeries` type: `{ points, source, available, generated_at }`. `available: false` is first-class.
- Backend: warehouse dashboard stops emitting fake `sparkline_*` / fake revenue (or marks `*_available: false`). This is a **honesty fix**, not a feature.
- Backend: supplier dashboard aggregate counts **all** `OrderStatus` keys (zeros included). Do not keep the 6-key map.
- Backend: supplier `fleet_vu_used` either real sum from open manifests or omitted.

**Out of scope:** new charts, Plan & Brain.

**Exit**

- `go test` on supplier dashboard aggregate + warehouse dashboard honesty.
- Type arrays exported; no client imports a 4-item “live statuses” local list.

**U-items owned:** U-T1, U-T2, U-T3, U-T5 (contract + warehouse JSON). U-T4 closed in U5.

---

## 9. GS-U1 — Shared visualization kit

**Goal:** One component family. Web first in `packages/ui-kit`, then iOS/Android twins.

| Component | Props (min) | Renders |
|-----------|-------------|---------|
| `StatusStack` | `counts: Record<Status, number>`, `dictionary`, `onSelect` | Stacked bar + chip rail including zeros |
| `KpiStat` | `label`, `value`, `delta?`, `spark?: HistorySeries`, `source` | Number + optional spark **only if available** |
| `HealthStrip` | `ok`, `warn`, `fail`, labels | 3 cells + icons |
| `TimeSeries` | `series[]`, `unit`, `range` | Line/area; empty/unavailable states |
| `ComboCapacity` | `demand[]`, `capacity[]` | Bar+line |
| `BulletMeter` | `used`, `cap`, `alertAt` | Utilization |
| `EntityBoard` | columns, rows, `onOpen` | Kanban / pager |
| `RangeToggle` | `today\|7d\|30d\|90d` | Segmented |
| `SourceChip` | `source` | Honesty |
| `PlanBrainTabs` | `planning` \| `brain` | Tabs (U3) |
| `CommandGrid` | children, breakpoints | 12-col → 1-col |

iOS: Swift Charts + existing `PegasusStateViews`. Android: existing design module + a small chart lib **or** Compose Canvas for `StatusStack` only.

**Exit:** Story / preview for empty, zero, unavailable, 18-status stack. One portal page swapped to `StatusStack` without visual regression on loading.

**Shipped 2026-08-16:** `StatusStack`, `SourceChip`, `KpiStat` spark guard, `guardHistorySeries`, `statusStackModel`. Preview covers empty / zero / unavailable / 17-key funnel (`ORDER_STATUS_FUNNEL`). Supplier portal Live orders + supplier iOS/Android dashboards bound. `PlanBrainTabs` shipped in U3. HealthStrip / TimeSeries / ComboCapacity / BulletMeter / EntityBoard / RangeToggle / CommandGrid stay for later slices.

**Motion:** none beyond platform default press states.

---

## 9a. GS-UN — Navigation + placement

**Goal:** The next action is where the hand already is. Deep tools are one search or one More tap away — not 40 siblings in the sidebar.

**In scope:** rewrite `NAV` in `SupplierShell`, `WarehouseShell`, `FactoryShell`, `RetailerShell` and native section enums. **Do not delete routes.** Collapse them under sections + command palette.

**U-items**

| ID | Gap | Fix |
|----|-----|-----|
| **U-N1** | Supplier `NAV` is ~60 hrefs; `/planning`, `/analytics`, `/analytics/knowledge-graph`, `/ai/recommendations` are siblings | Primary: Home · Orders · Dispatch · Plan & Brain · More. Analytics/KG/AI live **inside** Plan & Brain or More → Insights |
| **U-N2** | Retailer desktop is one flat list (dashboard through HQ) | Primary: Home · Buy · Incoming · Store · More. POS/shifts under Store when pack on |
| **U-N3** | Warehouse mixes dispatch, WMS, fleet, finance as peers | Primary: Home · Dispatch · Floor (WMS) · Plan · More |
| **U-N4** | Two “Planning” labels (`/planning` and `/settings/planning`) | Settings stays flags/kill-switch; `/planning` is the workbench |

**Exit:** each shell’s first group has ≤5 items. Playwright: Home / ritual / Plan reachable in one click. Native tab bars ≤5.

**Shipped 2026-08-16:** U-N1–U-N4. First group ≤5 on supplier/warehouse/factory/retailer shells. Overflow collapsed. `/planning` ≠ `/settings/planning`. Native compact tabs ≤5. Routes kept. Driver/payload/platform nav not this slice.

Full trees and flows: **§24–§25**.

---

## 9b. GS-UF — Freshness + cache (enterprise scale)

**Goal:** A 200-warehouse supplier with 40 open dashboards does **not** generate hundreds of GETs per second.

**In scope (backend + clients)**

- Dashboard rollup GET supports `If-None-Match` / `ETag` → 304.
- Cache keys for dashboard rollups: `dash:supplier:{id}:today`, TTL 15–30s; invalidate on order/manifest/fiscal outbox (after commit). Use existing `cache.GetOrLoad` + `singleflight`.
- Client: WS `onSignal` **coalesces 400–800ms** (supplier hook already defaults `debounceMs = 500` — `use-supplier-ws-refresh.ts:28`) and refetches **only the dirty card**, not `refresh()` of the whole page.
- `DRIVER_LOCATION_UPDATED` → map marker patch only. Not in `PULSE_REFRESH_EVENTS` for dashboard GET.
- Replace factory `setInterval` fleet poll with `usePolling` + `hiddenIntervalMs: 60_000`.
- Safety-net poll floors (do not go faster):

| Surface | Visible | Hidden | Trigger besides poll |
|---------|---------|--------|----------------------|
| Command dashboard Now | 60s | pause | WS status/money events (debounced) |
| Fleet / live map | 15s | 60s | location WS (marker only) |
| History / analytics | 5 min | pause | range change |
| Plan & Brain snapshot | on open + 2 min | pause | user Run |
| Driver now card | WS + 30s | pause | doorstep mutation |
| Notifications | WS | — | badge only |

**Must not:** 1s / 2s / 5s dashboard timers; refetch-all on every Kafka type; N+1 GET per status chip; poll while offline.

**Exit:** load test note or unit: 50 coalesced location events → 0 dashboard GETs. Factory fleet uses `usePolling`. Dashboard 304 path tested.

**Shipped 2026-08-16:** ETag 304 on supplier/warehouse/factory dashboard GETs. `dash:{role}:{scope}:today` + invalidate after order/manifest commit. Location → marker patch only. Factory fleet off `setInterval`. Safety-net floors: dashboard 60s (pause hidden), fleet 15s/60s.

Full architecture: **§26**.

---

## 10. GS-U2 — Supplier command dashboard

**Route:** `/dashboard` (portal + Tauri). Native: existing `DASHBOARD` section.

### 10.1 Desktop / tablet-wide (12-col)

```
[ PackChip ] [ Warehouse▾ All ] [ Range today|7d|30d ] [ as-of · live ]

[ Revenue today          ][ Completion %        ][ Retailers today/total ]
[ pack currency + Δ      ][ completed/attempted ][ activity               ]

[ ORDER STATUS STACK …………………… 8 col …………………… ][ HEALTH 4 col ]
[ every OrderStatus chip + funnel                     ][ fiscal fail   ]
[                                                     ][ shop-closed   ]
[                                                     ][ CT open       ]

[ MANIFESTS by state  4 ][ TRUCK DUTY 4 ][ DRIVERS 4 ]
[ DRAFT…COMPLETED       ][ all duty keys][ on_shift / online / unassigned ]

[ MAP 8  fleet live + exception H3 ][ PULSE 4 ]

[ HISTORY 12 — velocity line + revenue area   OR unavailable ]
[ RECENT MANIFESTS table + EXCEPTIONS scored list            ]
```

Phone: vertical pager — **Now** (3 KPIs) → **Orders** (`StatusStack`) → **Fleet** (trucks+drivers) → **Map** (collapsed) → **History** → **Pulse**. Bottom tab stays Home / Orders / Dispatch / More.

iPad: split — master = status stacks, detail = map + selected entity.

### 10.2 Widgets × fields × chart × API

| Widget | Fields | Chart | API |
|--------|--------|-------|-----|
| Revenue today | `today_revenue_minor`, `currency` from pack, Δ vs yesterday **only if** a real yesterday series exists | KPI + spark from analytics revenue | `GET /v1/supplier/dashboard` + `…/analytics/revenue` |
| Completion | `delivery_completion_rate_pct` + completed/attempted counts (expose both) | KPI | dashboard (extend if attempted missing) |
| Retailer activity | `retailers_ordered_today`, `total_retailers` | KPI fraction | dashboard |
| Order status stack | `orders_by_status[k]` for **every** `OrderStatus` | `StatusStack` + funnel | dashboard (U0 full map) |
| Manifest board | counts by `DRAFT…COMPLETED` | `StatusStack` | existing manifests list **or** add count rollup on dashboard (thin read) |
| Truck duty | counts by `TruckDutyStatus` | `StatusStack` | fleet list rollup (do not use `AVAILABLE` vs `IN_TRANSIT` only) |
| Drivers | `total_drivers`, `active_drivers` (on loaded/in_transit/arrived), `on_shift`, `is_online`, unassigned | Health strip + table | dashboard + fleet |
| VU | `fleet_vu_used` / `fleet_vu_total` **only if real** | `BulletMeter` | dashboard after U0 |
| Fiscal health | count `FISCAL_FAILED` + `RECONCILIATION_REQUIRED` | Health strip | `orders_by_status` |
| Shop-closed | `ARRIVED_SHOP_CLOSED` + `/v1/supplier/shop-closed/active` | KPI → list | existing |
| Control tower | scored exceptions count by severity | Health strip | `GET /v1/control-tower/exceptions/scored` |
| Map | live drivers + exception H3 | Map (existing) | fleet-live + exception-map |
| Pulse | `PulseEvent[]` | list | `GET /v1/supplier/pulse` |
| History velocity | `points[].orders_created/completed` | Line (2 series) | `GET` analytics velocity |
| History revenue | `series[].revenue_minor` | Area | analytics revenue |
| Demand today | `items[]` qty + `baseline_source` + `confidence` | H-bar top 10 | demand today |
| Payment configured | `is_configured` | banner (exists) | dashboard |
| Planning peek | utilization + capacity_alert | `BulletMeter` + “Open Plan & Brain” | S&OP (read-only peek) |

### 10.3 Interactions (not animation)

- Tap a status chip → `/orders?status=` (preserve range + warehouse).
- Tap fiscal fail → orders filtered `FISCAL_FAILED`.
- Tap a driver row → fleet / org-fleet driver sheet.
- Range change refetches history cards only; **Now** stays today unless the user pins a historic day (desktop).
- WS `ORDER_STATUS_CHANGED` increments the matching chip (optimistic) and reconciles on next GET.

### 10.4 Honesty

- If `orders_by_status` omits a key, UI still shows `0` from the client dictionary (U0 should make this unnecessary).
- If analytics GET 503 / empty: history card `unavailable` or `empty`, never a sine wave.
- Do not show yesterday Δ until a real previous-bucket exists.

**Exit:** portal + Android + iOS show 18 order chips. Playwright / XCTest / Compose test: a `FISCAL_FAILED` order increments that chip. Pack currency on revenue. No `UZS` literal.

**Shipped 2026-08-16:** 17-key funnel (U0 SoT; aliases fold in). Portal StatusStack + HealthStrip + pack money + honest history. Native revenue KPI + FISCAL_FAILED increment tests. No invented yesterday Δ. Truck duty not invented.

---

## 11. GS-U3 — Plan & Brain (one page, two tabs)

**Route:** `/planning` (supplier portal + Tauri). Native: replace / extend existing Planning settings + any JSON dumps with this page. Warehouse keeps `/demand-forecast` as a **node-scoped** Plan tab (no publish). Factory sees SLA + capacity **read** only unless `FACTORY_PLANNING_ENABLED` (still default off).

Analytics `/analytics` **stops embedding** `PlanningBrainPanel`. Analytics becomes the **history workbook** (velocity, revenue, route performance, demand flywheel) and links “Open in Brain” for SKU belief.

### 11.1 Shared context bar (both tabs)

```
Warehouse▾  Factory▾  Horizon 7|14|28  SKU search  [Planning | Digital Brain]
```

Context is query-string so a dashboard deep-link works: `/planning?tab=brain&sku=…`.

### 11.2 Tab A — Planning

Desktop:

```
[ S&OP meters 12 ]
  factory_capacity_units · warehouse in/out cap · utilization_pct · capacity_alert
  capacity_model / capacity_source (factories_column | env_default)  ← honesty

[ Scenario workbench 8 | Results 4 ]
  downtime hours · demand_delta_pct · horizon · Run
  saved scenarios list (status DRAFT/PUBLISHED/…)
  select 2 → Compare (delta table: sla_risk, fleet_volume, revenue_at_risk, stockouts, capacity_breach)

[ MEIO network 6 | Seasonal 6 ]
  network-summary
  builtin templates + overrides + estimate

[ Promo simulate 6 | Pull-matrix / predictive-push 6 ]
  simulate + performance GET
  preview only; 409 factory_planning_disabled when flag off — show the 409, do not hide the control
```

Charts:

| Widget | Chart | Fields |
|--------|-------|--------|
| S&OP utilization | `BulletMeter` + combo if demand vs cap series exists | `PlanningSAndOPSnapshot` |
| Scenario compare | Grouped bar of the 4 numeric deltas + stockout count | `PlanningScenarioCompareResult` |
| Revenue at risk | KPI minor + pack currency | `revenue_at_risk_minor`, `unit_value_source` |
| Stockout SKUs | H-bar / chip list | `stockout_skus[]` |
| Seasonal | Timeline of overrides on a year axis (desktop); list on phone | `SeasonalTemplatesResponse` |
| Promo | Before/after units (two bars) | simulate response |
| Sparsity | Health chip per retailer | `SparsityGateResult` (`allowed`, `completed_orders`, `label`) |

Phone: S&OP meters → Run scenario sheet → last result → “More planning” accordion (MEIO, seasonal, promo).

**Place stays off:** predictive-push / pull-matrix show `preview` / `disabled` with the server code. Never a “Orders placed: N” tile.

### 11.3 Tab B — Digital Brain

This is the **belief + twin** surface. Not a second S&OP.

```
[ Belief health 12 ]
  ForecastConfidence (low/high/pct/label/blocked_reason)
  Sparsity on selected retailer
  Accuracy MAPE (if admin/supplier accuracy GET is in role)

[ Twin map 8 | Route twin 4 ]
  GET twin active routes (zone H3)
  selected route inventory
  Dual-plane toggle: last-mile | factory  (two fetches, two layers — never merged)

[ Knowledge graph 12 ]
  nodes/edges; filter by type; click → dashboard entity
  Phone: search + neighbor list (no 80-node force graph)

[ Demand signals 6 | Accuracy 6 ]
  HOLIDAY/WEATHER/EVENT/PROMO · scope
  ForecastAccuracyDaily WAPE / signed error line

[ Governed agent 12 ]
  invoke approve_insight / open_supply_request / broadcast_template
  audit of last invocations — human-in-the-loop only
```

| Widget | Chart | API |
|--------|-------|-----|
| Confidence | `BulletMeter` + band on forecast line | `ForecastConfidence` on demand + forecast handlers |
| Twin routes | Map + polyline | `twin` active + get route |
| Twin inventory | Table VU/SKU on vehicle | `…/inventory` |
| Knowledge graph | Graph (desktop), list (phone) | `GET /v1/supplier/knowledge-graph` |
| Signals | Timeline + scope chips | `GET/POST` planning signals |
| Accuracy | Line MAPE/WAPE | `GET /v1/admin/planning/accuracy` (supplier may need a scoped read — if 403, hide with `unavailable`, do not fake) |
| Agents | Action list + result | `POST /v1/supplier/planning/agent/invoke` |

### 11.4 Warehouse / factory variants

| Role | Planning tab | Brain tab |
|------|--------------|-----------|
| Warehouse | `GET /v1/warehouse/demand/forecast` series (projected / committed / pending) + replenishment insights. **No** publish scenario. | Node twin: this warehouse coverage + inbound transfers + forecast `source`. |
| Factory | SLA board + daily capacity (`Factories.DailyOutputCapacity` or `env_default` honesty). Transfer-transit SLA only if flag on. | Factory-plane manifests + QC + exceptions. No last-mile trucks. |
| Retailer | Auto-order **preview** + AI predictions confirm/reject (existing). Not a supplier S&OP. | Sell-through + reorder suggestions + pulse — keep `/insights`; do **not** clone supplier Brain. |

**Exit:** one `/planning` route, two tabs, deep-link `?tab=`. Analytics no longer mounts `PlanningBrainPanel`. Native apps have the same two tabs (iPad split, phone segmented). Tests: sparsity blocked → Brain shows `blocked_reason`, no invented forecast line. Flag-off push → 409 banner.

**Shipped 2026-08-16:** `/planning` Planning | Digital Brain + `?tab=`. Analytics “Open in Brain” only. Portal `DigitalBrainPanel` + iOS/Android segmented tabs. `blocked_reason` on ForecastConfidence; `brainForecastLine` null when blocked or <2 points. 409 `factory_planning_disabled` on flag-off push. `FACTORY_PLANNING_ENABLED` stays off. `/settings/planning` kept.

---

## 12. GS-U4 — Warehouse command

**Route:** `/` portal. Native `DASHBOARD`.

### 12.1 Delete theatre first

Unbind `sparkline_*`, `today_revenue`, `completed_today` until U0 gives real series or `*_available: false`. The current JSON **must not** be drawn as history (`ops_portal.go:237-254`).

### 12.2 Boards

| Widget | Dictionary / fields | Chart | API |
|--------|---------------------|-------|-----|
| Orders now | **full** `OrderStatus` for this warehouse, not PENDING/LOADED/IN_TRANSIT/ARRIVED only | `StatusStack` | extend dashboard rollup or client-side from `GET /v1/warehouse/ops/orders` (cap + server counts preferred) |
| Pending dispatch | `pending_dispatch` | KPI → dispatch | dashboard |
| Truck duty | **all** `TruckDutyStatus` + vehicle hold reasons | `StatusStack` + table | drivers list (`PortalDriver.truck_status`, reasons) — **not** idle=everything-except-IN_TRANSIT |
| Drivers | `on_shift`, `is_online`, `is_active`, assigned vehicle, VU | Entity table | same |
| Vehicles | `is_active`, hold reason, assigned driver, class, `max_volume_vu` | Table | vehicles list |
| Inventory health | `low_stock_count` (define threshold in copy) + bins | KPI + H-bar top shortages | inventory + bins |
| WMS health | open pick waves, cycle counts due, cold excursions | Health strip | existing WMS GETs |
| Labor | capacity vs on-shift hours (pack TZ / max hours) | `BulletMeter` | labor-capacity |
| Coverage | cells / cities / pins (view) | Map | `GET /v1/warehouse/ops/coverage` (PUT 405 — view only) |
| Supply | open supply requests + QC state | Board | supply-requests |
| Demand | `series[]` projected vs committed | `ComboCapacity` | demand forecast (`source` chip) |
| Exceptions / CT | scored list | list | control-tower |
| Tomorrow board | existing | agenda | tomorrow-board |
| Pulse | events | list | `/v1/warehouse/ops/pulse` |

Desktop: map + dispatch queue + fleet table on first screen. Phone: ritual = **Dispatch** first, then status pager.

**Exit:** no fake sparkline in the DOM. Fleet board shows `OFF_SHIFT`, `RETURNING_TO_WAREHOUSE`, `UNASSIGNED`, `VEHICLE_INACTIVE`, hold reasons. Demand card shows `source: empty` honestly when planner empty (already a backend pattern).

**Shipped 2026-08-16:** Dashboard JSON emits `orders_by_status` (17 keys), `truck_duty` (8 keys), `hold_reasons`, `demand_source`. `fleet_status` is the full duty array (not idle=everything-except-IN_TRANSIT). Portal `/` StatusStacks + demand chip; no date-range control; KpiStat spark still guarded. iOS/Android same stacks. Tests: Go full funnel + duty keys + empty demand; portal no sparkline; Android decode + increment.

---

## 13. GS-U5 — Factory command (dual plane)

**Route:** `/` factory portal. Native `DASHBOARD`.

### 13.1 Honesty

`HandleDashboard` emits `source: spanner|memory|empty` and prefers Spanner counts (transfers, `FactoryTruckManifests`, exceptions, staff). Memory is allowed only when `FACTORY_PORTAL_SEED` and is labeled. **Shipped 2026-08-16.**

### 13.2 Boards

| Widget | Fields | Chart | Notes |
|--------|--------|-------|-------|
| Transfers | `CREATED` `APPROVED` `PENDING` `ASSIGNED` `LOADING` `DISPATCHED` `CANCELLED` (use live state enum from transfer rows, not a guessed set) | `StatusStack` | Factory plane only |
| Factory manifests | `DRAFT` `LOADING` `SEALED` `DISPATCHED` `COMPLETED` | `StatusStack` | **Not** supplier manifests |
| SLA board | request due `kind: supply_request`; hours from pack | Timeline + fail/warn/ok | Existing `GET /v1/factory/sla-board` |
| QC gate | PASS / FAIL / missing | Health | supply-request QC |
| Bay | loading-bay slots / seal | Existing bay grid | Keep; add counts on dashboard |
| Fleet (factory) | vehicles READY/AVAILABLE vs not; drivers on_shift | Duty stack | Factory drivers only |
| Exceptions | `ManifestExceptions` ⋈ factory manifests | list | `transfer_id` field honesty (already PARTIAL) |
| Insights | warehouse replenishment insights (factory-gated GET) | H-bar | existing |
| Analytics | `ProductionForecastChart` only if series is real | line | keep `/analytics` as history |

**Never** show last-mile `IN_TRANSIT` retailer orders as factory trucks.

**Exit:** dashboard JSON includes `source`. Spanner-backed counts in tests. Native + portal parity. Dual-plane copy in the header: “Factory trucks”. **Done 2026-08-16.**

---

## 14. GS-U6 — Retailer command (Retail OS)

**Routes:** `/dashboard` desktop; iOS `DashboardView`; Android home + pulse.

Retailer is **multi-supplier**. Every order widget groups by **child order / supplier**, never a blended fake total that hides a stuck child.

| Widget | Fields | Chart | API |
|--------|--------|-------|-----|
| Pulse strip | all `RetailerControlTowerPulse` fields | 8 tiles (already sketched) | `GET /v1/retailer/control-tower/pulse` |
| Orders by status | full dictionary, **faceted by supplier** | `StatusStack` + supplier legend | `GET /v1/orders` rollup (add server facet if list cap hides states) |
| Tracking now | active fulfillments | map + status | `GET /v1/retailer/tracking` + `active-fulfillment` |
| Dock | `dock_pending` | KPI → `/dock` | pulse |
| POS / shifts | open sessions, open shifts, cash variance 7d | Health | pulse + POS/shift GETs |
| Store stock | `low_stock_sku_bins` | KPI + H-bar | stock |
| Money 7d | `sales_minor_7d` pack currency | KPI + spark from reports if real | pulse + reports |
| Credit / AR | open invoices, overdue | Health | credit-profile + AR |
| AI predictions | pending preorders | list confirm/reject | `GET /v1/retailer/ai/predictions` — not SKU DemandForecast |
| Loyalty | `{enrolled:false}` honesty | card | loyalty tier |
| Insights peek | sell-through | H-bar | `/insights` |
| Auto-order | rules + last preview | list | settings auto-order — **place** flag-off |

Phone: pulse tiles are the home (keep). Tablet: pulse + tracking map. Desktop: pulse + status stack + map + insights peek.

HQ (franchise): same widgets with **location** facet (`switch-location`).

**Exit:** a child order `FISCAL_FAILED` at supplier A is visible on the retailer stack without opening `/orders`. Pulse `empty: true` shows empty, not demo tiles. No Bronze loyalty when unenrolled.

**Shipped 2026-08-16.** Pulse `source` + 17-key `orders_by_status` + `orders_by_supplier` child facets. Desktop `/dashboard`, iOS `DashboardView`, Android home bind the stack. Loyalty stays `{enrolled:false}` on pulse; no invented Bronze.

---

## 15. GS-U7 — Field (driver + payload)

### 15.1 Driver (phone primary; iPad split list/detail)

No portal. Ritual > dashboard.

| Widget | Fields | Visual |
|--------|--------|--------|
| Now card | current stop, order status, doorstep legal actions | one primary CTA (arrive / scan / cash / credit / shop-closed) |
| Remaining stops | each stop’s `OrderStatus` | vertical stepper (not a 18-chip wall) |
| Manifest | state `DRAFT…COMPLETED`, VU used/total | `BulletMeter` |
| Money | pending cash, open fiscal, credit-leave | Health strip |
| History | `GET /v1/driver/history` 30d completed | list; empty `[]` is fine |
| Earnings | existing earnings GET | KPI pack currency |
| Offline | sync queue depth | banner |

Tablet: left = stop list with status dots; right = now card + map.

### 15.2 Payload (terminal + phone + tablet)

| Widget | Fields | Visual |
|--------|--------|--------|
| Trucks | each truck `truck_status` **from manifest state** (`payload/tablet_wire.go`) | board by `DRAFT/LOADING/SEALED/DISPATCHED` |
| Seal queue | unsealed orders | list + seal-all (exists) |
| Inject | eligible orders | sheet |
| Inbound returns | inbound sessions | list |
| Capacity | **do not** call GONE `/v1/payload/capacity/{id}` | VU on the manifest only |

Tablet (Expo + native): two-pane truck list / seal detail — this is the **dock** layout.

**Exit:** payload board columns = manifest states, not a single “trucks” count. Driver stepper shows `ARRIVED_SHOP_CLOSED` and `FISCAL_FAILED` as first-class. No capacity theatre.

**Shipped 2026-08-16.** `HandleTrucks` attaches `truck_status` + VU from the current DRAFT/LOADING/SEALED/DISPATCHED manifest (`payload/board.go`). COMPLETED / empty is not invented as DRAFT. Capacity route stays 410. Payload Android `ManifestBoard`, iOS `ManifestBoard`, terminal `manifestBoard.ts` group four columns. Driver iOS/Android remaining-stops stepper + money strip + history/earnings peek. Used VU on the driver meter is unavailable (not 0/max).

---

## 16. GS-U8 — Platform admin (web only)

| Widget | Fields | Visual | API |
|--------|--------|--------|-----|
| Tenants | KYB / pack / cell / `IsRegistered` | table + status chips | tenants |
| Flags | pending dual-control | list | flags |
| Outbox | summary + dead-letters | Health + table | `/ops/outbox/*` |
| Runtime | workers / capabilities | key/value | `/ops/runtime` |
| Billing | invoices + fee schedules | table money | billing |
| Planning accuracy | `mape28`, `demoted` | line | `GET /v1/admin/planning/accuracy` |
| Match queue | entity matches | list | match queue |
| Partner | keys / AS2 / SFTP | table | partner |

No mobile. Desktop-width only; still must not horizontally scroll at 1280.

**Exit:** dead-letter count is a real query, not a placeholder card.

**Shipped 2026-08-16.** `countOutboxDeadLetters` is `SELECT COUNT(*) FROM OutboxDeadLetters`. Summary/list omit `dead_letter_count` when unavailable. Admin Command board + Ops panel bind that KPI (`deadLetterHealth`). Pending flags `GET /v1/platform-admin/flags`. Accuracy is a live table (not an invented Recharts line). Tenant pack/cell chips; `IsRegistered` stays unavailable on this list.

---

## 17. GS-U9 — Role-row + responsive lock

Walk §1 of pegasus-doctrine for every U2–U8 widget:

| Check | Pass |
|-------|------|
| Same fields on web, Tauri, iOS, Android (role row) | Yes, or dated skip (admin web-only; driver no portal) |
| Phone 1-col usable without pinch-zoom | Adaptive grids / 1-col stacks |
| iPad / Android tablet 2-col or split | `horizontalSizeClass` / window size class |
| Dynamic Type / font scale does not clip KPIs | Caption/label type; chip min 44/48 |
| Color not the only status encoding | Chip **label** + count |
| Empty / unavailable / zero all tested | U1 kit + per-app command tests |
| Deep link from notification → the same card filter | Chip tap uses the same status/state key |
| Pack currency / TZ | No invented UZS on command boards |
| Dual plane respected | Factory trucks ≠ last-mile |
| No new animation dependency | No 1s poll; no new motion kit |

Handoffs still allowed: topology polygon edit, pin draw, EDI cert upload, playbook authoring. **All read boards are in-app.**

**Exit:** lock test row per widget (`gs-u9-role-row-lock.test.ts`) — not “Wired” as a vibe. Dated skips: admin mobile, driver portal, Plan & Brain off supplier row, warehouse truck-duty filter, factory vehicle/driver filter, desktop handoffs.

**Shipped 2026-08-16.** Native `StatusStack` chips call `onSelect` with the dictionary key. Supplier `resolveSupplierOrdersQuery` uses `status=` for funnel keys. Retailer filters the live list by canonical status (+ supplier facet). Warehouse/factory pass `state=` into existing list queries. Factory trucks chip still opens loading-bay (same as web).

---

## 18. Per-role dashboard inventory (implementation checklist)

Use this as the build list. “Must visualize” = backend already has a durable read or a thin count can be added in the same slice. “Must not visualize” = GONE/theatre.

### 18.1 Supplier

**Must visualize:** every `OrderStatus`; every last-mile `ManifestState`; truck duty + driver flags; revenue today + velocity/revenue history; demand today + confidence; S&OP utilization; scenarios + compare; MEIO summary; KG; twin routes (brain); scored exceptions; shop-closed active; credit program health; payout rail honesty (`no_live_rail`); chargebacks/mismatches counts; returns inbound; inventory SKU count + low stock if listed; pack/PSP catalog (no Stripe on UZ); topology coverage as map **view**.

**Must not:** negotiations (410 unless flag); saved cards; inventory audit (410); invented VU; hardcoded UZS; H3 revenue heatmap; factory trucks on the last-mile map; placed auto-orders / placed factory push.

### 18.2 Warehouse

**Must visualize:** warehouse-scoped orders by **full** status; dispatch preview queue; **full** truck/driver/vehicle dictionary + hold reasons; pick waves / cycle / bins / cold / labor; supply requests + QC; transfers; demand series + `source`; coverage view; CRM retailer rollup (existing JSON); treasury invoices when query returns rows (`[]` vs 503); payment-config = pack catalog; control tower; tomorrow board; pulse.

**Must not:** fake sparklines; fake `today_revenue`; retailer-chosen warehouse; Stripe on UZ; coverage radius as catchment (display hint only); PUT coverage (405).

### 18.3 Factory

**Must visualize:** transfer states; **factory** manifests; SLA board; QC; bay seal; factory fleet; exceptions; staff on shift; insights; payload/load (factory plane).

**Must not:** supplier last-mile board; `pick_n_created_v1` labeled as live when Spanner path exists; unlabeled demo seed; transfer-transit SLA when flag off (hide or “flag off”).

### 18.4 Retailer

**Must visualize:** pulse; orders/status **per supplier**; tracking; dock; POS/shifts/sections/assist; store stock + local SKUs; reports that exist; credit/AR; sell-through; AI preorders; loyalty enrolled honesty; payment catalog (no invented Adyen); HQ multi-location.

**Must not:** saved cards; request-cancel 403 as a button that looks live; B2B 410 checkout; currency picker; warehouse picker; fake Bronze; `/v1/ai/predictions` alias.

### 18.5 Driver / Payload / Admin

As §15–16. Must not: negotiate 410; `PATCH /v1/orders/{id}/state` 501; payload capacity 410; driver force-complete.

---

## 19. Interaction grammar (no motion brief)

| Action | Result |
|--------|--------|
| Tap KPI | Filtered list for that metric |
| Tap status chip | Toggle filter (multi-select on desktop) |
| Tap chart point | Inspector: entity sample IDs + “open list” |
| Tap map marker | Same inspector |
| Change range | History cards refetch; Now stays today |
| Change warehouse / location | Whole dashboard refetch |
| Pull-to-refresh (native) | Same as ↻ |
| WS event | Patch the matching chip/row; do not reset scroll |
| Offline | Last as-of + banner; mutations queue only where the role already queues (driver/POS) |
| 403 / 404 / 501 / 422 | Machine `code` + sentence + next step (pack / keys / desktop) |
| Flag-off / 409 | Disabled preview with server code |

Destructive actions stay on deep routes with confirm. Dashboard is **read + filter + jump**.

---

## 20. Accessibility + performance (non-optional)

- Contrast 4.5:1; status not color-only; focus rings stay.
- Hit targets 44×44 (iOS) / 48 (Android); chip rails scroll rather than shrink below 44.
- Chart has `aria-label` / `accessibilityLabel` summarizing the insight (“18 pending, 2 fiscal failed”).
- Tables sortable on desktop (`aria-sort`).
- Virtualize lists ≥ 50 rows (already a kit rule).
- Reserve chart height. No layout jump when series arrives.
- Prefer one dashboard GET + card-level GETs. Do not N+1 a GET per chip.
- Stale reads OK for history (15s). Strong reads for Now money and fiscal fail counts.

---

## 21. What “done” is not

- A redesigned color theme.
- A new animation system.
- A second tenant key or a merged truck plane.
- Cloud apply, live PSP/fiscal keys.
- A 1-second “live” spinner that proves nothing except load.
- “We listed 50 widgets therefore they exist.” A widget is done when the live path is bound and the empty/unavailable/zero states are tested on the role row.

---

## 22. Suggested first implementation slice

When asked to implement (not this doc):

1. **U0** warehouse sparkline honesty + supplier full `orders_by_status` + pack currency helper.
2. **U1** `StatusStack` + `SourceChip` + `HistorySeries` guard. **Done 2026-08-16.**
3. **UN** collapse supplier + retailer primary nav to ≤5; `/planning` is the Plan & Brain entry. **Done 2026-08-16.**
4. **UF** dashboard dirty-card refresh + ETag; factory fleet off `setInterval`. **Done 2026-08-16.**
5. **U2** supplier command order stack on portal, then Android, then iOS. **Done 2026-08-16.**
6. **U3** `/planning` tabs; remove Planning panel from `/analytics`. **Done 2026-08-16.**
7. **U4** warehouse command stacks; no fake sparks. **Done 2026-08-16.**
8. **U5** factory command; `source` + Factory trucks plane. **Done 2026-08-16.**
9. **U6** retailer command; pulse source + per-supplier child stack. **Done 2026-08-16.**
10. **U7** field (driver + payload) ritual boards. **Done 2026-08-16.**
11. **U8** platform admin command; dead-letter COUNT(*). **Done 2026-08-16.**

U9 shipped 2026-08-16. Leftovers are continuous (flag, cells apply, live PSP). Not Layer B.

---

## 23. Pointers

| Doc | Role |
|-----|------|
| This file | Client visualization program (GS-U) |
| [`GLOBAL_SCALE_PROGRAM.md`](./GLOBAL_SCALE_PROGRAM.md) | Register + pack + cells |
| [`GLOBAL_SCALE_LOCAL_ECOSYSTEM.md`](./GLOBAL_SCALE_LOCAL_ECOSYSTEM.md) | Local matching + pack PSP |
| [`FEATURES_BY_APP_ROLE.md`](./FEATURES_BY_APP_ROLE.md) | Route + nav inventory |
| [`ROLE_ROW_PARITY_MATRIX.md`](./ROLE_ROW_PARITY_MATRIX.md) | Role × platform (Wired ≠ this program) |
| [`PROD_ECOSYSTEM_GOAL.md`](./PROD_ECOSYSTEM_GOAL.md) | Class A coverage, not destination |

---

## 24. Navigation trees (placement)

**Rule:** keep the **URL**. Change the **chrome**. Power users reach anything via search.

### 24.1 Supplier (`ADMIN`)

**Primary (sidebar first group / phone tabs)**

| # | Label | Route | Why here |
|---|-------|-------|----------|
| 1 | Home | `/dashboard` | Command Now |
| 2 | Orders | `/orders` | Case work from a chip |
| 3 | Dispatch | `/dispatch` | Ritual: put boxes on trucks |
| 4 | Plan | `/planning` | Planning \| Digital Brain tabs |
| 5 | More | overflow | Everything else |

**More / desktop sections** (collapsed, not 60 peers)

| Section | Contains (existing routes) |
|---------|----------------------------|
| Fleet | `/fleet`, `/fleet/orders`, `/manifests`, `/ops/map`, `/labor-capacity`, `/org-fleet` |
| Floor & stock | `/inventory`, `/inventory/import`, `/catalog`, `/pricing*`, `/promotions`, `/replenishment/suggestions` |
| Network | `/topology`, `/warehouses`, `/factories`, `/delivery-zones`, `/supply-lanes`, `/coverage` handoff, `/crm`, `/loyalty` |
| Exceptions | `/exceptions`, `/control-tower`, `/activity`, `/returns` |
| Insights | `/analytics`, `/analytics/demand*`, `/analytics/route-performance`, `/analytics/knowledge-graph` → **also Brain tab**, `/ai/recommendations` |
| Money | `/treasury*`, `/payments`, `/earnings`, `/finance/*`, `/ledger`, `/reconciliation`, `/chargebacks*`, `/credit/*`, `/compliance` |
| Settings | `/profile`, `/settings/*`, `/entity-resolution` |

**Phone tabs:** Home · Orders · Dispatch · Plan · More.  
**iPad / Android tablet:** rail = the 5; detail = page; trailing inspector on order/driver select.

### 24.2 Warehouse

| # | Label | Route |
|---|-------|-------|
| 1 | Home | `/` |
| 2 | Dispatch | `/dispatch` |
| 3 | Floor | `/inventory` (hub: bins, pick-waves, cycle, cold as **sub-tabs**) |
| 4 | Plan | `/demand-forecast` (warehouse Plan tab; Brain = coverage + inbound) |
| 5 | More | fleet, supply, returns, treasury, coverage, staff, settings |

Do **not** put `/dispatch-settings` and `/dispatch-locks` in primary — they are overflow under Dispatch.

### 24.3 Factory

| # | Label | Route |
|---|-------|-------|
| 1 | Home | `/` |
| 2 | Bay | `/loading-bay` |
| 3 | Payload | `/payload` (factory plane only) |
| 4 | Transfers | `/transfers` (supply-requests as a tab on this page) |
| 5 | More | fleet, staff, SLA/insights, analytics, settings |

### 24.4 Retailer

| # | Label | Route | Pack |
|---|-------|-------|------|
| 1 | Home | `/dashboard` | — |
| 2 | Buy | `/catalog` (cart drawer lives here; procurement + my-suppliers as sub-tabs) | `order.place` |
| 3 | Incoming | `/tracking` (dock as sub-tab) | `dock.receive` |
| 4 | Store | `/stock` or `/pos` if that pack is the daily ritual | STORE_STOCK / POS |
| 5 | More | credit, auto-order, insights, control-tower, reports, HQ, settings |

Capability packs **reveal** Store children; they do not add a 6th tab.

### 24.5 Driver (no sidebar)

| # | Label | Screen |
|---|-------|--------|
| 1 | Now | current stop + primary CTA |
| 2 | Route | remaining stops stepper |
| 3 | Money | cash / fiscal / credit |
| 4 | Sync | offline queue (badge) |
| 5 | Me | profile, availability, history |

Scanner is a **button on Now**, not a tab.

### 24.6 Payload

| # | Label | Screen |
|---|-------|--------|
| 1 | Dock | trucks by manifest state |
| 2 | Seal | selected truck |
| 3 | Inbound | returns |
| 4 | More | exceptions, profile |

Tablet: list of trucks | seal detail (native split).

### 24.7 Platform admin (web)

Tenants · Flags · Ops (outbox/dead-letters) · Billing · More (audit, match, partner). No mobile.

---

## 25. Screen + feature flows

These are the **paths a trained operator runs**. Every widget’s deep-link must land **mid-flow**, not on a blank home.

### 25.1 Shared grammar

```
Home (Now)
  → tap chip / row
    → List (filtered)          [desktop: same page inspector]
      → Case (order / stop / wave)
        → Legal action (confirm)
          → Home chip updates via WS (no full reload)
```

Back always returns to the **filtered list**, not Home. Phone uses the system back stack. Desktop keeps the list mounted.

### 25.2 Supplier — morning command → stuck order

```
1. Open Home. Zone A: revenue, completion, fiscal-fail count.
2. Fiscal health > 0 → tap → /orders?status=FISCAL_FAILED
3. Open case → fiscal retry (existing ADMIN path) or jump to driver
4. WS ORDER_STATUS_CHANGED → chip −1. Stay on the list.
```

### 25.3 Supplier — dispatch ritual

```
1. Home truck board: AVAILABLE vs IN_TRANSIT vs OFF_SHIFT
2. Pending chip → /dispatch (preview already scoped)
3. Execute → manifests DRAFT increment (WS MANIFEST_DRAFT_CREATED)
4. Do not navigate to /ops/map unless the user opens spatial
```

### 25.4 Supplier — Plan & Brain

```
1. Tab Planning: S&OP meters. Run scenario. Compare two. Publish is a
   deliberate footer action (not next to Run).
2. Tab Brain: sparsity blocked → no forecast line. Twin map is last-mile
   XOR factory (toggle).
3. Dashboard “Open in Brain” → /planning?tab=brain&sku=
4. /analytics is history only (velocity, revenue, route-performance).
```

### 25.5 Warehouse — load the wave

```
1. Home: pending_dispatch + drivers AVAILABLE
2. Dispatch → preview → execute (existing)
3. Floor → pick-waves (sub-tab) → complete
4. Supply short → supply-request (More or Floor overflow)
5. Coverage is view-only; pin edit = desktop handoff
```

### 25.6 Factory — bay seal (dual plane)

```
1. Home: factory manifests LOADING + SLA warn
2. Bay → start-loading / seal (factory APIs only)
3. Payload tab = same plane, not last-mile payloaderoutes
4. QC FAIL blocks accept (existing 409) — banner on the transfer row
```

### 25.7 Retailer — buy → door

```
1. Buy (catalog) → cart drawer → checkout (pack PSP only)
2. Home pulse open_orders +
3. Incoming (tracking) when IN_TRANSIT / ARRIVED
4. Dock when dock_pending
5. Child order of supplier B stuck does not hide inside a blended total
```

### 25.8 Retailer — store OS (packs)

```
Home pulse tile POS / Shifts / Stock / Assist
  → that Store sub-route
POS park/resume stays on POS (not a new tab)
```

### 25.9 Driver — doorstep legal path

```
Now: one primary CTA (arrive | scan | cash | credit | shop-closed)
  → success → next stop auto-advances
Money tab only for pending cash / open fiscal
Scanner is presented from Now, then popped
```

### 25.10 Payload — seal

```
Dock board → select truck (LOADING)
  → Seal tab → per-order or seal-all
Inject is a sheet on the selected truck, not a home CTA
```

### 25.11 Empty / blocked flows

| Block | Placement | Next action |
|-------|-----------|-------------|
| `geography_incomplete` | Home banner | Open warehouse location (desktop) |
| `no_live_keys` | Money / checkout | Honest 501 copy; no fake pay button |
| `factory_planning_disabled` | Plan tab, next to push | Leave control visible, disabled |
| Pulse `empty: true` | Home | “No open work” + Buy / Dispatch CTA |
| History `unavailable` | Zone D | Hide chart; do not draw zeros |

---

## 26. Enterprise freshness (backend + frontend)

### 26.1 Why this is a product law

A naive “live dashboard” (`setInterval(1000)` × N cards × M clerks) will DDoS the cell. Location telemetry is high-frequency. Status changes are not. Treat them differently.

**Already good (keep):**

- `usePolling` 60s dashboard, pause when hidden — `packages/api-client/usePolling.ts`
- Fleet 15s / hidden 60s — `supplier-portal/lib/use-fleet-live-map.ts`
- WS debounce default 500ms — `use-supplier-ws-refresh.ts:28`
- Desktop hydrate `cacheSet` / 15m TTL — `packages/desktop-cache/kv.ts`
- Server `cache.GetOrLoad` + `singleflight` + invalidate-after-commit — `cache/cache.go`
- Doctrine: EmitNotification is UX; outbox is truth

**Already wrong (fixed in UF unless noted):**

- Warehouse invented sparklines (U-T1 — honesty shipped in U0; delete leftover if any)
- Factory fleet `setInterval` — **fixed** (`usePolling` 15s / hidden 60s)
- Any `refresh()` of the **entire** dashboard on `DRIVER_LOCATION_UPDATED` — **fixed** (marker patch)
- Per-card uncoordinated 30s polls on factory pages that could share one WS — factory Home is 60s now; other factory pages not this slice

### 26.2 Write path (backend — already the doctrine)

```
mutation
  → Spanner txn + outbox.EmitJSON
  → commit
  → cache.Invalidate(dash:…, orders:…)     // AFTER commit
  → kafka.EmitNotification(eventType)      // best-effort WS/FCM
```

Do **not** add a second “dashboard pusher.” The existing hubs + `ws-refresh-contract` event sets are the bus.

### 26.3 Read path (backend)

| Read | Consistency | Cache |
|------|-------------|-------|
| Dashboard Now rollup | strong or 5–15s stale | `dash:{role}:{scope}:{day}` TTL 15–30s; ETag |
| Status counts | same rollup | included in dashboard JSON — **one GET** |
| History velocity/revenue | stale 15s OK | TTL 2–5 min |
| Fleet positions | last telemetry write | **not** in dashboard GET |
| Plan S&OP | on demand | short TTL; bust on scenario publish |
| Twin / KG | on tab focus | TTL 60s |

`If-None-Match` on dashboard + history: unchanged body → **304**. Clients treat 304 as “as-of unchanged.”

### 26.4 Read path (frontend)

```
mount
  1. paint from desktop-cache / native KV if age < 15m (stale-while-revalidate)
  2. GET rollup (send ETag). 200 → replace; 304 → keep
  3. subscribe WS room (supplier:id / warehouse:id / …)

on WS event
  1. if location-only → patch marker; return
  2. debounce 400–800ms; coalesce types
  3. GET only the dirty slice (orders rollup XOR money XOR exceptions)
  4. write through to desktop-cache

on hidden
  pause dashboard poll; fleet drops to 60s
on visible / online / focus
  one GET if last fetch > 15s (do not stampede)

safety-net poll
  dashboard 60s if no WS for 60s
```

**Dirty slices** (never “refetch the world”):

| Event set (`ws-refresh-contract`) | Refetch |
|-----------------------------------|---------|
| `ORDER_STATUS_*` / assign / amend | orders stack + Now KPIs that derive from it |
| `MANIFEST_*` / `DISPATCH_*` | manifest + truck boards |
| `PAYMENT_*` / fiscal | money + fiscal health |
| `SHOP_CLOSED*` | shop-closed + door column |
| `PULSE_*` | pulse list only |
| `DRIVER_LOCATION_UPDATED` | **map only** |
| scenario publish | Plan tab only |

### 26.5 Client cache keys

Use `scopedCacheKey` (already in `desktop-cache`) so org/location/supplier switches cannot leak:

`{role}:{supplier|warehouse|factory|retailerOrg}:{location?}:{resource}:{range}`

Clear prefix on logout and on org/location switch (retailer already has `clear-org-scoped-state`).

Native: same key scheme in SQLite / DataStore / Swift `URLCache` with the 15m default.

### 26.6 Backpressure

`usePolling` already listens for `backpressure` CustomEvent. If the API returns `Retry-After` or 429, dispatch that event (supplier payments already backoff). UF: do this on **all** dashboard GETs.

### 26.7 What we will not build

- A client-side “realtime bus” besides existing WS hubs
- GraphQL subscriptions
- Per-widget 1s SWR
- Server-sent pixels / canvas push
- Polling as the primary live path when WS is connected

---

## 27. Native interaction (feel, not animation)

| Concern | iOS | Android | Desktop / web |
|---------|-----|---------|---------------|
| Primary nav | Tab Bar, 5, label+icon | NavigationBar, 5 | Sidebar sections; first group = the 5 |
| Large screen | `NavigationSplitView` | `NavigationRail` + list/detail | Sidebar + inspector ≥1280 |
| Back | swipe-back; do not steal edge | predictive back | breadcrumbs + `Esc` closes inspector |
| Search | tab/More search | top app bar | `⌘K` / `Ctrl+K` |
| Refresh | pull on Home / lists only | same | ↻ in chrome; no auto-spin |
| Selection | tap row → push | tap row → push | tap row → inspector; `Enter` opens case |
| Primary CTA | trailing / bottom of Now | FAB only if it is **the** ritual (dispatch execute, seal) | header button on the ritual page, not on Home |
| Forms | native fields; 44pt | outlined 48dp | visible labels; validate on blur |
| Destructive | confirm sheet | same | dialog; not on Home |
| Offline | banner + queued (driver/POS only) | same | banner; mutations fail closed except those queues |
| Density | roomy | roomy | denser tables; same 18 chips |

Home is **not** a settings page. Settings never take a primary tab.

---

## 28. Component placement cheat-sheet

Put the component in the zone that matches its **job**. If you cannot name the zone, it does not ship on Home.

| Component | Zone | Pages allowed |
|-----------|------|----------------|
| Pack chip, as-of, range | Chrome | all |
| 3–5 KPIs | A Now | Home only |
| `StatusStack` (orders/trucks/drivers) | B | Home; filtered clone on the list page |
| Health strip (fiscal, SLA, cold) | B | Home |
| Fleet / coverage / twin map | C | Home (collapsed on phone), `/ops/map`, Brain tab |
| History line/area | D | Home ribbon; full on `/analytics` |
| Pulse list | E | Home (5 rows) + notifications |
| Scenario form | Plan tab | `/planning` only |
| Knowledge graph | Brain tab | `/planning?tab=brain` (redirect old `/analytics/knowledge-graph`) |
| Cart | Buy | drawer on catalog — not a tab |
| Scanner | Driver Now | presented, not a tab |
| Playbook editor | More → Control tower | desktop primary |
| Topology / pin draw | Network / coverage | desktop handoff on phone |
| Payment catalog | checkout sheet / billing | never a Home widget of Stripe on UZ |
| Kill-switch / flags | Settings | never Home |

---

## 29. UN / UF checklist (add to U9)

- [ ] Primary nav ≤5 on every client in the role row
- [ ] Old URLs still resolve (redirect or same page, new chrome)
- [ ] Plan & Brain is one entry; analytics is history
- [x] No dashboard GET on location events
- [x] WS debounce ≥400ms; dirty-card refetch only
- [x] Dashboard ETag 304
- [x] Poll floors respected; hidden tab paused or 60s
- [x] Factory fleet off raw `setInterval`
- [ ] Cache keys scoped; cleared on org/location switch
- [ ] Pull-to-refresh does not stampede (singleflight) |
