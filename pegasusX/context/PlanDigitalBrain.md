# PlanDigitalBrain — Enterprise Planning Layer (o9-inspired)

Last updated: 2026-07-01

**Authority:** Subordinate to [`plan.md`](plan.md). Extends [`plan_90.md`](plan_90.md) with the full enterprise feature baseline, Kafka integration contract, edge-case gates, and UI labeling spec for the digital planning brain.

**Scope boundary:**
- **pegasusX** — single-supplier planning brain; all analytics scoped by `supplier_id`.
- **pegasus** (reference) — multi-tenant IBP, cross-supplier collaboration, federated EKG — deferred to platform layer.

**Audience:** Backend engineers, ai-worker, supplier portal + native, warehouse insight surfaces. Driver row stays execution-only; planning **consumes** telemetry and order outcomes.

---

## I. How pegasusX handles rollbacks and failed transactions today

Before wiring the planning brain to Kafka, every slice must inherit the operational ecosystem's durability contract. pegasusX does **not** use distributed two-phase commit or saga orchestrators. Instead it uses **Spanner ACID + transactional outbox + idempotent consumers + client session reconcile**.

### 1. Server-side: atomic write or full rollback

All mutating domain paths follow one pattern:

```mermaid
sequenceDiagram
  participant Client
  participant API as backend-go
  participant Spanner
  participant Relay as Outbox Relay
  participant Kafka
  participant Redis
  participant WS as WebSocket Hub

  Client->>API: POST/PATCH (optional Idempotency-Key)
  API->>Spanner: ReadWriteTransaction
  Note over Spanner: Domain row + OutboxEvents row<br/>in same txn — all or nothing
  alt txn error
    Spanner-->>API: rollback (no row, no event)
    API-->>Client: 4xx/5xx
  else txn success
    Spanner-->>API: commit
    API->>Redis: post-commit cache invalidation
    API->>WS: fanout envelope (after commit)
    API-->>Client: 2xx
    Relay->>Spanner: Fetch unpublished outbox
    Relay->>Kafka: Publish (at-least-once)
    Relay->>Spanner: MarkPublished
  end
```

**Key files:**
- `apps/backend-go/outbox/outbox.go` — transactional outbox; domain row + `OutboxEvents` in one RW txn
- `apps/backend-go/spannerutils/retry.go` — retries `Aborted` / `Unavailable` with bounded exponential backoff (max 5 attempts)
- `apps/backend-go/spannerutils/chunker.go` — large writes split into chunks; **partial commit is possible**; callers must be idempotent

When a logistics app drops mid-request:
- If the HTTP response never arrives → **no Spanner commit** → state unchanged on server
- If commit succeeded but response lost → client retries with same `Idempotency-Key` → cached 2xx replay, no duplicate side effects
- If commit failed → Spanner auto-rollback; idempotency key **released** on non-2xx so client can safely retry

### 2. HTTP idempotency guard

`apps/backend-go/idempotency/middleware.go`:
- Mutating requests with `Idempotency-Key` / `X-Idempotency-Key` are deduplicated by key + body hash
- **2xx responses** cached for 24h — replays return identical response
- **Non-2xx responses** release the key — safe to retry after connection drop
- **409** `idempotency_key_payload_mismatch` when same key, different body
- **409** `request_in_progress` when concurrent duplicate in flight

Planning brain ingestion APIs **must** require idempotency keys on all write paths.

### 3. Partial commit recovery (warehouse dispatch exemplar)

`docs/PARTIAL_DISPATCH_RECOVERY_SOP.md` documents the operational pattern when chunked writes succeed partially:
- HTTP **409** `dispatch_partial_commit` — earlier chunks committed, later chunk failed
- Idempotency key released — operator re-runs with scoped execute
- **Do not** cancel committed orders to "reset"
- Clients call `reconcileSession` before retrying

This is the canonical model for any planning-brain batch writer that uses `RunChunkedTransaction`.

### 4. Kafka consumer failure handling

`apps/backend-go/kafka/consumer.go` + `kafka/workerpool/workerpool.go`:
- Handler retries with jittered backoff (`MaxAttempts`)
- Exhausted retries → **DLQ** topic
- DLQ routing failure → **offset not committed** (`ErrSkipCommit`) — message reprocessed
- Cross-pod dedup: `kafka/redis_event_dedup.go` — Redis `SETNX` with 7d TTL

