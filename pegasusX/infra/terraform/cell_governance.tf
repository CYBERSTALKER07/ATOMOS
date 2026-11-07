/**
 * Multi-Cell Isolation Governance & Terraform 1.5+ Check Assertions (GS-C1 to GS-C4)
 *
 * Enforces strict multi-cell isolation invariants at plan/validate time:
 * - Regional data residency (EU cell in europe-west1 cannot share UZ ssmr state or auto-replicated GSM)
 * - Project boundary isolation (non-UZ cells cannot target live UZ projects)
 * - Schema migration purity (non-UZ cells cannot restore UZ Spanner backups)
 * - Least-privilege IAM scoping (non-UZ cells must enforce database- and secret-scoped IAM)
 */

variable "cell_id" {
  description = "Logical cell identifier (e.g. uz, eu). One cell = one GCP project + region + isolated state prefix."
  type        = string
  default     = "uz"
  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{1,15}$", var.cell_id))
    error_message = "cell_id must start with a lowercase letter and contain only lowercase alphanumeric characters or hyphens."
  }
}

variable "tenant_slug" {
  description = "Tenant or deployment namespace slug for cell-local resource naming."
  type        = string
  default     = "prod"
}

variable "terraform_state_prefix" {
  description = "GCS backend state prefix for the cell (must match -backend-config prefix; non-uz cells must not use pegasusx/ssmr)."
  type        = string
  default     = ""
}

variable "gsm_regional_only" {
  description = "Enforce Secret Manager user-managed replication strictly in var.region (required for EU/non-UZ data residency)."
  type        = bool
  default     = false
}

variable "vpc_custom_mode" {
  description = "Provision custom-mode VPC with dedicated regional subnet (required for new cells)."
  type        = bool
  default     = true
}

variable "vpc_subnet_cidr" {
  description = "Primary CIDR override for the cell subnet when set via cell.tfvars."
  type        = string
  default     = ""
}

variable "vpc_pods_cidr" {
  description = "Secondary CIDR override for GKE pods when set via cell.tfvars."
  type        = string
  default     = ""
}

variable "vpc_services_cidr" {
  description = "Secondary CIDR override for GKE services when set via cell.tfvars."
  type        = string
  default     = ""
}

variable "cell_scoped_iam" {
  description = "Scope Workload Identity GSA permissions to the cell Spanner database and individual GSM secrets instead of project-wide."
  type        = bool
  default     = true
}

variable "allow_uz_backup_restore" {
  description = "Whether operators may restore a UZ Spanner backup into this cell. Must remain false for every non-UZ cell."
  type        = bool
  default     = false
}

variable "enable_gke" {
  description = "Whether GKE compute cluster provisioning is enabled for this cell."
  type        = bool
  default     = true
}

variable "enable_managed_kafka" {
  description = "Whether Google Managed Service for Apache Kafka is enabled for this cell."
  type        = bool
  default     = true
}

variable "kafka_bootstrap_servers" {
  description = "Optional external Kafka bootstrap servers override."
  type        = string
  default     = ""
}

variable "kafka_topic_main" {
  description = "Optional override for the primary orders Kafka topic name."
  type        = string
  default     = ""
}

variable "kafka_topic_main_dlq" {
  description = "Optional override for the primary orders DLQ Kafka topic name."
  type        = string
  default     = ""
}

variable "kafka_topic_spatial" {
  description = "Optional override for the spatial telemetry Kafka topic name."
  type        = string
  default     = ""
}

variable "kafka_topic_realtime" {
  description = "Optional override for the realtime WebSocket event Kafka topic name."
  type        = string
  default     = ""
}

variable "kafka_topic_webhooks" {
  description = "Optional override for the partner webhooks Kafka topic name."
  type        = string
  default     = ""
}

variable "kafka_topic_freeze_locks" {
  description = "Optional override for the tenant freeze locks Kafka topic name."
  type        = string
  default     = ""
}

variable "kafka_topic_inventory_import" {
  description = "Optional override for the inventory import Kafka topic name."
  type        = string
  default     = ""
}

variable "jwt_secret" {
  description = "Optional initial JWT signing secret seed (sensitive; prefer GSM sync in production)."
  type        = string
  default     = ""
  sensitive   = true
}

