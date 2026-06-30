# PegasusX Visuals (Remotion)

Monochrome line-art marketing videos for the Pegasus site (`softwareengineercv-main`).

## Quick start

```bash
cd pegasusX/visuals
npm i
npm run dev          # Remotion Studio
```

## Render to marketing site

Outputs land in `softwareengineercv-main/public/media/{category}/{slug}.mp4`:

```bash
npm run render:order-lifecycle
npm run render:ecosystem   # ~10 min, 14400 frames — expect several minutes
```

### Troubleshooting: `Z_BUF_ERROR` / `unexpected end of file`

First render downloads Chrome Headless Shell (~150MB). A interrupted download causes:

```
Error: unexpected end of file
code: 'Z_BUF_ERROR'
```

Fix:

```bash
npm run reset-chrome
npm run render:order-lifecycle
```

Or manually delete `node_modules/.remotion/chrome-headless-shell` and re-run render.

## Project layout

```
visuals/
├── src/
│   ├── compositions/     # One file per Remotion composition
│   ├── components/       # LineCanvas, StrokeDraw, MonoLabel
│   ├── style/tokens.ts   # #000 / #FFF, 1920×1080, 24fps
│   └── lib/
│       ├── compositions.ts  # Registry + durations
│       └── paths.ts         # Export paths to Next.js public/
├── public/                 # Static assets for Remotion
├── topics-manifest.json    # All nav topic slugs → output paths
└── .agents/skills/remotion-best-practices/  # Upstream Remotion skill
```

## Compositions

| ID | Duration | Output |
|---|---|---|
| `OrderLifecycle` | 10s | `public/media/platform/order-lifecycle.mp4` |
| `PegasusEcosystemFlow` | 10 min | `public/media/platform/pegasus-ecosystem-flow.mp4` |

## Agent skills

- **Project skill:** `.agents/skills/pegasus-remotion-visuals/` (repo root) — Pegasus visual language + topic map
- **Remotion skill:** `visuals/.agents/skills/remotion-best-practices/` — Remotion APIs and patterns

When prompting a coding agent:

```
cd pegasusX/visuals && npm run dev
Use pegasus-remotion-visuals + remotion-best-practices skills.
```

## Visual rules

- Black `#000000` background, white `#FFFFFF` 1–2px strokes only
- Stroke-draw reveals via `StrokeDraw` + `interpolate()` (no CSS transitions)
- 1920×1080, 24fps, silent
- End each clip with 0.5s hold frame

See `.agents/skills/pegasus-remotion-visuals/reference.md` for the full topic catalog and chapter map.
