/**
 * Module: Database
 * Cloud Spanner Multi-Zone Regional Instance, Database, Automated Daily Backup Schedule,
 * and Cloud Memorystore for Redis 7.0 HA Cluster with In-Transit TLS & AUTH.
 */

# 1. Cloud Spanner Instance
resource "google_spanner_instance" "ledger" {
  name         = var.spanner_instance_name
  config       = var.spanner_config
  display_name = "PegasusX FMCG Multi-Tenant Ledger (${var.environment})"
  project      = var.project_id

  # Processing units or Autoscaling configuration
  processing_units = var.spanner_autoscaling_config == null ? var.spanner_processing_units : null

  dynamic "autoscaling_config" {
    for_each = var.spanner_autoscaling_config != null ? [var.spanner_autoscaling_config] : []
    content {
      autoscaling_limits {
        min_processing_units = autoscaling_config.value.min_processing_units
        max_processing_units = autoscaling_config.value.max_processing_units
      }
      autoscaling_targets {
        high_priority_cpu_utilization_percent = autoscaling_config.value.high_priority_cpu_utilization_percent
        storage_utilization_percent           = autoscaling_config.value.storage_utilization_percent
      }
    }
  }

  labels = var.labels

  lifecycle {
    prevent_destroy = false
  }
}

# 2. Cloud Spanner Database (136 Tables, 13 Interleaved Hierarchies, 193 Indexes)
resource "google_spanner_database" "main" {
  instance                 = google_spanner_instance.ledger.name
  name                     = var.spanner_database_name
  project                  = var.project_id
  version_retention_period = "7d"
  deletion_protection      = var.deletion_protection

  ddl = var.spanner_ddl_statements

  lifecycle {
    ignore_changes = [
      ddl
    ]
  }
}

# 3. Automated Daily Full Backup Schedule for Cloud Spanner (30-day retention)
resource "google_spanner_backup_schedule" "daily_full" {
  instance           = google_spanner_instance.ledger.name
  database           = google_spanner_database.main.name
  name               = "${var.spanner_database_name}-daily-backup-schedule"
  project            = var.project_id
  retention_duration = "${var.spanner_backup_retention_days * 86400}s"

  spec {
    cron_spec {
      text = var.spanner_backup_cron_spec
    }
  }

  full_backup_spec {}
}

# 4. Cloud Memorystore for Redis 7.0 HA Instance (Standard HA Tier with Cross-Zone Replica)
resource "google_redis_instance" "cache" {
  name                    = var.redis_instance_name
  tier                    = var.redis_tier
  memory_size_gb          = var.redis_memory_size_gb
  region                  = var.region
  project                 = var.project_id
  authorized_network      = var.vpc_id
  connect_mode            = "PRIVATE_SERVICE_ACCESS"
  redis_version           = var.redis_version
  display_name            = "PegasusX Redis 7.0 HA Cache & PubSub Mesh"
  transit_encryption_mode = "SERVER_AUTHENTICATION"
  auth_enabled            = true

  maintenance_policy {
    weekly_maintenance_window {
      day = "SUNDAY"
      start_time {
        hours   = 2
        minutes = 0
        seconds = 0
        nanos   = 0
      }
    }
  }

  labels = var.labels

  # Explicit dependency on PSA connection so peering is active before Redis creation
  depends_on = [
    var.psa_connection
  ]
}
