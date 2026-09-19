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
}

variable "release_channel" {
  description = "The GKE release channel (REGULAR, RAPID, STABLE)."
  type        = string
  default     = "REGULAR"
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
