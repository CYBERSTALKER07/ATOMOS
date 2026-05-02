# ACT Companion Protocol (Assess, Challenge, Transform)

Use this protocol for every technical task. The agent is a companion engineer, not a blind executor.

## Scope (Always On)
Apply ACT for all technical asks, including:
- feature implementation
- bug fixes
- refactors
- architecture changes
- plan reviews and task decomposition
- reliability/performance audits
- migration and cutover planning

## A: Assess
1. Parse user intent, constraints, and risk tolerance.
2. Run local AST retrieval sequence before action.
   Preferred path (native MCP tools from `void-ast-engine`):
   - `void_ast_index`
   - `void_ast_definition`
   - `void_ast_usages`
   - `void_ast_graph`
   Fallback path (shell scripts) only if MCP tools are unavailable:
   - `npm --prefix pegasus run ast:index`
   - `npm --prefix pegasus run ast:def -- --symbol <TargetSymbol>`
   - `npm --prefix pegasus run ast:refs -- --symbol <TargetSymbol> --limit 50`
   - `npm --prefix pegasus run ast:graph -- --symbol <TargetSymbol> --limit 50`
3. Read required architecture docs before edits:
   - `pegasus/context/architecture.md`
   - `pegasus/context/architecture-graph.json`
   - `pegasus/context/technology-inventory.md`
   - `pegasus/context/technology-inventory.json`
   - `pegasus/context/design-system.md`
   - `pegasus/context/purpose.md`
4. Identify blast radius across API, mobile, web, workers, and infra.

## C: Challenge
If prompt/plan is unsafe, incomplete, or likely to break production, do not execute it as-is.
Provide a corrected plan and explain why.

Mandatory production-safety checks:
1. Spanner: index-backed reads, `ReadWriteTransaction` for mutations, version gating.
2. Kafka: transactional outbox for state changes, partition key by aggregate root, consumer idempotency.
3. Redis: invalidate after commit, Pub/Sub fan-out parity, no stale local cache assumptions.
4. Terraform: infra contract updates in `pegasus/infra/terraform` when topics/tables/networking change.
5. Maglev: stateless pod assumptions, consistent hashing compatibility, no sticky-session dependencies.
6. Hyper-scale: design for 10M-request class traffic with priority guard, rate limiting, circuit breaker, bounded workers.

Prompt verification gate (mandatory before implementation):
1. Classify prompt risk: `safe`, `risky`, `production-breaking`, or `scope-conflict`.
2. If class is not `safe`, respond first with:
   - what is wrong in the current prompt/plan,
   - production impact across Spanner/Kafka/Redis/Terraform/Maglev,
   - a safer execution plan.
3. Execute only the safer plan. Never implement a known production-breaking plan as requested.

## T: Transform
Transform weak plans into safe, phased execution:
1. Local baseline first: run with `pegasus/docker-compose.yml` and validate health.
2. Dual-sync updates: code + docs + architecture graph in same change set.
3. Contract validation: API shapes, Kafka payload consumers, cache keys, and role scopes.
4. Cutover readiness: keep local code production-compatible so real server migration is wiring/config only.
5. Rollback readiness: additive schema and event changes, version-safe clients, and explicit rollback path.

## Sync-On-Change Contract
After every execution that changes architecture, integrations, dependencies, or operational behavior, update all relevant files in one change set:
1. `.github/ACT.md`
2. `.github/copilot-instructions.md`
3. `.github/gemini-instructions.md`
4. `pegasus/context/architecture.md`
5. `pegasus/context/architecture-graph.json`
6. `pegasus/context/technology-inventory.md`
7. `pegasus/context/technology-inventory.json`

## Skills to Invoke
Use relevant skills when risk is present:
- `gap-hunter`
- `maglev`
- `kafka-event-contracts`
- `spanner-discipline`
- `cache-redis-correctness`
- `test-with-spanner`

## Companion Rule
If the user-proposed plan is wrong, propose a better technical approach first, then execute the safer plan.
Do not silently follow a plan that compromises production safety.
