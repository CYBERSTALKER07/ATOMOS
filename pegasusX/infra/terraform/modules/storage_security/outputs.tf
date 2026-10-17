/**
 * Module: Storage & Security
 * Output declarations
 */

output "media_bucket_name" {
  description = "The name of the GCS bucket for media and cryptographic evidence dossiers."
  value       = google_storage_bucket.media.name
}

output "media_bucket_url" {
  description = "The URL of the media bucket."
  value       = google_storage_bucket.media.url
}

output "updates_bucket_name" {
  description = "The name of the GCS bucket for OTA app updates."
  value       = google_storage_bucket.updates.name
}

output "updates_bucket_url" {
  description = "The URL of the updates bucket."
  value       = google_storage_bucket.updates.url
}

output "imports_bucket_name" {
  description = "The name of the GCS bucket for bulk imports and compliance exports."
  value       = google_storage_bucket.imports_exports.name
}

output "imports_bucket_url" {
  description = "The URL of the imports/exports bucket."
  value       = google_storage_bucket.imports_exports.url
}

output "tf_state_bucket_name" {
  description = "The name of the GCS bucket for Terraform state."
  value       = google_storage_bucket.tf_state.name
}

output "cloud_armor_policy_id" {
  description = "The ID of the Cloud Armor security policy."
  value       = google_compute_security_policy.edge_waf.id
}

output "cloud_armor_policy_name" {
  description = "The name of the Cloud Armor security policy."
  value       = google_compute_security_policy.edge_waf.name
}

output "secret_ids" {
  description = "Map of created Secret Manager secret IDs."
  value       = { for k, v in google_secret_manager_secret.app_secrets : k => v.secret_id }
}

output "service_account_emails" {
  description = "Map of created Google Service Account emails."
  value       = { for k, v in google_service_account.workload_sa : k => v.email }
}
