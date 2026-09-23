# Execution Plan — pegasus.x Codebase Hardening

## Overview
Autonomous audit and surgical hardening of `pegasus.x` against the Universal Engineering Doctrine (Google Principal Engineer & Limitless Hacker standards).

## Phase 0: Survey & Codebase Assessment (Current)
- Launch 3 specialized Explorers concurrently:
  - **Explorer 1 (In-Memory Fallback Hunter)**:
    - Target: `backend/internal/consignment`, `backend/internal/rebate`, `backend/internal/payout`, `backend/internal/wmsops`, and all other packages in `backend/internal/`.
    - Task: Locate all `MemoryRepository` definitions, silent fallbacks when DB pool is nil, mock seeds in production packages, constructor signatures (`NewService`, `NewRepository`, `NewPostgresRepository`), and affected tests.
  - **Explorer 2 (Currency & Domain Purity Auditor)**:
    - Target: All domain models, handlers, services, and DTOs in `backend/internal/`.
    - Task: Identify any floating-point currency calculations (`float32`, `float64`), verify strict 64-bit integer tiyin minor unit math (`int64`), find any naive CRUD endpoints lacking state machines, concurrency checks (`FOR UPDATE`, optimistic versioning), or validation guards.
  - **Explorer 3 (Real-Time Monotonic Pipeline Auditor)**:
    - Target: `backend/internal/outbox`, `backend/internal/realtime`, WebSocket hubs, client desktop app listeners.
    - Task: Verify outbox pairing in `pgx.Tx`, outbox relay polling (`FOR UPDATE SKIP LOCKED`), publishing to Redis 7 Streams (`XADD`), WebSocket Hub monotonic `RealtimeEnvelope` frames (`seq`, dual `event_type`/`type`), and desktop frontend subscription handling.

## Phase 1: Synthesis & Milestone Refinement
- Aggregate reports from all 3 Explorers.
- Update `PROJECT.md` and `progress.md` with exact file:line citations and precise remediation scopes.

## Phase 2: Milestone 1 — Purge In-Memory Fallbacks & Enforce Fail-Closed Constructors
- Dispatch Worker to:
  - Refactor `internal/consignment/service.go`, `internal/rebate/service.go`, `internal/payout/repository.go`, `internal/wmsops/repository.go`.
  - Require valid `*db.Pool` in constructors; return error or fail closed if nil.
  - Move in-memory mock repositories into corresponding `_test.go` files.
  - Fix any broken caller or test setup.
- Dispatch Reviewers & Challenger to verify.

## Phase 3: Milestone 2 — Currency Arithmetic & Domain State Machine Purity
- Dispatch Worker to:
  - Refactor any identified floating-point currency usage to `int64` tiyins.
  - Ensure all financial mutations use domain state machines and concurrency controls.
- Dispatch Reviewers & Challenger to verify.

## Phase 4: Milestone 3 — Cross-Role Real-Time Monotonic Pipeline Parity
- Dispatch Worker to:
  - Ensure all mutating state transitions pair entity updates and outbox events in the same `pgx.Tx`.
  - Ensure outbox relay uses `FOR UPDATE SKIP LOCKED` and Redis 7 Streams `XADD`.
  - Ensure WebSocket Hub broadcasts monotonic `RealtimeEnvelope` frames (`seq`, `event_type`, `type`).
  - Ensure desktop client handles real-time invalidation.
- Dispatch Reviewers & Challenger to verify.

## Phase 5: Milestone 4 — Comprehensive Automated Test Suite & Race Verification
- Dispatch Worker / Test Runner to execute `go test -v -race ./...` across all packages in `pegasus.x/backend`.
- Verify scale benchmarks (1,000-order H3 clustering, 100-order dispatch, fleet breakdown rescue hot-swap).
- Dispatch Reviewers to independently audit test logs and race detector results.

## Phase 6: Victory Synthesis & Sentinel Notification
- Synthesize all results into `handoff.md`.
- Send comprehensive completion report to Sentinel (`90867845-3df7-435e-82d3-3e3c0e0e9c8e`).
