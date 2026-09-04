---
name: codegraph-deep-audit
description: >
  Autonomous static analysis and deep code auditing using the PegasusX CodeGraph
  (64,432 Nodes / 176,106 Relationships across Tree-sitter AST and 5 Cross-Boundary Seams).
  Use before making changes or when auditing:
  1. Blast radius calculation before editing any function, class, or file.
  2. Spanner multi-tenancy isolation audit (identifying tables lacking SupplierId partitioning).
  3. Client-to-backend contract drift audit (detecting 404 hazards in Android Kotlin, iOS Swift, and Next.js).
  4. Dead/unconsumed backend routes audit.
  5. Kafka event pipeline integrity (unconsumed topics and outbox black holes).
version: 1.0.0
---

# CodeGraph Deep Audit — AI Agent Standard Operating Procedure

The PegasusX CodeGraph runs locally in Memgraph (`bolt://localhost:7687`) and provides deep architectural and syntactic visibility across Go backend, Android Kotlin, iOS Swift, Next.js TypeScript, Spanner DDL, and Kafka outbox streams.

## 0. The Two-Tier Verification Standard (CodeGraph + Targeted Raw Reading)

Industrial code intelligence requires **combining both tiers**:

| Tier | Role | What It Solves | What It Cannot Do |
|---|---|---|---|
| **Tier 1: Bazel/Kythe CodeGraph (Global Radar)** | Macro-level static & relational index across all 1,551 files | Computes multi-hop transitive call chains, Bazel test targets (`rdeps`), SQL tenancy taint, and cross-language contract drift in <50ms with 0 token overhead. | Cannot evaluate runtime conditions (`if !s.allocationRequired`), state checks, or error formatting. |
| **Tier 2: Targeted Raw Reading (Local Microscope)** | Micro-level source code inspection | Reads the exact guard clauses, error handling, Spanner RW transaction closures, and business logic on the files identified in Tier 1. | Blind to indirect transitive callers across thousands of files; manual grep has >60% false negative rate. |

**Standard Operating Procedure for Every AI Agent:**
1. **Always run Tier 1 FIRST:** Run `advanced_codegraph_analyzer.py --blast-radius <symbol>` or `bazel_target_graph.py --query-rdeps <target>` to discover all direct and indirect callers and affected test targets.
2. **Always run Tier 2 SECOND:** Open and raw-read the exact files identified by Tier 1. Inspect the logic, guard clauses, and transaction boundaries.
3. **Re-read after editing:** Verify your changes against the contracts and re-run Tier 1 to ensure zero introduced violations.

---

## 1. Mandatory Blast-Radius Check (Pre-Edit Gate)

Before modifying any Go function, repository method, mobile API client, or core file, an AI agent **MUST** run a blast-radius audit to understand upstream callers and downstream side-effects.

### A. Audit a Specific Symbol
```bash
python3 pegasusX/scripts/audit_codegraph.py --symbol <SymbolName> --json
```
**Example:**
```bash
python3 pegasusX/scripts/audit_codegraph.py --symbol AllocateOrder --json
```
**Agent Interpretations:**
* `total_inbound_callers`: How many call sites across the codebase depend on this function.
* `sample_callers`: List of `{file, line, name}` of callers. Re-verify every caller after editing!
* `associated_routes`: Backend Chi router paths powered by this function.
* `associated_tables`: Spanner tables read or mutated by this function.

### B. Audit an Entire File
```bash
python3 pegasusX/scripts/audit_codegraph.py --file <RelativeFilePath> --json
```
**Example:**
```bash
python3 pegasusX/scripts/audit_codegraph.py --file apps/backend-go/allocation/service.go --json
```
**Agent Interpretations:**
* `exported_symbols`: All functions, methods, classes, and types in the file.
* `external_callers`: Count of external files invoking symbols in this file.
* If `risk_level` is `CRITICAL` or `HIGH`, you must run package unit tests (`go test ./...`) after making changes.

---

## 2. Multi-Tenancy & Tenant Leakage Audit

PegasusX is a multi-supplier, local-first enterprise OS. Every tenant-scoped table **MUST** be partitioned by `SupplierId`.

### Run Multi-Tenancy Check:
```bash
python3 pegasusX/scripts/audit_codegraph.py --cypher "
MATCH (t:SpannerTable)
WHERE t.is_tenant_isolated = false
RETURN t.name AS table, t.file AS file
ORDER BY t.name
" --json
```

