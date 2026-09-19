---
name: void-overview
description: High-level map of the V.O.I.D / PegasusX monorepo for Deep Agents.
---

# V.O.I.D monorepo overview

## Layout

- `pegasusX/` — **source of truth** (backend-go, portals, native apps, contracts, infra, gap register).
- `pegasus/` — legacy / sidecar home (`services/optimizer-core`, `services/deep-agents` runtime).
- `pegasus/services/deep-agents/` — LangChain Deep Agents quality orchestra (not production AI).
- Production AI: `pegasusX/apps/ai-worker`.

## Guardrails

- Prefer real contracts under `pegasusX/contracts/` over invented shapes.
- Do not introduce mock data into production paths.
- Mutating handlers: Spanner txn + outbox + events (coverage rule).
- Audit via Chief Orchestrator + 12 specialist panels (`void-ecosystem-audit --full`).


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
