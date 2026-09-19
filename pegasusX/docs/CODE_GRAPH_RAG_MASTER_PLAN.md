# PegasusX Code-Graph-RAG Master Plan & Architecture Guide
**Universal Knowledge Graph for Polyglot Microservices, Mobile Apps & Outbox Pipelines**

---

## 1. Executive Summary

PegasusX is an event-driven, polyglot monorepo spanning:
- **Backend**: Go 1.22+ with 29 modular Gin route domains, transactional outbox, and multi-hub WebSockets.
- **Database**: Google Cloud Spanner with 3,700+ lines of DDL strictly keyed by `SupplierId STRING(36) NOT NULL`.
- **Event Bus**: Kafka dual-write domain topics (`pegasusx-orders`, `pegasusx-dispatch`, `pegasusx-realtime`, `pegasusx-main`, etc.).
- **Mobile Clients**: 6 role rows (Supplier, Retailer, Driver, Warehouse, Factory, Payload) on **Android (Kotlin Compose)** and **iOS (Swift/SwiftUI)**.
- **Web & Desktop Portals**: Next.js 15, React, and Tauri desktop apps consuming shared TypeScript SDKs (`@pegasusx/api-core`, `@pegasusx/types`).

Standard AST-based code graphs (vanilla Tree-sitter) fail in polyglot event-driven systems because they only link intra-language symbols (Go to Go, Swift to Swift), completely missing the critical RPC, event, database, and WebSocket seams connecting apps together.

This Master Plan details the architecture and turnkey setup of **Code-Graph-RAG** (`Tree-sitter` + `ast-grep` + `Memgraph` / `Neo4j` + `Model Context Protocol`), stitching together high-level architectural ground truth, micro AST tokens, and **5 foundational cross-boundary relationship seams** so that all AI agents (Cursor, Claude Code, Gemini CLI / Antigravity, Windsurf, Cline, Zed) can perform instantaneous, accurate blast-radius traversals.

---

## 2. The 5 Cross-Boundary Seams (Bridging Polyglot Silos)

```
[Mobile / Web Client App]
       │
       │ (1. CONSUMES_ROUTE)
       ▼
[Route Endpoint (Gin)]
       │
       │ (2. INVOKES_SERVICE / CALLS_REPO)
       ▼
[Domain Service & Repo] ──(3. MUTATES_TABLE / READS_TABLE)──► [Spanner DDL Table]
       │                                                          (SupplierId Key)
       ▼ (4. EMITS_EVENT)
[Transactional Outbox]
       │
       │ (RELAYED_TO_TOPIC)
       ▼
 [Kafka Topic] ──(CONSUMED_BY)──► [Kafka Consumer Worker]
                                           │
                                           │ (5. FANOUT_WS_ROOM)
                                           ▼
                                  [WebSocket Role Hub]
                                           │
                                           │ (RECEIVED_BY)
                                           ▼
                                  [Client App Inbox]
```

### Seam 1: Client API Call $\longrightarrow$ Backend HTTP Route
- **Client Side**:
  - **Android**: Retrofit `@GET/@POST/@PUT/@PATCH/@DELETE` interfaces (`DriverApi.kt`, `FactoryApi.kt`, `PayloadApi.kt`, `PegasusApi.kt`, `SupplierApi.kt`, `WarehouseApi.kt`).
  - **iOS**: Swift URLSession / APIClient instances (`apps/*-ios/**/Services/APIClient.swift`).
  - **Web/Desktop**: `@pegasusx/api-core` and `@pegasusx/api-client` methods (`packages/api-core/index.ts`).
- **Backend Side**:
  - Gin route declarations across the 29 route packages in `apps/backend-go/*routes/*.go` (e.g., `r.POST("/v1/order/create")`, `r.GET("/v1/supplier/orders")`).
- **Graph Edges**:
  `(:ApiClientMethod {platform, role, method_name, target_path})-[:CONSUMES_ROUTE]->(:RouteEndpoint {method, path, package, file})`

