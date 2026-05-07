---
name: design-system-design-md-generator-community
description: Creates implementation-ready design-system guidance derived from local visual references and applies it consistently across all Pegasus desktop apps.
argument-hint: Describe the desktop screen, role, and desired UI outcome.
user-invocable: true
---

<!-- TYPEUI_SH_MANAGED_START -->

# Desktop Design Skill (Community)

## Mission

Operationalize one desktop-first design language across all Pegasus desktop apps using the visual direction from `pegasus/assets`.

## Scope

This skill is for desktop UI work in:

- `pegasus/apps/admin-portal`
- `pegasus/apps/factory-portal`
- `pegasus/apps/warehouse-portal`
- `pegasus/apps/retailer-app-desktop`

## Visual Direction

Use a calm, high-density enterprise style:

- light neutral canvas and surfaces
- persistent left navigation rail
- compact KPI strips with trend context
- dense data tables with explicit row actions
- right-side inspector/detail panels
- selective warm accent for primary actions and selected states

## Source Inputs

Before generating guidance, read visual references from `pegasus/assets` and extract:

1. shell structure (left rail, top utility bar, content canvas)
2. hierarchy (KPI strip -> commands -> table/grid -> detail panel)
3. interaction patterns (search, filters, row selection, bulk action bar)
4. density profile (compact rows, thin dividers, low visual noise)

## Non-Negotiable Rules

### Must

- Guidance must apply to all desktop apps in scope, not one app.
- Guidance must define semantic tokens for color, typography, spacing, radius, and motion.
- Guidance must define states for default, hover, focus-visible, active, disabled, loading, empty, offline, stale, and restricted.
- Guidance must include keyboard, pointer, and touch behavior.
- Guidance must include long-content and overflow handling.
- Guidance must include testable accessibility criteria (WCAG 2.2 AA target).
- Guidance must preserve repository constraints: no decorative gradients and no emoji icons.

### Should

- Guidance should align with shared desktop foundations in `@pegasus/ui-kit/styles/desktop-foundation.css`.
- Guidance should keep component naming semantic and reusable.
- Guidance should include rollout notes per desktop app.

## Token Contract Template

Use this token structure in generated outputs:

- Color: canvas, surface, border, text-primary, text-secondary, accent, success, warning, danger, info, focus-ring
- Typography: display, heading, title, body, caption, metric
- Spacing: 4px base scale
- Radius: small, medium, large, pill
- Motion: fast/base/slow + standard easing

## Required Output Structure

1. Context and goals
2. Design tokens and foundations
3. Component-level rules (anatomy, variants, states, responsive behavior)
4. Accessibility requirements and testable acceptance criteria
5. Content and tone standards with examples
6. Anti-patterns and prohibited implementations
7. QA checklist

## Component Rule Expectations

Generated guidance must cover at least:

- app shell and navigation rail
- KPI strip cards
- search/filter command row
- dense data table
- right inspector or detail drawer
- status chips
- bulk action bar

Each component section must include:

- anatomy
- variants
- interaction states
- keyboard behavior
- overflow and long-content behavior

## Authoring Workflow

1. Restate design intent in one sentence.
2. Derive concrete tokens from the visual references.
3. Define shared desktop shell behavior.
4. Define component anatomy, variants, and states.
5. Add accessibility acceptance criteria.
6. Add anti-patterns and migration notes.
7. End with a QA checklist.

## Quality Gates

- Every non-negotiable rule uses `must`.
- Every recommendation uses `should`.
- Every accessibility requirement is testable.
- Guidance is concrete and implementation-ready.
- Output is reusable in other repositories with token substitution.

## Desktop Rollout Checklist

- `admin-portal`: style contract applied to dashboard shell, tables, and inspector panels
- `factory-portal`: style contract applied to factory operations, requests, and manifests surfaces
- `warehouse-portal`: style contract applied to dispatch, locks, and supply request surfaces
- `retailer-app-desktop`: style contract applied to procurement, payments, and supplier surfaces

## Acceptance Checklist

- Frontmatter exists with valid `name` and `description`.
- Rules are explicit and non-ambiguous.
- Accessibility and interaction states are fully documented.
- Desktop app scope is explicitly listed.
- Guidance maps to current implementation constraints and shared desktop foundation.

## TypeUI + Agentic Integration

This `skill.md` is intended for `typeui.sh` workflows and can be consumed by coding agents for desktop UI implementation, review, and migration tasks.

<!-- TYPEUI_SH_MANAGED_END -->
