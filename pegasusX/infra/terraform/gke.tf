variable "enable_gke" {
  description = "Provision GKE Autopilot cluster and Artifact Registry for pegasusX runtime."
  type        = bool
  default     = false
}

variable "gke_cluster_name" {
  description = "GKE cluster name."
  type        = string
  default     = ""
}

variable "artifact_registry_location" {
  description = "Artifact Registry region (defaults to var.region)."
  type        = string
  default     = ""
}

resource "google_project_service" "gke_apis" {
  for_each = var.enable_gke ? toset([
    "container.googleapis.com",
    "artifactregistry.googleapis.com",
    "iam.googleapis.com",
  ]) : toset([])
  service            = each.key
  disable_on_destroy = false
}

resource "google_artifact_registry_repository" "pegasusx" {
  count         = var.enable_gke ? 1 : 0
  location      = var.artifact_registry_location != "" ? var.artifact_registry_location : var.region
  repository_id = "${local.resource_prefix}-images"
  description   = "pegasusX backend-go and ai-worker container images"
  format        = "DOCKER"
  labels        = local.labels
  depends_on    = [google_project_service.gke_apis]
}

resource "google_container_cluster" "pegasusx" {
  count    = var.enable_gke ? 1 : 0
  name     = var.gke_cluster_name != "" ? var.gke_cluster_name : "${local.resource_prefix}-gke"
  location = var.region

  enable_autopilot = true
  networking_mode  = "VPC_NATIVE"
  # Auto-mode VPC has no pre-created secondary ranges named "pods"/"services".
  # Empty policy lets GKE allocate pod/service CIDRs (Autopilot-compatible).
  ip_allocation_policy {}
  network = google_compute_network.pegasusx_vpc.name
  # Auto-mode creates a regional subnet with the same name as the network.
  subnetwork = google_compute_network.pegasusx_vpc.name

  release_channel {
    channel = "REGULAR"
  }

  workload_identity_config {
    workload_pool = "${var.project_id}.svc.id.goog"
  }

  depends_on = [google_project_service.gke_apis]
}

resource "google_service_account" "backend_runtime" {
  count        = var.enable_gke ? 1 : 0
  account_id   = "${local.tenant_slug}-backend"
  display_name = "pegasusX backend runtime (${local.tenant_slug})"
}

resource "google_service_account_iam_member" "backend_wi" {
  count              = var.enable_gke ? 1 : 0
  service_account_id = google_service_account.backend_runtime[0].name
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:${var.project_id}.svc.id.goog[pegasusx/backend-go]"
  # Workload Identity pool is only available after the Autopilot cluster is fully ready.
  depends_on = [google_container_cluster.pegasusx]
}

resource "google_project_iam_member" "backend_spanner" {
  count   = var.enable_gke ? 1 : 0
  project = var.project_id
  role    = "roles/spanner.databaseUser"
  member  = "serviceAccount:${google_service_account.backend_runtime[0].email}"
}

resource "google_project_iam_member" "backend_redis" {
  count   = var.enable_gke ? 1 : 0
  project = var.project_id
  role    = "roles/redis.editor"
  member  = "serviceAccount:${google_service_account.backend_runtime[0].email}"
}

resource "google_project_iam_member" "backend_secrets" {
  count   = var.enable_gke ? 1 : 0
  project = var.project_id
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:${google_service_account.backend_runtime[0].email}"
}
