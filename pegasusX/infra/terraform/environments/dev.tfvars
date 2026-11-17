# ==============================================================================
# PegasusX GCP Infrastructure — Development Environment (environments/dev.tfvars)
# Usage: terraform plan -var-file=environments/dev.tfvars -out=tfplan
# ==============================================================================

project_id  = "pegasusx-dev-project"
region      = "europe-west3"
zone        = "europe-west3-a"
environment = "development"
cell_id     = "uz"

# Networking (FinOps tuned flow logs)
network_name          = "pegasusx-dev-vpc"
subnet_name           = "pegasusx-dev-subnet"
primary_cidr_block    = "10.40.0.0/20"
pod_cidr_block        = "10.41.0.0/16"
service_cidr_block    = "10.42.0.0/20"
vpc_flow_log_interval = "INTERVAL_30_SEC"
vpc_flow_log_sampling = 0.1

# Compute (Standard mode + Spot VM pool for maximum dev cost savings)
cluster_name                  = "pegasusx-dev-gke"
enable_autopilot              = false
enable_spot_node_pool         = true
spot_node_machine_type        = "e2-standard-4"
spot_node_min_count           = 0
spot_node_max_count           = 4
enable_committed_use_discount = false

# Database & Cache (Minimum PU autoscaling + BASIC Redis tier)
spanner_instance_name = "pegasusx-dev-spanner"
spanner_config        = "regional-europe-west3"
spanner_autoscaling_config = {
  min_processing_units                  = 100
  max_processing_units                  = 200
  high_priority_cpu_utilization_percent = 65
  storage_utilization_percent           = 80
}
spanner_database_name         = "main"
spanner_backup_retention_days = 7
deletion_protection           = false
redis_instance_name           = "pegasusx-dev-redis"
redis_tier                    = "BASIC"
redis_memory_size_gb          = 2

# Messaging
kafka_cluster_id   = "pegasusx-dev-events"
kafka_vcpu_count   = 3
kafka_memory_bytes = 17179869184

# Storage & Security
media_bucket_name    = "pegasusx-dev-media"
updates_bucket_name  = "pegasusx-dev-app-updates"
imports_bucket_name  = "pegasusx-dev-imports-exports"
tf_state_bucket_name = "pegasusx-dev-terraform-state"
k8s_namespace        = "pegasusx"

# Observability
alert_email_endpoints = ["dev-alerts@pegasusx.io"]
api_hostname          = "api-dev.pegasusx.io"

labels = {
  managed_by      = "terraform"
  system          = "pegasusx"
  environment     = "development"
  cost_center     = "engineering-dev"
  owner           = "sre-platform"
  finops_governed = "true"
}
