# Design System Master File

> **LOGIC:** When building a specific page, first check `design-system/pages/[page-name].md`.
> If that file exists, its rules **override** this Master file.
> If not, strictly follow the rules below.

---

**Project:** pegasusX Warehouse Portal
**Generated:** 2026-06-17
**Category:** B2B warehouse logistics / node operations dashboard

---

## Global Rules

### Color Palette

| Role | Token | Usage |
|------|-------|-------|
| Canvas | `--desk-canvas` | App background |
| Surface | `--desk-surface` | Cards, panels |
| Accent | `--desk-accent` | Primary actions, active nav |
| Text primary | `--desk-text-primary` | Headings, body |
| Danger | `--desk-danger` | Errors, alerts |

**Style:** Flat SaaS, desk accent orange, 150–200ms transitions, 44px touch targets, visible focus rings.

### Typography

- **Heading / Body:** Plus Jakarta Sans (`--font-sans`)
- **Accent:** EB Garamond (`--font-garamond`) — marketing/auth only
- **Mood:** professional, operational, dense-but-readable

### Primitives

Import from `@pegasusx/ui-kit/portal` via `@/components/portal` wrappers:

- `PageChrome` — page title, icon slot, loading/error/empty
- `PortalField` / `PortalInput` / `PortalSelect` — forms
- `PortalSection` — grouped settings blocks
- `DataList` / `DataListRow` — thin list rows
- `HubCard` — cross-link hubs (treasury → payment-config)
- `portal-btn` / `portal-btn--primary` — actions (prefer over HeroUI Button)

### CSS stack

```css
@import "@pegasusx/ui-kit/styles/desktop-foundation.css";
@import "@pegasusx/ui-kit/styles/auth-layout.css";
@import "@pegasusx/ui-kit/styles/portal-ui.css";
@import "@pegasusx/ui-kit/styles/setup-onboarding.css";
```

### Shell

- `WarehouseShell` wraps authenticated routes; bare routes: `/auth/*`, `/setup/*`
- Root `#app-splash` dismissed via `data-hydrated` on `<html>`
- Sidebar active route: `desk-sidebar-link--active` + `data-active="true"`
- Theme toggle: `portal-btn portal-btn--ghost` (no HeroUI Button in shell)

### HeroUI

- v3.1.0 — `@heroui/styles` for semantic tokens only
- No `HeroUIProvider`
- Prefer `portal-btn` / `portal-input` for forms and actions

---

## QA Matrix

| Viewport | Light | Dark | Notes |
|----------|-------|------|-------|
| 375px | ✓ | ✓ | Mobile nav drawer, auth split stacks |
| 768px | ✓ | ✓ | Setup mobile progress bar |
| 1024px | ✓ | ✓ | Setup rail visible |
| 1440px | ✓ | ✓ | Full sidebar + bento dashboard |

**Tauri:** verify `[data-tauri]` titlebar padding in `globals.css`.

**Checks:** Toast on settings save, middleware covers `/settings`, `/preorders`, `/stock-commitments`, no hydration splash loop, setup bare route without shell chrome.

---

## Component Specs

### Buttons

Use `.portal-btn`, `.portal-btn--primary`, `.portal-btn--outline`, `.portal-btn--ghost` from `portal-ui.css`.

### Tables

Wrap in `.desk-table-wrap`; table uses `.desk-table`.

### Maps

`FleetLiveMapPanel` / `DispatchPreviewMap` inside `PortalSection`.
