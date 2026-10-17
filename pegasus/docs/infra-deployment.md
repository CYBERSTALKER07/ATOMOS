# Pegasus Infrastructure, Deployment & Operational Governance

> **Ecosystem**: Pegasus Cloud Infrastructure, Kubernetes & Verification Tooling  
> **Source Path**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/docs/INFRASTRUCTURE.md`  
> **Referenced Codebases**:
> - Terraform IaC: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/infra/terraform`
> - Kubernetes Manifests: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/infra/k8s`
> - Docker Compose Fleet: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/docker-compose.yml`
> - Governance & Guard Scripts: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/scripts`

---

## 1. Google Cloud Terraform Topology

The cloud infrastructure for Pegasus is provisioned using declarative Terraform configurations located in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/infra/terraform`.

```
                    PEGASUS GCP CLUSTER TOPOLOGY
  ┌────────────────────────────────────────────────────────────────────────┐
  │                         Private VPC Network                            │
  │                           (pegasus-vpc)                                │
  ├────────────────────────────────────┬───────────────────────────────────┤
  │       Regional Layer               │      Multi-Region Scale-Out       │
  │         (main.tf)                  │         (multiregion.tf)          │
  ├────────────────────────────────────┼───────────────────────────────────┤
  │ • Cloud Memorystore (Redis 7.0 HA) │ • Global Spanner (nam-eur-asia3)  │
  │ • Regional Spanner (asia-south1)   │   3 Continents, 9 Nodes           │
  │ • Private GKE Cluster              │   30,000 Read QPS / 3,000 Write   │
  │ • Cloud Run Gateway                │ • Regional GKE Clusters           │
  │ • Artifact Registry (GAR)          │   asia-south1, europe-west1,      │
  │                                    │   us-central1                     │
  └────────────────────────────────────┴───────────────────────────────────┘
```

### 1.1 Single-Region Baseline (`main.tf`)
- **Source**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/infra/terraform/main.tf`
- **What It Does**: Provisions core resources within the primary Central Asian datacenter (`asia-south1`):
  - **Private VPC Network** (`main.tf:38`): Dedicated `pegasus-vpc` with automated subnet provisioning and private Google access.
  - **Google Cloud Memorystore Redis** (`main.tf:45`): `pegasus-memory-layer`, running Redis 7.0 in `STANDARD_HA` (active-passive multi-zone failover) to guarantee cache uptime during host maintenance.
  - **Cloud Spanner Regional Instance** (`main.tf:55`): `pegasus-ledger-instance` running on `regional-asia-south1`, serving as the primary ACID ledger.
  - **GKE Private Cluster** (`main.tf:70`): Multi-zone Kubernetes cluster running `apps/backend-go` and `apps/ai-worker`.
  - **Artifact Registry (GAR)**: Centralized Docker image repository for backend and worker containers.
- **Why It Is There**: Provides a cost-efficient, production-grade staging and single-region deployment configuration.

### 1.2 Multi-Region Scale-Out (`multiregion.tf`)
- **Source**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/infra/terraform/multiregion.tf`
- **What It Does**: Provides an additive, non-destructive scale-out triggered when `enable_multiregion = true` (`multiregion.tf:14`):
  - **Spanner `nam-eur-asia3` Topology** (`multiregion.tf:25`): Upgrades Cloud Spanner to a 3-continent multi-region instance (`void-db-global`). Provisions 3 nodes per continent (9 nodes total), delivering ~30,000 read QPS and ~3,000 write QPS with automatic cross-continental synchronous Paxos replication.
  - **Regional GKE Clusters** (`multiregion.tf:80-160`): Provisions compute clusters in `asia-south1` (Tashkent/Mumbai), `europe-west1` (Frankfurt), and `us-central1` (Iowa).
  - **Global Kafka Federation**: Interconnects regional event streams.
- **Why It Is There**: Supports international wholesale commerce across Eurasia and the Americas, eliminating single-datacenter failure risks and enabling sub-10ms localized read queries via the Maglev Spanner router.

---

## 2. Kubernetes Production Fleet & Autoscaling (`infra/k8s/`)

