---
name: design-system
description: "Leviathan Retailer Desktop design system. Defines brand identity, design principles, token architecture, and UI pattern catalog. Use when building UI, reviewing designs, adding new pages, or making any visual decision in retailer-app-desktop."
user-invocable: true
---

# Leviathan Retailer Desktop — Design System

## Brand Identity

### Personality

- **Operational**: Dense, data-driven layouts built for logistics execution speed. / Not: decorative, consumer-friendly, marketing-oriented.
- **Precise**: Tabular numbers, exact metrics, structured hierarchy, zero ambiguity. / Not: approximate, conversational, soft.
- **Commanding**: High-contrast monochrome palette (black on white), bold headlines, decisive actions. / Not: playful, whimsical, pastel, gradient-heavy.
- **Responsive**: Live KPIs, animated transitions, real-time status indicators, streaming telemetry. / Not: static, batch-updated, stale.

### Voice & Tone

- **Default**: Direct, crisp, operational. Labels are short nouns or noun phrases. No filler words.
- **Error**: Urgent but not alarming. Red semantic color + clear action path. "3 invoices · Due in 7 days" not "Warning! You have unpaid invoices!"
- **Success**: Muted confirmation. Green chip, no celebration. "Completed" not "Great job!"
- **Data**: Tabular numerals, abbreviated units (MTD, YTD, UZS), trend arrows (↑↓), percentage deltas.

### Values & Mission

**Mission**: Give retailers operational clarity over their supply chain — orders, procurement, inventory, and spend — in a single dense control surface.

**Core Values**:
1. **Density over decoration** — Every pixel serves an operational purpose
2. **Clarity over cleverness** — Information hierarchy must be scannable in under 2 seconds
3. **Consistency over novelty** — Every page follows the same structural pattern
4. **Real-time over batch** — KPIs and status must reflect live state

### Relationship: Professional Tool

The product is a no-nonsense logistics workstation. The relationship is **expert tool** — efficient, precise, trustworthy. The UI should feel like a Bloomberg terminal for supply chain, not a consumer shopping app.

## Translation Matrix

| Trait | Color | Typography | Shape | Spacing | Motion | Icons |
|-------|-------|------------|-------|---------|--------|-------|
| Operational | Monochrome primary (black/white), semantic-only chromatic colors | Dense type scale, title-small for metadata rows | Medium radius (16px) for info cards | Moderate density (gap-2 to gap-6), 4dp grid | Fast transitions (200ms standard) | Lucide outline, 1.5 stroke weight |
| Precise | Tabular-nums on all numbers, `.tabular-nums` class | Label-small (11px) for KPI headers, body-small for metadata | Consistent radius across all bento-cards | Strict alignment grid, no decorative spacing | Cubic easing (0.2, 0, 0, 1), no bounce | Consistent 18px for KPI icons, 20px for list icons |
| Commanding | High contrast: oklch(0.15 0 0) accent on oklch(1 0 0) background | Headline-large (32px, 600 weight) for page titles, bold section heads | Border weight 1px, hover lift 1px translateY | Clear 24-32px section gaps between page zones | Emphasized easing for reveals (0.05, 0.7, 0.1, 1) | Single-weight icon set, no fill variants |
| Responsive | Success/warning/danger with muted chroma (0.16-0.2 oklch) | CountUp animated values, blinking-free updates | Bento grid: 4→2→1 columns responsive | Compact KPI card internals, generous inter-section | IntersectionObserver staggered reveals (60ms offset) | Status dots (8px) with semantic color |

## Token Architecture

### Single Source of Truth

Token definitions live in:
- **`apps/retailer-app-desktop/app/globals.css`** — all CSS custom properties, component classes, animations, and responsive rules (1350+ lines)

### Active Token System (X-Style Monochrome)

The canonical token system uses HeroUI v3 oklch overrides. These are the tokens all new code must reference:

