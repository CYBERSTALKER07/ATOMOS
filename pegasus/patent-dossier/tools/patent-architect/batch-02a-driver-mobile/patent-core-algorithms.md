# Batch 02A - Driver Mobile Core Algorithms

## 1. H3 Dispatch With Driver Override Guard

```mermaid
flowchart TD
    A[Order + Driver Telemetry Ingest] --> B[Resolve H3 Cell at Resolution 7]
    B --> C[Build Ring Candidates and Distance Matrix]
    C --> D{Freeze Lock Active?}
    D -->|Yes| E[Keep Current Assignment]
    D -->|No| F[Run Binpack and Route Sequence]
    F --> G{Driver Manual Next Stop?}
    G -->|No| H[Persist Optimized Sequence]
    G -->|Yes| I[Persist Manual Override for Active Scope]
    H --> J[Emit ORDER_ASSIGNED + ROUTE_UPDATED via Outbox]
    I --> J
    E --> K[Emit LOCKED_STATE_NOTIFICATION]
    J --> L[Broadcast to Driver and Supplier Hubs]
```

Driver clients consume the same route lineage while preserving policy-safe manual override behavior.

## 2. Freeze Lock Cooperation Across Human and AI Dispatch

```mermaid
flowchart TD
    A[Operator or Driver Mutation Request] --> B[Acquire Dispatch Freeze Lock]
    B --> C[Persist Lock in DispatchLocks Table]
    C --> D[Emit EventFreezeLockAcquired via Outbox]
    D --> E[AI Worker Drops Entity from Active Queue]
    E --> F[Human Adjustment Window]
    F --> G{Window Completed or TTL Expired?}
    G -->|No| F
    G -->|Yes| H[Emit EventFreezeLockReleased]
    H --> I[AI Worker Re-enqueues Entity]
    I --> J[Driver App Receives Replan Notification]
```

Freeze locks prevent race conditions where optimization would overwrite human intervention mid-mission.

## 3. Idempotent Delivery Completion With Outbox Atomicity

```mermaid
flowchart TD
    A[Driver Submits Completion Payload] --> B[Check Idempotency Key Hash]
    B --> C{Replay Key Exists?}
    C -->|Yes| D[Return Stored Response]
    C -->|No| E[Begin Spanner ReadWriteTransaction]
    E --> F[Validate Geofence + State Transition]
    F --> G[Write Orders and Ledger Mutations]
    G --> H[Write Outbox Payment/Order Events]
    H --> I[Commit Transaction]
    I --> J[Post-Commit Cache Invalidate]
    J --> K[Kafka Relay Publishes Event Set]
    K --> L[Push Notification to Driver and Supplier]
```

This flow guarantees no ghost completion states and no duplicate settlement side effects.

## 4. Predictive Preorder Demand Assist for Driver Readiness

```mermaid
flowchart TD
    A[Historical Orders + Route Telemetry] --> B[Feature Windowing by Region and Time]
    B --> C[Demand Model Inference]
    C --> D[Generate Suggested Preload Buckets]
    D --> E[Check Warehouse and Truck Capacity]
    E --> F{Capacity and Policy Pass?}
    F -->|No| G[Emit Advisory Only]
    F -->|Yes| H[Create Preload Recommendation]
    H --> I[Surface Recommendation in Driver Mission Home]
    I --> J{Driver Accepts?}
    J -->|No| K[Keep Baseline Route]
    J -->|Yes| L[Attach Recommendation to Next Manifest Draft]
```

Preorder assist remains advisory until accepted under role-policy constraints.
