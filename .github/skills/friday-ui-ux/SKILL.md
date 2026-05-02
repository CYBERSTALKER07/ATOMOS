---
name: friday-ui-ux
description: "UI/UX design intelligence for web and mobile. Use for planning, building, designing, reviewing, fixing, improving, optimizing, and refactoring interfaces across React, Next.js, Vue, Svelte, SwiftUI, React Native, Flutter, Tailwind, shadcn/ui, and HTML/CSS. Covers style systems, color and typography, accessibility, animation, responsive layout, forms, navigation, charts, and pre-delivery quality gates."
argument-hint: "Describe product type, target platforms, stack, and UI/UX objective."
user-invocable: true
---

# Friday UI UX

## What This Skill Produces

- A clear visual direction matched to the product and audience
- A platform-aware UI structure plan (web, mobile, desktop)
- Interaction and motion guidance that preserves usability and performance
- A prioritized quality review with actionable fixes
- A pre-delivery checklist for accessibility, responsiveness, and reliability
- A structured UX and UI concept taxonomy for onboarding and design vocabulary alignment

## When To Use

Use this skill for tasks that change how the product looks, feels, moves, or is interacted with.

### Must Use

- New pages and screens
- New or refactored components (buttons, forms, cards, modals, tables, charts)
- Color, typography, spacing, and hierarchy decisions
- UX reviews and visual quality audits
- Navigation and interaction pattern design
- Responsive behavior and mobile-first layout adaptation

### Recommended

- UI feels unpolished but root cause is unclear
- Cross-platform parity work (web + iOS + Android)
- Pre-launch interface hardening
- Design system extraction or cleanup

### Skip

- Backend-only logic
- API or database-only work
- Infra or DevOps-only work
- Non-visual automation scripts

## Input Contract

Collect these inputs before proposing UI decisions:

- Product type: dashboard, SaaS, e-commerce, logistics, etc.
- Audience and context: operator vs consumer, office vs field
- Platforms: web, iOS, Android, desktop
- Stack: React, Next.js, SwiftUI, Compose, React Native, etc.
- Constraints: existing design system, brand requirements, accessibility targets

If inputs are missing, ask only for what blocks implementation.

## Foundational UX Topics

- Visual Hierarchy: Arrange size, color, and layout to signal importance.
- Consistency: Reuse standard patterns and elements such as fonts, buttons, and spacing logic.
- Typography: Select legible fonts that improve readability and reinforce product tone.
- Color Theory: Use color intentionally for mood, contrast, emphasis, and accessibility.
- Spacing and Proximity: Group related content and separate unrelated content to reduce clutter.
- Micro-interactions: Use subtle transitions and feedback to guide user actions.
- User Research and Personas: Ground decisions in user goals, behaviors, and constraints.
- Information Architecture: Organize content logically so users can predict where things are.
- Wireframing and Prototyping: Sketch structure first, then validate flow with interactive models.
- Usability Testing: Validate assumptions with real users and identify pain points early.
- User Journey Mapping: Map end-to-end user steps, friction points, and handoff states.
- Accessibility (a11y): Ensure inclusive interaction with screen reader support, keyboard access, and contrast safety.

## Extended Concept Library

Use the extended taxonomy in [UX and UI concepts](./references/ux-ui-concepts.md) when:

- onboarding junior designers
- creating a shared design vocabulary for product and engineering
- building training plans, interview prep guides, or design critique rubrics
- mapping project gaps across research, IA, interaction, visual design, and accessibility

## Color Risk Rules

Use the following colors cautiously, especially for large surfaces or long-duration viewing:

- Bright or cherry red: can create stress, urgency fatigue, and eye strain at high coverage.
- Neon shades (lime green, hot pink): can feel overstimulating and visually noisy.
- Pure white at full intensity: can feel clinical and cause glare fatigue.
- Dark black or charcoal dominance: can feel heavy and depressing without balanced contrast layers.
- Vibrant yellow overuse: can increase eye strain and visual tension.
- Muddy brown dominance: can make interfaces feel dull, dirty, or low-quality.

## Color Combination Warnings

- Red + yellow: can read like aggressive retail promotion and reduce perceived product trust.
- Red + green: often creates holiday association or semantic conflict unless carefully muted and balanced.
- Blue + pink: can clash when saturation and luminance are not controlled, causing disorganized hierarchy.

If these combinations are required by brand constraints, mitigate with muted tones, neutral support colors, strict contrast checks, and limited usage area.

## Workflow

### 1. Task Ingestion

- Classify the task: create, review, fix, optimize, or refactor
- Detect whether this is net-new UI or adaptation of an existing system
- Identify high-risk areas first: accessibility, interaction correctness, layout breakage

