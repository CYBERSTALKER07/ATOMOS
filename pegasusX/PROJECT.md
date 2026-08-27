# Project: PegasusX Enterprise Infrastructure on Google Cloud Platform (GCP)

## Architecture Overview
PegasusX is a multi-tier, multi-tenant FMCG supply chain operating system. The production GCP deployment architecture leverages:
- **Compute Layer**: Google Kubernetes Engine (GKE) Autopilot with Workload Identity, running containerized Go REST/WebSocket API gateways, asynchronous worker daemons, Python OR-Tools VRP solvers, and microservices.
- **Database Layer**: Cloud Spanner (multi-zone regional instance with 100+ tables, 13 interleaved hierarchies, and strict ACID guarantees) + Cloud Memorystore for Redis 7.0 (low-latency cache, request idempotency, and cross-pod WebSocket pub/sub mesh).
- **Event & Messaging Layer**: Google Managed Service for Apache Kafka (10 canonical topics with transactional outbox relay and consumer groups) + Redis Pub/Sub for real-time WebSocket fanout.
- **Object Storage & CDN**: Google Cloud Storage (GCS) buckets for evidence dossiers, app updates, and data imports/exports + Cloud CDN + Cloud Armor WAF for edge caching, DDoS protection, and SSL termination.
- **Security & Networking**: Private VPC with dual subnets (GKE nodes/pods/services), Cloud NAT for static outbound IP allowlisting (Soliq OFD, banking gateways), Secret Manager + External Secrets Operator, and IAM Workload Identity.
- **Third-Party Integrations**: Soliq OFD (PKCS#12 digital tax invoicing), Global Pay / Payme / Click payment gateways, PlayMobile / Twilio SMS aggregators, SendGrid email, Google Maps Platform / OSRM routing, and Firebase Phone Auth + FCM push notifications.

## Feature Inventory
| # | Feature | Description | Milestone | Source |
|---|---------|-------------|-----------|--------|
| 1 | Monorepo Service & Dependency Scan | Exhaustive catalog of all Go services, Python optimizer, web portals, mobile apps, database schemas, Kafka topics, GCS buckets, and third-party APIs | M1 | Survey Explorer 1, 2, 3 |
| 2 | Compute Platform Justification | Comprehensive trade-off analysis justifying GKE (Autopilot/Standard) over Cloud Run / VMs for CGO (h3-go), persistent WebSockets, and OpenMP Python solvers | M2 | Survey Explorer 1, 2, 3 |
| 3 | Phased GCP Architecture & Service Mapping | End-to-end wiring plan mapping all 25+ services, workers, and storage layers to GCP native resources across 4 rollout phases | M2 | Survey Explorer 1, 2, 3 |
| 4 | Third-Party Integration Architecture | Technical specs for Soliq OFD (E-IMZO PKCS#12), Payment rails (Global Pay/Payme/Click), SMS (PlayMobile/Twilio), Routing (Google Routes v2 + OSRM), and Firebase | M2 | Survey Explorer 2, 3 |
| 5 | Monthly TCO & Capacity Sizing Model | Detailed monthly cost breakdown for GCP infrastructure and 3rd-party services scaled for 1000 retailers, 50 suppliers, and 200 drivers | M2 | Survey Explorer 1, 2, 3 |
| 6 | HA, Autoscaling, DR & Monitoring Plan | Multi-zone SLA (99.99%), HPA rules, Cloud Monitoring dashboards, PromQL alerts, synthetic health probes, and disaster recovery RPO/RTO | M2 | Survey Explorer 1, 2, 3 |
| 7 | Security Hardening & Zero-Trust IAM | VPC firewall rules, Cloud Armor WAF policies, Cloud NAT static egress, Workload Identity bindings, and Secret Manager rotation | M2 | Survey Explorer 3 |
| 8 | Modular Terraform Networking Module | VPC, subnets, secondary pod/service IP ranges, Cloud Router, Cloud NAT with static EIPs, and firewall rules | M3 | Survey Explorer 3, IaC |
| 9 | Modular Terraform Compute / GKE Module | GKE Autopilot / Standard cluster, node pools, Workload Identity, Release Channel, and network policies | M3 | Survey Explorer 1, 3, IaC |
| 10 | Modular Terraform Database Module | Cloud Spanner instance (PU-based sizing), databases, IAM, and Cloud Memorystore for Redis 7.0 HA instance | M3 | Survey Explorer 1, 3, IaC |
| 11 | Modular Terraform Messaging / Kafka Module | Google Managed Service for Apache Kafka cluster, capacity sizing, and 10 canonical partitioned topics | M3 | Survey Explorer 2, 3, IaC |
| 12 | Modular Terraform Storage & Security Module | 4 GCS buckets with lifecycle rules/CORS, Cloud Armor security policies, Secret Manager secrets, and IAM service accounts | M3 | Survey Explorer 2, 3, IaC |
| 13 | Modular Terraform Monitoring Module | Cloud Monitoring notification channels, alerting policies (Spanner CPU, GKE pod crashloop, Kafka lag, HTTP 5xx, Redis memory), and dashboards | M3 | Survey Explorer 1, 3, IaC |
| 14 | Root Terraform Configuration & Staging/Prod Environments | Root `main.tf`, `variables.tf`, `outputs.tf`, `terraform.tfvars.example` linking all modules with clean variable passing and documentation | M3 | IaC Architecture |
| 15 | IaC Validation & Verification | Complete execution of `terraform init -backend=false` and `terraform validate` across all modules and root | M4 | Reviewer / IaC Verification |
| 16 | Agent-as-Judge Codebase Inventory Verification | Independent review audit verifying 100% fidelity between codebase scan findings and physical repository assets | M4 | Reviewer / Auditor |

## Milestones
| # | Name | Scope | Dependencies | Status |
|---|------|-------|-------------|--------|
| M1 | Codebase Infrastructure Inventory Document | Produce `INVENTORY.md` containing the exhaustive catalog of every microservice, worker daemon, CLI tool, database schema, cache, Kafka topic, client app, GCS bucket, and 3rd-party API | none | DONE |
| M2 | Phased GCP Architecture Document | Produce `GCP_ARCHITECTURE_PLAN.md` with compute justification, service mapping, 4-phase rollout, 3rd-party integrations, 1000-retailer cost model, HA/DR, autoscaling, Cloud Monitoring, and security hardening | M1 | DONE |
| M3 | Modularized Terraform IaC Suite | Build production-ready, modular Terraform templates under `infra/terraform/` (modules: networking, compute, database, messaging, storage_security, monitoring + root orchestration) passing validation | M2 | DONE |
| M4 | Independent Review & IaC Validation Gate | Run `terraform validate` across all templates, conduct independent Agent-as-Judge inventory review, and synthesize `VALIDATION_REPORT.md` | M3 | DONE |

## Interface Contracts & Module Boundaries
- **Networking Module (`infra/terraform/modules/networking`)**:
  - Outputs: `vpc_id`, `vpc_self_link`, `subnet_id`, `subnet_self_link`, `pod_ip_range_name`, `service_ip_range_name`, `nat_ip_addresses`
- **Database Module (`infra/terraform/modules/database`)**:
  - Inputs: `vpc_id`, `vpc_self_link`, `spanner_processing_units`, `redis_memory_size_gb`
  - Outputs: `spanner_instance_name`, `spanner_database_name`, `redis_host`, `redis_port`
- **Messaging Module (`infra/terraform/modules/messaging`)**:
  - Inputs: `vpc_id`, `subnet_id`, `kafka_cpu`, `kafka_memory_gib`
  - Outputs: `kafka_cluster_name`, `kafka_bootstrap_servers`, `topic_names`
- **Compute Module (`infra/terraform/modules/compute`)**:
  - Inputs: `vpc_id`, `subnet_id`, `pod_ip_range_name`, `service_ip_range_name`
  - Outputs: `gke_cluster_id`, `gke_cluster_name`, `gke_endpoint`, `gke_ca_certificate`
- **Storage & Security Module (`infra/terraform/modules/storage_security`)**:
  - Inputs: `project_id`, `region`, `environment`
  - Outputs: `media_bucket_name`, `updates_bucket_name`, `imports_bucket_name`, `cloud_armor_policy_id`, `service_account_emails`
- **Monitoring Module (`infra/terraform/modules/monitoring`)**:
  - Inputs: `project_id`, `alert_email_endpoints`, `gke_cluster_name`, `spanner_instance_name`, `redis_instance_name`, `kafka_cluster_name`
  - Outputs: `alert_policy_ids`, `dashboard_ids`

## Code Layout
- Documentation:
  - `INVENTORY.md` — Monorepo infrastructure and service inventory catalog
  - `GCP_ARCHITECTURE_PLAN.md` — Phased GCP infrastructure architecture and wiring plan
  - `VALIDATION_REPORT.md` — Independent validation, test logs, and review signoff
- Terraform IaC:
  - `infra/terraform/main.tf` — Root configuration calling all modules
  - `infra/terraform/variables.tf` — Root variable declarations
  - `infra/terraform/outputs.tf` — Root output declarations
  - `infra/terraform/terraform.tfvars.example` — Example variables for deployment
  - `infra/terraform/modules/networking/` — VPC, subnets, Cloud NAT, firewall rules
  - `infra/terraform/modules/compute/` — GKE Autopilot / Workload Identity
  - `infra/terraform/modules/database/` — Cloud Spanner & Memorystore Redis
  - `infra/terraform/modules/messaging/` — Managed Service for Apache Kafka & topics
  - `infra/terraform/modules/storage_security/` — GCS buckets, Cloud Armor, IAM, Secrets
  - `infra/terraform/modules/monitoring/` — Alert policies, notification channels, dashboards
