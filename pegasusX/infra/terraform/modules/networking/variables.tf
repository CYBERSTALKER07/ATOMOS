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
}

variable "pod_cidr_block" {
  description = "The secondary CIDR block allocated for GKE Pods."
  type        = string
  default     = "10.20.0.0/16"
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
}

variable "gke_master_cidr_block" {
  description = "The CIDR block reserved for the GKE master control plane."
  type        = string
  default     = "172.16.0.0/28"
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