Decision point:
- If an existing design system exists, preserve its visual language and extend it
- If no design system exists, generate a baseline system before coding

### 2. Design Direction Selection

- Pick one primary style direction based on product type
- Define semantic color roles (primary, surface, error, warning, success, info)
- Define typography scale and spacing rhythm
- Lock icon family consistency (no emoji as UI icons)

Decision point:
- If enterprise/operator UX: prioritize density + clarity over decoration
- If consumer/brand UX: allow higher visual expression while preserving usability

### 3. Structure And Layout

- Build mobile-first structure, then scale to tablet/desktop
- Set breakpoint behavior explicitly
- Protect safe areas and fixed element offsets
- Avoid horizontal overflow and nested scroll traps

Decision point:
- If data-heavy: prefer table/list + drilldown + filters
- If narrative/marketing: prioritize hierarchy, rhythm, and storytelling flow

### 4. Interaction And Motion

- Set state transitions for hover/pressed/focus/disabled/loading
- Use motion to explain cause-and-effect, not decoration
- Keep animations interruptible and input-safe
- Respect reduced-motion preferences

Decision point:
- If gesture is non-obvious, provide visible alternative controls
- If interaction risk is high, add confirm/undo/recovery path

### 5. Accessibility And Readability Pass

- Validate contrast and semantic structure
- Ensure labels, focus order, and screen-reader clarity
- Guarantee touch target minimums and spacing
- Verify dynamic text scalability does not break layout

Decision point:
- If color is conveying meaning alone, add icon/text redundancy
- If icon-only controls are used, add clear accessible labels

### 6. Performance And Reliability Pass

- Reserve visual space for async content to prevent layout shift
- Use lazy loading and virtualization where needed
- Keep motion and rendering costs bounded
- Provide explicit loading/empty/error/offline states

Decision point:
- If an effect harms readability or frame rate, simplify or remove it
- If content jumps under load, reserve dimensions or use skeletons

### 7. Stack-Specific Implementation Strategy

- React/Next.js/Tailwind/shadcn/ui: align with component primitives and tokens
- SwiftUI: follow HIG patterns, SF Symbols, and native navigation semantics
- React Native/Flutter: prioritize touch ergonomics, safe areas, and platform feedback

Decision point:
- If stack already has a component system, compose from it before creating custom primitives
- If customization is necessary, preserve semantic states and accessibility roles

### 8. Delivery Gates

Do not mark done until all gates pass:

- Accessibility: contrast, focus, labels, keyboard/screen reader support
- Interaction: touch target size, clear states, no gesture conflicts
- Layout: responsive integrity on small phone, large phone, tablet, desktop
- Theming: light/dark parity and semantic token usage
- Performance: no avoidable jank, no major layout shifts
- Feedback: loading/empty/error/offline/restricted states are explicit

## Priority Rule Order

Apply review priorities in this order:

1. Accessibility
2. Touch and interaction correctness
3. Performance and layout stability
4. Style consistency and product fit
5. Responsive and layout integrity
6. Typography and color semantics
7. Motion quality and reduced-motion support
8. Form and feedback clarity
9. Navigation predictability
10. Data visualization clarity

## Quick Decision Matrix

- Need strong usability fast: prioritize accessibility + interaction + layout first
- Need visual polish pass: prioritize style + typography + motion after critical gates pass
- Need dashboard/data UI: prioritize hierarchy, density, table/chart readability, drilldown
- Need cross-platform parity: lock information architecture first, then platform-specific controls

## Output Format

When running this skill, respond with:

1. UI direction summary
2. Component and layout plan
3. Interaction and motion plan
4. Risks and anti-patterns to avoid
5. Concrete implementation steps
6. Validation checklist with pass/fail criteria

## Example Prompts

- Design a logistics control dashboard with dense operational clarity and responsive behavior.
- Review this mobile checkout screen for UX bugs and accessibility failures.
- Refactor this table and filter panel to improve scanability and reduce cognitive load.
- Build a consistent form system with inline validation and error recovery states.
- Improve this page motion so transitions feel natural and remain reduced-motion safe.
- Choose a color and typography system for a B2B SaaS admin product.
- Convert this web-first UI to platform-aware iOS and Android patterns.
- Audit this chart section for accessibility and readability on small screens.

## Optional Integrations

- shadcn/ui MCP: use it for component lookup and implementation examples when working in shadcn-based stacks

## Anti-Patterns To Avoid

- Emoji as structural icons
- Placeholder-only labels
- Color-only status signals
- Decorative animations without meaning
- Fixed pixel layouts that break on smaller screens
- Mixed icon families and inconsistent component states
- Hidden destructive actions without confirmation or undo path
