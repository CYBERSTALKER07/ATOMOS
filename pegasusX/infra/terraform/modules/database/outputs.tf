/**
 * Module: Database
 * Output declarations
 */

output "spanner_instance_id" {
  description = "The identifier of the Cloud Spanner instance."
  value       = google_spanner_instance.ledger.id
}

output "spanner_instance_name" {
  description = "The name of the Cloud Spanner instance."
  value       = google_spanner_instance.ledger.name
}

output "spanner_database_id" {
  description = "The identifier of the primary Cloud Spanner database."
  value       = google_spanner_database.main.id
}

output "spanner_database_name" {
  description = "The name of the primary Cloud Spanner database."
  value       = google_spanner_database.main.name
}

output "spanner_backup_schedule_id" {
  description = "The identifier of the Cloud Spanner backup schedule."
  value       = google_spanner_backup_schedule.daily_full.id
}

output "redis_id" {
  description = "The identifier of the Cloud Memorystore Redis instance."
  value       = google_redis_instance.cache.id
}

output "redis_name" {
  description = "The name of the Cloud Memorystore Redis instance."
  value       = google_redis_instance.cache.name
}

output "redis_host" {
  description = "The IP address of the primary Redis node."
  value       = google_redis_instance.cache.host
}

output "redis_port" {
  description = "The port number on which Redis is accepting connections."
  value       = google_redis_instance.cache.port
}

output "redis_auth_string" {
  description = "The AUTH string for authenticating with the Redis instance."
  value       = google_redis_instance.cache.auth_string
  sensitive   = true
}

output "redis_current_location_id" {
  description = "The current zone where the Redis primary endpoint is hosted."
  value       = google_redis_instance.cache.current_location_id
}
