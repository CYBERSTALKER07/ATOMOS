/**
 * Root Terraform Configuration: PegasusX FMCG Infrastructure on GCP
 *
 * This configuration composes 6 modularized resource groups organized by rollout phase:
 * Phase 1: Networking (VPC, Subnets, Cloud NAT with 2 static egress IPs, PSA Peering, Firewalls)
 * Phase 2: Database (Cloud Spanner Ledger, Automated Backup Schedule, Memorystore Redis 7.0 HA)
 * Phase 2: Messaging (Google Managed Service for Apache Kafka, 10 Canonical Topics)
 * Phase 2: Storage & Security (4 GCS Buckets, Cloud Armor WAF, Secret Manager, Workload Identity IAM)
 * Phase 3: Compute (GKE Autopilot/Standard Multi-Zone Regional Cluster, Node SA)
 * Phase 4: Monitoring (Cloud Monitoring 12 Alert Policies, Notification Channels, Dashboard, Uptime Probes)
 */

# 1. Phase 1: Networking Module
module "networking" {
  source = "./modules/networking"

  project_id         = var.project_id
  region             = var.region
  network_name       = var.network_name
  subnet_name        = var.subnet_name
  primary_cidr_block = var.primary_cidr_block
  pod_cidr_block     = var.pod_cidr_block
  service_cidr_block = var.service_cidr_block
  environment        = var.environment
  labels             = var.labels
}

# 2. Phase 2: Database Module (Cloud Spanner & Memorystore Redis 7.0 HA)
module "database" {
  source = "./modules/database"

  project_id                    = var.project_id
  region                        = var.region
  spanner_instance_name         = var.spanner_instance_name
  spanner_config                = var.spanner_config
  spanner_processing_units      = var.spanner_processing_units
  spanner_database_name         = var.spanner_database_name
  spanner_backup_retention_days = var.spanner_backup_retention_days
  redis_instance_name           = var.redis_instance_name
  redis_memory_size_gb          = var.redis_memory_size_gb
  vpc_id                        = module.networking.network_id
  psa_connection                = module.networking.psa_connection_id
  deletion_protection           = var.deletion_protection
  environment                   = var.environment
  labels                        = var.labels
}

# 3. Phase 2: Messaging Module (Managed Apache Kafka & 10 Canonical Topics)
module "messaging" {
  source = "./modules/messaging"

  project_id         = var.project_id
  region             = var.region
  kafka_cluster_id   = var.kafka_cluster_id
  subnet_id          = module.networking.subnetwork_id
  kafka_vcpu_count   = var.kafka_vcpu_count
  kafka_memory_bytes = var.kafka_memory_bytes
  environment        = var.environment
  labels             = var.labels
}

# 4. Phase 2: Storage & Security Module (4 GCS Buckets, Cloud Armor WAF, Secrets, IAM Workload Identity)
module "storage_security" {
  source = "./modules/storage_security"

  project_id           = var.project_id
  region               = var.region
  environment          = var.environment
  media_bucket_name    = var.media_bucket_name
  updates_bucket_name  = var.updates_bucket_name
  imports_bucket_name  = var.imports_bucket_name
  tf_state_bucket_name = var.tf_state_bucket_name
  k8s_namespace        = var.k8s_namespace
  labels               = var.labels
}

# 5. Phase 3: Compute Module (GKE Autopilot / Standard Regional Cluster)
module "compute" {
  source = "./modules/compute"

  project_id             = var.project_id
  region                 = var.region
  cluster_name           = var.cluster_name
  network_name           = module.networking.network_name
  subnetwork_name        = module.networking.subnetwork_name
  pod_ip_range_name      = module.networking.pod_ip_range_name
  service_ip_range_name  = module.networking.service_ip_range_name
  enable_autopilot       = var.enable_autopilot
  master_ipv4_cidr_block = var.master_ipv4_cidr_block
  environment            = var.environment
  labels                 = var.labels

  depends_on = [
    module.networking
  ]
}

# 6. Phase 4: Monitoring Module (Alert Policies, Notification Channels, Dashboards, Uptime Probes)
module "monitoring" {
  source = "./modules/monitoring"

  project_id            = var.project_id
  environment           = var.environment
  alert_email_endpoints = var.alert_email_endpoints
  slack_webhook_url     = var.slack_webhook_url
  api_hostname          = var.api_hostname
  labels                = var.labels
}
