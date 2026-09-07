# V.O.I.D & Pegasus Unified UI Design System — Master Contract

> [!IMPORTANT]
> **BINDING AGENT RULE**: All AI agents, subagents, and human engineers generating or modifying user interfaces (web portals, Tauri v2 desktop apps, mobile screens, or design components) MUST strictly follow this specification. Do NOT generate generic SaaS cards, generic rounded boxes, low-contrast washed-out cards, or default Tailwind template styling. Every UI element must reflect the exact aesthetic, palette, typography, elevation, and layout patterns codified below, derived from the 72 reference design assets in this repository.

---

## 1. Visual Identity & Design Philosophy

The V.O.I.D & Pegasus visual language is an **Ultra-High-Density Tactical Control Tower & Industrial Precision Suite**. It balances two high-performance modes:
1. **Tactical Dark Mode (Primary Control Tower)**: Pitch-black canvas (`#09090B`), deep obsidian surfaces (`#121216`), crisp 1px hairline borders (`#22222C`), electric cobalt blue and safety orange accents, and glowing micro-indicators.
2. **Crisp Operational Light Mode (High-Density CRM & Data Table)**: Bright slate canvas (`#F8FAFC`), pure white cards (`#FFFFFF`), subtle slate dividers (`#E2E8F0`), and dark midnight slate typography (`#0F172A`).

### Core Aesthetic Pillars:
- **Hairline Precision**: 1px crisp borders (`border border-[var(--desk-border)]`) replace heavy, muddy drop shadows. Active states use delicate border highlights or inner/outer glow rings (`ring-1 ring-blue-500/30`), never blurred grey card shadows.
- **Data-Dense Tabular Typography**: Monospace numeric indicators (`font-mono tabular-nums`) for currency, quantities, ETAs, timestamps, coordinates, and VIN/plates.
- **Tactical Micro-Accents**: 10px-11px uppercase tracked category chips, circular quick action nodes, discrete LED block progress indicators, and radial speedometer gauges.
- **Clear Information Hierarchy**: Bold, geometric headings (`Plus Jakarta Sans`), high-contrast secondary labels (`#94A3B8` in dark, `#475569` in light), and muted tertiary micro-captions (`#64748B` in dark, `#94A3B8` in light).

---

## 2. Canonical Color Palette & Tokens

All components MUST reference CSS variables or semantic Tailwind classes. **Never hardcode arbitrary hex values into component files.**

### Theme Variable Reference Table

| Token Name | Dark Value (Primary) | Light Value | Semantic Usage |
|:---|:---|:---|:---|
| `--desk-canvas` | `#09090B` | `#F8FAFC` | App root background, deep page canvas |
| `--desk-surface` | `#121216` | `#FFFFFF` | Standard card background, panel surface |
| `--desk-surface-subtle` | `#181820` | `#F1F5F9` | Hover states, table row alternating tint |
| `--desk-surface-sunken` | `#0D0D11` | `#E2E8F0` | Input wells, sunken badge backgrounds |
| `--desk-surface-raised` | `#191924` | `#FFFFFF` | Modals, flyout menus, floating bars |
| `--desk-border` | `#22222C` | `#E2E8F0` | Standard 1px hairline card/divider border |
| `--desk-border-strong` | `#333342` | `#CBD5E1` | Active/focused border, hover border |
| `--desk-text-primary` | `#F8FAFC` | `#0F172A` | Primary headings, prominent values |
| `--desk-text-secondary` | `#94A3B8` | `#475569` | Subheadings, descriptions, labels |
| `--desk-text-tertiary` | `#64748B` | `#94A3B8` | Micro-labels, table headers, breadcrumbs |
| `--desk-accent` | `#3B82F6` | `#2563EB` | Electric Cobalt: primary CTA, active tabs |
| `--desk-accent-orange` | `#FF7A1A` | `#F97316` | Safety Orange: urgent alerts, sequence badges |
| `--desk-accent-lime` | `#E2FD52` | `#65A30D` | Tactical Lime: financial yield, telemetry badges |
| `--desk-accent-purple` | `#8B5CF6` | `#7C3AED` | Iris Purple: AI dispatch, optimization, missions |
| `--desk-accent-cyan` | `#06B6D4` | `#0284C7` | Tactical Cyan: sensor telemetry, live tracking |
| `--desk-success` | `#10B981` | `#059669` | Emerald Mint: active, on route, healthy, in stock |
| `--desk-warning` | `#F59E0B` | `#D97706` | Amber Gold: pending, delayed, low stock |
| `--desk-danger` | `#EF4444` | `#DC2626` | Crimson Red: blocked, expired, rejected, cancelled |

