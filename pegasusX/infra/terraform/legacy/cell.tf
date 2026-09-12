# GS-C1 — cell-safe same Terraform root (plan only; do not apply from the catalog).
# tenant_slug namespaces resource names. cell_id is the cell attribute.

variable "cell_id" {
  description = "Logical cell id (uz, eu, …). Not a tenant slug. One cell = one project + region + state prefix."
  type        = string
  default     = "uz"
}

variable "api_hostname" {
  description = "Public API hostname for this cell (ingress / PUBLIC_BASE_URL)."
  type        = string
  default     = "api.pegasusx.app"
}

variable "k8s_namespace" {
  description = "Kubernetes namespace for Workload Identity (backend-go SA). Live WI is pegasusx/backend-go."
  type        = string
  default     = "pegasusx"
}

variable "terraform_state_prefix" {
  description = "Must match the -backend-config prefix. europe-west1 / non-uz cells must not use pegasusx/ssmr."
  type        = string
  default     = ""
}

variable "gsm_regional_only" {
  description = "GSM user-managed replication in var.region only (C1 law for a new cell). Live ssmr/staging sets false so an accidental apply does not ForceNew JWT/PSP secrets."
  type        = bool
  default     = false
}

variable "vpc_custom_mode" {
  description = "Custom-mode VPC with one regional subnet (C1 law for a new cell). Live ssmr/staging sets false so an accidental apply does not ForceNew the auto-mode VPC."
  type        = bool
  default     = false
}

variable "vpc_subnet_cidr" {
  description = "Primary CIDR for the cell subnet when vpc_custom_mode=true."
  type        = string
  default     = "10.20.0.0/20"
}

variable "vpc_pods_cidr" {
  description = "Secondary CIDR for GKE pods when vpc_custom_mode=true."
  type        = string
  default     = "10.21.0.0/16"
}

variable "vpc_services_cidr" {
  description = "Secondary CIDR for GKE services when vpc_custom_mode=true."
  type        = string
  default     = "10.22.0.0/20"
}

variable "cell_scoped_iam" {
  description = "Bind the backend GSA at Spanner database + per-secret GSM (not project-wide). Memorystore BASIC has no instance IAM — Redis stays project-level."
  type        = bool
  default     = true
}

check "europe_west1_cannot_use_ssmr_state" {
  assert {
    condition = var.region != "europe-west1" || (
      trimspace(var.terraform_state_prefix) != "" &&
      trimspace(var.terraform_state_prefix) != "pegasusx/ssmr"
    )
    error_message = "GS-C1: region europe-west1 must not use terraform_state_prefix pegasusx/ssmr (empty is treated as the live ssmr prefix)."
  }
}

check "non_uz_cell_cannot_use_ssmr_state" {
  assert {
    condition = local.cell_id == "uz" || (
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
    condition     = local.cell_id == "uz" || var.project_id != "pegasus-503013"
    error_message = "GS-C2: a non-uz cell must not share project pegasus-503013."
  }
}

variable "allow_uz_backup_restore" {
  description = "If true, operators may restore a UZ Spanner backup into this cell. Must stay false for every non-uz cell (GS-C3)."
  type        = bool
  default     = false
}

check "non_uz_forbids_uz_restore" {
  assert {
    condition     = local.cell_id == "uz" || !var.allow_uz_backup_restore
    error_message = "GS-C3: a non-uz cell must not restore a UZ Spanner backup. Apply the same DDL (migrations) instead."
  }
}

check "non_uz_requires_cell_scoped_iam" {
  assert {
    condition     = local.cell_id == "uz" || var.cell_scoped_iam
    error_message = "GS-C4: a non-uz cell must use cell_scoped_iam so the EU GSA cannot be a project-wide UZ Spanner/GSM user."
  }
}

output "cell_id" {
  description = "Logical cell id for this root."
  value       = local.cell_id
}

output "api_hostname" {
  description = "Public API hostname for this cell."
  value       = var.api_hostname
}

output "k8s_namespace" {
  description = "Kubernetes namespace used for Workload Identity."
  value       = local.k8s_namespace
}

output "terraform_state_prefix" {
  description = "Declared state prefix (must match backend hcl). Empty means the operator must pass backend-ssmr.hcl for the live cell."
  value       = var.terraform_state_prefix
}
