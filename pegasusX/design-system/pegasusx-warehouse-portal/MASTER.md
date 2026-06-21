# Design System Master File

> **LOGIC:** When building a specific page, first check `design-system/pages/[page-name].md`.
> If that file exists, its rules **override** this Master file.
> If not, strictly follow the rules below.

---

**Project:** pegasusX Warehouse Portal
**Generated:** 2026-06-17
**Category:** B2B warehouse logistics / node operations dashboard

---

## Visual identity

**Subject:** Single-tenant warehouse node control plane — shift-long scanning of orders, dispatch, stock, and fleet.

**Signature:** Concrete canvas grid on the main work area + 3px left **bay stripe** on operational panels (color encodes domain: ops, inventory, fleet, finance).

**Aesthetic risk (justified):** Floor-grid background reads as warehouse concrete without decorative glass or generic SaaS gradients.

### Color palette

| Role | Token | Light | Usage |
|------|-------|-------|-------|
| Canvas | `--wh-canvas` | `#E8EDF2` | Main work area (grid overlay) |
| Surface | `--wh-surface` | `#FFFFFF` | Cards, sidebar, topbar |
| Ink | `--wh-ink` | `#1A2332` | Headings, primary text |
| Accent | `--wh-accent` | `#EA6C00` | Primary actions, active nav, live emphasis |
| Danger | `--wh-danger` | `#DC2626` | Reject, alerts |

Zone bay stripes: `--wh-zone-ops` (accent), `--wh-zone-inventory` (slate blue), `--wh-zone-fleet` (steel), `--wh-zone-finance` (green-grey).

### Typography

| Role | Face | Usage |
|------|------|-------|
| UI / headings | Plus Jakarta Sans (`--font-sans`) | Nav, titles, body |
| Ops data | IBM Plex Mono (`--font-plex-mono`) | Order IDs, amounts, page counts |
| Marketing / auth only | EB Garamond (`--font-garamond`) | Auth brand panel |

### Layout

- `WarehouseShell` — sidebar + sticky topbar + scrollable main canvas
- `desk-page` — max-width 1600px, padding 28px (32px at ≥1440px)
- `wh-kpi-grid` — dashboard metrics
- `wh-ops-grid` — order / ops card grids
- `wh-tab-bar` — segmented tabs (Active | Pre-orders)

### Primitives

Import from `@pegasusx/ui-kit/portal` via `@/components/portal` wrappers:

- `PageChrome` — page title, icon slot, loading/error/empty
- `PageSection` — bay-striped section with `bay` prop
- `KpiStatCard` / `KpiStatGrid` — dashboard metrics with mono values
- `ListToolbar` — pagination + export
- `portal-btn` / `portal-btn--primary` — actions

### CSS stack

```css
@import "@pegasusx/ui-kit/styles/desktop-foundation.css";
@import "@pegasusx/ui-kit/styles/auth-layout.css";
@import "@pegasusx/ui-kit/styles/portal-ui.css";
@import "@pegasusx/ui-kit/styles/setup-onboarding.css";
@import "../styles/warehouse-desktop.css";
```

Root: `<html data-app="warehouse">` scopes all warehouse tokens.

### Shell

- `WarehouseShell` wraps authenticated routes; bare routes: `/auth/*`, `/setup/*`
- Root `#app-splash` dismissed via `data-hydrated` on `<html>`
- Sidebar active route: `desk-sidebar-link--active` + `data-active="true"`
- Theme toggle: `portal-btn portal-btn--ghost`

### HeroUI

- v3.1.0 — `@heroui/styles` for semantic tokens only
- No `HeroUIProvider`
- Prefer `portal-btn` / `portal-input` for forms and actions

---

## QA Matrix

| Viewport | Light | Dark | Notes |
|----------|-------|------|-------|
| 375px | ✓ | ✓ | Mobile nav drawer |
| 768px | ✓ | ✓ | Setup mobile progress bar |
| 1024px | ✓ | ✓ | Setup rail visible |
| 1440px | ✓ | ✓ | Full sidebar + 4-col KPI grid |

**Tauri:** verify `[data-tauri]` titlebar padding in `globals.css`.

**Checks:** Toast on settings save, visible focus rings, reduced-motion respected, bay stripes on sections/cards.

---

## Component Specs

### Buttons

Use `.portal-btn`, `.portal-btn--primary`, `.portal-btn--outline`, `.portal-btn--ghost`.

### Tables

Wrap in `.desk-table-wrap`; table uses `.desk-table`.

### Maps

`FleetLiveMapPanel` inside `PageSection` with `bay="fleet"`.
