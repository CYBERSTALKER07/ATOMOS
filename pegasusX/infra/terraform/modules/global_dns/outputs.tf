output "zone_name" {
  value = google_dns_managed_zone.this.name
}

output "name_servers" {
  value = google_dns_managed_zone.this.name_servers
}

output "record_names" {
  value = [for r in google_dns_record_set.cell : r.name]
}
