# Prompt Engineering

## Basics

Higgsfield models reward concrete, sensory prompts.

- **Subject + setting + style**: "a red fox curled in a snowy pine forest, golden hour, cinematic"
- **Camera**: lens (35mm, 85mm), angle (low, overhead), motion (dolly in, tracking shot)
- **Lighting**: rim light, neon glow, moody backlight
- **Style/medium**: oil painting, watercolor, photograph, anime, 3D render

Keep it under ~200 tokens. Models distort with very long prompts.

## Image-to-image

When passing `--image`, the prompt should describe what changes, not redescribe the input.

Bad: "a man with brown hair in a leather jacket holding coffee, made into anime"
Good: "transform into anime style, vibrant colors, soft cel shading"

## Image-to-video

`--start-image` anchors the first frame. Prompt describes motion.

- Verbs: zooms in, dollies left, sweeping pan, slow push, fast whip
- Subject motion: "the dancer spins", "smoke rises slowly"
- Don't redescribe the static frame — model already has it.

## Negative phrasing

Most models don't expose a `negative_prompt`. Phrase positively:
- Instead of "no blur" → "tack sharp"
- Instead of "no people" → "uninhabited landscape"

## Aspect ratio guidance

- `16:9` — landscape, cinematic
- `9:16` — vertical, social
- `1:1` — square, profile / icon
- `4:3`, `3:4`, `21:9` — model-dependent, check `higgsfield model get <jst>`

## Safety

Models reject prompts with `nsfw` or `ip_detected` terminal status. Avoid:
- Real public figures
- Sexual content
- Trademarks / branded characters


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
