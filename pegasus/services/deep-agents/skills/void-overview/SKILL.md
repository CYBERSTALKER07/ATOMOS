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
