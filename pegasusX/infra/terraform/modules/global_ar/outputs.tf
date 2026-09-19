output "repository_id" {
  value = google_artifact_registry_repository.images.repository_id
}

output "location" {
  value = google_artifact_registry_repository.images.location
}

output "url" {
  value = "${google_artifact_registry_repository.images.location}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.images.repository_id}"
}
