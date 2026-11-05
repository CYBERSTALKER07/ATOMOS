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

locals {
  base_finops_labels = merge(
    {
      managed_by      = "terraform"
      system          = "pegasusx"
      cell            = local.effective_cell_id
      environment     = var.environment
      cost_center     = "platform-engineering"
      owner           = "sre-platform"
      finops_governed = "true"
    },
    var.labels
  )

  required_gcp_apis = toset([
    "compute.googleapis.com",
    "container.googleapis.com",
    "spanner.googleapis.com",
    "redis.googleapis.com",
    "servicenetworking.googleapis.com",
    "secretmanager.googleapis.com",
    "artifactregistry.googleapis.com",
    "managedkafka.googleapis.com",
    "cloudkms.googleapis.com",
    "monitoring.googleapis.com",
    "logging.googleapis.com",
    "cloudbilling.googleapis.com",
    "billingbudgets.googleapis.com",
    "routes.googleapis.com",
    "geocoding-backend.googleapis.com",
    "places.googleapis.com",
    "iam.googleapis.com",
  ])
}

# 0. Phase 0: Project API Activation
resource "google_project_service" "apis" {
  for_each                   = local.required_gcp_apis
  project                    = var.project_id
  service                    = each.value
  disable_dependent_services = false
  disable_on_destroy         = false
}

# 1. Phase 1: Networking Module
module "networking" {
  source = "./modules/networking"

  project_id         = var.project_id
  region             = var.region
  network_name       = var.network_name
  subnet_name        = var.subnet_name
  primary_cidr_block = local.effective_primary_cidr_block
  pod_cidr_block     = local.effective_pod_cidr_block
  service_cidr_block = local.effective_service_cidr_block
  flow_log_interval  = var.vpc_flow_log_interval
  flow_log_sampling  = var.vpc_flow_log_sampling
  environment        = var.environment
  labels             = merge(local.base_finops_labels, { tier = "networking" })

  depends_on = [
    google_project_service.apis
  ]
}

# 2. Phase 2: Database Module (Cloud Spanner & Memorystore Redis 7.0 HA)
module "database" {
  source = "./modules/database"

  project_id                    = var.project_id
  region                        = var.region
  spanner_instance_name         = var.spanner_instance_name
  spanner_config                = var.spanner_config
  spanner_processing_units      = var.spanner_processing_units
  spanner_autoscaling_config    = var.spanner_autoscaling_config
  spanner_database_name         = var.spanner_database_name
  spanner_backup_retention_days = var.spanner_backup_retention_days
  redis_instance_name           = var.redis_instance_name
  redis_tier                    = var.redis_tier
  redis_memory_size_gb          = var.redis_memory_size_gb
  vpc_id                        = module.networking.network_id
  psa_connection                = module.networking.psa_connection_id
  deletion_protection           = var.deletion_protection
  environment                   = var.environment
  labels                        = merge(local.base_finops_labels, { tier = "database" })

  depends_on = [
    google_project_service.apis
  ]
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
  labels             = merge(local.base_finops_labels, { tier = "messaging" })

  depends_on = [
    google_project_service.apis
  ]
}

# 4. Phase 2: Storage & Security Module (4 GCS Buckets, Cloud Armor WAF, Secrets, IAM Workload Identity)
module "storage_security" {
  source = "./modules/storage_security"

  project_id              = var.project_id
  region                  = var.region
  environment             = var.environment
  media_bucket_name       = var.media_bucket_name
  updates_bucket_name     = var.updates_bucket_name
  imports_bucket_name     = var.imports_bucket_name
  tf_state_bucket_name    = var.tf_state_bucket_name
  k8s_namespace           = var.k8s_namespace
  enable_cmek_iam_binding = var.enable_cmek_iam_binding
  gsm_regional_only       = var.gsm_regional_only
  cell_scoped_iam         = var.cell_scoped_iam
  spanner_instance_name   = module.database.spanner_instance_name
  spanner_database_name   = module.database.spanner_database_name
  labels                  = merge(local.base_finops_labels, { tier = "storage" })

  depends_on = [
    google_project_service.apis
  ]
}

# 5. Phase 3: Compute Module (GKE Autopilot / Standard Regional Cluster + Spot Pool + CUDs)
module "compute" {
  source = "./modules/compute"

  project_id                    = var.project_id
  region                        = var.region
  cluster_name                  = var.cluster_name
  network_name                  = module.networking.network_name
  subnetwork_name               = module.networking.subnetwork_name
  pod_ip_range_name             = module.networking.pod_ip_range_name
  service_ip_range_name         = module.networking.service_ip_range_name
  enable_autopilot              = var.enable_autopilot
  master_ipv4_cidr_block        = var.master_ipv4_cidr_block
  enable_spot_node_pool         = var.enable_spot_node_pool
  spot_node_machine_type        = var.spot_node_machine_type
  spot_node_min_count           = var.spot_node_min_count
  spot_node_max_count           = var.spot_node_max_count
  enable_committed_use_discount = var.enable_committed_use_discount
  cud_plan                      = var.cud_plan
  cud_vcpu_count                = var.cud_vcpu_count
  cud_memory_mb                 = var.cud_memory_mb
  environment                   = var.environment
  labels                        = merge(local.base_finops_labels, { tier = "compute" })

  depends_on = [
    google_project_service.apis,
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
  billing_account_id    = var.billing_account_id
  monthly_budget_usd    = var.monthly_budget_usd
  labels                = merge(local.base_finops_labels, { tier = "monitoring" })

  depends_on = [
    google_project_service.apis
  ]
}
