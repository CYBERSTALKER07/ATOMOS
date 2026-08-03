# Big-Platform Baseline Plan (O9 / Blue Yonder / Manhattan / Kinaxis + PegasusX)

**Status:** planning baseline (not fully implemented)  
**Date:** 2026-07-29  
**Spine (must not break):** Spanner + outbox + Kafka + Redis + WS + role clients + **int64 minor units** + canonical order status machine  

## Intent

Make **enterprise planning + execution depth** the default baseline (O9 / Kinaxis / Blue Yonder / Manhattan class), keep every **PegasusX differentiator** first-class, then add solutions for **cash-heavy, multi-supplier, fiscalized emerging-market B2B** that incumbents do not solve cleanly.

## How to read this tree

| Folder | Audience | Contents |
|--------|----------|----------|
| [`foundations/`](./foundations/) | Architects | Canonical model, multi-horizon planning, intelligence, labor/capacity |
| [`planning/`](./planning/) | Planning / product | Demand sensing, MEIO, scenarios, supplier collaboration |
| [`execution/`](./execution/) | Ops / WMS-TMS | Warehouse depth, TMS, labor management |
| [`last-mile/`](./last-mile/) | Driver / exception ops | Shop-closed, partial offload, damage, rescue, proximity settlement |
| [`collaboration/`](./collaboration/) | Network visibility | Control tower, exception rooms, scorecards |
| [`differentiators/`](./differentiators/) | Product strategy | Features no major suite owns cleanly |
| [`regulatory/`](./regulatory/) | Legal / finance / compliance | Soliq/EHF, tax regimes, audit, freezes, labor prep |
| [`technical/`](./technical/) | Engineering | Spine laws, schema, APIs, state machines, workers, role-row, edge matrix |
| [`phases/`](./phases/) | Program management | Phase 1–3 sequencing and DoD |

## Non-negotiable laws (every feature)

1. **Integer minor units only** — no float money.  
2. **Mutation** = Spanner RW txn + **outbox in same txn** + post-commit cache invalidation.  
3. **Idempotency keys** on all mutating paths.  
4. **Role-row parity** (portal/desktop + Android + iOS for that role) unless explicitly deferred in `context/parity-ledger.md`.  
5. **Status transitions** only via canonical `order.ValidateStatusTransition`.  
6. **Edge-case matrix** required: nulls, cancel, offline, concurrent, overflow, fiscal fail, claim window.

## Related existing docs

- [`../ECOSYSTEM_FEATURES_BY_ROLE.md`](../ECOSYSTEM_FEATURES_BY_ROLE.md) — current feature map by role  
- [`../CLAIM_ROLE_ROW.md`](../CLAIM_ROLE_ROW.md) — claims contracts & clients  
- [`../GCP_MIGRATION_CHECKLIST.md`](../GCP_MIGRATION_CHECKLIST.md) — cloud wiring status  

## Current platform facts (baseline to build on)

- Live SSMR: Spanner, Redis, Strimzi Kafka, GKE, Firebase Auth/FCM path, Maps Places geocode, `FISCAL_PROVIDER=PEGASUS` (Soliq deferred).  
- Order fiscal hard-gate (ADR-009): capture → `FISCALIZING` → `COMPLETED` / `FISCAL_FAILED`.  
- Claims: order-line pricing, 48h window, LEDGER_ONLY / STORE_CREDIT / GATEWAY_REFUND, INTERNAL cash clawback.  
- Dispatch: optimizer-core (OR-Tools) + heuristic fallback; freeze locks exist.  
- Explicit **Payload** role + seal + inject. Offline driver hashes exist.

## Next detail slices (pick one)

When implementing, request detail for:

1. **Schema** (`technical/schema-sketch.md` → Spanner DDL PR)  
2. **API contracts** (`technical/api-contracts-sketch.md` → types + routes)  
3. **State machines** (`technical/state-machines.md` → code owners)  
4. **Worker design** (`technical/workers-kafka.md` → consumer groups + idempotency)
