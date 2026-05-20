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
