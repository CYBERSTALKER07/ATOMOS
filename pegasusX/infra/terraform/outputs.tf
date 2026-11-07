/**
 * Root Terraform Output Declarations
 */

# 1. Networking Outputs
output "vpc_network_id" {
  description = "The ID of the provisioned VPC network."
  value       = module.networking.network_id
}

output "vpc_network_name" {
  description = "The name of the provisioned VPC network."
  value       = module.networking.network_name
}

output "primary_subnet_id" {
  description = "The ID of the primary subnetwork."
  value       = module.networking.subnetwork_id
}

output "cloud_nat_static_ips" {
  description = "The 2 deterministic static external egress IPs for Soliq OFD & Banking allowlists."
  value       = module.networking.nat_ip_addresses
}

# 2. Database & Cache Outputs
output "spanner_instance_name" {
  description = "The name of the Cloud Spanner instance."
  value       = module.database.spanner_instance_name
}

output "spanner_database_name" {
  description = "The name of the primary Cloud Spanner database."
  value       = module.database.spanner_database_name
}

output "redis_host" {
  description = "The private IP address of the Cloud Memorystore Redis 7.0 HA cluster."
  value       = module.database.redis_host
}

output "redis_port" {
  description = "The port on which Cloud Memorystore Redis is listening."
  value       = module.database.redis_port
}

output "redis_auth_string" {
  description = "The AUTH string for the Redis cluster."
  value       = module.database.redis_auth_string
  sensitive   = true
}

# 3. Messaging Outputs
output "kafka_cluster_id" {
  description = "The cluster ID of the Google Managed Service for Apache Kafka cluster."
  value       = module.messaging.kafka_cluster_id
}

output "kafka_topics" {
  description = "The list of canonical partitioned Kafka topic IDs."
  value       = module.messaging.kafka_topics
}

# 4. Storage & Security Outputs
output "media_bucket_name" {
  description = "GCS bucket name for evidence dossiers and media."
  value       = module.storage_security.media_bucket_name
}

output "updates_bucket_name" {
  description = "GCS bucket name for OTA mobile APKs and desktop updates."
  value       = module.storage_security.updates_bucket_name
}

output "imports_bucket_name" {
  description = "GCS bucket name for bulk imports and compliance exports."
  value       = module.storage_security.imports_bucket_name
}

output "cloud_armor_policy_id" {
  description = "The ID of the Cloud Armor Enterprise WAF policy."
  value       = module.storage_security.cloud_armor_policy_id
}

output "workload_service_account_emails" {
  description = "Map of Google Service Account emails bound to Workload Identity."
  value       = module.storage_security.service_account_emails
}

# 5. Compute Outputs
output "gke_cluster_name" {
  description = "The name of the GKE cluster."
  value       = module.compute.cluster_name
}

output "gke_cluster_endpoint" {
  description = "The endpoint IP for the GKE cluster control plane."
  value       = module.compute.cluster_endpoint
}

output "gke_ca_certificate" {
  description = "The public CA certificate of the GKE cluster."
  value       = module.compute.cluster_ca_certificate
  sensitive   = true
}

# 6. Monitoring Outputs
output "monitoring_alert_policy_ids" {
  description = "Map of created Cloud Monitoring alert policy IDs."
  value       = module.monitoring.alert_policy_ids
}

output "monitoring_dashboard_id" {
  description = "The ID of the unified platform telemetry dashboard."
  value       = module.monitoring.dashboard_id
}

# 7. Kubernetes Render & Multi-Cell Governance Outputs
output "artifact_registry_url" {
  description = "The Docker registry URL for the cell Artifact Registry repository."
  value       = module.compute.artifact_registry_url
}

output "backend_runtime_service_account" {
  description = "The email of the primary backend-go Workload Identity Google Service Account."
  value       = module.storage_security.service_account_emails["backend-go-sa"]
}

output "spanner_database_uri" {
  description = "The full GCP resource URI of the Cloud Spanner database."
  value       = module.database.spanner_database_uri
}

output "managed_kafka_bootstrap" {
  description = "Bootstrap host:port endpoint for the Google Managed Service for Apache Kafka cluster."
  value       = module.messaging.managed_kafka_bootstrap
}

output "jwt_secret_id" {
  description = "Secret Manager secret ID for JWT signing."
  value       = module.storage_security.secret_ids["pegasusx-jwt-signing-key"]
}

output "global_pay_webhook_secret_id" {
  description = "Secret Manager secret ID for Global Pay webhook verification."
  value       = module.storage_security.secret_ids["pegasusx-global-pay-secret"]
}

output "adyen_webhook_secret_id" {
  description = "Secret Manager secret ID for Adyen webhook verification."
  value       = module.storage_security.secret_ids["pegasusx-adyen-secret"]
}

output "stripe_webhook_secret_id" {
  description = "Secret Manager secret ID for Stripe webhook verification."
  value       = module.storage_security.secret_ids["pegasusx-stripe-secret"]
}

output "google_maps_api_key_secret_id" {
  description = "Secret Manager secret ID for Google Maps Platform server API key."
  value       = module.storage_security.secret_ids["pegasusx-google-maps-api-key"]
}

output "kafka_bootstrap_secret" {
  description = "Secret Manager secret ID for Kafka bootstrap servers."
  value       = module.storage_security.secret_ids["pegasusx-kafka-bootstrap-servers"]
}

output "cell_id" {
  description = "Logical cell identifier for this deployment."
  value       = local.effective_cell_id
}

output "api_hostname" {
  description = "Public API hostname for this cell."
  value       = var.api_hostname
}

output "k8s_namespace" {
  description = "Kubernetes namespace used for Workload Identity federation."
  value       = var.k8s_namespace
}

output "terraform_state_prefix" {
  description = "Declared GCS remote state prefix for this cell."
  value       = var.terraform_state_prefix
}
