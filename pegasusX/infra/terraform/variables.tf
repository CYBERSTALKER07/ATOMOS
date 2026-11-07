/**
 * Root Terraform Variable Declarations
 */

variable "project_id" {
  description = "The Google Cloud Platform project ID to deploy infrastructure into."
  type        = string
  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{4,28}[a-z0-9]$", var.project_id))
    error_message = "The project_id must be between 6 and 30 characters, start with a letter, and contain only lowercase alphanumeric characters or hyphens."
  }
}

variable "region" {
  description = "The primary Google Cloud region for regional services."
  type        = string
  default     = "europe-west3"
}

variable "zone" {
  description = "The primary Google Cloud availability zone."
  type        = string
  default     = "europe-west3-a"
}

variable "environment" {
  description = "The deployment environment tier."
  type        = string
  default     = "production"
  validation {
    condition     = contains(["production", "staging", "development", "dev", "ssmr", "cell"], var.environment)
    error_message = "The environment variable must be one of: 'production', 'staging', 'development', 'dev', 'ssmr', or 'cell'."
  }
}

# Networking
variable "network_name" {
  description = "Name of the VPC network."
  type        = string
  default     = "pegasusx-vpc"
}

variable "subnet_name" {
  description = "Name of the primary subnetwork."
  type        = string
  default     = "pegasusx-primary-subnet"
}

variable "primary_cidr_block" {
  description = "CIDR range for primary subnetwork (nodes)."
  type        = string
  default     = "10.10.0.0/20"
}

variable "pod_cidr_block" {
  description = "Secondary CIDR range for GKE Pods."
  type        = string
  default     = "10.20.0.0/16"
}

variable "service_cidr_block" {
  description = "Secondary CIDR range for GKE Services."
  type        = string
  default     = "10.30.0.0/20"
}

variable "vpc_flow_log_interval" {
  description = "Aggregation interval for VPC subnet flow logs (FinOps tuned default: INTERVAL_30_SEC)."
  type        = string
  default     = "INTERVAL_30_SEC"
}

variable "vpc_flow_log_sampling" {
  description = "Sampling rate (0.0 to 1.0) for VPC subnet flow logs (FinOps tuned default: 0.1)."
  type        = number
  default     = 0.1
}

# Compute / GKE
variable "cluster_name" {
  description = "Name of the GKE cluster."
  type        = string
  default     = "pegasusx-prod-gke"
}

variable "enable_autopilot" {
  description = "Whether to use GKE Autopilot mode."
  type        = bool
  default     = true
}

variable "master_ipv4_cidr_block" {
  description = "CIDR block for GKE master control plane."
  type        = string
  default     = "172.16.0.0/28"
}

variable "enable_spot_node_pool" {
  description = "Whether to provision a Spot VM node pool in GKE Standard mode for batch/cron/stateless workloads (up to 91% discount)."
  type        = bool
  default     = true
}

variable "spot_node_machine_type" {
  description = "Machine type for the GKE Spot VM node pool when enable_autopilot = false."
  type        = string
  default     = "e2-standard-4"
}

variable "spot_node_min_count" {
  description = "Minimum node count per zone for the Spot VM pool (0 enables scale-to-zero when idle)."
  type        = number
  default     = 0
}

variable "spot_node_max_count" {
  description = "Maximum node count per zone for the Spot VM pool."
  type        = number
  default     = 8
}

variable "enable_committed_use_discount" {
  description = "Whether to provision a GCP Compute Engine regional Committed Use Discount (CUD) for baseline vCPU/memory."
  type        = bool
  default     = false
}

variable "cud_plan" {
  description = "Commitment plan duration for Compute Engine CUD (TWELVE_MONTH for 37% savings or THIRTY_SIX_MONTH for 55% savings)."
  type        = string
  default     = "TWELVE_MONTH"
  validation {
    condition     = contains(["TWELVE_MONTH", "THIRTY_SIX_MONTH"], var.cud_plan)
    error_message = "cud_plan must be either TWELVE_MONTH or THIRTY_SIX_MONTH."
  }
}

variable "cud_vcpu_count" {
  description = "Baseline number of vCPUs to commit under Compute Engine CUD."
  type        = number
  default     = 8
}

variable "cud_memory_mb" {
  description = "Baseline memory in MiB to commit under Compute Engine CUD."
  type        = number
  default     = 32768
}

# Database & Cache
variable "spanner_instance_name" {
  description = "Name of the Cloud Spanner instance."
  type        = string
  default     = "pegasusx-ssmr-spanner"
}

