---
name: void-overview
description: High-level map of the V.O.I.D / Pegasus monorepo for Deep Agents.
---

# V.O.I.D monorepo overview

## Layout

- `pegasus/` — primary backend (Go services, Spanner, Kafka, Redis, k8s).
- `pegasusX/` — product surface (portals, native apps, contracts, infra).
- `pegasus/services/optimizer-core/` — OR-Tools Python sidecar.
- `pegasus/services/deep-agents/` — this LangChain Deep Agents workspace.

## Guardrails

- Prefer real contracts under `pegasusX/contracts/` over invented shapes.
- Do not introduce mock data into production paths.
- Mutating handlers follow monorepo doctrine (Spanner tx + outbox + events).
