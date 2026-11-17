# PegasusX Autonomous Agent Operating Manual & Session Protocols

> **Scope**: Universal Developer & AI Agent Guide for PegasusX  
> **Repository Roots**: `/Users/shakhzod/Desktop/pegasusx` & `/Users/shakhzod/Desktop/V.O.I.D/pegasusX` (Strict Parity)  
> **Authoritative Invariants**: Defined in [`agents.md`](../agents.md)

---

## 1. Master Session Initialization Message (First Best Message)

When starting a new session with an AI coding assistant (Cursor, Claude Code, Antigravity, Gemini), copy and paste this exact prompt as your **first message**:

```markdown
You are operating as a Staff Principal Engineer and Autonomous Agent on the PegasusX monorepo (/Users/shakhzod/Desktop/pegasusx).

Before touching any code or making edits, execute this mandatory Session Bootstrap Protocol:

1. READ THE CONSTITUTION & OPERATING DOCTRINE:
   - Read `agents.md` in full. Strictly enforce Anti-Reward-Hacking Laws 0.1–0.5.
   - Internalize the 4-Tier Solution Discovery Waterfall:
     [Tier 1: Web & RFC Research] ➔ [Tier 2: Proven Open-Source / Commercial Tooling] ➔ [Tier 3: Reverse-Engineer Big Tech] ➔ [Tier 4: Enterprise-Grade Innovation].
   - Enforce the Boy Scout Modernization Standard across all new and existing code (zero naive O(N²) loops, zero string-concatenated Spanner queries).
   - Enforce Zero Shared UI Packages across apps (local component ownership only).

2. ORIENT ON CURRENT ARCHITECTURE & IN-FLIGHT GOALS:
   - Inspect `PROJECT.md` and `task_plan.md` for active milestones, route contracts, and feature inventories across the 9 Operational Roles.
   - Use the Code Graph (`codegraph_explore` or `/graph-rag explore <symbol>`) to verify cross-boundary seams (Client API ➔ Backend Route ➔ Spanner Table ➔ Outbox ➔ Kafka ➔ WebSocket).

3. AVAILABLE SPECIALIZED SKILLS:
   - Master Multi-App Sync: `/pegasusx-deep-swarm`
   - Surface Specialists: `/surface-backend-go`, `/surface-web-desktop`, `/surface-android-kotlin`, `/surface-ios-swift`, `/surface-optimizer-solver`, `/surface-infra-sre`, `/surface-ui-ux-designer`

4. MANDATORY STOP CONDITION:
   Execute `bash scripts/ci_anti_theatre_gate.sh` to verify baseline repository hygiene. Then output a concise 4-bullet Readiness Dossier:
   - [ ] Constitution & Anti-Reward-Hacking Status
   - [ ] Code Graph / Memgraph Connectivity Status
   - [ ] Active Milestone Target from `task_plan.md`
   - [ ] Parity & Gate Baseline
   
Do NOT edit any production code, do NOT create files, and do NOT launch subagents yet. Output the dossier and await my first task.
```

---

## 2. The 4-Tier Solution Discovery Waterfall

Every function, algorithm, data structure, database query, and infrastructure component—for both **new code** and **existing code**—must follow this research waterfall before implementation:

```
[Tier 1: Deep Web & Literature Research]
       │ (State-of-the-art RFCs, ACM papers, production benchmarks)
       ▼
[Tier 2: Proven Open-Source & Commercial Tooling]
       │ (Paid/unpaid battle-tested libraries, battle-hardened algorithms)
       ▼
[Tier 3: Reverse-Engineer Big Tech Architecture]
       │ (Analyze how Google, AWS, Uber, Stripe, Palantir solve it at scale)
       ▼
[Tier 4: Enterprise-Grade Big Tech Innovation]
         (Design custom mathematically-proven, zero-contention algorithm)
```

1. **Tier 1 (Literature & RFCs)**: Benchmark against official RFCs, whitepapers, and cloud architecture guides. Never guess cryptographic handshakes, distributed consensus, or algorithmic complexity.
2. **Tier 2 (Battle-Tested Open Source)**: Adopt proven tools (e.g. Google OR-Tools for VRP, Uber H3 for spatial indexing, Pebble/RocksDB for LSM storage, Roaring Bitmaps) instead of reinventing solved wheels.
3. **Tier 3 (Reverse-Engineer Big Tech)**: If no drop-in library matches our constraints, reverse-engineer production patterns from market leaders:
   - *Palantir Foundry*: Copy-on-Write scenario branching, ontology object blast-radius graphs.
   - *Uber / DoorDash*: CVRPTW dynamic dispatching, geofence hysteresis, driver payout waterfalls.
   - *Amazon / Flexport*: Reverse-stop LIFO loading, 3D crushability bin-packing, SSCC-18 pallet serialization.
   - *Stripe / Adyen*: Dual-entry immutable ledgers, idempotent charge ratchets, split-tender waterfalls.
