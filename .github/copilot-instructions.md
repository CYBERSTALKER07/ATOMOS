# GitHub Copilot Instructions — V.O.I.D. Universal Enterprise Monorepo

This file provides the canonical Copilot engineering guidelines for the `V.O.I.D` monorepo, covering all three codebases (`pegasus`, `pegasusX`, `pegasus.x`).

---

## 1. Architectural Topology & Codebase Disambiguation

```
/Users/shakhzod/Desktop/V.O.I.D
├── pegasus/      # [LEGACY / REFERENCE] Historical prototype. Read-only architectural reference.
├── pegasusX/     # [GLOBAL CLOUD] Global Enterprise Multi-Tenant Architecture (Spanner + Kafka).
└── pegasus.x/    # [SOVEREIGN CORE] Sovereign Lean Single-Tenant National Operating Core (PG16 + Redis 7).
```

### Key Differences & Non-Negotiable Boundaries:
- **`pegasus/`**: Read-only historical reference. Do not add new code, create plans, or ship features from this tree.
- **`pegasusX/`**:
  - Cloud-native multi-tenant enterprise system.
  - Database: Google Cloud Spanner (`schema/spanner.ddl`, 3,750 lines) with root `SupplierId STRING(36)` partitioning and table interleaving.
  - Messaging: Apache Kafka (topic partitioning, `RequiredAcks=all`) + Spanner `OutboxEvents`.
  - Invariant: Strictly Google Cloud Spanner + Apache Kafka. NEVER downgrade to single-tenant PostgreSQL or import PostgreSQL drivers.
- **`pegasus.x/`**:
  - Sovereign lean single-tenant national operating core for Uzbekistan (Servercore Tashkent Tier III, TAS-IX domestic peering, $135/mo).
  - Database: PostgreSQL 16 (`pgx/v5` + TimescaleDB + PostGIS) + Redis 7 (`redis-go`, Streams, presence).
  - Messaging: PostgreSQL transactional outbox (`outbox_events` table) + Redis 7 Streams.
  - Clients: Tauri v2 Desktop (Next.js 15), Telegram Mini App & Bot for Retailers, Native Mobile for Drivers.
  - Invariant: Strictly PostgreSQL 16 + Redis 7. NEVER import Spanner libraries, Spanner DDL, or Apache Kafka.
- **Cross-Scanning & Feature Parity**:
  - Scan both codebases to detect feature parity gaps, micro-features, state machine rules, and UI/UX patterns from `pegasusX` and port them logically into `pegasus.x`.
  - Active Primary Gap: Full Fleet & Driver Management lifecycle (Vehicles/Trucks, Drivers, Dynamic Daily Shift Pairing, Mid-Shift Hot-Swapping, Pre-trip Vehicle Inspections [DVIR]) ported into `pegasus.x`.

---

## 2. Big Tech Engineering Rigor & Truth Protocol

- **Live Code is the Only Status SoT**: Repository source code and passing automated tests (`go test -v -race ./...`) are the sole status Source of Truth. Documentation and matrices labeled "Wired" are hypotheses.
- **Zero Mock Data Policy**: Zero mock data, in-memory repository fallbacks (e.g. `MemoryRepository`), or static fake seeds in production packages. All domain data must dynamically query and persist to PostgreSQL 16 (for `pegasus.x`) or Google Cloud Spanner (for `pegasusX`).
- **Strict Financial Currency Arithmetic**: All money, pricing, and balances must use 64-bit integer minor units (`tiyins` for UZS, `cents` for USD). Floating-point currency calculations are strictly forbidden. Double-entry ledger invariant: `Total Debits == Total Credits`.
- **Atomic Transactional Outbox**: All entity mutations and outbox events must be written in the **exact same database transaction** (`pgx.Tx` in PG16; `spanner.ReadWriteTransaction` in Spanner).

---

## 3. Two-Tier Verification Gate (Mandatory on Every Edit)

1. **Tier 1 — Bazel/Kythe Dynamic CodeGraph (Global Radar)**:
   - Run before touching code to identify mathematical blast radius and reverse dependencies:
   - `python3 pegasusX/scripts/advanced_codegraph_analyzer.py --blast-radius <symbol> --depth 3 --json`
   - `python3 pegasusX/scripts/bazel_target_graph.py --query-rdeps <target>`
2. **Tier 2 — Targeted Raw Reading (Local Microscope)**:
   - Open and raw-read the exact files identified in Tier 1. Inspect runtime conditionals, guard clauses, transaction boundaries, and error propagation. Re-read every edit after writing.
3. **Automated Testing**:
   - Immediately execute `go test -v -race ./...` on touched packages. If any test fails, replan immediately.

---

## 4. Web Search & Knowledge Sourcing Protocol

Always proactively search the web for:
- Statutory regulations: Uzbekistan Soliq OFD/EHF fiscalization (12% VAT, 25M UZS B2B cash limit, 17-digit MXIK codes, 9-digit STIR).
- Mathematical and algorithmic standards: GS1 EAN-13 Mod-10 checksum, Haversine distance, Uber H3 spatial index, Croston-SBA intermittent demand forecasting, Google OR-Tools CVRP.
- Gateways & APIs: Payme HMAC-SHA256, Click MD5/SHA1, Global Pay B2B Corporate Card, Telegram Bot API & Mini App SDK.
- Proven open-source libraries before rolling custom code (`pgx/v5`, `go-chi/chi/v5`, `redis-go`, `pydantic-v2`, `fastapi`).

---

## 5. User Confirmation Protocol

- **Proceed Autonomously**: Feature implementation per spec, forward database migrations, unit/integration test suites, refactoring code, eliminating mocks, syncing role-row client contracts, web search.
- **Confirm With User**: Dropping tables or irreversible DB migrations, provisioning paid cloud infrastructure incurring financial costs, inputting live production secrets/certificates, ambiguous business decisions.

---

## 6. Strict UI/UX Design System (Tactical Control Tower)

- **Palette**: Pitch-black canvas (`#09090B`) / crisp light (`#F8FAFC`), deep obsidian (`#121216`), crisp 1px borders (`#22222C`), cobalt blue (`#2563EB`/`#3B82F6`), safety orange (`#FF7A1A`), tactical lime (`#E2FD52`).
- **Layout**: 3-Column Control Tower: Navigation Rail → Center Density Feed → Right Inspector Drawer, with floating bottom command bar.
- **Typography**: Tabular monospace numbers (`font-mono tabular-nums`), 10px–11px bold uppercase micro-labels (`tracking-wider`).
- **Forbidden**: Zero generic gray templates, zero emoji icons (use real SVGs), zero blurry drop shadows.

---

## 7. Pre-Completion Verification Checklist

- [ ] Identified active codebase (`pegasus.x` vs `pegasusX`); 0 cross-contamination.
- [ ] Tier 1 blast radius checked; Tier 2 raw reading verified.
- [ ] 0 mock data in production packages; full DB persistence wired.
- [ ] 64-bit integer minor unit arithmetic used for all financial fields (`tiyins`).
- [ ] Outbox event paired atomically in same database transaction as entity write.
- [ ] All touched role-row clients updated with identical contracts.
- [ ] `go test -v -race ./...` passing cleanly.
