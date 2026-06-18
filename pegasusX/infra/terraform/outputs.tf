output "project_id" {
  description = "GCP project id passed to this module."
  value       = var.project_id
}

output "redis_host" {
  description = "Memorystore Redis host."
  value       = google_redis_instance.cache.host
  sensitive   = true
}

output "redis_port" {
  description = "Memorystore Redis port."
  value       = google_redis_instance.cache.port
}

output "spanner_database_uri" {
  description = "Spanner URI for runtime SPANNER_* env wiring."
  value       = "projects/${var.project_id}/instances/${google_spanner_instance.ledger.name}/databases/${google_spanner_database.main.name}"
}

output "tenant_slug" {
  description = "Tenant slug used to namespace the isolated SSMR sandbox resources."
  value       = local.tenant_slug
}

output "kafka_bootstrap_secret" {
  description = "Secret Manager secret name storing Kafka bootstrap servers."
  value       = google_secret_manager_secret.kafka_bootstrap_servers.secret_id
}

output "kafka_topic_main_secret" {
  description = "Secret Manager secret name storing kafka topic main."
  value       = google_secret_manager_secret.kafka_topic_main.secret_id
}

output "kafka_topic_spatial_secret" {
  description = "Secret Manager secret name storing kafka topic spatial."
  value       = google_secret_manager_secret.kafka_topic_spatial.secret_id
}

output "kafka_topic_realtime_secret" {
  description = "Secret Manager secret name storing kafka topic realtime."
  value       = google_secret_manager_secret.kafka_topic_realtime.secret_id
}

output "kafka_topic_webhooks_secret" {
  description = "Secret Manager secret name storing kafka topic webhooks."
  value       = google_secret_manager_secret.kafka_topic_webhooks.secret_id
}

output "firebase_project_id_secret" {
  description = "Secret Manager secret name storing Firebase project id."
  value       = google_secret_manager_secret.firebase_project_id.secret_id
}

output "firebase_auth_enabled_secret" {
  description = "Secret Manager secret name storing firebase auth toggle."
  value       = google_secret_manager_secret.firebase_auth_enabled.secret_id
}

output "gke_cluster_name" {
  description = "GKE cluster name when enable_gke=true."
  value       = var.enable_gke ? google_container_cluster.pegasusx[0].name : ""
}

output "artifact_registry_repository" {
  description = "Artifact Registry Docker repository id when enable_gke=true."
  value       = var.enable_gke ? google_artifact_registry_repository.pegasusx[0].repository_id : ""
}

output "backend_runtime_service_account" {
  description = "GCP service account email for backend-go workload identity."
  value       = var.enable_gke ? google_service_account.backend_runtime[0].email : ""
}

output "jwt_secret_id" {
  description = "Secret Manager secret id for JWT signing key."
  value       = google_secret_manager_secret.jwt_secret.secret_id
}

output "global_pay_webhook_secret_id" {
  description = "Secret Manager secret id for GlobalPay webhook HMAC."
  value       = google_secret_manager_secret.global_pay_webhook_secret.secret_id
}

output "adyen_webhook_secret_id" {
  description = "Secret Manager secret id for Adyen webhook secret."
  value       = google_secret_manager_secret.adyen_webhook_secret.secret_id
}

output "stripe_webhook_secret_id" {
  description = "Secret Manager secret id for Stripe webhook secret."
  value       = google_secret_manager_secret.stripe_webhook_secret.secret_id
}

output "google_maps_api_key_secret_id" {
  description = "Secret Manager secret id for Google Maps Platform API key."
  value       = google_secret_manager_secret.google_maps_api_key.secret_id
}

output "artifact_registry_url" {
  description = "Artifact Registry repository URL when enable_gke=true."
  value = var.enable_gke ? "${google_artifact_registry_repository.pegasusx[0].location}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.pegasusx[0].repository_id}" : ""
}