4. **Tier 4 (Proprietary Big-Tech Innovation)**: For bespoke Central Asian logistics constraints without industry precedent, design custom algorithms with explicit mathematical complexity bounds ($O(1)$ lookups, $O(N \log N)$ pipelines, zero $O(N^2)$ loops) and formal proofs of atomicity.

---

## 3. Specialized Skills Catalog

| Skill Command | Surface / Scope | Primary Responsibilities |
| :--- | :--- | :--- |
| **`/pegasusx-deep-swarm`** | Monorepo Omni-Sync | Master multi-agent orchestrator. Dispatches parallel swarms (Backend, Android, iOS, Web, Optimizer, SRE, UI/UX), coordinates 5-seam synchronization, and enforces zero-drift. |
| **`/surface-ui-ux-designer`** | UI/UX & Motion | Mobbin pattern grounding (Amazon Flex, Square POS, Uber Driver), 21st.dev component adaptation into local `components/ui/`, Pegasus Monochrome design system, role-specific ergonomics. |
| **`/surface-backend-go`** | Backend Fleet | Go 1.25+, Cloud Spanner transactional consistency (`ReadWriteTransaction` retry idempotency), Transactional Outbox relay to Kafka, Chi router, zero data races under `go test -race`. |
| **`/surface-web-desktop`** | Web & Desktop Portals | Next.js 15 App Router, React 19, TypeScript, Tailwind CSS, Tauri v2. Zero shared UI packages (local component ownership), enterprise SWR caching, offline resilience. |
| **`/surface-android-kotlin`** | Android Fleet | Kotlin 2.0+, Jetpack Compose, Coroutines/Flow, MVI architecture, `NetworkLifecycleEngine.kt`, zero mock data, hardware scanner binding (Zebra, Honeywell). |
| **`/surface-ios-swift`** | iOS Fleet | Swift 6, SwiftUI, Swift Concurrency (`async`/`await`, Actors, `@MainActor`), `NetworkLifecycleEngine.swift`, local design tokens, Apple HIG compliance. |
| **`/surface-optimizer-solver`** | Optimizer Core | Python Google OR-Tools + Rust CP-SAT solvers (`services/optimizer-core`). Mathematical solver honesty (`SolverStatus.HEURISTIC` vs `SolverStatus.OPTIMAL`), dynamic CVRPTW, 3D crushability bin-packing. |
| **`/surface-infra-sre`** | Cloud Infra & SRE | Terraform modules, Kubernetes (GKE) overlays, GCP Spanner, Memorystore, Kafka, strict multi-region cell isolation (`cell-uz`, `cell-eu`, `cell-kz`), immutable container digests. |
| **`/graph-rag`** | Code Graph Traversal | Memgraph & AST graph exploration (`graph_explore_symbol`, `graph_trace_flow`, `graph_blast_radius`, `graph_schema_dependencies`, `codegraph_explore`). |

---

## 4. Non-Negotiable Operational Invariants

1. **Money-Path Invariant**: Monetary amounts are strictly `int64` minor units (Uzbek Tiyin: `1 UZS = 100 Tiyin`). Floating-point arithmetic for currency is mechanically rejected by [`scripts/money_path_gate.sh`](../scripts/money_path_gate.sh).
2. **Zero Shared Cross-App UI Packages**: Client apps must own their UI, tokens, and composables locally (`components/ui/`, `ui/theme/`, `Theme/`). Shared packages are restricted to core networking and contracts.
3. **Spanner Outbox Pattern**: Handlers never publish directly to Kafka. Domain mutations write to `OutboxEvents` inside the same `spanner.ReadWriteTransaction`.
4. **Palantir Foundry Pre-Execution**: State transitions must pass the 12 Kinetic Edge Guards ([`apps/backend-go/foundry/edge_guards.go`](../apps/backend-go/foundry/edge_guards.go)) before committing.
5. **Strict Dual-Root Parity**: Changes made in `/Users/shakhzod/Desktop/pegasusx` must maintain 100% byte-for-byte parity with `/Users/shakhzod/Desktop/V.O.I.D/pegasusX`.
