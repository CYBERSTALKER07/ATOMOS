# HeroUI fallback — components Quanta doesn't have (`type: "app"` builds ONLY)

Quanta is always first. When the app genuinely needs a complex interactive
component Quanta lacks — date picker, calendar, date-range picker, sortable
data table, multiselect autocomplete/combo-box, color picker, OTP input,
tree/disclosure — use HeroUI v3 (`@heroui/react`), which is preinstalled in
the template and themed to the Higgsfield brand. Websites (`type: "website"`)
never use it.

## The rules

1. **Quanta first.** If Quanta ships the component (Button, Modal, Tabs,
   Select, Dropdown, Input, Textarea, Sidebar, Chip, Badge, Loader, Media,
   Grid, …), HeroUI is the wrong answer. HeroUI covers the gaps only.
2. **Never restyle.** The theming lives in `app/src/heroui-theme.css`
   (designer-authored: lime `#D1FE17` accent, quanta surface greys, Inter,
   moderate radii, dark-only). Do not pass color/size-changing classNames
   into HeroUI components, do not edit the theme file, do not add a light
   mode. Layout-only positioning classes on your own wrappers are fine.
3. **License bright line.** HeroUI core is Apache-2.0 and consumed as an npm
   dependency — fine. NEVER copy component source from HeroUI Pro
   (heroui.pro) into the app: Pro's license forbids redistributing source,
   and generated apps are redistributed source.

## The recipe

`@heroui/react` (tree-shakeable) and `@heroui/styles` are already installed,
and `src/styles.css` already loads HeroUI's base + theme + utilities. Using a
fallback component takes two steps:

1. Import it:

```tsx
import { Calendar } from "@heroui/react";
```

2. Add that component's structural CSS ONCE to `src/styles.css`, next to the
   existing HeroUI imports:

```css
@import "@heroui/styles/components/calendar.css" layer(components);
```

(The available names match `node_modules/@heroui/styles/dist/components/*.css`
— e.g. `date-picker.css`, `table.css`, `combo-box.css`, `color-picker.css`.
Some components layer on others' CSS — if something renders unstyled, add the
CSS of its building blocks too, e.g. `date-picker` also wants `calendar.css`
and `popover.css`.)

Do NOT `@import "@heroui/styles"` (the full index) — it re-imports all of
Tailwind and every component's CSS.

## Consistency checks when mixing

- HeroUI buttons inside fallback composites are fine, but standalone actions
  stay Quanta `Button` — one button system per screen.
- HeroUI's semantic variants are themed to the brand: `primary` = lime with
  dark text, `secondary`/`default` = dark surfaces, `danger` = destructive
  red (destructive confirms only — product error STATES stay neutral/grey).
- Icons inside HeroUI slots are still Material Symbols (the app's one icon
  family) — pass your own icon nodes rather than HeroUI's defaults where a
  slot allows it.


# Universal Agent & Engineering Guidelines
When developing, designing, or planning, always ensure to account for:
- Gaps, edge cases, and comprehensive feature validation.
- Best practices and optimized integration for Kafka, Redis, Backend, Optimizers, AI, and UI.
- Real-time concepts including WebSockets, webhooks, and their native app equivalents.
- Thorough business logic for features, understanding how the role, app, and ecosystem work together, and engagements with other roles and features.
- Best practices for backend, frontend, and infrastructure libraries/packages. Always prefer existing, high-quality open-source libraries and packages that best suit our features before creating our own.
- Optimal UI infrastructure and UX patterns (e.g., optimal screen positioning for drivers during an active route), applying the same high standards to backend and cloud architecture.
- ALWAYS search the web to find open-source code, libraries, packages, math, algorithms, approaches, and best practices for anything we are doing. If none exist, then create our own.
- Always search the web to get the correct logic, and incorporate edge cases, business logic for features, operations (ops), workflow, data consistency, finance, and AI into everything we do.
