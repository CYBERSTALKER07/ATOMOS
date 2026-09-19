# or-tools-sidecar-optimization

Use when integrating, extending, or auditing the OR-Tools optimization sidecar under `pegasus/services/optimizer-core`.

## Use Cases
- Add or modify VRP routing logic in `server/vrp_adapter.py`.
- Add or modify CP-SAT scheduling logic in `server/cpsat_adapter.py`.
- Change request/response contracts in `proto/optimizer_core.proto`.
- Extend Go worker adapter flow in `adapters/go/internal/adapter/worker.go`.
- Verify solver integration guardrails (timeouts, retries, outbox-only persistence).

## Non-Negotiable Guardrails
1. Keep integer scaling deterministic using `SCALE_FACTOR=10000`.
2. Keep UUID-index mapping deterministic and reversible.
3. Enforce bounded solver time limits and best-effort result semantics on timeout.
4. Keep worker retries bounded with exponential backoff + jitter.
5. Persist solver outputs via Spanner `OutboxEvents` only.
6. Do not mutate Inventory or Manifest tables directly from optimization worker paths.
7. Keep contracts additive; avoid breaking existing optimization payload consumers.

## Files To Check Together
- `pegasus/services/optimizer-core/proto/optimizer_core.proto`
- `pegasus/services/optimizer-core/server/scaling.py`
- `pegasus/services/optimizer-core/server/mapping.py`
- `pegasus/services/optimizer-core/server/vrp_adapter.py`
- `pegasus/services/optimizer-core/server/cpsat_adapter.py`
- `pegasus/services/optimizer-core/server/service.py`
- `pegasus/services/optimizer-core/server/main.py`
- `pegasus/services/optimizer-core/adapters/go/internal/model/types.go`
- `pegasus/services/optimizer-core/adapters/go/internal/adapter/translate.go`
- `pegasus/services/optimizer-core/adapters/go/internal/adapter/worker.go`
- `pegasus/services/optimizer-core/adapters/go/internal/optimizergrpc/client.go`

## Validation Checklist
- Python sidecar imports cleanly.
- Go adapter builds (`GOWORK=off go build ./...`).
- JSON/proto contract changes are reflected in both Python and Go layers.
- Outbox write path remains idempotent for `OPTIMIZATION_SOLVED`.
- No direct state mutation path bypassing transactional outbox is introduced.


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