| Role | Token | Light | Dark | Semantic Source |
|------|-------|-------|------|-----------------|
| Primary action | `--accent` | oklch(0.15 0 0) | oklch(0.93 0 0) | **Commanding** — near-black demands attention without competing with data |
| On-primary text | `--accent-foreground` | oklch(1 0 0) | oklch(0 0 0) | Inverted for maximum contrast on accent |
| Soft container | `--accent-soft` | oklch(0.93 0 0) | oklch(0.18 0 0) | Tonal step for secondary containers |
| Page background | `--background` | oklch(1 0 0) | oklch(0 0 0) | Pure white/black for maximum data clarity |
| Primary text | `--foreground` | oklch(0.1 0 0) | oklch(0.93 0 0) | Near-black/near-white for readability |
| Card surfaces | `--surface` | oklch(0.97 0 0) | oklch(0.1 0 0) | Subtle step from background for card elevation |
| Secondary text | `--muted` | oklch(0.55 0 0) | oklch(0.45 0 0) | **Precise** — mid-gray for metadata without competing with primary data |
| Borders | `--border` | oklch(0.88 0 0) | oklch(0.2 0 0) | Subtle structural lines |
| Input fields | `--field-background/foreground/border/placeholder` | white/dark variants | dark/light variants | Form control tokens |
| Success | `--success` | oklch(0.55 0.16 145) | oklch(0.68 0.18 145) | **Responsive** — muted green for completed/positive states |
| Warning | `--warning` | oklch(0.65 0.16 75) | oklch(0.72 0.16 75) | Muted amber for pending/caution states |
| Danger | `--danger` | oklch(0.55 0.2 25) | oklch(0.65 0.18 20) | Muted red for error/urgent states |

### Legacy Token System (M3 Compat)

The `--color-md-*` tokens exist in globals.css for backward compatibility with CSS component classes (`.md-btn-filled`, `.md-card`, etc.) and older components (CartDrawer, CheckoutModal). These map to equivalent monochrome values:

- `--color-md-primary` ≈ `--accent`
- `--color-md-surface` ≈ `--background`
- `--color-md-surface-container` ≈ `--surface`
- `--color-md-on-surface` ≈ `--foreground`
- `--color-md-on-surface-variant` ≈ `--muted`
- `--color-md-outline-variant` ≈ `--border`
- `--color-md-error` ≈ `--danger`
- `--color-md-warning` ≈ `--warning`
- `--color-md-success` ≈ `--success`

**Rule**: New page/component code must use the active tokens (`--accent`, `--muted`, `--surface`, etc.) exclusively. Legacy `--color-md-*` references should be migrated when a component is touched.

### Key Translation Rules

- `--accent` (black/white): **Commanding** — The single most decisive color in the palette. Used sparingly for primary actions, active navigation, and hero stat cards. Black in light mode demands immediate attention without introducing brand color competition with data.
- `--muted` (mid-gray): **Precise** — Secondary information must be clearly subordinate to primary data. 55% lightness in light mode creates a distinct but readable tier.
- `--surface` (off-white): **Operational** — Cards need barely-visible separation from the page background. 97% lightness creates depth without decorative effort.
- `--border` (light gray): **Consistent** — Structural boundaries are hairline and neutral. They organize, never decorate.
- `--success/warning/danger` (muted chromatic): **Responsive** — Status colors use low chroma (0.16-0.2) so they signal state without screaming. Operational tools need calm urgency, not alarm.
- Shape `16px` radius (bento-card): **Operational** — Rounded enough to feel modern, sharp enough to feel serious. Matches the "friendly tool" balance.
- Typography `tabular-nums`: **Precise** — All numeric displays must use tabular figures so columns of numbers align. This is non-negotiable for a data tool.
- Motion `200ms` standard: **Operational** — Transitions complete fast enough to feel instant but slow enough to be perceived. Longer animations (500ms) are reserved for IntersectionObserver reveals only.

### Typography Scale

Implementation: `.md-typescale-{role}-{size}` classes in globals.css.

| Role | Size | Font Size | Weight | Usage |
|------|------|-----------|--------|-------|
| Display | Large | 57px | 400 | Unused in current app (reserved for splash/hero) |
| Headline | Large | 32px | 600 | Page titles ("Orders & Tracking", "Supplier Catalog") |
| Headline | Small | 24px | 500 | Modal titles, hero stat numbers |
| Title | Large | 22px | 500 | Section headers ("Top Sellers", "Ledger Overview") |
| Title | Medium | 16px | 600 | Card headers, form section titles |
| Title | Small | 14px | 600 | List item names, product names |
| Body | Large | 16px | 400 | Primary body text |
| Body | Medium | 14px | 400 | Default body, descriptions, subtitles |
| Body | Small | 12px | 400 | Metadata, tertiary text, SLA details |
| Label | Large | 14px | 500 | Buttons, tab labels, breadcrumbs |
| Label | Medium | 12px | 500 | Small action labels |
| Label | Small | 11px | 500 | KPI headers (uppercase tracking), form hints |