### Seam 2: Service/Repository $\longrightarrow$ Spanner Entity Persistence
- **Repository Implementation**: 20 `repository_spanner.go` Go implementations under `apps/backend-go/{domain}/`.
- **Schema**: Table definitions in `apps/backend-go/schema/spanner.ddl` (3,700+ lines).
- **Tenant Isolation**: Flagging tables partitioned by `SupplierId STRING(36) NOT NULL`.
- **Graph Edges**:
  - `(:RepositoryMethod {repo, method, file})-[:MUTATES_TABLE]->(:SpannerTable {name, is_tenant_isolated})`
  - `(:RepositoryMethod {repo, method, file})-[:READS_TABLE]->(:SpannerTable {name, is_tenant_isolated})`

### Seam 3: Transactional Outbox $\longrightarrow$ Domain Event Emitter
- **Code Sites**: `outbox.EmitJSON(txn, events.Event..., ...)` executed within Cloud Spanner `ReadWriteTransaction`.
- **Event Contracts**: Declared in `apps/backend-go/events/events.go` and `contracts/events.schema.json`.
- **Graph Edges**:
  `(:ServiceMethod {service, method, file})-[:EMITS_EVENT]->(:EventDefinition {name})`

### Seam 4: Kafka Ingestion $\longrightarrow$ Downstream Consumer Handler
- **Topic Mapping**: Canonical routing rules declared in `apps/backend-go/events/topic_routing.go` (`pegasusx-orders`, `pegasusx-dispatch`, `pegasusx-realtime`, `logistics.exceptions.v1`, `pegasusx-main`).
- **Consumer Loops**: Kafka worker pool consumer loops in `apps/backend-go/{domain}/consumer.go` and `multi_topic_consumer.go`.
- **Graph Edges**:
  `(:EventDefinition)-[:ROUTED_TO_TOPIC]->(:KafkaTopic)-[:CONSUMED_BY]->(:KafkaConsumer {consumer, package, file})`

### Seam 5: WebSocket Realtime Hub $\longrightarrow$ Role Rooms & Client Inboxes
- **Role Hubs**: 8 dedicated role hubs in `apps/backend-go/ws/hub.go` (`RetailerHub`, `SupplierHub`, `DriverHub`, `PayloadHub`, `WarehouseHub`, `FactoryHub`, `TelemetryHub`, `PlatformAdminHub`).
- **Client App Inboxes**: Native WebSocket receivers across Android, iOS, and Web/Tauri desktops.
- **Graph Edges**:
  `(:KafkaConsumer)-[:FANOUT_WS_ROOM]->(:WSHubRoom {role, hub})-[:RECEIVED_BY]->(:ClientApp {role, platform, path})`

---

## 3. Two-Tier Graph Ingestion Strategy

```
┌─────────────────────────────────────────────────────────────┐
│ Tier 1: Macro Architecture Backbone                        │
│ Source: pegasusX/context/architecture-graph.json           │
│ Nodes:  88 Domain Nodes (:ArchitectureNode)                 │
│ Edges:  160 Architectural Edges (:ARCH_REL)                 │
└──────────────────────────────┬──────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────┐
│ Cross-Boundary Seam Enricher (5 Seams Engine)               │
│ Source: pegasusX/scripts/extract_codegraph_seams.py        │
│ Nodes:  RouteEndpoint, ApiClientMethod, SpannerTable,       │
│         EventDefinition, KafkaTopic, KafkaConsumer, HubRoom │
│ Edges:  CONSUMES_ROUTE, MUTATES_TABLE, EMITS_EVENT,        │
│         ROUTED_TO_TOPIC, CONSUMED_BY, FANOUT_WS_ROOM        │
└──────────────────────────────┬──────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────┐
│ Tier 2: Micro AST Code Graph (Tree-sitter & ast-grep)       │
│ Source: pegasusX/{apps,packages}                            │
│ Nodes:  Function, Method, Class, Struct, Interface          │
│ Edges:  CALLS, CONTAINS, IMPORTS, EXTENDS                   │
└─────────────────────────────────────────────────────────────┘
```

By loading Tier 1 and the Cross-Boundary Seams before or in conjunction with Tier 2 AST parsing, an agent can jump from a high-level domain change directly down to the exact Kotlin function or Spanner index affected.

---

## 4. Setup & Deployment Guide

### Prerequisites
- **Docker & Docker Compose**: To host Memgraph Platform.
- **Python 3.12+** & **uv**: Package management and execution.
- **System Tools**: `cmake`, `ripgrep`, `ast-grep` (e.g. `brew install cmake ripgrep ast-grep`).

