/**
 * Module: Compute
 * Variable declarations for GKE Cluster, Workload Identity, Node IAM, and network topology.
 */

variable "project_id" {
  description = "The Google Cloud Platform project ID."
  type        = string
}

variable "region" {
  description = "The GCP region for the regional GKE cluster (multi-zone HA)."
  type        = string
  default     = "europe-west3"
}

variable "cluster_name" {
  description = "The name of the GKE Kubernetes cluster."
  type        = string
  default     = "pegasusx-prod-gke"
}

variable "network_name" {
  description = "The VPC network name or self link."
  type        = string
}

variable "subnetwork_name" {
  description = "The subnetwork name or self link for GKE nodes."
  type        = string
}

variable "pod_ip_range_name" {
  description = "Secondary IP range name for Kubernetes Pods."
  type        = string
  default     = "gke-pods-secondary"
}

variable "service_ip_range_name" {
  description = "Secondary IP range name for Kubernetes Services."
  type        = string
  default     = "gke-services-secondary"
}

variable "enable_autopilot" {
  description = "Whether to enable GKE Autopilot mode (recommended for production multi-tenant isolation)."
  type        = bool
  default     = true
}

variable "master_ipv4_cidr_block" {
  description = "The /28 CIDR block reserved for the GKE master control plane."
  type        = string
  default     = "172.16.0.0/28"
  validation {
    condition     = can(cidrhost(var.master_ipv4_cidr_block, 0))
    error_message = "master_ipv4_cidr_block must be a valid IPv4 CIDR block."
  }
}

variable "release_channel" {
  description = "The GKE release channel (REGULAR, RAPID, STABLE)."
  type        = string
  default     = "REGULAR"
  validation {
    condition     = contains(["RAPID", "REGULAR", "STABLE"], var.release_channel)
    error_message = "release_channel must be one of RAPID, REGULAR, or STABLE."
  }
}

variable "authorized_ipv4_cidr_blocks" {
  description = "List of CIDR blocks authorized to access the GKE control plane endpoint."
  type = list(object({
    cidr_block   = string
    display_name = string
  }))
  default = [
    {
      cidr_block   = "0.0.0.0/0"
      display_name = "Global Authorized (Override with Bastion/VPN in strict environments)"
    }
  ]
}

variable "standard_node_machine_type" {
  description = "Machine type for standard node pool when enable_autopilot = false."
  type        = string
  default     = "e2-standard-4"
}

variable "standard_node_min_count" {
  description = "Minimum node count per zone when enable_autopilot = false."
  type        = number
  default     = 1
}

variable "standard_node_max_count" {
  description = "Maximum node count per zone when enable_autopilot = false."
  type        = number
  default     = 10
}

variable "enable_spot_node_pool" {
  description = "Whether to provision a Spot VM node pool when enable_autopilot = false (up to 91% savings for batch/stateless workloads)."
  type        = bool
  default     = true
}

variable "spot_node_machine_type" {
  description = "Machine type for the Spot VM node pool when enable_autopilot = false."
  type        = string
  default     = "e2-standard-4"
}

variable "spot_node_min_count" {
  description = "Minimum node count per zone for the Spot VM node pool (0 allows scale-to-zero)."
  type        = number
  default     = 0
}

variable "spot_node_max_count" {
  description = "Maximum node count per zone for the Spot VM node pool."
  type        = number
  default     = 8
}

variable "enable_committed_use_discount" {
  description = "Whether to provision a regional Compute Engine Committed Use Discount (CUD) for steady-state vCPU/memory."
  type        = bool
  default     = false
}

variable "cud_plan" {
  description = "Commitment duration for Compute Engine CUD (TWELVE_MONTH or THIRTY_SIX_MONTH)."
  type        = string
  default     = "TWELVE_MONTH"
}

variable "cud_vcpu_count" {
  description = "Baseline vCPU count for Compute Engine CUD."
  type        = number
  default     = 8
}

variable "cud_memory_mb" {
  description = "Baseline memory in MiB for Compute Engine CUD."
  type        = number
  default     = 32768
}

variable "environment" {
  description = "The deployment environment (e.g. production, staging, ssmr)."
  type        = string
  default     = "production"
}

variable "labels" {
  description = "Labels applied to the GKE cluster."
  type        = map(string)
  default = {
    managed_by = "terraform"
    system     = "pegasusx"
    tier       = "compute"
  }
}
