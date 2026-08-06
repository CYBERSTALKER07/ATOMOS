variable "project_id" {
  description = "Google Cloud project ID for pegasusX workloads."
  type        = string
}

variable "tenant_slug" {
  description = "Client or sandbox slug used to namespace isolated SSMR resources."
  type        = string
  default     = "ssmr"
}

variable "resource_prefix" {
  description = "Explicit resource prefix override. When empty, terraform uses pegasusx-<tenant_slug>."
  type        = string
  default     = ""
}

variable "region" {
  description = "Primary deployment region."
  type        = string
  default     = "asia-south1"
}

variable "environment" {
  description = "Environment label for resource tagging."
  type        = string
  default     = "dev"
}

variable "vpc_name" {
  description = "VPC name used by backend workloads and Memorystore."
  type        = string
  default     = ""
}

variable "spanner_instance_name" {
  description = "Cloud Spanner instance name."
  type        = string
  default     = ""
}

variable "spanner_database_name" {
  description = "Cloud Spanner database name."
  type        = string
  default     = ""
}

variable "spanner_display_name" {
  description = "Cloud Spanner display name. When empty, derived from the tenant slug."
  type        = string
  default     = ""
}

variable "redis_instance_name" {
  description = "Memorystore Redis instance name."
  type        = string
  default     = ""
}

variable "spanner_processing_units_cap" {
  description = "Target Spanner PU cap for year-1 pilot. Enforced in GCP console / COST_GOVERNANCE_RUNBOOK until Spanner TF supports hard caps."
  type        = number
  default     = 100
}

variable "redis_memory_size_gb" {
  description = "Memorystore Redis memory size in GB."
  type        = number
  default     = 1
}

variable "redis_auth_enabled" {
  description = "Enable AUTH on Memorystore Redis."
  type        = bool
  default     = true
}

variable "redis_transit_encryption_mode" {
  description = "Enable TLS on Memorystore Redis."
  type        = string
  default     = "SERVER_AUTHENTICATION"
}

variable "kafka_bootstrap_servers" {
  description = "Kafka bootstrap servers for app env (stored in Secret Manager)."
  type        = string
  default     = ""
  sensitive   = true
}

variable "kafka_topic_main" {
  description = "Default Kafka topic used by backend-go outbox relay for order and state events."
  type        = string
  default     = "ssmr.events.orders"
}

variable "kafka_topic_spatial" {
  description = "Kafka topic reserved for spatial or H3 fanout workloads."
  type        = string
  default     = "ssmr.events.spatial"
}

variable "kafka_topic_realtime" {
  description = "Kafka topic reserved for realtime socket and fleet fanout."
  type        = string
  default     = "ssmr.events.realtime"
}

variable "kafka_topic_webhooks" {
  description = "Kafka topic reserved for outbound webhook delivery work."
  type        = string
  default     = "ssmr.events.webhooks"
}

variable "firebase_project_id" {
  description = "Firebase project id for ID token verification."
  type        = string
  default     = ""
}

variable "firebase_auth_enabled" {
  description = "Whether Firebase bearer token verification is enabled at runtime."
  type        = bool
  default     = false
}

variable "enable_observability_resources" {
  description = "Whether launch-readiness observability resources (dashboard, alerts, optional uptime checks) are provisioned."
  type        = bool
  default     = false
}

variable "alert_notification_channels" {
  description = "Notification channel ids used by alert policies when observability resources are enabled."
  type        = list(string)
  default     = []
}

variable "ai_worker_monitoring_host" {
  description = "Resolvable host for ai-worker monitoring uptime checks (for example an ingress hostname or load balancer IP). Leave empty to skip uptime checks."
  type        = string
  default     = ""
}

variable "ai_worker_monitoring_port" {
  description = "TCP port used for ai-worker health and readiness monitoring."
  type        = number
  default     = 8081
}

variable "ai_worker_monitoring_use_ssl" {
  description = "Whether the ai-worker monitoring host is exposed over HTTPS."
  type        = bool
  default     = false
}

variable "jwt_secret" {
  description = "JWT signing secret for backend-go (stored in Secret Manager)."
  type        = string
  default     = ""
  sensitive   = true
}

variable "global_pay_webhook_secret" {
  description = "GlobalPay webhook HMAC secret (stored in Secret Manager)."
  type        = string
  default     = ""
  sensitive   = true
}

variable "adyen_webhook_secret" {
  description = "Adyen webhook secret (stored in Secret Manager)."
  type        = string
  default     = ""
  sensitive   = true
}

variable "stripe_webhook_secret" {
  description = "Stripe webhook secret (stored in Secret Manager)."
  type        = string
  default     = ""
  sensitive   = true
}

variable "google_maps_api_key" {
  description = "Server-side Google Maps Platform API key (Geocoding, Places, Routes). Restrict to backend egress IPs / GKE NAT. Stored in Secret Manager."
  type        = string
  default     = ""
  sensitive   = true
}

variable "internal_api_key" {
  description = "Shared INTERNAL_API_KEY for optimizer-core / ai-worker (Secret Manager)."
  type        = string
  default     = ""
  sensitive   = true
}

variable "payme_webhook_secret" {
  description = "Payme webhook secret. Unused rails may use unused-rail-placeholder via phase0_sync."
  type        = string
  default     = ""
  sensitive   = true
}

variable "click_webhook_secret" {
  description = "Click webhook secret. Unused rails may use unused-rail-placeholder via phase0_sync."
  type        = string
  default     = ""
  sensitive   = true
}

variable "global_pay_service_id" {
  description = "Global Pay merchant service id (Secret Manager)."
  type        = string
  default     = ""
  sensitive   = true
}

variable "global_pay_username" {
  description = "Global Pay merchant username (Secret Manager)."
  type        = string
  default     = ""
  sensitive   = true
}

variable "global_pay_password" {
  description = "Global Pay merchant password (Secret Manager)."
  type        = string
  default     = ""
  sensitive   = true
}

variable "redis_auth" {
  description = "Memorystore AUTH string mirrored into GSM for ExternalSecret redis-password."
  type        = string
  default     = ""
  sensitive   = true
}

variable "maps_android_api_key" {
  description = "Optional Android Maps SDK key (package name + SHA-1 restricted). Not used by backend-go."
  type        = string
  default     = ""
  sensitive   = true
}

variable "maps_ios_api_key" {
  description = "Optional iOS Maps SDK key (bundle ID restricted). Not used by backend-go."
  type        = string
  default     = ""
  sensitive   = true
}

variable "tauri_signing_private_key" {
  description = "Minisign private key for Tauri desktop updater bundles (Secret Manager)."
  type        = string
  default     = ""
  sensitive   = true
}

variable "tauri_updater_pubkey" {
  description = "Minisign public key embedded in desktop tauri.conf.json (Secret Manager)."
  type        = string
  default     = ""
  sensitive   = true
}

variable "windows_codesign_pfx_b64" {
  description = "Base64-encoded Authenticode PFX for Windows desktop installers."
  type        = string
  default     = ""
  sensitive   = true
}

variable "windows_codesign_password" {
  description = "Password for windows_codesign_pfx_b64."
  type        = string
  default     = ""
  sensitive   = true
}

variable "apple_notarize_apple_id" {
  description = "Apple ID for notarytool desktop releases."
  type        = string
  default     = ""
  sensitive   = true
}

variable "apple_notarize_team_id" {
  description = "Apple Team ID for notarization."
  type        = string
  default     = ""
  sensitive   = true
}

variable "apple_notarize_app_password" {
  description = "App-specific password for notarytool."
  type        = string
  default     = ""
  sensitive   = true
}