### Step 1: Launch Memgraph Platform
We provide a dedicated Docker Compose configuration in `pegasusX/infra/docker-compose.codegraph.yml`:

```bash
cd pegasusX
make codegraph-up
```

This starts:
- **Bolt Cypher Protocol**: `localhost:7687`
- **Memgraph Lab UI**: `http://localhost:3000`

### Step 2: Install Python Dependencies & Parser Grammars
Using `uv`:
```bash
uv pip install "code-graph-rag[treesitter-full,ast-grep]" neo4j gqlalchemy
```

### Step 3: Noise Reduction & Ignore Rules
Both `.codegraphignore` (in workspace root) and `pegasusX/.codegraphignore` strictly exclude:
- Legacy archived reference code (`pegasus/`, `archive/`).
- Build caches (`node_modules/`, `.gradle/`, `.next/`, `Pods/`, `DerivedData/`, `.turbo/`, `build/`, `dist/`).
- Binary assets, database emulator state, and test coverage snapshots.

### Step 4: Ingest Backbone & Seams
Run the automated seed and seam extractors:
```bash
# Export and inspect Cypher files offline
make codegraph-export

# Ingest into live Memgraph instance
make codegraph-seed
make codegraph-seams
```

### Step 5: Verify Graph Topology
Run the verification test suite:
```bash
make codegraph-verify
```
For offline validation without a running database container:
```bash
python3 pegasusX/scripts/verify_codegraph.py --offline
```

---

## 5. Universal MCP Server Setup for All IDEs & Agents

All IDEs connect to the knowledge graph via the universal launcher script:
[`pegasusX/scripts/run_mcp_server.sh`](../scripts/run_mcp_server.sh)

### 1. Cursor IDE
Configured in [`.cursor/mcp.json`](../../.cursor/mcp.json):
```json
{
  "mcpServers": {
    "code-graph": {
      "command": "bash",
      "args": ["/Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/run_mcp_server.sh"],
      "env": {
        "TARGET_REPO_PATH": "/Users/shakhzod/Desktop/V.O.I.D/pegasusX",
        "MEMGRAPH_BOLT_URL": "bolt://localhost:7687",
        "CYPHER_PROVIDER": "google",
        "CYPHER_MODEL": "gemini-2.0-flash"
      }
    }
  }
}
```

### 2. Claude Code CLI
Register using the Claude Code CLI:
```bash
claude mcp add code-graph -- bash /Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/run_mcp_server.sh
```
Or add directly to `~/.claude.json`:
```json
{
  "mcpServers": {
    "code-graph": {
      "command": "bash",
      "args": ["/Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/run_mcp_server.sh"]
    }
  }
}
```

### 3. Gemini CLI / Antigravity
Configured in `~/.gemini/antigravity-cli/mcp/code-graph.json` or active project settings:
```json
{
  "name": "code-graph",
  "command": "bash",
  "args": ["/Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/run_mcp_server.sh"]
}
```

### 4. Windsurf (Codeium)
Add to `~/.codeium/windsurf/mcp_config.json`:
```json
{
  "mcpServers": {
    "code-graph": {
      "command": "bash",
      "args": ["/Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/run_mcp_server.sh"],
      "env": {
        "TARGET_REPO_PATH": "/Users/shakhzod/Desktop/V.O.I.D/pegasusX",
        "MEMGRAPH_BOLT_URL": "bolt://localhost:7687"
      }
    }
  }
}
```

### 5. Zed Editor
Configured in [`.zed/settings.json`](../../.zed/settings.json):
```json
{
  "context_servers": {
    "code-graph": {
      "command": "bash",
      "args": ["/Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/run_mcp_server.sh"]
    }
  }
}
```

### 6. Cline / Roo Code
Add to `cline_mcp_settings.json`:
```json
{
  "mcpServers": {
    "code-graph": {
      "command": "bash",
      "args": ["/Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/run_mcp_server.sh"]
    }
  }
}
```

---

## 6. Agent Blast-Radius Query Recipes (Cypher)

Once indexed, any agent can execute natural-language queries translated to Cypher via MCP:

