# PegasusX — Enterprise B2B Supply Chain & FMCG Platform

PegasusX is a mission-critical, high-throughput B2B logistics, commerce, and warehouse orchestration platform engineered for enterprise FMCG operations. The platform guarantees strict multi-tenant isolation under the **Single-Supplier Multi-Retailer (SSMR)** architectural doctrine, executing real-time routing, transactional ledgers, fiscal compliance, and cold-chain safety.

---

## Architecture & System Topology

```
+-----------------------------------------------------------------------------------+
|                                  CLIENT CLIENTS                                   |
|   Web / Desktop Portals (Next.js + Tauri v2)    |    Mobile Native (Kotlin & Swift)  |
|   - Supplier Portal       - Admin Console       |    - Driver App    - Retailer App  |
|   - Warehouse Portal      - Factory Bay         |    - Warehouse App - Factory Bay   |
|   - Retailer Desktop App  - Payload Terminal    |    - Supplier App  - Payload App   |
+-----------------------------------------------------------------------------------+
                                         |
                                  gRPC / HTTPS / WSS
                                         v
+-----------------------------------------------------------------------------------+
|                                 BACKEND SERVICES                                  |
|                                                                                   |
|  [apps/backend-go]                                                                |
|  - High-concurrency Chi Router, Order State Machine, Outbox Relay                |
|  - Soliq Fiscal Integration (Fail-Closed), AR Invoicing, Dynamic Pricing          |
|  - High-entropy UUID PKs (zero Spanner tablet hotspotting)                       |
|                                                                                   |
|  [apps/ai-worker]                                                                 |
|  - Predictive Order Synthesis, Ingest Processing, Freeze Lock Coordination       |
|                                                                                   |
|  [services/optimizer-core]                                                        |
|  - Google OR-Tools VRP Solver (Cold-Chain, Hazmat, Time-Window & Volume Capped)  |
|                                                                                   |
|  [apps/handoff-service]                                                           |
|  - High-security Physical Custody Transfer with Constant-Time HMAC Verification   |
+-----------------------------------------------------------------------------------+
                                         |
                     +-------------------+-------------------+
                     |                                       |
                     v                                       v
+-----------------------------------------+ +---------------------------------------+
|          TRANSACTIONAL STORAGE          | |            EVENT BUS & CACHE          |
|  - Google Cloud Spanner (Global ACID)   | |  - Apache Kafka (At-least-once outbox)|
|  - Multi-tablet distributed tables      | |  - Redis Cluster (Cache-aside & rate) |
+-----------------------------------------+ +---------------------------------------+
```

---

## Core Invariants & Engineering Doctrine

All code in this repository enforces non-negotiable architectural laws:

1. **SSMR Tenancy Isolation**: Every order, payment, and inventory transaction is strictly scoped to the authenticated tenant. Cross-tenant reads and IDOR mutations fail closed with audit trails.
2. **Minor-Unit Financial Precision**: All monetary values are strictly represented as `int64` minor units (e.g., Tiyin for UZS). Floating-point math on currencies is strictly forbidden.
3. **Sole Transactional Authority**: Google Cloud Spanner is the single source of transactional truth. Mutations run atomically alongside outbox events in ReadWriteTransactions.
4. **Solver Honesty**: Logistics routing algorithms respect vehicle refrigeration and hazardous material constraints. Infeasible deliveries are isolated into unassigned orphans rather than dropped or routed onto uncertified trucks.
5. **No Coverage Theatre**: Unit and integration tests verify executable runtime behavior and invariants; test suites that read source code files to match regex strings are forbidden.
6. **Zero Tracked Binaries & Build Artifacts**: Build caches, compiler outputs, and sensitive state plans are kept out of source control.

---

## Repository Structure

```
pegasusx/
├── apps/
│   ├── backend-go/             # Main Go API server (~320K lines)
│   ├── ai-worker/              # Async planning, ingest & demand synthesis
│   ├── handoff-service/        # Secure QR/HMAC custody transfer microservice
│   ├── retailer-app-desktop/   # Tauri v2 + Next.js desktop retailer workstation
│   ├── supplier-portal/        # Web portal for supplier operations
│   ├── admin-portal/           # Platform governance & observability console
│   ├── warehouse-portal/       # Warehouse packing, staging & bay dispatch
│   ├── factory-portal/         # Factory intake & transfer management
│   ├── driver-app-android/     # Native Android driver app (Kotlin/Compose)
│   ├── retailer-app-android/   # Native Android retailer ordering client
│   ├── warehouse-app-android/  # Native Android inventory & barcode terminal
│   ├── driver-app-ios/         # Native iOS driver app (Swift/SwiftUI)
│   └── retailer-app-ios/       # Native iOS retailer client (Swift/SwiftUI)
├── services/
│   └── optimizer-core/         # OR-Tools VRP engine (Python/Rust)
├── packages/
│   ├── config/                 # Shared Go service configuration
│   ├── handoff/                # Cryptographic token definitions for handoffs
│   ├── optimizer-contract/     # Solver request/response definitions
│   └── types/                  # Cross-platform shared TypeScript contracts
├── contracts/
│   ├── events.schema.json      # Strict JSON Schema for all Kafka events
│   ├── partner.openapi.yaml    # Partner machine integration OpenAPI 3.0 spec
│   └── jwt-core.openapi.yaml   # Human JWT authentication contracts
├── infra/
│   ├── k8s/                    # Kubernetes manifests (Base + Prod/Staging/Pilot)
│   └── terraform/              # Terraform modules (GCP, Spanner, Networking, IAM)
└── scripts/                    # Parity gates, validation & deployment tooling
```

---

## Local Development & Prerequisites

### Prerequisites
- **Go**: 1.25+
- **Node.js**: 20+ with `pnpm` 9+
- **Python**: 3.12+ (for `optimizer-core`)
- **Docker**: For local infrastructure emulation (Spanner emulator, Kafka, Redis)

### Common Make Targets

```bash
# Display help and available gates
make help

# Verify cross-service contract parity
make parity-contract-full

# Verify contract schema strictness (additionalProperties: false)
make gen-contracts-gate

# Run comprehensive Kubernetes validation across all manifests & overlays
make validate-all-k8s

# Run partner OpenAPI schema verification
make partner-openapi-gate

# Verify Terraform formatting and hygiene
terraform fmt -check -recursive infra/terraform
```

---

## Build Verification

All Go modules are configured to compile independently without ambient workspace drift:

```bash
for m in apps/ai-worker apps/backend-go apps/handoff-service packages/config packages/handoff packages/optimizer-contract sdk/partner/go; do
  (cd "$m" && GOWORK=off go build ./...)
done
```

---

## License

This software and associated documentation files are proprietary and confidential to PegasusX. Unauthorized copying, distribution, modification, or deployment is strictly prohibited. See [LICENSE](LICENSE) for details.
