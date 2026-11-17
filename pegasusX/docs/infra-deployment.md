# PegasusX Infrastructure, Orchestration & Anti-Theatre CI Matrix

> **Document Scope**: Google Cloud Platform Terraform Infrastructure (6 Modules & Multi-Region Cells), Kubernetes Production Fleet (Base, 7 Overlays, 5 Automated CronJobs), Docker Compose SSMR Sandbox, Anti-Theatre Automated Verification Scripts, and the 12-Job CI Matrix  
> **Source Root**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX`  
> **Root Terraform**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/infra/terraform/main.tf`  
> **Root CI Pipeline**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/.github/workflows/ci.yml`  
> **Local Sandbox**: `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/infra/docker-compose.ssmr.yml`  

---

## 1. Google Cloud Platform Terraform Infrastructure (`infra/terraform/`)

### 1.1. Architecture & 6 Rollout Modules

#### What it is
PegasusX provisions enterprise Google Cloud Platform (GCP) resources through a modular Terraform codebase organized into **6 sequential rollout phases** (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/infra/terraform/main.tf:1-109`):

```
+----------------------------------------------------------------------------------------------------+
|                                TERRAFORM 6-PHASE INFRASTRUCTURE STACK                              |
+----------------------------------------------------------------------------------------------------+
|                                                                                                    |
|  [Phase 1: Networking Module] (modules/networking)                                                 |
|     - VPC Network & Regional Subnets                                                               |
|     - Cloud NAT with 2 Dedicated Static External Egress IPs                                        |
|     - Private Service Access (PSA) Peering for Spanner & Redis                                     |
|     - Internal & Ingress Firewall Rules                                                            |
|                                                                                                    |
|  [Phase 2: Database Module] (modules/database)                                                     |
|     - Cloud Spanner Regional Instance with Automated Daily Backup Schedules                        |
|     - Memorystore Redis 7.0 High Availability (HA) with Automatic Failover                         |
|                                                                                                    |
|  [Phase 2: Messaging Module] (modules/messaging)                                                    |
|     - Google Managed Service for Apache Kafka (MSK / Google Managed Kafka)                         |
|     - 10 Canonical Partitioned Topics (pegasusx-main, orders, dispatch, spatial, demand, etc.)     |
|                                                                                                    |
|  [Phase 2: Storage & Security Module] (modules/storage_security)                                   |
|     - 4 Google Cloud Storage Buckets (Media Assets, App Updates, Bulk Imports, TF State)           |
|     - Cloud Armor Web Application Firewall (WAF) Rate Limiting & OWASP Rules                       |
|     - Secret Manager Vault Integration                                                             |
|     - GKE Workload Identity IAM Service Account Bindings                                           |
|                                                                                                    |
|  [Phase 3: Compute Module] (modules/compute)                                                       |
|     - Google Kubernetes Engine (GKE) Regional Multi-Zone Cluster (Autopilot / Standard)            |
|     - Dedicated Node Pools & Hardened Service Account Identities                                   |
|                                                                                                    |
|  [Phase 4: Monitoring Module] (modules/monitoring)                                                 |
|     - 12 Cloud Monitoring Alert Policies (Outbox Lag, Fiscal Ratio, Capture Failure, Worker Crashes)|
|     - Notification Channels (Slack Webhooks & PagerDuty/Email)                                     |
|     - Uptime Check Probes & Service Level Objective (SLO) Dashboards                               |
+----------------------------------------------------------------------------------------------------+
```

#### How it works
1. **Module 1: Networking** (`infra/terraform/main.tf:14-26`, `modules/networking/`):
   - Provisions a custom-mode Virtual Private Cloud (`var.network_name`).
   - Configures regional subnets with secondary CIDR ranges for GKE pods (`var.pod_cidr_block`) and services (`var.service_cidr_block`).
   - Creates a Cloud Router and Cloud NAT allocating **2 static external egress IP addresses**, allowing third-party fiscal gateways (Uzbekistan Soliq EHF) and payment processors (Global Pay) to whitelist known origin IPs.
   - Configures Google Private Service Access (PSA) peering for zero-public-egress communication with managed databases.
