# Google Maps Platform — world-scale geometry (Routes) + geocode/Places.
# Server key lives in google_secret_manager_secret.google_maps_api_key (main.tf).
# Client SDK keys are separate restricted keys (package/SHA or bundle ID).

locals {
  secret_maps_android_api_key = "${local.resource_prefix}-maps-android-api-key"
  secret_maps_ios_api_key     = "${local.resource_prefix}-maps-ios-api-key"
  maps_platform_apis = toset([
    "routes.googleapis.com",
    "geocoding-backend.googleapis.com",
    "places-backend.googleapis.com",
    "places.googleapis.com",
    "maps-backend.googleapis.com",
  ])
}

resource "google_project_service" "maps_platform_apis" {
  for_each           = local.maps_platform_apis
  service            = each.key
  disable_on_destroy = false
}

# Optional GSM shells for mobile Maps SDK keys (not injected into backend-go).
resource "google_secret_manager_secret" "maps_android_api_key" {
  secret_id = local.secret_maps_android_api_key
  replication {
    auto {}
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "maps_android_api_key" {
  count       = trimspace(var.maps_android_api_key) != "" ? 1 : 0
  secret      = google_secret_manager_secret.maps_android_api_key.id
  secret_data = var.maps_android_api_key
}

resource "google_secret_manager_secret" "maps_ios_api_key" {
  secret_id = local.secret_maps_ios_api_key
  replication {
    auto {}
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "maps_ios_api_key" {
  count       = trimspace(var.maps_ios_api_key) != "" ? 1 : 0
  secret      = google_secret_manager_secret.maps_ios_api_key.id
  secret_data = var.maps_ios_api_key
}
