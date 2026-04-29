# ATOMOS

![Platform](https://img.shields.io/badge/Platform-Enterprise%20Logistics-191622?style=for-the-badge)
![Architecture](https://img.shields.io/badge/Architecture-Event%20Driven%20Control%20Plane-2F5BFF?style=for-the-badge)
![Dispatch](https://img.shields.io/badge/Dispatch-H3%20Geo%20Batching%20%2B%20Capacity%20Fit-00C96B?style=for-the-badge)
![Runtime](https://img.shields.io/badge/Runtime-Go%20%2B%20Next.js%20%2B%20Kotlin%20%2B%20SwiftUI-40E0FF?style=for-the-badge)
![Consistency](https://img.shields.io/badge/Consistency-Transactional%20Outbox%20and%20Version%20Gates-FF7A18?style=for-the-badge)

![Glass Hero Banner](the-lab-monorepo/docs/assets/glass-hero-variant-a.svg)

ATOMOS is an enterprise-grade logistics operating system that coordinates supplier, factory, warehouse, driver, retailer, and payload operations across web, desktop, and native mobile surfaces.

The platform is built for high-consequence physical operations where route sequencing, payment integrity, geofence rules, and telemetry accuracy must remain coherent under high concurrency.

Audience variants:

1. Engineering master document: this file.
2. Investor and partner narrative: [README-investors.md](README-investors.md).

## Table of Contents

- [Audience Variants](#audience-variants)
- [Executive Summary](#executive-summary)
- [Architecture Overview](#architecture-overview)
- [Maglev Load Balancing Coverage](#maglev-load-balancing-coverage)
- [Exceptional Capabilities](#exceptional-capabilities)
- [Auto-Dispatch Deep Dive](#auto-dispatch-deep-dive)
- [State Machines and Lifecycle Contracts](#state-machines-and-lifecycle-contracts)
- [Reliability Control Plane](#reliability-control-plane)
- [Security and Role Integrity](#security-and-role-integrity)
- [Role to Surface Matrix](#role-to-surface-matrix)
- [Technology Stack and Platforms](#technology-stack-and-platforms)
- [Repository Topology](#repository-topology)
- [Quick Start](#quick-start)
- [Run and Build Commands](#run-and-build-commands)
- [Testing and Quality Gates](#testing-and-quality-gates)
- [Observability and Operations](#observability-and-operations)
- [Engineering Doctrine](#engineering-doctrine)
- [Documentation and Diagram Assets](#documentation-and-diagram-assets)

## Audience Variants

1. Engineering master reference: [README.md](README.md).
2. External investor and partner variant: [README-investors.md](README-investors.md).

![Section Divider](the-lab-monorepo/docs/assets/omni-section-divider.svg)

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

![Section Divider](the-lab-monorepo/docs/assets/omni-section-divider.svg)

![ATOMOS Enterprise Architecture](the-lab-monorepo/docs/assets/architecture-overview.svg)

### Logical Architecture

```mermaid
%%{init: {"theme":"base","themeVariables":{"darkMode":true,"background":"#000000","primaryColor":"#0F1117","primaryTextColor":"#E8ECF6","primaryBorderColor":"#475569","secondaryColor":"#141923","secondaryTextColor":"#E8ECF6","secondaryBorderColor":"#475569","tertiaryColor":"#1A2230","tertiaryTextColor":"#E8ECF6","tertiaryBorderColor":"#475569","lineColor":"#D1D8E5","textColor":"#E8ECF6","mainBkg":"#0F1117","nodeBorder":"#475569","clusterBkg":"#0F1117","clusterBorder":"#475569","titleColor":"#E8ECF6","edgeLabelBackground":"#0F1117","actorBkg":"#0F1117","actorBorder":"#475569","actorTextColor":"#E8ECF6","actorLineColor":"#D1D8E5","signalColor":"#D1D8E5","signalTextColor":"#E8ECF6","labelBoxBkgColor":"#0F1117","labelBoxBorderColor":"#475569","labelTextColor":"#E8ECF6","loopTextColor":"#E8ECF6","activationBkgColor":"#1F2633","activationBorderColor":"#475569","sequenceNumberColor":"#000000","stateBkg":"#0F1117","stateBorder":"#475569","stateTextColor":"#E8ECF6","noteBkgColor":"#0F1117","noteTextColor":"#E8ECF6","noteBorderColor":"#475569"},"themeCSS":".edgeLabel text,.label text,.stateLabel text,.messageText,.nodeLabel{fill:#E8ECF6 !important;color:#E8ECF6 !important;} rect,.actor,.note,.labelBox,.edgeLabel rect{stroke:rgba(148,163,184,0.45) !important;stroke-width:1.2px !important;rx:40px !important;ry:40px !important;} .edgeLabel rect{fill:rgba(10,12,16,0.86) !important;opacity:1 !important;} .cluster > rect{fill:rgba(10,12,16,0.78) !important;} .messageLine0,.messageLine1,.loopLine{stroke:#D1D8E5 !important;}"}}%%
flowchart LR
   subgraph Clients[Execution Surfaces]
      SP[Supplier Portals]
      DP[Driver Apps]
      RP[Retailer Apps]
      PP[Payload Apps]
   end

   subgraph Ingress[Global Ingress]
      MAG[Maglev Ring-Hash Affinity\nX-Supplier-Id]
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

   SP --> MAG
   DP --> MAG
   RP --> MAG
   PP --> MAG
   MAG --> API

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

## Maglev Load Balancing Coverage

![Section Divider](the-lab-monorepo/docs/assets/omni-section-divider.svg)

![Maglev Load Balancer Coverage](the-lab-monorepo/docs/assets/maglev-load-balancers.svg)

Implemented Maglev or Maglev-derived load balancer paths:

1. Edge ingress ring-hash affinity at the global external backend service using `X-Supplier-Id`.
2. Backend Spanner read routing with a Maglev-derived pre-built lookup table pattern.
3. Internal optimizer gRPC xDS path (mesh-balanced), gated by `OPTIMIZER_GRPC_ADDR`.

Current activation status:

1. Edge ring-hash is configured in infrastructure.
2. Spanner read-router runtime currently boots in single-region mode (`NewSingleRegion`) unless multiregion activation is wired.
3. xDS optimizer path is active only when env-gated in deployment.

Implementation map:

1. Edge ring-hash infrastructure: `the-lab-monorepo/infra/terraform/networking.tf`.
2. Maglev-derived read-router engine: `the-lab-monorepo/apps/backend-go/bootstrap/spannerrouter/router.go`.
3. Current single-region boot mode: `the-lab-monorepo/apps/backend-go/bootstrap/new.go`.
4. xDS gRPC load-balanced client path: `the-lab-monorepo/apps/backend-go/internal/rpc/optimizergrpc/client.go`.
5. xDS gRPC optimizer server endpoint: `the-lab-monorepo/apps/ai-worker/grpc_server.go`.

Operational note:

1. Warehouse sibling reroute is operational load balancing logic, not Maglev ring-hash.

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

![Section Divider](the-lab-monorepo/docs/assets/omni-section-divider.svg)

![Auto Dispatch Pipeline](the-lab-monorepo/docs/assets/autodispatch-pipeline.svg)

### Dispatch Pipeline

```mermaid
%%{init: {"theme":"base","themeVariables":{"darkMode":true,"background":"#191622","primaryColor":"#232136","primaryTextColor":"#E1E1E6","primaryBorderColor":"#78D1E1","secondaryColor":"#2A2338","secondaryTextColor":"#E1E1E6","secondaryBorderColor":"#988BC7","tertiaryColor":"#1F3026","tertiaryTextColor":"#E1E1E6","tertiaryBorderColor":"#67E480","lineColor":"#E1E1E6","textColor":"#E1E1E6","mainBkg":"#232136","nodeBorder":"#78D1E1","clusterBkg":"#232136","clusterBorder":"#988BC7","titleColor":"#E1E1E6","edgeLabelBackground":"#232136","actorBkg":"#232136","actorBorder":"#78D1E1","actorTextColor":"#E1E1E6","actorLineColor":"#E1E1E6","signalColor":"#E1E1E6","signalTextColor":"#E1E1E6","labelBoxBkgColor":"#232136","labelBoxBorderColor":"#988BC7","labelTextColor":"#E1E1E6","loopTextColor":"#E1E1E6","activationBkgColor":"#2A2338","activationBorderColor":"#988BC7","sequenceNumberColor":"#191622","stateBkg":"#232136","stateBorder":"#67E480","stateTextColor":"#E1E1E6","noteBkgColor":"#232136","noteTextColor":"#E1E1E6","noteBorderColor":"#988BC7"},"themeCSS":".edgeLabel text,.label text,.stateLabel text,.messageText,.nodeLabel{fill:#E1E1E6 !important;color:#E1E1E6 !important;} .edgeLabel rect{fill:#232136 !important;opacity:1 !important;}"}}%%
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
%%{init: {"theme":"base","themeVariables":{"darkMode":true,"background":"#000000","primaryColor":"#171923","primaryTextColor":"#E8ECF6","primaryBorderColor":"#000000","secondaryColor":"#1F2430","secondaryTextColor":"#E8ECF6","secondaryBorderColor":"#000000","tertiaryColor":"#222A35","tertiaryTextColor":"#E8ECF6","tertiaryBorderColor":"#000000","lineColor":"#D1D8E5","textColor":"#E8ECF6","mainBkg":"#171923","nodeBorder":"#000000","clusterBkg":"#171923","clusterBorder":"#000000","titleColor":"#E8ECF6","edgeLabelBackground":"#171923","actorBkg":"#171923","actorBorder":"#000000","actorTextColor":"#E8ECF6","actorLineColor":"#D1D8E5","signalColor":"#D1D8E5","signalTextColor":"#E8ECF6","labelBoxBkgColor":"#171923","labelBoxBorderColor":"#000000","labelTextColor":"#E8ECF6","loopTextColor":"#E8ECF6","activationBkgColor":"#242B38","activationBorderColor":"#000000","sequenceNumberColor":"#000000","stateBkg":"#171923","stateBorder":"#000000","stateTextColor":"#E8ECF6","noteBkgColor":"#171923","noteTextColor":"#E8ECF6","noteBorderColor":"#000000"},"themeCSS":".edgeLabel text,.label text,.stateLabel text,.messageText,.nodeLabel{fill:#E8ECF6 !important;color:#E8ECF6 !important;} rect,.actor,.note,.labelBox,.edgeLabel rect{stroke:none !important;rx:40px !important;ry:40px !important;} .edgeLabel rect{fill:rgba(255,255,255,0.12) !important;opacity:1 !important;} .messageLine0,.messageLine1,.loopLine{stroke:#D1D8E5 !important;} .stateGroup circle,.stateGroup rect,.statediagram-state rect{stroke:none !important;}"}}%%
stateDiagram-v2
   [*] --> PENDING
   PENDING --> LOADED
   LOADED --> IN_TRANSIT
   IN_TRANSIT --> ARRIVED
   ARRIVED --> COMPLETED

   PENDING --> CANCELLED: policy-bound cancellation
   IN_TRANSIT --> EXCEPTION: delivery incident
   EXCEPTION --> IN_TRANSIT: resolved and resumed

   classDef omniState fill:#171923,stroke:none,color:#E8ECF6,stroke-width:0px
   class PENDING,LOADED,IN_TRANSIT,ARRIVED,COMPLETED,CANCELLED,EXCEPTION omniState
```

### Delivery Sequence and Control Points

```mermaid
%%{init: {"theme":"base","themeVariables":{"darkMode":true,"background":"#000000","primaryColor":"#171923","primaryTextColor":"#E8ECF6","primaryBorderColor":"#000000","secondaryColor":"#1F2430","secondaryTextColor":"#E8ECF6","secondaryBorderColor":"#000000","tertiaryColor":"#222A35","tertiaryTextColor":"#E8ECF6","tertiaryBorderColor":"#000000","lineColor":"#D1D8E5","textColor":"#E8ECF6","mainBkg":"#171923","nodeBorder":"#000000","clusterBkg":"#171923","clusterBorder":"#000000","titleColor":"#E8ECF6","edgeLabelBackground":"#171923","actorBkg":"#171923","actorBorder":"#000000","actorTextColor":"#E8ECF6","actorLineColor":"#D1D8E5","signalColor":"#D1D8E5","signalTextColor":"#E8ECF6","labelBoxBkgColor":"#171923","labelBoxBorderColor":"#000000","labelTextColor":"#E8ECF6","loopTextColor":"#E8ECF6","activationBkgColor":"#242B38","activationBorderColor":"#000000","sequenceNumberColor":"#000000","stateBkg":"#171923","stateBorder":"#000000","stateTextColor":"#E8ECF6","noteBkgColor":"#171923","noteTextColor":"#E8ECF6","noteBorderColor":"#000000"},"themeCSS":".edgeLabel text,.label text,.stateLabel text,.messageText,.nodeLabel{fill:#E8ECF6 !important;color:#E8ECF6 !important;} rect,.actor,.note,.labelBox,.edgeLabel rect{stroke:none !important;rx:40px !important;ry:40px !important;} .edgeLabel rect{fill:rgba(255,255,255,0.12) !important;opacity:1 !important;} .messageLine0,.messageLine1,.loopLine{stroke:#D1D8E5 !important;} .stateGroup circle,.stateGroup rect,.statediagram-state rect{stroke:none !important;}"}}%%
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

![Section Divider](the-lab-monorepo/docs/assets/omni-section-divider.svg)

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

## Technology Stack and Platforms

![Tech Stack Matrix](the-lab-monorepo/docs/assets/techstack-glass-matrix.svg)

![Tech Stack Compact](the-lab-monorepo/docs/assets/techstack-glass-compact.svg)

![Go](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![Kafka](https://img.shields.io/badge/Apache%20Kafka-231F20?style=for-the-badge&logo=apachekafka&logoColor=white)
![Redis](https://img.shields.io/badge/Redis-DC382D?style=for-the-badge&logo=redis&logoColor=white)
![Spanner](https://img.shields.io/badge/Cloud%20Spanner-4285F4?style=for-the-badge&logo=googlecloud&logoColor=white)
![Next.js](https://img.shields.io/badge/Next.js-000000?style=for-the-badge&logo=nextdotjs&logoColor=white)
![React](https://img.shields.io/badge/React-149ECA?style=for-the-badge&logo=react&logoColor=white)
![Tailwind CSS](https://img.shields.io/badge/Tailwind%20CSS-0EA5E9?style=for-the-badge&logo=tailwindcss&logoColor=white)
![Tauri](https://img.shields.io/badge/Tauri-FFC131?style=for-the-badge&logo=tauri&logoColor=111111)
![Kotlin](https://img.shields.io/badge/Kotlin-7F52FF?style=for-the-badge&logo=kotlin&logoColor=white)
![Swift](https://img.shields.io/badge/Swift-F05138?style=for-the-badge&logo=swift&logoColor=white)
![Terraform](https://img.shields.io/badge/Terraform-844FBA?style=for-the-badge&logo=terraform&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-2496ED?style=for-the-badge&logo=docker&logoColor=white)

| Layer | Primary Technologies | Notes |
|---|---|---|
| Backend | Go 1.22+, chi router, gRPC, websocket hubs | Domain APIs, realtime fanout, role-scoped control plane |
| Event and Cache Plane | Kafka, transactional outbox relay, Redis | Durable async workflows, invalidate bus, rate-limit primitives |
| Data | Google Cloud Spanner, H3 indexing | Transactional source of truth and geospatial dispatch grouping |
| Web and Desktop | Next.js 15, React 19, Tailwind v4, Tauri 2 | Supplier, retailer, factory, and warehouse operational portals |
| Mobile | Kotlin + Jetpack Compose M3, SwiftUI, Expo | Driver, retailer, payload, factory, and warehouse execution surfaces |
| Infra and Delivery | Terraform, GKE, Cloud LB ring-hash, Docker Compose | Infrastructure-as-code, scale routing, local emulator loops |
| Quality and Testing | go test, go vet, Playwright, Vitest, Gradle, Xcode | Cross-role validation across backend, web, desktop, and mobile |

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
|  |  |  |- maglev-load-balancers.svg
|  |  |  |- omni-hero-banner.svg
|  |  |  |- omni-section-divider.svg
|  |  |  |- omni-code-surface.svg
|  |  |  |- glass-hero-variant-a.svg
|  |  |  |- glass-hero-variant-b.svg
|  |  |  |- techstack-glass-matrix.svg
|  |  |  |- techstack-glass-compact.svg
|  |- infra/
|  |- tests/
```

## Quick Start

![Section Divider](the-lab-monorepo/docs/assets/omni-section-divider.svg)

![Glass Code Surface](the-lab-monorepo/docs/assets/omni-code-surface.svg)

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

![Section Divider](the-lab-monorepo/docs/assets/omni-section-divider.svg)

![Glass Code Surface](the-lab-monorepo/docs/assets/omni-code-surface.svg)

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

![Section Divider](the-lab-monorepo/docs/assets/omni-section-divider.svg)

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

![Section Divider](the-lab-monorepo/docs/assets/omni-section-divider.svg)

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
4. the-lab-monorepo/docs/assets/maglev-load-balancers.svg
5. the-lab-monorepo/docs/assets/omni-hero-banner.svg
6. the-lab-monorepo/docs/assets/omni-section-divider.svg
7. the-lab-monorepo/docs/assets/omni-code-surface.svg
8. the-lab-monorepo/docs/assets/glass-hero-variant-a.svg
9. the-lab-monorepo/docs/assets/glass-hero-variant-b.svg
10. the-lab-monorepo/docs/assets/techstack-glass-matrix.svg
11. the-lab-monorepo/docs/assets/techstack-glass-compact.svg

---

ATOMOS is designed as an execution-grade logistics system, not a demo dashboard. The architecture choices in this repository prioritize deterministic operations, high-scale resilience, and role-accurate workflows from first principles.
