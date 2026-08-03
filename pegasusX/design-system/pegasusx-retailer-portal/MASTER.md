# Design System Master File

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



> **LOGIC:** When building a specific page, first check `design-system/pegasusx-retailer-portal/pages/[page-name].md`.
> If that file exists, its rules **override** this Master file.
> If not, strictly follow the rules below.

---

**Project:** pegasusX Retailer Portal
**Generated:** 2026-06-21
**Category:** B2B retailer commerce / ordering / procurement dashboard

---

## Global Rules

### Color Palette

| Role | Token | Usage |
|------|-------|-------|
| Canvas | `--desk-canvas` | App background |
| Surface | `--desk-surface` | Cards, panels |
| Accent | `--desk-accent` | Primary actions, active nav, cart CTA |
| Text primary | `--desk-text-primary` | Headings, body |
| Danger | `--desk-danger` | Errors, alerts |

**Style:** Flat SaaS, desk accent, 150–200ms transitions, 44px touch targets, visible focus rings.

### Typography

- **Heading / Body:** Plus Jakarta Sans (`--font-sans`)
- **Accent:** EB Garamond (`--font-garamond`) — marketing/auth only
- **Mood:** professional, commerce-focused, catalog-friendly

### Primitives

Import from `@pegasusx/ui-kit/portal` via `@/components/portal` wrappers:

- `PageChrome` — page title, icon slot, loading/error/empty
- `PortalField` / `PortalInput` / `PortalSelect` — forms
- `PortalSection` — grouped blocks (preserve `PageSection` for inner bento)
- `portal-btn` / `portal-btn--primary` — actions (prefer over HeroUI Button)

### CSS stack

```css
@import "@pegasusx/ui-kit/styles/desktop-foundation.css";
@import "@pegasusx/ui-kit/styles/auth-layout.css";
@import "@pegasusx/ui-kit/styles/portal-ui.css";
@import "@pegasusx/ui-kit/styles/setup-onboarding.css";
```

### Shell

- `RetailerShell` wraps `(dashboard)` routes only; bare routes: `/auth/*`, `/setup/*`
- Root `#app-splash` dismissed via `data-hydrated` on `<html>`
- Sidebar active route: `desk-sidebar-link--active` + `data-active="true"`
- Theme toggle: `portal-btn portal-btn--ghost`

### HeroUI

- v3.1.0 — `@heroui/styles` for semantic tokens only
- No `HeroUIProvider`
- Prefer `portal-btn` / `portal-input` for forms and actions

### Retailer-specific (preserve)

- `CartProvider`, `CheckoutModal`, `CartDrawer`, `PaymentModal`, `ShopClosedModal`
- `PendingCheckoutFlusher`, `WebSocketProvider`, `ClientPolicyBanner`
- `LocaleBootstrap`, Tauri bridge

---

## QA Matrix

| Viewport | Light | Dark | Notes |
|----------|-------|------|-------|
| 375px | ✓ | ✓ | Mobile nav drawer, auth split stacks |
| 768px | ✓ | ✓ | Setup mobile progress bar |
| 1024px | ✓ | ✓ | Setup rail visible |
| 1440px | ✓ | ✓ | Full sidebar + catalog grid |

**Tauri:** verify `[data-tauri]` titlebar padding in `globals.css`.

**Checks:** Cart/checkout toasts, middleware `is_configured` → `/setup`, auth splash post-mount only, setup bare route without `RetailerShell`, `LocaleBootstrap` before providers.

```bash
cd pegasusX && pnpm --filter @pegasusx/retailer-app-desktop typecheck
```
