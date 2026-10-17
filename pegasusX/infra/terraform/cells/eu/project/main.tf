# GS-C3 — declare project pegasusx-cell-eu. Do not apply from the catalog.
# After apply (ops-owned): same-root cell stack uses cells/eu/cell.tfvars
# (europe-west1, own Spanner/Redis/Kafka/GKE, GSM regional, new JWT).
# Schema = migrations (scripts/cell_migrate.sh), never a UZ backup restore.

check "eu_project_is_not_live_uz" {
  assert {
    condition     = var.project_id != "pegasus-503013"
    error_message = "GS-C3: EU project_id must not be pegasus-503013."
  }
}

check "eu_project_region" {
  assert {
    condition     = var.region == "europe-west1"
    error_message = "GS-C3: EU project factory must use europe-west1."
  }
}

resource "google_project" "cell" {
  name            = var.project_name
  project_id      = var.project_id
  org_id          = trimspace(var.org_id) != "" ? var.org_id : null
  folder_id       = trimspace(var.folder_id) != "" ? var.folder_id : null
  billing_account = trimspace(var.billing_account_id) != "" ? var.billing_account_id : null

  labels = {
    app        = "pegasusx"
    cell       = var.cell_id
    managed_by = "terraform"
  }
}

resource "google_project_service" "cell" {
  for_each = toset([
    "compute.googleapis.com",
    "container.googleapis.com",
    "spanner.googleapis.com",
    "redis.googleapis.com",
    "secretmanager.googleapis.com",
    "managedkafka.googleapis.com",
    "iam.googleapis.com",
    "artifactregistry.googleapis.com",
    "servicenetworking.googleapis.com",
  ])
  project            = google_project.cell.project_id
  service            = each.key
  disable_on_destroy = false
}

output "project_id" {
  value = google_project.cell.project_id
}

output "cell_id" {
  value = var.cell_id
}

output "region" {
  value = var.region
}
