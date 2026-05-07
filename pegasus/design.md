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

- Use framer-motion for page transitions and staggered list entry.
- Keep motion meaningful and state-communicative.
- Respect reduced motion preferences.

### Microinteraction Rules

- hover-lift for card and row affordance.
- active-press for button and command feedback.
- skeleton-shimmer for loading placeholders.

## Icon Rules

- Use lucide-react icons only.
- No emoji icons anywhere.
- Empty states may use real illustration assets and optional icon overlays.

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