### Recipe 1: Full Mobile-to-Backend Blast Radius
*Question: "If I modify the driver manifest endpoint, what mobile clients and backend handlers are affected?"*
```cypher
MATCH (c:ApiClientMethod)-[:CONSUMES_ROUTE]->(r:RouteEndpoint)
WHERE r.path CONTAINS "/manifest"
RETURN c.platform, c.role, c.method_name, r.method, r.path, r.handler, r.file;
```

### Recipe 2: Event Outbox-to-Topic-to-Consumer Chain
*Question: "What happens after an Order is created? Trace from service method through Kafka down to consumer workers."*
```cypher
MATCH (s:ServiceMethod)-[:EMITS_EVENT]->(e:EventDefinition)-[:ROUTED_TO_TOPIC]->(t:KafkaTopic)-[:CONSUMED_BY]->(c:KafkaConsumer)
WHERE e.name CONTAINS "ORDER"
RETURN s.service, s.method, e.name, t.name, c.consumer, c.file;
```

### Recipe 3: Multi-Tenancy Compliance Audit
*Question: "Which Spanner tables mutated by repository methods lack the SupplierId tenant isolation key?"*
```cypher
MATCH (r:RepositoryMethod)-[:MUTATES_TABLE]->(t:SpannerTable)
WHERE t.is_tenant_isolated = false
RETURN DISTINCT t.name, collect(r.repo + '.' + r.method) AS mutators;
```

### Recipe 4: End-to-End Reactive WebSocket Fanout
*Question: "Show the real-time event pipeline reaching the retailer mobile apps."*
```cypher
MATCH (app:ClientApp {role: 'retailer'})<-[:RECEIVED_BY]-(hub:WSHubRoom)<-[:FANOUT_WS_ROOM]-(c:KafkaConsumer)<-[:CONSUMED_BY]-(t:KafkaTopic)
RETURN app.platform, app.path, hub.hub, c.consumer, t.name;
```

---

---

## 7. Visual Graph Exploration Interfaces

PegasusX provides two complementary visual graph interfaces:

### 1. PegasusX CodeGraph Studio (Custom Dashboard)
- **URL:** [http://localhost:3001](http://localhost:3001)
- **Launcher:** `make codegraph-ui` or `bash scripts/run_codegraph_ui.sh`
- **Engine:** Starlette + Uvicorn + Cytoscape.js + TailwindCSS.
- **Capabilities:**
  - **5-Seams Lens Filters:** Switch between Seam 1 (Client ➔ Route), Seam 2 (Repo ➔ Spanner), Seam 3 (Outbox), Seam 4 (Kafka), Seam 5 (WebSockets), and Architecture Backbone.
  - **Multi-Tenancy Auditor:** Visual `Tenant Isolated` badges on Spanner tables verifying `SupplierId` partitioning.
  - **Real-time Blast Radius Simulator:** Select any node (route, entity, or hub) and simulate upstream/downstream ripple effects across 2–3 hops with automatic node isolation and dimming.
  - **Role-Row Parity Selectors:** Filter by Supplier, Retailer, Driver, Warehouse, Factory, Payload, or Admin.
  - **Cypher Sandbox & Preset Library:** Built-in query presets with live graph visualization.

### 2. Memgraph Lab (Standard Explorer)
- **URL:** [http://localhost:3000](http://localhost:3000)
- **Included with:** `make codegraph-up`
- **Capabilities:** Raw schema inspection, ad-hoc Cypher console, query performance metrics, and database configuration.

---

## 8. Operational Lifecycle & Maintenance

| Command | Action | Frequency |
|---|---|---|
| `make codegraph-up` | Boots Memgraph container with volume persistence | Once per development session |
| `make codegraph-ui` | Launches PegasusX CodeGraph Studio at `http://localhost:3001` | For interactive visual inspection & blast radius |
| `make codegraph-export` | Re-generates Cypher scripts offline from codebase | Pre-commit / CI gate |
| `make codegraph-seed` | Ingests `architecture-graph.json` | When system architecture changes |
| `make codegraph-seams` | Re-scans all 5 polyglot seams and updates edges | When endpoints, DDL, or events change |
| `make codegraph-verify` | Executes graph assertions and blast radius checks | In CI pipeline or regression tests |
| `make codegraph-down` | Gracefully stops container and preserves volume | End of day / resource cleanup |
