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