variable "spanner_config" {
  description = "Instance configuration for Cloud Spanner."
  type        = string
  default     = "regional-europe-west3"
}

variable "spanner_processing_units" {
  description = "Processing units allocated to Cloud Spanner when autoscaling is disabled."
  type        = number
  default     = 100
}

variable "spanner_autoscaling_config" {
  description = "Autoscaling configuration for Cloud Spanner (scales between min and max PUs based on CPU/storage)."
  type = object({
    min_processing_units                  = number
    max_processing_units                  = number
    high_priority_cpu_utilization_percent = number
    storage_utilization_percent           = number
  })
  default = {
    min_processing_units                  = 100
    max_processing_units                  = 400
    high_priority_cpu_utilization_percent = 65
    storage_utilization_percent           = 80
  }
}

variable "spanner_database_name" {
  description = "Name of the Cloud Spanner database."
  type        = string
  default     = "main"
}

variable "spanner_backup_retention_days" {
  description = "Retention period in days for Spanner daily backups."
  type        = number
  default     = 30
}

variable "redis_instance_name" {
  description = "Name of the Cloud Memorystore for Redis instance."
  type        = string
  default     = "pegasusx-ssmr-redis"
}

variable "redis_tier" {
  description = "Service tier for Cloud Memorystore Redis (STANDARD_HA for production, BASIC for dev/staging cost savings)."
  type        = string
  default     = "STANDARD_HA"
}

variable "redis_memory_size_gb" {
  description = "Redis cache memory capacity in GiB."
  type        = number
  default     = 5
}

# Messaging / Kafka
variable "kafka_cluster_id" {
  description = "Identifier for the Google Managed Service for Apache Kafka cluster."
  type        = string
  default     = "pegasusx-events-cluster"
}

variable "kafka_vcpu_count" {
  description = "Number of vCPUs per Kafka broker."
  type        = number
  default     = 3
}

variable "kafka_memory_bytes" {
  description = "Memory capacity in bytes per Kafka broker."
  type        = number
  default     = 17179869184 # 16 GiB
}

# Storage & Security
variable "media_bucket_name" {
  description = "Name of the GCS bucket for proof-of-delivery and evidence dossiers."
  type        = string
  default     = "pegasusx-prod-media"
}

variable "updates_bucket_name" {
  description = "Name of the GCS bucket for mobile APKs and desktop OTA updates."
  type        = string
  default     = "pegasusx-prod-app-updates"
}

variable "imports_bucket_name" {
  description = "Name of the GCS bucket for supplier CSV/XLSX bulk imports and exports."
  type        = string
  default     = "pegasusx-prod-imports-exports"
}

variable "tf_state_bucket_name" {
  description = "Name of the GCS bucket for Terraform state."
  type        = string
  default     = "pegasusx-terraform-state"
}

variable "k8s_namespace" {
  description = "Kubernetes namespace for production workloads."
  type        = string
  default     = "pegasusx"
}

# Monitoring
variable "alert_email_endpoints" {
  description = "List of recipient email addresses for production SRE alerts."
  type        = list(string)
  default     = ["sre-alerts@pegasusx.io"]
}

variable "slack_webhook_url" {
  description = "Optional Slack webhook URL for real-time alert notifications."
  type        = string
  default     = ""
}

variable "api_hostname" {
  description = "Public API hostname for synthetic uptime checks."
  type        = string
  default     = "api.pegasusx.io"
}

variable "deletion_protection" {
  description = "Whether deletion protection is enabled on critical stateful resources (Spanner)."
  type        = bool
  default     = true
}

variable "labels" {
  description = "Common FinOps cost-allocation labels applied to all infrastructure resources."
  type        = map(string)
  default = {
    managed_by      = "terraform"
    system          = "pegasusx"
    environment     = "production"
    cost_center     = "platform-engineering"
    owner           = "sre-platform"
    finops_governed = "true"
  }
}

variable "billing_account_id" {
  description = "Google Cloud Billing Account ID for budget alerts. If empty, budget resource is skipped."
  type        = string
  default     = ""
}

variable "monthly_budget_usd" {
  description = "Monthly spend cap in USD for FinOps budget enforcement."
  type        = number
  default     = 0
}

variable "enable_cmek_iam_binding" {
  description = "Whether to create the GCS service account IAM binding on the KMS key. Default false for bootstrap/dry-run planning."
  type        = bool
  default     = false
}
