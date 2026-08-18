locals {
  cell_id                                = lower(trimspace(var.cell_id)) != "" ? lower(trimspace(var.cell_id)) : "uz"
  k8s_namespace                          = trimspace(var.k8s_namespace) != "" ? trimspace(var.k8s_namespace) : "pegasusx"
  tenant_slug                            = trimspace(var.tenant_slug) != "" ? lower(trimspace(var.tenant_slug)) : "ssmr"
  resource_prefix                        = trimspace(var.resource_prefix) != "" ? trimspace(var.resource_prefix) : "pegasusx-${local.tenant_slug}"
  vpc_name                               = trimspace(var.vpc_name) != "" ? trimspace(var.vpc_name) : "${local.resource_prefix}-vpc"
  kafka_topic_main                       = trimspace(var.kafka_topic_main) != "" ? trimspace(var.kafka_topic_main) : "cell-${local.cell_id}.events.orders"
  kafka_topic_spatial                    = trimspace(var.kafka_topic_spatial) != "" ? trimspace(var.kafka_topic_spatial) : "cell-${local.cell_id}.events.spatial"
  kafka_topic_realtime                   = trimspace(var.kafka_topic_realtime) != "" ? trimspace(var.kafka_topic_realtime) : "cell-${local.cell_id}.events.realtime"
  kafka_topic_webhooks                   = trimspace(var.kafka_topic_webhooks) != "" ? trimspace(var.kafka_topic_webhooks) : "cell-${local.cell_id}.events.webhooks"
  kafka_topic_main_dlq                   = trimspace(var.kafka_topic_main_dlq) != "" ? trimspace(var.kafka_topic_main_dlq) : "cell-${local.cell_id}.events.orders-dlq"
  kafka_topic_freeze_locks               = trimspace(var.kafka_topic_freeze_locks) != "" ? trimspace(var.kafka_topic_freeze_locks) : "cell-${local.cell_id}.events.freeze-locks"
  kafka_topic_inventory_import           = trimspace(var.kafka_topic_inventory_import) != "" ? trimspace(var.kafka_topic_inventory_import) : "cell-${local.cell_id}.events.inventory-import"
  spanner_instance_name                  = trimspace(var.spanner_instance_name) != "" ? trimspace(var.spanner_instance_name) : "${local.resource_prefix}-spanner"
  spanner_database_name                  = trimspace(var.spanner_database_name) != "" ? trimspace(var.spanner_database_name) : "${local.resource_prefix}-db"
  spanner_display_name                   = trimspace(var.spanner_display_name) != "" ? trimspace(var.spanner_display_name) : "${local.resource_prefix} ledger"
  redis_instance_name                    = trimspace(var.redis_instance_name) != "" ? trimspace(var.redis_instance_name) : "${local.resource_prefix}-redis"
  secret_kafka_bootstrap_servers         = "${local.resource_prefix}-kafka-bootstrap-servers"
  secret_kafka_topic_main                = "${local.resource_prefix}-kafka-topic-orders"
  secret_kafka_topic_spatial             = "${local.resource_prefix}-kafka-topic-spatial"
  secret_kafka_topic_realtime            = "${local.resource_prefix}-kafka-topic-realtime"
  secret_kafka_topic_webhooks            = "${local.resource_prefix}-kafka-topic-webhooks"
  secret_firebase_project_id             = "${local.resource_prefix}-firebase-project-id"
  secret_firebase_auth_enabled           = "${local.resource_prefix}-firebase-auth-enabled"
  secret_jwt                             = "${local.resource_prefix}-jwt-secret"
  secret_global_pay_webhook              = "${local.resource_prefix}-global-pay-webhook-secret"
  secret_adyen_webhook                   = "${local.resource_prefix}-adyen-webhook-secret"
  secret_stripe_webhook                  = "${local.resource_prefix}-stripe-webhook-secret"
  secret_google_maps_api_key             = "${local.resource_prefix}-google-maps-api-key"
  secret_internal_api_key                = "${local.resource_prefix}-internal-api-key"
  secret_payme_webhook                   = "${local.resource_prefix}-payme-webhook-secret"
  secret_click_webhook                   = "${local.resource_prefix}-click-webhook-secret"
  secret_global_pay_service_id           = "${local.resource_prefix}-global-pay-service-id"
  secret_global_pay_username             = "${local.resource_prefix}-global-pay-username"
  secret_global_pay_password             = "${local.resource_prefix}-global-pay-password"
  secret_redis_auth                      = "${local.resource_prefix}-redis-auth"
  secret_tauri_signing_private_key       = "${local.resource_prefix}-tauri-signing-private-key"
  secret_tauri_updater_pubkey            = "${local.resource_prefix}-tauri-updater-pubkey"
  secret_windows_codesign_pfx            = "${local.resource_prefix}-windows-codesign-pfx"
  secret_windows_codesign_password       = "${local.resource_prefix}-windows-codesign-password"
  secret_apple_notarize_apple_id         = "${local.resource_prefix}-apple-notarize-apple-id"
  secret_apple_notarize_team_id          = "${local.resource_prefix}-apple-notarize-team-id"
  secret_apple_notarize_app_password     = "${local.resource_prefix}-apple-notarize-app-password"
  secret_fiscal_my_soliq_base_url        = "${local.resource_prefix}-fiscal-my-soliq-base-url"
  secret_fiscal_my_soliq_api_key         = "${local.resource_prefix}-fiscal-my-soliq-api-key"
  secret_fiscal_my_soliq_tin             = "${local.resource_prefix}-fiscal-my-soliq-tin"
  secret_fiscal_my_soliq_signer          = "${local.resource_prefix}-fiscal-my-soliq-signer"
  secret_fiscal_my_soliq_pkcs12_password = "${local.resource_prefix}-fiscal-my-soliq-pkcs12-password"
  secret_playmobile_login                = "${local.resource_prefix}-playmobile-login"
  secret_playmobile_password             = "${local.resource_prefix}-playmobile-password"
  labels = {
    app         = "pegasusx"
    tenant      = local.tenant_slug
    cell        = local.cell_id
    environment = var.environment
    managed_by  = "terraform"
  }
}