Planning brain consumers (ai-worker, notification dispatcher) must:
- Treat all handlers as **at-least-once** — dedupe by `event_id` or aggregate+sequence
- Never write Spanner directly from Kafka without idempotency guard
- Prefer **read-model projection** tables over mutating operational aggregates

### 5. Client-side: connection drop ≠ state corruption

Every role row implements **reconnect + reconcile**:

| Layer | Pattern | Authority |
|---|---|---|
| WebSocket | Exponential backoff reconnect; `onReconnect` callback | `DriverWebSocket.kt`, `WarehouseRealtimeClient.swift`, etc. |
| Session reconcile | Parallel refetch of server-authoritative snapshots **before** retrying queued mutations | `packages/api-client/session-reconcile.ts` |
| Pending sync | WorkManager / background flush for offline mutations | `PendingOrderSyncWorker.kt`, `pending-checkout.ts` |
| Stale/live labeling | UI shows `live` vs `stale` vs `reconnecting` connection state | All realtime clients |

`reconcileSession` endpoints per role (from `session-reconcile.ts`):
- **driver:** `/v1/fleet/orders`, `/v1/driver/manifest`
- **warehouse:** dispatch preview, dispatch locks
- **factory:** manifests
- **payload:** manifests, trucks
- **supplier:** dispatch preview, manifests
- **retailer:** active fulfillment, pending payments, tracking

**Planning brain client rule:** After WS reconnect, refetch planning dashboards from read APIs — never assume last-rendered forecast is current.

### 6. Payment and ledger: append-only, no in-place rollback

`docs/PAYMENT_EXCEPTION_SOP.md`, `docs/INCIDENT_RESPONSE_RUNBOOK.md`:
- Ledger rows are **immutable** — corrections use reversing entries
- Webhook replays collapse via `idempotency.Guard`
- No distributed rollback across payment gateway + Spanner

Promotion P&L simulation (Phase 3) must use **sandbox tables** or read-only projections — never mutate live ledger for what-if.

### 7. Summary: integration contract for the planning brain

| Concern | pegasusX pattern | Planning brain MUST |
|---|---|---|
| State mutation | Spanner RW txn + outbox in same txn | Same — no direct Kafka→Spanner writes |
| Event delivery | At-least-once via outbox relay | Consume idempotently; project to planning tables |
| Client disconnect | Reconcile then retry | Dashboard refetch on reconnect |
| Partial failure | 409 + idempotency release + SOP | Chunked writers document partial state |
| Rollback | Spanner txn abort (automatic) | No compensating transactions; use append-only audit |
| Realtime | Post-commit WS fanout | Subscribe to planning envelopes; never block execution path |

---

## II. Enterprise feature baseline (o9-inspired)

Maps user requirements to pegasusX delivery status. See [`plan_90.md`](plan_90.md) for anchor-level detail.

| Module | Enterprise capability | pegasusX 90d scope | Status |
|---|---|---|---|
| **IBP** | Unified financial + commercial + supply chain planning | Treasury (PX3-A2) + S&OP API only | **deferred** — full IBP is pegasus platform |
| **Demand Planning** | Multi-horizon AI forecasting, collaborative forecast | `DemandForecastBaseline` + ai-worker signals + warehouse/supplier reads | **partial** — baseline shipped; ML training Phase 3 |
| **Revenue Growth / Promotions** | Pre-event P&L, closed-loop eval, causal analysis | Promotion evaluator exists; P&L simulation not wired | **pending** — Phase 3 |
| **Commercial & Category** | CDT, cannibalization, halo | Not in pegasusX (supplier owns catalog) | **skipped** |
| **Control Tower** | Real-time visibility + supplier collaboration | Zone overrides + dispatch integration + WS fanout | **shipped** |
| **EKG** | Knowledge graph of supply chain entities | `GET /v1/supplier/knowledge-graph` — factories, warehouses, SKUs | **partial** — v2 adds drivers, retailers, orders |

---

## III. Custom architecture & real-time data flow

### Event topology