variable "global_pay_webhook_secret" {
  description = "Optional Global Pay webhook secret seed (sensitive)."
  type        = string
  default     = ""
  sensitive   = true
}

variable "firebase_project_id" {
  description = "Optional Firebase project ID for mobile/web token verification."
  type        = string
  default     = ""
}

variable "firebase_auth_enabled" {
  description = "Whether Firebase Authentication token verification is enabled."
  type        = bool
  default     = false
}

variable "google_maps_api_key" {
  description = "Optional Google Maps Platform API key seed (sensitive)."
  type        = string
  default     = ""
  sensitive   = true
}

variable "enable_observability_resources" {
  description = "Whether Cloud Monitoring alert policies, uptime checks, and dashboards are enabled."
  type        = bool
  default     = true
}

variable "alert_notification_channels" {
  description = "Optional list of existing Cloud Monitoring notification channel IDs."
  type        = list(string)
  default     = []
}

variable "ai_worker_monitoring_host" {
  description = "Optional hostname for ai-worker synthetic uptime check."
  type        = string
  default     = ""
}

variable "budget_alert_emails" {
  description = "Optional list of billing budget notification email addresses."
  type        = list(string)
  default     = []
}

locals {
  effective_cell_id            = trimspace(var.cell_id) != "" ? trimspace(var.cell_id) : "uz"
  effective_primary_cidr_block = trimspace(var.vpc_subnet_cidr) != "" ? trimspace(var.vpc_subnet_cidr) : var.primary_cidr_block
  effective_pod_cidr_block     = trimspace(var.vpc_pods_cidr) != "" ? trimspace(var.vpc_pods_cidr) : var.pod_cidr_block
  effective_service_cidr_block = trimspace(var.vpc_services_cidr) != "" ? trimspace(var.vpc_services_cidr) : var.service_cidr_block
}

check "europe_west1_cannot_use_ssmr_state" {
  assert {
    condition = var.region != "europe-west1" || (
      trimspace(var.terraform_state_prefix) != "" &&
      trimspace(var.terraform_state_prefix) != "pegasusx/ssmr"
    )
    error_message = "GS-C1: region europe-west1 must not use terraform_state_prefix pegasusx/ssmr."
  }
}

check "non_uz_cell_cannot_use_ssmr_state" {
  assert {
    condition = local.effective_cell_id == "uz" || (
      trimspace(var.terraform_state_prefix) != "" &&
      trimspace(var.terraform_state_prefix) != "pegasusx/ssmr"
    )
    error_message = "GS-C1: non-uz cell_id must not share pegasusx/ssmr state."
  }
}

check "europe_west1_gsm_must_be_regional" {
  assert {
    condition     = var.region != "europe-west1" || var.gsm_regional_only
    error_message = "GS-C1: europe-west1 must set gsm_regional_only=true (no auto {} GSM)."
  }
}

check "europe_west1_vpc_must_be_custom" {
  assert {
    condition     = var.region != "europe-west1" || var.vpc_custom_mode
    error_message = "GS-C1: europe-west1 must set vpc_custom_mode=true (one regional subnet)."
  }
}

check "non_uz_cell_not_in_live_project" {
  assert {
    condition = local.effective_cell_id == "uz" || (
      var.project_id != "pegasusx-prod-uz" &&
      var.project_id != "pegasus-503013"
    )
    error_message = "GS-C2: a non-uz cell must not share project pegasusx-prod-uz or pegasus-503013."
  }
}

check "non_uz_forbids_uz_restore" {
  assert {
    condition     = local.effective_cell_id == "uz" || !var.allow_uz_backup_restore
    error_message = "GS-C3: a non-uz cell must not restore a UZ Spanner backup. Apply the same DDL (migrations) instead."
  }
}

check "non_uz_requires_cell_scoped_iam" {
  assert {
    condition     = local.effective_cell_id == "uz" || var.cell_scoped_iam
    error_message = "GS-C4: a non-uz cell must use cell_scoped_iam so the EU GSA cannot be a project-wide UZ Spanner/GSM user."
  }
}