resource "google_project_service" "required_apis" {
  for_each = toset([
    "compute.googleapis.com",
    "managedkafka.googleapis.com",
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
  auto_create_subnetworks = !var.vpc_custom_mode
  depends_on              = [google_project_service.required_apis]
}

# One regional subnet when vpc_custom_mode=true (C1). Live ssmr keeps auto-mode
# via staging.tfvars so an accidental apply does not ForceNew the VPC.
resource "google_compute_subnetwork" "cell" {
  count                    = var.vpc_custom_mode ? 1 : 0
  name                     = "${local.vpc_name}-subnet"
  ip_cidr_range            = var.vpc_subnet_cidr
  region                   = var.region
  network                  = google_compute_network.pegasusx_vpc.id
  private_ip_google_access = true

  secondary_ip_range {
    range_name    = "pods"
    ip_cidr_range = var.vpc_pods_cidr
  }
  secondary_ip_range {
    range_name    = "services"
    ip_cidr_range = var.vpc_services_cidr
  }
}

locals {
  cell_subnet_name = length(google_compute_subnetwork.cell) > 0 ? google_compute_subnetwork.cell[0].name : google_compute_network.pegasusx_vpc.name
}

resource "google_redis_instance" "cache" {
  name                    = local.redis_instance_name
  tier                    = "BASIC"
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
  labels = local.labels
}

resource "google_spanner_instance" "ledger" {
  name             = local.spanner_instance_name
  config           = "regional-${var.region}"
  display_name     = local.spanner_display_name
  processing_units = 100
  labels           = local.labels
  depends_on       = [google_project_service.required_apis]
}

resource "google_spanner_database" "main" {
  instance                 = google_spanner_instance.ledger.name
  name                     = local.spanner_database_name
  version_retention_period = var.spanner_version_retention_period
  enable_drop_protection   = true
}

# Kafka remains provider-agnostic (Confluent Cloud / managed Kafka / self-hosted).
# Store bootstrap servers in Secret Manager for runtime injection.
resource "google_secret_manager_secret" "kafka_bootstrap_servers" {
  secret_id = local.secret_kafka_bootstrap_servers
  replication {
    dynamic "auto" {
      for_each = var.gsm_regional_only ? [] : [1]
      content {}
    }
    dynamic "user_managed" {
      for_each = var.gsm_regional_only ? [1] : []
      content {
        replicas {
          location = var.region
        }
      }
    }
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
    dynamic "auto" {
      for_each = var.gsm_regional_only ? [] : [1]
      content {}
    }
    dynamic "user_managed" {
      for_each = var.gsm_regional_only ? [1] : []
      content {
        replicas {
          location = var.region
        }
      }
    }
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "kafka_topic_main" {
  secret      = google_secret_manager_secret.kafka_topic_main.id
  secret_data = local.kafka_topic_main
}

resource "google_secret_manager_secret" "kafka_topic_spatial" {
  secret_id = local.secret_kafka_topic_spatial
  replication {
    dynamic "auto" {
      for_each = var.gsm_regional_only ? [] : [1]
      content {}
    }
    dynamic "user_managed" {
      for_each = var.gsm_regional_only ? [1] : []
      content {
        replicas {
          location = var.region
        }
      }
    }
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "kafka_topic_spatial" {
  secret      = google_secret_manager_secret.kafka_topic_spatial.id
  secret_data = local.kafka_topic_spatial
}

resource "google_secret_manager_secret" "kafka_topic_realtime" {
  secret_id = local.secret_kafka_topic_realtime
  replication {
    dynamic "auto" {
      for_each = var.gsm_regional_only ? [] : [1]
      content {}
    }
    dynamic "user_managed" {
      for_each = var.gsm_regional_only ? [1] : []
      content {
        replicas {
          location = var.region
        }
      }
    }
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "kafka_topic_realtime" {
  secret      = google_secret_manager_secret.kafka_topic_realtime.id
  secret_data = local.kafka_topic_realtime
}

resource "google_secret_manager_secret" "kafka_topic_webhooks" {
  secret_id = local.secret_kafka_topic_webhooks
  replication {
    dynamic "auto" {
      for_each = var.gsm_regional_only ? [] : [1]
      content {}
    }
    dynamic "user_managed" {
      for_each = var.gsm_regional_only ? [1] : []
      content {
        replicas {
          location = var.region
        }
      }
    }
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "kafka_topic_webhooks" {
  secret      = google_secret_manager_secret.kafka_topic_webhooks.id
  secret_data = local.kafka_topic_webhooks
}

resource "google_secret_manager_secret" "firebase_project_id" {
  secret_id = local.secret_firebase_project_id
  replication {
    dynamic "auto" {
      for_each = var.gsm_regional_only ? [] : [1]
      content {}
    }
    dynamic "user_managed" {
      for_each = var.gsm_regional_only ? [1] : []
      content {
        replicas {
          location = var.region
        }
      }
    }
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
    dynamic "auto" {
      for_each = var.gsm_regional_only ? [] : [1]
      content {}
    }
    dynamic "user_managed" {
      for_each = var.gsm_regional_only ? [1] : []
      content {
        replicas {
          location = var.region
        }
      }
    }
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
    dynamic "auto" {
      for_each = var.gsm_regional_only ? [] : [1]
      content {}
    }
    dynamic "user_managed" {
      for_each = var.gsm_regional_only ? [1] : []
      content {
        replicas {
          location = var.region
        }
      }
    }
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
    dynamic "auto" {
      for_each = var.gsm_regional_only ? [] : [1]
      content {}
    }
    dynamic "user_managed" {
      for_each = var.gsm_regional_only ? [1] : []
      content {
        replicas {
          location = var.region
        }
      }
    }
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
    dynamic "auto" {
      for_each = var.gsm_regional_only ? [] : [1]
      content {}
    }
    dynamic "user_managed" {
      for_each = var.gsm_regional_only ? [1] : []
      content {
        replicas {
          location = var.region
        }
      }
    }
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
    dynamic "auto" {
      for_each = var.gsm_regional_only ? [] : [1]
      content {}
    }
    dynamic "user_managed" {
      for_each = var.gsm_regional_only ? [1] : []
      content {
        replicas {
          location = var.region
        }
      }
    }
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "stripe_webhook_secret" {
  count       = trimspace(var.stripe_webhook_secret) != "" ? 1 : 0
  secret      = google_secret_manager_secret.stripe_webhook_secret.id
  secret_data = var.stripe_webhook_secret
}

resource "google_secret_manager_secret" "google_maps_api_key" {
  secret_id = local.secret_google_maps_api_key
  replication {
    dynamic "auto" {
      for_each = var.gsm_regional_only ? [] : [1]
      content {}
    }
    dynamic "user_managed" {
      for_each = var.gsm_regional_only ? [1] : []
      content {
        replicas {
          location = var.region
        }
      }
    }
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "google_maps_api_key" {
  count       = trimspace(var.google_maps_api_key) != "" ? 1 : 0
  secret      = google_secret_manager_secret.google_maps_api_key.id
  secret_data = var.google_maps_api_key
}

# P0-8: shells for the remaining ExternalSecret remoteRefs (backend-go-secrets).
# Versions are conditional — phase0_sync / ops create stubs for unused PSP rails.

resource "google_secret_manager_secret" "internal_api_key" {
  secret_id = local.secret_internal_api_key
  replication {
    dynamic "auto" {
      for_each = var.gsm_regional_only ? [] : [1]
      content {}
    }
    dynamic "user_managed" {
      for_each = var.gsm_regional_only ? [1] : []
      content {
        replicas {
          location = var.region
        }
      }
    }
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "internal_api_key" {
  count       = trimspace(var.internal_api_key) != "" ? 1 : 0
  secret      = google_secret_manager_secret.internal_api_key.id
  secret_data = var.internal_api_key
}

resource "google_secret_manager_secret" "payme_webhook_secret" {
  secret_id = local.secret_payme_webhook
  replication {
    dynamic "auto" {
      for_each = var.gsm_regional_only ? [] : [1]
      content {}
    }
    dynamic "user_managed" {
      for_each = var.gsm_regional_only ? [1] : []
      content {
        replicas {
          location = var.region
        }
      }
    }
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "payme_webhook_secret" {
  count       = trimspace(var.payme_webhook_secret) != "" ? 1 : 0
  secret      = google_secret_manager_secret.payme_webhook_secret.id
  secret_data = var.payme_webhook_secret
}

resource "google_secret_manager_secret" "click_webhook_secret" {
  secret_id = local.secret_click_webhook
  replication {
    dynamic "auto" {
      for_each = var.gsm_regional_only ? [] : [1]
      content {}
    }
    dynamic "user_managed" {
      for_each = var.gsm_regional_only ? [1] : []
      content {
        replicas {
          location = var.region
        }
      }
    }
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "click_webhook_secret" {
  count       = trimspace(var.click_webhook_secret) != "" ? 1 : 0
  secret      = google_secret_manager_secret.click_webhook_secret.id
  secret_data = var.click_webhook_secret
}

resource "google_secret_manager_secret" "global_pay_service_id" {
  secret_id = local.secret_global_pay_service_id
  replication {
    dynamic "auto" {
      for_each = var.gsm_regional_only ? [] : [1]
      content {}
    }
    dynamic "user_managed" {
      for_each = var.gsm_regional_only ? [1] : []
      content {
        replicas {
          location = var.region
        }
      }
    }
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "global_pay_service_id" {
  count       = trimspace(var.global_pay_service_id) != "" ? 1 : 0
  secret      = google_secret_manager_secret.global_pay_service_id.id
  secret_data = var.global_pay_service_id
}

resource "google_secret_manager_secret" "global_pay_username" {
  secret_id = local.secret_global_pay_username
  replication {
    dynamic "auto" {
      for_each = var.gsm_regional_only ? [] : [1]
      content {}
    }
    dynamic "user_managed" {
      for_each = var.gsm_regional_only ? [1] : []
      content {
        replicas {
          location = var.region
        }
      }
    }
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "global_pay_username" {
  count       = trimspace(var.global_pay_username) != "" ? 1 : 0
  secret      = google_secret_manager_secret.global_pay_username.id
  secret_data = var.global_pay_username
}

resource "google_secret_manager_secret" "global_pay_password" {
  secret_id = local.secret_global_pay_password
  replication {
    dynamic "auto" {
      for_each = var.gsm_regional_only ? [] : [1]
      content {}
    }
    dynamic "user_managed" {
      for_each = var.gsm_regional_only ? [1] : []
      content {
        replicas {
          location = var.region
        }
      }
    }
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "global_pay_password" {
  count       = trimspace(var.global_pay_password) != "" ? 1 : 0
  secret      = google_secret_manager_secret.global_pay_password.id
  secret_data = var.global_pay_password
}

resource "google_secret_manager_secret" "redis_auth" {
  secret_id = local.secret_redis_auth
  replication {
    dynamic "auto" {
      for_each = var.gsm_regional_only ? [] : [1]
      content {}
    }
    dynamic "user_managed" {
      for_each = var.gsm_regional_only ? [1] : []
      content {
        replicas {
          location = var.region
        }
      }
    }
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "redis_auth" {
  count       = trimspace(var.redis_auth) != "" ? 1 : 0
  secret      = google_secret_manager_secret.redis_auth.id
  secret_data = var.redis_auth
}

resource "google_secret_manager_secret" "tauri_signing_private_key" {
  secret_id = local.secret_tauri_signing_private_key
  replication {
    dynamic "auto" {
      for_each = var.gsm_regional_only ? [] : [1]
      content {}
    }
    dynamic "user_managed" {
      for_each = var.gsm_regional_only ? [1] : []
      content {
        replicas {
          location = var.region
        }
      }
    }
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "tauri_signing_private_key" {
  count       = trimspace(var.tauri_signing_private_key) != "" ? 1 : 0
  secret      = google_secret_manager_secret.tauri_signing_private_key.id
  secret_data = var.tauri_signing_private_key
}

resource "google_secret_manager_secret" "tauri_updater_pubkey" {
  secret_id = local.secret_tauri_updater_pubkey
  replication {
    dynamic "auto" {
      for_each = var.gsm_regional_only ? [] : [1]
      content {}
    }
    dynamic "user_managed" {
      for_each = var.gsm_regional_only ? [1] : []
      content {
        replicas {
          location = var.region
        }
      }
    }
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "tauri_updater_pubkey" {
  count       = trimspace(var.tauri_updater_pubkey) != "" ? 1 : 0
  secret      = google_secret_manager_secret.tauri_updater_pubkey.id
  secret_data = var.tauri_updater_pubkey
}

resource "google_secret_manager_secret" "windows_codesign_pfx" {
  secret_id = local.secret_windows_codesign_pfx
  replication {
    dynamic "auto" {
      for_each = var.gsm_regional_only ? [] : [1]
      content {}
    }
    dynamic "user_managed" {
      for_each = var.gsm_regional_only ? [1] : []
      content {
        replicas {
          location = var.region
        }
      }
    }
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "windows_codesign_pfx" {
  count       = trimspace(var.windows_codesign_pfx_b64) != "" ? 1 : 0
  secret      = google_secret_manager_secret.windows_codesign_pfx.id
  secret_data = var.windows_codesign_pfx_b64
}

resource "google_secret_manager_secret" "windows_codesign_password" {
  secret_id = local.secret_windows_codesign_password
  replication {
    dynamic "auto" {
      for_each = var.gsm_regional_only ? [] : [1]
      content {}
    }
    dynamic "user_managed" {
      for_each = var.gsm_regional_only ? [1] : []
      content {
        replicas {
          location = var.region
        }
      }
    }
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "windows_codesign_password" {
  count       = trimspace(var.windows_codesign_password) != "" ? 1 : 0
  secret      = google_secret_manager_secret.windows_codesign_password.id
  secret_data = var.windows_codesign_password
}

resource "google_secret_manager_secret" "apple_notarize_apple_id" {
  secret_id = local.secret_apple_notarize_apple_id
  replication {
    dynamic "auto" {
      for_each = var.gsm_regional_only ? [] : [1]
      content {}
    }
    dynamic "user_managed" {
      for_each = var.gsm_regional_only ? [1] : []
      content {
        replicas {
          location = var.region
        }
      }
    }
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "apple_notarize_apple_id" {
  count       = trimspace(var.apple_notarize_apple_id) != "" ? 1 : 0
  secret      = google_secret_manager_secret.apple_notarize_apple_id.id
  secret_data = var.apple_notarize_apple_id
}

resource "google_secret_manager_secret" "apple_notarize_team_id" {
  secret_id = local.secret_apple_notarize_team_id
  replication {
    dynamic "auto" {
      for_each = var.gsm_regional_only ? [] : [1]
      content {}
    }
    dynamic "user_managed" {
      for_each = var.gsm_regional_only ? [1] : []
      content {
        replicas {
          location = var.region
        }
      }
    }
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "apple_notarize_team_id" {
  count       = trimspace(var.apple_notarize_team_id) != "" ? 1 : 0
  secret      = google_secret_manager_secret.apple_notarize_team_id.id
  secret_data = var.apple_notarize_team_id
}

resource "google_secret_manager_secret" "apple_notarize_app_password" {
  secret_id = local.secret_apple_notarize_app_password
  replication {
    dynamic "auto" {
      for_each = var.gsm_regional_only ? [] : [1]
      content {}
    }
    dynamic "user_managed" {
      for_each = var.gsm_regional_only ? [1] : []
      content {
        replicas {
          location = var.region
        }
      }
    }
  }
  labels = local.labels
}

resource "google_secret_manager_secret_version" "apple_notarize_app_password" {
  count       = trimspace(var.apple_notarize_app_password) != "" ? 1 : 0
  secret      = google_secret_manager_secret.apple_notarize_app_password.id
  secret_data = var.apple_notarize_app_password
}

# ------------------------------------------------------------------------------
# App Updates Storage Bucket (for Tauri OTA / Direct Website Distribution)
# ------------------------------------------------------------------------------

resource "google_storage_bucket" "app_updates" {
  # GCS names are global — include project_id so multi-account cutovers do not 409.
  name          = "${var.project_id}-${local.resource_prefix}-app-updates"
  location      = var.region
  force_destroy = false

  uniform_bucket_level_access = true
  public_access_prevention    = "inherited"

  cors {
    origin          = ["*"]
    method          = ["GET", "HEAD", "OPTIONS"]
    response_header = ["*"]
    max_age_seconds = 3600
  }

  labels = local.labels
}

# Make the updates bucket publicly readable so apps can download the binaries without auth
# resource "google_storage_bucket_iam_member" "public_updates" {
#   bucket = google_storage_bucket.app_updates.name
#   role   = "roles/storage.objectViewer"
#   member = "allUsers"
# }
