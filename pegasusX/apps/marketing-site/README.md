# @pegasusx/marketing-site

Cinematic marketing site for PegasusX / ATOMOS — scroll-driven landing, component docs, and architecture narratives.

## Dev

```bash
cd pegasusX
pnpm install
pnpm --filter @pegasusx/marketing-site dev
```

Runs on **http://localhost:3004**

## Stack

- Next.js 15 App Router, TypeScript strict
- Tailwind CSS v4 + marketing design tokens
- GSAP ScrollTrigger + Lenis smooth scroll
- React Three Fiber (hero, control plane, roles parade)
- MapLibre GL (fleet telemetry demo)

## Routes

| Path | Description |
|------|-------------|
| `/` | Long-form cinematic landing |
| `/platform` | Architecture overview |
| `/roles`, `/roles/[role]` | Six-role ecosystem |
| `/capabilities/[slug]` | Dispatch, outbox, payments, etc. |
| `/components`, `/components/[slug]` | Live component docs |
| `/playground` | Reduced motion toggle |
