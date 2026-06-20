# Dashboard page overrides

- Use `PageChrome` with `skeletonVariant="dashboard"` and optional `icon="overview"`.
- Content: `BentoGrid` with `theme="apple"` — high information density, no layout-shift hovers.
- KPI cells use `--desk-accent` for emphasis; charts use `--desk-border` / `--desk-text-secondary`.
- Billing gate banner links to `/setup/billing` when `isPaymentConfigured` is false.
