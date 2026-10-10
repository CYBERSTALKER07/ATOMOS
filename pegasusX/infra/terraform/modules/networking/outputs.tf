/**
 * Module: Networking
 * Output declarations
 */

output "network_id" {
  description = "The ID of the VPC network."
  value       = google_compute_network.vpc.id
}

output "network_self_link" {
  description = "The URI of the VPC network."
  value       = google_compute_network.vpc.self_link
}

output "network_name" {
  description = "The name of the VPC network."
  value       = google_compute_network.vpc.name
}

output "subnetwork_id" {
  description = "The ID of the primary subnetwork."
  value       = google_compute_subnetwork.subnet.id
}

output "subnetwork_self_link" {
  description = "The URI of the primary subnetwork."
  value       = google_compute_subnetwork.subnet.self_link
}

output "subnetwork_name" {
  description = "The name of the primary subnetwork."
  value       = google_compute_subnetwork.subnet.name
}

output "pod_ip_range_name" {
  description = "The secondary IP range name reserved for GKE Pods."
  value       = var.pod_ip_range_name
}

output "service_ip_range_name" {
  description = "The secondary IP range name reserved for GKE Services."
  value       = var.service_ip_range_name
}

output "nat_ip_addresses" {
  description = "List of static external IP addresses assigned to the Cloud NAT gateway."
  value       = google_compute_address.nat_ips[*].address
}

output "nat_ip_self_links" {
  description = "List of self links for the Cloud NAT static external IP addresses."
  value       = google_compute_address.nat_ips[*].self_link
}

output "psa_connection_id" {
  description = "The ID of the Private Service Access peering connection."
  value       = google_service_networking_connection.psa_peering.id
}

output "psa_peering_network" {
  description = "The network used for Private Service Access peering."
  value       = google_service_networking_connection.psa_peering.network
}
