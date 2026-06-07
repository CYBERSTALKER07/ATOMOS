locals {
  tenant_slug                    = trimspace(var.tenant_slug) != "" ? lower(trimspace(var.tenant_slug)) : "ssmr"
  resource_prefix                = trimspace(var.resource_prefix) != "" ? trimspace(var.resource_prefix) : "pegasusx-${local.tenant_slug}"
  vpc_name                       = trimspace(var.vpc_name) != "" ? trimspace(var.vpc_name) : "${local.resource_prefix}-vpc"
  spanner_instance_name          = trimspace(var.spanner_instance_name) != "" ? trimspace(var.spanner_instance_name) : "${local.resource_prefix}-spanner"
  spanner_database_name          = trimspace(var.spanner_database_name) != "" ? trimspace(var.spanner_database_name) : "${local.resource_prefix}-db"
  spanner_display_name           = trimspace(var.spanner_display_name) != "" ? trimspace(var.spanner_display_name) : "${local.resource_prefix} ledger"
  redis_instance_name            = trimspace(var.redis_instance_name) != "" ? trimspace(var.redis_instance_name) : "${local.resource_prefix}-redis"
  secret_kafka_bootstrap_servers = "${local.resource_prefix}-kafka-bootstrap-servers"
  secret_kafka_topic_main        = "${local.resource_prefix}-kafka-topic-orders"
  secret_kafka_topic_spatial     = "${local.resource_prefix}-kafka-topic-spatial"
  secret_kafka_topic_realtime    = "${local.resource_prefix}-kafka-topic-realtime"
  secret_kafka_topic_webhooks    = "${local.resource_prefix}-kafka-topic-webhooks"
  secret_firebase_project_id     = "${local.resource_prefix}-firebase-project-id"
  secret_firebase_auth_enabled   = "${local.resource_prefix}-firebase-auth-enabled"
  secret_jwt                     = "${local.resource_prefix}-jwt-secret"
  secret_global_pay_webhook      = "${local.resource_prefix}-global-pay-webhook-secret"
  secret_adyen_webhook           = "${local.resource_prefix}-adyen-webhook-secret"
  secret_stripe_webhook          = "${local.resource_prefix}-stripe-webhook-secret"
  labels = {
    app         = "pegasusx"
    tenant      = local.tenant_slug
    environment = var.environment
    managed_by  = "terraform"
  }
}

resource "google_project_service" "required_apis" {
  for_each = toset([
    "compute.googleapis.com",
    "monitoring.googleapis.com",
    "redis.googleapis.com",
    "secretmanager.googleapis.com",
    "spanner.googleapis.com"
  ])
  service            = each.key
  disable_on_destroy = false
}

resource "google_compute_network" "pegasusx_vpc" {
  name                    = local.vpc_name
  auto_create_subnetworks = true
  depends_on              = [google_project_service.required_apis]
}

resource "google_redis_instance" "cache" {
  name                    = local.redis_instance_name
  tier                    = "STANDARD_HA"
  memory_size_gb          = var.redis_memory_size_gb
  region                  = var.region
  redis_version           = "REDIS_7_0"
  authorized_network      = google_compute_network.pegasusx_vpc.id
  auth_enabled            = var.redis_auth_enabled
  transit_encryption_mode = var.redis_transit_encryption_mode
  redis_configs = {
    maxmemory-policy       = "allkeys-lru"
    notify-keyspace-events = ""
  }
  labels                  = local.labels
}

resource "google_spanner_instance" "ledger" {
  name         = local.spanner_instance_name
  config       = "regional-${var.region}"
  display_name = local.spanner_display_name
  num_nodes    = 1
  labels       = local.labels
  depends_on   = [google_project_service.required_apis]
}

resource "google_spanner_database" "main" {
  instance = google_spanner_instance.ledger.name
  name     = local.spanner_database_name
}

# Kafka remains provider-agnostic (Confluent Cloud / managed Kafka / self-hosted).
# Store bootstrap servers in Secret Manager for runtime injection.
resource "google_secret_manager_secret" "kafka_bootstrap_servers" {
  secret_id = local.secret_kafka_bootstrap_servers
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
  secret_id = local.secret_kafka_topic_main
  replication {
    auto {}
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "kafka_topic_main" {
  secret      = google_secret_manager_secret.kafka_topic_main.id
  secret_data = var.kafka_topic_main
}

resource "google_secret_manager_secret" "kafka_topic_spatial" {
  secret_id = local.secret_kafka_topic_spatial
  replication {
    auto {}
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "kafka_topic_spatial" {
  secret      = google_secret_manager_secret.kafka_topic_spatial.id
  secret_data = var.kafka_topic_spatial
}

resource "google_secret_manager_secret" "kafka_topic_realtime" {
  secret_id = local.secret_kafka_topic_realtime
  replication {
    auto {}
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "kafka_topic_realtime" {
  secret      = google_secret_manager_secret.kafka_topic_realtime.id
  secret_data = var.kafka_topic_realtime
}

resource "google_secret_manager_secret" "kafka_topic_webhooks" {
  secret_id = local.secret_kafka_topic_webhooks
  replication {
    auto {}
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "kafka_topic_webhooks" {
  secret      = google_secret_manager_secret.kafka_topic_webhooks.id
  secret_data = var.kafka_topic_webhooks
}

resource "google_secret_manager_secret" "firebase_project_id" {
  secret_id = local.secret_firebase_project_id
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
  secret_id = local.secret_firebase_auth_enabled
  replication {
    auto {}
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "firebase_auth_enabled" {
  secret      = google_secret_manager_secret.firebase_auth_enabled.id
  secret_data = tostring(var.firebase_auth_enabled)
}

resource "google_secret_manager_secret" "jwt_secret" {
  secret_id = local.secret_jwt
  replication {
    auto {}
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "jwt_secret" {
  count       = trimspace(var.jwt_secret) != "" ? 1 : 0
  secret      = google_secret_manager_secret.jwt_secret.id
  secret_data = var.jwt_secret
}

resource "google_secret_manager_secret" "global_pay_webhook_secret" {
  secret_id = local.secret_global_pay_webhook
  replication {
    auto {}
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "global_pay_webhook_secret" {
  count       = trimspace(var.global_pay_webhook_secret) != "" ? 1 : 0
  secret      = google_secret_manager_secret.global_pay_webhook_secret.id
  secret_data = var.global_pay_webhook_secret
}

resource "google_secret_manager_secret" "adyen_webhook_secret" {
  secret_id = local.secret_adyen_webhook
  replication {
    auto {}
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "adyen_webhook_secret" {
  count       = trimspace(var.adyen_webhook_secret) != "" ? 1 : 0
  secret      = google_secret_manager_secret.adyen_webhook_secret.id
  secret_data = var.adyen_webhook_secret
}

resource "google_secret_manager_secret" "stripe_webhook_secret" {
  secret_id = local.secret_stripe_webhook
  replication {
    auto {}
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "stripe_webhook_secret" {
  count       = trimspace(var.stripe_webhook_secret) != "" ? 1 : 0
  secret      = google_secret_manager_secret.stripe_webhook_secret.id
  secret_data = var.stripe_webhook_secret
}
