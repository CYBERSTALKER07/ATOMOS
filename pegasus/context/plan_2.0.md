# Pegasus OR-Tools Deployment Plan 2.0

Last updated: 2026-05-19
Owner scope: `pegasus/` only

## 1. Objective

Deploy OR-Tools in V.O.I.D. as a production-grade optimization capability without introducing API-path latency, data corruption, or runtime starvation.

Primary target:
- Keep solver execution asynchronous and isolated.
- Keep all optimization persistence additive and outbox-safe.
- Keep runtime behavior deterministic under load.

## 2. Non-Negotiable Failure Modes And Controls

### 2.1 Pitfall: Solver in synchronous API path

Do not run OR-Tools in request-response handlers for checkout, route execution, or allocation endpoints.

Control:
- API writes a pending optimization task in Spanner and emits Kafka request token.
- Worker consumes token and executes solver out-of-band.
- API returns immediately after persistence.

### 2.2 Pitfall: float to int truncation

Do not cast float to int directly for CP-SAT constraints.

Control:
- Use global scalar `SCALE_FACTOR=10000`.
- Convert `scaled = round(raw * SCALE_FACTOR)` before solver.
- Convert back with `raw = scaled / SCALE_FACTOR` at output boundaries only.
- Keep conversion symmetric and centralized.

### 2.3 Pitfall: combinatorial explosion

Do not run unconstrained solver loops.

Control:
- Set deterministic `max_time_in_seconds`.
- Set explicit acceptable optimality gap where applicable.
- Return best feasible solution within budget window.

### 2.4 Pitfall: native thread starvation

Do not allow OR-Tools default worker expansion to host core count.

Control:
- Clamp internal solver workers (`num_search_workers`) per job.
- Keep per-instance worker counts bounded (typically 2-4).

## 3. Mathematical Contract (Canonical)

Objective:

min sum(i in I) sum(j in J) c_ij * x_ij

Subject to:

sum(j in J) x_ij <= 1     for all i in I
sum(i in I) w_i * x_ij <= M_j   for all j in J
x_ij in {0,1}

Where:
- `I`: pending order/manifests
- `J`: eligible supplier nodes (factory/warehouse)
- `c_ij`: scaled integer route/allocation cost
- `w_i`: scaled integer weight/volume requirement
- `M_j`: scaled integer node capacity
- `x_ij`: binary assignment decision

## 4. Runtime Topology (Pattern-A)

1. Core platform writes optimization request state in Spanner.
2. Outbox relay emits request payload to Kafka optimization topic.
3. Go worker consumes Kafka event.
4. Go worker calls Rust gRPC sidecar (`optimizer-core/server-rust`) over localhost.
5. Sidecar executes VRP/CP-SAT with deterministic mapping/scaling.
6. Worker writes `OPTIMIZATION_SOLVED` to Spanner `OutboxEvents` only.
7. Downstream consumers project decisions into operational read models.

Canary note:
- Python sidecar remains available only for dual-run parity checks during rollout.

## 5. Canonical Repository Surfaces

### 5.1 Sidecar and contracts
- `pegasus/services/optimizer-core/proto/optimizer_core.proto`
- `pegasus/services/optimizer-core/server-rust/Cargo.toml`
- `pegasus/services/optimizer-core/server-rust/build.rs`
- `pegasus/services/optimizer-core/server-rust/src/main.rs`
- `pegasus/services/optimizer-core/server-rust/src/service.rs`
- `pegasus/services/optimizer-core/server-rust/src/solver/vrp.rs`
- `pegasus/services/optimizer-core/server-rust/src/solver/cpsat.rs`
- `pegasus/services/optimizer-core/server-rust/src/mapping.rs`
- `pegasus/services/optimizer-core/server-rust/src/scaling.rs`
- `pegasus/services/optimizer-core/Dockerfile.rust`
- `pegasus/services/optimizer-core/server/main.py` (canary parity baseline only)

