output "repository_id" {
  description = "The Artifact Registry Docker repository ID."
  value       = google_artifact_registry_repository.images.repository_id
}

output "location" {
  description = "The region or multi-region location of the Artifact Registry repository."
  value       = google_artifact_registry_repository.images.location
}

output "url" {
  description = "The full Artifact Registry Docker repository base URL."
  value       = "${google_artifact_registry_repository.images.location}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.images.repository_id}"
}
