/**
 * Module: Networking
 * Description: Provisions Custom VPC, Subnets with secondary IP ranges for GKE,
 *              Cloud Router, Cloud NAT with 2 static external IPs for deterministic
 *              egress allowlisting (Soliq OFD, banking rails), Private Service Access (PSA),
 *              and VPC firewall rules.
 */

variable "project_id" {
  description = "The Google Cloud Platform project ID."
  type        = string
}

variable "region" {
  description = "The GCP region for regional networking resources."
  type        = string
  default     = "europe-west3"
}

variable "network_name" {
  description = "The name of the VPC network."
  type        = string
  default     = "pegasusx-vpc"
}

variable "subnet_name" {
  description = "The name of the primary subnetwork."
  type        = string
  default     = "pegasusx-primary-subnet"
}

variable "primary_cidr_block" {
  description = "The primary CIDR block for GKE nodes and VMs."
  type        = string
  default     = "10.10.0.0/20"
  validation {
    condition     = can(cidrhost(var.primary_cidr_block, 0))
    error_message = "primary_cidr_block must be a valid IPv4 CIDR notation."
  }
}

variable "pod_cidr_block" {
  description = "The secondary CIDR block allocated for GKE Pods."
  type        = string
  default     = "10.20.0.0/16"
  validation {
    condition     = can(cidrhost(var.pod_cidr_block, 0))
    error_message = "pod_cidr_block must be a valid IPv4 CIDR notation."
  }
}

variable "pod_ip_range_name" {
  description = "The name of the secondary IP range for GKE Pods."
  type        = string
  default     = "gke-pods-secondary"
}

variable "service_cidr_block" {
  description = "The secondary CIDR block allocated for GKE ClusterIP Services."
  type        = string
  default     = "10.30.0.0/20"
  validation {
    condition     = can(cidrhost(var.service_cidr_block, 0))
    error_message = "service_cidr_block must be a valid IPv4 CIDR notation."
  }
}

variable "service_ip_range_name" {
  description = "The name of the secondary IP range for GKE Services."
  type        = string
  default     = "gke-services-secondary"
}

variable "psa_prefix_length" {
  description = "The prefix length for Private Service Access (PSA) global internal address range."
  type        = number
  default     = 24
  validation {
    condition     = var.psa_prefix_length >= 16 && var.psa_prefix_length <= 24
    error_message = "psa_prefix_length must be between 16 and 24."
  }
}

variable "gke_master_cidr_block" {
  description = "The CIDR block reserved for the GKE master control plane."
  type        = string
  default     = "172.16.0.0/28"
  validation {
    condition     = can(cidrhost(var.gke_master_cidr_block, 0))
    error_message = "gke_master_cidr_block must be a valid IPv4 CIDR block."
  }
}

variable "flow_log_interval" {
  description = "Aggregation interval for VPC subnet flow logs (e.g. INTERVAL_30_SEC to reduce Cloud Logging ingestion cost)."
  type        = string
  default     = "INTERVAL_30_SEC"
}

variable "flow_log_sampling" {
  description = "Sampling rate (0.0 to 1.0) for VPC subnet flow logs (default 0.1 for FinOps cost optimization)."
  type        = number
  default     = 0.1
  validation {
    condition     = var.flow_log_sampling >= 0.0 && var.flow_log_sampling <= 1.0
    error_message = "flow_log_sampling must be between 0.0 and 1.0."
  }
}

variable "environment" {
  description = "The deployment environment (e.g. production, staging, ssmr)."
  type        = string
  default     = "production"
}

variable "labels" {
  description = "Key-value resource labels applied to networking resources."
  type        = map(string)
  default = {
    managed_by = "terraform"
    system     = "pegasusx"
    tier       = "networking"
  }
}
