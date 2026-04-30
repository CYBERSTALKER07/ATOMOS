# Outputs expose the cluster coordinates and shared infrastructure details that
# GitHub Actions and the ops bootstrap job need after `terraform apply`.
# Reference these in CI with: terraform output -raw <name>

output "gke_cluster_name" {
  description = "GKE cluster name used by kubectl config and GitHub Actions deploy steps."
  value       = google_container_cluster.void_gke.name
}

output "gke_cluster_endpoint" {
  description = "Cluster API server endpoint — used by the kubernetes + helm providers."
  value       = google_container_cluster.void_gke.endpoint
  sensitive   = true
}

output "gke_cluster_ca_certificate" {
  description = "Base64-encoded cluster CA certificate for kubeconfig generation."
  value       = google_container_cluster.void_gke.master_auth[0].cluster_ca_certificate
  sensitive   = true
}

output "void_backend_sa_email" {
  description = "GCP service account email bound to the Kubernetes SA via Workload Identity."
  value       = google_service_account.void_backend_sa.email
}

output "redis_host" {
  description = "Memorystore Redis private IP — passed to backend-go as REDIS_ADDRESS host."
  value       = google_redis_instance.cache.host
  sensitive   = true
}

output "redis_port" {
  description = "Memorystore Redis port (default 6379)."
  value       = google_redis_instance.cache.port
}

output "spanner_database_uri" {
  description = "Full Spanner database URI for SPANNER_DATABASE_URI env var."
  value       = "projects/${var.gcp_project_id}/instances/${google_spanner_instance.ledger.name}/databases/${google_spanner_database.main_db.name}"
}

output "cloud_run_url" {
  description = "Cloud Run backend service URL (interim surface until GKE cutover)."
  value       = google_cloud_run_v2_service.backend.uri
}
