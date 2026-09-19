# GS-C5 — shared DNS zone. Records per cell. Do not apply from the catalog.
resource "google_dns_managed_zone" "this" {
  project     = var.project_id
  name        = var.zone_name
  dns_name    = var.dns_name
  description = "PegasusX cell API zone (GS-C5). Not a cell stack."

  labels = {
    app        = "pegasusx"
    plane      = "global"
    managed_by = "terraform"
  }
}

resource "google_dns_record_set" "cell" {
  for_each     = var.records
  project      = var.project_id
  managed_zone = google_dns_managed_zone.this.name
  name         = each.key
  type         = each.value.type
  ttl          = each.value.ttl
  rrdatas      = each.value.rrdatas
}
