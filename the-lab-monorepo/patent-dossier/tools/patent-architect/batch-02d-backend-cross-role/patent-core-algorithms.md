# Batch 02D - Cross-Role Backend Core Algorithms

## 1. H3 Dispatch and Inter-Node Assignment Across Factory and Warehouse

```mermaid
flowchart TD
    A[Warehouse Demand Signal] --> B[Factory Supply Candidate Generation]
    B --> C[H3 Cell and Ring Candidate Expansion]
    C --> D[Dispatch Binpack and Manifest Split]
    D --> E[Assign Driver/Truck by HomeNode Scope]
    E --> F[Persist Orders, Routes, Manifests]
    F --> G[Emit ORDER_ASSIGNED and ROUTE_CREATED via Outbox]
    G --> H[Fanout to Factory, Warehouse, and Driver Surfaces]
```

Cross-role dispatch preserves home-node constraints while coordinating supply movement from factories to warehouses.

## 2. Freeze Lock Protocol for Human Overrides in Cross-Role Operations

```mermaid
flowchart TD
    A[Manual Mutation Request from Factory/Warehouse/Supplier] --> B[Acquire Dispatch Freeze Lock]
    B --> C[Persist Lock + Emit Lock Event]
    C --> D[AI Worker Pauses Affected Entity]
    D --> E[Role-Scoped Human Adjustment]
    E --> F{Resolved or TTL Expired?}
    F -->|No| E
    F -->|Yes| G[Release Lock + Emit Release Event]
    G --> H[AI Worker Resumes Scheduling]
```

Freeze locks provide deterministic control transfer between automation and human operators.

## 3. Idempotent Mutations With Transactional Outbox for Cross-Role APIs

```mermaid
flowchart TD
    A[Cross-Role Mutating API Call] --> B[Auth Scope Resolution from Claims]
    B --> C[Idempotency Guard Check]
    C --> D{Replay Key Exists?}
    D -->|Yes| E[Replay Prior Response]
    D -->|No| F[ReadWriteTransaction]
    F --> G[Validate Version and Role Constraints]
    G --> H[Write Domain Rows]
    H --> I[Write Outbox Event Rows]
    I --> J[Commit]
    J --> K[Cache Invalidate]
    K --> L[Kafka Relay + WS Broadcast]
```

This pattern prevents ghost entities and duplicate side effects under retries.

## 4. Predictive Demand and Replenishment Engine Across Nodes

```mermaid
flowchart TD
    A[Historical Orders and Warehouse Loads] --> B[Feature Extraction and Time Windowing]
    B --> C[Demand Forecast Inference]
    C --> D[Look-Ahead and Pull-Matrix Recommendation]
    D --> E{Capacity + Policy + Node Scope Pass?}
    E -->|No| F[Advisory Output Only]
    E -->|Yes| G[Create Replenishment Recommendation]
    G --> H[Surface to Factory and Warehouse Insights]
    H --> I{Operator Accepts?}
    I -->|No| J[Archive Recommendation]
    I -->|Yes| K[Trigger Replenishment Workflow]
```

Forecast outputs remain assistive and policy-gated before becoming operational mutations.
