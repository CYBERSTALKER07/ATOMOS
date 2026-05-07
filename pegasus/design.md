# Design System: URBN SaaS

This document outlines the core design tokens, layout principles, and interactive guidelines that define the unified URBN aesthetic across all V.O.I.D. desktop portals (Admin, Factory, Warehouse, Retailer).

## 1. Core Philosophy
The design language shifts from a brutalist, strictly monochrome aesthetic to a modern, fluid, and inviting "Soft SaaS" look. Key characteristics include:
- **Depth & Layers:** Soft, multi-layered drop shadows instead of harsh borders.
- **Vibrancy:** Pastel gradients, subtle glows, and semantic colors that pop against neutral surfaces.
- **Motion-First:** Micro-interactions (hover lifts, active presses) and fluid page transitions.
- **Generous Geometry:** Larger border radii (16px - 24px) for cards, inputs, and modals.

## 2. Color Palette Tokens

### Neutral Foundation
- `--background`: `oklch(0.985 0 0)` — Crisp, off-white background.
- `--surface`: `oklch(1 0 0)` — Pure white surface for cards and elevated elements.
- `--foreground`: `oklch(0.145 0 0)` — Deep slate/charcoal for primary text.
- `--muted`: `oklch(0.556 0 0)` — Medium gray for secondary text and disabled states.
- `--border`: `oklch(0.922 0 0)` — Soft, almost invisible borders.

### Semantic Colors
- `--primary`: `oklch(0.21 0.006 285.885)` — Deep slate blue.
- `--accent`: `oklch(0.5 0.15 250)` — Vibrant periwinkle/blue for interactive elements.
- `--success`: `oklch(0.6 0.15 150)` — Soft emerald green.
- `--warning`: `oklch(0.7 0.15 70)` — Warm amber/orange.
- `--danger`: `oklch(0.6 0.2 25)` — Soft crimson red.

## 3. Typography
- **Font Family:** Inter or system sans-serif.
- **Headlines:** Semi-bold to bold, tight tracking (-0.02em).
- **Body:** Regular to medium, open tracking for legibility.
- **Labels (Caps):** Small font size (10-12px), high tracking (0.16em), uppercase, often muted color.

## 4. UI Components

### Cards (`.bento-card`)
- **Background:** `--surface`
- **Border Radius:** `16px` to `24px`
- **Border:** 1px solid `--border`
- **Hover State:** `.hover-lift` utility class. Applies `translateY(-2px)` and a soft shadow `0 10px 20px -10px rgba(0,0,0,0.1)`.

### Buttons (`.md-btn`)
- **Padding:** Generous horizontal padding (px-4 to px-6).
- **Height:** 40px (h-10) or 44px (h-11).
- **Border Radius:** `9999px` (pill shape) or `12px` depending on context.
- **Active State:** `.active-press` utility class `transform: scale(0.97)`.

### Empty States (`<EmptyState />`)
- Must feature a 3D pastel illustration (or high-quality icon if image unavailable).
- Centered layout, large image container with soft background (`bg-surface/50`).
- Headline, descriptive body text, and a primary CTA button.

## 5. Animation & Motion
- **Page Transitions:** `<PageTransition>` wrapper uses Framer Motion (`AnimatePresence`). Pages fade in and slide up slightly (`y: 15` to `0`).
- **List Staggering:** Data grids and lists should use `staggerChildren: 0.05` to cascade items onto the screen.
- **Loading:** Use `.skeleton-shimmer` with a soft gradient spanning the width of the placeholder.

## 6. Implementation Notes
- Global CSS is heavily tokenized in `:root` inside `globals.css`.
- Override HeroUI defaults by assigning our tokens to `--heroui-*` variables.
- Avoid raw hex codes in components; always use CSS variables (e.g., `var(--accent)`).
