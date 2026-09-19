/**
 * Module: Database
 * Variable declarations for Cloud Spanner (Multi-zone Ledger) and Cloud Memorystore for Redis 7.0 HA.
 */

variable "project_id" {
  description = "The Google Cloud Platform project ID."
  type        = string
}

variable "region" {
  description = "The GCP region for regional database and caching resources."
  type        = string
  default     = "europe-west3"
}

variable "spanner_instance_name" {
  description = "The name of the Cloud Spanner instance."
  type        = string
  default     = "pegasusx-ssmr-spanner"
}

variable "spanner_config" {
  description = "The instance config for Cloud Spanner (e.g. regional-europe-west3, regional-asia-south1, or nam3)."
  type        = string
  default     = "regional-europe-west3"
}

variable "spanner_processing_units" {
  description = "The number of processing units allocated to Cloud Spanner (100 to 1000 PU per node)."
  type        = number
  default     = 100
}

variable "spanner_autoscaling_config" {
  description = "Optional autoscaling configuration for Cloud Spanner instance."
  type = object({
    min_processing_units                  = number
    max_processing_units                  = number
    high_priority_cpu_utilization_percent = number
    storage_utilization_percent           = number
  })
  default = {
    min_processing_units                  = 100
    max_processing_units                  = 1000
    high_priority_cpu_utilization_percent = 65
    storage_utilization_percent           = 80
  }
}

variable "spanner_database_name" {
  description = "The name of the Cloud Spanner database."
  type        = string
  default     = "main"
}

variable "spanner_ddl_statements" {
  description = "List of DDL statements to execute on database creation."
  type        = list(string)
  default     = []
}

variable "spanner_backup_retention_days" {
  description = "Number of days to retain automated daily backups."
  type        = number
  default     = 30
}

variable "spanner_backup_cron_spec" {
  description = "Cron schedule for automated Spanner full backups."
  type        = string
  default     = "0 2 * * *"
}

variable "redis_instance_name" {
  description = "The name of the Cloud Memorystore for Redis instance."
  type        = string
  default     = "pegasusx-ssmr-redis"
}

variable "redis_tier" {
  description = "The service tier of the Redis instance (BASIC or STANDARD_HA)."
  type        = string
  default     = "STANDARD_HA"
}

variable "redis_memory_size_gb" {
  description = "Redis memory capacity in GiB (e.g. 2 for SSMR, 5 for Production)."
  type        = number
  default     = 5
}

variable "redis_version" {
  description = "The version of Redis software (REDIS_7_0 recommended)."
  type        = string
  default     = "REDIS_7_0"
}

variable "vpc_id" {
  description = "The ID or name of the VPC network."
  type        = string
}

variable "psa_connection" {
  description = "Dependency hook for Private Service Access (PSA) peering connection."
  type        = any
  default     = null
}

variable "deletion_protection" {
  description = "Whether to prevent accidental destruction of the database instance and database."
  type        = bool
  default     = true
}

variable "environment" {
  description = "The deployment environment (e.g. production, staging, ssmr)."
  type        = string
  default     = "production"
}

variable "labels" {
  description = "Resource labels applied to database and cache resources."
  type        = map(string)
  default = {
    managed_by = "terraform"
    system     = "pegasusx"
    tier       = "database"
  }
}
