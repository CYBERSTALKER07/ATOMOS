/**
 * Module: Analytics & ML Platform
 * Variable declarations for regional BigQuery datasets/tables, row-level access,
 * authorized tenant views, Vertex AI pipeline staging GCS bucket, and IAM bindings.
 */

variable "project_id" {
  description = "The Google Cloud Platform project ID for the cell analytics platform."
  type        = string
}

variable "region" {
  description = "The regional GCP location for BigQuery datasets and Vertex AI staging bucket (cell residency boundary)."
  type        = string
  default     = "europe-west3"
}

variable "cell_id" {
  description = "Logical cell identifier (e.g. uz, eu) enforcing regional residency labels and row predicates."
  type        = string
  default     = "uz"
}

variable "environment" {
  description = "Deployment environment (e.g. production, staging, cell, development)."
  type        = string
  default     = "production"
}

variable "bigquery_dataset_prefix" {
  description = "Optional prefix applied to raw, curated, and features BigQuery dataset IDs (sanitized to underscores)."
  type        = string
  default     = ""
}

variable "enable_cmek" {
  description = "Whether Customer-Managed Encryption Keys (CMEK) are attached to BigQuery datasets, tables, and the Vertex AI staging bucket."
  type        = bool
  default     = false
}

variable "enable_cmek_iam_binding" {
  description = "Whether to provision the BigQuery service-agent KMS encrypter/decrypter IAM binding (requires live project API lookup)."
  type        = bool
  default     = false
}

variable "kms_key_id" {
  description = "Optional Cloud KMS CryptoKey resource ID for CMEK encryption when enable_cmek = true."
  type        = string
  default     = ""
}

variable "vertex_staging_bucket_name" {
  description = "Optional globally unique GCS bucket name for Vertex AI pipeline staging. Defaults to <project_id>-<cell_id>-vertex-staging."
  type        = string
  default     = ""
}

variable "backend_sa_email" {
  description = "Email of the backend-go Workload Identity Google Service Account."
  type        = string
  default     = ""
}

variable "ai_worker_sa_email" {
  description = "Email of the ai-worker Workload Identity Google Service Account."
  type        = string
  default     = ""
}

variable "enable_iam_bindings" {
  description = "Whether to create BigQuery, Vertex AI, and GCS IAM bindings and row-access policies for workload SAs."
  type        = bool
  default     = true
}

variable "cell_scoped_iam" {
  description = "Scope BigQuery dataEditor IAM bindings to individual cell datasets instead of project-wide."
  type        = bool
  default     = true
}

variable "deletion_protection" {
  description = "Whether BigQuery analytical tables have deletion protection enabled."
  type        = bool
  default     = true
}

variable "bigquery_authorized_reader_email" {
  description = "Optional analyst/BI group or service account email granted BigQuery jobUser + metadataViewer for tenant-authorized views."
  type        = string
  default     = ""
}

variable "tenant_row_access_policies" {
  description = "Optional per-tenant row-access policies mapping policy key to { tenant_id, grantees } for direct BigQuery BI access."
  type = map(object({
    tenant_id = string
    grantees  = list(string)
  }))
  default = {}
}

variable "labels" {
  description = "Common FinOps and governance labels applied to BigQuery and Vertex AI staging resources."
  type        = map(string)
  default = {
    managed_by = "terraform"
    system     = "pegasusx"
    tier       = "analytics"
  }
}
