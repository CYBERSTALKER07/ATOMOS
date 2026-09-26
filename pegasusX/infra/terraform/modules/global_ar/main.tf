# GS-C5 — shared Artifact Registry. Cells may pull later. Do not apply from the catalog.
# Live UZ images stay in the cell project AR until ops wires this repo.
resource "google_artifact_registry_repository" "images" {
  project       = var.project_id
  location      = var.location
  repository_id = var.repository_id
  format        = "DOCKER"
  description   = "PegasusX shared images (GS-C5 global plane). Not a cell."

  labels = {
    app        = "pegasusx"
    plane      = "global"
    managed_by = "terraform"
  }
}
