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
   - `pegasus/context/ui-design.md` for every UI-affecting task (web, desktop, Android, iOS, Expo)
4. Apply codebase-first weighting: primary context must come from real runtime code (definitions, usages, graph). Docs are mandatory verification, not a substitute for code retrieval.
5. Identify blast radius across API, mobile, web, workers, and infra.
6. UI gate (mandatory for UI-affecting work): before editing any user-facing surface, enumerate the backend endpoint/event/DTO feeding the screen, the frontend data layer that maps it, and every client in the role row that also consumes the feature. Do not treat a single web page or app screen as complete context.
7. Warehouse live gate: treat `/ws/warehouse` plus the supply-request and dispatch-lock DTOs as one contract across warehouse portal, warehouse iOS, and warehouse Android. Do not change one consumer without checking the other two.
8. Warehouse live resilience gate: `/ws/warehouse` consumers must auto-reconnect and surface stale/offline state. A silent frozen live view does not count as complete.
9. Supplier geo-planning gate: treat `/v1/supplier/serving-warehouse`, `/v1/supplier/geo-report`, `/v1/supplier/zone-preview`, `/v1/supplier/warehouses/validate-coverage`, and `/v1/supplier/warehouse-loads` as one contract owned by `pegasus/apps/backend-go/proximityroutes/routes.go`; check the supplier portal coverage map and warehouse coverage-editor consumers before changing one path.
10. Supplier self-service gate: treat `/v1/supplier/configure`, `/v1/supplier/billing/setup`, `/v1/supplier/profile`, `/v1/supplier/shift`, `/v1/supplier/payment-config`, `/v1/supplier/gateway-onboarding`, and `/v1/supplier/payment/recipient/register` as one contract owned by `pegasus/apps/backend-go/supplierroutes/routes.go`; check the supplier profile, payment-config, setup-billing, and shift consumers before changing the route family.
11. Supplier warehouse-ops gate: treat `/v1/supplier/org/members*`, `/v1/supplier/staff/payloader*`, `/v1/supplier/warehouse-staff*`, `/v1/supplier/warehouses*`, `/v1/supplier/warehouses/{id}/coverage`, and `/v1/supplier/warehouse-inflight-vu` as one contract owned by `pegasus/apps/backend-go/supplierroutes/routes.go`; check the supplier org, staff, warehouses, coverage-editor, and factory-network-map consumers before changing the route family.
12. Supplier catalog-pricing gate: treat `/v1/supplier/products*`, `/v1/supplier/products/upload-ticket`, `/v1/supplier/pricing/rules*`, and `/v1/supplier/pricing/retailer-overrides*` as one contract owned by `pegasus/apps/backend-go/suppliercatalogroutes/routes.go`; check the supplier products, catalog, pricing, and retailer-overrides portal consumers before changing one path.
13. Supplier logistics gate: treat `/v1/supplier/picking-manifests*`, `/v1/supplier/manifests*`, `/v1/payload/manifest-exception`, `/v1/supplier/manifest-exceptions`, `/v1/supplier/fleet-volumetrics`, `/v1/supplier/dispatch-queue`, and `/v1/supplier/dispatch-preview` as one contract owned by `pegasus/apps/backend-go/supplierlogisticsroutes/routes.go`; check the supplier manifests, dispatch, orders, and payload-facing manifest exception consumers before changing one route.
14. Supplier insights gate: treat `/v1/supplier/country-overrides*`, `/v1/supplier/analytics/*`, `/v1/supplier/financials`, and `/v1/supplier/crm/retailers*` as one contract owned by `pegasus/apps/backend-go/supplierinsightsroutes/routes.go`; check the supplier country-overrides, analytics, dashboard, demand, and CRM consumers before changing one path.
15. Supplier operations gate: treat `/v1/supplier/fleet/*`, `/v1/supplier/fulfillment/pay`, `/v1/supplier/returns*`, `/v1/supplier/quarantine-stock`, and `/v1/inventory/reconcile-returns` as one contract owned by `pegasus/apps/backend-go/supplieroperationsroutes/routes.go`; check the supplier fleet, returns, depot-reconciliation, and supplier-side fulfillment consumers before changing one path.

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
4. Frontend-context validation for UI work: verify the backend contract is represented across every affected client surface for the role (web, desktop, Android, iOS, Expo as applicable), or explicitly gate hidden clients behind a rollout flag.
5. Cutover readiness: keep local code production-compatible so real server migration is wiring/config only.
6. Rollback readiness: additive schema and event changes, version-safe clients, and explicit rollback path.

## One-Eye Guard Suite (Mandatory On PR)
Run and pass all six guard scripts for pull requests:
1. `python3 pegasus/scripts/contract_guard_mcp.py --repo-root . --base-sha <base> --head-sha <head>`
2. `python3 pegasus/scripts/architecture_guard_mcp.py --repo-root . --base-sha <base> --head-sha <head>`
3. `python3 pegasus/scripts/design_system_guard_mcp.py --repo-root . --base-sha <base> --head-sha <head>`
4. `python3 pegasus/scripts/production_safety_guard.py --repo-root . --base-sha <base> --head-sha <head>`
5. `python3 pegasus/scripts/visual_test_intelligence_guard.py --repo-root . --base-sha <base> --head-sha <head>`
6. `python3 pegasus/scripts/security_guard.py --repo-root . --base-sha <base> --head-sha <head>`

`contract_guard_mcp.py`, `architecture_guard_mcp.py`, and `design_system_guard_mcp.py` enforce codebase-first MCP context discipline: trigger-scoped PRs must include stronger real-codebase coverage than context-doc coverage.

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