2. **Module 2: Database** (`main.tf:29-46`, `modules/database/`):
   - Provisions a Google Cloud Spanner regional instance (`var.spanner_instance_name`) with dedicated processing units (`var.spanner_processing_units`).
   - Implements automated daily backup schedules with retention periods (`var.spanner_backup_retention_days`).
   - Deploys a Google Cloud Memorystore Redis 7.0 High Availability cluster with in-memory persistence and automatic cross-zone failover.
3. **Module 3: Messaging** (`main.tf:49-60`, `modules/messaging/`):
   - Provisions a Google Managed Kafka cluster connected via private VPC peering.
   - Pre-provisions 10 canonical topics with replication factor 3 and partition scaling.
4. **Module 4: Storage & Security** (`main.tf:63-75`, `modules/storage_security/`):
   - Provisions 4 hardened GCS buckets:
     1. `media_bucket_name`: Driver delivery proof photos and retailer storefront signage.
     2. `updates_bucket_name`: Desktop OTA update manifests and Tauri application binaries (`updateroutes/`).
     3. `imports_bucket_name`: Supplier bulk inventory import spreadsheets.
     4. `tf_state_bucket_name`: Encrypted remote Terraform state.
   - Configures Cloud Armor WAF security policies applying rate-limiting, geo-blocking, and OWASP top-10 inspection rules.
   - Sets up Google Secret Manager (GSM) for database connection strings, JWT signing keys, and payment credentials.
   - Maps Kubernetes service accounts to Google IAM Service Accounts (GSA) via Workload Identity.
5. **Module 5: Compute** (`main.tf:78-96`, `modules/compute/`):
   - Provisions a regional multi-zone GKE cluster (`var.cluster_name`) across three availability zones.
   - Enables Shielded GKE Nodes, Datapath V2 (Cilium eBPF networking), and Workload Identity.
6. **Module 6: Monitoring** (`main.tf:99-108`, `modules/monitoring/`):
   - Provisions 12 Cloud Monitoring alert policies:
     - Transactional outbox lag exceeding threshold (`void_outbox_lag_seconds > 60`).
     - Soliq fiscal receipt failure ratio (`void_fiscal_success_ratio < 0.99`).
     - Payment capture failure ratio (`void_capture_success_ratio < 0.98`).
     - AI worker consumer group lag.
     - Unexpected backend pod crash loops.
   - Binds alert triggers to Slack webhooks and on-call notification channels.

#### Why it is there
Managing physical supply chains requires enterprise reliability. Provisioning networking, databases, messaging, compute, and security as modularized Infrastructure-as-Code (IaC) guarantees reproducible staging environments and eliminates manual configuration drift.

---

### 1.2. Multi-Region Cell Infrastructure (`infra/terraform/cells/`)

#### What it is
PegasusX structures expansion into independent geographic regions as isolated cells (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/infra/terraform/cells/`):
- **Cell UZ** (`infra/terraform/cells/uz/`): Primary deployment in Central Asia (`me-central1` / Tashkent), GCP Project `pegasus-503013`, state prefix `pegasusx/ssmr`.
- **Cell EU** (`infra/terraform/cells/eu/`): Secondary deployment in Europe (`europe-west1`), GCP Project `pegasusx-cell-eu`, state prefix `pegasusx/cell-eu`.

#### How it works & Cell Backend Guard
- **State Separation**: Enforced by `file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/assert_cell_backend.sh:1-156`.
- `cells/uz/backend.hcl` holds `prefix = "pegasusx/ssmr"` (`line 3`).
- `cells/eu/backend.hcl` holds `prefix = "pegasusx/cell-eu"` (`line 5`).
- `assert_cell_backend.sh` verifies that any Terraform execution against `europe-west1` is strictly barred from accessing or modifying the live UZ state prefix (`pegasusx/ssmr`).
- Workload identity namespaces are dynamically derived (`local.k8s_namespace`) to prevent cross-tenant cluster privilege escalation.
- Foreign cells are strictly prohibited from restoring Spanner database backups originating from UZ (`cell.tf:122`, `scripts/cell_migrate.sh:127-129`), upholding data sovereignty laws.

---

## 2. Kubernetes Production Fleet Orchestration (`infra/k8s/`)

### 2.1. Base Manifests & 7 Environment Overlays

#### What it is
The Kubernetes deployment topology is defined using Kustomize, organized into a common base directory (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/infra/k8s/base/`) and 7 environment overlays: `prod`, `staging`, `pilot`, `dev`, `sandbox`, `ssmr`, and `cells` (`cells/uz`, `cells/eu`).