**Principle**: Use `headline-large` for page title, `title-large` for section heads, `title-small` for item names, `label-small` (uppercase + tracking-widest) for KPI labels, `body-small` + `text-muted` for metadata.

### Spacing Scale

Base unit: 4dp. Tokens in globals.css as `--md-spacing-*`:

| Token | Value | Usage |
|-------|-------|-------|
| xs | 4px | Icon-to-text gap, tight inline spacing |
| sm | 8px | Intra-component gaps, chip padding |
| md | 12px | Card internal padding (compact) |
| lg | 16px | Standard card padding, list item gap |
| xl | 24px | Section gap, page padding on mobile |
| 2xl | 32px | Page padding on desktop (p-8 = 32px) |
| 3xl | 48px | Large section separation |

### Elevation Scale

| Level | Shadow | Usage |
|-------|--------|-------|
| 0 | none | Flat surfaces |
| 1 | 0 1px 3px rgba(0,0,0,0.08) | Bento cards at rest, default cards |
| 2 | 0 2px 8px rgba(0,0,0,0.08) | Search bars, elevated cards |
| 3 | 0 4px 16px rgba(0,0,0,0.1) | FABs, floating actions |
| 4 | 0 8px 24px rgba(0,0,0,0.12) | Navigation drawers |
| 5 | 0 12px 32px rgba(0,0,0,0.14) | Modals, dialogs |

**Dark mode**: Shadows are removed. Elevation is expressed through border + surface-container tint.

### Shape Scale

| Token | Value | Usage |
|-------|-------|-------|
| none | 0px | Full-bleed sections |
| xs | 4px | Checkboxes, small indicators |
| sm | 8px | Chips, small buttons |
| md | 16px | Bento-cards, standard containers |
| lg | 24px | Large cards, drawers |
| xl | 28px | Modals, dialogs |
| full | 9999px | Circular avatars, pills, toggles |

### Motion Tokens

| Token | Curve | Usage |
|-------|-------|-------|
| `--md-easing-emphasized` | cubic-bezier(0.2, 0, 0, 1) | Standard component transitions |
| `--md-easing-emphasized-decelerate` | cubic-bezier(0.05, 0.7, 0.1, 1) | Enter animations, IntersectionObserver reveals |
| `--md-easing-emphasized-accelerate` | cubic-bezier(0.3, 0, 0.8, 0.15) | Exit animations |
| `--md-easing-spring` | cubic-bezier(0.175, 0.885, 0.32, 1.275) | Overshoot effects (reserved, rare) |

**Duration principle**: 200ms for state changes (hover, focus, tab switch). 400-500ms for layout reveals (bento-enter). Never exceed 1000ms.

## Patterns

### Pattern Catalog

Patterns are named by their **role in the retailer logistics workflow**, not by generic UI library names.

#### 1. `kpi-strip`
**Role**: Show at-a-glance operational metrics at the top of every page.
**Structure**: `BentoGrid` containing 4 `BentoCard` components, each wrapping a `metric-cell`.
**Responsive**: 4 columns → 2 columns (≤1024px) → 1 column (≤640px).
**Stagger**: 60ms delay between cards for IntersectionObserver reveal.
**Relationship**: **Compositional** — contains `metric-cell` pattern.

#### 2. `metric-cell`
**Role**: Display a single KPI with label, animated value, trend sparkline, and context subtitle.
**Structure**: `.md-kpi-card` div → label (`.md-kpi-label`), value (`CountUp` with `.md-kpi-value`), sparkline (`MiniSparkline`), subtitle (`.md-kpi-sub`).
**States**: Loading (skeleton shimmer), populated (animated), stale (no animation).
**Token mapping**: `--muted` for icon + label, `--success/warning/danger` for trend indicators.

#### 3. `filter-tabs`
**Role**: Let user segment content by status or category.
**Structure**: Horizontal flex row of `<button>` elements with conditional `bg-accent text-accent-foreground` for active state.
**States**: Default (transparent + muted text), active (accent bg), hover (surface bg).
**Token mapping**: Active = `--accent` + `--accent-foreground`. Inactive = transparent + `--muted`.

