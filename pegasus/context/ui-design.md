# V.O.I.D. UI / UX Doctrine

This document defines the strict visual and behavioral rules for all user-facing surfaces.

## Core Directives
1. **Operational, Not Decorative**: Build for clarity and density. No decorative gradients, glassmorphism, or distracting dot/grid patterns. Solid surface tokens only.
2. **No Emojis**: All icons must be actual SVGs (Material Symbols, Heroicons, Lucide). Absolutely no emoji characters as indicators or markers.
3. **Data Completeness**: Every screen MUST account for:
   - Loading `BentoSkeleton` blocks
   - Empty states
   - Offline / disconnected states
   - Permission-restricted states
4. **Telemetry Visibility**: Show active execution states (drivers, routes, metrics) rather than static data. Drill-downs from aggregates are mandatory.

## The Bento Grid Dashboard Protocol
The Admin Portal heavily relies on CSS Grid logic.
- **Invariant**: The dashboard is a mosaic where cell size equals data priority.
- **Sizes**: `"stat"` (1x1), `"list"` (1x2), `"control"` (2x1), `"anchor"` (2x2).
- **Structure**: Every widget is wrapped in `<BentoCard size="...">` from `@/components/BentoGrid`.
- **Styling**: High-contrast borders (`1px solid var(--color-md-outline-variant)`). No shadows. Brutalist default (radius 0) or Apple theme (radius 24px).
- **Data Density**: Use `<MiniSparkline>` over large, wasteful charts where trend is enough.

## Platform Constraints
- **Web (Admin/Factory/Warehouse Portals)**: Next.js 15, React 19, Tailwind v4. Strict M3 CSS definitions.
- **Android**: Pure Kotlin Multiplatform / Jetpack Compose Material 3.
- **iOS**: Pure SwiftUI, strict Apple HIG, SF Symbols. No pseudo-material iOS.
- **Payload Terminal**: React Native + Expo enforcing M3 metrics.