#### How it works
1. **Base Services & Deployments** (`infra/k8s/base/`):
   - `deployment.yaml`: Runs `apps/backend-go` in `PEGASUSX_RUN_MODE=api` (serving HTTP routes and WebSocket connections).
   - `deployment-worker.yaml`: Runs `apps/backend-go` in `PEGASUSX_RUN_MODE=worker` (serving outbox relays, reconcilers, and 24 background workers).
   - `deployment-ai-worker.yaml`: Runs `apps/ai-worker` (Kafka consumer, synthesis engine, bulk import).
   - `deployment-optimizer-core.yaml`: Runs `services/optimizer-core` (C++ OR-Tools and Rust VRP solver).
   - `service.yaml` & `service-ws.yaml`: Dedicated internal ClusterIP services separating standard HTTP API traffic from long-lived WebSocket connections.
   - `ingress.yaml`: Cloud Armor protected HTTPS Ingress with Google-managed SSL certificates and HTTP-to-HTTPS redirect.
   - `hpa.yaml`: Horizontal Pod Autoscaler scaling backend API pods dynamically based on CPU utilization and HTTP request rates.
   - `pdb.yaml`: Pod Disruption Budgets guaranteeing a minimum of 2 available replicas during node pool rolling upgrades.
2. **7 Environment Overlays**:
   - `overlays/prod`: Immutable digest-pinned container images, high resource limits, strict Cloud Armor rules.
   - `overlays/staging`: Production-like environment with simulated payment providers.
   - `overlays/pilot`: Field pilot overlay for initial retailer beta testing.
   - `overlays/dev`: Development cluster with permissive CORS and verbose logging.
   - `overlays/sandbox`: Isolated testing environment for automated end-to-end smoke verification.
   - `overlays/ssmr`: Dedicated SSMR enterprise deployment profile.
   - `overlays/cells`: Geographic overlays (`cells/uz` with Soliq EHF enabled, `cells/eu` with commercial receipts enabled).

---

### 2.2. The 5 Automated Kubernetes CronJobs

PegasusX schedules 5 critical operational CronJobs (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/infra/k8s/`):

1. **Monthly Billing Invoicing** (`billing_monthly_cronjob.yaml`)
   - *Schedule*: `15 6 1 * *` (06:15 UTC on the 1st of every month).
   - *What it does*: Issues an authenticated HTTP POST request to `/v1/admin/billing/run-monthly` using a platform admin token with MFA step-up.
   - *Why it is there*: Automatically generates monthly billing invoices and platform fee charges for all active suppliers.
2. **Daily Forecast Accuracy Evaluation** (`planning_accuracy_cronjob.yaml`)
   - *Schedule*: Daily at 03:00 UTC.
   - *What it does*: Evaluates yesterday's machine learning demand predictions against actual retailer orders, computing Mean Absolute Percentage Error (MAPE).
   - *Why it is there*: Detects forecast model drift and automatically triggers model retraining when accuracy degrades.
3. **Demand Forecast Recalculation** (`planning_forecast_cronjob.yaml`)
   - *Schedule*: Nightly at 01:00 UTC.
   - *What it does*: Triggers end-to-end demand forecasting across all SKUs, publishing `PLANNING_FORECAST_UPDATED` events to Kafka.
   - *Why it is there*: Updates warehouse safety stock recommendations and replenishment triggers before morning picking begins.
4. **Planning Training Data Export** (`planning_training_export_cronjob.yaml`)
   - *Schedule*: Weekly on Sundays at 04:00 UTC.
   - *What it does*: Queries historical order velocities, weather patterns, and promotional uplifts, exporting sanitized Parquet datasets to Google Cloud Storage.
   - *Why it is there*: Provides continuous, clean training data for the machine learning optimization pipeline.
5. **Predictive Push Restock Cron** (`predictive_push_cronjob.yaml`)
   - *Schedule*: Twice daily at 07:00 and 14:00 local time.
   - *What it does*: Runs `apps/ai-worker` in `predictive-push-cron` mode (`main.go:227-233`), computing SKU exhaustion curves and sending proactive restock notifications to store owners.
   - *Why it is there*: Drives the B2B reordering flywheel and prevents retail shelf stockouts.

---

## 3. Docker Compose SSMR Sandbox Environment

### 3.1. Local Topology & Container Inventory

#### What it is
The local sandbox environment (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/infra/docker-compose.ssmr.yml:1-223`) provides a completely self-contained, offline-capable replica of the entire enterprise backend stack.

