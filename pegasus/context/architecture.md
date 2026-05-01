# V.O.I.D. Architecture & Topology

Use this map to understand how pieces connect before executing any end-to-end task.

```mermaid
graph TD
    %% Core Infrastructure
    Maglev[Maglev Reverse Proxy]
    Router[Chi Router]

    subgraph Data & Eventing
        Spanner[(Cloud Spanner\nHyperscale SQL)]
        Redis[(Redis Cache + Pub/Sub)]
        Kafka[[Managed Kafka\nOutbox Eventing]]
    end

    subgraph Backend [Go 1.22+ Backend]
        Handlers[Domain Handlers]
        WSHubs[WebSocket Hubs]
        OutboxRelay[Outbox Relay]
    end

    subgraph Workers
        AIWorker[AI Worker Choreography]
        Reconciler[Financial Reconciler]
    end

    %% Web Clients
    subgraph Web [Next.js 15 Web Portals]
        AP[Admin Portal]
        FP[Factory Portal]
        WP[Warehouse Portal]
    end

    %% Mobile Clients
    subgraph Mobile [Native Mobile Apps]
        Droid[Android - Jetpack Compose]
        iOS[iOS - SwiftUI]
    end

    %% Terminal Clients
    Terminal[Payload Expo Terminal]

    %% Wiring
    Web --> Maglev
    Mobile --> Maglev
    Terminal --> Maglev
    
    Maglev --> Router
    Maglev -.-> WSHubs
    
    Router --> Handlers
    Handlers -->|RWTxn + OutboxEmit| Spanner
    Handlers -->|Cache.Invalidate| Redis
    WSHubs -->|Fan-Out| Redis
    
    Spanner -.->|Tailed by| OutboxRelay
    OutboxRelay --> Kafka
    
    Kafka --> AIWorker
    Kafka --> Reconciler
```

## Implementation Rules
1. **The Outbox Primitive**: All entity creation and state transitions must write a domain row AND an `OutboxEvents` row in the same Spanner `ReadWriteTransaction`. DO NOT use direct `writer.WriteMessages` for entity CRUD.
2. **Version Gating**: All updates use optimistic concurrency (`If-Match: <version>`). Consumer events use the same version checking.
3. **Priority Guard**: Rate limiting and load shedding are enforced at the Maglev + Router layer. Keep handlers stateless and fast.