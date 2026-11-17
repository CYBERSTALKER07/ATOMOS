# Project: Pegasus and PegasusX Codebase Documentation & AI Instructions

## Architecture Overview
The project workspace contains two distinct logistics and commerce distributions:
1. `pegasus/`: Multi-supplier logistics execution engine and marketplace platform (Go 1.25, Chi, Spanner 94 tables, Kafka 8 topics, Rust/Go/LangGraph OR solvers, 18 apps, 2 services, 8 packages).
2. `pegasusX/`: Enterprise Single-Supplier Multi-Retailer (SSMR) distribution stack (Go 1.25, 411+ endpoints, 28 route modules, 24 runtime workers, Spanner 220+ tables, 125 migrations, 100+ Kafka events, 8 Redis-backed WebSocket hubs, 22 apps across 6 role-rows, Rust & Python OR-Tools solvers, cell isolation).

## Feature Inventory
| # | Feature / Component | Description | Ecosystem | Milestone | Source |
|---|---------------------|-------------|-----------|-----------|--------|
| 1 | Pegasus `agents.md` | Architecture, honesty rules, zero-theatre doctrine, dev workflows | pegasus | M1 | `pegasus/agents.md` |
| 2 | Pegasus Architecture Docs | Monorepo layout, Chi composition, Outbox pattern, Spanner 94 tables | pegasus | M1 | `pegasus/docs/ARCHITECTURE.md` |
| 3 | Pegasus Backend & Solvers Docs | Chi subrouters, 13 background crons, 6 WS hubs, ai-worker, optimizer-core | pegasus | M1 | `pegasus/docs/BACKEND_SERVICES.md` |
| 4 | Pegasus Features & Clients Docs | 4 Next.js/Tauri portals, 5 Android, 5 iOS, 1 Expo terminal, H3 spatial, treasury | pegasus | M1 | `pegasus/docs/FEATURES_AND_PORTALS.md` |
| 5 | Pegasus Infrastructure Docs | Terraform multi-region Spanner, K8s KEDA, Docker Compose emulators, parity gates | pegasus | M1 | `pegasus/docs/INFRASTRUCTURE.md` |
| 6 | PegasusX `agents.md` | SSMR doctrine, honesty commandments, zero-theatre rules, strict CI gates | pegasusX | M2 | `pegasusX/agents.md` |
| 7 | PegasusX Architecture Docs | SSMR model, Spanner 220+ tables, 125 migrations, Kafka 100+ events, cell isolation | pegasusX | M2 | `pegasusX/docs/ARCHITECTURE.md` |
| 8 | PegasusX Backend Services Docs | 411+ endpoints, 28 routes, 24 workers, 8 WS hubs, ai-worker, Rust/Python solvers | pegasusX | M2 | `pegasusX/docs/BACKEND_SERVICES.md` |
| 9 | PegasusX Features & Role-Rows Docs | 6 role-rows, 22 apps (Web/Desktop/Android/iOS/Expo), ParentOrders saga, money path | pegasusX | M2 | `pegasusX/docs/FEATURES_AND_ROLE_ROWS.md` |
| 10 | PegasusX Infrastructure Docs | Terraform 6 modules & cells, K8s 7 overlays & 5 CronJobs, CI anti-theatre gates | pegasusX | M2 | `pegasusX/docs/INFRASTRUCTURE.md` |
| 11 | Cross-Grounding & Link Verification | Audit 100% of generated links (`file:///...`) for real existence and line accuracy | All | M3 | Verified dual-ecosystem link parity |

## Milestones
| # | Name | Scope | Dependencies | Status | Key Deliverables |
|---|------|-------|-------------|--------|------------------|
| M0 | Ecosystem Survey | Deep inspection of pegasus and pegasusX codebases | None | DONE | Survey reports & architectural baseline |
| M1 | Pegasus Documentation & agents.md | `pegasus/agents.md` and `pegasus/docs/*.md` | M0 | DONE | `pegasus/agents.md`, `pegasus/docs/{ARCHITECTURE,BACKEND_SERVICES,FEATURES_AND_PORTALS,INFRASTRUCTURE}.md` |
| M2 | PegasusX Documentation & agents.md | `pegasusX/agents.md` and `pegasusX/docs/*.md` | M0 | DONE | `pegasusX/agents.md`, `pegasusX/docs/{ARCHITECTURE,BACKEND_SERVICES,FEATURES_AND_ROLE_ROWS,INFRASTRUCTURE}.md` |
| M3 | Grounding & Link Verification | Audit all generated docs for direct `file:///` links and zero theatre | M1, M2 | DONE | 100% resolution across active codebases |

## Final Documentation Catalog
- **Pegasus Ecosystem**:
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/agents.md`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/docs/ARCHITECTURE.md`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/docs/BACKEND_SERVICES.md`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/docs/FEATURES_AND_PORTALS.md`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/docs/INFRASTRUCTURE.md`
- **PegasusX Ecosystem**:
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/agents.md`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/docs/ARCHITECTURE.md`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/docs/BACKEND_SERVICES.md`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/docs/FEATURES_AND_ROLE_ROWS.md`
  - `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/docs/INFRASTRUCTURE.md`

## Interface Contracts & Documentation Standards
- **AI Agent Instructions (`agents.md`)**:
  - Purpose, scope, architectural style, and scale of that specific ecosystem.
  - Mandatory Honesty Rules ("Zero Theatre", "No fabricated APIs/endpoints", "No TODO injection", "No mock control tower data").
  - Development workflows, local emulator setup, test commands, lint/build gates.
- **Component & Feature Documentation**:
  - What it is: Technical definition and domain responsibility.
  - How it works: Architecture, protocols, algorithms, state machines, and data structures.
  - Why it is there: Business rationale, architectural trade-offs, and compliance/operational drivers.
- **Absolute Code Grounding**:
  - Every technical claim, route, schema table, worker, algorithm, and configuration cites direct `file:///Users/shakhzod/Desktop/V.O.I.D/...` links with exact lines.
