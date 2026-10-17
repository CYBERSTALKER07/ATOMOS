# GS-C5 — global DNS + Artifact Registry. Plan only. Do not apply from the catalog.
# Not a second cell. Does not share Spanner/Kafka/JWT with UZ or EU.

check "global_not_live_uz_project" {
  assert {
    condition     = var.project_id != "pegasus-503013"
    error_message = "GS-C5: global plane must not live in pegasus-503013 (that is the UZ cell)."
  }
}

check "global_not_eu_cell_project" {
  assert {
    condition     = var.project_id != "pegasusx-cell-eu"
    error_message = "GS-C5: global plane must not live in the EU cell project."
  }
}

check "global_prefix_is_global" {
  assert {
    condition     = var.terraform_state_prefix == "pegasusx/global"
    error_message = "GS-C5: terraform_state_prefix must be pegasusx/global (not ssmr / cell-eu)."
  }
}

module "dns" {
  source     = "../modules/global_dns"
  project_id = var.project_id
  records = {
    "api.pegasusx.app." = {
      type    = "A"
      ttl     = 300
      rrdatas = [var.uz_api_ipv4]
    }
    "api-eu.pegasusx.app." = {
      type    = "A"
      ttl     = 300
      rrdatas = [var.eu_api_ipv4]
    }
  }
}

module "ar" {
  source     = "../modules/global_ar"
  project_id = var.project_id
  location   = "europe"
}

output "project_id" {
  value = var.project_id
}

output "dns_zone" {
  value = module.dns.zone_name
}

output "dns_records" {
  value = module.dns.record_names
}

output "artifact_registry_url" {
  value = module.ar.url
}
