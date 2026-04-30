# Patent Core Algorithms

Batch: Batch 01 - Supplier Web Core
Figure Namespace: Global

## Figure B1-A: H3 Hexagon Grid Dispatch System
Caption: Hex-cell indexed dispatch planning that transforms order geography into capacity-constrained route assignments and draft manifests.

```mermaid
flowchart TD
  A[Order Intake] --> B[Normalize Coordinates]
  B --> C[Assign Hex Cell at Standard Resolution]
  C --> D[Compute Neighbor Rings for Coverage]
  D --> E[Group Orders by Cell Cohesion]
  E --> F[Apply Capacity Buffer Rule]
  F --> G[Smallest Fit Vehicle Selection]
  G --> H[Split Oversized Loads]
  H --> I[Persist Draft Manifest Records]
  I --> J[Emit Durable Manifest Draft Signal]
  J --> K[Expose Route Data to Operations Surfaces]

  F --> L{Freeze Lock Active}
  L -- Yes --> M[Hold Assignment from Active Dispatch Wave]
  L -- No --> G
```

Operational Notes:
- Spatial indexing uses hex-cell identity to avoid broad-distance scans.
- Vehicle matching follows a safety buffer policy before assignment finalization.
- Oversized shipments are segmented into manageable chunks before persistence.

## Figure B1-B: Freeze Lock and Manual Override State Machine
Caption: Operator intervention lock workflow that prevents automatic reassignment during manual dispatch governance.

```mermaid
flowchart TD
  A[Operator Requests Manual Control] --> B[Validate Role Scope]
  B --> C[Create Dispatch Lock Record]
  C --> D[Emit Dispatch Lock Acquired Signal]
  D --> E[Emit Freeze Lock Acquired Signal]
  E --> F[Autonomous Worker Drops Frozen Entity]
  F --> G[Manual Dispatch Operations Continue]
  G --> H[Operator Releases Lock]
  H --> I[Mark Lock as Released]
  I --> J[Emit Freeze Lock Released Signal]
  J --> K[Autonomous Worker Re-queues Entity]

  C --> L{Existing Active Lock}
  L -- Yes --> M[Reject with Conflict Response]
  L -- No --> D
```

Operational Notes:
- Scope derivation binds lock authority to authenticated node context.
- Manual lock state preserves human authority during route sequencing overrides.
- Release signaling restores optimization eligibility without historical loss.

## Figure B1-C: Payloader Idempotency and Transactional Outbox
Caption: Dual integrity path where mutating actions are replay-safe at the interface layer and atomically published through durable event relay.

```mermaid
flowchart TD
  A[Incoming Mutating Request] --> B{Idempotency Key Present}
  B -- No --> C[Execute Request Directly]
  B -- Yes --> D[Check Cached Response]
  D -- Hit --> E[Replay Cached Response]
  D -- Miss --> F[Acquire Processing Lock]
  F --> G[Execute Mutation]
  G --> H[Write Domain Records in Transaction]
  H --> I[Append Outbox Event in Same Transaction]
  I --> J[Commit Transaction]
  J --> K[Cache Successful Response]
  K --> L[Background Relay Reads Unpublished Outbox]
  L --> M[Publish to Event Stream with Aggregate Key]
  M --> N[Mark Outbox Event Published]

  F --> O{Lock Acquired}
  O -- No --> P[Return Duplicate In-Progress Conflict]
  O -- Yes --> G
```

Operational Notes:
- Interface deduplication prevents duplicate mutation side effects from retries.
- Outbox co-commit prevents divergence between state storage and event publication.
- Aggregate-key publication maintains per-entity ordering in downstream consumers.

## Figure B1-D: Predictive Preorder Demand Engine
Caption: Forecast-informed replenishment workflow that transforms waiting demand signals into preemptive internal transfer orders.

```mermaid
flowchart TD
  A[Forecast Intake and Waiting Prediction Set] --> B[Apply Supplier Horizon Window]
  B --> C[Aggregate Predicted Demand by Product]
  C --> D[Read Current Warehouse Inventory]
  D --> E[Compute Projected Post-Demand Stock]
  E --> F{Projected Stock Breaches Safety Level}
  F -- No --> G[No Transfer Created]
  F -- Yes --> H[Compute Deficit Quantity]
  H --> I[Select Optimal Factory by Lane and Mode]
  I --> J[Create Internal Transfer Draft]
  J --> K[Attach Transfer Line Items]
  K --> L[Emit Replenishment Transfer Signal]
  L --> M[Increment Factory Load and Traceability Link]

  B --> N[Look-Ahead Shadow Demand Scan]
  N --> C
```

Operational Notes:
- Safety horizon and threshold logic are evaluated before physical shortage occurs.
- Transfer generation remains node-aware through lane selection and mode constraints.
- Linked traceability supports downstream auditing from demand signal to replenishment order.