**Rule for AI Agents:**
* If introducing a new table in `schema/spanner.ddl`:
  * Operational tables (orders, stock, manifests, invoices, drivers) **MUST** include `SupplierId STRING(36) NOT NULL` and be in primary key or secondary index.
  * Only global catalogs (e.g. `GlobalProducts`, `SystemConfigs`) are permitted to be non-isolated.

---

## 3. Client-to-Backend Contract Parity Audit

To eliminate runtime 404 errors in mobile apps and web portals:

### Query Broken Client Endpoints:
```bash
python3 pegasusX/scripts/audit_codegraph.py --cypher "
MATCH (c:ApiClientMethod)
OPTIONAL MATCH (c)-[rel:CONSUMES_ROUTE]->(r:RouteEndpoint)
WITH c, count(rel) AS matched
WHERE matched = 0
RETURN c.platform AS platform, c.file AS file, c.method_name AS method, c.endpoint_template AS endpoint
ORDER BY c.platform, c.method_name
" --json
```

**Rule for AI Agents:**
* When altering an HTTP route in `apps/backend-go/*routes/routes.go`:
  * You MUST update the shared client in `packages/api-client` and all role-row apps (Supplier, Retailer, Driver, Warehouse, Factory, Payload).
  * Run the query above to confirm `matched > 0`.

---

## 4. Kafka Outbox Event Pipeline Audit

Ensure domain events written to the transactional outbox reach active consumers:

### Check Unconsumed Topics:
```bash
python3 pegasusX/scripts/audit_codegraph.py --cypher "
MATCH (t:KafkaTopic)
OPTIONAL MATCH (o:OutboxEmitter)-[r1:ROUTED_TO_TOPIC]->(t)
OPTIONAL MATCH (t)-[r2:CONSUMED_BY]->(c:KafkaConsumer)
WITH t, count(r1) AS emitters, count(r2) AS consumers
WHERE consumers = 0
RETURN t.name AS topic, emitters, consumers
" --json
```

**Rule for AI Agents:**
* If adding an outbox event in a Go service (`outbox.EmitJSON(...)`), ensure:
  1. The outbox processor routes it to a valid Kafka topic in `cmd/outbox-dispatcher/`.
  2. A consumer worker is registered in `apps/ai-worker/` or backend workers.
  3. Realtime WebSocket fanout is mapped if frontend/mobile roles must react live.

---

## 5. Advanced Compiler-Grade Static Analysis (Big-Tech Class)

To run the advanced static analysis engine (Spanner SQL Taint, Outbox Atomicity, Field-Level DTO Drift, and NetworkX Centrality):

```bash
make codegraph-advanced-audit
# Generates pegasusX/docs/ADVANCED_CODE_AUDIT_REPORT.md
```

### Targeted Advanced Suites:

1. **Spanner SQL Tenant Taint Analysis**:
   ```bash
   python3 pegasusX/scripts/advanced_codegraph_analyzer.py --suite sql_tenancy --json
   ```
   *Parses every SQL statement in Go repositories to ensure `WHERE SupplierId = @SupplierId` is parameterized on operational tables.*

2. **Transactional Outbox Dual-Write Verifier**:
   ```bash
   python3 pegasusX/scripts/advanced_codegraph_analyzer.py --suite outbox_atomicity --json
   ```
   *Verifies every `spanner.ReadWriteTransaction` has an atomic `outbox.Emit` / `TxnBuffer` to prevent dual-write state desync.*

3. **Field-Level Cross-Language DTO Contract Drift**:
   ```bash
   python3 pegasusX/scripts/advanced_codegraph_analyzer.py --suite field_drift --json
   ```
   *Compares Go struct JSON tags against TypeScript client interfaces (`packages/types`) to catch field omissions.*

4. **NetworkX Architecture Centrality & Choke-Points**:
   ```bash
   python3 pegasusX/scripts/advanced_codegraph_analyzer.py --suite centrality --json
   ```
   *Computes Betweenness Centrality and circular dependency loops across the monorepo.*

5. **Transitive Blast Radius Cone (Multi-Hop)**:
   ```bash
   python3 pegasusX/scripts/advanced_codegraph_analyzer.py --blast-radius <SymbolName> --depth 3 --json
   ```
   *Calculates the full multi-hop reachability cone of any function or route.*
