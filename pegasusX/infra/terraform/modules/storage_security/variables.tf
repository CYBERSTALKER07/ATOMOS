/**
 * Module: Storage & Security
 * Variable declarations for GCS Buckets, Cloud Armor WAF, Secret Manager, and Workload Identity IAM.
 */

variable "project_id" {
  description = "The Google Cloud Platform project ID."
  type        = string
}

variable "region" {
  description = "The GCP region for regional storage buckets."
  type        = string
  default     = "europe-west3"
}

variable "location" {
  description = "The multi-region or regional location for multi-region GCS buckets."
  type        = string
  default     = "EU"
}

variable "environment" {
  description = "The deployment environment (e.g. production, staging, ssmr)."
  type        = string
  default     = "production"
}

variable "media_bucket_name" {
  description = "The name of the GCS bucket for proof-of-delivery media and cryptographic evidence dossiers."
  type        = string
  default     = "pegasusx-prod-media"
}

variable "updates_bucket_name" {
  description = "The name of the GCS bucket for mobile APKs and desktop app OTA updates."
  type        = string
  default     = "pegasusx-prod-app-updates"
}

variable "imports_bucket_name" {
  description = "The name of the GCS bucket for supplier CSV/XLSX imports and compliance exports."
  type        = string
  default     = "pegasusx-prod-imports-exports"
}

variable "tf_state_bucket_name" {
  description = "The name of the GCS bucket for Terraform remote state."
  type        = string
  default     = "pegasusx-terraform-state"
}

variable "cors_origins" {
  description = "List of allowed CORS origins for media direct client uploads."
  type        = list(string)
  default     = ["https://*.pegasusx.uz", "https://*.pegasusx.io", "http://localhost:3000"]
}

variable "k8s_namespace" {
  description = "Kubernetes namespace where production workloads run."
  type        = string
  default     = "pegasusx"
}

variable "rate_limit_threshold_count" {
  description = "Number of requests allowed per rate limit interval in Cloud Armor."
  type        = number
  default     = 500
}

variable "rate_limit_interval_sec" {
  description = "Interval in seconds for Cloud Armor rate limiting."
  type        = number
  default     = 10
}

variable "blocked_ip_ranges" {
  description = "List of CIDR blocks to immediately deny in Cloud Armor WAF."
  type        = list(string)
  default     = []
}

variable "allowed_ip_ranges" {
  description = "List of trusted CIDR blocks to bypass WAF inspection."
  type        = list(string)
  default     = []
}

variable "labels" {
  description = "Resource labels applied to storage and security resources."
  type        = map(string)
  default = {
    managed_by = "terraform"
    system     = "pegasusx"
    tier       = "storage_security"
  }
}