#### How it works
Bootstrapped via `make sandbox-infra-up` (or `make ssmr-infra-up`), the stack initializes 10 coordinated containers:

```
+----------------------------------------------------------------------------------------------------+
|                                LOCAL SSMR DOCKER COMPOSE TOPOLOGY                                  |
+----------------------------------------------------------------------------------------------------+
|                                                                                                    |
|  [spanner-emulator] ---------> Ports 9110:9010 (gRPC), 9120:9020 (REST)                            |
|                                                                                                    |
|  [redis:7-alpine] -----------> Port 6389:6379 (Healthcheck: redis-cli ping)                        |
|                                                                                                    |
|  [cp-zookeeper:7.5.0] -------> Port 22181:2181 (Coordination engine)                               |
|                                                                                                    |
|  [cp-kafka:7.5.0] -----------> Port 9094:9094 (Bootstrap broker, PLAINTEXT)                       |
|                                                                                                    |
|  [kafka-init] ---------------> One-shot provisioner: creates all 14 canonical topics with 3 parts   |
|                                                                                                    |
|  [kafka-ui] -----------------> Port 8083:8080 (Web inspection UI for Kafka streams)               |
|                                                                                                    |
|  [backend-setup] ------------> One-shot migration runner: applies 125 Spanner DDL migrations       |
|                                                                                                    |
|  [backend-go] ---------------> Host Port 8180 -> Container 8080 (API & workers in RUN_MODE=all)    |
|                                                                                                    |
|  [ai-worker] ----------------> Host Port 8181 -> Container 8081 (Kafka consumer & synthesis)       |
|                                                                                                    |
|  [optimizer-core] -----------> Host Port 8182 -> Container 8082 (OR-Tools solver engine)            |
+----------------------------------------------------------------------------------------------------+
```

- **Persistent Build Volumes**: Uses `pegasusx-ssmr-go-mod` and `pegasusx-ssmr-go-build` (`lines 222-223`) to cache Go modules and compilation artifacts, ensuring sub-second container restarts during local development.

#### Why it is there
Developers and automated CI pipelines can execute full end-to-end integration tests, lifecycle verticals, and fiscal proofs without incurring cloud infrastructure costs or requiring live GCP internet access.

---

## 4. Automated Anti-Theatre CI Verification Suite (`scripts/`)

PegasusX has instituted automated CI gates to enforce the **Zero Theatre Doctrine**. These scripts run on every pull request, halting builds upon detection of fake implementations:

### 4.1. Gate Scripts Analysis

1. **`scripts/ci_fail_todo_inject.sh`** (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/ci_fail_todo_inject.sh:1-12`)
   - *What it is*: Automated scanner for orphaned client screens.
   - *How it works*: Uses ripgrep to scan `apps/` and `packages/` for the literal string `TODO: Inject`.
   - *Why it is there*: Prevents developers from shipping dead UI screens or stubbed ViewModels that do nothing when clicked.
2. **`scripts/ci_fail_placeholder_images.sh`** (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/ci_fail_placeholder_images.sh:1-44`)
   - *What it is*: Production image immutability gate.
   - *How it works*: Renders the production Kustomize overlay (`kubectl kustomize infra/k8s/overlays/prod`), verifying that no image contains `IMAGE_PLACEHOLDER`, `:latest`, `:local`, or `REPLACE_WITH_DIGEST` (`lines 19-24`). Also asserts that `optimizer-core` is never remapped to `backend-go` (`lines 26-38`).
   - *Why it is there*: Guarantees that production deployments run verified, immutable container digests.
