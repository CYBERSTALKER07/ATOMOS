# ==============================================================================
# PegasusX GCP Infrastructure — Staging Environment (environments/staging.tfvars)
# Usage: terraform plan -var-file=environments/staging.tfvars -out=tfplan
# ==============================================================================

project_id  = "pegasusx-staging-project"
region      = "europe-west3"
zone        = "europe-west3-a"
environment = "staging"
cell_id     = "uz"

# Networking
network_name          = "pegasusx-staging-vpc"
subnet_name           = "pegasusx-staging-subnet"
primary_cidr_block    = "10.50.0.0/20"
pod_cidr_block        = "10.51.0.0/16"
service_cidr_block    = "10.52.0.0/20"
vpc_flow_log_interval = "INTERVAL_30_SEC"
vpc_flow_log_sampling = 0.1

# Compute
cluster_name                  = "pegasusx-staging-gke"
enable_autopilot              = true
enable_spot_node_pool         = true
enable_committed_use_discount = false

# Database & Cache
spanner_instance_name = "pegasusx-staging-spanner"
spanner_config        = "regional-europe-west3"
spanner_autoscaling_config = {
  min_processing_units                  = 100
  max_processing_units                  = 300
  high_priority_cpu_utilization_percent = 65
  storage_utilization_percent           = 80
}
spanner_database_name         = "main"
spanner_backup_retention_days = 14
deletion_protection           = true
redis_instance_name           = "pegasusx-staging-redis"
redis_tier                    = "STANDARD_HA"
redis_memory_size_gb          = 3

# Messaging
kafka_cluster_id   = "pegasusx-staging-events"
kafka_vcpu_count   = 3
kafka_memory_bytes = 17179869184

# Storage & Security
media_bucket_name    = "pegasusx-staging-media"
updates_bucket_name  = "pegasusx-staging-app-updates"
imports_bucket_name  = "pegasusx-staging-imports-exports"
tf_state_bucket_name = "pegasusx-staging-terraform-state"
k8s_namespace        = "pegasusx"

# Observability
alert_email_endpoints = ["staging-alerts@pegasusx.io"]
api_hostname          = "api-staging.pegasusx.io"

labels = {
  managed_by      = "terraform"
  system          = "pegasusx"
  environment     = "staging"
  cost_center     = "platform-staging"
  owner           = "sre-platform"
  finops_governed = "true"
}
