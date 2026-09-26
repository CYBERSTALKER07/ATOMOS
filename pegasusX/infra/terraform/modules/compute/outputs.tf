/**
 * Module: Compute
 * Output declarations
 */

output "cluster_id" {
  description = "The ID of the GKE cluster."
  value       = google_container_cluster.primary.id
}

output "cluster_name" {
  description = "The name of the GKE cluster."
  value       = google_container_cluster.primary.name
}

output "cluster_endpoint" {
  description = "The IP address of the GKE cluster control plane endpoint."
  value       = google_container_cluster.primary.endpoint
}

output "cluster_ca_certificate" {
  description = "The public CA certificate used by clients to authenticate to the GKE cluster."
  value       = google_container_cluster.primary.master_auth[0].cluster_ca_certificate
  sensitive   = true
}

output "node_service_account_email" {
  description = "The email address of the GKE node service account."
  value       = google_service_account.gke_node_sa.email
}

output "workload_identity_pool" {
  description = "The Workload Identity pool name for the project."
  value       = "${var.project_id}.svc.id.goog"
}

output "location" {
  description = "The region of the GKE cluster."
  value       = google_container_cluster.primary.location
}
