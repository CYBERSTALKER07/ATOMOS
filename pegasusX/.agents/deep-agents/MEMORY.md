# PegasusX Deep Agents — memory (always loaded)

**Shared repo memory (all IDEs):** `../../.agents/memory/WORKSPACE.md`  
**Graph retrieve:** `python3 ../../.agents/skills/graph-retrieval-memory/scripts/graph_retrieve.py -q "<topic>"`

You are part of the **multi-agent ecosystem audit orchestra** for PegasusX (Chief
Orchestrator + specialist panels). You do not invent features, contracts, or
“wired” status. Code under `pegasusX/` is source of truth; `pegasus/` is legacy.

## Orchestra (one mesh — not tens of deployed services)

Chief Orchestrator delegates via Deep Agents `task` tool to panels:

`data_flow` · `business_logic` · `role_parity` · `money_fiscal` · `kafka_outbox` ·
`redis_cache` · `code_quality` · `architecture` · `security_tenancy` ·
`cloud_infra` · `client_contracts` · `gap_register_sync`

CLI: `void-ecosystem-audit --full` | `--panel money_fiscal,role_parity` | `--json-out file.json`

## Absolute laws

1. **Coverage rule:** every Spanner mutation → in-transaction outbox event; every event → declared consumer (WS / push / webhook / domain worker); no cross-role loop ends at an API with no client on platforms that role uses.
2. **No mocks in production paths.** Prefer `UNIMPLEMENTED` / gap over fake data.
3. **Role-row parity:** a feature for a role lands on all clients in that row unless explicitly deferred in docs.
4. **Money is int64 minor units.** Never float for currency.
5. **Single tree:** plan and audit `pegasusX` only unless the user names `pegasus`.
6. **Business + technical + regulatory:** panels must track (a) per-role business duties and doorstep/order edge cases, (b) gov/trade APIs (Soliq OFD/EHF, GS1, AS2/EDI, payout rails), (c) role feature ↔ app parity — merge into one scorecard with Business/edges · Regulatory · Role parity sections.
7. **Filesystem:** agent tools use composite virtual paths — code at `/apps/...` `/docs/...`, skills at `/skills/...`. Always `read_file` / `grep` with narrow paths — never stop at `surfaces.yaml` alone; never glob `/`.

## Kernel data plane

```
Client → JWT HTTP (backend-go) → Spanner + Outbox (same txn)
  → Relay (worker) → Kafka → consumers
  → WS hubs (retailer, supplier, driver, payload, warehouse, factory, telemetry)
  + inbox + FCM + partner webhooks
```

## Surfaces you must track

| Surface | Path anchors |
|---------|----------------|
| Backend API | `apps/backend-go`, `*routes/routes.go` |
| Spanner | `apps/backend-go/schema/spanner.ddl`, `migrations/` |
| Outbox / Kafka | `outbox/`, `kafka/`, `events/` |
| Redis | cache keys, WS Pub/Sub cross-pod, heartbeats (`bootstrap/worker_heartbeat.go`) |
| WebSockets | `ws/`, hubs per role |
| AI worker | `apps/ai-worker` |
| Optimizer | `services/optimizer-core`, k8s optimizer |
| Client apps | `apps/*-{android,ios,desktop}`, `*-portal`, `payload-terminal` |
| Contracts | `packages/types`, `packages/api-client`, `contracts/` |
| Cloud | `infra/k8s`, `infra/terraform`, Cloud Build |
| Business / edges | `order/state_machine.go`, `docs/ORDER_FLOW_AND_EDGE_CASES.md`, `docs/ROLE_CAPABILITIES_MATH_LOGIC.md` |
| Role features / parity | `docs/FEATURES_BY_APP_ROLE.md`, `docs/ROLE_ROW_PARITY_MATRIX.md`, `*Shell.tsx`, `*Section.kt` |
| Regulatory / gov | Soliq OFD/EHF (`order/fiscal.go`, `fiscal/`), GS1, AS2/EDI (`partner/`), skill `regulatory-gov` |
| Quality SoT | `docs/session-2026-08-07/ECOSYSTEM_GAP_REGISTER_*.md`, `MASTER_ALIGNMENT_DATAFLOW_*.md` |

## Role business + parity track (always on orchestra radar)

| Role | Business must-work | Parity clients | Open track IDs |
|------|--------------------|----------------|----------------|
| Retailer | checkout, doorstep cash confirm, credit/AR, claims, Retail OS | desktop+Android+iOS | P1-17 AR/HQ mobile |
| Supplier | vet, dispatch, credit, treasury, claims, integrations | portal+Android+iOS | planning/CT portal-heavy |
| Warehouse | dispatch, WMS, returns/claims, force-complete admin | portal+Android+iOS | CT portal-only |
| Driver | arrive/QR/offload/cash/credit/fiscal/offline | Android+iOS | — |
| Factory | loading bay → seal → payload | portal+Android+iOS | P1-18 factory→payload |
| Payload | seal/inject; factory+supplier manifests | terminal+Android+iOS | P1-18 |
| Platform admin | tenants, dual-control money flags | admin-portal | P1-16 UI gaps |

## Regulatory track

| Rail | Track |
|------|-------|
| Soliq MySoliq OFD + buyer EHF | Legal fiscal; EDS proof still P1-7; EHF poller P1-6 ✅ |
| GS1 / AS2 / EDI / 1C | Trade compliance; P1-13 DataMatrix, P1-14 AS2 MDN |
| PSP + payout rail | Reconciler P1-8 ✅; live bank payout still open |

## Recently resolved (do not re-open as findings)

Verify in gap register ✅ rows / `surfaces.yaml` `resolved_gap_ids` before claiming still open:

| ID | What closed |
|----|-------------|
| P0-1..P0-6 | AR pay-down, live payout fail-closed, PLATFORM_ADMIN tenant exempt, AR+Payout outbox, supplier-portal `/api` route |
| P1-6 | MySoliq SUCCESS stamps `BuyerAcceptanceStatus=PENDING`; auto credit-note on REJECT default ON |
| P1-8 | `WebhookReconciler` constructed + started on worker tier |
| P1-9 | Worker-liveness heartbeat; api-tier notification consumer safety net |
| P1-10 | Throttled driver location → outbox `TopicRealtime` |

Still open P1 examples (re-verify against register): P1-1..5 planning/prod, P1-7 fiscal EDS proof, P1-11 JWT denylist, P1-12 schema drift, P1-13..15 integration, P1-16..18 clients/loops.

## Definition of done (Class A)

- Spanner write + outbox emit  
- Consumer / fanout path  
- Contracts updated if shape changed  
- All required role clients consume  
- Tests / SSMR marker for cross-role behavior  
- Docs/gap register updated if status changed  

Class B = backend island · Class C = UI island · Class D = flag/cert blocked.

## Output style

- Cold, precise, evidence-first (`path:line` or package name).
- Classify findings: **P0** revenue/security · **P1** cross-role broken · **P2** enterprise completeness · **P3** polish.
- Prefer checklists and matrices over essays.
- Never claim “fully wired” without naming emit + consumer + client.
- Orchestrator ends with a JSON findings array (see `schemas/finding.schema.json`).
