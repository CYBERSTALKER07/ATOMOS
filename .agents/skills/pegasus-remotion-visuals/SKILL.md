---
name: pegasus-remotion-visuals
description: >-
  Creates and renders PegasusX marketing videos with Remotion in pegasusX/visuals.
  Monochrome line-art (#000/#FFF), stroke-draw animations, topic-aligned compositions.
  Use when building ecosystem flow videos, order lifecycle clips, topic page media,
  or wiring MP4 exports to softwareengineercv-main/public/media.
---

# Pegasus Remotion Visuals

## When to use

- User asks for Pegasus marketing videos, line-art animations, or ecosystem flow films
- Adding or editing compositions in `pegasusX/visuals/`
- Rendering MP4s for `softwareengineercv-main` topic pages or hero sections

**Also load:** `pegasusX/visuals/.agents/skills/remotion-best-practices` for Remotion API rules.

## Project location

```
pegasusX/visuals/          ← Remotion project (run all commands here)
pegasusX/softwareengineercv-main/public/media/  ← rendered output
```

## Commands

```bash
cd pegasusX/visuals
npm run dev                    # Remotion Studio preview
npm run lint                   # eslint + tsc
npm run render:order-lifecycle # → public/media/platform/order-lifecycle.mp4
npm run render:ecosystem       # → public/media/platform/pegasus-ecosystem-flow.mp4
```

## Visual language (required)

| Rule | Value |
|------|-------|
| Background | `#000000` only |
| Strokes | `#FFFFFF`, 1–2px, no fill |
| Labels | Uppercase monospace, max 60% white opacity |
| Resolution | 1920×1080, 24fps |
| Audio | Silent |
| Motion | Stroke-draw via `StrokeDraw` + `interpolate()` — **no CSS transitions** |
| Hold | Last 0.5s static on short clips; 3s on ecosystem film |

**Forbidden:** color, gradients, photorealism, logos, Tesla, glossy UI, CSS/Tailwind animations.

Use tokens from `src/style/tokens.ts`. Reuse `LineCanvas`, `StrokeDraw`, `MonoLabel`.

## Adding a composition

1. Create `src/compositions/{PascalCase}.tsx`
2. Register in `src/lib/compositions.ts` with `category`, `slug`, `durationSeconds`
3. Add `<Composition>` in `src/Root.tsx`
4. Add npm script: `remotion render {Id} ../softwareengineercv-main/public/media/{category}/{slug}.mp4`
5. Wire site asset path in `softwareengineercv-main` if needed (e.g. `lifecycleAssets.ts`)

## Two-tier video strategy

| Tier | Duration | Use |
|------|----------|-----|
| Short loop | 10s | Topic page hover cards |
| Ecosystem film | 10 min (`PegasusEcosystemFlow`) | Platform tour / hero |

Short clips can be **excerpts** cut from chapter renders. See `topics-manifest.json` for all 68 topic slugs.

## Ecosystem film structure

8 chapters × ~75s in `PegasusEcosystemFlow.tsx` (Series). Expand each `ChapterCard` into full scene sequences — no per-beat timestamps; use ordered scene lists.

Chapter map and duplicate-merge rules: [reference.md](reference.md)

## Site integration

After render, reference from Next.js:

```ts
export const TOPIC_VIDEO = '/media/platform/order-lifecycle.mp4';
```

Existing pattern: `softwareengineercv-main/app/lib/lifecycleAssets.ts`

## Checklist before render

- [ ] `npm run lint` passes in `pegasusX/visuals`
- [ ] Composition ends on hold frame
- [ ] Output path matches `topics-manifest.json` slug
- [ ] No forbidden colors or CSS animations
- [ ] Previewed in Remotion Studio at 24fps

## Additional resources

- Topic catalog + prompts: [reference.md](reference.md)
- Remotion upstream skill: `pegasusX/visuals/.agents/skills/remotion-best-practices/`
- Nav slug source of truth: `softwareengineercv-main/app/data/megaNavigation.ts`