#### 4. `split-panel`
**Role**: Browse a list of items while inspecting one in detail.
**Structure**: Flex row. Left: scrollable item list (flex-1). Right: detail/summary sidebar (360-480px, hidden below lg).
**Responsive**: Sidebar hidden on mobile. Full-width list only.
**Relationship**: **Compositional** — left pane contains `item-row` pattern, right pane contains `sidebar-detail`.

#### 5. `item-row`
**Role**: Represent a selectable entity (order, vendor, product, SKU) in a list.
**Structure**: `.bento-card` button/div → icon container (48px, rounded-xl, surface bg) + title + status `Chip` + metadata line + chevron.
**States**: Default, hover (bento-card lift), selected (2px accent ring), disabled.
**Token mapping**: Icon bg = `--surface`, title = `--foreground`, metadata = `--muted`, status = HeroUI `Chip` with semantic color.

#### 6. `sidebar-detail`
**Role**: Show contextual summary, stats, or actions alongside a list.
**Structure**: Vertical flex column of `info-card` and `hero-stat` patterns.
**Responsive**: Hidden below lg breakpoint. Width 360-400px.
**Relationship**: **Compositional** — contains `hero-stat` and `info-card` patterns.

#### 7. `hero-stat`
**Role**: Highlight the single most important metric in a sidebar.
**Structure**: `.bento-card` with `--accent` background + `--accent-foreground` text. Contains headline number + supporting metadata.
**Usage**: Outstanding Payables, Performance Score. One per sidebar maximum.
**Token mapping**: bg = `--accent`, text = `--accent-foreground`.

#### 8. `search-trigger`
**Role**: Let user search or filter content inline.
**Structure**: `.md-search-bar` container with `Search` icon + text input.
**States**: Default, focused (accent border), empty placeholder.
**Token mapping**: bg = `--field-background`, border = `--field-border`, placeholder = `--field-placeholder`.

#### 9. `nav-shell`
**Role**: Persistent navigation and context across all pages.
**Structure**: `RetailerShell` — collapsible sidebar (240px expanded / 64px rail) + sticky header (56px) with breadcrumbs + connection status.
**Responsive**: Desktop = sidebar. Mobile = hamburger → overlay drawer (280px).
**Token mapping**: Sidebar bg = `--background`, active item = `--accent` bg, dividers = `--border`.

#### 10. `info-card`
**Role**: Group related information in a bounded container.
**Structure**: `.bento-card` class (border + shadow + padding + hover lift).
**States**: Default (elevation-1 shadow), hover (translateY -1px + accent border).
**Token mapping**: bg = `--background`, border = `--border`, hover border = `--accent`.

#### 11. `progress-meter`
**Role**: Show proportional completion (budget usage, satisfaction score).
**Structure**: Full-width bar (h-2, rounded-full, surface bg) + filled bar (accent or semantic color).
**Token mapping**: Track = `--surface`, fill = `--accent` or `--warning`.

### Inter-Pattern Relationships

| Relationship | Patterns | Rule |
|-------------|----------|------|
| Compositional | `kpi-strip` contains `metric-cell` | Always 4 cells per strip |
| Compositional | `split-panel` contains `item-row` + `sidebar-detail` | Sidebar hidden on mobile |
| Compositional | `sidebar-detail` contains `hero-stat` + `info-card` | Max 1 hero-stat per sidebar |
| Alternative | `item-row` vs table row (`.md-table`) | Use item-row for ≤20 items, table for >20 |
| Exclusive | `hero-stat` and `metric-cell` in same container | hero-stat is sidebar-only, metric-cell is kpi-strip-only |

### Page Template

Every dashboard page follows this exact structure:

```
1. Header zone
   └─ h1.md-typescale-headline-large + p.md-typescale-body-medium (muted subtitle)
   └─ Action button (right-aligned)

2. KPI zone
   └─ kpi-strip (BentoGrid with 4 metric-cells)

3. Filter zone (optional)
   └─ filter-tabs or search-trigger

4. Content zone
   └─ split-panel (item list + sidebar-detail)
   OR
   └─ grid/table of items
```

