output "zone_name" {
  description = "The Cloud DNS managed zone resource name."
  value       = google_dns_managed_zone.this.name
}

output "name_servers" {
  description = "Authoritative Cloud DNS name servers assigned to the managed zone."
  value       = google_dns_managed_zone.this.name_servers
}

output "record_names" {
  description = "List of per-cell DNS record FQDNs managed in the global zone."
  value       = [for r in google_dns_record_set.cell : r.name]
}