```mermaid
flowchart LR
  subgraph ingestion [High-concurrency ingestion]
    WH[Warehouse sensors/logs]
    SUP[Supplier APIs]
    DRV[Driver telemetry]
    API[backend-go ingest APIs]
  end

  subgraph durable [Durable path — execution truth]
    Spanner[(Spanner RW txn)]
    Outbox[OutboxEvents]
    Relay[Outbox Relay]
  end

  subgraph stream [Event streaming]
    KafkaMain[KAFKA_TOPIC_MAIN]
    KafkaPlan[planning.* topics]
  end

  subgraph brain [Planning brain — read side]
    AIW[ai-worker]
    Redis[(Redis cache)]
    PlanDB[(Planning projections)]
  end

  subgraph clients [Realtime clients]
    WS[WebSocket hubs]
    Portal[Supplier portal]
    Native[iOS / Android]
  end

  WH & SUP & DRV --> API
  API --> Spanner
  Spanner --> Outbox
  Outbox --> Relay --> KafkaMain
  KafkaMain --> AIW
  KafkaMain --> WS
  AIW --> PlanDB
  AIW --> Redis
  PlanDB --> Portal & Native
  Redis --> Portal & Native
  WS --> Portal & Native
```

### Kafka topic contract (planning brain)

**Execution path (existing):** `KAFKA_TOPIC_MAIN` — all transactional outbox events (`ORDER_STATUS_CHANGED`, `INVENTORY_SYNC_COMPLETE`, `planning.meio.recommendation.v1`, `DEMAND_BASELINE_UPDATED`, etc.)

**Planning ingestion (new, Phase 1):**

| Topic | Producer | Consumer | Payload contract |
|---|---|---|---|
| `planning.signal.ingest.v1` | backend-go ingest API | ai-worker | Normalized demand/inventory signals |
| `planning.forecast.request.v1` | supplier portal / API | ai-worker | Forecast run request (idempotent) |
| `planning.forecast.result.v1` | ai-worker | backend-go projector | Baseline write (idempotent by `forecast_run_id`) |

**Rules:**
1. Ingest APIs validate + publish to Kafka — **no Spanner write on hot ingest path** for analytics-only signals
2. Operational mutations still use outbox → `TopicMain` (never bypass)
3. Planning projections written via dedicated RW txn in `planning/` package with outbox for `DEMAND_BASELINE_UPDATED`
4. JSON schemas in `contracts/events.schema.json`; gate via `make gen-contracts-gate`

### Redis cache keys (planning)

| Key pattern | TTL | Content |
|---|---|---|
| `planning:seasonal:{supplier_id}:{template_id}` | 24h | Hard-coded seasonal templates |
| `planning:forecast:agg:{supplier_id}:{granularity}:{window}` | 15m | Regional/macro aggregates |
| `planning:promo:active:{supplier_id}` | 5m | Active promotion snapshot |
| `planning:scenario:{supplier_id}:{hash}` | 15m | Scenario sandbox result (existing) |
| `dedup:event:{event_id}` | 7d | Cross-pod Kafka dedup (existing) |

### Dynamic data granularity

API query params on all planning read endpoints:

| Granularity | Scope | Example endpoint |
|---|---|---|
| `macro` | Supplier-wide / country | `GET /v1/supplier/analytics/demand?granularity=macro` |
| `regional` | Warehouse zone / H3 region | `?granularity=regional&region_id=...` |
| `micro` | Per-retailer / per-store | `?granularity=micro&retailer_id=...` |

Backend serves pre-aggregated payloads from `DemandForecastBaseline` + Redis — never scans raw `Orders` on dashboard switch.

### WebSocket envelopes (planning)

Existing (live):
- `DEMAND_BASELINE_UPDATED`
- `planning.meio.recommendation.v1`
- `REPLENISHMENT_AUTO_APPROVED`
- `DISPATCH_ZONE_OVERRIDE`

New (Phase 2–3):
- `PLANNING_FORECAST_UPDATED` — range + confidence score
- `PLANNING_PROMO_SIMULATION_READY` — pre-event P&L result
- `PLANNING_CONFIDENCE_DOWNGRADED` — sparsity gate or seasonal fallback triggered

