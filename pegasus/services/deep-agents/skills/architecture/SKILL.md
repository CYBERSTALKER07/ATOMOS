---
name: architecture
description: Blast-radius checklist, Class A DoD, layering, anti-islands, desktop stack decisions.
---

# Architecture

## Blast radius (every feature change)

1. Role(s) affected
2. Route owner (`*routes/routes.go`)
3. Spanner mutation + same-txn outbox
4. Consumers (Kafka / WS / push / webhook)
5. Role-row clients
6. Contracts / types if shape changes
7. Cloud flags / images if env changes

## Class A done

Spanner write + outbox + consumer/fanout + contracts if needed + all row clients + tests/SSMR.

Class B = backend island · Class C = UI island (ban for prod) · Class D = flag/cert blocked.

## Stack decisions

- Desktop: **Next.js + Tauri 2** — do not recommend Electron without a hard blocker
- Tree SoT: `pegasusX/`; `pegasus/` is legacy
- Quality harness: Deep Agents orchestra (not production AI; that is `ai-worker`)

## Evidence

`.agents/AGENTS.md`, `docs/ROLE_ROW_PARITY_MATRIX.md`, `docs/DATA_FLOW_AS_IMPLEMENTED.md`


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