### Principles for New Patterns

When adding new patterns:
1. **Role-based design** — Name by role ("delivery-timeline"), not appearance ("horizontal-stepper")
2. **Explicit state system** — Define default, hover, focused, pressed, disabled for every interactive element
3. **Container thinking** — Use `bento-card` as the base container. Background via `--background`, border via `--border`, padding via Tailwind (p-4 to p-6)
4. **Responsive** — Desktop sidebar content hides at `lg` breakpoint. Grids collapse at `1024px` and `640px`
5. **Accessibility** — 48×48dp touch targets, WCAG AA contrast (guaranteed by monochrome palette), keyboard navigation, semantic HTML

## Accessibility Audit Findings

### Compliant
- **Color contrast**: Monochrome palette guarantees WCAG AAA for text (oklch 0.1 on 1.0 = ~21:1 ratio)
- **Semantic status colors**: Success (oklch 0.55), warning (0.65), danger (0.55) all pass AA on white backgrounds
- **Typography**: Base body-medium (14px) exceeds minimum readable size. Label-small (11px) is used only for uppercase all-caps labels which have larger effective size
- **Reduced motion**: `@media (prefers-reduced-motion: reduce)` rule disables all animations to 0.01ms
- **Focus ring**: `.md-focus-ring` provides 2px outline offset for keyboard navigation

### Needs Attention
- **Touch targets**: Some custom toggle switches in Settings have visual 48×28px area but lack 48×48dp hit area (padding buffer needed)
- **Screen reader labels**: Bento-card buttons in Procurement vendor list use `<button>` without `aria-label`
- **CartDrawer/CheckoutModal**: Still use `--color-md-*` tokens — need migration to active token system for consistency

## Component Inventory

Implementation SSoT: `apps/retailer-app-desktop/components/`

| Component | File | Role | Props |
|-----------|------|------|-------|
| BentoGrid | `BentoGrid.tsx` | kpi-strip container | `children`, `className` |
| BentoCard | `BentoGrid.tsx` | metric-cell/info-card wrapper | `children`, `span`, `rowSpan`, `className`, `delay` |
| CountUp | `CountUp.tsx` | Animated number display | `end`, `duration`, `prefix`, `suffix`, `decimals`, `className` |
| MiniSparkline | `MiniSparkline.tsx` | Inline SVG trend chart | `data[]`, `width`, `height`, `color`, `className` |
| CartDrawer | `CartDrawer.tsx` | Cart overlay drawer | `isOpen`, `onClose`, `onCheckout` |
| CheckoutModal | `CheckoutModal.tsx` | Payment modal | `isOpen`, `onClose`, `total` |
| RetailerShell | `RetailerShell.tsx` | nav-shell layout wrapper | `children` |

## Dependencies

| Package | Version | Role |
|---------|---------|------|
| `@heroui/react` | ^3.0.2 | Button, Chip, Card (legacy), Skeleton primitives |
| `lucide-react` | ^1.7.0 | All iconography (outline, 1.5 stroke) |
| Tailwind CSS v4 | ^4 | Layout utility classes |
| Next.js 15 | 15.5.12 | App Router framework |
| React 19 | 19.1.0 | UI library |
| Tauri 2 | ^2.10.1 | Desktop shell (optional) |

## Governance Rules

1. **New page** → Must follow the Page Template (header → kpi-strip → filter → split-panel)
2. **New token** → Must use the active system (`--accent`, `--muted`, etc.). Never add new `--color-md-*` tokens
3. **New pattern** → Must be used in ≥3 pages before extracting to a named pattern
4. **Token change** → Propagates to all pages. Test light + dark mode after any globals.css edit
5. **Icon** → Must be from `lucide-react` with `size={18}` (KPI) or `size={20}` (list) and `strokeWidth={1.5}`
6. **Number display** → Must use `tabular-nums` class and `CountUp` component for KPI values
7. **Status color** → Only through HeroUI `Chip` with `color="success|warning|danger|default"` or inline `style={{ color: "var(--success)" }}`
8. **Animation** → IntersectionObserver reveals use 60ms stagger per item. State transitions use 200ms. No animation >1000ms
9. **HeroUI Button** → Must use `onPress` (not `onClick`). Must include `className="md-btn md-btn-filled md-typescale-label-large"`
