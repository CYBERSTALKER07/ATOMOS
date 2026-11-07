# Chaos Resilience & Failure Mode Validation Plan

## TL;DR
Harden the PegasusX distributed systems architecture against cascading failures, upstream timeouts, and broker degradation. Integrates outbound `circuit.Breaker` defenses into `dispatch/optimizerclient` (preventing 8-second solver stalls during optimizer overload) and `soliq` (shielding fiscal transactions during government OFD gateway degradation). Implements comprehensive chaos and failure-mode test suites across Circuit Breakers, Redis Failover, and Kafka Outbox Poison Pill Dead-Letter isolation. Ships a dedicated automated gate `scripts/ci_chaos_resilience.sh` wired into `Makefile`.

## Objective
Guarantee platform self-healing and zero cascading downtime across critical dependency outages:
1. **Optimizer Overload & Timeouts**: Fail-fast in <1ms to legacy Clarke-Wright/H3 binpacking when the Python solver experiences 504 timeouts or 5xx crashes, automatically recovering once healthy.
2. **Soliq Tax Gateway Degradation**: Fast-fail into the offline asynchronous fiscal retry queue (`SOLIQ_CIRCUIT_OPEN`) when tax authority endpoints hang or error, protecting customer checkout flows.
3. **Redis Degradation**: Verify in-memory fallback and safe fail-closed security invariants during Redis master/replica failovers.
4. **Kafka Outbox Poison Pill Isolation**: Guarantee that un-publishable poisoned events are quarantined to `OutboxDeadLetters` after exhausted retries without blocking healthy events in the drain loop.
5. **Unified Chaos Gate**: Provide an automated, race-detected gate (`make chaos-resilience-gate`) verifying all resilience contracts.

## Non-goals
- Replacing Kafka or Redis infrastructure providers.
- Changing VRP solver algorithm mathematics or order state machine transitions.
- Requiring live cloud connectivity for unit and chaos verification (all tests remain self-contained and fast).

---

## TODOs

### Wave 1: Outbound Circuit Breakers for Optimizer & Soliq OFD
- [x] **Task 1: Optimizer Client Circuit Breaker Integration**
  - Add `breaker *circuit.Breaker` and `WithBreaker(*circuit.Breaker)` to `apps/backend-go/dispatch/optimizerclient/client.go`.
  - Wrap solve execution in `c.breaker.Do()`, failing fast with `circuit.ErrUpstreamUnavailable` when open.
  - Wire `Optimizer: circuit.New("optimizer", cfg)` in `apps/backend-go/bootstrap/outbound_circuit.go` and `apps/backend-go/bootstrap/app.go`.
  - Write unit/chaos tests in `apps/backend-go/dispatch/optimizerclient/client_test.go` covering closed, open, fail-fast, half-open, and recovered states.
- [x] **Task 2: Soliq OFD Tax Gateway Circuit Breaker Integration**
  - Add `Breaker *circuit.Breaker` to `apps/backend-go/soliq/client.go` (`SoliqConfig` & `NewClientWithBreaker`).
  - Wrap HTTP requests in `c.breaker.Do()`, mapping open breaker to `SOLIQ_CIRCUIT_OPEN` with non-blocking offline queue return.
  - Wire `Soliq: circuit.New("soliq", cfg)` in `apps/backend-go/bootstrap/outbound_circuit.go`.
  - Write unit/chaos tests in `apps/backend-go/soliq/client_test.go` testing circuit tripping on 5xx/timeouts and auto-recovery.

### Wave 2: Redis Failover & Outbox Poison Pill Isolation Tests
- [x] **Task 3: Redis Degradation & Cache Chaos Tests**
  - Verify circuit breaker trips and in-memory/cache-miss fallback behaves correctly without panics or memory leaks in `apps/backend-go/cache/redis_circuit_test.go`.
- [x] **Task 4: Kafka Outbox Poison Pill Isolation Chaos Tests**
  - Add test in `apps/backend-go/outbox/relay_test.go` verifying that repeated broker rejects cause dead-lettering to `OutboxDeadLetters` after `MaxTotalAttempts` without halting subsequent valid events.

### Wave 3: Automated Chaos Gate Script & Integration
- [x] **Task 5: Dedicated Chaos Resilience Gate Script & Makefile Target**
  - Create `scripts/ci_chaos_resilience.sh` executing all chaos and resilience test suites under `-race`.
  - Wire `chaos-resilience-gate` into `Makefile`.
  - Verify `make chaos-resilience-gate` passes cleanly.
