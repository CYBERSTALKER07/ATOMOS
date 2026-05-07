---
name: design-md-generator-community
source_file: DESIGN.md Generator (Community)
source_page: Page 1
generated_at: 2026-05-08T00:00:00Z
scope: desktop-apps
---

# Desktop Design Contract (Community)

## Context And Goals

This document defines one shared desktop visual and interaction language for all Pegasus desktop surfaces.

### Source References

Visual direction is derived from the local image set in `pegasus/assets`, especially:

- Conceptzilla-style CRM desktop compositions (`1d42d3c569b68b3816f840c3b9066724.webp`, `62665a5798605f336decc4a26bbc4911.webp`, `8601eed85c98c6c01f57cc314eb77094.webp`)
- Fleet operations workspace composition (`original-6f769336f644695f12fff0002b352b2e.webp`)
- Clean analytics dashboard composition (`CD123FFA-EBD9-4790-A6F8-5788ACF80A54.jpeg`, `DB1E9E1A-315A-47B0-A2F7-9B9776725EE3.jpeg`)

### Desktop App Scope

This style contract applies to:

- `pegasus/apps/admin-portal` (web + Tauri desktop shell)
- `pegasus/apps/factory-portal` (web + Tauri desktop shell)
- `pegasus/apps/warehouse-portal` (web + Tauri desktop shell)
- `pegasus/apps/retailer-app-desktop` (web + Tauri desktop shell)

### Design Intent

Build a calm, high-density operational desktop UI: neutral canvas, strong data hierarchy, compact controls, and selective accent emphasis for actions and state.

## Design Tokens And Foundations

### Color Tokens

Use semantic tokens only. Raw hex in component code is prohibited.

| Token | Value | Purpose |
| --- | --- | --- |
| `--desk-canvas` | `#F3F4F6` | Global app background |
| `--desk-surface` | `#FFFFFF` | Cards, panes, drawers |
| `--desk-surface-subtle` | `#F8FAFC` | Alternate rows, grouped controls |
| `--desk-border` | `#E5E7EB` | Dividers, table rules, input outlines |
| `--desk-border-strong` | `#CBD5E1` | Active containers, emphasized boundaries |
| `--desk-text-primary` | `#111827` | Primary labels and values |
| `--desk-text-secondary` | `#6B7280` | Metadata and secondary text |
| `--desk-text-tertiary` | `#9CA3AF` | Placeholder and tertiary hints |
| `--desk-accent` | `#FF7A1A` | Primary CTA, selected tab underline, key trend glyph |
| `--desk-accent-soft` | `#FFF3EA` | Soft active backgrounds |
| `--desk-success` | `#16A34A` | Positive state |
| `--desk-warning` | `#D97706` | Caution state |
| `--desk-danger` | `#DC2626` | Error and destructive state |
| `--desk-info` | `#2563EB` | Informational state |
| `--desk-focus-ring` | `#111827` | Keyboard focus-visible outline |

### Typography Tokens

| Token | Value |
| --- | --- |
| `--type-display-xl` | `700 48px/56px Geist, Inter, "Segoe UI", sans-serif` |
| `--type-display-lg` | `700 40px/48px Geist, Inter, "Segoe UI", sans-serif` |
| `--type-heading-lg` | `600 32px/40px Geist, Inter, "Segoe UI", sans-serif` |
| `--type-heading-md` | `600 24px/32px Geist, Inter, "Segoe UI", sans-serif` |
| `--type-title` | `600 18px/26px Geist, Inter, "Segoe UI", sans-serif` |
| `--type-body-lg` | `400 16px/24px Geist, Inter, "Segoe UI", sans-serif` |
| `--type-body-md` | `400 14px/22px Geist, Inter, "Segoe UI", sans-serif` |
| `--type-caption-sm` | `500 12px/18px Geist, Inter, "Segoe UI", sans-serif` |
| `--type-metric` | `700 36px/40px Geist, Inter, "Segoe UI", sans-serif` |

All numeric KPI and financial values must use tabular numerals.

### Spacing Tokens

| Token | Value |
| --- | --- |
| `--space-0` | `0px` |
| `--space-1` | `4px` |
| `--space-2` | `8px` |
| `--space-3` | `12px` |
| `--space-4` | `16px` |
| `--space-5` | `20px` |
| `--space-6` | `24px` |
| `--space-8` | `32px` |
| `--space-10` | `40px` |

### Radius Tokens

| Token | Value |
| --- | --- |
| `--radius-sm` | `8px` |
| `--radius-md` | `12px` |
| `--radius-lg` | `16px` |
| `--radius-xl` | `20px` |
| `--radius-pill` | `9999px` |

### Motion Tokens

