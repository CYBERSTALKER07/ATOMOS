# Role 9 Specification: Enterprise SRE & FinOps Lead

> **Role Number**: Role 9 of 9  
> **Persona ID**: `ROLE_SRE_FINOPS` / `ROLE_ENTERPRISE_SRE`  
> **Domain Scope**: Multi-Cell Cloud Reliability, Spanner Distributed Topology, Kafka Partition Lag & Event Mesh, Outbox Dead-Letter Queue (DLQ) Diagnostic Replay, macOS Native Control Tower, Dynamic FinOps Cost Attribution & Chaos Resilience  
> **Authoritative Backend Seams**:  
> - Spanner Outbox & DLQ Engine: [`apps/backend-go/outbox`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/outbox)  
> - DLQ Replay CLI Tool: [`apps/backend-go/cmd/replay-dlq`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/cmd/replay-dlq)  
> - Schema Drift Detection Engine: [`apps/backend-go/schemadrift`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/schemadrift)  
> - Kafka Consumer Mesh & Worker Pools: [`apps/backend-go/kafka/workerpool`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/kafka/workerpool)  
> - Control Tower Telemetry Routes: [`apps/backend-go/controltowerroutes`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/controltowerroutes)  
> - Infrastructure Health Probes: [`apps/backend-go/infraroutes`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/infraroutes)  
> **Frontend & Native Surfaces**:  
> - macOS Native Control Tower (Swift 6 / SwiftUI): [`apps/control-tower-macos`](file:///Users/shakhzod/Desktop/pegasusx/apps/control-tower-macos)  
> - Admin Portal FinOps Dashboard: [`apps/admin-portal/app/finops`](file:///Users/shakhzod/Desktop/pegasusx/apps/admin-portal/app/finops)  
> - Admin Portal Telemetry Terminal: [`apps/admin-portal/app/telemetry`](file:///Users/shakhzod/Desktop/pegasusx/apps/admin-portal/app/telemetry)  
> - Desktop Client Runtime (Tauri 2 / Rust): [`apps/admin-portal/src-tauri`](file:///Users/shakhzod/Desktop/pegasusx/apps/admin-portal/src-tauri)  
> **Primary Cloud Infrastructure Components**: Google Cloud Spanner (Multi-Cell Instance `pegasusx-production`), Managed Kafka Mesh, Redis Sentinel Cluster, GKE Autopilot Workload Pools  

---

## 1. Executive Summary & Operational Charter

The **Enterprise SRE & FinOps Lead (Role 9)** safeguards the technical reliability, distributed ACID data plane integrity, and cloud unit-economics of the PegasusX multi-cell operating system. While operational roles focus on individual physical shipments, the Enterprise SRE oversees the entire underlying compute fabric, distributed event brokers, multi-region replication rings, and multi-tenant cloud cost waterfalls.

The Enterprise SRE role is founded upon four core mandates:
1. **Zero-Data-Loss ACID Data Plane**: Monitoring Cloud Spanner commit timestamps, read/write latency percentiles (P99 < 15ms), transaction abort rates, lock wait durations, and multi-region replication lags.
2. **Event Mesh Health & Outbox Zero-Lag**: Guaranteeing that every committed state mutation in Cloud Spanner is published to Kafka via the Transactional Outbox pattern with zero event loss and sub-second delivery to downstream consumers.
3. **Poison Payload Isolation & Atomic DLQ Replay**: Quarantining unprocessable messages into `OutboxDeadLetters`, generating encrypted diagnostic dump bundles, fixing root causes, and executing atomic, in-order message replays without database corruption.
4. **Tenant-Attributed FinOps Waterfall**: Dynamically allocating multi-cloud infrastructure spend (Spanner processing units, network egress, Redis memory, Kafka compute) to individual tenant organizations, reconciling cloud burn rates against dynamically configured platform take-rate inflows.

---

## 2. Account Setup, Identity, Hardware & Provisioning

```
[Workstation Security Enclave] -> [Multi-Cloud IAM Federation] -> [Native macOS App Setup] -> [Emergency Break-Glass Enrollment]
```

### 2.1. SRE Identity, Enclave & IAM Federation
1. **Administrative Provisioning**: The Lead SRE account is enrolled via enterprise identity federation (Okta / Google Cloud Identity) with role `ROLE_SRE_FINOPS`.
2. **Apple Silicon Secure Enclave & Hardware Key Binding**: Authentication credentials and mTLS certificates are stored in the Apple macOS Keychain / Secure Enclave. Mutations require biometric Touch ID or physical YubiKey 5C NFC hardware authentication.
3. **Multi-Cloud Least-Privilege IAM Roles**:
   - Google Cloud IAM: `roles/spanner.databaseUser`, `roles/monitoring.viewer`, `roles/logging.viewer`. Emergency break-glass roles (`roles/spanner.databaseAdmin`) require dual-control approval from the Platform Owner.
   - Kafka IAM: SASL/SCRAM-SHA-512 authenticated client credentials with read/write access to system outbox and DLQ topics.

### 2.2. Native macOS & Desktop Control Tower Installation
1. **Native macOS Application Deployment**: The SRE installs the native macOS Control Tower client ([`apps/control-tower-macos`](file:///Users/shakhzod/Desktop/pegasusx/apps/control-tower-macos)) built natively with Swift 6, SwiftUI, Swift Charts, and native AppKit menu bar extras.
2. **MenuBarExtra Status Pill Initialization**: Configures the persistent macOS MenuBarExtra indicator pill (`MenuBarStatusView.swift`), displaying a real-time color-coded health dot (Green = Nominal, Yellow = Warning, Red = Breach) and active incident counter.
3. **Cross-Platform Tauri 2 Desktop Standby**: Deploys the cross-platform Rust/Tauri 2 desktop client for cross-platform Linux/Windows operational parity.
4. **Local Telemetry Ring-Buffer Configuration**: Configures an in-memory 50-event ring buffer connected to backend WebSockets (`/v1/control-tower/stream`) to stream high-velocity error and warning events directly to the desktop notification center.

---

## 3. Comprehensive Role User Flow & Operational Cadence

```mermaid
sequenceDiagram
    autonumber
    actor SRE as Enterprise SRE & FinOps Lead
    participant MenuBar as macOS MenuBarExtra Pill
    participant App as Native macOS Control Tower
    participant Edge as Control Tower Gateway
    participant Spanner as Cloud Spanner Multi-Cell
    participant Kafka as Kafka Event Mesh
    participant FinOps as FinOps Settlement Engine

    %% Phase 1: Morning MenuBar Glance & Pulse Audit
    Note over SRE,MenuBar: Phase 1: Morning Health Glance & System Pulse
    SRE->>MenuBar: Glance at Menu Bar Pill (Displays: Green Dot · 0 Alerts · P99: 8.2ms)
    SRE->>App: Click MenuBar Pill to expand Native Control Tower HUD
    App->>Edge: GET /v1/control-tower/cells/status
    Edge->>Spanner: Probe Cell Latency (Toshkent, Samarqand, Fergana)
    Spanner-->>App: All 3 Regional Cells Healthy (Replication Lag < 12ms)

    %% Phase 2: Distributed Data Plane & Spanner Latency Tuning
    Note over SRE,Spanner: Phase 2: Spanner Multi-Cell Latency & Lock Contention Profiling
    SRE->>App: Open Spanner Multi-Cell EKG Dashboard
    App->>Edge: GET /v1/control-tower/metrics/spanner
    Edge->>Spanner: Query Lock Waits, Transaction Aborts, CPU Utilization
    Spanner-->>App: Read Latency: 2.1ms (P99), Write Latency: 11.4ms (P99), Abort Rate: 0.002%
    opt Bounded Staleness Tuning on High Read Load
        SRE->>App: Adjust Read Staleness Window to 5s for Read-Replicas
        App->>Edge: POST /v1/control-tower/spanner/staleness-tuning
        Edge->>Spanner: Update Session Pool Config
    end

    %% Phase 3: Kafka Partition Lag & Worker Auto-Scaling
    Note over SRE,Kafka: Phase 3: Kafka Partition Lag & Consumer Worker Scaling
    SRE->>App: Inspect Kafka Consumer Group Matrix (32 Topics, 128 Partitions)
    App->>Edge: GET /v1/control-tower/metrics/kafka-lag
    Edge->>Kafka: Query Consumer Offsets vs High Watermarks
    Kafka-->>App: Topic "orders.created" Lag: 420 messages across Partition 2
    SRE->>App: Trigger Worker Pool Scale-Up (Scale 4 -> 8 Workers)
    App->>Edge: POST /v1/control-tower/workers/scale (Pool: "wms-wave-planner", Count: 8)
    Edge->>Kafka: Worker Pool Scaled; Lag Drained in < 45 seconds

    %% Phase 4: Outbox Dead-Letter Queue Triage & Atomic Replay
    Note over SRE,Spanner: Phase 4: Outbox Dead-Letter Queue (DLQ) Diagnostic & Replay
    Kafka-->>App: Push Notification: Quarantined Event in OutboxDeadLetters!
    SRE->>App: Open Outbox DLQ Terminal
    App->>Edge: GET /v1/control-tower/dlq/events?status=QUARANTINED
    Edge->>Spanner: SELECT * FROM OutboxDeadLetters WHERE Status = 'QUARANTINED'
    Spanner-->>App: Event DLQ-88210 (Topic: "orders.fiscalized", Error: "PKCS#12 cert expired")
    SRE->>App: Export Encrypted Diagnostic Archive (.tar.gz)
    SRE->>Edge: Trigger Upstream Hotfix (Renew Soliq Fiscal Cert)
    SRE->>App: Execute Atomic In-Order Replay (Event ID: DLQ-88210)
    App->>Edge: POST /v1/control-tower/dlq/replay (EventID: "DLQ-88210")
    Edge->>Spanner: Begin ReadWriteTransaction: Re-insert Event & Update DLQ Status = REPLAYED
    Spanner-->>App: 200 OK: Message Successfully Reprocessed; DLQ Zero

    %% Phase 5: Tenant-Attributed FinOps Waterfall & Audit Sealing
    Note over SRE,FinOps: Phase 5: Multi-Tenant Cloud Spend Attribution & Cost Waterfall
    SRE->>App: Open FinOps Multi-Cloud Spend Attribution Center
    App->>Edge: GET /v1/control-tower/finops/attributed-costs
    Edge->>FinOps: Compute Attributed Cloud Costs per Tenant & Facility
    FinOps-->>App: Breakdown: Spanner Storage, Egress, Kafka Compute, Redis Memory
    App->>App: Compare Attributed Cloud Burn Rate vs Dynamically Configured Platform Take-Rate
    SRE->>App: Generate Monthly Infrastructure Capacity & SLA Report
    App->>Spanner: Record SRE Administrative Session Audit Signature
```

---

## 4. Exhaustive Feature Catalog

### 4.1. macOS Native MenuBarExtra Status Pill
- **Feature Name**: `MenuBarStatusView`
- **Purpose**: Provides persistent, zero-friction glanceability into ecosystem reliability directly from the macOS system menu bar without opening full browser dashboards.
- **UI Mechanics**: Native SwiftUI menu bar item displaying an interactive color-coded status circle:
  - Green Dot: Nominal health across all cells; zero active critical alerts.
  - Yellow Dot: Warning status (e.g., elevated Kafka lag > 500 msgs or P99 write latency > 25ms).
  - Red Dot: Critical breach (e.g., cell network partition, Spanner abort rate > 1%, or DLQ poison message quarantined).
- **Popover Contents**: Clicking the menu bar item opens a compact native popover detailing active cell latencies, total orders processed today, active outbox messages, and quick-launch buttons for full diagnostics.
- **Native Implementation**: [`apps/control-tower-macos/Sources/Views/MenuBarStatusView.swift`](file:///Users/shakhzod/Desktop/pegasusx/apps/control-tower-macos/Sources/Views/MenuBarStatusView.swift).

### 4.2. Multi-Cell Spanner EKG & Latency Profiler
- **Feature Name**: `SpannerEKGProfiler`
- **Purpose**: Continuously monitors Google Cloud Spanner distributed transaction performance, ensuring distributed ACID reservations meet the P99 < 15ms SLA.
- **UI Mechanics**: Real-time multi-line Swift Charts graph rendering P50, P90, P99 read and write latencies over 1-hour and 24-hour windows. Displays live bar meters for transaction aborts, deadlocks, and split-brain mutation warnings.
- **Data Attributes**:
  - `CellID`: Regional identifier (`cell-uz-toshkent`, `cell-uz-samarqand`, `cell-eu-frankfurt`).
  - `ReadLatencyMs`: Bounded read duration from nearest replica.
  - `WriteLatencyMs`: Paxos consensus write duration across replication quorums.
  - `AbortRate`: Fractional percentage of aborted transactions due to lock conflicts.
  - `LockWaitTimeMs`: Milliseconds spent waiting on row-level locks.

### 4.3. Kafka Consumer Partition Lag Matrix & Auto-Scaler
- **Feature Name**: `KafkaConsumerLagMatrix`
- **Purpose**: Prevents event-mesh backpressure by tracking consumer offsets across all core domain topics and automatically scaling background worker pools.
- **UI Mechanics**: Interactive heat grid displaying 32 topics (`orders.created`, `wms.picked`, `fleet.dvir`, `soliq.fiscalized`) and their 128 partitions. Color gradient shifts from deep green (lag < 10) to amber (lag 50..500) to pulsing red (lag > 500).
- **Backend API Seams**:
  - `GET /v1/control-tower/metrics/kafka-lag`
  - `POST /v1/control-tower/workers/scale`
- **Automated Scaling**: When partition lag exceeds configured thresholds for > 2 consecutive reporting intervals, the system invokes Kubernetes Horizontal Pod Autoscalers (HPA) to scale consumer workers up to 16 replicas.

### 4.4. Outbox Dead-Letter Queue (DLQ) Diagnostic & Replay Terminal
- **Feature Name**: `OutboxDLQTerminal`
- **Purpose**: Provides a safe, visual operational environment to inspect, troubleshoot, and replay poisoned or unprocessable outbox messages without risk of duplicate execution or out-of-order event corruption.
- **UI Mechanics**: Full-featured debugging terminal with formatted JSON payload decoders, stack trace viewers, upstream dependency health checks, and a two-step authenticated "Atomic In-Order Replay" button.
- **Backend Implementation**: [`apps/backend-go/cmd/replay-dlq`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/cmd/replay-dlq), [`apps/backend-go/outbox`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/outbox).
- **Replay Safety Protocol**:
  1. The SRE selects a quarantined event from `OutboxDeadLetters`.
  2. The tool verifies that upstream prerequisites are green (e.g., fiscal certificate renewed, external partner webhook reachable).
  3. Replay executes inside an atomic Cloud Spanner `ReadWriteTransaction`, re-inserting the message into the active outbox queue while marking the dead-letter record as `RESOLVED_AND_REPLAYED`.

### 4.5. Un-Sandboxed Diagnostic Bundle Exporter
- **Feature Name**: `DiagnosticBundleExporter`
- **Purpose**: Generates complete, self-contained diagnostic archives during critical incidents for post-mortem analysis and vendor support (Google Cloud, Confluent).
- **UI Mechanics**: One-click export button generating an AES-256 encrypted `.tar.gz` bundle.
- **Archive Contents**:
  - Redacted application error logs across all services for the preceding 2 hours.
  - Schema drift verification reports from [`schemadrift`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/schemadrift).
  - Current Spanner transaction lock tables and partition splits.
  - Active Kafka consumer group offset snapshots.
  - Recent outbox DLQ error traces and environmental configurations.

### 4.6. Multi-Tenant Cloud Spend Attribution Waterfall (FinOps)
- **Feature Name**: `FinOpsAttributedCostCenter`
- **Purpose**: Provides mathematical attribution of multi-cloud infrastructure spend down to individual suppliers, warehouse facilities, and retailer networks, ensuring cloud margins remain healthy.
- **UI Mechanics**: Interactive multi-tier Sankey waterfall diagram rendered with Swift Charts / HTML5 canvas showing the flow of cloud costs:
  - Ingestion: GCP Spanner Processing Units, GKE Autopilot vCPU/Memory, Redis Sentinel, Network Egress.
  - Allocation: Attributed proportionally to tenants based on transaction volume, stored SKU counts, and API call frequency.
  - Margin Analysis: Compares tenant cloud burn rate against the dynamically configured platform take-rate revenue.
- **Dynamic Policy Contract**: Replaces hardcoded percentages and figures with dynamic formulas calculated per tenant Market Pack contract and actual GCP billing exports.

### 4.7. Schema Drift Verifier & Zero-Drift Gate
- **Feature Name**: `SchemaDriftVerifier`
- **Purpose**: Guarantees 100% database schema parity across staging, sandbox, and multi-cell production environments, preventing runtime query failures during zero-downtime rolling deployments.
- **UI Mechanics**: Schema comparison diff viewer highlighting column additions, type modifications, index changes, and foreign key constraints between local DDL files (`apps/backend-go/schema/migrations/*.ddl`) and live Spanner information schema.
- **Backend Seam**: [`apps/backend-go/schemadrift`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/schemadrift)
- **CI/CD Enforcement**: The schema drift verifier runs as a mandatory pre-deployment gate; any unapplied migration or out-of-band schema mutation immediately halts deployment pipelines.

---

## 5. Security Markings, Error Matrix & Operational Invariants

### 5.1. SRE Error Matrix
| Error Code | HTTP Status | Root Cause | Remediation Procedure |
| :--- | :---: | :--- | :--- |
| `ERR_SPANNER_TRANSACTION_ABORTED` | `409 Conflict` | High concurrent write contention on shared lock rows. | Review lock order in transaction; enable random exponential backoff retry. |
| `ERR_OUTBOX_LAG_EXCEEDED` | `503 Service Unavailable` | Outbox publisher falling behind Spanner commit rate. | Scale outbox relay worker pool; verify Kafka broker write latencies. |
| `ERR_POISON_MESSAGE_QUARANTINED` | `422 Unprocessable` | Event handler failed repeatedly on malformed or invalid payload. | Inspect message in DLQ terminal; fix schema/service bug; trigger atomic replay. |
| `ERR_SCHEMA_DRIFT_DETECTED` | `412 Precondition Failed` | Production Spanner table definitions diverge from versioned DDL. | Halt deployments; apply missing DDL migration scripts via apply-migration CLI. |
| `ERR_FINOPS_ATTRIBUTION_ANOMALY` | `500 Internal Error` | Ingested cloud billing records do not reconcile with tenant volume. | Re-ingest raw GCP billing export; check tenant tagging on compute resources. |

### 5.2. Role Invariants Checklist
- [x] **Zero Event Loss Guarantee**: Every state transition written to Cloud Spanner must produce an immutable outbox record before transaction commit.
- [x] **Zero Direct DB Mutations**: Direct un-audited SQL write bypasses are strictly prohibited in production; all mutations must flow through audited Go service code.
- [x] **Dynamic Fee Architecture**: All FinOps calculations utilize dynamic Market Pack configurations and actual billing metrics rather than hardcoded literals.
- [x] **Hardware Enclave Step-Up**: Sensitive infrastructure operations require biometric or hardware FIDO2 authentication.