Fanout rooms: `supplier:{supplier_id}`, `warehouse:{home_node_id}` — via `notification_dispatcher.handlePlanningEvent`.

---

## IV. Edge cases, logic gates & front-end labeling

### PX91-A1: Data sparsity gate (zero/one order rule)

**Rule:** Block predictive analytics for any retailer with fewer than **2 completed orders**.

| Condition | Backend behavior | UI label |
|---|---|---|
| `completed_order_count < 2` | Return `403 forecast_insufficient_history` or `confidence: "blocked"` | **"Insufficient history"** badge; hide ML forecast |
| `completed_order_count == 2` | Allow baseline; cap confidence at 60% | **"Early signal"** badge |
| `completed_order_count >= 10` | Full model confidence | Standard confidence display |

**Implementation:**
- `planning/sparsity_gate.go` — `CanForecast(retailerID) (allowed bool, reason string)`
- Check in `DemandSignalProvider`, scenario sandbox, promo P&L
- SSMR: `PX_E2E_SPARSITY_GATE_OK`

### PX91-A2: Hard-coded seasonal fallbacks

**Rule:** Known seasonal surges use template curves, not ML extrapolation.

| Template ID | Window | Source |
|---|---|---|
| `holiday_peak` | Nov 15 – Jan 5 | Hard-coded multiplier curve |
| `summer_surge` | Jun 1 – Aug 31 | Regional default |
| `custom:{id}` | User-defined dates | Admin override (PX91-A4) |

When active template applies:
- ML model **disabled** for that window
- UI shows **"Seasonal template active"** chip
- Confidence floor: 75% (template-backed)

**Implementation:**
- `planning/seasonal_templates.go` + Redis cache
- `SeasonalTemplateOverrides` DDL (supplier_id, template_id, start_date, end_date)
- Warehouse + supplier dashboards read `baseline_source: "seasonal_template" | "ml" | "moving_average"`

### PX91-A3: Confidence labeling (UI task)

**All forecast displays MUST show ranges, not point estimates.**

| Element | Format | Example |
|---|---|---|
| Demand range | `{low} – {high} units` | `4,000 – 4,500 units` |
| Confidence score | Percentage + color band | `85% confidence` (green ≥80, amber 60–79, red <60) |
| Source badge | `ML` / `Baseline` / `Seasonal` / `Blocked` | Pill next to range |
| Staleness | `Updated {relative_time}` | `Updated 12m ago` when WS stale |

**Portal components:** `ForecastConfidenceCard.tsx` (new)
**Native:** `ForecastConfidenceView` on supplier iOS/Android MEIO/demand screens
**Contracts:** Extend `DemandForecastBaseline` in `packages/types` with `low_units`, `high_units`, `confidence_pct`, `baseline_source`, `blocked_reason`

### PX91-A4: Custom date range overrides

**Rule:** Admins define custom seasons that override global calendar.

- `POST /v1/supplier/planning/seasonal-overrides` — create custom window
- `GET /v1/supplier/planning/seasonal-overrides` — list active
- Isolated forecast model for that block (does not affect global baseline)
- UI: date picker on supplier portal Settings → Planning → Custom Seasons

---

## V. 90-day execution plan

Aligned with [`plan_90.md`](plan_90.md) waves. **Days 1–90 of plan_90 are largely shipped/partial** — this section extends into the next planning-brain tranche (PX91).

### Phase 1: Foundation & ingestion (Days 1–30) — **mostly shipped**

| Task | Owner | Status | Notes |
|---|---|---|---|
| EKG schema (entities + edges) | `planning/service.go` | **partial** | v1: factories, warehouses, SKUs |
| Kafka event contracts for planning | `contracts/events.schema.json` | **shipped** | MEIO, demand baseline events live |
| Ingest APIs → Kafka (no hot-path Spanner) | `planning/ingest.go` | **pending** | New Phase 1b |
| Spanner planning tables DDL | `schema/migrations/` | **shipped** | `20260630_plan90_planning_brain.ddl` |
| Outbox + idempotency on planning mutations | `planning/` | **shipped** | Same pattern as replenishment |

