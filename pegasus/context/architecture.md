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

    %% Agent Context Tooling
    subgraph AgentContext [Agent Context Tooling]
        Copilot[GitHub Copilot Agent]
        Gemini[Gemini Agent]
        MCPServer[AST MCP Server]
        ASTEngine[Local AST Symbol Graph Engine]
        ArchDocs[Architecture Docs + JSON Graph]
    end

    %% Wiring
    Web --> Maglev
    Mobile --> Maglev
    Terminal --> Maglev

    Copilot --> MCPServer
    Gemini --> MCPServer
    MCPServer --> ASTEngine
    Copilot --> ArchDocs
    Gemini --> ArchDocs
    ArchDocs -.Sync Contract.-> ASTEngine
    
    Maglev --> Router
    Maglev -.-> WSHubs
    
    Router --> Handlers
    Handlers -->|RWTxn + OutboxEmit| Spanner
    Handlers -->|Cache.Invalidate| Redis
    WSHubs -->|Fan-Out| Redis
    
    Spanner -.->|Tailed by| OutboxRelay
    OutboxRelay --> Kafka

    ASTEngine -.Definition + Usage Queries.-> Handlers
    ASTEngine -.Definition + Usage Queries.-> AP
    ASTEngine -.Definition + Usage Queries.-> FP
    ASTEngine -.Definition + Usage Queries.-> WP
    
    Kafka --> AIWorker
    Kafka --> Reconciler
```

## Implementation Rules
1. **The Outbox Primitive**: All entity creation and state transitions must write a domain row AND an `OutboxEvents` row in the same Spanner `ReadWriteTransaction`. DO NOT use direct `writer.WriteMessages` for entity CRUD.
2. **Version Gating**: All updates use optimistic concurrency (`If-Match: <version>`). Consumer events use the same version checking.
3. **Priority Guard**: Rate limiting and load shedding are enforced at the Maglev + Router layer. Keep handlers stateless and fast.

## Agent Context Rules
1. **MCP First**: Before any technical task, call native MCP tools `void_ast_index`, `void_ast_definition`, `void_ast_usages`, and `void_ast_graph`.
2. **Script Fallback**: If MCP tools are unavailable, run `npm --prefix pegasus run ast:index`, `ast:def`, `ast:refs`, and `ast:graph` for the target symbol.
3. **Dual Read Mandatory**: Agent retrieval is complete only after symbol graph queries plus architecture docs and technology inventory docs are read.
4. **Prompt Verification Gate**: Before implementation, classify request risk (`safe`, `risky`, `production-breaking`, `scope-conflict`). If not `safe`, propose the safer approach first.
5. **Dual Sync Mandatory**: If architecture, dependencies, services, or integrations change, update the full sync set in one change set:
    - `.github/ACT.md`
    - `.github/copilot-instructions.md`
    - `.github/gemini-instructions.md`
    - `pegasus/context/architecture.md`
    - `pegasus/context/architecture-graph.json`
    - `pegasus/context/technology-inventory.md`
    - `pegasus/context/technology-inventory.json`
6. **ACT Mandatory**: Follow `.github/ACT.md`; challenge unsafe plans and enforce Spanner, Kafka, Redis, Terraform, Maglev, and hyper-scale readiness checks before execution.