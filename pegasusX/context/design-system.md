# pegasusX Design System

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.


## Zero-Dependency Rule
Do NOT use `@material/web` Lit web components. No `<md-button>`, no `<md-filled-text-field>`. Hand-rolled M3 only.

## Tailwind v4 + Hand-Rolled M3 CSS
Layout via Tailwind. Components and identity via M3 CSS variables in each web app's `globals.css`. Shared foundation lives in `packages/ui-kit/styles/desktop-foundation.css`.

## Tokens
- **Colors**: `--color-md-primary`, `--color-md-on-primary`, `--color-md-surface`, `--color-md-surface-container`, `--color-md-outline`, `--color-md-error`.
- **Semantic**: `--color-md-success`, `--color-md-warning`, `--color-md-info`.
- **Typography**: `.md-typescale-display-large` → `.md-typescale-label-small`.
- **Elevation**: `.md-elevation-0` → `.md-elevation-5` (box-shadow).
- **Shape**: `.md-shape-none`, `.md-shape-xs`, `.md-shape-sm`, `.md-shape-md`, `.md-shape-lg`, `.md-shape-full`.

## Approved Pattern
```tsx
<button className="md-btn md-btn-filled md-typescale-label-large px-6 py-2">
  Save Configuration
</button>

<div
  className="md-card md-elevation-1 md-shape-md p-4"
  style={{ background: 'var(--color-md-surface-container)' }}
>
  <h3 className="md-typescale-title-medium" style={{ color: 'var(--color-md-on-surface)' }}>
    Node Metrics
  </h3>
</div>
```

## Cross-Platform Token Sync
Any new status (e.g., `PAYMENT_DISPUTED`) must surface in:
- Web (`md-chip md-bg-error`)
- iOS (`Color.red` or system error)
- Android (`MaterialTheme.colorScheme.error`)

## Bento Grid (supplier-portal dashboard)
Same protocol as Pegasus: `<BentoCard>` with sizes `stat | list | control | anchor | wide | full`; high-contrast borders, no shadows, brutalist radius default with Apple theme alternative.