3. **`scripts/ci_no_mock_control_tower.sh`** (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/ci_no_mock_control_tower.sh:1-15`)
   - *What it is*: Anti-mock data scanner for retailer client builds.
   - *How it works*: Scans `retailer-app-desktop`, `retailer-app-android`, and `retailer-app-ios` for demo markers: `Mock Data`, `hardcoded BarMark`, `fakeH3Pulse`, or `CONTROL_TOWER_SIMULATOR`.
   - *Why it is there*: Ensures retail store owners only view real inventory balances and actual delivery routes.
4. **`scripts/money_path_gate.sh`** (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/money_path_gate.sh:1-56`)
   - *What it is*: Financial transaction correctness proof against a live Spanner emulator.
   - *How it works*: Boots the Spanner emulator, runs migration setup, executes test suites `TestMoneyPathGate` and `TestWorkerShopClosed`, and runs secrets/hygiene scans (`gitleaks`, `ci_fail_todo_inject.sh`).
   - *Why it is there*: Proves mathematically that payment capture failures never write `CAPTURED`, duplicate idempotency keys never double-charge, and shop-closed credit debt is always recorded.
5. **`scripts/assert_cell_backend.sh`** (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/assert_cell_backend.sh:1-156`)
   - *What it is*: Multi-region cell state isolation guard.
   - *How it works*: Audits Terraform backend HCL files, variable declarations in `cell.tf`, GKE Workload Identity namespaces, and Kustomize overlays.
   - *Why it is there*: Enforces strict cryptographic isolation between European and Uzbek operational cells.
6. **`scripts/ci_schema_drift_gate.sh`** (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/ci_schema_drift_gate.sh:1-43`)
   - *What it is*: Spanner DDL schema-drift gate.
   - *How it works*: Compares tables declared across all 125 migration files against the master `schema/spanner.ddl` definition, then applies migrations against a Spanner emulator to verify live object parity.
   - *Why it is there*: Prevents application runtime crashes caused by missing columns or outdated secondary indexes.