**Phase 1b deliverables (new):**
- [ ] `POST /v1/supplier/planning/signals/ingest` — validate, publish `planning.signal.ingest.v1`
- [ ] ai-worker consumer → `PlanningSignalProjections` table
- [ ] EKG v2: drivers, vehicles, retailers, active orders as nodes/edges
- [ ] SSMR: `PX_E2E_PLANNING_INGEST_OK`

### Phase 2: Baseline math & user views (Days 31–60) — **mostly shipped**

| Task | Owner | Status | Notes |
|---|---|---|---|
| Moving average / trend baseline | `predictivepush/`, ai-worker | **shipped** | Writes `DemandForecastBaseline` |
| Seasonal templates + custom date pickers | `planning/seasonal_templates.go` | **pending** | PX91-A2, PX91-A4 |
| Sparsity gate | `planning/sparsity_gate.go` | **pending** | PX91-A1 |
| Confidence labeling UI | supplier portal + native | **pending** | PX91-A3 |
| WebSocket + multi-tier dashboards | portal + native | **shipped** | MEIO, control tower live |
| WS reconnect reconcile for planning | `session-reconcile.ts` | **pending** | Add planning endpoints |

**Phase 2b deliverables (new):**
- [ ] Extend `packages/types` with forecast range + confidence fields
- [ ] `ForecastConfidenceCard` on supplier portal dashboard
- [ ] Native confidence views on iOS/Android
- [ ] `reconcileSession` planning endpoints for supplier role
- [ ] SSMR: `PX_E2E_SPARSITY_GATE_OK`, `PX_E2E_CONFIDENCE_LABEL_OK`

### Phase 3: AI training & advanced analytics (Days 61–90) — **partial**

| Task | Owner | Status | Notes |
|---|---|---|---|
| AI model training on historical pipeline | ai-worker | **pending** | Requires ingest pipeline green |
| Pre-event P&L simulation (promotions) | `promotion/` + `planning/` | **pending** | Sandbox read-only |
| Closed-loop performance evaluation | `planning/promo_eval.go` | **pending** | Compare predicted vs actual |
| Causal factor analysis | ai-worker | **deferred** | pegasus platform |
| Load testing + error handling | SSMR + k6 | **pending** | Shadow mode deploy |
| Shadow deployment (planning beside execution) | infra | **pending** | No execution path mutation |

**Phase 3 deliverables (new):**
- [ ] `POST /v1/supplier/planning/promotions/simulate` — pre-event P&L
- [ ] `GET /v1/supplier/planning/promotions/{id}/performance` — closed-loop eval
- [ ] `PLANNING_PROMO_SIMULATION_READY` WS envelope
- [ ] Load test: 1000 concurrent ingest + forecast reads
- [ ] SSMR: `PX_E2E_PROMO_PL_SIM_OK`, `PX_E2E_CLOSED_LOOP_EVAL_OK`

---

## VI. PX91 anchors (planning brain extension)

| Anchor | Scope | Phase | Status |
|---|---|---|---|
| `PX91-A1` | Data sparsity gate (≥2 completed orders) | 2 | **pending** |
| `PX91-A2` | Hard-coded seasonal fallbacks | 2 | **pending** |
| `PX91-A3` | Confidence labeling (range + score UI) | 2 | **pending** |
| `PX91-A4` | Custom date range seasonal overrides | 2 | **pending** |
| `PX91-B1` | High-concurrency ingest API → Kafka | 1 | **pending** |
| `PX91-B2` | ai-worker signal projector | 1 | **pending** |
| `PX91-B3` | EKG v2 (drivers, retailers, orders) | 1 | **pending** |
| `PX91-C1` | AI forecast model training pipeline | 3 | **pending** |
| `PX91-C2` | Pre-event promotion P&L simulation | 3 | **pending** |
| `PX91-C3` | Closed-loop promo performance eval | 3 | **pending** |
| `PX91-C4` | Planning session reconcile on reconnect | 2 | **pending** |
| `PX91-C5` | Shadow-mode load test + deploy | 3 | **pending** |

---

## VII. Ecosystem blast-radius checklist (every PX91 slice)

