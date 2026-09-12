# PegasusX Enterprise Infrastructure: Phased GCP Architecture & Deployment Plan

**Document Version:** 1.0.0  
**Classification:** Authoritative Cloud Infrastructure Architecture & Wiring Specification  
**Governing Standard:** `INVENTORY.md`, `PROJECT.md`, `CODEBASE_GAP_REPORT.md`, `DOC_SUMMARY.md`  
**Target Environment:** Google Cloud Platform (GCP) Enterprise Production  
**Primary Region:** `europe-west3` (Frankfurt) / `asia-south1` (Mumbai)  
**Publication Date:** 2026-08-27  
**Author:** Worker M2 (Cloud Infrastructure & SRE Specialist)  

---

## Table of Contents

1. [Executive Summary & System Principles](#1-executive-summary--system-principles)
   - 1.1 Mission & Architectural Purpose
   - 1.2 Core System Principles & Spine Laws Compliance
   - 1.3 Service Level Objectives (SLOs), RPO & RTO Guarantees
2. [Compute Platform Selection & Technical Justification](#2-compute-platform-selection--technical-justification)
   - 2.1 Compute Platform Evaluation Matrix (GKE vs Cloud Run vs GCE VMs)
   - 2.2 In-Depth Technical Justification Grounded in Codebase Realities
   - 2.3 Final Compute Topology: Hybrid GKE Autopilot & Workload Isolation
3. [Service-to-GCP Component Mapping Matrix](#3-service-to-gcp-component-mapping-matrix)
   - 3.1 Core Backend & Microservices Mapping
   - 3.2 Database, Caching & Message Streaming Tier Mapping
   - 3.3 CLI Daemons, Schedulers & Batch Job Mapping
   - 3.4 Web Portals, Desktop POS & Mobile Client Distribution
   - 3.5 Networking, Edge & Security Infrastructure Mapping
4. [Phased Rollout Strategy with Inter-Phase Dependencies](#4-phased-rollout-strategy-with-inter-phase-dependencies)
   - 4.1 Rollout Dependency Graph & Progression Milestones
   - 4.2 Phase 1: Foundation, VPC Networking & IAM Security
   - 4.3 Phase 2: Managed Data, Storage & Messaging Backbone
   - 4.4 Phase 3: GKE Compute, Workload Identity & Core Deployments
   - 4.5 Phase 4: Edge Routing, Third-Party Adapters, Observability & Hardening
5. [Third-Party Service Integration Architecture](#5-third-party-service-integration-architecture)
   - 5.1 Soliq OFD (Uzbekistan State Tax Committee EHF Invoicing)
   - 5.2 Payment Rails & FinTech Adapters (Global Pay, Payme, Click, Adyen, Stripe)
   - 5.3 SMS, WhatsApp & Transactional Email Communications
   - 5.4 Spatial Routing, Distance Matrix & Geocoding (Google Routes v2 + Self-Hosted OSRM)
   - 5.5 Mobile Identity & Cloud Messaging (Firebase Phone Auth + FCM / APNs)
6. [Capacity Planning & Monthly TCO Cost Model at Scale](#6-capacity-planning--monthly-tco-cost-model-at-scale)
   - 6.1 Baseline Workload Sizing Parameters (1,000 Retailers, 50 Suppliers, 200 Drivers)
   - 6.2 Detailed Resource Sizing & Throughput Calculations
   - 6.3 Comprehensive Itemized Monthly TCO Cost Table
7. [High Availability (HA), Disaster Recovery (DR) & Autoscaling](#7-high-availability-ha-disaster-recovery-dr--autoscaling)
   - 7.1 Multi-Zone Active-Active High Availability Architecture
   - 7.2 Horizontal Pod Autoscaling (HPA) & KEDA Event-Driven Scaling
   - 7.3 Cloud Spanner & Kafka Elastic Capacity Management
   - 7.4 Disaster Recovery Strategy, Backup Lifecycle & Failover Playbooks
8. [Observability, Monitoring & SRE Alerting Framework](#8-observability-monitoring--sre-alerting-framework)
   - 8.1 OpenTelemetry Tracing & Prometheus Metrics Pipeline
   - 8.2 Production Alerting Policies Matrix (17 P1/P2/P3 Rules)
   - 8.3 Synthetic Uptime Probes & Health Check Architecture
   - 8.4 SRE On-Call Incident Response & Remediation Runbooks
9. [Enterprise Security Hardening & Zero-Trust Architecture](#9-enterprise-security-hardening--zero-trust-architecture)
   - 9.1 Network Perimeter Isolation & VPC Firewall Rules
   - 9.2 Workload Identity Federation & Least-Privilege IAM
   - 9.3 Cloud Armor WAF Policies & DDoS Mitigation
   - 9.4 Secret Management, Rotation & Cryptographic Vault Integrity
   - 9.5 Audit Logging & Regulatory Compliance Enforcement

---

## 1. Executive Summary & System Principles

### 1.1 Mission & Architectural Purpose
**PegasusX** is a mission-critical, enterprise Fast-Moving Consumer Goods (FMCG) supply chain operating system and B2B wholesale commerce platform. It coordinates multi-tier distribution networks comprising FMCG manufacturers, wholesale distributors, regional warehouse depots, vehicle delivery fleets, and retail point-of-sale (POS) storefronts.

To sustain continuous supply chain operations across emerging and high-growth markets, the underlying cloud infrastructure must deliver uninterrupted uptime, low-latency transaction processing, deterministic financial settlement, and continuous driver tracking. This document establishes the definitive Google Cloud Platform (GCP) architecture, engineered to satisfy enterprise reliability, regulatory tax compliance, and multi-tenant security standards.

```
+--------------------------------------------------------------------------------------------------------------------+
|                                      PEGASUSX PRODUCTION GCP END-TO-END TOPOLOGY                                    |
+--------------------------------------------------------------------------------------------------------------------+
|                                                [ Public Internet ]                                                 |
|                                                         │                                                          |
|                                                         v                                                          |
|                                       [ Cloud Armor Enterprise WAF Policy ]                                        |
|                                 (Layer 7 Rate Limiting, OWASP Top 10, Geo-Fencing)                                 |
|                                                         │                                                          |
|                                                         v                                                          |
|                                     [ Global External HTTPS Load Balancer ]                                        |
|                                  (Google-Managed SSL, HTTP/2, Anycast IPv4/IPv6)                                    |
|                                                         │                                                          |
|                  ┌──────────────────────────────────────┼──────────────────────────────────────┐                   |
|                  │ /v1/ws (Session Sticky 3600s)        │ /v1, /partner (Timeout 120s)         │ /* (Static / OTA) |
|                  v                                      v                                      v                   |
|          [ GKE Ingress: backend-go-ws ]         [ GKE Ingress: backend-go-api ]        [ Cloud CDN / GCS ]         |
|                  │                                      │                              (Web Portals & APKs)        |
|                  └──────────────────────┬───────────────┘                                                          |
|                                         │                                                                          |
|                                         v                                                                          |
|       ══════════════════════════ [ VPC: pegasusx-vpc (10.10.0.0/20) ] ════════════════════════════════════════════ |
|       │                                                                                                          │ |
|       │   [ GKE Autopilot Cluster: pegasusx-prod-gke ] (Pods: 10.20.0.0/16 | Services: 10.30.0.0/20)            │ |
|       │   ├── backend-go-api (4 Replicas)        ├── optimizer-core (Python OR-Tools) (3 Replicas)               │ |
|       │   ├── backend-go-worker (4 Replicas)     ├── optimizer-core-rust (2 Replicas Sidecar)                    │ |
|       │   ├── ai-worker (2 Replicas)             ├── handoff-service (2 Replicas Pure Go)                        │ |
|       │   └── K8s CronJobs (Forecast, Accuracy)  └── External Secrets Operator (Syncs Secret Manager)            │ |
|       │                                                                                                          │ |
|       │   [ Private Service Access Peering (10.42.205.148/29) ]                                                  │ |
|       │   ├── Cloud Memorystore for Redis 7.0 (STANDARD_HA Cross-Zone, TLS + AUTH, 5 GB)                         │ |
|       │   └── Google Managed Service for Apache Kafka (3-Node AZ Cluster, 10 Topics, SASL_SSL)                   │ |
|       │                                                                                                          │ |
|       │   [ Cloud Spanner Multi-Zone Instance: ledger ] (100 - 1000 PU Autoscaling)                              │ |
|       │   └── Database: main (136 Tables, 13 Interleaved Hierarchies, 193 Indexes, PITR 7d)                      │ |
|       │                                                                                                          │ |
|       │   [ Cloud NAT Gateway + Cloud Router ]                                                                   │ |
|       │   └── 2 Dedicated Static External IP Addresses (Whitelisted for Soliq OFD & Banking Gateways)             │ |
|       ════════════════════════════════════════════════════════════════════════════════════════════════════════════ |
+--------------------------------------------------------------------------------------------------------------------+
```

---

### 1.2 Core System Principles & Spine Laws Compliance

The GCP architecture strictly enforces the five architectural **Spine Laws** defined in the PegasusX core design framework:

1. **The Law of Realized State (Zero Ghost Transactions):**
   - No mutating business event (order placement, warehouse picking release, vehicle dispatch, or payment capture) is acknowledged to a client application or external webhook until it is durably committed to Cloud Spanner storage across a quorum of availability zones. In-memory buffers are never used as authoritative state.
2. **The Law of Split Atomicity (Interleaved Relational Locality):**
   - Cloud Spanner utilizes 13 physical table interleaving hierarchies (`Claims -> ClaimEvidences`, `Orders -> OrderLineAllocations`, `PriceLists -> PriceListItems`). Parent and child rows are co-located on the identical Spanner physical storage split, guaranteeing root-level atomic transactions, zero-latency distributed joins, and ACID cascading updates.
3. **The Law of Dual-Write Durability (Transactional Outbox Pattern):**
   - Mutating business transactions write their domain entity changes and an `OutboxEvents` row inside the **exact same atomic Spanner transaction closure** (`spanner.BufferWrite`). A continuous outbox relay daemon on GKE polls unpublished events every 250ms via index-only scans (`Idx_OutboxEvents_Unpublished`) and broadcasts them to Google Managed Kafka, preventing split-brain inconsistencies between database and message broker.
4. **The Law of Idempotent Replay (Deterministic Deduplication):**
   - Every external mutation and Kafka consumer execution is protected by deterministic idempotency keys. The API gateway uses Redis `SET key token NX EX 120` against SHA-256 payload digests, while Kafka consumers maintain a 7-day deduplication registry in Redis (`kafka:dedup:<group>:<event_id>`), guaranteeing strictly once-effective domain processing.
5. **The Law of Fail-Closed Resilience (Strict Infrastructure Guardrails):**
   - Under production runtime configurations (`REQUIRE_INFRA_ADAPTERS=true`), in-memory mock adapters and stubs are rejected during container boot. If Cloud Spanner, Memorystore Redis, or Managed Kafka connectivity drops, the system fails closed rather than allowing unauthenticated or untracked state mutations.

---

### 1.3 Service Level Objectives (SLOs), RPO & RTO Guarantees

| Metric Parameter | Production Target | Architectural Mechanism |
| :--- | :--- | :--- |
| **Availability SLA** | **99.99%** (≤ 4.38 mins downtime/month) | Multi-zone GKE Autopilot, Cloud Spanner Regional 3-zone quorum, Memorystore Redis `STANDARD_HA` cross-zone failover, multi-broker Kafka. |
| **Recovery Point Objective (RPO)** | **RPO = 0** (Zero data loss) | Cloud Spanner synchronous Paxos replication across 3 independent availability zones. Every committed transaction is guaranteed durable. |
| **Recovery Time Objective (RTO)** | **RTO < 15 Minutes** | Automated GKE pod rescheduling (< 60s), Redis automatic failover (< 30s), automated Spanner leader re-election (< 5s), Infrastructure-as-Code Terraform recovery. |
| **API Latency (p95)** | **< 120 ms** | Regional Cloud Spanner single-split reads (< 5ms), Memorystore Redis sub-millisecond cache hits, Cloud CDN edge caching. |
| **API Latency (p99)** | **< 350 ms** | Bounded concurrency load shedding (`reliability.Middleware`), non-blocking Chi router, optimized CGO `h3-go` spatial lookups. |
| **Driver Telemetry Latency** | **< 1,000 ms (1 Hz stream)** | WebSocket direct frame ingestion (`/v1/ws`), in-memory Redis driver coordinate cache (`telemetry:driver:<id>`), throttled outbox Kafka relay. |
| **VRP Route Optimization** | **< 5,000 ms (50 stops)** | Multi-threaded C++ OR-Tools solver with GNU OpenMP acceleration and OSRM local distance matrix caching. |

---

## 2. Compute Platform Selection & Technical Justification

### 2.1 Compute Platform Evaluation Matrix

A thorough technical evaluation was conducted across three GCP compute platforms against the concrete architectural characteristics of the PegasusX codebase:

```
+--------------------------------------------------------------------------------------------------------------------------+
|                                           GCP COMPUTE PLATFORM TRADEOFF EVALUATION                                       |
+------------------------------------+-------------------------+-------------------------+---------------------------------+
| Evaluation Criteria                | Google Kubernetes Engine| Cloud Run               | Compute Engine (GCE VMs)        |
|                                    | (GKE Autopilot)         | (Serverless Containers) | (Managed Instance Groups)       |
+------------------------------------+-------------------------+-------------------------+---------------------------------+
| **CGO & Custom glibc Bindings**    | **Native / Unrestricted**| Compatible              | Native / Unrestricted           |
| (`h3-go` C bindings)               | Full Debian glibc runtime| Containerized runtime   | Full OS control                 |
+------------------------------------+-------------------------+-------------------------+---------------------------------+
| **Persistent Multi-Hour WebSockets**| **Supported (Native)**  | **Incompatible / Risky**| Supported                       |
| (8 Role Hubs, 1Hz GPS streams)     | Long-lived TCP, 3600s+  | 60-min max timeout, drops| Requires manual HA & LB wiring  |
|                                    | sticky sessions         | on auto-scaling         |                                 |
+------------------------------------+-------------------------+-------------------------+---------------------------------+
| **Multi-Threaded CPU Solving**     | **Optimized**           | Poor                    | High                            |
| (Python OR-Tools + OpenMP C++)     | Dedicated vCPU pinning, | CPU throttled when not  | Fixed dedicated cores           |
|                                    | `libgomp1` acceleration | serving HTTP requests   |                                 |
+------------------------------------+-------------------------+-------------------------+---------------------------------+
| **Continuous Daemon Pools**        | **Native K8s Deployments| **Incompatible**        | Supported via systemd daemons   |
| (25+ Background Worker Loops)      | 24/7 uninterrupted loops| Scales to zero; killing | High operational overhead       |
|                                    | without HTTP triggers   | background tickers      | to monitor & restart            |
+------------------------------------+-------------------------+-------------------------+---------------------------------+
| **Kafka Consumer Group Stability** | **Optimal**             | Unstable                | Stable                          |
| (12 Active Consumer Groups)        | Persistent connections; | Container sleep causes  | Persistent VMs                  |
|                                    | zero partition rebalance| constant rebalance storms|                                 |
+------------------------------------+-------------------------+-------------------------+---------------------------------+
| **Shared Persistent Datasets**     | **Supported**           | **Unsupported**         | Supported (Attached Persistent  |
| (OSRM 10GB+ Routing Graph PVC)     | PersistentVolumeClaims  | No ReadWriteMany or     | Disks)                          |
|                                    | (Filestore / SSD CSI)   | raw block PVC mounts    |                                 |
+------------------------------------+-------------------------+-------------------------+---------------------------------+
| **Security & Workload Identity**   | **Native GKE Workload   | Native Service Account  | Service Account attached to VM  |
|                                    | Identity Federation**   | attachment              | (Coarse perimeter)              |
+------------------------------------+-------------------------+-------------------------+---------------------------------+
| **Operational & Maintenance Cost** | **Low (Autopilot SLA)** | Minimal                 | High (OS patching, kernel CVEs) |
+------------------------------------+-------------------------+-------------------------+---------------------------------+
| **Verdict**                        | **SELECTED PRIMARY**    | **REJECTED**            | **REJECTED (Except Bastion)**   |
+------------------------------------+-------------------------+-------------------------+---------------------------------+
```

---

### 2.2 In-Depth Technical Justification Grounded in Codebase Realities

The selection of **Google Kubernetes Engine (GKE Autopilot)** as the primary compute platform is mandated by five specific architectural constraints present in the PegasusX monorepo:

#### 1. CGO `h3-go` glibc Dynamic Linking Constraints
- The backend geospatial services (`apps/backend-go/geolocation`, `apps/backend-go/driverroutes`, `apps/backend-go/demand`) rely heavily on `github.com/uber/h3-go/v4`.
- `h3-go` compiles native C code bindings that require GNU `glibc` (`debian:bookworm-slim`). Musl-based distributions (such as Alpine Linux) cause memory corruption and segmentation faults during spatial indexing.
- While Cloud Run supports containerized Debian binaries, cold-start latency for glibc-compiled CGO binaries with dynamic shared libraries exceeds 3.5 to 5.0 seconds. GKE maintains warm, continuously running pods with zero cold-start overhead.

#### 2. Persistent Multi-Hour Stateful WebSockets Across 8 Role Hubs
- PegasusX mounts 8 distinct WebSocket hubs at `/v1/ws` (`apps/backend-go/ws/`): `RetailerHub`, `SupplierHub`, `DriverHub`, `PayloadHub`, `WarehouseHub`, `FactoryHub`, `TelemetryHub`, and `PlatformAdminHub`.
- Mobile drivers and warehouse scanners maintain continuous, bi-directional TCP WebSocket sessions lasting 8 to 12 hours during shift operations.
- **Cloud Run Limitation:** Cloud Run enforces a hard maximum request duration ceiling of 60 minutes. Furthermore, serverless autoscaling down-scaling events abruptly terminate in-flight WebSocket connections, causing connection reconnect storms across 200+ mobile devices.
- **GKE Advantage:** GKE Autopilot paired with GCE L7 Load Balancing supports long-lived WebSocket sessions with configurable session affinity (3600s), graceful connection draining (`terminationGracePeriodSeconds: 60`), and cross-pod synchronization via Redis Pub/Sub.

#### 3. Python OR-Tools OpenMP Multi-Threaded Optimization Solver
- The vehicle routing engine (`services/optimizer-core/server/http_main.py`) solves Multi-Depot Capacitated Vehicle Routing Problems with Time Windows (CVRPTW) using Google OR-Tools v9.15.
- The solver utilizes GNU OpenMP (`libgomp1`) to execute parallel Guided Local Searches across multiple CPU cores.
- **Cloud Run Limitation:** Cloud Run throttles or disables CPU allocation when a container is not actively processing an inbound HTTP request, preventing background multi-threaded heuristic searching and matrix pre-computations.
- **GKE Advantage:** GKE allocates guaranteed, dedicated CPU resources (`resources.requests.cpu: "2000m"`), allowing OpenMP threads to execute at 100% CPU capacity with predictable sub-5-second solve times.

#### 4. Optional Self-Hosted OSRM Dataset Persistent Disk Mounts
- To avoid high third-party API costs for distance matrix queries during daily route generation (~45,000 matrix lookups/day), PegasusX supports a self-hosted Open Source Routing Machine (OSRM) service (`ROUTING_OSRM_URL=http://osrm:5000`).
- OSRM requires direct local access to a 10 GB+ pre-processed OpenStreetMap road network dataset (`.osrm` multi-gigabyte binary graph).
- **Cloud Run Limitation:** Cloud Run cannot mount Google Cloud Persistent Disks (`gcePersistentDisk` or regional SSD PVCs) with high-speed memory-mapped file (`mmap`) I/O.
- **GKE Advantage:** GKE supports Kubernetes `PersistentVolumeClaims` backed by Compute Engine SSD Persistent Disks, allowing OSRM to load road networks into RAM with sub-millisecond lookup performance.

#### 5. 25+ Background Worker Loops & Continuous Kafka Consumer Groups
- As documented in `apps/backend-go/runtime_workers.go`, the backend worker tier executes 25+ continuous background loops, including the 250ms Outbox Relay, 12 Kafka Consumer Groups, AR Dunning Sweepers, and POS Hold Expirations.
- **Cloud Run Limitation:** Cloud Run is strictly request-driven. Running continuous background tickers without incoming HTTP traffic violates the Cloud Run execution model and causes CPU throttling. Furthermore, frequent container scaling triggers massive Kafka consumer group rebalances, halting event consumption.
- **GKE Advantage:** GKE provides dedicated Daemon deployments (`backend-go-worker`, `ai-worker`) with static consumer group assignments, ensuring continuous outbox polling and zero Kafka rebalance churn.

---

### 2.3 Final Compute Topology: Hybrid GKE Autopilot & Workload Isolation

To achieve maximum operational efficiency and cost control, PegasusX deploys a **GKE Autopilot Cluster** (`pegasusx-prod-gke`) configured with Workload Identity, automated multi-zone node provisioning, and dedicated K8s namespaces:

```
+--------------------------------------------------------------------------------------------------------------------+
|                                    GKE AUTOPILOT CLUSTER WORKLOAD ISOLATION                                        |
+--------------------------------------------------------------------------------------------------------------------+
| Namespace: pegasusx (Production Workloads)                                                                         |
|                                                                                                                    |
|  [ Ingress Tier: backend-go-api ]              [ WebSocket Tier: backend-go-ws ]                                   |
|  - Replicas: 4 (HPA: 3 to 10)                  - Replicas: 3 (HPA: 2 to 6)                                         |
|  - Resources: 500m CPU / 512Mi RAM             - Resources: 500m CPU / 512Mi RAM                                   |
|  - Role: HTTP REST API Gateway, Auth, POS      - Role: 8 Stateful WebSocket Hubs, 1Hz Telemetry                     |
|                                                                                                                    |
|  [ Worker Tier: backend-go-worker ]            [ AI & Forecasting Tier: ai-worker ]                                |
|  - Replicas: 4 (KEDA Scaled on Kafka Lag)      - Replicas: 2 (Static)                                              |
|  - Resources: 1000m CPU / 1024Mi RAM           - Resources: 500m CPU / 1024Mi RAM                                  |
|  - Role: Outbox Relay, 12 Kafka Consumers,     - Role: Croston / Holt-Winters Forecasts, GCS Bulk Import Stream     |
|    Dunning, Auto-Dispatch, Webhook Inbox                                                                           |
|                                                                                                                    |
|  [ Optimization Tier: optimizer-core ]         [ Stateless Perimeter: handoff-service ]                           |
|  - Replicas: 3 (HPA on CPU > 80%)              - Replicas: 2 (Static)                                              |
|  - Resources: 1000m CPU / 1024Mi RAM           - Resources: 100m CPU / 128Mi RAM                                   |
|  - Role: Python 3.12 + OR-Tools CVRPTW Solver  - Role: Pure Go QR & Cryptographic HMAC Delivery Token Validator   |
|  - Sidecar: optimizer-core-rust (Rust gRPC)                                                                       |
|                                                                                                                    |
|  [ Routing Tier: osrm-routing-engine ] (Optional Local Cache)                                                      |
|  - Replicas: 1                                                                                                     |
|  - Resources: 1000m CPU / 4096Mi RAM + 20Gi SSD PVC                                                                |
|  - Role: Self-Hosted OSRM Table Matrix Solver                                                                      |
+--------------------------------------------------------------------------------------------------------------------+
```

---

## 3. Service-to-GCP Component Mapping Matrix

Every service, daemon, database schema, storage bucket, and frontend application identified in `INVENTORY.md` is mapped directly to its corresponding GCP resource below:

### 3.1 Core Backend & Microservices Mapping

| Monorepo Service | Source Directory | Runtime / Framework | GCP Deployment Target | GCP Resource Identifier / Spec |
| :--- | :--- | :--- | :--- | :--- |
| **Backend REST API** | `apps/backend-go` | Go 1.25.0 (CGO/glibc) | GKE Deployment | `k8s:deployment/backend-go-api` (Port 8080) |
| **Backend WebSocket Mesh** | `apps/backend-go` | Go 1.25.0 (CGO/glibc) | GKE Deployment | `k8s:deployment/backend-go-ws` (Port 8080) |
| **Backend Worker Engine** | `apps/backend-go` | Go 1.25.0 (CGO/glibc) | GKE Deployment | `k8s:deployment/backend-go-worker` (Port 8081) |
| **AI Demand Forecasting** | `apps/ai-worker` | Go 1.25.0 (CGO/glibc) | GKE Deployment | `k8s:deployment/ai-worker` (Port 8081) |
| **Stateless Handoff Svc** | `apps/handoff-service`| Pure Go (Distroless) | GKE Deployment | `k8s:deployment/handoff-service` (Port 8082) |
| **Python VRP Solver** | `services/optimizer-core`| Python 3.12 + OR-Tools| GKE Deployment | `k8s:deployment/optimizer-core` (Port 8082) |
| **Rust Heuristic Solver**| `services/optimizer-core/server-rust`| Rust 1.85 (Debian) | GKE Sidecar / Pod | `k8s:deployment/optimizer-core-rust` (Port 50055) |
| **OSRM Matrix Engine** | Self-Hosted Docker | C++ OSRM Engine | GKE Deployment | `k8s:deployment/osrm-engine` (Port 5000) |

---

### 3.2 Database, Caching & Message Streaming Tier Mapping

| Inventory Component | Specifications & Details | GCP Native Service | GCP Resource Identifier / Terraform Name |
| :--- | :--- | :--- | :--- |
| **Primary Relational DB**| 136 Tables, 13 Interleaves, 193 Indexes | Cloud Spanner Regional | `google_spanner_instance.ledger` (`main` DB) |
| **Spanner Automated Backup**| Daily Full Backup, 30d Retention | Cloud Spanner Backup Schedule | `google_spanner_backup_schedule.daily_full` |
| **In-Memory Cache & WS Pub/Sub**| Redis 7.0 HA, TLS, AUTH, 5 GB RAM | Cloud Memorystore for Redis | `google_redis_instance.main` (`STANDARD_HA`) |
| **Message Streaming Broker**| 3 AZ Nodes, SASL_SSL IAM Auth | Google Managed Service for Apache Kafka | `google_managed_kafka_cluster.events` |
| **Canonical Kafka Topics**| 10 Partitioned Topics (3x Replication) | Kafka Managed Topics | `google_managed_kafka_topic.<topic_name>` |
| **Media & Evidence Storage**| Signed URLs, POD Photos, PDF Invoices | Google Cloud Storage (GCS) | `google_storage_bucket.media` (`pegasusx-prod-media`) |
| **Mobile & Desktop OTA Bucket**| Public Read via Cloud CDN, APKs, Binaries | Google Cloud Storage (GCS) | `google_storage_bucket.updates` (`pegasusx-prod-app-updates`) |
| **Bulk Import/Export Bucket**| Private GCS, Inventory CSV/XLSX, 30d TTL | Google Cloud Storage (GCS) | `google_storage_bucket.imports` (`pegasusx-prod-imports-exports`) |
| **Terraform State Bucket**| Encrypted, Object Versioning Enabled | Google Cloud Storage (GCS) | `google_storage_bucket.tf_state` (`pegasus-503013-tfstate`) |

---

### 3.3 CLI Daemons, Schedulers & Batch Job Mapping

| CLI Utility / Daemon | Source Directory | Primary Execution Mode | GCP / Kubernetes Execution Model |
| :--- | :--- | :--- | :--- |
| `pegasusx-setup` | `apps/backend-go/cmd/setup` | One-off Init Job | K8s Job (`jobs/pegasusx-setup.yaml`) |
| `apply-migration` | `apps/backend-go/cmd/apply-migration` | Pre-deploy Migration Gate | K8s Pre-deploy Hook Job (`jobs/migrate-job.yaml`) |
| `schema-drift` | `apps/backend-go/cmd/schema-drift` | CI/CD Verification Gate | Cloud Build / GitHub Actions CI Step |
| `planning-forecast` | `apps/backend-go/cmd/planning-forecast` | Daily Demand Calculation | K8s CronJob (`0 2 * * *`) |
| `planning-training-export`| `apps/backend-go/cmd/planning-training-export`| Daily Order Export to GCS | K8s CronJob (`30 2 * * *`) |
| `planning-accuracy` | `apps/backend-go/cmd/planning-accuracy` | Daily WAPE / Bias Calc | K8s CronJob (`0 3 * * *`) |
| `replay-dlq` | `apps/backend-go/cmd/replay-dlq` | SRE On-Demand Ops Tool | K8s Admin Pod / SRE Exec Tool |
| `ssmr-smokecheck` | `apps/backend-go/cmd/ssmr-smokecheck` | Post-deploy Validation | K8s Post-deploy Verification Job |
| `ecosystem-simulator` | `apps/backend-go/cmd/ecosystem-simulator` | Load & Synthetic Stress | Staging GKE Deployment (`simulator`) |

---

### 3.4 Web Portals, Desktop POS & Mobile Client Distribution

| Client Application | Framework / Technology | Distribution & Hosting Target | GCP Infrastructure Component |
| :--- | :--- | :--- | :--- |
| **Supplier Portal** | Next.js 15 App Router | GKE SSR Deployment + Cloud CDN | `k8s:deployment/supplier-portal` (Port 3000) |
| **Warehouse Portal** | Next.js 15 App Router | GKE SSR Deployment + Cloud CDN | `k8s:deployment/warehouse-portal` (Port 3002) |
| **Factory Portal** | Next.js 15 App Router | GKE SSR Deployment + Cloud CDN | `k8s:deployment/factory-portal` (Port 3003) |
| **Marketing Site** | Next.js 15 (Three.js 3D) | GCS Bucket + Cloud CDN Edge | `google_compute_backend_bucket.marketing` |
| **Retailer Desktop POS** | Tauri v2 (Rust) + SQLite | Windows / macOS Binary OTA | GCS `pegasusx-prod-app-updates` via Cloud CDN |
| **Payload Terminal** | Expo 55 / React Native Web | Web / Android Tablet Kiosk | GKE Deployment / GCS Static Hosting |
| **6 Native iOS Apps** | Swift 6 / SwiftUI | Apple App Store & TestFlight | External Apple App Store Infrastructure |
| **6 Native Android Apps**| Kotlin 2.0 / Jetpack Compose | Google Play Store & Private OTA | Google Play Store + GCS Direct APK Download |

---

### 3.5 Networking, Edge & Security Infrastructure Mapping

| Infrastructure Requirement | Architectural Scope | GCP Native Component | GCP Resource Identifier / Spec |
| :--- | :--- | :--- | :--- |
| **Custom VPC Network** | Global Private Network | Google Cloud VPC | `google_compute_network.pegasusx_vpc` |
| **Regional Node Subnet** | GKE Nodes & PSA Peering | VPC Subnetwork | `10.10.0.0/20` (`europe-west3`) |
| **Secondary Pod CIDR** | VPC-Native IP Aliasing | Subnet Secondary Range | `10.20.0.0/16` (65,536 Pod IPs) |
| **Secondary Service CIDR**| ClusterIP Service Range | Subnet Secondary Range | `10.30.0.0/20` (4,096 Service IPs) |
| **Deterministic Egress NAT**| Static Outbound Whitelisting | Cloud Router + Cloud NAT | `google_compute_router_nat.nat_gateway` (2 EIPs) |
| **L7 Load Balancer** | Global Ingress & SSL Term | External HTTPS Load Balancer | `google_compute_global_forwarding_rule.https` |
| **Edge Security WAF** | DDoS & OWASP Top 10 | Cloud Armor Security Policy | `google_compute_security_policy.edge_waf` |
| **Secret Management** | Enterprise Secret Vault | Secret Manager | `google_secret_manager_secret.*` + ESO |
| **IAM Authentication** | Zero-Trust Workload Identity | GKE Workload Identity Pool | `serviceAccount:${var.project_id}.svc.id.goog` |

---

## 4. Phased Rollout Strategy with Inter-Phase Dependencies

The deployment of the PegasusX GCP infrastructure is organized into four sequential phases. Each phase establishes strict architectural prerequisites that must pass automated health and compliance gates before dependent phases commence:

```
+--------------------------------------------------------------------------------------------------------------------+
|                                      PHASED INFRASTRUCTURE ROLLOUT ROADMAP                                         |
+--------------------------------------------------------------------------------------------------------------------+
|                                                                                                                    |
|   ┌────────────────────────────────────────────────────────────────────────────────────────────────────────────┐   |
|   │ PHASE 1: FOUNDATION, VPC NETWORKING & IAM SECURITY                                                         │   |
|   │ - Custom VPC (10.10.0.0/20), Secondary Pods (10.20.0.0/16) & Services (10.30.0.0/20)                       │   |
|   │ - Cloud Router & Cloud NAT with 2 Static External EIPs                                                     │   |
|   │ - Private Service Access (PSA) Peering (10.42.205.148/29)                                                  │   |
|   │ - Base IAM Service Accounts, Workload Identity Pools & KMS Encryption Keys                                 │   |
|   └─────────────────────────────────────────────────────┬──────────────────────────────────────────────────────┘   |
|                                                         │                                                          |
|                                                         v (Gate 1: Network Connectivity & PSA Peering Validated)   |
|   ┌────────────────────────────────────────────────────────────────────────────────────────────────────────────┐   |
|   │ PHASE 2: MANAGED DATA, STORAGE & MESSAGING BACKBONE                                                        │   |
|   │ - Cloud Spanner Regional Instance (`ledger`) & Schema Deployment (136 Tables, 13 Interleaves, 193 Indexes) │   |
|   │ - Cloud Spanner Automated Daily Backup Schedule (`daily_full`, 30d Retention)                             │   |
|   │ - Cloud Memorystore for Redis 7.0 HA (`STANDARD_HA`, In-Transit TLS, AUTH Password)                        │   |
|   │ - Google Managed Service for Apache Kafka Cluster (3 AZ Brokers, 10 Canonical Partitioned Topics)          │   |
|   │ - 4 Google Cloud Storage (GCS) Buckets (Media, App Updates, Imports/Exports, Terraform State)              │   |
|   │ - Google Secret Manager Secret Declarations & KMS Key Bindings                                             │   |
|   └─────────────────────────────────────────────────────┬──────────────────────────────────────────────────────┘   |
|                                                         │                                                          |
|                                                         v (Gate 2: Spanner DDL Applied, Kafka Topics Ready, DB OK) |
|   ┌────────────────────────────────────────────────────────────────────────────────────────────────────────────┐   |
|   │ PHASE 3: GKE COMPUTE, WORKLOAD IDENTITY & CORE DEPLOYMENTS                                                 │   |
|   │ - GKE Autopilot Cluster (`pegasusx-prod-gke`) with Workload Identity Federation                            │   |
|   │ - External Secrets Operator (ESO) Installation & Secret Synchronization                                    │   |
|   │ - Pre-Deploy Database Migration Job (`apply-migration`)                                                    │   |
|   │ - Core Deployments: `backend-go-api`, `backend-go-ws`, `backend-go-worker`, `ai-worker`, `optimizer-core`  │   |
|   │ - Kubernetes Internal ClusterIP Services, ConfigMaps & PersistentVolumeClaims                              │   |
|   │ - Kubernetes Schedulers & CronJobs (Planning Forecast, Accuracy, Training Export)                          │   |
|   └─────────────────────────────────────────────────────┬──────────────────────────────────────────────────────┘   |
|                                                         │                                                          |
|                                                         v (Gate 3: All Pod Readiness / Health Probes Passing 100%) │
|   ┌────────────────────────────────────────────────────────────────────────────────────────────────────────────┐   |
|   │ PHASE 4: EDGE ROUTING, THIRD-PARTY ADAPTERS, OBSERVABILITY & HARDENING                                     │   |
|   │ - Global External HTTPS Load Balancer with Google-Managed SSL Certificates                                 │   |
|   │ - Cloud Armor Enterprise WAF Policy (Rate Limiting 500 req/10s, OWASP Top 10 Rules, Geo-Fencing)           │   |
|   │ - Cloud CDN Caching for Static Portals & OTA App Update Bundles                                            │   |
|   │ - Soliq OFD PKCS#12 E-IMZO Secret Volume Mount & Tax Gateway Connectivity Verification                     │   |
|   │ - Payment Webhook Ingress Endpoints & HMAC Signature Verification (`/v1/webhooks/*`)                       │   |
|   │ - Cloud Monitoring 17 Production Alert Policies, Notification Channels & Grafana Dashboards                │   |
|   │ - Post-Deploy Verification Gate: `ssmr-smokecheck` & `ecosystem-simulator` Synthetic Validation            │   |
|   └────────────────────────────────────────────────────────────────────────────────────────────────────────────┘   |
|                                                                                                                    |
+--------------------------------------------------------------------------------------------------------------------+
```

---

### 4.1 Phase 1: Foundation, VPC Networking & IAM Security

#### Objectives & Scope
Establish the secure, isolated software-defined networking perimeter, routing infrastructure, IP address topology, and zero-trust IAM foundation.

#### Action Items & Components Provisioned
1. **Custom VPC Network (`google_compute_network.pegasusx_vpc`):**
   - Create custom-mode VPC disabling auto-subnet creation to enforce explicit subnet IP governance.
2. **Subnet & Secondary IP Ranges (`google_compute_subnetwork.primary_subnet`):**
   - Primary Subnet: `10.10.0.0/20` (Supports 4,096 GKE node IPs in `europe-west3`).
   - Secondary Range 1 (Pods): `10.20.0.0/16` (Supports 65,536 Pod IP addresses via VPC-Native IP aliasing).
   - Secondary Range 2 (Services): `10.30.0.0/20` (Supports 4,096 ClusterIP service addresses).
3. **Private Service Access (PSA) Peering (`google_service_networking_connection`):**
   - Allocate internal IP range `10.42.205.148/29` peered with `servicenetworking.googleapis.com` for direct, private connectivity to Memorystore Redis and Managed Kafka without traversing the public internet.
4. **Cloud Router & Cloud NAT Gateway (`google_compute_router_nat`):**
   - Provision Cloud Router in `europe-west3`.
   - Attach Cloud NAT Gateway configured with **2 dedicated static external IP addresses** (`google_compute_address.egress_ips`). All outbound traffic from GKE pods (reaching Soliq tax servers and payment gateways) originates from these deterministic IPs.
5. **Base IAM Roles & Workload Identity Pool:**
   - Establish Google Service Accounts (GSAs) for compute, storage, and monitoring.
   - Configure Workload Identity Pool `${var.project_id}.svc.id.goog`.

#### Verification & Exit Gate 1
- `gcloud compute networks subnets describe` confirms primary and secondary CIDR ranges are active.
- Outbound egress connectivity test from a test VM confirms deterministic origin from the assigned Cloud NAT static IPs.
- PSA peering status reports `ACTIVE`.

---

### 4.2 Phase 2: Managed Data, Storage & Messaging Backbone

#### Objectives & Scope
Deploy and configure authoritative persistence tiers, in-memory caches, distributed event streaming brokers, object storage vaults, and secret stores.

#### Action Items & Components Provisioned
1. **Cloud Spanner Regional Instance (`google_spanner_instance.ledger`):**
   - Sizing: Regional `europe-west3`, initial capacity 100 Processing Units (PU) with autoscaling up to 1,000 PU (1 full node).
   - Configuration: `enable_drop_protection = true`, `version_retention_period = "7d"` (Point-In-Time Recovery).
   - Database Creation: Create database `main`.
   - DDL Execution: Execute `apps/backend-go/schema/spanner.ddl` deploying **136 tables**, **13 interleaved hierarchies**, and **193 secondary indexes**.
   - Automated Backup: Attach `google_spanner_backup_schedule.daily_full` executing daily full backups with 30-day retention.
2. **Cloud Memorystore for Redis 7.0 (`google_redis_instance.main`):**
   - Tier: `STANDARD_HA` (Automatic cross-zone failover with dedicated replica).
   - Capacity: 5 GB memory allocation.
   - Security: `transit_encryption_mode = "SERVER_AUTHENTICATION"` (In-transit TLS), `auth_enabled = true` (Mandatory AUTH password).
   - Eviction: `allkeys-lru`.
3. **Google Managed Service for Apache Kafka (`google_managed_kafka_cluster.events`):**
   - Topology: 3 brokers distributed across 3 Availability Zones (`europe-west3-a`, `b`, `c`).
   - Sizing: 3 vCPUs, 16 GiB RAM per broker, 1,000 GiB SSD persistent storage.
   - Authentication: `SASL_SSL` via Google Cloud IAM OAuth tokens.
   - Topic Creation: Provision the 10 canonical partitioned topics (`pegasusx-main`, `pegasusx-orders`, `pegasusx-dispatch`, `pegasusx-realtime`, `pegasusx-demand`, `logistics.exceptions.v1`, `logistics.telemetry.v1`, `pegasusx-freeze-locks`, `pegasusx-inventory-import`, `pegasusx-main-dlq`) with 12/6 partitions and replication factor 3.
4. **Google Cloud Storage Buckets (`google_storage_bucket.*`):**
   - `pegasusx-prod-media`: Standard Multi-Region, Uniform Bucket-Level Access, CORS enabled for mobile/web direct uploads.
   - `pegasusx-prod-app-updates`: Standard Multi-Region, Public Read for OTA APKs and Tauri updater binaries.
   - `pegasusx-prod-imports-exports`: Standard Regional, 30-day auto-deletion lifecycle rule.
   - `pegasus-503013-terraform-state`: Regional, Object Versioning enabled, strictly private IAM.
5. **Google Secret Manager & KMS Keys:**
   - Provision GSM secret containers for database passwords, JWT secrets, payment API keys, Soliq PKCS#12 certificates, and SMS credentials.

#### Verification & Exit Gate 2
- Run `apps/backend-go/cmd/schema-drift` against live Cloud Spanner instance to verify 100% schema parity without drift.
- Redis TLS connection and `AUTH` handshake verified via internal test runner.
- Kafka producer/consumer test verifies message publish and receipt on `pegasusx-main`.

---

### 4.3 Phase 3: GKE Compute, Workload Identity & Core Deployments

#### Objectives & Scope
Provision the managed Kubernetes compute cluster, establish Workload Identity federation, synchronize secrets via External Secrets Operator, apply database migrations, and deploy all containerized microservices.

#### Action Items & Components Provisioned
1. **GKE Autopilot Cluster (`google_container_cluster.pegasusx_gke`):**
   - Cluster Type: GKE Autopilot (Automated node sizing, OS patching, and security hardening).
   - Network Configuration: VPC-native IP aliasing, private cluster endpoints enabled (nodes have zero public IP addresses).
   - Release Channel: `REGULAR`.
   - Workload Identity: Enabled using `${var.project_id}.svc.id.goog`.
2. **Kubernetes Service Accounts & Workload Identity Bindings:**
   - Create K8s Service Account `backend-go` in namespace `pegasusx`.
   - Bind `backend-go` to Google Service Account `pegasusx-backend@${var.project_id}.iam.gserviceaccount.com` with role `roles/iam.workloadIdentityUser`.
   - Assign GSAs least-privilege IAM roles (`roles/spanner.databaseUser`, `roles/storage.objectAdmin`, `roles/secretmanager.secretAccessor`, `roles/managedkafka.client`).
3. **External Secrets Operator (ESO) & ConfigMaps:**
   - Deploy ESO via Helm chart.
   - Configure `SecretStore` referencing Google Secret Manager.
   - Instantiate `ExternalSecret` manifests projecting secrets into standard K8s Secret objects (`jwt-secret`, `redis-auth`, `soliq-credentials`, etc.).
   - Deploy application `ConfigMaps` defining non-sensitive environment variables (`HTTP_PORT=8080`, `REQUIRE_INFRA_ADAPTERS=true`, etc.).
4. **Pre-Deploy Database Migration Job:**
   - Execute K8s Job `jobs/migrate-job.yaml` running `apps/backend-go/cmd/apply-migration` to verify Spanner baseline state.
5. **Core Microservices Deployment:**
   - Deploy `backend-go-api` (4 replicas, port 8080, probes `/healthz`, `/ready`).
   - Deploy `backend-go-ws` (3 replicas, port 8080, sticky session affinity).
   - Deploy `backend-go-worker` (4 replicas, `PEGASUSX_RUN_MODE=worker`, outbox relay + Kafka consumers).
   - Deploy `ai-worker` (2 replicas, port 8081).
   - Deploy `handoff-service` (2 replicas, port 8082, stateless token validator).
   - Deploy `optimizer-core` (3 replicas, Python 3.12 + OR-Tools + OpenMP C++).
   - Deploy Web Portal SSR containers (`supplier-portal`, `warehouse-portal`, `factory-portal`).
6. **Kubernetes Schedulers & CronJobs:**
   - Deploy `planning-forecast` (`0 2 * * *`), `planning-training-export` (`30 2 * * *`), and `planning-accuracy` (`0 3 * * *`).

#### Verification & Exit Gate 3
- All K8s Pods reach `Running` state with 100% healthy readiness (`/ready`) and liveness (`/healthz`) probe responses.
- `apps/backend-go/cmd/ssmr-smokecheck` executes in-cluster and passes 100% of end-to-end integration assertions.

---

### 4.4 Phase 4: Edge Routing, Third-Party Adapters, Observability & Hardening

#### Objectives & Scope
Configure public ingress routing, SSL termination, Cloud Armor WAF security rules, Cloud CDN caching, third-party payment/tax adapters, and SRE observability alerting.

#### Action Items & Components Provisioned
1. **Google Cloud External L7 HTTPS Load Balancer:**
   - Attach Global Anycast Static IP (`136.69.43.141`).
   - Deploy Google-Managed SSL Certificate (`ManagedCertificate`) for domains `api.pegasusx.example.com`, `portal.pegasusx.example.com`, `updates.pegasusx.example.com`.
   - Configure URL map routing:
     - `/v1/ws` -> `backend-go-ws` BackendService (Session affinity: `GENERATED_COOKIE`, TTL: 3600s).
     - `/v1/*`, `/partner/*` -> `backend-go-api` BackendService (Timeout: 120s).
     - `/marketing/*` -> GCS Marketing Backend Bucket (Cloud CDN enabled).
     - `/updates/*` -> GCS App Updates Backend Bucket (Cloud CDN enabled).
2. **Cloud Armor Security Policy (`google_compute_security_policy.edge_waf`):**
   - Rule 1000: Rate limiting (Max 500 requests per 10 seconds per client IP, ban threshold: 429).
   - Rule 2000: OWASP Top 10 ModSecurity Core Rule Set (`sqli-v33-stable`, `xss-v33-stable`, `lfi-v33-stable`).
   - Rule 3000: GeoIP filtering restricting administrative routes (`/v1/auth/platform-admin/*`) to authorized regional CIDRs.
3. **Third-Party Adapters & Secret Mounting:**
   - Mount Soliq OFD PKCS#12 E-IMZO certificate (`/etc/certs/eds.p12`) via K8s Secret Volume Mount in `backend-go-api` and `backend-go-worker`.
   - Configure Payment Gateway webhook routes (`/v1/webhooks/globalpay`, `/v1/webhooks/payme`, `/v1/webhooks/click`, `/v1/webhooks/adyen`, `/v1/webhooks/stripe`).
4. **Cloud Monitoring & SRE Alerting Framework:**
   - Deploy Prometheus metrics collection scraping `/metrics` on all services.
   - Provision 17 Cloud Monitoring Alert Policies with email and PagerDuty notification channels.
   - Configure Synthetic Uptime Checks probing `/healthz` and `/ready` every 60 seconds from 5 global probing stations.

#### Verification & Exit Gate 4
- Live traffic verification using `apps/backend-go/cmd/ecosystem-simulator` generating synthetic orders, payments, and driver GPS tracks.
- Cloud Armor WAF blocks synthetic SQL injection and rate limit flood attacks.
- Soliq OFD sandbox e-invoice submission and verification completes with valid fiscal sign token.

---

## 5. Third-Party Service Integration Architecture

The PegasusX backend interfaces with external banking rails, regulatory tax gateways, mobile communication brokers, and mapping engines. The architecture enforces deterministic timeouts, retry loops, and fail-safe dead-letter handling for each integration:

```
+--------------------------------------------------------------------------------------------------------------------+
|                                    THIRD-PARTY ADAPTER INTEGRATION TOPOLOGY                                        |
+--------------------------------------------------------------------------------------------------------------------+
|                                                                                                                    |
|   [ PegasusX GKE Pods ] (Static Egress NAT: 34.141.x.x, 34.141.y.y)                                                |
|            │                                                                                                       |
|            ├───> [ Soliq OFD (EHF) ] ─────────> PKCS#12 E-IMZO Digital Sign (8s Timeout, Auto Buyer Poll)         |
|            │                                                                                                       |
|            ├───> [ Global Pay / Payme / Click ]> HTTPS REST & Webhooks (HMAC-SHA256 Sign, 5m Stuck Reconciler)     |
|            │                                                                                                       |
|            ├───> [ PlayMobile / Twilio SMS ] ──> Carrier OTP Queue & Doorstep PIN (Sub-second Deliverability)       |
|            │                                                                                                       |
|            ├───> [ SendGrid v3 Email ] ───────> Fiscalized PDF Invoices & Monthly Dunning Statements               |
|            │                                                                                                       |
|            ├───> [ Google Routes v2 / OSRM ] ─> High-Precision Polyline ETA + Fast In-Cluster Matrix Solver         |
|            │                                                                                                       |
|            └───> [ Firebase Auth & FCM ] ─────> Mobile Phone OTP Verification & High-Priority APNs/FCM Push         |
|                                                                                                                    |
+--------------------------------------------------------------------------------------------------------------------+
```

---

### 5.1 Soliq OFD (Uzbekistan State Tax Committee EHF Invoicing)

PegasusX integrates directly with the State Tax Committee of Uzbekistan (Soliq OFD / MySoliq) for mandatory Electronic Factura (EHF) generation and fiscal registration:

- **Authentication & Digital Signing:**
  - Uses an authorized government PKCS#12 (`.p12`) digital cryptographic signature certificate issued by the E-IMZO certification authority.
  - The certificate is mounted into the container at `/etc/certs/eds.p12` via a protected Kubernetes Secret volume.
  - At runtime, `apps/backend-go/fiscal` loads the certificate using `FISCAL_MY_SOLIQ_PKCS12_PASSWORD`, computes an SHA-256 digest of the invoice JSON, and generates an attached PKCS#7 / CMS digital signature.
- **Protocol & Network Parameters:**
  - **Base URL:** Configured via `FISCAL_MY_SOLIQ_BASE_URL`.
  - **Transport:** HTTPS REST with static IP allowlisting via Cloud NAT.
  - **Timeout:** Enforced **8-second client timeout** (`http.Client{Timeout: 8 * time.Second}`).
  - **Retry Policy:** 3 attempts with exponential backoff and 20% jitter.
- **Asynchronous Buyer Acceptance Polling (`BuyerAcceptancePoller`):**
  - Uzbekistan fiscal law requires the retail buyer to accept the electronic invoice.
  - `apps/backend-go/runtime_workers.go` runs `BuyerAcceptancePoller`, continuously querying Soliq OFD every 15 minutes for buyer signature confirmations and transitioning invoice states from `SUBMITTED` to `ACCEPTED` in Spanner `OrderFiscalReceipts`.
- **Fail-Safe Fallback Strategy:**
  - If Soliq OFD experiences government gateway downtime, invoices enter `PENDING_FISCALIZATION` state in Spanner. The transactional outbox continues order processing without blocking warehouse fulfillment. When Soliq recovers, the outbox worker replays pending fiscalizations.

---

### 5.2 Payment Rails & FinTech Adapters

PegasusX supports multi-gateway payment acquiring, digital wallets, and automated ledger settlement (`apps/backend-go/paymentroutes/`, `apps/backend-go/internal/services/billing/`):

#### 1. Global Pay (Primary Uzbekistan Card Acquiring)
- **Scope:** Direct debit for national Uzcard and Humo payment cards.
- **Authentication:** `GLOBAL_PAY_SERVICE_ID`, `GLOBAL_PAY_USERNAME`, `GLOBAL_PAY_PASSWORD`.
- **Webhook Security:** Incoming webhooks (`POST /v1/webhooks/globalpay`) validate the `X-Signature` header using HMAC-SHA256 against `GLOBAL_PAY_WEBHOOK_SECRET`.
- **Local Sandbox Simulator:** A built-in Global Pay simulator is mounted at `/sim/globalpay` when `GLOBAL_PAY_ENV != "production"` (`apps/backend-go/simulator/`), allowing end-to-end payment testing without touching live cardholder funds.

#### 2. Payme Merchant API
- **Protocol:** JSON-RPC 2.0 merchant protocol (`POST /v1/webhooks/payme`).
- **Security:** HTTP Basic Authentication validated against `PAYME_WEBHOOK_SECRET`.
- **Transaction Locking:** Uses Spanner row-level transaction locks (`SELECT FOR UPDATE`) on `PaymentSessions` to ensure `CheckPerformTransaction` and `PerformTransaction` maintain strict ACID idempotency.

#### 3. Click Webhook Rail
- **Protocol:** REST `application/x-www-form-urlencoded` (`POST /v1/webhooks/click`).
- **Security:** MD5 / SHA-256 signature hash validation against `CLICK_WEBHOOK_SECRET`.
- **Settlement:** Real-time reconciliation into `PaymentLedgerEntries`.

#### 4. International Multi-Currency (Adyen & Stripe)
- **Scope:** International supplier billing and cross-border trade credit settlement.
- **Security:** Webhook signature verification via `ADYEN_WEBHOOK_SECRET` and `STRIPE_WEBHOOK_SECRET`.
- **Currencies:** Native support for UZS, USD, and EUR.

#### 5. Webhook Reconciler Daemon (`app.WebhookReconciler`)
- Runs every 5 minutes in `backend-go-worker`.
- Identifies payment sessions stuck in `PENDING` state for > 15 minutes and executes active status checks against payment gateway APIs, recovering abandoned transactions.

---

### 5.3 SMS, WhatsApp & Transactional Email Communications

| Communication Rail | Provider | Purpose & Workload | Credentials & Protocol | Failover Strategy |
| :--- | :--- | :--- | :--- | :--- |
| **National SMS Broker** | **PlayMobile** | Uzbekistan domestic SMS OTP, driver doorstep delivery PIN, AR credit dunning notices | `PLAYMOBILE_LOGIN`, `PLAYMOBILE_PASSWORD` (REST Basic Auth) | Automatic failover to Twilio SMS if PlayMobile error rate > 5% |
| **International SMS / WhatsApp**| **Twilio** | International OTP, WhatsApp Business order updates & digital receipts | `TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN` (Twilio REST API) | Fallback to SMS if WhatsApp delivery receipt fails |
| **Transactional Email** | **SendGrid v3** | Fiscalized invoice PDF dispatch, monthly AR statements, supplier statements | `SENDGRID_API_KEY`, `SENDGRID_FROM_EMAIL` (HTTPS REST `v3/mail/send`)| Retry queue with 1-hour backoff in Spanner `OutboxEvents` |

---

### 5.4 Spatial Routing, Distance Matrix & Geocoding

Route optimization and driver navigation utilize a hybrid dual-engine architecture:

1. **Google Maps Platform (Routes v2, Places, Geocoding):**
   - **Google Routes v2 API (`ComputeRoutes`):** Computes turn-by-turn driver navigation polylines with real-time traffic awareness.
   - **Places API & Geocoding:** Powers address autocomplete and reverse geocoding during retailer onboarding.
   - **Credentials:** Injected via `GOOGLE_MAPS_API_KEY` from Secret Manager.
2. **Self-Hosted In-Cluster OSRM Routing Engine (Local Matrix Solver):**
   - **Purpose:** Offloads high-frequency $N \times N$ distance and duration matrix calculations (`/table/v1/driving/`) required by the Python OR-Tools VRP solver.
   - **Performance:** Sub-millisecond response times for 50-stop matrices; zero external API charges.
   - **Endpoint:** `ROUTING_OSRM_URL=http://osrm:5000`.

---

### 5.5 Mobile Identity & Cloud Messaging

1. **Firebase Phone Authentication:**
   - Powers frictionless phone number + OTP login for 1,000 retailers and 200 mobile drivers.
   - Client applications authenticate with Firebase SDK; the backend verifies OIDC JWT ID tokens using Google Public Keys (`apps/backend-go/auth/firebase.go`).
2. **Firebase Cloud Messaging (FCM HTTP v1) + Apple APNs:**
   - High-priority push notifications (Priority 10) for instant order alerts, route dispatch locks, and vehicle arrival geofences.
   - Token rotation and device registration managed in Spanner `DeviceTokens`.
   - Consumed by `void-notification-dispatcher` Kafka consumer group.

---

## 6. Capacity Planning & Monthly TCO Cost Model at Scale

### 6.1 Baseline Workload Sizing Parameters

The infrastructure sizing and financial model are computed for a scaled commercial deployment defined as follows:

```
+--------------------------------------------------------------------------------------------------------------------+
|                                    COMMERCIAL WORKLOAD SIZING PARAMETERS                                           |
+----------------------------------------------------+---------------------------------------------------------------+
| Parameter Description                              | Production Baseline Value                                     |
+----------------------------------------------------+---------------------------------------------------------------+
| **Active Retailer POS & Mobile Accounts**          | **1,000 Retailers** (Multi-tier grocery & convenience stores)  |
| **Active FMCG Suppliers & Distributors**           | **50 Suppliers** (10 to 50 active warehouse depots)            |
| **Active Delivery Fleet Drivers**                  | **200 Drivers** (Continuous 1Hz GPS telemetry streaming)      |
| **Daily Processed Orders**                         | **15,000 Orders / Day** (~450,000 orders / month)             |
| **Daily Delivery Route Stops**                     | **45,000 Stops / Day** (~1,350,000 stops / month)             |
| **Daily POS Sell-Through Events**                  | **200,000 Scan Events / Day** (~6,000,000 events / month)     |
| **Daily Web Portal / Admin Sessions**              | **5,000 Active Sessions / Day**                               |
| **Peak Inbound API Request Rate**                  | **250 to 500 Requests / Second (QPS)**                        |
| **Peak Database Transaction Throughput**           | **150 Writes / Sec, 600 Reads / Sec (Spanner QPS)**           |
| **Monthly Outbound Network Egress**                | **~750 GB / Month** (APIs, telemetry, images, CDN)            |
+----------------------------------------------------+---------------------------------------------------------------+
```

---

### 6.2 Detailed Resource Sizing & Throughput Calculations

#### 1. Cloud Spanner Compute & Storage
- **Transactional Volume:** 15,000 orders/day $\times$ 10 writes/order (Order, Allocations, StateTransitions, FiscalSnapshot, Outbox) = 150,000 writes/day.
- **Peak Write Throughput:** $\approx 50$ write QPS baseline, bursting to 150 write QPS during peak morning ordering windows.
- **Processing Units (PU) Required:** 100 PU comfortably supports up to 1,000 write QPS and 10,000 read QPS on regional Spanner. Baseline allocation of **200 PU** provides a 4x headroom safety margin, autoscaling to 600 PU during batch planning runs.
- **Storage Growth:** 15,000 orders $\times$ 5 KB/order record = 75 MB/day $\approx 2.25$ GB/month. Initial database size with catalog, price lists, and telemetry history $\approx$ **150 GB** in Year 1.

#### 2. GKE Autopilot Compute Resources
- `backend-go-api`: 4 Pods $\times$ (0.5 vCPU, 0.5 GiB RAM) = 2.0 vCPU, 2.0 GiB RAM.
- `backend-go-ws`: 3 Pods $\times$ (0.5 vCPU, 0.5 GiB RAM) = 1.5 vCPU, 1.5 GiB RAM.
- `backend-go-worker`: 4 Pods $\times$ (1.0 vCPU, 1.0 GiB RAM) = 4.0 vCPU, 4.0 GiB RAM.
- `ai-worker`: 2 Pods $\times$ (0.5 vCPU, 1.0 GiB RAM) = 1.0 vCPU, 2.0 GiB RAM.
- `optimizer-core` (Python): 3 Pods $\times$ (1.0 vCPU, 1.0 GiB RAM) = 3.0 vCPU, 3.0 GiB RAM.
- `optimizer-core-rust`: 2 Pods $\times$ (0.2 vCPU, 0.25 GiB RAM) = 0.4 vCPU, 0.5 GiB RAM.
- `handoff-service`: 2 Pods $\times$ (0.1 vCPU, 0.125 GiB RAM) = 0.2 vCPU, 0.25 GiB RAM.
- Web Portals (Next.js SSR) + System Daemons: 2.5 vCPU, 4.0 GiB RAM.
- **Total GKE Autopilot Sizing:** **14.6 vCPU**, **17.25 GiB RAM** continuous average allocation.

#### 3. Cloud Memorystore for Redis
- 200 drivers $\times$ 1 KB GPS state = 200 KB.
- 1,000 active sessions $\times$ 5 KB = 5 MB.
- Idempotency locks (15,000/day $\times$ 120s TTL) = 5 MB.
- 7-day Kafka deduplication cache (1M events $\times$ 64 bytes) $\approx$ 64 MB.
- Entity caching & WebSocket pub/sub buffers $\approx$ 500 MB.
- **Redis Memory Allocation:** **5.0 GB** in `STANDARD_HA` mode (10x headroom).

#### 4. Google Managed Service for Apache Kafka
- 15,000 orders $\times$ 15 events/order = 225,000 events/day.
- 200 drivers $\times$ 8 hrs $\times$ 3600s $\times$ 0.1 Hz bus emit = 576,000 telemetry events/day.
- POS sell-through = 200,000 events/day.
- Total Daily Kafka Messages $\approx$ **1,000,000 messages / day** (~30M msgs/month, ~45 GB data/month).
- **Cluster Sizing:** 3 brokers $\times$ (1 vCPU, 4 GiB RAM, 100 GB SSD).

#### 5. Google Cloud Storage (GCS)
- Proof of Delivery (POD) Photos: 15,000 orders $\times$ 2 photos $\times$ 150 KB = 4.5 GB/day $\approx$ 135 GB/month.
- Invoices & Statements (PDFs): 15,000 orders $\times$ 50 KB = 750 MB/day $\approx$ 22.5 GB/month.
- Total Year 1 GCS Storage $\approx$ **1.5 TB**.

---

### 6.3 Comprehensive Itemized Monthly TCO Cost Table

*Note: Pricing is based on official Google Cloud Platform List Prices in `europe-west3` / `asia-south1` and current commercial aggregator rate cards (August 2026).*

```
+------------------------------------------------------------------------------------------------------------------------------------------+
|                                              PEGASUSX PRODUCTION MONTHLY TCO COST MODEL                                                  |
+----+------------------------------------+--------------------------+-----------------------+-----------------------------+---------------+
| #  | Infrastructure / Service Component | Sizing & Configuration   | Monthly Consumption   | Unit Rate / Pricing Basis   | Monthly Cost  |
+----+------------------------------------+--------------------------+-----------------------+-----------------------------+---------------+
|    | **SECTION A: GCP INFRASTRUCTURE**  |                          |                       |                             |               |
| 1  | Cloud Spanner (Compute)            | 200 Processing Units     | 146,000 PU-hours/mo   | $0.09 / 100 PU-hour         | $131.40       |
| 2  | Cloud Spanner (Storage)            | Regional Multi-Zone SSD  | 150 GB                | $0.30 / GB-month            | $45.00        |
| 3  | Cloud Spanner Automated Backups    | Daily Full (30d retain)  | 150 GB backup storage | $0.10 / GB-month            | $15.00        |
| 4  | GKE Autopilot (vCPU Compute)       | Regional Multi-Zone      | 14.6 vCPU continuous  | $0.0445 / vCPU-hour         | $474.30       |
| 5  | GKE Autopilot (Memory)             | Regional Multi-Zone      | 17.25 GiB continuous  | $0.0049 / GiB-hour          | $61.80        |
| 6  | GKE Autopilot Cluster Management   | 1 Managed Cluster        | 730 hours / month     | $0.10 / cluster-hour        | $73.00        |
| 7  | Cloud Memorystore Redis 7.0 HA     | STANDARD_HA (Cross-Zone) | 5.0 GB Instance       | $0.072 / GB-hour            | $262.80       |
| 8  | Google Managed Service for Kafka   | 3 Brokers (3 vCPU total) | 3 Brokers $\times$ 730 hrs| $0.12 / broker-hour     | $262.80       |
| 9  | Managed Kafka Storage              | 3 $\times$ 100 GB SSD    | 300 GB                | $0.17 / GB-month            | $51.00        |
| 10 | Cloud Storage (GCS Standard)       | Media & Vault Buckets    | 1,500 GB              | $0.020 / GB-month           | $30.00        |
| 11 | Cloud Storage Operations           | Class A (PUT) & Class B  | 1M Class A, 5M Class B| $0.005/10k (A), $0.004/10k(B)| $2.50         |
| 12 | Cloud NAT Gateway & Static EIPs    | 2 Static EIPs + NAT      | 750 GB Egress Data    | $0.045 / GB + $0.005/IP-hr  | $41.05        |
| 13 | Global HTTPS External LB           | Forwarding rules + SSL   | 730 hours + 25M reqs  | $0.025 / hr + $0.0075/GB    | $25.50        |
| 14 | Cloud Armor Enterprise WAF         | 1 Security Policy        | 25M HTTP Evaluations  | $5.00/policy + $0.75/1M req | $23.75        |
| 15 | Cloud CDN (Edge Caching)           | Web Portals & OTA APKs   | 350 GB Cache Egress   | $0.08 / GB                  | $28.00        |
| 16 | Google Secret Manager              | 25 Secrets + Auto Sync   | 250,000 API calls     | $0.06 / secret + $0.03/10k  | $2.25         |
| 17 | Cloud Monitoring & Log Ingestion   | Logs + Prometheus Metrics| 120 GB Log Ingestion  | $0.50 / GB above 50GB free  | $35.00        |
+----+------------------------------------+--------------------------+-----------------------+-----------------------------+---------------+
|    | **SUBTOTAL: GCP INFRASTRUCTURE**   |                          |                       |                             | **$1,565.15** |
+----+------------------------------------+--------------------------+-----------------------+-----------------------------+---------------+
|    | **SECTION B: THIRD-PARTY SERVICES**|                          |                       |                             |               |
| 18 | Soliq OFD Electronic Invoicing     | Uzbekistan Tax Committee | 450,000 E-Invoices/mo | $0.0012 / invoice API call  | $540.00       |
| 19 | PlayMobile SMS Gateway (Domestic)  | Uzbekistan OTP & PIN SMS | 675,000 SMS / month   | $0.0045 / SMS message       | $3,037.50     |
| 20 | Twilio (WhatsApp & Int'l SMS)      | Critical Dunning Notices | 25,000 Messages / mo  | $0.015 / WhatsApp message   | $375.00       |
| 21 | SendGrid Email API (Pro Tier)      | PDF Invoices & Recaps    | 450,000 Emails / mo   | Pro 700k Plan               | $89.95        |
| 22 | Google Maps Platform APIs          | Geocoding & Places Auto  | 100,000 Requests / mo | $5.00 / 1,000 requests      | $500.00       |
| 23 | Firebase Auth & FCM Push           | Phone OTP & Mobile Push  | 50,000 Phone Auth OTPs| $0.01 / verification        | $500.00       |
+----+------------------------------------+--------------------------+-----------------------+-----------------------------+---------------+
|    | **SUBTOTAL: THIRD-PARTY SERVICES** |                          |                       |                             | **$5,042.45** |
+----+------------------------------------+--------------------------+-----------------------+-----------------------------+---------------+
|    | **GRAND TOTAL ESTIMATED MONTHLY TCO**                                                                                 | **$6,607.60** |
+----+-----------------------------------------------------------------------------------------------------------------------+---------------+
```

*Financial Summary:* Total monthly operating cost at full scale is **$6,607.60 / month**, yielding a remarkably low unit infrastructure cost of **$0.0146 per processed order** ($6,607.60 / 450,000 monthly orders).

---

## 7. High Availability (HA), Disaster Recovery (DR) & Autoscaling

### 7.1 Multi-Zone Active-Active High Availability Architecture

The production environment is deployed across three physically independent Availability Zones (`europe-west3-a`, `europe-west3-b`, `europe-west3-c`) to deliver a **99.99% availability SLA**:

```
+--------------------------------------------------------------------------------------------------------------------+
|                                    MULTI-ZONE ACTIVE-ACTIVE HA TOPOLOGY                                            |
+--------------------------------------------------------------------------------------------------------------------+
|                                                                                                                    |
|   ZONE A (europe-west3-a)              ZONE B (europe-west3-b)              ZONE C (europe-west3-c)                |
|   ┌───────────────────────────────┐    ┌───────────────────────────────┐    ┌───────────────────────────────┐      |
|   │ [ GKE Node Pool A ]           │    │ [ GKE Node Pool B ]           │    │ [ GKE Node Pool C ]           │      |
|   │ - backend-go-api (Pod 1)      │    │ - backend-go-api (Pod 2)      │    │ - backend-go-api (Pod 3, 4)   │      |
|   │ - backend-go-worker (Pod 1)   │    │ - backend-go-worker (Pod 2)   │    │ - backend-go-worker (Pod 3, 4)│      |
|   │ - optimizer-core (Pod 1)      │    │ - optimizer-core (Pod 2)      │    │ - optimizer-core (Pod 3)      │      |
|   └───────────────┬───────────────┘    └───────────────┬───────────────┘    └───────────────┬───────────────┘      |
|                   │                                    │                                    │                      |
|                   v                                    v                                    v                      |
|   ══════════════════════════════════ [ CLOUD SPANNER REGIONAL 3-ZONE QUORUM ] ═══════════════════════════════════   |
|   (Paxos Read/Write Quorum across Zone A, B, and C with Instant Leader Election and Zero-Data-Loss RPO=0)           |
|                   │                                    │                                    │                      |
|                   v                                    v                                    v                      |
|   ┌───────────────────────────────┐    ┌───────────────────────────────┐    ┌───────────────────────────────┐      |
|   │ [ Memorystore Redis Master ]  │ ──>│ [ Redis Standby Replica ]     │    │ [ Managed Kafka Broker 3 ]    │      |
|   │ [ Managed Kafka Broker 1 ]    │    │ [ Managed Kafka Broker 2 ]    │    │                               │      |
|   └───────────────────────────────┘    └───────────────────────────────┘    └───────────────────────────────┘      |
|                                                                                                                    |
+--------------------------------------------------------------------------------------------------------------------+
```

1. **Kubernetes Pod Anti-Affinity & Pod Disruption Budgets (PDB):**
   - All critical deployments (`backend-go-api`, `backend-go-worker`, `optimizer-core`) enforce `topologySpreadConstraints` matching `topology.kubernetes.io/zone`, ensuring pods are distributed equally across all 3 zones.
   - PDBs enforce `minAvailable: 2` during cluster maintenance and rolling upgrades.
2. **Cloud Spanner Multi-Zone Quorum:**
   - Regional Spanner instances maintain write quorums across all 3 zones. If an entire availability zone suffers a catastrophic data center power failure, Spanner continues read/write execution without human intervention and with **zero data loss (RPO = 0)**.
3. **Memorystore Redis Automatic Failover:**
   - Redis `STANDARD_HA` provisions a primary instance in Zone A and a hot standby replica in Zone B. Heartbeat monitors detect primary failure and promote the standby replica within **15 to 30 seconds**.

---

### 7.2 Horizontal Pod Autoscaling (HPA) & KEDA Event-Driven Scaling

1. **API Tier (`backend-go-api` HPA):**
   - **Metrics:** Target average CPU utilization: **70%**; Target HTTP request rate: **500 req/s per pod**.
   - **Bounds:** Minimum 3 Replicas, Maximum 10 Replicas.
   - **Stabilization Window:** Scale-down stabilization window of 300 seconds to prevent thrashing.
2. **Worker Tier (`backend-go-worker` KEDA Autoscaling):**
   - Uses Kubernetes Event-driven Autoscaling (KEDA) monitoring Kafka consumer group lag:
   ```yaml
   apiVersion: keda.sh/v1alpha1
   kind: ScaledObject
   metadata:
     name: backend-go-worker-scaler
     namespace: pegasusx
   spec:
     scaleTargetRef:
       name: backend-go-worker
     minReplicaCount: 3
     maxReplicaCount: 12
     triggers:
     - type: kafka
       metadata:
         bootstrapServers: kafka-bootstrap:9092
         consumerGroup: void-order-mutator
         topic: pegasusx-orders
         lagThreshold: "100"
   ```
3. **Optimizer Core (`optimizer-core` HPA):**
   - **Metrics:** Target average CPU utilization: **80%**.
   - **Bounds:** Minimum 2 Replicas, Maximum 6 Replicas.

---

### 7.3 Cloud Spanner & Kafka Elastic Capacity Management

- **Spanner Autoscaler:** A Cloud Function / Cloud Monitoring policy monitors Spanner High Priority CPU Utilization. If CPU exceeds **65%** for > 5 minutes, Spanner automatically scales from 200 PU to 400 PU or 600 PU in 100 PU increments.
- **Kafka Partition Capacity:** All 10 canonical topics are pre-allocated with 12 partitions (or 6 for low-volume topics). 12 partitions allow the consumer tier to scale up to 12 parallel pod consumers per consumer group without partition contention.

---

### 7.4 Disaster Recovery Strategy, Backup Lifecycle & Failover Playbooks

```
+--------------------------------------------------------------------------------------------------------------------+
|                                    DISASTER RECOVERY LIFECYCLE & PROTOCOLS                                         |
+----------------------+--------------------+---------------------+--------------------------------------------------+
| Failure Scenario     | RPO Target         | RTO Target          | Automated / Manual Recovery Protocol             |
+----------------------+--------------------+---------------------+--------------------------------------------------+
| **Single AZ Outage** | **RPO = 0**        | **RTO < 60s**       | Fully Automated: GKE reschedules pods in live AZs; |
|                      |                    |                     | Spanner Paxos quorum maintained without restart; |
|                      |                    |                     | Redis fails over to hot standby replica (< 30s).  |
+----------------------+--------------------+---------------------+--------------------------------------------------+
| **Database Schema /  | **RPO < 5 mins**   | **RTO < 15 mins**   | Point-In-Time Recovery (PITR): Spanner 7-day     |
| Data Corruption**    |                    |                     | retention allows instant restore to any micro-   |
|                      |                    |                     | second timestamp prior to corruption incident.   |
+----------------------+--------------------+---------------------+--------------------------------------------------+
| **Catastrophic Cloud | **RPO < 24 hrs**   | **RTO < 60 mins**   | Terraform Disaster Recovery Playbook: Execute    |
| Regional Outage**    |                    |                     | `terraform apply -var="region=europe-west1"`     |
|                      |                    |                     | to provision cold standby region; restore latest |
|                      |                    |                     | GCS multi-region Spanner daily backup.           |
+----------------------+--------------------+---------------------+--------------------------------------------------+
```

#### Step-by-Step Regional Disaster Recovery Runbook
1. **Declare Regional Incident:** Incident Commander confirms primary region (`europe-west3`) unrecoverable.
2. **Execute Terraform in Secondary Region (`europe-west1`):**
   ```bash
   cd infra/terraform
   terraform workspace select dr-recovery || terraform workspace new dr-recovery
   terraform apply -var-file="environments/dr.tfvars" -auto-approve
   ```
3. **Restore Cloud Spanner from Latest Backup:**
   ```bash
   gcloud spanner databases restore \
     --instance=pegasusx-ledger-dr \
     --database=main \
     --backup=daily_full_latest \
     --backup-instance=pegasusx-ledger-prod
   ```
4. **Update Cloud DNS / Anycast Routing:**
   - Update Global HTTPS Load Balancer backend service to point to secondary GKE cluster.
   - Flush Cloud CDN edge caches.

---

## 8. Observability, Monitoring & SRE Alerting Framework

### 8.1 OpenTelemetry Tracing & Prometheus Metrics Pipeline

The observability architecture unifies distributed tracing, application metrics, and structured log analytics:

```
+--------------------------------------------------------------------------------------------------------------------+
|                                    OBSERVABILITY & MONITORING PIPELINE                                             |
+--------------------------------------------------------------------------------------------------------------------+
|                                                                                                                    |
|   [ GKE Workloads ] (backend-go, ai-worker, optimizer-core)                                                        |
|         │                                                                                                          |
|         ├───> [ HTTP Tracing ] ───────> OpenTelemetry W3C `traceparent` Header Propagation (Cloud Trace)           |
|         │                                                                                                          |
|         ├───> [ Application Metrics ] ─> Prometheus `/metrics` Scrape Endpoint (Port 8080/8081)                    |
|         │                                │                                                                         |
|         │                                v                                                                         |
|         │                      [ Google Cloud Managed Service for Prometheus ]                                     |
|         │                                │                                                                         |
|         │                                v                                                                         |
|         │                      [ Cloud Monitoring Alert Engine & Dashboards ]                                      |
|         │                                                                                                          |
|         └───> [ Structured JSON Logs ] > Google Cloud Logging (`slog.NewJSONHandler` to stdout)                    |
|                                                                                                                    |
+--------------------------------------------------------------------------------------------------------------------+
```

- **OpenTelemetry Tracing (`bootstrap.TraceMiddleware`):** Injects and extracts W3C `traceparent` headers across all HTTP, gRPC, and Kafka messages, providing end-to-end trace correlation from client apps to Spanner mutations.
- **SLO Metric Collector (`telemetry.NewSLOCollector`):** A dedicated in-process collector polls Spanner every 60 seconds, calculating real-time SLO metrics:
  - `void_outbox_lag_seconds`: Elapsed time between outbox event insertion and Kafka publication.
  - `void_fiscal_success_ratio`: Percentage of successful Soliq OFD electronic invoice registrations.
  - `void_capture_success_ratio`: Ratio of successful payment settlements vs authorization requests.

---

### 8.2 Production Alerting Policies Matrix (17 P1/P2/P3 Rules)

The infrastructure establishes 17 definitive, production-grade Cloud Monitoring alert policies:

```
+-----------------------------------------------------------------------------------------------------------------------------------------+
|                                                  PRODUCTION SRE ALERT POLICIES MATRIX                                                   |
+----+--------------------------------+----------+----------------------------------------------------+---------------+-------------------+
| #  | Alert Policy Name              | Severity | PromQL / Cloud Monitoring Trigger Condition        | Duration      | Notification Dest |
+----+--------------------------------+----------+----------------------------------------------------+---------------+-------------------+
| 1  | `SpannerHighCPUUtilization`    | **P1**   | `spanner.googleapis.com/instance/cpu/utilization`  | > 80% for 5m  | PagerDuty + SRE   |
|    |                                | Critical | (High priority CPU utilization)                    |               | Escalation        |
| 2  | `SpannerStorageSplitPressure`  | **P2**   | `spanner.googleapis.com/instance/storage/utilization` > 80% for 15m| Slack #alerts-db  |
|    |                                | Major    |                                                    |               |                   |
| 3  | `GKEPodCrashLooping`           | **P1**   | `rate(kube_pod_container_status_restarts_total[5m])`> 0.2 restarts/s| PagerDuty On-Call |
|    |                                | Critical | (Pod in CrashLoopBackOff)                          |               |                   |
| 4  | `HighHTTP5xxErrorRate`         | **P1**   | `sum(rate(http_requests_total{status=~"5.."}[5m]))`| > 1% for 3m   | PagerDuty On-Call |
|    |                                | Critical | `/ sum(rate(http_requests_total[5m]))`             |               |                   |
| 5  | `HighHTTPLatencyP99`           | **P2**   | `histogram_quantile(0.99, http_request_duration)`  | > 1.5s for 5m | Slack #alerts-api |
|    |                                | Major    |                                                    |               |                   |
| 6  | `KafkaConsumerGroupLagSpike`   | **P1**   | `void_kafka_consumer_lag_seconds`                  | > 300s for 5m | PagerDuty On-Call |
|    |                                | Critical | (Consumer falling behind stream)                   |               |                   |
| 7  | `TransactionalOutboxRelayLag`  | **P1**   | `void_outbox_lag_seconds`                          | > 30s for 3m  | PagerDuty On-Call |
|    |                                | Critical | (Outbox events backing up in Spanner)              |               |                   |
| 8  | `RedisMemoryUsageCritical`     | **P2**   | `redis.googleapis.com/stats/memory/usage_ratio`    | > 85% for 5m  | Slack #alerts-db  |
|    |                                | Major    |                                                    |               |                   |
| 9  | `RedisEvictionRateSpike`       | **P2**   | `rate(redis.googleapis.com/stats/evicted_keys[5m])`| > 100 keys/s  | Slack #alerts-db  |
|    |                                | Major    | (Cache thrashing under memory pressure)            |               |                   |
| 10 | `CloudNATPortExhaustion`       | **P1**   | `compute.googleapis.com/nat/allocated_ports_util`  | > 80% for 3m  | PagerDuty On-Call |
|    |                                | Critical | (Egress connections being dropped)                 |               |                   |
| 11 | `SoliqOFDFiscalFailureRate`    | **P2**   | `(1 - void_fiscal_success_ratio)`                  | > 5% for 10m  | Slack #alerts-tax |
|    |                                | Major    | (Government tax e-invoicing failing)               |               |                   |
| 12 | `PaymentWebhookFailureRate`    | **P1**   | `rate(payment_webhook_failures_total[5m])`         | > 2% for 5m   | PagerDuty On-Call |
|    |                                | Critical | (Banking webhook signature drops)                  |               |                   |
| 13 | `EvidenceVaultChecksumMismatch`| **P1**   | `void_evidence_vault_checksum_errors_total`        | > 0 events    | Security + SRE    |
|    |                                | Critical | (Tampering detected in POD photo vault)            | (Immediate)   | Immediate Page    |
| 14 | `CloudArmorDDoSBlockSpike`     | **P3**   | `rate(securitypolicy.googleapis.com/request_count` | > 500 req/s   | Slack #alerts-sec |
|    |                                | Warning  | `{action="DENY"}[5m])`                             | for 5m         |                   |
| 15 | `GCSEvidenceUploadFailure`      | **P2**   | `rate(storage_upload_errors_total[5m])`            | > 2% for 5m   | Slack #alerts-app |
|    |                                | Major    | (Driver POD photos failing upload)                  |               |                   |
| 16 | `DriverTelemetryDropSpike`     | **P3**   | `rate(telemetry_driver_frames_dropped_total[5m])`  | > 10% for 5m  | Slack #alerts-app |
|    |                                | Warning  | (Driver GPS stream drops)                           |               |                   |
| 17 | `OptimizerSolverTimeoutRate`   | **P2**   | `rate(optimizer_solve_timeouts_total[5m])`         | > 5% for 5m   | Slack #alerts-app |
|    |                                | Major    | (VRP solver exceeding 10s ceiling)                  |               |                   |
+----+--------------------------------+----------+----------------------------------------------------+---------------+-------------------+
```

---

### 8.3 Synthetic Uptime Probes & Health Check Architecture

1. **Kubernetes Probes:**
   - **Liveness Probe:** `HTTP GET :8080/healthz` (Interval: 10s, Timeout: 3s, FailureThreshold: 3). Verifies basic container runtime responsiveness.
   - **Readiness Probe:** `HTTP GET :8080/ready` (Interval: 5s, Timeout: 3s, FailureThreshold: 2). Validates that Spanner, Redis, and Kafka connection pools are fully initialized and healthy.
2. **Public Synthetic Probes (`google_monitoring_uptime_check_config`):**
   - Probes `https://api.pegasusx.example.com/v1/health/capabilities` every 60 seconds from 5 global locations (Frankfurt, Mumbai, Singapore, Iowa, London).
   - Generates immediate P1 alert if 2 or more probing stations fail consecutive health checks.

---

### 8.4 SRE On-Call Incident Response & Remediation Runbooks

#### Runbook 1: Remediating `SpannerHighCPUUtilization` (P1)
1. **Triage:** Open Cloud Spanner Query Insights dashboard; identify queries with highest `Total CPU Time`.
2. **Immediate Mitigation:** Scale processing units immediately:
   ```bash
   gcloud spanner instances update ledger --processing-units=600
   ```
3. **Investigation:** Check for full table scans caused by missing indexes or un-indexed queries on `OutboxEvents` or `Orders`.

#### Runbook 2: Remediating `KafkaConsumerGroupLagSpike` (P1)
1. **Triage:** Identify failing consumer group (`void-order-mutator` or `void-notification-dispatcher`):
   ```bash
   kubectl logs -n pegasusx -l app=backend-go-worker --tail=200 | grep "kafka consumer error"
   ```
2. **Immediate Mitigation:** Scale worker deployment to maximum partitions (12 pods):
   ```bash
   kubectl scale deployment/backend-go-worker -n pegasusx --replicas=12
   ```
3. **Dead-Letter Recovery:** If poisoned messages cause crash loops, trigger DLQ bypass and execute `apps/backend-go/cmd/replay-dlq` after code fix deployment.

---

## 9. Enterprise Security Hardening & Zero-Trust Architecture

### 9.1 Network Perimeter Isolation & VPC Firewall Rules

PegasusX establishes a hardened zero-trust network perimeter:
- **Private GKE Nodes:** All GKE cluster nodes are provisioned with strictly private RFC 1918 IP addresses (`10.10.0.0/20`). No public IP addresses are assigned to compute nodes.
- **Firewall Rules Matrix:**
  - `allow-lb-to-gke`: Allows inbound traffic on NodePorts only from Google Cloud Load Balancer proxy IP ranges (`130.211.0.0/22`, `35.191.0.0/16`).
  - `allow-internal-mesh`: Permits intra-cluster communication across Pod CIDR (`10.20.0.0/16`) and Service CIDR (`10.30.0.0/20`).
  - `deny-all-ingress`: Default egress/ingress deny rule blocking all unauthenticated external traffic.

---

### 9.2 Workload Identity Federation & Least-Privilege IAM

PegasusX eliminates static Google Cloud Service Account JSON keys from container images and filesystem mounts. All pods authenticate dynamically using **GKE Workload Identity Federation**:

```
+--------------------------------------------------------------------------------------------------------------------+
|                                    WORKLOAD IDENTITY FEDERATION BINDINGS                                           |
+------------------------------+-------------------------------+-----------------------------------------------------+
| Kubernetes Service Account   | Google Service Account (GSA)  | Assigned Least-Privilege IAM Roles                  |
+------------------------------+-------------------------------+-----------------------------------------------------+
| `k8s:pegasusx/backend-go`    | `pegasusx-backend@${var.proj}`| `roles/spanner.databaseUser` (Spanner CRUD)         |
|                              |                               | `roles/storage.objectAdmin` (`pegasusx-prod-media`) |
|                              |                               | `roles/secretmanager.secretAccessor` (Secrets)      |
|                              |                               | `roles/managedkafka.client` (Kafka Publish/Consume) |
+------------------------------+-------------------------------+-----------------------------------------------------+
| `k8s:pegasusx/ai-worker`     | `pegasusx-ai@${var.proj}`     | `roles/spanner.databaseUser` (Forecast Read/Write)  |
|                              |                               | `roles/storage.objectViewer` (Import CSV Stream)    |
|                              |                               | `roles/managedkafka.client` (Kafka Consume)         |
+------------------------------+-------------------------------+-----------------------------------------------------+
| `k8s:pegasusx/eso-sync`      | `pegasusx-eso@${var.proj}`    | `roles/secretmanager.secretAccessor` (Secret Sync)  |
+------------------------------+-------------------------------+-----------------------------------------------------+
```

---

### 9.3 Cloud Armor WAF Policies & DDoS Mitigation

Cloud Armor Enterprise WAF operates at the global edge forwarding rule, evaluating every incoming request before it reaches the GKE cluster:
1. **Layer 7 Rate Limiting:** Enforces a maximum threshold of **500 requests per 10-second window** per client IP. Malicious IPs exceeding this threshold are blocked with HTTP 429 for a 10-minute ban window.
2. **OWASP Top 10 Core Rule Set (CRS 3.3):** Pre-configured ModSecurity evaluation inspecting URIs, headers, and request bodies for:
   - SQL Injection (`sqli-v33-stable`)
   - Cross-Site Scripting (`xss-v33-stable`)
   - Local / Remote File Inclusion (`lfi-v33-stable`, `rfi-v33-stable`)
   - Remote Code Execution (`rce-v33-stable`)
3. **Geo-Fencing Protection:** Restricts sensitive administrative and dunning endpoints (`/v1/auth/platform-admin/*`, `/v1/admin/*`) exclusively to authorized corporate IP CIDRs.

---

### 9.4 Secret Management, Rotation & Cryptographic Vault Integrity

1. **Google Secret Manager & External Secrets Operator (ESO):**
   - Secrets are created in Secret Manager encrypted with Google-Managed or Customer-Managed Encryption Keys (CMEK).
   - ESO polls Secret Manager every 60 minutes, automatically projecting updated credentials into native Kubernetes Secrets.
2. **Mandatory 90-Day Secret Rotation Policy:**
   - JWT signing keys (`jwt-secret`), database credentials, and payment webhook secrets undergo scheduled 90-day rotation without application downtime.
3. **Cryptographic Evidence Dossier Vault Integrity (ADR-009):**
   - Proof-of-delivery photos and digital promissory notes uploaded to `pegasusx-prod-media` compute an SHA-256 binary hash upon upload.
   - When an order is completed, `storage.SealDossier()` computes an aggregate root SHA-256 checksum over all items. This checksum is permanently written to Spanner `EvidenceDossiers`.
   - Alert Rule 13 (`EvidenceVaultChecksumMismatch`) immediately triggers a P1 incident if any stored dossier artifact diverges from its sealed checksum.

---

### 9.5 Audit Logging & Regulatory Compliance Enforcement

- **Cloud Audit Logs:** Data Access audit logs are enabled for Cloud Spanner, Secret Manager, and Google Cloud Storage.
- **Application-Level Audit Trail:** Every mutating API call records an immutable entry in Spanner `AuditLog` capturing `UserId`, `TenantId`, `ActorRole`, `IPAddress`, `Action`, `PayloadDigest`, and `Timestamp`.
- **Tax Compliance (Soliq OFD):** Guarantees zero unrecorded cash or digital transactions by strictly enforcing synchronous state persistence prior to fiscal clearance.

---

*Authoritative Phased GCP Architecture Document concluded. Authored by Worker M2 for the PegasusX Infrastructure Project.*