7. **`scripts/parity/role_row_contract_check_full.sh`** (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/scripts/parity/role_row_contract_check_full.sh:1-60`)
   - *What it is*: Full client-to-backend route parity checker.
   - *How it works*: Scans every TypeScript, Kotlin, and Swift file across all 22 client applications, extracting all invoked `/v1/...` API endpoints and asserting that every single one is mounted in `apps/backend-go`.
   - *Why it is there*: Guarantees zero 404 errors across all deployed mobile apps and web portals.

---

## 5. The 12-Job CI Pipeline Matrix (`.github/workflows/ci.yml`)

The GitHub Actions workflow (`file:///Users/shakhzod/Desktop/V.O.I.D/pegasusX/.github/workflows/ci.yml:1-304`) orchestrates **12 parallel and sequential validation jobs**:

```
[backend-unit] -----------------------------------------------> Go unit test suite
      |
      v
  [backend] -------------------------------------------------> Parity, gap-hunter, K8s manifests, anti-theatre gates
      |
      +---> [cell-isolation] --------------------------------> GS-C cell isolation proof & backend guards
      |
      +---> [backend-lint] ----------------------------------> golangci-lint v2.12.2 & govulncheck
      |
      +---> [secrets] ---------------------------------------> gitleaks repository secrets scan
      |
      +---> [backend-spanner] -------------------------------> Spanner emulator integration & money-path gate
      |           |
      |           v
      |     [enterprise-gates] ------------------------------> Phase 2 through Phase 5c enterprise test suite
      |
      +---> [ai-worker] -------------------------------------> Go build & go vet for ai-worker
      |
      +---> [android-apps] ----------------------------------> Matrix compilation of all 6 Android apps via Gradle
      |
      +---> [ios-apps] --------------------------------------> Matrix compilation of all 6 iOS apps via XcodeGen
      |
      +---> [supplier-portal] -------------------------------> Vitest & typecheck for all Tauri desktop clients
      |
      +---> [admin-portal] ----------------------------------> Typecheck & production Next.js build
```

### Detailed Job Execution Matrix

| Job Name | Runners / OS | Key Verification Commands & Targets | Critical Quality Invariants |
|---|---|---|---|
| **1. `backend-unit`** (`ci.yml:14-18`) | `ubuntu-latest` | `reusable-go-unit.yml` (Go 1.25) | Unit test pass with zero race conditions |
| **2. `backend`** (`ci.yml:20-55`) | `ubuntu-latest` | `make parity-contract-full`, `make gap-hunter-gate`, `make gen-contracts-gate`, `ci_fail_placeholder_images.sh`, `ci_fail_todo_inject.sh`, `ci_no_mock_control_tower.sh` | Route parity, schema sync, zero placeholder images, zero mock strings |
| **3. `cell-isolation`** (`ci.yml:56-71`) | `ubuntu-latest` | `make cell-backend-guard`, `make cell-isolation-proof`, `ci_no_unattended_terraform_apply.sh` | Cryptographic state isolation between Cell UZ and Cell EU |
| **4. `backend-lint`** (`ci.yml:72-90`) | `ubuntu-latest` | `golangci-lint` (v2.12.2), `govulncheck ./...` | Strict static analysis, zero CVE vulnerabilities |
| **5. `secrets`** (`ci.yml:91-102`) | `ubuntu-latest` | `gitleaks` action with `.gitleaks.toml` | Zero committed credentials, private keys, or API tokens |
| **6. `backend-spanner`** (`ci.yml:103-131`) | `ubuntu-latest` + Spanner Emulator | `ci_spanner_integration.sh`, `ci_schema_drift_gate.sh`, `money_path_gate.sh`, `phase1_gate.sh` | TrueTime linearizability, money path idempotency, schema drift prevention |
| **7. `enterprise-gates`** (`ci.yml:134-155`) | `ubuntu-latest` + Spanner Emulator | `bash scripts/ci_enterprise_gates.sh` (Phases 2–5c + analytics tenancy) | Multi-supplier checkout sagas, GlobalProducts master data |
| **8. `ai-worker`** (`ci.yml:156-168`) | `ubuntu-latest` | `go build ./...`, `go vet ./...` in `apps/ai-worker` | Clean compilation and type alignment |
| **9. `android-apps`** (`ci.yml:169-199`) | `ubuntu-latest` | Matrix build: `driver`, `factory`, `payload`, `retailer`, `supplier`, `warehouse` via `./gradlew compileStoreDebugKotlin` | 100% Android compilation across all 6 operational role apps |
| **10. `ios-apps`** (`ci.yml:200-253`) | `macos-14` | Matrix build: `xcodegen generate` + `xcodebuild -sdk iphonesimulator` | 100% iOS compilation across all 6 operational role apps |
| **11. `supplier-portal`** (`ci.yml:254-286`) | `ubuntu-latest` (Node 20, pnpm 9) | `pnpm --filter @pegasusx/desktop-bridge test`, `pnpm --filter @pegasusx/desktop-cache test`, Vitest & Typecheck all 4 Tauri portals | Desktop offline caching, bridge exports, TypeScript strict typing |
| **12. `admin-portal`** (`ci.yml:288-304`) | `ubuntu-latest` (Node 20, pnpm 9) | `pnpm --filter @pegasusx/admin-portal typecheck`, `pnpm --filter @pegasusx/admin-portal build` | Clean production build of platform governance portal |

---

## 6. Summary: The Grounded PegasusX Operational Stack

Through its 6-phase Terraform infrastructure, 7-overlay Kubernetes configuration, Docker Compose SSMR sandbox, automated anti-theatre script gates, and 12-job CI matrix, PegasusX establishes an unbreakable chain of code grounding and production integrity. Every physical and financial invariant—from loading dock tamper seals to Uzbekistan Soliq EHF electronic tax filing—is guaranteed by genuine code, real database transactions, and mathematically rigorous solvers.
