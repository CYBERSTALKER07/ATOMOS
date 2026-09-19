---
name: data-flow-coverage
description: Enforce Spanner→outbox→Kafka→WS coverage rule and Class A/B/C/D classification for any feature.
---

# Data-flow coverage

## Coverage rule

Every Spanner state mutation must:

1. Emit an outbox event in the **same RW transaction** (`outbox.SpannerTxnBuffer` / `EmitJSON`).
2. Have a **declared consumer** (NotificationDispatcher fanout, domain mutator, webhook, or explicit “no fanout” ADR).
3. Reach **clients** that need to react (WS hub envelope + inbox/push as appropriate).

## Classification

| Class | Meaning | Action |
|-------|---------|--------|
| A | Full E2E wired | Ship |
| B | Backend island (API/schema, weak client/consumer) | Wire consumer + clients |
| C | UI island (screen without real hop) | Ban for prod; wire backend |
| D | Flag / cert / external key blocked | Document gate; do not claim live |

## Audit steps

1. Find mutation entrypoint (`service.go`, routes).
2. Search for `outbox.Emit` / `EmitJSON` in same txn path.
3. Map event type → `kafka/` consumer or dispatcher switch.
4. Map WS hub + `packages/types` envelope.
5. Check role apps consume the event or poll with documented reason.
6. Cite gap register if already tracked (`docs/session-2026-08-07/ECOSYSTEM_GAP_REGISTER_*.md`).

## Kernel references

- `docs/DATA_FLOW_AS_IMPLEMENTED.md`
- `docs/session-2026-08-07/MASTER_ALIGNMENT_DATAFLOW_*.md`
- `apps/backend-go/outbox/`, `kafka/`, `events/`, `ws/`


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
