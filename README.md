# ATOMOS

![Platform](https://img.shields.io/badge/Platform-Enterprise%20Logistics-0A66C2?style=for-the-badge)
![Architecture](https://img.shields.io/badge/Architecture-Event%20Driven%20Control%20Plane-0B7285?style=for-the-badge)
![Dispatch](https://img.shields.io/badge/Dispatch-H3%20Geo%20Batching%20%2B%20Capacity%20Fit-166534?style=for-the-badge)
![Runtime](https://img.shields.io/badge/Runtime-Go%20%2B%20Next.js%20%2B%20Kotlin%20%2B%20SwiftUI-5B21B6?style=for-the-badge)
![Consistency](https://img.shields.io/badge/Consistency-Transactional%20Outbox%20and%20Version%20Gates-7C2D12?style=for-the-badge)

ATOMOS is an enterprise-grade logistics operating system that coordinates supplier, factory, warehouse, driver, retailer, and payload operations across web, desktop, and native mobile surfaces.

The platform is built for high-consequence physical operations where route sequencing, payment integrity, geofence rules, and telemetry accuracy must remain coherent under high concurrency.

## Table of Contents

- [Executive Summary](#executive-summary)
- [Architecture Overview](#architecture-overview)
- [Exceptional Capabilities](#exceptional-capabilities)
- [Auto-Dispatch Deep Dive](#auto-dispatch-deep-dive)
- [State Machines and Lifecycle Contracts](#state-machines-and-lifecycle-contracts)
- [Reliability Control Plane](#reliability-control-plane)
- [Security and Role Integrity](#security-and-role-integrity)
- [Role to Surface Matrix](#role-to-surface-matrix)
- [Repository Topology](#repository-topology)
- [Quick Start](#quick-start)
- [Run and Build Commands](#run-and-build-commands)
- [Testing and Quality Gates](#testing-and-quality-gates)
- [Observability and Operations](#observability-and-operations)
- [Engineering Doctrine](#engineering-doctrine)
- [Documentation and Diagram Assets](#documentation-and-diagram-assets)

## Executive Summary

ATOMOS applies a control-plane architecture to real-world logistics execution.

Core system qualities:

1. Automation-first operations with policy-bounded human override.
2. Atomic state and event consistency using transactional outbox.
3. Geospatial dispatch intelligence using H3 cell clustering and capacity fitting.
4. Real-time execution visibility through role-scoped websocket hubs.
5. Cross-surface product coherence across web, desktop, Android, and iOS clients.

Business-critical invariants:

1. Order lifecycle integrity: `PENDING -> LOADED -> IN_TRANSIT -> ARRIVED -> COMPLETED`.
2. Financial correctness: double-entry compatible event-driven payment progression.
3. Route truthfulness: telemetry reflects planned vs actual execution.
4. Role safety: scope is resolved from claims, never trusted from request bodies.
5. Replay safety: version gates and idempotency guard against duplicate side effects.

## Architecture Overview

![ATOMOS Enterprise Architecture](the-lab-monorepo/docs/assets/architecture-overview.svg)

### Logical Architecture

```mermaid
flowchart LR
   subgraph Clients[Execution Surfaces]
      SP[Supplier Portals]
      DP[Driver Apps]
      RP[Retailer Apps]
      PP[Payload Apps]
   end

   subgraph Core[Platform Core]
      API[Go API and chi Router]
      DOM[Domain Services\nOrder Dispatch Fleet Payment Telemetry]
      HUB[WebSocket Hubs\nSupplier Driver Retailer Telemetry]
   end

   subgraph Plane[Data and Event Plane]
      SPN[Cloud Spanner]
      RED[Redis Cache and Invalidation Bus]
      KAF[Kafka Topics]
      OBX[Transactional Outbox Relay]
      AI[AI Worker and Optimization]
   end

   SP --> API
   DP --> API
   RP --> API
   PP --> API

   API --> DOM
   API --> HUB
   DOM --> SPN
   DOM --> RED
   DOM --> OBX
   OBX --> KAF
   KAF --> AI
   AI --> DOM
   DOM --> HUB
```

### Architecture Principles

1. Stateless service pods for clean scaling and safe rolling deploys.
2. Strong write consistency, stale-read options for read-heavy surfaces.
3. Event and state atomicity via outbox pattern in write transactions.
4. Partition-key ordering by aggregate identifier for deterministic consumers.
5. Degraded-mode tolerance where local user experience continues when possible.

## Exceptional Capabilities

| Capability | Technical Approach | Outcome |
|---|---|---|
| Auto-dispatch optimization | H3 geo-batching + capacity fit + route synthesis | Fewer empty miles and faster load-to-delivery cycles |
| Human override safety | Freeze-lock protocol for manual intervention windows | Operators can intervene without AI race conditions |
| Event consistency | Spanner RW transaction + outbox event row + relay | Prevents ghost state and missing downstream events |
| Realtime operations | Role-scoped hubs with Redis cross-pod relay | Shared live context across control surfaces |
| Payment correctness | Idempotent webhooks + versioned transitions + ledger-safe semantics | Financially auditable settlement flows |
| Execution telemetry | Planned vs actual route visibility with deviation signal | Faster intervention on delivery drift |
| Scale resilience | Priority guard, rate limiting, circuit breakers | Better tail behavior under burst load |
| Cross-role coherence | Shared contracts, role-specific clients, synchronized rollout protocol | Reduced product fragmentation |

## Auto-Dispatch Deep Dive

![Auto Dispatch Pipeline](the-lab-monorepo/docs/assets/autodispatch-pipeline.svg)

### Dispatch Pipeline

```mermaid
flowchart LR
   A[Demand and inventory signals] --> B[Eligibility filter\nstatus payment lock]
   B --> C[H3 geo batching]
   C --> D[Capacity fit engine]
   D --> E[Route synthesis and split]
   E --> F[Policy and scope guard]
   F --> G[Atomic write + outbox emit]
   G --> H[Kafka and websocket fanout]
   H --> I[Driver execution]
   I --> J[Telemetry feedback and re-plan]
```

### How It Works

1. Signals are ingested from pending order queues, stock thresholds, and SLA windows.
2. Eligibility filtering removes blocked entities (freeze-locked, unpaid, out-of-scope).
3. Orders are clustered by H3 cell and adjacency ring to preserve geographic cohesion.
4. Capacity fitting maps clusters to available drivers and vehicles using load-aware assignment.
5. Oversized manifests are split while preserving route integrity.
6. Mutations are committed with outbox events in the same transaction.
7. Fanout updates telemetry hubs and role-specific clients.
8. Deviations and exceptions feed the next optimization cycle.

### Why This Is Different

1. Automation is the default behavior, not an optional add-on.
2. Manual selection is supported without sacrificing auditability.
3. Dispatch decisions are evented and traceable end-to-end.
4. Route progress is measured against actual execution, not static plan assumptions.

## State Machines and Lifecycle Contracts

### Order Lifecycle

```mermaid
stateDiagram-v2
   [*] --> PENDING
   PENDING --> LOADED
   LOADED --> IN_TRANSIT
   IN_TRANSIT --> ARRIVED
   ARRIVED --> COMPLETED

   PENDING --> CANCELLED: policy-bound cancellation
   IN_TRANSIT --> EXCEPTION: delivery incident
   EXCEPTION --> IN_TRANSIT: resolved and resumed
```

### Delivery Sequence and Control Points

```mermaid
sequenceDiagram
   participant Portal as Supplier Portal
   participant API as Backend API
   participant DB as Spanner
   participant Outbox as Outbox Relay
   participant Kafka as Kafka
   participant Driver as Driver App
   participant Telemetry as Telemetry Hub

   Portal->>API: Trigger dispatch for eligible orders
   API->>DB: RW transaction for assignment and manifest mutation
   API->>DB: Write outbox event row in same transaction
   DB-->>API: Commit success
   API-->>Portal: Accepted with updated assignment
   Outbox->>Kafka: Publish ORDER_ASSIGNED and ROUTE_CREATED
   Kafka-->>Driver: Assignment consumer fanout
   Driver->>Telemetry: Route progression updates
   Telemetry-->>Portal: Live operational state
```

## Reliability Control Plane

![Reliability Control Plane](the-lab-monorepo/docs/assets/reliability-control-plane.svg)

### Reliability Invariants

| Invariant | Why It Matters | Enforced By |
|---|---|---|
| Mutation-event atomicity | No split-brain between database and event consumers | RW transaction + outbox write |
| Replay-safe consumers | Duplicate event deliveries do not corrupt state | Version gating + idempotency checks |
| Cache coherence | Reads do not stay stale after writes | Post-commit invalidation publish |
| Realtime continuity | Local websocket users still receive updates during pub-sub turbulence | Fail-open local fanout |
| Upstream failure isolation | External outages do not collapse core flows | Circuit breaker + bounded retry |
| Load shedding discipline | Critical paths remain alive under spikes | Priority guard + token bucket limits |

## Security and Role Integrity

Security posture is zero-trust at the handler boundary and policy-strict inside domain flows.

1. Role and node scope is resolved from signed claims.
2. Mutation endpoints do not trust supplier_id, factory_id, or warehouse_id from request body.
3. Webhooks validate signature before body parse and before any database writes.
4. Idempotency keys prevent duplicate external side effects.
5. Websocket subscriptions are auth-bound and room-scoped.
6. Structured logs carry trace_id for end-to-end forensic stitching.

Role naming note:

1. The Supplier Portal is implemented in code with ADMIN JWT naming compatibility.
2. Product user identity remains SUPPLIER for operational semantics.

## Role to Surface Matrix

| Role | Surface | Stack | Path |
|---|---|---|---|
| SUPPLIER | Admin Portal (web + desktop shell) | Next.js 15 + React 19 + Tailwind v4 | the-lab-monorepo/apps/admin-portal |
| DRIVER | Android | Kotlin + Jetpack Compose | the-lab-monorepo/apps/driver-app-android |
| DRIVER | iOS | SwiftUI | the-lab-monorepo/apps/driverappios |
| RETAILER | Android | Kotlin + Jetpack Compose | the-lab-monorepo/apps/retailer-app-android |
| RETAILER | iOS | SwiftUI | the-lab-monorepo/apps/retailer-app-ios |
| RETAILER | Desktop | Next.js + Tauri shell | the-lab-monorepo/apps/retailer-app-desktop |
| PAYLOAD | Terminal | Expo + React Native | the-lab-monorepo/apps/payload-terminal |
| PAYLOAD | iOS tablet | SwiftUI | the-lab-monorepo/apps/payload-app-ios |
| PAYLOAD | Android tablet | Kotlin + Jetpack Compose | the-lab-monorepo/apps/payload-app-android |
| FACTORY_ADMIN | Portal (web + desktop shell) | Next.js + Tailwind v4 | the-lab-monorepo/apps/factory-portal |
| FACTORY_ADMIN | Android | Kotlin + Jetpack Compose | the-lab-monorepo/apps/factory-app-android |
| FACTORY_ADMIN | iOS | SwiftUI | the-lab-monorepo/apps/factory-app-ios |
| WAREHOUSE_ADMIN | Portal (web + desktop shell) | Next.js + Tailwind v4 | the-lab-monorepo/apps/warehouse-portal |
| WAREHOUSE_ADMIN | Android | Kotlin + Jetpack Compose | the-lab-monorepo/apps/warehouse-app-android |
| WAREHOUSE_ADMIN | iOS | SwiftUI | the-lab-monorepo/apps/warehouse-app-ios |

## Repository Topology

```text
V.O.I.D/
|- README.md
|- the-lab-monorepo/
|  |- apps/
|  |  |- backend-go/
|  |  |- ai-worker/
|  |  |- admin-portal/
|  |  |- factory-portal/
|  |  |- warehouse-portal/
|  |  |- retailer-app-desktop/
|  |  |- driver-app-android/
|  |  |- driverappios/
|  |  |- retailer-app-android/
|  |  |- retailer-app-ios/
|  |  |- payload-terminal/
|  |  |- payload-app-ios/
|  |  |- payload-app-android/
|  |  |- factory-app-android/
|  |  |- factory-app-ios/
|  |  |- warehouse-app-android/
|  |  |- warehouse-app-ios/
|  |- packages/
|  |  |- api-client/
|  |  |- config/
|  |  |- optimizer-contract/
|  |  |- types/
|  |  |- ui-kit/
|  |  |- validation/
|  |- docs/
|  |  |- assets/
|  |  |  |- architecture-overview.svg
|  |  |  |- autodispatch-pipeline.svg
|  |  |  |- reliability-control-plane.svg
|  |- infra/
|  |- tests/
```

## Quick Start

### Prerequisites

1. Docker and Docker Compose
2. Go 1.22+
3. Node.js 20+
4. Xcode for iOS builds
5. Android Studio for Android builds

### Bootstrap Local Infrastructure

```bash
cd the-lab-monorepo
docker compose up -d
```

### Initialize Spanner Emulator and Seed Data

```bash
cd the-lab-monorepo
make spanner-init
make seed
```

### Build and Run Backend

```bash
cd the-lab-monorepo/apps/backend-go
go build ./...
go run .
```

## Run and Build Commands

### Core Environment

```bash
cd the-lab-monorepo
make env-up
make env-status
make env-down
```

### Web and Desktop Surfaces

```bash
cd the-lab-monorepo/apps/admin-portal && npm run dev
cd the-lab-monorepo/apps/admin-portal && npm run tauri:dev
cd the-lab-monorepo/apps/factory-portal && npm run dev
cd the-lab-monorepo/apps/warehouse-portal && npm run dev
cd the-lab-monorepo/apps/retailer-app-desktop && npm run tauri:dev
```

### Mobile Surfaces

```bash
cd the-lab-monorepo/apps/payload-terminal && npm run start
cd the-lab-monorepo/apps/driver-app-android && ./gradlew :app:assembleDebug
cd the-lab-monorepo/apps/retailer-app-android && ./gradlew :app:assembleDebug
cd the-lab-monorepo/apps/payload-app-android && ./gradlew :app:assembleDebug
```

### Desktop Scripts from Monorepo Root

```bash
cd the-lab-monorepo
npm run desktop:admin:dev
npm run desktop:factory:dev
npm run desktop:warehouse:dev
npm run desktop:retailer:dev
```

## Testing and Quality Gates

### Backend

```bash
cd the-lab-monorepo/apps/backend-go
go test ./...
go vet ./...
go build ./...
```

### Workspace E2E

```bash
cd the-lab-monorepo
npm run test:e2e
npm run test:e2e:admin
npm run test:e2e:retailer
npm run test:e2e:factory
npm run test:e2e:warehouse
npm run test:e2e:api
npm run test:e2e:cross
```

### Version Drift Guard

```bash
cd the-lab-monorepo
npm run versionscan:scan
npm run versionscan:enforce
```

## Observability and Operations

Operational telemetry is designed for incident triage, execution debugging, and audit reconstruction.

1. Structured JSON logs with request-level trace_id propagation.
2. Websocket and event-chain observability for route-level execution timelines.
3. Consumer lag and failure visibility for asynchronous pipelines.
4. Priority-based request shedding under surge conditions.
5. Event replay detection through version and idempotency enforcement.

### Recommended Incident Drill Path

1. Capture trace_id from ingress request.
2. Follow mutation commit in backend logs.
3. Confirm outbox publish and Kafka consumer apply.
4. Verify websocket room broadcast and client acknowledgment.
5. Compare expected and actual state in operational surface.

## Engineering Doctrine

This repository follows a systems doctrine focused on correctness under load and cross-surface coherence.

1. Domain packages own business logic. Route packages remain thin.
2. main.go is lifecycle orchestration, not business implementation.
3. Mutation handlers follow strict shape: auth gate -> validate -> transaction -> outbox -> invalidate cache -> structured response.
4. Any role feature must ship coherently across all client surfaces for that role.
5. Additive contract evolution is required to protect older client versions.

## Documentation and Diagram Assets

Primary docs:

1. the-lab-monorepo/docs/BARCODE_SCANNING.md
2. the-lab-monorepo/docs/CLOUD_RUN_TO_GKE_CUTOVER_RUNBOOK.md
3. the-lab-monorepo/docs/MAGLEV_READ_ROUTER_ROLLOUT.md
4. the-lab-monorepo/E2E_TEST_PROTOCOL.md

Architecture graphics in this README:

1. the-lab-monorepo/docs/assets/architecture-overview.svg
2. the-lab-monorepo/docs/assets/autodispatch-pipeline.svg
3. the-lab-monorepo/docs/assets/reliability-control-plane.svg

---

ATOMOS is designed as an execution-grade logistics system, not a demo dashboard. The architecture choices in this repository prioritize deterministic operations, high-scale resilience, and role-accurate workflows from first principles.