### 2.1 Namespace & Topology
All production components run within the `void-system` / `pegasus` namespace (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/infra/k8s/namespace.yaml`).

### 2.2 Backend Deployment (`infra/k8s/backend/`)
- **Deployment** (`infra/k8s/backend/deployment.yaml`): Runs the Go 1.25 Chi backend with 3 baseline replicas. Configured with liveness (`/v1/health`) and readiness probes.
- **PodDisruptionBudget (PDB)** (`infra/k8s/backend/pdb.yaml`): Enforces `minAvailable: 2`, ensuring that during node draining or rolling updates, at least 2 backend pods remain online to handle live WebSocket connections and driver traffic.
- **Horizontal Pod Autoscaler (HPA)** (`infra/k8s/backend/hpa.yaml`): Scales backend pods from 3 to 50 based on 70% average CPU utilization and 80% memory targets.

### 2.3 AI Worker & Rust Optimizer Sidecar (`infra/k8s/ai-worker/`)
- **Co-Located Pod Architecture** (`infra/k8s/ai-worker/deployment.yaml`): The `ai-worker` container is co-located with the `optimizer-core-rust` sidecar container in the same Kubernetes pod, communicating over loopback IPC (`127.0.0.1:50055`). This eliminates network latency during massive distance matrix transfers.
- **Event-Driven Autoscaling via KEDA** (`infra/k8s/ai-worker/keda-scaledobject.yaml`):
  Instead of relying solely on CPU (which lags behind sudden bursts of orders), scaling is governed by Kafka consumer group lag:
  - `pegasus-logistics-events`: Lag threshold of 50 messages triggers replica scale-out.
  - `pegasus-freeze-locks`: Lag threshold of 10 messages (stricter threshold due to the time-sensitive ≤250ms freeze-lock doctrine).
  - `pegasus-demand-forecast`: Lag threshold of 50 messages.
  - Scales dynamically from 1 to 20 replicas (`maxReplicaCount: 20`).

### 2.4 Prometheus Observability & Alert Rules (`infra/k8s/monitoring/`)
Defined in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/infra/k8s/monitoring/prometheusrule.yaml`:
- **`KafkaConsumerLagHigh`** (`prometheusrule.yaml:15`): Fires if consumer lag across any topic partition exceeds 10 seconds for >1 minute (`severity: warning`).
- **`OutboxRelayLagHigh`** (`prometheusrule.yaml:32`): Fires if `void_outbox_relay_lag_seconds > 60` for >1 minute (`severity: critical`). Indicates stuck outbox rows in Spanner.
- **`RedisCBOpen`** (`prometheusrule.yaml:49`): Alerts when the Redis circuit breaker enters `OPEN` state for >5 minutes, indicating backend fallback to degraded mode.

---

## 3. Local Simulation Fleet (`docker-compose.yml`)

### 3.1 What It Is
The local simulation environment (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/docker-compose.yml`) is a completely self-contained emulator fleet that enables full-stack local development and E2E testing without external cloud dependencies or credentials.

### 3.2 Service Catalog & Port Mappings

```
                           LOCAL DOCKER COMPOSE FLEET
 ┌──────────────────────┬─────────────┬──────────────────────────────────────────┐
 │ Service              │ Local Ports │ Functionality                            │
 ├──────────────────────┼─────────────┼──────────────────────────────────────────┤
 │ kafka                │ 9092, 9093  │ Apache Kafka 7.7.0 in KRaft mode         │
 │ kafka-ui             │ 8081        │ Web UI for topic inspection              │
 │ kafka-init           │ N/A (CLI)   │ Auto-provisions 8 topics with partitions │
 │ redis                │ 6379        │ Redis Alpine (Memorystore emulator)      │
 │ spanner              │ 9010, 9020  │ Google Cloud Spanner Emulator            │
 │ firebase-auth        │ 9099, 4000  │ Firebase Auth emulator & debug UI        │
 │ globalpay-mock       │ 8085        │ WireMock Global Pay gateway simulator    │
 ├──────────────────────┴─────────────┴──────────────────────────────────────────┤
 │ Optimizer Simulation Profile (--profile optimizer-sim)                        │
 ├──────────────────────┬─────────────┬──────────────────────────────────────────┤
 │ optimizer-core-rust  │ 50055       │ Rust Tonic/Prost CVRPTW gRPC sidecar     │
 │ optimizer-adapter    │ 8082        │ Go Kafka-to-gRPC adapter worker          │
 └──────────────────────┴─────────────┴──────────────────────────────────────────┘
```

1. **Apache Kafka (KRaft Mode)** (`docker-compose.yml:5`):
   - Image: `confluentinc/cp-kafka:7.7.0`.
   - Runs in native KRaft quorum mode without Zookeeper (`KAFKA_PROCESS_ROLES=broker,controller`).
   - `kafka-init` automatically provisions the 8 core event topics on boot (`pegasus-logistics-events` and `pegasus-demand-forecast` with 128 partitions; others with 16 partitions or 1 for DLQ).
2. **Google Cloud Spanner Emulator** (`docker-compose.yml:116`):
   - Image: `gcr.io/cloud-spanner-emulator/emulator:latest`.
   - Exposes gRPC API on port 9010 and REST API on port 9020.
3. **Redis Memorystore Emulator** (`docker-compose.yml:99`):
   - Image: `redis:alpine` listening on port 6379 with persistence disabled for clean test runs.
4. **Firebase Auth Emulator** (`docker-compose.yml:132`):
   - Node 22 container running `firebase emulators:start --only auth` on port 9099 with UI on port 4000.
5. **Global Pay WireMock Simulator** (`docker-compose.yml:165`):
   - Image: `wiremock/wiremock:latest` listening on port 8085, serving pre-recorded JSON responses (`mocks/globalpay`) for card tokenization and 3D-Secure webhooks.
6. **Optimizer Simulation Profile** (`docker-compose.yml:183-239`):
   - Started via `make optimizer-sim-up`.
   - Boots `optimizer-core-rust` on port 50055 and `optimizer-adapter-worker` with Prometheus metrics on port 8082.

---

## 4. Continuous Governance, Parity & Quality Guards (`scripts/`)

### 4.1 What It Is
Pegasus enforces automated architecture, security, and contract consistency through a collection of custom verification engines located in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/scripts`.

### 4.2 Guardrails Directory & Execution Mechanics

1. **Sprint-1 Execution Gate (`scripts/sprint1_execution_gate.py`)**:
   - Master Python test runner invoked via `make sprint1-gate`.
   - Executes changed-file analysis, contract drift checks, boundary tests, and emits a structured audit report (`pegasus/.execution/sprint1/gate-report.json`).
2. **Contract Drift Guard (`scripts/contract_drift_guard.py`)**:
   - Inspects Go Kafka event definitions in `apps/backend-go/kafka/events.go` and cross-references them against `contracts/events.schema.json` and TypeScript interfaces in `packages/types/src/ws-events.ts`.
   - Fails immediately if an engineer adds a field to a Go struct without updating the shared contract schema.
3. **Gen-Contracts Parity Gate (`scripts/parity/gen_contracts_gate.sh`)**:
   - Shell guard that runs `cmd/gen-contracts` against `kafka/events.go` to generate a temporary JSON schema and executes `diff -q` against `contracts/events.schema.json`.
4. **Architectural Boundary Guard (`scripts/architecture_boundary_guard.py`)**:
   - Analyzes Go import graphs, preventing illegal couplings (e.g. ensuring `packages/config` or domain models never import web route packages).
5. **Design Token Enforcement Guard (`scripts/design_token_enforcement_guard.py`)**:
   - Scans CSS, JSX, and TSX files across the 4 Next.js web portals (`admin-portal`, `factory-portal`, `warehouse-portal`, `retailer-app-desktop`).
   - Flags arbitrary unapproved hex colors or raw CSS properties that bypass `packages/ui-kit` Material 3 design tokens.
6. **Production Safety & Zero-Theatre Guard (`scripts/production_safety_guard.py`)**:
   - Enforces the Zero Theatre Mandate: scans source code for mock API URLs, fake development endpoints in production configurations, or commented-out stub logic in critical business paths.
7. **Security Guard (`scripts/security_guard.py`)**:
   - Scans for hardcoded secrets, unparameterized SQL strings, insecure cryptographic ciphers, and unvalidated URL parameters.
8. **Version Scanner (`scripts/versionscan.py`)**:
   - 34KB engine enforcing exact semantic version locks across Node `package.json`, Go `go.mod`, Rust `Cargo.toml`, and Python `pyproject.toml` manifests.

### 4.3 Unified Verification Command ("One-Eye")
Defined in `file:///Users/shakhzod/Desktop/V.O.I.D/pegasus/package.json:19`:
```bash
npm run guard:one-eye
```
Chains together all six critical MCP verification guards in a single deterministic pass: contract guard, architecture boundary guard, design system guard, production safety guard, visual test intelligence guard, and security guard.
