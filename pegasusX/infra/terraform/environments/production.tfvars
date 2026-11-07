# ==============================================================================
# PegasusX GCP Infrastructure — Production Environment (environments/production.tfvars)
# Usage: terraform plan -var-file=environments/production.tfvars -out=tfplan
# ==============================================================================

project_id  = "pegasusx-production-project"
region      = "europe-west3"
zone        = "europe-west3-a"
environment = "production"
cell_id     = "uz"

# Networking
network_name          = "pegasusx-vpc"
subnet_name           = "pegasusx-primary-subnet"
primary_cidr_block    = "10.10.0.0/20"
pod_cidr_block        = "10.20.0.0/16"
service_cidr_block    = "10.30.0.0/20"
vpc_flow_log_interval = "INTERVAL_30_SEC"
vpc_flow_log_sampling = 0.1

# Compute (GKE Autopilot + Regional Compute Engine CUD for steady-state baseline)
cluster_name                  = "pegasusx-prod-gke"
enable_autopilot              = true
enable_spot_node_pool         = true
enable_committed_use_discount = true
cud_plan                      = "TWELVE_MONTH"
cud_vcpu_count                = 8
cud_memory_mb                 = 32768

# Database & Cache
spanner_instance_name = "pegasusx-ssmr-spanner"
spanner_config        = "regional-europe-west3"
spanner_autoscaling_config = {
  min_processing_units                  = 100
  max_processing_units                  = 400
  high_priority_cpu_utilization_percent = 65
  storage_utilization_percent           = 80
}
spanner_database_name         = "main"
spanner_backup_retention_days = 30
deletion_protection           = true
redis_instance_name           = "pegasusx-ssmr-redis"
redis_tier                    = "STANDARD_HA"
redis_memory_size_gb          = 5

# Messaging
kafka_cluster_id   = "pegasusx-events-cluster"
kafka_vcpu_count   = 3
kafka_memory_bytes = 17179869184

# Storage & Security
media_bucket_name    = "pegasusx-prod-media"
updates_bucket_name  = "pegasusx-prod-app-updates"
imports_bucket_name  = "pegasusx-prod-imports-exports"
tf_state_bucket_name = "pegasusx-terraform-state"
k8s_namespace        = "pegasusx"

# Observability
alert_email_endpoints = ["sre-alerts@pegasusx.io"]
api_hostname          = "api.pegasusx.io"

labels = {
  managed_by      = "terraform"
  system          = "pegasusx"
  environment     = "production"
  cost_center     = "platform-engineering"
  owner           = "sre-platform"
  finops_governed = "true"
}