---

## 3. Typography Architecture

- **Primary Sans-Serif**: `Plus Jakarta Sans`, `-apple-system`, `BlinkMacSystemFont`, `Segoe UI`, `Roboto`, `sans-serif`.
  - Weights: `400` (Regular body), `500` (Medium UI items), `600` (Semi-bold subheadings), `700`/`800` (Bold metrics & headers).
- **Monospace Technical & Numbers**: `JetBrains Mono`, `Geist Mono`, `ui-monospace`, `monospace`.
  - MUST be used on: `tabular-nums` numbers, currencies, ETAs, coordinates, license plates, order IDs, and telemetry readouts.
- **Hierarchy Scale**:
  - `Hero / Page Title`: 24px–28px, Bold (`font-bold tracking-tight text-[var(--desk-text-primary)]`)
  - `Section / Card Title`: 16px–18px, Semi-Bold (`font-semibold text-[var(--desk-text-primary)]`)
  - `Body / Descriptions`: 13px–14px, Regular (`text-[13.5px] text-[var(--desk-text-secondary)]`)
  - `Micro Badges & Section Headers`: 10px–11px, Bold Uppercase (`text-[11px] font-bold uppercase tracking-wider text-[var(--desk-text-tertiary)]`)
  - `Metric Value Display`: 28px–36px, Bold Tabular Monospace (`text-3xl font-bold font-mono tabular-nums tracking-tight text-[var(--desk-text-primary)]`)

---

## 4. Layout Paradigms (From Reference Images)

### A. The 3-Column Control Tower (Primary Desktop Architecture)
Directly from `user_requested_layout.png` and `pinterest_935833997590542956.jpg`:
1. **Left Navigation / Sources Rail (240px–260px)**:
   - Logo mark (Pegasus winged horse icon) + Workspace selector.
   - Grouped navigation links with micro-icon badges, count pills, and indicator bar on active item.
   - Bottom profile capsule with role chip and connection pulse dot.
2. **Center Operations Stage (Flexible 1fr)**:
   - Sticky top command bar with global search, live status ticker (`● LIVE SYNC`), and action trigger buttons.
   - Filter / stage toggle bar (`All | Active | In Transit | Completed`).
   - Density-rich interactive feed: Bento grid of KPI metrics, live fleet map, or tabular entity list.
3. **Right Inspector / Drawer Panel (360px–440px)**:
   - Contextual slide-out or fixed inspector for the currently selected item (vehicle, driver, route, order).
   - Radial utilization gauge (e.g. `59% Truck Capacity`).
   - Detailed trip telemetry: Origin → Destination, stops checklist, photo report gallery, and action buttons (`Hot Swap`, `Reassign`, `Call Driver`).

### B. Floating AI / Command Bar (Bottom Studio Input)
Directly from `user_requested_layout.png`:
- Pill-shaped bottom container pinned to bottom center (`fixed bottom-6 left-1/2 -translate-x-1/2 w-full max-w-3xl`).
- Background: `var(--desk-surface-raised)` with `backdrop-blur-md` and `border border-[var(--desk-border-strong)]`.
- Inline quick action chips: Audio chip, Media attachment chip, Search chip, and Send button with glowing cyan/cobalt rim.

---

## 5. Signature Component Blueprints (Required for UI Tasks)

### 1. Metric Stat Tile (`MetricCard`)
- Container: `bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl p-5 hover:border-[var(--desk-border-strong)] transition-all`
- Top Row: Micro-label (`text-[11px] uppercase tracking-wider text-[var(--desk-text-tertiary)] font-bold`) + Icon Badge (`w-8 h-8 rounded-xl bg-[var(--desk-surface-subtle)] flex items-center justify-center text-[var(--desk-accent)]`).
- Main Metric: 32px Bold `font-mono tabular-nums text-[var(--desk-text-primary)]`.
- Bottom Row: Delta chip (`+8.4%` in emerald or `-2.1%` in crimson) + Context caption (`vs yesterday`).