- [ ] `schema/spanner.ddl` + migration DDL for new planning tables
- [ ] Canonical owner package (`planning/`) + `supplierroutes/routes.go`
- [ ] Outbox in same RW txn as row write (planning projections only)
- [ ] Post-commit Redis invalidation (`planning:*` keys)
- [ ] WS fanout via `notification_dispatcher`
- [ ] `packages/types` + `packages/api-client` + idempotency keys
- [ ] `contracts/events.schema.json` via `make gen-contracts-gate`
- [ ] Role-row UI — supplier portal + iOS + Android; warehouse insight reads
- [ ] Focused `*_test.go` in touched packages
- [ ] SSMR markers in `e2e_plan90.go` + `contracts/ssmr_ecosystem_markers.json`
- [ ] `docs/ROLE_ROW_PARITY_MATRIX.md` PX91 rows
- [ ] **Never** mutate operational aggregates from Kafka consumer without idempotency
- [ ] **Never** block execution path on planning compute

---

## VIII. Explicit deferrals

| Feature | Reason |
|---|---|
| Full IBP / financial scenario planning | pegasus platform; treasury sufficient for pegasusX |
| Consumer Decision Trees / cannibalization | Retail assortment — pegasus retail scope |
| Cross-supplier collaboration workspace | pegasus multi-tenant |
| Causal factor decomposition (full) | Phase 3+ / pegasus AI suite |
| Retailer-facing planning surfaces | Retail row = order/track only |
| Distributed saga / compensating transactions | pegasusX uses outbox + idempotency instead |

---

## IX. Success criteria (planning brain v1)

| # | Criterion | Target |
|---|---|---|
| 1 | Kafka ingest handles 1000 concurrent signal updates without Spanner hot-path writes | Phase 1b |
| 2 | Retailer with <2 orders sees blocked forecast, not hallucinated demand | PX91-A1 |
| 3 | Holiday peak uses seasonal template, not ML extrapolation | PX91-A2 |
| 4 | All forecast UI shows range + confidence, never bare integer | PX91-A3 |
| 5 | Custom season override generates isolated forecast block | PX91-A4 |
| 6 | WS reconnect triggers planning dashboard reconcile | PX91-C4 |
| 7 | Pre-event promo P&L returns margin/volume projection without ledger mutation | PX91-C2 |
| 8 | Closed-loop eval compares actual vs predicted promo outcome | PX91-C3 |
| 9 | Planning brain runs shadow mode with zero execution regressions | PX91-C5 |
| 10 | All PX91 SSMR markers green under `make test-ssmr-infra` | Infra verify |

---

## X. Reference: durability patterns by failure mode

| Failure mode | What happens | Safe client action |
|---|---|---|
| HTTP timeout mid-mutation | Txn may or may not have committed | Retry with same `Idempotency-Key` |
| WS disconnect mid-dashboard | Server state unchanged; UI stale | Reconnect → `reconcileSession` → refetch |
| Kafka consumer crash mid-handler | Offset not committed; redelivery | Idempotent handler + Redis dedup |
| Spanner `Aborted` conflict | Auto-retry (≤5 attempts) | Transparent to client |
| Chunked write partial success | Earlier chunks committed | 409 + SOP; reconcile before retry |
| Duplicate webhook / event | Idempotency guard / SETNX dedup | No-op |
| Payment failure | Append-only ledger; no rollback | Reversing entry via treasury SOP |
| Planning ingest flood | Kafka buffers; ai-worker projects async | Dashboard shows `stale` until projection catches up |

---

## Related documents

- [`plan_90.md`](plan_90.md) — PX90 anchor status and shipped waves
- [`plan.md`](plan.md) — master execution plan (PX0–PX12)
- [`architecture.md`](architecture.md) — transactional outbox, WS relay, reliability matrix
- [`technology-inventory.md`](technology-inventory.md) — event catalog freeze, reliability acceptance
- [`docs/PARTIAL_DISPATCH_RECOVERY_SOP.md`](../docs/PARTIAL_DISPATCH_RECOVERY_SOP.md) — partial commit recovery
- [`docs/INCIDENT_RESPONSE_RUNBOOK.md`](../docs/INCIDENT_RESPONSE_RUNBOOK.md) — rollback and DLQ replay
- [`docs/ROLE_ROW_PARITY_MATRIX.md`](../docs/ROLE_ROW_PARITY_MATRIX.md) — client parity ledger
