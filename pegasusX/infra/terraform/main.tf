locals {
  labels = {
    app         = "pegasusx"
    environment = var.environment
    managed_by  = "terraform"
  }
}

resource "google_project_service" "required_apis" {
  for_each = toset([
    "compute.googleapis.com",
    "redis.googleapis.com",
    "secretmanager.googleapis.com",
    "spanner.googleapis.com"
  ])
  service            = each.key
  disable_on_destroy = false
}

resource "google_compute_network" "pegasusx_vpc" {
  name                    = var.vpc_name
  auto_create_subnetworks = true
  depends_on              = [google_project_service.required_apis]
}

resource "google_redis_instance" "cache" {
  name               = var.redis_instance_name
  tier               = "STANDARD_HA"
  memory_size_gb     = var.redis_memory_size_gb
  region             = var.region
  redis_version      = "REDIS_7_0"
  authorized_network = google_compute_network.pegasusx_vpc.id
  labels             = local.labels
}

resource "google_spanner_instance" "ledger" {
  name         = var.spanner_instance_name
  config       = "regional-${var.region}"
  display_name = "pegasusX Ledger"
  num_nodes    = 1
  labels       = local.labels
  depends_on   = [google_project_service.required_apis]
}

resource "google_spanner_database" "main" {
  instance = google_spanner_instance.ledger.name
  name     = var.spanner_database_name
}

# Kafka remains provider-agnostic (Confluent Cloud / managed Kafka / self-hosted).
# Store bootstrap servers in Secret Manager for runtime injection.
resource "google_secret_manager_secret" "kafka_bootstrap_servers" {
  secret_id = "pegasusx-kafka-bootstrap-servers"
  replication {
    auto {}
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "kafka_bootstrap_servers" {
  count       = trimspace(var.kafka_bootstrap_servers) != "" ? 1 : 0
  secret      = google_secret_manager_secret.kafka_bootstrap_servers.id
  secret_data = var.kafka_bootstrap_servers
}

resource "google_secret_manager_secret" "kafka_topic_main" {
  secret_id = "pegasusx-kafka-topic-main"
  replication {
    auto {}
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "kafka_topic_main" {
  secret      = google_secret_manager_secret.kafka_topic_main.id
  secret_data = var.kafka_topic_main
}

resource "google_secret_manager_secret" "firebase_project_id" {
  secret_id = "pegasusx-firebase-project-id"
  replication {
    auto {}
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "firebase_project_id" {
  count       = trimspace(var.firebase_project_id) != "" ? 1 : 0
  secret      = google_secret_manager_secret.firebase_project_id.id
  secret_data = var.firebase_project_id
}

resource "google_secret_manager_secret" "firebase_auth_enabled" {
  secret_id = "pegasusx-firebase-auth-enabled"
  replication {
    auto {}
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "firebase_auth_enabled" {
  secret      = google_secret_manager_secret.firebase_auth_enabled.id
  secret_data = tostring(var.firebase_auth_enabled)
}
