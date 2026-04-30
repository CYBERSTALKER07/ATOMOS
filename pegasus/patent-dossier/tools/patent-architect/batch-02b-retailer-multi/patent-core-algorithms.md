# Batch 02B - Retailer Multi-Surface Core Algorithms

## 1. H3 Dispatch Visibility for Retailer Tracking

```mermaid
flowchart TD
    A[Order Accepted by Retailer] --> B[Assign Dispatch Cluster by H3 Cell]
    B --> C[Generate Route and ETA Envelope]
    C --> D[Persist to Orders and Route Tables]
    D --> E[Emit ORDER_ASSIGNED + ROUTE_UPDATED via Outbox]
    E --> F[Push to Retailer Web, Android, iOS Tracking Surfaces]
    F --> G{Retailer Requests Detail?}
    G -->|Yes| H[Open Order Detail or Tracking Inspector]
    G -->|No| I[Keep Summary Feed]
```

Retailer clients receive a single route truth from backend while preserving per-platform presentation differences.

## 2. Freeze Lock-Aware Manual Intervention

```mermaid
flowchart TD
    A[Retailer Raises Exception or Change Request] --> B[Acquire Freeze Lock on Target Entity]
    B --> C[Persist Lock + Emit Lock Event]
    C --> D[AI Worker Stops Auto-Reassignment]
    D --> E[Operator and Retailer Coordinate Resolution]
    E --> F{Resolved?}
    F -->|No| E
    F -->|Yes| G[Release Lock + Emit Release Event]
    G --> H[Optimizer Re-enters Normal Mode]
```

This pattern prevents auto-dispatch from overriding in-flight exception handling.

## 3. Idempotent Checkout and Payment Finalization

```mermaid
flowchart TD
    A[Retailer Submits Checkout or Payment Action] --> B[Validate Idempotency Key]
    B --> C{Duplicate Request?}
    C -->|Yes| D[Return Original Response]
    C -->|No| E[Begin ReadWriteTransaction]
    E --> F[Validate Order State and Amount]
    F --> G[Write Order Mutation + Ledger Entries]
    G --> H[Write Outbox Payment/Order Events]
    H --> I[Commit]
    I --> J[Invalidate Cache]
    J --> K[Relay Events to Kafka + WS + Notification]
```

No duplicate request can produce duplicate financial effects.

## 4. Predictive Preorder and Procurement Assist

```mermaid
flowchart TD
    A[Retailer History + Seasonality Features] --> B[Forecast Inference]
    B --> C[Generate Suggested Procurement Basket]
    C --> D[Display Suggestions in Procurement Surface]
    D --> E{Retailer Accepts Suggested Lines?}
    E -->|No| F[Save Draft and Continue Manual Edit]
    E -->|Yes| G[Merge Suggested Lines into Cart]
    G --> H[Checkout Flow]
    F --> H
```

Prediction remains assistive; retailer remains the approving actor.