| Token | Value |
| --- | --- |
| `--duration-fast` | `120ms` |
| `--duration-base` | `200ms` |
| `--duration-slow` | `320ms` |
| `--ease-standard` | `cubic-bezier(0.2, 0, 0, 1)` |
| `--ease-enter` | `cubic-bezier(0.05, 0.7, 0.1, 1)` |
| `--ease-exit` | `cubic-bezier(0.3, 0, 0.8, 0.15)` |

Motion must communicate state changes only. Decorative motion is prohibited.

## Component-Level Rules

### App Shell

- Left navigation rail must be persistent on desktop.
- Expanded rail width must be `264px`; collapsed width must be `72px`.
- Top utility bar must be `64px` high with global search and account actions.
- Content area must use `24px` page padding and `16px` internal card padding.

### KPI Strip

- KPI strip must appear at the top of analytics and operational pages.
- Each KPI card must include: label, value, trend indicator, and period context.
- KPI cards must support loading skeleton and stale-data timestamp.

### Search + Filter Command Row

- Command row must include search, filters, and sort controls in one horizontal group.
- On narrow desktop widths, controls must wrap without horizontal scroll.
- Filter state must remain visible and keyboard-resettable.

### Dense Data Table

- Tables must default to dense rows (`44px` row height target).
- Header row must stay sticky when vertical scroll is present.
- Long text must truncate with tooltip and optional detail drawer access.
- Empty state must include a clear action path (create/import/reset filters).

### Inspector Drawer / Detail Panel

- Detail panel should open from right side for selected entity context.
- Panel width should be between `360px` and `440px`.
- Panel must include close control, title, key metadata, and contextual actions.

### Status Chips

- Status chips must use semantic token mapping only.
- Chips must support compact mode (11px label) and standard mode (12px label).
- Color alone must not carry meaning; text label is mandatory.

### Bulk Action Bar

- Bulk action bar must appear as a floating bottom bar when rows are selected.
- Bar must show selected count and available actions.
- Destructive actions must require confirmation.

### Behavior Requirements (Keyboard, Pointer, Touch)

- Every interactive control must support `Tab` focus order and `Enter` activation.
- Row selection must support checkbox + keyboard selection.
- Hover states should exist for pointer users; focus-visible must exist for keyboard users.
- Touch compatibility must preserve minimum `40px` interactive target size.

### Overflow, Long Content, And Empty States

- Long labels must truncate gracefully and expose full text via tooltip or drawer.
- Metric cards must handle values up to 12 characters without clipping.
- Empty tables must include: title, explanation, and one recovery action.
- Offline states must include retry action and last successful sync time.

## Accessibility Requirements And Testable Acceptance Criteria

Target: WCAG 2.2 AA.

1. Interactive text and icons must meet contrast ratio >= 4.5:1.
2. Focus-visible indicators must be present and clearly visible on all controls.
3. Keyboard users must access all actions without pointer input.
4. Status communication must include text and not rely on color only.
5. Modals and drawers must trap focus and restore focus on close.
6. Tables must expose semantic headers and row navigation for assistive tech.
7. Reduced-motion preference must disable non-essential transitions.

## Content And Tone Standards

Tone must be concise, operational, and implementation-focused.

### Examples

- Preferred: `3 pending manifests require review.`
- Avoid: `Heads up! You might want to look at your manifests soon.`
- Preferred: `Sync failed. Retry or continue in offline mode.`
- Avoid: `Oops! Something weird happened.`

## Anti-Patterns And Prohibited Implementations

- Do not use decorative gradients, glassmorphism, or emoji icons.
- Do not hide critical actions behind hover-only affordances.
- Do not mix unrelated accent colors per page.
- Do not remove skeleton/empty/offline/restricted states for any live screen.
- Do not hard-code one-off spacing, radius, or color values in feature code.

## Desktop Rollout Matrix

| Desktop app | Required style outcome |
| --- | --- |
| `admin-portal` | Left rail + KPI strip + dense table + inspector drawer alignment |
| `factory-portal` | Same shell rhythm, status-chip semantics, and command row structure |
| `warehouse-portal` | Same shell rhythm, dispatch table density, and right-panel detail behavior |
| `retailer-app-desktop` | Same shell rhythm, procurement tables, and compact KPI treatment |

## QA Checklist

- [ ] Uses semantic tokens only, no raw hex in component logic.
- [ ] Includes loading, empty, offline, stale, and restricted states.
- [ ] Verifies keyboard navigation and focus-visible indicators.
- [ ] Verifies contrast ratios for text, icons, and control states.
- [ ] Verifies KPI, table, and drawer behavior under long-content scenarios.
- [ ] Verifies responsive desktop behavior at 1280, 1440, and 1728 widths.
- [ ] Verifies visual consistency across all four desktop apps.