### 2. Vehicle Tracking Card (`VehicleTrackingCard`)
Derived directly from `pinterest_935833997590542956.jpg`:
- Header: License plate badge in monospace (`01 772 AAA`), truck model (`Isuzu NPR 75`), status pill (`● ON ROUTE` in emerald).
- Body: Driver name & avatar thumbnail, active route code (`TASHKENT-NORTH-WAVE-2`), current stop counter (`Stop 4 of 9`), and ETA countdown badge (`ETA: 14:20`).
- Progress: Discrete LED block bar or continuous progress bar indicating completion percentage.
- Footer: Capacity metric (`59% Volume Filled`) + Quick actions (`Track`, `Inspect`, `Hot Swap`).

### 3. Radial Speedometer / Gauge (`GaugeChart`)
Derived from `pinterest_935833997590756543_*.jpg`:
- Semi-circular SVG arc meter with colored stroke (`stroke-[var(--desk-accent)]` or `stroke-[var(--desk-accent-lime)]`).
- Center readout: Bold percentage (`71.74%`), sub-label (`FLEET EFFICIENCY` or `CAPACITY CUBE`).
- Scale ticks indicating 0% to 100%.

### 4. Stage Progression Stepper (`StageStepper`)
Derived from `pinterest_935833997590756287.png`:
- Angled chevron or capsule step bar: `[ 1. New Lead ] → [ 2. Pick Wave ] → [ 3. Dispatched ] → [ 4. Delivered ]`.
- Active Step: Solid cobalt or orange fill with contrasting white text.
- Completed Step: Emerald icon checkmark + muted text.
- Future Step: Sunken surface (`bg-[var(--desk-surface-sunken)]`) with subtle border.

### 5. Discrete LED Block Progress (`LedProgressBar`)
Derived from `pinterest_935833997590754981.png`:
- Row of 10–20 discrete square/rectangular LED blocks.
- Active filled blocks: glowing solid emerald, cobalt, or lime.
- Inactive empty blocks: dark sunken border blocks (`bg-[var(--desk-surface-subtle)] border border-[var(--desk-border)]`).

### 6. Activity Timeline (`ActivityTimeline`)
Derived from `pinterest_935833997590756291.png`:
- Vertical dashed line (`border-l-2 border-dashed border-[var(--desk-border)]`).
- Circular event node badges with colored status icons.
- Timestamp in monospace (`10:42 AM`), bold event title, actor tag, and payload metadata pills.

### 7. Dense Data Table
Derived from `pinterest_935833997591669334.jpg`:
- Header: Clean 11px uppercase bold labels on sunken background (`bg-[var(--desk-surface-subtle)]/50`).
- Rows: 1px hairline border bottom (`border-b border-[var(--desk-border)]`), subtle hover highlight (`hover:bg-[var(--desk-surface-subtle)]/60`).
- Cell formatting: Monospace for numbers/dates, rounded-full status pill with dot, user avatar thumbnail with title & subtitle, toggle switch for status flags.

---

## 6. Forbidden Anti-Patterns (STRICTLY PROHIBITED)

- ❌ **NO Generic SaaS Grays**: Do NOT use `#6b7280`, `#9ca3af`, or default Tailwind `bg-gray-100` / `bg-gray-900`. Use canonical tokens (`--desk-canvas`, `--desk-surface`, etc.).
- ❌ **NO Heavy Blurry Shadows**: Do NOT use `shadow-2xl` with heavy dark spreads. Use crisp 1px borders (`border-[var(--desk-border)]`) and subtle rings.
- ❌ **NO Proportional Numbers for Metrics**: Do NOT render metrics, quantities, currency, or timers without `font-mono tabular-nums`.
- ❌ **NO Flat Unstyled Status Text**: Do NOT display raw text like "Active" or "Pending". Always wrap in a tactical badge with a live dot (`●`).
- ❌ **NO Low-Density Spacing**: Avoid huge empty whitespace blocks that waste screen real estate. Logistics, supply chain, and warehouse operators need dense, scannable information.
- ❌ **NO Raw Emojis as Icons**: Use proper SVG icons from Lucide / Heroicons.

---

## 7. Compliance Verification for Agents

Before concluding any UI task, every agent MUST verify:
1. Does the page match the palette and typography defined in this document?
2. Are all numbers, currencies, and timers formatted with `font-mono tabular-nums`?
3. Are all borders 1px hairline matching `--desk-border`?
4. Do active and hover states use crisp highlights instead of muddy drop shadows?
5. Does the application pass `npx tsc --noEmit` with zero errors?
