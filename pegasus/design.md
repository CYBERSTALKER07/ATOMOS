# Desktop Design Contract

This document is the canonical visual contract for all Pegasus desktop apps.

## Scope

Applies to:

- pegasus/apps/admin-portal
- pegasus/apps/factory-portal
- pegasus/apps/warehouse-portal
- pegasus/apps/retailer-app-desktop

## Style Direction

Visual direction is grounded in local references under pegasus/assets, especially:

- 1d42d3c569b68b3816f840c3b9066724.webp
- 62665a5798605f336decc4a26bbc4911.webp
- 8601eed85c98c6c01f57cc314eb77094.webp
- original-6f769336f644695f12fff0002b352b2e.webp
- CD123FFA-EBD9-4790-A6F8-5788ACF80A54.jpeg
- DB1E9E1A-315A-47B0-A2F7-9B9776725EE3.jpeg

Generated placeholder illustrations live in:

- pegasus/assets/illustrations/no-data.svg
- pegasus/assets/illustrations/no-results.svg
- pegasus/assets/illustrations/offline.svg
- pegasus/assets/illustrations/restricted.svg
- pegasus/assets/illustrations/error.svg

## Token Source Of Truth

Primary source:

- pegasus/packages/ui-kit/styles/desktop-foundation.css

All desktop apps import this file from globals.css before app-specific overrides.

## Canonical Tokens

### Color Tokens

- --desk-canvas: #F3F4F6
- --desk-surface: #FFFFFF
- --desk-surface-subtle: #F8FAFC
- --desk-border: #E5E7EB
- --desk-border-strong: #CBD5E1
- --desk-text-primary: #111827
- --desk-text-secondary: #6B7280
- --desk-text-tertiary: #9CA3AF
- --desk-accent: #FF7A1A
- --desk-accent-soft: #FFF3EA
- --desk-success: #16A34A
- --desk-warning: #D97706
- --desk-danger: #DC2626
- --desk-info: #2563EB
- --desk-focus-ring: #111827
- --glass-bg: rgba(255, 255, 255, 0.05)
- --glass-premium-bg: linear-gradient(135deg, rgba(255, 255, 255, 0.1) 0%, rgba(255, 255, 255, 0.05) 100%)
- --glass-border: rgba(255, 255, 255, 0.12)
- --glass-blur: 24px
- --glass-shadow: 0 12px 40px -12px rgba(0, 0, 0, 0.25)

### Typography Tokens

- --type-display-xl
- --type-display-lg
- --type-heading-lg
- --type-heading-md
- --type-title
- --type-body-lg
- --type-body-md
- --type-caption-sm
- --type-metric

### Spacing Tokens

- --space-0
- --space-1
- --space-2
- --space-3
- --space-4
- --space-5
- --space-6
- --space-8
- --space-10

### Radius Tokens

- --radius-sm
- --radius-md
- --radius-lg
- --radius-xl
- --radius-pill

### Motion Tokens

- --duration-fast
- --duration-base
- --duration-slow
- --ease-standard
- --ease-enter
- --ease-exit

## Interaction Contract

### Required States For Every Live Screen

- Loading
- Empty
- Offline or disconnected
- Stale data where applicable
- Permission-restricted
- Error

### Animation Rules

- Use `motion` (framer-motion v12+) for all non-trivial transitions.
- **Page Transitions**: Use `AnimatePresence` with a consistent `opacity` and `y` offset (e.g., `y: 10` to `y: 0`) for entry.
- **List Staggering**: Apply `staggerChildren` to all data lists and bento grids for a "wave" entry effect.
- **Sidebar**: Use spring physics (`type: "spring", stiffness: 300, damping: 30`) for collapse/expand.
- **Hover**: Cards must use `whileHover={{ y: -4, boxShadow: "..." }}` for premium tactile feel.
- Respect `prefers-reduced-motion` using the `useReducedMotion` hook.

### Microinteraction Rules

- **.hover-lift**: Apply to all interactive cards and list rows. Smoothly translate -2px or -4px on hover.
- **.active-press**: Mandatory for all buttons and nav items. Scale down to 0.98 or 0.96 on press.
- **.skeleton-shimmer**: Use a subtle, slow-moving gradient for loading states. Avoid jarring flashes.
- **Glassmorphism**: Use `backdrop-filter: blur(16px)` via the `.glass-premium` class for primary containers, sticky headers, and modal backdrops.
- **Input Focus**: Use a high-contrast focus ring (2px) with a subtle outer glow using the accent color.
- **Atmospheric Orbs**: Use `.orb-container` with `.gradient-orb-*` for adding depth to page backgrounds.
- **Floating Effect**: Use `.animate-float` for illustrations and highlight elements to create a sense of life.
- **Glow Pulse**: Use `.animate-glow` for critical alerts or primary action buttons.

- **Bento Grid**: Use `.bento-grid` for dashboard layouts. Cards should use `.bento-card` with appropriate span classes (`.bento-anchor`, `.bento-wide`, etc.) for visual interest.

### Empty State Rules

- **Illustrations**: Never use simple icons for primary empty states. Use the generated 3D assets.
- **Contextual Actions**: Every empty state should provide a clear "Next Step" button.
- **Animations**: Empty state illustrations should have a gentle "float" or "pulse" animation.

### Cinematic Excellence

- **Depth & Dimension**: Use `glass-premium` for containers that need to feel "above" the canvas.
- **Atmospheric Lighting**: Place `orb-container` with `gradient-orb-*` classes behind glass surfaces to create depth.
- **Micro-interactions**: Use `hover-lift` and `active-press` for all interactive elements.
- **Smooth Transitions**: Leverage `PageTransition` with scale-in effects for a premium feel.
- **Typography as Art**: Use `text-cinematic` and `glow-text` for hero headlines or critical metrics.

## Layout Contract

- Desktop shell keeps stable navigation rhythm and dense, readable operational layout.
- Use consistent command rows for search, filtering, and sorting.
- Detail inspection stays in right-side panel or drawer patterns.

## Implementation Notes

- Existing compatibility aliases (--desktop-*) remain supported during migration.
- App-specific globals.css may extend but should not redefine canonical token intent.
- New components must consume semantic tokens, not raw hex values.

## Quality Gate

Before completion of desktop UX changes:

- Lint and type checks pass for touched apps.
- Visual regressions are reviewed.
- Route-state matrix confirms all required states are implemented.
- Design and token docs remain synchronized with code.