### 5.2 Go adapter and ingestion
- `pegasus/services/optimizer-core/adapters/go/cmd/optimizer-worker/main.go`
- `pegasus/services/optimizer-core/adapters/go/internal/adapter/worker.go`
- `pegasus/services/optimizer-core/adapters/go/internal/adapter/translate.go`
- `pegasus/services/optimizer-core/adapters/go/internal/optimizergrpc/client.go`
- `pegasus/services/optimizer-core/adapters/go/internal/model/types.go`

### 5.3 Event and persistence integration
- Spanner `OutboxEvents` write path only for solved payload persistence.
- No direct inventory or manifest table mutations from optimizer worker.

## 6. Implementation Wrappers

### 6.1 Solver wrapper requirements
- Deterministic UUID <-> index map per request.
- Single source of truth scaling module.
- Bounded solver parameters injected for every run.
- Feasible-or-timeout semantics with explicit result status.

### 6.2 Worker wrapper requirements
- Bounded retry with attempt counters.
- Exponential backoff with jitter.
- Dead-letter route for malformed or irrecoverable jobs.
- Idempotent solved-event writes.

### 6.3 API wrapper requirements
- Request path records task and returns fast.
- No blocking wait for optimization completion.
- Trace ID propagation across API -> Kafka -> worker -> outbox.

## 7. Phased Delivery Plan

### Phase 0: Contract freeze
- Freeze `optimizer_core.proto` envelopes.
- Freeze scaling and mapping conventions.
- Define timeout budgets by use-case class.

Exit gate:
- Contract and scaling docs approved.

### Phase 1: Async ingestion path
- Ensure Kafka request topic is authoritative ingress.
- Ensure worker consumes only from Kafka and not API direct calls.

Exit gate:
- Load test confirms API p95 unchanged during optimization bursts.

### Phase 2: Deterministic compute safety
- Enforce global scaling conversion path.
- Enforce deterministic UUID index mapping.
- Enforce bounded solver parameters.

Exit gate:
- Golden test vectors reproduce identical assignment outputs.

### Phase 3: Persistence and event safety
- Persist solved outputs only via outbox writes.
- Verify no direct operational table mutation from worker path.
- Confirm downstream projection receives `OPTIMIZATION_SOLVED`.

Exit gate:
- Replay test confirms idempotent outcomes.

### Phase 4: Concurrency and resilience hardening
- Clamp solver threads.
- Clamp worker concurrency per instance.
- Validate backoff and DLQ behavior under fault injection.

Exit gate:
- No sustained CPU starvation and no retry storms under chaos scenarios.

### Phase 5: Production cutover
- Progressive rollout with canary.
- Metrics and alerts active before 100 percent rollout.
- Rollback is config-level (disable optimization worker intake).

Exit gate:
- SLO and guardrail thresholds pass for canary and full rollout.

## 8. SRE Verification Matrix

### 8.1 Performance
- API latency does not regress under solver load.
- Worker throughput remains stable at expected partition rates.

### 8.2 Correctness
- Scaling roundtrip correctness on constrained datasets.
- Capacity constraints hold for all accepted solutions.

### 8.3 Reliability
- Retry budgets and DLQ routes validated.
- Worker restarts do not duplicate solved events.

### 8.4 Safety
- Outbox-only mutation invariant verified.
- No direct Inventory/Manifest mutations from optimization adapter code.

## 9. Operational KPIs

- Optimization request accepted-to-solved latency.
- Feasible-solution success rate within timeout window.
- Solver timeout ratio by workload class.
- DLQ rate and top failure reasons.
- CPU and thread saturation per worker pod.
- Outbox publish lag for solved events.

## 10. Tactical Pitfall Checklist (Pre-merge)

- [ ] No solver call in synchronous API path.
- [ ] No raw float-to-int cast in constraint setup.
- [ ] Solver timeout and worker limits are explicitly set.
- [ ] Solver worker thread count is clamped.
- [ ] Retry policy is bounded with backoff + jitter.
- [ ] Solved result persistence uses outbox only.
- [ ] No direct Inventory/Manifest mutations in optimizer worker path.

## 11. Plan Reconciliation Template

Use this status model for every execution chunk:

- implemented:
- in progress:
- blocked:
- deferred:

Required evidence per chunk:
- files changed
- contracts impacted
- validation commands run
- rollback path
