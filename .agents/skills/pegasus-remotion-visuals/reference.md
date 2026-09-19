# Pegasus Visuals — Topic Catalog & Prompt Reference

Output path pattern for every topic:

```
softwareengineercv-main/public/media/{category}/{slug}.mp4
```

## Ecosystem film chapters (10 min, no timestamps)

Use **scene sequences** instead of beat sheets. One order dot carries through the film.

| Chapter | Topics covered | End label |
|---------|----------------|-----------|
| ONE CONTROL PLANE | atomos-control-plane, network-topology | ONE CONTROL PLANE |
| HOW PEGASUS WORKS | how-pegasus-works, order-lifecycle | COMPLETED |
| SIX ROLES, ONE ORDER | supplier, warehouse, factory, driver, retailer, payload-gate, role-parity-matrix | SAME CONTRACT EVERYWHERE |
| MORNING DISPATCH | dispatch-the-right-load, visual-dispatch-engine, smarter-dispatch, dispatch-preview, fleet-visibility, warehouse-operations | HUMAN IN CONTROL |
| WHEN THINGS BREAK | zone-miss-handling, concurrent-stock-reject, truck-too-small, partial-dispatch-commit, wrong-truck-sealed, driver-reassignment, shop-closed-at-delivery, returns-wrong-barcode, live-tracking-expectations | EXCEPTIONS ARE FIRST-CLASS |
| PAYMENT & TREASURY | payment-confidence, treasury-integrity, cash-collection, cash-at-door-cod | PAY AT DELIVERY |
| RELIABLE UPDATES | mutating-handler-contract, reliable-updates, trust-reliability, instant-coordination, network-coordination | TRUSTED STATE |
| UNDER THE HOOD | go-backend-platform, cloud-spanner, redis-kafka, websocket-hubs, smart-dispatch-assist, ai-worker-vrp, mobile-apps, web-apps, enterprise-rollout, request-demo | PEGASUS |

## Duplicates — show once in ecosystem film

| Keep | Skip |
|------|------|
| platform/reliable-updates | capabilities/reliable-updates |
| solutions/payment-confidence | capabilities/payment-confidence |
| solutions/live-fleet-tracking | capabilities/live-fleet-tracking |
| solutions/network-coordination | apps-deploy/realtime-coordination (merge) |

## Scene prompt template (no timing)

```
[GLOBAL STYLE — monochrome line art, 1920×1080, 24fps, silent]

CHAPTER: {title}
NARRATIVE JOB: {one sentence}

SCENE SEQUENCE:
1. {opening visual + label}
2. {action / morph}
3. {detail or complication}
4. {resolution}
5. {transition cue — order dot, hub pulse, pan}

TOPICS: {slug list}
ROLES: {Supplier, Warehouse, ...}
END LABEL: {2–4 WORDS}
TRANSITION: {morph | pan | hub recall | order dot carryover}
```

## Short clip template (10s loops)

```
DURATION: 10 seconds. 24fps. Hold last 0.5s.

VISUAL: Black void, white 1–2px strokes, stroke-draw reveals.
MOTION: ~1.2s per beat, ease-in-out, no cuts.
AUDIO: None.

TOPIC: {title}
FILENAME: public/media/{category}/{slug}.mp4

SCENE SEQUENCE: {ordered beats without timestamps}
END LABEL: {UPPERCASE}
```

## Full topic index (68)

See `pegasusX/visuals/topics-manifest.json` for machine-readable slugs by category.

Nav labels and descriptions: `softwareengineercv-main/app/data/megaNavigation.ts`

Interactive flow fallbacks on topic pages (when video missing): `softwareengineercv-main/app/components/flows/`


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
