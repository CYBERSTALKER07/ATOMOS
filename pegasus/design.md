# Desktop Design Contract (Pegasus Mirror)

This file is a mirror entrypoint for Pegasus-internal paths.

## Canonical Source Of Truth

- The canonical desktop UI contract is at `/design.md` in the repository root.
- All desktop UI system work must follow `/design.md`.
- If this file and `/design.md` differ, `/design.md` takes precedence.

## Scope

Applies to:

- pegasus/apps/admin-portal
- pegasus/apps/factory-portal
- pegasus/apps/warehouse-portal
- pegasus/apps/retailer-app-desktop

## Implementation Bridge

- Shared desktop token foundation remains `pegasus/packages/ui-kit/styles/desktop-foundation.css`.
- App-specific `globals.css` layers may extend but must not violate `/design.md` constraints.
- Repository constraints from `/design.md` are mandatory, including:
  - no decorative gradients on product surfaces
  - no emoji icons for UI semantics
  - explicit loading, empty, offline, stale, and restricted states for live screens
- Legacy decorative utility classes may remain for compatibility but must be visually disabled on product surfaces.

## Operational Note

Use this file for Pegasus-local discovery only.

Use `/design.md` for all normative design decisions.
