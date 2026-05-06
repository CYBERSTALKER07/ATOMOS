# V.O.I.D. Design System

The V.O.I.D. project uses a strict, hand-rolled Material Design 3 (M3) mapping. 

## The Zero-Dependency Rule
**Do NOT use `@material/web` Lit web components.** 
No `<md-button>`, `<md-filled-text-field>`, or external web component libraries.

## Tailwind v4 + Hand-Rolled M3 CSS
Layout and spacing rely on Tailwind CSS v4 (`@tailwindcss/postcss`).
Components and identity rely on our global M3 CSS variables defined in `globals.css`.

### Shared Desktop Foundation (Phase 1)
Desktop surfaces now share a foundation stylesheet at `pegasus/packages/ui-kit/styles/desktop-foundation.css`.

- Import this file at the top of each desktop app `globals.css`.
- Keep app-specific `globals.css` blocks additive; do not remove existing `md-*`, `button--*`, or `status-chip*` classes during rollout.
- The foundation file is compatibility-first and centralizes baseline tokens for body, card, button, input, chip, and status-chip semantics.
- Any new cross-desktop token should be introduced in the foundation file first, then consumed by app-specific layers.

### Tokens Available
- **Colors**: `--color-md-primary`, `--color-md-on-primary`, `--color-md-surface`, `--color-md-surface-container`, `--color-md-outline`, `--color-md-error`
- **Semantic colors**: `--color-md-success`, `--color-md-warning`, `--color-md-info`
- **Typography Scale**: `.md-typescale-display-large` through `.md-typescale-label-small`
- **Elevation**: `.md-elevation-0` through `.md-elevation-5` (Strict box-shadow usage)
- **Shape Tokens**: `.md-shape-none`, `.md-shape-xs`, `.md-shape-sm`, `.md-shape-md`, `.md-shape-lg`, `.md-shape-full`

### Approved Pattern Example
```tsx
// HTML/React - Proper Composition
<button className="md-btn md-btn-filled md-typescale-label-large px-6 py-2">
  Save Configuration
</button>

<div 
  className="md-card md-elevation-1 md-shape-md p-4"
  style={{ background: 'var(--color-md-surface-container)' }}
>
  <h3 className="md-typescale-title-medium text-[var(--color-md-on-surface)]">
    Node Metrics
  </h3>
</div>
```

## Consistency
If you build a feature that generates a new status flag (e.g., `PAYMENT_DISPUTED`), its corresponding visual token (`md-chip md-bg-error`) MUST be synchronized across the matching native iOS (`Color.red` / system standard equivalent) and native Android (Compose Material `MaterialTheme.colorScheme.error`).