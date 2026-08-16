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
  # Auto-mode: empty secondary names let GKE allocate CIDRs.
  # Custom-mode: use the one regional subnet + pods/services secondary ranges.
  ip_allocation_policy {
    cluster_secondary_range_name  = var.vpc_custom_mode ? "pods" : null
    services_secondary_range_name = var.vpc_custom_mode ? "services" : null
  }
  network    = google_compute_network.pegasusx_vpc.name
  subnetwork = local.cell_subnet_name

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
  member             = "serviceAccount:${var.project_id}.svc.id.goog[${local.k8s_namespace}/backend-go]"
  # Workload Identity pool is only available after the Autopilot cluster is fully ready.
  depends_on = [google_container_cluster.pegasusx]
}

resource "google_project_iam_member" "backend_spanner" {
  count   = var.enable_gke && !var.cell_scoped_iam ? 1 : 0
  project = var.project_id
  role    = "roles/spanner.databaseUser"
  member  = "serviceAccount:${google_service_account.backend_runtime[0].email}"
}

resource "google_spanner_database_iam_member" "backend" {
  count    = var.enable_gke && var.cell_scoped_iam ? 1 : 0
  instance = google_spanner_instance.ledger.name
  database = google_spanner_database.main.name
  role     = "roles/spanner.databaseUser"
  member   = "serviceAccount:${google_service_account.backend_runtime[0].email}"
}

# Memorystore Redis (BASIC) has no instance IAM in this provider — leftover project role.
resource "google_project_iam_member" "backend_redis" {
  count   = var.enable_gke ? 1 : 0
  project = var.project_id
  role    = "roles/redis.editor"
  member  = "serviceAccount:${google_service_account.backend_runtime[0].email}"
}

resource "google_project_iam_member" "backend_secrets" {
  count   = var.enable_gke && !var.cell_scoped_iam ? 1 : 0
  project = var.project_id
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:${google_service_account.backend_runtime[0].email}"
}

locals {
  gsm_secret_ids = toset([
    google_secret_manager_secret.kafka_bootstrap_servers.secret_id,
    google_secret_manager_secret.kafka_topic_main.secret_id,
    google_secret_manager_secret.kafka_topic_spatial.secret_id,
    google_secret_manager_secret.kafka_topic_realtime.secret_id,
    google_secret_manager_secret.kafka_topic_webhooks.secret_id,
    google_secret_manager_secret.firebase_project_id.secret_id,
    google_secret_manager_secret.firebase_auth_enabled.secret_id,
    google_secret_manager_secret.jwt_secret.secret_id,
    google_secret_manager_secret.global_pay_webhook_secret.secret_id,
    google_secret_manager_secret.adyen_webhook_secret.secret_id,
    google_secret_manager_secret.stripe_webhook_secret.secret_id,
    google_secret_manager_secret.google_maps_api_key.secret_id,
    google_secret_manager_secret.internal_api_key.secret_id,
    google_secret_manager_secret.payme_webhook_secret.secret_id,
    google_secret_manager_secret.click_webhook_secret.secret_id,
    google_secret_manager_secret.global_pay_service_id.secret_id,
    google_secret_manager_secret.global_pay_username.secret_id,
    google_secret_manager_secret.global_pay_password.secret_id,
    google_secret_manager_secret.redis_auth.secret_id,
    google_secret_manager_secret.tauri_signing_private_key.secret_id,
    google_secret_manager_secret.tauri_updater_pubkey.secret_id,
    google_secret_manager_secret.windows_codesign_pfx.secret_id,
    google_secret_manager_secret.windows_codesign_password.secret_id,
    google_secret_manager_secret.apple_notarize_apple_id.secret_id,
    google_secret_manager_secret.apple_notarize_team_id.secret_id,
    google_secret_manager_secret.apple_notarize_app_password.secret_id,
    google_secret_manager_secret.maps_android_api_key.secret_id,
    google_secret_manager_secret.maps_ios_api_key.secret_id,
  ])
}

resource "google_secret_manager_secret_iam_member" "backend" {
  for_each  = var.enable_gke && var.cell_scoped_iam ? local.gsm_secret_ids : toset([])
  project   = var.project_id
  secret_id = each.value
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.backend_runtime[0].email}"
}
